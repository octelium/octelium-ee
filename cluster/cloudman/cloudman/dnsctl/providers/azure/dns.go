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
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/asaskevich/govalidator"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Provider struct {
	zoneC   *armdns.ZonesClient
	recordC *armdns.RecordSetsClient
	c       *enterprisev1.DNSProvider
	domain  string
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, c *enterprisev1.DNSProvider, domain string) (*Provider, error) {

	azureCfg := c.Spec.GetAzure()

	cloudCfg, err := getCloudConfig(azureCfg.Cloud)
	if err != nil {
		return nil, err
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: azureCfg.ClientSecret.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	clientOpts := azcore.ClientOptions{
		Cloud: cloudCfg,
	}

	cred, err := azidentity.NewClientSecretCredential(
		azureCfg.TenantID,
		azureCfg.ClientID,
		uenterprisev1.ToSecret(sec).GetValueStr(),
		&azidentity.ClientSecretCredentialOptions{
			ClientOptions: clientOpts,
		},
	)
	if err != nil {
		return nil, err
	}

	factory, err := armdns.NewClientFactory(
		azureCfg.SubscriptionID,
		cred,
		&arm.ClientOptions{
			ClientOptions: clientOpts,
		},
	)
	if err != nil {
		return nil, err
	}

	return &Provider{
		recordC: factory.NewRecordSetsClient(),
		zoneC:   factory.NewZonesClient(),
		c:       c,
		domain:  domain,
	}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	if err := p.setCNAME(ctx, zoneID, name, val); err != nil {
		return err
	}

	return nil
}

func (p *Provider) Set(ctx context.Context, domain string, ipAddrs []string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	v4Addrs, v6Addrs, err := splitIPAddresses(ipAddrs)
	if err != nil {
		return err
	}

	if err := p.setAddresses(ctx, armdns.RecordTypeA, zoneID, domain, v4Addrs); err != nil {
		return err
	}

	if err := p.setAddresses(ctx, armdns.RecordTypeAAAA, zoneID, domain, v6Addrs); err != nil {
		return err
	}

	return nil
}

func trimDomain(arg, zoneID string) string {
	arg = strings.TrimSuffix(strings.TrimSpace(arg), ".")
	zoneID = strings.TrimSuffix(strings.TrimSpace(zoneID), ".")

	if strings.EqualFold(arg, zoneID) {
		return "@"
	}

	suffix := fmt.Sprintf(".%s", zoneID)
	if len(arg) > len(suffix) && strings.EqualFold(arg[len(arg)-len(suffix):], suffix) {
		return arg[:len(arg)-len(suffix)]
	}

	return arg
}

func (p *Provider) setAddresses(ctx context.Context, typ armdns.RecordType, zone *armdns.Zone, name string, vals []string) error {

	if zone.Name == nil {
		return errors.Errorf("nil zone name")
	}

	if len(vals) == 0 {
		return p.deleteRecord(ctx, typ, zone, name)
	}

	rs := armdns.RecordSet{
		Properties: &armdns.RecordSetProperties{
			TTL: to.Ptr[int64](300),
		},
	}

	switch typ {
	case armdns.RecordTypeA:
		records := make([]*armdns.ARecord, 0, len(vals))
		for _, val := range vals {
			records = append(records, &armdns.ARecord{
				IPv4Address: to.Ptr(val),
			})
		}
		rs.Properties.ARecords = records
	case armdns.RecordTypeAAAA:
		records := make([]*armdns.AaaaRecord, 0, len(vals))
		for _, val := range vals {
			records = append(records, &armdns.AaaaRecord{
				IPv6Address: to.Ptr(val),
			})
		}
		rs.Properties.AaaaRecords = records
	default:
		return errors.Errorf("Invalid record type: %s", typ)
	}

	_, err := p.recordC.CreateOrUpdate(ctx,
		p.c.Spec.GetAzure().ResourceGroupName,
		*zone.Name,
		trimDomain(name, *zone.Name),
		typ,
		rs,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) setCNAME(ctx context.Context, zone *armdns.Zone, name, val string) error {

	if zone.Name == nil {
		return errors.Errorf("nil zone name")
	}

	rs := armdns.RecordSet{
		Properties: &armdns.RecordSetProperties{
			TTL: to.Ptr[int64](300),
			CnameRecord: &armdns.CnameRecord{
				Cname: to.Ptr(val),
			},
		},
	}

	_, err := p.recordC.CreateOrUpdate(ctx,
		p.c.Spec.GetAzure().ResourceGroupName,
		*zone.Name,
		trimDomain(name, *zone.Name),
		armdns.RecordTypeCNAME,
		rs,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) deleteRecord(ctx context.Context, typ armdns.RecordType, zone *armdns.Zone, name string) error {

	_, err := p.recordC.Delete(ctx,
		p.c.Spec.GetAzure().ResourceGroupName,
		*zone.Name,
		trimDomain(name, *zone.Name),
		typ,
		nil,
	)
	if err != nil && !isNotFound(err) {
		return err
	}

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (*armdns.Zone, error) {

	pager := p.zoneC.NewListByResourceGroupPager(
		p.c.Spec.GetAzure().ResourceGroupName,
		nil,
	)

	domain := normalizeDomain(p.domain)
	var ret *armdns.Zone
	var retName string

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, zone := range page.Value {
			if zone == nil || zone.Name == nil {
				continue
			}

			name := normalizeDomain(*zone.Name)
			if !domainMatchesZone(domain, name) {
				continue
			}

			if len(name) > len(retName) {
				ret = zone
				retName = name
			}
		}
	}

	if ret == nil {
		return nil, errors.Errorf("Could not find zone")
	}

	return ret, nil
}

func splitIPAddresses(ipAddrs []string) ([]string, []string, error) {
	var v4Addrs []string
	var v6Addrs []string

	v4Set := make(map[string]struct{})
	v6Set := make(map[string]struct{})

	for _, ipAddr := range ipAddrs {
		zap.S().Debugf("Setting DNS record for the IP:%s", ipAddr)
		switch {
		case govalidator.IsIPv4(ipAddr):
			if _, ok := v4Set[ipAddr]; !ok {
				v4Set[ipAddr] = struct{}{}
				v4Addrs = append(v4Addrs, ipAddr)
			}
		case govalidator.IsIPv6(ipAddr):
			if _, ok := v6Set[ipAddr]; !ok {
				v6Set[ipAddr] = struct{}{}
				v6Addrs = append(v6Addrs, ipAddr)
			}
		default:
			return nil, nil, errors.Errorf("Invalid IP address: %s", ipAddr)
		}
	}

	return v4Addrs, v6Addrs, nil
}

func isNotFound(err error) bool {
	var responseErr *azcore.ResponseError
	return errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound
}

func getCloudConfig(name string) (cloud.Configuration, error) {
	switch name {
	case "public", "":
		return cloud.AzurePublic, nil
	case "china":
		return cloud.AzureChina, nil
	case "usgovernment":
		return cloud.AzureGovernment, nil
	case "german":
		return cloud.Configuration{}, errors.Errorf("Azure Germany is no longer supported")
	default:
		return cloud.Configuration{}, errors.Errorf("Invalid azure cloud: %s", name)
	}
}

func normalizeDomain(arg string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(arg), "."))
}

func domainMatchesZone(domain, zone string) bool {
	return domain == zone || strings.HasSuffix(domain, "."+zone)
}
