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
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
)

const (
	maxLastUpgradeRequests   = 200
	staleUpgradeRequestAfter = 20 * time.Minute
)

func (s *Server) UpgradeCluster(ctx context.Context, req *enterprisev1.UpgradeClusterRequest) (
	*enterprisev1.UpgradeClusterResponse, error,
) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	upgradeReq, err := validateAndNormalizeUpgradeRequest(req.Request)
	if err != nil {
		return nil, err
	}

	cc, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cc.Status == nil {
		cc.Status = &enterprisev1.ClusterConfig_Status{}
	}

	if cc.Status.UpgradeRequest != nil {
		if isActiveUpgradeRequest(cc.Status.UpgradeRequest) &&
			!isStaleUpgradeRequest(cc.Status.UpgradeRequest, staleUpgradeRequestAfter) {
			return nil, grpcutils.InvalidArg("Cluster upgrade is already requested or in progress")
		}

		cc.Status.LastUpgradeRequests = append(
			[]*enterprisev1.ClusterConfig_Status_UpgradeRequest{cc.Status.UpgradeRequest},
			cc.Status.LastUpgradeRequests...,
		)

		if len(cc.Status.LastUpgradeRequests) > maxLastUpgradeRequests {
			cc.Status.LastUpgradeRequests = cc.Status.LastUpgradeRequests[:maxLastUpgradeRequests]
		}
	}

	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request:   upgradeReq,
	}

	cc, err = s.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	zap.L().Debug("UpgradeCluster new request", zap.Any("req", cc.Status.UpgradeRequest))

	return &enterprisev1.UpgradeClusterResponse{}, nil
}

func validateAndNormalizeUpgradeRequest(
	req *enterprisev1.UpgradeClusterRequest_Request,
) (*enterprisev1.UpgradeClusterRequest_Request, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Upgrade request is required")
	}

	ret := &enterprisev1.UpgradeClusterRequest_Request{}

	numTargets := 0

	if req.Core != nil {
		numTargets++

		ver, err := validateUpgradeVersion(req.Core.Version, "core.version")
		if err != nil {
			return nil, err
		}

		ret.Core = &enterprisev1.UpgradeClusterRequest_Request_Core{
			Version: ver,
		}
	}

	if req.PackageEnterprise != nil {
		numTargets++

		ver, err := validateUpgradeVersion(req.PackageEnterprise.Version, "packageEnterprise.version")
		if err != nil {
			return nil, err
		}

		ret.PackageEnterprise = &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
			Version: ver,
		}
	}

	if req.PackageCordium != nil {
		numTargets++

		ver, err := validateUpgradeVersion(req.PackageCordium.Version, "packageCordium.version")
		if err != nil {
			return nil, err
		}

		ret.PackageCordium = &enterprisev1.UpgradeClusterRequest_Request_PackageCordium{
			Version: ver,
		}
	}

	if numTargets == 0 {
		return nil, grpcutils.InvalidArg("At least one upgrade target must be set")
	}

	return ret, nil
}

func validateUpgradeVersion(ver string, field string) (string, error) {
	ver = strings.TrimSpace(ver)
	if ver == "" {
		return "latest", nil
	}

	if len(ver) > 100 {
		return "", grpcutils.InvalidArg("%s is too long", field)
	}

	if _, err := version.NewSemver(ver); err == nil {
		return ver, nil
	}

	if err := apivalidation.ValidateName(ver, 0, 0); err == nil {
		return ver, nil
	}

	return "", grpcutils.InvalidArg("Could not verify upgrade version: %s", ver)
}

func isActiveUpgradeRequest(req *enterprisev1.ClusterConfig_Status_UpgradeRequest) bool {
	if req == nil {
		return false
	}

	switch req.State {
	case enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING:
		return true
	default:
		return false
	}
}

func isStaleUpgradeRequest(
	req *enterprisev1.ClusterConfig_Status_UpgradeRequest,
	maxAge time.Duration,
) bool {
	if req == nil || req.CreatedAt == nil || !req.CreatedAt.IsValid() {
		return true
	}

	return time.Since(req.CreatedAt.AsTime()) > maxAge
}
