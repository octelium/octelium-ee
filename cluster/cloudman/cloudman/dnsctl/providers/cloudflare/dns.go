// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cloudflare

import (
	"context"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/cloudflare/cloudflare-go"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Provider struct {
	c      *cloudflare.API
	p      *enterprisev1.DNSProvider
	domain string
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, p *enterprisev1.DNSProvider, domain string) (*Provider, error) {

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: p.Spec.GetCloudflare().ApiToken.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	client, err := cloudflare.NewWithAPIToken(uenterprisev1.ToSecret(sec).GetValueStr())
	if err != nil {
		return nil, err
	}
	zap.S().Debugf("Successfully obtained Cloudflare client")
	return &Provider{c: client, domain: domain, p: p}, nil
}

func (p *Provider) Set(ctx context.Context, domain string, ipAddrs []string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	zap.S().Debugf("Got zone ID: %s", zoneID)

	v4Addrs, v6Addrs, err := splitIPAddresses(ipAddrs)
	if err != nil {
		return err
	}

	if err := p.setRecords(ctx, "A", zoneID, domain, v4Addrs); err != nil {
		return err
	}

	if err := p.setRecords(ctx, "AAAA", zoneID, domain, v6Addrs); err != nil {
		return err
	}

	return nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	zap.S().Debugf("Got zone ID: %s", zoneID)

	if err := p.setRecords(ctx, "CNAME", zoneID, name, []string{val}); err != nil {
		return err
	}

	return nil
}

func (p *Provider) setRecords(ctx context.Context, typ, zoneID, name string, vals []string) error {

	records, _, err := p.c.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{})
	if err != nil {
		return err
	}

	desired := uniqueValues(vals)
	found := make(map[string]struct{}, len(desired))

	for _, record := range records {
		if !strings.EqualFold(record.Name, name) || record.ZoneID != zoneID || record.Type != typ {
			continue
		}

		if _, ok := desired[record.Content]; ok {
			if _, ok := found[record.Content]; !ok {
				found[record.Content] = struct{}{}
				continue
			}
		}

		if err := p.c.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), record.ID); err != nil {
			return err
		}
	}

	for _, val := range vals {
		if _, ok := found[val]; ok {
			continue
		}
		if _, ok := desired[val]; !ok {
			continue
		}

		if _, err := p.c.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.CreateDNSRecordParams{
			Type:    typ,
			Name:    name,
			Content: val,
			ZoneID:  zoneID,
			Proxied: utils_types.BoolToPtr(false),
		}); err != nil {
			return err
		}

		found[val] = struct{}{}
	}

	return nil
}

func (p *Provider) Delete(ctx context.Context, domain string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	if err := p.setRecords(ctx, "A", zoneID, domain, nil); err != nil {
		return err
	}

	if err := p.setRecords(ctx, "AAAA", zoneID, domain, nil); err != nil {
		return err
	}

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (string, error) {
	zones, err := p.c.ListZones(ctx)
	if err != nil {
		return "", err
	}

	domain := normalizeDomain(p.domain)
	var zoneID string
	var zoneName string

	for _, zone := range zones {
		name := normalizeDomain(zone.Name)
		if !domainMatchesZone(domain, name) {
			continue
		}

		if len(name) > len(zoneName) {
			zoneID = zone.ID
			zoneName = name
		}
	}

	if zoneID == "" {
		return "", errors.Errorf("Could not find zone ID")
	}

	return zoneID, nil
}

func splitIPAddresses(ipAddrs []string) ([]string, []string, error) {
	var v4Addrs []string
	var v6Addrs []string

	for _, ipAddr := range ipAddrs {
		switch {
		case govalidator.IsIPv4(ipAddr):
			v4Addrs = append(v4Addrs, ipAddr)
		case govalidator.IsIPv6(ipAddr):
			v6Addrs = append(v6Addrs, ipAddr)
		default:
			return nil, nil, errors.Errorf("Invalid IP address: %s", ipAddr)
		}
	}

	return v4Addrs, v6Addrs, nil
}

func uniqueValues(vals []string) map[string]struct{} {
	ret := make(map[string]struct{}, len(vals))
	for _, val := range vals {
		ret[val] = struct{}{}
	}
	return ret
}

func normalizeDomain(arg string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(arg), "."))
}

func domainMatchesZone(domain, zone string) bool {
	return domain == zone || strings.HasSuffix(domain, "."+zone)
}
