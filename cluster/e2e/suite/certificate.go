// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"slices"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const certificateBudget = 3 * time.Minute

func testCertificateSetAndServe(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	ns := h.CreateNamespace(t, nil)

	crt := waitNamespaceCertificate(t, h, ns)
	require.Equal(t, enterprisev1.Certificate_Spec_MANUAL, crt.Spec.Mode)

	svc := h.NewPublicService(t, ns.Metadata.Name)
	fqdn := vutils.GetServicePublicFQDN(svc, h.Domain)

	wildcard := fmt.Sprintf("*.%s.%s", ns.Metadata.Name, h.Domain)

	var before *x509.Certificate

	h.Eventually(t, "the ingress to terminate TLS for the new Service",
		eeharness.PropagationBudget, func(ctx context.Context) error {
			cur, err := peerCertificate(ctx, fqdn)
			if err != nil {
				return err
			}
			before = cur
			return nil
		})

	leaf, err := utils_cert.GenerateSelfSignedCert(wildcard,
		[]string{wildcard, fqdn}, 24*time.Hour)
	require.Nil(t, err)

	crtPEM, err := leaf.GetCertPEM()
	require.Nil(t, err)

	keyPEM, err := leaf.GetPrivateKeyPEM()
	require.Nil(t, err)

	t.Run("AMismatchedKeyPairIsRefused", func(t *testing.T) {
		other, err := utils_cert.GenerateSelfSignedCert(fqdn, []string{fqdn}, 24*time.Hour)
		require.Nil(t, err)

		otherKey, err := other.GetPrivateKeyPEM()
		require.Nil(t, err)

		_, err = h.EnterpriseC().SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: certificateRef(crt),
			Certificate:    crtPEM,
			PrivateKey:     otherKey,
		})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	})

	t.Run("AGarbagePEMIsRefused", func(t *testing.T) {
		_, err := h.EnterpriseC().SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
			CertificateRef: certificateRef(crt),
			Certificate:    "not-a-certificate",
			PrivateKey:     keyPEM,
		})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	})

	_, err = h.EnterpriseC().SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
		CertificateRef: certificateRef(crt),
		Certificate:    crtPEM,
		PrivateKey:     keyPEM,
	})
	require.Nil(t, err)

	t.Run("TheCertificateCarriesTheLeafInfo", func(t *testing.T) {
		cur, err := h.EnterpriseC().GetCertificate(ctx,
			&metav1.GetOptions{Uid: crt.Metadata.Uid})
		require.Nil(t, err)
		require.NotNil(t, cur.Status)
		require.NotNil(t, cur.Status.Info)
		require.NotNil(t, cur.Status.SecretRef)

		assert.Equal(t, wildcard, cur.Status.Info.CommonName)
		assert.True(t, slices.Contains(cur.Status.Info.DnsNames, fqdn))
		require.NotNil(t, cur.Status.Info.NotAfter)
		assert.True(t, cur.Status.Info.NotAfter.AsTime().After(time.Now()))
	})

	t.Run("TheIngressServesTheNewLeaf", func(t *testing.T) {
		h.Eventually(t, "the ingress to serve the new leaf certificate", certificateBudget,
			func(ctx context.Context) error {
				cur, err := peerCertificate(ctx, fqdn)
				if err != nil {
					return err
				}
				if cur.SerialNumber.Cmp(before.SerialNumber) == 0 {
					return errors.Errorf("the ingress still serves the previous leaf")
				}
				if !slices.Contains(cur.DNSNames, fqdn) {
					return errors.Errorf("the served leaf does not cover %s", fqdn)
				}
				return nil
			})
	})

	t.Run("TheClusterCertificateIsUntouched", func(t *testing.T) {
		cur, err := peerCertificate(ctx, h.Domain)
		require.Nil(t, err)
		assert.NotEqual(t, 0, cur.SerialNumber.Cmp(leaf.Certificate.SerialNumber))
	})

	t.Run("TheSystemCertificatesAreProtected", func(t *testing.T) {
		list, err := h.EnterpriseC().ListCertificate(ctx,
			&enterprisev1.ListCertificateOptions{})
		require.Nil(t, err)

		for _, itm := range list.Items {
			if !itm.Metadata.IsSystem {
				continue
			}

			_, err := h.EnterpriseC().SetCertificate(ctx,
				&enterprisev1.SetCertificateRequest{
					CertificateRef: certificateRef(itm),
					Certificate:    crtPEM,
					PrivateKey:     keyPEM,
				})
			assert.NotNil(t, err,
				"the system Certificate %s accepted a foreign leaf", itm.Metadata.Name)
		}
	})
}

func waitNamespaceCertificate(t *testing.T, h *eeharness.H,
	ns *corev1.Namespace) *enterprisev1.Certificate {
	t.Helper()

	var ret *enterprisev1.Certificate

	h.Eventually(t, fmt.Sprintf("cloudman to provision a Certificate for the Namespace %s",
		ns.Metadata.Name), eeharness.PropagationBudget, func(ctx context.Context) error {
		list, err := h.EnterpriseC().ListCertificate(ctx,
			&enterprisev1.ListCertificateOptions{})
		if err != nil {
			return err
		}

		idx := slices.IndexFunc(list.Items, func(itm *enterprisev1.Certificate) bool {
			return itm.Status != nil && itm.Status.ServiceRef == nil &&
				itm.Status.NamespaceRef != nil &&
				itm.Status.NamespaceRef.Uid == ns.Metadata.Uid
		})
		if idx < 0 {
			return errNotProvisioned("the Namespace Certificate")
		}

		ret = list.Items[idx]
		return nil
	})

	return ret
}

func certificateRef(crt *enterprisev1.Certificate) *metav1.ObjectReference {
	return &metav1.ObjectReference{
		ApiVersion: "enterprise/v1",
		Kind:       "Certificate",
		Name:       crt.Metadata.Name,
		Uid:        crt.Metadata.Uid,
	}
}

func peerCertificate(ctx context.Context, fqdn string) (*x509.Certificate, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	conn, err := tls.DialWithDialer(dialer, "tcp", fmt.Sprintf("%s:443", fqdn),
		&tls.Config{
			ServerName:         fqdn,
			InsecureSkipVerify: true,
		})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) < 1 {
		return nil, errors.Errorf("the peer presented no certificate")
	}

	return certs[0], nil
}
