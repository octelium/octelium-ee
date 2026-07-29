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
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
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
