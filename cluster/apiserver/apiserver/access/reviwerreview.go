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

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func (s *ServerReviewer) SetReviewDecision(ctx context.Context,
	req *accessv1.SetReviewDecisionRequest) (*accessv1.Review, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil || req.RequestRef == nil {
		return nil, grpcutils.InvalidArg("RequestRef must be set")
	}

	if err := apivalidation.CheckObjectRef(req.RequestRef,
		&apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	request, err := s.octeliumC.AccessC().GetRequest(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return accesscmd.SetReviewDecision(ctx, &accesscmd.SetReviewDecisionOpts{
		OcteliumC:        s.octeliumC,
		Reviewer:         i.User,
		Request:          request,
		Decision:         req.Decision,
		Justification:    req.Justification,
		ExpectedStepName: req.ExpectedStepName,
		Origin:           apiOrigin(i),
	})
}

func (s *ServerReviewer) CreateReview(ctx context.Context, req *accessv1.Review) (*accessv1.Review, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{},
		RequireStatus:        true,
	}); err != nil {
		return nil, err
	}

	if err := validateReviewerReviewSpec(req); err != nil {
		return nil, err
	}

	if req.Status.RequestRef == nil {
		return nil, grpcutils.InvalidArg("RequestRef must be set")
	}

	if err := apivalidation.CheckObjectRef(req.Status.RequestRef,
		&apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	request, err := s.octeliumC.AccessC().GetRequest(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.Status.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	step, stepIndex, err := accesscmd.CurrentStep(request)
	if err != nil {
		return nil, err
	}

	existing, err := accesscmd.GetStepReview(ctx, s.octeliumC, request, stepIndex,
		i.User.Metadata.Uid)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, grpcutils.AlreadyExists(
			"You already have a Review for this Request at the %s review Step", step.Name)
	}

	return accesscmd.SetReviewDecision(ctx, &accesscmd.SetReviewDecisionOpts{
		OcteliumC:        s.octeliumC,
		Reviewer:         i.User,
		Request:          request,
		Decision:         req.Spec.Decision,
		Justification:    req.Spec.Justification,
		ExpectedStepName: step.Name,
		Origin:           apiOrigin(i),
	})
}

func (s *ServerReviewer) GetReview(ctx context.Context, req *metav1.GetOptions) (*accessv1.Review, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetReview(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := checkReviewerOwnsReview(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *ServerReviewer) ListReview(ctx context.Context, req *accessv1.ListReviewerReviewOptions) (*accessv1.ReviewList, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	listOpts := urscsrv.GetPublicListOptions(req, urscsrv.FilterStatusUserUID(i.User.Metadata.Uid))
	listOpts.OrderBy = []*rmetav1.ListOptions_OrderBy{
		{
			Type: rmetav1.ListOptions_OrderBy_TYPE_CREATED_AT,
			Mode: rmetav1.ListOptions_OrderBy_MODE_DESC,
		},
	}

	itemList, err := s.octeliumC.AccessC().ListReview(ctx, listOpts)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerReviewer) UpdateReview(ctx context.Context, req *accessv1.Review) (*accessv1.Review, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
			ParentsMust: 1,
		},
	}); err != nil {
		return nil, err
	}

	if err := validateReviewerReviewSpec(req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetReview(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := checkReviewerOwnsReview(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	request, err := s.getReviewerReviewRequest(ctx, item)
	if err != nil {
		return nil, err
	}

	item, err = accesscmd.SetReviewDecision(ctx, &accesscmd.SetReviewDecisionOpts{
		OcteliumC:        s.octeliumC,
		Reviewer:         i.User,
		Request:          request,
		Decision:         req.Spec.Decision,
		Justification:    req.Spec.Justification,
		ExpectedStepName: item.Status.StepName,
		Origin:           apiOrigin(i),
	})
	if err != nil {
		return nil, err
	}

	next := pbutils.Clone(item).(*accessv1.Review)
	apisrvcommon.MetadataUpdate(next.Metadata, req.Metadata)

	if pbutils.IsEqual(item, next) {
		return item, nil
	}

	item, err = s.octeliumC.AccessC().UpdateReview(ctx, next)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerReviewer) CancelReview(ctx context.Context, req *accessv1.CancelReviewRequest) (*metav1.OperationResult, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil || req.ReviewRef == nil {
		return nil, grpcutils.InvalidArg("ReviewRef must be set")
	}

	if err := apivalidation.CheckObjectRef(req.ReviewRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetReview(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.ReviewRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := checkReviewerOwnsReview(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	if item.Spec.Decision == accessv1.Review_Spec_DECISION_UNSET {
		return &metav1.OperationResult{}, nil
	}

	request, err := s.getReviewerReviewRequest(ctx, item)
	if err != nil {
		return nil, err
	}

	if _, err := accesscmd.SetReviewDecision(ctx, &accesscmd.SetReviewDecisionOpts{
		OcteliumC:        s.octeliumC,
		Reviewer:         i.User,
		Request:          request,
		Decision:         accessv1.Review_Spec_DECISION_UNSET,
		ExpectedStepName: item.Status.StepName,
		Origin:           apiOrigin(i),
	}); err != nil {
		return nil, err
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerReviewer) getReviewerReviewRequest(ctx context.Context, review *accessv1.Review) (*accessv1.Request, error) {
	if review.Status.RequestRef == nil {
		return nil, grpcutils.InvalidArg("Review has no RequestRef")
	}

	request, err := s.octeliumC.AccessC().GetRequest(ctx,
		apivalidation.ObjectReferenceToRGetOptions(review.Status.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return request, nil
}

func validateReviewerReviewSpec(req *accessv1.Review) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	switch req.Spec.Decision {
	case accessv1.Review_Spec_DECISION_APPROVE,
		accessv1.Review_Spec_DECISION_REJECT:
	default:
		return grpcutils.InvalidArg("Decision must be APPROVE or REJECT")
	}

	if len(req.Spec.Justification) > accesscmd.MaxJustificationLen {
		return grpcutils.InvalidArg("Justification is too long")
	}

	return nil
}

func checkReviewerOwnsReview(userUID string, review *accessv1.Review) error {
	if review.Status.UserRef == nil || review.Status.UserRef.Uid != userUID {
		return grpcutils.Unauthorized("You are not allowed to access this Review")
	}

	return nil
}

func apiOrigin(i *userctx.UserCtx) *accessv1.Origin {
	ret := &accessv1.Origin{
		Type: accessv1.Origin_API,
	}

	if i.Session != nil {
		ret.SessionRef = umetav1.GetObjectReference(i.Session)
	}

	return ret
}
