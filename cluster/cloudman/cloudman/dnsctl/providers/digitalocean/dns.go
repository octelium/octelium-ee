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

	return &Provider{c: godo.NewFromToken(uenterprisev1.ToSecret(sec).GetValueStr()), domain: domain, p: p}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	zap.S().Debugf("Got zone ID: %s", zoneID)

	tVal := trimDomain(val, zoneID)
	if tVal != "@" {
		tVal = fmt.Sprintf("%s.", tVal)
	}

	if err := p.setRecord(ctx, "CNAME", zoneID, name, tVal); err != nil {
		return err
	}

	return nil
}

func (p *Provider) Set(ctx context.Context, domain string, ipAddrs []string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	zap.S().Debugf("Got zone ID: %s", zoneID)

	for _, ipAddr := range ipAddrs {
		zap.S().Debugf("Setting DNS record for the IP:%s", ipAddr)
		recordType := func() string {
			if govalidator.IsIPv4(ipAddr) {
				return "A"
			}
			return "AAAA"
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

func (p *Provider) setRecord(ctx context.Context, typ, zoneID, name, val string) error {

	trimmedName := trimDomain(name, zoneID)

	records, _, err := p.c.Domains.Records(ctx, zoneID, &godo.ListOptions{})
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.Name == trimmedName && record.Type == typ && record.Data == val {
			return nil
		}
		if record.Name == trimmedName && record.Type == typ && record.Data != val {
			if _, err := p.c.Domains.DeleteRecord(ctx, zoneID, record.ID); err != nil {
				return err
			}
		}
	}

	req := &godo.DomainRecordEditRequest{
		Type: typ,
		Name: trimmedName,
		Data: val,
		TTL:  300,
	}

	if _, _, err := p.c.Domains.CreateRecord(ctx, zoneID, req); err != nil {
		return err
	}

	return nil
}

func (p *Provider) Delete(ctx context.Context, domain string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}
	records, _, err := p.c.Domains.Records(ctx, zoneID, &godo.ListOptions{})
	if err != nil {
		return err
	}

	for _, record := range records {
		if record.Name == domain && (record.Type == "A" || record.Type == "AAAA") {
			if _, err := p.c.Domains.DeleteRecord(ctx, zoneID, record.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (string, error) {

	zones, _, err := p.c.Domains.List(ctx, &godo.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, zone := range zones {
		if strings.HasSuffix(p.domain, zone.Name) {
			return zone.Name, nil
		}
	}

	return "", errors.Errorf("Could not find domain")
}
