// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"strings"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestDNSProvider(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	sec := tstCreateDNSProviderSecret(ctx, t, srv)
	item := tstCreateDNSProvider(ctx, t, srv, tstDNSProviderCloudflareSpec(sec.Metadata.Name))
	assert.Equal(t, "admin@example.com", item.Spec.GetCloudflare().Email)

	{
		ret, err := srv.GetDNSProvider(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)

		ret, err = srv.GetDNSProvider(ctx, &metav1.GetOptions{Name: item.Metadata.Name})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)
	}

	{
		_, err := srv.GetDNSProvider(ctx, &metav1.GetOptions{})
		assert.NotNil(t, err)
	}

	{
		_, err := srv.GetDNSProvider(ctx, &metav1.GetOptions{Name: utilrand.GetRandomStringCanonical(8)})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		itemList, err := srv.ListDNSProvider(ctx, &enterprisev1.ListDNSProviderOptions{})
		assert.Nil(t, err, "%+v", err)
		found := false
		for _, listItem := range itemList.Items {
			if listItem.Metadata.Uid == item.Metadata.Uid {
				found = true
			}
		}
		assert.True(t, found)
	}

	{
		arg := tstCloneDNSProvider(item)
		arg.Spec = tstDNSProviderAWS(providerSecretName(sec), true)
		updated, err := srv.UpdateDNSProvider(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", updated.Spec.GetAws().AccessKeyID)
		assert.Equal(t, "us-east-1", updated.Spec.GetAws().Region)
		assert.Equal(t, "arn:aws:iam::123456789012:role/dns-manager", updated.Spec.GetAws().AssumeRoleARN)
		item = updated
	}

	{
		arg := tstCloneDNSProvider(item)
		arg.Spec = tstDNSProviderAzureSpec(providerSecretName(sec), "public")
		updated, err := srv.UpdateDNSProvider(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "client-id", updated.Spec.GetAzure().ClientID)
		assert.Equal(t, "subscription-id", updated.Spec.GetAzure().SubscriptionID)
		assert.Equal(t, "tenant-id", updated.Spec.GetAzure().TenantID)
		assert.Equal(t, "rg-octelium", updated.Spec.GetAzure().ResourceGroupName)
		assert.Equal(t, "public", updated.Spec.GetAzure().Cloud)
		item = updated
	}

	{
		arg := tstCloneDNSProvider(item)
		arg.Spec = tstDNSProviderGoogleSpec(providerSecretName(sec))
		updated, err := srv.UpdateDNSProvider(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "octelium-project", updated.Spec.GetGoogle().Project)
		item = updated
	}

	{
		arg := tstCloneDNSProvider(item)
		arg.Spec = tstDNSProviderOVHSpec(providerSecretName(sec))
		updated, err := srv.UpdateDNSProvider(ctx, arg)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "ovh-eu", updated.Spec.GetOvh().Endpoint)
		assert.Equal(t, "application-key", updated.Spec.GetOvh().ApplicationKey)
		assert.Equal(t, "consumer-key", updated.Spec.GetOvh().ConsumerKey)
		item = updated
	}

	{
		arg := tstCloneDNSProvider(item)
		arg.Metadata.Name = utilrand.GetRandomStringCanonical(8)
		_, err := srv.UpdateDNSProvider(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		arg := tstCloneDNSProvider(item)
		arg.Spec = tstDNSProviderCloudflareSpec(utilrand.GetRandomStringCanonical(8))
		_, err := srv.UpdateDNSProvider(ctx, arg)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestValidateDNSProvider(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	sec := tstCreateDNSProviderSecret(ctx, t, srv)
	secretName := providerSecretName(sec)

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderCloudflareSpec(secretName),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderAWS(secretName, true),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderAWS(secretName, false),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderAzureSpec(secretName, ""),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderAzureSpec(secretName, "china"),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderAzureSpec(secretName, "usgovernment"),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderAzureSpec(secretName, "german"),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderDigitalOceanSpec(secretName),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderGoogleSpec(secretName),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderLinodeSpec(secretName),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: tstDNSProviderOVHSpec(secretName),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: &enterprisev1.DNSProvider_Spec{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: &enterprisev1.DNSProvider_Spec{
				Type: &enterprisev1.DNSProvider_Spec_Cloudflare_{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderCloudflareSpec(secretName)
		spec.GetCloudflare().Email = "invalid"
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderCloudflareSpec("")
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderCloudflareSpec("bad secret")
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderCloudflareSpec(utilrand.GetRandomStringCanonical(8))
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: &enterprisev1.DNSProvider_Spec{
				Type: &enterprisev1.DNSProvider_Spec_Aws{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAWS(secretName, true)
		spec.GetAws().AccessKeyID = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAWS(secretName, true)
		spec.GetAws().SecretAccessKey = nil
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAWS(secretName, true)
		spec.GetAws().Region = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAWS(secretName, true)
		spec.GetAws().AssumeRoleARN = strings.Repeat("a", 257)
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAWS(secretName, true)
		spec.GetAws().Region = "us-éast-1"
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: &enterprisev1.DNSProvider_Spec{
				Type: &enterprisev1.DNSProvider_Spec_Azure_{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAzureSpec(secretName, "public")
		spec.GetAzure().ClientID = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAzureSpec(secretName, "public")
		spec.GetAzure().ClientSecret = nil
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAzureSpec(secretName, "public")
		spec.GetAzure().SubscriptionID = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAzureSpec(secretName, "public")
		spec.GetAzure().TenantID = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAzureSpec(secretName, "public")
		spec.GetAzure().ResourceGroupName = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAzureSpec(secretName, "private")
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderAzureSpec(secretName, "public")
		spec.GetAzure().ResourceGroupName = "rg-é"
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: &enterprisev1.DNSProvider_Spec{
				Type: &enterprisev1.DNSProvider_Spec_Digitalocean{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderDigitalOceanSpec(secretName)
		spec.GetDigitalocean().ApiToken = nil
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: &enterprisev1.DNSProvider_Spec{
				Type: &enterprisev1.DNSProvider_Spec_Google_{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderGoogleSpec(secretName)
		spec.GetGoogle().Project = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderGoogleSpec(secretName)
		spec.GetGoogle().ServiceAccount = nil
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: &enterprisev1.DNSProvider_Spec{
				Type: &enterprisev1.DNSProvider_Spec_Linode_{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderLinodeSpec(secretName)
		spec.GetLinode().ApiToken = nil
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{
			Spec: &enterprisev1.DNSProvider_Spec{
				Type: &enterprisev1.DNSProvider_Spec_Ovh{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderOVHSpec(secretName)
		spec.GetOvh().Endpoint = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderOVHSpec(secretName)
		spec.GetOvh().ApplicationKey = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderOVHSpec(secretName)
		spec.GetOvh().ApplicationSecret = nil
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderOVHSpec(secretName)
		spec.GetOvh().ConsumerKey = ""
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		spec := tstDNSProviderOVHSpec(secretName)
		spec.GetOvh().Endpoint = "ovh-é"
		err := srv.validateDNSProvider(ctx, &enterprisev1.DNSProvider{Spec: spec})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestValidateDNSProviderHelpers(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)

	sec := tstCreateDNSProviderSecret(ctx, t, srv)

	{
		err := srv.validateSecretOwner(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecretOwner(ctx, tstDNSProviderCloudflareAPIToken(""))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecretOwner(ctx, tstDNSProviderCloudflareAPIToken("bad secret"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecretOwner(ctx, tstDNSProviderCloudflareAPIToken(utilrand.GetRandomStringCanonical(8)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateSecretOwner(ctx, tstDNSProviderCloudflareAPIToken(providerSecretName(sec)))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateGenStr("value", true, "field")
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateGenStr("", false, "field")
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateGenStr("", true, "field")
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateGenStr(strings.Repeat("a", 257), true, "field")
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateGenStr("é", true, "field")
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := validateDNSProviderAzureCloud("public")
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := validateDNSProviderAzureCloud("private")
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func tstCreateDNSProviderSecret(ctx context.Context, t *testing.T, srv *Server) *enterprisev1.Secret {
	sec, err := srv.CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	return sec
}

func tstCreateDNSProvider(ctx context.Context, t *testing.T, srv *Server, spec *enterprisev1.DNSProvider_Spec) *enterprisev1.DNSProvider {
	item, err := srv.octeliumC.EnterpriseC().CreateDNSProvider(ctx, &enterprisev1.DNSProvider{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   spec,
		Status: &enterprisev1.DNSProvider_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	return item
}

func tstCloneDNSProvider(arg *enterprisev1.DNSProvider) *enterprisev1.DNSProvider {
	return &enterprisev1.DNSProvider{
		Metadata: &metav1.Metadata{
			Name: arg.Metadata.Name,
			Uid:  arg.Metadata.Uid,
		},
		Spec:   arg.Spec,
		Status: arg.Status,
	}
}

func providerSecretName(sec *enterprisev1.Secret) string {
	return sec.Metadata.Name
}

func tstDNSProviderCloudflareSpec(secretName string) *enterprisev1.DNSProvider_Spec {
	return &enterprisev1.DNSProvider_Spec{
		Type: &enterprisev1.DNSProvider_Spec_Cloudflare_{
			Cloudflare: &enterprisev1.DNSProvider_Spec_Cloudflare{
				Email:    "admin@example.com",
				ApiToken: tstDNSProviderCloudflareAPIToken(secretName),
				Proxied:  true,
			},
		},
	}
}

func tstDNSProviderCloudflareAPIToken(secretName string) *enterprisev1.DNSProvider_Spec_Cloudflare_APIToken {
	return &enterprisev1.DNSProvider_Spec_Cloudflare_APIToken{
		Type: &enterprisev1.DNSProvider_Spec_Cloudflare_APIToken_FromSecret{
			FromSecret: secretName,
		},
	}
}

func tstDNSProviderAWS(secretName string, assumeRole bool) *enterprisev1.DNSProvider_Spec {
	ret := &enterprisev1.DNSProvider_Spec{
		Type: &enterprisev1.DNSProvider_Spec_Aws{
			Aws: &enterprisev1.DNSProvider_Spec_AWS{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: tstDNSProviderAWSSecretAccessKey(secretName),
				Region:          "us-east-1",
			},
		},
	}
	if assumeRole {
		ret.GetAws().AssumeRoleARN = "arn:aws:iam::123456789012:role/dns-manager"
	}
	return ret
}

func tstDNSProviderAWSSecretAccessKey(secretName string) *enterprisev1.DNSProvider_Spec_AWS_SecretAccessKey {
	return &enterprisev1.DNSProvider_Spec_AWS_SecretAccessKey{
		Type: &enterprisev1.DNSProvider_Spec_AWS_SecretAccessKey_FromSecret{
			FromSecret: secretName,
		},
	}
}

func tstDNSProviderAzureSpec(secretName string, cloud string) *enterprisev1.DNSProvider_Spec {
	return &enterprisev1.DNSProvider_Spec{
		Type: &enterprisev1.DNSProvider_Spec_Azure_{
			Azure: &enterprisev1.DNSProvider_Spec_Azure{
				ClientID:          "client-id",
				ClientSecret:      tstDNSProviderAzureClientSecret(secretName),
				SubscriptionID:    "subscription-id",
				TenantID:          "tenant-id",
				ResourceGroupName: "rg-octelium",
				Cloud:             cloud,
			},
		},
	}
}

func tstDNSProviderAzureClientSecret(secretName string) *enterprisev1.DNSProvider_Spec_Azure_ClientSecret {
	return &enterprisev1.DNSProvider_Spec_Azure_ClientSecret{
		Type: &enterprisev1.DNSProvider_Spec_Azure_ClientSecret_FromSecret{
			FromSecret: secretName,
		},
	}
}

func tstDNSProviderDigitalOceanSpec(secretName string) *enterprisev1.DNSProvider_Spec {
	return &enterprisev1.DNSProvider_Spec{
		Type: &enterprisev1.DNSProvider_Spec_Digitalocean{
			Digitalocean: &enterprisev1.DNSProvider_Spec_DigitalOcean{
				ApiToken: tstDNSProviderDigitalOceanAPIToken(secretName),
			},
		},
	}
}

func tstDNSProviderDigitalOceanAPIToken(secretName string) *enterprisev1.DNSProvider_Spec_DigitalOcean_APIToken {
	return &enterprisev1.DNSProvider_Spec_DigitalOcean_APIToken{
		Type: &enterprisev1.DNSProvider_Spec_DigitalOcean_APIToken_FromSecret{
			FromSecret: secretName,
		},
	}
}

func tstDNSProviderGoogleSpec(secretName string) *enterprisev1.DNSProvider_Spec {
	return &enterprisev1.DNSProvider_Spec{
		Type: &enterprisev1.DNSProvider_Spec_Google_{
			Google: &enterprisev1.DNSProvider_Spec_Google{
				Project:        "octelium-project",
				ServiceAccount: tstDNSProviderGoogleServiceAccount(secretName),
			},
		},
	}
}

func tstDNSProviderGoogleServiceAccount(secretName string) *enterprisev1.DNSProvider_Spec_Google_ServiceAccount {
	return &enterprisev1.DNSProvider_Spec_Google_ServiceAccount{
		Type: &enterprisev1.DNSProvider_Spec_Google_ServiceAccount_FromSecret{
			FromSecret: secretName,
		},
	}
}

func tstDNSProviderLinodeSpec(secretName string) *enterprisev1.DNSProvider_Spec {
	return &enterprisev1.DNSProvider_Spec{
		Type: &enterprisev1.DNSProvider_Spec_Linode_{
			Linode: &enterprisev1.DNSProvider_Spec_Linode{
				ApiToken: tstDNSProviderLinodeAPIToken(secretName),
			},
		},
	}
}

func tstDNSProviderLinodeAPIToken(secretName string) *enterprisev1.DNSProvider_Spec_Linode_APIToken {
	return &enterprisev1.DNSProvider_Spec_Linode_APIToken{
		Type: &enterprisev1.DNSProvider_Spec_Linode_APIToken_FromSecret{
			FromSecret: secretName,
		},
	}
}

func tstDNSProviderOVHSpec(secretName string) *enterprisev1.DNSProvider_Spec {
	return &enterprisev1.DNSProvider_Spec{
		Type: &enterprisev1.DNSProvider_Spec_Ovh{
			Ovh: &enterprisev1.DNSProvider_Spec_OVH{
				Endpoint:          "ovh-eu",
				ApplicationKey:    "application-key",
				ApplicationSecret: tstDNSProviderOVHApplicationSecret(secretName),
				ConsumerKey:       "consumer-key",
			},
		},
	}
}

func tstDNSProviderOVHApplicationSecret(secretName string) *enterprisev1.DNSProvider_Spec_OVH_ApplicationSecret {
	return &enterprisev1.DNSProvider_Spec_OVH_ApplicationSecret{
		Type: &enterprisev1.DNSProvider_Spec_OVH_ApplicationSecret_FromSecret{
			FromSecret: secretName,
		},
	}
}
