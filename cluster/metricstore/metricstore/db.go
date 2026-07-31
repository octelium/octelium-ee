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
	"time"

	"go.uber.org/zap"
)

const (
	metricStoreSchemaVersion = 2
	rawMetricRetention       = 48 * time.Hour
	retentionInterval        = 30 * time.Minute
)

func (s *Server) initDB(ctx context.Context) error {
	if err := s.rejectLegacySchema(ctx); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS metricstore_schema (
		version INTEGER PRIMARY KEY,
		applied_at BIGINT NOT NULL
	)`); err != nil {
		return err
	}

	version, err := s.getSchemaVersion(ctx)
	if err != nil {
		return err
	}
	if version > metricStoreSchemaVersion {
		return fmt.Errorf("unsupported metricstore schema version: %d", version)
	}
	if version != 0 && version < metricStoreSchemaVersion {
		return fmt.Errorf("unsupported in-place metricstore schema migration from version %d to %d",
			version, metricStoreSchemaVersion)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := createMetricStoreTables(ctx, tx); err != nil {
		return err
	}

	if version == 0 {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO metricstore_schema (version, applied_at) VALUES (?, ?)`,
			metricStoreSchemaVersion,
			metricTimeToDB(time.Now()),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Server) getSchemaVersion(ctx context.Context) (int, error) {
	var version sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT MAX(version) FROM metricstore_schema`).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

func createMetricStoreTables(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
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
		`CREATE TABLE IF NOT EXISTS metric_number_staging AS SELECT * FROM metric_number_points WHERE false`,
		`CREATE TABLE IF NOT EXISTS metric_histogram_staging AS SELECT * FROM metric_histogram_points WHERE false`,
		`CREATE TABLE IF NOT EXISTS metric_exponential_histogram_staging AS
			SELECT * FROM metric_exponential_histogram_points WHERE false`,
	}

	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) rejectLegacySchema(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'main' AND table_name = 'metrics'
`).Scan(&count); err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf(
			"legacy metricstore schema detected; use a new metricstore v2 database or remove the experimental v1 database")
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
			}
		}
	}
}

func (s *Server) applyRetention(ctx context.Context) error {
	// Keep one extra retention-loop interval so periodic cleanup cannot shorten
	// the effective retention window.
	cutoff := time.Now().UTC().Add(-(rawMetricRetention + retentionInterval))
	pointTables := []string{
		"metric_number_points",
		"metric_histogram_points",
		"metric_exponential_histogram_points",
	}

	deleted := int64(0)
	for _, table := range pointTables {
		result, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE ingested_at < ?`, metricTimeToDB(cutoff))
		if err != nil {
			return err
		}
		if count, err := result.RowsAffected(); err == nil {
			deleted += count
		}
	}

	cleanupStatements := []string{
		`DELETE FROM metric_series_attributes
		 WHERE series_id NOT IN (
			SELECT series_id FROM metric_number_points
			UNION SELECT series_id FROM metric_histogram_points
			UNION SELECT series_id FROM metric_exponential_histogram_points
		 )`,
		`DELETE FROM metric_series
		 WHERE id NOT IN (
			SELECT series_id FROM metric_number_points
			UNION SELECT series_id FROM metric_histogram_points
			UNION SELECT series_id FROM metric_exponential_histogram_points
		 )`,
		`DELETE FROM metric_attribute_keys
		 WHERE (descriptor_id, key) NOT IN (
			SELECT DISTINCT descriptor_id, key FROM metric_series_attributes
		 )`,
		`DELETE FROM metric_descriptors
		 WHERE id NOT IN (SELECT DISTINCT descriptor_id FROM metric_series)`,
	}

	for _, statement := range cleanupStatements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	if deleted > 0 {
		if _, err := s.db.ExecContext(ctx, `CHECKPOINT`); err != nil {
			return err
		}
	}

	return nil
}
