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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeDomain(t *testing.T) {
	assert.Equal(t, "example.com", normalizeDomain("Example.Com."))
	assert.Equal(t, "example.com", normalizeDomain("  example.com  "))
	assert.Equal(t, "", normalizeDomain(""))
}

func TestJoinDomain(t *testing.T) {
	assert.Equal(t, "svc.example.com", joinDomain("svc", "example.com"))
	assert.Equal(t, "svc.ns.local.example.com", joinDomain("svc.ns", "local.example.com"))
	assert.Equal(t, "example.com", joinDomain("", "example.com"))
	assert.Equal(t, "svc.example.com", joinDomain("SVC.", "Example.com."))
}

func TestUniqueDomains(t *testing.T) {
	assert.Equal(t, []string{"example.com", "a.example.com"},
		uniqueDomains([]string{"example.com", "A.example.com.", "a.example.com", ""}))
}

func TestIsValidDNSName(t *testing.T) {
	assert.True(t, isValidDNSName("example.com", false))
	assert.True(t, isValidDNSName("a-b.c.example.com", false))
	assert.True(t, isValidDNSName("*.example.com", true))

	assert.False(t, isValidDNSName("*.example.com", false))
	assert.False(t, isValidDNSName("", true))
	assert.False(t, isValidDNSName("example", true))
	assert.False(t, isValidDNSName("-a.example.com", true))
	assert.False(t, isValidDNSName("a-.example.com", true))
	assert.False(t, isValidDNSName("a..example.com", true))
	assert.False(t, isValidDNSName("a_b.example.com", true))
	assert.False(t, isValidDNSName("*.*.example.com", true))
	assert.False(t, isValidDNSName(strings.Repeat("a", 64)+".example.com", true))
	assert.False(t, isValidDNSName(strings.Repeat("a.", 200)+"example.com", true))
}

func TestIsValidDNSLabel(t *testing.T) {
	assert.True(t, isValidDNSLabel("a"))
	assert.True(t, isValidDNSLabel("a-b-1"))
	assert.False(t, isValidDNSLabel(""))
	assert.False(t, isValidDNSLabel("-a"))
	assert.False(t, isValidDNSLabel("a."))
	assert.False(t, isValidDNSLabel(strings.Repeat("a", 64)))
}

func TestValidateDirectoryURL(t *testing.T) {
	assert.Nil(t, validateDirectoryURL("https://acme-v02.api.letsencrypt.org/directory"))

	assert.NotNil(t, validateDirectoryURL("ftp://example.com/directory"))
	assert.NotNil(t, validateDirectoryURL("https:///directory"))
	assert.NotNil(t, validateDirectoryURL("https://user:pass@example.com/directory"))
	assert.NotNil(t, validateDirectoryURL(""))
}

func TestGetCADirURL(t *testing.T) {
	{
		iss := &enterprisev1.CertificateIssuer{
			Spec: &enterprisev1.CertificateIssuer_Spec{
				Type: &enterprisev1.CertificateIssuer_Spec_Acme{
					Acme: &enterprisev1.CertificateIssuer_Spec_ACME{
						Server: " https://acme.example.com/directory ",
					},
				},
			},
		}

		assert.Equal(t, "https://acme.example.com/directory", getCADirURL(iss))
	}

	{
		iss := &enterprisev1.CertificateIssuer{
			Spec: &enterprisev1.CertificateIssuer_Spec{
				Type: &enterprisev1.CertificateIssuer_Spec_Acme{
					Acme: &enterprisev1.CertificateIssuer_Spec_ACME{},
				},
			},
		}

		assert.Nil(t, validateDirectoryURL(getCADirURL(iss)))
	}
}

func TestSleepContext(t *testing.T) {
	assert.Nil(t, sleepContext(context.Background(), time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.NotNil(t, sleepContext(ctx, time.Hour))
}

func TestACMEHTTPClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("octelium"))
	}))
	t.Cleanup(srv.Close)

	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		resp, err := newACMEHTTPClient(ctx).Get(srv.URL)
		assert.Nil(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		assert.Equal(t, "octelium", string(body))
	}

	{
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := newACMEHTTPClient(ctx).Get(srv.URL)
		assert.NotNil(t, err)
	}
}
