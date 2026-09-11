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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestHandleIntegration(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	srv, err := newServer(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(srv.integrationSrv.Close)

	signingSecret := utilrand.GetRandomString(32)

	sec, err := srv.octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: signingSecret,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	botSec, err := srv.octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	integration, err := srv.octeliumC.AccessC().CreateIntegration(ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Integration_Spec{
			Type: &accessv1.Integration_Spec_Slack_{
				Slack: &accessv1.Integration_Spec_Slack{
					BotToken: &accessv1.Integration_Spec_Slack_BotToken{
						Type: &accessv1.Integration_Spec_Slack_BotToken_FromSecret{
							FromSecret: botSec.Metadata.Name,
						},
					},
					SigningSecret: &accessv1.Integration_Spec_Slack_SigningSecret{
						Type: &accessv1.Integration_Spec_Slack_SigningSecret_FromSecret{
							FromSecret: sec.Metadata.Name,
						},
					},
				},
			},
		},
		Status: &accessv1.Integration_Status{
			Id:   utilrand.GetRandomStringCanonical(24),
			Type: accessv1.Integration_Status_SLACK,
			Capabilities: []accessv1.Integration_Status_Capability{
				accessv1.Integration_Status_INTERACTIVE_REVIEW,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	doRequest := func(path string, body []byte, sign bool,
		contentType string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(body)))
		r.Header.Set("Content-Type", contentType)

		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		r.Header.Set("X-Slack-Request-Timestamp", timestamp)

		signature := "v0=deadbeef"
		if sign {
			mac := hmac.New(sha256.New, []byte(signingSecret))
			mac.Write([]byte("v0:"))
			mac.Write([]byte(timestamp))
			mac.Write([]byte(":"))
			mac.Write(body)
			signature = fmt.Sprintf("v0=%s", hex.EncodeToString(mac.Sum(nil)))
		}
		r.Header.Set("X-Slack-Signature", signature)

		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)

		return w
	}

	basePath := fmt.Sprintf("%s%s", integrationPathPrefix, integration.Status.Id)

	{
		body := []byte(`{"type":"url_verification","challenge":"abc123"}`)
		w := doRequest(fmt.Sprintf("%s/slack/events", basePath), body, true, "application/json")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "abc123", w.Body.String())
	}

	{
		body := []byte(`{"type":"url_verification","challenge":"abc123"}`)
		w := doRequest(fmt.Sprintf("%s/slack/events", basePath), body, false, "application/json")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	{
		body := []byte(url.Values{"payload": []string{`{"type":"view_closed"}`}}.Encode())
		w := doRequest(fmt.Sprintf("%s/slack/interactions", basePath), body, true,
			"application/x-www-form-urlencoded")
		assert.Equal(t, http.StatusOK, w.Code)
	}

	{
		body := []byte(`{"type":"url_verification","challenge":"abc123"}`)
		w := doRequest(fmt.Sprintf("%s%s/slack/events", integrationPathPrefix,
			utilrand.GetRandomStringCanonical(24)), body, true, "application/json")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	{
		r := httptest.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/slack/events", basePath), nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	}

	{
		broken, err := srv.octeliumC.AccessC().CreateIntegration(ctx, &accessv1.Integration{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.Integration_Spec{
				Type: &accessv1.Integration_Spec_Slack_{
					Slack: &accessv1.Integration_Spec_Slack{
						BotToken: &accessv1.Integration_Spec_Slack_BotToken{
							Type: &accessv1.Integration_Spec_Slack_BotToken_FromSecret{
								FromSecret: utilrand.GetRandomStringCanonical(8),
							},
						},
						SigningSecret: &accessv1.Integration_Spec_Slack_SigningSecret{
							Type: &accessv1.Integration_Spec_Slack_SigningSecret_FromSecret{
								FromSecret: sec.Metadata.Name,
							},
						},
					},
				},
			},
			Status: &accessv1.Integration_Status{
				Id:   utilrand.GetRandomStringCanonical(24),
				Type: accessv1.Integration_Status_SLACK,
				Capabilities: []accessv1.Integration_Status_Capability{
					accessv1.Integration_Status_INTERACTIVE_REVIEW,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)

		body := []byte(`{"type":"url_verification","challenge":"abc123"}`)
		w := doRequest(fmt.Sprintf("%s%s/slack/events", integrationPathPrefix,
			broken.Status.Id), body, true, "application/json")
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	}

	{
		integration.Spec.IsDisabled = true
		_, err := srv.octeliumC.AccessC().UpdateIntegration(ctx, integration)
		assert.Nil(t, err, "%+v", err)

		body := []byte(`{"type":"url_verification","challenge":"abc123"}`)
		w := doRequest(fmt.Sprintf("%s/slack/events", basePath), body, true, "application/json")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}
}
