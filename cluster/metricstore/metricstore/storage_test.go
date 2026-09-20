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
	"fmt"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorageServer(t *testing.T) *Server {
	dir := t.TempDir()
	database := filepath.Join(dir, "metricstore.db")

	ret := &Server{
		dbConfig:    &metricStoreDBConfig{dsn: database, database: database},
		dbRecoverCh: make(chan struct{}, 1),
	}

	db, err := openMetricStoreDB(ret.dbConfig)
	require.NoError(t, err)
	ret.setDatabase(db)
	t.Cleanup(func() {
		if db := ret.database(); db != nil {
			_ = db.Close()
		}
	})

	require.NoError(t, ret.initDB(context.Background()))

	return ret
}

func fixedTestStatfs(totalBytes, availableBytes uint64) func(string) (uint64, uint64, error) {
	return func(string) (uint64, uint64, error) {
		return totalBytes, availableBytes, nil
	}
}

func seedTestMetricPoints(t *testing.T, s *Server, seriesID string, ages ...time.Duration) {
	t.Helper()
	ctx := context.Background()

	conn, err := s.database().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	now := time.Now().UTC()
	points := make([]numberPointRecord, 0, len(ages))
	for i, age := range ages {
		value := int64(i)
		at := now.Add(-age)
		points = append(points, numberPointRecord{
			pointID:    fmt.Sprintf("%s-%s", seriesID, age),
			timestamp:  at,
			ingestedAt: at,
			seriesID:   seriesID,
			intValue:   &value,
		})
	}

	require.NoError(t, replaceNumberPoints(ctx, conn, points))
}

func seedTestMetricMetadata(t *testing.T, s *Server, descriptorID, seriesID string, updatedAt time.Time) {
	t.Helper()
	ctx := context.Background()
	at := metricTimeToDB(updatedAt)

	_, err := s.database().ExecContext(ctx, `
INSERT INTO metric_descriptors (
	id, name, kind, number_value_type, unit, description, temporality,
	scope_name, scope_version, scope_schema_url, created_at, updated_at
) VALUES (?, 'test.metric', 'COUNTER', 'INT64', '', '', 'CUMULATIVE', 'default', '', '', ?, ?)
`, descriptorID, at, at)
	require.NoError(t, err)

	_, err = s.database().ExecContext(ctx, `
INSERT INTO metric_series (
	id, descriptor_id, labels, labels_key,
	component_type, component_namespace, component_name, created_at, updated_at
) VALUES (?, ?, '{}', '', 'vigil', 'octelium', 'vigil-1', ?, ?)
`, seriesID, descriptorID, at, at)
	require.NoError(t, err)

	_, err = s.database().ExecContext(ctx, `
INSERT INTO metric_series_attributes (
	series_id, descriptor_id, key, value_kind, value_key, value_string, source_mask, updated_at
) VALUES (?, ?, 'state', 'STRING', 'STRING:ok', 'ok', 1, ?)
`, seriesID, descriptorID, at)
	require.NoError(t, err)

	_, err = s.database().ExecContext(ctx, `
INSERT INTO metric_attribute_keys (
	descriptor_id, key, value_kind, source_mask, first_seen_at, last_seen_at
) VALUES (?, 'state', 'STRING', 1, ?, ?)
`, descriptorID, at, at)
	require.NoError(t, err)
}

func testMetricPointIDs(t *testing.T, s *Server) []string {
	t.Helper()

	rows, err := s.database().QueryContext(context.Background(), `SELECT point_id FROM metric_number_points`)
	require.NoError(t, err)
	defer rows.Close()

	ret := []string{}
	for rows.Next() {
		var pointID string
		require.NoError(t, rows.Scan(&pointID))
		ret = append(ret, pointID)
	}
	require.NoError(t, rows.Err())
	sort.Strings(ret)

	return ret
}

func testSetSchemaVersion(t *testing.T, s *Server, version int) {
	t.Helper()

	_, err := s.database().ExecContext(context.Background(),
		`DELETE FROM metricstore_schema`)
	require.NoError(t, err)

	_, err = s.database().ExecContext(context.Background(),
		`INSERT INTO metricstore_schema (version, applied_at) VALUES (?, ?)`,
		version, metricTimeToDB(time.Now().UTC()))
	require.NoError(t, err)
}

func testTableCount(t *testing.T, s *Server, table string) int64 {
	t.Helper()

	var count int64
	require.NoError(t, s.database().QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM `+table).Scan(&count))

	return count
}

func beginBlockingMetadataTx(t *testing.T, s *Server) *sql.Conn {
	t.Helper()
	ctx := context.Background()

	conn, err := s.database().Conn(ctx)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, `BEGIN TRANSACTION`)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, `
INSERT INTO metric_series (
	id, descriptor_id, labels, labels_key,
	component_type, component_namespace, component_name, created_at, updated_at
) VALUES ('blocking-series', 'blocking-descriptor', '{}', '', 'vigil', 'octelium', 'vigil-1', 0, 0)
ON CONFLICT (id) DO UPDATE SET updated_at = EXCLUDED.updated_at
`)
	require.NoError(t, err)

	return conn
}

func TestReadStorageUsage(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	s.statfsFn = fixedTestStatfs(1000, 400)

	usage, err := s.readStorageUsage(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1000), usage.TotalBytes)
	assert.Equal(t, uint64(400), usage.AvailableBytes)
	assert.Zero(t, usage.ReusableBytes)

	_, err = s.database().ExecContext(ctx,
		`INSERT INTO metric_number_points SELECT md5(i::VARCHAR), i, i, NULL, md5((i*7)::VARCHAR), i, NULL
FROM range(150000) tbl(i)`)
	require.NoError(t, err)
	require.NoError(t, s.checkpointStorage(ctx))

	_, err = s.database().ExecContext(ctx, `DELETE FROM metric_number_points WHERE timestamp < 120000`)
	require.NoError(t, err)
	require.NoError(t, s.checkpointStorage(ctx))

	usage, err = s.readStorageUsage(ctx)
	require.NoError(t, err)
	assert.NotZero(t, usage.ReusableBytes)
	assert.Equal(t, usage.AvailableBytes+usage.ReusableBytes, usage.FreeBytes())
}

func TestReadStorageUsageWithoutDatabasePath(t *testing.T) {
	s := &Server{db: newTestDuckDB(t)}

	usage, err := s.readStorageUsage(context.Background())
	require.NoError(t, err)
	assert.Zero(t, usage.TotalBytes)
	assert.Zero(t, usage.UsedRatio())
}

func TestCheckpointStorageWithActiveWriteTransaction(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	seedTestMetricPoints(t, s, "series-1", time.Minute)

	conn := beginBlockingMetadataTx(t, s)
	defer conn.Close()

	_, err := s.database().ExecContext(ctx, `CHECKPOINT`)
	require.NotNil(t, err)
	assert.True(t, ovutils.IsBlockedCheckpointErr(err))

	committed := make(chan struct{})
	go func() {
		defer close(committed)
		time.Sleep(200 * time.Millisecond)
		_, _ = conn.ExecContext(context.Background(), `COMMIT`)
	}()

	assert.Nil(t, s.checkpointStorage(ctx))
	<-committed

	assert.Equal(t, int64(1), testTableCount(t, s, "metric_series"))
}

func TestApplyRetentionWithActiveWriteTransaction(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	seedTestMetricPoints(t, s, "series-1", 72*time.Hour, time.Minute)

	conn := beginBlockingMetadataTx(t, s)
	defer conn.Close()

	committed := make(chan struct{})
	go func() {
		defer close(committed)
		time.Sleep(200 * time.Millisecond)
		_, _ = conn.ExecContext(context.Background(), `COMMIT`)
	}()

	assert.Nil(t, s.applyRetention(ctx))
	<-committed

	assert.Equal(t, []string{"series-1-1m0s"}, testMetricPointIDs(t, s))
}

func TestApplyRetentionRemovesExpiredPointsAndOrphanedMetadata(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	oldAt := time.Now().UTC().Add(-72 * time.Hour)
	seedTestMetricMetadata(t, s, "descriptor-1", "series-1", oldAt)
	seedTestMetricMetadata(t, s, "descriptor-2", "series-2", oldAt)
	seedTestMetricPoints(t, s, "series-1", 49*time.Hour, 47*time.Hour, time.Minute)
	seedTestMetricPoints(t, s, "series-2", 49*time.Hour)

	require.Nil(t, s.applyRetention(ctx))

	assert.Equal(t, []string{"series-1-1m0s", "series-1-47h0m0s"}, testMetricPointIDs(t, s))

	for _, table := range []string{
		"metric_series", "metric_series_attributes", "metric_descriptors", "metric_attribute_keys",
	} {
		assert.Equal(t, int64(1), testTableCount(t, s, table), table)
	}
}

func TestApplyRetentionKeepsMetadataWithoutExpiredPoints(t *testing.T) {
	s := newTestStorageServer(t)

	seedTestMetricMetadata(t, s, "descriptor-1", "series-1", time.Now().UTC())
	seedTestMetricPoints(t, s, "series-1", time.Minute)

	require.Nil(t, s.applyRetention(context.Background()))

	assert.Equal(t, []string{"series-1-1m0s"}, testMetricPointIDs(t, s))
	assert.Equal(t, int64(1), testTableCount(t, s, "metric_series"))
}

func TestApplyStoragePressureRetentionBelowWatermark(t *testing.T) {
	s := newTestStorageServer(t)

	seedTestMetricPoints(t, s, "series-1", 47*time.Hour, 30*time.Hour, time.Minute)
	s.statfsFn = fixedTestStatfs(100<<20, 50<<20)

	require.Nil(t, s.applyStoragePressureRetention(context.Background()))

	assert.Len(t, testMetricPointIDs(t, s), 3)
}

func TestApplyStoragePressureRetentionWithoutStatfs(t *testing.T) {
	s := &Server{db: newTestDuckDB(t)}

	require.Nil(t, s.applyStoragePressureRetention(context.Background()))
}

func TestApplyStoragePressureRetentionTrimsOldestDataPoints(t *testing.T) {
	s := newTestStorageServer(t)

	seedTestMetricPoints(t, s, "series-1",
		47*time.Hour, 30*time.Hour, 20*time.Hour, 10*time.Hour, 5*time.Hour, 30*time.Minute)

	reads := 0
	s.statfsFn = func(string) (uint64, uint64, error) {
		reads++
		if reads > 2 {
			return 100 << 20, 50 << 20, nil
		}
		return 100 << 20, 1 << 20, nil
	}

	require.Nil(t, s.applyStoragePressureRetention(context.Background()))

	assert.Equal(t, []string{"series-1-10h0m0s", "series-1-30m0s", "series-1-5h0m0s"},
		testMetricPointIDs(t, s))
	assert.Equal(t, 3, reads)
}

func TestApplyStoragePressureRetentionStopsAtMinimumRetention(t *testing.T) {
	s := newTestStorageServer(t)

	seedTestMetricPoints(t, s, "series-1",
		47*time.Hour, 30*time.Hour, 20*time.Hour, 10*time.Hour, 5*time.Hour,
		2*time.Hour, 90*time.Minute, 30*time.Minute)
	s.statfsFn = fixedTestStatfs(100<<20, 1<<20)

	require.Nil(t, s.applyStoragePressureRetention(context.Background()))

	assert.Equal(t, []string{"series-1-30m0s"}, testMetricPointIDs(t, s))
}

func TestApplyStoragePressureRetentionRemovesOrphanedMetadata(t *testing.T) {
	s := newTestStorageServer(t)

	seedTestMetricMetadata(t, s, "descriptor-1", "series-1", time.Now().UTC().Add(-47*time.Hour))
	seedTestMetricPoints(t, s, "series-1", 47*time.Hour)
	s.statfsFn = fixedTestStatfs(100<<20, 1<<20)

	require.Nil(t, s.applyStoragePressureRetention(context.Background()))

	assert.Empty(t, testMetricPointIDs(t, s))
	for _, table := range []string{
		"metric_series", "metric_series_attributes", "metric_descriptors", "metric_attribute_keys",
	} {
		assert.Zero(t, testTableCount(t, s, table), table)
	}
}

func TestApplyRetentionKeepsRecentlyUpdatedOrphanedMetadata(t *testing.T) {
	s := newTestStorageServer(t)

	seedTestMetricMetadata(t, s, "descriptor-1", "series-1", time.Now().UTC())
	seedTestMetricMetadata(t, s, "descriptor-2", "series-2", time.Now().UTC().Add(-72*time.Hour))
	seedTestMetricPoints(t, s, "series-3", 49*time.Hour)

	require.Nil(t, s.applyRetention(context.Background()))

	assert.Empty(t, testMetricPointIDs(t, s))
	for _, table := range []string{
		"metric_series", "metric_series_attributes", "metric_descriptors", "metric_attribute_keys",
	} {
		assert.Equal(t, int64(1), testTableCount(t, s, table), table)
	}
}

func TestApplyStoragePressureRetentionKeepsRecentDataPoints(t *testing.T) {
	s := newTestStorageServer(t)

	seedTestMetricPoints(t, s, "series-1", 30*time.Minute, 10*time.Minute)
	s.statfsFn = fixedTestStatfs(100<<20, 1<<20)

	require.Nil(t, s.applyStoragePressureRetention(context.Background()))

	assert.Equal(t, []string{"series-1-10m0s", "series-1-30m0s"}, testMetricPointIDs(t, s))
}

func TestDeleteMetricPointsBefore(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	seedTestMetricPoints(t, s, "series-1", 3*time.Hour, 2*time.Hour, time.Minute)

	deleted, err := s.deleteMetricPointsBefore(ctx, time.Now().UTC().Add(-90*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	deleted, err = s.deleteMetricPointsBefore(ctx, time.Now().UTC().Add(-90*time.Minute))
	require.NoError(t, err)
	assert.Zero(t, deleted)
	assert.Equal(t, []string{"series-1-1m0s"}, testMetricPointIDs(t, s))
}

func TestDeleteMetricPointsBeforeDeletesLargeBacklogInChunks(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	cutoff := time.Now().UTC().Add(-time.Hour)
	rows := int64(retentionDeleteChunkSize)*2 + 1
	_, err := s.database().ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO metric_number_points
SELECT md5(i::VARCHAR), %[1]d - i, %[1]d - i, NULL, 'series-1', i, NULL
FROM range(%[2]d) tbl(i)`, metricTimeToDB(cutoff)-1, rows))
	require.NoError(t, err)

	deleted, err := s.deleteMetricPointsBefore(ctx, cutoff)
	require.NoError(t, err)
	assert.Equal(t, rows, deleted)
	assert.Zero(t, testTableCount(t, s, "metric_number_points"))
}

func TestDeleteMetricPointsBeforeKeepsPointsAtTheChunkBoundary(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	cutoff := time.Now().UTC().Add(-time.Hour)
	seedTestMetricPoints(t, s, "series-1", time.Minute)
	_, err := s.database().ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO metric_number_points
SELECT md5(i::VARCHAR), %[1]d - i, %[1]d - i, NULL, 'series-2', i, NULL
FROM range(%[2]d) tbl(i)`, metricTimeToDB(cutoff)-1, retentionDeleteChunkSize))
	require.NoError(t, err)

	deleted, err := s.deleteMetricPointsBefore(ctx, cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(retentionDeleteChunkSize), deleted)
	assert.Equal(t, []string{"series-1-1m0s"}, testMetricPointIDs(t, s))
}

func TestRunStoragePressureLoopExitsOnContextCancellation(t *testing.T) {
	s := newTestStorageServer(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runStoragePressureLoop(ctx)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the metricstore storage pressure loop did not exit")
	}
}

func TestRetentionRunsAlongsideConcurrentIngestion(t *testing.T) {
	s := newTestStorageServer(t)
	srv := &srvMetric{s: s}
	ctx := context.Background()

	seedTestMetricPoints(t, s, "series-1", 72*time.Hour, 60*time.Hour)
	s.statfsFn = fixedTestStatfs(100<<20, 1<<20)

	ingestErr := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		now := time.Now().UTC()
		for i := 0; i < 100; i++ {
			metrics, sm := newVigilReplicaMetrics(fmt.Sprintf("octelium-vigil-%d", i%4))
			appendTestCounter(sm, "req.total", now.Add(-time.Minute),
				now.Add(time.Duration(i)*time.Millisecond), int64(i), map[string]any{"state": "ok"})

			batch, err := buildMetricWriteBatch(metrics)
			if err != nil {
				ingestErr <- err
				return
			}
			batch.normalize()

			if err := srv.storeMetricWriteBatch(ctx, batch); err != nil {
				ingestErr <- err
				return
			}
		}
	}()

	for i := 0; i < 10; i++ {
		require.Nil(t, s.applyRetention(ctx))
		require.Nil(t, s.applyStoragePressureRetention(ctx))
	}
	<-done

	select {
	case err := <-ingestErr:
		t.Fatalf("could not ingest metrics alongside the metricstore retention: %v", err)
	default:
	}

	assert.Equal(t, int64(100), testTableCount(t, s, "metric_number_points"))
	assert.Equal(t, int64(4), testTableCount(t, s, "metric_series"))
	assert.Equal(t, int64(1), testTableCount(t, s, "metric_descriptors"))
}

func TestDeleteOrphanedMetricMetadataKeepsConcurrentlyUpsertedSeries(t *testing.T) {
	s := newTestStorageServer(t)
	ctx := context.Background()

	now := time.Now().UTC()
	seedTestMetricMetadata(t, s, "descriptor-1", "series-1", now.Add(-5*time.Minute))

	conn, err := s.database().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.ExecContext(ctx, `BEGIN TRANSACTION`)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, `
INSERT INTO metric_series (
	id, descriptor_id, labels, labels_key,
	component_type, component_namespace, component_name, created_at, updated_at
) VALUES ('series-1', 'descriptor-1', '{}', '', 'vigil', 'octelium', 'vigil-1', ?, ?)
ON CONFLICT (id) DO UPDATE SET updated_at = EXCLUDED.updated_at
`, metricTimeToDB(now), metricTimeToDB(now))
	require.NoError(t, err)

	require.Nil(t, s.deleteOrphanedMetricMetadata(ctx, now.Add(-time.Hour)))

	_, err = conn.ExecContext(ctx, `COMMIT`)
	require.NoError(t, err)

	assert.Equal(t, int64(1), testTableCount(t, s, "metric_series"))
	assert.Equal(t, int64(1), testTableCount(t, s, "metric_descriptors"))
	assert.Equal(t, int64(1), testTableCount(t, s, "metric_series_attributes"))

	var updatedAt int64
	require.NoError(t, s.database().QueryRowContext(ctx,
		`SELECT updated_at FROM metric_series WHERE id = 'series-1'`).Scan(&updatedAt))
	assert.Equal(t, metricTimeToDB(now), updatedAt)
}
