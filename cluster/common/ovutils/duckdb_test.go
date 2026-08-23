// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package ovutils

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func parseDSN(t *testing.T, dsn string) (string, url.Values) {
	t.Helper()

	pth, query, _ := strings.Cut(dsn, "?")
	values, err := url.ParseQuery(query)
	assert.Nil(t, err, "%+v", err)

	return pth, values
}

func TestGetDuckDBDSN(t *testing.T) {
	t.Setenv("OCTELIUM_DUCKDB_PATH", "/tst-data")

	pth, values := parseDSN(t, GetDuckDBDSN())

	assert.Equal(t, "/tst-data/store.db", pth)
	assert.Equal(t, "/tst-data", values.Get("extension_directory"))
	assert.Len(t, values, 1)
}

func TestGetDuckDBDSNWithOpts(t *testing.T) {
	t.Setenv("OCTELIUM_DUCKDB_PATH", "/tst-data")

	{
		pth, values := parseDSN(t, GetDuckDBDSNWithOpts(nil))

		assert.Equal(t, "/tst-data/store.db", pth)
		assert.Equal(t, "/tst-data", values.Get("extension_directory"))
		assert.NotEmpty(t, values.Get("threads"))
		assert.NotEmpty(t, values.Get("memory_limit"))
		assert.Equal(t, defaultDuckDBMaxTempDirectorySize, values.Get("max_temp_directory_size"))
	}

	{
		_, values := parseDSN(t, GetDuckDBDSNWithOpts(&DuckDBOpts{
			Threads:              1,
			MemoryLimit:          "128MB",
			MaxTempDirectorySize: "1GB",
		}))

		assert.Equal(t, "1", values.Get("threads"))
		assert.Equal(t, "128MB", values.Get("memory_limit"))
		assert.Equal(t, "1GB", values.Get("max_temp_directory_size"))
	}
}

func TestDetectDuckDBMemoryLimitBytes(t *testing.T) {
	ret := DetectDuckDBMemoryLimitBytes()

	assert.GreaterOrEqual(t, ret, minimumDuckDBMemoryLimitBytes)
	assert.LessOrEqual(t, ret, maximumDuckDBMemoryLimitBytes)
}

func TestFormatDuckDBBytes(t *testing.T) {
	assert.Equal(t, "2GB", FormatDuckDBBytes(2<<30))
	assert.Equal(t, "512MB", FormatDuckDBBytes(512<<20))
	assert.Equal(t, "600MB", FormatDuckDBBytes(600<<20))
}
