// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"

	"github.com/octelium/octelium/apis/main/accessv1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

const (
	minClusterConfigAuthorizationPriority int32 = -4
	maxClusterConfigAuthorizationPriority int32 = 0
)

func (s *ServerMain) GetClusterConfig(ctx context.Context,
	req *accessv1.GetClusterConfigRequest) (*accessv1.ClusterConfig, error) {
	cc, err := s.octeliumC.AccessV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return cc, nil
}

func (s *ServerMain) UpdateClusterConfig(ctx context.Context,
	req *accessv1.ClusterConfig) (*accessv1.ClusterConfig, error) {
	if err := validateClusterConfig(req); err != nil {
		return nil, err
	}

	cc, err := s.octeliumC.AccessV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	apisrvcommon.MetadataUpdate(cc.Metadata, req.Metadata)
	cc.Spec = req.Spec

	ret, err := s.octeliumC.AccessC().UpdateClusterConfig(ctx, cc)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return ret, nil
}

func validateClusterConfig(req *accessv1.ClusterConfig) error {
	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return err
	}

	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	if authz := req.Spec.Authorization; authz != nil && authz.Priority != nil {
		if *authz.Priority < minClusterConfigAuthorizationPriority ||
			*authz.Priority > maxClusterConfigAuthorizationPriority {
			return grpcutils.InvalidArg("The authorization priority must be between %d and %d",
				minClusterConfigAuthorizationPriority, maxClusterConfigAuthorizationPriority)
		}
	}

	return nil
}
