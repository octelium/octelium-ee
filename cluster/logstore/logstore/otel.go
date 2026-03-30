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

	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
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
				s.itemCh <- lr
			}
		}
	}

	return plogotlp.NewExportResponse(), nil
}

func (s *srvLog) processLogRecord(lr plog.LogRecord) {

	switch lr.Body().Type() {
	case pcommon.ValueTypeMap:
		// lrMap := lr.Body().Map().AsRaw()
		// zap.L().Debug("New LogMap", zap.Any("log", lrMap))
	case pcommon.ValueTypeStr:
		// zap.L().Debug("New LogString", zap.Any("log", lr.Body().AsString()))
	default:
		zap.L().Debug("Unknown log type skipping", zap.Any("typ", lr.Body().Type()))
		return
	}

	bodyJSONMap := make(map[string]any)
	bodyStr := lr.Body().AsString()
	if err := json.Unmarshal([]byte(bodyStr), &bodyJSONMap); err != nil {
		zap.L().Debug("Could not unmarshal JSON log body", zap.Error(err), zap.Any("map", bodyJSONMap))
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
