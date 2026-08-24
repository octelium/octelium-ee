// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func testMockCounterQuery(from, to time.Time, step *metav1.Duration) *vmetricsv1.QueryMetricsRequest {
	return &vmetricsv1.QueryMetricsRequest{
		Metric: &vmetricsv1.MetricSelector{
			Selector: &vmetricsv1.MetricSelector_Name{Name: "req.total"},
		},
		TimeRange: &vmetricsv1.TimeRange{
			From: pbutils.Timestamp(from),
			To:   pbutils.Timestamp(to),
		},
		Step: step,
		Operation: &vmetricsv1.QueryOperation{
			Type: &vmetricsv1.QueryOperation_Counter{Counter: &vmetricsv1.CounterOperation{
				Function: vmetricsv1.CounterOperation_RATE,
			}},
		},
		SeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
		LimitBehavior:     vmetricsv1.QueryMetricsRequest_TRUNCATE,
	}
}

func TestMockAlignTimeDown(t *testing.T) {
	step := time.Minute
	aligned := time.Date(2026, time.August, 22, 10, 30, 0, 0, time.UTC)

	assert.Equal(t, aligned, mockAlignTimeDown(aligned, step))
	assert.Equal(t, aligned, mockAlignTimeDown(aligned.Add(time.Second), step))
	assert.Equal(t, aligned, mockAlignTimeDown(aligned.Add(59*time.Second), step))
	assert.Equal(t, aligned.Add(step), mockAlignTimeDown(aligned.Add(step), step))

	value := aligned.Add(time.Second)
	assert.Equal(t, value, mockAlignTimeDown(value, 0))
}

func TestMockPointCountDropsTheIncompleteTrailingBucket(t *testing.T) {
	from := time.Date(2026, time.August, 22, 10, 0, 0, 0, time.UTC)

	assert.Equal(t, 5, mockPointCount(from, from.Add(5*time.Minute), time.Minute, false))
	assert.Equal(t, 5, mockPointCount(from, from.Add(5*time.Minute+25*time.Second), time.Minute, false))
	assert.Equal(t, 1, mockPointCount(from, from.Add(30*time.Second), time.Minute, false))
}

func TestMockQueryAlignsTheBucketedWindow(t *testing.T) {
	s := &tstMetricsService{}
	minute := time.Now().UTC().Truncate(time.Minute)
	step := &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 1}}

	res, err := s.QueryMetrics(context.Background(),
		testMockCounterQuery(minute.Add(-10*time.Minute).Add(20*time.Second),
			minute.Add(-time.Minute).Add(40*time.Second), step))
	require.NoError(t, err)
	require.NotEmpty(t, res.Series)

	points := res.Series[0].GetNumber().Points
	require.Len(t, points, 9)
	for _, point := range points {
		at := point.Timestamp.AsTime()
		assert.Equal(t, at, at.Truncate(time.Minute))
		assert.False(t, at.After(minute.Add(-time.Minute)))
	}
}

func TestMockQueryRejectsARangeShorterThanASingleStep(t *testing.T) {
	s := &tstMetricsService{}
	minute := time.Now().UTC().Truncate(time.Minute)
	step := &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 1}}

	_, err := s.QueryMetrics(context.Background(),
		testMockCounterQuery(minute.Add(-time.Minute).Add(20*time.Second),
			minute.Add(-time.Minute).Add(50*time.Second), step))
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
