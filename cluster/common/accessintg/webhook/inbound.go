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
	"crypto/hmac"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/pkg/errors"
)

const maxTimestampSkew = 5 * time.Minute

type inboundPayload struct {
	Action string `json:"action,omitempty"`

	ExternalActorID string `json:"externalActorID,omitempty"`
	EventID         string `json:"eventID,omitempty"`

	BindingName string `json:"bindingName,omitempty"`

	Decision      string `json:"decision,omitempty"`
	Justification string `json:"justification,omitempty"`

	Resource string `json:"resource,omitempty"`
	Duration string `json:"duration,omitempty"`
	Urgency  string `json:"urgency,omitempty"`
}

func (p *Provider) DecodeInbound(ctx context.Context,
	in *accessintg.InboundRequest) (*accessintg.Inbound, error) {
	if in.Method != http.MethodPost {
		return nil, errors.Errorf("Invalid method")
	}

	if len(in.Path) != 0 {
		return nil, errors.Errorf("Invalid path")
	}

	if err := p.verifySignature(in); err != nil {
		return nil, err
	}

	payload := &inboundPayload{}
	if err := json.Unmarshal(in.Body, payload); err != nil {
		return nil, errors.Errorf("Could not unmarshal the inbound payload")
	}

	if payload.ExternalActorID == "" {
		return nil, errors.Errorf("The inbound payload has no externalActorID")
	}

	switch payload.Action {
	case "setReviewDecision":
		if payload.BindingName == "" {
			return nil, errors.Errorf("The inbound payload has no bindingName")
		}

		decision, ok := accessv1.Review_Spec_Decision_value[decisionName(payload.Decision)]
		if !ok {
			return nil, errors.Errorf("The inbound payload has an invalid decision")
		}

		return &accessintg.Inbound{
			Action: &accessintg.Action{
				Type:            accessintg.ActionTypeSetReviewDecision,
				ExternalActorID: payload.ExternalActorID,
				ExternalEventID: payload.EventID,
				BindingName:     payload.BindingName,
				Decision:        accessv1.Review_Spec_Decision(decision),
				Justification:   payload.Justification,
			},
		}, nil

	case "createRequest":
		action := &accessintg.Action{
			Type:            accessintg.ActionTypeCreateRequest,
			ExternalActorID: payload.ExternalActorID,
			ExternalEventID: payload.EventID,
			ResourceQuery:   payload.Resource,
			Justification:   payload.Justification,
		}

		if payload.Duration != "" {
			duration, err := time.ParseDuration(payload.Duration)
			if err != nil {
				return nil, errors.Errorf("The inbound payload has an invalid duration")
			}
			action.Duration = duration
		}

		if payload.Urgency != "" {
			urgency, ok := accessv1.Request_Spec_Urgency_value[strings.ToUpper(payload.Urgency)]
			if !ok {
				return nil, errors.Errorf("The inbound payload has an invalid urgency")
			}
			action.Urgency = accessv1.Request_Spec_Urgency(urgency)
		}

		return &accessintg.Inbound{
			Action: action,
		}, nil

	default:
		return nil, errors.Errorf("The inbound payload has an invalid action")
	}
}

func (p *Provider) RenderActionResponse(ctx context.Context, in *accessintg.InboundRequest,
	action *accessintg.Action, res *accessintg.ActionResult) *accessintg.InboundResponse {
	if res == nil {
		res = &accessintg.ActionResult{}
	}

	body, err := json.Marshal(map[string]any{
		"isAccepted":  res.IsAccepted,
		"message":     res.Message,
		"requestName": res.RequestName,
		"portalURL":   res.PortalURL,
	})
	if err != nil {
		return &accessintg.InboundResponse{
			StatusCode: http.StatusOK,
		}
	}

	return &accessintg.InboundResponse{
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		Body:        body,
	}
}

func (p *Provider) verifySignature(in *accessintg.InboundRequest) error {
	if len(p.inboundSecret) == 0 {
		return errors.Errorf("The Webhook Integration has no inbound secret")
	}

	timestamp := in.Header.Get("X-Octelium-Timestamp")
	signature := in.Header.Get("X-Octelium-Signature")

	if timestamp == "" || signature == "" {
		return errors.Errorf("Missing the inbound signature headers")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.Errorf("Invalid inbound timestamp")
	}

	skew := in.Now.Sub(time.Unix(ts, 0))
	if skew < 0 {
		skew = -skew
	}
	if skew > maxTimestampSkew {
		return errors.Errorf("The inbound timestamp is too old")
	}

	expected := sign(p.inboundSecret, timestamp, in.Body)

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.Errorf("Invalid inbound signature")
	}

	return nil
}

func decisionName(arg string) string {
	arg = strings.ToUpper(strings.TrimSpace(arg))
	if arg == "" {
		return accessv1.Review_Spec_DECISION_UNSET.String()
	}

	if strings.HasPrefix(arg, "DECISION_") {
		return arg
	}

	return "DECISION_" + arg
}
