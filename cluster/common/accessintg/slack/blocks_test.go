// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package slack

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestTruncateKeepsValidUTF8(t *testing.T) {
	arg := strings.Repeat("é", 64)

	for maxLen := 1; maxLen < len(arg); maxLen++ {
		ret := truncate(arg, maxLen)
		assert.True(t, utf8.ValidString(ret), "maxLen=%d", maxLen)
		assert.True(t, strings.HasSuffix(ret, "..."), "maxLen=%d", maxLen)
	}
}

func TestTruncateShorterThanMaxLen(t *testing.T) {
	assert.Equal(t, "octelium", truncate("octelium", 64))
	assert.Equal(t, "octelium", truncate("octelium", len("octelium")))
}
