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
)

func (s *ServerMain) GetRequest(ctx context.Context, req *metav1.GetOptions) (*accessv1.Request, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListRequest(ctx context.Context, req *accessv1.ListRequestOptions) (*accessv1.RequestList, error) {
	itemList, err := s.octeliumC.AccessC().ListRequest(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) DeleteRequest(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	_, err = s.octeliumC.AccessC().DeleteRequest(ctx, &rmetav1.DeleteOptions{
		Uid: item.Metadata.Uid,
	})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerMain) UpdateRequest(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if err := s.validateRequest(ctx, req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetRequest(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	item.Spec = req.Spec

	item, err = s.octeliumC.AccessC().UpdateRequest(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerMain) validateRequest(ctx context.Context, req *accessv1.Request) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	if req.Status == nil {
		return grpcutils.InvalidArg("Nil Status")
	}

	if req.Spec.Resource == nil {
		return grpcutils.InvalidArg("Resource must be set")
	}

	switch req.Spec.Resource.Type.(type) {
	case *accessv1.Request_Spec_Resource_ServiceRef:
		if err := apivalidation.CheckObjectRef(req.Spec.Resource.GetServiceRef(), &apivalidation.CheckGetOptionsOpts{
			ParentsMax: 1,
		}); err != nil {
			return err
		}
	case *accessv1.Request_Spec_Resource_Catalog_:
		if req.Spec.Resource.GetCatalog() == nil {
			return grpcutils.InvalidArg("Catalog resource must be set")
		}
		if err := apivalidation.CheckObjectRef(req.Spec.Resource.GetCatalog().CatalogRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return err
		}
	default:
		return grpcutils.InvalidArg("Resource type must be set")
	}

	if req.Spec.Subject != nil {
		switch req.Spec.Subject.Type.(type) {
		case *accessv1.Request_Spec_Subject_UserRef:
			if err := apivalidation.CheckObjectRef(req.Spec.Subject.GetUserRef(), &apivalidation.CheckGetOptionsOpts{}); err != nil {
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
