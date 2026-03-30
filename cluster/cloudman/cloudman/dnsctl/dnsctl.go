// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package dnsctl

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/cloudmanutils"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnsctl/providers/azure"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnsctl/providers/cloudflare"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnsctl/providers/digitalocean"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnsctl/providers/gcp"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnsctl/providers/linode"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnsctl/providers/route53"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
)

type providerInterface interface {
	Set(ctx context.Context, domain string, ipAddrs []string) error
	SetCNAME(ctx context.Context, name, val string) error
}

func SetClusterDomain(ctx context.Context, octeliumC octeliumc.ClientInterface, svc *k8scorev1.Service) error {

	provider, err := cloudmanutils.GetDefaultDNSProvider(ctx, octeliumC)
	if err != nil {
		return err
	}

	ccc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	ipAddrs := doGetIPs(svc)
	if len(ipAddrs) == 0 {
		return nil
	}

	i, err := getProvider(ctx, octeliumC, provider, ccc.Status.Domain)
	if err != nil {
		return err
	}
	if i == nil {
		return nil
	}

	return i.Set(ctx, ccc.Status.Domain, ipAddrs)
}

func SetCNAME(ctx context.Context, octeliumC octeliumc.ClientInterface, provider *enterprisev1.DNSProvider, ccc *corev1.ClusterConfig, name, value string) error {

	zap.S().Debugf("Setting CNAME for %s to %s", name, value)

	i, err := getProvider(ctx, octeliumC, provider, ccc.Status.Domain)
	if err != nil {
		return err
	}
	if i == nil {
		return nil
	}

	return i.SetCNAME(ctx, name, value)
}

func SetAddresses(ctx context.Context, octeliumC octeliumc.ClientInterface, provider *enterprisev1.DNSProvider, ccc *corev1.ClusterConfig, ipAddrs []string, domain string) error {

	zap.S().Debugf("Setting DNS for the domain: %s to the IPs: %+v", domain, ipAddrs)

	i, err := getProvider(ctx, octeliumC, provider, ccc.Status.Domain)
	if err != nil {
		return err
	}
	if i == nil {
		return nil
	}

	return i.Set(ctx, domain, ipAddrs)
}

func doGetIPs(svc *k8scorev1.Service) []string {
	var ret []string

	if svc.Spec.ExternalIPs != nil {
		return svc.Spec.ExternalIPs
	}

	for _, ing := range svc.Status.LoadBalancer.Ingress {
		ret = append(ret, ing.IP)
	}

	return ret
}

func getProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, provider *enterprisev1.DNSProvider, domain string) (providerInterface, error) {

	switch provider.Spec.Type.(type) {
	case *enterprisev1.DNSProvider_Spec_Cloudflare_:
		if provider.Spec.GetCloudflare() == nil {
			zap.S().Warnf("Cluster-config Cloudflare DNS options not set. Nothing to be done")
			return nil, nil
		}
		return cloudflare.NewProvider(ctx, octeliumC, provider, domain)
	case *enterprisev1.DNSProvider_Spec_Digitalocean:
		if provider.Spec.GetDigitalocean() == nil {
			zap.S().Warnf("Cluster-config DigitalOcean DNS options not set. Nothing to be done")
			return nil, nil
		}
		return digitalocean.NewProvider(ctx, octeliumC, provider, domain)
	case *enterprisev1.DNSProvider_Spec_Aws:
		if provider.Spec.GetAws() == nil {
			zap.S().Warnf("Cluster-config AWS DNS options not set. Nothing to be done")
			return nil, nil
		}
		return route53.NewProvider(ctx, octeliumC, provider, domain)
	case *enterprisev1.DNSProvider_Spec_Linode_:
		if provider.Spec.GetLinode() == nil {
			zap.S().Warnf("Cluster-config Linode DNS options not set. Nothing to be done")
			return nil, nil
		}
		return linode.NewProvider(ctx, octeliumC, provider, domain)
	case *enterprisev1.DNSProvider_Spec_Azure_:
		if provider.Spec.GetAzure() == nil {
			zap.S().Warnf("Cluster-config Azure DNS options not set. Nothing to be done")
			return nil, nil
		}
		return azure.NewProvider(ctx, octeliumC, provider, domain)
	case *enterprisev1.DNSProvider_Spec_Google_:
		if provider.Spec.GetGoogle() == nil {
			zap.S().Warnf("Cluster-config Google DNS options not set. Nothing to be done")
			return nil, nil
		}
		return gcp.NewProvider(ctx, octeliumC, provider, domain)
	default:
		zap.S().Warnf("DNS provider unsupported. Skipping...")
		return nil, nil
	}
}
