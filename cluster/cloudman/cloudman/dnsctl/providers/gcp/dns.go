// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package gcp

import (
	"context"
	"sort"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	gcpdns "google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
)

const recordTTL = 300

type Provider struct {
	c      *gcpdns.Service
	p      *enterprisev1.DNSProvider
	domain string
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, p *enterprisev1.DNSProvider, domain string) (*Provider, error) {

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: p.Spec.GetGoogle().ServiceAccount.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	client, err := gcpdns.NewService(ctx,
		option.WithCredentialsJSON([]byte(uenterprisev1.ToSecret(sec).GetValueStr())))
	if err != nil {
		return nil, err
	}

	zap.S().Debugf("Successfully obtained Google Cloud DNS client")

	return &Provider{c: client, domain: domain, p: p}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zone, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	return p.setRecords(ctx, zone, "CNAME", normalizeFQDN(name), []string{normalizeFQDN(val)})
}

func (p *Provider) Set(ctx context.Context, domain string, ipAddrs []string) error {
	zone, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	v4Addrs, v6Addrs, err := splitIPAddresses(ipAddrs)
	if err != nil {
		return err
	}

	if err := p.setRecords(ctx, zone, "A", normalizeFQDN(domain), v4Addrs); err != nil {
		return err
	}

	return p.setRecords(ctx, zone, "AAAA", normalizeFQDN(domain), v6Addrs)
}

func (p *Provider) Delete(ctx context.Context, domain string) error {
	zone, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	name := normalizeFQDN(domain)

	if err := p.setRecords(ctx, zone, "A", name, nil); err != nil {
		return err
	}

	return p.setRecords(ctx, zone, "AAAA", name, nil)
}

func (p *Provider) setRecords(ctx context.Context, zone *gcpdns.ManagedZone, typ, name string, vals []string) error {

	existing, err := p.listRecords(ctx, zone, typ, name)
	if err != nil {
		return err
	}

	desired := normalizeValues(vals)

	if len(existing) == 1 &&
		existing[0].Ttl == recordTTL &&
		equalValues(normalizeValues(existing[0].Rrdatas), desired) {
		return nil
	}

	change := &gcpdns.Change{
		Deletions: existing,
	}

	if len(desired) > 0 {
		change.Additions = []*gcpdns.ResourceRecordSet{
			{
				Name:    name,
				Ttl:     recordTTL,
				Rrdatas: desired,
				Type:    typ,
			},
		}
	}

	if len(change.Additions) == 0 && len(change.Deletions) == 0 {
		return nil
	}

	zap.S().Debugf("Applying Google Cloud DNS change for the %s record %s", typ, name)

	_, err = p.c.Changes.Create(p.p.Spec.GetGoogle().Project, zone.Name, change).Context(ctx).Do()

	return err
}

func (p *Provider) listRecords(ctx context.Context, zone *gcpdns.ManagedZone, typ, name string) ([]*gcpdns.ResourceRecordSet, error) {

	var ret []*gcpdns.ResourceRecordSet

	call := p.c.ResourceRecordSets.List(p.p.Spec.GetGoogle().Project, zone.Name).
		Name(name).
		Type(typ)

	if err := call.Pages(ctx, func(page *gcpdns.ResourceRecordSetsListResponse) error {
		for _, record := range page.Rrsets {
			if record.Name != name || record.Type != typ {
				continue
			}
			ret = append(ret, record)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (p *Provider) getZoneID(ctx context.Context) (*gcpdns.ManagedZone, error) {

	domain := normalizeDomain(p.domain)

	var ret *gcpdns.ManagedZone
	var retName string

	if err := p.c.ManagedZones.List(p.p.Spec.GetGoogle().Project).
		Pages(ctx, func(page *gcpdns.ManagedZonesListResponse) error {
			for _, zone := range page.ManagedZones {
				name := normalizeDomain(zone.DnsName)
				if !domainMatchesZone(domain, name) {
					continue
				}

				if len(name) > len(retName) {
					ret = zone
					retName = name
				}
			}
			return nil
		}); err != nil {
		return nil, err
	}

	if ret == nil {
		return nil, errors.Errorf("Could not find a Google Cloud DNS managed zone for the domain: %s", p.domain)
	}

	zap.S().Debugf("Got Google Cloud DNS managed zone %s for the domain %s", ret.Name, p.domain)

	return ret, nil
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

func normalizeValues(vals []string) []string {
	set := make(map[string]struct{}, len(vals))
	for _, val := range vals {
		if val == "" {
			continue
		}
		set[val] = struct{}{}
	}

	ret := make([]string, 0, len(set))
	for val := range set {
		ret = append(ret, val)
	}

	sort.Strings(ret)
	return ret
}

func equalValues(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for idx := range a {
		if a[idx] != b[idx] {
			return false
		}
	}

	return true
}

func normalizeFQDN(arg string) string {
	return normalizeDomain(arg) + "."
}

func normalizeDomain(arg string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(arg), "."))
}

func domainMatchesZone(domain, zone string) bool {
	return domain == zone || strings.HasSuffix(domain, "."+zone)
}
