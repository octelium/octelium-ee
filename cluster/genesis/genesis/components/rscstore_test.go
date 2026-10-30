package components

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateRscStoreUsesRecreateForNewAndExistingDeployments(t *testing.T) {
	for _, name := range []string{"New", "RollingUpdate"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			client := fake.NewClientset()
			opts := &CommonOpts{}
			opts.K8sC = client
			if name == "RollingUpdate" {
				previous := getRscStoreDeployment(opts)
				previous.Spec.Strategy = appsv1.DeploymentStrategy{
					Type:          appsv1.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &appsv1.RollingUpdateDeployment{},
				}
				_, err := client.AppsV1().Deployments(ns).Create(ctx, previous, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			require.NoError(t, CreateRscStore(ctx, opts))
			deployment, err := client.AppsV1().Deployments(ns).Get(ctx,
				getComponentName(componentRscStore), metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deployment.Spec.Strategy.Type)
			assert.Nil(t, deployment.Spec.Strategy.RollingUpdate)
			require.Len(t, deployment.Spec.Template.Spec.Volumes, 1)
			require.NotNil(t, deployment.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim)
			assert.Equal(t, "octelium-rscstore",
				deployment.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName)
		})
	}
}
