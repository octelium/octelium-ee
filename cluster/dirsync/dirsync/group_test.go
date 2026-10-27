// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package dirsync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/gorilla/mux"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/middlewares"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

func TestGroup(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	idp, err := fakeC.OcteliumC.EnterpriseC().CreateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.DirectoryProvider_Spec{},
		Status: &enterprisev1.DirectoryProvider_Status{
			Id: utilrand.GetRandomStringCanonical(4),
		},
	})
	assert.Nil(t, err)

	{
		reqURL, _ := url.Parse("http://localhost/scim/v2/Users")
		query := reqURL.Query()
		query.Add("filter", `displayName eq "group1@example.com"`)
		reqURL.RawQuery = query.Encode()

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))
		reqHTTP := httptest.NewRequest("GET", path, nil)

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		w := httptest.NewRecorder()

		srv.handleListGroup(w, reqHTTP)

		resp := w.Result()
		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		res := &responseGroupList{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		assert.Equal(t, 0, len(res.Resources))
	}

	var grp *corev1.Group
	var dpGrp *enterprisev1.DirectoryProviderGroup

	{
		reqURL, _ := url.Parse("http://localhost/scim/v2/Groups")

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		req := &resourceGroup{
			resourceCommon: resourceCommon{
				ExternalID: utilrand.GetRandomString(12),
				Schemas:    []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
				Meta: resourceMeta{
					ResourceType: "Group",
				},
			},
			DisplayName: "group1@example.com",
		}

		reqBody, err := json.Marshal(req)
		assert.Nil(t, err)
		reqHTTP := httptest.NewRequest("POST", path, bytes.NewBuffer(reqBody))

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		w := httptest.NewRecorder()

		srv.handleCreateGroup(w, reqHTTP)

		resp := w.Result()
		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		res := &resourceGroup{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		dpGrp, err = srv.octeliumC.EnterpriseC().GetDirectoryProviderGroup(ctx, &rmetav1.GetOptions{
			Uid: res.ID,
		})
		assert.Nil(t, err)
		grp, err = srv.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
			Uid: dpGrp.Status.GroupRef.Uid,
		})
		assert.Nil(t, err)

		scimGroup, err := srv.toGroupSCIM(dpGrp)
		assert.Nil(t, err)
		assert.True(t, reflect.DeepEqual(res, scimGroup))

		assert.Equal(t, "group1@example.com", grp.Metadata.DisplayName)

		{
			reqBody, err := json.Marshal(req)
			assert.Nil(t, err)
			reqHTTP := httptest.NewRequest("POST", path, bytes.NewBuffer(reqBody))

			reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
				DirectoryProvider: idp,
			}))

			w := httptest.NewRecorder()

			srv.handleCreateGroup(w, reqHTTP)

			resp := w.Result()

			assert.Equal(t, http.StatusConflict, resp.StatusCode)
		}
	}

	{
		reqURL, _ := url.Parse("http://localhost/scim/v2/Group")
		query := reqURL.Query()
		query.Add("filter", `displayName eq "group1@example.com"`)
		reqURL.RawQuery = query.Encode()

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))
		reqHTTP := httptest.NewRequest("GET", path, nil)

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		w := httptest.NewRecorder()

		srv.handleListGroup(w, reqHTTP)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		res := &responseGroupList{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		assert.Equal(t, 1, res.StartIndex)
		assert.Equal(t, 1, len(res.Resources))
		assert.Equal(t, res.Resources[0].ID, dpGrp.Metadata.Uid)
	}

	{
		reqURL, _ := url.Parse(fmt.Sprintf("http://localhost/scim/v2/Groups/%s", dpGrp.Metadata.Uid))

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))
		reqHTTP := httptest.NewRequest("GET", path, nil)

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		reqHTTP = mux.SetURLVars(reqHTTP, map[string]string{
			"uid": dpGrp.Metadata.Uid,
		})

		w := httptest.NewRecorder()

		srv.handleGetGroup(w, reqHTTP)

		resp := w.Result()
		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		res := &resourceGroup{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		assert.Equal(t, res.ID, dpGrp.Metadata.Uid)
	}

	{
		reqURL, _ := url.Parse(fmt.Sprintf("http://localhost/scim/v2/Groups/%s", dpGrp.Metadata.Uid))

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		req := &patchRequest{
			Operations: []patchRequestOperations{
				{
					Op:    "Replace",
					Path:  `displayName`,
					Value: "group1b@example.com",
				},
			},
		}
		reqBody, _ := json.Marshal(req)
		reqHTTP := httptest.NewRequest("PATCH", path, bytes.NewBuffer(reqBody))

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		reqHTTP = mux.SetURLVars(reqHTTP, map[string]string{
			"uid": dpGrp.Metadata.Uid,
		})

		w := httptest.NewRecorder()

		srv.handlePatchGroup(w, reqHTTP)

		resp := w.Result()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	}

	var usr *corev1.User
	var dpUsr *enterprisev1.DirectoryProviderUser

	{
		reqURL, _ := url.Parse("http://localhost/scim/v2/Users")

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		req := &resourceUser{
			resourceCommon: resourceCommon{
				ExternalID: utilrand.GetRandomString(12),
				Schemas:    []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
				Meta: resourceMeta{
					ResourceType: "User",
				},
			},
			UserName: "usr1@example.com",
			Active:   utils_types.BoolToPtr(true),
			Emails: []*resourceUserEmail{
				{
					Primary: true,
					Value:   "usr1@example.com",
					Type:    "work",
				},
			},
			Name: resourceUserName{
				Formatted:  "User One",
				GivenName:  "User",
				FamilyName: "One",
			},
		}

		reqBody, err := json.Marshal(req)
		assert.Nil(t, err)
		reqHTTP := httptest.NewRequest("POST", path, bytes.NewBuffer(reqBody))

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		w := httptest.NewRecorder()

		srv.handleCreateUser(w, reqHTTP)

		resp := w.Result()
		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		res := &resourceUser{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		dpUsr, err = srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{
			Uid: res.ID,
		})
		assert.Nil(t, err)
		usr, err = srv.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: dpUsr.Status.UserRef.Uid,
		})
		assert.Nil(t, err)

		scimUsr, err := srv.toUserSCIM(dpUsr)
		assert.Nil(t, err)
		assert.True(t, reflect.DeepEqual(res, scimUsr))

		assert.False(t, usr.Spec.IsDisabled)
	}

	{
		reqURL, _ := url.Parse(fmt.Sprintf("http://localhost/scim/v2/Groups/%s", dpGrp.Metadata.Uid))

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		scimUsr, err := srv.toUserSCIM(dpUsr)
		assert.Nil(t, err)

		req := &patchRequest{
			Operations: []patchRequestOperations{
				{
					Op:   "Add",
					Path: `Members`,
					Value: []resourceGroupMember{
						{
							Value:   dpUsr.Metadata.Uid,
							Display: scimUsr.UserName,
						},
					},
				},
			},
		}
		reqBody, _ := json.Marshal(req)
		reqHTTP := httptest.NewRequest("PATCH", path, bytes.NewBuffer(reqBody))

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		reqHTTP = mux.SetURLVars(reqHTTP, map[string]string{
			"uid": dpGrp.Metadata.Uid,
		})

		w := httptest.NewRecorder()

		srv.handlePatchGroup(w, reqHTTP)

		resp := w.Result()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		usr, err = srv.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Uid: usr.Metadata.Uid})
		assert.Nil(t, err)

		assert.True(t, ucorev1.ToUser(usr).HasGroupName(grp.Metadata.Name))
	}

	{
		reqURL, _ := url.Parse(fmt.Sprintf("http://localhost/scim/v2/Groups/%s", dpGrp.Metadata.Uid))

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		scimUsr, err := srv.toUserSCIM(dpUsr)
		assert.Nil(t, err)

		req := &patchRequest{
			Operations: []patchRequestOperations{
				{
					Op:   "Remove",
					Path: `Members`,
					Value: []resourceGroupMember{
						{
							Value:   dpUsr.Metadata.Uid,
							Display: scimUsr.UserName,
						},
					},
				},
			},
		}
		reqBody, _ := json.Marshal(req)
		reqHTTP := httptest.NewRequest("PATCH", path, bytes.NewBuffer(reqBody))

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		reqHTTP = mux.SetURLVars(reqHTTP, map[string]string{
			"uid": dpGrp.Metadata.Uid,
		})

		w := httptest.NewRecorder()

		srv.handlePatchGroup(w, reqHTTP)

		resp := w.Result()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		usr, err = srv.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Uid: usr.Metadata.Uid})
		assert.Nil(t, err)

		assert.False(t, ucorev1.ToUser(usr).HasGroupName(grp.Metadata.Name))
	}

	{
		reqURL, _ := url.Parse(fmt.Sprintf("http://localhost/scim/v2/Groups/%s", dpGrp.Metadata.Uid))

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		reqHTTP := httptest.NewRequest("PATCH", path, nil)

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		reqHTTP = mux.SetURLVars(reqHTTP, map[string]string{
			"uid": dpGrp.Metadata.Uid,
		})

		w := httptest.NewRecorder()

		srv.handleDeleteGroup(w, reqHTTP)

		resp := w.Result()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		_, err = srv.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{Uid: dpGrp.Metadata.Uid})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err))

	}

	{
		reqURL, _ := url.Parse("http://localhost/scim/v2/Groups")

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		req := &resourceGroup{
			resourceCommon: resourceCommon{
				ExternalID: utilrand.GetRandomString(12),
				Schemas:    []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
				Meta: resourceMeta{
					ResourceType: "Group",
				},
			},
			DisplayName: "group2@example.com",
			Members: []resourceGroupMember{
				{
					Value: dpUsr.Metadata.Uid,
				},
			},
		}

		reqBody, err := json.Marshal(req)
		assert.Nil(t, err)
		reqHTTP := httptest.NewRequest("POST", path, bytes.NewBuffer(reqBody))

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		w := httptest.NewRecorder()

		srv.handleCreateGroup(w, reqHTTP)

		resp := w.Result()
		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		res := &resourceGroup{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		dpGrp, err = srv.octeliumC.EnterpriseC().GetDirectoryProviderGroup(ctx, &rmetav1.GetOptions{
			Uid: res.ID,
		})
		assert.Nil(t, err)
		grp, err = srv.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
			Uid: dpGrp.Status.GroupRef.Uid,
		})
		assert.Nil(t, err)

		scimGroup, err := srv.toGroupSCIM(dpGrp)
		assert.Nil(t, err)
		assert.True(t, reflect.DeepEqual(res, scimGroup))

		assert.Equal(t, "group2@example.com", grp.Metadata.DisplayName)

		usr, err := srv.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: dpUsr.Status.UserRef.Uid,
		})
		assert.Nil(t, err)

		assert.True(t, slices.Contains(usr.Spec.Groups, grp.Metadata.Name))
	}

}

func TestSCIMGroupBodyValidationBoundaries(t *testing.T) {
	for _, input := range []string{"", "{", "[]", strings.Repeat("x", (1<<20)+1)} {
		request := httptest.NewRequest(http.MethodPost, "/scim/provider/Groups", strings.NewReader(input))
		_, err := (&server{}).unmarshalGroupSCIMFromReqBody(request, httptest.NewRecorder())
		assert.Error(t, err)
	}
	request := httptest.NewRequest(http.MethodPost, "/scim/provider/Groups", strings.NewReader(`{"externalId":"external"}`))
	value, err := (&server{}).unmarshalGroupSCIMFromReqBody(request, httptest.NewRecorder())
	require.NoError(t, err)
	assert.Equal(t, "external", value.ExternalID)
}

func TestSCIMGroupPatchBodyErrors(t *testing.T) {
	srv, handler, dp := newSCIMPatchTestServer(t)
	request := func(method, path string, body io.Reader) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, body)
		r.Header.Set("X-Octelium-Session-Uid", dp.Status.SessionRef.Uid)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	w := request(http.MethodPost, srv.getUserPrefix(dp), strings.NewReader(`{"externalId":"patch-user","userName":"patch@example.com","active":true,"emails":[{"value":"patch@example.com","type":"work","primary":true}]}`))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var scimUsr resourceUser
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &scimUsr))
	w = request(http.MethodPost, srv.getGroupPrefix(dp), strings.NewReader(fmt.Sprintf(`{"externalId":"patch-group","displayName":"Before","members":[{"value":%q}]}`, scimUsr.ID)))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var scimGrp resourceGroup
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &scimGrp))
	link, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderGroup(t.Context(), &rmetav1.GetOptions{Uid: scimGrp.ID})
	require.NoError(t, err)
	grp, err := srv.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Uid: link.Status.GroupRef.Uid})
	require.NoError(t, err)
	usrLink, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Uid: scimUsr.ID})
	require.NoError(t, err)
	usr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: usrLink.Status.UserRef.Uid})
	require.NoError(t, err)
	require.Contains(t, usr.Spec.Groups, grp.Metadata.Name)
	patch, err := json.Marshal(&patchRequest{Operations: []patchRequestOperations{
		{Op: "Remove", Path: fmt.Sprintf(`members[value eq "%s"]`, scimUsr.ID)},
	}})
	require.NoError(t, err)
	readErr := fmt.Errorf("private transport failure")
	for _, tc := range []struct {
		name   string
		body   io.Reader
		status int
	}{
		{"Oversized", strings.NewReader(string(patch) + strings.Repeat(" ", (1<<20)+1-len(patch))), http.StatusRequestEntityTooLarge},
		{"ReadFailure", iotest.ErrReader(readErr), http.StatusBadRequest},
		{"ReadFailureAfterValidPatch", io.MultiReader(bytes.NewReader(patch), iotest.ErrReader(readErr)), http.StatusBadRequest},
		{"WrappedSizeFailure", iotest.ErrReader(fmt.Errorf("body read: %w", &http.MaxBytesError{Limit: 1 << 20})), http.StatusRequestEntityTooLarge},
		{"MalformedJSON", strings.NewReader("{"), http.StatusBadRequest},
		{"ExactLimit", strings.NewReader(string(patch) + strings.Repeat(" ", (1<<20)-len(patch))), http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &scimPatchTestBody{Reader: tc.body}
			w := request(http.MethodPatch, srv.getGroupPrefix(dp)+"/"+scimGrp.ID, body)
			assert.True(t, body.closed)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			assert.Equal(t, "application/scim+json", w.Header().Get("Content-Type"))
			updatedGrp, err := srv.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Uid: grp.Metadata.Uid})
			require.NoError(t, err)
			updatedLink, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderGroup(t.Context(), &rmetav1.GetOptions{Uid: link.Metadata.Uid})
			require.NoError(t, err)
			updatedUsr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: usr.Metadata.Uid})
			require.NoError(t, err)
			if tc.status == http.StatusNoContent {
				assert.Equal(t, "Before", updatedGrp.Metadata.DisplayName)
				assert.NotContains(t, updatedUsr.Spec.Groups, grp.Metadata.Name)
				assert.Empty(t, w.Body.String())
				return
			}
			assert.Equal(t, grp.Metadata.ResourceVersion, updatedGrp.Metadata.ResourceVersion)
			assert.Equal(t, "Before", updatedGrp.Metadata.DisplayName)
			assert.Equal(t, link.Metadata.ResourceVersion, updatedLink.Metadata.ResourceVersion)
			assert.Equal(t, usr.Metadata.ResourceVersion, updatedUsr.Metadata.ResourceVersion)
			assert.Contains(t, updatedUsr.Spec.Groups, grp.Metadata.Name)
			var response responseError
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			assert.Equal(t, fmt.Sprint(tc.status), response.Status)
			assert.Equal(t, []string{"urn:ietf:params:scim:api:messages:2.0:Error"}, response.Schemas)
			assert.NotEmpty(t, response.Detail)
			assert.NotContains(t, response.Detail, readErr.Error())
		})
	}
}
