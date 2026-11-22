// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package support

import (
	"fmt"
	"net/url"
	"time"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type portalActor struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type portalSLA struct {
	Coverage           string `json:"coverage"`
	FirstResponseDueAt string `json:"firstResponseDueAt"`
	FirstRespondedAt   string `json:"firstRespondedAt"`
	Paused             bool   `json:"paused"`
	NextUpdateDueAt    string `json:"nextUpdateDueAt"`
}

type portalTicket struct {
	ID           string `json:"id"`
	Number       uint32 `json:"number"`
	Organization *struct {
		UID string `json:"uid"`
	} `json:"organization"`
	Subject           string `json:"subject"`
	Priority          string `json:"priority"`
	RequestedPriority string `json:"requestedPriority"`
	Status            string `json:"status"`
	Channel           string `json:"channel"`
	Requester         struct {
		Name     string `json:"name"`
		Verified bool   `json:"verified"`
	} `json:"requester"`
	Assignee *struct {
		Name string `json:"name"`
	} `json:"assignee"`
	SupportLevel string `json:"supportLevel"`
	Cluster      *struct {
		Domain string `json:"domain"`
	} `json:"cluster"`
	Environment     map[string]string `json:"environment"`
	SLA             *portalSLA        `json:"sla"`
	Escalated       bool              `json:"escalated"`
	CreatedAt       string            `json:"createdAt"`
	UpdatedAt       string            `json:"updatedAt"`
	LastActivityAt  string            `json:"lastActivityAt"`
	ResolvedAt      string            `json:"resolvedAt"`
	ClosedAt        string            `json:"closedAt"`
	MessageCount    uint32            `json:"messageCount"`
	AttachmentCount uint32            `json:"attachmentCount"`
}

type portalEvent struct {
	ID                 string      `json:"id"`
	Sequence           uint64      `json:"sequence"`
	Type               string      `json:"type"`
	Actor              portalActor `json:"actor"`
	Channel            string      `json:"channel"`
	At                 string      `json:"at"`
	Body               string      `json:"body"`
	AttachmentIDs      []string    `json:"attachmentIDs"`
	From               string      `json:"from"`
	To                 string      `json:"to"`
	Reason             string      `json:"reason"`
	SkippedAttachments []string    `json:"skippedAttachments"`
}

type portalAttachment struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	ContentType string      `json:"contentType"`
	Size        uint64      `json:"size"`
	SHA256      string      `json:"sha256"`
	Purpose     string      `json:"purpose"`
	UploadedBy  portalActor `json:"uploadedBy"`
	Channel     string      `json:"channel"`
	CreatedAt   string      `json:"createdAt"`
	Available   bool        `json:"available"`
}

type portalDetail struct {
	Ticket       portalTicket       `json:"ticket"`
	Events       []portalEvent      `json:"events"`
	EventsCursor string             `json:"eventsCursor"`
	Attachments  []portalAttachment `json:"attachments"`
	Actions      struct {
		Reply    bool `json:"reply"`
		Resolve  bool `json:"resolve"`
		Escalate bool `json:"escalate"`
	} `json:"actions"`
}

type portalTicketResponse struct {
	Ticket portalTicket `json:"ticket"`
}

type portalTicketList struct {
	Tickets []portalTicket `json:"tickets"`
	Cursor  string         `json:"cursor"`
}

type portalEventList struct {
	Events []portalEvent `json:"events"`
	Cursor string        `json:"cursor"`
}

type portalAttachmentResponse struct {
	Attachment portalAttachment `json:"attachment"`
}

type portalDescription struct {
	Organization struct {
		UID         string `json:"uid"`
		DisplayName string `json:"displayName"`
	} `json:"organization"`
	ClusterDomain  string `json:"clusterDomain"`
	InstallationID string `json:"installationID"`
	SupportLevel   string `json:"supportLevel"`
	Entitled       bool   `json:"entitled"`
	Attachments    struct {
		MaxBytes     uint64   `json:"maxBytes"`
		MaxPerTicket uint32   `json:"maxPerTicket"`
		Extensions   []string `json:"extensions"`
	} `json:"attachments"`
	MaxBodyLength uint32 `json:"maxBodyLength"`
	AutoCloseDays uint32 `json:"autoCloseDays"`
}

func toTimestamp(arg string) *timestamppb.Timestamp {
	if arg == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339Nano, arg)
	if err != nil {
		return nil
	}

	return timestamppb.New(t)
}

func toPriority(arg string) enterprisev1.SupportRequest_Priority {
	return enterprisev1.SupportRequest_Priority(enterprisev1.SupportRequest_Priority_value[arg])
}

func fromPriority(arg enterprisev1.SupportRequest_Priority) string {
	switch arg {
	case enterprisev1.SupportRequest_P1, enterprisev1.SupportRequest_P2,
		enterprisev1.SupportRequest_P3, enterprisev1.SupportRequest_P4:
		return arg.String()
	default:
		return enterprisev1.SupportRequest_P3.String()
	}
}

func toStatus(arg string) enterprisev1.SupportRequest_Status {
	return enterprisev1.SupportRequest_Status(enterprisev1.SupportRequest_Status_value[arg])
}

func toSupportLevel(arg string) enterprisev1.SupportRequest_SupportLevel {
	switch arg {
	case "NONE":
		return enterprisev1.SupportRequest_NONE
	case "STANDARD":
		return enterprisev1.SupportRequest_STANDARD
	case "PREMIUM":
		return enterprisev1.SupportRequest_PREMIUM
	default:
		return enterprisev1.SupportRequest_SUPPORT_LEVEL_UNKNOWN
	}
}

func toChannel(arg string) enterprisev1.SupportRequest_Channel {
	return enterprisev1.SupportRequest_Channel(enterprisev1.SupportRequest_Channel_value[arg])
}

func toActor(arg portalActor) *enterprisev1.SupportRequest_Actor {
	ret := &enterprisev1.SupportRequest_Actor{
		Name: arg.Name,
	}

	switch arg.Type {
	case "customer":
		ret.Type = enterprisev1.SupportRequest_Actor_CUSTOMER
	case "cluster":
		ret.Type = enterprisev1.SupportRequest_Actor_CLUSTER
	case "email":
		ret.Type = enterprisev1.SupportRequest_Actor_EMAIL
	case "agent":
		ret.Type = enterprisev1.SupportRequest_Actor_AGENT
	case "system":
		ret.Type = enterprisev1.SupportRequest_Actor_SYSTEM
	}

	return ret
}

func toEventType(arg string) enterprisev1.SupportRequest_Event_Type {
	switch arg {
	case "created":
		return enterprisev1.SupportRequest_Event_CREATED
	case "message":
		return enterprisev1.SupportRequest_Event_MESSAGE
	case "status":
		return enterprisev1.SupportRequest_Event_STATUS
	case "priority":
		return enterprisev1.SupportRequest_Event_PRIORITY
	case "category":
		return enterprisev1.SupportRequest_Event_CATEGORY
	case "assignment":
		return enterprisev1.SupportRequest_Event_ASSIGNMENT
	case "escalation":
		return enterprisev1.SupportRequest_Event_ESCALATION
	case "attachment":
		return enterprisev1.SupportRequest_Event_ATTACHMENT
	case "organization":
		return enterprisev1.SupportRequest_Event_ORGANIZATION
	default:
		return enterprisev1.SupportRequest_Event_TYPE_UNKNOWN
	}
}

func toEvent(arg *portalEvent) *enterprisev1.SupportRequest_Event {
	return &enterprisev1.SupportRequest_Event{
		Id:                 arg.ID,
		Sequence:           arg.Sequence,
		Type:               toEventType(arg.Type),
		Actor:              toActor(arg.Actor),
		Channel:            toChannel(arg.Channel),
		CreatedAt:          toTimestamp(arg.At),
		Body:               arg.Body,
		AttachmentIDs:      arg.AttachmentIDs,
		From:               arg.From,
		To:                 arg.To,
		Reason:             arg.Reason,
		SkippedAttachments: arg.SkippedAttachments,
	}
}

func toEvents(args []portalEvent) []*enterprisev1.SupportRequest_Event {
	ret := make([]*enterprisev1.SupportRequest_Event, 0, len(args))
	for i := range args {
		ret = append(ret, toEvent(&args[i]))
	}
	return ret
}

func toAttachment(arg *portalAttachment) *enterprisev1.SupportRequest_Attachment {
	ret := &enterprisev1.SupportRequest_Attachment{
		Id:          arg.ID,
		Name:        arg.Name,
		ContentType: arg.ContentType,
		Size:        arg.Size,
		Sha256:      arg.SHA256,
		Purpose:     enterprisev1.SupportRequest_Attachment_FILE,
		UploadedBy:  toActor(arg.UploadedBy),
		Channel:     toChannel(arg.Channel),
		CreatedAt:   toTimestamp(arg.CreatedAt),
		IsAvailable: arg.Available,
	}

	if arg.Purpose == "DIAGNOSTIC" {
		ret.Purpose = enterprisev1.SupportRequest_Attachment_DIAGNOSTIC
	}

	return ret
}

func (s *Server) toSupportRequest(arg *portalTicket) *enterprisev1.SupportRequest {
	ret := &enterprisev1.SupportRequest{
		Id:                arg.ID,
		Number:            arg.Number,
		Subject:           arg.Subject,
		Status:            toStatus(arg.Status),
		Priority:          toPriority(arg.Priority),
		RequestedPriority: toPriority(arg.RequestedPriority),
		SupportLevel:      toSupportLevel(arg.SupportLevel),
		Channel:           toChannel(arg.Channel),
		Reporter: &enterprisev1.SupportRequest_Reporter{
			Name:       arg.Requester.Name,
			IsVerified: arg.Requester.Verified,
		},
		IsEscalated:     arg.Escalated,
		Environment:     arg.Environment,
		CreatedAt:       toTimestamp(arg.CreatedAt),
		UpdatedAt:       toTimestamp(arg.UpdatedAt),
		LastActivityAt:  toTimestamp(arg.LastActivityAt),
		ResolvedAt:      toTimestamp(arg.ResolvedAt),
		ClosedAt:        toTimestamp(arg.ClosedAt),
		MessageCount:    arg.MessageCount,
		AttachmentCount: arg.AttachmentCount,
	}

	if arg.Assignee != nil {
		ret.Assignee = arg.Assignee.Name
	}

	if arg.Cluster != nil {
		ret.ClusterDomain = arg.Cluster.Domain
	}

	if arg.SLA != nil {
		ret.Sla = &enterprisev1.SupportRequest_SLA{
			Coverage:           arg.SLA.Coverage,
			FirstResponseDueAt: toTimestamp(arg.SLA.FirstResponseDueAt),
			FirstRespondedAt:   toTimestamp(arg.SLA.FirstRespondedAt),
			IsPaused:           arg.SLA.Paused,
			NextUpdateDueAt:    toTimestamp(arg.SLA.NextUpdateDueAt),
		}
	}

	if arg.Organization != nil && arg.Organization.UID != "" && arg.ID != "" {
		ret.PortalURL = fmt.Sprintf("%s/organizations/%s/support/%s",
			s.portal.baseURL, url.PathEscape(arg.Organization.UID), url.PathEscape(arg.ID))
	}

	return ret
}

func (s *Server) toSupportRequestDetail(arg *portalDetail) *enterprisev1.SupportRequest {
	ret := s.toSupportRequest(&arg.Ticket)
	ret.Events = toEvents(arg.Events)
	ret.EventsCursor = arg.EventsCursor
	ret.Actions = &enterprisev1.SupportRequest_Actions{
		CanReply:    arg.Actions.Reply,
		CanResolve:  arg.Actions.Resolve,
		CanEscalate: arg.Actions.Escalate,
	}

	for i := range arg.Attachments {
		ret.Attachments = append(ret.Attachments, toAttachment(&arg.Attachments[i]))
	}

	return ret
}
