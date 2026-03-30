// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package commands

import (
	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/commands/run"
	"github.com/octelium/octelium/client/common/cliutils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "clusterman",
	Short: `Manage Cluster resources`,
	// SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "run" {
			return nil
		}
		return cliutils.PreRun(cmd, args)
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "run" {
			return nil
		}

		return cliutils.PostRun(cmd, args)
	},
}

func InitCmds() {

	Cmd.AddCommand(run.Cmd)

}

func init() {

}
