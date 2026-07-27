// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package ovh

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	ovhapi "github.com/ovh/go-ovh/ovh"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Provider struct {
	c      *ovhapi.Client
	p      *enterprisev1.DNSProvider
	domain string
}

type record struct {
	ID        uint64 `json:"id"`
	FieldType string `json:"fieldType"`
	SubDomain string `json:"subDomain"`
	Target    string `json:"target"`
	TTL       int64  `json:"ttl"`
	Zone      string `json:"zone"`
}

type recordCreate struct {
	FieldType string `json:"fieldType"`
	SubDomain string `json:"subDomain"`
	Target    string `json:"target"`
	TTL       int64  `json:"ttl"`
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, p *enterprisev1.DNSProvider, domain string) (*Provider, error) {

	ovhCfg := p.Spec.GetOvh()

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: ovhCfg.ApplicationSecret.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	client, err := ovhapi.NewClient(
		ovhCfg.Endpoint,
		ovhCfg.ApplicationKey,
		uenterprisev1.ToSecret(sec).GetValueStr(),
		ovhCfg.ConsumerKey,
	)
	if err != nil {
		return nil, err
	}

	return &Provider{c: client, domain: domain, p: p}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	changed, err := p.setRecords(ctx, zoneID, "CNAME", name, []string{
		fmt.Sprintf("%s.", strings.TrimSuffix(val, ".")),
	})
	if err != nil {
		return err
	}

	if changed {
		if err := p.refresh(ctx, zoneID); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) Set(ctx context.Context, domain string, ipAddrs []string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

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
			return errors.Errorf("Invalid IP address: %s", ipAddr)
		}
	}

	changedA, err := p.setRecords(ctx, zoneID, "A", domain, v4Addrs)
	if err != nil {
		return err
	}

	changedAAAA, err := p.setRecords(ctx, zoneID, "AAAA", domain, v6Addrs)
	if err != nil {
		return err
	}

	if changedA || changedAAAA {
		if err := p.refresh(ctx, zoneID); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) setRecords(ctx context.Context, zoneID, typ, name string, vals []string) (bool, error) {

	subDomain := trimDomain(name, zoneID)

	records, err := p.getRecords(ctx, zoneID, typ, subDomain)
	if err != nil {
		return false, err
	}

	desired := make(map[string]struct{}, len(vals))
	for _, val := range vals {
		desired[val] = struct{}{}
	}

	found := make(map[string]struct{}, len(records))
	changed := false

	for _, record := range records {
		if _, ok := desired[record.Target]; ok {
			if _, ok := found[record.Target]; !ok {
				found[record.Target] = struct{}{}
				continue
			}
		}

		if err := p.c.DeleteWithContext(ctx,
			fmt.Sprintf("/domain/zone/%s/record/%d", url.PathEscape(zoneID), record.ID), nil); err != nil {
			return false, err
		}
		changed = true
	}

	for val := range desired {
		if _, ok := found[val]; ok {
			continue
		}

		if err := p.c.PostWithContext(ctx,
			fmt.Sprintf("/domain/zone/%s/record", url.PathEscape(zoneID)),
			&recordCreate{
				FieldType: typ,
				SubDomain: subDomain,
				Target:    val,
				TTL:       300,
			}, nil); err != nil {
			return false, err
		}
		changed = true
	}

	return changed, nil
}

func (p *Provider) getRecords(ctx context.Context, zoneID, typ, subDomain string) ([]record, error) {

	var ids []uint64
	if err := p.c.GetWithContext(ctx,
		fmt.Sprintf("/domain/zone/%s/record?fieldType=%s&subDomain=%s",
			url.PathEscape(zoneID), url.QueryEscape(typ), url.QueryEscape(subDomain)),
		&ids); err != nil {
		return nil, err
	}

	ret := make([]record, 0, len(ids))
	for _, id := range ids {
		var record record
		if err := p.c.GetWithContext(ctx,
			fmt.Sprintf("/domain/zone/%s/record/%d", url.PathEscape(zoneID), id),
			&record); err != nil {
			return nil, err
		}
		ret = append(ret, record)
	}

	return ret, nil
}

func (p *Provider) refresh(ctx context.Context, zoneID string) error {
	return p.c.PostWithContext(ctx,
		fmt.Sprintf("/domain/zone/%s/refresh", url.PathEscape(zoneID)), nil, nil)
}

func trimDomain(arg, zoneID string) string {
	if arg == zoneID {
		return ""
	} else {
		return strings.TrimSuffix(arg, fmt.Sprintf(".%s", zoneID))
	}
}

func (p *Provider) getZoneID(ctx context.Context) (string, error) {

	var zones []string
	if err := p.c.GetWithContext(ctx, "/domain/zone", &zones); err != nil {
		return "", err
	}

	var ret string
	for _, zone := range zones {
		if p.domain == zone || strings.HasSuffix(p.domain, fmt.Sprintf(".%s", zone)) {
			if len(zone) > len(ret) {
				ret = zone
			}
		}
	}

	if ret == "" {
		return "", errors.Errorf("Could not find domain")
	}

	return ret, nil
}
