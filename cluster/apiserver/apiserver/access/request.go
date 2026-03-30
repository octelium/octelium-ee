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
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *ServerUser) CreateRequest(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	{
		_, err := s.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
		if err == nil {
			return nil, grpcutils.AlreadyExists("The Request %s already exists", req.Metadata.Name)
		}
		if !grpcerr.IsNotFound(err) {
			return nil, grpcutils.InternalWithErr(err)
		}
	}

	if err := s.validateRequest(ctx, req); err != nil {
		return nil, err
	}

	item := &accessv1.Request{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &accessv1.Request_Status{},
	}

	item, err := s.octeliumC.AccessC().CreateRequest(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *ServerUser) GetRequest(ctx context.Context, req *metav1.GetOptions) (*accessv1.Request, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
		Uid:  req.Uid,
		Name: req.Name,
	})
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *ServerUser) ListRequest(ctx context.Context, req *accessv1.ListRequestOptions) (*accessv1.RequestList, error) {

	itemList, err := s.octeliumC.AccessC().ListRequest(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerUser) DeleteRequest(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {

	g, err := s.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{Name: req.Name, Uid: req.Uid})
	if err != nil {
		return nil, err
	}

	if g.Metadata.IsSystem {
		return nil, serr.InvalidArg("Cannot delete the system group: %s", req.Name)
	}

	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	_, err = s.octeliumC.AccessC().DeleteRequest(ctx, &rmetav1.DeleteOptions{Uid: g.Metadata.Uid})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerUser) UpdateRequest(ctx context.Context, req *accessv1.Request) (*accessv1.Request, error) {

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

	item, err := s.octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
	if err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	item, err = s.octeliumC.AccessC().UpdateRequest(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerUser) validateRequest(ctx context.Context, req *accessv1.Request) error {
	spec := req.Spec
	if spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	return nil
}
