// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package collectorutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectorUtils(t *testing.T) {
	assert.Equal(t, "collector.octelium.example.com:443", GetServiceEndpoint("example.com"))
	assert.Equal(t, "collector.octelium.example.com:443", GetServiceEndpoint("example.com."))
	assert.Equal(t, ":8081", GetIngestAddr())
	assert.Equal(t, "grpc://octeliumee-logstore.octelium.svc:8081", GetUpstreamURL("octeliumee-logstore.octelium.svc"))
}
