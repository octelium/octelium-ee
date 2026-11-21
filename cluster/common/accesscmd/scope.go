// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accesscmd

import (
	"context"
	"sort"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
)

const (
	DefaultAccessPriority    int32 = 0
	MinAccessPriority        int32 = -4
	MaxAccessPriority        int32 = 0
	DefaultMaxAccessDuration       = 24 * time.Hour
)

type Scope struct {
	Status     *accessv1.Request_Status_ResolvedScope
	Catalog    *accessv1.Catalog
	Level      *accessv1.Catalog_Spec_AccessLevel
	Services   []*corev1.Service
	Namespaces []*corev1.Namespace
}

type ResolveScopeOpts struct {
	Catalog         *accessv1.Catalog
	AccessLevel     string
	ServiceRefs     []*metav1.ObjectReference
	DefaultPriority int32
}

func GetCatalog(ctx context.Context, octeliumC octeliumc.ClientInterface,
	ref *metav1.ObjectReference) (*accessv1.Catalog, error) {
	if ref == nil || (ref.Uid == "" && ref.Name == "") {
		return nil, nil
	}

	catalog, err := octeliumC.AccessC().GetCatalog(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		if isUnresolvableRef(err) {
			return nil, nil
		}
		return nil, err
	}

	return catalog, nil
}

func GetAccessLevel(catalog *accessv1.Catalog, name string) *accessv1.Catalog_Spec_AccessLevel {
	if catalog == nil || catalog.Spec == nil || len(catalog.Spec.AccessLevels) == 0 {
		return nil
	}

	if name == "" {
		return catalog.Spec.AccessLevels[0]
	}

	for _, level := range catalog.Spec.AccessLevels {
		if level != nil && level.Name == name {
			return level
		}
	}

	return nil
}

func IsSystemService(svc *corev1.Service) bool {
	if svc == nil || svc.Metadata == nil {
		return false
	}

	return svc.Metadata.IsSystem || svc.Metadata.IsSystemHidden || svc.Metadata.IsUserHidden
}

func GetAccessLevelPriority(level *accessv1.Catalog_Spec_AccessLevel, defaultPriority int32) int32 {
	if level == nil {
		return defaultPriority
	}

	switch level.Type.(type) {
	case *accessv1.Catalog_Spec_AccessLevel_Full_:
		if full := level.GetFull(); full != nil && full.Priority != nil {
			return clampPriority(full.GetPriority())
		}

		return clampPriority(defaultPriority)

	case *accessv1.Catalog_Spec_AccessLevel_Restricted_:
		ret := DefaultAccessPriority
		for _, pol := range level.GetRestricted().GetInlinePolicies() {
			for _, rule := range pol.GetSpec().GetRules() {
				if rule != nil && rule.Priority < ret {
					ret = rule.Priority
				}
			}
		}

		return ret

	default:
		return clampPriority(defaultPriority)
	}
}

func IsFullAccessLevel(level *accessv1.Catalog_Spec_AccessLevel) bool {
	if level == nil {
		return false
	}

	_, ok := level.Type.(*accessv1.Catalog_Spec_AccessLevel_Full_)
	return ok
}

func clampPriority(p int32) int32 {
	switch {
	case p < MinAccessPriority:
		return MinAccessPriority
	case p > MaxAccessPriority:
		return MaxAccessPriority
	default:
		return p
	}
}

func GetDefaultAccessPriority(cc *accessv1.ClusterConfig) int32 {
	authz := cc.GetSpec().GetAuthorization()
	if authz == nil || authz.Priority == nil {
		return DefaultAccessPriority
	}

	return clampPriority(authz.GetPriority())
}

func ResolveScope(ctx context.Context, octeliumC octeliumc.ClientInterface,
	o *ResolveScopeOpts) (*Scope, []*Problem, error) {
	catalog := o.Catalog
	if catalog == nil || catalog.Spec == nil {
		return nil, []*Problem{newFatalProblem("spec.resource.catalog.catalogRef",
			"The Catalog does not exist")}, nil
	}

	level := GetAccessLevel(catalog, o.AccessLevel)
	if level == nil {
		if len(catalog.Spec.AccessLevels) == 0 {
			return nil, []*Problem{newFatalProblem("spec.resource.catalog.accessLevel",
				"The Catalog %s has no access level", catalog.Metadata.Name)}, nil
		}

		return nil, []*Problem{newFatalProblem("spec.resource.catalog.accessLevel",
			"The Catalog %s has no access level %s", catalog.Metadata.Name, o.AccessLevel)}, nil
	}

	services, namespaces, excluded, err := resolveCatalogServices(ctx, octeliumC, catalog)
	if err != nil {
		return nil, nil, err
	}

	if len(o.ServiceRefs) > 0 {
		if !catalog.Spec.AllowPartialRequests {
			return nil, []*Problem{newFatalProblem("spec.resource.catalog.serviceRefs",
				"The Catalog %s does not allow requesting only some of its Services", catalog.Metadata.Name)}, nil
		}

		subset, problem := selectServices(services, o.ServiceRefs)
		if problem != nil {
			return nil, []*Problem{problem}, nil
		}

		services = subset
		namespaces = nil
	}

	if len(services) == 0 && len(namespaces) == 0 {
		return nil, []*Problem{newFatalProblem("spec.resource",
			"The Catalog %s does not include any Service that can be requested", catalog.Metadata.Name)}, nil
	}

	priority := GetAccessLevelPriority(level, o.DefaultPriority)

	overrides, err := doesLevelOverrideGuardrails(ctx, octeliumC, level, priority)
	if err != nil {
		return nil, nil, err
	}

	status := &accessv1.Request_Status_ResolvedScope{
		CatalogRef:             objectRef(catalog),
		CatalogResourceVersion: catalog.Metadata.ResourceVersion,
		CatalogDisplayName:     catalog.Metadata.DisplayName,
		AccessLevel:            pbutils.Clone(level).(*accessv1.Catalog_Spec_AccessLevel),
		Membership:             GetMembership(catalog),
		ResolvedAt:             pbutils.Now(),
		Priority:               priority,
		OverridesGuardrails:    overrides,
		TotalExcludedService:   excluded,
	}

	for _, svc := range services {
		status.ServiceRefs = append(status.ServiceRefs, objectRef(svc))
	}

	for _, ns := range namespaces {
		status.NamespaceRefs = append(status.NamespaceRefs, objectRef(ns))
	}

	return &Scope{
		Status:     status,
		Catalog:    catalog,
		Level:      level,
		Services:   services,
		Namespaces: namespaces,
	}, nil, nil
}

func GetMembership(catalog *accessv1.Catalog) accessv1.Catalog_Spec_Membership {
	if catalog.GetSpec().GetMembership() == accessv1.Catalog_Spec_DYNAMIC {
		return accessv1.Catalog_Spec_DYNAMIC
	}

	return accessv1.Catalog_Spec_FIXED
}

func IsScopeEqual(a, b *accessv1.Request_Status_ResolvedScope) bool {
	if a == nil || b == nil {
		return a == b
	}

	ac := pbutils.Clone(a).(*accessv1.Request_Status_ResolvedScope)
	bc := pbutils.Clone(b).(*accessv1.Request_Status_ResolvedScope)
	ac.ResolvedAt = nil
	bc.ResolvedAt = nil

	return pbutils.IsEqual(ac, bc)
}

func ScopeIncludesService(scope *accessv1.Request_Status_ResolvedScope, svc *corev1.Service) bool {
	if scope == nil || svc == nil {
		return false
	}

	for _, ref := range scope.ServiceRefs {
		if ref.Uid == svc.Metadata.Uid {
			return true
		}
	}

	if svc.Status != nil && svc.Status.NamespaceRef != nil {
		for _, ref := range scope.NamespaceRefs {
			if ref.Uid == svc.Status.NamespaceRef.Uid {
				return true
			}
		}
	}

	return false
}

func CatalogMayIncludeService(catalog *accessv1.Catalog, svc *corev1.Service) bool {
	coll := catalog.GetSpec().GetResourceCollection().GetService()
	if coll == nil || svc == nil {
		return false
	}

	if IsSystemService(svc) && !catalog.Spec.AllowSystemResources {
		return false
	}

	for _, name := range coll.Services {
		if name != "" && vutils.GetServiceFullNameFromName(name) == svc.Metadata.Name {
			return true
		}
	}

	if svc.Status != nil && svc.Status.NamespaceRef != nil {
		for _, name := range coll.Namespaces {
			if name == svc.Status.NamespaceRef.Name {
				return true
			}
		}
	}

	return false
}

func resolveCatalogServices(ctx context.Context, octeliumC octeliumc.ClientInterface,
	catalog *accessv1.Catalog) ([]*corev1.Service, []*corev1.Namespace, uint32, error) {
	coll := catalog.Spec.GetResourceCollection().GetService()
	if coll == nil {
		return nil, nil, 0, nil
	}

	var excluded uint32
	seen := map[string]struct{}{}
	var services []*corev1.Service
	var namespaces []*corev1.Namespace

	addService := func(svc *corev1.Service) {
		if _, ok := seen[svc.Metadata.Uid]; ok {
			return
		}
		seen[svc.Metadata.Uid] = struct{}{}

		if IsSystemService(svc) && !catalog.Spec.AllowSystemResources {
			excluded++
			return
		}

		services = append(services, svc)
	}

	for _, name := range coll.Services {
		if name == "" {
			continue
		}

		svc, err := octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
			Name: vutils.GetServiceFullNameFromName(name),
		})
		if err != nil {
			if isUnresolvableRef(err) {
				continue
			}
			return nil, nil, 0, err
		}

		addService(svc)
	}

	for _, nsName := range coll.Namespaces {
		if nsName == "" {
			continue
		}

		ns, err := octeliumC.CoreC().GetNamespace(ctx, &rmetav1.GetOptions{
			Name: nsName,
		})
		if err != nil {
			if isUnresolvableRef(err) {
				continue
			}
			return nil, nil, 0, err
		}

		svcList, err := octeliumC.CoreC().ListService(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("status.namespaceRef.uid", ns.Metadata.Uid),
			},
		})
		if err != nil {
			return nil, nil, 0, err
		}

		for _, svc := range svcList.Items {
			addService(svc)
		}

		if GetMembership(catalog) == accessv1.Catalog_Spec_DYNAMIC {
			namespaces = append(namespaces, ns)
		}
	}

	sort.SliceStable(services, func(i, j int) bool {
		return services[i].Metadata.Name < services[j].Metadata.Name
	})

	sort.SliceStable(namespaces, func(i, j int) bool {
		return namespaces[i].Metadata.Name < namespaces[j].Metadata.Name
	})

	return services, namespaces, excluded, nil
}

func selectServices(services []*corev1.Service,
	refs []*metav1.ObjectReference) ([]*corev1.Service, *Problem) {
	seen := map[string]struct{}{}
	var ret []*corev1.Service

	for _, ref := range refs {
		if ref == nil || (ref.Uid == "" && ref.Name == "") {
			return nil, newFatalProblem("spec.resource.catalog.serviceRefs", "Invalid Service reference")
		}

		var found *corev1.Service
		for _, svc := range services {
			if (ref.Uid != "" && ref.Uid == svc.Metadata.Uid) ||
				(ref.Uid == "" && vutils.GetServiceFullNameFromName(ref.Name) == svc.Metadata.Name) {
				found = svc
				break
			}
		}

		if found == nil {
			name := ref.Name
			if name == "" {
				name = ref.Uid
			}
			return nil, newFatalProblem("spec.resource.catalog.serviceRefs",
				"The Service %s is not part of the Catalog", name)
		}

		if _, ok := seen[found.Metadata.Uid]; ok {
			continue
		}
		seen[found.Metadata.Uid] = struct{}{}

		ret = append(ret, found)
	}

	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i].Metadata.Name < ret[j].Metadata.Name
	})

	return ret, nil
}

func doesLevelOverrideGuardrails(ctx context.Context, octeliumC octeliumc.ClientInterface,
	level *accessv1.Catalog_Spec_AccessLevel, priority int32) (bool, error) {
	if priority < 0 {
		return true, nil
	}

	restricted := level.GetRestricted()
	if restricted == nil {
		return false, nil
	}

	for _, name := range restricted.Policies {
		pol, err := octeliumC.CoreC().GetPolicy(ctx, &rmetav1.GetOptions{
			Name: name,
		})
		if err != nil {
			if grpcerr.IsNotFound(err) {
				continue
			}
			return false, err
		}

		for _, rule := range pol.GetSpec().GetRules() {
			if rule != nil && rule.Priority < 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

func objectRef(itm umetav1.ResourceObjectI) *metav1.ObjectReference {
	return cloneRef(umetav1.GetObjectReference(itm))
}

func RefEqual(a, b *metav1.ObjectReference) bool {
	if a == nil || b == nil {
		return false
	}

	if a.Uid != "" && b.Uid != "" {
		return a.Uid == b.Uid
	}

	return false
}

func errNilArg(name string) error {
	return errors.Errorf("nil %s", name)
}
