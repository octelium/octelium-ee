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
	"time"

	"go.uber.org/zap"
)

const defaultRetention = 30 * 24 * time.Hour

func (s *Server) initDB(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS metrics (
    timestamp TIMESTAMP NOT NULL,

    name TEXT NOT NULL,
    unit TEXT,
    description TEXT,

    -- Visibility API kind:
    -- COUNTER, UP_DOWN_COUNTER, GAUGE, HISTOGRAM, EXPONENTIAL_HISTOGRAM
    kind TEXT NOT NULL,

    -- INT64 or DOUBLE
    value_type TEXT,

    -- DELTA, CUMULATIVE, or empty
    temporality TEXT,

    resource JSON,
    scope JSON,
    attributes JSON,

    component_type TEXT,
    component_namespace TEXT,
    component_name TEXT,

    -- Number points
    number_int BIGINT,
    number_double DOUBLE,

    -- Explicit histogram
    histogram_count UBIGINT,
    histogram_has_sum BOOLEAN,
    histogram_sum DOUBLE,
    histogram_min DOUBLE,
    histogram_max DOUBLE,
    histogram_bounds JSON,
    histogram_bucket_counts JSON,

    -- Exponential histogram
    exp_count UBIGINT,
    exp_has_sum BOOLEAN,
    exp_sum DOUBLE,
    exp_min DOUBLE,
    exp_max DOUBLE,
    exp_scale INTEGER,
    exp_zero_count UBIGINT,
    exp_zero_threshold DOUBLE,
    exp_positive_offset INTEGER,
    exp_positive_counts JSON,
    exp_negative_offset INTEGER,
    exp_negative_counts JSON
)
`)
	if err != nil {
		return err
	}

	stmts := []string{
		`CREATE INDEX IF NOT EXISTS idx_metrics_name_ts ON metrics(name, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_kind_ts ON metrics(kind, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_component ON metrics(component_type, component_namespace, component_name)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) runRetentionLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().Add(-defaultRetention)

			res, err := s.db.ExecContext(ctx, `DELETE FROM metrics WHERE timestamp < ?`, cutoff)
			if err != nil {
				zap.L().Warn("Could not apply metric retention", zap.Error(err))
				continue
			}

			if n, err := res.RowsAffected(); err == nil && n > 0 {
				if _, err := s.db.ExecContext(ctx, `CHECKPOINT`); err != nil {
					zap.L().Warn("Could not checkpoint metricstore after retention", zap.Error(err))
				}
			}
		}
	}
}