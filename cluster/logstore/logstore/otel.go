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
	"encoding/json"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
)

const (
	logQueueSize       = 256
	logBatchSize       = 500
	logBatchWait       = 100 * time.Millisecond
	logEnqueueTimeout  = 2 * time.Second
	logShutdownTimeout = 10 * time.Second
	maxLogsPerExport   = 5000
	maxLogRecordSize   = 256 * 1024
	maxAccessLogSize   = 25000
)

type logTable uint8

const (
	logTableUnknown logTable = iota
	logTableAccess
	logTableComponent
	logTableAuthentication
	logTableAudit
)

type pendingLog struct {
	table logTable
	data  []byte
}

type pendingLogBatch struct {
	items []pendingLog
	done  chan error
}

type srvLog struct {
	plogotlp.UnimplementedGRPCServer
	s       *Server
	batchCh chan *pendingLogBatch
}

func parseLogRecord(lr plog.LogRecord) (*pendingLog, error) {
	var body []byte

	switch lr.Body().Type() {
	case pcommon.ValueTypeStr:
		body = []byte(lr.Body().AsString())

	case pcommon.ValueTypeMap:
		var err error
		body, err = json.Marshal(lr.Body().Map().AsRaw())
		if err != nil {
			return nil, errors.Wrap(err, "Could not marshal OTLP map log body")
		}

	default:
		return nil, errors.Errorf("Unsupported OTLP log body type: %s", lr.Body().Type())
	}

	if len(body) == 0 {
		return nil, errors.Errorf("Empty OTLP log body")
	}

	if len(body) > maxLogRecordSize {
		return nil, errors.Errorf("OTLP log body is too large: %d", len(body))
	}

	var envelope struct {
		Kind  string `json:"kind"`
		Entry struct {
			Level string `json:"level"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, errors.Wrap(
			err,
			"Could not unmarshal OTLP log body",
		)
	}

	var table logTable

	switch envelope.Kind {
	case ucorev1.KindAccessLog:
		if len(body) > maxAccessLogSize {
			return nil, errors.Errorf("AccessLog is too large: %d", len(body))
		}
		table = logTableAccess

	case ucorev1.KindComponentLog:
		if ldflags.IsProduction() &&
			envelope.Entry.Level == corev1.ComponentLog_Entry_DEBUG.String() {
			return nil, nil
		}
		table = logTableComponent

	case uenterprisev1.KindAuthenticationLog:
		table = logTableAuthentication

	case uenterprisev1.KindAuditLog:
		table = logTableAudit

	default:
		return nil, errors.Errorf("Unsupported log kind: %s", envelope.Kind)
	}

	return &pendingLog{
		table: table,
		data:  append([]byte(nil), body...),
	}, nil
}

func (s *srvLog) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	var items []pendingLog

	for i := range req.Logs().ResourceLogs().Len() {
		scopeLogs := req.Logs().ResourceLogs().At(i).ScopeLogs()
		for j := range scopeLogs.Len() {
			logRecords := scopeLogs.At(j).LogRecords()
			for k := range logRecords.Len() {
				if len(items) >= maxLogsPerExport {
					return plogotlp.NewExportResponse(), errors.Errorf("Too many logs in one export request")
				}

				item, err := parseLogRecord(logRecords.At(k))
				if err != nil {
					return plogotlp.NewExportResponse(), err
				}

				if item == nil {
					continue
				}

				items = append(items, *item)
			}
		}
	}

	if len(items) == 0 {
		return plogotlp.NewExportResponse(), nil
	}

	batch := &pendingLogBatch{
		items: items,
		done:  make(chan error, 1),
	}

	timer := time.NewTimer(logEnqueueTimeout)
	defer timer.Stop()

	select {
	case s.batchCh <- batch:
	case <-ctx.Done():
		return plogotlp.NewExportResponse(), ctx.Err()
	case <-timer.C:
		return plogotlp.NewExportResponse(), errors.Errorf("LogStore queue is full")
	}

	select {
	case err := <-batch.done:
		if err != nil {
			return plogotlp.NewExportResponse(), err
		}
		return plogotlp.NewExportResponse(), nil

	case <-ctx.Done():
		return plogotlp.NewExportResponse(), ctx.Err()
	}
}

/*
func (s *srvLog) processLogRecord(lr plog.LogRecord) {

	bodyJSONMap := make(map[string]any)
	var bodyStr string

	switch lr.Body().Type() {
	case pcommon.ValueTypeStr:
		bodyStr = lr.Body().AsString()
		if err := json.Unmarshal([]byte(bodyStr), &bodyJSONMap); err != nil {
			zap.L().Debug("Could not unmarshal JSON log body", zap.Error(err), zap.Any("map", bodyJSONMap))
			return
		}

	case pcommon.ValueTypeMap:
		raw := lr.Body().Map().AsRaw()
		var err error
		bodyJSON, err := json.Marshal(raw)
		if err != nil {
			zap.L().Debug("Could not marshal OTLP map log body", zap.Error(err))
			return
		}

		bodyStr = string(bodyJSON)

		if err := json.Unmarshal(bodyJSON, &bodyJSONMap); err != nil {
			zap.L().Debug("Could not unmarshal JSON log body", zap.Error(err), zap.Any("map", bodyJSONMap))
			return
		}

	default:
		zap.L().Debug("Unknown log body type", zap.Any("type", lr.Body().Type()))
		return
	}

	kind, _ := bodyJSONMap["kind"].(string)

	switch kind {
	case ucorev1.KindAccessLog:
		// zap.L().Debug("Inserting accessLog", zap.String("log", string(bodyStr)))
		if err := s.s.insertAccessLog([]byte(bodyStr)); err != nil {
			zap.L().Warn("Could not insertAccessLog", zap.Error(err))
		}
	case ucorev1.KindComponentLog:
		// zap.L().Debug("Inserting componentLog", zap.String("log", string(bodyStr)))
		if err := s.s.insertComponentLog([]byte(bodyStr)); err != nil {
			zap.L().Warn("Could not insertComponentLog", zap.Error(err))
		}
	case uenterprisev1.KindAuthenticationLog:
		// zap.L().Debug("Inserting authenticationLog", zap.String("log", string(bodyStr)))
		if err := s.s.insertAuthenticationLog([]byte(bodyStr)); err != nil {
			zap.L().Warn("Could not insertAuthenticationLog", zap.Error(err))
		}
	case uenterprisev1.KindAuditLog:
		// zap.L().Debug("Inserting auditLog", zap.String("log", string(bodyStr)))
		if err := s.s.insertAuditLog([]byte(bodyStr)); err != nil {
			zap.L().Warn("Could not insertAuditLog", zap.Error(err))
		}
	default:
		zap.L().Debug("Unknown log type. Skipping inserting....",
			zap.String("kind", kind))
	}

}
*/

func (s *srvLog) startProcessLoop(ctx context.Context) {
	defer zap.L().Debug("Exiting LogStore process loop")

	ticker := time.NewTicker(logBatchWait)
	defer ticker.Stop()

	var pending []*pendingLogBatch
	var itemCount int

	flush := func(flushCtx context.Context) {
		if len(pending) == 0 {
			return
		}

		batches := pending
		pending = nil
		itemCount = 0

		err := s.s.insertLogBatches(flushCtx, batches)

		for _, batch := range batches {
			batch.done <- err
		}
	}

	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), logShutdownTimeout)
			flush(flushCtx)
			cancel()
			return

		case batch := <-s.batchCh:
			pending = append(pending, batch)
			itemCount += len(batch.items)

			if itemCount >= logBatchSize {
				flush(ctx)
			}

		case <-ticker.C:
			flush(ctx)
		}
	}
}

func (s *Server) newSrvLog() *srvLog {
	return &srvLog{
		s:       s,
		batchCh: make(chan *pendingLogBatch, logQueueSize),
	}
}
