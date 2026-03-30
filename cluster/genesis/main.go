// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package main

import (
	"context"

	eec "github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium-ee/cluster/genesis/genesis"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/components"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:  "genesis",
	Long: `genesis`,
}

var initCmd = &cobra.Command{
	Use: "init",
	RunE: func(cmd *cobra.Command, args []string) error {

		g, err := genesis.NewGenesis()
		if err != nil {
			return err
		}

		if err := g.RunInit(context.Background(), &genesis.InitOpts{
			EnableSPIFFECSI:         cmdArgs.EnableSPIFFECSIDriver,
			SPIFFECSIDriver:         cmdArgs.SPIFFECSIDriver,
			SPIFFETrustDomain:       cmdArgs.SPIFFETrustDomain,
			EnableIngressFrontProxy: cmdArgs.EnableIngressFrontProxy,
		}); err != nil {
			return err
		}

		return nil
	},
}

var upgradeCmd = &cobra.Command{
	Use: "upgrade",
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := genesis.NewGenesis()
		if err != nil {
			return err
		}

		if err := g.RunUpgrade(context.Background(), &genesis.UpgradeOpts{
			EnableSPIFFECSI:         cmdArgs.EnableSPIFFECSIDriver,
			SPIFFECSIDriver:         cmdArgs.SPIFFECSIDriver,
			SPIFFETrustDomain:       cmdArgs.SPIFFETrustDomain,
			EnableIngressFrontProxy: cmdArgs.EnableIngressFrontProxy,
		}); err != nil {
			return err
		}

		return nil
	},
}

var cmdArgs args

type args struct {
	EnableSPIFFECSIDriver   bool
	SPIFFECSIDriver         string
	SPIFFETrustDomain       string
	EnableIngressFrontProxy bool
}

func init() {
	initCmd.PersistentFlags().BoolVar(&cmdArgs.EnableSPIFFECSIDriver, "enable-spiffe-csi", false, "Enable SPIFFE CSI Driver")
	initCmd.PersistentFlags().StringVar(&cmdArgs.SPIFFECSIDriver, "spiffe-csi-driver", "", "SPIFFE CSI Driver name")
	initCmd.PersistentFlags().StringVar(&cmdArgs.SPIFFETrustDomain, "spiffe-trust-domain", "", "SPIFFE trust domain")
	initCmd.PersistentFlags().BoolVar(&cmdArgs.EnableIngressFrontProxy, "ingress-front-proxy", false, "Enable Ingress front proxy mode")
}

func init() {
	upgradeCmd.PersistentFlags().BoolVar(&cmdArgs.EnableSPIFFECSIDriver, "enable-spiffe-csi", false, "Enable SPIFFE CSI Driver")
	upgradeCmd.PersistentFlags().StringVar(&cmdArgs.SPIFFECSIDriver, "spiffe-csi-driver", "", "SPIFFE CSI Driver name")
	upgradeCmd.PersistentFlags().StringVar(&cmdArgs.SPIFFETrustDomain, "spiffe-trust-domain", "", "SPIFFE trust domain")
	upgradeCmd.PersistentFlags().BoolVar(&cmdArgs.EnableIngressFrontProxy, "ingress-front-proxy", false, "Enable Ingress front proxy mode")
}

func init() {
	components.SetComponentNamespace(eec.ComponentNamespaceOcteliumEE)
	components.SetComponentType(eec.Genesis)
}

func main() {

	components.RunComponent(func(ctx context.Context) error {
		rootCmd.AddCommand(initCmd)
		rootCmd.AddCommand(upgradeCmd)
		// rootCmd.AddCommand(joinCmd)

		if err := commoninit.Run(ctx, nil); err != nil {
			return err
		}

		if err := rootCmd.Execute(); err != nil {
			return err
		}
		return nil
	}, nil)

}
