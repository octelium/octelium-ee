// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package dnss

import (
	"context"
	"fmt"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnsctl"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func SetServiceCNAME(ctx context.Context,
	octeliumC octeliumc.ClientInterface, svc *corev1.Service,
	cc *corev1.ClusterConfig,
	provider *enterprisev1.DNSProvider) error {

	if svc.Status.RegionRef == nil {
		return errors.Errorf("nil regionRef")
	}

	region, err := octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Uid: svc.Status.RegionRef.Uid})
	if err != nil {
		return err
	}

	regionFQDN := fmt.Sprintf("%s.%s", region.Status.PublicHostname, cc.Status.Domain)
	svcFQDN := func() string {
		if svc.Status.PrimaryHostname == "" {
			return cc.Status.Domain
		}
		return fmt.Sprintf("%s.%s", svc.Status.PrimaryHostname, cc.Status.Domain)
	}()

	if err := dnsctl.SetCNAME(ctx, octeliumC, provider, cc, svcFQDN, regionFQDN); err != nil {
		zap.S().Errorf("Could not set CNAME of domain: %s. %+v", svcFQDN, err)
		return err
	}

	if ucorev1.ToService(svc).IsManagedService() && svc.Status.ManagedService.HasSubdomain {
		if err := dnsctl.SetCNAME(ctx, octeliumC, provider, cc, fmt.Sprintf("*.%s", svcFQDN), svcFQDN); err != nil {
			zap.S().Errorf("Could not set Workspace CNAME of domain: %s. %+v", svcFQDN, err)
			return err
		}
	}

	/*
		if region.Metadata.Name == "default" {
			zap.S().Debugf("Service %s is in default Region. Setting the default CNAME entry", svc.Metadata.Name)
			svcDefaultFQDN := fmt.Sprintf("%s.%s", ucorev1.ToService(svc).Name(), cc.Status.Domain)
			if err := dnsctl.SetCNAME(ctx, octeliumC, provider, cc, svcDefaultFQDN, regionFQDN); err != nil {
				zap.S().Errorf("Could not set CNAME of domain: %s. %+v", svcFQDN, err)
				return err
			}

			if ucorev1.ToService(svc).IsManagedService() && svc.Status.ManagedService.HasSubdomain {
				if err := dnsctl.SetCNAME(ctx,
					octeliumC, provider, cc, fmt.Sprintf("*.%s", svcDefaultFQDN), svcDefaultFQDN); err != nil {
					zap.S().Errorf("Could not set Workspace CNAME of domain: %s. %+v", svcDefaultFQDN, err)
					return err
				}
			}
		}
	*/

	return nil
}
