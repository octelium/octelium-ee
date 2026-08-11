// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/pkg/errors"
)

const keycloakBudget = 3 * time.Minute

type Keycloak struct {
	h *H

	URL      string
	HostURL  string
	Realm    string
	ClientID string
	Secret   string

	user     string
	password string
	c        *resty.Client
}

func (h *H) Keycloak(t *testing.T) *Keycloak {
	t.Helper()

	ret := &Keycloak{
		h:        h,
		URL:      h.StateValue(t, eescenario.StateKeycloakURL),
		HostURL:  h.StateValue(t, eescenario.StateKeycloakHost),
		user:     h.StateValue(t, eescenario.StateKeycloakUser),
		password: h.StateValue(t, eescenario.StateKeycloakPass),
	}

	ret.c = h.HTTP().SetBaseURL(ret.HostURL)

	h.Eventually(t, "the Keycloak admin API to answer", keycloakBudget,
		func(ctx context.Context) error {
			_, err := ret.token(ctx)
			return err
		})

	return ret
}

func (k *Keycloak) token(ctx context.Context) (string, error) {
	var out struct {
		AccessToken string `json:"access_token"`
	}

	res, err := k.c.R().SetContext(ctx).
		SetFormData(map[string]string{
			"grant_type": "password",
			"client_id":  "admin-cli",
			"username":   k.user,
			"password":   k.password,
		}).
		SetResult(&out).
		Post("/realms/master/protocol/openid-connect/token")
	if err != nil {
		return "", err
	}
	if res.IsError() {
		return "", errors.Errorf("Could not obtain a Keycloak admin token: %d: %s",
			res.StatusCode(), res.String())
	}
	if out.AccessToken == "" {
		return "", errors.Errorf("The Keycloak admin token response is empty")
	}

	return out.AccessToken, nil
}

func (k *Keycloak) admin(t *testing.T) *resty.Client {
	t.Helper()

	ctx, cancel := k.h.Ctx(t)
	defer cancel()

	tkn, err := k.token(ctx)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	return k.c.SetAuthScheme("Bearer").SetAuthToken(tkn)
}

func (k *Keycloak) do(t *testing.T, method, path string, body any, out any) *resty.Response {
	t.Helper()

	req := k.admin(t).R()
	if body != nil {
		req = req.SetBody(body)
	}
	if out != nil {
		req = req.SetResult(out)
	}

	res, err := req.Execute(method, path)
	if err != nil {
		t.Fatalf("Keycloak %s %s failed: %+v", method, path, err)
	}
	if res.IsError() {
		t.Fatalf("Keycloak %s %s returned %d: %s", method, path, res.StatusCode(), res.String())
	}

	return res
}

func (k *Keycloak) SeedRealm(t *testing.T) {
	t.Helper()

	k.Realm = k.h.Name()
	k.ClientID = "octelium-e2e"
	k.Secret = k.h.Name() + k.h.Name()

	k.do(t, http.MethodPost, "/admin/realms", map[string]any{
		"realm":   k.Realm,
		"enabled": true,
	}, nil)

	t.Cleanup(func() {
		k.admin(t).R().Delete(fmt.Sprintf("/admin/realms/%s", k.Realm))
	})

	k.do(t, http.MethodPost, fmt.Sprintf("/admin/realms/%s/clients", k.Realm), map[string]any{
		"clientId":                  k.ClientID,
		"enabled":                   true,
		"protocol":                  "openid-connect",
		"publicClient":              false,
		"serviceAccountsEnabled":    true,
		"standardFlowEnabled":       false,
		"directAccessGrantsEnabled": false,
		"secret":                    k.Secret,
	}, nil)

	k.grantServiceAccountRoles(t)
}

func (k *Keycloak) grantServiceAccountRoles(t *testing.T) {
	t.Helper()

	var clients []struct {
		ID string `json:"id"`
	}
	k.do(t, http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/clients?clientId=%s", k.Realm, k.ClientID), nil, &clients)
	if len(clients) != 1 {
		t.Fatalf("Expected exactly one Keycloak client %q, got %d", k.ClientID, len(clients))
	}

	var svcAccount struct {
		ID string `json:"id"`
	}
	k.do(t, http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/clients/%s/service-account-user", k.Realm, clients[0].ID),
		nil, &svcAccount)

	var realmMgmt []struct {
		ID string `json:"id"`
	}
	k.do(t, http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/clients?clientId=realm-management", k.Realm),
		nil, &realmMgmt)
	if len(realmMgmt) != 1 {
		t.Fatalf("Could not find the realm-management client of the realm %s", k.Realm)
	}

	var available []map[string]any
	k.do(t, http.MethodGet, fmt.Sprintf(
		"/admin/realms/%s/users/%s/role-mappings/clients/%s/available",
		k.Realm, svcAccount.ID, realmMgmt[0].ID), nil, &available)

	wanted := map[string]bool{
		"view-users":    true,
		"query-users":   true,
		"query-groups":  true,
		"view-clients":  true,
		"view-realm":    true,
		"query-clients": true,
	}

	var grant []map[string]any
	for _, role := range available {
		name, _ := role["name"].(string)
		if wanted[name] {
			grant = append(grant, role)
		}
	}

	if len(grant) == 0 {
		return
	}

	k.do(t, http.MethodPost, fmt.Sprintf(
		"/admin/realms/%s/users/%s/role-mappings/clients/%s",
		k.Realm, svcAccount.ID, realmMgmt[0].ID), grant, nil)
}

func (k *Keycloak) CreateUser(t *testing.T, username, email, first, last string) string {
	t.Helper()

	k.do(t, http.MethodPost, fmt.Sprintf("/admin/realms/%s/users", k.Realm), map[string]any{
		"username":      username,
		"email":         email,
		"firstName":     first,
		"lastName":      last,
		"enabled":       true,
		"emailVerified": true,
	}, nil)

	return k.UserID(t, username)
}

func (k *Keycloak) UserID(t *testing.T, username string) string {
	t.Helper()

	var users []struct {
		ID string `json:"id"`
	}
	k.do(t, http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/users?username=%s&exact=true", k.Realm, username),
		nil, &users)
	if len(users) != 1 {
		t.Fatalf("Expected exactly one Keycloak user %q, got %d", username, len(users))
	}

	return users[0].ID
}

func (k *Keycloak) SetUserEnabled(t *testing.T, userID string, enabled bool) {
	t.Helper()

	k.do(t, http.MethodPut, fmt.Sprintf("/admin/realms/%s/users/%s", k.Realm, userID),
		map[string]any{"enabled": enabled}, nil)
}

func (k *Keycloak) DeleteUser(t *testing.T, userID string) {
	t.Helper()
	k.do(t, http.MethodDelete, fmt.Sprintf("/admin/realms/%s/users/%s", k.Realm, userID), nil, nil)
}

func (k *Keycloak) CreateGroup(t *testing.T, name string) string {
	t.Helper()

	k.do(t, http.MethodPost, fmt.Sprintf("/admin/realms/%s/groups", k.Realm),
		map[string]any{"name": name}, nil)

	return k.GroupID(t, name)
}

func (k *Keycloak) GroupID(t *testing.T, name string) string {
	t.Helper()

	var groups []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	k.do(t, http.MethodGet,
		fmt.Sprintf("/admin/realms/%s/groups?search=%s", k.Realm, name), nil, &groups)

	for _, grp := range groups {
		if grp.Name == name {
			return grp.ID
		}
	}

	t.Fatalf("Could not find the Keycloak group %q", name)
	return ""
}

func (k *Keycloak) DeleteGroup(t *testing.T, groupID string) {
	t.Helper()
	k.do(t, http.MethodDelete,
		fmt.Sprintf("/admin/realms/%s/groups/%s", k.Realm, groupID), nil, nil)
}

func (k *Keycloak) AddMember(t *testing.T, userID, groupID string) {
	t.Helper()
	k.do(t, http.MethodPut,
		fmt.Sprintf("/admin/realms/%s/users/%s/groups/%s", k.Realm, userID, groupID), nil, nil)
}

func (k *Keycloak) RemoveMember(t *testing.T, userID, groupID string) {
	t.Helper()
	k.do(t, http.MethodDelete,
		fmt.Sprintf("/admin/realms/%s/users/%s/groups/%s", k.Realm, userID, groupID), nil, nil)
}
