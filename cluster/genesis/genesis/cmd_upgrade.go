// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesis

import (
	"context"
	"os"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	gc "github.com/octelium/octelium/cluster/genesis/genesis/components"
	"go.uber.org/zap"
)

type UpgradeOpts struct {
	EnableSPIFFECSI         bool
	SPIFFECSIDriver         string
	SPIFFETrustDomain       string
	EnableIngressFrontProxy bool
}

func (g *Genesis) RunUpgrade(ctx context.Context, o *UpgradeOpts) error {

	zap.L().Debug("Starting upgrade...")
	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	g.octeliumC = octeliumC
	g.octeliumCInit = octeliumC

	regionV, err := g.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: os.Getenv("OCTELIUM_REGION_NAME")})
	if err != nil {
		return err
	}

	cc, err := g.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	zap.S().Debugf("Installing components")

	if err := g.setVolumes(ctx); err != nil {
		zap.L().Warn("Could not setVolumes", zap.Error(err))
		return err
	}

	if err := g.installComponents(ctx, &components.CommonOpts{
		CommonOpts: gc.CommonOpts{
			Region:                  regionV,
			EnableSPIFFECSI:         o.EnableSPIFFECSI,
			SPIFFECSIDriver:         o.SPIFFECSIDriver,
			SPIFFETrustDomain:       o.SPIFFETrustDomain,
			EnableIngressFrontProxy: o.EnableIngressFrontProxy,
		},
	}, false); err != nil {
		return err
	}

	if err := g.installOcteliumResources(ctx, cc, regionV); err != nil {
		zap.L().Warn("Could not installOcteliumResources", zap.Error(err))
	}

	zap.L().Debug("Upgrade is successful")

	return nil
}
