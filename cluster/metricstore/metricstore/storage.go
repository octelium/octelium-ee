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
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const (
	storagePressureInterval    = 2 * time.Minute
	storagePressureHighRatio   = 0.9
	storagePressureTargetRatio = 0.8
	minimumRawMetricRetention  = time.Hour
	forceCheckpointTimeout     = 2 * time.Minute
)

type storageUsage struct {
	totalBytes     uint64
	availableBytes uint64
	reusableBytes  uint64
}

func (u *storageUsage) freeBytes() uint64 {
	return u.availableBytes + u.reusableBytes
}

func (u *storageUsage) usedRatio() float64 {
	if u.totalBytes == 0 {
		return 0
	}
	free := u.freeBytes()
	if free >= u.totalBytes {
		return 0
	}
	return float64(u.totalBytes-free) / float64(u.totalBytes)
}

func (u *storageUsage) zapFields() []zap.Field {
	return []zap.Field{
		zap.Uint64("storageTotalBytes", u.totalBytes),
		zap.Uint64("storageAvailableBytes", u.availableBytes),
		zap.Uint64("storageReusableBytes", u.reusableBytes),
		zap.Float64("storageUsedRatio", u.usedRatio()),
	}
}

func statfsBytes(path string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	if stat.Bsize <= 0 {
		return 0, 0, fmt.Errorf("invalid filesystem block size for %s", path)
	}

	blockSize := uint64(stat.Bsize)
	return stat.Blocks * blockSize, stat.Bavail * blockSize, nil
}

func (s *Server) readStorageUsage(ctx context.Context) (*storageUsage, error) {
	if s.dbConfig == nil || s.dbConfig.database == "" {
		return &storageUsage{}, nil
	}

	statfsFn := s.statfsFn
	if statfsFn == nil {
		statfsFn = statfsBytes
	}

	totalBytes, availableBytes, err := statfsFn(filepath.Dir(s.dbConfig.database))
	if err != nil {
		return nil, err
	}

	ret := &storageUsage{
		totalBytes:     totalBytes,
		availableBytes: availableBytes,
	}

	var blockSize sql.NullInt64
	var freeBlocks sql.NullInt64
	if err := s.db.QueryRowContext(ctx,
		`SELECT block_size, free_blocks FROM pragma_database_size()`).Scan(&blockSize, &freeBlocks); err != nil {
		return nil, err
	}
	if blockSize.Int64 > 0 && freeBlocks.Int64 > 0 {
		ret.reusableBytes = uint64(blockSize.Int64) * uint64(freeBlocks.Int64)
	}

	return ret, nil
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
	if usage.usedRatio() < storagePressureHighRatio {
		return nil
	}

	zap.L().Warn("MetricStore storage is nearly full. Trimming the oldest metric data points",
		usage.zapFields()...)

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
				}, usage.zapFields()...)...)

			if usage.usedRatio() <= storagePressureTargetRatio {
				return nil
			}
		}

		if retention <= minimumRawMetricRetention {
			break
		}
	}

	zap.L().Warn("MetricStore storage is still nearly full after trimming the oldest metric data points",
		append([]zap.Field{zap.Duration("minimumRetention", minimumRawMetricRetention)},
			usage.zapFields()...)...)

	return nil
}

func (s *Server) checkpointStorage(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CHECKPOINT`)
	if err == nil || !isBlockedCheckpointErr(err) {
		return err
	}

	forceCtx, cancel := context.WithTimeout(ctx, forceCheckpointTimeout)
	defer cancel()

	_, err = s.db.ExecContext(forceCtx, `FORCE CHECKPOINT`)
	return err
}

func isBlockedCheckpointErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Cannot CHECKPOINT")
}
