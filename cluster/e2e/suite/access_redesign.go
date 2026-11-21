// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"fmt"
	"testing"

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

func testAccessNameOnlyReferences(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.autoApproveRule("name-only"))

	probe := h.Probe(t, c.alice.User, c.alpha)
	probe.MustBeDenied(t)

	req := eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5))
	req.Spec.Resource.GetCatalog().CatalogRef = &metav1.ObjectReference{
		Name: c.alphaCatalog.Metadata.Name,
	}

	created := h.CreateRequest(t, c.alice, req)
	approved := h.WaitRequestState(t, created, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)
	approved = h.WaitRequestPolicyTrigger(t, approved, eeharness.RequestBudget)

	t.Run("TheStoredReferencesAreCanonical", func(t *testing.T) {
		assert.Equal(t, c.alphaCatalog.Metadata.Uid, approved.Spec.Resource.GetCatalog().GetCatalogRef().GetUid())
		assert.Equal(t, c.alice.User.Metadata.Uid, approved.Spec.Subject.GetUserRef().GetUid())
		assert.Equal(t, "full", approved.Spec.Resource.GetCatalog().GetAccessLevel())
	})

	t.Run("TheDataPlaneGrantsAccess", func(t *testing.T) {
		probe.MustBeAllowed(t)
	})
}

func testAccessSystemServicesAreNotRequestable(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	const apiService = "default.octelium-api"

	catalog := h.CreateCatalog(t, []string{apiService, c.alpha.Metadata.Name}, nil)
	h.CreateAccessPolicy(t, &accessv1.Policy_Spec_Rule{
		Name:      "auto-approve-everything",
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{MatchAny: true}},
	})

	t.Run("TheServiceShorthandIsRejected", func(t *testing.T) {
		_, err := h.AccessUserC(c.mallory.Conn).CreateRequest(t.Context(), &accessv1.Request{
			Metadata: &metav1.Metadata{},
			Spec: &accessv1.Request_Spec{
				Resource: &accessv1.Request_Spec_Resource{
					Type: &accessv1.Request_Spec_Resource_ServiceRef{
						ServiceRef: &metav1.ObjectReference{Name: apiService},
					},
				},
			},
		})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	})

	req := h.CreateRequest(t, c.mallory, eeharness.CatalogRequest(catalog, eeharness.Minutes(5)))
	approved := h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)

	t.Run("TheCatalogScopeExcludesTheSystemServices", func(t *testing.T) {
		require.Len(t, approved.Status.ResolvedScope.GetServiceRefs(), 1)
		assert.Equal(t, c.alpha.Metadata.Uid, approved.Status.ResolvedScope.ServiceRefs[0].Uid)
		assert.EqualValues(t, 1, approved.Status.ResolvedScope.TotalExcludedService)
	})
}

func testAccessGuardrailsWinByDefault(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	guarded := h.NewActorWithAuthorization(t, &corev1.User_Spec_Authorization{
		InlinePolicies: append(
			eeharness.APIPolicy("access-api", "octelium.api.main.access.v1.UserService"),
			&corev1.InlinePolicy{
				Name: "guardrail",
				Spec: &corev1.Policy_Spec{
					Rules: []*corev1.Policy_Spec_Rule{
						harness.MatchRule("deny-alpha", 0, corev1.Policy_Spec_Rule_DENY,
							fmt.Sprintf(`ctx.service.metadata.name == %q`, c.alpha.Metadata.Name)),
					},
				},
			}),
	})

	overriding := int32(-1)
	breakGlass := h.CreateCatalog(t, []string{c.alpha.Metadata.Name}, nil)
	breakGlass = h.UpdateCatalogLevels(t, breakGlass, &accessv1.Catalog_Spec_AccessLevel{
		Name: "break-glass",
		Type: &accessv1.Catalog_Spec_AccessLevel_Full_{
			Full: &accessv1.Catalog_Spec_AccessLevel_Full{Priority: &overriding},
		},
	})

	h.CreateAccessPolicy(t, &accessv1.Policy_Spec_Rule{
		Name:      "auto-approve",
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{MatchAny: true}},
	})

	probe := h.Probe(t, guarded.User, c.alpha)
	probe.MustBeDenied(t)

	defaultReq := h.CreateRequest(t, guarded, eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	defaultApproved := h.WaitRequestState(t, defaultReq, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)
	h.WaitRequestPolicyTrigger(t, defaultApproved, eeharness.RequestBudget)

	t.Run("ADefaultPriorityGrantDoesNotOverrideTheGuardrail", func(t *testing.T) {
		assert.False(t, defaultApproved.Status.ResolvedScope.GetOverridesGuardrails())
		probe.MustStayDenied(t, settleWindow)
	})

	overrideReq := h.CreateRequest(t, guarded, eeharness.CatalogRequest(breakGlass, eeharness.Minutes(5)))
	overrideApproved := h.WaitRequestState(t, overrideReq, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)
	h.WaitRequestPolicyTrigger(t, overrideApproved, eeharness.RequestBudget)

	t.Run("AnExplicitlyOverridingLevelWins", func(t *testing.T) {
		assert.True(t, overrideApproved.Status.ResolvedScope.GetOverridesGuardrails())
		assert.EqualValues(t, -1, overrideApproved.Status.ResolvedScope.GetPriority())
		probe.MustBeAllowed(t)
	})

	t.Run("EndingTheAccessRevokesIt", func(t *testing.T) {
		ended, err := h.AccessUserC(guarded.Conn).EndAccess(t.Context(), &accessv1.EndAccessRequest{
			RequestRef: umetav1.GetObjectReference(overrideApproved),
		})
		require.Nil(t, err)
		assert.Equal(t, accessv1.Request_Status_State_ENDED_EARLY, ended.Status.State.Reason)
		probe.MustBeDenied(t)
	})
}

func testAccessQuestionsBlockApproval(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.reviewRule("questions-block-approval",
		userReviewStep(eeharness.UserReviewer(c.rita.User))))

	probe := h.Probe(t, c.alice.User, c.alpha)
	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING,
		eeharness.RequestBudget)

	_, err := h.AccessReviewerC(c.rita.Conn).RequestInformation(t.Context(), &accessv1.RequestInformationRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Message:    "Which tables do you need?",
	})
	require.Nil(t, err)

	t.Run("AnUnansweredQuestionBlocksTheApproval", func(t *testing.T) {
		_, err := h.AccessReviewerC(c.rita.Conn).SetReviewDecision(t.Context(),
			eeharness.ReviewOf(h.GetRequest(t, req), accessv1.Review_Spec_DECISION_APPROVE))
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
		probe.MustStayDenied(t, settleWindow)
	})

	_, err = h.AccessUserC(c.alice.Conn).RespondToRequest(t.Context(), &accessv1.RespondToRequestRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Response:   "Only the invoices table",
	})
	require.Nil(t, err)

	t.Run("TheAnswerUnblocksTheApproval", func(t *testing.T) {
		h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
		h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
			eeharness.RequestBudget)
		probe.MustBeAllowed(t)
	})
}
