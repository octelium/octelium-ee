// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package slack

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/stretchr/testify/assert"
)

func tstProvider() *Provider {
	return &Provider{
		api:           newAPIClient("", "xoxb-token"),
		signingSecret: []byte("signing-secret"),
		teamID:        "T00000001",
	}
}

func tstSign(p *Provider, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, p.signingSecret)
	mac.Write([]byte("v0:"))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(":"))
	mac.Write(body)

	return fmt.Sprintf("v0=%s", hex.EncodeToString(mac.Sum(nil)))
}

func tstFormRequest(p *Provider, path string, now time.Time, form url.Values) *accessintg.InboundRequest {
	body := []byte(form.Encode())
	timestamp := fmt.Sprintf("%d", now.Unix())

	return &accessintg.InboundRequest{
		Method: http.MethodPost,
		Path:   []string{path},
		Header: http.Header{
			"Content-Type":              []string{"application/x-www-form-urlencoded"},
			"X-Slack-Request-Timestamp": []string{timestamp},
			"X-Slack-Signature":         []string{tstSign(p, timestamp, body)},
		},
		Body: body,
		Now:  now,
	}
}

func TestDecodeInteraction(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()
	now := time.Now()

	payload := `{"type":"block_actions","team":{"id":"T00000001"},` +
		`"user":{"id":"U00000001"},"trigger_id":"trig-1",` +
		`"actions":[{"action_id":"` + actionIDApprove + `","value":"b0123456789"}]}`

	in := tstFormRequest(p, "interactions", now, url.Values{"payload": []string{payload}})

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.NotNil(t, ret.Action)
	assert.Equal(t, accessintg.ActionTypeSetReviewDecision, ret.Action.Type)
	assert.Equal(t, "U00000001", ret.Action.ExternalActorID)
	assert.Equal(t, "trig-1", ret.Action.ExternalEventID)
	assert.Equal(t, "b0123456789", ret.Action.BindingName)
	assert.Equal(t, accessv1.Review_Spec_DECISION_APPROVE, ret.Action.Decision)
}

func TestDecodeInteractionInvalidSignature(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstFormRequest(p, "interactions", time.Now(), url.Values{"payload": []string{"{}"}})
	in.Header.Set("X-Slack-Signature", "v0=deadbeef")

	_, err := p.DecodeInbound(ctx, in)
	assert.NotNil(t, err)
}

func TestDecodeInteractionStaleTimestamp(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstFormRequest(p, "interactions", time.Now().Add(-time.Hour),
		url.Values{"payload": []string{"{}"}})
	in.Now = time.Now()

	_, err := p.DecodeInbound(ctx, in)
	assert.NotNil(t, err)
}

func TestDecodeInteractionWrongTeam(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	payload := `{"type":"block_actions","team":{"id":"T99999999"},` +
		`"user":{"id":"U00000001"},"actions":[{"action_id":"` + actionIDApprove +
		`","value":"b0123456789"}]}`

	in := tstFormRequest(p, "interactions", time.Now(), url.Values{"payload": []string{payload}})

	_, err := p.DecodeInbound(ctx, in)
	assert.NotNil(t, err)
}

func TestDecodeInteractionUnknownAction(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	payload := `{"type":"block_actions","team":{"id":"T00000001"},` +
		`"user":{"id":"U00000001"},"actions":[{"action_id":"other","value":"b0123456789"}]}`

	in := tstFormRequest(p, "interactions", time.Now(), url.Values{"payload": []string{payload}})

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, ret.Action)
	assert.Equal(t, http.StatusOK, ret.Response.StatusCode)
}

func TestDecodeCommand(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstFormRequest(p, "commands", time.Now(), url.Values{
		"team_id":    []string{"T00000001"},
		"user_id":    []string{"U00000001"},
		"trigger_id": []string{"trig-2"},
		"text":       []string{"request svc:example --duration=2h --urgency=HIGH need to debug"},
	})

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.NotNil(t, ret.Action)
	assert.Equal(t, accessintg.ActionTypeCreateRequest, ret.Action.Type)
	assert.Equal(t, "svc:example", ret.Action.ResourceQuery)
	assert.Equal(t, 2*time.Hour, ret.Action.Duration)
	assert.Equal(t, accessv1.Request_Spec_HIGH, ret.Action.Urgency)
	assert.Equal(t, "need to debug", ret.Action.Justification)
	assert.Equal(t, "U00000001", ret.Action.ExternalActorID)
}

func TestDecodeCommandInvalid(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstFormRequest(p, "commands", time.Now(), url.Values{
		"team_id": []string{"T00000001"},
		"user_id": []string{"U00000001"},
		"text":    []string{"unknown svc:example"},
	})

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, ret.Action)
	assert.Equal(t, http.StatusOK, ret.Response.StatusCode)
}

func TestDecodeEventURLVerification(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()
	now := time.Now()

	body := []byte(`{"type":"url_verification","challenge":"abc123"}`)
	timestamp := fmt.Sprintf("%d", now.Unix())

	in := &accessintg.InboundRequest{
		Method: http.MethodPost,
		Path:   []string{"events"},
		Header: http.Header{
			"Content-Type":              []string{"application/json"},
			"X-Slack-Request-Timestamp": []string{timestamp},
			"X-Slack-Signature":         []string{tstSign(p, timestamp, body)},
		},
		Body: body,
		Now:  now,
	}

	ret, err := p.DecodeInbound(ctx, in)
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, ret.Action)
	assert.Equal(t, "abc123", string(ret.Response.Body))
}

func TestDecodeInboundInvalidPath(t *testing.T) {
	ctx := context.Background()
	p := tstProvider()

	in := tstFormRequest(p, "unknown", time.Now(), url.Values{})

	_, err := p.DecodeInbound(ctx, in)
	assert.NotNil(t, err)
}
