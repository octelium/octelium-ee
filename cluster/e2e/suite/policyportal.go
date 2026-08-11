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
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isAuthorized(t *testing.T, h *eeharness.H,
	usr *corev1.User, svc *corev1.Service) bool {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	res, err := h.PolicyPortalC().IsAuthorized(ctx, &enterprisev1.IsAuthorizedRequest{
		Downstream: &enterprisev1.IsAuthorizedRequest_UserRef{
			UserRef: umetav1.GetObjectReference(usr),
		},
		Upstream: &enterprisev1.IsAuthorizedRequest_ServiceRef{
			ServiceRef: umetav1.GetObjectReference(svc),
		},
	})
	require.Nil(t, err)

	return res.IsAuthorized
}

func waitAuthorization(t *testing.T, h *eeharness.H,
	usr *corev1.User, svc *corev1.Service, want bool) {
	t.Helper()

	h.Eventually(t, fmt.Sprintf("the PolicyPortal to answer %v", want),
		eeharness.PropagationBudget, func(ctx context.Context) error {
			res, err := h.PolicyPortalC().IsAuthorized(ctx, &enterprisev1.IsAuthorizedRequest{
				Downstream: &enterprisev1.IsAuthorizedRequest_UserRef{
					UserRef: umetav1.GetObjectReference(usr),
				},
				Upstream: &enterprisev1.IsAuthorizedRequest_ServiceRef{
					ServiceRef: umetav1.GetObjectReference(svc),
				},
			})
			if err != nil {
				return err
			}
			if res.IsAuthorized != want {
				return errors.Errorf("the PolicyPortal says %v, want %v",
					res.IsAuthorized, want)
			}
			return nil
		})
}

func waitParity(t *testing.T, h *eeharness.H,
	usr *corev1.User, svc *corev1.Service, probe *eeharness.Probe, want bool) {
	t.Helper()

	wantStatus := http.StatusForbidden
	if want {
		wantStatus = http.StatusOK
	}

	h.Eventually(t, fmt.Sprintf("the data plane and the PolicyPortal to agree on %v", want),
		eeharness.PropagationBudget, func(ctx context.Context) error {
			got, err := probe.Status(ctx)
			if err != nil {
				return err
			}
			if got != wantStatus {
				return errUnexpectedStatus(got, wantStatus)
			}

			res, err := h.PolicyPortalC().IsAuthorized(ctx, &enterprisev1.IsAuthorizedRequest{
				Downstream: &enterprisev1.IsAuthorizedRequest_UserRef{
					UserRef: umetav1.GetObjectReference(usr),
				},
				Upstream: &enterprisev1.IsAuthorizedRequest_ServiceRef{
					ServiceRef: umetav1.GetObjectReference(svc),
				},
			})
			if err != nil {
				return err
			}
			if res.IsAuthorized != want {
				return errors.Errorf("the PolicyPortal says %v, the data plane says %v",
					res.IsAuthorized, want)
			}

			return nil
		})
}

func testPolicyPortalParity(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	svc := h.NewPublicService(t, "default")

	t.Run("UserPolicy", func(t *testing.T) {
		usr := h.CreateWorkloadUser(t, nil)
		probe := h.Probe(t, usr, svc)

		waitParity(t, h, usr, svc, probe, false)

		usr.Spec.Authorization = &corev1.User_Spec_Authorization{
			InlinePolicies: harness.InlineAllowAny("allow"),
		}
		usr = h.UpdateUser(t, usr)

		waitParity(t, h, usr, svc, probe, true)
	})

	t.Run("GroupPolicy", func(t *testing.T) {
		grp := h.CreateGroup(t, nil)
		usr := h.CreateUser(t, &corev1.User{
			Spec: &corev1.User_Spec{
				Type:   corev1.User_Spec_WORKLOAD,
				Groups: []string{grp.Metadata.Name},
			},
		})
		probe := h.Probe(t, usr, svc)

		waitParity(t, h, usr, svc, probe, false)

		grp.Spec.Authorization = &corev1.Group_Spec_Authorization{
			InlinePolicies: harness.InlineAllowAny("allow"),
		}
		h.UpdateGroup(t, grp)

		waitParity(t, h, usr, svc, probe, true)
	})

	t.Run("ServicePolicy", func(t *testing.T) {
		usr := h.CreateWorkloadUser(t, nil)
		other := h.NewPublicService(t, "default")
		probe := h.Probe(t, usr, other)

		waitParity(t, h, usr, other, probe, false)

		other.Spec.Authorization = &corev1.Service_Spec_Authorization{
			InlinePolicies: harness.InlineAllowAny("allow"),
		}
		other = h.UpdateService(t, other)

		waitParity(t, h, usr, other, probe, true)
	})

	t.Run("DisabledUser", func(t *testing.T) {
		usr := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
			InlinePolicies: harness.InlineAllowAny("allow"),
		})
		probe := h.Probe(t, usr, svc)

		waitParity(t, h, usr, svc, probe, true)

		usr.Spec.IsDisabled = true
		usr = h.UpdateUser(t, usr)

		waitParity(t, h, usr, svc, probe, false)
	})
}

func testPolicyPortalUnderChurn(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	svc := h.NewPublicService(t, "default")
	usr := h.CreateWorkloadUser(t, nil)
	probe := h.Probe(t, usr, svc)

	for range 3 {
		usr.Spec.Authorization = &corev1.User_Spec_Authorization{
			InlinePolicies: harness.InlineAllowAny("allow"),
		}
		usr = h.UpdateUser(t, usr)
		waitParity(t, h, usr, svc, probe, true)

		usr.Spec.Authorization = nil
		usr = h.UpdateUser(t, usr)
		waitParity(t, h, usr, svc, probe, false)
	}
}

func testPolicyPortalWithPolicyTrigger(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	c := newAccessCast(t, h)

	h.CreateAccessPolicy(t, c.autoApproveRule("portal-auto-approve"))

	probe := h.Probe(t, c.alice.User, c.alpha)

	assert.False(t, isAuthorized(t, h, c.alice.User, c.alpha))

	req := h.CreateRequest(t, c.alice,
		eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))

	h.WaitRequestState(t, req,
		accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)

	waitParity(t, h, c.alice.User, c.alpha, probe, true)
}
