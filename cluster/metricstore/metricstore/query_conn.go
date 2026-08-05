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
	"database/sql/driver"
	"fmt"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type querySeriesMapping struct {
	sourceID string
	outputID string
}

func (s *srvMetric) withSeriesMapping(ctx context.Context, mappings []querySeriesMapping,
	fn func(*sql.Conn) error) error {
	conn, err := s.s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `
CREATE OR REPLACE TEMP TABLE selected_metric_series (
	series_id VARCHAR,
	output_id VARCHAR
)
`); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `DROP TABLE IF EXISTS temp.main.selected_metric_series`)
	}()

	if err := conn.Raw(func(raw any) error {
		driverConn, ok := raw.(driver.Conn)
		if !ok {
			return fmt.Errorf("unexpected DuckDB driver connection type: %T", raw)
		}
		appender, err := duckdb.NewAppenderFromConn(driverConn, "", "selected_metric_series")
		if err != nil {
			return err
		}
		for _, mapping := range mappings {
			if err := appender.AppendRow(mapping.sourceID, mapping.outputID); err != nil {
				_ = appender.Close()
				return err
			}
		}
		return appender.Close()
	}); err != nil {
		return err
	}

	return fn(conn)
}

func ensureRawRowLimit(ctx context.Context, conn *sql.Conn, table string, query *querySpec,
	includePrevious bool, limit int) error {
	from := query.from
	if includePrevious {
		from = time.Unix(0, 0).UTC()
	}

	var count int64
	statement := `SELECT COUNT(*)
FROM ` + table + ` p
JOIN selected_metric_series m ON m.series_id = p.series_id
WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?`
	if err := conn.QueryRowContext(ctx, statement, metricTimeToDB(from), metricTimeToDB(query.to),
		metricTimeToDB(query.snapshot)).Scan(&count); err != nil {
		return err
	}
	if count > int64(limit) {
		return status.Error(codes.ResourceExhausted, "metric query scans too many raw points")
	}
	return nil
}
