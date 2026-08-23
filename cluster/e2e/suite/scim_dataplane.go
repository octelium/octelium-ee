// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/go-resty/resty/v2"
	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scimCtx struct {
	h  *eeharness.H
	dp *enterprisev1.DirectoryProvider
	c  *resty.Client
}

func newSCIMCtx(t *testing.T, h *eeharness.H) *scimCtx {
	t.Helper()

	dp := h.CreateDirectoryProvider(t, &enterprisev1.DirectoryProvider{
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{
				Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
			},
		},
	})

	ctx, cancel := h.Ctx(t)
	defer cancel()

	tknResp, err := h.EnterpriseC().GenerateDirectoryProviderCredential(ctx,
		&enterprisev1.GenerateDirectoryProviderCredentialRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
			Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
		})
	require.Nil(t, err)

	return &scimCtx{
		h:  h,
		dp: dp,
		c:  scimClient(h, dp, tknResp.GetBearer().AccessToken),
	}
}

func scimClient(h *eeharness.H, dp *enterprisev1.DirectoryProvider, token string) *resty.Client {
	return h.HTTP().
		SetRetryCount(6).
		SetBaseURL(fmt.Sprintf("https://dirsync.octelium.%s/scim/%s", h.Domain, dp.Status.Id)).
		SetAuthScheme("Bearer").
		SetAuthToken(token).
		SetHeader("Content-Type", "application/scim+json").
		SetHeader("Accept", "application/scim+json")
}

func (s *scimCtx) createUser(t *testing.T, userName, email string) string {
	t.Helper()

	var out map[string]any

	res, err := s.c.R().
		SetBody(map[string]any{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			"userName": userName,
			"name": map[string]any{
				"givenName":  "E2E",
				"familyName": "User",
			},
			"active": true,
			"emails": []map[string]any{
				{"primary": true, "value": email, "type": "work"},
			},
		}).
		SetResult(&out).
		Post("/Users")
	require.Nil(t, err)
	require.Equal(t, http.StatusCreated, res.StatusCode(), res.String())

	id, ok := out["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, id)

	return id
}

func (s *scimCtx) createGroup(t *testing.T, displayName string) string {
	t.Helper()

	var out map[string]any

	res, err := s.c.R().
		SetBody(map[string]any{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			"displayName": displayName,
		}).
		SetResult(&out).
		Post("/Groups")
	require.Nil(t, err)
	require.Equal(t, http.StatusCreated, res.StatusCode(), res.String())

	id, ok := out["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, id)

	return id
}

func (s *scimCtx) patchMembers(t *testing.T, groupID string, op map[string]any) {
	t.Helper()

	res, err := s.c.R().
		SetBody(map[string]any{
			"schemas":    []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{op},
		}).
		Patch(fmt.Sprintf("/Groups/%s", groupID))
	require.Nil(t, err)
	require.Contains(t, []int{http.StatusOK, http.StatusNoContent}, res.StatusCode(), res.String())
}

func (s *scimCtx) octeliumUser(t *testing.T) *corev1.User {
	t.Helper()

	items := s.h.DirectoryUsers(t, s.dp)
	require.Equal(t, 1, len(items))

	ctx, cancel := s.h.Ctx(t)
	defer cancel()

	usr, err := s.h.CoreC().GetUser(ctx, &metav1.GetOptions{Uid: items[0].Status.UserRef.Uid})
	require.Nil(t, err)

	return usr
}

func (s *scimCtx) octeliumGroup(t *testing.T) *corev1.Group {
	t.Helper()

	items := s.h.DirectoryGroups(t, s.dp)
	require.Equal(t, 1, len(items))

	ctx, cancel := s.h.Ctx(t)
	defer cancel()

	grp, err := s.h.CoreC().GetGroup(ctx, &metav1.GetOptions{Uid: items[0].Status.GroupRef.Uid})
	require.Nil(t, err)

	return grp
}

func testSCIMProtocol(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	s := newSCIMCtx(t, h)

	first := s.createUser(t, "pagination.one@octelium.com", "pagination.one@octelium.com")

	t.Run("Filter", func(t *testing.T) {
		var out map[string]any

		res, err := s.c.R().SetResult(&out).
			SetQueryParam("filter", `userName eq "pagination.one@octelium.com"`).
			Get("/Users")
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode(), res.String())

		resources, ok := out["Resources"].([]any)
		require.True(t, ok, res.String())
		assert.Equal(t, 1, len(resources), res.String())
	})

	t.Run("UnsupportedFilterIsRejected", func(t *testing.T) {
		res, err := s.c.R().
			SetQueryParam("filter", `displayName co "octelium"`).
			Get("/Users")
		require.Nil(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode(), res.String())
	})

	t.Run("Pagination", func(t *testing.T) {
		var out map[string]any

		res, err := s.c.R().SetResult(&out).
			SetQueryParams(map[string]string{"startIndex": "1", "count": "1"}).
			Get("/Users")
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode(), res.String())

		resources, ok := out["Resources"].([]any)
		require.True(t, ok)
		assert.True(t, len(resources) <= 1)
	})

	t.Run("DuplicateUserName", func(t *testing.T) {
		res, err := s.c.R().
			SetBody(map[string]any{
				"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
				"userName": "pagination.one@octelium.com",
				"active":   true,
			}).
			Post("/Users")
		require.Nil(t, err)
		assert.Equal(t, http.StatusConflict, res.StatusCode(), res.String())
	})

	t.Run("UnknownID", func(t *testing.T) {
		res, err := s.c.R().Get("/Users/" + vutils.UUIDv4())
		require.Nil(t, err)
		assert.True(t, res.StatusCode() >= 400 && res.StatusCode() < 500,
			"an unknown SCIM id must not fault the server, got %d", res.StatusCode())
	})

	t.Run("MalformedIDIsRejected", func(t *testing.T) {
		res, err := s.c.R().Get("/Users/" + h.Name())
		require.Nil(t, err)
		assert.True(t, res.StatusCode() >= 400 && res.StatusCode() < 500,
			"a malformed SCIM id must not fault the server, got %d", res.StatusCode())
	})

	t.Run("PatchActive", func(t *testing.T) {
		res, err := s.c.R().
			SetBody(map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
				"Operations": []map[string]any{
					{"op": "replace", "path": "active", "value": false},
				},
			}).
			Patch(fmt.Sprintf("/Users/%s", first))
		require.Nil(t, err)
		require.Contains(t, []int{http.StatusOK, http.StatusNoContent}, res.StatusCode(),
			res.String())

		assert.True(t, s.octeliumUser(t).Spec.IsDisabled)
	})

	t.Run("RevokedCredential", func(t *testing.T) {
		ctx, cancel := h.Ctx(t)
		defer cancel()

		_, err := h.EnterpriseC().GenerateDirectoryProviderCredential(ctx,
			&enterprisev1.GenerateDirectoryProviderCredentialRequest{
				DirectoryProviderRef: umetav1.GetObjectReference(s.dp),
				Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
			})
		require.Nil(t, err)

		h.Eventually(t, "the previous SCIM bearer token to be rejected",
			eeharness.PropagationBudget, func(ctx context.Context) error {
				res, err := s.c.R().SetContext(ctx).Get("/Users")
				if err != nil {
					return err
				}
				if res.StatusCode() != http.StatusUnauthorized {
					return errUnexpectedStatus(res.StatusCode(), http.StatusUnauthorized)
				}
				return nil
			})
	})
}

func testSCIMDisabledProvider(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	s := newSCIMCtx(t, h)

	ctx, cancel := h.Ctx(t)
	defer cancel()

	s.dp.Spec.IsDisabled = true
	_, err := h.EnterpriseC().UpdateDirectoryProvider(ctx, s.dp)
	require.Nil(t, err)

	h.Eventually(t, "the disabled DirectoryProvider to reject SCIM requests",
		eeharness.PropagationBudget, func(ctx context.Context) error {
			res, err := s.c.R().SetContext(ctx).Get("/Users")
			if err != nil {
				return err
			}
			if res.StatusCode() == http.StatusOK {
				return errUnexpectedStatus(res.StatusCode(), http.StatusForbidden)
			}
			return nil
		})
}

func testSCIMToDataPlane(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	s := newSCIMCtx(t, h)

	userID := s.createUser(t, "dataplane@octelium.com", "dataplane@octelium.com")
	groupID := s.createGroup(t, "e2e-dataplane")

	usr := s.octeliumUser(t)
	grp := s.octeliumGroup(t)

	svc := h.NewPublicService(t, "default")

	svc.Spec.Authorization = &corev1.Service_Spec_Authorization{
		InlinePolicies: []*corev1.InlinePolicy{
			{
				Name: "allow-group",
				Spec: &corev1.Policy_Spec{
					Rules: []*corev1.Policy_Spec_Rule{
						harness.MatchRule("allow-group", 0, corev1.Policy_Spec_Rule_ALLOW,
							fmt.Sprintf(`ctx.user.spec.groups.exists(g, g == %q)`,
								grp.Metadata.Name)),
					},
				},
			},
		},
	}
	h.UpdateService(t, svc)

	t.Run("DeniedBeforeMembership", func(t *testing.T) {
		waitAuthorization(t, h, usr, svc, false)
	})

	t.Run("AllowedAfterMembership", func(t *testing.T) {
		s.patchMembers(t, groupID, map[string]any{
			"op":    "add",
			"path":  "members",
			"value": []map[string]any{{"value": userID}},
		})

		waitAuthorization(t, h, usr, svc, true)
	})

	t.Run("TheCoreUserCarriesTheGroup", func(t *testing.T) {
		assert.True(t, slices.Contains(s.octeliumUser(t).Spec.Groups, grp.Metadata.Name))
	})

	t.Run("DeniedAfterRemoval", func(t *testing.T) {
		s.patchMembers(t, groupID, map[string]any{
			"op":   "remove",
			"path": fmt.Sprintf("members[value eq %q]", userID),
		})

		waitAuthorization(t, h, usr, svc, false)
	})
}
