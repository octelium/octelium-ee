// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"slices"
	"testing"

	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/cluster/e2e/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func phaseNames(phases []suite.Phase) []string {
	var ret []string
	for _, p := range phases {
		ret = append(ret, p.Name)
	}
	return ret
}

func TestAllSelectsTheSuite(t *testing.T) {
	multiRegion := phaseNames(MultiRegionPhases())
	require.NotEmpty(t, multiRegion)

	for _, tc := range []struct {
		val  string
		want []string
	}{
		{val: "", want: phaseNames(Enterprise())},
		{val: SuiteEnterprise, want: phaseNames(Enterprise())},
		{val: "unknown", want: phaseNames(Enterprise())},
		{val: SuiteAll, want: phaseNames(withCore())},
		{val: "ALL", want: phaseNames(withCore())},
		{val: SuiteMultiRegion, want: phaseNames(MultiRegion())},
		{val: " Multi-Region ", want: phaseNames(MultiRegion())},
	} {
		t.Run(tc.val, func(t *testing.T) {
			t.Setenv(EnvSuite, tc.val)
			assert.Equal(t, tc.want, phaseNames(All()))
		})
	}
}

func TestAllIgnoresTheCoreSuiteEnv(t *testing.T) {
	require.NotEqual(t, harness.SuiteEnv, EnvSuite)

	for _, val := range []string{harness.SuiteBase, SuiteAll, SuiteMultiRegion} {
		t.Run(val, func(t *testing.T) {
			t.Setenv(EnvSuite, "")
			t.Setenv(harness.SuiteEnv, val)
			assert.Equal(t, phaseNames(Enterprise()), phaseNames(All()))
		})
	}
}

func TestMultiRegionPhases(t *testing.T) {
	multiRegion := phaseNames(MultiRegionPhases())

	assert.Equal(t, []string{
		"MultiRegionJoin",
		"MultiRegionComponents",
		"MultiRegionTelemetry",
		"MultiRegionService",
		"MultiRegionUpgrade",
	}, multiRegion)

	got := phaseNames(MultiRegion())
	assert.Equal(t, []string{"ClusterReady", ReadyPhase().Name}, got[:2])
	assert.Equal(t, multiRegion, got[2:])

	for _, phases := range [][]suite.Phase{Enterprise(), withCore()} {
		for _, name := range multiRegion {
			assert.False(t, slices.Contains(phaseNames(phases), name), name)
		}
	}

	for _, p := range MultiRegion() {
		assert.NotNil(t, p.Run, p.Name)
	}
}

func TestClusterManagerPhases(t *testing.T) {
	clusterManager := phaseNames(ClusterManagerPhases())

	assert.Equal(t, []string{
		"ClusterManagerRestartDuringJoin",
		"ClusterManagerJobRetry",
		"ClusterManagerRestartDuringUpgrade",
	}, clusterManager)

	got := phaseNames(ClusterManager())
	assert.Equal(t, []string{"ClusterReady", ReadyPhase().Name}, got[:2])
	assert.Equal(t, clusterManager, got[2:])

	for _, phases := range [][]suite.Phase{Enterprise(), withCore(), MultiRegion()} {
		for _, name := range clusterManager {
			assert.False(t, slices.Contains(phaseNames(phases), name), name)
		}
	}

	for _, p := range ClusterManager() {
		assert.NotNil(t, p.Run, p.Name)
	}

	t.Setenv(EnvSuite, SuiteClusterManager)
	assert.Equal(t, phaseNames(ClusterManager()), phaseNames(All()))
}
