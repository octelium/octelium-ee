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
	console "github.com/octelium/octelium-ee/cluster/console/console"
	"github.com/octelium/octelium/cluster/common/components"
)

func init() {
	components.SetComponentNamespace(eec.ComponentNamespaceOcteliumEE)
	components.SetComponentType(eec.Console)
}

func main() {
	components.RunComponent(console.Run, nil)
}
