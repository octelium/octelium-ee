// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package acmec

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/octelium/octelium-ee/cluster/common/certutils"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func RegisterAccount(ctx context.Context, octeliumC octeliumc.ClientInterface, ci *enterprisev1.CertificateIssuer, force bool) error {

	if ci.Spec.GetAcme() == nil {
		zap.L().Debug("Not registering an ACME account. Not an ACME issuer", zap.Any("issuer", ci))
		return nil
	}

	switch ci.Status.State {
	case enterprisev1.CertificateIssuer_Status_READY:
		if !force {
			zap.L().Debug("Issuer already ready. No need to re-register", zap.Any("iss", ci))
			return nil
		}
	}

	zap.L().Debug("Starting registering an ACME account", zap.Any("issuer", ci))

	k, err := utils_cert.GenerateECDSA()
	if err != nil {
		return err
	}

	privateKeyBytes, err := k.GetPrivateKeyPEM()
	if err != nil {
		return err
	}

	email := ci.Spec.GetAcme().Email
	if email == "" {
		cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
		if err != nil {
			return err
		}

		email = fmt.Sprintf("contact@%s", cc.Status.Domain)
	}

	cfg := lego.NewConfig(&Account{
		Email: email,
		key:   k.PrivateKey,
	})

	cfg.CADirURL = getCADirURL(ci)

	client, err := lego.NewClient(cfg)
	if err != nil {
		return err
	}

	ci.Status.State = enterprisev1.CertificateIssuer_Status_PREPARING
	ci, err = octeliumC.EnterpriseC().UpdateCertificateIssuer(ctx, ci)
	if err != nil {
		return err
	}
	crtRsc, err := func() (*registration.Resource, error) {
		for i := 0; i < 300; i++ {
			rsc, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
			if err == nil {
				zap.L().Debug("Successfully registered ACME account", zap.Any("iss", ci))
				return rsc, nil
			}

			zap.S().Warnf("Could not register ACME account: %+v. Trying again...", err)
			time.Sleep(10 * time.Second)
		}
		return nil, errors.Errorf("Could not register ACME account")
	}()
	if err != nil {
		ci.Status.State = enterprisev1.CertificateIssuer_Status_NOT_READY
		_, err = octeliumC.EnterpriseC().UpdateCertificateIssuer(ctx, ci)
		if err != nil {
			return err
		}
		return err
	}

	acc := &Account{
		Email:        email,
		Registration: crtRsc,
	}

	accBytes, err := json.Marshal(acc)
	if err != nil {
		return err
	}

	acmeSecret, err := octeliumC.EnterpriseC().GetSecret(ctx,
		&rmetav1.GetOptions{Name: uenterprisev1.ToCertificateIssuer(ci).GetACMEAccountSecretName()})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}
		acmeSecret, err = octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{
				Name:           uenterprisev1.ToCertificateIssuer(ci).GetACMEAccountSecretName(),
				IsSystem:       true,
				IsUserHidden:   true,
				IsSystemHidden: true,
			},
			Spec:   &enterprisev1.Secret_Spec{},
			Status: &enterprisev1.Secret_Status{},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_DataMap_{
					DataMap: &enterprisev1.Secret_Data_DataMap{
						Map: map[string][]byte{
							"account":    accBytes,
							"privateKey": []byte(privateKeyBytes),
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

	} else {

		if acmeSecret.Data == nil || acmeSecret.Data.GetDataMap() == nil || acmeSecret.Data.GetDataMap().Map == nil {
			acmeSecret.Data = &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_DataMap_{
					DataMap: &enterprisev1.Secret_Data_DataMap{
						Map: make(map[string][]byte),
					},
				},
			}
		}

		acmeSecret.Data.GetDataMap().Map["account"] = accBytes
		acmeSecret.Data.GetDataMap().Map["privateKey"] = []byte(privateKeyBytes)

		acmeSecret, err = octeliumC.EnterpriseC().UpdateSecret(ctx, acmeSecret)
		if err != nil {
			return err
		}
	}

	ci.Status.State = enterprisev1.CertificateIssuer_Status_READY
	ci.Status.Type = &enterprisev1.CertificateIssuer_Status_Acme{
		Acme: &enterprisev1.CertificateIssuer_Status_ACME{
			SecretRef: umetav1.GetObjectReference(acmeSecret),
		},
	}
	ci, err = octeliumC.EnterpriseC().UpdateCertificateIssuer(ctx, ci)
	if err != nil {
		return err
	}

	zap.S().Info("Successfully registered an ACME account", zap.Any("issuer", ci))

	if err := issueCerts(ctx, octeliumC, ci); err != nil {
		return err
	}

	return nil
}

func issueCerts(ctx context.Context, octeliumC octeliumc.ClientInterface, iss *enterprisev1.CertificateIssuer) error {
	crtList, err := octeliumC.EnterpriseC().ListCertificate(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.certificateIssuerRef.uid", iss.Metadata.Uid),
		},
	})
	if err != nil {
		return err
	}

	for _, crt := range crtList.Items {

		crt, err := certutils.DoIssueCertificate(ctx, octeliumC, crt)
		if err != nil {
			zap.L().Warn("Could not DoIssueCertificate", zap.Error(err), zap.Any("crt", crt))
			continue
		}

		if err := IssueCertificate(ctx, octeliumC, crt); err != nil {
			zap.L().Warn("Could not issue cert", zap.Any("crt", zap.Error(err)))
		}
	}

	return nil
}

func getCADirURL(cc *enterprisev1.CertificateIssuer) string {
	if cc.Spec.GetAcme().Server != "" {
		return cc.Spec.GetAcme().Server
	}

	if ldflags.IsDev() || ldflags.IsTest() {
		return "https://acme-staging-v02.api.letsencrypt.org/directory"
	} else {
		return "https://acme-v02.api.letsencrypt.org/directory"
	}
}
