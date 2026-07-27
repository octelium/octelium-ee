// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package linode

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/linode/linodego"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type Provider struct {
	c      *linodego.Client
	p      *enterprisev1.DNSProvider
	domain string
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, p *enterprisev1.DNSProvider, domain string) (*Provider, error) {

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: p.Spec.GetLinode().ApiToken.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: uenterprisev1.ToSecret(sec).GetValueStr()})

	linodeClient := linodego.NewClient(&http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
			Base:   http.DefaultTransport,
		},
		Timeout: 30 * time.Second,
	})

	if ldflags.IsDev() {
		linodeClient.SetDebug(true)
	}

	return &Provider{c: &linodeClient, domain: domain, p: p}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	if err := p.setRecords(ctx, zoneID.ID, linodego.RecordTypeCNAME,
		trimDomain(name, zoneID.Domain), []string{val}); err != nil {
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

	name := trimDomain(domain, zoneID.Domain)

	if err := p.setRecords(ctx, zoneID.ID, linodego.RecordTypeA, name, v4Addrs); err != nil {
		return err
	}

	if err := p.setRecords(ctx, zoneID.ID, linodego.RecordTypeAAAA, name, v6Addrs); err != nil {
		return err
	}

	return nil
}

func (p *Provider) setRecords(ctx context.Context, zoneID int, typ linodego.DomainRecordType, name string, vals []string) error {

	records, err := p.c.ListDomainRecords(ctx, zoneID, &linodego.ListOptions{})
	if err != nil {
		return err
	}

	desired := uniqueValues(vals)
	found := make(map[string]struct{}, len(desired))

	for _, record := range records {
		if record.Name != name || record.Type != typ {
			continue
		}

		if _, ok := desired[record.Target]; ok {
			if _, ok := found[record.Target]; !ok {
				found[record.Target] = struct{}{}
				continue
			}
		}

		if err := p.c.DeleteDomainRecord(ctx, zoneID, record.ID); err != nil {
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

		if _, err := p.c.CreateDomainRecord(ctx, zoneID, linodego.DomainRecordCreateOptions{
			Type:   typ,
			Name:   name,
			Target: val,
			TTLSec: 300,
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

	name := trimDomain(domain, zoneID.Domain)

	if err := p.setRecords(ctx, zoneID.ID, linodego.RecordTypeA, name, nil); err != nil {
		return err
	}

	if err := p.setRecords(ctx, zoneID.ID, linodego.RecordTypeAAAA, name, nil); err != nil {
		return err
	}

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (*linodego.Domain, error) {

	zones, err := p.c.ListDomains(ctx, &linodego.ListOptions{})
	if err != nil {
		return nil, err
	}

	domain := normalizeDomain(p.domain)
	var ret *linodego.Domain
	var retName string

	for idx := range zones {
		name := normalizeDomain(zones[idx].Domain)
		if !domainMatchesZone(domain, name) {
			continue
		}

		if len(name) > len(retName) {
			ret = &zones[idx]
			retName = name
		}
	}

	if ret == nil {
		return nil, errors.Errorf("Could not find domain")
	}

	return ret, nil
}

func trimDomain(arg, zoneID string) string {
	if normalizeDomain(arg) == normalizeDomain(zoneID) {
		return ""
	} else {
		return strings.TrimSuffix(arg, fmt.Sprintf(".%s", zoneID))
	}
}

func splitIPAddresses(ipAddrs []string) ([]string, []string, error) {
	var v4Addrs []string
	var v6Addrs []string

	for _, ipAddr := range ipAddrs {
		zap.S().Debugf("Setting DNS record for the IP:%s", ipAddr)
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
