// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package scenario

import (
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
