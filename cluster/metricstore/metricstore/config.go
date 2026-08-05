// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/octelium/octelium/pkg/utils/ldflags"
)

const (
	defaultDuckDBDir              = "/octelium-data"
	defaultDatabaseName           = "metricstore.db"
	defaultDuckDBMemoryLimitBytes = int64(512 << 20)
	minimumDuckDBMemoryLimitBytes = int64(64 << 20)
	maximumDuckDBMemoryLimitBytes = int64(4 << 30)
	defaultMaxTempDirectorySize   = "20GB"
)

type metricStoreDBConfig struct {
	dsn       string
	database  string
	tempDir   string
	cleanupFn func()
}

func getMetricStoreDBConfig() (*metricStoreDBConfig, error) {
	var cleanupFn func()

	dir := strings.TrimSpace(os.Getenv("OCTELIUM_METRICSTORE_DUCKDB_PATH"))
	if dir == "" {
		dir = strings.TrimSpace(os.Getenv("OCTELIUM_DUCKDB_PATH"))
	}

	if ldflags.IsTest() {
		var err error
		dir, err = os.MkdirTemp("", "octelium-metricstore-*")
		if err != nil {
			return nil, err
		}
		cleanupFn = func() {
			_ = os.RemoveAll(dir)
		}
	} else if dir == "" {
		dir = defaultDuckDBDir
	}

	if !filepath.IsAbs(dir) || strings.ContainsAny(dir, `?#`) {
		if cleanupFn != nil {
			cleanupFn()
		}
		return nil, fmt.Errorf("invalid metricstore DuckDB directory: %s", dir)
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		if cleanupFn != nil {
			cleanupFn()
		}
		return nil, err
	}

	databaseName := strings.TrimSpace(os.Getenv("OCTELIUM_METRICSTORE_DUCKDB_DATABASE"))
	if databaseName == "" {
		databaseName = defaultDatabaseName
	}
	if filepath.Base(databaseName) != databaseName || strings.ContainsAny(databaseName, `/\\`) {
		if cleanupFn != nil {
			cleanupFn()
		}
		return nil, fmt.Errorf("invalid metricstore DuckDB database name: %s", databaseName)
	}

	tempDir := filepath.Join(dir, "metricstore-tmp")
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		if cleanupFn != nil {
			cleanupFn()
		}
		return nil, err
	}

	threads := 2
	if runtime.GOMAXPROCS(0) < threads {
		threads = runtime.GOMAXPROCS(0)
	}
	if val := strings.TrimSpace(os.Getenv("OCTELIUM_METRICSTORE_DUCKDB_THREADS")); val != "" {
		parsed, err := strconv.Atoi(val)
		if err != nil || parsed < 1 {
			if cleanupFn != nil {
				cleanupFn()
			}
			return nil, fmt.Errorf("invalid OCTELIUM_METRICSTORE_DUCKDB_THREADS")
		}
		threads = parsed
	}

	memoryLimit := strings.TrimSpace(os.Getenv("OCTELIUM_METRICSTORE_DUCKDB_MEMORY_LIMIT"))
	if memoryLimit == "" {
		memoryLimit = formatDuckDBBytes(detectDuckDBMemoryLimit())
	}

	maxTempSize := strings.TrimSpace(os.Getenv("OCTELIUM_METRICSTORE_DUCKDB_MAX_TEMP_SIZE"))
	if maxTempSize == "" {
		maxTempSize = defaultMaxTempDirectorySize
	}

	values := url.Values{}
	values.Set("access_mode", "read_write")
	values.Set("allow_community_extensions", "false")
	values.Set("autoinstall_known_extensions", "false")
	values.Set("enable_external_access", "false")
	values.Set("memory_limit", memoryLimit)
	values.Set("max_temp_directory_size", maxTempSize)
	values.Set("preserve_insertion_order", "false")
	values.Set("threads", strconv.Itoa(threads))
	values.Set("write_buffer_row_group_count", "1")

	database := filepath.Join(dir, databaseName)

	return &metricStoreDBConfig{
		dsn:       database + "?" + values.Encode(),
		database:  database,
		tempDir:   tempDir,
		cleanupFn: cleanupFn,
	}, nil
}

func detectDuckDBMemoryLimit() int64 {
	for _, path := range []string{
		"/sys/fs/cgroup/memory.max",
		"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		val := strings.TrimSpace(string(data))
		if val == "" || val == "max" {
			continue
		}

		limit, err := strconv.ParseInt(val, 10, 64)
		if err != nil || limit <= 0 || limit >= 1<<60 {
			continue
		}

		ret := limit * 40 / 100
		if ret < minimumDuckDBMemoryLimitBytes {
			ret = minimumDuckDBMemoryLimitBytes
		}
		if ret > maximumDuckDBMemoryLimitBytes {
			ret = maximumDuckDBMemoryLimitBytes
		}
		return ret
	}

	return defaultDuckDBMemoryLimitBytes
}

func formatDuckDBBytes(val int64) string {
	if val%(1<<30) == 0 {
		return fmt.Sprintf("%dGB", val>>30)
	}
	return fmt.Sprintf("%dMB", val>>20)
}
