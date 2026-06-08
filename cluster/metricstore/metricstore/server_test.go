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

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMetricstoreGaugeFunctions(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstGaugeMetric("gauge.queue_depth", base, 10, 20, 30))

	{
		resp, err := s.QueryMetrics(ctx, tstReq("gauge.queue_depth", base, tstGaugeOp(vmetricsv1.GaugeOperation_LAST)))
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Series))
		assert.Equal(t, uint32(1), resp.TotalSeries)
		assert.Equal(t, false, resp.Truncated)
		assert.Equal(t, 1, len(resp.Series[0].GetNumber().Points))
		assert.Equal(t, float64(30), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("gauge.queue_depth", base, tstGaugeOp(vmetricsv1.GaugeOperation_AVG)))
		assert.Nil(t, err)
		assert.Equal(t, float64(20), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("gauge.queue_depth", base, tstGaugeOp(vmetricsv1.GaugeOperation_MIN)))
		assert.Nil(t, err)
		assert.Equal(t, float64(10), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("gauge.queue_depth", base, tstGaugeOp(vmetricsv1.GaugeOperation_MAX)))
		assert.Nil(t, err)
		assert.Equal(t, float64(30), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("gauge.queue_depth", base, tstGaugeOp(vmetricsv1.GaugeOperation_SUM)))
		assert.Nil(t, err)
		assert.Equal(t, float64(60), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
}

func TestMetricstoreCounterDelta(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstCounterMetric("counter.delta.requests", base, true, pmetric.AggregationTemporalityDelta, 5, 10, 15))

	{
		resp, err := s.QueryMetrics(ctx, tstReq("counter.delta.requests", base, tstCounterOp(vmetricsv1.CounterOperation_INCREASE)))
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Series))
		assert.Equal(t, 1, len(resp.Series[0].GetNumber().Points))
		assert.Equal(t, float64(30), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("counter.delta.requests", base, tstCounterOp(vmetricsv1.CounterOperation_RATE)))
		assert.Nil(t, err)
		assert.Equal(t, float64(7.5), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
}

func TestMetricstoreCounterCumulative(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstCounterMetric("counter.cumul.requests", base, true, pmetric.AggregationTemporalityCumulative, 100, 160, 220))

	{
		resp, err := s.QueryMetrics(ctx, tstReq("counter.cumul.requests", base, tstCounterOp(vmetricsv1.CounterOperation_INCREASE)))
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Series))
		assert.Equal(t, float64(120), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("counter.cumul.requests", base, tstCounterOp(vmetricsv1.CounterOperation_RATE)))
		assert.Nil(t, err)
		assert.Equal(t, float64(30), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
}

func TestMetricstoreCounterCumulativeReset(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s,
		tstCounterMetric(
			"counter.reset.requests",
			base,
			true,
			pmetric.AggregationTemporalityCumulative,
			100, 150, 40, 60,
		),
	)

	req := tstReq("counter.reset.requests", base, tstCounterOp(vmetricsv1.CounterOperation_INCREASE))
	req.TimeRange.To = pbutils.Timestamp(base.Add(4 * time.Second))

	resp, err := s.QueryMetrics(ctx, req)
	assert.NoError(t, err)
	assert.Len(t, resp.Series, 1)
	assert.Len(t, resp.Series[0].GetNumber().Points, 1)

	assert.Equal(t, float64(110), resp.Series[0].GetNumber().Points[0].GetAsDouble())
}

func TestMetricstoreCounterRaw(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstCounterMetric("counter.raw.requests", base, true, pmetric.AggregationTemporalityCumulative, 100, 160, 220))

	resp, err := s.QueryMetrics(ctx, tstReq("counter.raw.requests", base, tstCounterOp(vmetricsv1.CounterOperation_RAW)))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(resp.Series))
	assert.Equal(t, 3, len(resp.Series[0].GetNumber().Points))
	assert.Equal(t, int64(100), resp.Series[0].GetNumber().Points[0].GetAsInt())
	assert.Equal(t, int64(160), resp.Series[0].GetNumber().Points[1].GetAsInt())
	assert.Equal(t, int64(220), resp.Series[0].GetNumber().Points[2].GetAsInt())
}

func TestMetricstoreHistogramFunctions(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstHistogramMetric("hist.latency", base, pmetric.AggregationTemporalityDelta,
		100, 15000, []float64{100, 500, 1000}, []uint64{50, 30, 15, 5}))

	{
		resp, err := s.QueryMetrics(ctx, tstReq("hist.latency", base, tstHistogramOp(vmetricsv1.HistogramOperation_COUNT)))
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Series))
		assert.Equal(t, float64(100), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("hist.latency", base, tstHistogramOp(vmetricsv1.HistogramOperation_SUM)))
		assert.Nil(t, err)
		assert.Equal(t, float64(15000), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("hist.latency", base, tstHistogramOp(vmetricsv1.HistogramOperation_AVG)))
		assert.Nil(t, err)
		assert.Equal(t, float64(150), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("hist.latency", base, tstHistogramOp(vmetricsv1.HistogramOperation_BUCKETS)))
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Series))
		buckets := resp.Series[0].GetHistogram().Points[0].Buckets
		assert.Equal(t, 4, len(buckets))
		assert.Equal(t, float64(100), buckets[0].Le)
		assert.Equal(t, uint64(50), buckets[0].Count)
		assert.Equal(t, uint64(80), buckets[1].Count)
		assert.Equal(t, uint64(95), buckets[2].Count)
		assert.Equal(t, uint64(100), buckets[3].Count)
		assert.Equal(t, true, buckets[3].IsInf)
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("hist.latency", base, tstHistogramOp(vmetricsv1.HistogramOperation_QUANTILE, 0.5)))
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Series))
		assert.Equal(t, "quantile", resp.Series[0].Labels[len(resp.Series[0].Labels)-1].Key)
		assert.Equal(t, "0.5", resp.Series[0].Labels[len(resp.Series[0].Labels)-1].Value)
		assert.Equal(t, float64(100), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("hist.latency", base, tstHistogramOp(vmetricsv1.HistogramOperation_QUANTILE, 0.9)))
		assert.Nil(t, err)
		assert.InDelta(t, 833.33, resp.Series[0].GetNumber().Points[0].GetAsDouble(), 0.1)
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("hist.latency", base, tstHistogramOp(vmetricsv1.HistogramOperation_QUANTILE, 0.5, 0.9, 0.99)))
		assert.Nil(t, err)
		assert.Equal(t, 3, len(resp.Series))
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("hist.latency", base, tstHistogramOp(vmetricsv1.HistogramOperation_MIN)))
		assert.NotNil(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
}

func TestMetricstoreGroupBy(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstGaugeByComponent("grp.gauge", base, map[string]float64{
		"vigil":  10,
		"portal": 20,
	}))

	req := tstReq("grp.gauge", base, tstGaugeOp(vmetricsv1.GaugeOperation_LAST))
	req.GroupBy = []string{"octelium.component.type"}

	resp, err := s.QueryMetrics(ctx, req)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(resp.Series))

	vigil := tstSeriesByLabel(resp.Series, "octelium.component.type", "vigil")
	portal := tstSeriesByLabel(resp.Series, "octelium.component.type", "portal")
	assert.NotNil(t, vigil)
	assert.NotNil(t, portal)
	assert.Equal(t, float64(10), vigil.GetNumber().Points[0].GetAsDouble())
	assert.Equal(t, float64(20), portal.GetNumber().Points[0].GetAsDouble())
}

func TestMetricstoreAttributeFilters(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstGaugeWithMethod("filt.gauge", base, map[string]float64{
		"POST": 10,
		"GET":  20,
	}))

	sumWith := func(filters ...*vmetricsv1.AttributeFilter) (*vmetricsv1.QueryMetricsResponse, error) {
		req := tstReq("filt.gauge", base, tstGaugeOp(vmetricsv1.GaugeOperation_SUM))
		req.Filters = filters
		return s.QueryMetrics(ctx, req)
	}

	{
		resp, err := sumWith(&vmetricsv1.AttributeFilter{Key: "http.method", Operator: vmetricsv1.AttributeFilter_EQ, Value: "POST"})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Series))
		assert.Equal(t, float64(10), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := sumWith(&vmetricsv1.AttributeFilter{Key: "http.method", Operator: vmetricsv1.AttributeFilter_NOT_EQ, Value: "POST"})
		assert.Nil(t, err)
		assert.Equal(t, float64(20), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := sumWith(&vmetricsv1.AttributeFilter{Key: "http.method", Operator: vmetricsv1.AttributeFilter_IN, Values: []string{"POST", "GET"}})
		assert.Nil(t, err)
		assert.Equal(t, float64(30), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := sumWith(&vmetricsv1.AttributeFilter{Key: "http.method", Operator: vmetricsv1.AttributeFilter_EXISTS})
		assert.Nil(t, err)
		assert.Equal(t, float64(30), resp.Series[0].GetNumber().Points[0].GetAsDouble())
	}
	{
		resp, err := sumWith(&vmetricsv1.AttributeFilter{Key: "http.method", Operator: vmetricsv1.AttributeFilter_NOT_EXISTS})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(resp.Series))
	}
}

func TestMetricstoreListMetricDescriptors(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstCounterMetric("ld.counter", base, true, pmetric.AggregationTemporalityCumulative, 1))
	tstStore(t, ctx, s, tstGaugeWithMethod("ld.gauge", base, map[string]float64{"GET": 5}))
	tstStore(t, ctx, s, tstHistogramMetric("ld.hist", base, pmetric.AggregationTemporalityDelta,
		10, 100, []float64{1, 2}, []uint64{3, 3, 4}))

	{
		resp, err := s.ListMetricDescriptors(ctx, &vmetricsv1.ListMetricDescriptorsRequest{NamePrefix: "ld."})
		assert.Nil(t, err)
		assert.Equal(t, 3, len(resp.Items))
		assert.Equal(t, "ld.counter", resp.Items[0].Name)
		assert.Equal(t, vmetricsv1.MetricDescriptor_COUNTER, resp.Items[0].Kind)
		assert.Equal(t, vmetricsv1.MetricDescriptor_CUMULATIVE, resp.Items[0].Temporality)
		assert.Equal(t, "ld.gauge", resp.Items[1].Name)
		assert.Equal(t, vmetricsv1.MetricDescriptor_GAUGE, resp.Items[1].Kind)
		assert.Equal(t, "ld.hist", resp.Items[2].Name)
		assert.Equal(t, vmetricsv1.MetricDescriptor_HISTOGRAM, resp.Items[2].Kind)
		assert.Equal(t, true, tstContains(resp.Items[1].AttributeKeys, "http.method"))
	}
	{
		resp, err := s.ListMetricDescriptors(ctx, &vmetricsv1.ListMetricDescriptorsRequest{NamePrefix: "ld.", Limit: 1})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, "ld.counter", resp.Items[0].Name)
		assert.Equal(t, "ld.counter", resp.NextPageToken)
	}
	{
		resp, err := s.ListMetricDescriptors(ctx, &vmetricsv1.ListMetricDescriptorsRequest{NamePrefix: "ld.", Limit: 1, PageToken: "ld.counter"})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, "ld.gauge", resp.Items[0].Name)
		assert.Equal(t, "ld.gauge", resp.NextPageToken)
	}
	{
		resp, err := s.ListMetricDescriptors(ctx, &vmetricsv1.ListMetricDescriptorsRequest{NamePrefix: "ld.", Limit: 1, PageToken: "ld.gauge"})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, "ld.hist", resp.Items[0].Name)
		assert.Equal(t, "", resp.NextPageToken)
	}
}

func TestMetricstoreListMetricCatalog(t *testing.T) {
	ctx, s := tstSetup(t)

	{
		resp, err := s.ListMetricCatalog(ctx, &vmetricsv1.ListMetricCatalogRequest{})
		assert.Nil(t, err)
		assert.Equal(t, true, tstHasCatalogID(resp.Items, "process_goroutines"))
		assert.Equal(t, true, tstHasCatalogID(resp.Items, "requests_rate"))
	}
	{
		resp, err := s.ListMetricCatalog(ctx, &vmetricsv1.ListMetricCatalogRequest{Component: &vmetricsv1.ComponentSelector{Type: "vigil"}})
		assert.Nil(t, err)
		assert.Equal(t, true, tstHasCatalogID(resp.Items, "process_goroutines"))
		assert.Equal(t, true, tstHasCatalogID(resp.Items, "requests_rate"))
	}
	{
		resp, err := s.ListMetricCatalog(ctx, &vmetricsv1.ListMetricCatalogRequest{Component: &vmetricsv1.ComponentSelector{Type: "gateway"}})
		assert.Nil(t, err)
		assert.Equal(t, true, tstHasCatalogID(resp.Items, "process_goroutines"))
		assert.Equal(t, false, tstHasCatalogID(resp.Items, "requests_rate"))
	}
}

func TestMetricstoreExponentialHistogramUnimplemented(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstExpHistogramMetric("eh.test", base))

	{
		resp, err := s.ListMetricDescriptors(ctx, &vmetricsv1.ListMetricDescriptorsRequest{NamePrefix: "eh."})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM, resp.Items[0].Kind)
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("eh.test", base, tstHistogramOp(vmetricsv1.HistogramOperation_BUCKETS)))
		assert.NotNil(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Unimplemented, status.Code(err))
	}
}

func TestMetricstoreSummaryDropped(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstSummaryMetric("sm.test", base))

	{
		resp, err := s.ListMetricDescriptors(ctx, &vmetricsv1.ListMetricDescriptorsRequest{NamePrefix: "sm."})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(resp.Items))
	}
	{
		resp, err := s.QueryMetrics(ctx, tstReq("sm.test", base, tstGaugeOp(vmetricsv1.GaugeOperation_LAST)))
		assert.NotNil(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.NotFound, status.Code(err))
	}
}

func TestMetricstoreQueryNotFound(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	resp, err := s.QueryMetrics(ctx, tstReq("does.not.exist", base, tstGaugeOp(vmetricsv1.GaugeOperation_LAST)))
	assert.NotNil(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestMetricstoreKindValidation(t *testing.T) {
	ctx, s := tstSetup(t)
	base := time.Now().UTC()

	tstStore(t, ctx, s, tstCounterMetric("km.counter", base, true, pmetric.AggregationTemporalityCumulative, 1, 2))

	{
		resp, err := s.QueryMetrics(ctx, tstReq("km.counter", base, tstGaugeOp(vmetricsv1.GaugeOperation_LAST)))
		assert.NotNil(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
	{
		req := tstReq("km.counter", base, tstCounterOp(vmetricsv1.CounterOperation_RATE))
		req.Metric.Kind = vmetricsv1.MetricDescriptor_GAUGE
		resp, err := s.QueryMetrics(ctx, req)
		assert.NotNil(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
	{
		req := tstReq("km.counter", base, tstCounterOp(vmetricsv1.CounterOperation_RATE))
		req.Metric.Kind = vmetricsv1.MetricDescriptor_COUNTER
		resp, err := s.QueryMetrics(ctx, req)
		assert.Nil(t, err)
		assert.NotNil(t, resp)
	}
}

func TestMetricstoreRequestValidation(t *testing.T) {
	ctx, s := tstSetup(t)

	{
		resp, err := s.QueryMetrics(ctx, nil)
		assert.NotNil(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}

	cases := []struct {
		name  string
		build func() *vmetricsv1.QueryMetricsRequest
	}{
		{
			name: "empty-name",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Metric.Name = ""
				return r
			},
		},
		{
			name: "nil-timerange",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.TimeRange = nil
				return r
			},
		},
		{
			name: "to-before-from",
			build: func() *vmetricsv1.QueryMetricsRequest {
				base := time.Now().UTC()
				r := tstValidReq()
				r.TimeRange = &vmetricsv1.TimeRange{From: timestamppb.New(base), To: timestamppb.New(base.Add(-time.Minute))}
				return r
			},
		},
		{
			name: "range-too-large",
			build: func() *vmetricsv1.QueryMetricsRequest {
				base := time.Now().UTC()
				r := tstValidReq()
				r.TimeRange = &vmetricsv1.TimeRange{From: timestamppb.New(base.Add(-31 * 24 * time.Hour)), To: timestamppb.New(base)}
				return r
			},
		},
		{
			name: "step-too-small",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Step = &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 500}}
				return r
			},
		},
		{
			name: "too-many-points",
			build: func() *vmetricsv1.QueryMetricsRequest {
				base := time.Now().UTC()
				r := tstValidReq()
				r.TimeRange = &vmetricsv1.TimeRange{From: timestamppb.New(base.Add(-24 * time.Hour)), To: timestamppb.New(base)}
				r.Step = tstSeconds(1)
				return r
			},
		},
		{
			name: "exceeds-limit-points",
			build: func() *vmetricsv1.QueryMetricsRequest {
				base := time.Now().UTC()
				r := tstValidReq()
				r.TimeRange = &vmetricsv1.TimeRange{From: timestamppb.New(base.Add(-100 * time.Second)), To: timestamppb.New(base)}
				r.Step = tstSeconds(1)
				r.LimitPointsPerSeries = 5
				return r
			},
		},
		{
			name: "too-many-groupby",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.GroupBy = []string{"service.a", "service.b", "service.c", "service.d", "service.e", "service.f", "service.g", "service.h", "service.i"}
				return r
			},
		},
		{
			name: "disallowed-groupby",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.GroupBy = []string{"forbidden.key"}
				return r
			},
		},
		{
			name: "duplicate-groupby",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.GroupBy = []string{"service.a", "service.a"}
				return r
			},
		},
		{
			name: "nil-operation",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Operation = nil
				return r
			},
		},
		{
			name: "raw-with-groupby",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Operation = tstCounterOp(vmetricsv1.CounterOperation_RAW)
				r.GroupBy = []string{"service.a"}
				return r
			},
		},
		{
			name: "quantile-empty",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Operation = tstHistogramOp(vmetricsv1.HistogramOperation_QUANTILE)
				return r
			},
		},
		{
			name: "quantile-out-of-range",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Operation = tstHistogramOp(vmetricsv1.HistogramOperation_QUANTILE, 1.5)
				return r
			},
		},
		{
			name: "too-many-quantiles",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Operation = tstHistogramOp(vmetricsv1.HistogramOperation_QUANTILE, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99)
				return r
			},
		},
		{
			name: "filter-eq-no-value",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Filters = []*vmetricsv1.AttributeFilter{{Key: "http.method", Operator: vmetricsv1.AttributeFilter_EQ}}
				return r
			},
		},
		{
			name: "filter-eq-with-values",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Filters = []*vmetricsv1.AttributeFilter{{Key: "http.method", Operator: vmetricsv1.AttributeFilter_EQ, Value: "a", Values: []string{"b"}}}
				return r
			},
		},
		{
			name: "filter-in-no-values",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Filters = []*vmetricsv1.AttributeFilter{{Key: "http.method", Operator: vmetricsv1.AttributeFilter_IN}}
				return r
			},
		},
		{
			name: "filter-in-too-many",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Filters = []*vmetricsv1.AttributeFilter{{Key: "http.method", Operator: vmetricsv1.AttributeFilter_IN, Values: tstStrings(129)}}
				return r
			},
		},
		{
			name: "filter-duplicate-key",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Filters = []*vmetricsv1.AttributeFilter{
					{Key: "http.method", Operator: vmetricsv1.AttributeFilter_EXISTS},
					{Key: "http.method", Operator: vmetricsv1.AttributeFilter_EXISTS},
				}
				return r
			},
		},
		{
			name: "filter-disallowed-key",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Filters = []*vmetricsv1.AttributeFilter{{Key: "forbidden", Operator: vmetricsv1.AttributeFilter_EXISTS}}
				return r
			},
		},
		{
			name: "filter-nil",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Filters = []*vmetricsv1.AttributeFilter{nil}
				return r
			},
		},
		{
			name: "filter-invalid-operator",
			build: func() *vmetricsv1.QueryMetricsRequest {
				r := tstValidReq()
				r.Filters = []*vmetricsv1.AttributeFilter{{Key: "http.method", Operator: vmetricsv1.AttributeFilter_OPERATOR_UNSET}}
				return r
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, err := s.QueryMetrics(ctx, c.build())
			assert.NotNil(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func tstSetup(t *testing.T) (context.Context, *srvMetric) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err)

	srv, err := newServer(ctx, tst.C.OcteliumC)
	assert.Nil(t, err)

	assert.Nil(t, srv.initDB(ctx))

	t.Cleanup(func() {
		if srv.db != nil {
			_ = srv.db.Close()
		}
		tst.Destroy()
	})

	return ctx, srv.newSrvMetric()
}

func tstStore(t *testing.T, ctx context.Context, s *srvMetric, md pmetric.Metrics) {
	assert.Nil(t, s.storeMetrics(ctx, md))
}

func tstReq(name string, base time.Time, op *vmetricsv1.QueryOperation) *vmetricsv1.QueryMetricsRequest {
	return &vmetricsv1.QueryMetricsRequest{
		Metric:    &vmetricsv1.MetricSelector{Name: name},
		TimeRange: &vmetricsv1.TimeRange{From: timestamppb.New(base.Add(-time.Second)), To: timestamppb.New(base.Add(3 * time.Second))},
		Step:      tstSeconds(4),
		Operation: op,
	}
}

func tstValidReq() *vmetricsv1.QueryMetricsRequest {
	base := time.Now().UTC()
	return &vmetricsv1.QueryMetricsRequest{
		Metric:    &vmetricsv1.MetricSelector{Name: "valid.metric"},
		TimeRange: &vmetricsv1.TimeRange{From: timestamppb.New(base.Add(-time.Minute)), To: timestamppb.New(base)},
		Step:      tstSeconds(30),
		Operation: tstGaugeOp(vmetricsv1.GaugeOperation_LAST),
	}
}

func tstSeconds(s uint32) *metav1.Duration {
	return &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: s}}
}

func tstCounterOp(fn vmetricsv1.CounterOperation_Function) *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{Type: &vmetricsv1.QueryOperation_Counter{Counter: &vmetricsv1.CounterOperation{Function: fn}}}
}

func tstGaugeOp(fn vmetricsv1.GaugeOperation_Function) *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{Type: &vmetricsv1.QueryOperation_Gauge{Gauge: &vmetricsv1.GaugeOperation{Function: fn}}}
}

func tstHistogramOp(fn vmetricsv1.HistogramOperation_Function, quantiles ...float64) *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{Type: &vmetricsv1.QueryOperation_Histogram{Histogram: &vmetricsv1.HistogramOperation{Function: fn, Quantiles: quantiles}}}
}

func tstGaugeMetric(name string, base time.Time, vals ...float64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	g := m.SetEmptyGauge()

	for i, v := range vals {
		dp := g.DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.Timestamp(base.Add(time.Duration(i) * time.Second).UnixNano()))
		dp.SetDoubleValue(v)
	}

	return md
}

func tstCounterMetric(name string, base time.Time, monotonic bool, temporality pmetric.AggregationTemporality, vals ...int64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	sum := m.SetEmptySum()
	sum.SetIsMonotonic(monotonic)
	sum.SetAggregationTemporality(temporality)

	for i, v := range vals {
		dp := sum.DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.Timestamp(base.Add(time.Duration(i) * time.Second).UnixNano()))
		dp.SetIntValue(v)
	}

	return md
}

func tstHistogramMetric(name string, base time.Time, temporality pmetric.AggregationTemporality, count uint64, sum float64, bounds []float64, bucketCounts []uint64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	h := m.SetEmptyHistogram()
	h.SetAggregationTemporality(temporality)

	dp := h.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.Timestamp(base.UnixNano()))
	dp.SetCount(count)
	dp.SetSum(sum)
	dp.ExplicitBounds().FromRaw(bounds)
	dp.BucketCounts().FromRaw(bucketCounts)

	return md
}

func tstExpHistogramMetric(name string, base time.Time) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	eh := m.SetEmptyExponentialHistogram()
	eh.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	dp := eh.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.Timestamp(base.UnixNano()))
	dp.SetCount(10)
	dp.SetSum(100)
	dp.SetScale(1)
	dp.SetZeroCount(2)
	dp.Positive().SetOffset(0)
	dp.Positive().BucketCounts().FromRaw([]uint64{3, 5})
	dp.Negative().SetOffset(0)
	dp.Negative().BucketCounts().FromRaw([]uint64{})

	return md
}

func tstSummaryMetric(name string, base time.Time) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	summary := m.SetEmptySummary()

	dp := summary.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.Timestamp(base.UnixNano()))
	dp.SetCount(5000)
	dp.SetSum(125.0)

	q := dp.QuantileValues().AppendEmpty()
	q.SetQuantile(0.5)
	q.SetValue(0.015)

	return md
}

func tstGaugeByComponent(name string, base time.Time, byType map[string]float64) pmetric.Metrics {
	md := pmetric.NewMetrics()

	for typ, v := range byType {
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().PutStr("octelium.component.type", typ)
		sm := rm.ScopeMetrics().AppendEmpty()
		m := sm.Metrics().AppendEmpty()
		m.SetName(name)
		g := m.SetEmptyGauge()
		dp := g.DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.Timestamp(base.UnixNano()))
		dp.SetDoubleValue(v)
	}

	return md
}

func tstGaugeWithMethod(name string, base time.Time, byMethod map[string]float64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	g := m.SetEmptyGauge()

	for method, v := range byMethod {
		dp := g.DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.Timestamp(base.UnixNano()))
		dp.SetDoubleValue(v)
		dp.Attributes().PutStr("http.method", method)
	}

	return md
}

func tstSeriesByLabel(series []*vmetricsv1.TimeSeries, key, val string) *vmetricsv1.TimeSeries {
	for _, ts := range series {
		for _, l := range ts.Labels {
			if l.Key == key && l.Value == val {
				return ts
			}
		}
	}

	return nil
}

func tstHasCatalogID(items []*vmetricsv1.MetricCatalogItem, id string) bool {
	for _, item := range items {
		if item.Id == id {
			return true
		}
	}

	return false
}

func tstContains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}

	return false
}

func tstStrings(n int) []string {
	ret := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ret = append(ret, string(rune('a'+(i%26)))+string(rune('0'+(i%10))))
	}
	return ret
}
