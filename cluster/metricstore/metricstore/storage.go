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

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"go.uber.org/zap"
)

const (
	storagePressureInterval    = 2 * time.Minute
	storagePressureHighRatio   = 0.9
	storagePressureTargetRatio = 0.8
	minimumRawMetricRetention  = time.Hour
	forceCheckpointTimeout     = 2 * time.Minute
)

func (s *Server) readStorageUsage(ctx context.Context) (*ovutils.StorageUsage, error) {
	database := ""
	if s.dbConfig != nil {
		database = s.dbConfig.database
	}

	return ovutils.ReadStorageUsage(ctx, s.database(), database, s.statfsFn)
}

func (s *Server) runStoragePressureLoop(ctx context.Context) {
	ticker := time.NewTicker(storagePressureInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := s.applyStoragePressureRetention(ctx); err != nil {
				zap.L().Warn("Could not apply metricstore storage pressure retention", zap.Error(err))
				s.requestDatabaseRecovery(err)
			}
		}
	}
}

func (s *Server) applyStoragePressureRetention(ctx context.Context) error {
	s.retentionMu.Lock()
	defer s.retentionMu.Unlock()

	usage, err := s.readStorageUsage(ctx)
	if err != nil {
		return err
	}
	if usage.UsedRatio() < storagePressureHighRatio {
		return nil
	}

	zap.L().Warn("MetricStore storage is nearly full. Trimming the oldest metric data points",
		usage.ZapFields()...)

	for retention := rawMetricRetention / 2; ; retention = retention / 2 {
		if retention < minimumRawMetricRetention {
			retention = minimumRawMetricRetention
		}

		cutoff := time.Now().UTC().Add(-retention)
		deleted, err := s.deleteMetricPointsBefore(ctx, cutoff)
		if err != nil {
			return err
		}

		if deleted > 0 {
			if err := s.deleteOrphanedMetricMetadata(ctx, cutoff); err != nil {
				return err
			}
			if err := s.checkpointStorage(ctx); err != nil {
				return err
			}

			usage, err = s.readStorageUsage(ctx)
			if err != nil {
				return err
			}

			zap.L().Info("Trimmed the oldest metricstore data points",
				append([]zap.Field{
					zap.Duration("retention", retention),
					zap.Int64("deletedDataPoints", deleted),
				}, usage.ZapFields()...)...)

			if usage.UsedRatio() <= storagePressureTargetRatio {
				return nil
			}
		}

		if retention <= minimumRawMetricRetention {
			break
		}
	}

	zap.L().Warn("MetricStore storage is still nearly full after trimming the oldest metric data points",
		append([]zap.Field{zap.Duration("minimumRetention", minimumRawMetricRetention)},
			usage.ZapFields()...)...)

	return nil
}

func (s *Server) checkpointStorage(ctx context.Context) error {
	return ovutils.CheckpointDuckDB(ctx, s.database(), forceCheckpointTimeout)
}
