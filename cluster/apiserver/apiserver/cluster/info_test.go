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
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
)

func TestGetClusterInfo(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	oldGetLatestVersionFunc := getLatestVersionFunc
	getLatestVersionFunc = func(ctx context.Context, pkg string) (string, error) {
		switch pkg {
		case "octelium":
			return "1.1.0", nil
		case "octelium-ee":
			return "2.0.0", nil
		case "cordium":
			return "1.0.0", nil
		default:
			return "", nil
		}
	}
	t.Cleanup(func() {
		getLatestVersionFunc = oldGetLatestVersionFunc
	})

	{
		_, err := srv.GetClusterInfo(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: "default"})
		assert.Nil(t, err, "%+v", err)
		if rgn.Status == nil {
			rgn.Status = &corev1.Region_Status{}
		}

		setAt := pbutils.Now()
		rgn.Status.VersionInfoMap = map[string]*corev1.Region_Status_VersionInfo{
			"octelium": {
				Version: "1.0.0",
				SetAt:   setAt,
			},
			"octeliumee": {
				Version: "2.0.0",
				SetAt:   setAt,
			},
			"cordium": {
				Version: "dev",
				SetAt:   setAt,
			},
		}
		_, err = srv.octeliumC.CoreC().UpdateRegion(ctx, rgn)
		assert.Nil(t, err, "%+v", err)

		resp, err := srv.GetClusterInfo(ctx, &enterprisev1.GetClusterInfoRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, resp.Core)
		assert.NotNil(t, resp.PackageEnterprise)
		assert.NotNil(t, resp.PackageCordium)
		assert.Equal(t, "1.0.0", resp.Core.CurrentVersion)
		assert.Equal(t, "1.1.0", resp.Core.LatestVersion)
		assert.True(t, resp.Core.CanUpgrade)
		assert.Equal(t, setAt.AsTime(), resp.Core.SetAt.AsTime())
		assert.Equal(t, "2.0.0", resp.PackageEnterprise.CurrentVersion)
		assert.Equal(t, "2.0.0", resp.PackageEnterprise.LatestVersion)
		assert.False(t, resp.PackageEnterprise.CanUpgrade)
		assert.Equal(t, "dev", resp.PackageCordium.CurrentVersion)
		assert.Equal(t, "1.0.0", resp.PackageCordium.LatestVersion)
		assert.False(t, resp.PackageCordium.CanUpgrade)
	}

	{
		assert.True(t, canUpgradeVersion("1.0.0", "1.0.1"))
		assert.False(t, canUpgradeVersion("1.0.1", "1.0.0"))
		assert.False(t, canUpgradeVersion("", "1.0.0"))
		assert.False(t, canUpgradeVersion("1.0.0", ""))
		assert.False(t, canUpgradeVersion("dev", "1.0.0"))
		assert.False(t, canUpgradeVersion("1.0.0", "latest"))
	}
}
