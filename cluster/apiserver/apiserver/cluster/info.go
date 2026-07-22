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
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-version"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
)

var getLatestVersionFunc = getLatestVersion

func (s *Server) GetClusterInfo(ctx context.Context, req *enterprisev1.GetClusterInfoRequest) (*enterprisev1.GetClusterInfoResponse, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	ret := &enterprisev1.GetClusterInfoResponse{}

	rgn, err := s.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	if err != nil {
		return nil, err
	}

	latestCore, _ := getLatestVersionFunc(ctx, "octelium")
	latestEE, _ := getLatestVersionFunc(ctx, "octelium-ee")
	latestCordium, _ := getLatestVersionFunc(ctx, "cordium")

	if info, ok := rgn.Status.VersionInfoMap["octelium"]; ok && info != nil {
		ret.Core = &enterprisev1.GetClusterInfoResponse_Core{
			CurrentVersion: info.Version,
			SetAt:          info.SetAt,
			LatestVersion:  latestCore,
			CanUpgrade:     canUpgradeVersion(info.Version, latestCore),
		}
	}

	if info, ok := rgn.Status.VersionInfoMap["octeliumee"]; ok && info != nil {
		ret.PackageEnterprise = &enterprisev1.GetClusterInfoResponse_PackageEnterprise{
			CurrentVersion: info.Version,
			SetAt:          info.SetAt,
			LatestVersion:  latestEE,
			CanUpgrade:     canUpgradeVersion(info.Version, latestEE),
		}
	}

	if info, ok := rgn.Status.VersionInfoMap["cordium"]; ok && info != nil {
		ret.PackageCordium = &enterprisev1.GetClusterInfoResponse_PackageCordium{
			CurrentVersion: info.Version,
			SetAt:          info.SetAt,
			LatestVersion:  latestCordium,
			CanUpgrade:     canUpgradeVersion(info.Version, latestCordium),
		}
	}

	return ret, nil
}

func canUpgradeVersion(cur string, latest string) bool {
	cur = strings.TrimSpace(cur)
	latest = strings.TrimSpace(latest)

	if cur == "" || latest == "" {
		return false
	}

	curSemver, err := version.NewSemver(cur)
	if err != nil {
		return false
	}

	latestSemver, err := version.NewSemver(latest)
	if err != nil {
		return false
	}

	return latestSemver.GreaterThan(curSemver)
}

func getLatestVersion(ctx context.Context, pkg string) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	releaseURL := fmt.Sprintf("https://raw.githubusercontent.com/octelium/%s/refs/heads/main/unsorted/latest_release", pkg)
	resp, err := resty.New().SetDebug(ldflags.IsDev()).
		R().
		SetContext(ctx).
		Get(releaseURL)
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", errors.Errorf("Could not get latest version release for package: %s", pkg)
	}

	return strings.TrimSpace(string(resp.Body())), nil
}
