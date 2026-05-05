// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cluster

import (
	"context"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/stretchr/testify/assert"
)

func TestClusterConfig(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	{
		_, err := srv.GetClusterInfo(ctx, &enterprisev1.GetClusterInfoRequest{})
		assert.Nil(t, err)
	}
}
