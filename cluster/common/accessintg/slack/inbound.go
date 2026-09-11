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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/pkg/errors"
)

const maxTimestampSkew = 5 * time.Minute

type interactionPayload struct {
	Type string `json:"type,omitempty"`
	Team struct {
		ID string `json:"id,omitempty"`
	} `json:"team"`
	User struct {
		ID string `json:"id,omitempty"`
	} `json:"user"`
	TriggerID string `json:"trigger_id,omitempty"`
	Actions   []struct {
		ActionID string `json:"action_id,omitempty"`
		Value    string `json:"value,omitempty"`
	} `json:"actions,omitempty"`
}

type eventPayload struct {
	Type      string `json:"type,omitempty"`
	Challenge string `json:"challenge,omitempty"`
	TeamID    string `json:"team_id,omitempty"`
	EventID   string `json:"event_id,omitempty"`
}

func (p *Provider) DecodeInbound(ctx context.Context,
	in *accessintg.InboundRequest) (*accessintg.Inbound, error) {
	if in.Method != http.MethodPost {
		return nil, errors.Errorf("Invalid method")
	}

	if err := p.verifySignature(in); err != nil {
		return nil, err
	}

	if len(in.Path) != 1 {
		return nil, errors.Errorf("Invalid path")
	}

	switch in.Path[0] {
	case "interactions":
		return p.decodeInteraction(in)
	case "commands":
		return p.decodeCommand(in)
	case "events":
		return p.decodeEvent(in)
	default:
		return nil, errors.Errorf("Invalid path")
	}
}

func (p *Provider) verifySignature(in *accessintg.InboundRequest) error {
	if len(p.signingSecret) == 0 {
		return errors.Errorf("The Slack Integration has no signing secret")
	}

	timestamp := in.Header.Get("X-Slack-Request-Timestamp")
	signature := in.Header.Get("X-Slack-Signature")

	if timestamp == "" || signature == "" {
		return errors.Errorf("Missing Slack signature headers")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.Errorf("Invalid Slack timestamp")
	}

	skew := in.Now.Sub(time.Unix(ts, 0))
	if skew < 0 {
		skew = -skew
	}
	if skew > maxTimestampSkew {
		return errors.Errorf("The Slack timestamp is too old")
	}

	mac := hmac.New(sha256.New, p.signingSecret)
	mac.Write([]byte("v0:"))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(":"))
	mac.Write(in.Body)

	expected := fmt.Sprintf("v0=%s", hex.EncodeToString(mac.Sum(nil)))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.Errorf("Invalid Slack signature")
	}

	return nil
}

func (p *Provider) decodeInteraction(in *accessintg.InboundRequest) (*accessintg.Inbound, error) {
	form, err := parseForm(in)
	if err != nil {
		return nil, err
	}

	payload := &interactionPayload{}
	if err := json.Unmarshal([]byte(form.Get("payload")), payload); err != nil {
		return nil, errors.Errorf("Could not unmarshal the Slack interaction payload")
	}

	if err := p.checkTeamID(payload.Team.ID); err != nil {
		return nil, err
	}

	if payload.Type != "block_actions" || len(payload.Actions) == 0 {
		return &accessintg.Inbound{
			Response: emptyResponse(),
		}, nil
	}

	action := payload.Actions[0]

	decision := accessv1.Review_Spec_DECISION_UNSET
	switch action.ActionID {
	case actionIDApprove:
		decision = accessv1.Review_Spec_DECISION_APPROVE
	case actionIDReject:
		decision = accessv1.Review_Spec_DECISION_REJECT
	default:
		return &accessintg.Inbound{
			Response: emptyResponse(),
		}, nil
	}

	if payload.User.ID == "" || action.Value == "" {
		return nil, errors.Errorf("Incomplete Slack interaction payload")
	}

	return &accessintg.Inbound{
		Action: &accessintg.Action{
			Type:            accessintg.ActionTypeSetReviewDecision,
			ExternalActorID: payload.User.ID,
			ExternalEventID: payload.TriggerID,
			BindingName:     action.Value,
			Decision:        decision,
		},
	}, nil
}

func (p *Provider) decodeCommand(in *accessintg.InboundRequest) (*accessintg.Inbound, error) {
	form, err := parseForm(in)
	if err != nil {
		return nil, err
	}

	if err := p.checkTeamID(form.Get("team_id")); err != nil {
		return nil, err
	}

	userID := form.Get("user_id")
	if userID == "" {
		return nil, errors.Errorf("Incomplete Slack command payload")
	}

	action, err := parseCommand(form.Get("text"))
	if err != nil {
		return &accessintg.Inbound{
			Response: ephemeralResponse(err.Error()),
		}, nil
	}

	action.ExternalActorID = userID
	action.ExternalEventID = form.Get("trigger_id")

	return &accessintg.Inbound{
		Action: action,
	}, nil
}

func (p *Provider) decodeEvent(in *accessintg.InboundRequest) (*accessintg.Inbound, error) {
	payload := &eventPayload{}
	if err := json.Unmarshal(in.Body, payload); err != nil {
		return nil, errors.Errorf("Could not unmarshal the Slack event payload")
	}

	if payload.Type == "url_verification" {
		return &accessintg.Inbound{
			Response: &accessintg.InboundResponse{
				StatusCode:  http.StatusOK,
				ContentType: "text/plain; charset=utf-8",
				Body:        []byte(payload.Challenge),
			},
		}, nil
	}

	if err := p.checkTeamID(payload.TeamID); err != nil {
		return nil, err
	}

	return &accessintg.Inbound{
		Response: emptyResponse(),
	}, nil
}

func (p *Provider) checkTeamID(teamID string) error {
	if p.teamID == "" {
		return nil
	}

	if teamID != p.teamID {
		return errors.Errorf("The request belongs to the Slack workspace %s instead of %s",
			teamID, p.teamID)
	}

	return nil
}

func (p *Provider) RenderActionResponse(ctx context.Context, in *accessintg.InboundRequest,
	action *accessintg.Action, res *accessintg.ActionResult) *accessintg.InboundResponse {
	if res == nil {
		return ephemeralResponse("Octelium could not process this action.")
	}

	text := res.Message
	if res.PortalURL != "" {
		text = fmt.Sprintf("%s\n<%s|Open in the Octelium access portal>",
			text, escapeURL(res.PortalURL))
	}

	return ephemeralResponse(text)
}

func parseCommand(text string) (*accessintg.Action, error) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return nil, errors.Errorf(
			"Usage: request <svc:NAME|cat:NAME> [--duration=8h] [--urgency=HIGH] [justification]")
	}

	if !strings.EqualFold(fields[0], "request") {
		return nil, errors.Errorf("Unknown command %q. The only command is `request`.", fields[0])
	}

	fields = fields[1:]
	if len(fields) == 0 {
		return nil, errors.Errorf(
			"Usage: request <svc:NAME|cat:NAME> [--duration=8h] [--urgency=HIGH] [justification]")
	}

	ret := &accessintg.Action{
		Type:          accessintg.ActionTypeCreateRequest,
		ResourceQuery: fields[0],
	}

	rest := []string{}

	for _, field := range fields[1:] {
		switch {
		case strings.HasPrefix(field, "--duration="):
			duration, err := time.ParseDuration(strings.TrimPrefix(field, "--duration="))
			if err != nil {
				return nil, errors.Errorf("Invalid duration %q", field)
			}
			ret.Duration = duration

		case strings.HasPrefix(field, "--urgency="):
			urgency, ok := accessv1.Request_Spec_Urgency_value[strings.ToUpper(
				strings.TrimPrefix(field, "--urgency="))]
			if !ok {
				return nil, errors.Errorf("Invalid urgency %q", field)
			}
			ret.Urgency = accessv1.Request_Spec_Urgency(urgency)

		default:
			rest = append(rest, field)
		}
	}

	ret.Justification = strings.Join(rest, " ")

	return ret, nil
}

func parseForm(in *accessintg.InboundRequest) (url.Values, error) {
	contentType := in.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		return nil, errors.Errorf("Invalid content type")
	}

	form, err := url.ParseQuery(string(in.Body))
	if err != nil {
		return nil, errors.Errorf("Could not parse the Slack payload")
	}

	return form, nil
}

func emptyResponse() *accessintg.InboundResponse {
	return &accessintg.InboundResponse{
		StatusCode: http.StatusOK,
	}
}

func ephemeralResponse(text string) *accessintg.InboundResponse {
	body, err := json.Marshal(map[string]any{
		"response_type": "ephemeral",
		"text":          text,
	})
	if err != nil {
		return emptyResponse()
	}

	return &accessintg.InboundResponse{
		StatusCode:  http.StatusOK,
		ContentType: "application/json; charset=utf-8",
		Body:        body,
	}
}
