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
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
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
		{val: SuiteReleaseUpgrade, want: phaseNames(ReleaseUpgrade())},
		{val: SuiteInstallRecovery, want: phaseNames(InstallRecovery())},
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

func TestReleaseUpgradePhases(t *testing.T) {
	releaseUpgrade := phaseNames(ReleaseUpgradePhases())

	assert.Equal(t, []string{
		"ReleaseUpgradeSource",
		"MultiRegionJoin",
		"ReleaseUpgradeClusterConfigApply",
		"ReleaseUpgrade",
		"MultiRegionComponents",
		"MultiRegionService",
	}, releaseUpgrade)

	got := phaseNames(ReleaseUpgrade())
	assert.Equal(t, []string{"ClusterReady", ReadyPhase().Name}, got[:2])
	assert.Equal(t, releaseUpgrade, got[2:])

	for _, phases := range [][]suite.Phase{Enterprise(), withCore(), MultiRegion(), ClusterManager(), InstallRecovery()} {
		for _, name := range []string{"ReleaseUpgradeSource", "ReleaseUpgradeClusterConfigApply", "ReleaseUpgrade"} {
			assert.False(t, slices.Contains(phaseNames(phases), name), name)
		}
	}

	for _, p := range ReleaseUpgrade() {
		assert.NotNil(t, p.Run, p.Name)
	}
	require.NoError(t, suite.Validate(ReleaseUpgrade()))

	t.Setenv(EnvSuite, SuiteReleaseUpgrade)
	assert.Equal(t, phaseNames(ReleaseUpgrade()), phaseNames(All()))
}

func TestInstallRecoveryPhases(t *testing.T) {
	installRecovery := phaseNames(InstallRecoveryPhases())

	assert.Equal(t, []string{"InstallRecoveryRetries", "ClusterUpgrade"}, installRecovery)

	got := phaseNames(InstallRecovery())
	assert.Equal(t, []string{"ClusterReady", ReadyPhase().Name}, got[:2])
	assert.Equal(t, installRecovery, got[2:])

	for _, phases := range [][]suite.Phase{Enterprise(), withCore(), MultiRegion(), ClusterManager(), ReleaseUpgrade()} {
		assert.False(t, slices.Contains(phaseNames(phases), "InstallRecoveryRetries"))
	}

	for _, p := range InstallRecovery() {
		assert.NotNil(t, p.Run, p.Name)
	}
	require.NoError(t, suite.Validate(InstallRecovery()))

	t.Setenv(EnvSuite, SuiteInstallRecovery)
	assert.Equal(t, phaseNames(InstallRecovery()), phaseNames(All()))
}

func TestHasImageVersion(t *testing.T) {
	dep := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: k8scorev1.PodTemplateSpec{
				Spec: k8scorev1.PodSpec{
					Containers: []k8scorev1.Container{{Image: "ghcr.io/octelium/octelium-nocturne:main"}},
				},
			},
		},
	}

	assert.True(t, hasImageVersion(dep, "main"))
	assert.False(t, hasImageVersion(dep, "0.49.0"))
	assert.False(t, hasImageVersion(&appsv1.Deployment{}, "main"))
}

func TestGetGenesisContainers(t *testing.T) {
	newJob := func(name string, conditionType batchv1.JobConditionType) batchv1.Job {
		return batchv1.Job{
			Spec: batchv1.JobSpec{
				Template: k8scorev1.PodTemplateSpec{
					Spec: k8scorev1.PodSpec{Containers: []k8scorev1.Container{{Name: name}}},
				},
			},
			Status: batchv1.JobStatus{
				Conditions: []batchv1.JobCondition{{Type: conditionType, Status: k8scorev1.ConditionTrue}},
			},
		}
	}

	assert.Equal(t, []string{"octelium-genesis", "cordium-genesis"}, getGenesisContainers([]batchv1.Job{
		newJob("octelium-genesis", batchv1.JobComplete),
		newJob("octeliumee-genesis", batchv1.JobFailed),
		newJob("cordium-genesis", batchv1.JobComplete),
	}))
	assert.Empty(t, getGenesisContainers(nil))
}
