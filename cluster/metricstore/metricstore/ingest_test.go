// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
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

func TestValidateMetricPointTimes(t *testing.T) {
	now := time.Now().UTC()
	timestamp := pcommon.NewTimestampFromTime(now)
	start := pcommon.NewTimestampFromTime(now.Add(-time.Minute))

	assert.Nil(t, validateMetricPointTimes(timestamp, start, now))
	assert.NotNil(t, validateMetricPointTimes(0, 0, now))
	assert.NotNil(t, validateMetricPointTimes(timestamp, pcommon.NewTimestampFromTime(now.Add(time.Minute)), now))
}
