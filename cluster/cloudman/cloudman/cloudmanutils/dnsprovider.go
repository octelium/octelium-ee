// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cloudmanutils

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
)

func GetDefaultDNSProvider(ctx context.Context, octeliumC octeliumc.ClientInterface) (*enterprisev1.DNSProvider, error) {

	/*
		cc, err := octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		if err != nil {
			return nil, err
		}
		if cc.Spec.Dns == nil || cc.Spec.Dns.DefaultDNSProvider == "" {
			return octeliumC.EnterpriseC().GetDNSProvider(ctx, &rmetav1.GetOptions{Name: "default"})
		}
	*/

	return octeliumC.EnterpriseC().GetDNSProvider(ctx, &rmetav1.GetOptions{Name: "default"})
}

func GetDefaultCertificateIssuer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*enterprisev1.CertificateIssuer, error) {

	return octeliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{Name: "default"})
}
