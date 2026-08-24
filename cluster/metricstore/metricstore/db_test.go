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
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDuckDB(t *testing.T) *sql.DB {
	db, err := sql.Open("duckdb", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, createMetricStoreTables(context.Background(), tx))
	require.NoError(t, tx.Commit())

	return db
}

func mapSeries(t *testing.T, conn *sql.Conn, seriesID, outputID string) {
	t.Helper()
	_, err := conn.ExecContext(context.Background(), `
CREATE OR REPLACE TEMP TABLE selected_metric_series (
	series_id VARCHAR,
	output_id VARCHAR
)`)
	require.NoError(t, err)

	err = conn.Raw(func(raw any) error {
		driverConn, ok := raw.(driver.Conn)
		if !ok {
			return fmt.Errorf("unexpected DuckDB driver connection type: %T", raw)
		}
		appender, err := duckdb.NewAppenderFromConn(driverConn, "", "selected_metric_series")
		if err != nil {
			return err
		}
		if err := appender.AppendRow(seriesID, outputID); err != nil {
			_ = appender.Close()
			return err
		}
		return appender.Close()
	})
	require.NoError(t, err)
}

func TestHistogramIngestQueryRoundTrip(t *testing.T) {
	db := newTestDuckDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	point := histogramPointRecord{
		pointID:      "point-1",
		timestamp:    now,
		ingestedAt:   now,
		seriesID:     "series-1",
		count:        6,
		hasSum:       true,
		sum:          new(float64(42)),
		bucketCounts: []uint64{1, 2, 3},
	}

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, replaceHistogramPoints(ctx, conn, []histogramPointRecord{point}))

	mapSeries(t, conn, "series-1", "output-1")

	query := &querySpec{
		from:     now.Add(-time.Minute),
		to:       now.Add(time.Minute),
		step:     time.Minute,
		snapshot: now.Add(time.Minute),
	}

	rows, err := loadExplicitHistogramQueryRows(ctx, conn, query, vmetricsv1.MetricDescriptor_DELTA)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())
	raw, err := scanExplicitHistogramRawPoint(rows)
	require.NoError(t, err)
	assert.Equal(t, []uint64{1, 2, 3}, raw.bucketCounts)
	assert.Equal(t, uint64(6), raw.count)
}

func TestNumberPointReplacementIsIdempotent(t *testing.T) {
	db := newTestDuckDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	value := int64(1)
	point := numberPointRecord{
		pointID: "point-1", timestamp: now, ingestedAt: now,
		seriesID: "series-1", intValue: &value,
	}

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, replaceNumberPoints(ctx, conn, []numberPointRecord{point}))
	point.ingestedAt = now.Add(time.Second)
	updatedValue := int64(2)
	point.intValue = &updatedValue
	require.NoError(t, replaceNumberPoints(ctx, conn, []numberPointRecord{point}))

	var count int
	var storedValue int64
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM metric_number_points WHERE point_id = ?`, point.pointID).Scan(&count))
	require.NoError(t, conn.QueryRowContext(ctx,
		`SELECT number_int FROM metric_number_points WHERE point_id = ?`, point.pointID).Scan(&storedValue))
	assert.Equal(t, 1, count)
	assert.Equal(t, value, storedValue)
}

func TestExponentialHistogramIngestQueryRoundTrip(t *testing.T) {
	db := newTestDuckDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	point := exponentialHistogramPointRecord{
		pointID:        "point-1",
		timestamp:      now,
		ingestedAt:     now,
		seriesID:       "series-1",
		count:          10,
		scale:          2,
		zeroThreshold:  0,
		positiveOffset: 0,
		positiveCounts: []uint64{4, 5},
		negativeOffset: 0,
		negativeCounts: []uint64{1},
	}

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, replaceExponentialHistogramPoints(ctx, conn, []exponentialHistogramPointRecord{point}))

	mapSeries(t, conn, "series-1", "output-1")

	query := &querySpec{
		from:     now.Add(-time.Minute),
		to:       now.Add(time.Minute),
		step:     time.Minute,
		snapshot: now.Add(time.Minute),
	}

	rows, err := loadExponentialHistogramQueryRows(ctx, conn, query, vmetricsv1.MetricDescriptor_DELTA)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())
	raw, err := scanExponentialHistogramRawPoint(rows)
	require.NoError(t, err)
	assert.Equal(t, map[int32]uint64{0: 4, 1: 5}, raw.positive)
	assert.Equal(t, map[int32]uint64{0: 1}, raw.negative)
}

func TestCreateMetricStoreTablesIsIdempotent(t *testing.T) {
	db := newTestDuckDB(t)

	for i := 0; i < 2; i++ {
		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		require.NoError(t, createMetricStoreTables(context.Background(), tx))
		require.NoError(t, tx.Commit())
	}

	for _, name := range []string{
		"metric_number_points_series_timestamp",
		"metric_histogram_points_series_timestamp",
		"metric_exponential_histogram_points_series_timestamp",
		"metric_number_points_point_id",
		"metric_series_descriptor_id",
	} {
		var count int
		require.NoError(t, db.QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = ?`, name).Scan(&count))
		assert.Equal(t, 1, count, name)
	}
}
