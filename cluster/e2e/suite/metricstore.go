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
	"fmt"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func metricDuration(in *metav1.Duration) time.Duration {
	switch {
	case in.GetMilliseconds() > 0:
		return time.Duration(in.GetMilliseconds()) * time.Millisecond
	case in.GetSeconds() > 0:
		return time.Duration(in.GetSeconds()) * time.Second
	case in.GetMinutes() > 0:
		return time.Duration(in.GetMinutes()) * time.Minute
	case in.GetHours() > 0:
		return time.Duration(in.GetHours()) * time.Hour
	case in.GetDays() > 0:
		return time.Duration(in.GetDays()) * 24 * time.Hour
	case in.GetWeeks() > 0:
		return time.Duration(in.GetWeeks()) * 7 * 24 * time.Hour
	default:
		return 0
	}
}

func metricTimeRange(from, to time.Time) *vmetricsv1.TimeRange {
	return &vmetricsv1.TimeRange{
		From: pbutils.Timestamp(from),
		To:   pbutils.Timestamp(to),
	}
}

func counterRaw() *vmetricsv1.QueryOperation {
	return &vmetricsv1.QueryOperation{
		Type: &vmetricsv1.QueryOperation_Counter{
			Counter: &vmetricsv1.CounterOperation{
				Function: vmetricsv1.CounterOperation_RAW,
			},
		},
	}
}

func testMetricStoreLimits(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	caps, err := h.MetricsC().GetMetricsCapabilities(ctx,
		&vmetricsv1.GetMetricsCapabilitiesRequest{})
	require.Nil(t, err)
	require.NotNil(t, caps.QueryLimits)
	require.NotNil(t, caps.IngestionLimits)

	limits := caps.QueryLimits

	minStep := metricDuration(limits.MinimumStep)
	maxRange := metricDuration(limits.MaximumTimeRange)
	futureSkew := metricDuration(caps.IngestionLimits.MaximumFutureSkew)

	require.True(t, minStep > time.Millisecond)
	require.True(t, maxRange > time.Hour)
	require.True(t, limits.MaximumGroupByAttributes > 0)
	require.True(t, limits.MaximumFilters > 0)

	catalog, err := h.MetricsC().ListMetricCatalog(ctx,
		&vmetricsv1.ListMetricCatalogRequest{})
	require.Nil(t, err)
	require.NotEmpty(t, catalog.Items)

	selector := catalog.Items[0].Metric
	require.NotNil(t, selector)

	refuses := func(t *testing.T, req *vmetricsv1.QueryMetricsRequest) {
		t.Helper()

		_, err := h.MetricsC().QueryMetrics(ctx, req)
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err),
			"the metricstore answered %v, want InvalidArgument", err)
	}

	t.Run("AStepBelowTheMinimumIsRefused", func(t *testing.T) {
		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric:    selector,
			TimeRange: metricTimeRange(time.Now().Add(-time.Hour), time.Now()),
			Step:      &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 1}},
			Operation: gaugeLast(),
		})
	})

	t.Run("ARangeBeyondTheRetentionIsRefused", func(t *testing.T) {
		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric: selector,
			TimeRange: metricTimeRange(
				time.Now().Add(-maxRange-time.Hour), time.Now()),
			Step:      metricStep(),
			Operation: gaugeLast(),
		})
	})

	t.Run("AFutureRangeIsRefused", func(t *testing.T) {
		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric: selector,
			TimeRange: metricTimeRange(
				time.Now(), time.Now().Add(futureSkew+time.Hour)),
			Step:      metricStep(),
			Operation: gaugeLast(),
		})
	})

	t.Run("AnInvertedRangeIsRefused", func(t *testing.T) {
		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric:    selector,
			TimeRange: metricTimeRange(time.Now(), time.Now().Add(-time.Hour)),
			Step:      metricStep(),
			Operation: gaugeLast(),
		})
	})

	t.Run("ARawCounterRefusesAStep", func(t *testing.T) {
		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric:    selector,
			TimeRange: metricTimeRange(time.Now().Add(-time.Hour), time.Now()),
			Step:      metricStep(),
			Operation: counterRaw(),
		})
	})

	t.Run("ABucketedQueryRequiresAStep", func(t *testing.T) {
		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric:    selector,
			TimeRange: metricTimeRange(time.Now().Add(-time.Hour), time.Now()),
			Operation: gaugeLast(),
		})
	})

	t.Run("AMonthlyStepIsRefused", func(t *testing.T) {
		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric:    selector,
			TimeRange: metricTimeRange(time.Now().Add(-time.Hour), time.Now()),
			Step:      &metav1.Duration{Type: &metav1.Duration_Months{Months: 1}},
			Operation: gaugeLast(),
		})
	})

	t.Run("TooManyGroupByAttributesAreRefused", func(t *testing.T) {
		groupBy := make([]string, 0, limits.MaximumGroupByAttributes+1)
		for i := range limits.MaximumGroupByAttributes + 1 {
			groupBy = append(groupBy, fmt.Sprintf("octelium.attribute.%d", i))
		}

		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric:    selector,
			TimeRange: metricTimeRange(time.Now().Add(-time.Hour), time.Now()),
			Step:      metricStep(),
			Operation: gaugeLast(),
			GroupBy:   groupBy,
		})
	})

	t.Run("TooManyFiltersAreRefused", func(t *testing.T) {
		filters := make([]*vmetricsv1.AttributeFilter, 0, limits.MaximumFilters+1)
		for i := range limits.MaximumFilters + 1 {
			filters = append(filters, &vmetricsv1.AttributeFilter{
				Key:      fmt.Sprintf("octelium.attribute.%d", i),
				Operator: vmetricsv1.AttributeFilter_EQ,
				Value: &vmetricsv1.AttributeValue{
					Value: &vmetricsv1.AttributeValue_StringValue{StringValue: "e2e"},
				},
			})
		}

		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric:    selector,
			TimeRange: metricTimeRange(time.Now().Add(-time.Hour), time.Now()),
			Step:      metricStep(),
			Operation: gaugeLast(),
			Filters:   filters,
		})
	})

	t.Run("AnEmptySelectorIsRefused", func(t *testing.T) {
		refuses(t, &vmetricsv1.QueryMetricsRequest{
			TimeRange: metricTimeRange(time.Now().Add(-time.Hour), time.Now()),
			Step:      metricStep(),
			Operation: gaugeLast(),
		})
	})

	t.Run("TooManyBucketsAreRefused", func(t *testing.T) {
		span := minStep * time.Duration(limits.MaximumPointsPerSeries+10)
		if span > maxRange {
			t.Skip("the advertised retention cannot hold that many buckets")
		}

		refuses(t, &vmetricsv1.QueryMetricsRequest{
			Metric:    selector,
			TimeRange: metricTimeRange(time.Now().Add(-span), time.Now()),
			Step:      limits.MinimumStep,
			Operation: gaugeLast(),
		})
	})

	t.Run("AQueryWithinTheLimitsIsAccepted", func(t *testing.T) {
		h.Eventually(t, "a catalog metric to answer within the advertised limits",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				return queryAnyCatalogMetric(ctx, h)
			})
	})
}
