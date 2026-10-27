package cluster_config

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	appsv1 "k8s.io/api/apps/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func scalingConfig(collector, vigil, ingress uint32) *enterprisev1.ClusterConfig {
	ret := &enterprisev1.ClusterConfig{}
	body, _ := json.Marshal(map[string]any{"spec": map[string]any{"scaler": map[string]any{
		"collector": map[string]any{"replicas": collector},
		"octovigil": map[string]any{"replicas": vigil},
		"ingress":   map[string]any{"replicas": ingress},
	}}})
	if err := protojson.Unmarshal(body, ret); err != nil {
		panic(err)
	}
	return ret
}

func scalingDeployment(name string, replicas *int32) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: k8smetav1.ObjectMeta{Name: name, Namespace: vutils.K8sNS, Labels: map[string]string{"keep": "value"}},
		Spec: appsv1.DeploymentSpec{Replicas: replicas}}
}

func TestClusterConfigReconcilesAllReplicaCounts(t *testing.T) {
	k8sC := fake.NewSimpleClientset(
		scalingDeployment("octeliumee-collector", utils_types.Int32ToPtr(1)),
		scalingDeployment("octelium-octovigil", utils_types.Int32ToPtr(1)),
		scalingDeployment("octelium-ingress-dataplane", utils_types.Int32ToPtr(1)),
	)
	ctl := NewController(nil, k8sC)
	next := scalingConfig(3, 4, 5)
	require.NoError(t, ctl.OnUpdate(t.Context(), next, scalingConfig(1, 1, 1)))
	for _, tc := range []struct {
		name     string
		replicas int32
	}{
		{"octeliumee-collector", 3}, {"octelium-octovigil", 4}, {"octelium-ingress-dataplane", 5},
	} {
		depl, err := k8sC.AppsV1().Deployments(vutils.K8sNS).Get(t.Context(), tc.name, k8smetav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, tc.replicas, *depl.Spec.Replicas)
		assert.Equal(t, "value", depl.Labels["keep"])
	}
	k8sC.ClearActions()
	require.NoError(t, ctl.OnUpdate(t.Context(), next, pbutils.Clone(next).(*enterprisev1.ClusterConfig)))
	assert.Empty(t, k8sC.Actions())
}

func TestClusterConfigResetsUnsetAndZeroReplicas(t *testing.T) {
	for _, next := range []*enterprisev1.ClusterConfig{scalingConfig(0, 0, 0), scalingConfig(1, 1, 1), {Spec: &enterprisev1.ClusterConfig_Spec{}}} {
		k8sC := fake.NewSimpleClientset(
			scalingDeployment("octeliumee-collector", utils_types.Int32ToPtr(3)),
			scalingDeployment("octelium-octovigil", utils_types.Int32ToPtr(3)),
			scalingDeployment("octelium-ingress-dataplane", utils_types.Int32ToPtr(3)),
		)
		require.NoError(t, NewController(nil, k8sC).OnUpdate(t.Context(), next, scalingConfig(3, 3, 3)))
		for _, name := range []string{"octeliumee-collector", "octelium-octovigil", "octelium-ingress-dataplane"} {
			depl, err := k8sC.AppsV1().Deployments(vutils.K8sNS).Get(t.Context(), name, k8smetav1.GetOptions{})
			require.NoError(t, err)
			assert.EqualValues(t, 1, *depl.Spec.Replicas)
		}
	}
}

func TestDeploymentReplicaNoopsAndDefaults(t *testing.T) {
	for _, tc := range []struct {
		current *int32
		desired int32
		updates int
	}{
		{nil, 1, 0}, {utils_types.Int32ToPtr(2), 2, 0}, {nil, 3, 1}, {utils_types.Int32ToPtr(1), 0, 1},
	} {
		k8sC := fake.NewSimpleClientset(scalingDeployment("component", tc.current))
		require.NoError(t, NewController(nil, k8sC).updateReplicasDeployment(t.Context(), "component", tc.desired))
		updates := 0
		for _, action := range k8sC.Actions() {
			if action.GetVerb() == "update" {
				updates++
			}
		}
		assert.Equal(t, tc.updates, updates)
	}
	k8sC := fake.NewSimpleClientset()
	require.NoError(t, NewController(nil, k8sC).updateReplicasDeployment(t.Context(), "missing", -1))
	assert.Empty(t, k8sC.Actions())
	assert.Error(t, NewController(nil, k8sC).updateReplicasDeployment(t.Context(), "missing", 2))
}

func TestClusterConfigStopsOnKubernetesUpdateFailure(t *testing.T) {
	failure := errors.New("deployment update failed")
	k8sC := fake.NewSimpleClientset(
		scalingDeployment("octeliumee-collector", utils_types.Int32ToPtr(1)),
		scalingDeployment("octelium-octovigil", utils_types.Int32ToPtr(1)),
		scalingDeployment("octelium-ingress-dataplane", utils_types.Int32ToPtr(1)),
	)
	k8sC.PrependReactor("update", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
		if action.(ktesting.UpdateAction).GetObject().(*appsv1.Deployment).Name == "octelium-octovigil" {
			return true, nil, failure
		}
		return false, nil, nil
	})
	err := NewController(nil, k8sC).OnUpdate(t.Context(), scalingConfig(3, 4, 5), scalingConfig(1, 1, 1))
	assert.ErrorIs(t, err, failure)
	for _, action := range k8sC.Actions() {
		if get, ok := action.(ktesting.GetAction); ok {
			assert.NotEqual(t, "octelium-ingress-dataplane", get.GetName())
		}
	}
}
