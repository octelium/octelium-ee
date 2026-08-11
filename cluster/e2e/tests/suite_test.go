//go:build e2e

// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package tests

import (
	"os"
	"testing"

	eesuite "github.com/octelium/octelium-ee/cluster/e2e/suite"
	"github.com/octelium/octelium/cluster/e2e/suite"
)

func TestMain(m *testing.M) {
	os.Exit(suite.Bootstrap(m))
}

func TestE2E(t *testing.T) {
	suite.Run(t, eesuite.All())
}
