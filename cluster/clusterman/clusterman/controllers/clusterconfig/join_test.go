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
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	"github.com/octelium/octelium/apis/cluster/cbootstrapv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func getTestJoinKubernetes() *enterprisev1.JoinClusterRequest_Kubernetes {
	return &enterprisev1.JoinClusterRequest_Kubernetes{
		Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
			Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_FromSecret{
					FromSecret: "west-kubeconfig",
				},
			},
		},
	}
}

func setTestJoinRequest(t *testing.T, ctl *Controller, cc *enterprisev1.ClusterConfig) *enterprisev1.ClusterConfig {
	t.Helper()
	cc.Status.JoinRequest = &enterprisev1.ClusterConfig_Status_JoinRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED,
		Request: &enterprisev1.JoinClusterRequest{
			Name:       "west",
			Kubernetes: getTestJoinKubernetes(),
		},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	return cc
}

func TestJoinIgnoresAbsentAndTerminalRequests(t *testing.T) {
	ctl := NewController(nil, nil)
	for _, req := range []*enterprisev1.ClusterConfig_Status_JoinRequest{
		nil,
		{State: enterprisev1.ClusterConfig_Status_JoinRequest_JOINING},
		{State: enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS},
		{State: enterprisev1.ClusterConfig_Status_JoinRequest_FAILED},
	} {
		cc := &enterprisev1.ClusterConfig{Status: &enterprisev1.ClusterConfig_Status{JoinRequest: req}}
		require.NoError(t, ctl.doJoin(t.Context(), cc))
		require.NoError(t, ctl.OnUpdate(t.Context(), cc, pbutils.Clone(cc).(*enterprisev1.ClusterConfig)))
	}
}

func TestJoinFailureIsRecorded(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	failure := errors.New("the kubeconfig Secret is not found")
	ctl.newRegionK8sC = func(ctx context.Context, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error) {
		return nil, failure
	}

	cc.Status.LastJoinRequests = []*enterprisev1.ClusterConfig_Status_JoinRequest{{State: enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS}}
	cc = setTestJoinRequest(t, ctl, cc)

	assert.ErrorContains(t, ctl.doJoin(t.Context(), cc), failure.Error())

	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Nil(t, current.Status.JoinRequest)
	require.Len(t, current.Status.LastJoinRequests, 2)
	last := current.Status.LastJoinRequests[0]
	assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_FAILED, last.State)
	assert.Contains(t, last.Message, failure.Error())
	assert.NotNil(t, last.DoneAt)
	assert.Nil(t, last.RegionRef)
	assert.Equal(t, "west", last.Request.Name)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS, current.Status.LastJoinRequests[1].State)

	require.NoError(t, ctl.doJoin(t.Context(), cc))
	current, err = ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Len(t, current.Status.LastJoinRequests, 2)
}

func TestJoinSuccessIsRecorded(t *testing.T) {
	ctx := t.Context()
	ctl, cc := newUpgradeControllerTest(t)

	ctl.k8sC = fake.NewSimpleClientset(
		&k8scorev1.Namespace{ObjectMeta: k8smetav1.ObjectMeta{Name: "kube-system", UID: "default-cluster"}},
		&k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "octelium-postgres", Namespace: vutils.K8sNS}},
		&k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "octelium-redis", Namespace: vutils.K8sNS}},
		&k8scorev1.Secret{ObjectMeta: k8smetav1.ObjectMeta{Name: "sys-init-kek", Namespace: vutils.K8sNS}},
	)

	targetC := fake.NewSimpleClientset(&k8scorev1.Namespace{ObjectMeta: k8smetav1.ObjectMeta{Name: "kube-system", UID: "west-cluster"}})
	targetC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
		job.UID = types.UID("job-" + job.Name)
		job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}
		err := targetC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
		return true, job, err
	})
	ctl.newRegionK8sC = func(ctx context.Context, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error) {
		return targetC, nil
	}

	_, err := ctl.octeliumC.CoreC().CreateSecret(ctx, &corev1.Secret{
		Metadata: &metav1.Metadata{Name: "sys:bootstrap-config"},
		Spec:     &corev1.Secret_Spec{},
		Status:   &corev1.Secret_Status{},
		Data: &corev1.Secret_Data{Type: &corev1.Secret_Data_ValueBytes{ValueBytes: pbutils.MarshalMust(&cbootstrapv1.Config{
			Spec: &cbootstrapv1.Config_Spec{
				PrimaryStorage: &cbootstrapv1.Config_Spec_PrimaryStorage{
					Type: &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql_{
						Postgresql: &cbootstrapv1.Config_Spec_PrimaryStorage_Postgresql{Host: "postgres"},
					},
				},
				SecondaryStorage: &cbootstrapv1.Config_Spec_SecondaryStorage{
					Type: &cbootstrapv1.Config_Spec_SecondaryStorage_Redis_{
						Redis: &cbootstrapv1.Config_Spec_SecondaryStorage_Redis{Host: "redis"},
					},
				},
			},
		})}},
	})
	require.NoError(t, err)

	defaultRgn, err := ctl.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: "default"})
	require.NoError(t, err)
	defaultRgn.Status.VersionInfoMap = map[string]*corev1.Region_Status_VersionInfo{
		regionutils.PackageCore: {Package: regionutils.PackageCore, Version: "2.0.0"},
	}
	_, err = ctl.octeliumC.CoreC().UpdateRegion(ctx, defaultRgn)
	require.NoError(t, err)

	rgn := &corev1.Region{
		Metadata: &metav1.Metadata{Name: "west"},
		Spec:     &corev1.Region_Spec{},
		Status: &corev1.Region_Status{
			Index: 1,
			VersionInfoMap: map[string]*corev1.Region_Status_VersionInfo{
				regionutils.PackageEnterprise: {Package: regionutils.PackageEnterprise, Version: "1.0.0"},
			},
		},
	}
	require.NoError(t, regionutils.SetRegionExtInfo(rgn, &enterprisev1.RegionExtInfo{Kubernetes: getTestJoinKubernetes()}))
	rgn, err = ctl.octeliumC.CoreC().CreateRegion(ctx, rgn)
	require.NoError(t, err)

	cc = setTestJoinRequest(t, ctl, cc)
	require.NoError(t, ctl.doJoin(ctx, cc))

	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	require.NoError(t, err)
	assert.Nil(t, current.Status.JoinRequest)
	require.Len(t, current.Status.LastJoinRequests, 1)
	last := current.Status.LastJoinRequests[0]
	assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS, last.State)
	assert.Empty(t, last.Message)
	assert.NotNil(t, last.DoneAt)
	require.NotNil(t, last.RegionRef)
	assert.Equal(t, rgn.Metadata.Uid, last.RegionRef.Uid)
	assert.Equal(t, "west", last.RegionRef.Name)

	jobs, err := targetC.BatchV1().Jobs(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, jobs.Items, 1)
	assert.Equal(t, []string{"join", "--core-version", "2.0.0"}, jobs.Items[0].Spec.Template.Spec.Containers[0].Args)
}

func TestJoinSkipsReplacedRequest(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc = setTestJoinRequest(t, ctl, cc)
	stale := pbutils.Clone(cc).(*enterprisev1.ClusterConfig)
	stale.Status.JoinRequest.CreatedAt = timestamppb.New(time.Now().Add(-time.Hour))

	ctl.newRegionK8sC = func(ctx context.Context, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error) {
		return nil, errors.New("must not be called")
	}

	require.NoError(t, ctl.doJoin(t.Context(), stale))

	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	require.NotNil(t, current.Status.JoinRequest)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED, current.Status.JoinRequest.State)
	assert.Empty(t, current.Status.LastJoinRequests)
}

func TestJoinWatcherEventRecordsFailureAsynchronously(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	ctl.newRegionK8sC = func(ctx context.Context, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error) {
		return nil, errors.New("unreachable Kubernetes cluster")
	}
	old := pbutils.Clone(cc).(*enterprisev1.ClusterConfig)
	cc = setTestJoinRequest(t, ctl, cc)
	require.NoError(t, ctl.OnUpdate(t.Context(), cc, old))
	require.Eventually(t, func() bool {
		current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
		return err == nil && current.Status.JoinRequest == nil && len(current.Status.LastJoinRequests) == 1 &&
			current.Status.LastJoinRequests[0].State == enterprisev1.ClusterConfig_Status_JoinRequest_FAILED
	}, 5*time.Second, 10*time.Millisecond)
}

func TestSetUpgradeRegions(t *testing.T) {
	ctl, cc := newUpgradeControllerTest(t)
	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING,
		Request:   &enterprisev1.UpgradeClusterRequest_Request{},
	}
	cc, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)

	regions := []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region{
		{RegionRef: &metav1.ObjectReference{Name: "default"}, State: enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS},
		{RegionRef: &metav1.ObjectReference{Name: "west"}, State: enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING},
	}

	other := pbutils.Clone(cc.Status.UpgradeRequest).(*enterprisev1.ClusterConfig_Status_UpgradeRequest)
	other.CreatedAt = timestamppb.New(time.Now().Add(-time.Hour))
	ctl.setUpgradeRegions(t.Context(), other, regions)
	ctl.setUpgradeRegions(t.Context(), nil, regions)

	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Empty(t, current.Status.UpgradeRequest.Regions)

	ctl.setUpgradeRegions(t.Context(), cc.Status.UpgradeRequest, regions)

	current, err = ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	require.Len(t, current.Status.UpgradeRequest.Regions, 2)
	assert.Equal(t, "west", current.Status.UpgradeRequest.Regions[1].RegionRef.Name)

	current.Status.UpgradeRequest.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS
	_, err = ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), current)
	require.NoError(t, err)
	ctl.setUpgradeRegions(t.Context(), cc.Status.UpgradeRequest, regions[:1])

	current, err = ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.Len(t, current.Status.UpgradeRequest.Regions, 2)
}

func TestUpdateStatusRetriesOnConflict(t *testing.T) {
	ctl, _ := newUpgradeControllerTest(t)

	calls := 0
	require.NoError(t, ctl.updateStatus(t.Context(), func(cc *enterprisev1.ClusterConfig) bool {
		calls++
		if calls == 1 {
			concurrent := pbutils.Clone(cc).(*enterprisev1.ClusterConfig)
			concurrent.Status.TotalFailedUpgrades = 7
			_, err := ctl.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), concurrent)
			require.NoError(t, err)
		}
		cc.Status.TotalSuccessfulUpgrades = 3
		return true
	}))
	assert.Equal(t, 2, calls)

	current, err := ctl.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	assert.EqualValues(t, 3, current.Status.TotalSuccessfulUpgrades)
	assert.EqualValues(t, 7, current.Status.TotalFailedUpgrades)

	calls = 0
	require.NoError(t, ctl.updateStatus(t.Context(), func(cc *enterprisev1.ClusterConfig) bool {
		calls++
		return false
	}))
	assert.Equal(t, 1, calls)

	failure := errors.New("status write failed")
	client := &upgradeStatusTestClient{
		ClientInterface: ctl.octeliumC,
		resources:       &failingUpgradeStatusClient{ResourceServiceClient: ctl.octeliumC.EnterpriseC(), err: failure},
	}
	assert.ErrorIs(t, (&Controller{octeliumC: client}).updateStatus(t.Context(), func(cc *enterprisev1.ClusterConfig) bool {
		return true
	}), failure)
}

func TestJoinHistoryAndMessageHelpers(t *testing.T) {
	var history []*enterprisev1.ClusterConfig_Status_JoinRequest
	for i := range maxLastJoinRequests + 3 {
		req := &enterprisev1.ClusterConfig_Status_JoinRequest{DoneAt: pbutils.Timestamp(time.Unix(int64(i), 0))}
		history = prependJoinHistory(history, req)
		assert.Same(t, req, history[0])
	}
	assert.Len(t, history, maxLastJoinRequests)
	assert.EqualValues(t, 202, history[0].DoneAt.Seconds)
	assert.Equal(t, history, prependJoinHistory(history, nil))

	assert.Equal(t, "failure", truncateStatusMessage("failure"))
	assert.Len(t, truncateStatusMessage(strings.Repeat("a", maxStatusMessageLen+10)), maxStatusMessageLen)

	now := pbutils.Now()
	assert.True(t, isSameTimestamp(nil, nil))
	assert.False(t, isSameTimestamp(now, nil))
	assert.True(t, isSameTimestamp(now, pbutils.Clone(now).(*timestamppb.Timestamp)))
	assert.False(t, isSameJoinRequest(nil, &enterprisev1.ClusterConfig_Status_JoinRequest{}))
	assert.False(t, isSameJoinRequest(
		&enterprisev1.ClusterConfig_Status_JoinRequest{CreatedAt: now, Request: &enterprisev1.JoinClusterRequest{Name: "west"}},
		&enterprisev1.ClusterConfig_Status_JoinRequest{CreatedAt: now, Request: &enterprisev1.JoinClusterRequest{Name: "east"}},
	))
}
