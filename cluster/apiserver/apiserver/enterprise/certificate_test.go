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
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestSetCertificate(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)

	{
		crt, err := srv.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANUAL,
			},
			Status: &enterprisev1.Certificate_Status{},
		})
		assert.Nil(t, err)

		_, err = srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))

		root, err := utils_cert.GenerateCARoot()
		assert.Nil(t, err)
		sn, err := utils_cert.GenerateSerialNumber()
		assert.Nil(t, err)
		cert1, err := utils_cert.GenerateCertificate(&x509.Certificate{
			BasicConstraintsValid: true,
			SerialNumber:          sn,
			Subject: pkix.Name{
				CommonName: "localhost",
			},

			DNSNames: []string{"localhost"},

			NotBefore:   time.Now(),
			NotAfter:    time.Now().Add(24 * time.Hour),
			KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}, root.Certificate, root.PrivateKey, false)
		assert.Nil(t, err)

		_, err = srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    string(cert1.MustGetCertPEM()),
			PrivateKey:     string(cert1.MustGetPrivateKeyPEM()),
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    string(cert1.MustGetCertPEM()),
			PrivateKey:     string(cert1.MustGetPrivateKeyPEM()),
		})
		assert.Nil(t, err)

		sec, err := srv.octeliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: uenterprisev1.ToCertificate(crt).GetSecretName(),
		})
		assert.Nil(t, err)
		assert.Equal(t, "true", sec.Metadata.SystemLabels["octelium-cert"])

		{
			chain, key, err := ucorev1.ToSecret(sec).GetCertificateChainAndKey()
			assert.Nil(t, err)
			ccert1, err := tls.X509KeyPair(chain, key)
			cert1.Certificate.Equal(ccert1.Leaf)
		}

		cert2, err := utils_cert.GenerateCertificate(&x509.Certificate{
			BasicConstraintsValid: true,
			SerialNumber:          sn,
			Subject: pkix.Name{
				CommonName: "localhost",
			},

			DNSNames: []string{"localhost"},

			NotBefore:   time.Now(),
			NotAfter:    time.Now().Add(24 * time.Hour),
			KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}, root.Certificate, root.PrivateKey, false)
		assert.Nil(t, err)

		_, err = srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    string(cert2.MustGetCertPEM()),
			PrivateKey:     string(cert2.MustGetPrivateKeyPEM()),
		})
		assert.Nil(t, err)
	}

}
