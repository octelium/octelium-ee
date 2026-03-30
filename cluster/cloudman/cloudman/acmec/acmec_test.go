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
	"fmt"
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestACME(t *testing.T) {

	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	assert.Nil(t, err)
	zap.ReplaceGlobals(logger)

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	cc, err := fakeC.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err)
	cc.Status.Domain = fmt.Sprintf("%s.octelium.org", utilrand.GetRandomStringCanonical(8))
	cc, err = fakeC.OcteliumC.CoreC().UpdateClusterConfig(ctx, cc)
	assert.Nil(t, err)

	coreSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  fakeC.OcteliumC,
		IsEmbedded: true,
	})

	svc, err := coreSrv.CreateService(ctx, tests.GenService(""))
	assert.Nil(t, err)

	iss, err := fakeC.OcteliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	assert.Nil(t, err)

	iss.Spec.GetAcme().Email = fmt.Sprintf("contact@%s", cc.Status.Domain)

	iss, err = fakeC.OcteliumC.EnterpriseC().UpdateCertificateIssuer(ctx, iss)
	assert.Nil(t, err)

	err = RegisterAccount(ctx, fakeC.OcteliumC, iss, false)
	assert.Nil(t, err)

	sec, err := fakeC.OcteliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   &enterprisev1.Secret_Spec{},
		Status: &enterprisev1.Secret_Status{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		},
	})
	assert.Nil(t, err)

	_, err = fakeC.OcteliumC.EnterpriseC().CreateDNSProvider(ctx, &enterprisev1.DNSProvider{
		Metadata: &metav1.Metadata{
			Name: "default",
		},
		Spec: &enterprisev1.DNSProvider_Spec{
			Type: &enterprisev1.DNSProvider_Spec_Cloudflare_{
				Cloudflare: &enterprisev1.DNSProvider_Spec_Cloudflare{
					Email: fmt.Sprintf("contact@%s", cc.Status.Domain),
					ApiToken: &enterprisev1.DNSProvider_Spec_Cloudflare_APIToken{
						Type: &enterprisev1.DNSProvider_Spec_Cloudflare_APIToken_FromSecret{
							FromSecret: sec.Metadata.Name,
						},
					},
				},
			},
		},
	})
	assert.Nil(t, err)

	crt, err := fakeC.OcteliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
		Metadata: &metav1.Metadata{
			Name: fmt.Sprintf("svc-%s-%s", ucorev1.ToService(svc).Name(), svc.Status.NamespaceRef.Name),
		},
		Spec: &enterprisev1.Certificate_Spec{},
		Status: &enterprisev1.Certificate_Status{
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				CreatedAt: pbutils.Now(),
				State:     enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
			},
			ServiceRef:           umetav1.GetObjectReference(svc),
			NamespaceRef:         svc.Status.NamespaceRef,
			CertificateIssuerRef: umetav1.GetObjectReference(iss),
		},
	})
	assert.Nil(t, err)

	acmeC, err := NewACMEClient(ctx, fakeC.OcteliumC, crt)
	assert.Nil(t, err, "%+v", err)

	domains, err := acmeC.preIssueCrt(ctx)
	assert.Nil(t, err)

	assert.True(t, len(domains) > 0)

	crtK, err := utils_cert.GenerateCARoot()
	assert.Nil(t, err)

	err = acmeC.postIssueCrt(ctx, crtK.MustGetCertPEM(), crtK.MustGetPrivateKeyPEM())
	assert.Nil(t, err)

	crt, err = fakeC.OcteliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
		Uid: crt.Metadata.Uid,
	})
	assert.Nil(t, err)

	assert.Equal(t, crt.Status.Issuance.State, enterprisev1.Certificate_Status_Issuance_SUCCESS)
}
