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
	"fmt"
	"slices"
	"time"

	"github.com/octelium/octelium/cluster/e2e/scenario"
)

const DefaultScenario = "k3s-flannel-ee"

const Package = "octeliumee"

func init() {
	scenario.Register(DefaultScenario, k3sFlannelEE)
}

var Components = slices.Concat(scenario.DefaultComponents, []string{
	"clusterman",
	"secretman",
	"logstore",
	"rscstore",
	"metricstore",
	"cloudman",
	"collector",
	"policyportal",
})

var Deployments = []string{
	"octeliumee-rscserver",
	"octeliumee-nocturne",
	"octeliumee-secretman",
	"octeliumee-clusterman",
	"octeliumee-cloudman",
	"octeliumee-collector",
	"octeliumee-logstore",
	"octeliumee-rscstore",
	"octeliumee-metricstore",
	"octeliumee-policyportal",

	"svc-enterprise-octelium-api",
	"svc-public-octelium",
	"svc-dirsync-octelium",
	"svc-console-octelium",
	"svc-access-octelium",
}

func k3sFlannelEE() *scenario.Scenario {
	ret := scenario.MustGet("k3s-flannel")

	ret.Description = "Single-node k3s with flannel, SPIRE and the Enterprise components."

	ret.Install.EnableSPIFFECSI = true

	ret.Components = Components

	ret.Caps = append(ret.Caps, scenario.CapSPIFFE)

	ret.Hooks.PostPrepare = []scenario.Step{
		{Name: "spire/install", Run: stepSPIRE},
	}

	ret.Hooks.PostInstall = []scenario.Step{
		{Name: "octeliumee/install-package", Run: stepInstallPackage},
		{Name: "octeliumee/readiness", Run: stepWaitDeployments},
	}

	ret.Budget = 75 * time.Minute

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

func stepInstallPackage(ctx context.Context, r *scenario.Runner) error {
	versionArg := ""
	if version := r.Scenario.Install.Version; version != "" {
		versionArg = fmt.Sprintf(" --version %s", version)
	}

	return r.Bash(ctx, fmt.Sprintf("octops install-package %s --package %s --kubeconfig %s%s",
		r.Scenario.Domain, Package, r.State.KubeconfigPath, versionArg))
}
