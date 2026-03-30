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

	"github.com/octelium/octelium-ee/cluster/common/certutils"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *Server) CreateCertificate(ctx context.Context, req *enterprisev1.Certificate) (*enterprisev1.Certificate, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	{
		_, err := s.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
		if err == nil {
			return nil, grpcutils.AlreadyExists("The Certificate %s already exists", req.Metadata.Name)
		}
		if !grpcerr.IsNotFound(err) {
			return nil, grpcutils.InternalWithErr(err)
		}
	}

	if err := s.validateCertificate(ctx, req); err != nil {
		return nil, err
	}

	item := &enterprisev1.Certificate{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &enterprisev1.Certificate_Status{},
	}

	item, err := s.octeliumC.EnterpriseC().CreateCertificate(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *Server) GetCertificate(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.Certificate, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
		Uid:  req.Uid,
		Name: req.Name,
	})
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) ListCertificate(ctx context.Context, req *enterprisev1.ListCertificateOptions) (*enterprisev1.CertificateList, error) {

	itemList, err := s.octeliumC.EnterpriseC().ListCertificate(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) DeleteCertificate(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {

	g, err := s.octeliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{Name: req.Name, Uid: req.Uid})
	if err != nil {
		return nil, err
	}

	if g.Metadata.IsSystem {
		return nil, serr.InvalidArg("Cannot delete the system group: %s", req.Name)
	}

	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	_, err = s.octeliumC.EnterpriseC().DeleteCertificate(ctx, &rmetav1.DeleteOptions{Uid: g.Metadata.Uid})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *Server) UpdateCertificate(ctx context.Context, req *enterprisev1.Certificate) (*enterprisev1.Certificate, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	if err := s.validateCertificate(ctx, req); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetCertificate(ctx,
		&rmetav1.GetOptions{
			Name: req.Metadata.Name,
			Uid:  req.Metadata.Uid,
		})
	if err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	item, err = s.octeliumC.EnterpriseC().UpdateCertificate(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *Server) validateCertificate(ctx context.Context, req *enterprisev1.Certificate) error {
	spec := req.Spec
	if spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	switch spec.Mode {
	case enterprisev1.Certificate_Spec_MODE_UNSET:
		return grpcutils.InvalidArg("Mode must be set")
	}

	return nil
}

func (s *Server) IssueCertificate(ctx context.Context, req *enterprisev1.IssueCertificateRequest) (*enterprisev1.IssueCertificateResponse, error) {
	if err := apivalidation.CheckObjectRef(req.CertificateRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	crt, err := s.octeliumC.EnterpriseC().GetCertificate(ctx, apivalidation.ObjectReferenceToRGetOptions(req.CertificateRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if crt.Spec.Mode != enterprisev1.Certificate_Spec_MANAGED {
		return nil, grpcutils.InvalidArg("Certificate Mode is not MANAGED")
	}

	_, err = certutils.DoIssueCertificate(ctx, s.octeliumC, crt)
	if err != nil {
		return nil, err
	}

	return &enterprisev1.IssueCertificateResponse{}, nil
}
