// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSCIM(t *testing.T, h *harness.H) {
	ctx := t.Context()

	coreC := h.CoreC()
	enterpriseC := enterprisev1.NewMainServiceClient(h.Conn())

	dp, err := enterpriseC.CreateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{
				Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
			},
		},
	})
	require.Nil(t, err)

	tknResp, err := enterpriseC.GenerateDirectoryProviderCredential(ctx,
		&enterprisev1.GenerateDirectoryProviderCredentialRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
			Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
		})
	require.Nil(t, err)

	scimC := h.HTTP().
		SetBaseURL(fmt.Sprintf("https://dirsync.octelium.%s/scim/%s", h.Domain, dp.Status.Id)).
		SetAuthScheme("Bearer").
		SetAuthToken(tknResp.GetBearer().AccessToken).
		SetHeader("Content-Type", "application/scim+json").
		SetHeader("Accept", "application/scim+json")

	t.Run("Discovery", func(t *testing.T) {
		for _, ep := range []string{"/ServiceProviderConfig", "/ResourceTypes", "/Schemas"} {
			res, err := scimC.R().Get(ep)
			require.Nil(t, err)
			assert.Equal(t, http.StatusOK, res.StatusCode(), "endpoint %s", ep)
		}
	})

	var userRes map[string]any

	createRes, err := scimC.R().
		SetBody(map[string]any{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			"userName": "john.doe@octelium.com",
			"name": map[string]any{
				"givenName":  "John",
				"familyName": "Doe",
			},
			"active": true,
			"emails": []map[string]any{
				{"primary": true, "value": "john.doe@octelium.com", "type": "work"},
			},
		}).
		SetResult(&userRes).
		Post("/Users")
	require.Nil(t, err)
	require.Equal(t, http.StatusCreated, createRes.StatusCode())

	usrList, err := enterpriseC.ListDirectoryProviderUser(ctx,
		&enterprisev1.ListDirectoryProviderUserOptions{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
	require.Nil(t, err)
	require.Equal(t, 1, len(usrList.Items))

	userUID := usrList.Items[0].Status.UserRef.Uid

	userID, ok := userRes["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, userID)

	t.Run("ProvisionedUser", func(t *testing.T) {
		usr, err := coreC.GetUser(ctx, &metav1.GetOptions{Uid: userUID})
		require.Nil(t, err)
		assert.Equal(t, "john.doe@octelium.com", usr.Spec.Email)

		res, err := scimC.R().Get(fmt.Sprintf("/Users/%s", userID))
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())
	})

	t.Run("ReplaceUser", func(t *testing.T) {
		res, err := scimC.R().
			SetBody(map[string]any{
				"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
				"userName": "test.scim.updated@octelium.com",
				"name": map[string]any{
					"givenName":  "John",
					"familyName": "Linus",
				},
				"active": false,
				"emails": []map[string]any{
					{"primary": true, "value": "john.doe.updated@octelium.com", "type": "work"},
				},
			}).
			Put(fmt.Sprintf("/Users/%s", userID))
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode())

		usr, err := coreC.GetUser(ctx, &metav1.GetOptions{Uid: userUID})
		require.Nil(t, err)
		assert.Equal(t, "john.doe.updated@octelium.com", usr.Spec.Email)
		assert.True(t, usr.Spec.IsDisabled)
	})

	var groupRes map[string]any

	cGroupRes, err := scimC.R().
		SetBody(map[string]any{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			"displayName": "Test Group",
		}).
		SetResult(&groupRes).
		Post("/Groups")
	require.Nil(t, err)
	require.Equal(t, http.StatusCreated, cGroupRes.StatusCode())

	groupID, ok := groupRes["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, groupID)

	groupList, err := enterpriseC.ListDirectoryProviderGroup(ctx,
		&enterprisev1.ListDirectoryProviderGroupOptions{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
	require.Nil(t, err)
	require.Equal(t, 1, len(groupList.Items))

	groupUID := groupList.Items[0].Status.GroupRef.Uid

	grp, err := coreC.GetGroup(ctx, &metav1.GetOptions{Uid: groupUID})
	require.Nil(t, err)

	t.Run("PatchAddMember", func(t *testing.T) {
		res, err := scimC.R().
			SetBody(map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
				"Operations": []map[string]any{
					{
						"op":    "add",
						"path":  "members",
						"value": []map[string]any{{"value": userID}},
					},
				},
			}).
			Patch(fmt.Sprintf("/Groups/%s", groupID))
		require.Nil(t, err)
		require.Contains(t, []int{http.StatusOK, http.StatusNoContent}, res.StatusCode())

		usr, err := coreC.GetUser(ctx, &metav1.GetOptions{Uid: userUID})
		require.Nil(t, err)
		assert.True(t, slices.Contains(usr.Spec.Groups, grp.Metadata.Name))
	})

	t.Run("PatchRemoveMember", func(t *testing.T) {
		res, err := scimC.R().
			SetBody(map[string]any{
				"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
				"Operations": []map[string]any{
					{
						"op":   "remove",
						"path": fmt.Sprintf("members[value eq %q]", userID),
					},
				},
			}).
			Patch(fmt.Sprintf("/Groups/%s", groupID))
		require.Nil(t, err)
		require.Contains(t, []int{http.StatusOK, http.StatusNoContent}, res.StatusCode())

		usr, err := coreC.GetUser(ctx, &metav1.GetOptions{Uid: userUID})
		require.Nil(t, err)
		assert.False(t, slices.Contains(usr.Spec.Groups, grp.Metadata.Name))
	})

	t.Run("Deprovision", func(t *testing.T) {
		res, err := scimC.R().Delete(fmt.Sprintf("/Groups/%s", groupID))
		require.Nil(t, err)
		assert.Equal(t, http.StatusNoContent, res.StatusCode())

		res, err = scimC.R().Delete(fmt.Sprintf("/Users/%s", userID))
		require.Nil(t, err)
		assert.Equal(t, http.StatusNoContent, res.StatusCode())

		_, err = coreC.GetUser(ctx, &metav1.GetOptions{Uid: userUID})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err))

		_, err = coreC.GetGroup(ctx, &metav1.GetOptions{Uid: groupUID})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err))
	})
}
