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
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-version"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
)

func (s *Server) GetClusterInfo(ctx context.Context, req *enterprisev1.GetClusterInfoRequest) (*enterprisev1.GetClusterInfoResponse, error) {

	ret := &enterprisev1.GetClusterInfoResponse{}

	rgn, err := s.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	if err != nil {
		return nil, err
	}

	latestCore, _ := getLatestVersion(ctx, "octelium")
	latestEE, _ := getLatestVersion(ctx, "octelium-ee")
	latestCordium, _ := getLatestVersion(ctx, "cordium")

	canUpgrade := func(cur, latest string) bool {
		if cur, err := version.NewSemver(cur); err == nil {
			if latest, err := version.NewSemver(latest); err == nil {
				return latest.GreaterThan(cur)
			}
		}
		return false
	}

	if info, ok := rgn.Status.VersionInfoMap["octelium"]; ok {
		ret.Core = &enterprisev1.GetClusterInfoResponse_Core{
			CurrentVersion: info.Version,
			LatestVersion:  latestCore,
			CanUpgrade:     canUpgrade(info.Version, latestCore),
		}
	}

	if info, ok := rgn.Status.VersionInfoMap["octeliumee"]; ok {
		ret.Core = &enterprisev1.GetClusterInfoResponse_Core{
			CurrentVersion: info.Version,
			LatestVersion:  latestEE,
			CanUpgrade:     canUpgrade(info.Version, latestEE),
		}
	}

	if info, ok := rgn.Status.VersionInfoMap["cordium"]; ok {
		ret.Core = &enterprisev1.GetClusterInfoResponse_Core{
			CurrentVersion: info.Version,
			LatestVersion:  latestCordium,
			CanUpgrade:     canUpgrade(info.Version, latestCordium),
		}
	}

	return ret, nil
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

	return (string(resp.Body())), nil
}
