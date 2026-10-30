// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"compress/gzip"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestPointReplacementDeduplicatesAcrossTheBatchTimeRange(t *testing.T) {
	db := newTestDuckDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	numberPoint := func(pointID string, at time.Time) numberPointRecord {
		value := int64(1)
		return numberPointRecord{
			pointID: pointID, timestamp: at, ingestedAt: now,
			seriesID: "series-1", intValue: &value,
		}
	}
	histogramPoint := func(pointID string, at time.Time) histogramPointRecord {
		return histogramPointRecord{
			pointID: pointID, timestamp: at, ingestedAt: now,
			seriesID: "series-1", count: 1, bucketCounts: []uint64{1},
		}
	}
	exponentialPoint := func(pointID string, at time.Time) exponentialHistogramPointRecord {
		return exponentialHistogramPointRecord{
			pointID: pointID, timestamp: at, ingestedAt: now,
			seriesID: "series-1", count: 1, positiveCounts: []uint64{1}, negativeCounts: []uint64{},
		}
	}

	stored := []string{"point-1", "point-2", "point-3"}
	storedAt := []time.Time{now.Add(-2 * time.Hour), now.Add(-time.Hour), now}
	retried := []string{"point-4", "point-1", "point-2", "point-3", "point-5"}
	retriedAt := []time.Time{now.Add(-3 * time.Hour), storedAt[0], storedAt[1], storedAt[2], now.Add(time.Minute)}

	var numberPoints, retriedNumberPoints []numberPointRecord
	var histogramPoints, retriedHistogramPoints []histogramPointRecord
	var exponentialPoints, retriedExponentialPoints []exponentialHistogramPointRecord
	for i, pointID := range stored {
		numberPoints = append(numberPoints, numberPoint(pointID, storedAt[i]))
		histogramPoints = append(histogramPoints, histogramPoint(pointID, storedAt[i]))
		exponentialPoints = append(exponentialPoints, exponentialPoint(pointID, storedAt[i]))
	}
	for i, pointID := range retried {
		retriedNumberPoints = append(retriedNumberPoints, numberPoint(pointID, retriedAt[i]))
		retriedHistogramPoints = append(retriedHistogramPoints, histogramPoint(pointID, retriedAt[i]))
		retriedExponentialPoints = append(retriedExponentialPoints, exponentialPoint(pointID, retriedAt[i]))
	}

	require.NoError(t, replaceNumberPoints(ctx, conn, numberPoints))
	require.NoError(t, replaceNumberPoints(ctx, conn, retriedNumberPoints))
	require.NoError(t, replaceNumberPoints(ctx, conn, numberPoints[1:2]))
	require.NoError(t, replaceHistogramPoints(ctx, conn, histogramPoints))
	require.NoError(t, replaceHistogramPoints(ctx, conn, retriedHistogramPoints))
	require.NoError(t, replaceHistogramPoints(ctx, conn, histogramPoints[1:2]))
	require.NoError(t, replaceExponentialHistogramPoints(ctx, conn, exponentialPoints))
	require.NoError(t, replaceExponentialHistogramPoints(ctx, conn, retriedExponentialPoints))
	require.NoError(t, replaceExponentialHistogramPoints(ctx, conn, exponentialPoints[1:2]))

	for _, table := range metricPointTables {
		var count, distinct int
		require.NoError(t, conn.QueryRowContext(ctx,
			`SELECT COUNT(*), COUNT(DISTINCT point_id) FROM `+table).Scan(&count, &distinct))
		assert.Equal(t, 5, count, table)
		assert.Equal(t, 5, distinct, table)
	}
}

func TestEnsureRawRowLimitCountsTheSelectedSeriesPointsInRange(t *testing.T) {
	db := newTestDuckDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	value := int64(1)
	require.NoError(t, replaceNumberPoints(ctx, conn, []numberPointRecord{
		{pointID: "point-1", timestamp: now.Add(-30 * time.Second), ingestedAt: now, seriesID: "series-1", intValue: &value},
		{pointID: "point-2", timestamp: now.Add(-20 * time.Second), ingestedAt: now, seriesID: "series-1", intValue: &value},
		{pointID: "point-3", timestamp: now.Add(-10 * time.Second), ingestedAt: now, seriesID: "series-1", intValue: &value},
		{pointID: "point-4", timestamp: now.Add(-2 * time.Hour), ingestedAt: now, seriesID: "series-1", intValue: &value},
		{pointID: "point-5", timestamp: now.Add(-10 * time.Second), ingestedAt: now, seriesID: "series-2", intValue: &value},
	}))

	mapSeries(t, conn, "series-1", "output-1")

	query := &querySpec{
		from:     now.Add(-time.Minute),
		to:       now.Add(time.Minute),
		step:     time.Minute,
		snapshot: now.Add(time.Minute),
	}

	require.NoError(t, ensureRawRowLimit(ctx, conn, "metric_number_points", query, false, 3))
	err = ensureRawRowLimit(ctx, conn, "metric_number_points", query, false, 2)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
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

func TestInitDBRemovesOnlyLegacyMetricStoreStorage(t *testing.T) {
	metricsTable := `CREATE TABLE metrics (
	timestamp TIMESTAMP, name VARCHAR, resource JSON, scope JSON, attributes JSON,
	component_type VARCHAR, component_namespace VARCHAR, histogram_bucket_counts JSON
)`
	for _, tc := range []struct {
		name       string
		statements []string
		remove     bool
	}{
		{"Metricstore", []string{metricsTable}, true},
		{"Logstore", []string{`CREATE TABLE access_logs (rsc JSON)`}, false},
		{"Rscstore", []string{`CREATE TABLE resources (uid VARCHAR)`}, false},
		{"Mixed", []string{metricsTable, `CREATE TABLE resources (uid VARCHAR)`}, false},
		{"Unknown", []string{`CREATE TABLE metrics (timestamp TIMESTAMP, name VARCHAR)`}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestStorageServer(t)
			seedTestMetricPoints(t, s, "series-1", time.Minute)
			dir := filepath.Dir(s.dbConfig.database)
			legacy := filepath.Join(dir, "store.db")
			db, err := sql.Open("duckdb", legacy+"?storage_compatibility_version=v1.5.5")
			require.NoError(t, err)
			defer db.Close()
			for _, statement := range tc.statements {
				_, err := db.Exec(statement)
				require.NoError(t, err)
			}
			require.NoError(t, db.Close())

			legacyTempDir := filepath.Join(dir, "metricstore-tmp")
			for _, tempDir := range []string{legacyTempDir, legacy + ".tmp"} {
				require.NoError(t, os.MkdirAll(tempDir, 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(tempDir, "spill"), []byte("tmp"), 0o600))
			}

			require.NoError(t, s.initDB(context.Background()))
			assert.NoDirExists(t, legacyTempDir)
			if tc.remove {
				assert.NoFileExists(t, legacy)
				assert.NoDirExists(t, legacy+".tmp")
			} else {
				assert.FileExists(t, legacy)
				assert.FileExists(t, filepath.Join(legacy+".tmp", "spill"))
			}
			assert.Equal(t, []string{"series-1-1m0s"}, testMetricPointIDs(t, s))
		})
	}
}

func TestRemoveLegacyMetricStoreStorageKeepsTheConfiguredDatabase(t *testing.T) {
	database := filepath.Join(t.TempDir(), "store.db")
	require.NoError(t, os.WriteFile(database, []byte("configured database"), 0o600))
	require.NoError(t, removeLegacyMetricStoreStorage(context.Background(), database))
	assert.FileExists(t, database)
	assert.NoError(t, removeLegacyMetricStoreStorage(context.Background(), ""))
}

func TestInitDBKeepsUnreadableLegacyStorage(t *testing.T) {
	s := newTestStorageServer(t)
	legacy := filepath.Join(filepath.Dir(s.dbConfig.database), "store.db")
	data := []byte("unrecognized database")
	require.NoError(t, os.WriteFile(legacy, data, 0o600))

	require.NoError(t, s.initDB(context.Background()))
	stored, err := os.ReadFile(legacy)
	require.NoError(t, err)
	assert.Equal(t, data, stored)
	require.NoError(t, s.database().PingContext(context.Background()))
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

func newTestUpgradeMetrics(at time.Time) pmetric.Metrics {
	metrics, sm := newVigilResourceMetrics()
	start := at.Add(-time.Hour)
	appendTestCounter(sm, "upgrade.counter", start, at, 42, map[string]any{
		"state": "ok", "authorized": true, "attempt": int64(2),
	})
	appendTestHistogram(sm, "upgrade.histogram", "us", start, at)

	metric := sm.Metrics().AppendEmpty()
	metric.SetName("upgrade.gauge")
	point := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	point.SetTimestamp(pcommon.NewTimestampFromTime(at))
	point.SetDoubleValue(1.25)
	point.Attributes().PutDouble("ratio", 0.5)

	metric = sm.Metrics().AppendEmpty()
	metric.SetName("upgrade.exponential_histogram")
	histogram := metric.SetEmptyExponentialHistogram()
	histogram.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
	hp := histogram.DataPoints().AppendEmpty()
	hp.SetStartTimestamp(pcommon.NewTimestampFromTime(start))
	hp.SetTimestamp(pcommon.NewTimestampFromTime(at))
	hp.SetScale(2)
	hp.SetCount(10)
	hp.SetSum(25)
	hp.SetZeroCount(1)
	hp.Positive().SetOffset(-1)
	hp.Positive().BucketCounts().FromRaw([]uint64{4, 3})
	hp.Negative().SetOffset(1)
	hp.Negative().BucketCounts().FromRaw([]uint64{2})

	return metrics
}

func TestMetricStoreRecreatesADuckDBV155Database(t *testing.T) {
	testMetricStoreUpgrade(t, false)
}

func TestMetricStoreRecreatesADuckDBV155DatabaseWithWAL(t *testing.T) {
	testMetricStoreUpgrade(t, true)
}

func TestMetricStoreRollsBackAnAppenderFlushFailure(t *testing.T) {
	srv := newTestSrvMetric(t)
	ctx := context.Background()
	_, err := srv.s.database().ExecContext(ctx, `CREATE OR REPLACE TABLE metric_number_staging (
	point_id VARCHAR NOT NULL,
	timestamp BIGINT NOT NULL,
	ingested_at BIGINT NOT NULL,
	start_timestamp BIGINT,
	series_id VARCHAR NOT NULL,
	number_int BIGINT CHECK (number_int <> 42),
	number_double DOUBLE
)`)
	require.NoError(t, err)

	metrics := newTestUpgradeMetrics(time.Now().UTC())
	err = storeTestMetricsErr(t, srv, metrics)
	require.ErrorContains(t, err, "CHECK constraint failed")
	for _, table := range []string{
		"metric_descriptors", "metric_series", "metric_series_attributes", "metric_attribute_keys",
		"metric_number_staging", "metric_number_points", "metric_histogram_points", "metric_exponential_histogram_points",
	} {
		assert.Zero(t, testTableCount(t, srv.s, table), table)
	}

	metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Sum().DataPoints().At(0).SetIntValue(43)
	require.NoError(t, storeTestMetricsErr(t, srv, metrics))
	assert.Equal(t, int64(4), testTableCount(t, srv.s, "metric_descriptors"))
	assert.Equal(t, int64(2), testTableCount(t, srv.s, "metric_number_points"))
	assert.Equal(t, int64(1), testTableCount(t, srv.s, "metric_histogram_points"))
	assert.Equal(t, int64(1), testTableCount(t, srv.s, "metric_exponential_histogram_points"))
}

func restoreTestDatabaseFile(t *testing.T, database, fixture string) {
	t.Helper()

	file, err := os.Open(fixture)
	require.NoError(t, err)
	defer file.Close()
	reader, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer reader.Close()
	dst, err := os.OpenFile(database, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = io.Copy(dst, reader)
	require.NoError(t, dst.Close())
	require.NoError(t, err)
}

func testMetricStoreUpgrade(t *testing.T, recoverWAL bool) {
	t.Helper()

	t.Setenv("OCTELIUM_METRICSTORE_DUCKDB_PATH", t.TempDir())
	dbConfig, err := getMetricStoreDBConfig()
	require.NoError(t, err)
	if dbConfig.cleanupFn != nil {
		defer dbConfig.cleanupFn()
	}

	restoreTestDatabaseFile(t, dbConfig.database, "testdata/duckdb-v1.5.5.db.gz")
	numberPoints := int64(2)
	if recoverWAL {
		restoreTestDatabaseFile(t, dbConfig.database+".wal", "testdata/duckdb-v1.5.5.db.wal.gz")
		numberPoints++
	}

	s := &Server{dbConfig: dbConfig}
	t.Cleanup(func() {
		if db := s.database(); db != nil {
			_ = db.Close()
		}
	})
	ctx := context.Background()
	db, err := openMetricStoreDB(dbConfig)
	require.NoError(t, err)
	s.setDatabase(db)
	version, err := s.getSchemaVersion(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, version)
	require.Equal(t, numberPoints, testTableCount(t, s, "metric_number_points"))
	before := testDatabaseSize(t, s)

	require.NoError(t, s.initDB(ctx))
	assert.NotSame(t, db, s.database())
	assert.Less(t, testDatabaseSize(t, s), before)
	for _, table := range []string{
		"metric_descriptors", "metric_series", "metric_series_attributes", "metric_attribute_keys",
		"metric_number_points", "metric_histogram_points", "metric_exponential_histogram_points",
	} {
		assert.Zero(t, testTableCount(t, s, table), table)
	}
	version, err = s.getSchemaVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, metricStoreSchemaVersion, version)

	at := time.Date(2026, time.October, 1, 12, 1, 0, 0, time.UTC)
	srv := &srvMetric{s: s}
	require.NoError(t, storeTestMetricsErr(t, srv, newTestUpgradeMetrics(at)))
	require.NoError(t, storeTestMetricsErr(t, srv, newTestUpgradeMetrics(at)))
	require.NoError(t, s.checkpointStorage(ctx))
	require.NoError(t, s.database().Close())

	for i := 0; i < 2; i++ {
		db, err := openMetricStoreDB(dbConfig)
		require.NoError(t, err)
		s.setDatabase(db)
		require.NoError(t, s.initDB(ctx))
		assert.Same(t, db, s.database())
		var storageVersion string
		require.NoError(t, db.QueryRow(`SELECT tags['storage_version'] FROM duckdb_databases() WHERE path = ?`, dbConfig.database).Scan(&storageVersion))
		assert.Equal(t, "v2.0.0+", storageVersion)
		assert.Equal(t, int64(4), testTableCount(t, s, "metric_descriptors"))
		assert.Equal(t, int64(4), testTableCount(t, s, "metric_series"))
		assert.Equal(t, int64(2), testTableCount(t, s, "metric_number_points"))
		assert.Equal(t, int64(1), testTableCount(t, s, "metric_histogram_points"))
		assert.Equal(t, int64(1), testTableCount(t, s, "metric_exponential_histogram_points"))
		if recoverWAL {
			var count int
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM metric_number_points WHERE point_id = 'wal-point'`).Scan(&count))
			assert.Zero(t, count)
		}

		var intValue int64
		var doubleValue float64
		require.NoError(t, db.QueryRow(`SELECT MIN(number_int), MIN(number_double) FROM metric_number_points`).Scan(&intValue, &doubleValue))
		assert.Equal(t, int64(42), intValue)
		assert.Equal(t, 1.25, doubleValue)

		var bounds, buckets, positive, negative string
		require.NoError(t, db.QueryRow(`SELECT CAST(explicit_bounds AS VARCHAR) FROM metric_descriptors WHERE name = 'upgrade.histogram'`).Scan(&bounds))
		require.NoError(t, db.QueryRow(`SELECT CAST(bucket_counts AS VARCHAR) FROM metric_histogram_points LIMIT 1`).Scan(&buckets))
		require.NoError(t, db.QueryRow(`SELECT CAST(positive_counts AS VARCHAR), CAST(negative_counts AS VARCHAR) FROM metric_exponential_histogram_points LIMIT 1`).Scan(&positive, &negative))
		assert.JSONEq(t, `[10,100]`, bounds)
		assert.JSONEq(t, `[1,1,1]`, buckets)
		assert.JSONEq(t, `[4,3]`, positive)
		assert.JSONEq(t, `[2]`, negative)

		var attributes int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM metric_series_attributes WHERE
	(key = 'state' AND value_string = 'ok') OR
	(key = 'authorized' AND value_bool = true) OR
	(key = 'attempt' AND value_int = 2) OR
	(key = 'ratio' AND value_double = 0.5)`).Scan(&attributes))
		assert.Equal(t, 4, attributes)

		require.NoError(t, db.Close())
	}
}
