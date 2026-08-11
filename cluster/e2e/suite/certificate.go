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
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	utils_cert "github.com/octelium/octelium/pkg/utils/cert"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const certificateBudget = 3 * time.Minute

func testCertificateSetAndServe(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	certList, err := h.EnterpriseC().ListCertificate(ctx,
		&enterprisev1.ListCertificateOptions{})
	require.Nil(t, err)
	require.True(t, len(certList.Items) > 0, "the Cluster has no Certificate resources")

	cert := certList.Items[0]

	svc := h.NewPublicService(t, "default")
	fqdn := vutils.GetServicePublicFQDN(svc, h.Domain)

	serial := func(t *testing.T) string {
		t.Helper()

		leaf, err := peerCertificate(ctx, fqdn)
		require.Nil(t, err)

		return leaf.SerialNumber.String()
	}

	before := serial(t)

	crt, err := utils_cert.GenerateSelfSignedCert(h.Domain, []string{
		h.Domain,
		fmt.Sprintf("*.%s", h.Domain),
		fmt.Sprintf("*.default.%s", h.Domain),
		fmt.Sprintf("*.octelium.%s", h.Domain),
		fmt.Sprintf("*.octelium-api.%s", h.Domain),
	}, 24*time.Hour)
	require.Nil(t, err)

	crtPEM, err := crt.GetCertPEM()
	require.Nil(t, err)

	keyPEM, err := crt.GetPrivateKeyPEM()
	require.Nil(t, err)

	_, err = h.EnterpriseC().SetCertificate(ctx, &enterprisev1.SetCertificateRequest{
		CertificateRef: &metav1.ObjectReference{
			ApiVersion: "enterprise/v1",
			Kind:       "Certificate",
			Name:       cert.Metadata.Name,
			Uid:        cert.Metadata.Uid,
		},
		Certificate: crtPEM,
		PrivateKey:  keyPEM,
	})
	require.Nil(t, err)

	t.Run("TheIngressServesTheNewLeaf", func(t *testing.T) {
		h.Eventually(t, "the ingress to serve the new leaf certificate", certificateBudget,
			func(ctx context.Context) error {
				leaf, err := peerCertificate(ctx, fqdn)
				if err != nil {
					return err
				}
				if leaf.SerialNumber.String() == before {
					return errors.Errorf("the ingress still serves the previous leaf")
				}
				return nil
			})
	})

	t.Run("TheCertificateResourceIsUpdated", func(t *testing.T) {
		cur, err := h.EnterpriseC().GetCertificate(ctx,
			&metav1.GetOptions{Uid: cert.Metadata.Uid})
		require.Nil(t, err)
		assert.NotNil(t, cur.Status)
	})
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
