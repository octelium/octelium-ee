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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/accessv1"
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

func testPublicServerIntegrationCallback(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	ctx := t.Context()
	createSecret := func(value string) *accessv1.Secret {
		t.Helper()
		secret, err := h.AccessC().CreateSecret(ctx, &accessv1.Secret{
			Metadata: &metav1.Metadata{Name: h.Name()},
			Spec:     &accessv1.Secret_Spec{},
			Data:     &accessv1.Secret_Data{Type: &accessv1.Secret_Data_Value{Value: value}},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_, err := h.AccessC().DeleteSecret(cleanup, &metav1.DeleteOptions{Uid: secret.Metadata.Uid})
			assert.NoError(t, err)
		})
		return secret
	}
	signing := utilrand.GetRandomStringCanonical(32)
	signSecret, botSecret := createSecret(signing), createSecret(utilrand.GetRandomStringCanonical(32))
	integration, err := h.AccessC().CreateIntegration(ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{Name: h.Name()},
		Spec: &accessv1.Integration_Spec{
			Type: &accessv1.Integration_Spec_Slack_{
				Slack: &accessv1.Integration_Spec_Slack{
					SigningSecret: &accessv1.Integration_Spec_Slack_SigningSecret{
						Type: &accessv1.Integration_Spec_Slack_SigningSecret_FromSecret{FromSecret: signSecret.Metadata.Name},
					},
					BotToken: &accessv1.Integration_Spec_Slack_BotToken{Type: &accessv1.Integration_Spec_Slack_BotToken_FromSecret{FromSecret: botSecret.Metadata.Name}},
				},
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, err := h.AccessC().DeleteIntegration(cleanup, &metav1.DeleteOptions{Uid: integration.Metadata.Uid})
		assert.NoError(t, err)
	})
	require.NotEmpty(t, integration.Status.GetId())
	endpoint := publicServerURL(h) + "/integration/v1/callback/" + integration.Status.Id + "/slack/events"
	body := `{"type":"url_verification","challenge":"octelium-e2e"}`
	request := func(ctx context.Context, valid bool) (*resty.Response, error) {
		timestamp := fmt.Sprint(time.Now().Unix())
		mac := hmac.New(sha256.New, []byte(signing))
		mac.Write([]byte("v0:" + timestamp + ":" + body))
		signature := "v0=" + hex.EncodeToString(mac.Sum(nil))
		if !valid {
			signature = "v0=invalid"
		}
		return h.HTTP().R().SetContext(ctx).SetHeader("Content-Type", "application/json").SetHeader("X-Slack-Request-Timestamp", timestamp).SetHeader("X-Slack-Signature", signature).SetBody(body).Post(endpoint)
	}
	h.Eventually(t, "the public callback to validate a signed Slack challenge", eeharness.PropagationBudget, func(ctx context.Context) error {
		res, err := request(ctx, true)
		if err != nil {
			return err
		}
		if res.StatusCode() != http.StatusOK {
			return errUnexpectedStatus(res.StatusCode(), http.StatusOK)
		}
		if res.String() != "octelium-e2e" {
			return errors.Errorf("unexpected callback response")
		}
		return nil
	})
	res, err := request(ctx, false)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode())
	res, err = h.HTTP().R().SetContext(ctx).Get(endpoint)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode())
	assert.Equal(t, "POST", res.Header().Get("Allow"))
	res, err = h.HTTP().R().SetContext(ctx).SetBody(strings.Repeat("x", (1<<20)+1)).Post(endpoint)
	require.NoError(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode())
	integration.Spec.IsDisabled = true
	_, err = h.AccessC().UpdateIntegration(ctx, integration)
	require.NoError(t, err)
	h.Eventually(t, "disabled integrations to reject signed callbacks", eeharness.PropagationBudget, func(ctx context.Context) error {
		res, err := request(ctx, true)
		if err != nil {
			return err
		}
		if res.StatusCode() != http.StatusUnauthorized {
			return errUnexpectedStatus(res.StatusCode(), http.StatusUnauthorized)
		}
		return nil
	})
}

func testPublicServerMetadataHeaders(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	region, err := h.CoreC().GetRegion(t.Context(), &metav1.GetOptions{Name: "default"})
	require.NoError(t, err)
	for _, document := range []string{"openid-configuration", "jwks"} {
		path := regionDocPath(region.Metadata.Uid, document)
		get, err := publicServerClient(h).R().SetContext(t.Context()).Get(path)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, get.StatusCode())
		assert.NotEmpty(t, get.Body())
		assert.Contains(t, get.Header().Get("Cache-Control"), "max-age=300")
		head, err := publicServerClient(h).R().SetContext(t.Context()).Head(path)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, head.StatusCode())
		assert.Empty(t, head.Body())
		assert.Equal(t, get.Header().Get("Content-Type"), head.Header().Get("Content-Type"))
		assert.Equal(t, get.Header().Get("Cache-Control"), head.Header().Get("Cache-Control"))
	}
}
