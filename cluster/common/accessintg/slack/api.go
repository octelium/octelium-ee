// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const defaultBaseURL = "https://slack.com/api"

const maxRespBytes = 4 << 20

type apiClient struct {
	baseURL  string
	botToken string
	hc       *http.Client
}

func newAPIClient(baseURL, botToken string) *apiClient {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &apiClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		botToken: botToken,
		hc: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type apiResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type authTestResponse struct {
	apiResponse
	TeamID string `json:"team_id,omitempty"`
	Team   string `json:"team,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

type userProfile struct {
	Email    string `json:"email,omitempty"`
	RealName string `json:"real_name,omitempty"`
}

type user struct {
	ID      string       `json:"id,omitempty"`
	Name    string       `json:"name,omitempty"`
	Deleted bool         `json:"deleted,omitempty"`
	Profile *userProfile `json:"profile,omitempty"`
}

type userResponse struct {
	apiResponse
	User *user `json:"user,omitempty"`
}

type conversation struct {
	ID string `json:"id,omitempty"`
}

type conversationsOpenResponse struct {
	apiResponse
	Channel *conversation `json:"channel,omitempty"`
}

type chatResponse struct {
	apiResponse
	Channel string `json:"channel,omitempty"`
	TS      string `json:"ts,omitempty"`
}

type permalinkResponse struct {
	apiResponse
	Permalink string `json:"permalink,omitempty"`
}

func (c *apiClient) get(ctx context.Context, method string,
	params url.Values, out any) error {
	u := fmt.Sprintf("%s/%s", c.baseURL, method)
	if len(params) > 0 {
		u = fmt.Sprintf("%s?%s", u, params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	return c.do(req, method, out)
}

func (c *apiClient) post(ctx context.Context, method string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s", c.baseURL, method), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	return c.do(req, method, out)
}

func (c *apiClient) do(req *http.Request, method string, out any) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.botToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return errors.Errorf("Slack %s is rate limited", method)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("Slack %s returned the status code %d", method, resp.StatusCode)
	}

	if err := json.Unmarshal(b, out); err != nil {
		return errors.Errorf("Could not unmarshal the Slack %s response: %+v", method, err)
	}

	return nil
}

func (c *apiClient) authTest(ctx context.Context) (*authTestResponse, error) {
	ret := &authTestResponse{}
	if err := c.post(ctx, "auth.test", map[string]any{}, ret); err != nil {
		return nil, err
	}
	if !ret.OK {
		return nil, errors.Errorf("Slack auth.test failed: %s", ret.Error)
	}

	return ret, nil
}

func (c *apiClient) usersInfo(ctx context.Context, externalID string) (*user, error) {
	ret := &userResponse{}
	if err := c.get(ctx, "users.info", url.Values{"user": []string{externalID}}, ret); err != nil {
		return nil, err
	}
	if !ret.OK {
		if isNotFoundError(ret.Error) {
			return nil, nil
		}
		return nil, errors.Errorf("Slack users.info failed: %s", ret.Error)
	}

	return ret.User, nil
}

func (c *apiClient) usersLookupByEmail(ctx context.Context, email string) (*user, error) {
	ret := &userResponse{}
	if err := c.get(ctx, "users.lookupByEmail",
		url.Values{"email": []string{email}}, ret); err != nil {
		return nil, err
	}
	if !ret.OK {
		if isNotFoundError(ret.Error) {
			return nil, nil
		}
		return nil, errors.Errorf("Slack users.lookupByEmail failed: %s", ret.Error)
	}

	return ret.User, nil
}

func (c *apiClient) conversationsOpen(ctx context.Context, externalID string) (string, error) {
	ret := &conversationsOpenResponse{}
	if err := c.post(ctx, "conversations.open", map[string]any{
		"users": externalID,
	}, ret); err != nil {
		return "", err
	}
	if !ret.OK {
		return "", errors.Errorf("Slack conversations.open failed: %s", ret.Error)
	}
	if ret.Channel == nil || ret.Channel.ID == "" {
		return "", errors.Errorf("Slack conversations.open returned no channel")
	}

	return ret.Channel.ID, nil
}

func (c *apiClient) chatPostMessage(ctx context.Context,
	channel, text string, blocks []any) (*chatResponse, error) {
	ret := &chatResponse{}
	if err := c.post(ctx, "chat.postMessage", map[string]any{
		"channel": channel,
		"text":    text,
		"blocks":  blocks,
	}, ret); err != nil {
		return nil, err
	}
	if !ret.OK {
		return nil, errors.Errorf("Slack chat.postMessage failed: %s", ret.Error)
	}

	return ret, nil
}

func (c *apiClient) chatUpdate(ctx context.Context,
	channel, ts, text string, blocks []any) (*chatResponse, error) {
	ret := &chatResponse{}
	if err := c.post(ctx, "chat.update", map[string]any{
		"channel": channel,
		"ts":      ts,
		"text":    text,
		"blocks":  blocks,
	}, ret); err != nil {
		return nil, err
	}
	if !ret.OK {
		return nil, errors.Errorf("Slack chat.update failed: %s", ret.Error)
	}

	return ret, nil
}

func (c *apiClient) chatGetPermalink(ctx context.Context, channel, ts string) string {
	ret := &permalinkResponse{}
	if err := c.get(ctx, "chat.getPermalink", url.Values{
		"channel":    []string{channel},
		"message_ts": []string{ts},
	}, ret); err != nil {
		return ""
	}
	if !ret.OK {
		return ""
	}

	return ret.Permalink
}

func isNotFoundError(err string) bool {
	switch err {
	case "users_not_found", "user_not_found", "users_not_visible":
		return true
	default:
		return false
	}
}
