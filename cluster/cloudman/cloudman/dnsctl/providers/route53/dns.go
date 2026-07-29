// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package route53

import (
	"context"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	recordTTL       = 300
	listPageSize    = 100
	maxListPages    = 100
	defaultAWSRegin = "us-east-1"
)

type Provider struct {
	c      *awsroute53.Client
	p      *enterprisev1.DNSProvider
	domain string
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, p *enterprisev1.DNSProvider, domain string) (*Provider, error) {

	awsCfg := p.Spec.GetAws()
	if awsCfg == nil {
		return nil, errors.Errorf("Not an AWS DNSProvider")
	}

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: awsCfg.SecretAccessKey.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	region := awsCfg.Region
	if region == "" {
		region = defaultAWSRegin
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			awsCfg.AccessKeyID, uenterprisev1.ToSecret(sec).GetValueStr(), "")))
	if err != nil {
		return nil, err
	}

	if awsCfg.AssumeRoleARN != "" {
		cfg.Credentials = aws.NewCredentialsCache(
			stscreds.NewAssumeRoleProvider(sts.NewFromConfig(cfg), awsCfg.AssumeRoleARN))
	}

	zap.S().Debugf("Successfully obtained Route53 client")

	return &Provider{
		c:      awsroute53.NewFromConfig(cfg),
		domain: domain,
		p:      p,
	}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	return p.setRecords(ctx, zoneID, types.RRTypeCname, name, []string{normalizeFQDN(val)})
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

	if err := p.setRecords(ctx, zoneID, types.RRTypeA, domain, v4Addrs); err != nil {
		return err
	}

	return p.setRecords(ctx, zoneID, types.RRTypeAaaa, domain, v6Addrs)
}

func (p *Provider) Delete(ctx context.Context, domain string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	if err := p.setRecords(ctx, zoneID, types.RRTypeA, domain, nil); err != nil {
		return err
	}

	return p.setRecords(ctx, zoneID, types.RRTypeAaaa, domain, nil)
}

func (p *Provider) setRecords(ctx context.Context, zoneID string, typ types.RRType, name string, vals []string) error {

	fqdn := normalizeFQDN(name)

	vals = uniqueValues(vals)
	if len(vals) == 0 {
		return p.deleteRecordSet(ctx, zoneID, typ, fqdn)
	}

	records := make([]types.ResourceRecord, 0, len(vals))
	for _, val := range vals {
		records = append(records, types.ResourceRecord{
			Value: aws.String(val),
		})
	}

	zap.S().Debugf("Upserting Route53 %s record set %s with %d values", typ, fqdn, len(records))

	_, err := p.c.ChangeResourceRecordSets(ctx, &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: types.ChangeActionUpsert,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name:            aws.String(fqdn),
						Type:            typ,
						TTL:             aws.Int64(recordTTL),
						ResourceRecords: records,
					},
				},
			},
		},
	})

	return err
}

func (p *Provider) deleteRecordSet(ctx context.Context, zoneID string, typ types.RRType, fqdn string) error {

	out, err := p.c.ListResourceRecordSets(ctx, &awsroute53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(zoneID),
		StartRecordName: aws.String(fqdn),
		StartRecordType: typ,
		MaxItems:        aws.Int32(1),
	})
	if err != nil {
		return err
	}

	for _, rs := range out.ResourceRecordSets {
		if rs.Name == nil || rs.Type != typ {
			continue
		}

		if normalizeRecordName(*rs.Name) != fqdn {
			continue
		}

		if len(rs.ResourceRecords) == 0 {
			continue
		}

		zap.S().Debugf("Deleting Route53 %s record set %s", typ, fqdn)

		if _, err := p.c.ChangeResourceRecordSets(ctx, &awsroute53.ChangeResourceRecordSetsInput{
			HostedZoneId: aws.String(zoneID),
			ChangeBatch: &types.ChangeBatch{
				Changes: []types.Change{
					{
						Action:            types.ChangeActionDelete,
						ResourceRecordSet: &rs,
					},
				},
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (string, error) {

	domain := normalizeDomain(p.domain)

	var zoneID string
	var zoneName string

	input := &awsroute53.ListHostedZonesInput{
		MaxItems: aws.Int32(listPageSize),
	}

	for range maxListPages {
		out, err := p.c.ListHostedZones(ctx, input)
		if err != nil {
			return "", err
		}

		for _, zone := range out.HostedZones {
			if zone.Name == nil || zone.Id == nil {
				continue
			}

			if zone.Config != nil && zone.Config.PrivateZone {
				continue
			}

			name := normalizeDomain(*zone.Name)
			if !domainMatchesZone(domain, name) {
				continue
			}

			if len(name) > len(zoneName) {
				zoneID = trimHostedZoneID(*zone.Id)
				zoneName = name
			}
		}

		if !out.IsTruncated {
			break
		}

		input.Marker = out.NextMarker
	}

	if zoneID == "" {
		return "", errors.Errorf("Could not find a public Route53 hosted zone for the domain: %s", p.domain)
	}

	zap.S().Debugf("Got Route53 zone ID %s for the domain %s", zoneID, p.domain)

	return zoneID, nil
}

func trimHostedZoneID(id string) string {
	return strings.TrimPrefix(id, "/hostedzone/")
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

func uniqueValues(vals []string) []string {
	ret := make([]string, 0, len(vals))
	seen := make(map[string]struct{}, len(vals))

	for _, val := range vals {
		if val == "" {
			continue
		}
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		ret = append(ret, val)
	}

	return ret
}

func normalizeFQDN(arg string) string {
	return normalizeDomain(arg) + "."
}

func normalizeRecordName(arg string) string {
	return normalizeFQDN(strings.ReplaceAll(arg, `\052`, "*"))
}

func normalizeDomain(arg string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(arg), "."))
}

func domainMatchesZone(domain, zone string) bool {
	return domain == zone || strings.HasSuffix(domain, "."+zone)
}
