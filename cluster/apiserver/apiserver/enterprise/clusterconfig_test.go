// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestClusterConfig(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)

	cc, err := srv.GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
	assert.Nil(t, err)
	cc.Metadata.Labels = map[string]string{
		"key1": utilrand.GetRandomStringCanonical(8),
	}

	zap.L().Debug("clusercon", zap.Any("cc", cc))

	cc, err = srv.UpdateClusterConfig(ctx, cc)
	assert.Nil(t, err)
}

/*
func TestUpgradeCluster(t *testing.T) {

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})



	srv := NewServer(tst.C.OcteliumC)

	u, err := tests.NewUser(srv.octeliumC, admin.NewServer(&admin.Opts{
		OcteliumC:  tst.C.OcteliumC,
		IsEmbedded: true,
	}), nil, nil)
	assert.Nil(t, err)


		_, err = srv.UpgradeCluster(u.Ctx(), &clusterv1.UpgradeClusterRequest{})
		assert.Nil(t, err)

}
*/
