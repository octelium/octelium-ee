// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package regioncontroller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/cloudmanutils"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnsctl"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
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

func (c *Controller) OnAdd(ctx context.Context, region *corev1.Region) error {
	zap.S().Debugf("New Region: %s", region.Metadata.Name)
	if err := SetRegionDNS(ctx, c.octeliumC, region); err != nil {
		return err
	}
	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *corev1.Region) error {

	if reflect.DeepEqual(new.Status.IngressAddresses, old.Status.IngressAddresses) {
		zap.S().Debugf("No need to update Region DNS entries: %s", new.Metadata.Name)
		return nil
	}

	if err := SetRegionDNS(ctx, c.octeliumC, new); err != nil {
		return err
	}

	return nil
}

func (c *Controller) OnDelete(ctx context.Context, region *corev1.Region) error {

	return nil
}

func SetRegionDNS(ctx context.Context, octeliumC octeliumc.ClientInterface, region *corev1.Region) error {

	if region.Status == nil || len(region.Status.IngressAddresses) == 0 {
		zap.S().Debugf("Region %s has no ingress addresses for now. Skipping setting public DNS addrs",
			region.Metadata.Name)
		return nil
	}

	ccc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	clusterDomain := ccc.Status.Domain
	publicFQDN := fmt.Sprintf("%s.%s", region.Status.PublicHostname, clusterDomain)

	provider, err := cloudmanutils.GetDefaultDNSProvider(ctx, octeliumC)
	if err != nil {
		return err
	}

	zap.S().Debugf("Setting the Region %s FQDN %s public DNS to the IPs: %+v", region.Metadata.Name, publicFQDN, region.Status.IngressAddresses)
	if err := dnsctl.SetAddresses(ctx, octeliumC, provider, ccc, region.Status.IngressAddresses, publicFQDN); err != nil {
		zap.L().Warn("Could not SetAddresses for Cluster domain", zap.Error(err))
	}

	if region.Metadata.Name == "default" {
		zap.S().Debugf("Setting the `default` Region s subdomain CNAME entry")
		if err := dnsctl.SetCNAME(ctx, octeliumC, provider, ccc, fmt.Sprintf("*.s.%s", clusterDomain), publicFQDN); err != nil {
			zap.L().Warn("Could not SetCNAME for default domain", zap.Error(err))
		}

		zap.S().Debugf("Setting the `default` Region address entry")
		if err := dnsctl.SetAddresses(ctx, octeliumC, provider, ccc, region.Status.IngressAddresses, clusterDomain); err != nil {
			zap.L().Warn("Could not SetAddresses for Cluster default domain", zap.Error(err))
		}
	}

	zap.S().Debugf("Successfully set the DNS entries of the Region %s", region.Metadata.Name)

	return nil
}
