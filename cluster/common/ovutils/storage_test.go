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
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageUsageRatio(t *testing.T) {
	for _, tst := range []struct {
		usage StorageUsage
		ratio float64
	}{
		{usage: StorageUsage{}, ratio: 0},
		{usage: StorageUsage{TotalBytes: 1000, AvailableBytes: 1000}, ratio: 0},
		{usage: StorageUsage{TotalBytes: 1000, AvailableBytes: 100}, ratio: 0.9},
		{usage: StorageUsage{TotalBytes: 1000, AvailableBytes: 50, ReusableBytes: 350}, ratio: 0.6},
		{usage: StorageUsage{TotalBytes: 1000}, ratio: 1},
		{usage: StorageUsage{TotalBytes: 1000, AvailableBytes: 900, ReusableBytes: 900}, ratio: 0},
	} {
		assert.InDelta(t, tst.ratio, tst.usage.UsedRatio(), 0.0001)
	}

	usage := StorageUsage{TotalBytes: 1000, AvailableBytes: 100, ReusableBytes: 200}
	assert.Equal(t, uint64(300), usage.FreeBytes())
	assert.Len(t, usage.ZapFields(), 4)
}

func TestStatfsBytes(t *testing.T) {
	totalBytes, availableBytes, err := StatfsBytes(t.TempDir())
	assert.Nil(t, err, "%+v", err)
	assert.NotZero(t, totalBytes)
	assert.LessOrEqual(t, availableBytes, totalBytes)

	_, _, err = StatfsBytes(filepath.Join(t.TempDir(), "does-not-exist"))
	assert.NotNil(t, err)
}

func TestReadStorageUsageWithoutDatabasePath(t *testing.T) {
	usage, err := ReadStorageUsage(context.Background(), nil, "", nil)
	assert.Nil(t, err, "%+v", err)
	assert.Zero(t, usage.TotalBytes)
	assert.Zero(t, usage.UsedRatio())
}

func TestIsBlockedCheckpointErr(t *testing.T) {
	assert.False(t, IsBlockedCheckpointErr(nil))
	assert.False(t, IsBlockedCheckpointErr(errors.New("IO Error: could not write to the database file")))
	assert.True(t, IsBlockedCheckpointErr(errors.New(
		"TransactionContext Error: Cannot CHECKPOINT: there are other write transactions active")))
}

func TestDetectTempDirectoryLimitBytes(t *testing.T) {
	assert.Equal(t, defaultMaxTempDirectoryBytes, DetectTempDirectoryLimitBytes(0, 0))
	assert.Equal(t, defaultMaxTempDirectoryBytes, DetectTempDirectoryLimitBytes(1<<60, 0))
	assert.Equal(t, minimumTempDirectoryBytes, DetectTempDirectoryLimitBytes(64<<20, 0))
	assert.Equal(t, int64(1250)<<20, DetectTempDirectoryLimitBytes(5000<<20, 0))
	assert.Equal(t, defaultMaxTempDirectoryBytes, DetectTempDirectoryLimitBytes(100<<30, 0))

	assert.Equal(t, int64(5)<<30, DetectTempDirectoryLimitBytes(20<<30, 20<<30))
	assert.Equal(t, int64(20)<<30, DetectTempDirectoryLimitBytes(1<<50, 20<<30))
}

func TestGetDuckDBDatabasePath(t *testing.T) {
	t.Setenv("OCTELIUM_DUCKDB_PATH", "/tst-data")

	assert.Equal(t, "/tst-data/store.db", GetDuckDBDatabasePath(GetDuckDBDSNWithOpts(nil)))
	assert.Equal(t, "/tst-data/store.db", GetDuckDBDatabasePath(GetDuckDBDSN()))
	assert.Equal(t, "/tst-data/store.db", GetDuckDBDatabasePath("/tst-data/store.db"))
	assert.Empty(t, GetDuckDBDatabasePath(""))
}
