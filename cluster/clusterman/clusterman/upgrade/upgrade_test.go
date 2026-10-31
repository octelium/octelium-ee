// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package upgrade

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/regionutils"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/client/octops/commands/install"
	cocteliumc "github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
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

func getTestRegion(name string, index int32) *corev1.Region {
	return &corev1.Region{
		Metadata: &metav1.Metadata{Name: name, Uid: name + "-uid"},
		Spec:     &corev1.Region_Spec{},
		Status:   &corev1.Region_Status{Index: index},
	}
}

func getTestJoinedRegion(t *testing.T, name string, index int32, packages ...string) *corev1.Region {
	ret := getTestRegion(name, index)
	ret.Status.VersionInfoMap = map[string]*corev1.Region_Status_VersionInfo{}
	for _, pkg := range packages {
		ret.Status.VersionInfoMap[pkg] = &corev1.Region_Status_VersionInfo{Package: pkg, Version: "1.0.0"}
	}
	require.NoError(t, regionutils.SetRegionExtInfo(ret, &enterprisev1.RegionExtInfo{
		Kubernetes: &enterprisev1.JoinClusterRequest_Kubernetes{
			Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
				Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{
					Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_FromSecret{
						FromSecret: name + "-kubeconfig",
					},
				},
			},
		},
	}))
	return ret
}

func newCompletingK8sC(order *[]string, region string, fail bool) *fake.Clientset {
	k8sC := fake.NewSimpleClientset()
	k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		job := action.(ktesting.CreateAction).GetObject().(*batchv1.Job).DeepCopy()
		*order = append(*order, region+"/"+job.Spec.Template.Spec.Containers[0].Name)
		if fail {
			job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: k8scorev1.ConditionTrue, Reason: "BackoffLimitExceeded"}}
		} else {
			job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: k8scorev1.ConditionTrue}}
		}
		err := k8sC.Tracker().Create(batchv1.SchemeGroupVersion.WithResource("jobs"), job, job.Namespace)
		return true, job, err
	})
	return k8sC
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
	err := (&controller{domain: "example.com"}).runUpgradeJob(t.Context(),
		&regionTarget{region: getTestRegion("default", 0), k8sC: k8sC}, "octeliumee", "v1.2.3")
	assert.ErrorContains(t, err, "could not create upgrade job")
	assert.ErrorContains(t, err, "forbidden")
	assert.ErrorContains(t, (&controller{}).runUpgradeJob(t.Context(),
		&regionTarget{region: getTestRegion("default", 0), k8sC: k8sC}, "octeliumee", ""), "empty upgrade version")
}

func TestUpgradeAllPackagesRunInOrder(t *testing.T) {
	var order []string
	ctl := &controller{domain: "example.com", req: &enterprisev1.ClusterConfig_Status_UpgradeRequest{Request: &enterprisev1.UpgradeClusterRequest_Request{
		Core:              &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"},
		PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "v1.2.3"},
		PackageCordium:    &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{Version: "v1.2.3"},
	}}}
	require.NoError(t, ctl.runTargets(t.Context(), []*regionTarget{
		{region: getTestRegion("default", 0), k8sC: newCompletingK8sC(&order, "default", false)},
	}))
	assert.Equal(t, []string{"default/octelium-genesis", "default/octeliumee-genesis", "default/cordium-genesis"}, order)
}

func TestUpgradeAllRegionsRunInOrder(t *testing.T) {
	var order []string
	var reports [][]*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region
	ctl := &controller{
		domain: "example.com",
		req: &enterprisev1.ClusterConfig_Status_UpgradeRequest{Request: &enterprisev1.UpgradeClusterRequest_Request{
			Core:           &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"},
			PackageCordium: &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{Version: "v1.2.3"},
		}},
		onRegions: func(ctx context.Context, regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region) {
			reports = append(reports, regions)
		},
	}

	targets := []*regionTarget{
		{region: getTestRegion("default", 0), k8sC: newCompletingK8sC(&order, "default", false)},
		{region: getTestJoinedRegion(t, "west", 1, regionutils.PackageEnterprise), k8sC: newCompletingK8sC(&order, "west", false)},
		{region: getTestJoinedRegion(t, "east", 2, regionutils.PackageEnterprise, regionutils.PackageCordium), k8sC: newCompletingK8sC(&order, "east", false)},
	}

	require.NoError(t, ctl.runTargets(t.Context(), targets))
	assert.Equal(t, []string{
		"default/octelium-genesis", "default/cordium-genesis",
		"west/octelium-genesis",
		"east/octelium-genesis", "east/cordium-genesis",
	}, order)

	require.Len(t, reports, 7)
	for _, rgn := range reports[0] {
		assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED, rgn.State)
	}
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING, reports[1][0].State)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED, reports[1][1].State)
	last := reports[len(reports)-1]
	require.Len(t, last, 3)
	for i, name := range []string{"default", "west", "east"} {
		assert.Equal(t, name, last[i].RegionRef.Name)
		assert.Equal(t, name+"-uid", last[i].RegionRef.Uid)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last[i].State)
		assert.NotNil(t, last[i].DoneAt)
	}
}

func TestUpgradeStopsAfterTerminalCoreFailure(t *testing.T) {
	var order []string
	var reports [][]*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region
	ctl := &controller{
		domain: "example.com",
		req: &enterprisev1.ClusterConfig_Status_UpgradeRequest{Request: &enterprisev1.UpgradeClusterRequest_Request{
			Core:              &enterprisev1.UpgradeClusterRequest_Request_Core{Version: "v1.2.3"},
			PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: "v1.2.3"},
		}},
		onRegions: func(ctx context.Context, regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region) {
			reports = append(reports, regions)
		},
	}
	err := ctl.runTargets(t.Context(), []*regionTarget{
		{region: getTestRegion("default", 0), k8sC: newCompletingK8sC(&order, "default", false)},
		{region: getTestJoinedRegion(t, "west", 1, regionutils.PackageEnterprise), k8sC: newCompletingK8sC(&order, "west", true)},
		{region: getTestJoinedRegion(t, "east", 2, regionutils.PackageEnterprise), k8sC: newCompletingK8sC(&order, "east", false)},
	})
	assert.ErrorContains(t, err, "BackoffLimitExceeded")
	assert.ErrorContains(t, err, "west")
	assert.Equal(t, []string{"default/octelium-genesis", "default/octeliumee-genesis", "west/octelium-genesis"}, order)

	last := reports[len(reports)-1]
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, last[0].State)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED, last[1].State)
	assert.Contains(t, last[1].Message, "BackoffLimitExceeded")
	assert.NotNil(t, last[1].DoneAt)
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED, last[2].State)
}

func TestUpgradeCancellationAfterJobSubmission(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	k8sC := fake.NewSimpleClientset()
	k8sC.PrependReactor("create", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
		cancel()
		return false, nil, nil
	})
	err := (&controller{}).runUpgradeJob(ctx, &regionTarget{region: getTestRegion("default", 0), k8sC: k8sC}, "octeliumee", "v1.2.3")
	assert.ErrorContains(t, err, context.Canceled.Error())
}

func TestUpgradeTargets(t *testing.T) {
	ctx := t.Context()

	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })

	for _, rgn := range []*corev1.Region{
		getTestJoinedRegion(t, "east", 5, regionutils.PackageCore, regionutils.PackageEnterprise),
		getTestJoinedRegion(t, "west", 2, regionutils.PackageCore, regionutils.PackageEnterprise),
		getTestJoinedRegion(t, "partial", 3, regionutils.PackageCore),
		getTestRegion("manual", 4),
	} {
		rgn.Metadata.Uid = ""
		_, err := tst.C.OcteliumC.CoreC().CreateRegion(ctx, rgn)
		require.NoError(t, err)
	}

	defaultK8sC := fake.NewSimpleClientset()
	var secrets []string
	ctl := &controller{
		octeliumC: tst.C.OcteliumC,
		k8sC:      defaultK8sC,
		newRegionK8sC: func(ctx context.Context, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error) {
			secrets = append(secrets, k.GetKubeconfig().GetFromSecret())
			return fake.NewSimpleClientset(), nil
		},
	}

	targets, err := ctl.getTargets(ctx)
	require.NoError(t, err)
	require.Len(t, targets, 3)
	assert.Equal(t, "default", targets[0].region.Metadata.Name)
	assert.Same(t, defaultK8sC, targets[0].k8sC)
	assert.Equal(t, "west", targets[1].region.Metadata.Name)
	assert.Equal(t, "east", targets[2].region.Metadata.Name)
	assert.Equal(t, []string{"west-kubeconfig", "east-kubeconfig"}, secrets)

	failure := errors.New("kubeconfig is invalid")
	ctl.newRegionK8sC = func(ctx context.Context, k *enterprisev1.JoinClusterRequest_Kubernetes) (kubernetes.Interface, error) {
		return nil, failure
	}
	_, err = ctl.getTargets(ctx)
	assert.ErrorContains(t, err, failure.Error())
	assert.ErrorContains(t, err, "west")
	assert.Empty(t, defaultK8sC.Actions())
}
