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
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestCertificate(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)

	{
		_, err := srv.CreateCertificate(ctx, &enterprisev1.Certificate{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.GetCertificate(ctx, &metav1.GetOptions{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.DeleteCertificate(ctx, &metav1.DeleteOptions{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.GetCertificate(ctx, &metav1.GetOptions{Name: utilrand.GetRandomStringCanonical(8)})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		crt, err := srv.CreateCertificate(ctx, tstCertificate(enterprisev1.Certificate_Spec_MANUAL))
		assert.Nil(t, err, "%+v", err)
		assert.NotEmpty(t, crt.Metadata.Uid)
		assert.Equal(t, enterprisev1.Certificate_Spec_MANUAL, crt.Spec.Mode)

		_, err = srv.CreateCertificate(ctx, tstCloneCertificate(crt))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)

		crtG, err := srv.GetCertificate(ctx, &metav1.GetOptions{Uid: crt.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, crt.Metadata.Uid, crtG.Metadata.Uid)

		crtG, err = srv.GetCertificate(ctx, &metav1.GetOptions{Name: crt.Metadata.Name})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, crt.Metadata.Uid, crtG.Metadata.Uid)

		list, err := srv.ListCertificate(ctx, nil)
		assert.Nil(t, err, "%+v", err)
		assert.True(t, len(list.Items) > 0)

		crtU := tstCloneCertificate(crt)
		crtU.Spec.Mode = enterprisev1.Certificate_Spec_MANUAL
		crtU.Status = &enterprisev1.Certificate_Status{
			CertificateIssuerRef: &metav1.ObjectReference{Name: utilrand.GetRandomStringCanonical(8)},
		}

		updated, err := srv.UpdateCertificate(ctx, crtU)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, crt.Metadata.Uid, updated.Metadata.Uid)
		assert.Equal(t, enterprisev1.Certificate_Spec_MANUAL, updated.Spec.Mode)
		assert.Nil(t, updated.Status.CertificateIssuerRef)

		_, err = srv.DeleteCertificate(ctx, &metav1.DeleteOptions{Uid: crt.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GetCertificate(ctx, &metav1.GetOptions{Uid: crt.Metadata.Uid})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		crt, err := srv.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name:     utilrand.GetRandomStringCanonical(8),
				IsSystem: true,
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANUAL,
			},
			Status: &enterprisev1.Certificate_Status{},
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.UpdateCertificate(ctx, tstCloneCertificate(crt))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)

		_, err = srv.DeleteCertificate(ctx, &metav1.DeleteOptions{Uid: crt.Metadata.Uid})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)
	}

	{
		_, err := srv.UpdateCertificate(ctx, tstCertificate(enterprisev1.Certificate_Spec_MANUAL))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		_, err := srv.DeleteCertificate(ctx, &metav1.DeleteOptions{Uid: vutils.UUIDv4()})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}
}

func TestValidateCertificate(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)

	{
		err := srv.validateCertificate(ctx, tstCertificate(enterprisev1.Certificate_Spec_MANUAL))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateCertificate(ctx, tstCertificate(enterprisev1.Certificate_Spec_MANAGED))
		assert.Nil(t, err, "%+v", err)
	}

	{
		err := srv.validateCertificate(ctx, &enterprisev1.Certificate{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificate(ctx, &enterprisev1.Certificate{
			Spec: &enterprisev1.Certificate_Spec{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificate(ctx, tstCertificate(enterprisev1.Certificate_Spec_MODE_UNSET))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		err := srv.validateCertificate(ctx, tstCertificate(enterprisev1.Certificate_Spec_Mode(1000)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestIssueCertificate(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)

	{
		_, err := srv.IssueCertificate(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.IssueCertificate(ctx, &enterprisev1.IssueCertificateRequest{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.IssueCertificate(ctx, &enterprisev1.IssueCertificateRequest{
			CertificateRef: &metav1.ObjectReference{Uid: vutils.UUIDv4()},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		crt, err := srv.CreateCertificate(ctx, tstCertificate(enterprisev1.Certificate_Spec_MANUAL))
		assert.Nil(t, err, "%+v", err)

		_, err = srv.IssueCertificate(ctx, &enterprisev1.IssueCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestSetCertificate(t *testing.T) {
	ctx := context.Background()
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv := NewServer(tst.C.OcteliumC)
	crt, err := srv.CreateCertificate(ctx, tstCertificate(enterprisev1.Certificate_Spec_MANUAL))
	assert.Nil(t, err, "%+v", err)

	certPEM, keyPEM, certLeaf := tstCertificatePEM(t, time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))

	{
		_, err := srv.SetCertificate(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    certPEM,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    strings.Repeat("a", (256<<10)+1),
			PrivateKey:     keyPEM,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    certPEM,
			PrivateKey:     strings.Repeat("a", (64<<10)+1),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    "invalid",
			PrivateKey:     "invalid",
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, keyPEM2, _ := tstCertificatePEM(t, time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    certPEM,
			PrivateKey:     keyPEM2,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		futureCert, futureKey, _ := tstCertificatePEM(t, time.Now().Add(time.Hour), time.Now().Add(24*time.Hour))
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    futureCert,
			PrivateKey:     futureKey,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		expiredCert, expiredKey, _ := tstCertificatePEM(t, time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    expiredCert,
			PrivateKey:     expiredKey,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: &metav1.ObjectReference{Uid: vutils.UUIDv4()},
			Certificate:    certPEM,
			PrivateKey:     keyPEM,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		managed, err := srv.CreateCertificate(ctx, tstCertificate(enterprisev1.Certificate_Spec_MANAGED))
		assert.Nil(t, err, "%+v", err)

		_, err = srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(managed),
			Certificate:    certPEM,
			PrivateKey:     keyPEM,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		system, err := srv.octeliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name:     utilrand.GetRandomStringCanonical(8),
				IsSystem: true,
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANUAL,
			},
			Status: &enterprisev1.Certificate_Status{},
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(system),
			Certificate:    certPEM,
			PrivateKey:     keyPEM,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    certPEM,
			PrivateKey:     keyPEM,
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: umetav1.GetObjectReference(crt),
			Certificate:    certPEM,
			PrivateKey:     keyPEM,
		})
		assert.Nil(t, err, "%+v", err)

		sec, err := srv.octeliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: uenterprisev1.ToCertificate(crt).GetSecretName(),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "true", sec.Metadata.SystemLabels["octelium-cert"])
		assert.True(t, sec.Metadata.IsSystem)
		assert.True(t, sec.Metadata.IsUserHidden)
		assert.True(t, sec.Metadata.IsSystemHidden)

		chain, key, err := ucorev1.ToSecret(sec).GetCertificateChainAndKey()
		assert.Nil(t, err, "%+v", err)
		ccert, err := tls.X509KeyPair(chain, key)
		assert.Nil(t, err, "%+v", err)
		storedLeaf, err := x509.ParseCertificate(ccert.Certificate[0])
		assert.Nil(t, err, "%+v", err)
		assert.True(t, certLeaf.Equal(storedLeaf))

		crtG, err := srv.GetCertificate(ctx, &metav1.GetOptions{Uid: crt.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, crtG.Status.SecretRef)
		assert.NotNil(t, crtG.Status.Info)
		assert.Equal(t, "localhost", crtG.Status.Info.CommonName)
	}
}

func TestValidatePrivateKeyStrength(t *testing.T) {
	{
		certPEM, keyPEM, _ := tstCertificatePEM(t, time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
		keyPair, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, validatePrivateKeyStrength(keyPair.PrivateKey))
	}

	{
		err := validatePrivateKeyStrength("invalid")
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func tstCertificate(mode enterprisev1.Certificate_Spec_Mode) *enterprisev1.Certificate {
	return &enterprisev1.Certificate{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Certificate_Spec{
			Mode: mode,
		},
	}
}

func tstCloneCertificate(arg *enterprisev1.Certificate) *enterprisev1.Certificate {
	return &enterprisev1.Certificate{
		Metadata: &metav1.Metadata{
			Name: arg.Metadata.Name,
			Uid:  arg.Metadata.Uid,
		},
		Spec: &enterprisev1.Certificate_Spec{
			Mode: arg.Spec.Mode,
		},
		Status: &enterprisev1.Certificate_Status{},
	}
}

func tstCertificatePEM(t *testing.T, notBefore time.Time, notAfter time.Time) (string, string, *x509.Certificate) {
	root, err := utils_cert.GenerateCARoot()
	assert.Nil(t, err, "%+v", err)

	sn, err := utils_cert.GenerateSerialNumber()
	assert.Nil(t, err, "%+v", err)

	cert, err := utils_cert.GenerateCertificate(&x509.Certificate{
		BasicConstraintsValid: true,
		SerialNumber:          sn,
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		DNSNames: []string{"localhost"},

		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, root.Certificate, root.PrivateKey, false)
	assert.Nil(t, err, "%+v", err)

	return string(cert.MustGetCertPEM()), string(cert.MustGetPrivateKeyPEM()), cert.Certificate
}
