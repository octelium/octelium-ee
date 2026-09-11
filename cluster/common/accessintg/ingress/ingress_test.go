// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package ingress

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

const (
	tstSigningSecret = "signing-secret"
	tstTeamID        = "T00000001"
	tstSlackUserID   = "U00000001"
)

type ingressTest struct {
	ctx       context.Context
	srv       *Server
	octeliumC octeliumc.ClientInterface

	integration *accessv1.Integration
}

func newIngressTest(t *testing.T) *ingressTest {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	slackSrv := httptest.NewServer(http.HandlerFunc(tstSlackHandler))
	t.Cleanup(slackSrv.Close)

	octeliumC := tst.C.OcteliumC

	ret := &ingressTest{
		ctx:       ctx,
		srv:       NewServer(octeliumC, "example.com"),
		octeliumC: octeliumC,
	}
	t.Cleanup(ret.srv.Close)

	ret.integration = ret.createIntegration(t, slackSrv.URL)

	return ret
}

func tstSlackHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/auth.test":
		fmt.Fprintf(w, `{"ok":true,"team_id":%q,"team":"Example"}`, tstTeamID)
	case "/users.info":
		fmt.Fprintf(w, `{"ok":true,"user":{"id":%q,"name":"example"}}`, tstSlackUserID)
	default:
		fmt.Fprint(w, `{"ok":false,"error":"not_implemented"}`)
	}
}

func (i *ingressTest) createSecret(t *testing.T, value string) string {
	sec, err := i.octeliumC.EnterpriseC().CreateSecret(i.ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: value,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return sec.Metadata.Name
}

func (i *ingressTest) createIntegration(t *testing.T, baseURL string) *accessv1.Integration {
	item, err := i.octeliumC.AccessC().CreateIntegration(i.ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Integration_Spec{
			Type: &accessv1.Integration_Spec_Slack_{
				Slack: &accessv1.Integration_Spec_Slack{
					BotToken: &accessv1.Integration_Spec_Slack_BotToken{
						Type: &accessv1.Integration_Spec_Slack_BotToken_FromSecret{
							FromSecret: i.createSecret(t, "xoxb-token"),
						},
					},
					SigningSecret: &accessv1.Integration_Spec_Slack_SigningSecret{
						Type: &accessv1.Integration_Spec_Slack_SigningSecret_FromSecret{
							FromSecret: i.createSecret(t, tstSigningSecret),
						},
					},
					TeamID:  tstTeamID,
					BaseURL: baseURL,
				},
			},
		},
		Status: &accessv1.Integration_Status{
			Id:   utilrand.GetRandomStringCanonical(24),
			Type: accessv1.Integration_Status_SLACK,
			Capabilities: []accessv1.Integration_Status_Capability{
				accessv1.Integration_Status_NOTIFICATION,
				accessv1.Integration_Status_DIRECT_USER_DELIVERY,
				accessv1.Integration_Status_INTERACTIVE_REVIEW,
				accessv1.Integration_Status_REQUEST_CREATION,
				accessv1.Integration_Status_IDENTITY_RESOLUTION,
				accessv1.Integration_Status_PRESENTATION_UPDATE,
			},
			ExternalTenantID: tstTeamID,
		},
	})
	assert.Nil(t, err, "%+v", err)

	return item
}

func (i *ingressTest) createUser(t *testing.T) *corev1.User {
	usr, err := i.octeliumC.CoreC().CreateUser(i.ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_HUMAN,
		},
		Status: &corev1.User_Status{},
	})
	assert.Nil(t, err, "%+v", err)

	return usr
}

func (i *ingressTest) createIdentity(t *testing.T, usr *corev1.User, externalID string) {
	_, err := i.octeliumC.AccessC().CreateIntegrationIdentity(i.ctx, &accessv1.IntegrationIdentity{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.IntegrationIdentity_Spec{
			IntegrationRef: umetav1.GetObjectReference(i.integration),
			UserRef:        umetav1.GetObjectReference(usr),
			ExternalID:     externalID,
		},
		Status: &accessv1.IntegrationIdentity_Status{
			Source: accessv1.IntegrationIdentity_Status_MANUAL,
		},
	})
	assert.Nil(t, err, "%+v", err)
}

func (i *ingressTest) createRequest(t *testing.T, reviewer *corev1.User) *accessv1.Request {
	requester := i.createUser(t)

	req, err := i.octeliumC.AccessC().CreateRequest(i.ctx, &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_NORMAL,
			Subject: &accessv1.Request_Spec_Subject{
				Type: &accessv1.Request_Spec_Subject_UserRef{
					UserRef: umetav1.GetObjectReference(requester),
				},
			},
		},
		Status: &accessv1.Request_Status{
			UserRef: umetav1.GetObjectReference(requester),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_PENDING,
			},
			Rule: &accessv1.Policy_Spec_Rule{
				Name:   utilrand.GetRandomStringCanonical(6),
				Effect: accessv1.Policy_Spec_Rule_REVIEW,
				Action: &accessv1.Policy_Spec_Rule_Action{
					Type: &accessv1.Policy_Spec_Rule_Action_Review_{
						Review: &accessv1.Policy_Spec_Rule_Action_Review{
							Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
								{
									Name:                "first",
									ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
									Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
										{
											Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
												User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
													UserRef: umetav1.GetObjectReference(reviewer),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Review: &accessv1.Request_Status_Review{
				CurrentStep: 0,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return req
}

func (i *ingressTest) createBinding(t *testing.T, req *accessv1.Request,
	interactionMode accessv1.Policy_Spec_Rule_Surface_InteractionMode) *accessv1.IntegrationBinding {
	item, err := i.octeliumC.AccessC().CreateIntegrationBinding(i.ctx, &accessv1.IntegrationBinding{
		Metadata: &metav1.Metadata{
			Name: fmt.Sprintf("b%s", utilrand.GetRandomStringCanonical(16)),
		},
		Spec: &accessv1.IntegrationBinding_Spec{
			IntegrationRef:  umetav1.GetObjectReference(i.integration),
			RequestRef:      umetav1.GetObjectReference(req),
			StepIndex:       0,
			StepName:        "first",
			Purpose:         accessv1.IntegrationBinding_Spec_REVIEW_SURFACE,
			InteractionMode: interactionMode,
		},
		Status: &accessv1.IntegrationBinding_Status{
			State:      accessv1.IntegrationBinding_Status_READY,
			ExternalID: utilrand.GetRandomStringCanonical(10),
		},
	})
	assert.Nil(t, err, "%+v", err)

	return item
}

func (i *ingressTest) interaction(t *testing.T, actionID, bindingName,
	triggerID string) *accessintg.InboundRequest {
	payload := fmt.Sprintf(
		`{"type":"block_actions","team":{"id":%q},"user":{"id":%q},"trigger_id":%q,`+
			`"actions":[{"action_id":%q,"value":%q}]}`,
		tstTeamID, tstSlackUserID, triggerID, actionID, bindingName)

	body := []byte(url.Values{"payload": []string{payload}}.Encode())

	now := time.Now()
	timestamp := fmt.Sprintf("%d", now.Unix())

	mac := hmac.New(sha256.New, []byte(tstSigningSecret))
	mac.Write([]byte("v0:"))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(":"))
	mac.Write(body)

	return &accessintg.InboundRequest{
		Method: http.MethodPost,
		Path:   []string{"slack", "interactions"},
		Header: http.Header{
			"Content-Type":              []string{"application/x-www-form-urlencoded"},
			"X-Slack-Request-Timestamp": []string{timestamp},
			"X-Slack-Signature":         []string{fmt.Sprintf("v0=%s", hex.EncodeToString(mac.Sum(nil)))},
		},
		Body: body,
		Now:  now,
	}
}

func (i *ingressTest) listReviews(t *testing.T, req *accessv1.Request) []*accessv1.Review {
	itemList, err := i.octeliumC.AccessC().ListReview(i.ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.requestRef.uid", req.Metadata.Uid),
		},
	})
	assert.Nil(t, err, "%+v", err)

	return itemList.Items
}

func tstResponseText(t *testing.T, resp *accessintg.InboundResponse) string {
	ret := map[string]any{}
	assert.Nil(t, json.Unmarshal(resp.Body, &ret))

	text, _ := ret["text"].(string)
	return text
}

func TestInboundReviewDecision(t *testing.T) {
	i := newIngressTest(t)

	reviewer := i.createUser(t)
	i.createIdentity(t, reviewer, tstSlackUserID)

	req := i.createRequest(t, reviewer)
	binding := i.createBinding(t, req, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE)

	resp, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.interaction(t, "octelium-access-approve", binding.Metadata.Name, "trig-1"))
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	reviews := i.listReviews(t, req)
	assert.Equal(t, 1, len(reviews))
	assert.Equal(t, accessv1.Review_Spec_DECISION_APPROVE, reviews[0].Spec.Decision)
	assert.Equal(t, reviewer.Metadata.Uid, reviews[0].Status.UserRef.Uid)
	assert.Equal(t, "first", reviews[0].Status.StepName)
	assert.Equal(t, accessv1.Origin_INTEGRATION, reviews[0].Status.Origin.Type)
	assert.Equal(t, tstSlackUserID, reviews[0].Status.Origin.ExternalActorID)
	assert.Equal(t, i.integration.Metadata.Uid, reviews[0].Status.Origin.IntegrationRef.Uid)
}

func TestInboundDuplicateEventIsIgnored(t *testing.T) {
	i := newIngressTest(t)

	reviewer := i.createUser(t)
	i.createIdentity(t, reviewer, tstSlackUserID)

	req := i.createRequest(t, reviewer)
	binding := i.createBinding(t, req, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE)

	_, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.interaction(t, "octelium-access-approve", binding.Metadata.Name, "trig-1"))
	assert.Nil(t, err, "%+v", err)

	resp, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.interaction(t, "octelium-access-reject", binding.Metadata.Name, "trig-1"))
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, tstResponseText(t, resp), "already been recorded")

	reviews := i.listReviews(t, req)
	assert.Equal(t, 1, len(reviews))
	assert.Equal(t, accessv1.Review_Spec_DECISION_APPROVE, reviews[0].Spec.Decision)
}

func TestInboundUnknownExternalActor(t *testing.T) {
	i := newIngressTest(t)

	reviewer := i.createUser(t)
	req := i.createRequest(t, reviewer)
	binding := i.createBinding(t, req, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE)

	resp, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.interaction(t, "octelium-access-approve", binding.Metadata.Name, "trig-1"))
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, tstResponseText(t, resp), "could not recognize you")

	assert.Equal(t, 0, len(i.listReviews(t, req)))
}

func TestInboundIneligibleReviewer(t *testing.T) {
	i := newIngressTest(t)

	reviewer := i.createUser(t)
	other := i.createUser(t)
	i.createIdentity(t, other, tstSlackUserID)

	req := i.createRequest(t, reviewer)
	binding := i.createBinding(t, req, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE)

	resp, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.interaction(t, "octelium-access-approve", binding.Metadata.Name, "trig-1"))
	assert.Nil(t, err, "%+v", err)
	assert.NotEmpty(t, tstResponseText(t, resp))

	assert.Equal(t, 0, len(i.listReviews(t, req)))
}

func TestInboundDeepLinkOnlyBindingIsRefused(t *testing.T) {
	i := newIngressTest(t)

	reviewer := i.createUser(t)
	i.createIdentity(t, reviewer, tstSlackUserID)

	req := i.createRequest(t, reviewer)
	binding := i.createBinding(t, req, accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY)

	_, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.interaction(t, "octelium-access-approve", binding.Metadata.Name, "trig-1"))
	assert.NotNil(t, err)

	assert.Equal(t, 0, len(i.listReviews(t, req)))
}

func TestInboundUnknownBinding(t *testing.T) {
	i := newIngressTest(t)

	reviewer := i.createUser(t)
	i.createIdentity(t, reviewer, tstSlackUserID)

	req := i.createRequest(t, reviewer)

	resp, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.interaction(t, "octelium-access-approve",
			fmt.Sprintf("b%s", utilrand.GetRandomStringCanonical(16)), "trig-1"))
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, tstResponseText(t, resp), "no longer presented")

	assert.Equal(t, 0, len(i.listReviews(t, req)))
}

func TestInboundUnknownIntegration(t *testing.T) {
	i := newIngressTest(t)

	_, err := i.srv.Handle(i.ctx, utilrand.GetRandomStringCanonical(24),
		i.interaction(t, "octelium-access-approve", "b0123", "trig-1"))
	assert.NotNil(t, err)
}

func TestInboundDisabledIntegration(t *testing.T) {
	i := newIngressTest(t)

	i.integration.Spec.IsDisabled = true
	_, err := i.octeliumC.AccessC().UpdateIntegration(i.ctx, i.integration)
	assert.Nil(t, err, "%+v", err)

	_, err = i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.interaction(t, "octelium-access-approve", "b0123", "trig-1"))
	assert.NotNil(t, err)
}

func (i *ingressTest) createService(t *testing.T) *corev1.Service {
	svc, err := i.octeliumC.CoreC().CreateService(i.ctx, &corev1.Service{
		Metadata: &metav1.Metadata{
			Name: fmt.Sprintf("%s.default", utilrand.GetRandomStringCanonical(8)),
		},
		Spec:   &corev1.Service_Spec{},
		Status: &corev1.Service_Status{},
	})
	assert.Nil(t, err, "%+v", err)

	return svc
}

func (i *ingressTest) command(t *testing.T, text string) *accessintg.InboundRequest {
	body := []byte(url.Values{
		"team_id":    []string{tstTeamID},
		"user_id":    []string{tstSlackUserID},
		"trigger_id": []string{utilrand.GetRandomStringCanonical(12)},
		"text":       []string{text},
	}.Encode())

	now := time.Now()
	timestamp := fmt.Sprintf("%d", now.Unix())

	mac := hmac.New(sha256.New, []byte(tstSigningSecret))
	mac.Write([]byte("v0:"))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(":"))
	mac.Write(body)

	return &accessintg.InboundRequest{
		Method: http.MethodPost,
		Path:   []string{"slack", "commands"},
		Header: http.Header{
			"Content-Type":              []string{"application/x-www-form-urlencoded"},
			"X-Slack-Request-Timestamp": []string{timestamp},
			"X-Slack-Signature":         []string{fmt.Sprintf("v0=%s", hex.EncodeToString(mac.Sum(nil)))},
		},
		Body: body,
		Now:  now,
	}
}

func (i *ingressTest) listRequests(t *testing.T, usr *corev1.User) []*accessv1.Request {
	itemList, err := i.octeliumC.AccessC().ListRequest(i.ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterStatusUserUID(usr.Metadata.Uid),
		},
	})
	assert.Nil(t, err, "%+v", err)

	return itemList.Items
}

func TestInboundCreateRequest(t *testing.T) {
	i := newIngressTest(t)

	usr := i.createUser(t)
	i.createIdentity(t, usr, tstSlackUserID)
	svc := i.createService(t)

	resp, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.command(t, fmt.Sprintf("request svc:%s need it", svc.Metadata.Name)))
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, tstResponseText(t, resp), "was created")

	assert.Equal(t, 1, len(i.listRequests(t, usr)))
}

func TestInboundCreateRequestByDisabledUser(t *testing.T) {
	i := newIngressTest(t)

	usr := i.createUser(t)
	usr.Spec.IsDisabled = true
	usr, err := i.octeliumC.CoreC().UpdateUser(i.ctx, usr)
	assert.Nil(t, err, "%+v", err)

	i.createIdentity(t, usr, tstSlackUserID)
	svc := i.createService(t)

	resp, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.command(t, fmt.Sprintf("request svc:%s need it", svc.Metadata.Name)))
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, tstResponseText(t, resp), "not allowed")

	assert.Equal(t, 0, len(i.listRequests(t, usr)))
}

func TestInboundCreateRequestByLockedUser(t *testing.T) {
	i := newIngressTest(t)

	usr := i.createUser(t)
	usr.Status.IsLocked = true
	usr, err := i.octeliumC.CoreC().UpdateUser(i.ctx, usr)
	assert.Nil(t, err, "%+v", err)

	i.createIdentity(t, usr, tstSlackUserID)
	svc := i.createService(t)

	resp, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.command(t, fmt.Sprintf("request svc:%s need it", svc.Metadata.Name)))
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, tstResponseText(t, resp), "not allowed")

	assert.Equal(t, 0, len(i.listRequests(t, usr)))
}

func TestInboundCreateRequestOutOfRangeDuration(t *testing.T) {
	i := newIngressTest(t)

	usr := i.createUser(t)
	i.createIdentity(t, usr, tstSlackUserID)
	svc := i.createService(t)

	resp, err := i.srv.Handle(i.ctx, i.integration.Status.Id,
		i.command(t, fmt.Sprintf("request svc:%s --duration=2000000h need it", svc.Metadata.Name)))
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, tstResponseText(t, resp), "out of range")

	assert.Equal(t, 0, len(i.listRequests(t, usr)))
}
