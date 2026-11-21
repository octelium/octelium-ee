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
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func newBindingTest(t *testing.T) (context.Context, octeliumc.ClientInterface) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	return ctx, tst.C.OcteliumC
}

func tstCreateIntegration(ctx context.Context, t *testing.T,
	octeliumC octeliumc.ClientInterface,
	capabilities ...Capability) *accessv1.Integration {
	item, err := octeliumC.AccessC().CreateIntegration(ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Integration_Spec{
			Type: &accessv1.Integration_Spec_Slack_{
				Slack: &accessv1.Integration_Spec_Slack{
					ChannelID: "C12345678",
				},
			},
		},
		Status: &accessv1.Integration_Status{
			Id:           utilrand.GetRandomStringCanonical(24),
			Type:         accessv1.Integration_Status_SLACK,
			Capabilities: capabilities,
		},
	})
	assert.Nil(t, err, "%+v", err)

	return item
}

func tstCreateUser(ctx context.Context, t *testing.T,
	octeliumC octeliumc.ClientInterface) *corev1.User {
	usr, err := octeliumC.CoreC().CreateUser(ctx, &corev1.User{
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

func tstCreateRequest(ctx context.Context, t *testing.T, octeliumC octeliumc.ClientInterface,
	requester *corev1.User, rule *accessv1.Policy_Spec_Rule) *accessv1.Request {
	req, err := octeliumC.AccessC().CreateRequest(ctx, &accessv1.Request{
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
			Rule: rule,
			Review: &accessv1.Request_Status_Review{
				CurrentStep: 0,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return req
}

func tstReviewRule(steps ...*accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Policy_Spec_Rule {
	return &accessv1.Policy_Spec_Rule{
		Name:   utilrand.GetRandomStringCanonical(6),
		Effect: accessv1.Policy_Spec_Rule_REVIEW,
		Action: &accessv1.Policy_Spec_Rule_Action{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_{
				Review: &accessv1.Policy_Spec_Rule_Action_Review{
					Steps: steps,
				},
			},
		},
	}
}

func tstStep(name string, reviewer *corev1.User,
	surfaces ...*accessv1.Policy_Spec_Rule_Surface) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Name:                name,
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
		Surfaces: surfaces,
	}
}

func tstSurface(integration *accessv1.Integration,
	audience accessv1.Policy_Spec_Rule_Surface_Destination_Audience,
	mode accessv1.Policy_Spec_Rule_Surface_InteractionMode) *accessv1.Policy_Spec_Rule_Surface {
	return &accessv1.Policy_Spec_Rule_Surface{
		InteractionMode: mode,
		Destination: &accessv1.Policy_Spec_Rule_Surface_Destination{
			IntegrationRef: umetav1.GetObjectReference(integration),
			Audience:       audience,
		},
	}
}

func tstSharedSurface(integration *accessv1.Integration,
	mode accessv1.Policy_Spec_Rule_Surface_InteractionMode) *accessv1.Policy_Spec_Rule_Surface {
	return tstSurface(integration, accessv1.Policy_Spec_Rule_Surface_Destination_SHARED, mode)
}

func tstReviewersSurface(integration *accessv1.Integration,
	mode accessv1.Policy_Spec_Rule_Surface_InteractionMode) *accessv1.Policy_Spec_Rule_Surface {
	return tstSurface(integration, accessv1.Policy_Spec_Rule_Surface_Destination_REVIEWERS, mode)
}

func TestDesiredBindingsShared(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	integration := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_NOTIFICATION)

	reviewer := tstCreateUser(ctx, t, octeliumC)
	requester := tstCreateUser(ctx, t, octeliumC)

	req := tstCreateRequest(ctx, t, octeliumC, requester, tstReviewRule(
		tstStep("first", reviewer,
			tstSharedSurface(integration, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))

	items, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 1, len(items))

	item := items[0]
	assert.Equal(t, accessv1.IntegrationBinding_Status_REVIEW_SURFACE, item.Status.Purpose)
	assert.Equal(t, "first", item.Status.StepName)
	assert.Nil(t, item.Status.UserRef)
	assert.Equal(t, accessv1.Policy_Spec_Rule_Surface_Destination_SHARED, item.Status.Audience)
	assert.Equal(t, integration.Metadata.Uid, item.Status.IntegrationRef.Uid)
	assert.Equal(t, req.Metadata.Uid, item.Status.RequestRef.Uid)
	assert.Equal(t, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE, item.Status.InteractionMode)
	assert.Equal(t, accessv1.IntegrationBinding_Status_PENDING, item.Status.State)
	assert.Equal(t, req.Status.SpecRevision, item.Status.SpecRevision)
	assert.Equal(t, BindingName(req.Metadata.Uid, item.Status, 0), item.Name)
}

func TestDesiredBindingsSharedPerIntegration(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	first := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_NOTIFICATION)
	second := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_NOTIFICATION)

	reviewer := tstCreateUser(ctx, t, octeliumC)
	requester := tstCreateUser(ctx, t, octeliumC)

	req := tstCreateRequest(ctx, t, octeliumC, requester, tstReviewRule(
		tstStep("first", reviewer,
			tstSharedSurface(first, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE),
			tstSharedSurface(second, accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY))))

	items, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 2, len(items))

	assert.Equal(t, first.Metadata.Uid, items[0].Status.IntegrationRef.Uid)
	assert.Equal(t, second.Metadata.Uid, items[1].Status.IntegrationRef.Uid)
	assert.NotEqual(t, items[0].Name, items[1].Name)
}

func TestDesiredBindingsReviewers(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	integration := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_DIRECT_USER_DELIVERY)

	reviewer := tstCreateUser(ctx, t, octeliumC)
	requester := tstCreateUser(ctx, t, octeliumC)

	req := tstCreateRequest(ctx, t, octeliumC, requester, tstReviewRule(
		tstStep("first", reviewer,
			tstReviewersSurface(integration, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))

	items, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, reviewer.Metadata.Uid, items[0].Status.UserRef.Uid)
	assert.Equal(t, accessv1.Policy_Spec_Rule_Surface_Destination_REVIEWERS,
		items[0].Status.Audience)
}

func TestDesiredBindingsSkipRequesterReviewer(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	integration := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_DIRECT_USER_DELIVERY)

	requester := tstCreateUser(ctx, t, octeliumC)

	req := tstCreateRequest(ctx, t, octeliumC, requester, tstReviewRule(
		tstStep("first", requester,
			tstReviewersSurface(integration, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))

	items, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 0, len(items))
}

func TestDesiredBindingsSkipMissingCapability(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	integration := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_NOTIFICATION)

	reviewer := tstCreateUser(ctx, t, octeliumC)
	requester := tstCreateUser(ctx, t, octeliumC)

	req := tstCreateRequest(ctx, t, octeliumC, requester, tstReviewRule(
		tstStep("first", reviewer,
			tstReviewersSurface(integration, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))

	items, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 0, len(items))
}

func TestDesiredBindingsSkipDisabledIntegration(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	integration := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_NOTIFICATION)

	integration.Spec.IsDisabled = true
	_, err := octeliumC.AccessC().UpdateIntegration(ctx, integration)
	assert.Nil(t, err, "%+v", err)

	reviewer := tstCreateUser(ctx, t, octeliumC)
	requester := tstCreateUser(ctx, t, octeliumC)

	req := tstCreateRequest(ctx, t, octeliumC, requester, tstReviewRule(
		tstStep("first", reviewer,
			tstSharedSurface(integration, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))

	items, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 0, len(items))
}

func TestDesiredBindingsNotification(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	integration := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_DIRECT_USER_DELIVERY)

	reviewer := tstCreateUser(ctx, t, octeliumC)
	requester := tstCreateUser(ctx, t, octeliumC)

	rule := tstReviewRule(tstStep("first", reviewer))
	rule.Notifications = []*accessv1.Policy_Spec_Rule_Surface{
		tstSurface(integration, accessv1.Policy_Spec_Rule_Surface_Destination_REQUESTER,
			accessv1.Policy_Spec_Rule_Surface_INTERACTIVE),
	}

	req := tstCreateRequest(ctx, t, octeliumC, requester, rule)

	items, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, accessv1.IntegrationBinding_Status_NOTIFICATION, items[0].Status.Purpose)
	assert.Equal(t, requester.Metadata.Uid, items[0].Status.UserRef.Uid)
	assert.Equal(t, accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY,
		items[0].Status.InteractionMode)
}

func TestDesiredBindingsSubject(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	integration := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_DIRECT_USER_DELIVERY)

	reviewer := tstCreateUser(ctx, t, octeliumC)
	requester := tstCreateUser(ctx, t, octeliumC)

	rule := tstReviewRule(tstStep("first", reviewer))
	rule.Notifications = []*accessv1.Policy_Spec_Rule_Surface{
		tstSurface(integration, accessv1.Policy_Spec_Rule_Surface_Destination_SUBJECT,
			accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY),
	}

	req := tstCreateRequest(ctx, t, octeliumC, requester, rule)

	items, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, requester.Metadata.Uid, items[0].Status.UserRef.Uid)
	assert.Equal(t, accessv1.Policy_Spec_Rule_Surface_Destination_SUBJECT,
		items[0].Status.Audience)
}

func TestDesiredBindingsSkipUnknownIntegration(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	reviewer := tstCreateUser(ctx, t, octeliumC)
	requester := tstCreateUser(ctx, t, octeliumC)

	req := tstCreateRequest(ctx, t, octeliumC, requester, tstReviewRule(
		tstStep("first", reviewer, &accessv1.Policy_Spec_Rule_Surface{
			Destination: &accessv1.Policy_Spec_Rule_Surface_Destination{
				IntegrationRef: &metav1.ObjectReference{
					Name: utilrand.GetRandomStringCanonical(8),
				},
				Audience: accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
			},
		})))

	items, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 0, len(items))
}

func TestPresentationRevision(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	integration := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_NOTIFICATION)

	reviewer := tstCreateUser(ctx, t, octeliumC)
	requester := tstCreateUser(ctx, t, octeliumC)

	req := tstCreateRequest(ctx, t, octeliumC, requester, tstReviewRule(
		tstStep("first", reviewer,
			tstSharedSurface(integration, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))))

	desired, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)

	binding := &accessv1.IntegrationBinding{
		Metadata: &metav1.Metadata{
			Name: desired[0].Name,
		},
		Spec:   &accessv1.IntegrationBinding_Spec{},
		Status: desired[0].Status,
	}

	opts := &BuildPresentationOpts{
		OcteliumC:     octeliumC,
		Binding:       binding,
		Request:       req,
		ClusterDomain: "example.com",
	}

	first, err := BuildPresentation(ctx, opts)
	assert.Nil(t, err, "%+v", err)
	assert.True(t, first.IsActionable)
	assert.False(t, first.IsClosed)
	assert.Equal(t, "first", first.StepName)
	assert.Equal(t, 1, first.StepIndex)
	assert.Equal(t, 1, first.StepCount)
	assert.Contains(t, first.PortalURL, req.Metadata.Name)

	second, err := BuildPresentation(ctx, opts)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, first.Revision(), second.Revision())

	for _, revision := range []uint32{req.Status.SpecRevision + 1, req.Status.SpecRevision} {
		_, err = octeliumC.AccessC().CreateReview(ctx, &accessv1.Review{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.Review_Spec{
				Decision: accessv1.Review_Spec_DECISION_APPROVE,
			},
			Status: &accessv1.Review_Status{
				RequestRef:   umetav1.GetObjectReference(req),
				UserRef:      umetav1.GetObjectReference(reviewer),
				SpecRevision: revision,
			},
		})
		assert.Nil(t, err, "%+v", err)
	}

	decided, err := BuildPresentation(ctx, opts)
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, decided.Decisions, 1)
	assert.Equal(t, "1 of 1 approval", decided.ApprovalSummary)

	req.Status.State.Status = accessv1.Request_Status_State_APPROVED
	req, err = octeliumC.AccessC().UpdateRequest(ctx, req)
	assert.Nil(t, err, "%+v", err)

	opts.Request = req

	third, err := BuildPresentation(ctx, opts)
	assert.Nil(t, err, "%+v", err)
	assert.False(t, third.IsActionable)
	assert.True(t, third.IsClosed)
	assert.NotEqual(t, first.Revision(), third.Revision())
}

func TestRemovedReviewerBindingIsNoLongerCurrent(t *testing.T) {
	ctx, octeliumC := newBindingTest(t)

	integration := tstCreateIntegration(ctx, t, octeliumC,
		accessv1.Integration_Status_DIRECT_USER_DELIVERY)

	grp, err := octeliumC.CoreC().CreateGroup(ctx, &corev1.Group{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   &corev1.Group_Spec{},
		Status: &corev1.Group_Status{},
	})
	assert.Nil(t, err, "%+v", err)

	var reviewers []*corev1.User
	for range 3 {
		usr := tstCreateUser(ctx, t, octeliumC)
		usr.Spec.Groups = []string{grp.Metadata.Name}
		usr, err = octeliumC.CoreC().UpdateUser(ctx, usr)
		assert.Nil(t, err, "%+v", err)
		reviewers = append(reviewers, usr)
	}
	removed, voter := reviewers[0], reviewers[1]

	step := tstStep("review", removed,
		tstReviewersSurface(integration, accessv1.Policy_Spec_Rule_Surface_INTERACTIVE))
	step.ApprovalRequirement = accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT
	step.ApprovalCount = 2
	step.Reviewers = []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
		{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_{
				Group: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group{
					GroupRef: umetav1.GetObjectReference(grp),
				},
			},
		},
	}

	req := tstCreateRequest(ctx, t, octeliumC, tstCreateUser(ctx, t, octeliumC), tstReviewRule(step))
	req.Status.SpecRevision = 1
	assert.Nil(t, accesscmd.StartReviewStep(ctx, &accesscmd.StartReviewStepOpts{
		OcteliumC: octeliumC,
		Request:   req,
		Index:     0,
	}))
	req, err = octeliumC.AccessC().UpdateRequest(ctx, req)
	assert.Nil(t, err, "%+v", err)

	desired, err := DesiredBindings(ctx, octeliumC, req)
	assert.Nil(t, err, "%+v", err)

	var binding *accessv1.IntegrationBinding
	for _, itm := range desired {
		if itm.Status.UserRef.Uid == removed.Metadata.Uid {
			binding = &accessv1.IntegrationBinding{
				Metadata: &metav1.Metadata{Name: itm.Name},
				Spec:     &accessv1.IntegrationBinding_Spec{},
				Status:   itm.Status,
			}
		}
	}
	assert.NotNil(t, binding)

	isCurrent, err := IsBindingCurrent(ctx, octeliumC, binding, req)
	assert.Nil(t, err, "%+v", err)
	assert.True(t, isCurrent)

	removed.Spec.Groups = nil
	_, err = octeliumC.CoreC().UpdateUser(ctx, removed)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, accesscmd.RefreshStepReviewers(ctx, octeliumC, req, nil))
	req, err = octeliumC.AccessC().UpdateRequest(ctx, req)
	assert.Nil(t, err, "%+v", err)

	_, err = accesscmd.SetReviewDecision(ctx, &accesscmd.SetReviewDecisionOpts{
		OcteliumC:            octeliumC,
		Reviewer:             voter,
		Request:              req,
		Decision:             accessv1.Review_Spec_DECISION_APPROVE,
		Justification:        "Checked with the on-call lead",
		ExpectedStepName:     "review",
		ExpectedSpecRevision: 1,
	})
	assert.Nil(t, err, "%+v", err)

	isCurrent, err = IsBindingCurrent(ctx, octeliumC, binding, req)
	assert.Nil(t, err, "%+v", err)
	assert.False(t, isCurrent)

	presentation, err := BuildPresentation(ctx, &BuildPresentationOpts{
		OcteliumC:     octeliumC,
		Binding:       binding,
		Request:       req,
		ClusterDomain: "example.com",
	})
	assert.Nil(t, err, "%+v", err)
	assert.False(t, presentation.IsActionable)
	assert.Empty(t, presentation.Decisions)
	assert.Empty(t, presentation.ApprovalSummary)

	for _, itm := range desired {
		if itm.Status.UserRef.Uid != voter.Metadata.Uid {
			continue
		}

		presentation, err := BuildPresentation(ctx, &BuildPresentationOpts{
			OcteliumC: octeliumC,
			Binding: &accessv1.IntegrationBinding{
				Metadata: &metav1.Metadata{Name: itm.Name},
				Spec:     &accessv1.IntegrationBinding_Spec{},
				Status:   itm.Status,
			},
			Request:       req,
			ClusterDomain: "example.com",
		})
		assert.Nil(t, err, "%+v", err)
		assert.True(t, presentation.IsActionable)
		assert.Len(t, presentation.Decisions, 1)
	}
}
