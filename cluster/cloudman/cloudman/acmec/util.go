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
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/lego"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
)

type contextRoundTripper struct {
	ctx  context.Context
	base http.RoundTripper
}

type cancelReadCloser struct {
	io.ReadCloser
	stop   func() bool
	cancel context.CancelFunc
}

func (c *cancelReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.stop()
	c.cancel()
	return err
}

func (t *contextRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.ctx.Err(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(req.Context())
	stop := context.AfterFunc(t.ctx, cancel)

	resp, err := t.base.RoundTrip(req.Clone(ctx))
	if err != nil {
		stop()
		cancel()
		return nil, err
	}

	resp.Body = &cancelReadCloser{
		ReadCloser: resp.Body,
		stop:       stop,
		cancel:     cancel,
	}

	return resp, nil
}

func newACMEHTTPClient(ctx context.Context) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 20
	transport.MaxIdleConnsPerHost = 10
	transport.IdleConnTimeout = 90 * time.Second
	transport.TLSHandshakeTimeout = 15 * time.Second
	transport.ResponseHeaderTimeout = 60 * time.Second
	transport.ExpectContinueTimeout = 2 * time.Second

	return &http.Client{
		Transport: &contextRoundTripper{
			ctx:  ctx,
			base: transport,
		},
		Timeout: acmeHTTPTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.Errorf("Too many ACME redirects")
			}
			if !isAllowedACMEScheme(req.URL.Scheme) {
				return errors.Errorf("ACME redirects must use HTTPS")
			}
			return nil
		},
	}
}

func getCADirURL(iss *enterprisev1.CertificateIssuer) string {
	if iss.Spec.GetAcme() != nil && iss.Spec.GetAcme().Server != "" {
		return strings.TrimSpace(iss.Spec.GetAcme().Server)
	}

	if ldflags.IsDev() || ldflags.IsTest() {
		return lego.LEDirectoryStaging
	}

	return lego.LEDirectoryProduction
}

func isAllowedACMEScheme(scheme string) bool {
	switch scheme {
	case "https":
		return true
	case "http":
		return ldflags.IsDev() || ldflags.IsTest()
	default:
		return false
	}
}

func validateDirectoryURL(arg string) error {
	u, err := url.Parse(arg)
	if err != nil {
		return errors.Errorf("Invalid ACME directory URL: %s", arg)
	}

	if !isAllowedACMEScheme(u.Scheme) {
		return errors.Errorf("The ACME directory URL must use HTTPS: %s", arg)
	}

	if u.Hostname() == "" || u.User != nil {
		return errors.Errorf("Invalid ACME directory URL: %s", arg)
	}

	return nil
}

func normalizeDomain(arg string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(arg), "."))
}

func joinDomain(prefix, domain string) string {
	prefix = normalizeDomain(prefix)
	domain = normalizeDomain(domain)

	if prefix == "" {
		return domain
	}

	return prefix + "." + domain
}

func uniqueDomains(domains []string) []string {
	ret := make([]string, 0, len(domains))
	seen := make(map[string]struct{}, len(domains))

	for _, domain := range domains {
		domain = normalizeDomain(domain)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		ret = append(ret, domain)
	}

	return ret
}

func isValidDNSName(arg string, allowWildcard bool) bool {
	arg = normalizeDomain(arg)
	if arg == "" || len(arg) > 253 {
		return false
	}

	if strings.HasPrefix(arg, "*.") {
		if !allowWildcard {
			return false
		}
		arg = strings.TrimPrefix(arg, "*.")
	}

	labels := strings.Split(arg, ".")
	if len(labels) < 2 {
		return false
	}

	return !slices.ContainsFunc(labels, func(label string) bool {
		return !isValidDNSLabel(label)
	})
}

func isValidDNSLabel(arg string) bool {
	if arg == "" || len(arg) > 63 ||
		arg[0] == '-' || arg[len(arg)-1] == '-' {
		return false
	}

	for _, c := range arg {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '-':
		default:
			return false
		}
	}

	return true
}

func sleepContext(ctx context.Context, arg time.Duration) error {
	timer := time.NewTimer(arg)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
