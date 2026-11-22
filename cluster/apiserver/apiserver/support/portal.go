// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package support

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
)

const (
	pathSupport      = "/api/cluster/v1/support"
	pathSupportToken = "/api/cluster/v1/support/token"
	pathTickets      = "/api/cluster/v1/support/tickets"
	pathTicketsList  = "/api/cluster/v1/support/tickets/list"

	tokenRefreshMargin = 2 * time.Minute
	maxResponseBytes   = 16 << 20
	maxErrorBytes      = 64 << 10
)

type credential struct {
	license         string
	clusterDomain   string
	installationKey string
}

func (c *credential) cacheKey() string {
	h := sha256.New()
	h.Write([]byte(c.license))
	h.Write([]byte{0})
	h.Write([]byte(c.clusterDomain))
	h.Write([]byte{0})
	h.Write([]byte(c.installationKey))
	return hex.EncodeToString(h.Sum(nil))
}

type accessToken struct {
	key       string
	value     string
	expiresAt time.Time
}

type portalError struct {
	status  int
	code    string
	message string
}

func (e *portalError) Error() string {
	return fmt.Sprintf("The customer portal returned HTTP %d (%s): %s", e.status, e.code, e.message)
}

type portalRequest struct {
	method         string
	path           string
	body           any
	raw            []byte
	headers        map[string]string
	idempotencyKey string
}

type portalClient struct {
	baseURL string
	httpC   *http.Client
	now     func() time.Time

	mu    sync.Mutex
	token *accessToken
	group singleflight.Group
}

func newPortalClient(baseURL string) *portalClient {
	return &portalClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpC: &http.Client{
			Transport: http.DefaultTransport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now: time.Now,
	}
}

func (c *portalClient) getToken(ctx context.Context, cred *credential) (*accessToken, error) {
	key := cred.cacheKey()

	c.mu.Lock()
	cur := c.token
	c.mu.Unlock()

	if cur != nil && cur.key == key && c.now().Add(tokenRefreshMargin).Before(cur.expiresAt) {
		return cur, nil
	}

	ch := c.group.DoChan(key, func() (any, error) {
		c.mu.Lock()
		cur := c.token
		c.mu.Unlock()
		if cur != nil && cur.key == key && c.now().Add(tokenRefreshMargin).Before(cur.expiresAt) {
			return cur, nil
		}

		tkn, err := c.exchange(context.WithoutCancel(ctx), cred)
		if err != nil {
			return nil, err
		}

		c.mu.Lock()
		c.token = tkn
		c.mu.Unlock()
		return tkn, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Val.(*accessToken), nil
	}
}

func (c *portalClient) invalidateToken(tkn *accessToken) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == tkn {
		c.token = nil
	}
}

func (c *portalClient) exchange(ctx context.Context, cred *credential) (*accessToken, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	resp, err := c.send(ctx, &portalRequest{
		method: http.MethodPost,
		path:   pathSupportToken,
		body: map[string]string{
			"license":         cred.license,
			"clusterDomain":   cred.clusterDomain,
			"installationKey": cred.installationKey,
		},
	}, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	out := &struct {
		AccessToken string `json:"accessToken"`
		ExpiresAt   string `json:"expiresAt"`
	}{}
	if err := decodeResponse(resp, out); err != nil {
		return nil, err
	}

	expiresAt, err := time.Parse(time.RFC3339Nano, out.ExpiresAt)
	if err != nil || out.AccessToken == "" {
		return nil, errors.Errorf("The customer portal returned an invalid support token")
	}

	return &accessToken{
		key:       cred.cacheKey(),
		value:     out.AccessToken,
		expiresAt: expiresAt,
	}, nil
}

func (c *portalClient) do(ctx context.Context, cred *credential, req *portalRequest, out any) error {
	resp, err := c.open(ctx, cred, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decodeResponse(resp, out)
}

func (c *portalClient) open(ctx context.Context, cred *credential, req *portalRequest) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		tkn, err := c.getToken(ctx, cred)
		if err != nil {
			return nil, err
		}

		resp, err := c.send(ctx, req, tkn.value)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			drain(resp)
			c.invalidateToken(tkn)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			defer drain(resp)
			return nil, readPortalError(resp)
		}

		return resp, nil
	}
}

func (c *portalClient) send(ctx context.Context, req *portalRequest, token string) (*http.Response, error) {
	u, err := url.Parse(c.baseURL + req.path)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	contentType := ""
	switch {
	case req.raw != nil:
		body = bytes.NewReader(req.raw)
		contentType = "application/octet-stream"
	case req.body != nil:
		payload, err := json.Marshal(req.body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(payload)
		contentType = "application/json"
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method, u.String(), body)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "octelium-enterprise-apiserver")
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}
	if req.idempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.idempotencyKey)
	}
	for k, v := range req.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.httpC.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if token == "" && (resp.StatusCode < 200 || resp.StatusCode > 299) {
		defer drain(resp)
		return nil, readPortalError(resp)
	}

	return resp, nil
}

func decodeResponse(resp *http.Response, out any) error {
	if out == nil {
		drain(resp)
		return nil
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxResponseBytes {
		return errors.Errorf("The customer portal response is too large")
	}

	if err := json.Unmarshal(data, out); err != nil {
		return errors.Errorf("The customer portal returned an invalid response")
	}

	return nil
}

func readPortalError(resp *http.Response) error {
	ret := &portalError{
		status: resp.StatusCode,
	}

	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
	out := &struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if err := json.Unmarshal(data, out); err == nil {
		ret.code = out.Error.Code
		ret.message = out.Error.Message
	}

	return ret
}

func drain(resp *http.Response) {
	io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBytes))
	resp.Body.Close()
}
