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
	"fmt"

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
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

	if err := apivalidation.CheckObjectRef(req.Status.RequestRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	request, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectReferenceToRGetOptions(req.Status.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	ok, err := s.canReviewRequest(ctx, i.User, request)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, grpcutils.Unauthorized("You are not allowed to review this Request")
	}

	if err := s.ensureNoExistingReviewerReview(ctx, i.User.Metadata.Uid, request.Metadata.Uid); err != nil {
		return nil, err
	}

	name, err := s.generateReviewName(ctx, request.Metadata.Name)
	if err != nil {
		return nil, err
	}

	metadata := apisrvcommon.MetadataFrom(req.Metadata)
	metadata.Name = name

	item := &accessv1.Review{
		Metadata: metadata,
		Spec:     pbutils.Clone(req.Spec).(*accessv1.Review_Spec),
		Status: &accessv1.Review_Status{
			UserRef:    umetav1.GetObjectReference(i.User),
			RequestRef: umetav1.GetObjectReference(request),
			SetAt:      pbutils.Now(),
			StepIndex:  currentReviewStep(request),
		},
	}

	item, err = s.octeliumC.AccessC().CreateReview(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
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

	ok, err := s.canReviewRequest(ctx, i.User, request)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, grpcutils.InvalidArg("This Review can no longer be updated")
	}

	if hasReviewerReviewBeenApplied(request, item) {
		return nil, grpcutils.InvalidArg("Applied Reviews cannot be updated")
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)

	if !pbutils.IsEqual(item.Spec, req.Spec) {
		appendReviewerReviewRevision(item)
		item.Spec = pbutils.Clone(req.Spec).(*accessv1.Review_Spec)
		item.Status.SetAt = pbutils.Now()
	}

	item, err = s.octeliumC.AccessC().UpdateReview(ctx, item)
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

	item, err := s.octeliumC.AccessC().GetReview(ctx, apivalidation.ObjectReferenceToRGetOptions(req.ReviewRef))
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

	if hasReviewerReviewBeenApplied(request, item) {
		return nil, grpcutils.InvalidArg("Applied Reviews cannot be cancelled")
	}

	if item.Spec.Decision == accessv1.Review_Spec_DECISION_UNSET {
		return &metav1.OperationResult{}, nil
	}

	appendReviewerReviewRevision(item)

	item.Spec.Decision = accessv1.Review_Spec_DECISION_UNSET
	item.Spec.Justification = ""
	item.Status.SetAt = pbutils.Now()

	if _, err := s.octeliumC.AccessC().UpdateReview(ctx, item); err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerReviewer) getReviewerReviewRequest(ctx context.Context, review *accessv1.Review) (*accessv1.Request, error) {
	if review.Status.RequestRef == nil {
		return nil, grpcutils.InvalidArg("Review has no RequestRef")
	}

	request, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectReferenceToRGetOptions(review.Status.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return request, nil
}

func (s *ServerReviewer) ensureNoExistingReviewerReview(ctx context.Context, userUID, requestUID string) error {
	itemList, err := s.octeliumC.AccessC().ListReview(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterStatusUserUID(userUID),
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", requestUID),
		},
	})
	if err != nil {
		return serr.InternalWithErr(err)
	}

	if len(itemList.Items) > 0 {
		return grpcutils.AlreadyExists("You already have a Review for this Request")
	}

	return nil
}

func (s *ServerReviewer) generateReviewName(ctx context.Context, requestName string) (string, error) {
	const attemptsPerLength = 32

	for n := 3; n <= 8; n++ {
		for i := 0; i < attemptsPerLength; i++ {
			name := fmt.Sprintf("%s.%s", utilrand.GetRandomStringCanonical(n), requestName)

			_, err := s.octeliumC.AccessC().GetReview(ctx, &rmetav1.GetOptions{
				Name: name,
			})
			if err == nil {
				continue
			}

			if grpcerr.IsNotFound(err) {
				return name, nil
			}

			return "", serr.InternalWithErr(err)
		}
	}

	return "", grpcutils.Internal("Could not generate a unique Review name")
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

	return nil
}

func checkReviewerOwnsReview(userUID string, review *accessv1.Review) error {
	if review.Status.UserRef == nil || review.Status.UserRef.Uid != userUID {
		return grpcutils.Unauthorized("You are not allowed to access this Review")
	}

	return nil
}

func hasReviewerReviewBeenApplied(req *accessv1.Request, review *accessv1.Review) bool {
	if req.Status.Review == nil {
		return false
	}

	for _, step := range req.Status.Review.LastSteps {
		if step.ReviewRef != nil &&
			step.ReviewRef.Uid != "" &&
			step.ReviewRef.Uid == review.Metadata.Uid {
			return true
		}
	}

	return false
}

const maxReviewRevisions = 100

func appendReviewerReviewRevision(review *accessv1.Review) {
	revisionSetAt := review.Status.SetAt
	if revisionSetAt == nil {
		revisionSetAt = pbutils.Now()
	}

	if review.Spec != nil {
		revision := &accessv1.Review_Status_Revision{
			Spec:  pbutils.Clone(review.Spec).(*accessv1.Review_Spec),
			SetAt: revisionSetAt,
		}

		review.Status.LastRevisions = append(
			[]*accessv1.Review_Status_Revision{revision},
			review.Status.LastRevisions...,
		)

		if len(review.Status.LastRevisions) > maxReviewRevisions {
			review.Status.LastRevisions = review.Status.LastRevisions[:maxReviewRevisions]
		}
	}

	if review.Status.SetAt != nil {
		review.Status.LastSetsAt = append(
			[]*timestamppb.Timestamp{review.Status.SetAt},
			review.Status.LastSetsAt...,
		)

		if len(review.Status.LastSetsAt) > maxReviewRevisions {
			review.Status.LastSetsAt = review.Status.LastSetsAt[:maxReviewRevisions]
		}
	}
}
