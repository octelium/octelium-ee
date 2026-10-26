// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cluster

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetClusterInfo(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv, err := NewServer(tst.C.OcteliumC)
	assert.Nil(t, err)

	oldGetLatestVersionFunc := getLatestVersionFunc
	getLatestVersionFunc = func(ctx context.Context, pkg string) (string, error) {
		switch pkg {
		case "octelium":
			return "1.1.0", nil
		case "octelium-ee":
			return "2.0.0", nil
		case "cordium":
			return "1.0.0", nil
		default:
			return "", nil
		}
	}
	t.Cleanup(func() {
		getLatestVersionFunc = oldGetLatestVersionFunc
	})

	{
		_, err := srv.GetClusterInfo(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: "default"})
		assert.Nil(t, err, "%+v", err)
		if rgn.Status == nil {
			rgn.Status = &corev1.Region_Status{}
		}

		setAt := pbutils.Now()
		rgn.Status.VersionInfoMap = map[string]*corev1.Region_Status_VersionInfo{
			"octelium": {
				Version: "1.0.0",
				SetAt:   setAt,
			},
			"octeliumee": {
				Version: "2.0.0",
				SetAt:   setAt,
			},
			"cordium": {
				Version: "dev",
				SetAt:   setAt,
			},
		}
		_, err = srv.octeliumC.CoreC().UpdateRegion(ctx, rgn)
		assert.Nil(t, err, "%+v", err)

		resp, err := srv.GetClusterInfo(ctx, &enterprisev1.GetClusterInfoRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, resp.Core)
		assert.NotNil(t, resp.PackageEnterprise)
		assert.NotNil(t, resp.PackageCordium)
		assert.Equal(t, "1.0.0", resp.Core.CurrentVersion)
		assert.Equal(t, "1.1.0", resp.Core.LatestVersion)
		assert.True(t, resp.Core.CanUpgrade)
		assert.Equal(t, setAt.AsTime(), resp.Core.SetAt.AsTime())
		assert.Equal(t, "2.0.0", resp.PackageEnterprise.CurrentVersion)
		assert.Equal(t, "2.0.0", resp.PackageEnterprise.LatestVersion)
		assert.False(t, resp.PackageEnterprise.CanUpgrade)
		assert.Equal(t, "dev", resp.PackageCordium.CurrentVersion)
		assert.Equal(t, "1.0.0", resp.PackageCordium.LatestVersion)
		assert.False(t, resp.PackageCordium.CanUpgrade)
	}

	{
		assert.True(t, canUpgradeVersion("1.0.0", "1.0.1"))
		assert.False(t, canUpgradeVersion("1.0.1", "1.0.0"))
		assert.False(t, canUpgradeVersion("", "1.0.0"))
		assert.False(t, canUpgradeVersion("1.0.0", ""))
		assert.False(t, canUpgradeVersion("dev", "1.0.0"))
		assert.False(t, canUpgradeVersion("1.0.0", "latest"))
	}
}

type versionRoundTripper func(*http.Request) (*http.Response, error)

func (f versionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestVersionFetcher(t *testing.T, handler func(*http.Request) (*http.Response, error)) *versionFetcher {
	t.Helper()

	f := newVersionFetcher()
	f.client.SetDebug(false).SetTransport(versionRoundTripper(handler))
	return f
}

func versionHTTPResponse(code int, body string, headers http.Header) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: code,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestVersionFetcherReleaseCache(t *testing.T) {
	ctx := context.Background()
	calls := 0
	f := newTestVersionFetcher(t, func(req *http.Request) (*http.Response, error) {
		calls++
		assert.Equal(t, "api.github.com", req.URL.Host)
		assert.Equal(t, "/repos/octelium/octelium-ee/releases/latest", req.URL.Path)
		assert.Equal(t, "application/vnd.github+json", req.Header.Get("Accept"))
		assert.Equal(t, githubAPIVersion, req.Header.Get("X-GitHub-Api-Version"))
		if calls == 1 {
			assert.Empty(t, req.Header.Get("If-None-Match"))
			return versionHTTPResponse(http.StatusOK, `{"tag_name":"v1.2.3"}`,
				http.Header{"Etag": {`"release-1"`}}), nil
		}
		assert.Equal(t, `"release-1"`, req.Header.Get("If-None-Match"))
		return versionHTTPResponse(http.StatusNotModified, "", nil), nil
	})

	for range 2 {
		ver, err := f.fromReleases(ctx, repoPackageEnterprise)
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, "1.2.3", ver)
	}
	assert.Equal(t, 2, calls)
	assert.Empty(t, f.entries[repoCore].version)
}

func TestVersionFetcherRejectsRelease(t *testing.T) {
	for _, tt := range []struct {
		name string
		code int
		body string
	}{
		{name: "Draft", code: http.StatusOK, body: `{"tag_name":"v1.2.3","draft":true}`},
		{name: "Prerelease", code: http.StatusOK, body: `{"tag_name":"v1.2.3-rc.1","prerelease":true}`},
		{name: "EmptyTag", code: http.StatusOK, body: `{}`},
		{name: "InvalidTag", code: http.StatusOK, body: `{"tag_name":"bad tag"}`},
		{name: "InvalidJSON", code: http.StatusOK, body: `{`},
		{name: "NotFound", code: http.StatusNotFound, body: `{}`},
		{name: "ServerError", code: http.StatusInternalServerError, body: `{}`},
		{name: "UncachedNotModified", code: http.StatusNotModified},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestVersionFetcher(t, func(req *http.Request) (*http.Response, error) {
				return versionHTTPResponse(tt.code, tt.body, nil), nil
			})
			ver, err := f.fromReleases(t.Context(), repoCore)
			assert.NotNil(t, err)
			assert.Empty(t, ver)
			assert.Empty(t, f.entries)
		})
	}
}

func TestVersionFetcherFallback(t *testing.T) {
	for _, tt := range []struct {
		name string
		code int
		body string
	}{
		{name: "Unavailable", code: http.StatusServiceUnavailable},
		{name: "UnusableRelease", code: http.StatusOK, body: `{"tag_name":""}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var hosts []string
			f := newTestVersionFetcher(t, func(req *http.Request) (*http.Response, error) {
				hosts = append(hosts, req.URL.Host)
				if req.URL.Host == "api.github.com" {
					return versionHTTPResponse(tt.code, tt.body, nil), nil
				}
				assert.Equal(t, "/octelium/octelium/refs/heads/main/unsorted/latest_release", req.URL.Path)
				return versionHTTPResponse(http.StatusOK, "  1.4.2\n", nil), nil
			})
			ver, err := f.latestVersion(t.Context(), repoCore)
			require.Nil(t, err, "%+v", err)
			assert.Equal(t, "1.4.2", ver)
			assert.Equal(t, []string{"api.github.com", "raw.githubusercontent.com"}, hosts)
		})
	}

	t.Run("BothSourcesFail", func(t *testing.T) {
		f := newTestVersionFetcher(t, func(req *http.Request) (*http.Response, error) {
			return versionHTTPResponse(http.StatusServiceUnavailable, "", nil), nil
		})
		ver, err := f.latestVersion(t.Context(), repoCore)
		assert.NotNil(t, err)
		assert.Empty(t, ver)
	})

	t.Run("CancelledRequestDoesNotFallBack", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		calls := 0
		f := newTestVersionFetcher(t, func(req *http.Request) (*http.Response, error) {
			calls++
			cancel()
			return nil, context.Canceled
		})
		_, err := f.latestVersion(ctx, repoCore)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, calls)
	})

	t.Run("TransportFailure", func(t *testing.T) {
		failure := errors.New("connection refused")
		f := newTestVersionFetcher(t, func(req *http.Request) (*http.Response, error) {
			return nil, failure
		})
		_, err := f.latestVersion(t.Context(), repoCore)
		assert.ErrorIs(t, err, failure)
	})
}

func TestVersionFetcherRateLimit(t *testing.T) {
	for _, tt := range []struct {
		name    string
		code    int
		headers http.Header
		limited bool
	}{
		{name: "Success", code: http.StatusOK, headers: http.Header{"Retry-After": {"60"}}},
		{name: "ForbiddenWithoutLimit", code: http.StatusForbidden},
		{name: "RetryAfter", code: http.StatusTooManyRequests, headers: http.Header{"Retry-After": {"60"}}, limited: true},
		{name: "Exhausted", code: http.StatusForbidden, headers: http.Header{"X-Ratelimit-Remaining": {"0"}}, limited: true},
		{name: "Reset", code: http.StatusForbidden, headers: http.Header{
			"X-Ratelimit-Remaining": {"0"},
			"X-Ratelimit-Reset":     {strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10)},
		}, limited: true},
		{name: "InvalidReset", code: http.StatusForbidden, headers: http.Header{
			"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {"invalid"},
		}, limited: true},
		{name: "DistantReset", code: http.StatusForbidden, headers: http.Header{
			"X-Ratelimit-Remaining": {"0"},
			"X-Ratelimit-Reset":     {strconv.FormatInt(time.Now().Add(48*time.Hour).Unix(), 10)},
		}, limited: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newVersionFetcher()
			f.trackRateLimit(nil)
			f.trackRateLimit(&resty.Response{RawResponse: &http.Response{StatusCode: tt.code, Header: tt.headers}})
			assert.Equal(t, tt.limited, f.isRateLimited())
			if tt.limited {
				assert.WithinDuration(t, time.Now(), f.rateLimitedUntil, versionMaxRateLimitDelay)
			}
		})
	}

	t.Run("BackoffOnlyExtends", func(t *testing.T) {
		f := newVersionFetcher()
		until := time.Now().Add(time.Hour)
		f.setRateLimitedUntil(until)
		f.setRateLimitedUntil(time.Now())
		assert.Equal(t, until, f.rateLimitedUntil)
		f.client.SetTransport(versionRoundTripper(func(req *http.Request) (*http.Response, error) {
			t.Error("rate-limited fetch must not issue an HTTP request")
			return nil, errors.New("unexpected request")
		}))
		_, err := f.fromReleases(t.Context(), repoCore)
		assert.NotNil(t, err)
	})
}

func TestParseRetryAfter(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  time.Duration
	}{
		{value: ""},
		{value: "invalid"},
		{value: "-1"},
		{value: "0"},
		{value: " 42 ", want: 42 * time.Second},
		{value: "172800", want: versionMaxRateLimitDelay},
		{value: time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)},
		{value: time.Now().Add(48 * time.Hour).UTC().Format(http.TimeFormat), want: versionMaxRateLimitDelay},
	} {
		t.Run(tt.value, func(t *testing.T) {
			assert.Equal(t, tt.want, parseRetryAfter(tt.value))
		})
	}

	retryAt := time.Now().Add(time.Minute).UTC().Truncate(time.Second)
	delay := parseRetryAfter(retryAt.Format(http.TimeFormat))
	assert.InDelta(t, time.Until(retryAt), delay, float64(time.Second))
}

func TestSanitizeVersion(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  string
	}{
		{value: ""},
		{value: " \n\t "},
		{value: " v1.2.3 \n", want: "1.2.3"},
		{value: "V1.2.3", want: "1.2.3"},
		{value: "v1.2.3-rc.1+build", want: "1.2.3-rc.1+build"},
		{value: "1.2.3\nignored", want: "1.2.3"},
		{value: "1.2.3\rignored", want: "1.2.3"},
		{value: "dev", want: "dev"},
		{value: "vnext", want: "vnext"},
		{value: "v", want: "v"},
		{value: "bad tag"},
		{value: "1.2.\x003"},
		{value: "1.2.\x7f3"},
		{value: "版本"},
		{value: strings.Repeat("a", versionMaxLen), want: strings.Repeat("a", versionMaxLen)},
		{value: strings.Repeat("a", versionMaxLen+1)},
	} {
		t.Run(tt.value, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeVersion(tt.value))
		})
	}
}

func TestVersionCacheRefresh(t *testing.T) {
	ver := "v1.2.3"
	var fetchErr error
	c := newVersionCacheWithFetch(func(ctx context.Context, repo string) (string, error) {
		assert.Equal(t, repoCore, repo)
		return ver, fetchErr
	})

	c.refresh(t.Context(), repoCore)
	entry := c.entries[repoCore]
	assert.Equal(t, "1.2.3", entry.version)
	assert.False(t, entry.fetchedAt.IsZero())
	assert.Equal(t, entry.fetchedAt, entry.lastAttemptAt)
	assert.Zero(t, entry.failures)

	fetchErr = errors.New("offline")
	c.refresh(t.Context(), repoCore)
	assert.Equal(t, entry.version, c.entries[repoCore].version)
	assert.Equal(t, entry.fetchedAt, c.entries[repoCore].fetchedAt)
	assert.Equal(t, uint32(1), c.entries[repoCore].failures)

	fetchErr = nil
	ver = "bad version"
	c.refresh(t.Context(), repoCore)
	assert.Equal(t, "1.2.3", c.entries[repoCore].version)
	assert.Equal(t, uint32(2), c.entries[repoCore].failures)

	ver = "v1.3.0"
	c.refresh(t.Context(), repoCore)
	assert.Equal(t, "1.3.0", c.entries[repoCore].version)
	assert.Zero(t, c.entries[repoCore].failures)

	c.entries[repoCore] = versionEntry{version: "1.0.0", fetchedAt: time.Now().Add(-2 * versionStaleAfter)}
	c.recordFailure(repoCore, errors.New("offline"))
	assert.Equal(t, "1.0.0", c.entries[repoCore].version)
	c.recordFailure(repoPackageCordium, errors.New("offline"))
	assert.Empty(t, c.entries[repoPackageCordium].version)
	assert.Equal(t, uint32(1), c.entries[repoPackageCordium].failures)
}

func TestVersionCachePolling(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var calls atomic.Int32
	c := newVersionCacheWithFetch(func(ctx context.Context, repo string) (string, error) {
		calls.Add(1)
		return "1.2.3", nil
	})

	c.start(ctx)
	c.start(ctx)
	require.Eventually(t, func() bool {
		select {
		case <-c.ready:
			return true
		default:
			return false
		}
	}, 3*time.Second, 10*time.Millisecond)
	assert.Equal(t, int32(len(versionRepos)), calls.Load())
	got := c.getAll(ctx)
	for _, repo := range versionRepos {
		assert.Equal(t, "1.2.3", got[repo])
	}
	got[repoCore] = "changed"
	assert.Equal(t, "1.2.3", c.getAll(ctx)[repoCore])
	c.markReady()

	cancel()
	require.Eventually(t, func() bool {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return !c.running
	}, time.Second, time.Millisecond)
}

func TestVersionCacheCancellation(t *testing.T) {
	t.Run("BeforeRefresh", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		c := newVersionCacheWithFetch(func(ctx context.Context, repo string) (string, error) {
			t.Error("cancelled poll must not fetch")
			return "", nil
		})
		assert.False(t, c.refreshAll(ctx))
		c.waitReady(ctx)
	})

	t.Run("BetweenRepositories", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		calls := 0
		c := newVersionCacheWithFetch(func(ctx context.Context, repo string) (string, error) {
			calls++
			cancel()
			return "1.2.3", nil
		})
		assert.False(t, c.refreshAll(ctx))
		assert.Equal(t, 1, calls)
	})
}

func TestVersionCachePollInterval(t *testing.T) {
	c := newVersionCacheWithFetch(nil)
	for _, tt := range []struct {
		name string
		full bool
		min  time.Duration
	}{
		{name: "MissingVersions", min: versionPollFailureInterval},
		{name: "AllVersions", full: true, min: versionPollInterval},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.full {
				for _, repo := range versionRepos {
					c.entries[repo] = versionEntry{version: "1.2.3"}
				}
			}
			interval := c.nextInterval()
			assert.GreaterOrEqual(t, interval, tt.min)
			assert.Less(t, interval, tt.min+versionPollJitter)
		})
	}
}
