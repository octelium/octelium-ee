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
		},
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

	nameTrimmed := strings.TrimSuffix(name, fmt.Sprintf(".%s", zoneID.Domain))

	if err := p.setRecord(ctx, zoneID.ID, linodego.RecordTypeCNAME, nameTrimmed, val); err != nil {
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
		recordType := func() linodego.DomainRecordType {
			if govalidator.IsIPv4(ipAddr) {
				return linodego.RecordTypeA
			}
			return linodego.RecordTypeAAAA
		}()

		if err := p.setRecord(ctx, zoneID.ID, recordType, domain, ipAddr); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) setRecord(ctx context.Context, zoneID int, typ linodego.DomainRecordType, name, val string) error {

	records, err := p.c.ListDomainRecords(ctx, zoneID, &linodego.ListOptions{})
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.Name == name && record.Type == typ && record.Target == val {
			return nil
		}
		if record.Name == name && record.Type == typ && record.Target != val {
			if err := p.c.DeleteDomainRecord(ctx, zoneID, record.ID); err != nil {
				return err
			}
		}
	}

	if _, err := p.c.CreateDomainRecord(ctx, zoneID, linodego.DomainRecordCreateOptions{
		Type:   typ,
		Name:   name,
		Target: val,
		TTLSec: 300,
	}); err != nil {
		return err
	}

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (*linodego.Domain, error) {

	zones, err := p.c.ListDomains(ctx, &linodego.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		if strings.HasSuffix(p.domain, zone.Domain) {
			return &zone, nil
		}
	}

	return nil, errors.Errorf("Could not find domain")
}
