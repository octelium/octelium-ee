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
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func newControllerTest(t *testing.T) (context.Context, *Controller, octeliumc.ClientInterface) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	celEngine, err := celengine.New(ctx, nil)
	assert.Nil(t, err)

	ctrl, err := NewController(ctx, tst.C.OcteliumC, celEngine)
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

func serviceRef(svc *corev1.Service) *metav1.ObjectReference {
	return &metav1.ObjectReference{
		ApiVersion: "core/v1",
		Kind:       "Service",
		Name:       svc.Metadata.Name,
		Uid:        svc.Metadata.Uid,
	}
}

func fakeServiceRef() *metav1.ObjectReference {
	return objRef("Service")
}

func createUserRef(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface) *metav1.ObjectReference {
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

	return &metav1.ObjectReference{
		ApiVersion: "core/v1",
		Kind:       "User",
		Name:       usr.Metadata.Name,
		Uid:        usr.Metadata.Uid,
	}
}

func createService(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface) *corev1.Service {
	svc, err := octeliumC.CoreC().CreateService(ctx, &corev1.Service{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   &corev1.Service_Spec{},
		Status: &corev1.Service_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	return svc
}

func matchAny() *accessv1.Policy_Spec_Rule_Condition {
	return &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
			MatchAny: true,
		},
	}
}

func celMatch(exp string) *accessv1.Policy_Spec_Rule_Condition {
	return &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_Match{
			Match: exp,
		},
	}
}

func requesterCondition(ref *metav1.ObjectReference) *accessv1.Policy_Spec_Rule_Condition {
	return &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_UserRef{
			UserRef: ref,
		},
	}
}

func allOf(conds ...*accessv1.Policy_Spec_Rule_Condition) *accessv1.Policy_Spec_Rule_Condition {
	return &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_All_{
			All: &accessv1.Policy_Spec_Rule_Condition_All{
				Of: conds,
			},
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

func serviceResource(ref *metav1.ObjectReference) *accessv1.Request_Spec_Resource {
	return &accessv1.Request_Spec_Resource{
		Type: &accessv1.Request_Spec_Resource_ServiceRef{
			ServiceRef: ref,
		},
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

func anyStep(reviewers ...*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Reviewers:           reviewers,
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
	}
}

func reviewAction(steps ...*accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Policy_Spec_Rule_Action {
	return &accessv1.Policy_Spec_Rule_Action{
		Type: &accessv1.Policy_Spec_Rule_Action_Review_{
			Review: &accessv1.Policy_Spec_Rule_Action_Review{
				Steps: steps,
			},
		},
	}
}

func baseRequest(uref *metav1.ObjectReference, svcRef *metav1.ObjectReference) *accessv1.Request {
	return &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Request_Spec{
			Urgency:  accessv1.Request_Spec_NORMAL,
			Resource: serviceResource(svcRef),
		},
		Status: &accessv1.Request_Status{
			UserRef: uref,
		},
	}
}

func createPolicy(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, disabled bool, rules ...*accessv1.Policy_Spec_Rule) *accessv1.Policy {
	return createNamedPolicy(t, ctx, octeliumC, utilrand.GetRandomStringCanonical(8), disabled, rules...)
}

func createNamedPolicy(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, name string, disabled bool, rules ...*accessv1.Policy_Spec_Rule) *accessv1.Policy {
	pol, err := octeliumC.AccessC().CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{
			Name: name,
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

func converge(t *testing.T, ctx context.Context, ctrl *Controller, octeliumC octeliumc.ClientInterface, uid string) *accessv1.Request {
	for i := 0; i < 4; i++ {
		req := getRequest(t, ctx, octeliumC, uid)
		assert.Nil(t, ctrl.OnAdd(ctx, req), "pass %d", i)
	}
	return getRequest(t, ctx, octeliumC, uid)
}

func TestRequestAutoApprove(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := objRef("User")
	svc := createService(t, ctx, octeliumC)
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

	req := createRequest(t, ctx, octeliumC, baseRequest(uref, serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.Rule)
	assert.Equal(t, accessv1.Policy_Spec_Rule_AUTO_APPROVE, reqG.Status.Rule.Effect)
	assert.NotNil(t, reqG.Status.PolicyRef)
	assert.Equal(t, pol.Metadata.Uid, reqG.Status.PolicyRef.Uid)
	assert.NotNil(t, reqG.Status.ApprovalStartAt)
	assert.NotNil(t, reqG.Status.ApprovalEndAt)
	assert.NotNil(t, reqG.Status.AccessEndsAt)
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
	assert.True(t, len(pt.Status.InlinePolicies) >= 1)
	assert.Equal(t, "access-request-default", pt.Status.InlinePolicies[0].Name)

	all := pt.Status.PreCondition.GetAll()
	assert.NotNil(t, all)
	assert.Equal(t, 3, len(all.Of))
	assert.NotNil(t, all.Of[0].GetUserRef())
	assert.Equal(t, uref.Uid, all.Of[0].GetUserRef().Uid)
	assert.NotNil(t, all.Of[1].GetServiceRef())
	assert.Equal(t, svc.Metadata.Uid, all.Of[1].GetServiceRef().Uid)
	assert.NotNil(t, all.Of[2].GetNotAfter())
	assert.Equal(t, reqG.Status.AccessEndsAt.AsTime(), all.Of[2].GetNotAfter().AsTime())
}

func TestRequestAutoApproveNoAuthorization(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := objRef("User")
	svc := createService(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(uref, serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.PolicyTriggerRef)

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: reqG.Status.PolicyTriggerRef.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Empty(t, pt.Status.Policies)
	assert.Equal(t, 1, len(pt.Status.InlinePolicies))
	assert.Equal(t, "access-request-default", pt.Status.InlinePolicies[0].Name)

	all := pt.Status.PreCondition.GetAll()
	assert.NotNil(t, all)
	assert.Equal(t, 3, len(all.Of))
	assert.NotNil(t, all.Of[0].GetUserRef())
	assert.NotNil(t, all.Of[1].GetServiceRef())
	assert.Equal(t, svc.Metadata.Uid, all.Of[1].GetServiceRef().Uid)
	assert.NotNil(t, all.Of[2].GetNotAfter())
}

func TestRequestAutoApproveNoDuration(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := objRef("User")
	svc := createService(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(uref, serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.AccessEndsAt)
	assert.Equal(t, reqG.Status.ApprovalEndAt.AsTime().Add(defaultMaxAccessDuration),
		reqG.Status.AccessEndsAt.AsTime())
	assert.NotNil(t, reqG.Status.PolicyTriggerRef)

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: reqG.Status.PolicyTriggerRef.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	all := pt.Status.PreCondition.GetAll()
	assert.NotNil(t, all)
	assert.Equal(t, 3, len(all.Of))
	assert.NotNil(t, all.Of[0].GetUserRef())
	assert.NotNil(t, all.Of[1].GetServiceRef())
	assert.Equal(t, svc.Metadata.Uid, all.Of[1].GetServiceRef().Uid)
	assert.NotNil(t, all.Of[2].GetNotAfter())
	assert.Equal(t, reqG.Status.AccessEndsAt.AsTime(), all.Of[2].GetNotAfter().AsTime())
}

func TestRequestAutoApprovePendingCarriesAccessEndsAt(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := objRef("User")
	svc := createService(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			MaxAccessDuration: durationHours(1),
		},
	})

	pending := baseRequest(uref, serviceRef(svc))
	pending.Status.State = &accessv1.Request_Status_State{
		CreatedAt: pbutils.Now(),
		Status:    accessv1.Request_Status_State_PENDING,
	}

	req := createRequest(t, ctx, octeliumC, pending)
	assert.Nil(t, ctrl.OnAdd(ctx, getRequest(t, ctx, octeliumC, req.Metadata.Uid)))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.Rule)
	assert.NotNil(t, reqG.Status.AccessEndsAt)
	assert.Equal(t, reqG.Status.ApprovalEndAt.AsTime().Add(time.Hour),
		reqG.Status.AccessEndsAt.AsTime())

	reqG = converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.PolicyTriggerRef)
}

func TestRequestReview(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := objRef("User")
	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(anyStep(userReviewer(objRef("User")))),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(uref, fakeServiceRef()))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.Rule)
	assert.Equal(t, accessv1.Policy_Spec_Rule_REVIEW, reqG.Status.Rule.Effect)
	assert.NotNil(t, reqG.Status.Review)
	assert.Equal(t, int32(0), reqG.Status.Review.CurrentStep)
	assert.NotNil(t, reqG.Status.ApprovalStartAt)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
	assert.Nil(t, reqG.Status.AccessEndsAt)
}

func TestRequestDeny(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_DENY,
		Condition: matchAny(),
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), fakeServiceRef()))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.Rule)
	assert.Equal(t, accessv1.Policy_Spec_Rule_DENY, reqG.Status.Rule.Effect)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
}

func TestRequestNoMatch(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), fakeServiceRef()))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Nil(t, reqG.Status.Rule)
	assert.Nil(t, reqG.Status.PolicyRef)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
}

func TestRequestDisabledPolicySkipped(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, true, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), fakeServiceRef()))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Nil(t, reqG.Status.Rule)
}

func TestRequestPriorityOrder(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)

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

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Policy_Spec_Rule_AUTO_APPROVE, reqG.Status.Rule.Effect)
}

func TestRequestPriorityAcrossPolicies(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)
	requester := createUserRef(t, ctx, octeliumC)

	defaultPol := createNamedPolicy(t, ctx, octeliumC,
		fmt.Sprintf("a-%s", utilrand.GetRandomStringCanonical(6)), false,
		&accessv1.Policy_Spec_Rule{
			Name:      "default",
			Effect:    accessv1.Policy_Spec_Rule_REVIEW,
			Condition: matchAny(),
			Action:    reviewAction(anyStep(userReviewer(objRef("User")))),
		})

	autoPol := createNamedPolicy(t, ctx, octeliumC,
		fmt.Sprintf("z-%s", utilrand.GetRandomStringCanonical(6)), false,
		&accessv1.Policy_Spec_Rule{
			Name:     utilrand.GetRandomStringCanonical(6),
			Effect:   accessv1.Policy_Spec_Rule_AUTO_APPROVE,
			Priority: -1,
			Condition: allOf(
				requesterCondition(requester),
				celMatch(`time.isWeekdayInTZ(now(), "Africa/Cairo") || time.isWeekendInTZ(now(), "Africa/Cairo")`),
			),
		})

	reqMatch := createRequest(t, ctx, octeliumC, baseRequest(requester, serviceRef(svc)))
	reqMatchG := converge(t, ctx, ctrl, octeliumC, reqMatch.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)
	assert.Equal(t, autoPol.Metadata.Uid, reqMatchG.Status.PolicyRef.Uid)
	assert.Equal(t, autoPol.Spec.Rules[0].Name, reqMatchG.Status.Rule.Name)
	assert.Nil(t, reqMatchG.Status.Review)

	reqMiss := createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), serviceRef(svc)))
	reqMissG := converge(t, ctx, ctrl, octeliumC, reqMiss.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqMissG.Status.State.Status)
	assert.Equal(t, defaultPol.Metadata.Uid, reqMissG.Status.PolicyRef.Uid)
	assert.Equal(t, "default", reqMissG.Status.Rule.Name)
}

func TestRequestPriorityTieBreak(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)

	createNamedPolicy(t, ctx, octeliumC,
		fmt.Sprintf("z-%s", utilrand.GetRandomStringCanonical(6)), false,
		&accessv1.Policy_Spec_Rule{
			Name:      "last-policy",
			Effect:    accessv1.Policy_Spec_Rule_DENY,
			Condition: matchAny(),
		})

	firstPol := createNamedPolicy(t, ctx, octeliumC,
		fmt.Sprintf("a-%s", utilrand.GetRandomStringCanonical(6)), false,
		&accessv1.Policy_Spec_Rule{
			Name:      "first-rule",
			Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
			Condition: matchAny(),
		},
		&accessv1.Policy_Spec_Rule{
			Name:      "second-rule",
			Effect:    accessv1.Policy_Spec_Rule_DENY,
			Condition: matchAny(),
		})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, firstPol.Metadata.Uid, reqG.Status.PolicyRef.Uid)
	assert.Equal(t, "first-rule", reqG.Status.Rule.Name)
}

func TestGetSortedPolicyRules(t *testing.T) {
	newPol := func(name string, disabled bool, rules ...*accessv1.Policy_Spec_Rule) *accessv1.Policy {
		return &accessv1.Policy{
			Metadata: &metav1.Metadata{
				Name: name,
			},
			Spec: &accessv1.Policy_Spec{
				IsDisabled: disabled,
				Rules:      rules,
			},
		}
	}

	newRule := func(name string, priority int32) *accessv1.Policy_Spec_Rule {
		return &accessv1.Policy_Spec_Rule{
			Name:     name,
			Priority: priority,
		}
	}

	rules := getSortedPolicyRules([]*accessv1.Policy{
		newPol("default", false, newRule("default", 0)),
		nil,
		newPol("usr1-weekday-auto-approve", false,
			newRule("usr1-weekdays-cairo", -1), nil, newRule("usr1-late", 5)),
		newPol("disabled", true, newRule("disabled", -100)),
		newPol("break-glass", false, newRule("deny-all", 0), newRule("deny-first", -10)),
	})

	var names []string
	for _, itm := range rules {
		names = append(names, fmt.Sprintf("%s/%s", itm.pol.Metadata.Name, itm.rule.Name))
	}

	assert.Equal(t, []string{
		"break-glass/deny-first",
		"usr1-weekday-auto-approve/usr1-weekdays-cairo",
		"break-glass/deny-all",
		"default/default",
		"usr1-weekday-auto-approve/usr1-late",
	}, names)
}

func TestRequestSubjectUserRefCondition(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)
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

	reqMatch := baseRequest(objRef("User"), serviceRef(svc))
	reqMatch.Spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: subjRef,
		},
	}
	reqMatch = createRequest(t, ctx, octeliumC, reqMatch)
	reqMatchG := converge(t, ctx, ctrl, octeliumC, reqMatch.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: reqMatchG.Status.PolicyTriggerRef.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, subjRef.Uid, pt.Status.PreCondition.GetAll().Of[0].GetUserRef().Uid)
	assert.Equal(t, svc.Metadata.Uid, pt.Status.PreCondition.GetAll().Of[1].GetServiceRef().Uid)

	reqMiss := baseRequest(objRef("User"), serviceRef(svc))
	reqMiss.Spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: objRef("User"),
		},
	}
	reqMiss = createRequest(t, ctx, octeliumC, reqMiss)
	reqMissG := converge(t, ctx, ctrl, octeliumC, reqMiss.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqMissG.Status.State.Status)
}

func TestRequestUserRefCondition(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)
	requester := objRef("User")

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:   utilrand.GetRandomStringCanonical(6),
		Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: &accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_UserRef{
				UserRef: requester,
			},
		},
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	reqMatch := createRequest(t, ctx, octeliumC, baseRequest(requester, serviceRef(svc)))
	reqMatchG := converge(t, ctx, ctrl, octeliumC, reqMatch.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)

	reqMiss := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqMissG := converge(t, ctx, ctrl, octeliumC, reqMiss.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqMissG.Status.State.Status)
}

func TestRequestReviewNoSteps(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), fakeServiceRef()))
	assert.NotNil(t, ctrl.OnAdd(ctx, req))
}

func TestRequestReviewUnsetRequirement(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action: reviewAction(&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				userReviewer(objRef("User")),
			},
		}),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), fakeServiceRef()))
	assert.NotNil(t, ctrl.OnAdd(ctx, req))
}

func TestRequestReviewCountZero(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action: reviewAction(&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				userReviewer(objRef("User")),
			},
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT,
		}),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), fakeServiceRef()))
	assert.NotNil(t, ctrl.OnAdd(ctx, req))
}

func TestRequestCELMatch(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)

	pol := createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: celMatch("true"),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies: []string{utilrand.GetRandomStringCanonical(6)},
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, pol.Metadata.Uid, reqG.Status.PolicyRef.Uid)
}

func TestRequestCELMatchFallsThrough(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false,
		&accessv1.Policy_Spec_Rule{
			Name:      utilrand.GetRandomStringCanonical(6),
			Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
			Priority:  -1,
			Condition: celMatch("false"),
		},
		&accessv1.Policy_Spec_Rule{
			Name:      utilrand.GetRandomStringCanonical(6),
			Effect:    accessv1.Policy_Spec_Rule_DENY,
			Condition: matchAny(),
		},
	)

	req := createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), fakeServiceRef()))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Policy_Spec_Rule_DENY, reqG.Status.Rule.Effect)
}

func TestRequestCELMatchContext(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)

	userRef := createUserRef(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:   utilrand.GetRandomStringCanonical(6),
		Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: celMatch(fmt.Sprintf(
			`ctx.user.metadata.name == "%s" && ctx.request.spec.resource.serviceRef.uid == "%s"`,
			userRef.Name, svc.Metadata.Uid)),
	})

	reqMatch := createRequest(t, ctx, octeliumC, baseRequest(userRef, serviceRef(svc)))
	reqMatchG := converge(t, ctx, ctrl, octeliumC, reqMatch.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)

	reqOtherSvc := createRequest(t, ctx, octeliumC, baseRequest(userRef, fakeServiceRef()))
	reqOtherSvcG := converge(t, ctx, ctrl, octeliumC, reqOtherSvc.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqOtherSvcG.Status.State.Status)

	reqOtherUser := createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), serviceRef(svc)))
	reqOtherUserG := converge(t, ctx, ctrl, octeliumC, reqOtherUser.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqOtherUserG.Status.State.Status)
	assert.Nil(t, reqOtherUserG.Status.Rule)
}

func TestRequestCELMatchInvalid(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: celMatch("ctx.request.("),
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), fakeServiceRef()))
	assert.NotNil(t, ctrl.OnAdd(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Nil(t, reqG.Status.Rule)
}

func TestRequestExpire(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies:          []string{utilrand.GetRandomStringCanonical(6)},
			MaxAccessDuration: durationHours(1),
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqApproved := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	assert.NotNil(t, reqApproved.Status.PolicyTriggerRef)
	ptUid := reqApproved.Status.PolicyTriggerRef.Uid

	reqApproved.Status.AccessEndsAt = pbutils.Timestamp(time.Now().Add(-2 * time.Hour))
	_, err := octeliumC.AccessC().UpdateRequest(ctx, reqApproved)
	assert.Nil(t, err, "%+v", err)

	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED, reqG.Status.State.Status)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)

	_, err = octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{Uid: ptUid})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestRequestRevoke(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies:          []string{utilrand.GetRandomStringCanonical(6)},
			MaxAccessDuration: durationHours(1),
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqApproved := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	ptUid := reqApproved.Status.PolicyTriggerRef.Uid

	reqApproved.Status.State = &accessv1.Request_Status_State{
		CreatedAt: pbutils.Now(),
		Status:    accessv1.Request_Status_State_REVOKED,
	}
	_, err := octeliumC.AccessC().UpdateRequest(ctx, reqApproved)
	assert.Nil(t, err, "%+v", err)

	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REVOKED, reqG.Status.State.Status)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)

	_, err = octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{Uid: ptUid})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestRequestOnDelete(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies:          []string{utilrand.GetRandomStringCanonical(6)},
			MaxAccessDuration: durationHours(1),
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqApproved := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	ptUid := reqApproved.Status.PolicyTriggerRef.Uid

	assert.Nil(t, ctrl.OnDelete(ctx, reqApproved))

	_, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{Uid: ptUid})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestRequestAnchorNoSliding(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			Policies:          []string{utilrand.GetRandomStringCanonical(6)},
			MaxAccessDuration: durationHours(1),
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqA := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	ptUidA := reqA.Status.PolicyTriggerRef.Uid
	accessEndsA := reqA.Status.AccessEndsAt.AsTime()

	reqB := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqB.Status.State.Status)
	assert.Equal(t, ptUidA, reqB.Status.PolicyTriggerRef.Uid)
	assert.Equal(t, accessEndsA, reqB.Status.AccessEndsAt.AsTime())

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{Uid: ptUidA})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, accessEndsA, pt.Status.PreCondition.GetAll().Of[2].GetNotAfter().AsTime())
	assert.Equal(t, svc.Metadata.Uid, pt.Status.PreCondition.GetAll().Of[1].GetServiceRef().Uid)
}

func TestRequestGetAccessDuration(t *testing.T) {
	ctrl := &Controller{}

	none := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{}},
		&accessv1.Policy_Spec_Rule_Authorization{},
	)
	assert.Equal(t, defaultMaxAccessDuration, none)

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

	clampedToDefault := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{Duration: durationHours(72)}},
		&accessv1.Policy_Spec_Rule_Authorization{},
	)
	assert.Equal(t, defaultMaxAccessDuration, clampedToDefault)

	aboveDefault := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{Duration: durationHours(72)}},
		&accessv1.Policy_Spec_Rule_Authorization{MaxAccessDuration: durationHours(96)},
	)
	assert.Equal(t, 72*time.Hour, aboveDefault)

	clamped := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{Duration: durationHours(5)}},
		&accessv1.Policy_Spec_Rule_Authorization{MaxAccessDuration: durationHours(2)},
	)
	assert.Equal(t, 2*time.Hour, clamped)

	nilAuthz := ctrl.getAccessDuration(
		&accessv1.Request{Spec: &accessv1.Request_Spec{Duration: durationHours(1)}},
		nil,
	)
	assert.Equal(t, 1*time.Hour, nilAuthz)
}

func autoApprovedRequest(t *testing.T, ctx context.Context, ctrl *Controller,
	octeliumC octeliumc.ClientInterface) *accessv1.Request {

	svc := createService(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Authorization: &accessv1.Policy_Spec_Rule_Authorization{
			MaxAccessDuration: durationHours(1),
		},
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(objRef("User"), serviceRef(svc)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.NotNil(t, reqG.Status.PolicyTriggerRef)

	return reqG
}

func policyTriggerExists(t *testing.T, ctx context.Context,
	octeliumC octeliumc.ClientInterface, req *accessv1.Request) bool {

	_, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Name: fmt.Sprintf("access-request-%s", req.Metadata.Uid),
	})
	if err != nil {
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
		return false
	}

	return true
}

func TestRequestDeleteRevokesPolicyTriggerWithoutRef(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	stale := pbutils.Clone(reqG).(*accessv1.Request)
	stale.Status.PolicyTriggerRef = nil

	_, err := octeliumC.AccessC().DeleteRequest(ctx, &rmetav1.DeleteOptions{
		Uid: reqG.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, ctrl.OnDelete(ctx, stale))
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, reqG))
}

func TestRequestDeleteIsNotUndoneByALateReconcile(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	_, err := octeliumC.AccessC().DeleteRequest(ctx, &rmetav1.DeleteOptions{
		Uid: reqG.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, ctrl.OnDelete(ctx, reqG))
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, reqG))

	assert.Nil(t, ctrl.OnUpdate(ctx, reqG, reqG))
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, reqG))
}

func TestRequestReconcileIsIdempotent(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	req := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	for i := 0; i < 4; i++ {
		before := getRequest(t, ctx, octeliumC, req.Metadata.Uid)

		ptBefore, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
			Name: policyTriggerName(before),
		})
		assert.Nil(t, err, "%+v", err)

		assert.Nil(t, ctrl.OnAdd(ctx, before), "pass %d", i)

		after := getRequest(t, ctx, octeliumC, req.Metadata.Uid)

		ptAfter, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
			Name: policyTriggerName(before),
		})
		assert.Nil(t, err, "%+v", err)

		assert.Equal(t, before.Metadata.ResourceVersion, after.Metadata.ResourceVersion, "pass %d", i)
		assert.Equal(t, ptBefore.Metadata.ResourceVersion, ptAfter.Metadata.ResourceVersion, "pass %d", i)
	}
}

func TestRequestRefsHaveNoResourceVersion(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	req := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	assert.Empty(t, req.Status.PolicyTriggerRef.ResourceVersion)
	assert.Empty(t, req.Status.PolicyRef.ResourceVersion)

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Name: policyTriggerName(req),
	})
	assert.Nil(t, err, "%+v", err)

	assert.Empty(t, pt.Status.OwnerRef.ResourceVersion)

	all := pt.Status.PreCondition.GetAll()
	assert.NotNil(t, all)
	assert.Empty(t, all.Of[0].GetUserRef().ResourceVersion)
	assert.Empty(t, all.Of[1].GetServiceRef().ResourceVersion)
}

func policyTriggerAllowPriority(t *testing.T, ctx context.Context,
	octeliumC octeliumc.ClientInterface, req *accessv1.Request) int32 {

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Name: fmt.Sprintf("access-request-%s", req.Metadata.Uid),
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, "access-request-default", pt.Status.InlinePolicies[0].Name)
	assert.Equal(t, 1, len(pt.Status.InlinePolicies[0].Spec.Rules))

	return pt.Status.InlinePolicies[0].Spec.Rules[0].Priority
}

func setAccessRequestAllowPriority(t *testing.T, ctx context.Context,
	octeliumC octeliumc.ClientInterface, priority *int32) (*accessv1.ClusterConfig, *accessv1.ClusterConfig) {

	old, err := octeliumC.AccessV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err, "%+v", err)

	next := pbutils.Clone(old).(*accessv1.ClusterConfig)
	next.Spec.Authorization = &accessv1.ClusterConfig_Spec_Authorization{
		Priority: priority,
	}

	next, err = octeliumC.AccessC().UpdateClusterConfig(ctx, next)
	assert.Nil(t, err, "%+v", err)

	return next, old
}

func TestRequestPolicyTriggerDefaultPriority(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	assert.Equal(t, int32(-1), policyTriggerAllowPriority(t, ctx, octeliumC, reqG))
}

func TestRequestPolicyTriggerClusterConfigPriority(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	priority := int32(-3)
	setAccessRequestAllowPriority(t, ctx, octeliumC, &priority)

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	assert.Equal(t, int32(-3), policyTriggerAllowPriority(t, ctx, octeliumC, reqG))

	zero := int32(0)
	next, old := setAccessRequestAllowPriority(t, ctx, octeliumC, &zero)
	assert.Nil(t, ctrl.OnClusterConfigUpdate(ctx, next, old))
	assert.Equal(t, int32(0), policyTriggerAllowPriority(t, ctx, octeliumC, reqG))

	next, old = setAccessRequestAllowPriority(t, ctx, octeliumC, nil)
	assert.Nil(t, ctrl.OnClusterConfigUpdate(ctx, next, old))
	assert.Equal(t, int32(-1), policyTriggerAllowPriority(t, ctx, octeliumC, reqG))

	reqG = getRequest(t, ctx, octeliumC, reqG.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
}

func TestRequestClusterConfigUpdateSkipsSamePriority(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: reqG.Status.PolicyTriggerRef.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	priority := int32(-1)
	next, old := setAccessRequestAllowPriority(t, ctx, octeliumC, &priority)
	assert.Nil(t, ctrl.OnClusterConfigUpdate(ctx, next, old))

	ptG, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Uid: reqG.Status.PolicyTriggerRef.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, pt.Metadata.ResourceVersion, ptG.Metadata.ResourceVersion)
}

func TestGetAccessRequestAllowPriority(t *testing.T) {
	priority := func(v int32) *accessv1.ClusterConfig {
		return &accessv1.ClusterConfig{
			Spec: &accessv1.ClusterConfig_Spec{
				Authorization: &accessv1.ClusterConfig_Spec_Authorization{
					Priority: &v,
				},
			},
		}
	}

	tests := []struct {
		name string
		cc   *accessv1.ClusterConfig
		want int32
	}{
		{"nil", nil, -1},
		{"no spec", &accessv1.ClusterConfig{}, -1},
		{"no authorization", &accessv1.ClusterConfig{Spec: &accessv1.ClusterConfig_Spec{}}, -1},
		{"no priority", &accessv1.ClusterConfig{
			Spec: &accessv1.ClusterConfig_Spec{
				Authorization: &accessv1.ClusterConfig_Spec_Authorization{},
			},
		}, -1},
		{"zero", priority(0), 0},
		{"highest", priority(-4), -4},
		{"middle", priority(-2), -2},
		{"below range", priority(-9), -4},
		{"above range", priority(3), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getAccessRequestAllowPriority(tt.cc))
		})
	}
}

func TestRequestGroupConditionsUseTheDelegatedSubject(t *testing.T) {
	ctx, ctl, client := newControllerTest(t)
	group, err := client.CoreC().CreateGroup(ctx, &corev1.Group{Metadata: &metav1.Metadata{Name: "request-group"}, Spec: &corev1.Group_Spec{}, Status: &corev1.Group_Status{}})
	require.NoError(t, err)
	member, err := client.CoreC().CreateUser(ctx, &corev1.User{Metadata: &metav1.Metadata{Name: "member"}, Spec: &corev1.User_Spec{Type: corev1.User_Spec_HUMAN, Groups: []string{group.Metadata.Name}}, Status: &corev1.User_Status{}})
	require.NoError(t, err)
	requester, err := client.CoreC().CreateUser(ctx, &corev1.User{Metadata: &metav1.Metadata{Name: "requester"}, Spec: &corev1.User_Spec{Type: corev1.User_Spec_HUMAN}, Status: &corev1.User_Status{}})
	require.NoError(t, err)
	req := baseRequest(umetav1.GetObjectReference(requester), objRef("Service"))
	req.Spec.Subject = &accessv1.Request_Spec_Subject{Type: &accessv1.Request_Spec_Subject_UserRef{UserRef: umetav1.GetObjectReference(member)}}
	condition := &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{
		Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef{GroupRef: umetav1.GetObjectReference(group)}},
	}}
	matched, err := ctl.matchCondition(ctx, req, condition)
	require.NoError(t, err)
	assert.True(t, matched)
	req.Spec.Subject = nil
	matched, err = ctl.matchCondition(ctx, req, condition)
	require.NoError(t, err)
	assert.False(t, matched)
	for _, userRef := range []*metav1.ObjectReference{nil, {}, {Uid: vutils.UUIDv4()}} {
		matched, err := ctl.matchSubjectGroup(ctx, userRef, umetav1.GetObjectReference(group))
		require.NoError(t, err)
		assert.False(t, matched)
	}
	matched, err = ctl.matchSubjectGroup(ctx, umetav1.GetObjectReference(member), &metav1.ObjectReference{Uid: vutils.UUIDv4()})
	require.NoError(t, err)
	assert.False(t, matched)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = ctl.matchSubjectGroup(canceled, umetav1.GetObjectReference(member), umetav1.GetObjectReference(group))
	assert.Error(t, err)
}

func TestRequestResourceAndLogicalConditions(t *testing.T) {
	ctl := &Controller{}
	anyMatch := func() *accessv1.Policy_Spec_Rule_Condition {
		return &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{MatchAny: true}}
	}
	service := objRef("Service")
	catalog := objRef("Catalog")
	req := baseRequest(objRef("User"), service)
	for _, tc := range []struct {
		condition *accessv1.Policy_Spec_Rule_Condition
		want      bool
	}{
		{nil, false},
		{&accessv1.Policy_Spec_Rule_Condition{}, false},
		{&accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef{ServiceRef: service}}}}, true},
		{&accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef{ServiceRef: objRef("Service")}}}}, false},
		{allOf(), false},
		{allOf(anyMatch(), &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{MatchAny: false}}), false},
		{&accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Any_{Any: &accessv1.Policy_Spec_Rule_Condition_Any{}}}, false},
		{&accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Any_{Any: &accessv1.Policy_Spec_Rule_Condition_Any{Of: []*accessv1.Policy_Spec_Rule_Condition{nil, anyMatch()}}}}, true},
	} {
		matched, err := ctl.matchCondition(t.Context(), req, tc.condition)
		require.NoError(t, err)
		assert.Equal(t, tc.want, matched)
	}
	req.Spec.Resource = &accessv1.Request_Spec_Resource{Type: &accessv1.Request_Spec_Resource_Catalog_{Catalog: &accessv1.Request_Spec_Resource_Catalog{CatalogRef: catalog}}}
	assert.True(t, ctl.matchResource(req, &accessv1.Policy_Spec_Rule_Condition_Resource{Type: &accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef{CatalogRef: catalog}}))
	assert.False(t, ctl.matchResource(req, nil))
	assert.False(t, ctl.matchResource(req, &accessv1.Policy_Spec_Rule_Condition_Resource{}))
}

func TestCatalogRequestTargetsServicesAndNamespaces(t *testing.T) {
	ctx, ctl, client := newControllerTest(t)
	svc, err := client.CoreC().CreateService(ctx, &corev1.Service{Metadata: &metav1.Metadata{Name: "catalog-service.default"}, Spec: &corev1.Service_Spec{}, Status: &corev1.Service_Status{}})
	require.NoError(t, err)
	ns, err := client.CoreC().CreateNamespace(ctx, &corev1.Namespace{Metadata: &metav1.Metadata{Name: "catalog-namespace"}, Spec: &corev1.Namespace_Spec{}, Status: &corev1.Namespace_Status{}})
	require.NoError(t, err)
	catalog, err := client.AccessC().CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{Name: "test-catalog"},
		Spec: &accessv1.Catalog_Spec{ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
			Services:   []string{"", svc.Metadata.Name, "missing.default"},
			Namespaces: []string{"", ns.Metadata.Name, "missing"},
		}}},
	})
	require.NoError(t, err)
	req := baseRequest(objRef("User"), nil)
	req.Spec.Resource = &accessv1.Request_Spec_Resource{Type: &accessv1.Request_Spec_Resource_Catalog_{Catalog: &accessv1.Request_Spec_Resource_Catalog{CatalogRef: umetav1.GetObjectReference(catalog)}}}
	target, err := ctl.buildTargetPreCondition(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, target.GetAny())
	require.Len(t, target.GetAny().Of, 2)
	assert.Equal(t, svc.Metadata.Uid, target.GetAny().Of[0].GetServiceRef().Uid)
	assert.Equal(t, ns.Metadata.Uid, target.GetAny().Of[1].GetNamespaceRef().Uid)
	assert.Empty(t, target.GetAny().Of[0].GetServiceRef().ResourceVersion)
	catalog.Spec.ResourceCollection.Service.Services = nil
	target, err = ctl.buildCatalogTargetPreCondition(ctx, req, catalog)
	require.NoError(t, err)
	assert.Equal(t, ns.Metadata.Uid, target.GetNamespaceRef().Uid)
	catalog.Spec.ResourceCollection.Service.Namespaces = []string{"missing"}
	_, err = ctl.buildCatalogTargetPreCondition(ctx, req, catalog)
	assert.ErrorContains(t, err, "does not resolve")
	catalog.Spec.ResourceCollection = nil
	_, err = ctl.buildCatalogTargetPreCondition(ctx, req, catalog)
	assert.ErrorContains(t, err, "no Service resource collection")
}

func TestInvalidRequestTargetReferencesAreRejected(t *testing.T) {
	ctx, ctl, _ := newControllerTest(t)
	for _, resource := range []*accessv1.Request_Spec_Resource{
		nil,
		{},
		{Type: &accessv1.Request_Spec_Resource_ServiceRef{}},
		{Type: &accessv1.Request_Spec_Resource_Catalog_{Catalog: &accessv1.Request_Spec_Resource_Catalog{}}},
		{Type: &accessv1.Request_Spec_Resource_Catalog_{Catalog: &accessv1.Request_Spec_Resource_Catalog{CatalogRef: &metav1.ObjectReference{Uid: vutils.UUIDv4()}}}},
	} {
		req := baseRequest(objRef("User"), nil)
		req.Spec.Resource = resource
		_, err := ctl.buildTargetPreCondition(ctx, req)
		assert.Error(t, err)
	}
	catalog, err := ctl.getCatalog(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, catalog)
	group, err := ctl.getGroup(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, group)
}

func TestReviewActionValidationBoundaries(t *testing.T) {
	base := func() *accessv1.Policy_Spec_Rule_Action_Review {
		return &accessv1.Policy_Spec_Rule_Action_Review{Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{anyStep(userReviewer(objRef("User")))}}
	}
	assert.Error(t, validateReviewAction(nil))
	assert.Error(t, validateReviewAction(&accessv1.Policy_Spec_Rule_Action_Review{}))
	assert.Error(t, validateReviewAction(&accessv1.Policy_Spec_Rule_Action_Review{Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{nil}}))
	for _, mutate := range []func(*accessv1.Policy_Spec_Rule_Action_Review_Step){
		func(step *accessv1.Policy_Spec_Rule_Action_Review_Step) { step.Reviewers = nil },
		func(step *accessv1.Policy_Spec_Rule_Action_Review_Step) { step.ApprovalCount = 1 },
		func(step *accessv1.Policy_Spec_Rule_Action_Review_Step) {
			step.ApprovalRequirement = accessv1.Policy_Spec_Rule_Action_Review_Step_ALL
			step.ApprovalCount = 1
		},
		func(step *accessv1.Policy_Spec_Rule_Action_Review_Step) {
			step.ApprovalRequirement = accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT
		},
		func(step *accessv1.Policy_Spec_Rule_Action_Review_Step) {
			step.ApprovalRequirement = accessv1.Policy_Spec_Rule_Action_Review_Step_APPROVAL_REQUIREMENT_UNSET
		},
		func(step *accessv1.Policy_Spec_Rule_Action_Review_Step) { step.ApprovalRequirement = 999 },
		func(step *accessv1.Policy_Spec_Rule_Action_Review_Step) { step.OnTimeout = 999 },
	} {
		review := base()
		mutate(review.Steps[0])
		assert.Error(t, validateReviewAction(review))
	}
	for _, requirement := range []accessv1.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement{
		accessv1.Policy_Spec_Rule_Action_Review_Step_ANY, accessv1.Policy_Spec_Rule_Action_Review_Step_ALL, accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT,
	} {
		review := base()
		review.Steps[0].ApprovalRequirement = requirement
		if requirement == accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT {
			review.Steps[0].ApprovalCount = 1
		}
		require.NoError(t, validateReviewAction(review))
	}
}

func TestAccessDurationLimitsAndPolicyCopyIsolation(t *testing.T) {
	ctl := &Controller{}
	for _, tc := range []struct{ requested, maximum, want uint32 }{
		{0, 0, 86400}, {3600, 7200, 3600}, {10800, 7200, 7200}, {0, 7200, 7200},
	} {
		req := baseRequest(objRef("User"), objRef("Service"))
		req.Spec.Duration = &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: tc.requested}}
		authz := &accessv1.Policy_Spec_Rule_Authorization{MaxAccessDuration: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: tc.maximum}}}
		ctl.setEffectiveDuration(req, authz)
		assert.EqualValues(t, tc.want, req.Status.EffectiveDuration.GetSeconds())
	}
	original := buildDefaultAccessRequestInlinePolicy(10)
	saved := pbutils.Clone(original).(*corev1.InlinePolicy)
	clones := cloneInlinePolicies([]*corev1.InlinePolicy{nil, original})
	require.Len(t, clones, 1)
	proto.Reset(clones[0].Spec)
	assert.True(t, proto.Equal(saved, original))
	ref := &metav1.ObjectReference{Uid: "uid", Name: "original", ResourceVersion: "version"}
	copy := cloneRef(ref)
	assert.Empty(t, copy.ResourceVersion)
	copy.Name = "changed"
	assert.Equal(t, "original", ref.Name)
	assert.Equal(t, "version", ref.ResourceVersion)
	assert.Nil(t, cloneRef(nil))
	req := baseRequest(nil, nil)
	assert.Equal(t, PolicyTriggerNamePrefix+req.Metadata.Name, policyTriggerName(req))
}
