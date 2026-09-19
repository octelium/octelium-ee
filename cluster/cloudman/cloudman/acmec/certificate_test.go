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
	"slices"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestValidateIssuedCertificate(t *testing.T) {
	domains := []string{"example.com", "a.example.com", "*.a.example.com"}

	crtK, err := utils_cert.GenerateSelfSignedCert("example.com", domains, 24*time.Hour)
	assert.Nil(t, err)

	certPEM := crtK.MustGetCertPEM()
	keyPEM := crtK.MustGetPrivateKeyPEM()

	{
		info, x509Crt, err := validateIssuedCertificate(certPEM, keyPEM, domains)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "example.com", info.CommonName)
		assert.True(t, x509Crt.NotAfter.After(time.Now()))
	}

	{
		_, _, err := validateIssuedCertificate(certPEM, keyPEM,
			append(slices.Clone(domains), "other.example.com"))
		assert.NotNil(t, err)
	}

	{
		_, _, err := validateIssuedCertificate(certPEM, keyPEM,
			append(slices.Clone(domains), "*.example.com"))
		assert.NotNil(t, err)
	}

	{
		otherK, err := utils_cert.GenerateSelfSignedCert("example.com", domains, 24*time.Hour)
		assert.Nil(t, err)

		_, _, err = validateIssuedCertificate(certPEM, otherK.MustGetPrivateKeyPEM(), domains)
		assert.NotNil(t, err)
	}

	{
		_, _, err := validateIssuedCertificate([]byte("not-a-certificate"), keyPEM, domains)
		assert.NotNil(t, err)
	}

	{
		expiredK, err := utils_cert.GenerateSelfSignedCert("example.com", domains, -time.Hour)
		assert.Nil(t, err)

		_, _, err = validateIssuedCertificate(expiredK.MustGetCertPEM(),
			expiredK.MustGetPrivateKeyPEM(), domains)
		assert.NotNil(t, err)
	}
}

func TestCertificate(t *testing.T) {
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

	c := NewController(fakeC.OcteliumC)
	t.Cleanup(c.shutdown)

	cc, err := fakeC.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err)
	cc.Status.Domain = fmt.Sprintf("%s.octelium.org", utilrand.GetRandomStringCanonical(8))
	cc, err = fakeC.OcteliumC.CoreC().UpdateClusterConfig(ctx, cc)
	assert.Nil(t, err)

	domain := cc.Status.Domain

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

	newCrt := func(status *enterprisev1.Certificate_Status) *enterprisev1.Certificate {
		status.CertificateIssuerRef = umetav1.GetObjectReference(iss)

		crt, err := fakeC.OcteliumC.EnterpriseC().CreateCertificate(ctx, &enterprisev1.Certificate{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &enterprisev1.Certificate_Spec{
				Mode: enterprisev1.Certificate_Spec_MANAGED,
			},
			Status: status,
		})
		assert.Nil(t, err, "%+v", err)

		return crt
	}

	t.Run("ServiceDomains", func(t *testing.T) {
		crt := newCrt(&enterprisev1.Certificate_Status{
			ServiceRef:   umetav1.GetObjectReference(svc),
			NamespaceRef: svc.Status.NamespaceRef,
		})

		domains, err := c.getCertificateDomains(ctx, crt)
		assert.Nil(t, err, "%+v", err)

		assert.Equal(t, []string{
			fmt.Sprintf("%s.%s", svc.Status.PrimaryHostname, domain),
			fmt.Sprintf("%s.local.%s", svc.Status.PrimaryHostname, domain),
		}, domains)
	})

	t.Run("DefaultNamespaceDomains", func(t *testing.T) {
		crt := newCrt(&enterprisev1.Certificate_Status{
			NamespaceRef: &metav1.ObjectReference{
				Name: "default",
				Uid:  utilrand.GetRandomStringCanonical(8),
			},
		})

		domains, err := c.getCertificateDomains(ctx, crt)
		assert.Nil(t, err, "%+v", err)

		assert.Equal(t, []string{
			domain,
			fmt.Sprintf("*.default.%s", domain),
			fmt.Sprintf("*.default.local.%s", domain),
			fmt.Sprintf("*.%s", domain),
			fmt.Sprintf("*.local.%s", domain),
		}, domains)
	})

	t.Run("NamespaceDomains", func(t *testing.T) {
		ns := utilrand.GetRandomStringCanonical(8)

		crt := newCrt(&enterprisev1.Certificate_Status{
			NamespaceRef: &metav1.ObjectReference{
				Name: ns,
				Uid:  utilrand.GetRandomStringCanonical(8),
			},
		})

		domains, err := c.getCertificateDomains(ctx, crt)
		assert.Nil(t, err, "%+v", err)

		assert.Equal(t, []string{
			fmt.Sprintf("*.%s.%s", ns, domain),
			fmt.Sprintf("*.%s.local.%s", ns, domain),
		}, domains)
	})

	t.Run("InvalidDomains", func(t *testing.T) {
		{
			crt := newCrt(&enterprisev1.Certificate_Status{})
			_, err := c.getCertificateDomains(ctx, crt)
			assert.NotNil(t, err)
		}

		{
			crt := newCrt(&enterprisev1.Certificate_Status{
				NamespaceRef: &metav1.ObjectReference{
					Name: "not_a_label",
					Uid:  utilrand.GetRandomStringCanonical(8),
				},
			})

			_, err := c.getCertificateDomains(ctx, crt)
			assert.NotNil(t, err)
		}

		{
			crt := newCrt(&enterprisev1.Certificate_Status{
				ServiceRef: &metav1.ObjectReference{
					Name: utilrand.GetRandomStringCanonical(8),
					Uid:  utilrand.GetRandomStringCanonical(8),
				},
			})

			_, err := c.getCertificateDomains(ctx, crt)
			assert.NotNil(t, err)
		}
	})

	t.Run("PrepareSkipsHealthyCertificates", func(t *testing.T) {
		{
			crt := newCrt(&enterprisev1.Certificate_Status{
				NamespaceRef: svc.Status.NamespaceRef,
				Issuance: &enterprisev1.Certificate_Status_Issuance{
					CreatedAt: pbutils.Now(),
					State:     enterprisev1.Certificate_Status_Issuance_SUCCESS,
					ExpiresAt: pbutils.Timestamp(time.Now().Add(2 * renewBefore)),
				},
			})

			got, err := c.prepareCertificate(ctx, crt.Metadata.Uid)
			assert.Nil(t, err)
			assert.Nil(t, got)
		}

		{
			crt := newCrt(&enterprisev1.Certificate_Status{
				NamespaceRef: svc.Status.NamespaceRef,
				Issuance: &enterprisev1.Certificate_Status_Issuance{
					CreatedAt:         pbutils.Now(),
					State:             enterprisev1.Certificate_Status_Issuance_ISSUING,
					IssuanceStartedAt: pbutils.Now(),
				},
			})

			got, err := c.prepareCertificate(ctx, crt.Metadata.Uid)
			assert.Nil(t, err)
			assert.Nil(t, got)
		}

		{
			got, err := c.prepareCertificate(ctx, vutils.UUIDv4())
			assert.Nil(t, err)
			assert.Nil(t, got)
		}
	})

	t.Run("PrepareRequestsIssuance", func(t *testing.T) {
		for _, issuance := range []*enterprisev1.Certificate_Status_Issuance{
			nil,
			{
				CreatedAt: pbutils.Now(),
				State:     enterprisev1.Certificate_Status_Issuance_FAILED,
			},
			{
				CreatedAt:         pbutils.Now(),
				State:             enterprisev1.Certificate_Status_Issuance_ISSUING,
				IssuanceStartedAt: pbutils.Timestamp(time.Now().Add(-2 * staleIssuanceAfter)),
			},
			{
				CreatedAt: pbutils.Now(),
				State:     enterprisev1.Certificate_Status_Issuance_SUCCESS,
				ExpiresAt: pbutils.Timestamp(time.Now().Add(renewBefore / 2)),
			},
		} {
			crt := newCrt(&enterprisev1.Certificate_Status{
				NamespaceRef: svc.Status.NamespaceRef,
				Issuance:     issuance,
			})

			got, err := c.prepareCertificate(ctx, crt.Metadata.Uid)
			assert.Nil(t, err, "%+v", err)
			assert.NotNil(t, got)
			assert.Equal(t, enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
				got.Status.Issuance.State)

			if issuance != nil {
				assert.Equal(t, 1, len(got.Status.LastIssuances))
			}
		}
	})

	t.Run("PrepareSkipsManualCertificates", func(t *testing.T) {
		crt := newCrt(&enterprisev1.Certificate_Status{
			NamespaceRef: svc.Status.NamespaceRef,
		})

		crt.Spec.Mode = enterprisev1.Certificate_Spec_MANUAL
		crt, err := fakeC.OcteliumC.EnterpriseC().UpdateCertificate(ctx, crt)
		assert.Nil(t, err)

		got, err := c.prepareCertificate(ctx, crt.Metadata.Uid)
		assert.Nil(t, err)
		assert.Nil(t, got)
	})

	t.Run("ClaimAndPersist", func(t *testing.T) {
		crt := newCrt(&enterprisev1.Certificate_Status{
			ServiceRef:   umetav1.GetObjectReference(svc),
			NamespaceRef: svc.Status.NamespaceRef,
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				CreatedAt: pbutils.Now(),
				State:     enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
			},
		})

		domains, err := c.getCertificateDomains(ctx, crt)
		assert.Nil(t, err)

		claimed, attempt, err := c.claimCertificate(ctx, crt)
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, claimed)
		assert.Equal(t, enterprisev1.Certificate_Status_Issuance_ISSUING,
			claimed.Status.Issuance.State)
		assert.NotNil(t, claimed.Status.Issuance.IssuanceStartedAt)
		assert.True(t, isSameAttempt(claimed, attempt))

		{
			_, _, err := c.claimCertificate(ctx, claimed)
			assert.Nil(t, err)
		}

		crtK, err := utils_cert.GenerateSelfSignedCert(domains[0], domains, 24*time.Hour)
		assert.Nil(t, err)

		err = c.persistIssuedCertificate(ctx, claimed, attempt, domains,
			crtK.MustGetCertPEM(), crtK.MustGetPrivateKeyPEM())
		assert.Nil(t, err, "%+v", err)

		got, err := fakeC.OcteliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.Equal(t, enterprisev1.Certificate_Status_Issuance_SUCCESS,
			got.Status.Issuance.State)
		assert.Equal(t, uint32(1), got.Status.SuccessfulIssuances)
		assert.NotNil(t, got.Status.SecretRef)
		assert.NotNil(t, got.Status.Info)
		assert.Equal(t, domains[0], got.Status.Info.CommonName)
		assert.Equal(t, crtK.Certificate.NotAfter.Unix(),
			got.Status.Issuance.ExpiresAt.AsTime().Unix())

		sec, err := fakeC.OcteliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: uenterprisev1.ToCertificate(got).GetSecretName(),
		})
		assert.Nil(t, err)
		assert.True(t, sec.Metadata.IsSystem)

		chain, key, err := ucorev1.ToSecret(sec).GetCertificateChainAndKey()
		assert.Nil(t, err)
		assert.Equal(t, string(crtK.MustGetCertPEM()), string(chain))
		assert.Equal(t, string(crtK.MustGetPrivateKeyPEM()), string(key))

		{
			err := c.persistIssuedCertificate(ctx, claimed, attempt, domains,
				crtK.MustGetCertPEM(), crtK.MustGetPrivateKeyPEM())
			assert.Nil(t, err, "%+v", err)

			cur, err := fakeC.OcteliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
				Uid: crt.Metadata.Uid,
			})
			assert.Nil(t, err)
			assert.Equal(t, uint32(1), cur.Status.SuccessfulIssuances)
		}

		{
			err := c.persistIssuedCertificate(ctx, claimed, attempt, domains,
				crtK.MustGetCertPEM(), []byte("not-a-key"))
			assert.NotNil(t, err)
		}
	})

	t.Run("MarkAttemptFailed", func(t *testing.T) {
		crt := newCrt(&enterprisev1.Certificate_Status{
			NamespaceRef: svc.Status.NamespaceRef,
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				CreatedAt: pbutils.Now(),
				State:     enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
			},
		})

		claimed, attempt, err := c.claimCertificate(ctx, crt)
		assert.Nil(t, err)

		c.markAttemptFailed(ctx, claimed.Metadata.Uid, attempt,
			fmt.Errorf("Could not obtain the Certificate"))

		got, err := fakeC.OcteliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.Equal(t, enterprisev1.Certificate_Status_Issuance_FAILED,
			got.Status.Issuance.State)
		assert.Equal(t, uint32(1), got.Status.FailedIssuances)
		assert.NotNil(t, got.Status.Issuance.IssuanceCompletedAt)

		c.markAttemptFailed(ctx, claimed.Metadata.Uid, attempt,
			fmt.Errorf("Could not obtain the Certificate"))

		got, err = fakeC.OcteliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)
		assert.Equal(t, uint32(1), got.Status.FailedIssuances)
	})

	t.Run("UpsertCertificateSecret", func(t *testing.T) {
		crt := newCrt(&enterprisev1.Certificate_Status{
			NamespaceRef: svc.Status.NamespaceRef,
		})

		crtK, err := utils_cert.GenerateCARoot()
		assert.Nil(t, err)

		sec, err := c.upsertCertificateSecret(ctx, crt,
			crtK.MustGetCertPEM(), crtK.MustGetPrivateKeyPEM())
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uenterprisev1.ToCertificate(crt).GetSecretName(), sec.Metadata.Name)
		assert.Equal(t, "true", sec.Metadata.SystemLabels["octelium-cert"])

		otherK, err := utils_cert.GenerateCARoot()
		assert.Nil(t, err)

		updated, err := c.upsertCertificateSecret(ctx, crt,
			otherK.MustGetCertPEM(), otherK.MustGetPrivateKeyPEM())
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, sec.Metadata.Uid, updated.Metadata.Uid)

		chain, _, err := ucorev1.ToSecret(updated).GetCertificateChainAndKey()
		assert.Nil(t, err)
		assert.Equal(t, string(otherK.MustGetCertPEM()), string(chain))
	})

	t.Run("GetReadyIssuer", func(t *testing.T) {
		{
			_, err := c.getReadyIssuer(ctx, &metav1.ObjectReference{
				Uid: utilrand.GetRandomStringCanonical(8),
			})
			assert.NotNil(t, err)
		}

		{
			_, err := c.getReadyIssuer(ctx, umetav1.GetObjectReference(iss))
			assert.NotNil(t, err)
		}

		cur, err := fakeC.OcteliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{
			Uid: iss.Metadata.Uid,
		})
		assert.Nil(t, err)

		cur.Status.State = enterprisev1.CertificateIssuer_Status_READY
		cur.Status.Type = &enterprisev1.CertificateIssuer_Status_Acme{
			Acme: &enterprisev1.CertificateIssuer_Status_ACME{
				SecretRef: &metav1.ObjectReference{
					Uid: utilrand.GetRandomStringCanonical(8),
				},
			},
		}

		cur, err = fakeC.OcteliumC.EnterpriseC().UpdateCertificateIssuer(ctx, cur)
		assert.Nil(t, err)

		got, err := c.getReadyIssuer(ctx, umetav1.GetObjectReference(cur))
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, cur.Metadata.Uid, got.Metadata.Uid)
	})

	t.Run("ReconcileSkipsHealthyCertificates", func(t *testing.T) {
		crt := newCrt(&enterprisev1.Certificate_Status{
			NamespaceRef: svc.Status.NamespaceRef,
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				CreatedAt: pbutils.Now(),
				State:     enterprisev1.Certificate_Status_Issuance_SUCCESS,
				ExpiresAt: pbutils.Timestamp(time.Now().Add(2 * renewBefore)),
			},
		})

		assert.Nil(t, c.reconcileCertificate(ctx, crt.Metadata.Uid))

		got, err := fakeC.OcteliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)
		assert.Equal(t, crt.Metadata.ResourceVersion, got.Metadata.ResourceVersion)
	})

	t.Run("ReconcileDoesNotClaimOnAPreFlightFailure", func(t *testing.T) {
		crt := newCrt(&enterprisev1.Certificate_Status{
			ServiceRef:   umetav1.GetObjectReference(svc),
			NamespaceRef: svc.Status.NamespaceRef,
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				CreatedAt: pbutils.Now(),
				State:     enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
			},
		})

		err := c.reconcileCertificate(ctx, crt.Metadata.Uid)
		assert.NotNil(t, err)

		got, err := fakeC.OcteliumC.EnterpriseC().GetCertificate(ctx, &rmetav1.GetOptions{
			Uid: crt.Metadata.Uid,
		})
		assert.Nil(t, err)

		assert.Equal(t, enterprisev1.Certificate_Status_Issuance_ISSUANCE_REQUESTED,
			got.Status.Issuance.State)
		assert.Equal(t, uint32(0), got.Status.FailedIssuances)
		assert.Nil(t, got.Status.Issuance.IssuanceStartedAt)
	})

	t.Run("SetDNSProvider", func(t *testing.T) {
		crt := newCrt(&enterprisev1.Certificate_Status{
			NamespaceRef: svc.Status.NamespaceRef,
			Issuance: &enterprisev1.Certificate_Status_Issuance{
				CreatedAt: pbutils.Now(),
				State:     enterprisev1.Certificate_Status_Issuance_FAILED,
			},
		})

		key := workKey{
			typ: workTypeCertificate,
			uid: crt.Metadata.Uid,
		}

		c.scheduleRetry(key, fmt.Errorf("Could not obtain the Certificate"))

		assert.Nil(t, c.SetDNSProvider(ctx))

		c.mu.Lock()
		_, hasFailures := c.failures[key]
		_, isPending := c.pending[key]
		c.mu.Unlock()

		assert.False(t, hasFailures)
		assert.True(t, isPending)
	})

	t.Run("Resync", func(t *testing.T) {
		assert.Nil(t, c.resync(ctx))
	})
}
