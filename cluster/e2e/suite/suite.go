// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"github.com/octelium/octelium/cluster/e2e/suite"
)

// Phases are the Enterprise-only phases.
func Phases() []suite.Phase {
	return []suite.Phase{
		{Name: "EnterpriseSDK", Run: testEnterpriseSDK},
		{Name: "SCIM", Run: testSCIM},
	}
}

// ReadyPhase asserts that the Enterprise package came up. It runs right after
// the core readiness phase so that a broken package install fails the suite
// before anything else gets a chance to time out against it.
func ReadyPhase() suite.Phase {
	return suite.Phase{Name: "EnterpriseReady", Run: testEnterpriseReady}
}

// All is the core suite with the Enterprise phases woven in.
func All() []suite.Phase {
	ret := suite.InsertAfter(suite.Phases(), "ClusterReady", ReadyPhase())
	return suite.InsertBefore(ret, "ComponentHealth", Phases()...)
}
