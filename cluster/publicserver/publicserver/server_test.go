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
