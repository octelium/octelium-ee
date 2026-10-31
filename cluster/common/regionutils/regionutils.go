// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package regionutils

import (
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
)

const DefaultRegion = "default"

const (
	PackageCore       = "octelium"
	PackageEnterprise = "octeliumee"
	PackageCordium    = "cordium"
)

func GetRegionExtInfo(rgn *corev1.Region) (*enterprisev1.RegionExtInfo, error) {
	ret := &enterprisev1.RegionExtInfo{}

	if rgn == nil || rgn.Status == nil || rgn.Status.Ext == nil {
		return ret, nil
	}

	val, ok := rgn.Status.Ext[ovutils.ExtInfoKeyEnterprise]
	if !ok || val == nil {
		return ret, nil
	}

	if err := pbutils.StructToMessage(val, ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func SetRegionExtInfo(rgn *corev1.Region, info *enterprisev1.RegionExtInfo) error {
	if rgn == nil {
		return errors.Errorf("Nil Region")
	}

	if rgn.Status == nil {
		rgn.Status = &corev1.Region_Status{}
	}

	val, err := pbutils.MessageToStruct(info)
	if err != nil {
		return err
	}

	if rgn.Status.Ext == nil {
		rgn.Status.Ext = make(map[string]*structpb.Struct)
	}

	rgn.Status.Ext[ovutils.ExtInfoKeyEnterprise] = val

	return nil
}

func IsJoinedRegion(rgn *corev1.Region) bool {
	info, err := GetRegionExtInfo(rgn)
	if err != nil {
		return false
	}

	return info.Kubernetes != nil
}

func GetPackageVersion(rgn *corev1.Region, pkg string) string {
	if rgn == nil || rgn.Status == nil || rgn.Status.VersionInfoMap == nil {
		return ""
	}

	info, ok := rgn.Status.VersionInfoMap[pkg]
	if !ok || info == nil {
		return ""
	}

	return info.Version
}

func HasPackage(rgn *corev1.Region, pkg string) bool {
	return GetPackageVersion(rgn, pkg) != ""
}

func IsJoinCompleted(rgn *corev1.Region) bool {
	return IsJoinedRegion(rgn) && HasPackage(rgn, PackageEnterprise)
}
