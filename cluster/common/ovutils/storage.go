// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package ovutils

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

const defaultForceCheckpointTimeout = 2 * time.Minute

type StorageUsage struct {
	TotalBytes     uint64
	AvailableBytes uint64
	ReusableBytes  uint64
}

func (u *StorageUsage) FreeBytes() uint64 {
	return u.AvailableBytes + u.ReusableBytes
}

func (u *StorageUsage) UsedRatio() float64 {
	if u.TotalBytes == 0 {
		return 0
	}
	free := u.FreeBytes()
	if free >= u.TotalBytes {
		return 0
	}
	return float64(u.TotalBytes-free) / float64(u.TotalBytes)
}

func (u *StorageUsage) ZapFields() []zap.Field {
	return []zap.Field{
		zap.Uint64("storageTotalBytes", u.TotalBytes),
		zap.Uint64("storageAvailableBytes", u.AvailableBytes),
		zap.Uint64("storageReusableBytes", u.ReusableBytes),
		zap.Float64("storageUsedRatio", u.UsedRatio()),
	}
}

func StatfsBytes(path string) (uint64, uint64, error) {
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

func ReadDuckDBReusableBytes(ctx context.Context, db *sql.DB) (uint64, error) {
	var blockSize sql.NullInt64
	var freeBlocks sql.NullInt64
	if err := db.QueryRowContext(ctx,
		`SELECT block_size, free_blocks FROM pragma_database_size()`).Scan(&blockSize, &freeBlocks); err != nil {
		return 0, err
	}
	if blockSize.Int64 <= 0 || freeBlocks.Int64 <= 0 {
		return 0, nil
	}

	return uint64(blockSize.Int64) * uint64(freeBlocks.Int64), nil
}

func ReadStorageUsage(ctx context.Context, db *sql.DB, databasePath string,
	statfsFn func(path string) (uint64, uint64, error)) (*StorageUsage, error) {
	if databasePath == "" {
		return &StorageUsage{}, nil
	}

	if statfsFn == nil {
		statfsFn = StatfsBytes
	}

	totalBytes, availableBytes, err := statfsFn(filepath.Dir(databasePath))
	if err != nil {
		return nil, err
	}

	reusableBytes, err := ReadDuckDBReusableBytes(ctx, db)
	if err != nil {
		return nil, err
	}

	return &StorageUsage{
		TotalBytes:     totalBytes,
		AvailableBytes: availableBytes,
		ReusableBytes:  reusableBytes,
	}, nil
}

func CheckpointDuckDB(ctx context.Context, db *sql.DB, forceTimeout time.Duration) error {
	_, err := db.ExecContext(ctx, `CHECKPOINT`)
	if err == nil || !IsBlockedCheckpointErr(err) {
		return err
	}

	if forceTimeout <= 0 {
		forceTimeout = defaultForceCheckpointTimeout
	}

	forceCtx, cancel := context.WithTimeout(ctx, forceTimeout)
	defer cancel()

	_, err = db.ExecContext(forceCtx, `FORCE CHECKPOINT`)
	return err
}

func IsBlockedCheckpointErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Cannot CHECKPOINT")
}

func IsInvalidatedDatabaseErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "database has been invalidated")
}

func GetDuckDBDatabasePath(dsn string) string {
	path, _, _ := strings.Cut(dsn, "?")
	return path
}
