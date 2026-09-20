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
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

func TestInitDBSetsTheCurrentSchemaVersion(t *testing.T) {
	s := newTestStorageServer(t)

	version, err := s.getSchemaVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, metricStoreSchemaVersion, version)
}

func TestInitDBDoesNotResetAFreshDatabase(t *testing.T) {
	database := filepath.Join(t.TempDir(), "metricstore.db")

	s := &Server{dbConfig: &metricStoreDBConfig{dsn: database, database: database}}
	db, err := openMetricStoreDB(s.dbConfig)
	require.NoError(t, err)
	s.setDatabase(db)
	t.Cleanup(func() { _ = s.database().Close() })

	require.NoError(t, s.initDB(context.Background()))

	assert.Same(t, db, s.database())
}

func TestInitDBKeepsTheStorageOfAnUpToDateDatabase(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	seedTestMetricMetadata(t, s, "descriptor-1", "series-1", time.Now().UTC())
	seedTestMetricPoints(t, s, "series-1", time.Minute)

	for i := 0; i < 3; i++ {
		require.NoError(t, s.initDB(ctx))
	}

	assert.Equal(t, []string{"series-1-1m0s"}, testMetricPointIDs(t, s))
	assert.Equal(t, int64(1), testTableCount(t, s, "metric_series"))
}

func TestInitDBResetsTheStorageOfAnOutdatedDatabase(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	seedTestMetricMetadata(t, s, "descriptor-1", "series-1", time.Now().UTC())
	seedTestMetricPoints(t, s, "series-1", time.Minute)
	testSetSchemaVersion(t, s, metricStoreSchemaVersion-1)

	previous := s.database()
	require.NoError(t, s.initDB(ctx))

	assert.NotSame(t, previous, s.database())
	assert.Empty(t, testMetricPointIDs(t, s))
	assert.Zero(t, testTableCount(t, s, "metric_series"))
	assert.Zero(t, testTableCount(t, s, "metric_descriptors"))

	version, err := s.getSchemaVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, metricStoreSchemaVersion, version)
}

func TestInitDBResetsTheStorageOfANewerDatabase(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	seedTestMetricPoints(t, s, "series-1", time.Minute)
	testSetSchemaVersion(t, s, metricStoreSchemaVersion+1)

	require.NoError(t, s.initDB(ctx))

	assert.Empty(t, testMetricPointIDs(t, s))

	version, err := s.getSchemaVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, metricStoreSchemaVersion, version)
}

func TestInitDBResetsTheStorageOfALegacyDatabase(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	_, err := s.database().ExecContext(ctx, `DROP TABLE metricstore_schema`)
	require.NoError(t, err)
	_, err = s.database().ExecContext(ctx, `CREATE TABLE metrics (id VARCHAR PRIMARY KEY)`)
	require.NoError(t, err)
	_, err = s.database().ExecContext(ctx, `INSERT INTO metrics VALUES ('legacy')`)
	require.NoError(t, err)

	require.NoError(t, s.initDB(ctx))

	var count int
	require.NoError(t, s.database().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'main' AND table_name = 'metrics'
`).Scan(&count))
	assert.Zero(t, count)

	version, err := s.getSchemaVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, metricStoreSchemaVersion, version)
}

func TestInitDBRemovesTheDatabaseFilesOnAReset(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	_, err := s.database().ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO metric_number_points
SELECT md5(i::VARCHAR), i, i, NULL, 'series-1', i, NULL FROM range(%d) tbl(i)`, 200000))
	require.NoError(t, err)
	require.NoError(t, s.checkpointStorage(ctx))
	testSetSchemaVersion(t, s, metricStoreSchemaVersion-1)

	before := testDatabaseSize(t, s)
	require.NoError(t, s.initDB(ctx))

	assert.Less(t, testDatabaseSize(t, s), before)
	assert.Zero(t, testTableCount(t, s, "metric_number_points"))
}

func testDatabaseSize(t *testing.T, s *Server) int64 {
	t.Helper()

	info, err := os.Stat(s.dbConfig.database)
	require.NoError(t, err)

	return info.Size()
}

func TestRemoveMetricStoreDatabaseRemovesEveryDatabaseFile(t *testing.T) {
	database := filepath.Join(t.TempDir(), "metricstore.db")

	require.NoError(t, os.WriteFile(database, []byte("db"), 0o600))
	require.NoError(t, os.WriteFile(database+".wal", []byte("wal"), 0o600))
	require.NoError(t, os.MkdirAll(database+".tmp", 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(database+".tmp", "spill"), []byte("tmp"), 0o600))

	require.NoError(t, removeMetricStoreDatabase(database))

	assert.NoFileExists(t, database)
	assert.NoFileExists(t, database+".wal")
	assert.NoDirExists(t, database+".tmp")
}

func TestRemoveMetricStoreDatabaseWithoutAPath(t *testing.T) {
	assert.Nil(t, removeMetricStoreDatabase(""))
	assert.Nil(t, removeMetricStoreDatabase(filepath.Join(t.TempDir(), "does-not-exist.db")))
}

func TestReopenDatabaseReplacesTheDatabaseHandle(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	seedTestMetricPoints(t, s, "series-1", time.Minute)
	previous := s.database()

	s.reopenDatabase(ctx)

	assert.NotSame(t, previous, s.database())
	assert.NotNil(t, previous.PingContext(ctx))
	assert.Nil(t, s.database().PingContext(ctx))
	assert.Equal(t, []string{"series-1-1m0s"}, testMetricPointIDs(t, s))

	version, err := s.getSchemaVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, metricStoreSchemaVersion, version)
}

func TestRecoverDatabaseKeepsAHealthyDatabase(t *testing.T) {
	s := newTestStorageServer(t)
	previous := s.database()

	s.recoverDatabase(context.Background())

	assert.Same(t, previous, s.database())
	assert.Nil(t, s.database().PingContext(context.Background()))
}

func TestRequestDatabaseRecoverySignalsOnlyInvalidatedErrors(t *testing.T) {
	s := newTestStorageServer(t)

	s.requestDatabaseRecovery(nil)
	s.requestDatabaseRecovery(errors.New("Out of Memory Error: could not allocate block of size 256.0 KiB"))
	assert.Empty(t, s.dbRecoverCh)

	invalidated := errors.New(
		"FATAL Error: Failed: database has been invalidated because of a previous fatal error")
	s.requestDatabaseRecovery(invalidated)
	s.requestDatabaseRecovery(invalidated)
	assert.Len(t, s.dbRecoverCh, 1)
}

func TestRunDatabaseRecoveryLoopExitsOnContextCancellation(t *testing.T) {
	s := newTestStorageServer(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runDatabaseRecoveryLoop(ctx)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the metricstore database recovery loop did not exit")
	}
}

func TestCreateMetricStoreTablesIsIdempotent(t *testing.T) {
	db := newTestDuckDB(t)

	for i := 0; i < 2; i++ {
		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		require.NoError(t, createMetricStoreTables(context.Background(), tx))
		require.NoError(t, tx.Commit())
	}

	var count int
	require.NoError(t, db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = ?`,
		"metric_series_descriptor_id").Scan(&count))
	assert.Equal(t, 1, count)

	for _, table := range metricPointTables {
		require.NoError(t, db.QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM duckdb_indexes() WHERE table_name = ?`, table).Scan(&count))
		assert.Zero(t, count, table)
	}
}
