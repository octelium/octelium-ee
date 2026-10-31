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
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const regionJoinBudget = 5 * time.Minute

const unreachableKubeconfig = `apiVersion: v1
kind: Config
current-context: unreachable
clusters:
- name: unreachable
  cluster:
    server: https://127.0.0.1:1
    insecure-skip-tls-verify: true
contexts:
- name: unreachable
  context:
    cluster: unreachable
    user: admin
users:
- name: admin
  user:
    token: e2e-token
`

func getJoinKubeconfig(secretName string) *enterprisev1.JoinClusterRequest_Kubernetes {
	return &enterprisev1.JoinClusterRequest_Kubernetes{
		Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_{
			Kubeconfig: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_Kubeconfig_FromSecret{
					FromSecret: secretName,
				},
			},
		},
	}
}

func testRegionJoin(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	sec, err := h.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: h.Name()},
		Spec:     &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{Value: unreachableKubeconfig},
		},
	})
	require.Nil(t, err)
	t.Cleanup(func() {
		h.EnterpriseC().DeleteSecret(context.Background(), &metav1.DeleteOptions{Uid: sec.Metadata.Uid})
	})

	regionName := h.Name()

	t.Run("InvalidRequestsAreRejected", func(t *testing.T) {
		for _, req := range []*enterprisev1.JoinClusterRequest{
			{},
			{Name: "default", Kubernetes: getJoinKubeconfig(sec.Metadata.Name)},
			{Name: "Invalid_Name", Kubernetes: getJoinKubeconfig(sec.Metadata.Name)},
			{Name: regionName},
			{Name: regionName, Kubernetes: getJoinKubeconfig(h.Name())},
			{Name: regionName, Kubernetes: &enterprisev1.JoinClusterRequest_Kubernetes{
				Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_{
					BearerToken: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken{
						Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_FromSecret{
							FromSecret: sec.Metadata.Name,
						},
						Server: "http://127.0.0.1:1",
					},
				},
			}},
		} {
			_, err := h.ClusterC().JoinCluster(ctx, req)
			require.NotNil(t, err)
			assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
		}
	})

	t.Run("UnreachableClusterFailsWithoutCreatingTheRegion", func(t *testing.T) {
		before := h.EnterpriseClusterConfig(t)
		beforeCount := len(before.Status.LastJoinRequests)

		_, err := h.ClusterC().JoinCluster(ctx, &enterprisev1.JoinClusterRequest{
			Name:        regionName,
			DisplayName: "E2E unreachable Region",
			Kubernetes:  getJoinKubeconfig(sec.Metadata.Name),
		})
		require.Nil(t, err)

		var last *enterprisev1.ClusterConfig_Status_JoinRequest
		h.Eventually(t, "the join to fail", regionJoinBudget,
			func(ctx context.Context) error {
				cc, err := h.EnterpriseC().GetClusterConfig(ctx,
					&enterprisev1.GetClusterConfigRequest{})
				if err != nil {
					return err
				}
				if cc.Status.JoinRequest != nil {
					return errors.Errorf("the join is %s", cc.Status.JoinRequest.State)
				}
				if len(cc.Status.LastJoinRequests) <= beforeCount {
					return errors.Errorf("the join is not recorded yet")
				}
				last = cc.Status.LastJoinRequests[0]
				return nil
			})

		require.NotNil(t, last)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_JoinRequest_FAILED, last.State)
		assert.Equal(t, regionName, last.Request.Name)
		assert.Equal(t, "E2E unreachable Region", last.Request.DisplayName)
		assert.NotEmpty(t, last.Message)
		assert.NotNil(t, last.CreatedAt)
		assert.NotNil(t, last.DoneAt)
		assert.Nil(t, last.RegionRef)

		_, err = h.CoreC().GetRegion(ctx, &metav1.GetOptions{Name: regionName})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

		rgnList, err := h.CoreC().ListRegion(ctx, &corev1.ListRegionOptions{})
		require.Nil(t, err)
		for _, rgn := range rgnList.Items {
			assert.NotEqual(t, regionName, rgn.Metadata.Name)
		}
	})
}
