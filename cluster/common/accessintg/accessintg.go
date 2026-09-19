// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accessintg

import (
	"context"
	"net/http"
	"time"

	"github.com/octelium/octelium/apis/main/accessv1"
)

type Type = accessv1.Integration_Status_Type

type Capability = accessv1.Integration_Status_Capability

type TenantInfo struct {
	ID   string
	Name string
}

type ExternalUser struct {
	ID       string
	Username string
	Email    string
}

type Provider interface {
	Type() Type
	Capabilities() []Capability
	Validate(ctx context.Context) (*TenantInfo, error)
	Close() error
}

type IdentityResolver interface {
	GetExternalUser(ctx context.Context, externalID string) (*ExternalUser, error)
	GetExternalUserByEmail(ctx context.Context, email string) (*ExternalUser, error)
}

type PresentationDelivery struct {
	Binding      *accessv1.IntegrationBinding
	Integration  *accessv1.Integration
	RecipientID  string
	Presentation *Presentation
}

type DeliveryResult struct {
	ExternalID          string
	ExternalURL         string
	ExternalRecipientID string
}

type PresentationDeliverer interface {
	CreatePresentation(ctx context.Context, in *PresentationDelivery) (*DeliveryResult, error)
	UpdatePresentation(ctx context.Context, in *PresentationDelivery) (*DeliveryResult, error)
	ClosePresentation(ctx context.Context, in *PresentationDelivery) (*DeliveryResult, error)
}

type InboundRequest struct {
	Method string
	Path   []string
	Header http.Header
	Body   []byte
	Now    time.Time
}

type InboundResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

type ActionType int

const (
	ActionTypeUnset ActionType = iota
	ActionTypeSetReviewDecision
	ActionTypeCreateRequest
)

type Action struct {
	Type ActionType

	ExternalActorID string
	ExternalEventID string

	BindingName      string
	ExternalObjectID string
	ExternalStatus   string

	Decision      accessv1.Review_Spec_Decision
	Justification string

	ResourceQuery string
	Duration      time.Duration
	Urgency       accessv1.Request_Spec_Urgency
}

type Inbound struct {
	Action   *Action
	Response *InboundResponse
}

type ActionResult struct {
	IsAccepted bool
	Message    string

	RequestName string
	PortalURL   string
}

type DecisionResolver interface {
	ResolveDecision(ctx context.Context, action *Action,
		integration *accessv1.Integration) (accessv1.Review_Spec_Decision, error)
}

type InboundHandler interface {
	DecodeInbound(ctx context.Context, in *InboundRequest) (*Inbound, error)
	RenderActionResponse(ctx context.Context, in *InboundRequest,
		action *Action, res *ActionResult) *InboundResponse
}

func HasCapability(itm *accessv1.Integration, capability Capability) bool {
	if itm == nil || itm.Status == nil {
		return false
	}

	for _, c := range itm.Status.Capabilities {
		if c == capability {
			return true
		}
	}

	return false
}
