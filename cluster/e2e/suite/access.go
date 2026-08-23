// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type accessCast struct {
	h *eeharness.H

	sre      *corev1.Group
	security *corev1.Group

	alice   *eeharness.Actor
	bob     *eeharness.Actor
	rita    *eeharness.Actor
	raj     *eeharness.Actor
	rosa    *eeharness.Actor
	mallory *eeharness.Actor

	alpha *corev1.Service

	alphaCatalog *accessv1.Catalog
}

func newAccessCast(t *testing.T, h *eeharness.H) *accessCast {
	t.Helper()

	ret := &accessCast{h: h}

	ret.sre = h.CreateGroup(t, nil)
	ret.security = h.CreateGroup(t, nil)

	ret.alice = h.NewActor(t, ret.sre.Metadata.Name)
	ret.bob = h.NewActor(t)
	ret.rita = h.NewActor(t)
	ret.raj = h.NewActor(t)
	ret.rosa = h.NewActor(t, ret.security.Metadata.Name)
	ret.mallory = h.NewActor(t)

	ret.alpha = h.NewPublicService(t, "default")

	ret.alphaCatalog = h.CreateCatalog(t, []string{ret.alpha.Metadata.Name}, nil)

	return ret
}

func (c *accessCast) autoApproveRule(name string) *accessv1.Policy_Spec_Rule {
	return &accessv1.Policy_Spec_Rule{
		Name:      name,
		Priority:  0,
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: eeharness.CatalogResource(c.alphaCatalog),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			InlinePolicies:    harness.InlineAllowAny("granted"),
			MaxAccessDuration: eeharness.Minutes(5),
		},
	}
}

func (c *accessCast) reviewRule(name string,
	steps ...*accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Policy_Spec_Rule {
	return c.reviewRuleFor(name, c.alphaCatalog, steps...)
}

func (c *accessCast) reviewRuleFor(name string, cat *accessv1.Catalog,
	steps ...*accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Policy_Spec_Rule {
	return &accessv1.Policy_Spec_Rule{
		Name:      name,
		Priority:  0,
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: eeharness.CatalogResource(cat),
		Action: &accessv1.Policy_Spec_Rule_Action{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_{
				Review: &accessv1.Policy_Spec_Rule_Action_Review{Steps: steps},
			},
		},
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			InlinePolicies:    harness.InlineAllowAny("granted"),
			MaxAccessDuration: eeharness.Minutes(5),
		},
	}
}

func testAccessAutoApprove(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, &accessv1.Policy_Spec_Rule{
		Name:     "auto-approve-sre",
		Priority: 0,
		Effect:   accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: eeharness.AllOf(
			eeharness.GroupSubject(c.sre),
			eeharness.CatalogResource(c.alphaCatalog)),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			InlinePolicies:    harness.InlineAllowAny("granted"),
			MaxAccessDuration: eeharness.Minutes(5),
		},
	})

	alphaProbe := h.Probe(t, c.alice.User, c.alpha)
	alphaProbe.MustBeDenied(t)

	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))

	approved := h.WaitRequestState(t, req,
		accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)

	t.Run("ApprovedWithoutAReview", func(t *testing.T) {
		assert.Nil(t, approved.Status.Review)
		assert.NotNil(t, approved.Status.PolicyTriggerRef)
		assert.NotNil(t, approved.Status.AccessEndsAt)
		assert.True(t, approved.Status.AccessEndsAt.AsTime().After(time.Now()))
	})

	t.Run("TheDataPlaneGrantsAccess", func(t *testing.T) {
		alphaProbe.MustBeAllowed(t)
	})

	t.Run("TheGrantIsScopedToTheCatalog", func(t *testing.T) {
		beta := h.NewPublicService(t, "default")
		h.Probe(t, c.alice.User, beta).MustBeDenied(t)
	})

	t.Run("OtherUsersAreUnaffected", func(t *testing.T) {
		h.Probe(t, c.bob.User, c.alpha).MustBeDenied(t)
	})
}

func testAccessSingleStepReview(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.reviewRule("single-step",
		&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				eeharness.UserReviewer(c.rita.User),
				eeharness.GroupReviewer(c.security),
			},
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
		}))

	probe := h.Probe(t, c.alice.User, c.alpha)
	probe.MustBeDenied(t)

	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))

	pending := h.WaitRequestState(t, req,
		accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)
	require.NotNil(t, pending)

	ctx := t.Context()

	t.Run("VisibleToTheReviewers", func(t *testing.T) {
		res, err := h.AccessReviewerC(c.rita.Conn).ListRequest(ctx,
			&accessv1.ListReviewerRequestOptions{})
		require.Nil(t, err)
		assert.True(t, containsRequest(res.Items, req))

		res, err = h.AccessReviewerC(c.rosa.Conn).ListRequest(ctx,
			&accessv1.ListReviewerRequestOptions{})
		require.Nil(t, err)
		assert.True(t, containsRequest(res.Items, req))
	})

	t.Run("HiddenFromEveryoneElse", func(t *testing.T) {
		res, err := h.AccessReviewerC(c.mallory.Conn).ListRequest(ctx,
			&accessv1.ListReviewerRequestOptions{})
		require.Nil(t, err)
		assert.False(t, containsRequest(res.Items, req))
	})

	t.Run("AnUnrelatedUserCannotReview", func(t *testing.T) {
		_, err := h.AccessReviewerC(c.mallory.Conn).CreateReview(ctx,
			eeharness.ReviewOf(req, accessv1.Review_Spec_DECISION_APPROVE))
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err))
	})

	review := h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)

	approved := h.WaitRequestState(t, req,
		accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)

	t.Run("TheReviewIsLinkedToTheRequest", func(t *testing.T) {
		require.NotNil(t, approved.Status.Review)
		require.True(t, len(approved.Status.Review.LastSteps) > 0)
		assert.Equal(t, review.Metadata.Uid,
			approved.Status.Review.LastSteps[0].ReviewRef.Uid)
		assert.Equal(t, req.Metadata.Uid, review.Status.RequestRef.Uid)
	})

	t.Run("TheDataPlaneGrantsAccess", func(t *testing.T) {
		probe.MustBeAllowed(t)
	})

	t.Run("VisibleToTheAdministrator", func(t *testing.T) {
		res, err := h.AccessC().ListReview(ctx, &accessv1.ListReviewOptions{})
		require.Nil(t, err)

		found := false
		for _, itm := range res.Items {
			if itm.Metadata.Uid == review.Metadata.Uid {
				found = true
			}
		}
		assert.True(t, found)
	})
}

func testAccessMultiStepQuorum(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.reviewRule("two-step",
		&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				eeharness.UserReviewer(c.rita.User),
				eeharness.UserReviewer(c.raj.User),
			},
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ALL,
		},
		&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				eeharness.GroupReviewer(c.security),
			},
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
		}))

	probe := h.Probe(t, c.alice.User, c.alpha)
	probe.MustBeDenied(t)

	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))

	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)

	h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)

	t.Run("OneOfTwoIsNotEnough", func(t *testing.T) {
		cur := h.GetRequest(t, req)
		assert.Equal(t, accessv1.Request_Status_State_PENDING, cur.Status.State.Status)
		if cur.Status.Review != nil {
			assert.Equal(t, int32(0), cur.Status.Review.CurrentStep)
		}
	})

	h.Review(t, c.raj, req, accessv1.Review_Spec_DECISION_APPROVE)

	t.Run("TheSecondStepOpens", func(t *testing.T) {
		cur := h.WaitRequestStep(t, req, 1, eeharness.RequestBudget)
		assert.NotNil(t, cur.Status.Review.CurrentStepStartedAt)
	})

	t.Run("AStepZeroReviewerCannotCloseStepOne", func(t *testing.T) {
		_, err := h.AccessReviewerC(c.rita.Conn).CreateReview(t.Context(),
			eeharness.ReviewOf(req, accessv1.Review_Spec_DECISION_APPROVE))
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err))
	})

	h.Review(t, c.rosa, req, accessv1.Review_Spec_DECISION_APPROVE)

	h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)

	t.Run("TheDataPlaneGrantsAccess", func(t *testing.T) {
		probe.MustBeAllowed(t)
	})
}

func testAccessRejectCancelRevoke(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.reviewRule("reject-cancel",
		&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				eeharness.UserReviewer(c.rita.User),
			},
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
		}))

	probe := h.Probe(t, c.alice.User, c.alpha)
	ctx := t.Context()

	t.Run("Rejection", func(t *testing.T) {
		req := h.CreateRequest(t, c.alice,
			eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
		h.WaitRequestState(t, req,
			accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)

		h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_REJECT)

		rejected := h.WaitRequestState(t, req,
			accessv1.Request_Status_State_REJECTED, eeharness.RequestBudget)
		assert.Nil(t, rejected.Status.PolicyTriggerRef)

		probe.MustStayDenied(t, settleWindow)
	})

	t.Run("Cancellation", func(t *testing.T) {
		req := h.CreateRequest(t, c.alice,
			eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
		h.WaitRequestState(t, req,
			accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)

		_, err := h.AccessUserC(c.alice.Conn).CancelRequest(ctx,
			&accessv1.CancelRequestRequest{RequestRef: umetav1.GetObjectReference(req)})
		require.Nil(t, err)

		cancelled := h.WaitRequestState(t, req,
			accessv1.Request_Status_State_CANCELLED, eeharness.RequestBudget)
		assert.Nil(t, cancelled.Status.PolicyTriggerRef)
	})

	t.Run("Revocation", func(t *testing.T) {
		req := h.CreateRequest(t, c.alice,
			eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
		h.WaitRequestState(t, req,
			accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)

		review := h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
		h.WaitRequestState(t, req,
			accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)

		probe.MustBeAllowed(t)

		_, err := h.AccessC().RevokeRequest(ctx,
			&accessv1.RevokeRequestRequest{RequestRef: umetav1.GetObjectReference(req)})
		require.Nil(t, err)

		revoked := h.WaitRequestState(t, req,
			accessv1.Request_Status_State_REVOKED, eeharness.RequestBudget)
		h.WaitRequestNoPolicyTrigger(t, req, eeharness.RequestBudget)

		probe.MustBeDenied(t)

		t.Run("TheAuditTrailSurvives", func(t *testing.T) {
			require.NotNil(t, revoked.Status.Review)
			require.True(t, len(revoked.Status.Review.LastSteps) > 0)
			assert.Equal(t, review.Metadata.Uid,
				revoked.Status.Review.LastSteps[0].ReviewRef.Uid)

			states := []accessv1.Request_Status_State_Status{}
			for _, itm := range revoked.Status.LastStates {
				states = append(states, itm.Status)
			}
			assert.Contains(t, states, accessv1.Request_Status_State_APPROVED)
		})

		t.Run("RevokingAgainIsANoOp", func(t *testing.T) {
			_, err := h.AccessC().RevokeRequest(ctx,
				&accessv1.RevokeRequestRequest{RequestRef: umetav1.GetObjectReference(req)})
			require.Nil(t, err)
		})
	})

	t.Run("Deletion", func(t *testing.T) {
		req := h.CreateRequest(t, c.alice,
			eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
		h.WaitRequestState(t, req,
			accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)

		h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
		h.WaitRequestState(t, req,
			accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)

		probe.MustBeAllowed(t)

		_, err := h.AccessC().DeleteRequest(ctx,
			&metav1.DeleteOptions{Uid: req.Metadata.Uid})
		require.Nil(t, err)

		probe.MustBeDenied(t)
	})
}

func testAccessTimeouts(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	c := newAccessCast(t, h)

	rejectCatalog := h.CreateCatalog(t, []string{c.alpha.Metadata.Name}, nil)
	expiryCatalog := h.CreateCatalog(t, []string{c.alpha.Metadata.Name}, nil)

	h.CreateAccessPolicy(t,
		c.reviewRuleFor("timeout-next", c.alphaCatalog,
			&accessv1.Policy_Spec_Rule_Action_Review_Step{
				Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
					eeharness.UserReviewer(c.rita.User),
				},
				ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
				Timeout:             eeharness.Seconds(30),
				OnTimeout: accessv1.
					Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_GOTO_NEXT_STEP,
			},
			&accessv1.Policy_Spec_Rule_Action_Review_Step{
				Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
					eeharness.UserReviewer(c.raj.User),
				},
				ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
			}),
		c.reviewRuleFor("timeout-reject", rejectCatalog,
			&accessv1.Policy_Spec_Rule_Action_Review_Step{
				Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
					eeharness.UserReviewer(c.rita.User),
				},
				ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
				Timeout:             eeharness.Seconds(30),
				OnTimeout:           accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_REJECT,
			}),
		&accessv1.Policy_Spec_Rule{
			Name:      "auto-approve-short",
			Priority:  0,
			Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
			Condition: eeharness.CatalogResource(expiryCatalog),
			Authorization: &accessv1.Policy_Spec_Rule_Authorization{
				InlinePolicies:    harness.InlineAllowAny("granted"),
				MaxAccessDuration: eeharness.Seconds(60),
			},
		})

	probe := h.Probe(t, c.alice.User, c.alpha)
	probe.MustBeDenied(t)

	nextReq := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	rejectReq := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(rejectCatalog, eeharness.Minutes(5)))
	expiryReq := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(expiryCatalog, eeharness.Seconds(60)))

	h.WaitRequestState(t, nextReq,
		accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)
	h.WaitRequestState(t, rejectReq,
		accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)

	approved := h.WaitRequestState(t, expiryReq,
		accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)

	t.Run("TheApprovalCarriesAnAccessDeadline", func(t *testing.T) {
		require.NotNil(t, approved.Status.AccessEndsAt)
		assert.True(t, approved.Status.AccessEndsAt.AsTime().
			Before(time.Now().Add(2*time.Minute)))
	})

	t.Run("TheDataPlaneGrantsAccess", func(t *testing.T) {
		probe.MustBeAllowed(t)
	})

	t.Run("TheStepAdvancesOnTimeout", func(t *testing.T) {
		h.WaitRequestStep(t, nextReq, 1, eeharness.TimeoutBudget)
	})

	t.Run("TheRequestIsRejectedOnTimeout", func(t *testing.T) {
		h.WaitRequestState(t, rejectReq,
			accessv1.Request_Status_State_REJECTED, eeharness.TimeoutBudget)
	})

	t.Run("TheRequestExpires", func(t *testing.T) {
		expired := h.WaitRequestState(t, expiryReq,
			accessv1.Request_Status_State_EXPIRED, eeharness.TimeoutBudget)
		assert.Nil(t, expired.Status.PolicyTriggerRef)
	})

	t.Run("TheDataPlaneRevokesAccess", func(t *testing.T) {
		probe.MustBeDenied(t)
	})
}

func containsRequest(items []*accessv1.Request, req *accessv1.Request) bool {
	for _, itm := range items {
		if itm.Metadata.Uid == req.Metadata.Uid {
			return true
		}
	}
	return false
}
