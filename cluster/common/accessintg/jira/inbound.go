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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/pkg/errors"
)

type webhookPayload struct {
	WebhookEvent string `json:"webhookEvent,omitempty"`
	Timestamp    int64  `json:"timestamp,omitempty"`
	Issue        *struct {
		ID  string `json:"id,omitempty"`
		Key string `json:"key,omitempty"`
	} `json:"issue,omitempty"`
	User *struct {
		AccountID string `json:"accountId,omitempty"`
	} `json:"user,omitempty"`
	Changelog *struct {
		ID    string `json:"id,omitempty"`
		Items []struct {
			Field    string `json:"field,omitempty"`
			ToString string `json:"toString,omitempty"`
		} `json:"items,omitempty"`
	} `json:"changelog,omitempty"`
}

func (p *Provider) DecodeInbound(ctx context.Context,
	in *accessintg.InboundRequest) (*accessintg.Inbound, error) {
	if in.Method != http.MethodPost {
		return nil, errors.Errorf("Invalid method")
	}

	if len(in.Path) != 1 || in.Path[0] != "webhook" {
		return nil, errors.Errorf("Invalid path")
	}

	if err := p.verifySignature(in); err != nil {
		return nil, err
	}

	payload := &webhookPayload{}
	if err := json.Unmarshal(in.Body, payload); err != nil {
		return nil, errors.Errorf("Could not unmarshal the Jira webhook payload")
	}

	if payload.WebhookEvent != "jira:issue_updated" ||
		payload.Issue == nil || payload.Issue.Key == "" ||
		payload.User == nil || payload.User.AccountID == "" {
		return &accessintg.Inbound{
			Response: emptyResponse(),
		}, nil
	}

	externalStatus, ok := statusChange(payload)
	if !ok {
		return &accessintg.Inbound{
			Response: emptyResponse(),
		}, nil
	}

	eventID := fmt.Sprintf("%s.%d", payload.Issue.Key, payload.Timestamp)
	if payload.Changelog != nil && payload.Changelog.ID != "" {
		eventID = fmt.Sprintf("%s.%s", payload.Issue.Key, payload.Changelog.ID)
	}

	return &accessintg.Inbound{
		Action: &accessintg.Action{
			Type:             accessintg.ActionTypeSetReviewDecision,
			ExternalActorID:  payload.User.AccountID,
			ExternalEventID:  eventID,
			ExternalObjectID: payload.Issue.Key,
			ExternalStatus:   externalStatus,
		},
	}, nil
}

func (p *Provider) RenderActionResponse(ctx context.Context, in *accessintg.InboundRequest,
	action *accessintg.Action, res *accessintg.ActionResult) *accessintg.InboundResponse {
	return emptyResponse()
}

func (p *Provider) verifySignature(in *accessintg.InboundRequest) error {
	if len(p.webhookSecret) == 0 {
		return errors.Errorf("The Jira Integration has no webhook secret")
	}

	signature := in.Header.Get("X-Hub-Signature")
	if signature == "" {
		return errors.Errorf("Missing the Jira webhook signature")
	}

	mac := hmac.New(sha256.New, p.webhookSecret)
	mac.Write(in.Body)

	expected := fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return errors.Errorf("Invalid Jira webhook signature")
	}

	return nil
}

func statusChange(payload *webhookPayload) (string, bool) {
	if payload.Changelog == nil {
		return "", false
	}

	for _, item := range payload.Changelog.Items {
		if !strings.EqualFold(item.Field, "status") {
			continue
		}

		status := strings.TrimSpace(item.ToString)
		if status == "" {
			return "", false
		}

		return status, true
	}

	return "", false
}

func emptyResponse() *accessintg.InboundResponse {
	return &accessintg.InboundResponse{
		StatusCode: http.StatusOK,
	}
}
