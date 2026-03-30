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
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestServer(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	s := srv.newSrvMetric()
	{
		err = s.storeMetrics(tstCreateCounterMetrics())
		assert.Nil(t, err)
	}
	{
		err = s.storeMetrics(tstTestCreateGaugeMetrics())
		assert.Nil(t, err)
	}
	{
		err = s.storeMetrics(tstCreateHistogramMetrics())
		assert.Nil(t, err)
	}
	{
		err = s.storeMetrics(tstCreateSummaryMetrics())
		assert.Nil(t, err)
	}

	{
		res, err := s.doQueryCounter(ctx, "http_requests_total", time.Now().Add(-10*time.Second), time.Now(), nil)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(res.Points))

	}

	{
		res, err := s.doQueryGauge(ctx, "queue_depth", time.Now().Add(-10*time.Second), time.Now(), nil)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(res.Points))
	}

	{
		res, err := s.doQueryHistogram(ctx, "request_latency", time.Now().Add(-10*time.Second), time.Now(), nil)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(res))

	}

}

func tstCreateCounterMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()

	rm := metrics.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "test-counter-service")

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("my-test-library")

	m := sm.Metrics().AppendEmpty()
	m.SetName("http_requests_total")
	m.SetDescription("Total number of HTTP requests processed")
	m.SetUnit("1")

	sum := m.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dp := sum.DataPoints().AppendEmpty()

	start := pcommon.Timestamp(time.Now().Add(-1 * time.Minute).UnixNano())
	end := pcommon.Timestamp(time.Now().UnixNano())

	dp.SetStartTimestamp(start)
	dp.SetTimestamp(end)

	dp.SetIntValue(12345)

	dp.Attributes().PutStr("method", "POST")
	dp.Attributes().PutInt("status_code", 200)

	return metrics
}

func tstTestCreateGaugeMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()

	m := sm.Metrics().AppendEmpty()
	m.SetName("queue_depth")
	m.SetDescription("Current number of items in the processing queue")
	m.SetUnit("items")

	gauge := m.SetEmptyGauge()

	dp := gauge.DataPoints().AppendEmpty()

	end := pcommon.Timestamp(time.Now().UnixNano())
	dp.SetTimestamp(end)

	dp.SetDoubleValue(15.5)

	dp.Attributes().PutStr("queue_name", "priority_q")

	return metrics
}

func tstCreateHistogramMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()

	m := sm.Metrics().AppendEmpty()
	m.SetName("request_latency")
	m.SetDescription("Distribution of request latency in milliseconds")
	m.SetUnit("ms")

	hist := m.SetEmptyHistogram()
	hist.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)

	dp := hist.DataPoints().AppendEmpty()

	start := pcommon.Timestamp(time.Now().Add(-10 * time.Second).UnixNano())
	end := pcommon.Timestamp(time.Now().UnixNano())
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(end)

	dp.Attributes().PutStr("endpoint", "/api/v1/users")

	dp.SetCount(100)
	dp.SetSum(15000.0)

	dp.ExplicitBounds().FromRaw([]float64{100.0, 500.0, 1000.0})

	dp.BucketCounts().FromRaw([]uint64{50, 30, 15, 5})

	dp.SetMin(5.0)
	dp.SetMax(2500.0)

	return metrics
}

func tstCreateSummaryMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()

	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()

	m := sm.Metrics().AppendEmpty()
	m.SetName("db_query_time_summary")
	m.SetDescription("Summary of database query execution times in seconds")
	m.SetUnit("s")

	summary := m.SetEmptySummary()

	dp := summary.DataPoints().AppendEmpty()

	start := pcommon.Timestamp(time.Now().Add(-5 * time.Minute).UnixNano())
	end := pcommon.Timestamp(time.Now().UnixNano())
	dp.SetStartTimestamp(start)
	dp.SetTimestamp(end)

	dp.Attributes().PutStr("db_table", "users")

	dp.SetCount(5000)
	dp.SetSum(125.0)

	q1 := dp.QuantileValues().AppendEmpty()
	q1.SetQuantile(0.5)
	q1.SetValue(0.015)

	q2 := dp.QuantileValues().AppendEmpty()
	q2.SetQuantile(0.9)
	q2.SetValue(0.050)

	q3 := dp.QuantileValues().AppendEmpty()
	q3.SetQuantile(0.99)
	q3.SetValue(0.120)

	return metrics
}
