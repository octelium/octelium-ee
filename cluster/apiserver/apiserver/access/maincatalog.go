// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"
	"regexp"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *ServerMain) CreateCatalog(ctx context.Context, req *accessv1.Catalog) (*accessv1.Catalog, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.AccessC().GetCatalog(ctx, apivalidation.ObjectToRGetOptions(req))
	if err == nil {
		return nil, grpcutils.AlreadyExists("The Catalog %s already exists", req.Metadata.Name)
	}
	if !grpcerr.IsNotFound(err) {
		return nil, grpcutils.InternalWithErr(err)
	}

	if err := s.validateCatalog(ctx, req); err != nil {
		return nil, err
	}

	item := &accessv1.Catalog{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &accessv1.Catalog_Status{},
	}

	item, err = s.octeliumC.AccessC().CreateCatalog(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) GetCatalog(ctx context.Context, req *metav1.GetOptions) (*accessv1.Catalog, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetCatalog(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListCatalog(ctx context.Context, req *accessv1.ListCatalogOptions) (*accessv1.CatalogList, error) {
	itemList, err := s.octeliumC.AccessC().ListCatalog(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) DeleteCatalog(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetCatalog(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	refs, err := s.getCatalogReferences(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}
	if len(refs) > 0 {
		return nil, grpcutils.InvalidArg(
			"The Catalog is still referenced by the access Policies: %s", strings.Join(refs, ", "))
	}

	_, err = s.octeliumC.AccessC().DeleteCatalog(ctx, &rmetav1.DeleteOptions{
		Uid: item.Metadata.Uid,
	})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerMain) UpdateCatalog(ctx context.Context, req *accessv1.Catalog) (*accessv1.Catalog, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if err := s.validateCatalog(ctx, req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetCatalog(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	if item.Status == nil {
		item.Status = &accessv1.Catalog_Status{}
	}

	item, err = s.octeliumC.AccessC().UpdateCatalog(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerMain) validateCatalog(ctx context.Context, req *accessv1.Catalog) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	if req.Spec.ResourceCollection == nil {
		return grpcutils.InvalidArg("ResourceCollection must be set")
	}

	if req.Spec.ResourceCollection.Service == nil {
		return grpcutils.InvalidArg("ResourceCollection Service must be set")
	}

	if len(req.Spec.ResourceCollection.Service.Services) == 0 &&
		len(req.Spec.ResourceCollection.Service.Namespaces) == 0 {
		return grpcutils.InvalidArg("ResourceCollection Service must include at least one Service or Namespace")
	}

	for _, svc := range req.Spec.ResourceCollection.Service.Services {
		if svc == "" {
			return grpcutils.InvalidArg("Service name cannot be empty")
		}
	}

	for _, ns := range req.Spec.ResourceCollection.Service.Namespaces {
		if ns == "" {
			return grpcutils.InvalidArg("Namespace name cannot be empty")
		}
	}

	if len(req.Spec.AccessLevels) == 0 {
		return grpcutils.InvalidArg("The Catalog must have at least one access level")
	}

	if len(req.Spec.AccessLevels) > maxCatalogAccessLevels {
		return grpcutils.InvalidArg("The Catalog cannot have more than %d access levels", maxCatalogAccessLevels)
	}

	seenLevels := map[string]struct{}{}
	for _, level := range req.Spec.AccessLevels {
		if err := s.validateCatalogAccessLevel(ctx, level); err != nil {
			return err
		}

		if _, ok := seenLevels[level.Name]; ok {
			return grpcutils.InvalidArg("Duplicate access level name: %s", level.Name)
		}
		seenLevels[level.Name] = struct{}{}
	}

	switch req.Spec.Membership {
	case accessv1.Catalog_Spec_MEMBERSHIP_UNSET,
		accessv1.Catalog_Spec_FIXED,
		accessv1.Catalog_Spec_DYNAMIC:
	default:
		return grpcutils.InvalidArg("Invalid membership")
	}

	if audience := req.Spec.Audience; audience != nil {
		if err := validateUserGroupNames("audience", audience.Users, audience.Groups); err != nil {
			return err
		}

		if audience.Match != "" {
			if err := validatePolicyMatch(ctx, audience.Match); err != nil {
				return err
			}
		}

		if onBehalf := audience.OnBehalf; onBehalf != nil {
			if err := validateUserGroupNames("on-behalf audience", onBehalf.Users, onBehalf.Groups); err != nil {
				return err
			}
		}
	}

	if owners := req.Spec.Owners; owners != nil {
		if err := validateUserGroupNames("owners", owners.Users, owners.Groups); err != nil {
			return err
		}
	}

	if requirements := req.Spec.Requirements; requirements != nil {
		if j := requirements.Justification; j != nil && j.MinLength > accesscmd.MaxJustificationLen {
			return grpcutils.InvalidArg("The justification minimum length cannot exceed %d",
				accesscmd.MaxJustificationLen)
		}

		if t := requirements.Ticket; t != nil {
			if t.Pattern != "" {
				if len(t.Pattern) > 256 {
					return grpcutils.InvalidArg("The ticket pattern is too long")
				}

				if _, err := regexp.Compile(t.Pattern); err != nil {
					return grpcutils.InvalidArg("Invalid ticket pattern: %s", err.Error())
				}
			}

			if accesscmd.TextLen(t.Label) > 64 {
				return grpcutils.InvalidArg("The ticket label is too long")
			}
		}
	}

	return nil
}

const maxCatalogAccessLevels = 16

func (s *ServerMain) validateCatalogAccessLevel(ctx context.Context,
	level *accessv1.Catalog_Spec_AccessLevel) error {
	if level == nil {
		return grpcutils.InvalidArg("Nil access level")
	}

	if err := apivalidation.ValidateName(level.Name, 0, 0); err != nil {
		return grpcutils.InvalidArg("Invalid access level name: %s", level.Name)
	}

	if accesscmd.TextLen(level.DisplayName) > 128 {
		return grpcutils.InvalidArg("The display name of the access level %s is too long", level.Name)
	}

	if accesscmd.TextLen(level.Description) > 1000 {
		return grpcutils.InvalidArg("The description of the access level %s is too long", level.Name)
	}

	switch level.Type.(type) {
	case *accessv1.Catalog_Spec_AccessLevel_Full_:
		if full := level.GetFull(); full != nil && full.Priority != nil &&
			(full.GetPriority() < accesscmd.MinAccessPriority || full.GetPriority() > accesscmd.MaxAccessPriority) {
			return grpcutils.InvalidArg("The priority of the access level %s must be between %d and %d",
				level.Name, accesscmd.MinAccessPriority, accesscmd.MaxAccessPriority)
		}

	case *accessv1.Catalog_Spec_AccessLevel_Restricted_:
		restricted := level.GetRestricted()
		if restricted == nil || (len(restricted.Policies) == 0 && len(restricted.InlinePolicies) == 0) {
			return grpcutils.InvalidArg("The restricted access level %s must have at least one Policy", level.Name)
		}

		for _, pol := range restricted.Policies {
			if err := apivalidation.ValidateName(pol, 0, vutils.MaxPolicyParents); err != nil {
				return err
			}

			if _, err := s.octeliumC.CoreC().GetPolicy(ctx, &rmetav1.GetOptions{
				Name: pol,
			}); err != nil {
				if grpcerr.IsNotFound(err) {
					return serr.InvalidArg("Policy not found: %s", pol)
				}
				return serr.K8sNotFoundOrInternalWithErr(err)
			}
		}

		for _, pol := range restricted.InlinePolicies {
			if err := validateInlinePolicy(ctx, level.Name, pol); err != nil {
				return err
			}
		}

	default:
		return grpcutils.InvalidArg("The type of the access level %s must be set", level.Name)
	}

	for _, d := range []*metav1.Duration{level.DefaultDuration, level.MaxDuration} {
		if d == nil {
			continue
		}

		if err := apivalidation.ValidateDuration(d); err != nil {
			return err
		}
	}

	if level.DefaultDuration != nil && level.MaxDuration != nil &&
		umetav1.ToDuration(level.DefaultDuration).ToGo() > umetav1.ToDuration(level.MaxDuration).ToGo() {
		return grpcutils.InvalidArg("The default duration of the access level %s exceeds its maximum duration",
			level.Name)
	}

	if v := level.Verification; v != nil {
		switch v.MinAAL {
		case corev1.Session_Status_Authentication_Info_AAL_UNSET,
			corev1.Session_Status_Authentication_Info_AAL1,
			corev1.Session_Status_Authentication_Info_AAL2,
			corev1.Session_Status_Authentication_Info_AAL3:
		default:
			return grpcutils.InvalidArg("Invalid minimum AAL of the access level %s", level.Name)
		}
	}

	return nil
}

func validateInlinePolicy(ctx context.Context, levelName string, pol *corev1.InlinePolicy) error {
	if pol == nil || pol.Spec == nil {
		return grpcutils.InvalidArg("The access level %s has an empty inline Policy", levelName)
	}

	if len(pol.Spec.Rules) == 0 {
		return grpcutils.InvalidArg("The inline Policy of the access level %s must have at least one rule", levelName)
	}

	for _, rule := range pol.Spec.Rules {
		if rule == nil || rule.Condition == nil {
			return grpcutils.InvalidArg("The inline Policy of the access level %s has a rule without a condition",
				levelName)
		}

		switch rule.Effect {
		case corev1.Policy_Spec_Rule_ALLOW, corev1.Policy_Spec_Rule_DENY:
		default:
			return grpcutils.InvalidArg("The inline Policy of the access level %s has a rule without an effect",
				levelName)
		}

		if rule.Priority < -4 || rule.Priority > 4 {
			return grpcutils.InvalidArg("The inline Policy rule priority of the access level %s must be between -4 and 4",
				levelName)
		}

		if err := validateCoreCondition(ctx, rule.Condition, 0); err != nil {
			return err
		}
	}

	return nil
}

func validateCoreCondition(ctx context.Context, cond *corev1.Condition, depth int) error {
	if cond == nil {
		return grpcutils.InvalidArg("Nil Condition")
	}

	if depth > 16 {
		return grpcutils.InvalidArg("Condition is too deeply nested")
	}

	switch cond.Type.(type) {
	case *corev1.Condition_Match:
		return validatePolicyMatch(ctx, cond.GetMatch())
	case *corev1.Condition_All_:
		for _, sub := range cond.GetAll().GetOf() {
			if err := validateCoreCondition(ctx, sub, depth+1); err != nil {
				return err
			}
		}
	case *corev1.Condition_Any_:
		for _, sub := range cond.GetAny().GetOf() {
			if err := validateCoreCondition(ctx, sub, depth+1); err != nil {
				return err
			}
		}
	case *corev1.Condition_None_:
		for _, sub := range cond.GetNone().GetOf() {
			if err := validateCoreCondition(ctx, sub, depth+1); err != nil {
				return err
			}
		}
	case *corev1.Condition_Not:
		return validatePolicyMatch(ctx, cond.GetNot())
	case nil:
		return grpcutils.InvalidArg("Condition type must be set")
	}

	return nil
}

func validateUserGroupNames(what string, users, groups []string) error {
	if len(users) > 256 || len(groups) > 256 {
		return grpcutils.InvalidArg("Too many Users or Groups in the %s", what)
	}

	for _, name := range users {
		if err := apivalidation.ValidateName(name, 0, 0); err != nil {
			return grpcutils.InvalidArg("Invalid User name in the %s: %s", what, name)
		}
	}

	for _, name := range groups {
		if err := apivalidation.ValidateName(name, 0, 0); err != nil {
			return grpcutils.InvalidArg("Invalid Group name in the %s: %s", what, name)
		}
	}

	return nil
}

func (s *ServerMain) getCatalogReferences(ctx context.Context, catalog *accessv1.Catalog) ([]string, error) {
	polList, err := s.octeliumC.AccessC().ListPolicy(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var ret []string
	for _, pol := range polList.Items {
		for _, rule := range pol.GetSpec().GetRules() {
			if conditionReferencesCatalog(rule.GetCondition(), catalog.Metadata.Uid, 0) {
				ret = append(ret, pol.Metadata.Name)
				break
			}
		}
	}

	return ret, nil
}

func conditionReferencesCatalog(cond *accessv1.Policy_Spec_Rule_Condition, uid string, depth int) bool {
	if cond == nil || depth > 16 {
		return false
	}

	switch cond.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_Resource_:
		ref := cond.GetResource().GetCatalogRef()
		return ref != nil && ref.Uid == uid
	case *accessv1.Policy_Spec_Rule_Condition_All_:
		for _, sub := range cond.GetAll().GetOf() {
			if conditionReferencesCatalog(sub, uid, depth+1) {
				return true
			}
		}
	case *accessv1.Policy_Spec_Rule_Condition_Any_:
		for _, sub := range cond.GetAny().GetOf() {
			if conditionReferencesCatalog(sub, uid, depth+1) {
				return true
			}
		}
	}

	return false
}
