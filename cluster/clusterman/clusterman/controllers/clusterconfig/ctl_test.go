// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package clusterconfig

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func newUpgradeControllerTest(t *testing.T) (*Controller, *enterprisev1.ClusterConfig) {
	t.Helper()
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	cc, err := tst.C.OcteliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	return NewController(tst.C.OcteliumC, fake.NewSimpleClientset()), cc
}

func TestUpgradeHistoryIsBoundedAndPreservesOrder(t *testing.T) {
	var history []*enterprisev1.ClusterConfig_Status_UpgradeRequest
	for i := range maxLastUpgradeRequests + 3 {
		req := &enterprisev1.ClusterConfig_Status_UpgradeRequest{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, DoneAt: pbutils.Timestamp(time.Unix(int64(i), 0))}
		previous := history
		history = prependUpgradeHistory(history, req)
		assert.Same(t, req, history[0])
		assert.LessOrEqual(t, len(history), maxLastUpgradeRequests)
		if len(previous) > 0 {
			assert.Same(t, previous[0], history[1])
		}
	}
	assert.Len(t, history, maxLastUpgradeRequests)
	assert.EqualValues(t, 202, history[0].DoneAt.Seconds)
	assert.EqualValues(t, 3, history[len(history)-1].DoneAt.Seconds)
	assert.Equal(t, history, prependUpgradeHistory(history, nil))
}

func TestUpgradeIgnoresAbsentAndTerminalRequests(t *testing.T) {
	ctl := NewController(nil, nil)
	for _, req := range []*enterprisev1.ClusterConfig_Status_UpgradeRequest{
		nil,
		{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS},
		{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED},
	} {
		cc := &enterprisev1.ClusterConfig{Status: &enterprisev1.ClusterConfig_Status{UpgradeRequest: req}}
		require.NoError(t, ctl.doUpgrade(t.Context(), cc))
		require.NoError(t, ctl.OnUpdate(t.Context(), cc, pbutils.Clone(cc).(*enterprisev1.ClusterConfig)))
	}
}

func TestUpgradeSuccessfulRequestUpdatesHistoryAndCounters(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.TotalSuccessfulUpgrades = 4
	cc.Status.TotalFailedUpgrades = 2
	cc.Status.LastUpgradeRequests = []*enterprisev1.ClusterConfig_Status_UpgradeRequest{{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED}}
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State:   enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request: &enterprisev1.UpgradeClusterRequest_Request{},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	require.NoError(t, ctl.doUpgrade(t.Context(), cc))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.UpgradeRequest)
	assert.EqualValues(t, 5, current.Status.TotalSuccessfulUpgrades)
	assert.EqualValues(t, 2, current.Status.TotalFailedUpgrades)
	require.Len(t, current.Status.LastUpgradeRequests, 2)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, current.Status.LastUpgradeRequests[0].State)
	assert.NotNil(t, current.Status.LastUpgradeRequests[0].DoneAt)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, current.Status.LastUpgradeRequests[1].State)
}

func TestUpgradeJobFailureIsRecordedWithoutSuccessIncrement(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.TotalSuccessfulUpgrades = 4
	cc.Status.TotalFailedUpgrades = 2
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State:   enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request: &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"}},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	ctl.k8sC.(*fake.Clientset).PrependReactor("create", "jobs", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("Job creation denied")
	})
	ctl.doUpgrade(t.Context(), cc)
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.UpgradeRequest)
	assert.EqualValues(t, 4, current.Status.TotalSuccessfulUpgrades)
	assert.EqualValues(t, 3, current.Status.TotalFailedUpgrades)
	require.Len(t, current.Status.LastUpgradeRequests, 1)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, current.Status.LastUpgradeRequests[0].State)
	assert.NotNil(t, current.Status.LastUpgradeRequests[0].DoneAt)
}

type upgradeStatusTestClient struct {
	octeliumc.ClientInterface
	resources renterprisev1.ResourceServiceClient
}

func (c *upgradeStatusTestClient) EnterpriseC() renterprisev1.ResourceServiceClient {
	return c.resources
}

type failingUpgradeStatusClient struct {
	renterprisev1.ResourceServiceClient
	err error
}

func (c *failingUpgradeStatusClient) UpdateClusterConfig(context.Context, *enterprisev1.ClusterConfig, ...grpc.CallOption) (*enterprisev1.ClusterConfig, error) {
	return nil, c.err
}

func TestUpgradeStartPersistenceFailureIsReturned(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	failure := errors.New("status write failed")
	client := &upgradeStatusTestClient{ClientInterface: ctl.octeliumC, resources: &failingUpgradeStatusClient{ResourceServiceClient: ctl.octeliumC.EnterpriseC(), err: failure}}
	assert.ErrorIs(t, NewController(client, fake.NewSimpleClientset()).doUpgrade(t.Context(), cc), failure)
}

func TestUpgradeWatcherEventRecordsFailureAsynchronously(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	old := pbutils.Clone(cc).(*enterprisev1.ClusterConfig)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State:   enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request: &enterprisev1.UpgradeClusterRequest_Request{PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{}},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	require.NoError(t, ctl.OnUpdate(t.Context(), cc, old))
	require.Eventually(t, func() bool {
		current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
		return err == nil && current.Status.UpgradeRequest == nil && current.Status.TotalFailedUpgrades == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func TestSuccessfulEnterpriseUpgradeReconcilesClusterManager(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request: &enterprisev1.UpgradeClusterRequest_Request{
			Core:              &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"},
			PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "v1.2.3"},
			PackageCordium:    &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{Version: "v1.2.3"},
		},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	k8sC := ctl.k8sC.(*fake.Clientset)
	var completed []string
	k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
		job.UID = types.UID("job-" + job.Name)
		completed = append(completed, job.Spec.Template.Spec.Containers[0].Name)
		job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}
		err := k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
		return true, job, err
	})
	k8sC.PrependReactor("create", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
		current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
		require.NoError(t, err)
		assert.Nil(t, current.Status.UpgradeRequest)
		assert.EqualValues(t, 1, current.Status.TotalSuccessfulUpgrades)
		assert.Equal(t, []string{"octelium-genesis", "octeliumee-genesis", "cordium-genesis"}, completed)
		return false, nil, nil
	})
	require.NoError(t, ctl.doUpgrade(t.Context(), cc))
	deployments, err := k8sC.AppsV1().Deployments(vutils.K8sNS).List(t.Context(), k8smetav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, deployments.Items, 1)
	assert.Contains(t, deployments.Items[0].Spec.Template.Spec.Containers[0].Image, "v1.2.3")
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.EqualValues(t, 1, current.Status.TotalSuccessfulUpgrades)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, current.Status.LastUpgradeRequests[0].State)
}

type upgradeReloadTestUtils struct {
	inner    octeliumc.EnterpriseV1Utils
	failures int
	err      error
}

func (u *upgradeReloadTestUtils) GetClusterConfig(ctx context.Context) (*enterprisev1.ClusterConfig, error) {
	if u.failures > 0 {
		u.failures--
		return nil, u.err
	}
	return u.inner.GetClusterConfig(ctx)
}

type upgradeReloadTestClient struct {
	octeliumc.ClientInterface
	utils octeliumc.EnterpriseV1Utils
}

func (c *upgradeReloadTestClient) EnterpriseV1Utils() octeliumc.EnterpriseV1Utils { return c.utils }

func TestUpgradeReloadFailureIsReturned(t *testing.T) {
	for _, failJob := range []bool{false, true} {
		ctl, cc := newUpgradeControllerTest(t)
		failure := errors.New("upgrade status reload unavailable")
		base := ctl.octeliumC
		ctl.octeliumC = &upgradeReloadTestClient{ClientInterface: base, utils: &upgradeReloadTestUtils{inner: base.EnterpriseV1Utils(), failures: 1, err: failure}}
		request := &enterprisev1.UpgradeClusterRequest_Request{}
		if failJob {
			request.Core = &enterprisev1.UpgradeClusterRequest_Request_Core{}
		}
		cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED, Request: request}
		cc, err := base.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
		require.NoError(t, err)
		assert.ErrorIs(t, ctl.doUpgrade(t.Context(), cc), failure)
	}
}

func TestUpgradeAsyncFallbackRecordsFailureAfterReloadRecovers(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	base := ctl.octeliumC
	ctl.octeliumC = &upgradeReloadTestClient{ClientInterface: base, utils: &upgradeReloadTestUtils{
		inner: base.EnterpriseV1Utils(), failures: 1, err: status.Error(codes.Unavailable, "transient status reload failure"),
	}}
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State:   enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request: &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{}},
	}
	cc, err := base.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	ctl.update(cc)
	require.Eventually(t, func() bool {
		current, err := base.EnterpriseC().GetClusterConfig(t.Context(), &rmetav1.GetOptions{Name: cc.Metadata.Name})
		return err == nil && current.Status.UpgradeRequest == nil && current.Status.TotalFailedUpgrades == 1 &&
			len(current.Status.LastUpgradeRequests) == 1 && current.Status.LastUpgradeRequests[0].State == enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
	}, 5*time.Second, 10*time.Millisecond)
}

func assertNoK8sWrites(t *testing.T, k8sC *fake.Clientset) {
	t.Helper()
	for _, action := range k8sC.Actions() {
		assert.Contains(t, []string{"get", "list", "watch"}, action.GetVerb(), action.GetResource().Resource)
	}
}

func TestRecoverInterruptedUpgrade(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.TotalFailedUpgrades = 2
	cc.Status.TotalSuccessfulUpgrades = 4
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
		CreatedAt: pbutils.Now(),
		Request:   &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"}},
		Regions: []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region{
			{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS},
			{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, Message: "original failure"},
			{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING},
			{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED},
		},
	}
	_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	require.NoError(t, ctl.doReconcile(t.Context()))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.UpgradeRequest)
	assert.EqualValues(t, 3, current.Status.TotalFailedUpgrades)
	assert.EqualValues(t, 4, current.Status.TotalSuccessfulUpgrades)
	require.Len(t, current.Status.LastUpgradeRequests, 1)
	history := current.Status.LastUpgradeRequests[0]
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, history.State)
	assert.NotNil(t, history.DoneAt)
	assert.Equal(t, "v1.2.3", history.Request.Core.Version)
	require.Len(t, history.Regions, 4)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, history.Regions[0].State)
	assert.Empty(t, history.Regions[0].Message)
	assert.Equal(t, "original failure", history.Regions[1].Message)
	for _, rgn := range history.Regions[2:] {
		assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, rgn.State)
		assert.Equal(t, interruptedRequestMessage, rgn.Message)
		assert.NotNil(t, rgn.DoneAt)
	}
	assertNoK8sWrites(t, ctl.k8sC.(*fake.Clientset))
}

func TestRecoverInterruptedJoin(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc = setTestJoinRequest(t, ctl, cc)
	cc.Status.JoinRequest.State = enterprisev1.ClusterConfig_Status_JoinRequest_JOINING
	_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	require.NoError(t, ctl.doReconcile(t.Context()))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.JoinRequest)
	require.Len(t, current.Status.LastJoinRequests, 1)
	history := current.Status.LastJoinRequests[0]
	assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_FAILED, history.State)
	assert.Equal(t, interruptedRequestMessage, history.Message)
	assert.NotNil(t, history.DoneAt)
	assert.Empty(t, ctl.k8sC.(*fake.Clientset).Actions())
}

func TestReconcileKeepsClusterWithoutRequests(t *testing.T) {
	ctl, before := newUpgradeControllerTest(t)
	require.NoError(t, ctl.doReconcile(t.Context()))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.True(t, pbutils.IsEqual(before, current))
	assert.Empty(t, ctl.k8sC.(*fake.Clientset).Actions())
}

func TestReconcileStartsPendingRequests(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	ctl.newRegionK8sC = func(ctx context.Context, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error) {
		return nil, errors.New("unreachable Kubernetes cluster")
	}
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request:   &enterprisev1.UpgradeClusterRequest_Request{},
	}
	_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	cc, err = ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	setTestJoinRequest(t, ctl, cc)

	require.NoError(t, ctl.doReconcile(t.Context()))
	require.Eventually(t, func() bool {
		current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
		return err == nil && current.Status.UpgradeRequest == nil && current.Status.JoinRequest == nil &&
			current.Status.TotalSuccessfulUpgrades == 1 && len(current.Status.LastJoinRequests) == 1 &&
			current.Status.LastJoinRequests[0].State == enterprisev1.ClusterConfig_Status_JoinRequest_FAILED
	}, 5*time.Second, 10*time.Millisecond)
}

func TestRecoverInterruptedRetriesUnavailableStatus(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State:   enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
		Request: &enterprisev1.UpgradeClusterRequest_Request{},
	}
	_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	base := ctl.octeliumC
	ctl.octeliumC = &upgradeReloadTestClient{ClientInterface: base, utils: &upgradeReloadTestUtils{inner: base.EnterpriseV1Utils(), failures: 1, err: errors.New("status unavailable")}}
	ctl.RunReconciler(t.Context())
	assert.Eventually(t, func() bool {
		current, err := base.EnterpriseV1Utils().GetClusterConfig(t.Context())
		return err == nil && current.Status.UpgradeRequest == nil && current.Status.TotalFailedUpgrades == 1
	}, 10*time.Second, 50*time.Millisecond)
}

func TestRecoveredUpgradeCanBeRequestedAgain(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State:   enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
		Request: &enterprisev1.UpgradeClusterRequest_Request{},
	}
	_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	require.NoError(t, ctl.doReconcile(t.Context()))
	cc, err = ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		CreatedAt: pbutils.Now(),
		Request:   &enterprisev1.UpgradeClusterRequest_Request{},
	}
	cc, err = ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	require.NoError(t, ctl.doUpgrade(t.Context(), cc))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.UpgradeRequest)
	assert.EqualValues(t, 1, current.Status.TotalSuccessfulUpgrades)
	assert.EqualValues(t, 1, current.Status.TotalFailedUpgrades)
}

func TestRolloutAfterSuccessfulUpgradeIsNotInterrupted(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request: &enterprisev1.UpgradeClusterRequest_Request{
			PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "v1.2.3"},
		},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	k8sC := ctl.k8sC.(*fake.Clientset)
	k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
		job.UID = types.UID("job-" + job.Name)
		job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}
		return true, job, k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
	})
	recovered := false
	k8sC.PrependReactor("create", "deployments", func(ktesting.Action) (bool, runtime.Object, error) {
		restarted := NewController(ctl.octeliumC, fake.NewSimpleClientset())
		require.NoError(t, restarted.doReconcile(t.Context()))
		recovered = true
		return false, nil, nil
	})
	require.NoError(t, ctl.doUpgrade(t.Context(), cc))
	require.True(t, recovered)
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.EqualValues(t, 1, current.Status.TotalSuccessfulUpgrades)
	assert.Zero(t, current.Status.TotalFailedUpgrades)
	require.Len(t, current.Status.LastUpgradeRequests, 1)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, current.Status.LastUpgradeRequests[0].State)
}

func TestUpgradeKeepsRecordOfReplacementProcess(t *testing.T) {
	for name, finish := range map[string]func(*batchv1.Job) error{
		"job succeeds": func(job *batchv1.Job) error {
			job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}
			return nil
		},
		"job fails": func(*batchv1.Job) error { return errors.New("Job creation denied") },
	} {
		t.Run(name, func(t *testing.T) {
			ctl, cc := newUpgradeControllerTest(t)
			cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
				State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
				CreatedAt: pbutils.Now(),
				Request: &enterprisev1.UpgradeClusterRequest_Request{
					PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "v1.2.3"},
				},
			}
			cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
			require.NoError(t, err)
			k8sC := ctl.k8sC.(*fake.Clientset)
			k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
				restarted := NewController(ctl.octeliumC, fake.NewSimpleClientset())
				require.NoError(t, restarted.doReconcile(t.Context()))
				job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
				job.UID = types.UID("job-" + job.Name)
				if err := finish(job); err != nil {
					return true, nil, err
				}
				return true, job, k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
			})
			require.NoError(t, ctl.doUpgrade(t.Context(), cc))
			current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
			require.NoError(t, err)
			assert.Nil(t, current.Status.UpgradeRequest)
			assert.EqualValues(t, 1, current.Status.TotalFailedUpgrades)
			assert.Zero(t, current.Status.TotalSuccessfulUpgrades)
			require.Len(t, current.Status.LastUpgradeRequests, 1)
			deployments, err := k8sC.AppsV1().Deployments(vutils.K8sNS).List(t.Context(), k8smetav1.ListOptions{})
			require.NoError(t, err)
			assert.Empty(t, deployments.Items)
		})
	}
}

type flakyUpgradeStatusClient struct {
	renterprisev1.ResourceServiceClient
	fail            func(cc *enterprisev1.ClusterConfig) error
	failAfterCommit func(cc *enterprisev1.ClusterConfig) error
}

func (c *flakyUpgradeStatusClient) UpdateClusterConfig(ctx context.Context, cc *enterprisev1.ClusterConfig, opts ...grpc.CallOption) (*enterprisev1.ClusterConfig, error) {
	if c.fail != nil {
		if err := c.fail(cc); err != nil {
			return nil, err
		}
	}
	ret, err := c.ResourceServiceClient.UpdateClusterConfig(ctx, cc, opts...)
	if err == nil && c.failAfterCommit != nil {
		if err := c.failAfterCommit(cc); err != nil {
			return nil, err
		}
	}
	return ret, err
}

func newFlakyUpgradeStatusClient(ctl *Controller) *flakyUpgradeStatusClient {
	ret := &flakyUpgradeStatusClient{ResourceServiceClient: ctl.octeliumC.EnterpriseC()}
	ctl.octeliumC = &upgradeStatusTestClient{ClientInterface: ctl.octeliumC, resources: ret}
	return ret
}

func completeTestJobs(k8sC *fake.Clientset, onCreate func()) {
	k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
		job.UID = types.UID("job-" + job.Name)
		job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}
		if onCreate != nil {
			onCreate()
		}
		return true, job, k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
	})
}

func setTestDefaultRegionVersions(t *testing.T, ctl *Controller, versions map[string]*corev1.Region_Status_VersionInfo) *corev1.Region {
	t.Helper()
	rgn, err := ctl.octeliumC.CoreC().GetRegion(t.Context(), &rmetav1.GetOptions{Name: "default"})
	require.NoError(t, err)
	rgn.Status.VersionInfoMap = versions
	rgn, err = ctl.octeliumC.CoreC().UpdateRegion(t.Context(), rgn)
	require.NoError(t, err)
	return rgn
}

func TestUpgradeRecordRetriesTransientStatusFailures(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.TotalSuccessfulUpgrades = 128
	cc.Status.TotalFailedUpgrades = 1
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request:   &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"}},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)

	connReset := status.Error(codes.Internal, "read tcp 10.42.0.93:40420->10.43.159.75:5432: read: connection reset by peer")
	connRefused := status.Error(codes.Internal, "dial tcp 10.43.159.75:5432: connect: connection refused")
	isDone := false
	var reportFailures, recordFailures int
	flaky := newFlakyUpgradeStatusClient(ctl)
	flaky.fail = func(cc *enterprisev1.ClusterConfig) error {
		req := cc.Status.UpgradeRequest
		switch {
		case !isDone:
			return nil
		case req == nil && recordFailures < 3:
			recordFailures++
			return connRefused
		case req != nil && reportFailures < 2:
			reportFailures++
			return connReset
		default:
			return nil
		}
	}
	completeTestJobs(ctl.k8sC.(*fake.Clientset), func() { isDone = true })

	require.NoError(t, ctl.doUpgrade(t.Context(), cc))
	assert.Equal(t, 2, reportFailures)
	assert.Equal(t, 3, recordFailures)

	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.UpgradeRequest)
	assert.EqualValues(t, 129, current.Status.TotalSuccessfulUpgrades)
	assert.EqualValues(t, 1, current.Status.TotalFailedUpgrades)
	require.Len(t, current.Status.LastUpgradeRequests, 1)
	last := current.Status.LastUpgradeRequests[0]
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last.State)
	assert.NotNil(t, last.DoneAt)
	require.Len(t, last.Regions, 1)
	assert.Equal(t, "default", last.Regions[0].RegionRef.Name)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last.Regions[0].State)
	assert.NotNil(t, last.Regions[0].DoneAt)
}

func TestUpgradeStartSurvivesLostAcknowledgement(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request:   &enterprisev1.UpgradeClusterRequest_Request{},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)

	isLost := false
	flaky := newFlakyUpgradeStatusClient(ctl)
	flaky.failAfterCommit = func(cc *enterprisev1.ClusterConfig) error {
		if !isLost && cc.Status.UpgradeRequest.GetState() == enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING {
			isLost = true
			return status.Error(codes.Unavailable, "connection reset")
		}
		return nil
	}

	require.NoError(t, ctl.doUpgrade(t.Context(), cc))
	assert.True(t, isLost)
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.UpgradeRequest)
	assert.EqualValues(t, 1, current.Status.TotalSuccessfulUpgrades)
	assert.Zero(t, current.Status.TotalFailedUpgrades)
}

func TestReconcileRecordsInterruptedUpgradeFromRegionVersions(t *testing.T) {
	for name, tc := range map[string]struct {
		version   string
		isSuccess bool
	}{
		"applied":     {version: "b-w207", isSuccess: true},
		"not applied": {version: "b-w206"},
	} {
		t.Run(name, func(t *testing.T) {
			ctl, cc := newUpgradeControllerTest(t)
			createdAt := time.Now().Add(-48 * time.Hour)
			setAt := timestamppb.New(createdAt.Add(78 * time.Second))
			rgn := setTestDefaultRegionVersions(t, ctl, map[string]*corev1.Region_Status_VersionInfo{
				regionutils.PackageCore: {Package: regionutils.PackageCore, Version: tc.version, SetAt: setAt},
			})

			cc.Status.TotalSuccessfulUpgrades = 128
			cc.Status.TotalFailedUpgrades = 1
			cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
				CreatedAt: timestamppb.New(createdAt),
				State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
				Request:   &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "b-w207"}},
				Regions: []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region{
					{RegionRef: umetav1.GetObjectReference(rgn), State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING},
				},
			}
			_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
			require.NoError(t, err)

			require.NoError(t, ctl.doReconcile(t.Context()))
			current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
			require.NoError(t, err)
			assert.Nil(t, current.Status.UpgradeRequest)
			require.Len(t, current.Status.LastUpgradeRequests, 1)
			last := current.Status.LastUpgradeRequests[0]
			require.Len(t, last.Regions, 1)
			assert.NotNil(t, last.DoneAt)
			if tc.isSuccess {
				assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last.State)
				assert.EqualValues(t, 129, current.Status.TotalSuccessfulUpgrades)
				assert.EqualValues(t, 1, current.Status.TotalFailedUpgrades)
				assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last.Regions[0].State)
				assert.True(t, pbutils.IsEqual(setAt, last.Regions[0].DoneAt))
				assert.Empty(t, last.Regions[0].Message)
			} else {
				assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, last.State)
				assert.EqualValues(t, 128, current.Status.TotalSuccessfulUpgrades)
				assert.EqualValues(t, 2, current.Status.TotalFailedUpgrades)
				assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, last.Regions[0].State)
				assert.Equal(t, interruptedRequestMessage, last.Regions[0].Message)
			}
			assertNoK8sWrites(t, ctl.k8sC.(*fake.Clientset))

			require.NoError(t, ctl.doReconcile(t.Context()))
			again, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
			require.NoError(t, err)
			assert.True(t, pbutils.IsEqual(current.Status, again.Status))
		})
	}
}

func TestReconcileWaitsForRunningGenesisJob(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
		Request:   &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"}},
	}
	_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)

	k8sC := ctl.k8sC.(*fake.Clientset)
	job := &batchv1.Job{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:              "octelium-genesis-upgrade-test",
			Namespace:         vutils.K8sNS,
			Labels:            map[string]string{"octelium.com/component": "genesis"},
			CreationTimestamp: k8smetav1.Now(),
		},
		Status: batchv1.JobStatus{Active: 1},
	}
	require.NoError(t, k8sC.Tracker().Add(job))

	require.NoError(t, ctl.doReconcile(t.Context()))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	require.NotNil(t, current.Status.UpgradeRequest)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING, current.Status.UpgradeRequest.State)

	setTestDefaultRegionVersions(t, ctl, map[string]*corev1.Region_Status_VersionInfo{
		regionutils.PackageCore: {Package: regionutils.PackageCore, Version: "v1.2.3", SetAt: pbutils.Now()},
	})
	job.Status = batchv1.JobStatus{Succeeded: 1, Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}}
	require.NoError(t, k8sC.Tracker().Update(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace))

	require.NoError(t, ctl.doReconcile(t.Context()))
	current, err = ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.UpgradeRequest)
	assert.EqualValues(t, 1, current.Status.TotalSuccessfulUpgrades)
	require.Len(t, current.Status.LastUpgradeRequests, 1)
	last := current.Status.LastUpgradeRequests[0]
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last.State)
	require.Len(t, last.Regions, 1)
	assert.Equal(t, "default", last.Regions[0].RegionRef.Name)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last.Regions[0].State)
}

func TestReconcileRecordsReplacedUpgradeRequests(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	base := time.Now()
	at := func(d time.Duration) *timestamppb.Timestamp {
		return timestamppb.New(base.Add(d))
	}

	rgn := setTestDefaultRegionVersions(t, ctl, map[string]*corev1.Region_Status_VersionInfo{
		regionutils.PackageCore: {Package: regionutils.PackageCore, Version: "v1.2.3", SetAt: at(70 * time.Minute)},
	})

	core := func(version string) *enterprisev1.UpgradeClusterRequest_Request {
		return &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{Version: version}}
	}

	cc.Status.TotalSuccessfulUpgrades = 1
	cc.Status.LastUpgradeRequests = []*enterprisev1.ClusterConfig_Status_UpgradeRequest{
		{
			CreatedAt: at(100 * time.Minute), DoneAt: at(101 * time.Minute),
			State:   enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS,
			Request: &enterprisev1.UpgradeClusterRequest_Request{PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "v1.2.4"}},
		},
		{
			CreatedAt: at(60 * time.Minute),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
			Request:   core("v1.2.3"),
			Regions: []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region{
				{RegionRef: umetav1.GetObjectReference(rgn), State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING},
			},
		},
		{
			CreatedAt: at(30 * time.Minute),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
			Request:   core("v1.2.2"),
		},
		{
			CreatedAt: at(10 * time.Minute),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
			Request:   core("v1.2.3"),
		},
	}
	_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)

	require.NoError(t, ctl.doReconcile(t.Context()))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.UpgradeRequest)
	assert.EqualValues(t, 2, current.Status.TotalSuccessfulUpgrades)
	assert.EqualValues(t, 2, current.Status.TotalFailedUpgrades)

	history := current.Status.LastUpgradeRequests
	require.Len(t, history, 4)
	assert.True(t, pbutils.IsEqual(cc.Status.LastUpgradeRequests[0], history[0]))

	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, history[1].State)
	require.Len(t, history[1].Regions, 1)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, history[1].Regions[0].State)
	assert.True(t, pbutils.IsEqual(at(70*time.Minute), history[1].Regions[0].DoneAt))

	for _, itm := range history[2:] {
		assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, itm.State)
		assert.NotNil(t, itm.DoneAt)
		require.Len(t, itm.Regions, 1)
		assert.Equal(t, "default", itm.Regions[0].RegionRef.Name)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, itm.Regions[0].State)
		assert.Equal(t, supersededRequestMessage, itm.Regions[0].Message)
	}

	assert.Empty(t, ctl.k8sC.(*fake.Clientset).Actions())
}

func TestReconcileSkipsRequestsHandledByThisProcess(t *testing.T) {
	for _, isCurrentOwned := range []bool{true, false} {
		ctl, cc := newUpgradeControllerTest(t)
		cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
			CreatedAt: pbutils.Now(),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
			Request:   &enterprisev1.UpgradeClusterRequest_Request{},
		}
		cc.Status.LastUpgradeRequests = []*enterprisev1.ClusterConfig_Status_UpgradeRequest{{
			CreatedAt: timestamppb.New(time.Now().Add(-time.Hour)),
			State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
			Request:   &enterprisev1.UpgradeClusterRequest_Request{},
		}}
		cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
		require.NoError(t, err)

		owned := cc.Status.LastUpgradeRequests[0]
		if isCurrentOwned {
			owned = cc.Status.UpgradeRequest
		}
		require.True(t, ctl.acquire(taskUpgrade, owned.CreatedAt))
		require.False(t, ctl.acquire(taskUpgrade, owned.CreatedAt))

		require.NoError(t, ctl.doReconcile(t.Context()))
		current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
		require.NoError(t, err)
		require.NotNil(t, current.Status.UpgradeRequest)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING, current.Status.UpgradeRequest.State)
		require.Len(t, current.Status.LastUpgradeRequests, 1)
		if isCurrentOwned {
			assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, current.Status.LastUpgradeRequests[0].State)
		} else {
			assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING, current.Status.LastUpgradeRequests[0].State)
		}

		ctl.release(taskUpgrade)
		require.NoError(t, ctl.doReconcile(t.Context()))
		current, err = ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
		require.NoError(t, err)
		assert.Nil(t, current.Status.UpgradeRequest)
		require.Len(t, current.Status.LastUpgradeRequests, 2)
		for _, itm := range current.Status.LastUpgradeRequests {
			assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, itm.State)
		}
		assert.EqualValues(t, 2, current.Status.TotalFailedUpgrades)
	}
}

func TestUpgradeIsHandledOnceAtATime(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request:   &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"}},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)

	require.True(t, ctl.acquire(taskUpgrade, timestamppb.New(time.Now().Add(-time.Hour))))
	require.NoError(t, ctl.doUpgrade(t.Context(), cc))
	require.NoError(t, ctl.doReconcile(t.Context()))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	require.NotNil(t, current.Status.UpgradeRequest)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED, current.Status.UpgradeRequest.State)
	assert.Empty(t, ctl.k8sC.(*fake.Clientset).Actions())

	ctl.release(taskUpgrade)
	completeTestJobs(ctl.k8sC.(*fake.Clientset), nil)
	require.NoError(t, ctl.doUpgrade(t.Context(), cc))
	current, err = ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.UpgradeRequest)
	assert.EqualValues(t, 1, current.Status.TotalSuccessfulUpgrades)
	assert.False(t, ctl.isBusy(taskUpgrade))
}

func TestUpgradeRecordsRequestReplacedWhileRunning(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request:   &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"}},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)

	replacement := &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: timestamppb.New(time.Now().Add(time.Second)),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request:   &enterprisev1.UpgradeClusterRequest_Request{},
	}
	completeTestJobs(ctl.k8sC.(*fake.Clientset), func() {
		current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
		require.NoError(t, err)
		current.Status.LastUpgradeRequests = prependUpgradeHistory(current.Status.LastUpgradeRequests, current.Status.UpgradeRequest)
		current.Status.UpgradeRequest = replacement
		_, err = ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), current)
		require.NoError(t, err)
	})

	require.NoError(t, ctl.doUpgrade(t.Context(), cc))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.True(t, pbutils.IsEqual(replacement, current.Status.UpgradeRequest))
	assert.EqualValues(t, 1, current.Status.TotalSuccessfulUpgrades)
	require.Len(t, current.Status.LastUpgradeRequests, 1)
	last := current.Status.LastUpgradeRequests[0]
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last.State)
	assert.NotNil(t, last.DoneAt)
	require.Len(t, last.Regions, 1)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last.Regions[0].State)
}

func TestReconcileRecordsJoinsFromRegions(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc = setTestJoinRequest(t, ctl, cc)
	cc.Status.JoinRequest.State = enterprisev1.ClusterConfig_Status_JoinRequest_JOINING
	cc.Status.LastJoinRequests = []*enterprisev1.ClusterConfig_Status_JoinRequest{{
		CreatedAt: timestamppb.New(time.Now().Add(-time.Hour)),
		State:     enterprisev1.ClusterConfig_Status_JoinRequest_JOINING,
		Request:   &enterprisev1.JoinClusterRequest{Name: "east", Kubernetes: getTestJoinKubernetes()},
	}}
	_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)

	rgn := &corev1.Region{
		Metadata: &metav1.Metadata{Name: "west"},
		Spec:     &corev1.Region_Spec{},
		Status: &corev1.Region_Status{
			Index: 1,
			VersionInfoMap: map[string]*corev1.Region_Status_VersionInfo{
				regionutils.PackageEnterprise: {Package: regionutils.PackageEnterprise, Version: "1.0.0", SetAt: pbutils.Now()},
			},
		},
	}
	require.NoError(t, regionutils.SetRegionExtInfo(rgn, &enterprisev1.RegionExtInfo{Kubernetes: getTestJoinKubernetes()}))
	rgn, err = ctl.octeliumC.CoreC().CreateRegion(t.Context(), rgn)
	require.NoError(t, err)

	require.NoError(t, ctl.doReconcile(t.Context()))
	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.JoinRequest)
	require.Len(t, current.Status.LastJoinRequests, 2)

	joined := current.Status.LastJoinRequests[0]
	assert.Equal(t, "west", joined.Request.Name)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS, joined.State)
	assert.Empty(t, joined.Message)
	assert.NotNil(t, joined.DoneAt)
	require.NotNil(t, joined.RegionRef)
	assert.Equal(t, rgn.Metadata.Uid, joined.RegionRef.Uid)

	replaced := current.Status.LastJoinRequests[1]
	assert.Equal(t, "east", replaced.Request.Name)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_FAILED, replaced.State)
	assert.Equal(t, supersededRequestMessage, replaced.Message)
	assert.NotNil(t, replaced.DoneAt)

	assert.Empty(t, ctl.k8sC.(*fake.Clientset).Actions())
}

func TestUpdateStatusRetriesTransientErrors(t *testing.T) {
	ctl, _ := newUpgradeControllerTest(t)
	flaky := newFlakyUpgradeStatusClient(ctl)

	calls := 0
	for i, code := range []codes.Code{codes.Internal, codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.OutOfRange} {
		calls = 0
		flaky.fail = func(*enterprisev1.ClusterConfig) error {
			calls++
			if calls == 1 {
				return status.Error(code, "transient")
			}
			return nil
		}
		require.NoError(t, ctl.updateStatus(t.Context(), func(cc *enterprisev1.ClusterConfig) bool {
			cc.Status.TotalSuccessfulUpgrades = uint64(i + 1)
			return true
		}), code.String())
		assert.Equal(t, 2, calls, code.String())
		current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
		require.NoError(t, err)
		assert.EqualValues(t, i+1, current.Status.TotalSuccessfulUpgrades, code.String())
	}

	calls = 0
	flaky.fail = func(*enterprisev1.ClusterConfig) error {
		calls++
		return status.Error(codes.InvalidArgument, "invalid status")
	}
	err := ctl.updateStatus(t.Context(), func(cc *enterprisev1.ClusterConfig) bool { return true })
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Equal(t, 1, calls)

	flaky.fail = func(*enterprisev1.ClusterConfig) error {
		return status.Error(codes.Internal, "database is down")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	err = ctl.updateStatus(ctx, func(cc *enterprisev1.ClusterConfig) bool { return true })
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Error(t, ctx.Err())
}
