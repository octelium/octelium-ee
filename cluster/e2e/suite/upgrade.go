// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	releaseUpgradeBudget = 75 * time.Minute

	genesisInitIDKey = "octelium.com/genesis-init-id"

	coreGenesisJob       = "octelium-genesis"
	coreGenesisNamespace = "default"
)

func hasImageVersion(dep *appsv1.Deployment, version string) bool {
	containers := dep.Spec.Template.Spec.Containers
	return len(containers) > 0 && strings.HasSuffix(containers[0].Image, ":"+version)
}

func waitDeploymentVersion(t *testing.T, h *eeharness.H, name, version string) {
	t.Helper()

	h.Eventually(t, fmt.Sprintf("the Deployment %s to use %s", name, version), clusterManStartBudget,
		func(ctx context.Context) error {
			dep, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
			if err != nil {
				return err
			}
			if !hasImageVersion(dep, version) {
				return errors.Errorf("the Deployment %s does not use %s yet", name, version)
			}
			return nil
		})
}

func assertDeploymentVersion(t *testing.T, k8sC kubernetes.Interface, name, version string) {
	t.Helper()

	dep, err := k8sC.AppsV1().Deployments(vutils.K8sNS).Get(t.Context(), name, k8smetav1.GetOptions{})
	require.Nil(t, err)
	assert.True(t, hasImageVersion(dep, version), "the Deployment %s does not use %s", name, version)
}

func getGenesisContainers(jobs []batchv1.Job) []string {
	var ret []string
	for i := range jobs {
		if !isJobComplete(&jobs[i]) {
			continue
		}
		for _, c := range jobs[i].Spec.Template.Spec.Containers {
			ret = append(ret, c.Name)
		}
	}

	return ret
}

func testReleaseUpgradeSource(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapReleaseUpgrade)

	info, err := h.ClusterC().GetClusterInfo(t.Context(), &enterprisev1.GetClusterInfoRequest{})
	require.Nil(t, err)
	require.NotNil(t, info.Core)
	require.NotNil(t, info.PackageEnterprise)
	require.NotNil(t, info.PackageCordium)

	target := eescenario.UpgradeTarget()
	require.NotEqual(t, target.Core, info.Core.CurrentVersion)
	require.NotEqual(t, target.Enterprise, info.PackageEnterprise.CurrentVersion)
	require.NotEqual(t, target.Cordium, info.PackageCordium.CurrentVersion)
}

func testReleaseUpgradeClusterConfigApply(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapReleaseUpgrade)

	cc := h.ClusterConfig(t)
	h.UpdateClusterConfig(t, &corev1.ClusterConfig{
		Kind:       cc.Kind,
		ApiVersion: cc.ApiVersion,
		Metadata:   &metav1.Metadata{Name: cc.Metadata.Name},
		Spec:       cc.Spec,
	})

	assert.Empty(t, h.ClusterConfig(t).Metadata.Annotations[genesisInitIDKey])
}

func testReleaseUpgrade(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapReleaseUpgrade)

	rgn := h.Region(t)
	dumpRegionOnFailure(t, h, rgn)
	getJoinedRegion(t, h, rgn.Name)

	target := eescenario.UpgradeTarget()

	before := h.EnterpriseClusterConfig(t)
	require.Nil(t, before.Status.UpgradeRequest)

	existing := getGenesisJobUIDs(t, h.K8sC())
	regionExisting := getGenesisJobUIDs(t, rgn.K8sC())
	rollout := getClusterManagerRollout(t, h)

	_, err := h.ClusterC().UpgradeCluster(t.Context(), &enterprisev1.UpgradeClusterRequest{
		Request: &enterprisev1.UpgradeClusterRequest_Request{
			Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
				Version: target.Core,
			},
			PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
				Version: target.Enterprise,
			},
			PackageCordium: &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{
				Version: target.Cordium,
			},
		},
	})
	require.Nil(t, err)

	job := waitGenesisJobStarted(t, h, h.K8sC(), existing)
	waitDeploymentVersion(t, h, "octelium-nocturne", target.Core)
	interruptGenesisJob(t, h.K8sC(), job)

	history := waitClusterManagerUpgrade(t, h, before, releaseUpgradeBudget)
	require.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, history.State, "%+v", history)

	t.Run("TheInterruptedJobIsRetried", func(t *testing.T) {
		current := waitGenesisJobComplete(t, h, h.K8sC(), job, clusterManRecordBudget)
		assert.GreaterOrEqual(t, current.Status.Failed, int32(1))
	})

	t.Run("EveryRegionIsUpgradedInOrder", func(t *testing.T) {
		require.Len(t, history.Regions, 2)

		assert.Equal(t, "default", history.Regions[0].RegionRef.GetName())
		assert.Equal(t, rgn.Name, history.Regions[1].RegionRef.GetName())

		for _, itm := range history.Regions {
			assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, itm.State,
				"%s: %s", itm.RegionRef.GetName(), itm.Message)
		}
	})

	t.Run("EveryPackageIsRecordedInEveryRegion", func(t *testing.T) {
		for _, name := range []string{"default", rgn.Name} {
			region, err := h.CoreC().GetRegion(t.Context(), &metav1.GetOptions{Name: name})
			require.Nil(t, err)

			for pkg, version := range map[string]string{
				"octelium":                target.Core,
				eescenario.Package:        target.Enterprise,
				eescenario.PackageCordium: target.Cordium,
			} {
				require.Contains(t, region.Status.VersionInfoMap, pkg, name)
				assert.Equal(t, version, region.Status.VersionInfoMap[pkg].Version, "%s: %s", name, pkg)
			}
		}
	})

	t.Run("TheRegionRanTheUpgradeJobs", func(t *testing.T) {
		containers := getGenesisContainers(newRegionJobs(t, rgn, regionExisting))

		assert.Contains(t, containers, "octelium-genesis")
		assert.Contains(t, containers, "octeliumee-genesis")
		assert.Contains(t, containers, "cordium-genesis")
	})

	t.Run("TheComponentsRunTheTargetVersions", func(t *testing.T) {
		waitClusterManagerRollout(t, h, rollout)

		for _, name := range eescenario.Deployments {
			h.MustWaitDeployment(t, name)
		}
		for _, name := range eescenario.CordiumDeployments {
			h.MustWaitDeployment(t, name)
		}

		assertDeploymentVersion(t, h.K8sC(), "octelium-nocturne", target.Core)
		assertDeploymentVersion(t, h.K8sC(), "octeliumee-nocturne", target.Enterprise)
		assertDeploymentVersion(t, h.K8sC(), clusterManDeployment, target.Enterprise)
		assertDeploymentVersion(t, h.K8sC(), "cordium-nocturne", target.Cordium)

		assertDeploymentVersion(t, rgn.K8sC(), "octelium-nocturne", target.Core)
		assertDeploymentVersion(t, rgn.K8sC(), "octeliumee-nocturne", target.Enterprise)
	})

	t.Run("TheCountersRecordOneSuccess", func(t *testing.T) {
		after := h.EnterpriseClusterConfig(t)
		assert.Equal(t, before.Status.TotalFailedUpgrades, after.Status.TotalFailedUpgrades)
		assert.Equal(t, before.Status.TotalSuccessfulUpgrades+1, after.Status.TotalSuccessfulUpgrades)
	})
}

func testInstallRecoveryRetries(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapInstallRecovery)

	h.MustRun(t, "cat "+eescenario.InterruptWorkersLog)

	t.Run("TheCoreInitJobIsRetried", func(t *testing.T) {
		job, err := h.K8sC().BatchV1().Jobs(coreGenesisNamespace).Get(t.Context(), coreGenesisJob, k8smetav1.GetOptions{})
		require.Nil(t, err)
		assert.True(t, isJobComplete(job))
		assert.GreaterOrEqual(t, job.Status.Failed, int32(1))
	})

	t.Run("TheEnterpriseInitJobIsRetried", func(t *testing.T) {
		jobs, err := h.K8sC().BatchV1().Jobs(vutils.K8sNS).List(t.Context(), k8smetav1.ListOptions{LabelSelector: genesisSelector})
		require.Nil(t, err)

		var retried []string
		for i := range jobs.Items {
			job := &jobs.Items[i]
			if strings.HasPrefix(job.Name, "octelium-genesis-init-"+eescenario.Package+"-") &&
				isJobComplete(job) && job.Status.Failed >= 1 {
				retried = append(retried, job.Name)
			}
		}
		assert.Len(t, retried, 1)
	})
}
