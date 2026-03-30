// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package components

import (
	"fmt"

	"github.com/octelium/octelium/cluster/common/components"
	"github.com/octelium/octelium/pkg/utils/ldflags"
)

const APIServer components.ComponentType = "apiserver"
const Nocturne components.ComponentType = "nocturne"
const DirSync components.ComponentType = "dirsync"
const SecretMan components.ComponentType = "secretman"
const Genesis components.ComponentType = "genesis"
const CloudMan components.ComponentType = "cloudman"
const Collector components.ComponentType = "collector"
const RscServer components.ComponentType = "rscserver"
const PublicServer components.ComponentType = "publicserver"
const LogStore components.ComponentType = "logstore"
const RscStore components.ComponentType = "rscstore"
const MetricStore components.ComponentType = "metricstore"
const PolicyPortal components.ComponentType = "policyportal"

const ClusterMan = "clusterman"
const Console = "console"

const ComponentNamespaceOcteliumEE components.ComponentNamespace = "octeliumee"

func GetImage(component, version string) string {
	return ldflags.GetImage(fmt.Sprintf("octeliumee-%s", component), version)
}

func OcteliumEnterpriseComponent(arg string) string {
	return fmt.Sprintf("octeliumee-%s", arg)
}
