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
	"time"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"go.uber.org/zap"
)

const (
	storagePressureInterval    = 2 * time.Minute
	storagePressureHighRatio   = 0.9
	storagePressureTargetRatio = 0.8
	maximumCleanupDivisor      = 128
	minimumCleanupLimit        = 1000
	forceCheckpointTimeout     = 2 * time.Minute
)

func (s *Server) readStorageUsage(ctx context.Context) (*ovutils.StorageUsage, error) {
	return ovutils.ReadStorageUsage(ctx, s.db, s.dbPath, s.statfsFn)
}

func (s *Server) startStoragePressureLoop(ctx context.Context) {
	ticker := time.NewTicker(storagePressureInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.applyStoragePressureCleanup(ctx); err != nil {
				zap.L().Warn("Could not apply the LogStore storage pressure cleanup", zap.Error(err))
			}
		}
	}
}

func (s *Server) applyStoragePressureCleanup(ctx context.Context) error {
	usage, err := s.readStorageUsage(ctx)
	if err != nil {
		return err
	}

	zap.L().Debug("LogStore storage usage", usage.ZapFields()...)

	if usage.UsedRatio() < storagePressureHighRatio {
		return nil
	}

	zap.L().Warn("LogStore storage is nearly full. Trimming the oldest logs", usage.ZapFields()...)

	for divisor := 2; ; divisor *= 2 {
		deleted := int64(0)

		for _, c := range getCleanupMaxCounts() {
			limit := c.limit / divisor
			if limit < minimumCleanupLimit {
				limit = minimumCleanupLimit
			}

			count, err := s.cleanupByMaxCount(ctx, c.table, c.where, limit)
			if err != nil {
				zap.L().Warn("Could not trim logs by max count",
					zap.String("table", c.table), zap.String("where", c.where), zap.Error(err))
				continue
			}
			deleted += count
		}

		if deleted > 0 {
			if err := s.checkpointStorage(ctx); err != nil {
				return err
			}

			usage, err = s.readStorageUsage(ctx)
			if err != nil {
				return err
			}

			zap.L().Info("Trimmed the oldest logs",
				append([]zap.Field{zap.Int("limitDivisor", divisor), zap.Int64("deletedLogs", deleted)},
					usage.ZapFields()...)...)

			if usage.UsedRatio() <= storagePressureTargetRatio {
				return nil
			}
		}

		if divisor >= maximumCleanupDivisor {
			break
		}
	}

	zap.L().Warn("LogStore storage is still nearly full after trimming the oldest logs",
		usage.ZapFields()...)

	return nil
}

func (s *Server) checkpointStorage(ctx context.Context) error {
	return ovutils.CheckpointDuckDB(ctx, s.db, forceCheckpointTimeout)
}
