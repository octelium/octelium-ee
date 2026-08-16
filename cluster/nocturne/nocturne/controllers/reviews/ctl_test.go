// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package reviews

import (
	"context"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func newControllerTest(t *testing.T) (context.Context, *Controller, octeliumc.ClientInterface) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	ctrl, err := NewController(ctx, tst.C.OcteliumC)
	assert.Nil(t, err)

	return ctx, ctrl, tst.C.OcteliumC
}

func objRef(kind string) *metav1.ObjectReference {
	return &metav1.ObjectReference{
		ApiVersion: "core/v1",
		Kind:       kind,
		Name:       utilrand.GetRandomStringCanonical(8),
		Uid:        utilrand.GetRandomStringCanonical(16),
	}
}

func userReviewer(ref *metav1.ObjectReference) *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
		Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
			User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
				UserRef: ref,
			},
		},
	}
}

func groupReviewer(ref *metav1.ObjectReference) *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
		Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_{
			Group: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group{
				GroupRef: ref,
			},
		},
	}
}

func createGroup(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface) *corev1.Group {
	grp, err := octeliumC.CoreC().CreateGroup(ctx, &corev1.Group{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   &corev1.Group_Spec{},
		Status: &corev1.Group_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	return grp
}

func createUser(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, groups ...string) *corev1.User {
	usr, err := octeliumC.CoreC().CreateUser(ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type:   corev1.User_Spec_HUMAN,
			Groups: groups,
		},
		Status: &corev1.User_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	return usr
}

func anyStep(reviewers ...*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Reviewers:           reviewers,
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
	}
}

func allStep(reviewers ...*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Reviewers:           reviewers,
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ALL,
	}
}

func countStep(count uint32, reviewers ...*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Reviewers:           reviewers,
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT,
		ApprovalCount:       count,
	}
}

func reviewRule(steps ...*accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Policy_Spec_Rule {
	return &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{MatchAny: true}},
		Action: &accessv1.Policy_Spec_Rule_Action{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_{
				Review: &accessv1.Policy_Spec_Rule_Action_Review{
					Steps: steps,
				},
			},
		},
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	}
}

func createPendingRequest(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, rule *accessv1.Policy_Spec_Rule, currentStep int32) *accessv1.Request {
	req, err := octeliumC.AccessC().CreateRequest(ctx, &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_NORMAL,
		},
		Status: &accessv1.Request_Status{
			UserRef: objRef("User"),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_PENDING,
			},
			Rule: rule,
			Review: &accessv1.Request_Status_Review{
				CurrentStep: currentStep,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	return req
}

func createReview(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, reqRef, userRef *metav1.ObjectReference, stepIndex int32, decision accessv1.Review_Spec_Decision) *accessv1.Review {
	rev, err := octeliumC.AccessC().CreateReview(ctx, &accessv1.Review{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Review_Spec{
			Decision:      decision,
			Justification: utilrand.GetRandomString(16),
		},
		Status: &accessv1.Review_Status{
			RequestRef: reqRef,
			UserRef:    userRef,
			StepIndex:  stepIndex,
			SetAt:      pbutils.Now(),
		},
	})
	assert.Nil(t, err, "%+v", err)
	return rev
}

func getRequest(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, uid string) *accessv1.Request {
	out, err := octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{Uid: uid})
	assert.Nil(t, err, "%+v", err)
	return out
}

func TestReviewSingleStepAnyApprove(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC, reviewRule(anyStep(userReviewer(userA))), 0)
	reqRef := umetav1.GetObjectReference(req)

	rev := createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, rev))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, 1, len(reqG.Status.Review.LastSteps))
	assert.Equal(t, int32(0), reqG.Status.Review.LastSteps[0].StepIndex)
}

func TestReviewSingleStepReject(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC, reviewRule(anyStep(userReviewer(userA))), 0)
	reqRef := umetav1.GetObjectReference(req)

	rev := createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_REJECT)
	assert.Nil(t, ctrl.OnAdd(ctx, rev))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
}

func TestReviewNonReviewerIgnored(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC, reviewRule(anyStep(userReviewer(userA))), 0)
	reqRef := umetav1.GetObjectReference(req)

	rev := createReview(t, ctx, octeliumC, reqRef, objRef("User"), 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, rev))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.Equal(t, 0, len(reqG.Status.Review.LastSteps))
}

func TestReviewWrongStepIndexIgnored(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC, reviewRule(anyStep(userReviewer(userA))), 0)
	reqRef := umetav1.GetObjectReference(req)

	rev := createReview(t, ctx, octeliumC, reqRef, userA, 1, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, rev))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.Equal(t, 0, len(reqG.Status.Review.LastSteps))
}

func TestReviewCountQuorum(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	userB := objRef("User")
	userC := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(countStep(2, userReviewer(userA), userReviewer(userB), userReviewer(userC))), 0)
	reqRef := umetav1.GetObjectReference(req)

	revA := createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revA))

	reqAfterA := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqAfterA.Status.State.Status)
	assert.Equal(t, 1, len(reqAfterA.Status.Review.LastSteps))

	revB := createReview(t, ctx, octeliumC, reqRef, userB, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revB))

	reqAfterB := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqAfterB.Status.State.Status)
	assert.Equal(t, 2, len(reqAfterB.Status.Review.LastSteps))
}

func TestReviewAllQuorum(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	userB := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(allStep(userReviewer(userA), userReviewer(userB))), 0)
	reqRef := umetav1.GetObjectReference(req)

	revA := createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revA))
	assert.Equal(t, accessv1.Request_Status_State_PENDING, getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)

	revB := createReview(t, ctx, octeliumC, reqRef, userB, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revB))
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestReviewMultiStep(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	userB := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(anyStep(userReviewer(userA)), anyStep(userReviewer(userB))), 0)
	reqRef := umetav1.GetObjectReference(req)

	revA := createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revA))

	reqAfterA := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqAfterA.Status.State.Status)
	assert.Equal(t, int32(1), reqAfterA.Status.Review.CurrentStep)

	revB := createReview(t, ctx, octeliumC, reqRef, userB, 1, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revB))

	reqAfterB := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqAfterB.Status.State.Status)
}

func TestReviewMultiStepMixed(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	userB := objRef("User")
	userD := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(
			countStep(2, userReviewer(userA), userReviewer(userB)),
			anyStep(userReviewer(userD)),
		), 0)
	reqRef := umetav1.GetObjectReference(req)

	assert.Nil(t, ctrl.OnAdd(ctx, createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_APPROVE)))
	assert.Nil(t, ctrl.OnAdd(ctx, createReview(t, ctx, octeliumC, reqRef, userB, 0, accessv1.Review_Spec_DECISION_APPROVE)))

	reqMid := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqMid.Status.State.Status)
	assert.Equal(t, int32(1), reqMid.Status.Review.CurrentStep)

	assert.Nil(t, ctrl.OnAdd(ctx, createReview(t, ctx, octeliumC, reqRef, userD, 1, accessv1.Review_Spec_DECISION_APPROVE)))

	reqEnd := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqEnd.Status.State.Status)
}

func TestReviewWrongStepReviewerIgnored(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	userB := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(anyStep(userReviewer(userA)), anyStep(userReviewer(userB))), 0)
	reqRef := umetav1.GetObjectReference(req)

	rev := createReview(t, ctx, octeliumC, reqRef, userB, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, rev))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.Equal(t, int32(0), reqG.Status.Review.CurrentStep)
	assert.Equal(t, 0, len(reqG.Status.Review.LastSteps))
}

func TestReviewDuplicateIgnored(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	userB := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(countStep(2, userReviewer(userA), userReviewer(userB))), 0)
	reqRef := umetav1.GetObjectReference(req)

	revA := createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revA))
	assert.Nil(t, ctrl.OnAdd(ctx, revA))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.Equal(t, 1, len(reqG.Status.Review.LastSteps))
}

func TestReviewForceFlipToReject(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	userB := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(countStep(2, userReviewer(userA), userReviewer(userB))), 0)
	reqRef := umetav1.GetObjectReference(req)

	revA := createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revA))
	assert.Equal(t, accessv1.Request_Status_State_PENDING, getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)

	flipped := pbutils.Clone(revA).(*accessv1.Review)
	flipped.Spec.Decision = accessv1.Review_Spec_DECISION_REJECT
	assert.Nil(t, ctrl.OnUpdate(ctx, flipped, revA))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
}

func TestReviewRequestNotPending(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	req, err := octeliumC.AccessC().CreateRequest(ctx, &accessv1.Request{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &accessv1.Request_Spec{Urgency: accessv1.Request_Spec_NORMAL},
		Status: &accessv1.Request_Status{
			UserRef: objRef("User"),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_APPROVED,
			},
			Rule: reviewRule(anyStep(userReviewer(userA))),
			Review: &accessv1.Request_Status_Review{
				CurrentStep: 0,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	reqRef := umetav1.GetObjectReference(req)

	rev := createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_REJECT)
	assert.Nil(t, ctrl.OnAdd(ctx, rev))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
}

func TestReviewPendingNoRule(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	req, err := octeliumC.AccessC().CreateRequest(ctx, &accessv1.Request{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &accessv1.Request_Spec{Urgency: accessv1.Request_Spec_NORMAL},
		Status: &accessv1.Request_Status{
			UserRef: objRef("User"),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_PENDING,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	reqRef := umetav1.GetObjectReference(req)

	rev := createReview(t, ctx, octeliumC, reqRef, objRef("User"), 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.NotNil(t, ctrl.OnAdd(ctx, rev))
}

func TestReviewMissingRequestRef(t *testing.T) {
	ctx, ctrl, _ := newControllerTest(t)

	rev := &accessv1.Review{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &accessv1.Review_Spec{Decision: accessv1.Review_Spec_DECISION_APPROVE},
		Status:   &accessv1.Review_Status{UserRef: objRef("User")},
	}
	assert.NotNil(t, ctrl.OnAdd(ctx, rev))
}

func TestReviewMissingUserRef(t *testing.T) {
	ctx, ctrl, _ := newControllerTest(t)

	rev := &accessv1.Review{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &accessv1.Review_Spec{Decision: accessv1.Review_Spec_DECISION_APPROVE},
		Status:   &accessv1.Review_Status{RequestRef: objRef("Request")},
	}
	assert.NotNil(t, ctrl.OnAdd(ctx, rev))
}

func TestReviewDecisionUnset(t *testing.T) {
	ctx, ctrl, _ := newControllerTest(t)

	rev := &accessv1.Review{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &accessv1.Review_Spec{Decision: accessv1.Review_Spec_DECISION_UNSET},
		Status: &accessv1.Review_Status{
			RequestRef: objRef("Request"),
			UserRef:    objRef("User"),
		},
	}
	assert.Nil(t, ctrl.OnAdd(ctx, rev))
}

func TestReviewAllQuorumFromStore(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	userB := objRef("User")
	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(allStep(userReviewer(userA), userReviewer(userB))), 0)
	reqRef := umetav1.GetObjectReference(req)

	createReview(t, ctx, octeliumC, reqRef, userA, 0, accessv1.Review_Spec_DECISION_APPROVE)

	revB := createReview(t, ctx, octeliumC, reqRef, userB, 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revB))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, 1, len(reqG.Status.Review.LastSteps))
}

func TestReviewQuorumIgnoresOtherRequests(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	userA := objRef("User")
	userB := objRef("User")

	other := createPendingRequest(t, ctx, octeliumC,
		reviewRule(allStep(userReviewer(userA), userReviewer(userB))), 0)
	createReview(t, ctx, octeliumC, umetav1.GetObjectReference(other), userA, 0,
		accessv1.Review_Spec_DECISION_APPROVE)

	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(allStep(userReviewer(userA), userReviewer(userB))), 0)

	revB := createReview(t, ctx, octeliumC, umetav1.GetObjectReference(req), userB, 0,
		accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revB))

	assert.Equal(t, accessv1.Request_Status_State_PENDING,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestReviewAllQuorumGroupMembers(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	grp := createGroup(t, ctx, octeliumC)
	usrA := createUser(t, ctx, octeliumC, grp.Metadata.Name)
	usrB := createUser(t, ctx, octeliumC, grp.Metadata.Name)

	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(allStep(groupReviewer(umetav1.GetObjectReference(grp)))), 0)
	reqRef := umetav1.GetObjectReference(req)

	revA := createReview(t, ctx, octeliumC, reqRef, umetav1.GetObjectReference(usrA), 0,
		accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revA))
	assert.Equal(t, accessv1.Request_Status_State_PENDING,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)

	revB := createReview(t, ctx, octeliumC, reqRef, umetav1.GetObjectReference(usrB), 0,
		accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revB))
	assert.Equal(t, accessv1.Request_Status_State_APPROVED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestReviewAllQuorumGroupAndUserOverlap(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	grp := createGroup(t, ctx, octeliumC)
	usrA := createUser(t, ctx, octeliumC, grp.Metadata.Name)
	usrB := createUser(t, ctx, octeliumC, grp.Metadata.Name)

	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(allStep(
			userReviewer(umetav1.GetObjectReference(usrA)),
			groupReviewer(umetav1.GetObjectReference(grp)))), 0)
	reqRef := umetav1.GetObjectReference(req)

	revA := createReview(t, ctx, octeliumC, reqRef, umetav1.GetObjectReference(usrA), 0,
		accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revA))
	assert.Equal(t, accessv1.Request_Status_State_PENDING,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)

	revB := createReview(t, ctx, octeliumC, reqRef, umetav1.GetObjectReference(usrB), 0,
		accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, revB))
	assert.Equal(t, accessv1.Request_Status_State_APPROVED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestReviewAllQuorumEmptyGroup(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	grp := createGroup(t, ctx, octeliumC)
	usr := createUser(t, ctx, octeliumC, grp.Metadata.Name)
	emptyGrp := createGroup(t, ctx, octeliumC)

	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(allStep(
			groupReviewer(umetav1.GetObjectReference(grp)),
			groupReviewer(umetav1.GetObjectReference(emptyGrp)))), 0)

	rev := createReview(t, ctx, octeliumC, umetav1.GetObjectReference(req),
		umetav1.GetObjectReference(usr), 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, rev))

	assert.Equal(t, accessv1.Request_Status_State_APPROVED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestReviewAnyQuorumGroupSingleMember(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	grp := createGroup(t, ctx, octeliumC)
	usrA := createUser(t, ctx, octeliumC, grp.Metadata.Name)
	createUser(t, ctx, octeliumC, grp.Metadata.Name)

	req := createPendingRequest(t, ctx, octeliumC,
		reviewRule(anyStep(groupReviewer(umetav1.GetObjectReference(grp)))), 0)

	rev := createReview(t, ctx, octeliumC, umetav1.GetObjectReference(req),
		umetav1.GetObjectReference(usrA), 0, accessv1.Review_Spec_DECISION_APPROVE)
	assert.Nil(t, ctrl.OnAdd(ctx, rev))

	assert.Equal(t, accessv1.Request_Status_State_APPROVED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}
