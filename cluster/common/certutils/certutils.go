// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package certutils

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func DoIssueCertificate(ctx context.Context, octeliumC octeliumc.ClientInterface, crt *enterprisev1.Certificate) (*enterprisev1.Certificate, error) {

	switch crt.Spec.Mode {
	case enterprisev1.Certificate_Spec_MANAGED:
	default:
		return crt, nil
	}

	if crt.Status.Issuance != nil {
		crt.Status.LastIssuances = append([]*enterprisev1.Certificate_Status_Issuance{
			crt.Status.Issuance,
		}, crt.Status.LastIssuances...)
		if len(crt.Status.LastIssuances) > 200 {
			crt.Status.LastIssuances = crt.Status.LastIssuances[:200]
		}
	}

	crt.Status.Issuance = &enterprisev1.Certificate_Status_Issuance{
		CreatedAt: pbutils.Now(),
		State:     enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
	}

	crt, err := octeliumC.EnterpriseC().UpdateCertificate(ctx, crt)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return crt, nil
}
