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
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/userctx"
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
	return accesscmd.CanReview(ctx, s.octeliumC, usr, req)
}
