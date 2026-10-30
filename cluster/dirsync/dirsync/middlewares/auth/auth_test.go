// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/enterprise"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/dirsync/dirsync/middlewares"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSCIMAuthFixture(t *testing.T) (*middleware, *enterprisev1.DirectoryProvider, context.Context) {
	t.Helper()
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	api := enterprise.NewServer(tst.C.OcteliumC)
	dp, err := api.CreateDirectoryProvider(t.Context(), &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &enterprisev1.DirectoryProvider_Spec{Type: &enterprisev1.DirectoryProvider_Spec_Scim{Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{}}},
	})
	require.NoError(t, err)
	_, err = api.GenerateDirectoryProviderCredential(t.Context(), &enterprisev1.GenerateDirectoryProviderCredentialRequest{
		DirectoryProviderRef: umetav1.GetObjectReference(dp),
		Mode:                 enterprisev1.GenerateDirectoryProviderCredentialRequest_BEARER,
	})
	require.NoError(t, err)
	dp, err = tst.C.OcteliumC.EnterpriseC().GetDirectoryProvider(t.Context(), &rmetav1.GetOptions{Uid: dp.Metadata.Uid})
	require.NoError(t, err)
	ctx := middlewares.SetCtxRequestContext(t.Context(), &middlewares.RequestContext{})
	return &middleware{octeliumC: tst.C.OcteliumC}, dp, ctx
}

func TestSCIMAuthenticationProviderBinding(t *testing.T) {
	m, dp, ctx := newSCIMAuthFixture(t)
	called := false
	m.next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		req := middlewares.GetCtxRequestContext(r.Context())
		assert.Equal(t, dp.Metadata.Uid, req.DirectoryProvider.Metadata.Uid)
		assert.Equal(t, dp.Status.SessionRef.Uid, req.Session.Metadata.Uid)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, tc := range []struct {
		session, path string
		want          int
	}{
		{dp.Status.SessionRef.Uid, "/scim/" + dp.Status.Id + "/Users", http.StatusNoContent},
		{"", "/scim/" + dp.Status.Id + "/Users", http.StatusUnauthorized},
		{vutils.UUIDv4(), "/scim/" + dp.Status.Id + "/Users", http.StatusUnauthorized},
		{dp.Status.SessionRef.Uid, "/scim/wrong-provider/Users", http.StatusUnauthorized},
		{dp.Status.SessionRef.Uid, "/wrong/" + dp.Status.Id + "/Users", http.StatusUnauthorized},
		{dp.Status.SessionRef.Uid, "/", http.StatusUnauthorized},
	} {
		called = false
		r := httptest.NewRequest(http.MethodGet, tc.path, nil).WithContext(ctx)
		r.Header.Set("X-Octelium-Session-Uid", tc.session)
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)
		assert.Equal(t, tc.want, w.Code)
		assert.Equal(t, tc.want == http.StatusNoContent, called)
	}
	dp.Spec.IsDisabled = true
	_, err := m.octeliumC.EnterpriseC().UpdateDirectoryProvider(t.Context(), dp)
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodGet, "/scim/"+dp.Status.Id+"/Users", nil).WithContext(ctx)
	r.Header.Set("X-Octelium-Session-Uid", dp.Status.SessionRef.Uid)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSCIMAuthenticationRejectsWrongSessionType(t *testing.T) {
	m, dp, ctx := newSCIMAuthFixture(t)
	session, err := m.octeliumC.CoreC().GetSession(t.Context(), &rmetav1.GetOptions{Uid: dp.Status.SessionRef.Uid})
	require.NoError(t, err)
	session.Status.Type = corev1.Session_Status_CLIENT
	_, err = m.octeliumC.CoreC().UpdateSession(t.Context(), session)
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodGet, "/scim/"+dp.Status.Id+"/Users", nil).WithContext(ctx)
	r.Header.Set("X-Octelium-Session-Uid", session.Metadata.Uid)
	_, err = m.getContext(r)
	assert.ErrorContains(t, err, "CLIENTLESS")
	handler, err := New(t.Context(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected downstream handler") }), m.octeliumC)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSCIMAuthenticationRejectsShortPaths(t *testing.T) {
	m, dp, ctx := newSCIMAuthFixture(t)
	called := false
	m.next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	for _, path := range []string{"/", "//", "///", "/scim", "/scim/", "/scim//", "///scim///", "/wrong/"} {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx)
			r.Header.Set("X-Octelium-Session-Uid", dp.Status.SessionRef.Uid)
			w := httptest.NewRecorder()
			assert.NotPanics(t, func() { m.ServeHTTP(w, r) })
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.False(t, called)
		})
	}

	r := httptest.NewRequest(http.MethodGet, "/scim/"+dp.Status.Id+"/Users", nil).WithContext(ctx)
	r.Header.Set("X-Octelium-Session-Uid", dp.Status.SessionRef.Uid)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, called)
}
