// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type collectorConfigClient struct {
	enterprisev1.MainServiceClient
	t  *testing.T
	cc *enterprisev1.ClusterConfig
}

type secretStoreSyncClient struct {
	enterprisev1.MainServiceClient
	t     *testing.T
	calls int
}

func (c *secretStoreSyncClient) GetSecretStore(ctx context.Context,
	req *metav1.GetOptions, opts ...grpc.CallOption) (*enterprisev1.SecretStore, error) {
	require.Nil(c.t, ctx.Err())
	_, ok := ctx.Deadline()
	require.True(c.t, ok)
	c.calls++

	sync := &enterprisev1.SecretStore_Status_Synchronization{
		State:       enterprisev1.SecretStore_Status_Synchronization_SUCCESS,
		CompletedAt: pbutils.Now(),
	}
	return &enterprisev1.SecretStore{
		Status: &enterprisev1.SecretStore_Status{
			Type:                 enterprisev1.SecretStore_Status_KUBERNETES,
			State:                enterprisev1.SecretStore_Status_OK,
			Synchronization:      sync,
			LastSynchronizations: []*enterprisev1.SecretStore_Status_Synchronization{sync},
		},
	}, nil
}

func TestWaitSecretStoreSynchronizedDuringCleanup(t *testing.T) {
	h := Wrap(&harness.H{})
	c := &secretStoreSyncClient{t: t}

	t.Run("Restore", func(t *testing.T) {
		t.Cleanup(func() {
			require.NotNil(t, t.Context().Err())
			h.waitSecretStoreSynchronized(t, c, "default",
				enterprisev1.SecretStore_Status_KUBERNETES, time.Second)
		})
	})
	assert.Equal(t, 1, c.calls)
}

func (c *collectorConfigClient) GetClusterConfig(ctx context.Context,
	req *enterprisev1.GetClusterConfigRequest, opts ...grpc.CallOption) (*enterprisev1.ClusterConfig, error) {
	require.Nil(c.t, ctx.Err())
	_, ok := ctx.Deadline()
	require.True(c.t, ok)
	return pbutils.Clone(c.cc).(*enterprisev1.ClusterConfig), nil
}

func (c *collectorConfigClient) UpdateClusterConfig(ctx context.Context,
	cc *enterprisev1.ClusterConfig, opts ...grpc.CallOption) (*enterprisev1.ClusterConfig, error) {
	require.Nil(c.t, ctx.Err())
	_, ok := ctx.Deadline()
	require.True(c.t, ok)
	c.cc = pbutils.Clone(cc).(*enterprisev1.ClusterConfig)
	return pbutils.Clone(c.cc).(*enterprisev1.ClusterConfig), nil
}

func TestSetCollectorPipelinesRestoresConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name      string
		collector *enterprisev1.ClusterConfig_Spec_Collector
	}{
		{name: "NoCollector"},
		{name: "EmptyCollector", collector: &enterprisev1.ClusterConfig_Spec_Collector{}},
		{
			name: "ExistingPipeline",
			collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					{Name: "original", Exporters: []string{"original-exporter"}},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &collectorConfigClient{
				t: t,
				cc: &enterprisev1.ClusterConfig{
					Spec: &enterprisev1.ClusterConfig_Spec{Collector: tc.collector},
				},
			}
			before := pbutils.Clone(c.cc).(*enterprisev1.ClusterConfig)

			t.Run("Configure", func(t *testing.T) {
				setCollectorPipelines(t, c,
					&enterprisev1.ClusterConfig_Spec_Collector_Pipeline{Name: "first"})
				setCollectorPipelines(t, c,
					&enterprisev1.ClusterConfig_Spec_Collector_Pipeline{Name: "second"})
				require.Len(t, c.cc.Spec.Collector.Pipelines,
					len(before.Spec.GetCollector().GetPipelines())+2)
				c.cc.Spec.DeviceManagers = []string{"another-test"}
			})

			assert.Equal(t, before.Spec.Collector, c.cc.Spec.Collector)
			assert.Equal(t, []string{"another-test"}, c.cc.Spec.DeviceManagers)
		})
	}
}
