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

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *ServerMain) CreatePolicy(ctx context.Context, req *accessv1.Policy) (*accessv1.Policy, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.AccessC().GetPolicy(ctx, apivalidation.ObjectToRGetOptions(req))
	if err == nil {
		return nil, grpcutils.AlreadyExists("The Policy %s already exists", req.Metadata.Name)
	}
	if !grpcerr.IsNotFound(err) {
		return nil, grpcutils.InternalWithErr(err)
	}

	if err := s.validatePolicy(ctx, req); err != nil {
		return nil, err
	}

	item := &accessv1.Policy{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &accessv1.Policy_Status{},
	}

	item, err = s.octeliumC.AccessC().CreatePolicy(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) GetPolicy(ctx context.Context, req *metav1.GetOptions) (*accessv1.Policy, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetPolicy(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListPolicy(ctx context.Context, req *accessv1.ListPolicyOptions) (*accessv1.PolicyList, error) {
	itemList, err := s.octeliumC.AccessC().ListPolicy(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) DeletePolicy(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetPolicy(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	_, err = s.octeliumC.AccessC().DeletePolicy(ctx, &rmetav1.DeleteOptions{
		Uid: item.Metadata.Uid,
	})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerMain) UpdatePolicy(ctx context.Context, req *accessv1.Policy) (*accessv1.Policy, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if err := s.validatePolicy(ctx, req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetPolicy(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	item, err = s.octeliumC.AccessC().UpdatePolicy(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerMain) validatePolicy(ctx context.Context, req *accessv1.Policy) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	seenRules := map[string]struct{}{}

	for _, rule := range req.Spec.Rules {
		if rule == nil {
			return grpcutils.InvalidArg("Nil Rule")
		}

		if rule.Name == "" {
			return grpcutils.InvalidArg("Rule name must be set")
		}

		if _, ok := seenRules[rule.Name]; ok {
			return grpcutils.InvalidArg("Duplicate Rule name: %s", rule.Name)
		}
		seenRules[rule.Name] = struct{}{}

		if err := s.validatePolicyCondition(ctx, rule.Condition, 0); err != nil {
			return err
		}

		switch rule.Effect {
		case accessv1.Policy_Spec_Rule_DENY:
			if rule.Action != nil && rule.Action.GetReview() != nil {
				return grpcutils.InvalidArg("DENY Rule %s cannot have Review action", rule.Name)
			}
			if rule.Authorization != nil &&
				(len(rule.Authorization.Policies) > 0 || len(rule.Authorization.InlinePolicies) > 0) {
				return grpcutils.InvalidArg("DENY Rule %s cannot have Authorization", rule.Name)
			}

		case accessv1.Policy_Spec_Rule_REVIEW:
			if rule.Action == nil || rule.Action.GetReview() == nil {
				return grpcutils.InvalidArg("REVIEW Rule %s must have Review action", rule.Name)
			}
			if err := s.validatePolicyReview(ctx, rule.Name, rule.Action.GetReview()); err != nil {
				return err
			}
			if rule.Authorization != nil {
				if err := s.validatePolicyAuthorization(ctx, rule.Name, rule.Authorization); err != nil {
					return err
				}
			}

		case accessv1.Policy_Spec_Rule_AUTO_APPROVE:
			if rule.Action != nil && rule.Action.GetReview() != nil {
				return grpcutils.InvalidArg("AUTO_APPROVE Rule %s cannot have Review action", rule.Name)
			}
			if rule.Authorization != nil {
				if err := s.validatePolicyAuthorization(ctx, rule.Name, rule.Authorization); err != nil {
					return err
				}
			}

		case accessv1.Policy_Spec_Rule_EFFECT_UNSET:
			return grpcutils.InvalidArg("Rule %s Effect must be set", rule.Name)

		default:
			return grpcutils.InvalidArg("Rule %s has invalid Effect", rule.Name)
		}
	}

	return nil
}

func (s *ServerMain) validatePolicyCondition(ctx context.Context,
	cond *accessv1.Policy_Spec_Rule_Condition, depth int) error {
	const maxDepth = 16

	if cond == nil {
		return grpcutils.InvalidArg("Condition must be set")
	}

	if depth > maxDepth {
		return grpcutils.InvalidArg("Condition is too deeply nested")
	}

	switch cond.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Condition_MatchAny:
		if !cond.GetMatchAny() {
			return grpcutils.InvalidArg("matchAny must be true")
		}

	case *accessv1.Policy_Spec_Rule_Condition_Subject_:
		subject := cond.GetSubject()
		if subject == nil {
			return grpcutils.InvalidArg("Subject condition must be set")
		}

		switch subject.Type.(type) {
		case *accessv1.Policy_Spec_Rule_Condition_Subject_UserRef:
			if err := apivalidation.CheckObjectRef(subject.GetUserRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return err
			}
		case *accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef:
			if err := apivalidation.CheckObjectRef(subject.GetGroupRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return err
			}
		default:
			return grpcutils.InvalidArg("Subject condition type must be set")
		}

	case *accessv1.Policy_Spec_Rule_Condition_Resource_:
		resource := cond.GetResource()
		if resource == nil {
			return grpcutils.InvalidArg("Resource condition must be set")
		}

		switch resource.Type.(type) {
		case *accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef:
			if err := apivalidation.CheckObjectRef(resource.GetServiceRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return err
			}
		case *accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef:
			if err := apivalidation.CheckObjectRef(resource.GetCatalogRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return err
			}
		default:
			return grpcutils.InvalidArg("Resource condition type must be set")
		}

	case *accessv1.Policy_Spec_Rule_Condition_All_:
		all := cond.GetAll()
		if all == nil || len(all.Of) == 0 {
			return grpcutils.InvalidArg("All condition must include at least one child condition")
		}

		for _, sub := range all.Of {
			if err := s.validatePolicyCondition(ctx, sub, depth+1); err != nil {
				return err
			}
		}

	case *accessv1.Policy_Spec_Rule_Condition_Any_:
		any := cond.GetAny()
		if any == nil || len(any.Of) == 0 {
			return grpcutils.InvalidArg("Any condition must include at least one child condition")
		}

		for _, sub := range any.Of {
			if err := s.validatePolicyCondition(ctx, sub, depth+1); err != nil {
				return err
			}
		}

	case *accessv1.Policy_Spec_Rule_Condition_UserRef:
		if err := apivalidation.CheckObjectRef(cond.GetUserRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return err
		}

	case *accessv1.Policy_Spec_Rule_Condition_Match:
		if cond.GetMatch() == "" {
			return grpcutils.InvalidArg("CEL match expression cannot be empty")
		}

	default:
		return grpcutils.InvalidArg("Condition type must be set")
	}

	return nil
}

func (s *ServerMain) validatePolicyReview(ctx context.Context,
	ruleName string, review *accessv1.Policy_Spec_Rule_Action_Review) error {
	if review == nil {
		return grpcutils.InvalidArg("Rule %s Review must be set", ruleName)
	}

	if len(review.Steps) == 0 {
		return grpcutils.InvalidArg("Rule %s Review must include at least one Step", ruleName)
	}

	for idx, step := range review.Steps {
		if step == nil {
			return grpcutils.InvalidArg("Rule %s Review Step %d is nil", ruleName, idx)
		}

		if len(step.Reviewers) == 0 {
			return grpcutils.InvalidArg("Rule %s Review Step %d must include at least one Reviewer", ruleName, idx)
		}

		for _, reviewer := range step.Reviewers {
			if reviewer == nil {
				return grpcutils.InvalidArg("Rule %s Review Step %d has nil Reviewer", ruleName, idx)
			}

			switch reviewer.Type.(type) {
			case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_:
				if err := apivalidation.CheckObjectRef(reviewer.GetUser().GetUserRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
					return err
				}
			case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_:
				if err := apivalidation.CheckObjectRef(reviewer.GetGroup().GetGroupRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
					return err
				}
			default:
				return grpcutils.InvalidArg("Rule %s Review Step %d Reviewer type must be set", ruleName, idx)
			}
		}

		switch step.OnTimeout {
		case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_UNSET,
			accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_GOTO_NEXT_STEP,
			accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_REJECT:
		default:
			return grpcutils.InvalidArg("Rule %s Review Step %d has invalid OnTimeout", ruleName, idx)
		}

		switch step.OnApproval {
		case accessv1.Policy_Spec_Rule_Action_Review_Step_ON_APPROVAL_UNSET,
			accessv1.Policy_Spec_Rule_Action_Review_Step_ON_APPROVAL_APPROVE,
			accessv1.Policy_Spec_Rule_Action_Review_Step_ON_APPROVAL_GOTO_NEXT_STEP:
		default:
			return grpcutils.InvalidArg("Rule %s Review Step %d has invalid OnApproval", ruleName, idx)
		}
	}

	switch review.ApprovalMode {
	case accessv1.Policy_Spec_Rule_Action_Review_APPROVAL_MODE_UNSET,
		accessv1.Policy_Spec_Rule_Action_Review_APPROVAL_MODE_FIRST,
		accessv1.Policy_Spec_Rule_Action_Review_APPROVAL_MODE_ALL_STEPS:
	default:
		return grpcutils.InvalidArg("Rule %s Review has invalid ApprovalMode", ruleName)
	}

	return nil
}

func (s *ServerMain) validatePolicyAuthorization(ctx context.Context,
	ruleName string, authorization *accessv1.Policy_Spec_Rule_Authorization) error {
	if authorization == nil {
		return grpcutils.InvalidArg("Rule %s Authorization must be set", ruleName)
	}

	for _, pol := range authorization.Policies {
		if err := apivalidation.ValidateName(pol, 0, vutils.MaxPolicyParents); err != nil {
			return err
		}

		if _, err := s.octeliumC.CoreC().GetPolicy(ctx, &rmetav1.GetOptions{
			Name: pol,
		}); err != nil {
			return serr.K8sNotFoundOrInternalWithErr(err)
		}
	}

	for _, pol := range authorization.InlinePolicies {
		if pol == nil {
			return grpcutils.InvalidArg("Rule %s Authorization has nil InlinePolicy", ruleName)
		}
	}

	return nil
}
