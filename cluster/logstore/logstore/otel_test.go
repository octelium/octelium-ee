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
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newExportRequest(logs ...[]byte) plogotlp.ExportRequest {
	req := plogotlp.NewExportRequest()
	logRecords := req.Logs().ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()

	for _, log := range logs {
		logRecords.AppendEmpty().Body().SetStr(string(log))
	}

	return req
}

func startTestLogProcessor(t *testing.T, ctx context.Context, srv *Server) (*srvLog, context.CancelFunc) {
	t.Helper()

	processCtx, cancel := context.WithCancel(ctx)
	doneCh := make(chan struct{})
	ret := srv.newSrvLog()

	go func() {
		defer close(doneCh)
		ret.startProcessLoop(processCtx)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-doneCh:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "Log processor did not stop")
		}
	})

	return ret, cancel
}

func TestOTLPBatchExportAllLogTypes(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	srvLog, _ := startTestLogProcessor(t, ts.ctx, ts.srv)
	now := time.Now().UTC()

	req := newExportRequest(
		marshalLog(t, newAccessLog(&accessLogOptions{CreatedAt: now})),
		marshalLog(t, newAuthenticationLog(&authenticationLogOptions{
			CreatedAt: now,
			Type:      corev1.Session_Status_Authentication_Info_CREDENTIAL,
			AAL:       corev1.Session_Status_Authentication_Info_AAL1,
		})),
		marshalLog(t, newAuditLog(&auditLogOptions{CreatedAt: now})),
		marshalLog(t, newComponentLog(now, corev1.ComponentLog_Entry_INFO, "test")),
	)

	_, err := srvLog.Export(ts.ctx, req)
	assert.Nil(t, err, "%+v", err)

	assert.Equal(t, 1, getTableCount(t, ts.srv, "access_logs"))
	assert.Equal(t, 1, getTableCount(t, ts.srv, "authentication_logs"))
	assert.Equal(t, 1, getTableCount(t, ts.srv, "audit_logs"))
	assert.Equal(t, 1, getTableCount(t, ts.srv, "component_logs"))
}

func TestOTLPBatchFlushBySize(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	srvLog, _ := startTestLogProcessor(t, ts.ctx, ts.srv)
	now := time.Now().UTC()
	logs := make([][]byte, 0, logBatchSize)

	for idx := 0; idx < logBatchSize; idx++ {
		logs = append(logs, marshalLog(t, newComponentLog(now.Add(time.Duration(idx)*time.Millisecond), corev1.ComponentLog_Entry_INFO, "batch")))
	}

	_, err := srvLog.Export(ts.ctx, newExportRequest(logs...))
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, logBatchSize, getTableCount(t, ts.srv, "component_logs"))
}

func TestOTLPBatchTransactionRollback(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	_, err := ts.srv.db.Exec("DROP TABLE audit_logs")
	assert.Nil(t, err, "%+v", err)

	srvLog, _ := startTestLogProcessor(t, ts.ctx, ts.srv)
	now := time.Now().UTC()

	_, err = srvLog.Export(ts.ctx, newExportRequest(
		marshalLog(t, newAccessLog(&accessLogOptions{CreatedAt: now})),
		marshalLog(t, newAuditLog(&auditLogOptions{CreatedAt: now})),
	))
	assert.NotNil(t, err)
	assert.Equal(t, 0, getTableCount(t, ts.srv, "access_logs"))
}

func TestOTLPBatchRejectsInvalidInput(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	srvLog, _ := startTestLogProcessor(t, ts.ctx, ts.srv)

	{
		_, err := srvLog.Export(ts.ctx, newExportRequest([]byte("{invalid")))
		assert.NotNil(t, err)
	}

	{
		unknown := &enterprisev1.AuditLog{
			ApiVersion: "enterprise/v1",
			Kind:       "UnknownLog",
		}
		_, err := srvLog.Export(ts.ctx, newExportRequest(marshalLog(t, unknown)))
		assert.NotNil(t, err)
	}

	{
		oversized := newAccessLog(&accessLogOptions{CreatedAt: time.Now()})
		oversized.Entry.Info = &corev1.AccessLog_Entry_Info{
			Type: &corev1.AccessLog_Entry_Info_Http{
				Http: &corev1.AccessLog_Entry_Info_HTTP{
					Request: &corev1.AccessLog_Entry_Info_HTTP_Request{
						Body: []byte(strings.Repeat("a", maxAccessLogSize)),
					},
				},
			},
		}

		data, err := pbutils.MarshalJSON(oversized, false)
		assert.Nil(t, err, "%+v", err)

		_, err = srvLog.Export(ts.ctx, newExportRequest(data))
		assert.NotNil(t, err)
	}

	{
		req := plogotlp.NewExportRequest()
		req.Logs().ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetInt(1)

		_, err := srvLog.Export(ts.ctx, req)
		assert.NotNil(t, err)
	}

	assert.Equal(t, 0, getTableCount(t, ts.srv, "access_logs"))
	assert.Equal(t, 0, getTableCount(t, ts.srv, "authentication_logs"))
	assert.Equal(t, 0, getTableCount(t, ts.srv, "audit_logs"))
	assert.Equal(t, 0, getTableCount(t, ts.srv, "component_logs"))
}

func TestOTLPBatchRejectsTooManyLogs(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	srvLog, _ := startTestLogProcessor(t, ts.ctx, ts.srv)
	body := marshalLog(t, newComponentLog(time.Now(), corev1.ComponentLog_Entry_INFO, "test"))
	logs := make([][]byte, maxLogsPerExport+1)

	for idx := range logs {
		logs[idx] = body
	}

	_, err := srvLog.Export(ts.ctx, newExportRequest(logs...))
	assert.NotNil(t, err)
	assert.Equal(t, 0, getTableCount(t, ts.srv, "component_logs"))
}

func TestOTLPDropsDebugComponentLogsInProduction(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	srvLog, _ := startTestLogProcessor(t, ts.ctx, ts.srv)
	now := time.Now().UTC()

	assert.True(t, ldflags.IsProduction())

	{
		_, err := srvLog.Export(ts.ctx, newExportRequest(
			marshalLog(t, newComponentLog(now, corev1.ComponentLog_Entry_DEBUG, "prod")),
			marshalLog(t, newComponentLog(now, corev1.ComponentLog_Entry_INFO, "prod")),
			marshalLog(t, newAccessLog(&accessLogOptions{CreatedAt: now})),
		))
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, getTableCount(t, ts.srv, "component_logs"))
		assert.Equal(t, 1, getTableCount(t, ts.srv, "access_logs"))
	}

	{
		_, err := srvLog.Export(ts.ctx, newExportRequest(
			marshalLog(t, newComponentLog(now, corev1.ComponentLog_Entry_DEBUG, "prod")),
		))
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, getTableCount(t, ts.srv, "component_logs"))
	}

	t.Setenv("OCTELIUM_DEV", "true")
	assert.True(t, ldflags.IsDev())

	{
		_, err := srvLog.Export(ts.ctx, newExportRequest(
			marshalLog(t, newComponentLog(now, corev1.ComponentLog_Entry_DEBUG, "dev")),
		))
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, getTableCount(t, ts.srv, "component_logs"))
	}
}

func TestParseLogRecordEncodingsAndLimits(t *testing.T) {
	for _, tt := range []struct {
		kind  string
		table logTable
		limit int
	}{
		{kind: "AccessLog", table: logTableAccess, limit: maxAccessLogSize},
		{kind: "ComponentLog", table: logTableComponent, limit: maxLogRecordSize},
		{kind: "AuthenticationLog", table: logTableAuthentication, limit: maxLogRecordSize},
		{kind: "AuditLog", table: logTableAudit, limit: maxLogRecordSize},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			base := `{"kind":"` + tt.kind + `","padding":""}`
			body := strings.Replace(base, `"padding":""`, `"padding":"`+strings.Repeat("x", tt.limit-len(base))+`"`, 1)
			require.Len(t, body, tt.limit)
			lr := plog.NewLogRecord()
			lr.Body().SetStr(body)
			parsed, err := parseLogRecord(lr)
			require.Nil(t, err)
			assert.Equal(t, tt.table, parsed.table)
			assert.Equal(t, body, string(parsed.data))
			lr.Body().SetStr(body + " ")
			_, err = parseLogRecord(lr)
			assert.NotNil(t, err)
			lr.Body().SetEmptyMap().PutStr("kind", tt.kind)
			parsed, err = parseLogRecord(lr)
			require.Nil(t, err)
			before := append([]byte(nil), parsed.data...)
			lr.Body().Map().PutStr("kind", "changed")
			assert.Equal(t, before, parsed.data)
		})
	}
	for _, setup := range []func(plog.LogRecord){
		func(lr plog.LogRecord) { lr.Body().SetStr("") },
		func(lr plog.LogRecord) { lr.Body().SetBool(true) },
		func(lr plog.LogRecord) { lr.Body().SetEmptyBytes().FromRaw([]byte("data")) },
		func(lr plog.LogRecord) { lr.Body().SetEmptySlice() },
		func(lr plog.LogRecord) { lr.Body().SetEmptyMap().PutDouble("value", math.NaN()) },
	} {
		lr := plog.NewLogRecord()
		setup(lr)
		_, err := parseLogRecord(lr)
		assert.NotNil(t, err)
	}
}

func TestOTLPExportCancellationAndBackpressure(t *testing.T) {
	body := marshalLog(t, newComponentLog(time.Now(), corev1.ComponentLog_Entry_INFO, "queue"))
	t.Run("BeforeEnqueue", func(t *testing.T) {
		srv := &srvLog{batchCh: make(chan *pendingLogBatch)}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := srv.Export(ctx, newExportRequest(body))
		assert.ErrorIs(t, err, context.Canceled)
	})
	t.Run("WaitingForCommit", func(t *testing.T) {
		srv := &srvLog{batchCh: make(chan *pendingLogBatch, 1)}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { _, err := srv.Export(ctx, newExportRequest(body)); done <- err }()
		batch := <-srv.batchCh
		cancel()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled)
		case <-time.After(time.Second):
			t.Fatal("Canceled export did not return")
		}
		batch.done <- nil
	})
	t.Run("QueueFull", func(t *testing.T) {
		srv := &srvLog{batchCh: make(chan *pendingLogBatch, 1)}
		srv.batchCh <- &pendingLogBatch{}
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		_, err := srv.Export(ctx, newExportRequest(body))
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "queue is full")
		assert.Len(t, srv.batchCh, 1)
	})
}

func TestOTLPMultipleScopesAndConcurrentExports(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	srv, _ := startTestLogProcessor(t, ts.ctx, ts.srv)
	req := plogotlp.NewExportRequest()
	for i := range 3 {
		resource := req.Logs().ResourceLogs().AppendEmpty()
		for j := range 2 {
			scope := resource.ScopeLogs().AppendEmpty()
			scope.LogRecords().AppendEmpty().Body().SetStr(string(marshalLog(t,
				newComponentLog(time.Now(), corev1.ComponentLog_Entry_INFO, fmt.Sprintf("scope-%d-%d", i, j)))))
		}
	}
	_, err := srv.Export(ts.ctx, req)
	require.Nil(t, err)
	assert.Equal(t, 6, getTableCount(t, ts.srv, "component_logs"))
	requests := make([]plogotlp.ExportRequest, 12)
	for i := range requests {
		requests[i] = newExportRequest(marshalLog(t, newAccessLog(&accessLogOptions{CreatedAt: time.Now()})))
	}
	results := make(chan error, len(requests))
	for _, request := range requests {
		go func() { _, err := srv.Export(ts.ctx, request); results <- err }()
	}
	for range requests {
		select {
		case err := <-results:
			assert.Nil(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent export did not finish")
		}
	}
	assert.Equal(t, len(requests), getTableCount(t, ts.srv, "access_logs"))
}

func TestOTLPInsertBatchRollbackAndRecovery(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	valid := pendingLog{table: logTableComponent, data: marshalLog(t, newComponentLog(time.Now(), corev1.ComponentLog_Entry_INFO, "rollback"))}
	for _, invalid := range []pendingLog{{table: logTableUnknown, data: []byte(`{}`)}, {table: logTableAudit, data: []byte(`{invalid`)}} {
		err := ts.srv.insertLogBatches(ts.ctx, []*pendingLogBatch{{items: []pendingLog{valid}}, {items: []pendingLog{invalid}}})
		assert.NotNil(t, err)
		assert.Zero(t, getTableCount(t, ts.srv, "component_logs"))
	}
	ctx, cancel := context.WithCancel(ts.ctx)
	cancel()
	assert.ErrorIs(t, ts.srv.insertLogBatches(ctx, []*pendingLogBatch{{items: []pendingLog{valid}}}), context.Canceled)
	require.Nil(t, ts.srv.insertLogBatches(ts.ctx, []*pendingLogBatch{{items: []pendingLog{valid}}}))
	assert.Equal(t, 1, getTableCount(t, ts.srv, "component_logs"))
}

func TestOTLPProcessorFlushesPendingOnShutdown(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	ctx, cancel := context.WithCancel(ts.ctx)
	srv := ts.srv.newSrvLog()
	batch := &pendingLogBatch{
		items: []pendingLog{{table: logTableComponent, data: marshalLog(t, newComponentLog(time.Now(), corev1.ComponentLog_Entry_INFO, "shutdown"))}},
		done:  make(chan error, 1),
	}
	done := make(chan struct{})
	go func() { defer close(done); srv.startProcessLoop(ctx) }()
	srv.batchCh <- batch
	require.Eventually(t, func() bool { return len(srv.batchCh) == 0 }, time.Second, time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Log processor did not stop")
	}
	assert.Nil(t, <-batch.done)
	assert.Equal(t, 1, getTableCount(t, ts.srv, "component_logs"))
}

func TestOTLPProcessorDrainsQueueOnShutdown(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("DatabaseFailure=%t", fail), func(t *testing.T) {
			ts := newTestServer(t)
			require.NotNil(t, ts)
			srvLog := ts.srv.newSrvLog()
			now := time.Now().UTC()
			items := []pendingLog{
				{table: logTableAccess, data: marshalLog(t, newAccessLog(nil))},
				{table: logTableAuthentication, data: marshalLog(t, newAuthenticationLog(nil))},
				{table: logTableAudit, data: marshalLog(t, newAuditLog(nil))},
				{table: logTableComponent, data: marshalLog(t, newComponentLog(now, corev1.ComponentLog_Entry_INFO, "queued"))},
			}
			var batches []*pendingLogBatch
			for range logQueueSize {
				batch := &pendingLogBatch{items: items, done: make(chan error, 1)}
				srvLog.batchCh <- batch
				batches = append(batches, batch)
			}
			if fail {
				require.NoError(t, ts.srv.db.Close())
			}

			ctx, cancel := context.WithCancel(ts.ctx)
			cancel()
			done := make(chan struct{})
			go func() {
				defer close(done)
				srvLog.startProcessLoop(ctx)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Log processor did not drain its queue")
			}

			assert.Empty(t, srvLog.batchCh)
			for _, batch := range batches {
				select {
				case err := <-batch.done:
					if fail {
						assert.Error(t, err)
					} else {
						assert.NoError(t, err)
					}
				default:
					t.Fatal("Accepted batch was not acknowledged")
				}
			}
			if !fail {
				for _, table := range []string{"access_logs", "authentication_logs", "audit_logs", "component_logs"} {
					assert.Equal(t, logQueueSize, getTableCount(t, ts.srv, table))
				}
			}
		})
	}
}

func TestOTLPExportDuringShutdown(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	srvLog := ts.srv.newSrvLog()
	data := marshalLog(t, newComponentLog(time.Now().UTC(), corev1.ComponentLog_Entry_INFO, "queued"))
	var batches []*pendingLogBatch
	for range logQueueSize {
		batch := &pendingLogBatch{items: []pendingLog{{table: logTableComponent, data: data}}, done: make(chan error, 1)}
		srvLog.batchCh <- batch
		batches = append(batches, batch)
	}

	ctx, cancel := context.WithTimeout(ts.ctx, 5*time.Second)
	defer cancel()
	started := make(chan struct{}, 32)
	results := make(chan error, 32)
	for range 32 {
		go func() {
			req := newExportRequest(data)
			started <- struct{}{}
			_, err := srvLog.Export(ctx, req)
			results <- err
		}()
	}
	for range 32 {
		<-started
	}

	processCtx, stop := context.WithCancel(ts.ctx)
	stop()
	done := make(chan struct{})
	go func() {
		defer close(done)
		srvLog.startProcessLoop(processCtx)
	}()

	for range 32 {
		select {
		case err := <-results:
			assert.Equal(t, codes.Unavailable, status.Code(err))
		case <-ctx.Done():
			t.Fatal("Export did not return during shutdown")
		}
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("Log processor did not stop")
	}
	assert.Empty(t, srvLog.batchCh)
	for _, batch := range batches {
		select {
		case err := <-batch.done:
			assert.NoError(t, err)
		default:
			t.Fatal("Accepted batch was not acknowledged")
		}
	}
	assert.Equal(t, logQueueSize, getTableCount(t, ts.srv, "component_logs"))

	_, err := srvLog.Export(ctx, newExportRequest(data))
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestOTLPProcessorShutdownTimeout(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	ts.srv.db.SetMaxOpenConns(1)
	conn, err := ts.srv.db.Conn(ts.ctx)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })
	srvLog := ts.srv.newSrvLog()
	batch := &pendingLogBatch{
		items: []pendingLog{{table: logTableComponent, data: marshalLog(t, newComponentLog(time.Now().UTC(), corev1.ComponentLog_Entry_INFO, "queued"))}},
		done:  make(chan error, 1),
	}
	srvLog.batchCh <- batch
	ctx, cancel := context.WithCancel(ts.ctx)
	cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		srvLog.startProcessLoop(ctx)
	}()

	select {
	case <-done:
	case <-time.After(logShutdownTimeout + 2*time.Second):
		t.Fatal("Log processor exceeded its shutdown timeout")
	}
	select {
	case err := <-batch.done:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	default:
		t.Fatal("Timed out batch was not acknowledged")
	}
	assert.Empty(t, srvLog.batchCh)
}
