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

		if err := g.RunInit(context.Background(), &genesis.InitOpts{}); err != nil {
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

		if err := g.RunUpgrade(context.Background(), &genesis.UpgradeOpts{}); err != nil {
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

func setDeprecatedFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.BoolVar(&cmdArgs.EnableSPIFFECSIDriver, "enable-spiffe-csi", false, "Deprecated. Set via the bootstrap Config instead")
	f.StringVar(&cmdArgs.SPIFFECSIDriver, "spiffe-csi-driver", "", "Deprecated. Set via the bootstrap Config instead")
	f.StringVar(&cmdArgs.SPIFFETrustDomain, "spiffe-trust-domain", "", "Deprecated. Set via the bootstrap Config instead")
	f.BoolVar(&cmdArgs.EnableIngressFrontProxy, "ingress-front-proxy", false, "Deprecated. Set via the bootstrap Config instead")

	for _, name := range []string{"enable-spiffe-csi", "spiffe-csi-driver", "spiffe-trust-domain",
		"ingress-front-proxy"} {
		f.MarkHidden(name)
	}
}

func init() {
	setDeprecatedFlags(initCmd)
	setDeprecatedFlags(upgradeCmd)
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
