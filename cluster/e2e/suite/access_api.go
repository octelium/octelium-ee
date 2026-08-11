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
	"fmt"
	"net/http"
	"testing"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	accessPortalService  = "access.octelium"
	consolePortalService = "console.octelium"
)

func testAccessAPIScoping(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	c := newAccessCast(t, h)
	ctx := t.Context()

	h.CreateAccessPolicy(t, c.reviewRule("scoping",
		&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				eeharness.UserReviewer(c.rita.User),
			},
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
		}))

	aliceReq := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	bobReq := h.CreateRequest(t, c.bob,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))

	t.Run("RequestersCannotUseTheMainService", func(t *testing.T) {
		_, err := h.AccessMainC(c.alice.Conn).ListPolicy(ctx, &accessv1.ListPolicyOptions{})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err))

		_, err = h.AccessMainC(c.alice.Conn).CreateCatalog(ctx, &accessv1.Catalog{
			Metadata: &metav1.Metadata{Name: h.Name()},
			Spec: &accessv1.Catalog_Spec{
				ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
					Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
						Services: []string{c.alpha.Metadata.Name},
					},
				},
			},
		})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err))
	})

	t.Run("RequestersOnlySeeTheirOwnRequests", func(t *testing.T) {
		res, err := h.AccessUserC(c.alice.Conn).ListRequest(ctx,
			&accessv1.ListUserRequestOptions{})
		require.Nil(t, err)

		assert.True(t, containsRequest(res.Items, aliceReq))
		assert.False(t, containsRequest(res.Items, bobReq))
	})

	t.Run("RequestersCannotReadAnotherRequest", func(t *testing.T) {
		_, err := h.AccessUserC(c.alice.Conn).GetRequest(ctx,
			&metav1.GetOptions{Uid: bobReq.Metadata.Uid})
		assert.NotNil(t, err)
	})

	t.Run("RequestersCannotCancelAnotherRequest", func(t *testing.T) {
		_, err := h.AccessUserC(c.alice.Conn).CancelRequest(ctx,
			&accessv1.CancelRequestRequest{
				RequestRef: umetav1.GetObjectReference(bobReq),
			})
		assert.NotNil(t, err)
	})

	t.Run("CatalogListingIsScoped", func(t *testing.T) {
		_, err := h.AccessUserC(c.alice.Conn).ListCatalog(ctx,
			&accessv1.ListUserCatalogOptions{})
		assert.Nil(t, err)

		_, err = h.AccessUserC(c.alice.Conn).ListCatalogService(ctx,
			&accessv1.ListUserCatalogServiceOptions{})
		assert.Nil(t, err)
	})

	t.Run("ReviewersCannotUseTheMainService", func(t *testing.T) {
		_, err := h.AccessMainC(c.rita.Conn).ListRequest(ctx, &accessv1.ListRequestOptions{})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err))
	})

	t.Run("TheAdministratorSeesEverything", func(t *testing.T) {
		res, err := h.AccessC().ListRequest(ctx, &accessv1.ListRequestOptions{})
		require.Nil(t, err)

		assert.True(t, containsRequest(res.Items, aliceReq))
		assert.True(t, containsRequest(res.Items, bobReq))
	})
}

func testAccessAuditTrail(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.reviewRule("audit",
		&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				eeharness.UserReviewer(c.rita.User),
			},
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
		}))

	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))
	h.WaitRequestState(t, req, accessv1.Request_Status_State_PENDING, eeharness.RequestBudget)

	h.Review(t, c.rita, req, accessv1.Review_Spec_DECISION_APPROVE)
	h.WaitRequestState(t, req, accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)

	t.Run("TheRequesterIsAudited", func(t *testing.T) {
		waitAuditLogMethod(t, h, c.alice, "CreateRequest")
	})

	t.Run("TheReviewerIsAudited", func(t *testing.T) {
		waitAuditLogMethod(t, h, c.rita, "CreateReview")
	})
}

func waitAuditLogMethod(t *testing.T, h *eeharness.H, a *eeharness.Actor, method string) {
	t.Helper()

	h.Eventually(t, fmt.Sprintf("the audit log to carry %s", method), eeharness.IngestionBudget,
		func(ctx context.Context) error {
			res, err := h.AuditLogC().ListAuditLog(ctx, &visibilityv1.ListAuditLogRequest{
				UserRef: umetav1.GetObjectReference(a.User),
			})
			if err != nil {
				return err
			}

			for _, itm := range res.Items {
				if itm.Entry.Method == method {
					if itm.Entry.UserRef.Uid != a.User.Metadata.Uid {
						return errors.Errorf("the entry belongs to another User")
					}
					return nil
				}
			}

			return errors.Errorf("no %s entry among %d audit log entries",
				method, len(res.Items))
		})
}

func testAccessPortal(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	svc, err := h.CoreC().GetService(ctx, &metav1.GetOptions{Name: accessPortalService})
	require.Nil(t, err)

	a := h.NewActorWithAuthorization(t, &corev1.User_Spec_Authorization{
		InlinePolicies: eeharness.ServicePolicy("access-portal", svc),
	})

	t.Run("RequiresAuthentication", func(t *testing.T) {
		h.GetStatus(t, h.HTTPPublic(accessPortalService), "/", http.StatusUnauthorized)
	})

	t.Run("DeniedForAnUnauthorizedSession", func(t *testing.T) {
		other := h.CreateWorkloadUser(t, nil)
		h.WaitGetStatus(t, h.HTTPPublicToken(accessPortalService, h.AccessToken(t, other)),
			"/", http.StatusForbidden)
	})

	t.Run("ServedToAnAuthorizedSession", func(t *testing.T) {
		h.WaitGetStatus(t, h.HTTPPublicToken(accessPortalService, a.Token), "/",
			http.StatusOK)
	})
}

func testConsolePortal(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	h.GetStatus(t, h.HTTPPublic(consolePortalService), "/", http.StatusUnauthorized)
}
