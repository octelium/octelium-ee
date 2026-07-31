// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package googleworkspace

import (
	"context"
	"crypto/tls"
	stderrors "errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/syncprovider"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	userMaxResults   = 500
	groupMaxResults  = 200
	memberMaxResults = 200

	requestTimeout        = 45 * time.Second
	dialTimeout           = 10 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 30 * time.Second
	idleConnTimeout       = 90 * time.Second
	expectContinueTimeout = time.Second

	retryCount       = 5
	retryInitialWait = 500 * time.Millisecond
	retryMaxWait     = 15 * time.Second

	maxResponseBytes       = 16 << 20
	maxPaginationPages     = 100000
	maxUsers               = 1000000
	maxGroups              = 100000
	maxGroupMembers        = 1000000
	groupMemberWorkerCount = 4

	defaultCustomer = "my_customer"
	userAgent       = "octelium-dirsync-googleworkspace"
)

var errResponseTooLarge = stderrors.New("Google Workspace response exceeds maximum size")

type Provider struct {
	octeliumC octeliumc.ClientInterface
	dp        *enterprisev1.DirectoryProvider
	srv       *admin.Service
	customer  string

	transport *http.Transport
	closeOnce sync.Once
}

func NewProvider(ctx context.Context, octeliumC octeliumc.ClientInterface, dp *enterprisev1.DirectoryProvider) (*Provider, error) {
	if octeliumC == nil {
		return nil, errors.Errorf("Nil Octelium client")
	}
	if dp == nil || dp.Spec == nil {
		return nil, errors.Errorf("Nil DirectoryProvider")
	}

	spec := dp.Spec.GetGoogleWorkspace()
	if spec == nil {
		return nil, errors.Errorf("DirectoryProvider is not of type GoogleWorkspace")
	}

	subject := strings.TrimSpace(spec.GetImpersonateSubject())
	if subject == "" {
		return nil, errors.Errorf("GoogleWorkspace requires impersonateSubject")
	}

	secretName := strings.TrimSpace(spec.GetServiceAccount().GetFromSecret())
	if secretName == "" {
		return nil, errors.Errorf("GoogleWorkspace requires serviceAccount")
	}

	serviceAccountJSON, err := syncprovider.GetSecretValue(
		ctx,
		octeliumC,
		secretName,
	)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(serviceAccountJSON) == "" {
		return nil, errors.Errorf("GoogleWorkspace service account Secret is empty")
	}

	cfg, err := google.JWTConfigFromJSON(
		[]byte(serviceAccountJSON),
		admin.AdminDirectoryUserReadonlyScope,
		admin.AdminDirectoryGroupReadonlyScope,
		admin.AdminDirectoryGroupMemberReadonlyScope,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Could not parse GoogleWorkspace service account")
	}
	cfg.Subject = subject

	transport := newTransport()
	limitedTransport := &responseLimitTransport{
		base:     transport,
		maxBytes: maxResponseBytes,
	}
	baseHTTPClient := &http.Client{
		Transport: limitedTransport,
		Timeout:   requestTimeout,
	}

	authCtx := context.WithValue(ctx, oauth2.HTTPClient, baseHTTPClient)
	tokenSource := oauth2.ReuseTokenSource(nil, cfg.TokenSource(authCtx))

	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Base:   limitedTransport,
			Source: tokenSource,
		},
		Timeout: requestTimeout,
	}

	srv, err := admin.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	srv.UserAgent = userAgent

	customer := strings.TrimSpace(spec.GetCustomer())
	if customer == "" {
		customer = defaultCustomer
	}

	return &Provider{
		octeliumC: octeliumC,
		dp:        dp,
		srv:       srv,
		customer:  customer,
		transport: transport,
	}, nil
}

func newTransport() *http.Transport {
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
		MinVersion: tls.VersionTLS12,
	}

	return transport
}

type responseLimitTransport struct {
	base     http.RoundTripper
	maxBytes int64
}

func (t *responseLimitTransport) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil && t.maxBytes > 0 {
		resp.Body = &responseLimitReadCloser{
			ReadCloser: resp.Body,
			remaining:  t.maxBytes,
		}
	}

	return resp, nil
}

type responseLimitReadCloser struct {
	io.ReadCloser
	remaining int64
	exceeded  bool
}

func (r *responseLimitReadCloser) Read(buf []byte) (int, error) {
	if r.exceeded {
		return 0, errResponseTooLarge
	}

	readBuf := buf
	if int64(len(readBuf)) > r.remaining+1 {
		readBuf = readBuf[:r.remaining+1]
	}

	n, err := r.ReadCloser.Read(readBuf)
	r.remaining -= int64(n)
	if r.remaining < 0 {
		r.exceeded = true
		return n, errResponseTooLarge
	}

	return n, err
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
		ret = append(ret, toUser(user))
	}

	return ret, nil
}

func (p *Provider) listUsers(ctx context.Context) ([]*admin.User, error) {
	ret := make([]*admin.User, 0)
	seenIDs := make(map[string]struct{})
	seenTokens := make(map[string]struct{})
	pageToken := ""

	for page := 0; page < maxPaginationPages; page++ {
		var resp *admin.Users
		err := p.doWithRetry(ctx, "list GoogleWorkspace Users", func() error {
			req := p.srv.Users.
				List().
				Context(ctx).
				Customer(p.customer).
				MaxResults(userMaxResults).
				OrderBy("email").
				Fields(googleapi.Field(
					"nextPageToken,users(id,primaryEmail,suspended,thumbnailPhotoUrl,name)",
				))

			if pageToken != "" {
				req = req.PageToken(pageToken)
			}

			var err error
			resp, err = req.Do()
			return err
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.Errorf("GoogleWorkspace Users list returned a nil response")
		}

		for _, user := range resp.Users {
			if user == nil || strings.TrimSpace(user.Id) == "" {
				return nil, errors.Errorf("GoogleWorkspace Users response contains a User with no ID")
			}
			if _, ok := seenIDs[user.Id]; ok {
				return nil, errors.Errorf(
					"GoogleWorkspace Users response contains duplicate User ID: %s",
					user.Id,
				)
			}

			seenIDs[user.Id] = struct{}{}
			ret = append(ret, user)
		}

		if len(ret) > maxUsers {
			return nil, errors.Errorf("GoogleWorkspace returned more than %d Users", maxUsers)
		}

		nextPageToken := strings.TrimSpace(resp.NextPageToken)
		if nextPageToken == "" {
			sort.Slice(ret, func(i, j int) bool {
				return ret[i].Id < ret[j].Id
			})
			return ret, nil
		}
		if nextPageToken == pageToken {
			return nil, errors.Errorf("GoogleWorkspace Users pagination did not advance")
		}
		if _, ok := seenTokens[nextPageToken]; ok {
			return nil, errors.Errorf("GoogleWorkspace Users pagination repeated a page token")
		}

		seenTokens[nextPageToken] = struct{}{}
		pageToken = nextPageToken
	}

	return nil, errors.Errorf(
		"GoogleWorkspace Users pagination exceeded %d pages",
		maxPaginationPages,
	)
}

func (p *Provider) ListGroups(ctx context.Context) ([]*syncprovider.Group, error) {
	groups, err := p.listGroups(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]*syncprovider.Group, len(groups))
	if len(groups) == 0 {
		return ret, nil
	}

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
				memberIDs, err := p.listGroupMembers(memberCtx, group.Id)
				if err != nil {
					errOnce.Do(func() {
						workerErr = err
						cancel()
					})
					return
				}

				ret[groupIdx] = &syncprovider.Group{
					ExternalID:        group.Id,
					DisplayName:       groupDisplayName(group),
					MemberExternalIDs: memberIDs,
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
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return ret, nil
}

func (p *Provider) listGroups(ctx context.Context) ([]*admin.Group, error) {
	ret := make([]*admin.Group, 0)
	seenIDs := make(map[string]struct{})
	seenTokens := make(map[string]struct{})
	pageToken := ""

	for page := 0; page < maxPaginationPages; page++ {
		var resp *admin.Groups
		err := p.doWithRetry(ctx, "list GoogleWorkspace Groups", func() error {
			req := p.srv.Groups.
				List().
				Context(ctx).
				Customer(p.customer).
				MaxResults(groupMaxResults).
				OrderBy("email").
				Fields(googleapi.Field("nextPageToken,groups(id,name,email)"))

			if pageToken != "" {
				req = req.PageToken(pageToken)
			}

			var err error
			resp, err = req.Do()
			return err
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.Errorf("GoogleWorkspace Groups list returned a nil response")
		}

		for _, group := range resp.Groups {
			if group == nil || strings.TrimSpace(group.Id) == "" {
				return nil, errors.Errorf("GoogleWorkspace Groups response contains a Group with no ID")
			}
			if _, ok := seenIDs[group.Id]; ok {
				return nil, errors.Errorf(
					"GoogleWorkspace Groups response contains duplicate Group ID: %s",
					group.Id,
				)
			}

			seenIDs[group.Id] = struct{}{}
			ret = append(ret, group)
		}

		if len(ret) > maxGroups {
			return nil, errors.Errorf("GoogleWorkspace returned more than %d Groups", maxGroups)
		}

		nextPageToken := strings.TrimSpace(resp.NextPageToken)
		if nextPageToken == "" {
			sort.Slice(ret, func(i, j int) bool {
				return ret[i].Id < ret[j].Id
			})
			return ret, nil
		}
		if nextPageToken == pageToken {
			return nil, errors.Errorf("GoogleWorkspace Groups pagination did not advance")
		}
		if _, ok := seenTokens[nextPageToken]; ok {
			return nil, errors.Errorf("GoogleWorkspace Groups pagination repeated a page token")
		}

		seenTokens[nextPageToken] = struct{}{}
		pageToken = nextPageToken
	}

	return nil, errors.Errorf(
		"GoogleWorkspace Groups pagination exceeded %d pages",
		maxPaginationPages,
	)
}

func (p *Provider) listGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	ret := make([]string, 0)
	seenIDs := make(map[string]struct{})
	seenTokens := make(map[string]struct{})
	pageToken := ""

	for page := 0; page < maxPaginationPages; page++ {
		var resp *admin.Members
		err := p.doWithRetry(
			ctx,
			fmt.Sprintf("list GoogleWorkspace Group %s members", groupID),
			func() error {
				req := p.srv.Members.
					List(groupID).
					Context(ctx).
					MaxResults(memberMaxResults).
					Fields(googleapi.Field("nextPageToken,members(id,type)"))

				if pageToken != "" {
					req = req.PageToken(pageToken)
				}

				var err error
				resp, err = req.Do()
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.Errorf(
				"GoogleWorkspace Group %s members list returned a nil response",
				groupID,
			)
		}

		for _, member := range resp.Members {
			if member == nil || strings.TrimSpace(member.Id) == "" {
				return nil, errors.Errorf(
					"GoogleWorkspace Group %s contains a member with no ID",
					groupID,
				)
			}
			if !strings.EqualFold(member.Type, "USER") {
				continue
			}
			if _, ok := seenIDs[member.Id]; ok {
				return nil, errors.Errorf(
					"GoogleWorkspace Group %s contains duplicate User member ID: %s",
					groupID,
					member.Id,
				)
			}

			seenIDs[member.Id] = struct{}{}
			ret = append(ret, member.Id)
		}

		if len(ret) > maxGroupMembers {
			return nil, errors.Errorf(
				"GoogleWorkspace Group %s has more than %d direct User members",
				groupID,
				maxGroupMembers,
			)
		}

		nextPageToken := strings.TrimSpace(resp.NextPageToken)
		if nextPageToken == "" {
			sort.Strings(ret)
			return ret, nil
		}
		if nextPageToken == pageToken {
			return nil, errors.Errorf(
				"GoogleWorkspace Group %s member pagination did not advance",
				groupID,
			)
		}
		if _, ok := seenTokens[nextPageToken]; ok {
			return nil, errors.Errorf(
				"GoogleWorkspace Group %s member pagination repeated a page token",
				groupID,
			)
		}

		seenTokens[nextPageToken] = struct{}{}
		pageToken = nextPageToken
	}

	return nil, errors.Errorf(
		"GoogleWorkspace Group %s member pagination exceeded %d pages",
		groupID,
		maxPaginationPages,
	)
}

func (p *Provider) doWithRetry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= retryCount; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !isRetryableGoogleError(lastErr) {
			return wrapGoogleError(operation, lastErr)
		}
		if attempt == retryCount {
			break
		}

		delay := googleRetryDelay(attempt, lastErr)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}

	return wrapGoogleError(operation, lastErr)
}

func isRetryableGoogleError(err error) bool {
	if err == nil ||
		stderrors.Is(err, context.Canceled) ||
		stderrors.Is(err, context.DeadlineExceeded) ||
		stderrors.Is(err, errResponseTooLarge) {
		return false
	}

	var apiErr *googleapi.Error
	if stderrors.As(err, &apiErr) {
		switch apiErr.Code {
		case http.StatusRequestTimeout,
			http.StatusTooEarly,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true

		case http.StatusForbidden:
			for _, item := range apiErr.Errors {
				switch item.Reason {
				case "rateLimitExceeded",
					"userRateLimitExceeded",
					"quotaExceeded",
					"backendError",
					"internalError":
					return true
				}
			}
		}

		return false
	}

	var netErr net.Error
	return stderrors.As(err, &netErr)
}

func googleRetryDelay(attempt int, err error) time.Duration {
	if retryAfter := googleRetryAfter(err); retryAfter > 0 {
		if retryAfter > retryMaxWait {
			return retryMaxWait
		}
		return retryAfter
	}

	delay := retryInitialWait
	for idx := 0; idx < attempt; idx++ {
		if delay >= retryMaxWait/2 {
			delay = retryMaxWait
			break
		}
		delay *= 2
	}

	if delay > retryMaxWait {
		delay = retryMaxWait
	}

	jitterLimit := delay / 4
	if jitterLimit <= 0 {
		return delay
	}

	return delay + time.Duration(rand.Int63n(int64(jitterLimit)))
}

func googleRetryAfter(err error) time.Duration {
	var apiErr *googleapi.Error
	if !stderrors.As(err, &apiErr) || apiErr.Header == nil {
		return 0
	}

	value := strings.TrimSpace(apiErr.Header.Get("Retry-After"))
	if value == "" {
		return 0
	}

	if seconds, parseErr := strconv.ParseInt(value, 10, 64); parseErr == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}

	retryAt, parseErr := http.ParseTime(value)
	if parseErr != nil {
		return 0
	}

	delay := time.Until(retryAt)
	if delay <= 0 {
		return 0
	}

	return delay
}

func wrapGoogleError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, context.Canceled) ||
		stderrors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if stderrors.Is(err, errResponseTooLarge) {
		return errors.Wrap(err, operation)
	}

	var apiErr *googleapi.Error
	if stderrors.As(err, &apiErr) {
		reasons := make([]string, 0, len(apiErr.Errors))
		for _, item := range apiErr.Errors {
			if item.Reason != "" {
				reasons = append(reasons, item.Reason)
			}
		}
		sort.Strings(reasons)

		message := strings.Join(strings.Fields(apiErr.Message), " ")
		if message == "" {
			message = http.StatusText(apiErr.Code)
		}

		if len(reasons) > 0 {
			return errors.Errorf(
				"%s failed with HTTP %d (%s): %s",
				operation,
				apiErr.Code,
				strings.Join(reasons, ","),
				message,
			)
		}

		return errors.Errorf(
			"%s failed with HTTP %d: %s",
			operation,
			apiErr.Code,
			message,
		)
	}

	return errors.Wrap(err, operation)
}

func toUser(user *admin.User) *syncprovider.User {
	ret := &syncprovider.User{
		ExternalID:  user.Id,
		Email:       user.PrimaryEmail,
		DisplayName: user.PrimaryEmail,
		IsDisabled:  user.Suspended,
		PicURL:      user.ThumbnailPhotoUrl,
	}

	if user.Name != nil {
		ret.FirstName = user.Name.GivenName
		ret.LastName = user.Name.FamilyName
		if fullName := strings.TrimSpace(user.Name.FullName); fullName != "" {
			ret.DisplayName = fullName
		} else if displayName := strings.TrimSpace(
			fmt.Sprintf("%s %s", user.Name.GivenName, user.Name.FamilyName),
		); displayName != "" {
			ret.DisplayName = displayName
		}
	}

	return ret
}

func groupDisplayName(group *admin.Group) string {
	if group.Name != "" {
		return group.Name
	}

	return group.Email
}
