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
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
)

type LicenseCredential struct {
	JWT           string
	License       *enterprisev1.License
	ClusterDomain string
}

type LicenseUnavailableError struct {
	Reason string
}

func (e *LicenseUnavailableError) Error() string {
	return e.Reason
}

func GetLicenseCredential(ctx context.Context, octeliumC octeliumc.ClientInterface) (*LicenseCredential, error) {
	s := &Server{
		octeliumC: octeliumC,
	}

	cc, err := octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	if cc.Status == nil || cc.Status.LicenseInfo == nil ||
		cc.Status.LicenseInfo.LicenseSecretRef == nil ||
		cc.Status.LicenseInfo.State == enterprisev1.ClusterConfig_Status_LicenseInfo_NONE {
		return nil, &LicenseUnavailableError{
			Reason: "The Cluster does not have a License",
		}
	}

	jwtStr, err := s.getLicenseJWT(ctx, cc.Status.LicenseInfo.LicenseSecretRef)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, &LicenseUnavailableError{
				Reason: "The Cluster does not have a License",
			}
		}
		return nil, err
	}

	clusterDomain, err := s.getLicenseClusterDomain(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	lic, err := verifyLicense(jwtStr, clusterDomain, now)
	if err != nil {
		zap.L().Debug("Could not verify the License of the Cluster", zap.Error(err))
		return nil, &LicenseUnavailableError{
			Reason: "The License of the Cluster is not valid",
		}
	}

	switch getLicenseState(lic, now) {
	case enterprisev1.ClusterConfig_Status_LicenseInfo_ACTIVE:
	case enterprisev1.ClusterConfig_Status_LicenseInfo_EXPIRED:
		return nil, &LicenseUnavailableError{
			Reason: "The License of the Cluster has expired",
		}
	default:
		return nil, &LicenseUnavailableError{
			Reason: "The License of the Cluster is not active yet",
		}
	}

	return &LicenseCredential{
		JWT:           jwtStr,
		License:       lic,
		ClusterDomain: clusterDomain,
	}, nil
}
