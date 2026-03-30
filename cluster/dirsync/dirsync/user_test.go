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
	"testing"

	"github.com/gorilla/mux"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/middlewares"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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
			Active:   true,
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
		assert.False(t, res.Active)

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
