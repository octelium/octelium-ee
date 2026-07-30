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
	"runtime"
	"time"

	"go.uber.org/zap"
)

const diagnosticsInterval = time.Minute

type queueBudgetSnapshot struct {
	requests int64
	points   int64
	bytes    int64
}

func (b *queueBudget) snapshot() queueBudgetSnapshot {
	b.mu.Lock()
	defer b.mu.Unlock()

	return queueBudgetSnapshot{
		requests: b.requests,
		points:   b.points,
		bytes:    b.bytes,
	}
}

func (s *Server) runDiagnosticsLoop(ctx context.Context) {
	ticker := time.NewTicker(diagnosticsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.logDiagnostics(ctx)
		}
	}
}

func (s *Server) logDiagnostics(ctx context.Context) {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)

	fields := []zap.Field{
		zap.Uint64("goHeapAllocBytes", memory.HeapAlloc),
		zap.Uint64("goHeapInuseBytes", memory.HeapInuse),
		zap.Uint64("goHeapSysBytes", memory.HeapSys),
		zap.Int("goroutines", runtime.NumGoroutine()),
	}

	if s.metricSrv != nil {
		queue := s.metricSrv.budget.snapshot()
		fields = append(fields,
			zap.Int64("queuedExports", queue.requests),
			zap.Int64("queuedDataPoints", queue.points),
			zap.Int64("queuedEstimatedBytes", queue.bytes),
			zap.Uint64("acceptedExports", s.metricSrv.acceptedExports.Load()),
			zap.Uint64("rejectedExports", s.metricSrv.rejectedExports.Load()),
			zap.Uint64("storedDataPoints", s.metricSrv.storedPoints.Load()),
		)
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT tag, memory_usage_bytes, temporary_storage_bytes
FROM duckdb_memory()
WHERE memory_usage_bytes != 0 OR temporary_storage_bytes != 0
ORDER BY memory_usage_bytes DESC
`)
	if err != nil {
		zap.L().Debug("Could not read DuckDB memory diagnostics", zap.Error(err))
		zap.L().Info("MetricStore diagnostics", fields...)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tag string
		var memoryUsage int64
		var temporaryStorage int64
		if err := rows.Scan(&tag, &memoryUsage, &temporaryStorage); err != nil {
			zap.L().Debug("Could not scan DuckDB memory diagnostics", zap.Error(err))
			break
		}
		fields = append(fields,
			zap.Int64("duckdb."+tag+".memoryBytes", memoryUsage),
			zap.Int64("duckdb."+tag+".temporaryBytes", temporaryStorage),
		)
	}

	zap.L().Info("MetricStore diagnostics", fields...)
}
