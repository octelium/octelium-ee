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
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/pkg/errors"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
)

type srvLog struct {
	plogotlp.UnimplementedGRPCServer
	s      *Server
	itemCh chan plog.LogRecord
}

func (s *srvLog) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {

	for i := range req.Logs().ResourceLogs().Len() {
		scopeLogs := req.Logs().ResourceLogs().At(i).ScopeLogs()
		for j := range scopeLogs.Len() {
			logRecords := scopeLogs.At(j).LogRecords()
			for k := range logRecords.Len() {
				lr := logRecords.At(k)
				destLr := plog.NewLogRecord()
				lr.CopyTo(destLr)
				select {
				case s.itemCh <- destLr:
				case <-ctx.Done():
					return plogotlp.NewExportResponse(), ctx.Err()
				case <-time.After(2 * time.Second):
					return plogotlp.NewExportResponse(), errors.Errorf("Logstore itemCh queue is full")
				}
			}
		}
	}

	return plogotlp.NewExportResponse(), nil
}

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
			zap.L().Warn("Could not insertAccessLog", zap.Error(err), zap.String("rsc", bodyStr))
		}
	case ucorev1.KindComponentLog:
		// zap.L().Debug("Inserting componentLog", zap.String("log", string(bodyStr)))
		if err := s.s.insertComponentLog([]byte(bodyStr)); err != nil {
			zap.L().Warn("Could not insertComponentLog", zap.Error(err), zap.String("rsc", bodyStr))
		}
	case uenterprisev1.KindAuthenticationLog:
		// zap.L().Debug("Inserting authenticationLog", zap.String("log", string(bodyStr)))
		if err := s.s.insertAuthenticationLog([]byte(bodyStr)); err != nil {
			zap.L().Warn("Could not insertAuthenticationLog", zap.Error(err), zap.String("rsc", bodyStr))
		}
	case uenterprisev1.KindAuditLog:
		// zap.L().Debug("Inserting auditLog", zap.String("log", string(bodyStr)))
		if err := s.s.insertAuditLog([]byte(bodyStr)); err != nil {
			zap.L().Warn("Could not insertAuditLog", zap.Error(err), zap.String("rsc", bodyStr))
		}
	default:
		zap.L().Debug("Unknown log type. Skipping inserting....",
			zap.String("kind", kind), zap.String("rsc", bodyStr))
	}

}

func (s *srvLog) startProcessLoop(ctx context.Context) {
	defer zap.L().Debug("Exiting process loop")

	for {
		select {
		case <-ctx.Done():
			return
		case lr := <-s.itemCh:
			s.processLogRecord(lr)
		}
	}
}

func (s *Server) newSrvLog() *srvLog {
	return &srvLog{
		s:      s,
		itemCh: make(chan plog.LogRecord, 10000),
	}
}
