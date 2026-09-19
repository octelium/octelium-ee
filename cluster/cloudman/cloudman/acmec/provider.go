// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package acmec

import (
	"context"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	awscred "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/azuredns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/digitalocean"
	"github.com/go-acme/lego/v4/providers/dns/gcloud"
	"github.com/go-acme/lego/v4/providers/dns/linode"
	"github.com/go-acme/lego/v4/providers/dns/ovh"
	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/cloudmanutils"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

const (
	dnsPollingInterval = 5 * time.Second
	defaultAWSRegion   = "us-east-1"
)

func (c *Controller) getProvider(ctx context.Context) (challenge.Provider, error) {
	p, err := cloudmanutils.GetDefaultDNSProvider(ctx, c.octeliumC)
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, &deferredError{
				err:   errors.Errorf("The default DNSProvider does not exist yet"),
				delay: time.Minute,
			}
		}
		return nil, err
	}

	if p.Spec == nil {
		return nil, &permanentError{
			err: errors.Errorf("Invalid default DNSProvider"),
		}
	}

	switch p.Spec.Type.(type) {
	case *enterprisev1.DNSProvider_Spec_Cloudflare_:
		spec := p.Spec.GetCloudflare()

		token, err := c.getSecretValue(ctx, spec.GetApiToken().GetFromSecret())
		if err != nil {
			return nil, err
		}

		cfg := cloudflare.NewDefaultConfig()
		cfg.AuthEmail = spec.Email
		cfg.AuthToken = token
		cfg.PropagationTimeout = dnsPropagationTimeout
		cfg.PollingInterval = dnsPollingInterval
		cfg.HTTPClient = &http.Client{
			Timeout: acmeHTTPTimeout,
		}

		return cloudflare.NewDNSProviderConfig(cfg)

	case *enterprisev1.DNSProvider_Spec_Digitalocean:
		spec := p.Spec.GetDigitalocean()

		token, err := c.getSecretValue(ctx, spec.GetApiToken().GetFromSecret())
		if err != nil {
			return nil, err
		}

		cfg := digitalocean.NewDefaultConfig()
		cfg.AuthToken = token
		cfg.PropagationTimeout = dnsPropagationTimeout
		cfg.PollingInterval = dnsPollingInterval
		cfg.HTTPClient = &http.Client{
			Timeout: acmeHTTPTimeout,
		}

		return digitalocean.NewDNSProviderConfig(cfg)

	case *enterprisev1.DNSProvider_Spec_Google_:
		spec := p.Spec.GetGoogle()

		svcAccount, err := c.getSecretValue(ctx, spec.GetServiceAccount().GetFromSecret())
		if err != nil {
			return nil, err
		}

		jwtCfg, err := google.JWTConfigFromJSON([]byte(svcAccount),
			dns.NdevClouddnsReadwriteScope)
		if err != nil {
			return nil, &permanentError{
				err: errors.Errorf("Could not parse the Google service account: %+v", err),
			}
		}

		httpC := jwtCfg.Client(context.WithoutCancel(ctx))
		httpC.Timeout = acmeHTTPTimeout

		cfg := gcloud.NewDefaultConfig()
		cfg.Project = spec.Project
		cfg.PropagationTimeout = dnsPropagationTimeout
		cfg.PollingInterval = dnsPollingInterval
		cfg.HTTPClient = httpC

		return gcloud.NewDNSProviderConfig(cfg)

	case *enterprisev1.DNSProvider_Spec_Azure_:
		spec := p.Spec.GetAzure()

		clientSecret, err := c.getSecretValue(ctx, spec.GetClientSecret().GetFromSecret())
		if err != nil {
			return nil, err
		}

		environment, err := getAzureCloud(spec.Cloud)
		if err != nil {
			return nil, &permanentError{err: err}
		}

		cfg := azuredns.NewDefaultConfig()
		cfg.AuthMethod = "env"
		cfg.ClientID = spec.ClientID
		cfg.ClientSecret = clientSecret
		cfg.TenantID = spec.TenantID
		cfg.SubscriptionID = spec.SubscriptionID
		cfg.ResourceGroup = spec.ResourceGroupName
		cfg.Environment = environment
		cfg.PropagationTimeout = dnsPropagationTimeout
		cfg.PollingInterval = dnsPollingInterval
		cfg.HTTPClient = &http.Client{
			Timeout: acmeHTTPTimeout,
		}

		return azuredns.NewDNSProviderConfig(cfg)

	case *enterprisev1.DNSProvider_Spec_Aws:
		spec := p.Spec.GetAws()

		secretAccessKey, err := c.getSecretValue(ctx, spec.GetSecretAccessKey().GetFromSecret())
		if err != nil {
			return nil, err
		}

		region := spec.Region
		if region == "" {
			region = defaultAWSRegion
		}

		awsC, err := awscfg.LoadDefaultConfig(context.WithoutCancel(ctx),
			awscfg.WithRegion(region),
			awscfg.WithCredentialsProvider(awscred.NewStaticCredentialsProvider(
				spec.AccessKeyID, secretAccessKey, "")))
		if err != nil {
			return nil, err
		}

		if spec.AssumeRoleARN != "" {
			awsC.Credentials = aws.NewCredentialsCache(
				stscreds.NewAssumeRoleProvider(sts.NewFromConfig(awsC), spec.AssumeRoleARN))
		}

		cfg := route53.NewDefaultConfig()
		cfg.Client = awsroute53.NewFromConfig(awsC)
		cfg.PropagationTimeout = dnsPropagationTimeout
		cfg.PollingInterval = dnsPollingInterval

		return route53.NewDNSProviderConfig(cfg)

	case *enterprisev1.DNSProvider_Spec_Linode_:
		spec := p.Spec.GetLinode()

		token, err := c.getSecretValue(ctx, spec.GetApiToken().GetFromSecret())
		if err != nil {
			return nil, err
		}

		cfg := linode.NewDefaultConfig()
		cfg.Token = token
		cfg.PropagationTimeout = dnsPropagationTimeout
		cfg.PollingInterval = dnsPollingInterval
		cfg.HTTPTimeout = acmeHTTPTimeout

		return linode.NewDNSProviderConfig(cfg)

	case *enterprisev1.DNSProvider_Spec_Ovh:
		spec := p.Spec.GetOvh()

		appSecret, err := c.getSecretValue(ctx, spec.GetApplicationSecret().GetFromSecret())
		if err != nil {
			return nil, err
		}

		cfg := ovh.NewDefaultConfig()
		cfg.APIEndpoint = spec.Endpoint
		cfg.ApplicationKey = spec.ApplicationKey
		cfg.ApplicationSecret = appSecret
		cfg.ConsumerKey = spec.ConsumerKey
		cfg.PropagationTimeout = dnsPropagationTimeout
		cfg.PollingInterval = dnsPollingInterval
		cfg.HTTPClient = &http.Client{
			Timeout: acmeHTTPTimeout,
		}

		return ovh.NewDNSProviderConfig(cfg)

	default:
		return nil, &permanentError{
			err: errors.Errorf("Invalid DNSProvider type for: %s", p.Metadata.Name),
		}
	}
}

func (c *Controller) getSecretValue(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", &permanentError{
			err: errors.Errorf("Empty DNSProvider Secret name"),
		}
	}

	sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: name,
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return "", &deferredError{
				err:   errors.Errorf("The DNSProvider Secret %s does not exist", name),
				delay: time.Minute,
			}
		}
		return "", err
	}

	val := uenterprisev1.ToSecret(sec).GetValueStr()
	if val == "" {
		return "", &permanentError{
			err: errors.Errorf("The DNSProvider Secret %s has an empty value", name),
		}
	}

	return val, nil
}

func getAzureCloud(name string) (cloud.Configuration, error) {
	switch name {
	case "public", "":
		return cloud.AzurePublic, nil
	case "china":
		return cloud.AzureChina, nil
	case "usgovernment":
		return cloud.AzureGovernment, nil
	case "german":
		return cloud.Configuration{}, errors.Errorf("Azure Germany is no longer supported")
	default:
		return cloud.Configuration{}, errors.Errorf("Invalid azure cloud: %s", name)
	}
}
