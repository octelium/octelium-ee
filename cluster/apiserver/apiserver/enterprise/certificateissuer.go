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

	"github.com/asaskevich/govalidator"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
)

const (
	maxCertificateIssuerServerURLBytes = 2048
	maxCertificateIssuerEmailBytes     = 254
)

func (s *Server) GetCertificateIssuer(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.CertificateIssuer, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
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
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
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
	if req == nil {
		return grpcutils.InvalidArg("Nil CertificateIssuer")
	}

	spec := req.Spec
	if spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	switch spec.Type.(type) {
	case *enterprisev1.CertificateIssuer_Spec_Acme:
		return validateCertificateIssuerACME(spec.GetAcme())
	default:
		return grpcutils.InvalidArg("You must set CertificateIssuer type")
	}
}

func validateCertificateIssuerACME(acme *enterprisev1.CertificateIssuer_Spec_ACME) error {
	if acme == nil {
		return grpcutils.InvalidArg("Nil ACME spec")
	}

	if acme.GetServer() == "" {
		return grpcutils.InvalidArg("ACME server is required")
	}
	if len(acme.GetServer()) > maxCertificateIssuerServerURLBytes {
		return grpcutils.InvalidArg("ACME server URL is too long")
	}
	if !govalidator.IsURL(acme.GetServer()) {
		return grpcutils.InvalidArg("Invalid ACME server URL")
	}

	if acme.GetEmail() == "" {
		return grpcutils.InvalidArg("ACME email is required")
	}
	if len(acme.GetEmail()) > maxCertificateIssuerEmailBytes {
		return grpcutils.InvalidArg("ACME email is too long")
	}
	if !govalidator.IsEmail(acme.GetEmail()) {
		return grpcutils.InvalidArg("Invalid ACME email")
	}

	if acme.GetSolver() == nil {
		return grpcutils.InvalidArg("ACME solver is required")
	}

	switch acme.GetSolver().Type.(type) {
	case *enterprisev1.CertificateIssuer_Spec_ACME_Solver_Dns:
		if acme.GetSolver().GetDns() == nil {
			return grpcutils.InvalidArg("Nil ACME DNS solver")
		}
	default:
		return grpcutils.InvalidArg("You must set ACME solver type")
	}

	return nil
}
