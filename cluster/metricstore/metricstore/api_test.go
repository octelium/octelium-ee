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

func TestMetricAttributeGroupCapability(t *testing.T) {
	for _, key := range []string{
		"octelium.vigil.svc.name",
		"octelium.vigil.svc.namespace.name",
		"octelium.vigil.svc.region.name",
		"octelium.vigil.svc.mode",
		"reason",
		"state",
	} {
		groupable, reason := metricAttributeGroupCapability(key, maxSeriesPerQuery*10)
		assert.True(t, groupable, key)
		assert.Empty(t, reason, key)
	}

	groupable, reason := metricAttributeGroupCapability("octelium.vigil.user.id", 1)
	assert.False(t, groupable)
	assert.Equal(t, "attribute key is not in the low-cardinality groupBy allowlist", reason)

	groupable, reason = metricAttributeGroupCapability("state", maxSeriesPerQuery*10+1)
	assert.False(t, groupable)
	assert.Equal(t, "observed attribute cardinality exceeds the groupBy safety threshold", reason)
}
