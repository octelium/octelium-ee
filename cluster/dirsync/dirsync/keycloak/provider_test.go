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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newKeycloakHTTPTestProvider(t *testing.T, handle http.HandlerFunc) (*Provider, *atomic.Int32) {
	t.Helper()
	calls := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
			assert.Equal(t, "secret", r.Form.Get("client_secret"))
			json.NewEncoder(w).Encode(map[string]any{"access_token": fmt.Sprintf("token-%d", calls.Add(1)), "expires_in": 3600})
			return
		}
		handle(w, r)
	}))
	t.Cleanup(srv.Close)
	transport := newTransport(false)
	client := newRestyClient(transport, "127.0.0.1").SetRetryCount(0)
	p := &Provider{
		client:    client,
		transport: transport,
		baseURL:   srv.URL,
		realm:     "test",
		tokenSrc:  &tokenSource{client: client, tokenURL: srv.URL + "/token", clientID: "client", clientSecret: "secret"},
	}
	t.Cleanup(p.Close)
	return p, calls
}

func TestKeycloakURLNormalization(t *testing.T) {
	for _, invalid := range []string{"", "/relative", "ftp://example.com", "https:///missing", "https://user:password@example.com", "https://example.com?query=true", "https://example.com#fragment", "https://%zz"} {
		_, err := normalizeBaseURL(invalid)
		assert.Error(t, err, invalid)
	}
	value, err := normalizeBaseURL(" https://example.com/keycloak/// ")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/keycloak", value)
	assert.Equal(t, "First Last", userDisplayName(kcUser{FirstName: "First", LastName: "Last", Username: "login"}))
	assert.Equal(t, "login", userDisplayName(kcUser{Username: "login", Email: "a@example.com"}))
	assert.Equal(t, "a@example.com", userDisplayName(kcUser{Email: "a@example.com"}))
	assert.Equal(t, "/team/child", groupDisplayName(kcGroup{Path: "/team/child"}))
	values := url.Values{"key": {"one", "two"}}
	cloned := cloneValues(values)
	cloned["key"][0] = "changed"
	assert.Equal(t, "one", values.Get("key"))
	assert.NotNil(t, cloneValues(nil))
}

func TestKeycloakUsersPaginationAndConversion(t *testing.T) {
	p, tokens := newKeycloakHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
		assert.Equal(t, "/admin/realms/test/users", r.URL.Path)
		assert.Equal(t, "100", r.URL.Query().Get("max"))
		switch r.URL.Query().Get("first") {
		case "0":
			json.NewEncoder(w).Encode([]kcUser{{ID: "b", Email: "b@example.com", Enabled: false}, {ID: "a", FirstName: "First", LastName: "Last", Enabled: true}})
		case "2":
			json.NewEncoder(w).Encode([]kcUser{{ID: "b", Email: "b@example.com"}, {ID: "c", Username: "third", Enabled: true}})
		default:
			json.NewEncoder(w).Encode([]kcUser{})
		}
	})
	users, err := p.ListUsers(t.Context())
	require.NoError(t, err)
	require.Len(t, users, 3)
	assert.Equal(t, "a", users[0].ExternalID)
	assert.Equal(t, "First Last", users[0].DisplayName)
	assert.True(t, users[1].IsDisabled)
	assert.Equal(t, "third", users[2].DisplayName)
	assert.EqualValues(t, 1, tokens.Load())
}

func TestKeycloakRejectsBrokenPagination(t *testing.T) {
	for _, body := range []string{`[{"id":"same"}]`, `[{"username":"missing-id"}]`, `{invalid`} {
		p, _ := newKeycloakHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })
		users, err := p.ListUsers(t.Context())
		assert.Nil(t, users)
		assert.Error(t, err)
	}
}

func TestKeycloakNestedGroupsAndMembers(t *testing.T) {
	p, _ := newKeycloakHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("first") != "0" {
			io.WriteString(w, "[]")
			return
		}
		switch {
		case r.URL.Path == "/admin/realms/test/groups":
			io.WriteString(w, `[{"id":"parent","name":"team","subGroupCount":1}]`)
		case strings.HasSuffix(r.URL.Path, "/children"):
			io.WriteString(w, `[{"id":"child","path":"/team/child"}]`)
		case strings.HasSuffix(r.URL.Path, "/members"):
			io.WriteString(w, `[{"id":"user-b"},{"id":"user-a"},{"id":"user-b"}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	groups, err := p.ListGroups(t.Context())
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, "child", groups[0].ExternalID)
	assert.Equal(t, "/team/child", groups[0].DisplayName)
	for _, group := range groups {
		assert.Equal(t, []string{"user-a", "user-b"}, group.MemberExternalIDs)
	}
}

func TestKeycloakRejectsCyclicGroupHierarchy(t *testing.T) {
	p, _ := newKeycloakHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("first") != "0" {
			io.WriteString(w, "[]")
			return
		}
		io.WriteString(w, `[{"id":"cycle","subGroups":[{"id":"cycle"}]}]`)
	})
	groups, err := p.ListGroups(t.Context())
	assert.Nil(t, groups)
	assert.ErrorContains(t, err, "cyclic")
}

func TestKeycloakRefreshesUnauthorizedToken(t *testing.T) {
	p, tokens := newKeycloakHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer token-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		assert.Equal(t, "Bearer token-2", r.Header.Get("Authorization"))
		io.WriteString(w, "[]")
	})
	users, err := p.ListUsers(t.Context())
	require.NoError(t, err)
	assert.Empty(t, users)
	assert.EqualValues(t, 2, tokens.Load())
}

func TestKeycloakTokenCacheConcurrentAccess(t *testing.T) {
	p, calls := newKeycloakHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "[]") })
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := p.tokenSrc.get(t.Context())
			assert.NoError(t, err)
			assert.Equal(t, "token-1", token)
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, calls.Load())
	p.tokenSrc.invalidate("stale-token")
	assert.Equal(t, "token-1", p.tokenSrc.cachedToken())
	p.tokenSrc.invalidate("token-1")
	assert.Empty(t, p.tokenSrc.cachedToken())
	token, err := p.tokenSrc.get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "token-2", token)
	p.Close()
	p.Close()
}

func TestKeycloakCancellationAndResponseLimits(t *testing.T) {
	p, _ := newKeycloakHTTPTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, strings.Repeat("x", maxResponseBytes+1))
	})
	_, err := p.ListUsers(t.Context())
	assert.Error(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = p.ListUsers(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}
