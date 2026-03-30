// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package main

import (
	eec "github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium-ee/cluster/rscserver/rscserver"
	"github.com/octelium/octelium/cluster/common/components"
	"github.com/spf13/cobra"
)

func init() {
	components.SetComponentNamespace(eec.ComponentNamespaceOcteliumEE)
	components.SetComponentType(eec.RscServer)
}

var rootCmd = &cobra.Command{
	Use:  "rscserver",
	Long: `rscserver`,
}

func main() {

	components.RunComponent(rscserver.Run, nil)
}
