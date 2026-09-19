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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestGetAzureCloud(t *testing.T) {
	for _, itm := range []string{"", "public"} {
		got, err := getAzureCloud(itm)
		assert.Nil(t, err)
		assert.Equal(t, cloud.AzurePublic, got)
	}

	{
		got, err := getAzureCloud("china")
		assert.Nil(t, err)
		assert.Equal(t, cloud.AzureChina, got)
	}

	{
		got, err := getAzureCloud("usgovernment")
		assert.Nil(t, err)
		assert.Equal(t, cloud.AzureGovernment, got)
	}

	for _, itm := range []string{"german", utilrand.GetRandomStringCanonical(8)} {
		_, err := getAzureCloud(itm)
		assert.NotNil(t, err)
	}
}

func genGoogleServiceAccount(t *testing.T) string {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(key)
	assert.Nil(t, err)

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})

	ret, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"project_id":   utilrand.GetRandomStringCanonical(8),
		"private_key":  string(keyPEM),
		"client_email": "dns@octelium.iam.gserviceaccount.com",
		"token_uri":    "https://oauth2.googleapis.com/token",
	})
	assert.Nil(t, err)

	return string(ret)
}

func TestGetProvider(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	c := NewController(fakeC.OcteliumC)
	t.Cleanup(c.shutdown)

	newSecret := func(val string) string {
		sec, err := fakeC.OcteliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec:   &enterprisev1.Secret_Spec{},
			Status: &enterprisev1.Secret_Status{},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_Value{
					Value: val,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)

		return sec.Metadata.Name
	}

	setProvider := func(typ any) {
		spec := &enterprisev1.DNSProvider_Spec{}

		switch arg := typ.(type) {
		case *enterprisev1.DNSProvider_Spec_Cloudflare_:
			spec.Type = arg
		case *enterprisev1.DNSProvider_Spec_Digitalocean:
			spec.Type = arg
		case *enterprisev1.DNSProvider_Spec_Google_:
			spec.Type = arg
		case *enterprisev1.DNSProvider_Spec_Azure_:
			spec.Type = arg
		case *enterprisev1.DNSProvider_Spec_Aws:
			spec.Type = arg
		case *enterprisev1.DNSProvider_Spec_Linode_:
			spec.Type = arg
		case *enterprisev1.DNSProvider_Spec_Ovh:
			spec.Type = arg
		}

		cur, err := fakeC.OcteliumC.EnterpriseC().GetDNSProvider(ctx, &rmetav1.GetOptions{
			Name: "default",
		})
		if err != nil {
			assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

			_, err = fakeC.OcteliumC.EnterpriseC().CreateDNSProvider(ctx,
				&enterprisev1.DNSProvider{
					Metadata: &metav1.Metadata{
						Name: "default",
					},
					Spec:   spec,
					Status: &enterprisev1.DNSProvider_Status{},
				})
			assert.Nil(t, err, "%+v", err)
			return
		}

		cur.Spec = spec
		_, err = fakeC.OcteliumC.EnterpriseC().UpdateDNSProvider(ctx, cur)
		assert.Nil(t, err, "%+v", err)
	}

	t.Run("NoDefaultProvider", func(t *testing.T) {
		_, err := c.getProvider(ctx)
		assert.NotNil(t, err)
		assert.True(t, getRetryDelay(1, err) >= 30*time.Second)
	})

	t.Run("Cloudflare", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Cloudflare_{
			Cloudflare: &enterprisev1.DNSProvider_Spec_Cloudflare{
				Email: "contact@example.com",
				ApiToken: &enterprisev1.DNSProvider_Spec_Cloudflare_APIToken{
					Type: &enterprisev1.DNSProvider_Spec_Cloudflare_APIToken_FromSecret{
						FromSecret: newSecret(utilrand.GetRandomString(32)),
					},
				},
			},
		})

		p, err := c.getProvider(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, p)
	})

	t.Run("DigitalOcean", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Digitalocean{
			Digitalocean: &enterprisev1.DNSProvider_Spec_DigitalOcean{
				ApiToken: &enterprisev1.DNSProvider_Spec_DigitalOcean_APIToken{
					Type: &enterprisev1.DNSProvider_Spec_DigitalOcean_APIToken_FromSecret{
						FromSecret: newSecret(utilrand.GetRandomString(32)),
					},
				},
			},
		})

		p, err := c.getProvider(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, p)
	})

	t.Run("Google", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Google_{
			Google: &enterprisev1.DNSProvider_Spec_Google{
				Project: utilrand.GetRandomStringCanonical(8),
				ServiceAccount: &enterprisev1.DNSProvider_Spec_Google_ServiceAccount{
					Type: &enterprisev1.DNSProvider_Spec_Google_ServiceAccount_FromSecret{
						FromSecret: newSecret(genGoogleServiceAccount(t)),
					},
				},
			},
		})

		p, err := c.getProvider(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, p)
	})

	t.Run("GoogleWithAnInvalidServiceAccount", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Google_{
			Google: &enterprisev1.DNSProvider_Spec_Google{
				Project: utilrand.GetRandomStringCanonical(8),
				ServiceAccount: &enterprisev1.DNSProvider_Spec_Google_ServiceAccount{
					Type: &enterprisev1.DNSProvider_Spec_Google_ServiceAccount_FromSecret{
						FromSecret: newSecret("not-a-service-account"),
					},
				},
			},
		})

		_, err := c.getProvider(ctx)
		assert.NotNil(t, err)
	})

	t.Run("AzureUsesTheClientSecret", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Azure_{
			Azure: &enterprisev1.DNSProvider_Spec_Azure{
				ClientID:          "5ba6be00-7ad6-4b5e-9c1a-9ac2b13a6dbf",
				TenantID:          "8aa19a2f-80a4-4b6e-b84e-1b3a0e7e2a2f",
				SubscriptionID:    "b2d0c2ab-0c0e-4a29-a8f9-0b1a0a4f6fa0",
				ResourceGroupName: utilrand.GetRandomStringCanonical(8),
				ClientSecret: &enterprisev1.DNSProvider_Spec_Azure_ClientSecret{
					Type: &enterprisev1.DNSProvider_Spec_Azure_ClientSecret_FromSecret{
						FromSecret: newSecret(utilrand.GetRandomString(32)),
					},
				},
			},
		})

		_, err := c.getProvider(ctx)
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "ClientSecretCredential"), "%+v", err)
	})

	t.Run("AzureWithAnInvalidCloud", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Azure_{
			Azure: &enterprisev1.DNSProvider_Spec_Azure{
				Cloud: "german",
				ClientSecret: &enterprisev1.DNSProvider_Spec_Azure_ClientSecret{
					Type: &enterprisev1.DNSProvider_Spec_Azure_ClientSecret_FromSecret{
						FromSecret: newSecret(utilrand.GetRandomString(32)),
					},
				},
			},
		})

		_, err := c.getProvider(ctx)
		assert.NotNil(t, err)
	})

	t.Run("AWS", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Aws{
			Aws: &enterprisev1.DNSProvider_Spec_AWS{
				AccessKeyID: utilrand.GetRandomStringCanonical(16),
				Region:      "eu-central-1",
				SecretAccessKey: &enterprisev1.DNSProvider_Spec_AWS_SecretAccessKey{
					Type: &enterprisev1.DNSProvider_Spec_AWS_SecretAccessKey_FromSecret{
						FromSecret: newSecret(utilrand.GetRandomString(32)),
					},
				},
			},
		})

		p, err := c.getProvider(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, p)
	})

	t.Run("AWSWithAnAssumedRole", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Aws{
			Aws: &enterprisev1.DNSProvider_Spec_AWS{
				AccessKeyID:   utilrand.GetRandomStringCanonical(16),
				AssumeRoleARN: "arn:aws:iam::123456789012:role/dns-manager",
				SecretAccessKey: &enterprisev1.DNSProvider_Spec_AWS_SecretAccessKey{
					Type: &enterprisev1.DNSProvider_Spec_AWS_SecretAccessKey_FromSecret{
						FromSecret: newSecret(utilrand.GetRandomString(32)),
					},
				},
			},
		})

		p, err := c.getProvider(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, p)
	})

	t.Run("Linode", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Linode_{
			Linode: &enterprisev1.DNSProvider_Spec_Linode{
				ApiToken: &enterprisev1.DNSProvider_Spec_Linode_APIToken{
					Type: &enterprisev1.DNSProvider_Spec_Linode_APIToken_FromSecret{
						FromSecret: newSecret(utilrand.GetRandomString(32)),
					},
				},
			},
		})

		p, err := c.getProvider(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, p)
	})

	t.Run("OVH", func(t *testing.T) {
		setProvider(&enterprisev1.DNSProvider_Spec_Ovh{
			Ovh: &enterprisev1.DNSProvider_Spec_OVH{
				Endpoint:       "ovh-eu",
				ApplicationKey: utilrand.GetRandomStringCanonical(16),
				ConsumerKey:    utilrand.GetRandomStringCanonical(16),
				ApplicationSecret: &enterprisev1.DNSProvider_Spec_OVH_ApplicationSecret{
					Type: &enterprisev1.DNSProvider_Spec_OVH_ApplicationSecret_FromSecret{
						FromSecret: newSecret(utilrand.GetRandomString(32)),
					},
				},
			},
		})

		p, err := c.getProvider(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, p)
	})

	t.Run("InvalidProviderType", func(t *testing.T) {
		setProvider(nil)

		_, err := c.getProvider(ctx)
		assert.NotNil(t, err)
	})

	t.Run("SecretValues", func(t *testing.T) {
		_, err := c.getSecretValue(ctx, "")
		assert.NotNil(t, err)

		_, err = c.getSecretValue(ctx, utilrand.GetRandomStringCanonical(8))
		assert.NotNil(t, err)

		_, err = c.getSecretValue(ctx, newSecret(""))
		assert.NotNil(t, err)

		val := utilrand.GetRandomString(32)
		got, err := c.getSecretValue(ctx, newSecret(val))
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, val, got)
	})
}
