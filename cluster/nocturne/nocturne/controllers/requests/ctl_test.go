// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package requests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
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

func matchAny() *accessv1.Policy_Spec_Rule_Condition {
	return &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
			MatchAny: true,
		},
	}
}

func durationHours(h int64) *metav1.Duration {
	return &metav1.Duration{
		Type: &metav1.Duration_Hours{
			Hours: uint32(h),
		},
	}
}

func catalogResource(ref *metav1.ObjectReference) *accessv1.Request_Spec_Resource {
	return &accessv1.Request_Spec_Resource{
		Type: &accessv1.Request_Spec_Resource_Catalog_{
			Catalog: &accessv1.Request_Spec_Resource_Catalog{
				CatalogRef: ref,
			},
		},
	}
}

func baseRequest(uref *metav1.ObjectReference) *accessv1.Request {
	return &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_NORMAL,
		},
		Status: &accessv1.Request_Status{
			UserRef: uref,
		},
	}
}

func createPolicy(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, disabled bool, rules ...*accessv1.Policy_Spec_Rule) *accessv1.Policy {
	pol, err := octeliumC.AccessC().CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{
			IsDisabled: disabled,
			Rules:      rules,
		},
	})
	assert.Nil(t, err, "%+v", err)
	return pol
}

func createRequest(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, req *accessv1.Request) *accessv1.Request {
	out, err := octeliumC.AccessC().CreateRequest(ctx, req)
	assert.Nil(t, err, "%+v", err)
	return out
}

func getRequest(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, uid string) *accessv1.Request {
	out, err := octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{Uid: uid})
	assert.Nil(t, err, "%+v", err)
	return out
}

func setupApproved(t *testing.T, ctx context.Context, ctrl *Controller, octeliumC octeliumc.ClientInterface) *accessv1.Request {
	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := baseRequest(objRef("User"))
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.Nil(t, ctrl.OnAdd(ctx, req))

	out := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, out.Status.State.Status)
	assert.NotNil(t, out.Status.PolicyTriggerRef)
	return out
}

func TestControllerAutoApprove(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := objRef("User")
	authzPol := utilrand.GetRandomStringCanonical(6)

	pol := createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies:          []string{authzPol},
			MaxAccessDuration: durationHours(1),
		},
	})

	req := baseRequest(uref)
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.Nil(t, ctrl.OnAdd(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.Rule)
	assert.Equal(t, accessv1.Policy_Spec_Rule_AUTO_APPROVE, reqG.Status.Rule.Effect)
	assert.NotNil(t, reqG.Status.PolicyRef)
	assert.Equal(t, pol.Metadata.Uid, reqG.Status.PolicyRef.Uid)
	assert.NotNil(t, reqG.Status.ApprovalStartAt)
	assert.NotNil(t, reqG.Status.ApprovalEndAt)
	assert.NotNil(t, reqG.Status.PolicyTriggerRef)

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: reqG.Status.PolicyTriggerRef.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, fmt.Sprintf("access-request-%s", req.Metadata.Uid), pt.Metadata.Name)
	assert.True(t, pt.Metadata.IsSystem)
	assert.NotNil(t, pt.Status.OwnerRef)
	assert.Equal(t, req.Metadata.Uid, pt.Status.OwnerRef.Uid)
	assert.Equal(t, []string{authzPol}, pt.Status.Policies)

	all := pt.Status.PreCondition.GetAll()
	assert.NotNil(t, all)
	assert.Equal(t, 2, len(all.Of))
	assert.NotNil(t, all.Of[0].GetUserRef())
	assert.Equal(t, uref.Uid, all.Of[0].GetUserRef().Uid)
	assert.NotNil(t, all.Of[1].GetNotAfter())
}

func TestControllerAutoApproveNoDuration(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := objRef("User")
	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := baseRequest(uref)
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.Nil(t, ctrl.OnAdd(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.PolicyTriggerRef)

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: reqG.Status.PolicyTriggerRef.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	all := pt.Status.PreCondition.GetAll()
	assert.NotNil(t, all)
	assert.Equal(t, 1, len(all.Of))
	assert.NotNil(t, all.Of[0].GetUserRef())
	assert.Equal(t, uref.Uid, all.Of[0].GetUserRef().Uid)
}

func TestControllerReview(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := objRef("User")
	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action: &accessv1.Policy_Spec_Rule_Action{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_{
				Review: &accessv1.Policy_Spec_Rule_Action_Review{
					Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
						{
							Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
								{
									Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
										User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
											UserRef: objRef("User"),
										},
									},
								},
							},
							OnApproval: accessv1.Policy_Spec_Rule_Action_Review_Step_ON_APPROVAL_APPROVE,
						},
					},
				},
			},
		},
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := baseRequest(uref)
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.Nil(t, ctrl.OnAdd(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.Rule)
	assert.Equal(t, accessv1.Policy_Spec_Rule_REVIEW, reqG.Status.Rule.Effect)
	assert.NotNil(t, reqG.Status.Review)
	assert.Equal(t, int32(0), reqG.Status.Review.CurrentStep)
	assert.NotNil(t, reqG.Status.ApprovalStartAt)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
}

func TestControllerDeny(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := objRef("User")
	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_DENY,
		Condition: matchAny(),
	})

	req := baseRequest(uref)
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.Nil(t, ctrl.OnAdd(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.Rule)
	assert.Equal(t, accessv1.Policy_Spec_Rule_DENY, reqG.Status.Rule.Effect)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
}

func TestControllerNoMatch(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	req := baseRequest(objRef("User"))
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.Nil(t, ctrl.OnAdd(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Nil(t, reqG.Status.Rule)
	assert.Nil(t, reqG.Status.PolicyRef)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
}

func TestControllerDisabledPolicySkipped(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, true, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := baseRequest(objRef("User"))
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.Nil(t, ctrl.OnAdd(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Nil(t, reqG.Status.Rule)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
}

func TestControllerReviewNoSteps(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action: &accessv1.Policy_Spec_Rule_Action{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_{
				Review: &accessv1.Policy_Spec_Rule_Action_Review{
					Steps: nil,
				},
			},
		},
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := baseRequest(objRef("User"))
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.NotNil(t, ctrl.OnAdd(ctx, req))
}

func TestControllerCELMatchErrors(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:   utilrand.GetRandomStringCanonical(6),
		Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: &accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_Match{
				Match: "true",
			},
		},
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := baseRequest(objRef("User"))
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.NotNil(t, ctrl.OnAdd(ctx, req))
}

func TestControllerPriorityOrder(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false,
		&accessv1.Policy_Spec_Rule{
			Name:      utilrand.GetRandomStringCanonical(6),
			Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
			Priority:  1,
			Condition: matchAny(),
			Authorization: &accessv1.Policy_Spec_Rule_Authorization{
				Policies: []string{utilrand.GetRandomStringCanonical(6)},
			},
		},
		&accessv1.Policy_Spec_Rule{
			Name:      utilrand.GetRandomStringCanonical(6),
			Effect:    accessv1.Policy_Spec_Rule_DENY,
			Priority:  10,
			Condition: matchAny(),
		},
	)

	req := baseRequest(objRef("User"))
	req.Spec.Resource = catalogResource(objRef("Catalog"))
	req = createRequest(t, ctx, octeliumC, req)

	assert.Nil(t, ctrl.OnAdd(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.Rule)
	assert.Equal(t, accessv1.Policy_Spec_Rule_DENY, reqG.Status.Rule.Effect)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
}

func TestControllerResourceCatalogCondition(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	catRef := objRef("Catalog")
	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:   utilrand.GetRandomStringCanonical(6),
		Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: &accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{
				Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{
					Type: &accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef{
						CatalogRef: catRef,
					},
				},
			},
		},
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	reqMatch := baseRequest(objRef("User"))
	reqMatch.Spec.Resource = catalogResource(catRef)
	reqMatch = createRequest(t, ctx, octeliumC, reqMatch)
	assert.Nil(t, ctrl.OnAdd(ctx, reqMatch))
	reqMatchG := getRequest(t, ctx, octeliumC, reqMatch.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)

	reqMiss := baseRequest(objRef("User"))
	reqMiss.Spec.Resource = catalogResource(objRef("Catalog"))
	reqMiss = createRequest(t, ctx, octeliumC, reqMiss)
	assert.Nil(t, ctrl.OnAdd(ctx, reqMiss))
	reqMissG := getRequest(t, ctx, octeliumC, reqMiss.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqMissG.Status.State.Status)
	assert.Nil(t, reqMissG.Status.Rule)
}

func TestControllerSubjectUserRefCondition(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	subjRef := objRef("User")
	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:   utilrand.GetRandomStringCanonical(6),
		Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: &accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{
				Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{
					Type: &accessv1.Policy_Spec_Rule_Condition_Subject_UserRef{
						UserRef: subjRef,
					},
				},
			},
		},
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	reqMatch := baseRequest(objRef("User"))
	reqMatch.Spec.Resource = catalogResource(objRef("Catalog"))
	reqMatch.Spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: subjRef,
		},
	}
	reqMatch = createRequest(t, ctx, octeliumC, reqMatch)
	assert.Nil(t, ctrl.OnAdd(ctx, reqMatch))
	reqMatchG := getRequest(t, ctx, octeliumC, reqMatch.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: reqMatchG.Status.PolicyTriggerRef.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, subjRef.Uid, pt.Status.PreCondition.GetAll().Of[0].GetUserRef().Uid)

	reqMiss := baseRequest(objRef("User"))
	reqMiss.Spec.Resource = catalogResource(objRef("Catalog"))
	reqMiss.Spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: objRef("User"),
		},
	}
	reqMiss = createRequest(t, ctx, octeliumC, reqMiss)
	assert.Nil(t, ctrl.OnAdd(ctx, reqMiss))
	reqMissG := getRequest(t, ctx, octeliumC, reqMiss.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqMissG.Status.State.Status)
}

func TestControllerIdempotent(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqApproved := setupApproved(t, ctx, ctrl, octeliumC)
	ptUid := reqApproved.Status.PolicyTriggerRef.Uid

	assert.Nil(t, ctrl.OnAdd(ctx, reqApproved))

	reqG := getRequest(t, ctx, octeliumC, reqApproved.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.PolicyTriggerRef)
	assert.Equal(t, ptUid, reqG.Status.PolicyTriggerRef.Uid)

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: ptUid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, ptUid, pt.Metadata.Uid)
}

func TestControllerRevoke(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqApproved := setupApproved(t, ctx, ctrl, octeliumC)
	ptUid := reqApproved.Status.PolicyTriggerRef.Uid

	reqApproved.Status.State = &accessv1.Request_Status_State{
		CreatedAt: pbutils.Now(),
		Status:    accessv1.Request_Status_State_REVOKED,
	}
	reqRevoked, err := octeliumC.AccessC().UpdateRequest(ctx, reqApproved)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, ctrl.OnUpdate(ctx, reqRevoked, reqApproved))

	reqG := getRequest(t, ctx, octeliumC, reqApproved.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REVOKED, reqG.Status.State.Status)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)

	_, err = octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: ptUid,
	})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestControllerOnDelete(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqApproved := setupApproved(t, ctx, ctrl, octeliumC)
	ptUid := reqApproved.Status.PolicyTriggerRef.Uid

	assert.Nil(t, ctrl.OnDelete(ctx, reqApproved))

	_, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: ptUid,
	})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestControllerGetAccessDuration(t *testing.T) {
	ctrl := &Controller{}

	none := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{}},
		&accessv1.Policy_Spec_Rule_Authorization{},
	)
	assert.Equal(t, time.Duration(0), none)

	maxOnly := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{}},
		&accessv1.Policy_Spec_Rule_Authorization{MaxAccessDuration: durationHours(2)},
	)
	assert.Equal(t, 2*time.Hour, maxOnly)

	reqOnly := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{Duration: durationHours(3)}},
		&accessv1.Policy_Spec_Rule_Authorization{},
	)
	assert.Equal(t, 3*time.Hour, reqOnly)

	clamped := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{Duration: durationHours(5)}},
		&accessv1.Policy_Spec_Rule_Authorization{MaxAccessDuration: durationHours(2)},
	)
	assert.Equal(t, 2*time.Hour, clamped)

	underMax := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{Duration: durationHours(1)}},
		&accessv1.Policy_Spec_Rule_Authorization{MaxAccessDuration: durationHours(4)},
	)
	assert.Equal(t, 1*time.Hour, underMax)
}
