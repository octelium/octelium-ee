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

	zap.S().Debugf("Created client")
	return &Provider{c: client, domain: domain, p: p}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	if err := p.setRecords(ctx, zoneID, "CNAME", normalizeFQDN(name), []string{normalizeFQDN(val)}); err != nil {
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

	if err := p.setRecords(ctx, zoneID, "A", normalizeFQDN(domain), v4Addrs); err != nil {
		return err
	}

	if err := p.setRecords(ctx, zoneID, "AAAA", normalizeFQDN(domain), v6Addrs); err != nil {
		return err
	}

	return nil
}

func (p *Provider) setRecords(ctx context.Context, zone *gcpdns.ManagedZone, typ, name string, vals []string) error {

	out, err := p.c.ResourceRecordSets.List(p.p.Spec.GetGoogle().Project, zone.Name).Context(ctx).Do()
	if err != nil {
		return err
	}

	desired := normalizeValues(vals)
	var existing []*gcpdns.ResourceRecordSet

	for _, record := range out.Rrsets {
		if record.Name == name && record.Type == typ {
			existing = append(existing, record)
		}
	}

	if len(existing) == 1 &&
		existing[0].Ttl == 300 &&
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
				Ttl:     300,
				Rrdatas: desired,
				Type:    typ,
			},
		}
	}

	if len(change.Additions) == 0 && len(change.Deletions) == 0 {
		return nil
	}

	_, err = p.c.Changes.Create(p.p.Spec.GetGoogle().Project, zone.Name, change).Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) Delete(ctx context.Context, domain string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	name := normalizeFQDN(domain)

	if err := p.setRecords(ctx, zoneID, "A", name, nil); err != nil {
		return err
	}

	if err := p.setRecords(ctx, zoneID, "AAAA", name, nil); err != nil {
		return err
	}

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (*gcpdns.ManagedZone, error) {

	out, err := p.c.ManagedZones.List(p.p.Spec.GetGoogle().Project).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	domain := normalizeDomain(p.domain)
	var ret *gcpdns.ManagedZone
	var retName string

	for _, zone := range out.ManagedZones {
		name := normalizeDomain(zone.DnsName)
		if !domainMatchesZone(domain, name) {
			continue
		}

		if len(name) > len(retName) {
			ret = zone
			retName = name
		}
	}

	if ret == nil {
		return nil, errors.Errorf("Could not find zone ID")
	}

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
	return strings.TrimSuffix(strings.TrimSpace(arg), ".") + "."
}

func normalizeDomain(arg string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(arg), "."))
}

func domainMatchesZone(domain, zone string) bool {
	return domain == zone || strings.HasSuffix(domain, "."+zone)
}
