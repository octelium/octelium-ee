// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package webhook

import (
	"context"
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
		url:           "https://example.com/hook",
		signingSecret: []byte("outbound-secret"),
		inboundSecret: []byte("inbound-secret"),
		hc:            &http.Client{},
	}
}

func tstRequest(p *Provider, now time.Time, body string) *accessintg.InboundRequest {
	timestamp := fmt.Sprintf("%d", now.Unix())

	return &accessintg.InboundRequest{
		Method: http.MethodPost,
		Header: http.Header{
			"Content-Type":         []string{"application/json"},
			"X-Octelium-Timestamp": []string{timestamp},
			"X-Octelium-Signature": []string{sign(p.inboundSecret, timestamp, []byte(body))},
		},
		Body: []byte(body),
		Now:  now,
	}
}

func TestDecodeSetReviewDecision(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, time.Now(), `{"action":"setReviewDecision","externalActorID":"actor-1",`+
		`"eventID":"ev-1","bindingName":"b0123","decision":"APPROVE","justification":"ok"}`)

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, accessintg.ActionTypeSetReviewDecision, ret.Action.Type)
	assert.Equal(t, "actor-1", ret.Action.ExternalActorID)
	assert.Equal(t, "ev-1", ret.Action.ExternalEventID)
	assert.Equal(t, "b0123", ret.Action.BindingName)
	assert.Equal(t, accessv1.Review_Spec_DECISION_APPROVE, ret.Action.Decision)
	assert.Equal(t, "ok", ret.Action.Justification)
}

func TestDecodeCreateRequest(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, time.Now(), `{"action":"createRequest","externalActorID":"actor-1",`+
		`"resource":"cat:prod","duration":"30m","urgency":"VERY_HIGH"}`)

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, accessintg.ActionTypeCreateRequest, ret.Action.Type)
	assert.Equal(t, "cat:prod", ret.Action.ResourceQuery)
	assert.Equal(t, 30*time.Minute, ret.Action.Duration)
	assert.Equal(t, accessv1.Request_Spec_VERY_HIGH, ret.Action.Urgency)
}

func TestDecodeInvalidSignature(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, time.Now(), `{"action":"createRequest","externalActorID":"actor-1"}`)
	in.Header.Set("X-Octelium-Signature", "sha256=deadbeef")

	_, err := p.DecodeInbound(ctx, in)
	assert.NotNil(t, err)
}

func TestDecodeStaleTimestamp(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, time.Now().Add(-time.Hour),
		`{"action":"createRequest","externalActorID":"actor-1"}`)
	in.Now = time.Now()

	_, err := p.DecodeInbound(ctx, in)
	assert.NotNil(t, err)
}

func TestDecodeMissingActor(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, time.Now(), `{"action":"createRequest"}`)

	_, err := p.DecodeInbound(ctx, in)
	assert.NotNil(t, err)
}

func TestDecodeUnknownAction(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstRequest(p, time.Now(), `{"action":"other","externalActorID":"actor-1"}`)

	_, err := p.DecodeInbound(ctx, in)
	assert.NotNil(t, err)
}

func TestCapabilitiesWithoutInboundSecret(t *testing.T) {
	spec := &accessv1.Integration_Spec_Webhook{
		Url: "https://example.com/hook",
	}

	capabilities := Capabilities(spec)
	assert.Contains(t, capabilities, accessv1.Integration_Status_NOTIFICATION)
	assert.NotContains(t, capabilities, accessv1.Integration_Status_INTERACTIVE_REVIEW)
	assert.NotContains(t, capabilities, accessv1.Integration_Status_REQUEST_CREATION)
}
