package clusterconfig

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	failure := errors.New("status write failed")
	cc := &enterprisev1.ClusterConfig{Status: &enterprisev1.ClusterConfig_Status{UpgradeRequest: &enterprisev1.ClusterConfig_Status_UpgradeRequest{State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED}}}
	client := &upgradeStatusTestClient{resources: &failingUpgradeStatusClient{err: failure}}
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
		State:   enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request: &enterprisev1.UpgradeClusterRequest_Request{PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "v1.2.3"}},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	k8sC := ctl.k8sC.(*fake.Clientset)
	k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
		job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}
		err := k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
		return true, job, err
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
		inner: base.EnterpriseV1Utils(), failures: 1, err: errors.New("transient status reload failure"),
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
