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
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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

}
