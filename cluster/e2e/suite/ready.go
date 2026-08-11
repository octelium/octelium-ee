// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"testing"

	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/cluster/e2e/harness"
)

func testEnterpriseReady(t *testing.T, h *harness.H) {
	for _, name := range eescenario.Deployments {
		h.MustWaitDeployment(t, name)
	}

	h.StartLogStream(t.Context(), "-l octelium.com/component=secretman")
	h.StartLogStream(t.Context(), "-l octelium.com/component=policyportal")

	h.MustRun(t, "kubectl get pods -n octelium")
}
