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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/syncprovider"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/pkg/errors"
)

const pageSize = 100
const maxResponseBytes = 64 << 20

type Provider struct {
	octeliumC octeliumc.ClientInterface
	dp        *enterprisev1.DirectoryProvider
	hc        *http.Client
	baseURL   string
	realm     string
	tokenSrc  *tokenSource
}

func NewProvider(ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	dp *enterprisev1.DirectoryProvider) (*Provider, error) {

	spec := dp.Spec.GetKeycloak()
	if spec == nil {
		return nil, errors.Errorf("DirectoryProvider is not of type Keycloak")
	}
	if spec.GetUrl() == "" || spec.GetRealm() == "" || spec.GetClientID() == "" {
		return nil, errors.Errorf("Keycloak requires url, realm and clientID")
	}

	clientSecret, err := syncprovider.GetSecretValue(ctx, octeliumC, spec.GetClientSecret().GetFromSecret())
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(spec.GetUrl(), "/")
	realm := spec.GetRealm()

	hc := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: spec.GetInsecureSkipVerify(),
			},
		},
	}

	return &Provider{
		octeliumC: octeliumC,
		dp:        dp,
		hc:        hc,
		baseURL:   baseURL,
		realm:     realm,
		tokenSrc: &tokenSource{
			hc:           hc,
			tokenURL:     fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", baseURL, realm),
			clientID:     spec.GetClientID(),
			clientSecret: clientSecret,
		},
	}, nil
}

func (p *Provider) Synchronize(ctx context.Context) error {
	return syncprovider.NewReconciler(p.octeliumC, p.dp).Sync(ctx, p)
}

func (p *Provider) ListUsers(ctx context.Context) ([]*syncprovider.User, error) {
	var ret []*syncprovider.User
	first := 0
	for {
		q := url.Values{}
		q.Set("first", strconv.Itoa(first))
		q.Set("max", strconv.Itoa(pageSize))

		var batch []kcUser
		if err := p.doGET(ctx, fmt.Sprintf("/admin/realms/%s/users", p.realm), q, &batch); err != nil {
			return nil, err
		}

		for i := range batch {
			u := batch[i]
			if u.ID == "" {
				continue
			}
			ret = append(ret, &syncprovider.User{
				ExternalID:  u.ID,
				Email:       u.Email,
				DisplayName: userDisplayName(u),
				FirstName:   u.FirstName,
				LastName:    u.LastName,
				IsDisabled:  !u.Enabled,
			})
		}

		if len(batch) < pageSize {
			break
		}
		first += len(batch)
	}
	return ret, nil
}

func (p *Provider) ListGroups(ctx context.Context) ([]*syncprovider.Group, error) {
	top, err := p.listTopGroups(ctx)
	if err != nil {
		return nil, err
	}

	var flat []kcGroup
	if err := p.flattenGroups(ctx, top, &flat); err != nil {
		return nil, err
	}

	var ret []*syncprovider.Group
	for _, g := range flat {
		if g.ID == "" {
			continue
		}
		members, err := p.listGroupMembers(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &syncprovider.Group{
			ExternalID:        g.ID,
			DisplayName:       groupDisplayName(g),
			MemberExternalIDs: members,
		})
	}
	return ret, nil
}

func (p *Provider) listTopGroups(ctx context.Context) ([]kcGroup, error) {
	var ret []kcGroup
	first := 0
	for {
		q := url.Values{}
		q.Set("first", strconv.Itoa(first))
		q.Set("max", strconv.Itoa(pageSize))
		q.Set("briefRepresentation", "false")

		var batch []kcGroup
		if err := p.doGET(ctx, fmt.Sprintf("/admin/realms/%s/groups", p.realm), q, &batch); err != nil {
			return nil, err
		}
		ret = append(ret, batch...)
		if len(batch) < pageSize {
			break
		}
		first += len(batch)
	}
	return ret, nil
}

func (p *Provider) listChildGroups(ctx context.Context, groupID string) ([]kcGroup, error) {
	var ret []kcGroup
	first := 0
	for {
		q := url.Values{}
		q.Set("first", strconv.Itoa(first))
		q.Set("max", strconv.Itoa(pageSize))
		q.Set("briefRepresentation", "false")

		var batch []kcGroup
		if err := p.doGET(ctx, fmt.Sprintf("/admin/realms/%s/groups/%s/children", p.realm, groupID), q, &batch); err != nil {
			return nil, err
		}
		ret = append(ret, batch...)
		if len(batch) < pageSize {
			break
		}
		first += len(batch)
	}
	return ret, nil
}

func (p *Provider) flattenGroups(ctx context.Context, groups []kcGroup, out *[]kcGroup) error {
	for _, g := range groups {
		if g.ID == "" {
			continue
		}
		*out = append(*out, g)

		children := g.SubGroups
		if len(children) == 0 && g.SubGroupCount > 0 {
			c, err := p.listChildGroups(ctx, g.ID)
			if err != nil {
				return err
			}
			children = c
		}
		if len(children) > 0 {
			if err := p.flattenGroups(ctx, children, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Provider) listGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	var ret []string
	first := 0
	for {
		q := url.Values{}
		q.Set("first", strconv.Itoa(first))
		q.Set("max", strconv.Itoa(pageSize))

		var batch []kcMember
		if err := p.doGET(ctx, fmt.Sprintf("/admin/realms/%s/groups/%s/members", p.realm, groupID), q, &batch); err != nil {
			return nil, err
		}
		for _, m := range batch {
			if m.ID != "" {
				ret = append(ret, m.ID)
			}
		}
		if len(batch) < pageSize {
			break
		}
		first += len(batch)
	}
	return ret, nil
}

func (p *Provider) doGET(ctx context.Context, path string, query url.Values, out any) error {
	token, err := p.tokenSrc.get(ctx)
	if err != nil {
		return err
	}

	u := fmt.Sprintf("%s%s", p.baseURL, path)
	if len(query) > 0 {
		u = fmt.Sprintf("%s?%s", u, query.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/json")

	resp, err := p.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("Keycloak GET %s returned %d: %s", path, resp.StatusCode, string(body))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

type tokenSource struct {
	hc           *http.Client
	tokenURL     string
	clientID     string
	clientSecret string

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func (t *tokenSource) get(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.token != "" && time.Now().Before(t.expiresAt.Add(-10*time.Second)) {
		return t.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", t.clientID)
	form.Set("client_secret", t.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := t.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Keycloak token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}
	if tr.AccessToken == "" {
		return "", errors.Errorf("Keycloak token endpoint returned an empty access token")
	}

	ttl := time.Duration(tr.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 60 * time.Second
	}

	t.token = tr.AccessToken
	t.expiresAt = time.Now().Add(ttl)

	return t.token, nil
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

func userDisplayName(u kcUser) string {
	if name := strings.TrimSpace(fmt.Sprintf("%s %s", u.FirstName, u.LastName)); name != "" {
		return name
	}
	if u.Username != "" {
		return u.Username
	}
	return u.Email
}

func groupDisplayName(g kcGroup) string {
	if g.Name != "" {
		return g.Name
	}
	return g.Path
}
