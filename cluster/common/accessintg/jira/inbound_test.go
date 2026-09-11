// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package jira

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/stretchr/testify/assert"
)

func tstProvider() *Provider {
	return &Provider{
		api:           newAPIClient("https://example.atlassian.net", "admin@example.com", "token"),
		webhookSecret: []byte("webhook-secret"),
		siteURL:       "https://example.atlassian.net",
	}
}

func tstRequest(p *Provider, body string) *accessintg.InboundRequest {
	mac := hmac.New(sha256.New, p.webhookSecret)
	mac.Write([]byte(body))

	return &accessintg.InboundRequest{
		Method: http.MethodPost,
		Path:   []string{"webhook"},
		Header: http.Header{
			"Content-Type":    []string{"application/json"},
			"X-Hub-Signature": []string{fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))},
		},
		Body: []byte(body),
		Now:  time.Now(),
	}
}

func TestDecodeIssueUpdated(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, `{"webhookEvent":"jira:issue_updated","timestamp":1700000000,`+
		`"issue":{"id":"1","key":"OPS-12"},"user":{"accountId":"acc-1"},`+
		`"changelog":{"id":"99","items":[{"field":"status","toString":"Approved"}]}}`)

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.NotNil(t, ret.Action)
	assert.Equal(t, accessintg.ActionTypeSetReviewDecision, ret.Action.Type)
	assert.Equal(t, "acc-1", ret.Action.ExternalActorID)
	assert.Equal(t, "OPS-12", ret.Action.ExternalObjectID)
	assert.Equal(t, "OPS-12.99", ret.Action.ExternalEventID)
	assert.Equal(t, accessv1.Review_Spec_DECISION_UNSET, ret.Action.Decision)
}

func TestDecodeInvalidSignature(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, `{"webhookEvent":"jira:issue_updated"}`)
	in.Header.Set("X-Hub-Signature", "sha256=deadbeef")

	_, err := p.DecodeInbound(ctx, in)
	assert.NotNil(t, err)
}

func TestDecodeWithoutStatusChange(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, `{"webhookEvent":"jira:issue_updated","issue":{"key":"OPS-12"},`+
		`"user":{"accountId":"acc-1"},`+
		`"changelog":{"id":"99","items":[{"field":"summary","toString":"x"}]}}`)

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, ret.Action)
	assert.Equal(t, http.StatusOK, ret.Response.StatusCode)
}

func TestDecodeUnrelatedEvent(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, `{"webhookEvent":"jira:issue_created","issue":{"key":"OPS-12"}}`)

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, ret.Action)
}

func TestCapabilitiesWithoutWebhookSecret(t *testing.T) {
	spec := &accessv1.Integration_Spec_Jira{
		Url: "https://example.atlassian.net",
	}

	capabilities := Capabilities(spec)
	assert.Contains(t, capabilities, accessv1.Integration_Status_NOTIFICATION)
	assert.NotContains(t, capabilities, accessv1.Integration_Status_INTERACTIVE_REVIEW)
}
