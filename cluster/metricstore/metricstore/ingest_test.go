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
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestQueueBudgetBoundsAndRelease(t *testing.T) {
	budget := &queueBudget{}
	assert.True(t, budget.reserve(100, 1000))

	snapshot := budget.snapshot()
	assert.Equal(t, int64(1), snapshot.requests)
	assert.Equal(t, int64(100), snapshot.points)
	assert.Equal(t, int64(1000), snapshot.bytes)

	assert.False(t, budget.reserve(maxQueuedDataPoints, 1))
	budget.release(100, 1000)

	snapshot = budget.snapshot()
	assert.Zero(t, snapshot.requests)
	assert.Zero(t, snapshot.points)
	assert.Zero(t, snapshot.bytes)
}

func TestInspectMetricsRejectsZeroTimestamp(t *testing.T) {
	metrics := pmetric.NewMetrics()
	metric := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("test.metric")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)

	_, _, err := inspectMetrics(metrics)
	assert.NotNil(t, err)
}

func TestInspectMetricsRejectsNonFiniteAttribute(t *testing.T) {
	metrics := pmetric.NewMetrics()
	metric := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("test.metric")
	point := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	point.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	point.SetIntValue(1)
	point.Attributes().PutDouble("invalid", math.NaN())

	_, _, err := inspectMetrics(metrics)
	assert.NotNil(t, err)
}

func TestValidateExplicitHistogram(t *testing.T) {
	assert.Nil(t, validateExplicitHistogram([]uint64{2, 3, 5}, []float64{10, 100}, 10))
	assert.NotNil(t, validateExplicitHistogram([]uint64{2, 3}, []float64{10, 100}, 5))
	assert.NotNil(t, validateExplicitHistogram([]uint64{2, 3, 5}, []float64{100, 10}, 10))
	assert.NotNil(t, validateExplicitHistogram([]uint64{2, 3, 5}, []float64{10, math.Inf(1)}, 10))
}

func TestMetricPointTimeRange(t *testing.T) {
	now := time.Now().UTC()
	points := []numberPointRecord{
		{timestamp: now},
		{timestamp: now.Add(-time.Hour)},
		{timestamp: now.Add(time.Minute)},
		{timestamp: now.Add(-time.Minute)},
	}

	from, to := metricPointTimeRange(points, func(point numberPointRecord) time.Time { return point.timestamp })
	assert.Equal(t, metricTimeToDB(now.Add(-time.Hour)), from)
	assert.Equal(t, metricTimeToDB(now.Add(time.Minute)), to)

	from, to = metricPointTimeRange(points[:1], func(point numberPointRecord) time.Time { return point.timestamp })
	assert.Equal(t, metricTimeToDB(now), from)
	assert.Equal(t, metricTimeToDB(now), to)
}

func TestValidateMetricPointTimes(t *testing.T) {
	now := time.Now().UTC()
	timestamp := pcommon.NewTimestampFromTime(now)
	start := pcommon.NewTimestampFromTime(now.Add(-time.Minute))

	assert.Nil(t, validateMetricPointTimes(timestamp, start, now))
	assert.NotNil(t, validateMetricPointTimes(0, 0, now))
	assert.NotNil(t, validateMetricPointTimes(timestamp, pcommon.NewTimestampFromTime(now.Add(time.Minute)), now))
}

func TestMetricIngestionValidationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(pmetric.Metric, pmetric.NumberDataPoint)
	}{
		{"EmptyName", func(m pmetric.Metric, p pmetric.NumberDataPoint) { m.SetName("") }},
		{"LongName", func(m pmetric.Metric, p pmetric.NumberDataPoint) {
			m.SetName(strings.Repeat("a", maxMetricNameLength+1))
		}},
		{"LongUnit", func(m pmetric.Metric, p pmetric.NumberDataPoint) {
			m.SetUnit(strings.Repeat("a", maxMetricUnitLength+1))
		}},
		{"LongDescription", func(m pmetric.Metric, p pmetric.NumberDataPoint) {
			m.SetDescription(strings.Repeat("a", maxMetricDescriptionLen+1))
		}},
		{"NonFinite", func(m pmetric.Metric, p pmetric.NumberDataPoint) { p.SetDoubleValue(math.Inf(1)) }},
		{"Future", func(m pmetric.Metric, p pmetric.NumberDataPoint) {
			p.SetTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(maximumFutureSkew + time.Hour)))
		}},
		{"Past", func(m pmetric.Metric, p pmetric.NumberDataPoint) {
			p.SetTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-maximumPastSkew - time.Hour)))
		}},
		{"EmptyAttributeKey", func(m pmetric.Metric, p pmetric.NumberDataPoint) { p.Attributes().PutStr("", "value") }},
		{"LongAttributeKey", func(m pmetric.Metric, p pmetric.NumberDataPoint) {
			p.Attributes().PutStr(strings.Repeat("a", maxAttributeKeyBytes+1), "value")
		}},
		{"LongAttributeValue", func(m pmetric.Metric, p pmetric.NumberDataPoint) {
			p.Attributes().PutStr("key", strings.Repeat("a", maxAttributeValueBytes+1))
		}},
		{"TooManyAttributes", func(m pmetric.Metric, p pmetric.NumberDataPoint) {
			for i := range maxAttributesPerPoint + 1 {
				p.Attributes().PutInt(fmt.Sprint(i), 1)
			}
		}},
		{"CompositeAttribute", func(m pmetric.Metric, p pmetric.NumberDataPoint) { p.Attributes().PutEmptyMap("map") }},
		{"Summary", func(m pmetric.Metric, p pmetric.NumberDataPoint) { m.SetEmptySummary() }},
		{"UnsetType", func(m pmetric.Metric, p pmetric.NumberDataPoint) { m.SetEmptyGauge(); m.SetEmptySum() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metrics := pmetric.NewMetrics()
			m := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
			m.SetName("test")
			p := m.SetEmptyGauge().DataPoints().AppendEmpty()
			p.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			p.SetIntValue(1)
			tc.change(m, p)
			_, _, err := inspectMetrics(metrics)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestMetricQueueBudgetExactLimits(t *testing.T) {
	budget := &queueBudget{}
	for range maxQueuedExports {
		assert.True(t, budget.reserve(1, 1))
	}
	assert.False(t, budget.reserve(1, 1))
	for range maxQueuedExports {
		budget.release(1, 1)
	}
	assert.Equal(t, queueBudgetSnapshot{}, budget.snapshot())
	assert.True(t, budget.reserve(maxQueuedDataPoints, maxQueuedEstimatedBytes))
	assert.False(t, budget.reserve(1, 0))
	assert.False(t, budget.reserve(0, 1))
	budget.release(maxQueuedDataPoints, maxQueuedEstimatedBytes)
	assert.Equal(t, queueBudgetSnapshot{}, budget.snapshot())
}

func TestMetricExportRejectsInvalidBatchWithoutWrites(t *testing.T) {
	conn, metric := newMetricStoreRPCConn(t)
	client := pmetricotlp.NewGRPCClient(conn)
	req := metricStoreAllKindsExport(time.Now().Add(-time.Minute), 0)
	req.Metrics().ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(3).Histogram().DataPoints().At(0).SetCount(99)
	_, err := client.Export(t.Context(), req)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	var count int
	require.NoError(t, metric.s.database().QueryRow(`SELECT COUNT(*) FROM metric_descriptors`).Scan(&count))
	assert.Zero(t, count)
	assert.Equal(t, queueBudgetSnapshot{}, metric.budget.snapshot())
	_, err = client.Export(t.Context(), metricStoreAllKindsExport(time.Now().Add(-time.Minute), 0))
	require.NoError(t, err)
	metric.stopAccepting()
	_, err = client.Export(t.Context(), pmetricotlp.NewExportRequest())
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestMetricProcessorReleasesCanceledExports(t *testing.T) {
	metric := (&Server{db: newTestDuckDB(t)}).newSrvMetric()
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() {
		_, err := metric.Export(ctx, metricStoreAllKindsExport(time.Now().Add(-time.Minute), 0))
		result <- err
	}()
	require.Eventually(t, func() bool { return len(metric.itemCh) == 1 }, time.Second, time.Millisecond)
	cancel()
	assert.ErrorIs(t, <-result, context.Canceled)
	metric.stopAccepting()
	metric.closeQueue()
	done := make(chan struct{})
	go func() { defer close(done); metric.startProcessLoop(t.Context()) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("processor did not drain")
	}
	assert.Equal(t, queueBudgetSnapshot{}, metric.budget.snapshot())
	assert.Zero(t, metric.storedPoints.Load())
}

func TestMetricHistogramValidationEdgeCases(t *testing.T) {
	for _, tc := range []struct {
		counts []uint64
		bounds []float64
		count  uint64
	}{
		{[]uint64{1, 1}, []float64{1}, 3},
		{[]uint64{math.MaxUint64, 1}, []float64{1}, 0},
		{[]uint64{1, 1, 1}, []float64{1, 1}, 3},
		{[]uint64{1, 1}, []float64{math.NaN()}, 2},
	} {
		assert.Error(t, validateExplicitHistogram(tc.counts, tc.bounds, tc.count))
	}
	assert.Error(t, validateExponentialHistogram([]uint64{1}, []uint64{1}, 1, 2))
	assert.NoError(t, validateExponentialHistogram([]uint64{1}, []uint64{1}, 1, 3))
	assert.Error(t, validateBucketOffset(math.MaxInt32, 2))
	assert.NoError(t, validateBucketOffset(math.MaxInt32, 1))
	assert.NoError(t, validateBucketOffset(math.MinInt32, 0))
	assert.Error(t, validateHistogramValues(true, math.Inf(1), false, 0, false, 0))
	assert.Error(t, validateHistogramValues(false, 0, true, math.NaN(), false, 0))
	assert.Error(t, validateHistogramValues(false, 0, false, 0, true, math.Inf(-1)))
	assert.Error(t, validateHistogramValues(true, 1, true, 2, true, 1))
}
