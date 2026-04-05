// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package dirsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/enterprise"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/middlewares"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	handler, err := srv.getHTTPHandler(ctx)
	assert.Nil(t, err)

	httpSrv := &http.Server{
		Handler:      handler,
		Addr:         fmt.Sprintf("localhost:%d", tests.GetPort()),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return middlewares.SetCtxRequestContext(ctx, &middlewares.RequestContext{})
		},
	}
	go func() {
		httpSrv.ListenAndServe()
	}()

	time.Sleep(1 * time.Second)

	apiSrv := enterprise.NewServer(srv.octeliumC)

	dp, err := apiSrv.CreateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{
				Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
			},
		},
	})
	assert.Nil(t, err)

	resp, err := apiSrv.GenerateDirectoryProviderCredential(ctx, &enterprisev1.GenerateDirectoryProviderCredentialRequest{
		DirectoryProviderRef: umetav1.GetObjectReference(dp),
		Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp.GetBearer())

	dp, err = srv.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
		Uid: dp.Metadata.Uid,
	})
	assert.Nil(t, err)

	client := resty.New().SetHeader("X-Octelium-Session-Uid", dp.Status.SessionRef.Uid)

	{
		r, err := client.R().SetBody(&resourceUser{
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
		}).Post(fmt.Sprintf("http://%s/scim/%s/Users", httpSrv.Addr, dp.Status.Id))
		assert.Nil(t, err, "%+v", err)
		assert.True(t, r.IsSuccess())
	}

	{
		r, err := client.R().SetBody(&resourceGroup{
			resourceCommon: resourceCommon{
				ExternalID: utilrand.GetRandomString(12),
				Schemas:    []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
				Meta: resourceMeta{
					ResourceType: "Group",
				},
			},
			DisplayName: utilrand.GetRandomStringCanonical(8),
		}).Post(fmt.Sprintf("http://%s/scim/%s/Groups", httpSrv.Addr, dp.Status.Id))
		assert.Nil(t, err, "%+v", err)
		assert.True(t, r.IsSuccess())
	}

	{
		r, err := client.R().Get(fmt.Sprintf("http://%s/scim/%s/ServiceProviderConfig", httpSrv.Addr, dp.Status.Id))
		assert.Nil(t, err, "%+v", err)
		assert.True(t, r.IsSuccess())

		resp := &serviceProviderConfig{}
		err = json.Unmarshal(r.Body(), resp)
		assert.Nil(t, err)
		assert.True(t, resp.Patch.Supported)
	}

	{
		r, err := client.R().Get(fmt.Sprintf("http://%s/scim/%s/ResourceTypes", httpSrv.Addr, dp.Status.Id))
		assert.Nil(t, err, "%+v", err)
		assert.True(t, r.IsSuccess())

		resp := &responseResourceTypes{}
		err = json.Unmarshal(r.Body(), resp)
		assert.Nil(t, err)
		assert.Equal(t, 2, len(resp.Resources))
		assert.Equal(t, "User", resp.Resources[0].Name)
	}

	{
		r, err := client.R().Get(fmt.Sprintf("http://%s/scim/%s/Schemas", httpSrv.Addr, dp.Status.Id))
		assert.Nil(t, err, "%+v", err)
		assert.True(t, r.IsSuccess())

		resp := &schemasResponse{}
		err = json.Unmarshal(r.Body(), resp)
		assert.Nil(t, err)
		assert.Equal(t, 2, len(resp.Resources))
		assert.Equal(t, "User", resp.Resources[0].Name)
		assert.Equal(t, "userName", resp.Resources[0].Attributes[0].Name)
	}
}
