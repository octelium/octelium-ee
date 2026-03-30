// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/asaskevich/govalidator"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Provider struct {
	zoneC   *dns.ZonesClient
	recordC *dns.RecordSetsClient
	c       *enterprisev1.DNSProvider
	domain  string
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, c *enterprisev1.DNSProvider, domain string) (*Provider, error) {

	azureCfg := c.Spec.GetAzure()

	recordC := dns.NewRecordSetsClient(azureCfg.SubscriptionID)
	zoneC := dns.NewZonesClient(azureCfg.SubscriptionID)

	var azureCloud string

	switch azureCfg.Cloud {
	case "public", "":
		azureCloud = "AZUREPUBLICCLOUD"
	case "china":
		azureCloud = "AZURECHINACLOUD"
	case "german":
		azureCloud = "AZUREGERMANCLOUD"
	case "usgovernment":
		azureCloud = "AZUREUSGOVERNMENTCLOUD"
	}

	env, err := azure.EnvironmentFromName(azureCloud)
	if err != nil {
		return nil, err
	}
	oauthCfg, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, azureCfg.TenantID)
	if err != nil {
		return nil, err
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: azureCfg.ClientSecret.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	token, err := adal.NewServicePrincipalToken(*oauthCfg, azureCfg.ClientID,
		uenterprisev1.ToSecret(sec).GetValueStr(), env.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}
	recordC.Authorizer = autorest.NewBearerAuthorizer(token)
	zoneC.Authorizer = autorest.NewBearerAuthorizer(token)

	return &Provider{recordC: &recordC, zoneC: &zoneC, c: c, domain: domain}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	if err := p.setRecord(ctx, dns.CNAME, zoneID, name, val); err != nil {
		return err
	}

	return nil
}

func (p *Provider) Set(ctx context.Context, domain string, ipAddrs []string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	for _, ipAddr := range ipAddrs {
		zap.S().Debugf("Setting DNS record for the IP:%s", ipAddr)
		recordType := func() dns.RecordType {
			if govalidator.IsIPv4(ipAddr) {
				return dns.A
			}
			return dns.AAAA
		}()
		if err := p.setRecord(ctx, recordType, zoneID, domain, ipAddr); err != nil {
			return err
		}
	}

	return nil
}

func trimDomain(arg, zoneID string) string {
	if arg == zoneID {
		return "@"
	} else {
		return strings.TrimSuffix(arg, fmt.Sprintf(".%s", zoneID))
	}
}

func (p *Provider) setRecord(ctx context.Context, typ dns.RecordType, zone *dns.Zone, name, val string) error {

	rs := dns.RecordSet{
		RecordSetProperties: &dns.RecordSetProperties{
			TTL: utils_types.Int64ToPtr(300),
		},
	}

	switch typ {
	case dns.A:
		records := []dns.ARecord{
			{
				Ipv4Address: &val,
			},
		}

		rs.RecordSetProperties.ARecords = &records
	case dns.AAAA:
		records := []dns.AaaaRecord{
			{
				Ipv6Address: &val,
			},
		}

		rs.RecordSetProperties.AaaaRecords = &records
	case dns.CNAME:
		rs.RecordSetProperties.CnameRecord = &dns.CnameRecord{
			Cname: utils_types.StrToPtr(val),
		}
	}

	_, err := p.recordC.CreateOrUpdate(ctx,
		p.c.Spec.GetAzure().ResourceGroupName, *zone.Name, trimDomain(name, *zone.Name),
		typ, rs, "", "")
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (*dns.Zone, error) {

	out, err := p.zoneC.ListByResourceGroup(ctx,
		p.c.Spec.GetAzure().ResourceGroupName, utils_types.Int32ToPtr(300))
	if err != nil {
		return nil, err
	}

	zones := out.Values()

	for _, zone := range zones {
		if strings.HasSuffix(p.domain, *zone.Name) {
			return &zone, nil
		}
	}

	return nil, errors.Errorf("Could not find zone")
}
