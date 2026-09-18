// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/stretchr/testify/assert"
)

func fixedTestStatfs(totalBytes, availableBytes uint64) func(string) (uint64, uint64, error) {
	return func(string) (uint64, uint64, error) {
		return totalBytes, availableBytes, nil
	}
}

func setTestCleanupMaxCounts(t *testing.T, accessLogs, authenticationLogs, auditLogs,
	componentLogs, componentLogsDebug int) {
	t.Helper()

	oldAccessMax := maxDBAccessLogs
	oldAuthenticationMax := maxDBAuthenticationLogs
	oldAuditMax := maxDBAuditLogs
	oldComponentMax := maxDBComponentLogs
	oldComponentDebugMax := maxDBComponentLogsDebug
	t.Cleanup(func() {
		maxDBAccessLogs = oldAccessMax
		maxDBAuthenticationLogs = oldAuthenticationMax
		maxDBAuditLogs = oldAuditMax
		maxDBComponentLogs = oldComponentMax
		maxDBComponentLogsDebug = oldComponentDebugMax
	})

	maxDBAccessLogs = accessLogs
	maxDBAuthenticationLogs = authenticationLogs
	maxDBAuditLogs = auditLogs
	maxDBComponentLogs = componentLogs
	maxDBComponentLogsDebug = componentLogsDebug
}

func insertTestAccessLogs(t *testing.T, srv *Server, count int, now time.Time) {
	t.Helper()

	for idx := range count {
		insertLogJSON(t, srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{
			CreatedAt: now.Add(time.Duration(idx) * time.Second),
		})))
	}
}

func TestReadStorageUsage(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	usage, err := ts.srv.readStorageUsage(ts.ctx)
	assert.Nil(t, err, "%+v", err)
	assert.NotZero(t, usage.TotalBytes)
	assert.LessOrEqual(t, usage.AvailableBytes, usage.TotalBytes)

	ts.srv.statfsFn = fixedTestStatfs(1000, 400)

	usage, err = ts.srv.readStorageUsage(ts.ctx)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, uint64(1000), usage.TotalBytes)
	assert.Equal(t, uint64(400), usage.AvailableBytes)
	assert.Equal(t, usage.AvailableBytes+usage.ReusableBytes, usage.FreeBytes())
}

func TestReadStorageUsageReportsReusableBytes(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	ts.srv.statfsFn = fixedTestStatfs(1<<30, 1<<29)

	_, err := ts.srv.db.ExecContext(ts.ctx, `
INSERT INTO access_logs
SELECT json_object('n', i, 'v', md5((i*7)::VARCHAR), 'w', md5((i*13)::VARCHAR))
FROM range(300000) tbl(i)`)
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, ts.srv.checkpointStorage(ts.ctx))

	_, err = ts.srv.db.ExecContext(ts.ctx,
		`DELETE FROM access_logs WHERE CAST(rsc->>'n' AS BIGINT) < 240000`)
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, ts.srv.checkpointStorage(ts.ctx))

	usage, err := ts.srv.readStorageUsage(ts.ctx)
	assert.Nil(t, err, "%+v", err)
	assert.NotZero(t, usage.ReusableBytes)
}

func beginBlockingCatalogTx(t *testing.T, srv *Server) *sql.Conn {
	t.Helper()
	ctx := context.Background()

	conn, err := srv.db.Conn(ctx)
	assert.Nil(t, err, "%+v", err)

	_, err = conn.ExecContext(ctx, `BEGIN TRANSACTION`)
	assert.Nil(t, err, "%+v", err)

	_, err = conn.ExecContext(ctx, `CREATE TABLE blocking_logs (rsc JSON)`)
	assert.Nil(t, err, "%+v", err)

	_, err = conn.ExecContext(ctx, `INSERT INTO access_logs VALUES ('{"id": "pending"}')`)
	assert.Nil(t, err, "%+v", err)

	return conn
}

func TestCheckpointStorageWithConcurrentWrites(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	conn, err := ts.srv.db.Conn(ts.ctx)
	assert.Nil(t, err, "%+v", err)
	defer conn.Close()

	_, err = conn.ExecContext(ts.ctx, `BEGIN TRANSACTION`)
	assert.Nil(t, err, "%+v", err)

	_, err = conn.ExecContext(ts.ctx, `INSERT INTO access_logs VALUES ('{"id": "pending"}')`)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, ts.srv.checkpointStorage(ts.ctx))

	_, err = conn.ExecContext(ts.ctx, `COMMIT`)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 1, getTableCount(t, ts.srv, "access_logs"))
}

func TestCheckpointStorageWaitsForBlockingTransaction(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	conn := beginBlockingCatalogTx(t, ts.srv)
	defer conn.Close()

	_, err := ts.srv.db.ExecContext(ts.ctx, `CHECKPOINT`)
	assert.NotNil(t, err)
	assert.True(t, ovutils.IsBlockedCheckpointErr(err))

	committed := make(chan struct{})
	go func() {
		defer close(committed)
		time.Sleep(200 * time.Millisecond)
		_, _ = conn.ExecContext(context.Background(), `COMMIT`)
	}()

	assert.Nil(t, ts.srv.checkpointStorage(ts.ctx))
	<-committed

	assert.Equal(t, 1, getTableCount(t, ts.srv, "access_logs"))
}

func TestCheckpointStorageGivesUpOnContextCancellation(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	conn := beginBlockingCatalogTx(t, ts.srv)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ts.ctx, time.Second)
	defer cancel()

	assert.NotNil(t, ts.srv.checkpointStorage(ctx))

	_, err := conn.ExecContext(ts.ctx, `COMMIT`)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 1, getTableCount(t, ts.srv, "access_logs"))
}

func TestApplyStoragePressureCleanupBelowWatermark(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	setTestCleanupMaxCounts(t, 40, 40, 40, 40, 40)
	insertTestAccessLogs(t, ts.srv, 60, time.Now().UTC().Truncate(time.Second))
	ts.srv.statfsFn = fixedTestStatfs(100<<20, 50<<20)

	assert.Nil(t, ts.srv.applyStoragePressureCleanup(ts.ctx))

	assert.Equal(t, 60, getTableCount(t, ts.srv, "access_logs"))
}

func TestApplyStoragePressureCleanupTrimsOldestLogs(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	setTestCleanupMaxCounts(t, 4000, 4000, 4000, 4000, 4000)

	now := time.Now().UTC().Truncate(time.Second)
	insertTestAccessLogs(t, ts.srv, 3000, now)
	ts.srv.statfsFn = fixedTestStatfs(100<<20, 1<<20)

	assert.Nil(t, ts.srv.applyStoragePressureCleanup(ts.ctx))

	assert.Equal(t, minimumCleanupLimit, getTableCount(t, ts.srv, "access_logs"))

	var oldest string
	assert.Nil(t, ts.srv.db.QueryRowContext(ts.ctx,
		`SELECT min(rsc->'metadata'->>'createdAt') FROM access_logs`).Scan(&oldest))

	oldestAt, err := time.Parse(time.RFC3339Nano, oldest)
	assert.Nil(t, err, "%+v", err)
	assert.False(t, oldestAt.Before(now.Add(2000*time.Second)))
}

func TestApplyStoragePressureCleanupStopsWhenPressureIsRelieved(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	setTestCleanupMaxCounts(t, 4000, 4000, 4000, 4000, 4000)
	insertTestAccessLogs(t, ts.srv, 3000, time.Now().UTC().Truncate(time.Second))

	reads := 0
	ts.srv.statfsFn = func(string) (uint64, uint64, error) {
		reads++
		if reads > 1 {
			return 100 << 20, 50 << 20, nil
		}
		return 100 << 20, 1 << 20, nil
	}

	assert.Nil(t, ts.srv.applyStoragePressureCleanup(ts.ctx))

	assert.Equal(t, 2000, getTableCount(t, ts.srv, "access_logs"))
	assert.Equal(t, 2, reads)
}

func TestApplyStoragePressureCleanupTrimsEveryLogTable(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	setTestCleanupMaxCounts(t, 2400, 2400, 2400, 2400, 2200)

	now := time.Now().UTC().Truncate(time.Second)
	for idx := range 1400 {
		createdAt := now.Add(time.Duration(idx) * time.Second)
		insertLogJSON(t, ts.srv, "access_logs",
			marshalLog(t, newAccessLog(&accessLogOptions{CreatedAt: createdAt})))
		insertLogJSON(t, ts.srv, "authentication_logs",
			marshalLog(t, newAuthenticationLog(&authenticationLogOptions{CreatedAt: createdAt})))
		insertLogJSON(t, ts.srv, "audit_logs",
			marshalLog(t, newAuditLog(&auditLogOptions{CreatedAt: createdAt})))
		insertLogJSON(t, ts.srv, "component_logs",
			marshalLog(t, newComponentLog(createdAt, corev1.ComponentLog_Entry_DEBUG, "debug")))
	}

	reads := 0
	ts.srv.statfsFn = func(string) (uint64, uint64, error) {
		reads++
		if reads > 1 {
			return 100 << 20, 50 << 20, nil
		}
		return 100 << 20, 1 << 20, nil
	}

	assert.Nil(t, ts.srv.applyStoragePressureCleanup(ts.ctx))

	for _, table := range []string{"access_logs", "authentication_logs", "audit_logs"} {
		assert.Equal(t, 1200, getTableCount(t, ts.srv, table), table)
	}
	assert.Equal(t, 1100, getTableCount(t, ts.srv, "component_logs"))
}

func TestApplyStoragePressureCleanupWithoutTrimmableLogs(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	setTestCleanupMaxCounts(t, 4000, 4000, 4000, 4000, 4000)
	insertTestAccessLogs(t, ts.srv, 100, time.Now().UTC().Truncate(time.Second))
	ts.srv.statfsFn = fixedTestStatfs(100<<20, 1<<20)

	assert.Nil(t, ts.srv.applyStoragePressureCleanup(ts.ctx))

	assert.Equal(t, 100, getTableCount(t, ts.srv, "access_logs"))
}

func TestGetCleanupMaxCounts(t *testing.T) {
	counts := getCleanupMaxCounts()
	assert.Len(t, counts, 5)

	for _, c := range counts {
		assert.NotEmpty(t, c.table)
		assert.NotZero(t, c.limit)
	}

	assert.Equal(t, "component_logs", counts[2].table)
	assert.Contains(t, counts[2].where, corev1.ComponentLog_Entry_DEBUG.String())
	assert.Equal(t, maxDBComponentLogsDebug, counts[2].limit)
}

func TestStartStoragePressureLoopExitsOnContextCancellation(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	ctx, cancel := context.WithCancel(ts.ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		ts.srv.startStoragePressureLoop(ctx)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the LogStore storage pressure loop did not exit")
	}
}
