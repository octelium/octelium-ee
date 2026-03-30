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
	"crypto"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	aazure "github.com/Azure/go-autorest/autorest/azure"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscred "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/azure"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/digitalocean"
	"github.com/go-acme/lego/v4/providers/dns/gcloud"
	"github.com/go-acme/lego/v4/providers/dns/linode"
	"github.com/go-acme/lego/v4/providers/dns/ovh"
	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/go-acme/lego/v4/registration"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/cloudmanutils"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

func (c *ACMEClient) getProvider(ctx context.Context) (challenge.Provider, error) {

	provider, err := cloudmanutils.GetDefaultDNSProvider(ctx, c.octeliumC)
	if err != nil {
		return nil, err
	}

	switch provider.Spec.Type.(type) {
	case *enterprisev1.DNSProvider_Spec_Cloudflare_:

		sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: provider.Spec.GetCloudflare().GetApiToken().GetFromSecret(),
		})
		if err != nil {
			return nil, err
		}

		return cloudflare.NewDNSProviderConfig(&cloudflare.Config{
			AuthEmail:          provider.Spec.GetCloudflare().Email,
			AuthToken:          uenterprisev1.ToSecret(sec).GetValueStr(),
			TTL:                120,
			PropagationTimeout: 2 * time.Minute,
			PollingInterval:    2 * time.Second,
			HTTPClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		})
	case *enterprisev1.DNSProvider_Spec_Digitalocean:
		cfg := digitalocean.NewDefaultConfig()

		sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: provider.Spec.GetDigitalocean().GetApiToken().GetFromSecret(),
		})
		if err != nil {
			return nil, err
		}
		cfg.AuthToken = uenterprisev1.ToSecret(sec).GetValueStr()

		return digitalocean.NewDNSProviderConfig(cfg)
	case *enterprisev1.DNSProvider_Spec_Google_:
		cfg := gcloud.NewDefaultConfig()
		cfg.Project = provider.Spec.GetGoogle().Project

		sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: provider.Spec.GetGoogle().GetServiceAccount().GetFromSecret(),
		})
		if err != nil {
			return nil, err
		}

		conf, err := google.JWTConfigFromJSON(
			[]byte(uenterprisev1.ToSecret(sec).GetValueStr()), dns.NdevClouddnsReadwriteScope)
		if err != nil {
			return nil, err
		}

		cfg.HTTPClient = conf.Client(context.Background())
		/*
			if provider.Spec.GetGoogle().GetServiceAccountBase64() != "" {


			} else {
				httpC, err := google.DefaultClient(context.Background(), dns.NdevClouddnsReadwriteScope)
				if err != nil {
					return nil, err
				}
				cfg.HTTPClient = httpC
			}
		*/

		return gcloud.NewDNSProviderConfig(cfg)
	case *enterprisev1.DNSProvider_Spec_Azure_:
		azureCfg := provider.Spec.GetAzure()

		sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: provider.Spec.GetAzure().GetClientSecret().GetFromSecret(),
		})
		if err != nil {
			return nil, err
		}

		cfg := azure.NewDefaultConfig()
		cfg.ClientID = azureCfg.ClientID
		cfg.ClientSecret = uenterprisev1.ToSecret(sec).GetValueStr()
		cfg.SubscriptionID = azureCfg.SubscriptionID
		cfg.ResourceGroup = azureCfg.ResourceGroupName
		cfg.TenantID = azureCfg.TenantID

		var environment aazure.Environment
		switch azureCfg.Cloud {
		case "china":
			environment = aazure.ChinaCloud
		case "german":
			environment = aazure.GermanCloud
		case "public", "":
			environment = aazure.PublicCloud
		case "usgovernment":
			environment = aazure.USGovernmentCloud
		default:
			return nil, errors.Errorf("Invalid azure cloud: %s", azureCfg.Cloud)
		}

		cfg.ResourceManagerEndpoint = environment.ResourceManagerEndpoint
		cfg.ActiveDirectoryEndpoint = environment.ActiveDirectoryEndpoint

		return azure.NewDNSProviderConfig(cfg)
	case *enterprisev1.DNSProvider_Spec_Aws:
		spec := provider.Spec.GetAws()
		cfg := route53.NewDefaultConfig()

		sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: provider.Spec.GetAws().GetSecretAccessKey().GetFromSecret(),
		})
		if err != nil {
			return nil, err
		}

		acfg, err := awscfg.LoadDefaultConfig(ctx,
			awscfg.WithRegion(spec.Region),
			awscfg.WithCredentialsProvider(awscred.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     spec.AccessKeyID,
					SecretAccessKey: sec.GetData().GetValue(),
				},
			}))
		if err != nil {
			return nil, err
		}

		route53C := awsroute53.NewFromConfig(acfg)
		cfg.Client = route53C

		return route53.NewDNSProviderConfig(cfg)
	case *enterprisev1.DNSProvider_Spec_Linode_:
		cfg := linode.NewDefaultConfig()

		sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: provider.Spec.GetLinode().GetApiToken().GetFromSecret(),
		})
		if err != nil {
			return nil, err
		}

		cfg.Token = uenterprisev1.ToSecret(sec).GetValueStr()

		return linode.NewDNSProviderConfig(cfg)
	case *enterprisev1.DNSProvider_Spec_Ovh:
		ovhC := provider.Spec.GetOvh()
		sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: provider.Spec.GetOvh().GetApplicationSecret().GetFromSecret(),
		})
		if err != nil {
			return nil, err
		}

		cfg := ovh.NewDefaultConfig()
		cfg.APIEndpoint = ovhC.Endpoint
		cfg.ApplicationKey = ovhC.ApplicationKey
		cfg.ApplicationSecret = uenterprisev1.ToSecret(sec).GetValueStr()
		cfg.ConsumerKey = ovhC.ConsumerKey

		return ovh.NewDNSProviderConfig(cfg)

		/*
			case *enterprisev1.DNSProvider_Spec_Alibaba_:
				aliC := provider.Spec.GetAlibaba()
				sec, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
					Name: provider.Spec.GetAlibaba().GetAccessKeySecret().GetFromSecret(),
				})
				if err != nil {
					return nil, err
				}

				cfg := alidns.NewDefaultConfig()
				cfg.APIKey = aliC.AccessKeyID
				cfg.SecretKey = uenterprisev1.ToSecret(sec).GetValueStr()
				cfg.RegionID = aliC.RegionID

				return alidns.NewDNSProviderConfig(cfg)
		*/
	default:
		return nil, errors.Errorf("Invalid provider type for: %s", provider.Metadata.Name)
	}

}

func (c *ACMEClient) getLegoClient(ctx context.Context, iss *enterprisev1.CertificateIssuer) (*lego.Client, error) {

	acc, err := c.getACMEAccount(ctx, iss)
	if err != nil {
		return nil, err
	}

	cfg := lego.NewConfig(acc)

	cfg.CADirURL = uenterprisev1.ToCertificateIssuer(iss).GetDirectoryURL()

	return lego.NewClient(cfg)
}

func (c *ACMEClient) getACMEAccount(ctx context.Context, iss *enterprisev1.CertificateIssuer) (*Account, error) {
	if iss.Status.GetAcme() == nil || iss.Status.GetAcme().SecretRef == nil {
		return nil, errors.Errorf("Issuer has no ACME secret")
	}

	var acc Account

	secret, err := c.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Uid: iss.Status.GetAcme().SecretRef.Uid,
	})
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(secret.Data.GetDataMap().Map["account"], &acc); err != nil {
		return nil, err
	}

	privateKey, err := utils_cert.ParsePrivateKeyPEM(secret.Data.GetDataMap().Map["privateKey"])
	if err != nil {
		return nil, err
	}

	acc.key = privateKey
	zap.L().Debug("Successfully found ACME account", zap.Any("iss", iss))

	return &acc, nil
}

type Account struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	key          crypto.PrivateKey
}

func (a *Account) GetEmail() string {
	return a.Email
}

func (a *Account) GetPrivateKey() crypto.PrivateKey {
	return a.key
}

func (a *Account) GetRegistration() *registration.Resource {
	return a.Registration
}

type ACMEClient struct {
	c         *lego.Client
	octeliumC octeliumc.ClientInterface
	crt       *enterprisev1.Certificate
}

func NewACMEClient(ctx context.Context, octeliumC octeliumc.ClientInterface, crt *enterprisev1.Certificate) (*ACMEClient, error) {

	ret := &ACMEClient{
		octeliumC: octeliumC,
		crt:       crt,
	}
	if crt.Status.CertificateIssuerRef == nil {
		return nil, errors.Errorf("nil CertificateIssuerRef")
	}

	issuer, err := getReadyIssuer(ctx, octeliumC, crt.Status.CertificateIssuerRef)
	if err != nil {
		return nil, err
	}

	c, err := ret.getLegoClient(ctx, issuer)
	if err != nil {
		return nil, err
	}
	ret.c = c

	provider, err := ret.getProvider(ctx)
	if err != nil {
		return nil, err
	}

	if err := c.Challenge.SetDNS01Provider(provider,
		dns01.AddRecursiveNameservers([]string{"8.8.8.8", "1.1.1.1"}),
		dns01.AddDNSTimeout(120*time.Second),
	); err != nil {
		return nil, err
	}

	zap.L().Debug("Successfully initialized ACME client", zap.Any("crt", crt))

	return ret, nil
}

func getReadyIssuer(ctx context.Context, octeliumC octeliumc.ClientInterface, issuerRef *metav1.ObjectReference) (*enterprisev1.CertificateIssuer, error) {
	for range 600 {
		issuer, err := octeliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{
			Uid: issuerRef.Uid,
		})
		if err != nil {
			return nil, err
		}

		if issuer.Spec.GetAcme() == nil {
			return nil, errors.Errorf("issuer type is not ACME")
		}

		switch issuer.Status.State {
		case enterprisev1.CertificateIssuer_Status_READY:
			return issuer, nil
		default:
			zap.L().Debug("certificateIssuer is preparing. Waiting and trying again...",
				zap.String("state", issuer.Status.State.String()))
			time.Sleep(10 * time.Second)
		}
	}

	return nil, errors.Errorf("Could not get a READY certificateIssuer after 10 tries")
}

func (c *ACMEClient) IssueCertificate(ctx context.Context) error {
	var err error
	crt := c.crt

	if crt.Status.Issuance == nil ||
		crt.Status.Issuance.State != enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED {
		zap.L().Info("No need to issue the Certificate. Exiting..", zap.Any("crt", crt))
		return nil
	}

	domains, err := c.preIssueCrt(ctx)
	if err != nil {
		return err
	}

	crtRsc, err := c.doIssueCrt(ctx, domains)
	if err != nil {
		zap.L().Warn("Could not doIssueCrt", zap.Error(err))
		crt.Status.Issuance.IssuanceCompletedAt = pbutils.Now()
		crt.Status.Issuance.State = enterprisev1.Certificate_Status_Issuance_FAILED
		crt.Status.FailedIssuances = crt.Status.FailedIssuances + 1

		c.crt, err = c.octeliumC.EnterpriseC().UpdateCertificate(ctx, crt)
		if err != nil {
			return err
		}

		return err
	}

	zap.L().Debug("Successful doIssueCrt", zap.Any("crt", crt))

	if err := c.postIssueCrt(ctx, crtRsc.Certificate, crtRsc.PrivateKey); err != nil {
		return err
	}

	zap.L().Info("Successfully issued Certificate", zap.Any("crt", crt))
	return nil
}

func (c *ACMEClient) preIssueCrt(ctx context.Context) ([]string, error) {
	crt := c.crt
	zap.L().Debug("Starting to issue Certificate", zap.Any("cert", crt))

	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}
	domain := cc.Status.Domain

	var domains []string

	switch {
	case crt.Status.ServiceRef != nil:
		svc, err := c.octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
			Uid: crt.Status.ServiceRef.Uid,
		})
		if err != nil {
			return nil, err
		}

		/*
			if (ucorev1.ToService(svc).Name() == "default" || svc.Status.NamespaceRef.Name == "default") &&
				(!ucorev1.ToService(svc).IsManagedService() ||
					(ucorev1.ToService(svc).IsManagedService() && !svc.Status.ManagedService.HasSubdomain)) {
						zap.L().Debug("No need to issueCert for this Service")
				return nil
			}
		*/

		domains = []string{
			fmt.Sprintf("%s.%s", svc.Status.PrimaryHostname, domain),
			fmt.Sprintf("%s.local.%s", svc.Status.PrimaryHostname, domain),
		}

		if svc.Status.ManagedService != nil && svc.Status.ManagedService.HasSubdomain {
			domains = append(domains,
				fmt.Sprintf("*.%s.%s", svc.Status.PrimaryHostname, domain),
				fmt.Sprintf("*.%s.local.%s", svc.Status.PrimaryHostname, domain),
			)
			/*
				domains = append(domains,
					fmt.Sprintf("*.%s.%s.%s", ucorev1.ToService(svc).Name(), svc.Status.NamespaceRef.Name, domain),
					fmt.Sprintf("*.%s.%s.local.%s", ucorev1.ToService(svc).Name(), svc.Status.NamespaceRef.Name, domain))
			*/
		}

		/*
			if svc.Status.NamespaceRef.Name == "default" {
				domains = append(domains, fmt.Sprintf("%s.%s", ucorev1.ToService(svc).Name(), domain),
					fmt.Sprintf("%s.local.%s", ucorev1.ToService(svc).Name(), domain))
			}
		*/

	case crt.Status.ServiceRef == nil && crt.Status.NamespaceRef != nil:
		domains = []string{
			fmt.Sprintf("*.%s.%s", crt.Status.NamespaceRef.Name, domain),
			fmt.Sprintf("*.%s.local.%s", crt.Status.NamespaceRef.Name, domain),
		}
		if crt.Status.NamespaceRef.Name == "default" {
			domains = append(domains, []string{
				fmt.Sprintf("*.%s", domain),
				fmt.Sprintf("*.local.%s", domain),
			}...)

			domains = append([]string{
				domain,
			}, domains...)
		}
	default:
		return nil, errors.Errorf("Could not find the domains for the crt: %s", crt.Metadata.Name)
	}

	crt.Status.Issuance.IssuanceStartedAt = pbutils.Now()
	crt.Status.Issuance.State = enterprisev1.Certificate_Status_Issuance_ISSUING

	c.crt, err = c.octeliumC.EnterpriseC().UpdateCertificate(ctx, crt)
	if err != nil {
		return nil, err
	}

	return domains, nil
}

func (c *ACMEClient) postIssueCrt(ctx context.Context, cert, privateKey []byte) error {
	crt := c.crt
	x509Crt, err := utils_cert.ParsePEMCertificate(string(cert))
	if err != nil {
		return errors.Errorf("Could not parse PEM of issued crt: %+v", err)
	}

	var sec *corev1.Secret
	sec, err = c.octeliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: uenterprisev1.ToCertificate(crt).GetSecretName(),
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}

		sec = &corev1.Secret{
			Metadata: &metav1.Metadata{
				Name:           uenterprisev1.ToCertificate(crt).GetSecretName(),
				IsSystem:       true,
				IsUserHidden:   true,
				IsSystemHidden: true,
				SystemLabels: map[string]string{
					"octelium-cert": "true",
				},
			},
			Spec:   &corev1.Secret_Spec{},
			Status: &corev1.Secret_Status{},
		}
		ucorev1.ToSecret(sec).SetCertificate(string(cert), string(privateKey))
		sec, err = c.octeliumC.CoreC().CreateSecret(ctx, sec)
		if err != nil {
			return err
		}
	} else {
		ucorev1.ToSecret(sec).SetCertificate(string(cert), string(privateKey))
		sec.Metadata.IsSystem = true
		sec, err = c.octeliumC.CoreC().UpdateSecret(ctx, sec)
		if err != nil {
			return err
		}
	}

	crt.Status.Issuance.State = enterprisev1.Certificate_Status_Issuance_SUCCESS

	crt.Status.Issuance.IssuanceCompletedAt = pbutils.Now()
	crt.Status.Issuance.ExpiresAt = pbutils.Timestamp(x509Crt.NotAfter)

	crt.Status.SuccessfulIssuances = crt.Status.SuccessfulIssuances + 1
	crt.Status.SecretRef = umetav1.GetObjectReference(sec)

	c.crt, err = c.octeliumC.EnterpriseC().UpdateCertificate(ctx, crt)
	if err != nil {
		return err
	}
	return nil
}

func (c *ACMEClient) doIssueCrt(ctx context.Context, domains []string) (*certificate.Resource, error) {

	crt := c.crt
	for {
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("ctx done")
		case <-time.After(10 * time.Minute):
			return nil, errors.Errorf("timeout exceeded trying to issueCrt: %+v", crt)
		default:
			zap.L().Debug("Starting issuing certificate",
				zap.Any("crt", crt), zap.Strings("domains", domains))
			rsc, err := c.c.Certificate.Obtain(certificate.ObtainRequest{
				Domains: domains,
				Bundle:  true,
			})
			if err == nil {
				zap.L().Debug("Successfully obtained crt from the ACME server", zap.Any("crt", crt))
				return rsc, nil
			}

			zap.S().Warnf("Could not obtain cert: %+v. Trying again...", err)
			time.Sleep(10 * time.Second)

		}
	}
}

/*
func (c *ACMEClient) RenewCertificate(ctx context.Context, crt *corev1.Secret) error {
	if crt.Data == nil || crt.Data.GetCertificate() == nil || crt.Data.GetCertificate().PrivateKey == nil {
		return errors.Errorf("Cannot renew Certificate: %s as private key does not exist", crt.Metadata.Name)
	}

	privKey, err := utils_cert.DecodePrivateKeyBytes(crt.Data.GetCertificate().PrivateKey)
	if err != nil {
		return err
	}


	crtRsc, err := func() (*certificate.Resource, error) {

		for i := 0; i < 300; i++ {
			rsc, err := c.c.Certificate.Obtain(certificate.ObtainRequest{
				Domains:    crt.Spec.Domains,
				PrivateKey: privKey,
			})
			if err == nil {
				return rsc, nil
			}

			zap.S().Warnf("Could not renew cert: %+v. Trying again...", err)
			time.Sleep(10 * time.Second)
		}
		return nil, errors.Errorf("Could not obtain cert")
	}()
	if err != nil {
		return err
	}

	crt.Data = &corev1.Secret_Data{
		Type: &corev1.Secret_Data_Certificate_{
			Certificate: &corev1.Secret_Data_Certificate{
				Cert:       crtRsc.Certificate,
				PrivateKey: crtRsc.PrivateKey,
			},
		},
	}

	zap.S().Debugf("acme client successfully renewed cert: %s", crt.Metadata.Name)
	return nil
}
*/

/*
func RegisterAccount(ctx context.Context, octeliumC octeliumc.ClientInterface, cc *enterprisev1.ClusterConfig) error {

	if cc.Spec.Certificate == nil || cc.Spec.Certificate.GetAcme() == nil {
		zap.L().Debug("No set ACME info. Skipping registering an account")
		return nil
	}

	zap.S().Debugf("Starting registering an ACME account")

	k, err := utils_cert.GenerateECDSA()
	if err != nil {
		return err
	}

	privateKeyBytes, err := k.GetPrivateKeyPEM()
	if err != nil {
		return err
	}

	cfg := lego.NewConfig(&Account{
		Email: cc.Spec.Certificate.GetAcme().Email,
		key:   k.PrivateKey,
	})

	cfg.CADirURL = getCADirURL(cc)

	client, err := lego.NewClient(cfg)
	if err != nil {
		return err
	}

	crtRsc, err := func() (*registration.Resource, error) {

		for i := 0; i < 300; i++ {
			rsc, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
			if err == nil {
				return rsc, nil
			}

			zap.S().Warnf("Could not register ACME account: %+v. Trying again...", err)
			time.Sleep(10 * time.Second)
		}
		return nil, errors.Errorf("Could not obtain cert")
	}()
	if err != nil {
		return err
	}

	acc := &Account{
		Email:        cc.Spec.Certificate.GetAcme().Email,
		Registration: crtRsc,
	}

	accBytes, err := json.Marshal(acc)
	if err != nil {
		return err
	}

	acmeSecret, err := octeliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{Name: "sys:acme-account"})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}
		if _, err := octeliumC.CoreC().CreateSecret(ctx, &corev1.Secret{
			Metadata: &metav1.Metadata{
				Name:     "sys:acme-account",
				IsSystem: true,
				IsHidden: true,
			},
			Spec:   &corev1.Secret_Spec{},
			Status: &corev1.Secret_Status{},
			Data: &corev1.Secret_Data{
				Type: &corev1.Secret_Data_DataMap_{
					DataMap: &corev1.Secret_Data_DataMap{
						Map: map[string][]byte{
							"account":    accBytes,
							"privateKey": []byte(privateKeyBytes),
						},
					},
				},
			},
		}); err != nil {
			return err
		}

		return nil
	}

	if acmeSecret.Data == nil {
		acmeSecret.Data = &corev1.Secret_Data{}
	}

	if acmeSecret.Data == nil {
		acmeSecret.Data.Type = &corev1.Secret_Data_DataMap_{
			DataMap: &corev1.Secret_Data_DataMap{
				Map: make(map[string][]byte),
			},
		}
	}

	acmeSecret.Data.GetDataMap().Map["account"] = accBytes
	acmeSecret.Data.GetDataMap().Map["privateKey"] = []byte(privateKeyBytes)

	if _, err := octeliumC.CoreC().UpdateSecret(ctx, acmeSecret); err != nil {
		return err
	}

	zap.S().Info("Successfully registered an ACME account")

	return nil
}
*/

func IssueCertificate(ctx context.Context, octeliumC octeliumc.ClientInterface, crt *enterprisev1.Certificate) error {

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		zap.L().Info("Starting ACME IssueCertificate", zap.Any("crt", crt))

		time.Sleep(time.Duration(utilrand.GetRandomRangeMath(1, 8)) * time.Second)

		acmeC, err := NewACMEClient(ctx, octeliumC, crt)
		if err != nil {
			zap.L().Error("Could not create ACME client", zap.Error(err), zap.Any("crt", crt))
			return
		}

		if err := acmeC.IssueCertificate(ctx); err != nil {
			zap.L().Error("Could not issue ACME certificate", zap.Error(err), zap.Any("crt", crt))
		} else {
			zap.L().Info("Successfully issued ACME certificate", zap.Any("crt", crt))
		}
	}()

	return nil
}
