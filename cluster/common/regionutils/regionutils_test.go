// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package regionutils

import (
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRegionExtInfo(t *testing.T) {
	{
		for _, rgn := range []*corev1.Region{nil, {}, {Status: &corev1.Region_Status{}}} {
			info, err := GetRegionExtInfo(rgn)
			require.NoError(t, err)
			assert.Nil(t, info.Kubernetes)
			assert.False(t, IsJoinedRegion(rgn))
			assert.False(t, IsJoinCompleted(rgn))
			assert.Empty(t, GetPackageVersion(rgn, PackageCore))
		}
	}

	{
		assert.Error(t, SetRegionExtInfo(nil, &enterprisev1.RegionExtInfo{}))
	}

	rgn := &corev1.Region{}
	require.NoError(t, SetRegionExtInfo(rgn, &enterprisev1.RegionExtInfo{
		Kubernetes: &enterprisev1.JoinClusterRequest_Kubernetes{
			Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_{
				BearerToken: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken{
					Type: &enterprisev1.JoinClusterRequest_Kubernetes_BearerToken_FromSecret{
						FromSecret: "token",
					},
					Server: "https://203.0.113.10:6443",
				},
			},
		},
	}))
	require.NotNil(t, rgn.Status.Ext[ovutils.ExtInfoKeyEnterprise])

	info, err := GetRegionExtInfo(rgn)
	require.NoError(t, err)
	assert.Equal(t, "token", info.Kubernetes.GetBearerToken().GetFromSecret())
	assert.Equal(t, "https://203.0.113.10:6443", info.Kubernetes.GetBearerToken().Server)
	assert.True(t, IsJoinedRegion(rgn))
	assert.False(t, IsJoinCompleted(rgn))

	rgn.Status.VersionInfoMap = map[string]*corev1.Region_Status_VersionInfo{
		PackageCore:    {Version: "1.0.0"},
		PackageCordium: nil,
	}
	assert.Equal(t, "1.0.0", GetPackageVersion(rgn, PackageCore))
	assert.True(t, HasPackage(rgn, PackageCore))
	assert.False(t, HasPackage(rgn, PackageCordium))
	assert.False(t, IsJoinCompleted(rgn))

	rgn.Status.VersionInfoMap[PackageEnterprise] = &corev1.Region_Status_VersionInfo{Version: "2.0.0"}
	assert.True(t, IsJoinCompleted(rgn))

	rgn.Status.Ext[ovutils.ExtInfoKeyEnterprise] = &structpb.Struct{Fields: map[string]*structpb.Value{
		"kubernetes": structpb.NewStringValue("invalid"),
	}}
	_, err = GetRegionExtInfo(rgn)
	assert.Error(t, err)
	assert.False(t, IsJoinedRegion(rgn))
}

func TestIsPackageSetWithin(t *testing.T) {
	from := time.Now()
	at := func(d time.Duration) *timestamppb.Timestamp {
		return timestamppb.New(from.Add(d))
	}

	rgn := &corev1.Region{Status: &corev1.Region_Status{VersionInfoMap: map[string]*corev1.Region_Status_VersionInfo{
		PackageCore:       {Version: "1.0.0", SetAt: at(time.Minute)},
		PackageEnterprise: {Version: "1.0.0"},
		PackageCordium:    {SetAt: at(time.Minute)},
	}}}

	assert.True(t, IsPackageSetWithin(rgn, PackageCore, at(0), nil))
	assert.True(t, IsPackageSetWithin(rgn, PackageCore, at(time.Minute), at(2*time.Minute)))
	assert.False(t, IsPackageSetWithin(rgn, PackageCore, at(2*time.Minute), nil))
	assert.False(t, IsPackageSetWithin(rgn, PackageCore, at(0), at(time.Minute)))
	assert.False(t, IsPackageSetWithin(rgn, PackageCore, nil, nil))
	assert.False(t, IsPackageSetWithin(rgn, PackageEnterprise, at(0), nil))
	assert.False(t, IsPackageSetWithin(rgn, PackageCordium, at(0), nil))
	assert.False(t, IsPackageSetWithin(rgn, "unknown", at(0), nil))
	assert.False(t, IsPackageSetWithin(nil, PackageCore, at(0), nil))
}

func TestCheckJoinCoreVersion(t *testing.T) {
	for _, tc := range []struct {
		version string
		ok      bool
	}{
		{"0.53.0", true},
		{"v0.53.0", true},
		{"0.53.0-rc1", true},
		{"0.60.1", true},
		{"1.0.0", true},
		{"9.9.9", true},
		{"main", true},
		{"latest", true},
		{"0.52.0", false},
		{"v0.52.9", false},
		{"0.39.0", false},
	} {
		t.Run(tc.version, func(t *testing.T) {
			err := CheckJoinCoreVersion(tc.version)
			if tc.ok {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.version)
			}
		})
	}
}
