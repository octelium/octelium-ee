// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package certificateissuers

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/acmec"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
	acmeC     acmec.CertificateIssuerSetter
}

func NewController(
	octeliumC octeliumc.ClientInterface,
	acmeC acmec.CertificateIssuerSetter,
) *Controller {
	return &Controller{
		octeliumC: octeliumC,
		acmeC:     acmeC,
	}
}

func (c *Controller) OnAdd(ctx context.Context, iss *enterprisev1.CertificateIssuer) error {
	if iss.Spec.GetAcme() == nil {
		return nil
	}

	return c.acmeC.SetCertificateIssuer(ctx, iss, false)
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *enterprisev1.CertificateIssuer) error {
	if new.Spec.GetAcme() == nil {
		return nil
	}

	return c.acmeC.SetCertificateIssuer(ctx, new,
		!pbutils.IsEqual(new.Spec.GetAcme(), old.Spec.GetAcme()))
}

func (c *Controller) OnDelete(ctx context.Context, iss *enterprisev1.CertificateIssuer) error {

	return nil
}
