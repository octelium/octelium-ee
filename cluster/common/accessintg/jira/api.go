// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const maxRespBytes = 4 << 20

type apiClient struct {
	baseURL string
	auth    string
	hc      *http.Client
}

func newAPIClient(baseURL, email, apiToken string) *apiClient {
	return &apiClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		auth: base64.StdEncoding.EncodeToString(
			[]byte(fmt.Sprintf("%s:%s", email, apiToken))),
		hc: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

type jiraUser struct {
	AccountID    string `json:"accountId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	Active       bool   `json:"active,omitempty"`
}

type jiraIssue struct {
	ID  string `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}

func (c *apiClient) do(ctx context.Context, method, path string,
	query url.Values, body any, out any) error {
	u := fmt.Sprintf("%s%s", c.baseURL, path)
	if len(query) > 0 {
		u = fmt.Sprintf("%s?%s", u, query.Encode())
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", c.auth))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.Errorf("Jira %s %s returned the status code %d",
			method, path, resp.StatusCode)
	}

	if out == nil || len(b) == 0 {
		return nil
	}

	if err := json.Unmarshal(b, out); err != nil {
		return errors.Errorf("Could not unmarshal the Jira %s response: %+v", path, err)
	}

	return nil
}

var errNotFound = errors.Errorf("Jira resource not found")

func (c *apiClient) myself(ctx context.Context) (*jiraUser, error) {
	ret := &jiraUser{}
	if err := c.do(ctx, http.MethodGet, "/rest/api/3/myself", nil, nil, ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *apiClient) getUser(ctx context.Context, accountID string) (*jiraUser, error) {
	ret := &jiraUser{}
	err := c.do(ctx, http.MethodGet, "/rest/api/3/user",
		url.Values{"accountId": []string{accountID}}, nil, ret)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return ret, nil
}

func (c *apiClient) searchUsers(ctx context.Context, query string) ([]*jiraUser, error) {
	ret := []*jiraUser{}
	err := c.do(ctx, http.MethodGet, "/rest/api/3/user/search",
		url.Values{"query": []string{query}, "maxResults": []string{"50"}}, nil, &ret)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return ret, nil
}

func (c *apiClient) createIssue(ctx context.Context, fields map[string]any) (*jiraIssue, error) {
	ret := &jiraIssue{}
	if err := c.do(ctx, http.MethodPost, "/rest/api/3/issue", nil, map[string]any{
		"fields": fields,
	}, ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *apiClient) updateIssue(ctx context.Context, key string, fields map[string]any) error {
	return c.do(ctx, http.MethodPut, fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(key)),
		nil, map[string]any{
			"fields": fields,
		}, nil)
}

func (c *apiClient) addComment(ctx context.Context, key string, body any) error {
	return c.do(ctx, http.MethodPost,
		fmt.Sprintf("/rest/api/3/issue/%s/comment", url.PathEscape(key)),
		nil, map[string]any{
			"body": body,
		}, nil)
}
