// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestSrvMetric(t *testing.T) *srvMetric {
	return &srvMetric{s: &Server{db: newTestDuckDB(t)}}
}

func putTestAttributes(attrs pcommon.Map, values map[string]any) {
	for key, value := range values {
		switch typed := value.(type) {
		case string:
			attrs.PutStr(key, typed)
		case bool:
			attrs.PutBool(key, typed)
		case int64:
			attrs.PutInt(key, typed)
		}
	}
}

func newVigilResourceMetrics() (pmetric.Metrics, pmetric.ScopeMetrics) {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	putTestAttributes(rm.Resource().Attributes(), map[string]any{
		"octelium.component.type":      "vigil",
		"octelium.component.namespace": "octelium",
		"octelium.component.uid":       "octelium-vigil-1",
		"octelium.region.name":         "default",
	})

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("default")

	return metrics, sm
}

func appendTestCounter(sm pmetric.ScopeMetrics, name string, start, at time.Time,
	value int64, attrs map[string]any) {
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)

	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	point := sum.DataPoints().AppendEmpty()
	point.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	point.SetTimestamp(pcommon.NewTimestampFromTime(at))
	point.SetIntValue(value)
	putTestAttributes(point.Attributes(), attrs)
}

func appendTestUpDownCounter(sm pmetric.ScopeMetrics, name string, start, at time.Time,
	value int64, attrs map[string]any) {
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)

	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	point := sum.DataPoints().AppendEmpty()
	point.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	point.SetTimestamp(pcommon.NewTimestampFromTime(at))
	point.SetIntValue(value)
	putTestAttributes(point.Attributes(), attrs)
}

func appendTestHistogram(sm pmetric.ScopeMetrics, name, unit string, start, at time.Time) {
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetUnit(unit)

	histogram := metric.SetEmptyHistogram()
	histogram.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	point := histogram.DataPoints().AppendEmpty()
	point.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	point.SetTimestamp(pcommon.NewTimestampFromTime(at))
	point.ExplicitBounds().FromRaw([]float64{10, 100})
	point.BucketCounts().FromRaw([]uint64{1, 1, 1})
	point.SetCount(3)
	point.SetSum(100)
}

func storeTestMetrics(t *testing.T, s *srvMetric, metrics pmetric.Metrics) {
	t.Helper()

	_, _, err := inspectMetrics(metrics)
	require.NoError(t, err)

	batch, err := buildMetricWriteBatch(metrics)
	require.NoError(t, err)
	batch.normalize()

	require.NoError(t, s.storeMetricWriteBatch(context.Background(), batch))
}

func testMetricTimeRange(from, to time.Time) *vmetricsv1.TimeRange {
	return &vmetricsv1.TimeRange{
		From: pbutils.Timestamp(from),
		To:   pbutils.Timestamp(to),
	}
}

func testMetricStep() *metav1.Duration {
	return &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 1}}
}

func TestVigilRequestAttributesAreQueryable(t *testing.T) {
	s := newTestSrvMetric(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Minute)
	start := now.Add(-10 * time.Minute)

	for idx, at := range []time.Time{now.Add(-3 * time.Minute), now.Add(-time.Minute)} {
		metrics, sm := newVigilResourceMetrics()
		for _, status := range []string{"OK", "PERMISSION_DENIED"} {
			appendTestCounter(sm, "req.total", start, at, int64(idx+1)*10, map[string]any{
				"octelium.vigil.svc.name":    "svc",
				"octelium.vigil.svc.mode":    "GRPC",
				"state":                      "ALLOWED",
				"req.grpc.status":            status,
				"req.grpc.service_full_name": "octelium.api.main.core.v1.MainService",
				"req.grpc.method":            "GetService",
				"req.authenticated":          true,
			})
		}
		storeTestMetrics(t, s, metrics)
	}

	descriptors, err := s.ListMetricDescriptors(ctx, &vmetricsv1.ListMetricDescriptorsRequest{
		NamePrefix: "req.total",
	})
	require.NoError(t, err)
	require.Len(t, descriptors.Items, 1)

	byKey := map[string]*vmetricsv1.MetricAttributeDescriptor{}
	for _, attribute := range descriptors.Items[0].Attributes {
		byKey[attribute.Key] = attribute
	}

	for _, key := range []string{
		"req.grpc.status",
		"req.grpc.service_full_name",
		"req.grpc.method",
		"req.authenticated",
		"octelium.vigil.svc.mode",
		"state",
	} {
		attribute := byKey[key]
		require.NotNil(t, attribute, key)
		assert.True(t, attribute.Groupable, key)
		assert.True(t, attribute.Filterable, key)
		assert.Empty(t, attribute.GroupUnsupportedReason, key)
		assert.Empty(t, attribute.FilterUnsupportedReason, key)
	}

	assert.Equal(t, vmetricsv1.AttributeValue_BOOL, byKey["req.authenticated"].ValueKind)

	res, err := s.QueryMetrics(ctx, &vmetricsv1.QueryMetricsRequest{
		Metric: &vmetricsv1.MetricSelector{
			Selector: &vmetricsv1.MetricSelector_Name{Name: "req.total"},
			Kind:     vmetricsv1.MetricDescriptor_COUNTER,
		},
		TimeRange: testMetricTimeRange(now.Add(-10*time.Minute), now),
		Step:      testMetricStep(),
		Operation: &vmetricsv1.QueryOperation{
			Type: &vmetricsv1.QueryOperation_Counter{Counter: &vmetricsv1.CounterOperation{
				Function: vmetricsv1.CounterOperation_RATE,
			}},
		},
		GroupBy:           []string{"req.grpc.status"},
		SeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
	})
	require.NoError(t, err)
	assert.Len(t, res.Series, 2)

	res, err = s.QueryMetrics(ctx, &vmetricsv1.QueryMetricsRequest{
		Metric: &vmetricsv1.MetricSelector{
			Selector: &vmetricsv1.MetricSelector_Name{Name: "req.total"},
			Kind:     vmetricsv1.MetricDescriptor_COUNTER,
		},
		TimeRange: testMetricTimeRange(now.Add(-10*time.Minute), now),
		Step:      testMetricStep(),
		Operation: &vmetricsv1.QueryOperation{
			Type: &vmetricsv1.QueryOperation_Counter{Counter: &vmetricsv1.CounterOperation{
				Function: vmetricsv1.CounterOperation_RATE,
			}},
		},
		Filters: []*vmetricsv1.AttributeFilter{
			{
				Key:      "req.authenticated",
				Operator: vmetricsv1.AttributeFilter_EQ,
				Value: &vmetricsv1.AttributeValue{
					Value: &vmetricsv1.AttributeValue_BoolValue{BoolValue: true},
				},
			},
			{
				Key:      "req.grpc.status",
				Operator: vmetricsv1.AttributeFilter_EQ,
				Value: &vmetricsv1.AttributeValue{
					Value: &vmetricsv1.AttributeValue_StringValue{StringValue: "OK"},
				},
			},
		},
		GroupBy:           []string{"octelium.vigil.svc.mode"},
		SeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
	})
	require.NoError(t, err)
	assert.Len(t, res.Series, 1)
}

func TestVigilActiveSessionsAreSummedAcrossServices(t *testing.T) {
	s := newTestSrvMetric(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Minute)
	start := now.Add(-10 * time.Minute)
	at := now.Add(-time.Minute)

	metrics, sm := newVigilResourceMetrics()
	appendTestUpDownCounter(sm, "session.active", start, at, 3, map[string]any{
		"octelium.vigil.svc.name": "svc-tcp",
		"octelium.vigil.svc.mode": "TCP",
	})
	appendTestUpDownCounter(sm, "session.active", start, at, 4, map[string]any{
		"octelium.vigil.svc.name": "svc-ssh",
		"octelium.vigil.svc.mode": "SSH",
	})
	storeTestMetrics(t, s, metrics)

	res, err := s.QueryMetrics(ctx, &vmetricsv1.QueryMetricsRequest{
		Metric: &vmetricsv1.MetricSelector{
			Selector: &vmetricsv1.MetricSelector_Name{Name: "session.active"},
			Kind:     vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		},
		TimeRange: testMetricTimeRange(now.Add(-10*time.Minute), now),
		Step:      testMetricStep(),
		Operation: &vmetricsv1.QueryOperation{
			Type: &vmetricsv1.QueryOperation_Gauge{Gauge: &vmetricsv1.GaugeOperation{
				Function: vmetricsv1.GaugeOperation_LAST,
			}},
		},
		GroupBy:           []string{"octelium.component.type"},
		SeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
	})
	require.NoError(t, err)
	require.Len(t, res.Series, 1)

	points := res.Series[0].GetNumber().Points
	require.Len(t, points, 1)
	assert.Equal(t, float64(7), points[0].GetAsDouble())
}

func TestVigilCatalogItemsResolveTheirGroupByKeys(t *testing.T) {
	s := newTestSrvMetric(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Minute)
	start := now.Add(-10 * time.Minute)
	at := now.Add(-time.Minute)

	metrics, sm := newVigilResourceMetrics()
	appendTestUpDownCounter(sm, "session.active", start, at, 2, map[string]any{
		"octelium.vigil.svc.name": "svc",
		"octelium.vigil.svc.mode": "TCP",
	})
	appendTestCounter(sm, "conn.rejected", start, at, 5, map[string]any{
		"octelium.vigil.svc.name": "svc",
		"octelium.vigil.svc.mode": "TCP",
		"stage":                   "UPSTREAM_DIAL",
	})
	storeTestMetrics(t, s, metrics)

	catalog, err := s.ListMetricCatalog(ctx, &vmetricsv1.ListMetricCatalogRequest{
		Component: &vmetricsv1.ComponentSelector{Type: "vigil"},
	})
	require.NoError(t, err)

	for _, item := range catalog.Items {
		if item.Id != "active_sessions" && item.Id != "rejected_connections_rate" {
			continue
		}

		res, err := s.QueryMetrics(ctx, &vmetricsv1.QueryMetricsRequest{
			Metric:            item.Metric,
			TimeRange:         testMetricTimeRange(now.Add(-10*time.Minute), now),
			Step:              item.DefaultStep,
			Operation:         item.DefaultOperation,
			GroupBy:           item.DefaultGroupBy,
			Filters:           item.DefaultFilters,
			SeriesAggregation: item.DefaultSeriesAggregation,
		})
		require.NoError(t, err, item.Id)
		assert.NotEmpty(t, res.Series, item.Id)
	}
}

func testTokensQuery(from, to time.Time, component *vmetricsv1.ComponentSelector,
	filters []*vmetricsv1.AttributeFilter) *vmetricsv1.QueryMetricsRequest {
	return &vmetricsv1.QueryMetricsRequest{
		Metric: &vmetricsv1.MetricSelector{
			Selector: &vmetricsv1.MetricSelector_Name{Name: "llm.tokens.input"},
			Kind:     vmetricsv1.MetricDescriptor_COUNTER,
		},
		Component: component,
		TimeRange: testMetricTimeRange(from, to),
		Step:      testMetricStep(),
		Operation: &vmetricsv1.QueryOperation{
			Type: &vmetricsv1.QueryOperation_Counter{Counter: &vmetricsv1.CounterOperation{
				Function: vmetricsv1.CounterOperation_RATE,
			}},
		},
		Filters:           filters,
		SeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
	}
}

func TestQuietTimeRangeReturnsAnEmptyResultInsteadOfNotFound(t *testing.T) {
	s := newTestSrvMetric(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Minute)
	component := &vmetricsv1.ComponentSelector{Type: "vigil", Namespace: "octelium"}

	metrics, sm := newVigilResourceMetrics()
	appendTestCounter(sm, "llm.tokens.input", now.Add(-3*time.Hour), now.Add(-2*time.Hour), 100,
		map[string]any{
			"octelium.vigil.svc.name": "svc-llm",
			"octelium.vigil.svc.mode": "LLM",
			"req.llm.model":           "claude-opus-5",
		})
	storeTestMetrics(t, s, metrics)

	res, err := s.QueryMetrics(ctx, testTokensQuery(now.Add(-4*time.Hour), now, component, nil))
	require.NoError(t, err)
	assert.NotEmpty(t, res.Series)

	res, err = s.QueryMetrics(ctx, testTokensQuery(now.Add(-15*time.Minute), now, component, nil))
	require.NoError(t, err)
	assert.Empty(t, res.Series)
	assert.NotNil(t, res.SourceDescriptor)

	res, err = s.QueryMetrics(ctx, testTokensQuery(now.Add(-4*time.Hour), now, component,
		[]*vmetricsv1.AttributeFilter{
			{
				Key:      "octelium.vigil.svc.name",
				Operator: vmetricsv1.AttributeFilter_EQ,
				Value: &vmetricsv1.AttributeValue{
					Value: &vmetricsv1.AttributeValue_StringValue{StringValue: "svc-other"},
				},
			},
		}))
	require.NoError(t, err)
	assert.Empty(t, res.Series)
}

func TestUnknownMetricNameStaysNotFound(t *testing.T) {
	s := newTestSrvMetric(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Minute)

	metrics, sm := newVigilResourceMetrics()
	appendTestCounter(sm, "llm.tokens.input", now.Add(-3*time.Hour), now.Add(-2*time.Hour), 100,
		map[string]any{"octelium.vigil.svc.name": "svc-llm"})
	storeTestMetrics(t, s, metrics)

	req := testTokensQuery(now.Add(-4*time.Hour), now, nil, nil)
	req.Metric.Selector = &vmetricsv1.MetricSelector_Name{Name: "llm.tokens.nonexistent"}

	_, err := s.QueryMetrics(ctx, req)
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))

	req = testTokensQuery(now.Add(-4*time.Hour), now,
		&vmetricsv1.ComponentSelector{Type: "rscserver"}, nil)

	_, err = s.QueryMetrics(ctx, req)
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestAmbiguousMetricNameIsResolvedByComponent(t *testing.T) {
	s := newTestSrvMetric(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Minute)
	start, at := now.Add(-3*time.Hour), now.Add(-2*time.Hour)

	vigil, vigilScope := newVigilResourceMetrics()
	appendTestHistogram(vigilScope, "req.duration", "ms", start, at)
	storeTestMetrics(t, s, vigil)

	rscserver := pmetric.NewMetrics()
	rm := rscserver.ResourceMetrics().AppendEmpty()
	putTestAttributes(rm.Resource().Attributes(), map[string]any{
		"octelium.component.type":      "rscserver",
		"octelium.component.namespace": "octelium",
		"octelium.component.uid":       "octelium-rscserver-1",
	})
	rscserverScope := rm.ScopeMetrics().AppendEmpty()
	rscserverScope.Scope().SetName("default")
	appendTestHistogram(rscserverScope, "req.duration", "us", start, at)
	storeTestMetrics(t, s, rscserver)

	query := func(component *vmetricsv1.ComponentSelector, from time.Time) error {
		_, err := s.QueryMetrics(ctx, &vmetricsv1.QueryMetricsRequest{
			Metric: &vmetricsv1.MetricSelector{
				Selector: &vmetricsv1.MetricSelector_Name{Name: "req.duration"},
				Kind:     vmetricsv1.MetricDescriptor_HISTOGRAM,
			},
			Component: component,
			TimeRange: testMetricTimeRange(from, now),
			Step:      testMetricStep(),
			Operation: &vmetricsv1.QueryOperation{
				Type: &vmetricsv1.QueryOperation_Histogram{Histogram: &vmetricsv1.HistogramOperation{
					Function:  vmetricsv1.HistogramOperation_QUANTILE,
					Quantiles: []float64{0.95},
				}},
			},
			SeriesAggregation: vmetricsv1.QueryMetricsRequest_MERGE,
		})
		return err
	}

	assert.NoError(t, query(&vmetricsv1.ComponentSelector{Type: "vigil"}, now.Add(-4*time.Hour)))
	assert.NoError(t, query(&vmetricsv1.ComponentSelector{Type: "rscserver"}, now.Add(-4*time.Hour)))
	assert.NoError(t, query(&vmetricsv1.ComponentSelector{Type: "vigil"}, now.Add(-15*time.Minute)))

	err := query(nil, now.Add(-4*time.Hour))
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}
