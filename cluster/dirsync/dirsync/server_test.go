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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/enterprise"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestSCIMQueryPaginationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		query              string
		count, start, page uint32
	}{
		{"", maxItemsPerPage, 1, 0},
		{"?count=0", 0, 1, 0},
		{"?count=-1&startIndex=-1", maxItemsPerPage, 1, 0},
		{"?count=invalid&startIndex=invalid", maxItemsPerPage, 1, 0},
		{"?count=1000000&startIndex=100001", maxItemsPerPage, 1, 0},
		{"?count=2&startIndex=5", 2, 5, 2},
		{"?count=3&startIndex=2", 3, 2, 0},
	} {
		r := httptest.NewRequest(http.MethodGet, "/scim/provider/Users"+tc.query, nil)
		assert.Equal(t, tc.count, getItemsPerPage(r))
		assert.Equal(t, tc.count, getSCIMCount(r))
		assert.Equal(t, tc.start, getSCIMStartIndex(r))
		assert.Equal(t, tc.page, getBackendPage(r))
	}
}

func TestSCIMErrorResponsesHideInternalDetails(t *testing.T) {
	for _, tc := range []struct {
		code codes.Code
		want int
	}{
		{codes.NotFound, http.StatusNotFound},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.Canceled, http.StatusBadRequest},
		{codes.DeadlineExceeded, http.StatusBadRequest},
		{codes.Unauthenticated, http.StatusForbidden},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.Internal, http.StatusInternalServerError},
	} {
		w := httptest.NewRecorder()
		(&server{}).setErrorFromAPI(w, status.Error(tc.code, "private database password"))
		assert.Equal(t, tc.want, w.Code)
		var body responseError
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, fmt.Sprint(tc.want), body.Status)
		assert.Equal(t, []string{"urn:ietf:params:scim:api:messages:2.0:Error"}, body.Schemas)
		assert.NotContains(t, body.Detail, "password")
	}
}

func TestSCIMMetadataDocuments(t *testing.T) {
	srv := &server{}
	for _, tc := range []struct {
		name   string
		handle func(http.ResponseWriter, *http.Request)
	}{
		{"ServiceProviderConfig", srv.handleServiceProviderConfig},
		{"Schemas", srv.handleSchemas},
		{"ResourceTypes", srv.handleResourceTypes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handle(w, httptest.NewRequest(http.MethodGet, "/scim/provider/"+tc.name, nil))
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/scim+json", w.Header().Get("Content-Type"))
			var doc map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))
			assert.NotEmpty(t, doc)
		})
	}
}

func TestDirectorySynchronizationHistoryBounded(t *testing.T) {
	var history []*enterprisev1.DirectoryProvider_Status_Synchronization
	for range maxLastSynchronizations + 3 {
		rec := &enterprisev1.DirectoryProvider_Status_Synchronization{State: enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS}
		history = appendLastSync(history, rec)
		assert.NotSame(t, rec, history[0])
		rec.State = enterprisev1.DirectoryProvider_Status_Synchronization_FAILED
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS, history[0].State)
	}
	assert.Len(t, history, maxLastSynchronizations)
}
