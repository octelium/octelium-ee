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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/certutils"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
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
		_, err := s.octeliumC.EnterpriseC().GetCertificate(ctx, apivalidation.ObjectToRGetOptions(req))
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
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetCertificate(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) ListCertificate(ctx context.Context, req *enterprisev1.ListCertificateOptions) (*enterprisev1.CertificateList, error) {
	if req == nil {
		req = &enterprisev1.ListCertificateOptions{}
	}

	itemList, err := s.octeliumC.EnterpriseC().ListCertificate(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) DeleteCertificate(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {
	if err := apivalidation.CheckDeleteOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	g, err := s.octeliumC.EnterpriseC().GetCertificate(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(g); err != nil {
		return nil, err
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

	item, err := s.octeliumC.EnterpriseC().GetCertificate(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if err := apivalidation.CheckIsSystem(item); err != nil {
		return nil, err
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	switch item.Spec.Mode {
	case enterprisev1.Certificate_Spec_MANAGED:
		iss, err := s.octeliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{
			Name: "default",
		})
		if err != nil {
			return nil, serr.K8sNotFoundOrInternalWithErr(err)
		}
		item.Status.CertificateIssuerRef = umetav1.GetObjectReference(iss)
	case enterprisev1.Certificate_Spec_MANUAL:
		item.Status.CertificateIssuerRef = nil
	}

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
	case enterprisev1.Certificate_Spec_MANUAL,
		enterprisev1.Certificate_Spec_MANAGED:
		return nil
	case enterprisev1.Certificate_Spec_MODE_UNSET:
		return grpcutils.InvalidArg("Mode must be set")
	default:
		return grpcutils.InvalidArg("Invalid Certificate mode")
	}
}

func (s *Server) IssueCertificate(ctx context.Context, req *enterprisev1.IssueCertificateRequest) (*enterprisev1.IssueCertificateResponse, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	if err := apivalidation.CheckObjectRef(req.CertificateRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	crt, err := s.octeliumC.EnterpriseC().GetCertificate(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.CertificateRef))
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

func (s *Server) SetCertificate(ctx context.Context, req *enterprisev1.SetCertificateRequest) (*enterprisev1.SetCertificateResponse, error) {
	const (
		maxCertificatePEMBytes = 256 << 10
		maxPrivateKeyPEMBytes  = 64 << 10
	)

	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	if err := apivalidation.CheckObjectRef(req.CertificateRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	if req.Certificate == "" {
		return nil, grpcutils.InvalidArg("Certificate is required")
	}
	if req.PrivateKey == "" {
		return nil, grpcutils.InvalidArg("Private key is required")
	}

	if len(req.Certificate) > maxCertificatePEMBytes {
		return nil, grpcutils.InvalidArg("Certificate PEM is too large")
	}
	if len(req.PrivateKey) > maxPrivateKeyPEMBytes {
		return nil, grpcutils.InvalidArg("Private key PEM is too large")
	}

	keyPair, err := tls.X509KeyPair([]byte(req.Certificate), []byte(req.PrivateKey))
	if err != nil {
		return nil, serr.InvalidArg("Could not parse TLS key pair")
	}

	if len(keyPair.Certificate) == 0 {
		return nil, serr.InvalidArg("Certificate chain is empty")
	}

	leaf, err := x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, serr.InvalidArg("Could not parse leaf certificate")
	}

	now := time.Now()
	if now.Before(leaf.NotBefore) {
		return nil, serr.InvalidArg("Certificate is not valid yet")
	}
	if now.After(leaf.NotAfter) {
		return nil, serr.InvalidArg("Certificate is expired")
	}

	switch pub := leaf.PublicKey.(type) {
	case *rsa.PublicKey:
		if pub.N.BitLen() < 2048 {
			return nil, serr.InvalidArg("RSA certificate key is too small")
		}
	case *ecdsa.PublicKey:
		switch pub.Curve {
		case elliptic.P256(), elliptic.P384(), elliptic.P521():
		default:
			return nil, serr.InvalidArg("Unsupported ECDSA certificate curve")
		}
	case ed25519.PublicKey:
	default:
		return nil, serr.InvalidArg("Unsupported certificate public key type")
	}

	if err := validatePrivateKeyStrength(keyPair.PrivateKey); err != nil {
		return nil, err
	}

	crt, err := s.octeliumC.EnterpriseC().GetCertificate(ctx,
		apivalidation.ObjectReferenceToRGetOptions(req.CertificateRef))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	if crt.Metadata.IsSystem {
		return nil, serr.InvalidArg("Cannot set a system Certificate")
	}

	if crt.Spec == nil {
		return nil, grpcutils.InvalidArg("Certificate has nil spec")
	}

	if crt.Spec.Mode != enterprisev1.Certificate_Spec_MANUAL {
		return nil, grpcutils.InvalidArg("Certificate Mode must be MANUAL")
	}

	secretName := uenterprisev1.ToCertificate(crt).GetSecretName()
	if secretName == "" {
		return nil, grpcutils.Internal("Could not determine Certificate Secret name")
	}

	var sec *corev1.Secret
	sec, err = s.octeliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: secretName,
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return nil, serr.InternalWithErr(err)
		}

		sec = &corev1.Secret{
			Metadata: &metav1.Metadata{
				Name:           secretName,
				IsSystem:       true,
				IsUserHidden:   true,
				IsSystemHidden: true,
				SystemLabels: map[string]string{
					"octelium-cert": "true",
				},
			},
			Spec:   &corev1.Secret_Spec{},
			Status: &corev1.Secret_Status{},
		}

		ucorev1.ToSecret(sec).SetCertificate(req.Certificate, req.PrivateKey)

		sec, err = s.octeliumC.CoreC().CreateSecret(ctx, sec)
		if err != nil {
			return nil, serr.InternalWithErr(err)
		}
	} else {

		sec.Metadata.IsSystem = true
		sec.Metadata.IsUserHidden = true
		sec.Metadata.IsSystemHidden = true
		if sec.Metadata.SystemLabels == nil {
			sec.Metadata.SystemLabels = map[string]string{}
		}
		sec.Metadata.SystemLabels["octelium-cert"] = "true"

		ucorev1.ToSecret(sec).SetCertificate(req.Certificate, req.PrivateKey)

		sec, err = s.octeliumC.CoreC().UpdateSecret(ctx, sec)
		if err != nil {
			return nil, serr.InternalWithErr(err)
		}
	}

	crt.Status.SecretRef = umetav1.GetObjectReference(sec)

	info, err := certutils.GetInfo(req.Certificate, req.PrivateKey)
	if err != nil {
		return nil, serr.InvalidArg("Could not extract certificate info")
	}
	crt.Status.Info = info

	if _, err := s.octeliumC.EnterpriseC().UpdateCertificate(ctx, crt); err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return &enterprisev1.SetCertificateResponse{}, nil
}

func validatePrivateKeyStrength(privateKey any) error {
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		if key.N.BitLen() < 2048 {
			return serr.InvalidArg("RSA private key is too small")
		}
	case *ecdsa.PrivateKey:
		switch key.Curve {
		case elliptic.P256(), elliptic.P384(), elliptic.P521():
		default:
			return serr.InvalidArg("Unsupported ECDSA private key curve")
		}
	case ed25519.PrivateKey:
	default:
		return serr.InvalidArg("Unsupported private key type")
	}

	return nil
}
