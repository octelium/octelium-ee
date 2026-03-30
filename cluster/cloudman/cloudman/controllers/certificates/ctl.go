// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package certificates

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/acmec"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
}

func NewController(
	octeliumC octeliumc.ClientInterface,
) *Controller {
	return &Controller{
		octeliumC: octeliumC,
	}
}

func (c *Controller) OnAdd(ctx context.Context, crt *enterprisev1.Certificate) error {

	if err := c.handleIssuance(ctx, crt); err != nil {
		return err
	}

	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *enterprisev1.Certificate) error {
	if !pbutils.IsEqual(new.Status, old.Status) {
		if err := c.handleIssuance(ctx, new); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) OnDelete(ctx context.Context, crt *enterprisev1.Certificate) error {

	_, err := c.octeliumC.CoreC().DeleteSecret(ctx, &rmetav1.DeleteOptions{
		Name: uenterprisev1.ToCertificate(crt).GetSecretName(),
	})
	if err != nil && !grpcerr.IsNotFound(err) {
		return err
	}

	return nil
}

func (c *Controller) handleIssuance(ctx context.Context, crt *enterprisev1.Certificate) error {

	switch crt.Spec.Mode {
	case enterprisev1.Certificate_Spec_MANUAL:
		return c.handleManualIssuance(ctx, crt)
	case enterprisev1.Certificate_Spec_MANAGED:
		if crt.Status.CertificateIssuerRef == nil {
			zap.L().Debug("No certIssuer. Nothing to be done", zap.Any("crt", crt))
			return nil
		}

		iss, err := c.octeliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{
			Uid: crt.Status.CertificateIssuerRef.Uid,
		})
		if err != nil {
			return err
		}

		switch iss.Spec.Type.(type) {
		case *enterprisev1.CertificateIssuer_Spec_Acme:
			if err := acmec.IssueCertificate(ctx, c.octeliumC, crt); err != nil {
				return err
			}
		default:
			zap.L().Debug("Unknown certificateIssuer type. Nothing to be done...")
		}
	}

	return nil
}

func (c *Controller) handleManualIssuance(ctx context.Context, crt *enterprisev1.Certificate) error {
	return nil
}
