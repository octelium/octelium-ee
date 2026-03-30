// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package gwcontroller

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/acmec"
	regioncontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/regions"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnss"
	"github.com/octelium/octelium-ee/cluster/common/certutils"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
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

func (c *Controller) OnAdd(ctx context.Context, p *enterprisev1.DNSProvider) error {
	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *enterprisev1.DNSProvider) error {

	if pbutils.IsEqual(new.Spec, old.Spec) {
		return nil
	}

	{
		zap.L().Debug("Initializing issuing certs after DNSProvider update")

		crtList, err := c.octeliumC.EnterpriseC().ListCertificate(ctx, &rmetav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, crt := range crtList.Items {
			crt, err := certutils.DoIssueCertificate(ctx, c.octeliumC, crt)
			if err != nil {
				zap.L().Warn("Could not doIssueCertificate", zap.Error(err), zap.Any("crt", crt))
				continue
			}

			if err := acmec.IssueCertificate(ctx, c.octeliumC, crt); err != nil {
				zap.L().Warn("Could not issueCrt", zap.Error(err), zap.Any("crt", crt))
			}
		}
	}
	{
		rgnList, err := c.octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, rgn := range rgnList.Items {
			if err := regioncontroller.SetRegionDNS(ctx, c.octeliumC, rgn); err != nil {
				zap.L().Warn("Could not setRegionDNS", zap.Error(err), zap.Any("region", rgn))
			}
		}
	}
	{
		svcList, err := c.octeliumC.CoreC().ListService(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldBooleanTrue("spec.isPublic"),
				urscsrv.FilterFieldEQValStr("status.regionRef.uid", vutils.GetMyRegionUID()),
			},
		})
		if err != nil {
			return err
		}

		cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
		if err != nil {
			return err
		}

		for _, svc := range svcList.Items {
			if err := dnss.SetServiceCNAME(ctx, c.octeliumC, svc, cc, new); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Controller) OnDelete(ctx context.Context, p *enterprisev1.DNSProvider) error {

	return nil
}

func (c *Controller) startApply(ctx context.Context, p *enterprisev1.DNSProvider) error {

	return nil
}

func (c *Controller) needsACMERegistration(cc *enterprisev1.ClusterConfig, p *enterprisev1.DNSProvider, iss *enterprisev1.CertificateIssuer) bool {
	if iss.Spec.GetAcme() == nil || iss.Spec.GetAcme().GetSolver() == nil ||
		iss.Spec.GetAcme().GetSolver().GetDns() == nil {
		return false
	}

	/*
		if cc.Spec.Dns != nil && cc.Spec.Dns.DefaultDNSProvider != "" {
			return cc.Spec.Dns.DefaultDNSProvider == p.Metadata.Name
		}
	*/

	return p.Metadata.Name == "default"
}

/*
func (c *Controller) setIssuers(ctx context.Context, p *enterprisev1.DNSProvider) error {
	cc, err := c.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	issList, err := c.octeliumC.EnterpriseC().ListCertificateIssuer(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, iss := range issList.Items {
		if !c.needsACMERegistration(cc, p, iss) {
			continue
		}
		if iss.Status.State != enterprisev1.CertificateIssuer_Status_NOT_READY {
			continue
		}

		if err := acmec.RegisterAccount(ctx, c.octeliumC, iss); err != nil {
			zap.L().Error("Could not register ACME acc after DNSProvider update",
				zap.Any("dnsProvider", p), zap.Error(err))
		}
	}

	return nil
}
*/
