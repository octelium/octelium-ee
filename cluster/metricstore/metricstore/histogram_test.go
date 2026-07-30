// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeltaExplicitHistogramPoint(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(-time.Hour)
	previous := explicitHistogramRawPoint{
		timestamp:      now.Add(-time.Minute),
		startTimestamp: &start,
		count:          10,
		hasSum:         true,
		sum:            50,
		bucketCounts:   []uint64{3, 4, 3},
	}
	current := explicitHistogramRawPoint{
		timestamp:      now,
		startTimestamp: &start,
		count:          20,
		hasSum:         true,
		sum:            120,
		bucketCounts:   []uint64{5, 8, 7},
	}

	delta := deltaExplicitHistogramPoint(previous, current)
	assert.Equal(t, uint64(10), delta.count)
	assert.Equal(t, float64(70), delta.sum)
	assert.Equal(t, []uint64{2, 4, 4}, delta.bucketCounts)
	assert.Nil(t, delta.min)
	assert.Nil(t, delta.max)
}

func TestDeltaExplicitHistogramResetIsAtomic(t *testing.T) {
	previous := explicitHistogramRawPoint{count: 20, bucketCounts: []uint64{5, 10, 5}}
	current := explicitHistogramRawPoint{count: 4, bucketCounts: []uint64{1, 2, 1}}

	delta := deltaExplicitHistogramPoint(previous, current)
	assert.Equal(t, current.count, delta.count)
	assert.Equal(t, current.bucketCounts, delta.bucketCounts)
}

func TestExplicitHistogramQuantile(t *testing.T) {
	value, ok := explicitHistogramQuantile(0.5, []float64{10, 100}, []uint64{5, 3, 2}, 10)
	assert.True(t, ok)
	assert.Equal(t, float64(10), value)
}

func TestDeltaExponentialHistogramResetOnScaleChange(t *testing.T) {
	previous := exponentialHistogramRawPoint{
		count:    10,
		scale:    2,
		positive: map[int32]uint64{0: 10},
		negative: map[int32]uint64{},
	}
	current := exponentialHistogramRawPoint{
		count:    4,
		scale:    1,
		positive: map[int32]uint64{0: 4},
		negative: map[int32]uint64{},
	}

	delta := deltaExponentialHistogramPoint(previous, current)
	assert.Equal(t, current.count, delta.count)
	assert.Equal(t, current.scale, delta.scale)
	assert.Equal(t, current.positive, delta.positive)
}
