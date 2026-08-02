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
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-version"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	repoCore              = "octelium"
	repoPackageEnterprise = "octelium-ee"
	repoPackageCordium    = "cordium"

	versionInfoKeyCore              = "octelium"
	versionInfoKeyPackageEnterprise = "octeliumee"
	versionInfoKeyPackageCordium    = "cordium"
)

const (
	versionPollInterval        = time.Hour
	versionPollJitter          = 10 * time.Minute
	versionPollFailureInterval = 5 * time.Minute
	versionPollRepoGap         = 250 * time.Millisecond
	versionInitialWait         = 3 * time.Second
	versionRequestTimeout      = 5 * time.Second
	versionStaleAfter          = 24 * time.Hour
	versionMaxRateLimitDelay   = 24 * time.Hour
	versionMaxResponseBytes    = 1 << 20
	versionMaxLen              = 128

	githubAPIBaseURL = "https://api.github.com"
	githubAPIVersion = "2022-11-28"
	githubRawBaseURL = "https://raw.githubusercontent.com"
)

var versionRepos = []string{
	repoCore,
	repoPackageEnterprise,
	repoPackageCordium,
}

var defaultVersionFetcher = newVersionFetcher()

var getLatestVersionFunc = getLatestVersion

var defaultVersionCache = newVersionCacheWithFetch(func(ctx context.Context, repo string) (string, error) {
	return getLatestVersionFunc(ctx, repo)
})

func StartVersionPoller(ctx context.Context) {
	defaultVersionCache.start(ctx)
}

func (s *Server) GetClusterInfo(ctx context.Context, req *enterprisev1.GetClusterInfoRequest) (*enterprisev1.GetClusterInfoResponse, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	ret := &enterprisev1.GetClusterInfoResponse{}

	rgn, err := s.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	if err != nil {
		return nil, err
	}

	if rgn == nil || rgn.Status == nil {
		return ret, nil
	}

	latest := defaultVersionCache.getAll(ctx)

	latestCore := latest[repoCore]
	latestEE := latest[repoPackageEnterprise]
	latestCordium := latest[repoPackageCordium]

	if info, ok := rgn.Status.VersionInfoMap[versionInfoKeyCore]; ok && info != nil {
		ret.Core = &enterprisev1.GetClusterInfoResponse_Core{
			CurrentVersion: info.Version,
			SetAt:          info.SetAt,
			LatestVersion:  latestCore,
			CanUpgrade:     canUpgradeVersion(info.Version, latestCore),
		}
	}

	if info, ok := rgn.Status.VersionInfoMap[versionInfoKeyPackageEnterprise]; ok && info != nil {
		ret.PackageEnterprise = &enterprisev1.GetClusterInfoResponse_PackageEnterprise{
			CurrentVersion: info.Version,
			SetAt:          info.SetAt,
			LatestVersion:  latestEE,
			CanUpgrade:     canUpgradeVersion(info.Version, latestEE),
		}
	}

	if info, ok := rgn.Status.VersionInfoMap[versionInfoKeyPackageCordium]; ok && info != nil {
		ret.PackageCordium = &enterprisev1.GetClusterInfoResponse_PackageCordium{
			CurrentVersion: info.Version,
			SetAt:          info.SetAt,
			LatestVersion:  latestCordium,
			CanUpgrade:     canUpgradeVersion(info.Version, latestCordium),
		}
	}

	return ret, nil
}

func canUpgradeVersion(cur string, latest string) bool {
	cur = strings.TrimSpace(cur)
	latest = strings.TrimSpace(latest)

	if cur == "" || latest == "" {
		return false
	}

	curSemver, err := version.NewSemver(cur)
	if err != nil {
		return false
	}

	latestSemver, err := version.NewSemver(latest)
	if err != nil {
		return false
	}

	return latestSemver.GreaterThan(curSemver)
}

type versionFetchFunc func(ctx context.Context, repo string) (string, error)

type fetcherEntry struct {
	etag    string
	version string
}

type versionFetcher struct {
	mu               sync.Mutex
	entries          map[string]fetcherEntry
	rateLimitedUntil time.Time

	client *resty.Client
}

func newVersionFetcher() *versionFetcher {
	return &versionFetcher{
		entries: make(map[string]fetcherEntry, len(versionRepos)),
		client: resty.New().
			SetDebug(ldflags.IsDev()).
			SetTimeout(versionRequestTimeout).
			SetResponseBodyLimit(versionMaxResponseBytes),
	}
}

func (f *versionFetcher) latestVersion(ctx context.Context, repo string) (string, error) {
	apiCtx, apiCancel := context.WithTimeout(ctx, versionRequestTimeout)
	ret, err := f.fromReleases(apiCtx, repo)
	apiCancel()

	if err == nil {
		return ret, nil
	}

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	zap.L().Debug("Could not get the latest release from the GitHub API. Falling back to the raw release file",
		zap.String("repo", repo),
		zap.Error(err))

	rawCtx, rawCancel := context.WithTimeout(ctx, versionRequestTimeout)
	defer rawCancel()

	return f.fromRawFile(rawCtx, repo)
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

func (f *versionFetcher) fromReleases(ctx context.Context, repo string) (string, error) {
	if f.isRateLimited() {
		return "", errors.Errorf("The GitHub API rate limit is currently exhausted")
	}

	release := &githubRelease{}

	req := f.client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/vnd.github+json").
		SetHeader("X-GitHub-Api-Version", githubAPIVersion).
		SetResult(release)

	etag, cached := f.getEntry(repo)
	if etag != "" {
		req = req.SetHeader("If-None-Match", etag)
	}

	resp, err := req.Get(fmt.Sprintf("%s/repos/octelium/%s/releases/latest", githubAPIBaseURL, repo))
	if err != nil {
		return "", errors.Wrapf(err, "Could not get the latest release for the repo: %s", repo)
	}

	f.trackRateLimit(resp)

	switch resp.StatusCode() {
	case http.StatusOK:

	case http.StatusNotModified:
		if cached != "" {
			return cached, nil
		}
		return "", errors.Errorf("The GitHub API returned 304 without a resolved version for the repo: %s", repo)

	default:
		return "", errors.Errorf("The GitHub API returned HTTP %d for the repo: %s", resp.StatusCode(), repo)
	}

	if release.Draft || release.Prerelease {
		return "", errors.Errorf("The latest GitHub release is a draft or a prerelease for the repo: %s", repo)
	}

	ret := sanitizeVersion(release.TagName)
	if ret == "" {
		return "", errors.Errorf("The latest GitHub release has no usable tag for the repo: %s", repo)
	}

	f.setEntry(repo, strings.TrimSpace(resp.Header().Get("ETag")), ret)

	return ret, nil
}

func (f *versionFetcher) fromRawFile(ctx context.Context, pkg string) (string, error) {
	releaseURL := fmt.Sprintf("%s/octelium/%s/refs/heads/main/unsorted/latest_release", githubRawBaseURL, pkg)

	resp, err := f.client.R().
		SetContext(ctx).
		Get(releaseURL)
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", errors.Errorf("Could not get latest version release for package: %s", pkg)
	}

	return strings.TrimSpace(string(resp.Body())), nil
}

func (f *versionFetcher) getEntry(repo string) (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entry := f.entries[repo]
	return entry.etag, entry.version
}

func (f *versionFetcher) setEntry(repo, etag, resolved string) {
	if etag == "" || resolved == "" {
		return
	}

	f.mu.Lock()
	f.entries[repo] = fetcherEntry{
		etag:    etag,
		version: resolved,
	}
	f.mu.Unlock()
}

func (f *versionFetcher) isRateLimited() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return time.Now().Before(f.rateLimitedUntil)
}

func (f *versionFetcher) setRateLimitedUntil(until time.Time) {
	f.mu.Lock()
	if until.After(f.rateLimitedUntil) {
		f.rateLimitedUntil = until
	}
	f.mu.Unlock()
}

func (f *versionFetcher) trackRateLimit(resp *resty.Response) {
	if resp == nil {
		return
	}

	switch resp.StatusCode() {
	case http.StatusForbidden, http.StatusTooManyRequests:
	default:
		return
	}

	if delay := parseRetryAfter(resp.Header().Get("Retry-After")); delay > 0 {
		zap.L().Debug("The GitHub API asked us to back off", zap.Duration("delay", delay))
		f.setRateLimitedUntil(time.Now().Add(delay))
		return
	}

	if strings.TrimSpace(resp.Header().Get("X-RateLimit-Remaining")) != "0" {
		return
	}

	until := time.Now().Add(versionPollInterval)

	if reset := strings.TrimSpace(resp.Header().Get("X-RateLimit-Reset")); reset != "" {
		if seconds, err := strconv.ParseInt(reset, 10, 64); err == nil {
			if resetAt := time.Unix(seconds, 0); time.Until(resetAt) <= versionMaxRateLimitDelay {
				until = resetAt
			}
		}
	}

	zap.L().Debug("The GitHub API rate limit is exhausted", zap.Time("until", until))

	f.setRateLimitedUntil(until)
}

func getLatestVersion(ctx context.Context, pkg string) (string, error) {
	return defaultVersionFetcher.latestVersion(ctx, pkg)
}

type versionEntry struct {
	version       string
	fetchedAt     time.Time
	lastAttemptAt time.Time
	failures      uint32
}

type versionCache struct {
	mu      sync.RWMutex
	entries map[string]versionEntry
	running bool

	ready     chan struct{}
	readyOnce sync.Once

	fetch versionFetchFunc
}

func newVersionCache() *versionCache {
	return newVersionCacheWithFetch(newVersionFetcher().latestVersion)
}

func newVersionCacheWithFetch(fetch versionFetchFunc) *versionCache {
	return &versionCache{
		entries: make(map[string]versionEntry, len(versionRepos)),
		ready:   make(chan struct{}),
		fetch:   fetch,
	}
}

func (c *versionCache) start(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	go func() {
		defer func() {
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
		}()

		c.run(ctx)
	}()
}

func (c *versionCache) run(ctx context.Context) {
	for {
		if c.refreshAll(ctx) {
			c.markReady()
		}

		timer := time.NewTimer(c.nextInterval())

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (c *versionCache) nextInterval() time.Duration {
	interval := versionPollInterval

	c.mu.RLock()
	for _, repo := range versionRepos {
		if c.entries[repo].version == "" {
			interval = versionPollFailureInterval
			break
		}
	}
	c.mu.RUnlock()

	return interval + time.Duration(rand.Int63n(int64(versionPollJitter)))
}

func (c *versionCache) refreshAll(ctx context.Context) bool {
	for idx, repo := range versionRepos {
		if ctx.Err() != nil {
			return false
		}

		if idx > 0 {
			timer := time.NewTimer(versionPollRepoGap)

			select {
			case <-ctx.Done():
				timer.Stop()
				return false
			case <-timer.C:
			}
		}

		c.refresh(ctx, repo)
	}

	return ctx.Err() == nil
}

func (c *versionCache) refresh(ctx context.Context, repo string) {
	latest, err := c.fetch(ctx, repo)
	if err == nil {
		latest = sanitizeVersion(latest)
		if latest == "" {
			err = errors.Errorf("The resolved version is not usable")
		}
	}

	if err != nil {
		c.recordFailure(repo, err)
		return
	}

	now := time.Now()

	c.mu.Lock()
	c.entries[repo] = versionEntry{
		version:       latest,
		fetchedAt:     now,
		lastAttemptAt: now,
	}
	c.mu.Unlock()
}

func (c *versionCache) recordFailure(repo string, cause error) {
	now := time.Now()

	c.mu.Lock()
	entry := c.entries[repo]
	entry.lastAttemptAt = now
	entry.failures++
	c.entries[repo] = entry
	failures := entry.failures
	fetchedAt := entry.fetchedAt
	cached := entry.version
	c.mu.Unlock()

	if cached != "" && !fetchedAt.IsZero() && now.Sub(fetchedAt) <= versionStaleAfter {
		zap.L().Debug("Could not refresh the latest version",
			zap.String("repo", repo),
			zap.Uint32("failures", failures),
			zap.Error(cause))
		return
	}

	zap.L().Warn("Could not resolve a fresh latest version",
		zap.String("repo", repo),
		zap.String("cachedVersion", cached),
		zap.Time("fetchedAt", fetchedAt),
		zap.Uint32("failures", failures),
		zap.Error(cause))
}

func (c *versionCache) markReady() {
	c.readyOnce.Do(func() {
		close(c.ready)
	})
}

func (c *versionCache) getAll(ctx context.Context) map[string]string {
	c.start(context.WithoutCancel(ctx))
	c.waitReady(ctx)

	c.mu.RLock()
	defer c.mu.RUnlock()

	ret := make(map[string]string, len(versionRepos))
	for _, repo := range versionRepos {
		ret[repo] = c.entries[repo].version
	}

	return ret
}

func (c *versionCache) waitReady(ctx context.Context) {
	select {
	case <-c.ready:
		return
	default:
	}

	timer := time.NewTimer(versionInitialWait)
	defer timer.Stop()

	select {
	case <-c.ready:
	case <-timer.C:
	case <-ctx.Done():
	}
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds <= 0 {
			return 0
		}

		delay := time.Duration(seconds) * time.Second
		if delay > versionMaxRateLimitDelay {
			return versionMaxRateLimitDelay
		}
		return delay
	}

	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0
	}

	delay := time.Until(retryAt)
	if delay <= 0 {
		return 0
	}
	if delay > versionMaxRateLimitDelay {
		return versionMaxRateLimitDelay
	}

	return delay
}

func sanitizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	if idx := strings.IndexAny(v, "\r\n"); idx >= 0 {
		v = strings.TrimSpace(v[:idx])
	}

	if len(v) > versionMaxLen {
		return ""
	}

	for i := 0; i < len(v); i++ {
		if v[i] < 0x21 || v[i] > 0x7e {
			return ""
		}
	}

	if len(v) > 1 && (v[0] == 'v' || v[0] == 'V') {
		if _, err := version.NewSemver(v[1:]); err == nil {
			return v[1:]
		}
	}

	return v
}
