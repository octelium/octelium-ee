// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package publicserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: vutils.GetMyRegionName(),
	})
	assert.Nil(t, err)

	_, err = srv.octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name:           ovutils.GetOIDCConfigSecretName(rgn.Metadata.Name),
			IsSystem:       true,
			IsSystemHidden: true,
		},
		Spec:   &enterprisev1.Secret_Spec{},
		Status: &enterprisev1.Secret_Status{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: `
{"issuer":"https://kubernetes.default.svc.cluster.local","jwks_uri":"https://135.181.29.238:6443/openid/v1/jwks","response_types_supported":["id_token"],"subject_types_supported":["public"],"id_token_signing_alg_values_supported":["RS256"]}
`,
			},
		},
	})
	assert.Nil(t, err)

	_, err = srv.octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: ovutils.GetOIDC_JWKSSecretName(rgn.Metadata.Name),
		},
		Spec:   &enterprisev1.Secret_Spec{},
		Status: &enterprisev1.Secret_Status{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: `
{"keys":[{"use":"sig","kty":"RSA","kid":"v3sfENI7ouiABNhjP5dlrf7nPHfp_w88e6_a4jXLvIc","alg":"RS256","n":"yvSlNwUG5nSeMMMDBsVGO-iZgylO-E2F2yHJQv3suV-kEOy7syqqYCr5eyU4YnGKFo5huPXEc51WscHTDbjNaUKYRhUU7l6GwZqcWTLTojIfxn58h2T-WrGZ0Vmfn7qj2ghESoguomEevOelZCrc1pSkooKf_vKdWG1JHPA-8l__i_x7plPoPYX9-azx_Nd4TgbzXGPSHZBTQEJn6kMbgLknBBGu6PuwZ5a_DA8SKjKZO-yxQHbe8m6_NCSB9JH--khOl9CimokjNIBs8qdYgEUUDg2E3O5uCBoCGbEOSgoeEr_2QrArjDHvJjR-sEv2OKteZaQhYEkSnTEMJ6AGSQ","e":"AQAB"}]}
`,
			},
		},
	})
	assert.Nil(t, err)

	err = srv.setupK8sOIDC(ctx)
	assert.Nil(t, err)

	client := resty.New()
	r, err := client.R().Get(fmt.Sprintf("http://%s/.well-known/regions/%s/openid-configuration", httpSrv.Addr, rgn.Metadata.Uid))
	assert.Nil(t, err, "%+v", err)
	assert.True(t, r.IsSuccess(), "%d", r.StatusCode())

	oidcCfg := &oidcProviderJSON{}
	err = json.Unmarshal(r.Body(), oidcCfg)
	assert.Nil(t, err)

	assert.Equal(t, fmt.Sprintf("%s/.well-known/regions/%s/jwks", srv.rootURL, rgn.Metadata.Uid), oidcCfg.JWKSURL)

	{

		r, err := client.R().Get(fmt.Sprintf("http://%s/.well-known/regions/%s/jwks", httpSrv.Addr, rgn.Metadata.Uid))
		assert.Nil(t, err, "%+v", err)
		assert.True(t, r.IsSuccess(), "%d", r.StatusCode())

		jwks := &jose.JSONWebKeySet{}

		err = json.Unmarshal(r.Body(), jwks)
		assert.Nil(t, err)

		assert.True(t, len(jwks.Keys) > 0)
	}
}

func TestOIDCProviderValidationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name  string
		info  *oidcProviderJSON
		valid bool
	}{
		{name: "Nil"},
		{name: "Empty", info: &oidcProviderJSON{}},
		{name: "Malformed", info: &oidcProviderJSON{Issuer: "https://%zz"}},
		{name: "Insecure", info: &oidcProviderJSON{Issuer: "http://issuer.example.com"}},
		{name: "MissingHost", info: &oidcProviderJSON{Issuer: "https:///issuer"}},
		{name: "Relative", info: &oidcProviderJSON{Issuer: "/issuer"}},
		{name: "HTTPS", info: &oidcProviderJSON{Issuer: "https://issuer.example.com"}, valid: true},
		{
			name:  "PathAndPort",
			info:  &oidcProviderJSON{Issuer: "https://issuer.example.com:8443/cluster"},
			valid: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOIDCProviderJSON(tc.info)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestJWKSValidationBoundaries(t *testing.T) {
	for _, value := range []string{"", " ", "null", "{}", "[]", `{"keys":[]}`, `{"keys":null}`, `{"keys":"invalid"}`, `{"keys":[`} {
		assert.Error(t, validateJWKS([]byte(value)), value)
	}
	assert.NoError(t, validateJWKS([]byte(`{"keys":[{"kty":"RSA","kid":"key","e":"AQAB","n":"AQAB"}]}`)))
}

func TestWellKnownRoutingAndMethods(t *testing.T) {
	uid := vutils.UUIDv4()
	info := &oidcInfo{oidc: []byte(`{"issuer":"https://issuer.example.com"}`), jwks: []byte(`{"keys":[{"kid":"one"}]}`)}
	srv := &Server{}
	srv.oidcInfo.Store(map[string]*oidcInfo{uid: info})
	for _, tc := range []struct {
		method string
		path   string
		code   int
		body   string
	}{
		{http.MethodGet, "/.well-known/regions/" + uid + "/openid-configuration", http.StatusOK, string(info.oidc)},
		{http.MethodGet, "/.well-known/regions/" + uid + "/jwks", http.StatusOK, string(info.jwks)},
		{http.MethodHead, "/.well-known/regions/" + uid + "/openid-configuration", http.StatusOK, ""},
		{http.MethodHead, "/.well-known/regions/" + uid + "/jwks", http.StatusOK, ""},
		{http.MethodPost, "/.well-known/regions/" + uid + "/jwks", http.StatusMethodNotAllowed, ""},
		{http.MethodDelete, "/.well-known/regions/" + uid + "/jwks", http.StatusMethodNotAllowed, ""},
		{http.MethodGet, "/.well-known", http.StatusNotFound, ""},
		{http.MethodGet, "/.well-known/regions/not-a-uuid/jwks", http.StatusNotFound, ""},
		{http.MethodGet, "/.well-known/regions/" + vutils.UUIDv4() + "/jwks", http.StatusNotFound, ""},
		{http.MethodGet, "/.well-known/region/" + uid + "/jwks", http.StatusNotFound, ""},
		{http.MethodGet, "/.well-known/regions/" + uid + "/unknown", http.StatusNotFound, ""},
		{http.MethodGet, "/.well-known/regions/" + uid + "/jwks/extra", http.StatusNotFound, ""},
		{http.MethodGet, "/unknown", http.StatusNotFound, ""},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			assert.Equal(t, tc.code, w.Code)
			if tc.code == http.StatusOK {
				assert.Equal(t, tc.body, w.Body.String())
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Equal(t, "public, max-age=300", w.Header().Get("Cache-Control"))
			}
			if tc.code == http.StatusMethodNotAllowed {
				assert.Equal(t, "GET, HEAD", w.Header().Get("Allow"))
			}
		})
	}
	_, ok := (&Server{}).getOIDCInfo(uid)
	assert.False(t, ok)
}

func TestOIDCRefreshPreservesLastGoodSnapshot(t *testing.T) {
	ctx := t.Context()
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	srv, err := newServer(ctx, tst.C.OcteliumC)
	require.NoError(t, err)
	t.Cleanup(srv.integrationSrv.Close)
	region, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: "default"})
	require.NoError(t, err)
	createSecret := func(name, value string) *enterprisev1.Secret {
		sec, err := srv.octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
			Metadata: &metav1.Metadata{Name: name},
			Spec:     &enterprisev1.Secret_Spec{},
			Data:     &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: value}},
		})
		require.NoError(t, err)
		return sec
	}
	config := createSecret(ovutils.GetOIDCConfigSecretName(region.Metadata.Name), `{"issuer":"https://issuer.example.com","jwks_uri":"https://private.example.com/keys"}`)
	keys := createSecret(ovutils.GetOIDC_JWKSSecretName(region.Metadata.Name), `{"keys":[{"kty":"RSA","kid":"one","e":"AQAB","n":"AQAB"}]}`)
	require.NoError(t, srv.setupK8sOIDC(ctx))
	original, ok := srv.getOIDCInfo(region.Metadata.Uid)
	require.True(t, ok)
	var document oidcProviderJSON
	require.NoError(t, json.Unmarshal(original.oidc, &document))
	assert.Equal(t, srv.rootURL+"/.well-known/regions/"+region.Metadata.Uid+"/jwks", document.JWKSURL)
	for _, invalid := range []string{"{", `{"issuer":"http://insecure.example.com"}`} {
		config.Data = &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: invalid}}
		config, err = srv.octeliumC.EnterpriseC().UpdateSecret(ctx, config)
		require.NoError(t, err)
		assert.Error(t, srv.setupK8sOIDC(ctx))
		current, ok := srv.getOIDCInfo(region.Metadata.Uid)
		require.True(t, ok)
		assert.Same(t, original, current)
	}
	config.Data = &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: `{"issuer":"https://issuer.example.com"}`}}
	_, err = srv.octeliumC.EnterpriseC().UpdateSecret(ctx, config)
	require.NoError(t, err)
	_, err = srv.octeliumC.EnterpriseC().DeleteSecret(ctx, &rmetav1.DeleteOptions{Uid: keys.Metadata.Uid})
	require.NoError(t, err)
	require.NoError(t, srv.setupK8sOIDC(ctx))
	_, ok = srv.getOIDCInfo(region.Metadata.Uid)
	assert.False(t, ok)
	createSecret(keys.Metadata.Name, `{"keys":[{"kty":"RSA","kid":"two","e":"AQAB","n":"AQAB"}]}`)
	require.NoError(t, srv.setupK8sOIDC(ctx))
	current, ok := srv.getOIDCInfo(region.Metadata.Uid)
	require.True(t, ok)
	assert.Contains(t, string(current.jwks), `"two"`)
}

func TestOIDCRefreshLoopCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		(&Server{}).startOIDCRefreshLoop(ctx)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("OIDC refresh loop did not stop")
	}
}
