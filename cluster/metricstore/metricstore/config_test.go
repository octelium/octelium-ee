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
)

func TestDetectTempDirectoryLimit(t *testing.T) {
	assert.Equal(t, maximumTempDirectoryBytes, detectTempDirectoryLimit(0))
	assert.Equal(t, maximumTempDirectoryBytes, detectTempDirectoryLimit(1<<60))
	assert.Equal(t, minimumTempDirectoryBytes, detectTempDirectoryLimit(64<<20))
	assert.Equal(t, maximumTempDirectoryBytes, detectTempDirectoryLimit(200<<30))

	assert.Equal(t, int64(1250)<<20, detectTempDirectoryLimit(5000<<20))
	assert.Equal(t, int64(5)<<30, detectTempDirectoryLimit(20<<30))
}

func TestFormatDuckDBBytes(t *testing.T) {
	assert.Equal(t, "2GB", formatDuckDBBytes(2<<30))
	assert.Equal(t, "1250MB", formatDuckDBBytes(1250<<20))
	assert.Equal(t, "512MB", formatDuckDBBytes(512<<20))
}
