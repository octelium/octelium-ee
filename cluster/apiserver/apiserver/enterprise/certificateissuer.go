// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
)

/*
func (s *Server) CreateCertificateIssuer(ctx context.Context, req *enterprisev1.CertificateIssuer) (*enterprisev1.CertificateIssuer, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	{
		_, err := s.octeliumC.EnterpriseC().GetCertificateIssuer(ctx, apivalidation.ObjectToRGetOptions(req))
		if err == nil {
			return nil, grpcutils.AlreadyExists("The CertificateIssuer %s already exists", req.Metadata.Name)
		}
		if !grpcerr.IsNotFound(err) {
			return nil, grpcutils.InternalWithErr(err)
		}
	}

	if err := s.validateCertificateIssuer(ctx, req); err != nil {
		return nil, err
	}

	item := &enterprisev1.CertificateIssuer{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &enterprisev1.CertificateIssuer_Status{},
	}

	item, err := s.octeliumC.EnterpriseC().CreateCertificateIssuer(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}
*/

func (s *Server) GetCertificateIssuer(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.CertificateIssuer, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetCertificateIssuer(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) ListCertificateIssuer(ctx context.Context, req *enterprisev1.ListCertificateIssuerOptions) (*enterprisev1.CertificateIssuerList, error) {

	itemList, err := s.octeliumC.EnterpriseC().ListCertificateIssuer(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

/*
func (s *Server) DeleteCertificateIssuer(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {

	g, err := s.octeliumC.EnterpriseC().GetCertificateIssuer(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, err
	}

	if g.Metadata.IsSystem {
		return nil, serr.InvalidArg("Cannot delete the system group: %s", req.Name)
	}

	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	_, err = s.octeliumC.EnterpriseC().DeleteCertificateIssuer(ctx, &rmetav1.DeleteOptions{Uid: g.Metadata.Uid})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

*/

func (s *Server) UpdateCertificateIssuer(ctx context.Context, req *enterprisev1.CertificateIssuer) (*enterprisev1.CertificateIssuer, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if err := s.validateCertificateIssuer(ctx, req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetCertificateIssuer(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	item, err = s.octeliumC.EnterpriseC().UpdateCertificateIssuer(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *Server) validateCertificateIssuer(ctx context.Context, req *enterprisev1.CertificateIssuer) error {
	spec := req.Spec
	if spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	return nil
}
