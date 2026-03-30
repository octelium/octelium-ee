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
	api "github.com/octelium/octelium-ee/cluster/mockapiserver/mockapiserver"
	"github.com/octelium/octelium/cluster/common/components"
)

func init() {
	components.SetComponentNamespace(eec.ComponentNamespaceOcteliumEE)
	components.SetComponentType(eec.ClusterMan)
}

func main() {
	components.RunComponent(api.Run, nil)
}
