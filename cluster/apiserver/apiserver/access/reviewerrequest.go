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
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *ServerReviewer) GetRequest(ctx context.Context, req *metav1.GetOptions) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	ok, err := s.canReviewRequest(ctx, i.User, item)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, grpcutils.Unauthorized("You are not allowed to access this Request")
	}

	return item, nil
}

func (s *ServerReviewer) ListRequest(ctx context.Context,
	req *accessv1.ListReviewerRequestOptions) (*accessv1.RequestList, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	listOpts := urscsrv.GetPublicListOptions(req,
		urscsrv.FilterFieldEQValStr("status.state.status", "PENDING"))
	listOpts.OrderBy = []*rmetav1.ListOptions_OrderBy{
		{
			Type: rmetav1.ListOptions_OrderBy_TYPE_CREATED_AT,
			Mode: rmetav1.ListOptions_OrderBy_MODE_DESC,
		},
	}

	itemList, err := s.octeliumC.AccessC().ListRequest(ctx, listOpts)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	ret := &accessv1.RequestList{
		ApiVersion:       itemList.ApiVersion,
		Kind:             itemList.Kind,
		Items:            []*accessv1.Request{},
		ListResponseMeta: itemList.ListResponseMeta,
	}

	for _, item := range itemList.Items {
		ok, err := s.canReviewRequest(ctx, i.User, item)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		ret.Items = append(ret.Items, item)
	}

	if ret.ListResponseMeta == nil {
		ret.ListResponseMeta = &metav1.ListResponseMeta{}
	}
	ret.ListResponseMeta.TotalCount = uint32(len(ret.Items))

	return ret, nil
}

func (s *ServerReviewer) canReviewRequest(ctx context.Context,
	usr *corev1.User, req *accessv1.Request) (bool, error) {
	if req.Status.State == nil ||
		req.Status.State.Status != accessv1.Request_Status_State_PENDING {
		return false, nil
	}

	if req.Status.Rule == nil ||
		req.Status.Rule.Action == nil ||
		req.Status.Rule.Action.GetReview() == nil {
		return false, nil
	}

	actionReview := req.Status.Rule.Action.GetReview()
	if len(actionReview.Steps) == 0 {
		return false, nil
	}

	currentStep := int32(0)
	if req.Status.Review != nil {
		currentStep = req.Status.Review.CurrentStep
	}

	if currentStep < 0 || int(currentStep) >= len(actionReview.Steps) {
		return false, nil
	}

	step := actionReview.Steps[currentStep]

	for _, reviewer := range step.Reviewers {
		if reviewer == nil {
			continue
		}

		switch reviewer.Type.(type) {
		case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_:
			if reviewer.GetUser().GetUserRef() != nil &&
				reviewer.GetUser().GetUserRef().Uid != "" &&
				reviewer.GetUser().GetUserRef().Uid == usr.Metadata.Uid {
				return true, nil
			}

		case *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_:
			ok, err := s.userMatchesReviewerGroup(ctx, usr, reviewer.GetGroup().GetGroupRef())
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *ServerReviewer) userMatchesReviewerGroup(ctx context.Context,
	usr *corev1.User, groupRef *metav1.ObjectReference) (bool, error) {
	if groupRef == nil {
		return false, nil
	}

	grp, err := s.octeliumC.CoreC().GetGroup(ctx, apivalidation.ObjectReferenceToRGetOptions(groupRef))
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return false, nil
		}
		return false, serr.K8sNotFoundOrInternalWithErr(err)
	}

	for _, groupName := range usr.Spec.Groups {
		if groupName == grp.Metadata.Name {
			return true, nil
		}
	}

	return false, nil
}
