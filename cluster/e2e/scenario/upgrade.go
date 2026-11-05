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
	"time"

	"github.com/octelium/octelium/cluster/e2e/scenario"
	"go.uber.org/zap"
)

const (
	ReleaseUpgradeScenario  = "k3s-flannel-ee-release-upgrade"
	InstallRecoveryScenario = "k3s-flannel-ee-install-recovery"
)

const (
	CapReleaseUpgrade  scenario.Capability = "release-upgrade"
	CapInstallRecovery scenario.Capability = "install-recovery"
)

const (
	EnvFromCoreVersion    = "OCTELIUM_E2E_FROM_CORE_VERSION"
	EnvFromPackageVersion = "OCTELIUM_E2E_FROM_PACKAGE_VERSION"
	EnvFromCordiumVersion = "OCTELIUM_E2E_FROM_CORDIUM_VERSION"
	EnvCordiumVersion     = "OCTELIUM_E2E_CORDIUM_VERSION"
)

const (
	PackageCordium      = "cordium"
	defaultTarget       = "main"
	cordiumNodeLabel    = "octelium.com/node-mode-cordium"
	InterruptWorkersLog = "/tmp/octelium-e2e-interrupt-workers.log"
)

const interruptWorkersBudget = time.Hour

var CordiumDeployments = []string{
	"cordium-nocturne",
	"cordium-rscserver",

	"svc-default-cordium-octelium-api",
	"svc-default-cordium",
	"svc-default-ssh-cordium",
}

func init() {
	register(ReleaseUpgradeScenario, opts{
		description: "The last release of core, Enterprise and Cordium with a joined Region, " +
			"upgraded to the candidate versions by Cluster Manager",
		multiRegion:    true,
		releaseUpgrade: true,
		upgrade:        true,
		budget:         120 * time.Minute,
	})

	register(InstallRecoveryScenario, opts{
		description:     "The Enterprise scenario with the genesis workers interrupted during the installation",
		installRecovery: true,
		upgrade:         true,
	})
}

type Versions struct {
	Core       string
	Enterprise string
	Cordium    string
}

func UpgradeTarget() Versions {
	return Versions{
		Core:       cmpOr(coreVersion(""), defaultTarget),
		Enterprise: cmpOr(packageVersion(), defaultTarget),
		Cordium:    cmpOr(normalizeVersion(os.Getenv(EnvCordiumVersion)), defaultTarget),
	}
}

func withReleaseUpgrade(ret *scenario.Scenario) {
	ret.Caps = append(ret.Caps, CapReleaseUpgrade)
	ret.Topology.Labels = append(ret.Topology.Labels, cordiumNodeLabel)

	ret.Hooks.PostInstall = append(ret.Hooks.PostInstall,
		scenario.Step{Name: "cordium/install-package", Run: stepInstallCordium},
		scenario.Step{Name: "cordium/readiness", Run: stepWaitCordium},
	)
}

func withInstallRecovery(ret *scenario.Scenario) {
	ret.Caps = append(ret.Caps, CapInstallRecovery)

	ret.Hooks.PostPrepare = append(ret.Hooks.PostPrepare,
		scenario.Step{Name: "install-recovery/interrupt-workers", Run: stepInterruptWorkers})
}

func stepInstallCordium(ctx context.Context, r *scenario.Runner) error {
	version := normalizeVersion(os.Getenv(EnvFromCordiumVersion))

	zap.L().Info("Installing the Cordium package",
		zap.String("packageVersion", displayVersion(version)))

	versionArg := ""
	if version != "" {
		versionArg = fmt.Sprintf(" --version %s", version)
	}

	return r.Bash(ctx, fmt.Sprintf("octops install-package %s --package %s --kubeconfig %s%s",
		r.Scenario.Domain, PackageCordium, r.State.KubeconfigPath, versionArg))
}

func stepWaitCordium(ctx context.Context, r *scenario.Runner) error {
	return waitPackage(ctx, r, PackageCordium, CordiumDeployments)
}

func stepInterruptWorkers(ctx context.Context, r *scenario.Runner) error {
	return r.Bash(ctx, interruptWorkersScript(InterruptWorkersLog))
}

func interruptWorkersScript(logPath string) string {
	return fmt.Sprintf(`
setsid nohup timeout %[2]d bash -c '
interrupt() {
  until kubectl get deployment --namespace octelium "$1" >/dev/null 2>&1; do
    sleep 1
  done
  until kubectl delete pod --namespace "$2" --selector octelium.com/component=genesis \
    --field-selector status.phase=Running 2>&1 | grep -q deleted; do
    sleep 1
  done
  echo "Interrupted the genesis worker in the namespace $2 after the Deployment $1 was created"
}
interrupt octelium-nocturne default
interrupt octeliumee-secretman octelium
' >%[1]s 2>&1 </dev/null &
`, logPath, int(interruptWorkersBudget.Seconds()))
}
