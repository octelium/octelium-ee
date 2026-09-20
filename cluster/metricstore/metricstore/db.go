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
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"go.uber.org/zap"
)

const (
	metricStoreSchemaVersion = 4
	rawMetricRetention       = 48 * time.Hour
	retentionInterval        = 30 * time.Minute
	retentionDeleteChunkSize = 50000
	databaseRecoveryInterval = 5 * time.Second
)

var metricPointTables = []string{
	"metric_number_points",
	"metric_histogram_points",
	"metric_exponential_histogram_points",
}

func (s *Server) initDB(ctx context.Context) error {
	version, err := s.getSchemaVersion(ctx)
	if err != nil {
		return err
	}

	if version != metricStoreSchemaVersion {
		if err := s.resetOutdatedStorage(ctx, version); err != nil {
			return err
		}
	}

	return s.createSchema(ctx)
}

func (s *Server) createSchema(ctx context.Context) error {
	tx, err := s.database().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := createMetricStoreTables(ctx, tx); err != nil {
		return err
	}
	if err := setMetricStoreSchemaVersion(ctx, tx, metricStoreSchemaVersion); err != nil {
		return err
	}

	return tx.Commit()
}

func setMetricStoreSchemaVersion(ctx context.Context, tx *sql.Tx, version int) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO metricstore_schema (version, applied_at) VALUES (?, ?)
ON CONFLICT (version) DO UPDATE SET applied_at = EXCLUDED.applied_at
`, version, metricTimeToDB(time.Now()))
	return err
}

func (s *Server) resetOutdatedStorage(ctx context.Context, version int) error {
	tables, err := s.countStoredTables(ctx)
	if err != nil {
		return err
	}
	if tables == 0 {
		return nil
	}

	zap.L().Warn("The metricstore database does not use the current schema. Resetting its storage",
		zap.Int("databaseSchemaVersion", version),
		zap.Int("schemaVersion", metricStoreSchemaVersion))

	if db := s.database(); db != nil {
		if err := db.Close(); err != nil {
			return err
		}
		s.setDatabase(nil)
	}

	if err := removeMetricStoreDatabase(s.dbConfig.database); err != nil {
		return err
	}

	db, err := openMetricStoreDB(s.dbConfig)
	if err != nil {
		return err
	}
	s.setDatabase(db)

	return db.PingContext(ctx)
}

func removeMetricStoreDatabase(database string) error {
	if database == "" {
		return nil
	}

	for _, path := range []string{database, database + ".wal"} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return os.RemoveAll(database + ".tmp")
}

func (s *Server) countStoredTables(ctx context.Context) (int, error) {
	var count int
	err := s.database().QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'main'
`).Scan(&count)
	return count, err
}

func (s *Server) getSchemaVersion(ctx context.Context) (int, error) {
	var stored int
	if err := s.database().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'main' AND table_name = 'metricstore_schema'
`).Scan(&stored); err != nil {
		return 0, err
	}
	if stored == 0 {
		return 0, nil
	}

	var version sql.NullInt64
	if err := s.database().QueryRowContext(ctx,
		`SELECT MAX(version) FROM metricstore_schema`).Scan(&version); err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}

	return int(version.Int64), nil
}

func createMetricStoreTables(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS metricstore_schema (
			version INTEGER PRIMARY KEY,
			applied_at BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_descriptors (
			id VARCHAR PRIMARY KEY,
			name VARCHAR NOT NULL,
			kind VARCHAR NOT NULL,
			number_value_type VARCHAR NOT NULL,
			unit VARCHAR NOT NULL,
			description VARCHAR NOT NULL,
			temporality VARCHAR NOT NULL,
			scope_name VARCHAR NOT NULL,
			scope_version VARCHAR NOT NULL,
			scope_schema_url VARCHAR NOT NULL,
			explicit_bounds JSON,
			exp_min_scale INTEGER,
			exp_max_scale INTEGER,
			exp_zero_threshold_min DOUBLE,
			exp_zero_threshold_max DOUBLE,
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_series (
			id VARCHAR PRIMARY KEY,
			descriptor_id VARCHAR NOT NULL,
			labels JSON NOT NULL,
			labels_key VARCHAR NOT NULL,
			component_type VARCHAR NOT NULL,
			component_namespace VARCHAR NOT NULL,
			component_name VARCHAR NOT NULL,
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_series_attributes (
			series_id VARCHAR NOT NULL,
			descriptor_id VARCHAR NOT NULL,
			key VARCHAR NOT NULL,
			value_kind VARCHAR NOT NULL,
			value_key VARCHAR NOT NULL,
			value_string VARCHAR,
			value_bool BOOLEAN,
			value_int BIGINT,
			value_double DOUBLE,
			source_mask INTEGER NOT NULL,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY(series_id, key)
		)`,
		`CREATE TABLE IF NOT EXISTS metric_attribute_keys (
			descriptor_id VARCHAR NOT NULL,
			key VARCHAR NOT NULL,
			value_kind VARCHAR NOT NULL,
			source_mask INTEGER NOT NULL,
			first_seen_at BIGINT NOT NULL,
			last_seen_at BIGINT NOT NULL,
			PRIMARY KEY(descriptor_id, key)
		)`,
		`CREATE TABLE IF NOT EXISTS metric_number_points (
			point_id VARCHAR NOT NULL,
			timestamp BIGINT NOT NULL,
			ingested_at BIGINT NOT NULL,
			start_timestamp BIGINT,
			series_id VARCHAR NOT NULL,
			number_int BIGINT,
			number_double DOUBLE
		)`,
		`CREATE TABLE IF NOT EXISTS metric_histogram_points (
			point_id VARCHAR NOT NULL,
			timestamp BIGINT NOT NULL,
			ingested_at BIGINT NOT NULL,
			start_timestamp BIGINT,
			series_id VARCHAR NOT NULL,
			count UBIGINT NOT NULL,
			has_sum BOOLEAN NOT NULL,
			sum DOUBLE,
			min DOUBLE,
			max DOUBLE,
			bucket_counts JSON NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_exponential_histogram_points (
			point_id VARCHAR NOT NULL,
			timestamp BIGINT NOT NULL,
			ingested_at BIGINT NOT NULL,
			start_timestamp BIGINT,
			series_id VARCHAR NOT NULL,
			count UBIGINT NOT NULL,
			has_sum BOOLEAN NOT NULL,
			sum DOUBLE,
			min DOUBLE,
			max DOUBLE,
			scale INTEGER NOT NULL,
			zero_count UBIGINT NOT NULL,
			zero_threshold DOUBLE NOT NULL,
			positive_offset INTEGER NOT NULL,
			positive_counts JSON NOT NULL,
			negative_offset INTEGER NOT NULL,
			negative_counts JSON NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_number_staging (
			point_id VARCHAR NOT NULL,
			timestamp BIGINT NOT NULL,
			ingested_at BIGINT NOT NULL,
			start_timestamp BIGINT,
			series_id VARCHAR NOT NULL,
			number_int BIGINT,
			number_double DOUBLE
		)`,
		`CREATE TABLE IF NOT EXISTS metric_descriptors_staging (
			id VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			kind VARCHAR NOT NULL,
			number_value_type VARCHAR NOT NULL,
			unit VARCHAR NOT NULL,
			description VARCHAR NOT NULL,
			temporality VARCHAR NOT NULL,
			scope_name VARCHAR NOT NULL,
			scope_version VARCHAR NOT NULL,
			scope_schema_url VARCHAR NOT NULL,
			explicit_bounds VARCHAR,
			exp_min_scale INTEGER,
			exp_max_scale INTEGER,
			exp_zero_threshold_min DOUBLE,
			exp_zero_threshold_max DOUBLE,
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_series_staging (
			id VARCHAR NOT NULL,
			descriptor_id VARCHAR NOT NULL,
			labels VARCHAR NOT NULL,
			labels_key VARCHAR NOT NULL,
			component_type VARCHAR NOT NULL,
			component_namespace VARCHAR NOT NULL,
			component_name VARCHAR NOT NULL,
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_series_attributes_staging (
			series_id VARCHAR NOT NULL,
			descriptor_id VARCHAR NOT NULL,
			key VARCHAR NOT NULL,
			value_kind VARCHAR NOT NULL,
			value_key VARCHAR NOT NULL,
			value_string VARCHAR,
			value_bool BOOLEAN,
			value_int BIGINT,
			value_double DOUBLE,
			source_mask INTEGER NOT NULL,
			updated_at BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_attribute_keys_staging (
			descriptor_id VARCHAR NOT NULL,
			key VARCHAR NOT NULL,
			value_kind VARCHAR NOT NULL,
			source_mask INTEGER NOT NULL,
			first_seen_at BIGINT NOT NULL,
			last_seen_at BIGINT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS metric_histogram_staging (
			point_id VARCHAR NOT NULL,
			timestamp BIGINT NOT NULL,
			ingested_at BIGINT NOT NULL,
			start_timestamp BIGINT,
			series_id VARCHAR NOT NULL,
			count UBIGINT NOT NULL,
			has_sum BOOLEAN NOT NULL,
			sum DOUBLE,
			min DOUBLE,
			max DOUBLE,
			bucket_counts VARCHAR NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS metric_exponential_histogram_staging (
			point_id VARCHAR NOT NULL,
			timestamp BIGINT NOT NULL,
			ingested_at BIGINT NOT NULL,
			start_timestamp BIGINT,
			series_id VARCHAR NOT NULL,
			count UBIGINT NOT NULL,
			has_sum BOOLEAN NOT NULL,
			sum DOUBLE,
			min DOUBLE,
			max DOUBLE,
			scale INTEGER NOT NULL,
			zero_count UBIGINT NOT NULL,
			zero_threshold DOUBLE NOT NULL,
			positive_offset INTEGER NOT NULL,
			positive_counts VARCHAR NOT NULL,
			negative_offset INTEGER NOT NULL,
			negative_counts VARCHAR NOT NULL
		)`,

		`CREATE INDEX IF NOT EXISTS metric_series_descriptor_id
			ON metric_series (descriptor_id)`,
	}

	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) runRetentionLoop(ctx context.Context) {
	ticker := time.NewTicker(retentionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := s.applyRetention(ctx); err != nil {
				zap.L().Warn("Could not apply metricstore retention", zap.Error(err))
				s.requestDatabaseRecovery(err)
			}
		}
	}
}

func (s *Server) requestDatabaseRecovery(err error) {
	if !ovutils.IsInvalidatedDatabaseErr(err) {
		return
	}

	select {
	case s.dbRecoverCh <- struct{}{}:
	default:
	}
}

func (s *Server) runDatabaseRecoveryLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-s.dbRecoverCh:
			s.recoverDatabase(ctx)
		}
	}
}

func (s *Server) recoverDatabase(ctx context.Context) {
	db := s.database()
	if db == nil || !ovutils.IsInvalidatedDatabaseErr(db.PingContext(ctx)) {
		return
	}

	zap.L().Warn("The metricstore database has been invalidated by a fatal error. Reopening it")

	s.reopenDatabase(ctx)
}

func (s *Server) reopenDatabase(ctx context.Context) {
	if db := s.database(); db != nil {
		_ = db.Close()
	}

	for {
		db, err := openMetricStoreDB(s.dbConfig)
		if err == nil {
			if err = db.PingContext(ctx); err == nil {
				s.setDatabase(db)
				zap.L().Info("Reopened the metricstore database")
				return
			}
			_ = db.Close()
		}

		zap.L().Warn("Could not reopen the metricstore database", zap.Error(err))

		select {
		case <-ctx.Done():
			return
		case <-time.After(databaseRecoveryInterval):
		}
	}
}

func (s *Server) applyRetention(ctx context.Context) error {
	s.retentionMu.Lock()
	defer s.retentionMu.Unlock()

	cutoff := time.Now().UTC().Add(-(rawMetricRetention + retentionInterval))

	deleted, err := s.deleteMetricPointsBefore(ctx, cutoff)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return nil
	}

	if err := s.deleteOrphanedMetricMetadata(ctx, cutoff); err != nil {
		return err
	}

	return s.checkpointStorage(ctx)
}

func (s *Server) deleteMetricPointsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	deleted := int64(0)

	for _, table := range metricPointTables {
		count, err := s.deleteMetricPointsBeforeFromTable(ctx, table, cutoff)
		deleted += count
		if err != nil {
			return deleted, err
		}
	}

	return deleted, nil
}

func (s *Server) deleteMetricPointsBeforeFromTable(ctx context.Context,
	table string, cutoff time.Time) (int64, error) {
	statement := fmt.Sprintf(`DELETE FROM %[1]s WHERE rowid IN (
	SELECT rowid FROM %[1]s WHERE ingested_at < ? LIMIT %[2]d
)`, table, retentionDeleteChunkSize)

	deleted := int64(0)

	for {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}

		result, err := s.database().ExecContext(ctx, statement, metricTimeToDB(cutoff))
		if err != nil {
			return deleted, err
		}

		count, err := result.RowsAffected()
		if err != nil {
			return deleted, err
		}

		deleted += count
		if count < retentionDeleteChunkSize {
			return deleted, nil
		}
	}
}

func (s *Server) deleteOrphanedMetricMetadata(ctx context.Context, cutoff time.Time) error {
	cleanupStatements := []string{
		`DELETE FROM metric_series_attributes
		 WHERE updated_at < ? AND series_id NOT IN (
			SELECT series_id FROM metric_number_points
			UNION SELECT series_id FROM metric_histogram_points
			UNION SELECT series_id FROM metric_exponential_histogram_points
		 )`,
		`DELETE FROM metric_series
		 WHERE updated_at < ? AND id NOT IN (
			SELECT series_id FROM metric_number_points
			UNION SELECT series_id FROM metric_histogram_points
			UNION SELECT series_id FROM metric_exponential_histogram_points
		 )`,
		`DELETE FROM metric_attribute_keys
		 WHERE last_seen_at < ? AND (descriptor_id, key) NOT IN (
			SELECT DISTINCT descriptor_id, key FROM metric_series_attributes
		 )`,
		`DELETE FROM metric_descriptors
		 WHERE updated_at < ? AND id NOT IN (SELECT DISTINCT descriptor_id FROM metric_series)`,
	}

	for _, statement := range cleanupStatements {
		if _, err := s.database().ExecContext(ctx, statement, metricTimeToDB(cutoff)); err != nil {
			return err
		}
	}

	return nil
}
