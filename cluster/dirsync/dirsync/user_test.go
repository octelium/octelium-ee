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
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func TestUser(t *testing.T) {
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
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{
				Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
			},
		},
		Status: &enterprisev1.DirectoryProvider_Status{
			Id: utilrand.GetRandomStringCanonical(4),
		},
	})
	assert.Nil(t, err)

	{
		reqURL, _ := url.Parse("http://localhost/scim/v2/Users")
		query := reqURL.Query()
		query.Add("filter", `userName eq "usr1@example.com"`)
		reqURL.RawQuery = query.Encode()

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))
		reqHTTP := httptest.NewRequest("GET", path, nil)

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		w := httptest.NewRecorder()

		srv.handleListUser(w, reqHTTP)

		resp := w.Result()
		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		res := &responseUserList{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		assert.Equal(t, 0, len(res.Resources))
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
		assert.Nil(t, err, "%+v", err)

		scimUsr, err := srv.toUserSCIM(dpUsr)
		assert.Nil(t, err)
		assert.True(t, reflect.DeepEqual(res, scimUsr))

		assert.Equal(t, idp.Metadata.Uid, dpUsr.Status.DirectoryProviderRef.Uid)
		assert.False(t, usr.Spec.IsDisabled)

		{
			reqBody, err := json.Marshal(req)
			assert.Nil(t, err)
			reqHTTP := httptest.NewRequest("POST", path, bytes.NewBuffer(reqBody))

			reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
				DirectoryProvider: idp,
			}))

			w := httptest.NewRecorder()

			srv.handleCreateUser(w, reqHTTP)

			resp := w.Result()

			assert.Equal(t, http.StatusConflict, resp.StatusCode)
		}
	}

	{
		reqURL, _ := url.Parse("http://localhost/scim/v2/Users")
		query := reqURL.Query()
		query.Add("filter", `userName eq "usr1@example.com"`)
		reqURL.RawQuery = query.Encode()

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))
		reqHTTP := httptest.NewRequest("GET", path, nil)

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		w := httptest.NewRecorder()

		srv.handleListUser(w, reqHTTP)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		res := &responseUserList{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		assert.Equal(t, maxItemsPerPage, res.ItemsPerPage)
		assert.Equal(t, 1, res.StartIndex)
		assert.Equal(t, 1, len(res.Resources))
		assert.Equal(t, res.Resources[0].ID, dpUsr.Metadata.Uid)
	}

	{
		reqURL, _ := url.Parse(fmt.Sprintf("http://localhost/scim/v2/Users/%s", dpUsr.Metadata.Uid))

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))
		reqHTTP := httptest.NewRequest("GET", path, nil)

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		reqHTTP = mux.SetURLVars(reqHTTP, map[string]string{
			"uid": dpUsr.Metadata.Uid,
		})

		w := httptest.NewRecorder()

		srv.handleGetUser(w, reqHTTP)

		resp := w.Result()
		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		res := &resourceUser{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		assert.Equal(t, res.ID, dpUsr.Metadata.Uid)
	}

	{
		reqURL, _ := url.Parse(fmt.Sprintf("http://localhost/scim/v2/Users/%s", dpUsr.Metadata.Uid))

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		req := &patchRequest{
			Operations: []patchRequestOperations{
				{
					Op:    "Replace",
					Path:  `userName`,
					Value: "usr1b@example.com",
				},
			},
		}
		reqBody, _ := json.Marshal(req)
		reqHTTP := httptest.NewRequest("PATCH", path, bytes.NewBuffer(reqBody))

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		reqHTTP = mux.SetURLVars(reqHTTP, map[string]string{
			"uid": dpUsr.Metadata.Uid,
		})

		w := httptest.NewRecorder()

		srv.handlePatchUser(w, reqHTTP)

		resp := w.Result()
		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		res := &resourceUser{}
		err = json.Unmarshal(bb, res)
		assert.Nil(t, err)

		assert.Equal(t, res.ID, dpUsr.Metadata.Uid)
		assert.Equal(t, "usr1b@example.com", res.UserName)

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
		assert.Equal(t, "usr1b@example.com", scimUsr.UserName)
	}

	{
		reqURL, _ := url.Parse(fmt.Sprintf("http://localhost/scim/v2/Users/%s", dpUsr.Metadata.Uid))

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		req := &patchRequest{
			Operations: []patchRequestOperations{
				{
					Op:    "Replace",
					Path:  `emails[type eq "work"].value`,
					Value: "workemail1234@example.com",
				},
				{
					Op:    "Replace",
					Path:  `active`,
					Value: false,
				},
			},
		}
		reqBody, _ := json.Marshal(req)
		reqHTTP := httptest.NewRequest("PATCH", path, bytes.NewBuffer(reqBody))

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		reqHTTP = mux.SetURLVars(reqHTTP, map[string]string{
			"uid": dpUsr.Metadata.Uid,
		})

		w := httptest.NewRecorder()

		srv.handlePatchUser(w, reqHTTP)

		resp := w.Result()
		bb, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

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

		assert.Equal(t, res.ID, dpUsr.Metadata.Uid)
		assert.Equal(t, "workemail1234@example.com", res.Emails[0].Value)
		assert.False(t, *res.Active)

		dpUsr, err = srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{
			Uid: res.ID,
		})
		assert.Nil(t, err)
		usr, err = srv.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: dpUsr.Status.UserRef.Uid,
		})
		assert.Nil(t, err)
		assert.True(t, usr.Spec.IsDisabled)

		scimUsr, err := srv.toUserSCIM(dpUsr)
		assert.Nil(t, err)
		assert.Equal(t, "usr1b@example.com", scimUsr.UserName)
	}

	{
		reqURL, _ := url.Parse(fmt.Sprintf("http://localhost/scim/v2/Users/%s", dpUsr.Metadata.Uid))

		path := reqURL.String()
		zap.L().Debug("path", zap.String("val", path))

		reqHTTP := httptest.NewRequest("PATCH", path, nil)

		reqHTTP = reqHTTP.WithContext(context.WithValue(reqHTTP.Context(), middlewares.CtxRequestContext, &middlewares.RequestContext{
			DirectoryProvider: idp,
		}))

		reqHTTP = mux.SetURLVars(reqHTTP, map[string]string{
			"uid": dpUsr.Metadata.Uid,
		})

		w := httptest.NewRecorder()

		srv.handleDeleteUser(w, reqHTTP)

		resp := w.Result()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		_, err = srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(ctx, &rmetav1.GetOptions{
			Uid: dpUsr.Metadata.Uid,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err))

		_, err = srv.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: usr.Metadata.Uid,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err))

	}

}

func TestSCIMUserBodyValidationBoundaries(t *testing.T) {
	for _, input := range []string{"", "{", "[]", strings.Repeat("x", (1<<20)+1)} {
		request := httptest.NewRequest(http.MethodPost, "/scim/provider/Users", strings.NewReader(input))
		_, err := (&server{}).unmarshalUserSCIMFromReqBody(request, httptest.NewRecorder())
		assert.Error(t, err)
	}
	request := httptest.NewRequest(http.MethodPost, "/scim/provider/Users", strings.NewReader(`{"externalId":"external"}`))
	value, err := (&server{}).unmarshalUserSCIMFromReqBody(request, httptest.NewRecorder())
	require.NoError(t, err)
	assert.Equal(t, "external", value.ExternalID)
}

type scimPatchTestBody struct {
	io.Reader
	closed bool
}

func (b *scimPatchTestBody) Close() error {
	b.closed = true
	return nil
}

func TestSCIMUserPatchBodyErrors(t *testing.T) {
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
	link, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Uid: scimUsr.ID})
	require.NoError(t, err)
	usr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: link.Status.UserRef.Uid})
	require.NoError(t, err)
	require.False(t, usr.Spec.IsDisabled)
	patch := `{"Operations":[{"op":"Replace","path":"active","value":false}]}`
	readErr := fmt.Errorf("private transport failure")
	for _, tc := range []struct {
		name   string
		body   io.Reader
		status int
	}{
		{"Oversized", strings.NewReader(patch + strings.Repeat(" ", (1<<20)+1-len(patch))), http.StatusRequestEntityTooLarge},
		{"ReadFailure", iotest.ErrReader(readErr), http.StatusBadRequest},
		{"ReadFailureAfterValidPatch", io.MultiReader(strings.NewReader(patch), iotest.ErrReader(readErr)), http.StatusBadRequest},
		{"WrappedSizeFailure", iotest.ErrReader(fmt.Errorf("body read: %w", &http.MaxBytesError{Limit: 1 << 20})), http.StatusRequestEntityTooLarge},
		{"MalformedJSON", strings.NewReader("{"), http.StatusBadRequest},
		{"ExactLimit", strings.NewReader(patch + strings.Repeat(" ", (1<<20)-len(patch))), http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &scimPatchTestBody{Reader: tc.body}
			w := request(http.MethodPatch, srv.getUserPrefix(dp)+"/"+scimUsr.ID, body)
			assert.True(t, body.closed)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			assert.Equal(t, "application/scim+json", w.Header().Get("Content-Type"))
			updatedUsr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: usr.Metadata.Uid})
			require.NoError(t, err)
			updatedLink, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Uid: link.Metadata.Uid})
			require.NoError(t, err)
			if tc.status == http.StatusOK {
				assert.True(t, updatedUsr.Spec.IsDisabled)
				var response resourceUser
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				require.NotNil(t, response.Active)
				assert.False(t, *response.Active)
				assert.Equal(t, scimUsr.ID, response.ID)
				return
			}
			assert.Equal(t, usr.Metadata.ResourceVersion, updatedUsr.Metadata.ResourceVersion)
			assert.False(t, updatedUsr.Spec.IsDisabled)
			assert.Equal(t, link.Metadata.ResourceVersion, updatedLink.Metadata.ResourceVersion)
			var response responseError
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			assert.Equal(t, fmt.Sprint(tc.status), response.Status)
			assert.Equal(t, []string{"urn:ietf:params:scim:api:messages:2.0:Error"}, response.Schemas)
			assert.NotEmpty(t, response.Detail)
			assert.NotContains(t, response.Detail, readErr.Error())
		})
	}
}

func TestSCIMUserConversionRejectsNullEntries(t *testing.T) {
	srv := &server{}
	dp := &enterprisev1.DirectoryProvider{Status: &enterprisev1.DirectoryProvider_Status{Id: "provider"}}
	for _, tc := range []struct {
		name string
		user *resourceUser
	}{
		{"NilUser", nil},
		{"NullEmail", &resourceUser{Emails: []*resourceUserEmail{nil}}},
		{"TrailingNullEmail", &resourceUser{Emails: []*resourceUserEmail{{Primary: true, Value: "user@example.com"}, nil}}},
		{"NullPhoto", &resourceUser{Photos: []*resourceUserPhoto{nil}}},
		{"TrailingNullPhoto", &resourceUser{Photos: []*resourceUserPhoto{{Primary: true, Value: "https://example.com/avatar.png"}, nil}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				usr, err := srv.getUserFromSCIM(tc.user, dp)
				assert.Error(t, err)
				assert.Nil(t, usr)
			})
		})
	}
}

func TestSCIMUserNestedEntryValidation(t *testing.T) {
	srv, handler, dp := newSCIMPatchTestServer(t)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("X-Octelium-Session-Uid", dp.Status.SessionRef.Uid)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	bodyFor := func(name, fields string, active bool) string {
		if fields != "" {
			fields = "," + fields
		}
		return fmt.Sprintf(`{"externalId":%q,"userName":%q,"active":%t,"displayName":"Nested User"%s}`, name, name, active, fields)
	}
	w := request(http.MethodPost, srv.getUserPrefix(dp), bodyFor("existing@example.com", `"emails":[{"primary":true,"value":"existing@example.com"}]`, true))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var existing resourceUser
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &existing))
	link, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Uid: existing.ID})
	require.NoError(t, err)
	usr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: link.Status.UserRef.Uid})
	require.NoError(t, err)
	users, err := srv.octeliumC.CoreC().ListUser(t.Context(), &rmetav1.ListOptions{})
	require.NoError(t, err)
	links, err := srv.octeliumC.EnterpriseC().ListDirectoryProviderUser(t.Context(), &rmetav1.ListOptions{})
	require.NoError(t, err)
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for i, tc := range []struct {
			name, fields string
		}{
			{"NullEmail", `"emails":[null]`},
			{"LeadingNullEmail", `"emails":[null,{"primary":true,"value":"candidate@example.com"}]`},
			{"TrailingNullEmail", `"emails":[{"primary":true,"value":"candidate@example.com"},null]`},
			{"NullPhoto", `"photos":[null]`},
			{"LeadingNullPhoto", `"photos":[null,{"primary":true,"value":"https://example.com/avatar.png"}]`},
			{"TrailingNullPhoto", `"photos":[{"primary":true,"value":"https://example.com/avatar.png"},null]`},
			{"NullEmailAndPhoto", `"emails":[null],"photos":[null]`},
			{"InvalidEmailType", `"emails":["invalid"]`},
			{"InvalidPhotoType", `"photos":[123]`},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				path := srv.getUserPrefix(dp)
				if method == http.MethodPut {
					path += "/" + existing.ID
				}
				var w *httptest.ResponseRecorder
				require.NotPanics(t, func() {
					w = request(method, path, bodyFor(fmt.Sprintf("candidate%d@example.com", i), tc.fields, false))
				})
				require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
				assert.Equal(t, "application/scim+json", w.Header().Get("Content-Type"))
				var response responseError
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				assert.Equal(t, "400", response.Status)
				assert.Equal(t, []string{"urn:ietf:params:scim:api:messages:2.0:Error"}, response.Schemas)
				assert.NotEmpty(t, response.Detail)
				updatedUsr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: usr.Metadata.Uid})
				require.NoError(t, err)
				assert.True(t, proto.Equal(usr, updatedUsr))
				updatedLink, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Uid: link.Metadata.Uid})
				require.NoError(t, err)
				assert.True(t, proto.Equal(link, updatedLink))
				updatedUsers, err := srv.octeliumC.CoreC().ListUser(t.Context(), &rmetav1.ListOptions{})
				require.NoError(t, err)
				assert.Len(t, updatedUsers.Items, len(users.Items))
				updatedLinks, err := srv.octeliumC.EnterpriseC().ListDirectoryProviderUser(t.Context(), &rmetav1.ListOptions{})
				require.NoError(t, err)
				assert.Len(t, updatedLinks.Items, len(links.Items))
			})
		}
	}

	for i, tc := range []struct {
		name, fields, email, photo string
	}{
		{name: "Omitted"},
		{name: "NullArrays", fields: `"emails":null,"photos":null`},
		{name: "EmptyArrays", fields: `"emails":[],"photos":[]`},
		{
			name:   "ValidEntries",
			fields: `"emails":[{"value":"secondary@example.com"},{"primary":true,"value":"profile@example.com"}],"photos":[{"value":"https://example.com/secondary.png","type":"thumbnail"},{"primary":true,"type":"photo","value":"https://example.com/avatar.png"}]`,
			email:  "profile@example.com",
			photo:  "https://example.com/avatar.png",
		},
	} {
		t.Run("Valid/"+tc.name, func(t *testing.T) {
			body := bodyFor(fmt.Sprintf("valid%d@example.com", i), tc.fields, false)
			w := request(http.MethodPost, srv.getUserPrefix(dp), body)
			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			var response resourceUser
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			link, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Uid: response.ID})
			require.NoError(t, err)
			usr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: link.Status.UserRef.Uid})
			require.NoError(t, err)
			assert.Equal(t, tc.email, usr.Spec.Email)
			assert.Equal(t, tc.photo, usr.Metadata.PicURL)
			assert.True(t, usr.Spec.IsDisabled)
			body = bodyFor(fmt.Sprintf("valid%d@example.com", i), tc.fields, true)
			w = request(http.MethodPut, srv.getUserPrefix(dp)+"/"+response.ID, body)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			updatedUsr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: usr.Metadata.Uid})
			require.NoError(t, err)
			assert.False(t, updatedUsr.Spec.IsDisabled)
			assert.Equal(t, tc.email, updatedUsr.Spec.Email)
			assert.Equal(t, tc.photo, updatedUsr.Metadata.PicURL)
		})
	}
}

func TestSCIMStoredUserNullEntriesReturnErrors(t *testing.T) {
	for _, field := range []string{"emails", "photos"} {
		t.Run(field, func(t *testing.T) {
			srv, handler, dp := newSCIMPatchTestServer(t)
			request := func(method, path, body string) *httptest.ResponseRecorder {
				r := httptest.NewRequest(method, path, strings.NewReader(body))
				r.Header.Set("X-Octelium-Session-Uid", dp.Status.SessionRef.Uid)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				return w
			}
			body := `{"externalId":"stored-user","userName":"stored@example.com","active":true,"emails":[{"primary":true,"value":"stored@example.com","type":"work"}],"photos":[{"primary":true,"value":"https://example.com/avatar.png"}]}`
			w := request(http.MethodPost, srv.getUserPrefix(dp), body)
			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			var response resourceUser
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			link, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Uid: response.ID})
			require.NoError(t, err)
			usr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: link.Status.UserRef.Uid})
			require.NoError(t, err)
			attrs, err := pbutils.ConvertToMap(link.Status.Attrs)
			require.NoError(t, err)
			attrs[field] = append(attrs[field].([]any), nil)
			link.Status.Attrs, err = pbutils.MapToStruct(attrs)
			require.NoError(t, err)
			link, err = srv.octeliumC.EnterpriseC().UpdateDirectoryProviderUser(t.Context(), link)
			require.NoError(t, err)
			path := srv.getUserPrefix(dp) + "/" + response.ID
			require.NotPanics(t, func() {
				w = request(http.MethodPatch, path, `{"Operations":[{"op":"Replace","path":"emails[type eq \"work\"].value","value":"changed@example.com"}]}`)
			})
			require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
			var errorResponse responseError
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errorResponse))
			assert.Equal(t, "500", errorResponse.Status)
			assert.Equal(t, []string{"urn:ietf:params:scim:api:messages:2.0:Error"}, errorResponse.Schemas)
			assert.Equal(t, "Internal error", errorResponse.Detail)
			updatedUsr, err := srv.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: usr.Metadata.Uid})
			require.NoError(t, err)
			assert.True(t, proto.Equal(usr, updatedUsr))
			updatedLink, err := srv.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Uid: link.Metadata.Uid})
			require.NoError(t, err)
			assert.True(t, proto.Equal(link, updatedLink))
			w = request(http.MethodPut, path, body)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			w = request(http.MethodGet, path, "")
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		})
	}
}
