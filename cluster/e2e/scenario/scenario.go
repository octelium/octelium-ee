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
	"os"
	"slices"
	"time"

	"github.com/octelium/octelium/cluster/e2e/scenario"
	"go.uber.org/zap"
)

const (
	DefaultScenario  = "k3s-flannel-ee"
	SPIREScenario    = "k3s-flannel-ee-spire"
	CoreMainScenario = "k3s-flannel-ee-core-main"
	FullScenario     = "k3s-flannel-ee-full"
)

const Package = "octeliumee"

const (
	EnvCoreVersion    = "OCTELIUM_E2E_CORE_VERSION"
	EnvPackageVersion = "OCTELIUM_E2E_PACKAGE_VERSION"
)

const VersionLatest = "latest"

var packageVersions = map[string]string{}

func init() {
	register(DefaultScenario, opts{
		description: "Single-node k3s with flannel and the Enterprise components",
	})

	register(SPIREScenario, opts{
		description: "The Enterprise scenario with SPIRE and the SPIFFE CSI driver",
		spire:       true,
	})

	register(CoreMainScenario, opts{
		description: "The Enterprise scenario against the core main branch images",
		coreVersion: "main",
	})
}

type opts struct {
	description string
	coreVersion string
	spire       bool
	fixtures    bool
	budget      time.Duration
}

func register(id string, o opts) {
	packageVersions[id] = packageVersion()

	scenario.Register(id, func() *scenario.Scenario {
		ret := scenario.MustGet("k3s-flannel")

		ret.Description = o.description
		ret.Components = Components
		ret.Install.Version = coreVersion(o.coreVersion)

		if o.spire {
			ret.Install.EnableSPIFFECSI = true
			ret.Caps = append(ret.Caps, scenario.CapSPIFFE)
			ret.Hooks.PostPrepare = append(ret.Hooks.PostPrepare,
				scenario.Step{Name: "spire/install", Run: stepSPIRE})
		}

		if o.fixtures {
			withFixtures(ret)
		}

		ret.Hooks.PostInstall = []scenario.Step{
			{Name: "octeliumee/install-package", Run: stepInstallPackage},
			{Name: "octeliumee/readiness", Run: stepWaitDeployments},
		}

		ret.Budget = cmpOr(o.budget, 75*time.Minute)

		return ret
	})
}

func PackageVersionFor(id string) string {
	return displayVersion(packageVersions[id])
}

func coreVersion(def string) string {
	if val := os.Getenv(EnvCoreVersion); val != "" {
		return normalizeVersion(val)
	}

	return normalizeVersion(def)
}

func packageVersion() string {
	if val := os.Getenv(EnvPackageVersion); val != "" {
		return normalizeVersion(val)
	}

	return normalizeVersion(os.Getenv("GITHUB_REF_NAME"))
}

func normalizeVersion(arg string) string {
	if arg == VersionLatest {
		return ""
	}

	return arg
}

func cmpOr[T comparable](vals ...T) T {
	var zero T
	for _, val := range vals {
		if val != zero {
			return val
		}
	}
	return zero
}

func displayVersion(arg string) string {
	return cmpOr(arg, VersionLatest)
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

func stepSPIRE(ctx context.Context, r *scenario.Runner) error {
	return r.Bash(ctx, `
helm repo add spire https://spiffe.github.io/helm-charts-hardened/
helm repo update spire
helm upgrade --install spire-crds spire/spire-crds --namespace spire --create-namespace
helm upgrade --install spire spire/spire --namespace spire --wait --timeout 10m
`)
}

func stepInstallPackage(ctx context.Context, r *scenario.Runner) error {
	version := packageVersions[r.Scenario.ID]

	zap.L().Info("Installing the Octelium Enterprise package",
		zap.String("package", Package),
		zap.String("packageVersion", displayVersion(version)),
		zap.String("coreVersion", displayVersion(r.Scenario.Install.Version)))

	versionArg := ""
	if version != "" {
		versionArg = fmt.Sprintf(" --version %s", version)
	}

	return r.Bash(ctx, fmt.Sprintf("octops install-package %s --package %s --kubeconfig %s%s",
		r.Scenario.Domain, Package, r.State.KubeconfigPath, versionArg))
}
