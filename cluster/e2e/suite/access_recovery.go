package suite

import (
	"context"
	"net/http"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAccessControllerRecovery(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	c := newAccessCast(t, h)
	beta := h.NewPublicService(t, "default")

	short := serviceAutoApproveRule("short-grant", c.alpha)
	short.Authorization.MaxAccessDuration = eeharness.Seconds(120)
	h.CreateAccessPolicy(t, short, serviceReviewRule("queued-review", beta,
		userReviewStep(eeharness.UserReviewer(c.rita.User))))

	alphaProbe := h.Probe(t, c.alice.User, c.alpha)
	betaProbe := h.Probe(t, c.alice.User, beta)
	alphaProbe.MustBeDenied(t)
	betaProbe.MustBeDenied(t)

	pending := h.CreateRequest(t, c.alice, eeharness.ServiceRequest(beta, eeharness.Minutes(5)))
	h.WaitRequestState(t, pending, accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)
	req := h.CreateRequest(t, c.alice, eeharness.ServiceRequest(c.alpha, eeharness.Seconds(120)))
	approved := h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED,
		eeharness.RequestBudget)
	h.WaitRequestPolicyTrigger(t, req, eeharness.RequestBudget)
	require.NotNil(t, approved.Status.AccessEndsAt)
	alphaProbe.MustBeAllowed(t)

	restore := h.StopEnterpriseWithin(t, "nocturne", eeharness.PropagationBudget)
	require.True(t, approved.Status.AccessEndsAt.AsTime().After(time.Now()),
		"the access grant expired before nocturne was stopped")
	review := h.Review(t, c.rita, pending, accessv1.Review_Spec_DECISION_APPROVE)

	t.Run("AQueuedReviewDoesNotGrantAccessBeforeReconciliation", func(t *testing.T) {
		cur := h.GetRequest(t, pending)
		assert.Equal(t, accessv1.Request_Status_State_PENDING, cur.Status.State.Status)
		betaProbe.MustBeDenied(t)
	})

	t.Run("TheDataPlaneExpiresTheGrantWithoutTheController", func(t *testing.T) {
		h.Eventually(t, "the access grant deadline to pass", eeharness.IngestionBudget,
			func(ctx context.Context) error {
				if time.Now().Before(approved.Status.AccessEndsAt.AsTime()) {
					return errors.Errorf("the access grant has not expired")
				}
				return nil
			})
		alphaProbe.MustBeDenied(t)
		cur := h.GetRequest(t, req)
		assert.Equal(t, accessv1.Request_Status_State_APPROVED, cur.Status.State.Status)
		require.NotNil(t, cur.Status.PolicyTriggerRef)
	})

	restore()

	t.Run("TheExpiredGrantIsCleanedUpWithoutBeingRenewed", func(t *testing.T) {
		expired := h.WaitRequestState(t, req, accessv1.Request_Status_State_EXPIRED,
			eeharness.RequestBudget)
		assert.Equal(t, approved.Status.AccessEndsAt.AsTime(), expired.Status.AccessEndsAt.AsTime())
		h.WaitRequestNoPolicyTrigger(t, req, eeharness.RequestBudget)
		waitVisibilityRequestState(t, h, req, accessv1.Request_Status_State_EXPIRED, true)
		h.Consistently(t, "the expired grant to stay denied after recovery", 5*time.Second,
			func(ctx context.Context) error {
				got, err := alphaProbe.Status(ctx)
				if err != nil {
					return err
				}
				if got != http.StatusForbidden {
					return errUnexpectedStatus(got, http.StatusForbidden)
				}
				return nil
			})
	})

	t.Run("ThePersistedReviewIsAppliedAfterRecovery", func(t *testing.T) {
		cur := h.WaitRequestState(t, pending, accessv1.Request_Status_State_APPROVED,
			eeharness.RequestBudget)
		require.NotNil(t, cur.Status.Review)
		require.Len(t, cur.Status.Review.LastSteps, 1)
		assert.Equal(t, review.Metadata.Uid, cur.Status.Review.LastSteps[0].ReviewRef.Uid)
		h.WaitRequestPolicyTrigger(t, pending, eeharness.RequestBudget)
		betaProbe.MustBeAllowed(t)
		waitVisibilityRequestState(t, h, pending, accessv1.Request_Status_State_APPROVED, true)
	})
}
