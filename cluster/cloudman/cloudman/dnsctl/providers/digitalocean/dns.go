// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package digitalocean

import (
	"context"
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/digitalocean/godo"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	recordTTL    = 300
	listPageSize = 200
	maxListPages = 100
)

type Provider struct {
	c      *godo.Client
	p      *enterprisev1.DNSProvider
	domain string
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, p *enterprisev1.DNSProvider, domain string) (*Provider, error) {

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: p.Spec.GetDigitalocean().ApiToken.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	return &Provider{
		c:      godo.NewFromToken(uenterprisev1.ToSecret(sec).GetValueStr()),
		domain: domain,
		p:      p,
	}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	target := fmt.Sprintf("%s.", strings.TrimSuffix(val, "."))

	return p.setRecords(ctx, "CNAME", zoneID, name, []string{target})
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

	if err := p.setRecords(ctx, "A", zoneID, domain, v4Addrs); err != nil {
		return err
	}

	return p.setRecords(ctx, "AAAA", zoneID, domain, v6Addrs)
}

func (p *Provider) Delete(ctx context.Context, domain string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	if err := p.setRecords(ctx, "A", zoneID, domain, nil); err != nil {
		return err
	}

	return p.setRecords(ctx, "AAAA", zoneID, domain, nil)
}

func (p *Provider) setRecords(ctx context.Context, typ, zoneID, name string, vals []string) error {

	trimmedName, err := trimDomain(name, zoneID)
	if err != nil {
		return err
	}

	records, err := p.listRecords(ctx, zoneID)
	if err != nil {
		return err
	}

	desired := uniqueValues(vals)
	found := make(map[string]struct{}, len(desired))

	for _, record := range records {
		if record.Type != typ || !strings.EqualFold(record.Name, trimmedName) {
			continue
		}

		if _, ok := desired[record.Data]; ok {
			if _, ok := found[record.Data]; !ok {
				found[record.Data] = struct{}{}
				continue
			}
		}

		if _, err := p.c.Domains.DeleteRecord(ctx, zoneID, record.ID); err != nil {
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

		if _, _, err := p.c.Domains.CreateRecord(ctx, zoneID, &godo.DomainRecordEditRequest{
			Type: typ,
			Name: trimmedName,
			Data: val,
			TTL:  recordTTL,
		}); err != nil {
			return err
		}

		found[val] = struct{}{}
	}

	return nil
}

func (p *Provider) listRecords(ctx context.Context, zoneID string) ([]godo.DomainRecord, error) {

	var ret []godo.DomainRecord

	opts := &godo.ListOptions{
		Page:    1,
		PerPage: listPageSize,
	}

	for range maxListPages {
		records, resp, err := p.c.Domains.Records(ctx, zoneID, opts)
		if err != nil {
			return nil, err
		}

		ret = append(ret, records...)

		if resp == nil || resp.Links == nil || resp.Links.IsLastPage() {
			return ret, nil
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		opts.Page = page + 1
	}

	return nil, errors.Errorf("Too many DigitalOcean DNS record pages for the zone: %s", zoneID)
}

func (p *Provider) getZoneID(ctx context.Context) (string, error) {

	domain := normalizeDomain(p.domain)

	var ret string
	var retName string

	opts := &godo.ListOptions{
		Page:    1,
		PerPage: listPageSize,
	}

	for range maxListPages {
		zones, resp, err := p.c.Domains.List(ctx, opts)
		if err != nil {
			return "", err
		}

		for _, zone := range zones {
			name := normalizeDomain(zone.Name)
			if !domainMatchesZone(domain, name) {
				continue
			}

			if len(name) > len(retName) {
				ret = zone.Name
				retName = name
			}
		}

		if resp == nil || resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return "", err
		}

		opts.Page = page + 1
	}

	if ret == "" {
		return "", errors.Errorf("Could not find a DigitalOcean domain for: %s", p.domain)
	}

	zap.S().Debugf("Got DigitalOcean zone %s for the domain %s", ret, p.domain)

	return ret, nil
}

func trimDomain(arg, zoneID string) (string, error) {
	name := normalizeDomain(arg)
	zone := normalizeDomain(zoneID)

	if name == zone {
		return "@", nil
	}

	if strings.HasSuffix(name, "."+zone) {
		return name[:len(name)-len(zone)-1], nil
	}

	return "", errors.Errorf("The name %s is not within the zone %s", arg, zoneID)
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
		if val == "" {
			continue
		}
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
