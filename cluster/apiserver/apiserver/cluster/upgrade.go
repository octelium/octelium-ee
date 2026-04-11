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

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
)

func (s *Server) UpgradeCluster(ctx context.Context, req *enterprisev1.UpgradeClusterRequest) (
	*enterprisev1.UpgradeClusterResponse, error) {

	cc, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cc.Status.UpgradeRequest != nil {
		cc.Status.LastUpgradeRequests = append(
			[]*enterprisev1.ClusterConfig_Status_UpgradeRequest{cc.Status.UpgradeRequest},
			cc.Status.LastUpgradeRequests...)
	}

	cc.Status.UpgradeRequest = &enterprisev1.ClusterConfig_Status_UpgradeRequest{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED,
		Request:   req.Request,
	}

	cc, err = s.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	zap.L().Debug("UpgradeCluster new req", zap.Any("req", cc.Status.UpgradeRequest))

	return &enterprisev1.UpgradeClusterResponse{}, nil
}
