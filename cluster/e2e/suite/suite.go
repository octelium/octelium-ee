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

	_ "github.com/octelium/octelium-ee/cluster/e2e/scenario"
)

func Phases() []suite.Phase {
	return []suite.Phase{
		{Name: "EnterpriseSDK", Run: testEnterpriseSDK},
		{Name: "SCIM", Run: testSCIM},
	}
}

func All() []suite.Phase {
	return suite.InsertBefore(suite.Phases(), "ComponentHealth", Phases()...)
}
