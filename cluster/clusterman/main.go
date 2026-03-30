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

	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/commands"
	eec "github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium/cluster/common/components"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	cobra.OnInitialize()
	commands.InitCmds()
}

func init() {
	components.SetComponentNamespace(eec.ComponentNamespaceOcteliumEE)
	components.SetComponentType(eec.ClusterMan)
}

func main() {
	components.RunComponent(func(ctx context.Context) error {
		if err := commands.Cmd.Execute(); err != nil {
			zap.L().Fatal("main err", zap.Error(err))
		}

		return nil
	}, nil)
}
