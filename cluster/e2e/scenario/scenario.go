// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package scenario

import (
	"context"

	"github.com/octelium/octelium/cluster/e2e/scenario"
)

const DefaultScenario = "k3s-flannel-ee"

func init() {
	scenario.Register(DefaultScenario, k3sFlannelEE)
}

var Components = append(scenario.DefaultComponents,
	"clusterman",
	"secretman",
	"logstore",
	"rscstore",
	"metricstore",
	"cloudman",
	"collector",
	"policyportal",
)

func k3sFlannelEE() *scenario.Scenario {
	ret := scenario.MustGet("k3s-flannel")

	ret.Description = "Single-node k3s with flannel, SPIRE and the Enterprise components."

	ret.Install.EnableSPIFFECSI = true
	ret.Install.WaitDeployments = append(ret.Install.WaitDeployments,
		"svc-enterprise-octelium-api")

	ret.Components = Components

	ret.Caps = append(ret.Caps, scenario.CapSPIFFE)

	ret.Hooks.PostPrepare = []scenario.Step{
		{Name: "spire/install", Run: stepSPIRE},
	}

	return ret
}

func stepSPIRE(ctx context.Context, r *scenario.Runner) error {
	return r.Bash(ctx, `
helm repo add spire https://spiffe.github.io/helm-charts-hardened/
helm repo update spire
helm upgrade --install spire-crds spire/spire-crds --namespace spire --create-namespace
helm upgrade --install spire spire/spire --namespace spire --wait --timeout 10m
`)
}
