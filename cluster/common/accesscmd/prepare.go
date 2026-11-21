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
	"regexp"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
)

const (
	DefaultRequestDeadline = 7 * 24 * time.Hour
	DefaultMaxStartDelay   = 30 * 24 * time.Hour
)

type PrepareRequestOpts struct {
	OcteliumC octeliumc.ClientInterface
	CELEngine *celengine.CELEngine

	Requester *corev1.User
	Spec      *accessv1.Request_Spec

	ForSubject bool

	ClusterConfig *accessv1.ClusterConfig

	SkipDefaults bool
}

type PreparedRequest struct {
	Spec          *accessv1.Request_Spec
	Requester     *corev1.User
	Subject       *corev1.User
	Catalog       *accessv1.Catalog
	Scope         *Scope
	ClusterConfig *accessv1.ClusterConfig
	Candidates    []*accessv1.Catalog
	Problems      []*Problem
}

func (p *PreparedRequest) IsRequestable() bool {
	return !HasFatalProblem(p.Problems)
}

func GetClusterConfig(ctx context.Context, octeliumC octeliumc.ClientInterface) (*accessv1.ClusterConfig, error) {
	cc, err := octeliumC.AccessV1Utils().GetClusterConfig(ctx)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return &accessv1.ClusterConfig{
				Spec:   &accessv1.ClusterConfig_Spec{},
				Status: &accessv1.ClusterConfig_Status{},
			}, nil
		}
		return nil, err
	}

	return cc, nil
}

func PrepareRequest(ctx context.Context, o *PrepareRequestOpts) (*PreparedRequest, error) {
	ret := &PreparedRequest{
		Requester:     o.Requester,
		ClusterConfig: o.ClusterConfig,
	}

	if ret.ClusterConfig == nil {
		cc, err := GetClusterConfig(ctx, o.OcteliumC)
		if err != nil {
			return nil, err
		}
		ret.ClusterConfig = cc
	}

	addProblem := func(p *Problem) {
		ret.Problems = append(ret.Problems, p)
	}

	if o.Spec == nil {
		addProblem(newFatalProblem("spec", "The Request spec must be set"))
		return ret, nil
	}

	spec := pbutils.Clone(o.Spec).(*accessv1.Request_Spec)
	ret.Spec = spec

	if o.ForSubject {
		ref := spec.GetSubject().GetUserRef()
		if ref == nil {
			addProblem(newFatalProblem("spec.subject.userRef", "The subject must be set"))
			return ret, nil
		}

		if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			addProblem(newFatalProblem("spec.subject.userRef", "Invalid subject reference"))
			return ret, nil
		}

		subject, err := GetUser(ctx, o.OcteliumC, ref)
		if err != nil {
			return nil, err
		}
		if subject == nil {
			addProblem(newFatalProblem("spec.subject.userRef", "The subject User does not exist"))
			return ret, nil
		}

		ret.Subject = subject
	} else {
		ret.Subject = o.Requester
	}

	if !IsUserEligible(ret.Subject) {
		addProblem(newFatalProblem("spec.subject.userRef", "The subject User cannot be granted any access"))
		return ret, nil
	}

	spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: objectRef(ret.Subject),
		},
	}

	if spec.Resource == nil {
		addProblem(newFatalProblem("spec.resource", "The resource must be set"))
		return ret, nil
	}

	var shorthandSvc *corev1.Service

	switch spec.Resource.Type.(type) {
	case *accessv1.Request_Spec_Resource_ServiceRef:
		catalog, svc, candidates, problem, err := resolveServiceShorthand(ctx, o, ret.Subject,
			spec.Resource.GetServiceRef())
		if err != nil {
			return nil, err
		}

		ret.Candidates = candidates

		if problem != nil {
			addProblem(problem)
			return ret, nil
		}

		resCatalog := &accessv1.Request_Spec_Resource_Catalog{
			CatalogRef: objectRef(catalog),
		}

		if catalog.Spec.AllowPartialRequests {
			resCatalog.ServiceRefs = []*metav1.ObjectReference{objectRef(svc)}
		}

		spec.Resource = &accessv1.Request_Spec_Resource{
			Type: &accessv1.Request_Spec_Resource_Catalog_{
				Catalog: resCatalog,
			},
		}

		ret.Catalog = catalog
		shorthandSvc = svc

	case *accessv1.Request_Spec_Resource_Catalog_:
		ref := spec.Resource.GetCatalog().GetCatalogRef()
		if ref == nil {
			addProblem(newFatalProblem("spec.resource.catalog.catalogRef", "The Catalog must be set"))
			return ret, nil
		}

		if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			addProblem(newFatalProblem("spec.resource.catalog.catalogRef", "Invalid Catalog reference"))
			return ret, nil
		}

		catalog, err := GetCatalog(ctx, o.OcteliumC, ref)
		if err != nil {
			return nil, err
		}
		if catalog == nil {
			addProblem(newFatalProblem("spec.resource.catalog.catalogRef", "The Catalog does not exist"))
			return ret, nil
		}

		spec.Resource.GetCatalog().CatalogRef = objectRef(catalog)
		ret.Catalog = catalog

	default:
		addProblem(newFatalProblem("spec.resource", "The resource type must be set"))
		return ret, nil
	}

	problem, err := CheckRequestable(ctx, &CheckRequestableOpts{
		CELEngine: o.CELEngine,
		Catalog:   ret.Catalog,
		Requester: o.Requester,
		Subject:   ret.Subject,
	})
	if err != nil {
		return nil, err
	}
	if problem != nil {
		addProblem(problem)
		return ret, nil
	}

	resCatalog := spec.Resource.GetCatalog()

	scope, problems, err := ResolveScope(ctx, o.OcteliumC, &ResolveScopeOpts{
		Catalog:         ret.Catalog,
		AccessLevel:     resCatalog.AccessLevel,
		ServiceRefs:     resCatalog.ServiceRefs,
		DefaultPriority: GetDefaultAccessPriority(ret.ClusterConfig),
	})
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		ret.Problems = append(ret.Problems, problems...)
		return ret, nil
	}

	if shorthandSvc != nil && !isScopeOnlyService(scope.Status, shorthandSvc) {
		addProblem(newFatalProblem("spec.resource.serviceRef",
			"The Service %s can only be requested as a part of the Catalog %s which grants more than this Service. Request the Catalog instead",
			shorthandSvc.Metadata.Name, ret.Catalog.Metadata.Name))
		return ret, nil
	}

	ret.Scope = scope
	resCatalog.AccessLevel = scope.Level.Name
	if len(resCatalog.ServiceRefs) > 0 {
		resCatalog.ServiceRefs = cloneRefs(scope.Status.ServiceRefs)
	}

	ret.Problems = append(ret.Problems, ValidateRequestFields(spec, ret.Catalog, ret.ClusterConfig)...)

	if spec.ExtendsRef != nil {
		problem, err := checkExtendsRef(ctx, o.OcteliumC, spec, o.Requester, ret.Subject)
		if err != nil {
			return nil, err
		}
		if problem != nil {
			addProblem(problem)
		}
	}

	if !o.SkipDefaults && spec.Deadline == nil {
		spec.Deadline = pbutils.Timestamp(time.Now().Add(GetDefaultDeadline(ret.ClusterConfig)))
	}

	return ret, nil
}

func ValidateRequestFields(spec *accessv1.Request_Spec,
	catalog *accessv1.Catalog, cc *accessv1.ClusterConfig) []*Problem {
	var ret []*Problem

	switch spec.Urgency {
	case accessv1.Request_Spec_URGENCY_UNSET,
		accessv1.Request_Spec_VERY_LOW,
		accessv1.Request_Spec_LOW,
		accessv1.Request_Spec_NORMAL,
		accessv1.Request_Spec_HIGH,
		accessv1.Request_Spec_VERY_HIGH,
		accessv1.Request_Spec_HIGHEST:
	default:
		ret = append(ret, newFatalProblem("spec.urgency", "Invalid urgency"))
	}

	if TextLen(spec.Justification) > MaxJustificationLen {
		ret = append(ret, newFatalProblem("spec.justification",
			"The justification cannot be longer than %d characters", MaxJustificationLen))
	}

	if TextLen(spec.Ticket) > MaxTicketLen {
		ret = append(ret, newFatalProblem("spec.ticket",
			"The ticket cannot be longer than %d characters", MaxTicketLen))
	}

	if requirements := catalog.GetSpec().GetRequirements(); requirements != nil {
		if j := requirements.Justification; j != nil {
			switch {
			case j.IsRequired && spec.Justification == "":
				ret = append(ret, newProblem("spec.justification", "A justification is required"))
			case j.MinLength > 0 && spec.Justification != "" && uint32(TextLen(spec.Justification)) < j.MinLength:
				ret = append(ret, newProblem("spec.justification",
					"The justification must be at least %d characters long", j.MinLength))
			}
		}

		if t := requirements.Ticket; t != nil {
			label := t.Label
			if label == "" {
				label = "ticket"
			}

			switch {
			case t.IsRequired && spec.Ticket == "":
				ret = append(ret, newProblem("spec.ticket", "A %s is required", label))
			case t.Pattern != "" && spec.Ticket != "":
				re, err := regexp.Compile(t.Pattern)
				if err == nil && !isFullMatch(re, spec.Ticket) {
					ret = append(ret, newProblem("spec.ticket", "The %s has an invalid format", label))
				}
			}
		}
	}

	if spec.Duration != nil {
		if err := apivalidation.ValidateDuration(spec.Duration); err != nil {
			ret = append(ret, newFatalProblem("spec.duration", "Invalid duration"))
		}
	}

	now := time.Now()

	if spec.Deadline != nil {
		if !spec.Deadline.IsValid() || !spec.Deadline.AsTime().After(now) {
			ret = append(ret, newProblem("spec.deadline", "The deadline must be in the future"))
		}
	}

	if spec.StartAt != nil {
		if !spec.StartAt.IsValid() {
			ret = append(ret, newProblem("spec.startAt", "Invalid start time"))
		} else if spec.StartAt.AsTime().After(now.Add(GetMaxStartDelay(cc))) {
			ret = append(ret, newProblem("spec.startAt",
				"The access cannot start later than %s from now", GetMaxStartDelay(cc).String()))
		}
	}

	return ret
}

func isFullMatch(re *regexp.Regexp, s string) bool {
	loc := re.FindStringIndex(s)
	return loc != nil && loc[0] == 0 && loc[1] == len(s)
}

func GetDefaultDeadline(cc *accessv1.ClusterConfig) time.Duration {
	if d := umetav1.ToDuration(cc.GetSpec().GetRequest().GetDefaultDeadline()).ToGo(); d > 0 {
		return d
	}

	return DefaultRequestDeadline
}

func GetMaxStartDelay(cc *accessv1.ClusterConfig) time.Duration {
	if d := umetav1.ToDuration(cc.GetSpec().GetRequest().GetMaxStartDelay()).ToGo(); d > 0 {
		return d
	}

	return DefaultMaxStartDelay
}

func GetEffectiveDuration(spec *accessv1.Request_Spec,
	level *accessv1.Catalog_Spec_AccessLevel, rule *accessv1.Policy_Spec_Rule) time.Duration {
	maxDuration := GetMaxDuration(level, rule)

	ret := umetav1.ToDuration(spec.GetDuration()).ToGo()
	if ret <= 0 {
		ret = umetav1.ToDuration(level.GetDefaultDuration()).ToGo()
	}

	if ret <= 0 || ret > maxDuration {
		return maxDuration
	}

	return ret
}

func GetMaxDuration(level *accessv1.Catalog_Spec_AccessLevel, rule *accessv1.Policy_Spec_Rule) time.Duration {
	ret := DefaultMaxAccessDuration
	if d := umetav1.ToDuration(level.GetMaxDuration()).ToGo(); d > 0 {
		ret = d
	}

	if rule != nil {
		if d := umetav1.ToDuration(rule.MaxDuration).ToGo(); d > 0 && d < ret {
			ret = d
		}
	}

	return ret
}

func DurationFromGo(d time.Duration) *metav1.Duration {
	if d <= 0 {
		return nil
	}

	return &metav1.Duration{
		Type: &metav1.Duration_Seconds{
			Seconds: uint32(d.Seconds()),
		},
	}
}

func resolveServiceShorthand(ctx context.Context, o *PrepareRequestOpts, subject *corev1.User,
	ref *metav1.ObjectReference) (*accessv1.Catalog, *corev1.Service, []*accessv1.Catalog, *Problem, error) {
	if ref == nil || (ref.Uid == "" && ref.Name == "") {
		return nil, nil, nil, newFatalProblem("spec.resource.serviceRef", "The Service must be set"), nil
	}

	getOpts := &rmetav1.GetOptions{
		Uid: ref.Uid,
	}
	if ref.Uid == "" {
		getOpts = &rmetav1.GetOptions{
			Name: vutils.GetServiceFullNameFromName(ref.Name),
		}
	}

	svc, err := o.OcteliumC.CoreC().GetService(ctx, getOpts)
	if err != nil {
		if isUnresolvableRef(err) {
			return nil, nil, nil, newFatalProblem("spec.resource.serviceRef", "The Service does not exist"), nil
		}
		return nil, nil, nil, nil, err
	}

	catalogList, err := o.OcteliumC.AccessC().ListCatalog(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var candidates []*accessv1.Catalog

	for _, catalog := range catalogList.Items {
		if !CatalogMayIncludeService(catalog, svc) {
			continue
		}

		problem, err := CheckRequestable(ctx, &CheckRequestableOpts{
			CELEngine: o.CELEngine,
			Catalog:   catalog,
			Requester: o.Requester,
			Subject:   subject,
		})
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if problem != nil {
			continue
		}

		candidates = append(candidates, catalog)
	}

	switch len(candidates) {
	case 0:
		return nil, svc, nil, newFatalProblem("spec.resource.serviceRef",
			"The Service %s is not part of any Catalog that can be requested", svc.Metadata.Name), nil
	case 1:
		return candidates[0], svc, candidates, nil, nil
	default:
		return nil, svc, candidates, newFatalProblem("spec.resource.serviceRef",
			"The Service %s is part of several Catalogs. Choose the Catalog to request", svc.Metadata.Name), nil
	}
}

func checkExtendsRef(ctx context.Context, octeliumC octeliumc.ClientInterface,
	spec *accessv1.Request_Spec, requester, subject *corev1.User) (*Problem, error) {
	if err := apivalidation.CheckObjectRef(spec.ExtendsRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return newFatalProblem("spec.extendsRef", "Invalid extended Request reference"), nil
	}

	extended, err := octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectReferenceToRGetOptions(spec.ExtendsRef))
	if err != nil {
		if isUnresolvableRef(err) {
			return newFatalProblem("spec.extendsRef", "The extended Request does not exist"), nil
		}
		return nil, err
	}

	if !IsApproved(extended) {
		return newFatalProblem("spec.extendsRef", "Only approved Requests can be extended"), nil
	}

	if !RefEqual(GetSubjectUserRef(extended), objectRef(subject)) {
		return newFatalProblem("spec.extendsRef", "The extended Request has a different subject"), nil
	}

	if requester != nil &&
		!RefEqual(extended.Status.UserRef, objectRef(requester)) &&
		!RefEqual(GetSubjectUserRef(extended), objectRef(requester)) {
		return newFatalProblem("spec.extendsRef", "You are not allowed to extend this Request"), nil
	}

	extCatalog := extended.Spec.GetResource().GetCatalog()
	resCatalog := spec.Resource.GetCatalog()

	if extCatalog == nil || !RefEqual(extCatalog.CatalogRef, resCatalog.CatalogRef) {
		return newFatalProblem("spec.extendsRef", "The extended Request is for a different Catalog"), nil
	}

	if extCatalog.AccessLevel != resCatalog.AccessLevel {
		return newFatalProblem("spec.extendsRef", "The extended Request is for a different access level"), nil
	}

	spec.ExtendsRef = objectRef(extended)

	return nil, nil
}

func isScopeOnlyService(scope *accessv1.Request_Status_ResolvedScope, svc *corev1.Service) bool {
	return len(scope.NamespaceRefs) == 0 && len(scope.ServiceRefs) == 1 &&
		scope.ServiceRefs[0].Uid == svc.Metadata.Uid
}

func cloneRefs(refs []*metav1.ObjectReference) []*metav1.ObjectReference {
	ret := make([]*metav1.ObjectReference, 0, len(refs))
	for _, ref := range refs {
		ret = append(ret, cloneRef(ref))
	}

	return ret
}
