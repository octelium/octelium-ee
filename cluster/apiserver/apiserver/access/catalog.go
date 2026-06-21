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

func (s *ServerMain) CreateCatalog(ctx context.Context, req *accessv1.Catalog) (*accessv1.Catalog, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.AccessC().GetCatalog(ctx, apivalidation.ObjectToRGetOptions(req))
	if err == nil {
		return nil, grpcutils.AlreadyExists("The Catalog %s already exists", req.Metadata.Name)
	}
	if !grpcerr.IsNotFound(err) {
		return nil, grpcutils.InternalWithErr(err)
	}

	if err := s.validateCatalog(ctx, req); err != nil {
		return nil, err
	}

	item := &accessv1.Catalog{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &accessv1.Catalog_Status{},
	}

	item, err = s.octeliumC.AccessC().CreateCatalog(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) GetCatalog(ctx context.Context, req *metav1.GetOptions) (*accessv1.Catalog, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetCatalog(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return item, nil
}

func (s *ServerMain) ListCatalog(ctx context.Context, req *accessv1.ListCatalogOptions) (*accessv1.CatalogList, error) {
	itemList, err := s.octeliumC.AccessC().ListCatalog(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *ServerMain) DeleteCatalog(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	item, err := s.octeliumC.AccessC().GetCatalog(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	_, err = s.octeliumC.AccessC().DeleteCatalog(ctx, &rmetav1.DeleteOptions{
		Uid: item.Metadata.Uid,
	})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *ServerMain) UpdateCatalog(ctx context.Context, req *accessv1.Catalog) (*accessv1.Catalog, error) {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if err := s.validateCatalog(ctx, req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.AccessC().GetCatalog(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	item, err = s.octeliumC.AccessC().UpdateCatalog(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *ServerMain) validateCatalog(ctx context.Context, req *accessv1.Catalog) error {
	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	if req.Spec.ResourceCollection == nil {
		return grpcutils.InvalidArg("ResourceCollection must be set")
	}

	if req.Spec.ResourceCollection.Service == nil {
		return grpcutils.InvalidArg("ResourceCollection Service must be set")
	}

	if len(req.Spec.ResourceCollection.Service.Services) == 0 &&
		len(req.Spec.ResourceCollection.Service.Namespaces) == 0 {
		return grpcutils.InvalidArg("ResourceCollection Service must include at least one Service or Namespace")
	}

	for _, svc := range req.Spec.ResourceCollection.Service.Services {
		if svc == "" {
			return grpcutils.InvalidArg("Service name cannot be empty")
		}
	}

	for _, ns := range req.Spec.ResourceCollection.Service.Namespaces {
		if ns == "" {
			return grpcutils.InvalidArg("Namespace name cannot be empty")
		}
	}

	return nil
}
