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
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
)

func (s *ServerUser) CreateRequest(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{},
	}); err != nil {
		return nil, err
	}

	name, err := s.generateRequestName(ctx)
	if err != nil {
		return nil, err
	}

	spec := pbutils.Clone(req.Spec).(*accessv1.Request_Spec)
	if spec.Subject == nil || spec.Subject.GetUserRef() == nil {
		spec.Subject = &accessv1.Request_Spec_Subject{
			Type: &accessv1.Request_Spec_Subject_UserRef{
				UserRef: umetav1.GetObjectReference(i.User),
			},
		}
	}

	item := &accessv1.Request{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     spec,
		Status: &accessv1.Request_Status{
			UserRef: umetav1.GetObjectReference(i.User),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_PENDING,
			},
		},
	}

	item.Metadata.Name = name

	if err := s.validateUserRequest(ctx, item); err != nil {
		return nil, err
	}

	item, err = s.octeliumC.AccessC().CreateRequest(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *ServerUser) GetRequest(ctx context.Context, req *metav1.GetOptions) (*accessv1.Request, error) {
	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
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

	itemList, err := s.octeliumC.AccessC().ListRequest(ctx,
		urscsrv.GetPublicListOptions(req, urscsrv.FilterStatusUserUID(i.User.Metadata.Uid)))
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

	if err := s.validateUserRequest(ctx, req); err != nil {
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

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
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

	setUserRequestState(item, accessv1.Request_Status_State_CANCELLED)

	if _, err := s.octeliumC.AccessC().UpdateRequest(ctx, item); err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerUser) validateUserRequest(ctx context.Context, req *accessv1.Request) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	if req.Spec.Resource == nil {
		return grpcutils.InvalidArg("Resource must be set")
	}

	switch req.Spec.Resource.Type.(type) {
	case *accessv1.Request_Spec_Resource_ServiceRef:
		if err := s.validateUserRequestService(ctx, req.Spec.Resource.GetServiceRef()); err != nil {
			return err
		}

	case *accessv1.Request_Spec_Resource_Catalog_:
		if req.Spec.Resource.GetCatalog() == nil {
			return grpcutils.InvalidArg("Catalog resource must be set")
		}
		if err := s.validateUserRequestCatalog(ctx, req.Spec.Resource.GetCatalog().CatalogRef); err != nil {
			return err
		}

	default:
		return grpcutils.InvalidArg("Resource type must be set")
	}

	if req.Spec.Subject != nil {
		switch req.Spec.Subject.Type.(type) {
		case *accessv1.Request_Spec_Subject_UserRef:
			if err := s.validateUserRequestSubjectUser(ctx, req.Spec.Subject.GetUserRef()); err != nil {
				return err
			}

		default:
			return grpcutils.InvalidArg("Subject type must be set")
		}
	}

	switch req.Spec.Urgency {
	case accessv1.Request_Spec_URGENCY_UNSET,
		accessv1.Request_Spec_VERY_LOW,
		accessv1.Request_Spec_LOW,
		accessv1.Request_Spec_NORMAL,
		accessv1.Request_Spec_HIGH,
		accessv1.Request_Spec_VERY_HIGH,
		accessv1.Request_Spec_HIGHEST:
	default:
		return grpcutils.InvalidArg("Invalid Urgency")
	}

	return nil
}

func (s *ServerUser) validateUserRequestService(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}

	_, err := s.octeliumC.CoreC().GetService(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		return serr.K8sNotFoundOrInternalWithErr(err)
	}

	return nil
}

func (s *ServerUser) validateUserRequestCatalog(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}

	_, err := s.octeliumC.AccessC().GetCatalog(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		return serr.K8sNotFoundOrInternalWithErr(err)
	}

	return nil
}

func (s *ServerUser) validateUserRequestSubjectUser(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}

	_, err := s.octeliumC.CoreC().GetUser(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	if err != nil {
		return serr.K8sNotFoundOrInternalWithErr(err)
	}

	return nil
}

func (s *ServerUser) generateRequestName(ctx context.Context) (string, error) {
	const attemptsPerLength = 32

	for n := 3; n <= 8; n++ {
		for i := 0; i < attemptsPerLength; i++ {
			name := utilrand.GetRandomStringCanonical(n)

			_, err := s.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
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

	return "", grpcutils.Internal("Could not generate a unique Request name")
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

func setUserRequestState(req *accessv1.Request, status accessv1.Request_Status_State_Status) {
	now := pbutils.Now()

	if req.Status.State != nil && req.Status.State.Status == status {
		return
	}

	if req.Status.State != nil &&
		req.Status.State.Status != accessv1.Request_Status_State_STATUS_UNKNOWN {
		req.Status.LastStates = append(req.Status.LastStates,
			pbutils.Clone(req.Status.State).(*accessv1.Request_Status_State))
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
