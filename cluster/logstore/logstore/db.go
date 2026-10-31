package logstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/mod/semver"
)

func openLogStoreDB(ctx context.Context, dsn string) (*sql.DB, error) {
	database, rawQuery, _ := strings.Cut(dsn, "?")
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return nil, err
	}
	values.Set("storage_compatibility_version", "v2.0.0")
	dsn = database + "?" + values.Encode()

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, err
	}
	if database == "" || database == ":memory:" {
		return db, nil
	}

	var storageVersion string
	if err := db.QueryRowContext(ctx, `
SELECT tags['storage_version'] FROM duckdb_databases() WHERE database_name = current_database()
`).Scan(&storageVersion); err != nil {
		_ = db.Close()
		return nil, err
	}
	version := strings.TrimSuffix(storageVersion, "+")
	if !semver.IsValid(version) {
		_ = db.Close()
		return nil, fmt.Errorf("cannot reset logstore storage with an unrecognized storage version: %s", storageVersion)
	}
	if semver.Compare(version, "v2.0.0") >= 0 {
		return db, nil
	}

	var tables, logTables, logColumns, schemas int
	if err := db.QueryRowContext(ctx, `SELECT
	(SELECT COUNT(*) FROM information_schema.tables WHERE table_catalog = current_database()),
	(SELECT COUNT(*) FROM information_schema.tables WHERE table_catalog = current_database()
	 AND table_schema = 'main' AND table_type = 'BASE TABLE'
	 AND table_name IN ('access_logs', 'component_logs', 'authentication_logs', 'audit_logs')),
	(SELECT COUNT(*) FROM information_schema.columns WHERE table_catalog = current_database()
	 AND table_schema = 'main' AND table_name IN ('access_logs', 'component_logs', 'authentication_logs', 'audit_logs')
	 AND column_name = 'rsc' AND data_type = 'JSON'),
	(SELECT COUNT(*) FROM information_schema.schemata WHERE catalog_name = current_database()
	 AND schema_name NOT IN ('main', 'information_schema', 'pg_catalog'))
`).Scan(&tables, &logTables, &logColumns, &schemas); err != nil {
		_ = db.Close()
		return nil, err
	}
	if tables != logTables || logTables != logColumns || schemas != 0 {
		_ = db.Close()
		return nil, fmt.Errorf("cannot reset a database that does not contain logstore storage: %s", database)
	}
	if err := db.Close(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	zap.L().Warn("Resetting logstore storage to use the DuckDB v2 format",
		zap.String("database", database), zap.String("storageVersion", storageVersion))
	if err := removeLogStoreDatabase(database); err != nil {
		return nil, err
	}
	return sql.Open("duckdb", dsn)
}

func removeLogStoreDatabase(database string) error {
	if database == "" {
		return nil
	}
	if err := os.Remove(database + ".wal"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.RemoveAll(database + ".tmp"); err != nil {
		return err
	}
	if err := os.Remove(database); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
