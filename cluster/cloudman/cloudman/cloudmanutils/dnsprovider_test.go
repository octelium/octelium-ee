// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cloudmanutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterDomainBoundaries(t *testing.T) {
	for _, tc := range []struct {
		domain string
		want   bool
	}{{"example.com", true}, {"api.example.com", true}, {"example.com.evil", false}, {"notexample.com", false}, {"", false}} {
		assert.Equal(t, tc.want, IsSameClusterDomain(tc.domain, "example.com"))
	}
}
