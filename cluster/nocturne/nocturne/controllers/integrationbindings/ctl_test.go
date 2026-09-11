// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package integrationbindings

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

type slackFake struct {
	postMessages atomic.Int64
	updates      atomic.Int64
	fail         atomic.Bool
}

func (f *slackFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if f.fail.Load() {
		fmt.Fprint(w, `{"ok":false,"error":"internal_error"}`)
		return
	}

	switch r.URL.Path {
	case "/auth.test":
		fmt.Fprint(w, `{"ok":true,"team_id":"T00000001","team":"Example"}`)
	case "/chat.postMessage":
		f.postMessages.Add(1)
		fmt.Fprint(w, `{"ok":true,"ts":"1700000000.000100","channel":"C12345678"}`)
	case "/chat.update":
		f.updates.Add(1)
		fmt.Fprint(w, `{"ok":true,"ts":"1700000000.000100","channel":"C12345678"}`)
	case "/chat.getPermalink":
		fmt.Fprint(w, `{"ok":true,"permalink":"https://example.slack.com/archives/C12345678/p1"}`)
	default:
		fmt.Fprint(w, `{"ok":false,"error":"not_implemented"}`)
	}
}

type bindingTest struct {
	ctx       context.Context
	ctrl      *Controller
	octeliumC octeliumc.ClientInterface
	slack     *slackFake

	integration *accessv1.Integration
	target      *accessv1.IntegrationTarget
}

func newBindingTest(t *testing.T) *bindingTest {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	slack := &slackFake{}
	slackSrv := httptest.NewServer(slack)
	t.Cleanup(slackSrv.Close)

	ctrl, err := NewController(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)

	ret := &bindingTest{
		ctx:       ctx,
		ctrl:      ctrl,
		octeliumC: tst.C.OcteliumC,
		slack:     slack,
	}

	ret.integration = ret.createIntegration(t, slackSrv.URL)
	ret.target = ret.createTarget(t)

	return ret
}

func (b *bindingTest) createSecret(t *testing.T) string {
	sec, err := b.octeliumC.EnterpriseC().CreateSecret(b.ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(24),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return sec.Metadata.Name
}

func (b *bindingTest) createIntegration(t *testing.T, baseURL string) *accessv1.Integration {
	item, err := b.octeliumC.AccessC().CreateIntegration(b.ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Integration_Spec{
			Type: &accessv1.Integration_Spec_Slack_{
				Slack: &accessv1.Integration_Spec_Slack{
					BotToken: &accessv1.Integration_Spec_Slack_BotToken{
						Type: &accessv1.Integration_Spec_Slack_BotToken_FromSecret{
							FromSecret: b.createSecret(t),
						},
					},
					SigningSecret: &accessv1.Integration_Spec_Slack_SigningSecret{
						Type: &accessv1.Integration_Spec_Slack_SigningSecret_FromSecret{
							FromSecret: b.createSecret(t),
						},
					},
					BaseURL: baseURL,
				},
			},
		},
		Status: &accessv1.Integration_Status{
			Id:   utilrand.GetRandomStringCanonical(24),
			Type: accessv1.Integration_Status_SLACK,
			Capabilities: []accessv1.Integration_Status_Capability{
				accessv1.Integration_Status_NOTIFICATION,
				accessv1.Integration_Status_PRESENTATION_UPDATE,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return item
}

func (b *bindingTest) createTarget(t *testing.T) *accessv1.IntegrationTarget {
	item, err := b.octeliumC.AccessC().CreateIntegrationTarget(b.ctx, &accessv1.IntegrationTarget{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.IntegrationTarget_Spec{
			IntegrationRef: umetav1.GetObjectReference(b.integration),
			Type: &accessv1.IntegrationTarget_Spec_Slack_{
				Slack: &accessv1.IntegrationTarget_Spec_Slack{
					ChannelID: "C12345678",
				},
			},
		},
		Status: &accessv1.IntegrationTarget_Status{
			Type: accessv1.Integration_Status_SLACK,
		},
	})
	assert.Nil(t, err, "%+v", err)

	return item
}

func (b *bindingTest) createUser(t *testing.T) *corev1.User {
	usr, err := b.octeliumC.CoreC().CreateUser(b.ctx, &corev1.User{
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

func (b *bindingTest) createRequest(t *testing.T) *accessv1.Request {
	reviewer := b.createUser(t)
	requester := b.createUser(t)

	req, err := b.octeliumC.AccessC().CreateRequest(b.ctx, &accessv1.Request{
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

func (b *bindingTest) createNotificationBinding(t *testing.T,
	req *accessv1.Request) *accessv1.IntegrationBinding {
	item, err := b.octeliumC.AccessC().CreateIntegrationBinding(b.ctx,
		&accessv1.IntegrationBinding{
			Metadata: &metav1.Metadata{
				Name: accessintg.BindingName(req.Metadata.Uid,
					accessv1.IntegrationBinding_Spec_NOTIFICATION, 0, 0, b.target.Metadata.Uid),
				IsSystem: true,
			},
			Spec: &accessv1.IntegrationBinding_Spec{
				IntegrationRef: umetav1.GetObjectReference(b.integration),
				RequestRef:     umetav1.GetObjectReference(req),
				TargetRef:      umetav1.GetObjectReference(b.target),
				Purpose:        accessv1.IntegrationBinding_Spec_NOTIFICATION,
			},
			Status: &accessv1.IntegrationBinding_Status{
				State: accessv1.IntegrationBinding_Status_PENDING,
			},
		})
	assert.Nil(t, err, "%+v", err)

	return item
}

func (b *bindingTest) createBinding(t *testing.T,
	req *accessv1.Request) *accessv1.IntegrationBinding {
	item, err := b.octeliumC.AccessC().CreateIntegrationBinding(b.ctx,
		&accessv1.IntegrationBinding{
			Metadata: &metav1.Metadata{
				Name: accessintg.BindingName(req.Metadata.Uid,
					accessv1.IntegrationBinding_Spec_REVIEW_SURFACE, 0, 0, b.target.Metadata.Uid),
				IsSystem: true,
			},
			Spec: &accessv1.IntegrationBinding_Spec{
				IntegrationRef:  umetav1.GetObjectReference(b.integration),
				RequestRef:      umetav1.GetObjectReference(req),
				TargetRef:       umetav1.GetObjectReference(b.target),
				StepIndex:       0,
				StepName:        "first",
				Purpose:         accessv1.IntegrationBinding_Spec_REVIEW_SURFACE,
				InteractionMode: accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY,
			},
			Status: &accessv1.IntegrationBinding_Status{
				State: accessv1.IntegrationBinding_Status_PENDING,
			},
		})
	assert.Nil(t, err, "%+v", err)

	return item
}

func (b *bindingTest) getBinding(t *testing.T, uid string) *accessv1.IntegrationBinding {
	item, err := b.octeliumC.AccessC().GetIntegrationBinding(b.ctx, &rmetav1.GetOptions{
		Uid: uid,
	})
	assert.Nil(t, err, "%+v", err)

	return item
}

func TestBindingReconcileCreatesPresentationOnce(t *testing.T) {
	b := newBindingTest(t)

	req := b.createRequest(t)
	binding := b.createBinding(t, req)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))

	binding = b.getBinding(t, binding.Metadata.Uid)
	assert.Equal(t, accessv1.IntegrationBinding_Status_READY, binding.Status.State)
	assert.Equal(t, "1700000000.000100", binding.Status.ExternalID)
	assert.Equal(t, "C12345678", binding.Status.ExternalRecipientID)
	assert.NotEmpty(t, binding.Status.AppliedRevision)
	assert.Equal(t, binding.Status.DesiredRevision, binding.Status.AppliedRevision)
	assert.Equal(t, int64(1), b.slack.postMessages.Load())

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))
	assert.Equal(t, int64(1), b.slack.postMessages.Load())
	assert.Equal(t, int64(0), b.slack.updates.Load())
}

func TestBindingReconcileUpdatesOnRequestChange(t *testing.T) {
	b := newBindingTest(t)

	req := b.createRequest(t)
	binding := b.createBinding(t, req)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))
	binding = b.getBinding(t, binding.Metadata.Uid)

	req.Status.State.Status = accessv1.Request_Status_State_APPROVED
	_, err := b.octeliumC.AccessC().UpdateRequest(b.ctx, req)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))

	binding = b.getBinding(t, binding.Metadata.Uid)
	assert.Equal(t, accessv1.IntegrationBinding_Status_CLOSED, binding.Status.State)
	assert.Equal(t, int64(1), b.slack.postMessages.Load())
	assert.Equal(t, int64(1), b.slack.updates.Load())
}

func TestBindingReconcileClosesOnStepChange(t *testing.T) {
	b := newBindingTest(t)

	req := b.createRequest(t)
	binding := b.createBinding(t, req)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))
	binding = b.getBinding(t, binding.Metadata.Uid)

	req.Status.Review.CurrentStep = 1
	_, err := b.octeliumC.AccessC().UpdateRequest(b.ctx, req)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))

	binding = b.getBinding(t, binding.Metadata.Uid)
	assert.Equal(t, accessv1.IntegrationBinding_Status_CLOSED, binding.Status.State)
}

func TestBindingReconcileBacksOffOnFailure(t *testing.T) {
	b := newBindingTest(t)

	req := b.createRequest(t)
	binding := b.createBinding(t, req)

	b.slack.fail.Store(true)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))

	binding = b.getBinding(t, binding.Metadata.Uid)
	assert.Equal(t, accessv1.IntegrationBinding_Status_PENDING, binding.Status.State)
	assert.Equal(t, uint32(1), binding.Status.Attempts)
	assert.NotEmpty(t, binding.Status.LastError)
	assert.NotNil(t, binding.Status.NextAttemptAt)
	assert.Empty(t, binding.Status.ExternalID)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))
	assert.Equal(t, uint32(1), b.getBinding(t, binding.Metadata.Uid).Status.Attempts)

	b.slack.fail.Store(false)

	binding.Status.NextAttemptAt = nil
	binding, err := b.octeliumC.AccessC().UpdateIntegrationBinding(b.ctx, binding)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))

	binding = b.getBinding(t, binding.Metadata.Uid)
	assert.Equal(t, accessv1.IntegrationBinding_Status_READY, binding.Status.State)
	assert.Equal(t, uint32(0), binding.Status.Attempts)
	assert.Empty(t, binding.Status.LastError)
	assert.Equal(t, int64(1), b.slack.postMessages.Load())
}

func TestBindingReconcileSkipsDisabledIntegration(t *testing.T) {
	b := newBindingTest(t)

	req := b.createRequest(t)
	binding := b.createBinding(t, req)

	b.integration.Spec.IsDisabled = true
	_, err := b.octeliumC.AccessC().UpdateIntegration(b.ctx, b.integration)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))
	assert.Equal(t, int64(0), b.slack.postMessages.Load())
}

func TestBindingDeleteBindingsOf(t *testing.T) {
	b := newBindingTest(t)

	req := b.createRequest(t)
	binding := b.createBinding(t, req)

	assert.Nil(t, b.ctrl.DeleteBindingsOf(b.ctx, req))

	_, err := b.octeliumC.AccessC().GetIntegrationBinding(b.ctx, &rmetav1.GetOptions{
		Uid: binding.Metadata.Uid,
	})
	assert.NotNil(t, err)
}

func TestBindingNotificationIsDeliveredForTerminalRequest(t *testing.T) {
	b := newBindingTest(t)

	req := b.createRequest(t)

	req.Status.State.Status = accessv1.Request_Status_State_APPROVED
	req, err := b.octeliumC.AccessC().UpdateRequest(b.ctx, req)
	assert.Nil(t, err, "%+v", err)

	binding := b.createNotificationBinding(t, req)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))

	binding = b.getBinding(t, binding.Metadata.Uid)
	assert.Equal(t, accessv1.IntegrationBinding_Status_CLOSED, binding.Status.State)
	assert.Equal(t, "1700000000.000100", binding.Status.ExternalID)
	assert.Equal(t, int64(1), b.slack.postMessages.Load())
}

func TestBindingReviewSurfaceIsNotDeliveredForTerminalRequest(t *testing.T) {
	b := newBindingTest(t)

	req := b.createRequest(t)

	req.Status.State.Status = accessv1.Request_Status_State_REJECTED
	req, err := b.octeliumC.AccessC().UpdateRequest(b.ctx, req)
	assert.Nil(t, err, "%+v", err)

	binding := b.createBinding(t, req)

	assert.Nil(t, b.ctrl.Reconcile(b.ctx, binding))

	binding = b.getBinding(t, binding.Metadata.Uid)
	assert.Equal(t, accessv1.IntegrationBinding_Status_CLOSED, binding.Status.State)
	assert.Empty(t, binding.Status.ExternalID)
	assert.Equal(t, int64(0), b.slack.postMessages.Load())
}
