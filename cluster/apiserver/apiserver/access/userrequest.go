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

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func (s *ServerUser) CreateRequest(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {
	return s.doCreateRequest(ctx, req, false)
}

func (s *ServerUser) CreateRequestForSubject(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {
	return s.doCreateRequest(ctx, req, true)
}

func (s *ServerUser) doCreateRequest(ctx context.Context, req *accessv1.Request, forSubject bool) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{},
	}); err != nil {
		return nil, err
	}

	return accesscmd.CreateRequest(ctx, &accesscmd.CreateRequestOpts{
		OcteliumC:  s.octeliumC,
		Requester:  i.User,
		Spec:       req.Spec,
		ForSubject: forSubject,
		Origin:     apiOrigin(i),
	})
}

func (s *ServerUser) GetRequest(ctx context.Context, req *metav1.GetOptions) (*accessv1.Request, error) {
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

	if err := checkUserOwnsRequest(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *ServerUser) ListRequest(ctx context.Context, req *accessv1.ListUserRequestOptions) (*accessv1.RequestList, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	filters := []*rmetav1.ListOptions_Filter{
		urscsrv.FilterStatusUserUID(i.User.Metadata.Uid),
	}

	if req.State != accessv1.Request_Status_State_STATUS_UNKNOWN {
		filters = append(filters,
			urscsrv.FilterFieldEQValStr("status.state.status", req.State.String()))
	}

	if req.Urgency != accessv1.Request_Spec_URGENCY_UNSET {
		filters = append(filters,
			urscsrv.FilterFieldEQValStr("spec.urgency", req.Urgency.String()))
	}

	filters, err = appendRefFilter(filters, "spec.resource.serviceRef", req.ServiceRef,
		&apivalidation.CheckGetOptionsOpts{
			ParentsMax: 1,
		})
	if err != nil {
		return nil, err
	}

	filters, err = appendRefFilter(filters, "spec.resource.catalog.catalogRef", req.CatalogRef,
		&apivalidation.CheckGetOptionsOpts{})
	if err != nil {
		return nil, err
	}

	itemList, err := s.octeliumC.AccessC().ListRequest(ctx,
		withDefaultOrder(urscsrv.GetPublicListOptions(req, filters...)))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerUser) UpdateRequest(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if err := accesscmd.ValidateRequestSpec(ctx, s.octeliumC, req.Spec); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := checkUserOwnsRequest(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	if err := checkUserRequestPending(item); err != nil {
		return nil, err
	}

	if !pbutils.IsEqual(item.Spec.Resource, req.Spec.Resource) {
		return nil, grpcutils.InvalidArg("Cannot change Request resource")
	}

	if !pbutils.IsEqual(item.Spec.Subject, req.Spec.Subject) {
		return nil, grpcutils.InvalidArg("Cannot change Request subject")
	}

	item.Spec.Urgency = req.Spec.Urgency
	item.Spec.Justification = req.Spec.Justification
	item.Spec.Deadline = req.Spec.Deadline
	item.Spec.Duration = req.Spec.Duration

	item, err = s.octeliumC.AccessC().UpdateRequest(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerUser) CancelRequest(ctx context.Context, req *accessv1.CancelRequestRequest) (*metav1.OperationResult, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil || req.RequestRef == nil {
		return nil, grpcutils.InvalidArg("RequestRef must be set")
	}

	if err := apivalidation.CheckObjectRef(req.RequestRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectReferenceToRGetOptions(req.RequestRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := checkUserOwnsRequest(i.User.Metadata.Uid, item); err != nil {
		return nil, err
	}

	if item.Status.State != nil &&
		item.Status.State.Status == accessv1.Request_Status_State_CANCELLED {
		return &metav1.OperationResult{}, nil
	}

	if err := checkUserRequestPending(item); err != nil {
		return nil, err
	}

	setRequestState(item, accessv1.Request_Status_State_CANCELLED)

	if _, err := s.octeliumC.AccessC().UpdateRequest(ctx, item); err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return &metav1.OperationResult{}, nil
}

func appendRefFilter(filters []*rmetav1.ListOptions_Filter,
	field string, ref *metav1.ObjectReference,
	opts *apivalidation.CheckGetOptionsOpts) ([]*rmetav1.ListOptions_Filter, error) {
	if ref == nil || (ref.Uid == "" && ref.Name == "") {
		return filters, nil
	}

	if err := apivalidation.CheckObjectRef(ref, opts); err != nil {
		return nil, err
	}

	if ref.Uid != "" {
		return append(filters, urscsrv.FilterFieldEQValStr(fmt.Sprintf("%s.uid", field), ref.Uid)), nil
	}

	return append(filters, urscsrv.FilterFieldEQValStr(fmt.Sprintf("%s.name", field), ref.Name)), nil
}

func withDefaultOrder(listOpts *rmetav1.ListOptions) *rmetav1.ListOptions {
	if len(listOpts.OrderBy) > 0 {
		return listOpts
	}

	listOpts.OrderBy = []*rmetav1.ListOptions_OrderBy{
		{
			Type: rmetav1.ListOptions_OrderBy_TYPE_CREATED_AT,
			Mode: rmetav1.ListOptions_OrderBy_MODE_DESC,
		},
	}

	return listOpts
}

func checkUserOwnsRequest(userUID string, req *accessv1.Request) error {
	if req.Status.UserRef == nil || req.Status.UserRef.Uid != userUID {
		return grpcutils.Unauthorized("You are not allowed to access this Request")
	}

	return nil
}

func checkUserRequestPending(req *accessv1.Request) error {
	if req.Status.State == nil ||
		req.Status.State.Status == accessv1.Request_Status_State_STATUS_UNKNOWN ||
		req.Status.State.Status == accessv1.Request_Status_State_PENDING {
		return nil
	}

	return grpcutils.InvalidArg("Only pending Requests can be updated or cancelled")
}

const maxRequestStates = 100

func setRequestState(req *accessv1.Request, status accessv1.Request_Status_State_Status) {
	now := pbutils.Now()

	if req.Status.State != nil && req.Status.State.Status == status {
		return
	}

	if req.Status.State != nil &&
		req.Status.State.Status != accessv1.Request_Status_State_STATUS_UNKNOWN {
		prevState := pbutils.Clone(req.Status.State).(*accessv1.Request_Status_State)

		req.Status.LastStates = append(
			[]*accessv1.Request_Status_State{prevState},
			req.Status.LastStates...,
		)

		if len(req.Status.LastStates) > maxRequestStates {
			req.Status.LastStates = req.Status.LastStates[:maxRequestStates]
		}
	}

	req.Status.State = &accessv1.Request_Status_State{
		CreatedAt: now,
		Status:    status,
	}

	switch status {
	case accessv1.Request_Status_State_APPROVED,
		accessv1.Request_Status_State_REJECTED,
		accessv1.Request_Status_State_REVOKED,
		accessv1.Request_Status_State_EXPIRED,
		accessv1.Request_Status_State_CANCELLED:
		if req.Status.ApprovalEndAt == nil {
			req.Status.ApprovalEndAt = now
		}
	}
}
