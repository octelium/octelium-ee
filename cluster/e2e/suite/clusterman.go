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
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	clusterManDeployment   = "octeliumee-clusterman"
	clusterManRolloutKey   = "octelium.com/last-upgrade"
	clusterManSelector     = "octelium.com/app=octeliumee,octelium.com/component=clusterman"
	genesisSelector        = "octelium.com/component=genesis"
	clusterManStartBudget  = 10 * time.Minute
	clusterManRecordBudget = 5 * time.Minute
)

func getGenesisJobUIDs(t *testing.T, k8sC kubernetes.Interface) map[types.UID]bool {
	t.Helper()

	jobs, err := k8sC.BatchV1().Jobs(vutils.K8sNS).List(t.Context(), k8smetav1.ListOptions{LabelSelector: genesisSelector})
	require.Nil(t, err)

	ret := map[types.UID]bool{}
	for _, job := range jobs.Items {
		ret[job.UID] = true
	}

	return ret
}

func waitGenesisJobStarted(t *testing.T, h *eeharness.H, k8sC kubernetes.Interface,
	existing map[types.UID]bool) *batchv1.Job {
	t.Helper()

	var ret *batchv1.Job
	h.Eventually(t, "a new genesis Job to start", clusterManStartBudget,
		func(ctx context.Context) error {
			jobs, err := k8sC.BatchV1().Jobs(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{LabelSelector: genesisSelector})
			if err != nil {
				return err
			}
			for i := range jobs.Items {
				if job := &jobs.Items[i]; !existing[job.UID] && job.Status.Active > 0 {
					ret = job
					return nil
				}
			}
			return errors.Errorf("no new genesis Job is active yet")
		})

	require.NotNil(t, ret)
	return ret
}

func waitGenesisJobComplete(t *testing.T, h *eeharness.H, k8sC kubernetes.Interface,
	job *batchv1.Job, budget time.Duration) *batchv1.Job {
	t.Helper()

	var ret *batchv1.Job
	h.Eventually(t, "the genesis Job to complete", budget,
		func(ctx context.Context) error {
			current, err := k8sC.BatchV1().Jobs(job.Namespace).Get(ctx, job.Name, k8smetav1.GetOptions{})
			if err != nil {
				return err
			}
			if current.UID != job.UID {
				return errors.Errorf("the genesis Job %s was replaced", job.Name)
			}
			if !isJobComplete(current) {
				return errors.Errorf("the genesis Job %s is still running", job.Name)
			}
			ret = current
			return nil
		})

	require.NotNil(t, ret)
	return ret
}

func killClusterManager(t *testing.T, h *eeharness.H) {
	t.Helper()

	pods := h.K8sC().CoreV1().Pods(vutils.K8sNS)
	list, err := pods.List(t.Context(), k8smetav1.ListOptions{LabelSelector: clusterManSelector})
	require.Nil(t, err)
	require.NotEmpty(t, list.Items)

	for _, pod := range list.Items {
		require.Nil(t, pods.Delete(t.Context(), pod.Name, k8smetav1.DeleteOptions{GracePeriodSeconds: new(int64(0))}))
	}

	h.MustWaitDeployment(t, clusterManDeployment)
}

func getClusterManagerRollout(t *testing.T, h *eeharness.H) string {
	t.Helper()

	dep, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).Get(t.Context(), clusterManDeployment, k8smetav1.GetOptions{})
	require.Nil(t, err)

	return dep.Annotations[clusterManRolloutKey]
}

func waitClusterManagerRollout(t *testing.T, h *eeharness.H, before string) {
	t.Helper()

	h.Eventually(t, "Cluster Manager to be rolled out", clusterManRecordBudget,
		func(ctx context.Context) error {
			dep, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).Get(ctx, clusterManDeployment, k8smetav1.GetOptions{})
			if err != nil {
				return err
			}
			if dep.Annotations[clusterManRolloutKey] == before {
				return errors.Errorf("Cluster Manager is not rolled out yet")
			}
			return nil
		})

	h.MustWaitDeployment(t, clusterManDeployment)
}

func requestClusterManagerUpgrade(t *testing.T, h *eeharness.H) *enterprisev1.ClusterConfig {
	t.Helper()

	info, err := h.ClusterC().GetClusterInfo(t.Context(), &enterprisev1.GetClusterInfoRequest{})
	require.Nil(t, err)
	require.NotNil(t, info.Core)
	require.NotNil(t, info.PackageEnterprise)

	before := h.EnterpriseClusterConfig(t)
	require.Nil(t, before.Status.UpgradeRequest)

	_, err = h.ClusterC().UpgradeCluster(t.Context(), &enterprisev1.UpgradeClusterRequest{
		Request: &enterprisev1.UpgradeClusterRequest_Request{
			Core: &enterprisev1.UpgradeClusterRequest_Request_Core{
				Version: info.Core.CurrentVersion,
			},
			PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
				Version: info.PackageEnterprise.CurrentVersion,
			},
		},
	})
	require.Nil(t, err)

	return before
}

func waitClusterManagerUpgrade(t *testing.T, h *eeharness.H, before *enterprisev1.ClusterConfig,
	budget time.Duration) *enterprisev1.ClusterConfig_Status_UpgradeRequest {
	t.Helper()

	var ret *enterprisev1.ClusterConfig_Status_UpgradeRequest
	h.Eventually(t, "the upgrade to be recorded", budget,
		func(ctx context.Context) error {
			cc, err := h.EnterpriseC().GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
			if err != nil {
				return err
			}
			if cc.Status.UpgradeRequest != nil {
				return errors.Errorf("the upgrade is %s", cc.Status.UpgradeRequest.State)
			}
			if len(cc.Status.LastUpgradeRequests) == 0 ||
				cc.Status.TotalSuccessfulUpgrades+cc.Status.TotalFailedUpgrades <=
					before.Status.TotalSuccessfulUpgrades+before.Status.TotalFailedUpgrades {
				return errors.Errorf("the upgrade is not recorded yet")
			}
			ret = cc.Status.LastUpgradeRequests[0]
			return nil
		})

	require.NotNil(t, ret)
	return ret
}

func testClusterManagerRestartDuringJoin(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapMultiRegion)

	rgn := h.Region(t)
	dumpRegionOnFailure(t, h, rgn)

	ctx := t.Context()

	sec := getRegionSecret(t, h, rgn)
	request := &enterprisev1.JoinClusterRequest{
		Name:        rgn.Name,
		DisplayName: multiRegionDisplayName,
		Kubernetes:  getJoinKubeconfig(sec.Metadata.Name),
	}

	beforeCount := len(h.EnterpriseClusterConfig(t).Status.LastJoinRequests)

	_, err := h.ClusterC().JoinCluster(ctx, request)
	require.Nil(t, err)

	job := waitGenesisJobStarted(t, h, rgn.K8sC(), nil)

	killClusterManager(t, h)

	last := waitJoinDone(t, h, beforeCount, clusterManRecordBudget)
	require.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_FAILED, last.State)
	assert.Contains(t, last.Message, "Cluster Manager restarted")
	assert.NotNil(t, last.DoneAt)

	t.Run("TheJoinJobIsLeftToFinish", func(t *testing.T) {
		waitGenesisJobComplete(t, h, rgn.K8sC(), job, multiRegionJoinBudget)
		getJoinedRegion(t, h, rgn.Name)
	})

	t.Run("ThePersistedStateAllowsNoDuplicateJoin", func(t *testing.T) {
		_, err := h.ClusterC().JoinCluster(ctx, request)
		require.NotNil(t, err)
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)
	})
}

func testClusterManagerJobRetry(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapMultiRegion)

	rgn := h.Region(t)
	dumpRegionOnFailure(t, h, rgn)

	existing := getGenesisJobUIDs(t, h.K8sC())
	rollout := getClusterManagerRollout(t, h)
	before := requestClusterManagerUpgrade(t, h)

	job := waitGenesisJobStarted(t, h, h.K8sC(), existing)

	pods := h.K8sC().CoreV1().Pods(job.Namespace)
	selector := batchv1.JobNameLabel + "=" + job.Name
	list, err := pods.List(t.Context(), k8smetav1.ListOptions{LabelSelector: selector})
	require.Nil(t, err)
	require.NotEmpty(t, list.Items)
	for _, pod := range list.Items {
		require.Nil(t, pods.Delete(t.Context(), pod.Name, k8smetav1.DeleteOptions{}))
	}

	history := waitClusterManagerUpgrade(t, h, before, multiRegionUpgradeBudget)
	require.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, history.State, "%+v", history)

	current := waitGenesisJobComplete(t, h, h.K8sC(), job, clusterManRecordBudget)
	assert.GreaterOrEqual(t, current.Status.Failed, int32(1))

	after := h.EnterpriseClusterConfig(t)
	assert.Equal(t, before.Status.TotalFailedUpgrades, after.Status.TotalFailedUpgrades)

	waitClusterManagerRollout(t, h, rollout)
}

func testClusterManagerRestartDuringUpgrade(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapMultiRegion)

	rgn := h.Region(t)
	dumpRegionOnFailure(t, h, rgn)

	existing := getGenesisJobUIDs(t, h.K8sC())
	before := requestClusterManagerUpgrade(t, h)

	job := waitGenesisJobStarted(t, h, h.K8sC(), existing)

	killClusterManager(t, h)

	history := waitClusterManagerUpgrade(t, h, before, clusterManRecordBudget)
	require.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, history.State, "%+v", history)
	require.NotNil(t, history.DoneAt)

	t.Run("TheUnfinishedRegionsRecordTheInterruption", func(t *testing.T) {
		require.NotEmpty(t, history.Regions)
		for _, itm := range history.Regions {
			assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, itm.State)
			assert.Contains(t, itm.Message, "Cluster Manager restarted")
		}
	})

	t.Run("TheRunningJobIsLeftToFinish", func(t *testing.T) {
		waitGenesisJobComplete(t, h, h.K8sC(), job, multiRegionUpgradeBudget)
	})

	t.Run("TheRestartStartsNoNewWork", func(t *testing.T) {
		jobs, err := h.K8sC().BatchV1().Jobs(vutils.K8sNS).List(t.Context(), k8smetav1.ListOptions{LabelSelector: genesisSelector})
		require.Nil(t, err)

		var added []string
		for _, itm := range jobs.Items {
			if !existing[itm.UID] {
				added = append(added, itm.Name)
			}
		}
		assert.Equal(t, []string{job.Name}, added)
	})

	t.Run("TheUpgradeCanBeRequestedAgain", func(t *testing.T) {
		before := requestClusterManagerUpgrade(t, h)

		history := waitClusterManagerUpgrade(t, h, before, multiRegionUpgradeBudget)
		require.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, history.State, "%+v", history)
	})
}
