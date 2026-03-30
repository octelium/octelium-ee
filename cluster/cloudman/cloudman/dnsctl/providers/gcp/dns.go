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
	"fmt"
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

	client, err := gcpdns.NewService(context.Background(),
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

	if err := p.setRecord(ctx, zoneID, "CNAME", fmt.Sprintf("%s.", name), fmt.Sprintf("%s.", val)); err != nil {
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
		recordType := func() string {
			if govalidator.IsIPv4(ipAddr) {
				return "A"
			}
			return "AAAA"
		}()

		if err := p.setRecord(ctx, zoneID, recordType, fmt.Sprintf("%s.", domain), ipAddr); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) setRecord(ctx context.Context, zone *gcpdns.ManagedZone, typ, name, val string) error {

	o, err := p.c.ResourceRecordSets.List(p.p.Spec.GetGoogle().Project, zone.Name).Context(ctx).Do()
	if err != nil {
		return err
	}

	change := &gcpdns.Change{
		Additions: []*gcpdns.ResourceRecordSet{
			{
				Name:    name,
				Ttl:     300,
				Rrdatas: []string{val},
				Type:    typ,
			},
		},
	}

	for _, rec := range o.Rrsets {
		if rec.Name == name && rec.Type == typ {
			zap.S().Debugf("Deleting record: %+v", rec)
			change.Deletions = append(change.Deletions, rec)
		}
	}

	_, err = p.c.Changes.Create(p.p.Spec.GetGoogle().Project, zone.Name, change).Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) Delete(ctx context.Context, domain string) error {

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (*gcpdns.ManagedZone, error) {

	out, err := p.c.ManagedZones.List(p.p.Spec.GetGoogle().Project).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	zap.S().Debugf("Listed zones: %+v", out)

	zoneID := func() *gcpdns.ManagedZone {
		for _, zone := range out.ManagedZones {
			if strings.HasSuffix(fmt.Sprintf("%s.", p.domain), zone.DnsName) {
				return zone
			}
		}
		return nil
	}()
	if zoneID == nil {
		return nil, errors.Errorf("Could not find zone ID")
	}

	return zoneID, nil
}
