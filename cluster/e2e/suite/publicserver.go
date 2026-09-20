// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const unknownRegionUID = "3f3d9cbe-1f0f-4b37-9c4f-5d5c2bb9f0a1"

type oidcDiscovery struct {
	Issuer  string `json:"issuer"`
	JWKSURL string `json:"jwks_uri"`
}

type oidcJWKS struct {
	Keys []json.RawMessage `json:"keys"`
}

func publicServerURL(h *eeharness.H) string {
	return fmt.Sprintf("https://public.octelium.%s", h.Domain)
}

func publicServerClient(h *eeharness.H) *resty.Client {
	return h.HTTP().SetBaseURL(publicServerURL(h))
}

func regionDocPath(regionUID, doc string) string {
	return fmt.Sprintf("/.well-known/regions/%s/%s", regionUID, doc)
}

func testPublicServerOIDC(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	rgn, err := h.CoreC().GetRegion(ctx, &metav1.GetOptions{Name: "default"})
	require.Nil(t, err)

	c := publicServerClient(h)

	var discovery oidcDiscovery

	t.Run("TheDiscoveryIsServedAnonymously", func(t *testing.T) {
		h.Eventually(t, "the publicserver to serve the Region OIDC discovery",
			eeharness.PropagationBudget, func(ctx context.Context) error {
				res, err := c.R().SetContext(ctx).
					Get(regionDocPath(rgn.Metadata.Uid, "openid-configuration"))
				if err != nil {
					return err
				}
				if res.StatusCode() != http.StatusOK {
					return errUnexpectedStatus(res.StatusCode(), http.StatusOK)
				}

				cur := oidcDiscovery{}
				if err := json.Unmarshal(res.Body(), &cur); err != nil {
					return errors.Wrap(err, "could not decode the OIDC discovery document")
				}
				if cur.Issuer == "" {
					return errors.Errorf("the OIDC discovery document carries no issuer")
				}
				if cur.JWKSURL == "" {
					return errors.Errorf("the OIDC discovery document carries no jwks_uri")
				}

				discovery = cur
				return nil
			})

		issuer, err := url.Parse(discovery.Issuer)
		require.Nil(t, err)
		assert.Equal(t, "https", issuer.Scheme)
		assert.NotEmpty(t, issuer.Host)
	})

	t.Run("TheJWKSURLPointsAtThisRegion", func(t *testing.T) {
		require.NotEmpty(t, discovery.JWKSURL)
		assert.Equal(t,
			publicServerURL(h)+regionDocPath(rgn.Metadata.Uid, "jwks"),
			discovery.JWKSURL)
	})

	t.Run("TheAdvertisedJWKSIsServed", func(t *testing.T) {
		require.NotEmpty(t, discovery.JWKSURL)

		res, err := h.HTTP().R().SetContext(ctx).Get(discovery.JWKSURL)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode())
		assert.True(t,
			strings.HasPrefix(res.Header().Get("Content-Type"), "application/json"))

		jwks := oidcJWKS{}
		require.Nil(t, json.Unmarshal(res.Body(), &jwks))
		assert.NotEmpty(t, jwks.Keys)
	})

	t.Run("AHeadRequestCarriesNoBody", func(t *testing.T) {
		res, err := c.R().SetContext(ctx).
			Head(regionDocPath(rgn.Metadata.Uid, "openid-configuration"))
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode())
		assert.Empty(t, res.Body())
	})

	t.Run("AnUnknownRegionIsNotFound", func(t *testing.T) {
		res, err := c.R().SetContext(ctx).
			Get(regionDocPath(utilrand.GetRandomStringCanonical(8), "openid-configuration"))
		require.Nil(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode())

		res, err = c.R().SetContext(ctx).
			Get(regionDocPath(unknownRegionUID, "openid-configuration"))
		require.Nil(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode())
	})

	t.Run("AnUnknownDocumentIsNotFound", func(t *testing.T) {
		res, err := c.R().SetContext(ctx).Get(regionDocPath(rgn.Metadata.Uid, "keys"))
		require.Nil(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode())

		res, err = c.R().SetContext(ctx).Get("/.well-known/regions")
		require.Nil(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode())
	})

	t.Run("AWriteMethodIsRefused", func(t *testing.T) {
		res, err := c.R().SetContext(ctx).
			Post(regionDocPath(rgn.Metadata.Uid, "openid-configuration"))
		require.Nil(t, err)
		assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode())
		assert.Contains(t, res.Header().Get("Allow"), http.MethodGet)
	})

	t.Run("AnUnroutedPathIsNotFound", func(t *testing.T) {
		res, err := c.R().SetContext(ctx).Get("/" + utilrand.GetRandomStringCanonical(8))
		require.Nil(t, err)
		assert.Equal(t, http.StatusNotFound, res.StatusCode())
	})

	t.Run("TheDiscoverySurvivesARestart", func(t *testing.T) {
		h.RestartService(t, "public.octelium")

		h.Eventually(t, "the publicserver to serve the discovery after the restart",
			eeharness.PropagationBudget, func(ctx context.Context) error {
				res, err := c.R().SetContext(ctx).
					Get(regionDocPath(rgn.Metadata.Uid, "openid-configuration"))
				if err != nil {
					return err
				}
				if res.StatusCode() != http.StatusOK {
					return errUnexpectedStatus(res.StatusCode(), http.StatusOK)
				}

				cur := oidcDiscovery{}
				if err := json.Unmarshal(res.Body(), &cur); err != nil {
					return err
				}
				if cur.Issuer != discovery.Issuer || cur.JWKSURL != discovery.JWKSURL {
					return errors.Errorf("the discovery document changed after the restart")
				}
				return nil
			})
	})
}
