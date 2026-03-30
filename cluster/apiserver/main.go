// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package main

import (
	api "github.com/octelium/octelium-ee/cluster/apiserver/apiserver"
	"github.com/octelium/octelium/cluster/common/components"
)

func init() {
	components.SetComponentNamespace("")
	components.SetComponentNamespace(components.ComponentNamespaceOctelium)
	components.SetComponentType(components.APIServer)
}

func main() {
	components.RunComponent(api.Run, nil)
}
