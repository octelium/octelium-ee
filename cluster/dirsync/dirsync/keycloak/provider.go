// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package keycloak

import (
	"context"
	"crypto/tls"
	stderrors "errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/syncprovider"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	pageSize = 100

	requestTimeout         = 30 * time.Second
	dialTimeout            = 10 * time.Second
	tlsHandshakeTimeout    = 10 * time.Second
	responseHeaderTimeout  = 30 * time.Second
	idleConnTimeout        = 90 * time.Second
	expectContinueTimeout  = time.Second
	retryCount             = 4
	retryWaitTime          = 500 * time.Millisecond
	retryMaxWaitTime       = 10 * time.Second
	maxResponseBytes       = 16 << 20
	maxTokenResponseBytes  = 1 << 20
	maxErrorResponseBytes  = 4 << 10
	maxPaginationPages     = 100000
	maxUsers               = 1000000
	maxGroups              = 100000
	maxGroupMembers        = 1000000
	maxGroupDepth          = 64
	groupMemberWorkerCount = 8
)

type Provider struct {
	octeliumC octeliumc.ClientInterface
	dp        *enterprisev1.DirectoryProvider

	client    *resty.Client
	transport *http.Transport
	baseURL   string
	realm     string
	tokenSrc  *tokenSource

	closeOnce sync.Once
}

func NewProvider(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	dp *enterprisev1.DirectoryProvider,
) (*Provider, error) {
	if octeliumC == nil {
		return nil, errors.Errorf("Nil Octelium client")
	}
	if dp == nil || dp.Spec == nil {
		return nil, errors.Errorf("Nil DirectoryProvider")
	}

	spec := dp.Spec.GetKeycloak()
	if spec == nil {
		return nil, errors.Errorf("DirectoryProvider is not of type Keycloak")
	}

	baseURL, err := normalizeBaseURL(spec.GetUrl())
	if err != nil {
		return nil, err
	}

	realm := strings.TrimSpace(spec.GetRealm())
	clientID := strings.TrimSpace(spec.GetClientID())
	if realm == "" {
		return nil, errors.Errorf("Keycloak requires realm")
	}
	if clientID == "" {
		return nil, errors.Errorf("Keycloak requires clientID")
	}

	clientSecret, err := syncprovider.GetSecretValue(
		ctx,
		octeliumC,
		spec.GetClientSecret().GetFromSecret(),
	)
	if err != nil {
		return nil, err
	}
	if clientSecret == "" {
		return nil, errors.Errorf("Keycloak client Secret is empty")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	if spec.GetInsecureSkipVerify() {
		zap.L().Warn("Keycloak TLS certificate verification is disabled",
			zap.String("directoryProvider", dp.GetMetadata().GetName()),
			zap.String("host", parsedURL.Host),
		)
	}

	transport := newTransport(spec.GetInsecureSkipVerify())
	client := newRestyClient(transport, parsedURL.Hostname())

	tokenSrc := &tokenSource{
		client:       client,
		tokenURL:     fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", baseURL, url.PathEscape(realm)),
		clientID:     clientID,
		clientSecret: clientSecret,
	}

	return &Provider{
		octeliumC: octeliumC,
		dp:        dp,
		client:    client,
		transport: transport,
		baseURL:   baseURL,
		realm:     realm,
		tokenSrc:  tokenSrc,
	}, nil
}

func normalizeBaseURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.Errorf("Keycloak requires url")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", errors.Wrap(err, "Could not parse Keycloak URL")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.Errorf("Keycloak URL scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", errors.Errorf("Keycloak URL has no host")
	}
	if parsed.User != nil {
		return "", errors.Errorf("Keycloak URL must not contain user information")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.Errorf("Keycloak URL must not contain a query or fragment")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func newTransport(insecureSkipVerify bool) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment
	transport.DialContext = (&net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
	}).DialContext
	transport.ForceAttemptHTTP2 = true
	transport.MaxIdleConns = 32
	transport.MaxIdleConnsPerHost = 16
	transport.MaxConnsPerHost = 32
	transport.IdleConnTimeout = idleConnTimeout
	transport.TLSHandshakeTimeout = tlsHandshakeTimeout
	transport.ResponseHeaderTimeout = responseHeaderTimeout
	transport.ExpectContinueTimeout = expectContinueTimeout
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecureSkipVerify,
	}

	return transport
}

func newRestyClient(transport *http.Transport, hostname string) *resty.Client {
	client := resty.New().
		SetTransport(transport).
		SetTimeout(requestTimeout).
		SetHeader("Accept", "application/json").
		SetResponseBodyLimit(maxResponseBytes).
		SetRetryCount(retryCount).
		SetRetryWaitTime(retryWaitTime).
		SetRetryMaxWaitTime(retryMaxWaitTime).
		SetRetryAfter(keycloakRetryAfter)

	if hostname != "" {
		client.SetRedirectPolicy(resty.DomainCheckRedirectPolicy(hostname))
	}

	client.AddRetryCondition(func(resp *resty.Response, err error) bool {
		if err != nil {
			if stderrors.Is(err, context.Canceled) ||
				stderrors.Is(err, context.DeadlineExceeded) {
				return false
			}
			if resp != nil &&
				resp.Request != nil &&
				resp.Request.Context() != nil &&
				resp.Request.Context().Err() != nil {
				return false
			}
			return true
		}
		if resp == nil {
			return false
		}

		switch resp.StatusCode() {
		case http.StatusRequestTimeout,
			http.StatusTooEarly,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	})

	return client
}

func keycloakRetryAfter(_ *resty.Client, resp *resty.Response) (time.Duration, error) {
	if resp == nil {
		return 0, nil
	}

	value := strings.TrimSpace(resp.Header().Get("Retry-After"))
	if value == "" {
		return 0, nil
	}

	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds <= 0 {
			return 0, nil
		}

		delay := time.Duration(seconds) * time.Second
		if delay > retryMaxWaitTime {
			delay = retryMaxWaitTime
		}
		return delay, nil
	}

	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0, nil
	}

	delay := time.Until(retryAt)
	if delay <= 0 {
		return 0, nil
	}
	if delay > retryMaxWaitTime {
		delay = retryMaxWaitTime
	}

	return delay, nil
}

func (p *Provider) Synchronize(ctx context.Context) error {
	defer p.Close()

	return syncprovider.NewReconciler(p.octeliumC, p.dp).Sync(ctx, p)
}

func (p *Provider) Close() {
	if p == nil {
		return
	}

	p.closeOnce.Do(func() {
		if p.transport != nil {
			p.transport.CloseIdleConnections()
		}
	})
}

func (p *Provider) ListUsers(ctx context.Context) ([]*syncprovider.User, error) {
	users, err := p.listUsers(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]*syncprovider.User, 0, len(users))
	for _, user := range users {
		ret = append(ret, &syncprovider.User{
			ExternalID:  user.ID,
			Email:       user.Email,
			DisplayName: userDisplayName(user),
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			IsDisabled:  !user.Enabled,
		})
	}

	return ret, nil
}

func (p *Provider) listUsers(ctx context.Context) ([]kcUser, error) {
	ret := make([]kcUser, 0)
	seen := make(map[string]struct{})
	first := 0

	for page := 0; page < maxPaginationPages; page++ {
		var batch []kcUser
		err := p.doGET(
			ctx,
			fmt.Sprintf("/admin/realms/%s/users", url.PathEscape(p.realm)),
			url.Values{
				"first":               []string{strconv.Itoa(first)},
				"max":                 []string{strconv.Itoa(pageSize)},
				"briefRepresentation": []string{"true"},
			},
			&batch,
		)
		if err != nil {
			return nil, err
		}

		if len(batch) == 0 {
			sort.Slice(ret, func(i, j int) bool {
				return ret[i].ID < ret[j].ID
			})
			return ret, nil
		}

		added := 0
		for _, user := range batch {
			if user.ID == "" {
				return nil, errors.Errorf("Keycloak users page at offset %d contains a User with no ID", first)
			}
			if _, ok := seen[user.ID]; ok {
				continue
			}

			seen[user.ID] = struct{}{}
			ret = append(ret, user)
			added++
		}

		if added == 0 {
			return nil, errors.Errorf("Keycloak users pagination did not advance at offset %d", first)
		}
		if len(ret) > maxUsers {
			return nil, errors.Errorf("Keycloak returned more than %d Users", maxUsers)
		}

		first += len(batch)
	}

	return nil, errors.Errorf("Keycloak users pagination exceeded %d pages", maxPaginationPages)
}

func (p *Provider) ListGroups(ctx context.Context) ([]*syncprovider.Group, error) {
	groups, err := p.listAllGroups(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]*syncprovider.Group, len(groups))

	memberCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan int)
	var wg sync.WaitGroup
	var errOnce sync.Once
	var workerErr error

	workerCount := groupMemberWorkerCount
	if len(groups) < workerCount {
		workerCount = len(groups)
	}

	for idx := 0; idx < workerCount; idx++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for groupIdx := range jobs {
				if memberCtx.Err() != nil {
					return
				}

				group := groups[groupIdx]
				members, err := p.listGroupMembers(memberCtx, group.ID)
				if err != nil {
					errOnce.Do(func() {
						workerErr = err
						cancel()
					})
					return
				}

				ret[groupIdx] = &syncprovider.Group{
					ExternalID:        group.ID,
					DisplayName:       groupDisplayName(group),
					MemberExternalIDs: members,
				}
			}
		}()
	}

sendLoop:
	for idx := range groups {
		select {
		case jobs <- idx:
		case <-memberCtx.Done():
			break sendLoop
		}
	}
	close(jobs)
	wg.Wait()

	if workerErr != nil {
		return nil, workerErr
	}
	if err := memberCtx.Err(); err != nil && !stderrors.Is(err, context.Canceled) {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return ret, nil
}

type groupQueueItem struct {
	group kcGroup
	depth int
}

func (p *Provider) listAllGroups(ctx context.Context) ([]kcGroup, error) {
	topGroups, err := p.listTopGroups(ctx)
	if err != nil {
		return nil, err
	}

	queue := make([]groupQueueItem, 0, len(topGroups))
	for _, group := range topGroups {
		queue = append(queue, groupQueueItem{
			group: group,
			depth: 0,
		})
	}

	ret := make([]kcGroup, 0, len(topGroups))
	seen := make(map[string]struct{})

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		group := item.group
		if group.ID == "" {
			return nil, errors.Errorf("Keycloak returned a Group with no ID")
		}
		if _, ok := seen[group.ID]; ok {
			return nil, errors.Errorf("Keycloak Group hierarchy contains duplicate or cyclic Group ID: %s", group.ID)
		}

		seen[group.ID] = struct{}{}
		ret = append(ret, group)

		if len(ret) > maxGroups {
			return nil, errors.Errorf("Keycloak returned more than %d Groups", maxGroups)
		}

		hasChildren := group.SubGroupCount > 0 || len(group.SubGroups) > 0
		if !hasChildren {
			continue
		}
		if item.depth >= maxGroupDepth {
			return nil, errors.Errorf("Keycloak Group hierarchy exceeds maximum depth %d at Group %s", maxGroupDepth, group.ID)
		}

		var children []kcGroup
		if group.SubGroupCount > 0 {
			children, err = p.listChildGroups(ctx, group.ID)
			if err != nil {
				return nil, err
			}
			if len(children) == 0 {
				return nil, errors.Errorf(
					"Keycloak Group %s reports child Groups but returned none",
					group.ID,
				)
			}
		} else {
			children = append(children, group.SubGroups...)
		}

		for _, child := range children {
			queue = append(queue, groupQueueItem{
				group: child,
				depth: item.depth + 1,
			})
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].ID < ret[j].ID
	})

	return ret, nil
}

func (p *Provider) listTopGroups(ctx context.Context) ([]kcGroup, error) {
	return p.listGroupsPageByPage(
		ctx,
		fmt.Sprintf("/admin/realms/%s/groups", url.PathEscape(p.realm)),
		url.Values{
			"briefRepresentation": []string{"false"},
			"populateHierarchy":   []string{"false"},
			"subGroupsCount":      []string{"true"},
		},
	)
}

func (p *Provider) listChildGroups(ctx context.Context, groupID string) ([]kcGroup, error) {
	return p.listGroupsPageByPage(
		ctx,
		fmt.Sprintf(
			"/admin/realms/%s/groups/%s/children",
			url.PathEscape(p.realm),
			url.PathEscape(groupID),
		),
		url.Values{
			"briefRepresentation": []string{"false"},
			"subGroupsCount":      []string{"true"},
		},
	)
}

func (p *Provider) listGroupsPageByPage(ctx context.Context, path string, baseQuery url.Values) ([]kcGroup, error) {
	ret := make([]kcGroup, 0)
	seen := make(map[string]struct{})
	first := 0

	for page := 0; page < maxPaginationPages; page++ {
		query := cloneValues(baseQuery)
		query.Set("first", strconv.Itoa(first))
		query.Set("max", strconv.Itoa(pageSize))

		var batch []kcGroup
		if err := p.doGET(ctx, path, query, &batch); err != nil {
			return nil, err
		}

		if len(batch) == 0 {
			return ret, nil
		}

		added := 0
		for _, group := range batch {
			if group.ID == "" {
				return nil, errors.Errorf(
					"Keycloak Groups page at offset %d contains a Group with no ID",
					first,
				)
			}
			if _, ok := seen[group.ID]; ok {
				continue
			}

			seen[group.ID] = struct{}{}
			ret = append(ret, group)
			added++
		}

		if added == 0 {
			return nil, errors.Errorf(
				"Keycloak Groups pagination did not advance at offset %d",
				first,
			)
		}
		if len(ret) > maxGroups {
			return nil, errors.Errorf(
				"Keycloak returned more than %d Groups in one hierarchy level",
				maxGroups,
			)
		}

		first += len(batch)
	}

	return nil, errors.Errorf(
		"Keycloak Groups pagination exceeded %d pages",
		maxPaginationPages,
	)
}

func (p *Provider) listGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	ret := make([]string, 0)
	seen := make(map[string]struct{})
	first := 0
	path := fmt.Sprintf(
		"/admin/realms/%s/groups/%s/members",
		url.PathEscape(p.realm),
		url.PathEscape(groupID),
	)

	for page := 0; page < maxPaginationPages; page++ {
		var batch []kcMember
		if err := p.doGET(ctx, path, url.Values{
			"first":               []string{strconv.Itoa(first)},
			"max":                 []string{strconv.Itoa(pageSize)},
			"briefRepresentation": []string{"true"},
		}, &batch); err != nil {
			return nil, err
		}

		if len(batch) == 0 {
			sort.Strings(ret)
			return ret, nil
		}

		added := 0
		for _, member := range batch {
			if member.ID == "" {
				return nil, errors.Errorf(
					"Keycloak Group %s members page at offset %d contains a User with no ID",
					groupID,
					first,
				)
			}
			if _, ok := seen[member.ID]; ok {
				continue
			}

			seen[member.ID] = struct{}{}
			ret = append(ret, member.ID)
			added++
		}

		if added == 0 {
			return nil, errors.Errorf(
				"Keycloak Group %s member pagination did not advance at offset %d",
				groupID,
				first,
			)
		}
		if len(ret) > maxGroupMembers {
			return nil, errors.Errorf(
				"Keycloak Group %s has more than %d members",
				groupID,
				maxGroupMembers,
			)
		}

		first += len(batch)
	}

	return nil, errors.Errorf(
		"Keycloak Group %s member pagination exceeded %d pages",
		groupID,
		maxPaginationPages,
	)
}

func cloneValues(in url.Values) url.Values {
	if in == nil {
		return url.Values{}
	}

	ret := make(url.Values, len(in))
	for key, values := range in {
		ret[key] = append([]string(nil), values...)
	}

	return ret
}

func (p *Provider) doGET(ctx context.Context, path string, query url.Values, out any) error {
	token, err := p.tokenSrc.get(ctx)
	if err != nil {
		return err
	}

	resp, err := p.executeGET(ctx, path, query, token, out)
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		p.tokenSrc.invalidate(token)

		token, err = p.tokenSrc.get(ctx)
		if err != nil {
			return err
		}

		resp, err = p.executeGET(ctx, path, query, token, out)
		if err != nil {
			return err
		}
	}

	if !resp.IsSuccess() {
		return keycloakResponseError(http.MethodGet, path, resp)
	}

	return nil
}

func (p *Provider) executeGET(ctx context.Context, path string, query url.Values, token string, out any) (*resty.Response, error) {
	req := p.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetQueryParamsFromValues(query).
		SetResponseBodyLimit(maxResponseBytes)

	if out != nil {
		req.SetResult(out)
	}

	resp, err := req.Get(p.baseURL + path)
	if err != nil {
		return resp, errors.Wrapf(err, "Could not execute Keycloak GET %s", path)
	}

	return resp, nil
}

func keycloakResponseError(method string, path string, resp *resty.Response) error {
	if resp == nil {
		return errors.Errorf("Keycloak %s %s returned no response", method, path)
	}

	detail := responseBodySnippet(resp.Body())
	if detail == "" {
		return errors.Errorf(
			"Keycloak %s %s returned HTTP %d",
			method,
			path,
			resp.StatusCode(),
		)
	}

	return errors.Errorf(
		"Keycloak %s %s returned HTTP %d: %s",
		method,
		path,
		resp.StatusCode(),
		detail,
	)
}

func responseBodySnippet(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	truncated := len(body) > maxErrorResponseBytes
	if truncated {
		body = body[:maxErrorResponseBytes]
	}

	value := strings.Join(strings.Fields(string(body)), " ")
	if truncated {
		value += "..."
	}

	return value
}

type tokenSource struct {
	client       *resty.Client
	tokenURL     string
	clientID     string
	clientSecret string

	mu        sync.RWMutex
	refreshMu sync.Mutex
	token     string
	expiresAt time.Time
}

func (t *tokenSource) get(ctx context.Context) (string, error) {
	if token := t.cachedToken(); token != "" {
		return token, nil
	}

	t.refreshMu.Lock()
	defer t.refreshMu.Unlock()

	if token := t.cachedToken(); token != "" {
		return token, nil
	}

	return t.refresh(ctx)
}

func (t *tokenSource) cachedToken() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.token == "" || !time.Now().Before(t.expiresAt) {
		return ""
	}

	return t.token
}

func (t *tokenSource) refresh(ctx context.Context) (string, error) {
	result := &tokenResponse{}

	resp, err := t.client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetFormData(map[string]string{
			"grant_type":    "client_credentials",
			"client_id":     t.clientID,
			"client_secret": t.clientSecret,
		}).
		SetResult(result).
		SetResponseBodyLimit(maxTokenResponseBytes).
		Post(t.tokenURL)
	if err != nil {
		return "", errors.Wrap(err, "Could not request Keycloak access token")
	}
	if !resp.IsSuccess() {
		return "", keycloakResponseError(http.MethodPost, "token endpoint", resp)
	}
	if result.AccessToken == "" {
		return "", errors.Errorf("Keycloak token endpoint returned an empty access token")
	}

	ttl := time.Duration(result.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 60 * time.Second
	}

	skew := ttl / 10
	if skew < time.Second {
		skew = time.Second
	}
	if skew > 30*time.Second {
		skew = 30 * time.Second
	}
	if skew >= ttl {
		skew = ttl / 2
	}

	t.mu.Lock()
	t.token = result.AccessToken
	t.expiresAt = time.Now().Add(ttl - skew)
	t.mu.Unlock()

	return result.AccessToken, nil
}

func (t *tokenSource) invalidate(token string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if token != "" && t.token != token {
		return
	}

	t.token = ""
	t.expiresAt = time.Time{}
}

type tokenResponse struct {
	AccessToken string `json:"access_token,omitempty"`
	ExpiresIn   int64  `json:"expires_in,omitempty"`
}

type kcUser struct {
	ID        string `json:"id,omitempty"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Enabled   bool   `json:"enabled,omitempty"`
}

type kcGroup struct {
	ID            string    `json:"id,omitempty"`
	Name          string    `json:"name,omitempty"`
	Path          string    `json:"path,omitempty"`
	SubGroupCount int       `json:"subGroupCount,omitempty"`
	SubGroups     []kcGroup `json:"subGroups,omitempty"`
}

type kcMember struct {
	ID string `json:"id,omitempty"`
}

func userDisplayName(user kcUser) string {
	if name := strings.TrimSpace(
		fmt.Sprintf("%s %s", user.FirstName, user.LastName),
	); name != "" {
		return name
	}
	if user.Username != "" {
		return user.Username
	}

	return user.Email
}

func groupDisplayName(group kcGroup) string {
	if group.Name != "" {
		return group.Name
	}

	return group.Path
}
