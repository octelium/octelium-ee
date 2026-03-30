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

	for _, ipAddr := range ipAddrs {
		recordType := func() string {
			if govalidator.IsIPv4(ipAddr) {
				return "A"
			}
			return "AAAA"
		}()
		zap.S().Debugf("Setting DNS record for the IP:%s", ipAddr)
		if err := p.setRecord(ctx, recordType, zoneID, domain, ipAddr); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	zap.S().Debugf("Got zone ID: %s", zoneID)

	if err := p.setRecord(ctx, "CNAME", zoneID, name, val); err != nil {
		return err
	}

	return nil
}

func (p *Provider) setRecord(ctx context.Context, typ, zoneID, name, val string) error {

	records, _, err := p.c.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{})
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.Name == name && record.ZoneID == zoneID && record.Type == typ && record.Content == val {
			return nil
		}
		if record.Name == name && record.ZoneID == zoneID && record.Type == typ && record.Content != val {
			if err := p.c.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), record.ID); err != nil {
				return err
			}
		}
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

	return nil
}

func (p *Provider) Delete(ctx context.Context, domain string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}
	records, _, err := p.c.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{})
	if err != nil {
		return err
	}

	for _, record := range records {
		if record.Name == domain && (record.Type == "A" || record.Type == "AAAA") {
			if err := p.c.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), record.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (string, error) {
	zones, err := p.c.ListZones(ctx)
	if err != nil {
		return "", err
	}

	zoneID := func() string {
		for _, zone := range zones {
			if strings.HasSuffix(p.domain, zone.Name) {
				return zone.ID
			}
		}
		return ""
	}()
	if zoneID == "" {
		return "", errors.Errorf("Could not find zone ID")
	}
	return zoneID, nil
}
