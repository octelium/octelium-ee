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
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Provider struct {
	c      *awsroute53.Client
	p      *enterprisev1.DNSProvider
	domain string
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, p *enterprisev1.DNSProvider, domain string) (*Provider, error) {

	awsCfg := p.Spec.GetAws()

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: awsCfg.SecretAccessKey.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(awsCfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			awsCfg.AccessKeyID, uenterprisev1.ToSecret(sec).GetValueStr(), "")))
	if err != nil {
		return nil, err
	}

	client := awsroute53.NewFromConfig(cfg)
	zap.S().Debugf("Successfully obtained Route53 client")
	return &Provider{c: client, domain: domain, p: p}, nil
}

func (p *Provider) SetCNAME(ctx context.Context, name, val string) error {
	zoneID, err := p.getZoneID(ctx)
	if err != nil {
		return err
	}

	zap.S().Debugf("Got zone ID: %s", zoneID)

	if err := p.setRecord(ctx, types.RRTypeCname, zoneID, name, val); err != nil {
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
		recordType := func() types.RRType {
			if govalidator.IsIPv4(ipAddr) {
				return types.RRTypeA
			}
			return types.RRTypeAaaa
		}()
		if err := p.setRecord(ctx, recordType, zoneID, domain, ipAddr); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) setRecord(ctx context.Context, typ types.RRType, zoneID, name, val string) error {

	_, err := p.c.ChangeResourceRecordSets(ctx, &awsroute53.ChangeResourceRecordSetsInput{
		HostedZoneId: utils_types.StrToPtr(zoneID),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: types.ChangeActionUpsert,
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: &name,
						TTL:  utils_types.Int64ToPtr(300),
						Type: typ,
						ResourceRecords: []types.ResourceRecord{
							{
								Value: &val,
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) Delete(ctx context.Context, domain string) error {

	return nil
}

func (p *Provider) getZoneID(ctx context.Context) (string, error) {

	out, err := p.c.ListHostedZones(ctx, &awsroute53.ListHostedZonesInput{
		MaxItems: utils_types.Int32ToPtr(300),
	})
	if err != nil {
		return "", err
	}

	zoneID := func() string {
		for _, zone := range out.HostedZones {
			if strings.HasSuffix(fmt.Sprintf("%s.", p.domain), *zone.Name) {
				return *zone.Id
			}
		}
		return ""
	}()
	if zoneID == "" {
		return "", errors.Errorf("Could not find zone ID")
	}
	return zoneID, nil
}
