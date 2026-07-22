// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"strings"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestClusterConfig(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	exporter := tstCreateClusterConfigCollectorExporter(ctx, t, srv)

	cc, err := srv.GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
	assert.Nil(t, err, "%+v", err)
	assert.NotNil(t, cc)
	assert.NotNil(t, cc.Metadata)
	assert.NotNil(t, cc.Spec)

	{
		arg := tstCloneClusterConfig(cc)
		arg.Spec = &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				AdditionalInlineConfig: "receivers:\n  otlp:\n",
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					{
						Name:      utilrand.GetRandomStringCanonical(8),
						Type:      enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS,
						Exporters: []string{exporter.Metadata.Name},
					},
					{
						Name:       utilrand.GetRandomStringCanonical(8),
						Type:       enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS,
						IsDisabled: true,
						Exporters:  []string{exporter.Metadata.Name},
					},
				},
			},
			Scaler: &enterprisev1.ClusterConfig_Spec_Scaler{
				Octovigil: &enterprisev1.ClusterConfig_Spec_Scaler_Octovigil{
					Replicas: 3,
				},
				Ingress: &enterprisev1.ClusterConfig_Spec_Scaler_Ingress{
					Replicas: 4,
				},
				Collector: &enterprisev1.ClusterConfig_Spec_Scaler_Collector{
					Replicas: 2,
				},
			},
			Certificate: &enterprisev1.ClusterConfig_Spec_Certificate{
				DefaultMode: enterprisev1.Certificate_Spec_MANAGED,
			},
		}
		arg.Status = &enterprisev1.ClusterConfig_Status{
			TotalSuccessfulUpgrades: 100,
			TotalFailedUpgrades:     50,
		}

		updated, err := srv.UpdateClusterConfig(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, updated.Spec.Collector)
		assert.Len(t, updated.Spec.Collector.Pipelines, 2)
		assert.Equal(t, "receivers:\n  otlp:\n", updated.Spec.Collector.AdditionalInlineConfig)
		assert.Equal(t, enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, updated.Spec.Collector.Pipelines[0].Type)
		assert.Equal(t, enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS, updated.Spec.Collector.Pipelines[1].Type)
		assert.True(t, updated.Spec.Collector.Pipelines[1].IsDisabled)
		assert.Equal(t, int32(3), updated.Spec.Scaler.Octovigil.Replicas)
		assert.Equal(t, int32(4), updated.Spec.Scaler.Ingress.Replicas)
		assert.Equal(t, int32(2), updated.Spec.Scaler.Collector.Replicas)
		assert.Equal(t, enterprisev1.Certificate_Spec_MANAGED, updated.Spec.Certificate.DefaultMode)
		assert.NotEqual(t, uint64(100), updated.GetStatus().GetTotalSuccessfulUpgrades())
		assert.NotEqual(t, uint64(50), updated.GetStatus().GetTotalFailedUpgrades())
		cc = updated
	}

	{
		ret, err := srv.GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, cc.Metadata.Uid, ret.Metadata.Uid)
		assert.NotNil(t, ret.Spec.Collector)
		assert.Len(t, ret.Spec.Collector.Pipelines, 2)
		assert.Equal(t, enterprisev1.Certificate_Spec_MANAGED, ret.Spec.Certificate.DefaultMode)
	}

	{
		arg := tstCloneClusterConfig(cc)
		arg.Spec = &enterprisev1.ClusterConfig_Spec{}
		updated, err := srv.UpdateClusterConfig(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, updated.Spec)
		assert.Nil(t, updated.Spec.Collector)
		assert.Nil(t, updated.Spec.Scaler)
		assert.Nil(t, updated.Spec.Certificate)
	}
}

func TestValidateClusterConfig(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	exporter := tstCreateClusterConfigCollectorExporter(ctx, t, srv)
	cc, err := srv.GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
	assert.Nil(t, err, "%+v", err)

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{}))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{},
		}))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				AdditionalInlineConfig: "exporters:\n  debug:\n",
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline(utilrand.GetRandomStringCanonical(8), enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, exporter.Metadata.Name),
					tstClusterConfigPipeline(utilrand.GetRandomStringCanonical(8), enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS, exporter.Metadata.Name),
				},
			},
		}))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Scaler: &enterprisev1.ClusterConfig_Spec_Scaler{
				Octovigil: &enterprisev1.ClusterConfig_Spec_Scaler_Octovigil{},
				Ingress: &enterprisev1.ClusterConfig_Spec_Scaler_Ingress{
					Replicas: maxClusterConfigScalerReplicas,
				},
				Collector: &enterprisev1.ClusterConfig_Spec_Scaler_Collector{
					Replicas: 1,
				},
			},
		}))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Certificate: &enterprisev1.ClusterConfig_Spec_Certificate{
				DefaultMode: enterprisev1.Certificate_Spec_MODE_UNSET,
			},
		}))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Certificate: &enterprisev1.ClusterConfig_Spec_Certificate{
				DefaultMode: enterprisev1.Certificate_Spec_MANUAL,
			},
		}))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, &enterprisev1.ClusterConfig{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, &enterprisev1.ClusterConfig{
			Metadata: &metav1.Metadata{Name: cc.Metadata.Name},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, &enterprisev1.ClusterConfig{
			Spec: &enterprisev1.ClusterConfig_Spec{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, nil))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				AdditionalInlineConfig: strings.Repeat("a", maxClusterConfigCollectorInlineConfigBytes+1),
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				AdditionalInlineConfig: "abc\x00def",
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: tstClusterConfigPipelines(maxClusterConfigCollectorPipelines+1, exporter.Metadata.Name),
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{nil},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, exporter.Metadata.Name),
				},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, exporter.Metadata.Name),
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS, exporter.Metadata.Name),
				},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_TYPE_UNSET, exporter.Metadata.Name),
				},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_Type(1000), exporter.Metadata.Name),
				},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS),
				},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, tstClusterConfigExporterNames(maxClusterConfigCollectorExportersPerPipeline+1)...),
				},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, "bad exporter"),
				},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, exporter.Metadata.Name, exporter.Metadata.Name),
				},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, utilrand.GetRandomStringCanonical(8)),
				},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Scaler: &enterprisev1.ClusterConfig_Spec_Scaler{
				Octovigil: &enterprisev1.ClusterConfig_Spec_Scaler_Octovigil{Replicas: -1},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Scaler: &enterprisev1.ClusterConfig_Spec_Scaler{
				Ingress: &enterprisev1.ClusterConfig_Spec_Scaler_Ingress{Replicas: -1},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Scaler: &enterprisev1.ClusterConfig_Spec_Scaler{
				Collector: &enterprisev1.ClusterConfig_Spec_Scaler_Collector{Replicas: -1},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Scaler: &enterprisev1.ClusterConfig_Spec_Scaler{
				Octovigil: &enterprisev1.ClusterConfig_Spec_Scaler_Octovigil{Replicas: maxClusterConfigScalerReplicas + 1},
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateClusterConfig(ctx, tstClusterConfigWithSpec(cc, &enterprisev1.ClusterConfig_Spec{
			Certificate: &enterprisev1.ClusterConfig_Spec_Certificate{
				DefaultMode: enterprisev1.Certificate_Spec_Mode(1000),
			},
		}))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestUpdateClusterConfigInvalid(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	cc, err := srv.GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
	assert.Nil(t, err, "%+v", err)

	{
		arg := tstCloneClusterConfig(cc)
		arg.Spec = nil
		_, err = srv.UpdateClusterConfig(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		arg := tstCloneClusterConfig(cc)
		arg.Spec = &enterprisev1.ClusterConfig_Spec{
			Collector: &enterprisev1.ClusterConfig_Spec_Collector{
				Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
					tstClusterConfigPipeline("logs", enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, utilrand.GetRandomStringCanonical(8)),
				},
			},
		}
		_, err = srv.UpdateClusterConfig(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func tstCreateClusterConfigCollectorExporter(ctx context.Context, t *testing.T, srv *Server) *enterprisev1.CollectorExporter {
	item, err := srv.octeliumC.EnterpriseC().CreateCollectorExporter(ctx, &enterprisev1.CollectorExporter{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   &enterprisev1.CollectorExporter_Spec{},
		Status: &enterprisev1.CollectorExporter_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	return item
}

func tstClusterConfigWithSpec(base *enterprisev1.ClusterConfig, spec *enterprisev1.ClusterConfig_Spec) *enterprisev1.ClusterConfig {
	ret := tstCloneClusterConfig(base)
	ret.Spec = spec
	return ret
}

func tstCloneClusterConfig(arg *enterprisev1.ClusterConfig) *enterprisev1.ClusterConfig {
	return &enterprisev1.ClusterConfig{
		Metadata: &metav1.Metadata{
			Name: arg.Metadata.Name,
			Uid:  arg.Metadata.Uid,
		},
		Spec:   arg.Spec,
		Status: arg.Status,
	}
}

func tstClusterConfigPipeline(name string, typ enterprisev1.ClusterConfig_Spec_Collector_Pipeline_Type, exporters ...string) *enterprisev1.ClusterConfig_Spec_Collector_Pipeline {
	return &enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		Name:      name,
		Type:      typ,
		Exporters: exporters,
	}
}

func tstClusterConfigPipelines(n int, exporter string) []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline {
	ret := make([]*enterprisev1.ClusterConfig_Spec_Collector_Pipeline, 0, n)
	for i := 0; i < n; i++ {
		ret = append(ret, tstClusterConfigPipeline(utilrand.GetRandomStringCanonical(8), enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, exporter))
	}
	return ret
}

func tstClusterConfigExporterNames(n int) []string {
	ret := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ret = append(ret, utilrand.GetRandomStringCanonical(8))
	}
	return ret
}
