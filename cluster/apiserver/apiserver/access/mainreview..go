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
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
)

func (s *ServerMain) GetReview(ctx context.Context, req *metav1.GetOptions) (*accessv1.Review, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetReview(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListReview(ctx context.Context, req *accessv1.ListReviewOptions) (*accessv1.ReviewList, error) {
	itemList, err := s.octeliumC.AccessC().ListReview(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) DeleteReview(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetReview(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	_, err = s.octeliumC.AccessC().DeleteReview(ctx, &rmetav1.DeleteOptions{
		Uid: item.Metadata.Uid,
	})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerMain) validateReview(ctx context.Context, req *accessv1.Review) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	if req.Status == nil {
		return grpcutils.InvalidArg("Nil Status")
	}

	switch req.Spec.Decision {
	case accessv1.Review_Spec_DECISION_UNSET,
		accessv1.Review_Spec_DECISION_APPROVE,
		accessv1.Review_Spec_DECISION_REJECT:
	default:
		return grpcutils.InvalidArg("Invalid Decision")
	}

	if req.Status.UserRef != nil {
		if err := apivalidation.CheckObjectRef(req.Status.UserRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return err
		}
	}

	if req.Status.RequestRef != nil {
		if err := apivalidation.CheckObjectRef(req.Status.RequestRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return err
		}
	}

	return nil
}
