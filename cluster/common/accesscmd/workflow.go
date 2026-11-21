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

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func BuildWorkflow(ctx context.Context, octeliumC octeliumc.ClientInterface,
	rule *accessv1.Policy_Spec_Rule) (*accessv1.Request_Status_Workflow, error) {
	if rule == nil {
		return nil, nil
	}

	ret := &accessv1.Request_Status_Workflow{
		Effect:  rule.Effect,
		Message: rule.Message,
	}

	review := rule.GetAction().GetReview()
	if rule.Effect != accessv1.Policy_Spec_Rule_REVIEW || review == nil {
		return ret, nil
	}

	for idx, step := range review.Steps {
		if step == nil {
			continue
		}

		wStep := &accessv1.Request_Status_Workflow_Step{
			Name:                StepName(step, int32(idx)),
			ApprovalRequirement: step.ApprovalRequirement,
			ApprovalCount:       step.ApprovalCount,
			Timeout:             step.Timeout,
			OnTimeout:           step.OnTimeout,
			ReasonRequirement:   step.ReasonRequirement,
			HasEscalation:       step.Escalation != nil && len(step.Escalation.Reviewers) > 0,
		}

		for _, reviewer := range step.Reviewers {
			itm, err := summarizeReviewer(ctx, octeliumC, reviewer)
			if err != nil {
				return nil, err
			}
			if itm != nil {
				wStep.Reviewers = append(wStep.Reviewers, itm)
			}
		}

		ret.Steps = append(ret.Steps, wStep)
	}

	return ret, nil
}

func summarizeReviewer(ctx context.Context, octeliumC octeliumc.ClientInterface,
	reviewer *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer) (*accessv1.Request_Status_Workflow_Step_Reviewer, error) {
	if reviewer == nil {
		return nil, nil
	}

	switch reviewer.Type.(type) {
	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_:
		ref := reviewer.GetUser().GetUserRef()
		usr, err := GetUser(ctx, octeliumC, ref)
		if err != nil {
			return nil, err
		}

		ret := &accessv1.Request_Status_Workflow_Step_Reviewer{
			Type: accessv1.Request_Status_Workflow_Step_Reviewer_USER,
		}

		switch {
		case usr != nil:
			ret.DisplayName = GetUserDisplayName(usr)
		case ref != nil:
			ret.DisplayName = ref.Name
		}

		return ret, nil

	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_:
		ref := reviewer.GetGroup().GetGroupRef()
		grp, err := GetGroup(ctx, octeliumC, ref)
		if err != nil {
			return nil, err
		}

		ret := &accessv1.Request_Status_Workflow_Step_Reviewer{
			Type: accessv1.Request_Status_Workflow_Step_Reviewer_GROUP,
		}

		switch {
		case grp != nil && grp.Metadata.DisplayName != "":
			ret.DisplayName = grp.Metadata.DisplayName
		case grp != nil:
			ret.DisplayName = grp.Metadata.Name
		case ref != nil:
			ret.DisplayName = ref.Name
		}

		return ret, nil

	case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners_:
		return &accessv1.Request_Status_Workflow_Step_Reviewer{
			Type:        accessv1.Request_Status_Workflow_Step_Reviewer_CATALOG_OWNERS,
			DisplayName: "Catalog owners",
		}, nil

	default:
		return nil, nil
	}
}

func GetUserDisplayName(usr *corev1.User) string {
	if usr == nil || usr.Metadata == nil {
		return ""
	}

	if usr.Metadata.DisplayName != "" {
		return usr.Metadata.DisplayName
	}

	return usr.Metadata.Name
}

func GetUserInfo(usr *corev1.User) *accessv1.UserInfo {
	if usr == nil || usr.Metadata == nil {
		return nil
	}

	ret := &accessv1.UserInfo{
		UserRef:     objectRef(usr),
		DisplayName: GetUserDisplayName(usr),
		PicURL:      usr.Metadata.PicURL,
	}

	if usr.Spec != nil {
		ret.Email = usr.Spec.Email
		ret.Type = usr.Spec.Type
	}

	return ret
}

func RedactAccessLevel(level *accessv1.Catalog_Spec_AccessLevel) *accessv1.Catalog_Spec_AccessLevel {
	if level == nil {
		return nil
	}

	ret := pbutils.Clone(level).(*accessv1.Catalog_Spec_AccessLevel)
	if _, ok := ret.Type.(*accessv1.Catalog_Spec_AccessLevel_Restricted_); ok {
		ret.Type = &accessv1.Catalog_Spec_AccessLevel_Restricted_{
			Restricted: &accessv1.Catalog_Spec_AccessLevel_Restricted{},
		}
	}

	return ret
}

func RedactRequest(req *accessv1.Request) *accessv1.Request {
	if req == nil {
		return nil
	}

	ret := pbutils.Clone(req).(*accessv1.Request)
	if ret.Status == nil {
		return ret
	}

	ret.Status.Rule = nil
	ret.Status.PolicyRef = nil
	ret.Status.PolicyTriggerRef = nil

	if ret.Status.ResolvedScope != nil {
		ret.Status.ResolvedScope.AccessLevel = RedactAccessLevel(ret.Status.ResolvedScope.AccessLevel)
	}

	if ret.Status.Evaluation != nil {
		ret.Status.Evaluation.SkippedRules = nil
		ret.Status.Evaluation.LastError = ""
	}

	if ret.Status.Grant != nil {
		ret.Status.Grant.LastError = ""
	}

	return ret
}

func RedactRequestList(list *accessv1.RequestList) *accessv1.RequestList {
	if list == nil {
		return nil
	}

	ret := &accessv1.RequestList{
		ApiVersion:       list.ApiVersion,
		Kind:             list.Kind,
		ListResponseMeta: list.ListResponseMeta,
	}

	for _, itm := range list.Items {
		ret.Items = append(ret.Items, RedactRequest(itm))
	}

	return ret
}
