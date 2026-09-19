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
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const logBackfillBatchSize = 20000

const (
	colCreatedAt    = "created_at"
	colStatus       = "status"
	colLevel        = "level"
	colUserUID      = "user_uid"
	colDeviceUID    = "device_uid"
	colSessionUID   = "session_uid"
	colServiceUID   = "service_uid"
	colNamespaceUID = "namespace_uid"
	colRegionUID    = "region_uid"
	colPolicyUID    = "policy_uid"
)

type logColumn struct {
	name string
	kind string
	path string
}

func (c *logColumn) sourceExpr(source string) string {
	if c.kind == "TIMESTAMP" {
		return fmt.Sprintf(`TRY_CAST(json_extract_string(%s, '%s') AS TIMESTAMP)`, source, c.path)
	}

	return fmt.Sprintf(`json_extract_string(%s, '%s')`, source, c.path)
}

func getLogTables() []string {
	return []string{"access_logs", "component_logs", "authentication_logs", "audit_logs"}
}

func getLogTableName(table logTable) (string, error) {
	switch table {
	case logTableAccess:
		return "access_logs", nil
	case logTableComponent:
		return "component_logs", nil
	case logTableAuthentication:
		return "authentication_logs", nil
	case logTableAudit:
		return "audit_logs", nil
	default:
		return "", errors.Errorf("Invalid log table: %d", table)
	}
}

func getLogTableColumns(table string) []*logColumn {
	ret := []*logColumn{
		{name: colCreatedAt, kind: "TIMESTAMP", path: "$.metadata.createdAt"},
	}

	switch table {
	case "access_logs":
		ret = append(ret,
			&logColumn{name: colStatus, kind: "VARCHAR", path: "$.entry.common.status"},
			&logColumn{name: colUserUID, kind: "VARCHAR", path: "$.entry.common.userRef.uid"},
			&logColumn{name: colDeviceUID, kind: "VARCHAR", path: "$.entry.common.deviceRef.uid"},
			&logColumn{name: colSessionUID, kind: "VARCHAR", path: "$.entry.common.sessionRef.uid"},
			&logColumn{name: colServiceUID, kind: "VARCHAR", path: "$.entry.common.serviceRef.uid"},
			&logColumn{name: colNamespaceUID, kind: "VARCHAR", path: "$.entry.common.namespaceRef.uid"},
			&logColumn{name: colRegionUID, kind: "VARCHAR", path: "$.entry.common.regionRef.uid"},
			&logColumn{name: colPolicyUID, kind: "VARCHAR",
				path: "$.entry.common.reason.details.policyMatch.policy.policyRef.uid"},
		)
	case "component_logs":
		ret = append(ret,
			&logColumn{name: colLevel, kind: "VARCHAR", path: "$.entry.level"},
		)
	}

	return ret
}

func getLogInsertQuery(table logTable) (string, error) {
	name, err := getLogTableName(table)
	if err != nil {
		return "", err
	}

	return getLogTableInsertQuery(name), nil
}

func getLogTableInsertQuery(table string) string {
	columns := getLogTableColumns(table)

	names := make([]string, 0, len(columns)+1)
	values := make([]string, 0, len(columns)+1)

	names = append(names, "rsc")
	values = append(values, "CAST($1 AS JSON)")

	for _, column := range columns {
		names = append(names, column.name)
		values = append(values, column.sourceExpr("$1"))
	}

	return fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`,
		table, strings.Join(names, ", "), strings.Join(values, ", "))
}

func (s *Server) initLogColumns(ctx context.Context) error {
	for _, table := range getLogTables() {
		columns := getLogTableColumns(table)

		for _, column := range columns {
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf(
				`ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s`, table, column.name, column.kind)); err != nil {
				return err
			}
		}

		if err := s.backfillLogColumns(ctx, table, columns); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) backfillLogColumns(ctx context.Context, table string, columns []*logColumn) error {
	assignments := make([]string, 0, len(columns))
	for _, column := range columns {
		assignments = append(assignments, fmt.Sprintf("%s = %s", column.name, column.sourceExpr("rsc")))
	}

	remaining := int64(-1)
	total := int64(0)

	for {
		var pending int64
		if err := s.db.QueryRowContext(ctx, fmt.Sprintf(
			`SELECT COUNT(*) FROM %s WHERE %s IS NULL`, table, colCreatedAt)).Scan(&pending); err != nil {
			return err
		}

		if pending == 0 || (remaining >= 0 && pending >= remaining) {
			break
		}

		remaining = pending

		result, err := s.db.ExecContext(ctx, fmt.Sprintf(`
UPDATE %s SET %s WHERE rowid IN (
	SELECT rowid FROM %s WHERE %s IS NULL ORDER BY rowid LIMIT %d
)`, table, strings.Join(assignments, ", "), table, colCreatedAt, logBackfillBatchSize))
		if err != nil {
			return err
		}
		if count, err := result.RowsAffected(); err == nil {
			total += count
		}

		if err := s.checkpointStorage(ctx); err != nil {
			return err
		}
	}

	if total > 0 {
		zap.L().Info("Backfilled the LogStore indexed columns",
			zap.String("table", table), zap.Int64("logs", total))
	}

	return nil
}
