// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package scenario

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/cluster/e2e/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScenarioSPIFFE(t *testing.T) {
	for _, tc := range []struct {
		id       string
		spiffe   bool
		fixtures bool
	}{
		{id: DefaultScenario},
		{id: CoreMainScenario},
		{id: SPIREScenario, spiffe: true},
		{id: FullScenario, spiffe: true, fixtures: true},
	} {
		t.Run(tc.id, func(t *testing.T) {
			for _, val := range []string{"", "false", "true", "1"} {
				t.Run("SPIFFE="+val, func(t *testing.T) {
					t.Setenv(EnvSPIFFE, val)
					t.Setenv("OCTELIUM_SPIFFE_TRUST_DOMAIN", "")
					t.Setenv(EnvCoreVersion, "")
					s, err := scenario.Get(tc.id)
					require.NoError(t, err)

					enabled := tc.spiffe || val == "true" || val == "1"
					assert.Equal(t, tc.id, s.ID)
					assert.Equal(t, enabled, s.Install.EnableSPIFFECSI)
					assert.Equal(t, enabled, s.Caps.Has(scenario.CapSPIFFE))
					assert.Equal(t, Components, s.Components)
					assert.False(t, s.Install.OTelCollector)
					if tc.id == CoreMainScenario {
						assert.Equal(t, "main", s.Install.Version)
					}

					var steps []string
					if enabled {
						assert.Equal(t, scenario.SPIFFETrustDomainDefault, s.Install.SPIFFETrustDomain)
						steps = append(steps, "spiffe/spire")
					} else {
						assert.Empty(t, s.Install.SPIFFETrustDomain)
					}
					if tc.fixtures {
						steps = append(steps, "fixtures/namespace", "fixtures/vault",
							"fixtures/keycloak", "fixtures/otlp-sink")
						assert.Equal(t, 120*time.Minute, s.Budget)
					}
					var got []string
					for _, step := range s.Hooks.PostPrepare {
						require.NotNil(t, step.Run)
						got = append(got, step.Name)
					}
					assert.Equal(t, steps, got)
					assert.Equal(t, tc.fixtures, s.Caps.Has(CapVault))
					assert.Equal(t, tc.fixtures, s.Caps.Has(CapKeycloak))
					assert.Equal(t, tc.fixtures, s.Caps.Has(CapOTLPSink))
					assert.Equal(t, tc.fixtures, s.Caps.Has(scenario.CapUpgrade))
					require.NotEmpty(t, s.Hooks.PostInstall)
					assert.Equal(t, "octeliumee/install-package", s.Hooks.PostInstall[0].Name)
				})
			}
		})
	}
}

func TestScenarioSPIFFETrustDomain(t *testing.T) {
	t.Setenv(EnvSPIFFE, "true")
	t.Setenv("OCTELIUM_SPIFFE_TRUST_DOMAIN", "enterprise.test")

	for _, id := range []string{DefaultScenario, CoreMainScenario, SPIREScenario, FullScenario} {
		t.Run(id, func(t *testing.T) {
			s, err := scenario.Get(id)
			require.NoError(t, err)
			assert.Equal(t, "enterprise.test", s.Install.SPIFFETrustDomain)
		})
	}
}

func TestScenarioReleaseUpgrade(t *testing.T) {
	t.Setenv(EnvCoreVersion, "main")
	t.Setenv(EnvFromCoreVersion, "0.49.0")

	s, err := scenario.Get(ReleaseUpgradeScenario)
	require.NoError(t, err)

	assert.Equal(t, "0.49.0", s.Install.Version)
	assert.Equal(t, VersionLatest, PackageVersionFor(ReleaseUpgradeScenario))
	assert.True(t, s.Caps.Has(CapReleaseUpgrade))
	assert.True(t, s.Caps.Has(CapMultiRegion))
	assert.True(t, s.Caps.Has(scenario.CapUpgrade))
	assert.False(t, s.Caps.Has(CapInstallRecovery))
	assert.Contains(t, s.Topology.Labels, cordiumNodeLabel)

	var steps []string
	for _, step := range s.Hooks.PostInstall {
		require.NotNil(t, step.Run)
		steps = append(steps, step.Name)
	}
	assert.Equal(t, []string{
		"octeliumee/install-package",
		"octeliumee/readiness",
		"octeliumee/e2e-tuning",
		"octeliumee/rollout",
		"cordium/install-package",
		"cordium/readiness",
	}, steps)

	t.Setenv(EnvFromCoreVersion, "")
	s, err = scenario.Get(ReleaseUpgradeScenario)
	require.NoError(t, err)
	assert.Empty(t, s.Install.Version)

	for _, id := range []string{DefaultScenario, CoreMainScenario, SPIREScenario, FullScenario, MultiRegionScenario} {
		s, err := scenario.Get(id)
		require.NoError(t, err)
		assert.False(t, s.Caps.Has(CapReleaseUpgrade), id)
		assert.False(t, s.Caps.Has(CapInstallRecovery), id)
		assert.NotContains(t, s.Topology.Labels, cordiumNodeLabel, id)
	}
}

func TestScenarioInstallRecovery(t *testing.T) {
	t.Setenv(EnvCoreVersion, "main")
	t.Setenv(EnvFromCoreVersion, "0.49.0")

	s, err := scenario.Get(InstallRecoveryScenario)
	require.NoError(t, err)

	assert.Equal(t, "main", s.Install.Version)
	assert.True(t, s.Caps.Has(CapInstallRecovery))
	assert.True(t, s.Caps.Has(scenario.CapUpgrade))
	assert.False(t, s.Caps.Has(CapMultiRegion))
	assert.False(t, s.Caps.Has(CapReleaseUpgrade))

	var steps []string
	for _, step := range s.Hooks.PostPrepare {
		require.NotNil(t, step.Run)
		steps = append(steps, step.Name)
	}
	assert.Equal(t, []string{"install-recovery/interrupt-workers"}, steps)
}

func TestUpgradeTarget(t *testing.T) {
	t.Setenv("GITHUB_REF_NAME", "")
	t.Setenv(EnvCoreVersion, "")
	t.Setenv(EnvPackageVersion, "")
	t.Setenv(EnvCordiumVersion, "")
	assert.Equal(t, Versions{Core: "main", Enterprise: "main", Cordium: "main"}, UpgradeTarget())

	t.Setenv(EnvCoreVersion, "0.50.0")
	t.Setenv(EnvPackageVersion, "feature")
	t.Setenv(EnvCordiumVersion, "0.19.0")
	assert.Equal(t, Versions{Core: "0.50.0", Enterprise: "feature", Cordium: "0.19.0"}, UpgradeTarget())

	t.Setenv(EnvCoreVersion, VersionLatest)
	t.Setenv(EnvPackageVersion, VersionLatest)
	t.Setenv(EnvCordiumVersion, VersionLatest)
	assert.Equal(t, Versions{Core: "main", Enterprise: "main", Cordium: "main"}, UpgradeTarget())
}

func TestInterruptWorkersScript(t *testing.T) {
	dir := t.TempDir()
	calls := filepath.Join(dir, "calls")
	stub := fmt.Sprintf(`#!/bin/sh
echo "$*" >> %[1]s
case "$1" in
get)
  count=$(grep -c "^get deployment" %[1]s)
  [ "$count" -ge 3 ]
  ;;
delete)
  echo "pod \"worker\" force deleted"
  ;;
esac
`, calls)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte(stub), 0755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	logPath := filepath.Join(dir, "interrupt-workers.log")
	cmd := exec.CommandContext(t.Context(), "bash", "-c", "set -e\n"+interruptWorkersScript(logPath))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	assert.Eventually(t, func() bool {
		data, err := os.ReadFile(logPath)
		return err == nil && strings.Count(string(data), "Interrupted the genesis worker") == 2
	}, 30*time.Second, 50*time.Millisecond)

	data, err := os.ReadFile(calls)
	require.NoError(t, err)
	var deletes []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.HasPrefix(line, "delete") {
			deletes = append(deletes, line)
		}
	}
	require.Len(t, deletes, 2)
	for i, namespace := range []string{"default", "octelium"} {
		assert.Contains(t, deletes[i], "--namespace "+namespace)
		assert.Contains(t, deletes[i], "--selector octelium.com/component=genesis")
		assert.Contains(t, deletes[i], "--field-selector status.phase=Running")
	}
	assert.Less(t, strings.Index(string(data), "octelium-nocturne"), strings.Index(string(data), "delete pod --namespace default"))
	assert.Less(t, strings.Index(string(data), "delete pod --namespace default"), strings.Index(string(data), "octeliumee-secretman"))
}
