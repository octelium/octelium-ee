// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package ovutils

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/octelium/octelium/pkg/utils/ldflags"
)

const (
	defaultDuckDBThreads              = 2
	defaultDuckDBMemoryLimitBytes     = int64(512 << 20)
	minimumDuckDBMemoryLimitBytes     = int64(64 << 20)
	maximumDuckDBMemoryLimitBytes     = int64(4 << 30)
	defaultDuckDBMaxTempDirectorySize = "4GB"
)

type DuckDBOpts struct {
	Threads              int
	MemoryLimit          string
	MaxTempDirectorySize string
}

func GetDuckDBDSNWithOpts(o *DuckDBOpts) string {
	if o == nil {
		o = &DuckDBOpts{}
	}

	dir := resolveDuckDBDir()

	threads := o.Threads
	if threads < 1 {
		threads = defaultDuckDBThreads
	}
	if procs := runtime.GOMAXPROCS(0); procs > 0 && procs < threads {
		threads = procs
	}

	memoryLimit := strings.TrimSpace(o.MemoryLimit)
	if memoryLimit == "" {
		memoryLimit = FormatDuckDBBytes(DetectDuckDBMemoryLimitBytes())
	}

	maxTempDirectorySize := strings.TrimSpace(o.MaxTempDirectorySize)
	if maxTempDirectorySize == "" {
		maxTempDirectorySize = defaultDuckDBMaxTempDirectorySize
	}

	values := url.Values{}
	values.Set("extension_directory", dir)
	values.Set("threads", strconv.Itoa(threads))
	values.Set("memory_limit", memoryLimit)
	values.Set("max_temp_directory_size", maxTempDirectorySize)

	return fmt.Sprintf("%s?%s", path.Join(dir, duckDBFileName), values.Encode())
}

func DetectDuckDBMemoryLimitBytes() int64 {
	for _, pth := range []string{
		"/sys/fs/cgroup/memory.max",
		"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	} {
		data, err := os.ReadFile(pth)
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

func FormatDuckDBBytes(val int64) string {
	if val%(1<<30) == 0 {
		return fmt.Sprintf("%dGB", val>>30)
	}
	return fmt.Sprintf("%dMB", val>>20)
}

func resolveDuckDBDir() string {
	if ldflags.IsTest() {
		dir, _ := os.MkdirTemp("", "duckdb-*")
		return dir
	}

	if val := os.Getenv("OCTELIUM_DUCKDB_PATH"); val != "" {
		return val
	}

	return getDuckDBDir()
}
