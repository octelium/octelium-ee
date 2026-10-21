// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDuckDBBytes(t *testing.T) {
	assert.Equal(t, "2GB", formatDuckDBBytes(2<<30))
	assert.Equal(t, "1250MB", formatDuckDBBytes(1250<<20))
	assert.Equal(t, "512MB", formatDuckDBBytes(512<<20))
}

func TestMetricStoreDBConfigEnablesTheAllocatorBackgroundThreads(t *testing.T) {
	t.Setenv("OCTELIUM_METRICSTORE_DUCKDB_PATH", t.TempDir())

	dbConfig, err := getMetricStoreDBConfig()
	require.NoError(t, err)
	if dbConfig.cleanupFn != nil {
		defer dbConfig.cleanupFn()
	}

	db, err := openMetricStoreDB(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	var value string
	require.NoError(t, db.QueryRow(`SELECT current_setting('allocator_background_threads')`).Scan(&value))
	assert.Equal(t, "true", value)
}
