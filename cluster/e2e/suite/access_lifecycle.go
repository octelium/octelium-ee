// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serviceReviewRule(name string, svc *corev1.Service,
	steps ...*accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Policy_Spec_Rule {
	return &accessv1.Policy_Spec_Rule{
		Name:      name,
		Priority:  0,
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: eeharness.ServiceResource(svc),
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

func serviceAutoApproveRule(name string, svc *corev1.Service) *accessv1.Policy_Spec_Rule {
	return &accessv1.Policy_Spec_Rule{
		Name:      name,
		Priority:  0,
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: eeharness.ServiceResource(svc),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			InlinePolicies:    harness.InlineAllowAny("granted"),
			MaxAccessDuration: eeharness.Minutes(5),
		},
	}
}

func userReviewStep(reviewers ...*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Reviewers:           reviewers,
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
	}
}

func testAccessDirectService(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, serviceReviewRule("direct-service", c.alpha,
		userReviewStep(eeharness.UserReviewer(c.rita.User))))

	probe := h.Probe(t, c.alice.User, c.alpha)
	other := h.NewPublicService(t, "default")
	otherProbe := h.Probe(t, c.alice.User, other)

	probe.MustBeDenied(t)
	otherProbe.MustBeDenied(t)

	req := h.CreateRequest(t, c.alice,
		eeharness.ServiceRequest(c.alpha, eeharness.Minutes(5)))
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
		eeharness.RequestBudget)

	h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
	approved := h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)

	t.Run("TheRequestTargetsTheService", func(t *testing.T) {
		assert.Equal(t, c.alpha.Metadata.Uid, approved.Spec.Resource.GetServiceRef().Uid)
		require.NotNil(t, approved.Status.PolicyTriggerRef)
	})

	t.Run("OnlyTheRequestedServiceIsAllowed", func(t *testing.T) {
		probe.MustBeAllowed(t)
		otherProbe.MustStayDenied(t, settleWindow)
	})

	_, err := h.AccessC().RevokeRequest(t.Context(),
		&accessv1.RevokeRequestRequest{RequestRef: umetav1.GetObjectReference(req)})
	require.Nil(t, err)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_REVOKED,
		eeharness.RequestBudget)
	probe.MustBeDenied(t)
}

func testAccessDelegatedSubject(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	rule := c.reviewRule("delegated-subject",
		userReviewStep(eeharness.UserReviewer(c.rita.User)))
	rule.Condition = eeharness.AllOf(
		eeharness.CatalogResource(c.alphaCatalog),
		eeharness.UserSubject(c.bob.User))
	h.CreateAccessPolicy(t, rule)

	reqSpec := eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5))
	reqSpec.Spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: umetav1.GetObjectReference(c.bob.User),
		},
	}
	req := h.CreateRequestForSubject(t, c.alice, reqSpec)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
		eeharness.RequestBudget)

	h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
	approved := h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)

	t.Run("RequesterAndSubjectRemainDistinct", func(t *testing.T) {
		assert.Equal(t, c.alice.User.Metadata.Uid, approved.Status.UserRef.Uid)
		assert.Equal(t, c.bob.User.Metadata.Uid, approved.Spec.Subject.GetUserRef().Uid)
	})

	t.Run("OnlyTheSubjectReceivesAccess", func(t *testing.T) {
		h.Probe(t, c.bob.User, c.alpha).MustBeAllowed(t)
		h.Probe(t, c.alice.User, c.alpha).MustStayDenied(t, settleWindow)
	})

	t.Run("TheRequesterOwnsTheRequest", func(t *testing.T) {
		alice, err := h.AccessUserC(c.alice.Conn).ListRequest(t.Context(),
			&accessv1.ListUserRequestOptions{})
		require.Nil(t, err)
		assert.True(t, containsRequest(alice.Items, req))

		bob, err := h.AccessUserC(c.bob.Conn).ListRequest(t.Context(),
			&accessv1.ListUserRequestOptions{})
		require.Nil(t, err)
		assert.False(t, containsRequest(bob.Items, req))
	})
}

func testAccessOverlappingGrants(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, serviceAutoApproveRule("overlapping-grants", c.alpha))
	probe := h.Probe(t, c.alice.User, c.alpha)
	probe.MustBeDenied(t)

	first := h.CreateRequest(t, c.alice,
		eeharness.ServiceRequest(c.alpha, eeharness.Minutes(5)))
	second := h.CreateRequest(t, c.alice,
		eeharness.ServiceRequest(c.alpha, eeharness.Minutes(5)))

	h.WaitRequestState(t, first, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)
	h.WaitRequestState(t, second, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)

	first = h.WaitRequestPolicyTrigger(t, first, eeharness.RequestBudget)
	second = h.WaitRequestPolicyTrigger(t, second, eeharness.RequestBudget)
	assert.NotEqual(t, first.Status.PolicyTriggerRef.Uid, second.Status.PolicyTriggerRef.Uid)
	probe.MustBeAllowed(t)

	_, err := h.AccessC().RevokeRequest(t.Context(),
		&accessv1.RevokeRequestRequest{RequestRef: umetav1.GetObjectReference(first)})
	require.Nil(t, err)
	h.WaitRequestState(t, first, accessv1.Request_Status_State_REVOKED,
		eeharness.RequestBudget)
	probe.MustStayAllowed(t, settleWindow)

	_, err = h.AccessC().RevokeRequest(t.Context(),
		&accessv1.RevokeRequestRequest{RequestRef: umetav1.GetObjectReference(second)})
	require.Nil(t, err)
	h.WaitRequestState(t, second, accessv1.Request_Status_State_REVOKED,
		eeharness.RequestBudget)
	probe.MustBeDenied(t)
}

func testAccessReviewRevision(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.reviewRule("review-revision",
		&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				eeharness.UserReviewer(c.rita.User),
				eeharness.UserReviewer(c.raj.User),
			},
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT,
			ApprovalCount:       2,
		}))

	probe := h.Probe(t, c.alice.User, c.alpha)
	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
		eeharness.RequestBudget)

	ritaReview := h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
	_, err := h.AccessReviewerC(c.rita.Conn).CancelReview(t.Context(),
		&accessv1.CancelReviewRequest{ReviewRef: umetav1.GetObjectReference(ritaReview)})
	require.Nil(t, err)

	h.Review(t, c.raj, req, accessv1.Review_Spec_DECISION_APPROVE)
	probe.MustStayDenied(t, settleWindow)
	assert.Equal(t, accessv1.Request_Status_State_PENDING,
		h.GetRequest(t, req).Status.State.Status)

	ritaReview, err = h.AccessC().GetReview(t.Context(),
		&metav1.GetOptions{Uid: ritaReview.Metadata.Uid})
	require.Nil(t, err)
	assert.Equal(t, accessv1.Review_Spec_DECISION_UNSET, ritaReview.Spec.Decision)
	require.NotEmpty(t, ritaReview.Status.LastRevisions)
	assert.Equal(t, accessv1.Review_Spec_DECISION_APPROVE,
		ritaReview.Status.LastRevisions[0].Spec.Decision)

	ritaReview.Spec.Decision = accessv1.Review_Spec_DECISION_APPROVE
	ritaReview.Spec.Justification = "e2e revised"
	ritaReview, err = h.AccessReviewerC(c.rita.Conn).UpdateReview(t.Context(), ritaReview)
	require.Nil(t, err)

	h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)
	probe.MustBeAllowed(t)

	ritaReview.Spec.Decision = accessv1.Review_Spec_DECISION_REJECT
	_, err = h.AccessReviewerC(c.rita.Conn).UpdateReview(t.Context(), ritaReview)
	require.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err))

	t.Run("ARevisionCanRejectBeforeQuorum", func(t *testing.T) {
		req := h.CreateRequest(t, c.alice,
			eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
		h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
			eeharness.RequestBudget)

		review := h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
		review.Spec.Decision = accessv1.Review_Spec_DECISION_REJECT
		review.Spec.Justification = "e2e rejected"
		_, err := h.AccessReviewerC(c.rita.Conn).UpdateReview(t.Context(), review)
		require.Nil(t, err)

		h.WaitRequestState(t, req, accessv1.Request_Status_State_REJECTED,
			eeharness.RequestBudget)
	})
}

func testAccessPendingDeadline(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.reviewRule("pending-deadline",
		userReviewStep(eeharness.UserReviewer(c.rita.User))))

	request := eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5))
	request.Spec.Deadline = pbutils.Timestamp(time.Now().Add(10 * time.Second))
	req := h.CreateRequest(t, c.alice, request)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
		eeharness.RequestBudget)

	expired := h.WaitRequestState(t, req, accessv1.Request_Status_State_EXPIRED,
		eeharness.RequestBudget)
	assert.Nil(t, expired.Status.PolicyTriggerRef)
	h.Probe(t, c.alice.User, c.alpha).MustStayDenied(t, settleWindow)

	_, err := h.AccessReviewerC(c.rita.Conn).CreateReview(t.Context(),
		eeharness.ReviewOf(req, accessv1.Review_Spec_DECISION_APPROVE))
	require.NotNil(t, err)
}

func testAccessReviewerMembership(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.reviewRule("reviewer-membership",
		userReviewStep(eeharness.GroupReviewer(c.security))))

	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
		eeharness.RequestBudget)

	waitReviewerRequest(t, h, c.rosa, req, true)

	c.rosa.User.Spec.Groups = nil
	c.rosa.User = h.UpdateUser(t, c.rosa.User)
	c.rosa.Conn = h.UserConn(t, c.rosa.User)
	waitReviewerRequest(t, h, c.rosa, req, false)

	_, err := h.AccessReviewerC(c.rosa.Conn).CreateReview(t.Context(),
		eeharness.ReviewOf(req, accessv1.Review_Spec_DECISION_APPROVE))
	require.NotNil(t, err)
	assert.True(t, grpcerr.IsUnauthorized(err))

	c.mallory.User.Spec.Groups = []string{c.security.Metadata.Name}
	c.mallory.User = h.UpdateUser(t, c.mallory.User)
	c.mallory.Conn = h.UserConn(t, c.mallory.User)
	waitReviewerRequest(t, h, c.mallory, req, true)

	h.Review(t, c.mallory, req, accessv1.Review_Spec_DECISION_APPROVE)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)
	h.Probe(t, c.alice.User, c.alpha).MustBeAllowed(t)
}

func waitReviewerRequest(t *testing.T, h *eeharness.H, reviewer *eeharness.Actor,
	req *accessv1.Request, want bool) {
	t.Helper()

	h.Eventually(t, "the reviewer Request visibility to converge",
		eeharness.PropagationBudget, func(ctx context.Context) error {
			res, err := h.AccessReviewerC(reviewer.Conn).ListRequest(ctx,
				&accessv1.ListReviewerRequestOptions{})
			if err != nil {
				return err
			}
			if got := containsRequest(res.Items, req); got != want {
				return errors.Errorf("the Request visibility is %v, want %v", got, want)
			}
			return nil
		})
}

func testAccessCatalogResources(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	ns := h.EnsureTestNamespace(t)
	inNamespace := h.NewPublicService(t, ns.Metadata.Name)
	outside := h.NewPublicService(t, "default")
	catalog := h.CreateCatalog(t, []string{c.alpha.Metadata.Name},
		[]string{ns.Metadata.Name})

	h.CreateAccessPolicy(t, c.reviewRuleFor("catalog-resources", catalog,
		userReviewStep(eeharness.UserReviewer(c.rita.User))))

	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(catalog, eeharness.Minutes(5)))
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
		eeharness.RequestBudget)
	h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)

	h.Probe(t, c.alice.User, c.alpha).MustBeAllowed(t)
	h.Probe(t, c.alice.User, inNamespace).MustBeAllowed(t)
	h.Probe(t, c.alice.User, outside).MustStayDenied(t, settleWindow)
}

func testAccessUIDBinding(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, serviceAutoApproveRule("uid-binding", c.alpha))
	req := h.CreateRequest(t, c.alice,
		eeharness.ServiceRequest(c.alpha, eeharness.Minutes(5)))
	h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)
	h.Probe(t, c.alice.User, c.alpha).MustBeAllowed(t)

	name := c.alpha.Metadata.Name
	spec := pbutils.Clone(c.alpha.Spec).(*corev1.Service_Spec)
	h.DeleteService(t, c.alpha)

	replacement := h.CreateService(t, &corev1.Service{
		Metadata: &metav1.Metadata{Name: name},
		Spec:     spec,
	})
	h.MustWaitService(t, replacement.Metadata.Name)
	assert.NotEqual(t, c.alpha.Metadata.Uid, replacement.Metadata.Uid)
	h.Probe(t, c.alice.User, replacement).MustStayDenied(t, settleWindow)
}
