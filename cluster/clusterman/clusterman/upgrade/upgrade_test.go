package upgrade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/client/octops/commands/install"
	cocteliumc "github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

type upgradeTestClient struct {
	octeliumc.ClientInterface
	utils *upgradeTestCoreUtils
}

func (c *upgradeTestClient) CoreV1Utils() cocteliumc.CoreV1Utils { return c.utils }

type upgradeTestCoreUtils struct {
	config *corev1.ClusterConfig
	err    error
}

func (u *upgradeTestCoreUtils) GetClusterConfig(ctx context.Context) (*corev1.ClusterConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return u.config, u.err
}

func TestNewUpgradeControllerConfiguration(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		cfg := &corev1.ClusterConfig{}
		body, err := json.Marshal(map[string]any{"status": map[string]any{
			"domain":       "example.com",
			"installation": map[string]any{"spiffe": map[string]any{"enable": enabled, "trustDomain": "cluster.test", "csiDriver": map[string]any{"name": "test.csi"}}},
		}})
		require.NoError(t, err)
		require.NoError(t, protojson.Unmarshal(body, cfg))
		client := &upgradeTestClient{utils: &upgradeTestCoreUtils{config: cfg}}
		ctl, err := newController(t.Context(), &UpgradeClusterOpts{OcteliumC: client, K8sC: fake.NewSimpleClientset()})
		require.NoError(t, err)
		assert.Equal(t, "example.com", ctl.domain)
		if enabled {
			require.NotNil(t, ctl.spiffe)
			assert.Equal(t, "test.csi", ctl.spiffe.CSIDriver)
			assert.Equal(t, "cluster.test", ctl.spiffe.TrustDomain)
		} else {
			assert.Nil(t, ctl.spiffe)
		}
		require.NoError(t, UpgradeCluster(t.Context(), &UpgradeClusterOpts{OcteliumC: client, K8sC: fake.NewSimpleClientset()}))
	}
}

func TestNewUpgradeControllerErrors(t *testing.T) {
	_, err := newController(t.Context(), nil)
	require.Error(t, err)
	failure := errors.New("cluster config unavailable")
	client := &upgradeTestClient{utils: &upgradeTestCoreUtils{err: failure}}
	_, err = newController(t.Context(), &UpgradeClusterOpts{OcteliumC: client})
	assert.ErrorIs(t, err, failure)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = newController(ctx, &UpgradeClusterOpts{OcteliumC: client})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGenesisJobPackageIdentityAndSPIFFE(t *testing.T) {
	for _, pkg := range []string{"octelium", "octeliumee", "cordium"} {
		for _, spiffe := range []*install.SPIFFEOpts{nil, {CSIDriver: "test.csi", TrustDomain: "cluster.test"}} {
			job := getGenesisJob("example.com", "west", pkg, "v1.2.3", spiffe)
			assert.Equal(t, vutils.K8sNS, job.Namespace)
			assert.True(t, strings.HasPrefix(job.Name, "octelium-genesis-upgrade-"))
			assert.Equal(t, "genesis", job.Labels["octelium.com/component"])
			assert.Equal(t, job.Labels, job.Spec.Template.Labels)
			pod := job.Spec.Template.Spec
			assert.Equal(t, "octelium-nocturne", pod.ServiceAccountName)
			assert.Equal(t, k8scorev1.RestartPolicyNever, pod.RestartPolicy)
			require.Len(t, pod.Containers, 1)
			container := pod.Containers[0]
			assert.Equal(t, pkg+"-genesis", container.Name)
			assert.True(t, strings.HasSuffix(container.Image, ":v1.2.3"), container.Image)
			assert.Equal(t, []string{"upgrade"}, container.Args)
			assert.Equal(t, k8scorev1.PullAlways, container.ImagePullPolicy)
			env := map[string]string{}
			for _, entry := range container.Env {
				env[entry.Name] = entry.Value
			}
			assert.Equal(t, "example.com", env["OCTELIUM_DOMAIN"])
			assert.Equal(t, "west", env["OCTELIUM_REGION_NAME"])
			if spiffe == nil {
				assert.Empty(t, pod.Volumes)
				assert.Empty(t, container.VolumeMounts)
			} else {
				require.Len(t, pod.Volumes, 1)
				assert.Equal(t, "test.csi", pod.Volumes[0].CSI.Driver)
				assert.Equal(t, "true", env["OCTELIUM_ENABLE_SPIFFE_CSI"])
				assert.Equal(t, "cluster.test", env["OCTELIUM_SPIFFE_TRUST_DOMAIN"])
				require.Len(t, container.VolumeMounts, 1)
				assert.True(t, container.VolumeMounts[0].ReadOnly)
			}
			assert.NotEqual(t, job.Name, getGenesisJob("example.com", "west", pkg, "v1.2.3", spiffe).Name)
		}
	}
}

func TestUpgradeJobStatusClassification(t *testing.T) {
	for _, tc := range []struct {
		name              string
		status            batchv1.JobStatus
		success, terminal bool
	}{
		{name: "Complete", status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}}, success: true},
		{name: "Succeeded", status: batchv1.JobStatus{Succeeded: 1}, success: true},
		{name: "Active", status: batchv1.JobStatus{Active: 1}},
		{name: "Retrying", status: batchv1.JobStatus{Failed: 1}},
		{name: "FalseComplete", status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionFalse}}}},
		{name: "FalseFailed", status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: k8scorev1.ConditionFalse}}}},
		{name: "TerminalFailure", status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: k8scorev1.ConditionTrue, Reason: "BackoffLimitExceeded", Message: "genesis failed"}}}, terminal: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k8sC := fake.NewSimpleClientset(&batchv1.Job{ObjectMeta: k8smetav1.ObjectMeta{Name: "upgrade", Namespace: vutils.K8sNS}, Status: tc.status})
			err := (&controller{k8sC: k8sC}).checkUpgradeJob(t.Context(), "upgrade")
			if tc.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			assert.Equal(t, tc.terminal, isTerminalUpgradeJobErr(err))
			if tc.terminal {
				assert.ErrorContains(t, err, "BackoffLimitExceeded")
				assert.ErrorContains(t, err, "genesis failed")
				assert.True(t, isTerminalUpgradeJobErr(fmt.Errorf("wrapped: %w", err)))
			}
		})
	}
}

func TestUpgradeJobGetFailureIsRetriable(t *testing.T) {
	failure := errors.New("Kubernetes temporarily unavailable")
	k8sC := fake.NewSimpleClientset()
	k8sC.PrependReactor("get", "jobs", func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, failure })
	err := (&controller{k8sC: k8sC}).checkUpgradeJob(t.Context(), "upgrade")
	assert.ErrorIs(t, err, failure)
	assert.False(t, isTerminalUpgradeJobErr(err))
	_, err = k8sC.BatchV1().Jobs(vutils.K8sNS).Get(t.Context(), "missing", k8smetav1.GetOptions{})
	assert.Error(t, err)
}

func TestUpgradeWaitValidationAndCancellation(t *testing.T) {
	ctl := &controller{k8sC: fake.NewSimpleClientset()}
	assert.Error(t, ctl.waitUpgrade(t.Context(), nil))
	assert.Error(t, ctl.waitUpgrade(t.Context(), &batchv1.Job{}))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.ErrorIs(t, ctl.waitUpgrade(ctx, &batchv1.Job{ObjectMeta: k8smetav1.ObjectMeta{Name: "upgrade"}}), context.Canceled)
	ctx, cancel = context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	assert.ErrorIs(t, ctl.waitUpgrade(ctx, &batchv1.Job{ObjectMeta: k8smetav1.ObjectMeta{Name: "upgrade"}}), context.DeadlineExceeded)
	assert.Empty(t, ctl.k8sC.(*fake.Clientset).Actions())
}

func TestUpgradeRejectsEmptyVersionBeforeCreatingJob(t *testing.T) {
	for _, tc := range []struct {
		pkg     string
		request *enterprisev1.UpgradeClusterRequest_Request
	}{
		{"core", &enterprisev1.UpgradeClusterRequest_Request{Core: &enterprisev1.UpgradeClusterRequest_Request_Core{}}},
		{"enterprise", &enterprisev1.UpgradeClusterRequest_Request{PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{}}},
		{"cordium", &enterprisev1.UpgradeClusterRequest_Request{PackageCordium: &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{}}},
	} {
		k8sC := fake.NewSimpleClientset()
		ctl := &controller{k8sC: k8sC, req: &enterprisev1.ClusterConfig_Status_UpgradeRequest{Request: tc.request}}
		err := ctl.run(t.Context())
		if tc.pkg == "core" {
			assert.ErrorContains(t, err, "core upgrade failed")
		} else {
			assert.ErrorContains(t, err, tc.pkg+" package upgrade failed")
		}
		assert.Empty(t, k8sC.Actions())
	}
}

func TestUpgradeJobCreationFailure(t *testing.T) {
	k8sC := fake.NewSimpleClientset()
	k8sC.PrependReactor("create", "jobs", func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, errors.New("forbidden") })
	err := (&controller{k8sC: k8sC, domain: "example.com"}).runUpgradeJob(t.Context(), "octeliumee", "v1.2.3")
	assert.ErrorContains(t, err, "could not create upgrade job")
	assert.ErrorContains(t, err, "forbidden")
}

func TestUpgradeAllPackagesRunInOrder(t *testing.T) {
	k8sC := fake.NewSimpleClientset()
	var order []string
	k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
		order = append(order, job.Spec.Template.Spec.Containers[0].Name)
		job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}
		err := k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
		return true, job, err
	})
	ctl := &controller{k8sC: k8sC, domain: "example.com", req: &enterprisev1.ClusterConfig_Status_UpgradeRequest{Request: &enterprisev1.UpgradeClusterRequest_Request{
		Core:              &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"},
		PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "v1.2.3"},
		PackageCordium:    &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{Version: "v1.2.3"},
	}}}
	require.NoError(t, ctl.run(t.Context()))
	assert.Equal(t, []string{"octelium-genesis", "octeliumee-genesis", "cordium-genesis"}, order)
}

func TestUpgradeStopsAfterTerminalCoreFailure(t *testing.T) {
	k8sC := fake.NewSimpleClientset()
	creates := 0
	k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		creates++
		job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
		job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: k8scorev1.ConditionTrue, Reason: "BackoffLimitExceeded"}}
		err := k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
		return true, job, err
	})
	ctl := &controller{k8sC: k8sC, domain: "example.com", req: &enterprisev1.ClusterConfig_Status_UpgradeRequest{Request: &enterprisev1.UpgradeClusterRequest_Request{
		Core:              &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"},
		PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "v1.2.3"},
	}}}
	assert.ErrorContains(t, ctl.run(t.Context()), "BackoffLimitExceeded")
	assert.Equal(t, 1, creates)
}

func TestUpgradeCancellationAfterJobSubmission(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	k8sC := fake.NewSimpleClientset()
	k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		cancel()
		return false, nil, nil
	})
	err := (&controller{k8sC: k8sC}).runUpgradeJob(ctx, "octeliumee", "v1.2.3")
	assert.ErrorContains(t, err, context.Canceled.Error())
}

func TestTerminalUpgradeErrorUnwrap(t *testing.T) {
	failure := errors.New("genesis terminated")
	err := &terminalUpgradeJobErr{err: failure}
	assert.ErrorIs(t, err, failure)
	assert.True(t, isTerminalUpgradeJobErr(fmt.Errorf("wrapped: %w", err)))
	var nilErr *terminalUpgradeJobErr
	assert.NotEmpty(t, nilErr.Error())
	assert.Nil(t, nilErr.Unwrap())
	assert.NotEmpty(t, (&terminalUpgradeJobErr{}).Error())
}

func TestUpgradeWaitRetriesTransientKubernetesReadFailure(t *testing.T) {
	job := &batchv1.Job{ObjectMeta: k8smetav1.ObjectMeta{Name: "upgrade", Namespace: vutils.K8sNS}, Status: batchv1.JobStatus{
		Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}},
	}}
	k8sC := fake.NewSimpleClientset(job)
	calls := 0
	k8sC.PrependReactor("get", "jobs", func(ktesting.Action) (bool, runtime.Object, error) {
		calls++
		if calls == 1 {
			return true, nil, errors.New("temporary API failure")
		}
		return false, nil, nil
	})
	require.NoError(t, (&controller{k8sC: k8sC}).waitUpgrade(t.Context(), job))
	assert.Equal(t, 2, calls)
}
