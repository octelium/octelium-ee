// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package watcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/requests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func newWatcherTest(t *testing.T) (context.Context, *Watcher, octeliumc.ClientInterface) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	return ctx, InitWatcher(tst.C.OcteliumC), tst.C.OcteliumC
}

func objRef(kind string) *metav1.ObjectReference {
	return &metav1.ObjectReference{
		ApiVersion: "core/v1",
		Kind:       kind,
		Name:       utilrand.GetRandomStringCanonical(8),
		Uid:        utilrand.GetRandomStringCanonical(16),
	}
}

func seconds(arg uint32) *metav1.Duration {
	return &metav1.Duration{
		Type: &metav1.Duration_Seconds{
			Seconds: arg,
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

func reviewRule(steps ...*accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Policy_Spec_Rule {
	return &accessv1.Policy_Spec_Rule{
		Name:   utilrand.GetRandomStringCanonical(6),
		Effect: accessv1.Policy_Spec_Rule_REVIEW,
		Condition: &accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_MatchAny{
				MatchAny: true,
			},
		},
		Action: &accessv1.Policy_Spec_Rule_Action{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_{
				Review: &accessv1.Policy_Spec_Rule_Action_Review{
					Steps: steps,
				},
			},
		},
	}
}

func timeoutStep(timeout *metav1.Duration,
	onTimeout accessv1.Policy_Spec_Rule_Action_Review_Step_OnTimeout) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Reviewers:           []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{userReviewer(objRef("User"))},
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
		Timeout:             timeout,
		OnTimeout:           onTimeout,
	}
}

func createRequest(t *testing.T, ctx context.Context,
	octeliumC octeliumc.ClientInterface, status *accessv1.Request_Status) *accessv1.Request {
	req, err := octeliumC.AccessC().CreateRequest(ctx, &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_NORMAL,
		},
		Status: status,
	})
	assert.Nil(t, err, "%+v", err)
	return req
}

func createPendingRequest(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface,
	rule *accessv1.Policy_Spec_Rule, startedAt time.Time) *accessv1.Request {
	return createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
		UserRef: objRef("User"),
		State: &accessv1.Request_Status_State{
			CreatedAt: pbutils.Now(),
			Status:    accessv1.Request_Status_State_PENDING,
		},
		Rule:            rule,
		ApprovalStartAt: pbutils.Timestamp(startedAt),
		Review: &accessv1.Request_Status_Review{
			CurrentStep:          0,
			CurrentStepStartedAt: pbutils.Timestamp(startedAt),
		},
	})
}

func getRequest(t *testing.T, ctx context.Context,
	octeliumC octeliumc.ClientInterface, uid string) *accessv1.Request {
	out, err := octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{Uid: uid})
	assert.Nil(t, err, "%+v", err)
	return out
}

func createAccessPolicyTrigger(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface,
	name string, ownerRef *metav1.ObjectReference, notAfter *time.Time) *corev1.PolicyTrigger {
	preConditions := []*corev1.PolicyTrigger_Status_PreCondition{
		{
			Type: &corev1.PolicyTrigger_Status_PreCondition_UserRef{
				UserRef: objRef("User"),
			},
		},
	}

	if notAfter != nil {
		preConditions = append(preConditions, &corev1.PolicyTrigger_Status_PreCondition{
			Type: &corev1.PolicyTrigger_Status_PreCondition_NotAfter{
				NotAfter: pbutils.Timestamp(*notAfter),
			},
		})
	}

	pt, err := octeliumC.CoreC().CreatePolicyTrigger(ctx, &corev1.PolicyTrigger{
		Metadata: &metav1.Metadata{
			Name:           name,
			IsSystem:       true,
			IsSystemHidden: true,
		},
		Spec: &corev1.PolicyTrigger_Spec{},
		Status: &corev1.PolicyTrigger_Status{
			OwnerRef: ownerRef,
			PreCondition: &corev1.PolicyTrigger_Status_PreCondition{
				Type: &corev1.PolicyTrigger_Status_PreCondition_All_{
					All: &corev1.PolicyTrigger_Status_PreCondition_All{
						Of: preConditions,
					},
				},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	return pt
}

func policyTriggerExists(t *testing.T, ctx context.Context,
	octeliumC octeliumc.ClientInterface, name string) bool {
	_, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Name: name,
	})
	if err != nil {
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
		return false
	}

	return true
}

func accessRequestPolicyTriggerName(req *accessv1.Request) string {
	return fmt.Sprintf("%s%s", requests.PolicyTriggerNamePrefix, req.Metadata.Uid)
}

func TestPolicyTriggerOwnerNotFound(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	req := createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
		UserRef: objRef("User"),
		State: &accessv1.Request_Status_State{
			CreatedAt: pbutils.Now(),
			Status:    accessv1.Request_Status_State_APPROVED,
		},
	})

	name := accessRequestPolicyTriggerName(req)
	pt := createAccessPolicyTrigger(t, ctx, octeliumC, name, umetav1.GetObjectReference(req), nil)

	_, err := octeliumC.AccessC().DeleteRequest(ctx, &rmetav1.DeleteOptions{
		Uid: req.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, w.doCheckPolicyTrigger(ctx, pt))
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, name))
}

func TestPolicyTriggerNoOwnerRef(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	name := fmt.Sprintf("%s%s", requests.PolicyTriggerNamePrefix, utilrand.GetRandomStringCanonical(16))
	pt := createAccessPolicyTrigger(t, ctx, octeliumC, name, nil, nil)

	assert.Nil(t, w.doCheckPolicyTrigger(ctx, pt))
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, name))
}

func TestPolicyTriggerApprovedOwnerKept(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	req := createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
		UserRef: objRef("User"),
		State: &accessv1.Request_Status_State{
			CreatedAt: pbutils.Now(),
			Status:    accessv1.Request_Status_State_APPROVED,
		},
	})

	name := accessRequestPolicyTriggerName(req)
	notAfter := time.Now().Add(time.Hour)
	pt := createAccessPolicyTrigger(t, ctx, octeliumC, name, umetav1.GetObjectReference(req), &notAfter)

	assert.Nil(t, w.doCheckPolicyTrigger(ctx, pt))
	assert.True(t, policyTriggerExists(t, ctx, octeliumC, name))
}

func TestPolicyTriggerTerminalOwnerDeleted(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	for _, status := range []accessv1.Request_Status_State_Status{
		accessv1.Request_Status_State_REJECTED,
		accessv1.Request_Status_State_REVOKED,
		accessv1.Request_Status_State_EXPIRED,
		accessv1.Request_Status_State_CANCELLED,
	} {
		req := createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
			UserRef: objRef("User"),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    status,
			},
		})

		name := accessRequestPolicyTriggerName(req)
		pt := createAccessPolicyTrigger(t, ctx, octeliumC, name, umetav1.GetObjectReference(req), nil)

		assert.Nil(t, w.doCheckPolicyTrigger(ctx, pt), "%s", status)
		assert.False(t, policyTriggerExists(t, ctx, octeliumC, name), "%s", status)
	}
}

func TestPolicyTriggerNotAfterPassed(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	req := createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
		UserRef: objRef("User"),
		State: &accessv1.Request_Status_State{
			CreatedAt: pbutils.Now(),
			Status:    accessv1.Request_Status_State_APPROVED,
		},
	})

	name := accessRequestPolicyTriggerName(req)
	notAfter := time.Now().Add(-time.Minute)
	pt := createAccessPolicyTrigger(t, ctx, octeliumC, name, umetav1.GetObjectReference(req), &notAfter)

	assert.Nil(t, w.doCheckPolicyTrigger(ctx, pt))
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, name))
}

func TestPolicyTriggerUnrelatedIgnored(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	name := utilrand.GetRandomStringCanonical(8)
	pt := createAccessPolicyTrigger(t, ctx, octeliumC, name, objRef("Request"), nil)

	assert.Nil(t, w.doCheckPolicyTrigger(ctx, pt))
	assert.True(t, policyTriggerExists(t, ctx, octeliumC, name))
}

func TestWatcherPendingDeadlinePassed(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	req := createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
		UserRef: objRef("User"),
		State: &accessv1.Request_Status_State{
			CreatedAt: pbutils.Now(),
			Status:    accessv1.Request_Status_State_PENDING,
		},
	})

	req.Spec.Deadline = pbutils.Timestamp(time.Now().Add(-time.Minute))
	req, err := octeliumC.AccessC().UpdateRequest(ctx, req)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, w.doCheckRequest(ctx, req))
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestWatcherPendingDeadlineNotPassed(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	req := createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
		UserRef: objRef("User"),
		State: &accessv1.Request_Status_State{
			CreatedAt: pbutils.Now(),
			Status:    accessv1.Request_Status_State_PENDING,
		},
	})

	req.Spec.Deadline = pbutils.Timestamp(time.Now().Add(time.Hour))
	req, err := octeliumC.AccessC().UpdateRequest(ctx, req)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, w.doCheckRequest(ctx, req))
	assert.Equal(t, accessv1.Request_Status_State_PENDING,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestWatcherStepTimeoutGotoNextStep(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	rule := reviewRule(
		timeoutStep(seconds(30), accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_GOTO_NEXT_STEP),
		timeoutStep(nil, accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_UNSET))

	req := createPendingRequest(t, ctx, octeliumC, rule, time.Now().Add(-time.Hour))

	assert.Nil(t, w.doCheckRequest(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.Equal(t, int32(1), reqG.Status.Review.CurrentStep)
	assert.NotNil(t, reqG.Status.Review.CurrentStepStartedAt)
}

func TestWatcherStepTimeoutGotoNextStepAtLastStep(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	rule := reviewRule(
		timeoutStep(seconds(30), accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_GOTO_NEXT_STEP))

	req := createPendingRequest(t, ctx, octeliumC, rule, time.Now().Add(-time.Hour))

	assert.Nil(t, w.doCheckRequest(ctx, req))
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestWatcherStepTimeoutReject(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	rule := reviewRule(
		timeoutStep(seconds(30), accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_REJECT))

	req := createPendingRequest(t, ctx, octeliumC, rule, time.Now().Add(-time.Hour))

	assert.Nil(t, w.doCheckRequest(ctx, req))
	assert.Equal(t, accessv1.Request_Status_State_REJECTED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestWatcherStepTimeoutUnsetExpires(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	rule := reviewRule(
		timeoutStep(seconds(30), accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_UNSET))

	req := createPendingRequest(t, ctx, octeliumC, rule, time.Now().Add(-time.Hour))

	assert.Nil(t, w.doCheckRequest(ctx, req))
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestWatcherStepTimeoutNotPassed(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	rule := reviewRule(
		timeoutStep(seconds(3600), accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_REJECT))

	req := createPendingRequest(t, ctx, octeliumC, rule, time.Now().Add(-time.Minute))

	assert.Nil(t, w.doCheckRequest(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.Equal(t, int32(0), reqG.Status.Review.CurrentStep)
}

func TestWatcherStepWithoutTimeoutNeverExpires(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	rule := reviewRule(
		timeoutStep(nil, accessv1.Policy_Spec_Rule_Action_Review_Step_ON_TIMEOUT_REJECT))

	req := createPendingRequest(t, ctx, octeliumC, rule, time.Now().Add(-8760*time.Hour))

	assert.Nil(t, w.doCheckRequest(ctx, req))
	assert.Equal(t, accessv1.Request_Status_State_PENDING,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestWatcherApprovedAccessEndsAtPassed(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	req := createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
		UserRef: objRef("User"),
		State: &accessv1.Request_Status_State{
			CreatedAt: pbutils.Now(),
			Status:    accessv1.Request_Status_State_APPROVED,
		},
		AccessEndsAt: pbutils.Timestamp(time.Now().Add(-time.Minute)),
	})

	assert.Nil(t, w.doCheckRequest(ctx, req))
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestWatcherApprovedAccessEndsAtNotPassed(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	req := createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
		UserRef: objRef("User"),
		State: &accessv1.Request_Status_State{
			CreatedAt: pbutils.Now(),
			Status:    accessv1.Request_Status_State_APPROVED,
		},
		AccessEndsAt: pbutils.Timestamp(time.Now().Add(time.Hour)),
	})

	assert.Nil(t, w.doCheckRequest(ctx, req))
	assert.Equal(t, accessv1.Request_Status_State_APPROVED,
		getRequest(t, ctx, octeliumC, req.Metadata.Uid).Status.State.Status)
}

func TestWatcherTerminalRequestsUntouched(t *testing.T) {
	ctx, w, octeliumC := newWatcherTest(t)

	for _, status := range []accessv1.Request_Status_State_Status{
		accessv1.Request_Status_State_REJECTED,
		accessv1.Request_Status_State_REVOKED,
		accessv1.Request_Status_State_EXPIRED,
		accessv1.Request_Status_State_CANCELLED,
	} {
		req := createRequest(t, ctx, octeliumC, &accessv1.Request_Status{
			UserRef: objRef("User"),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    status,
			},
			AccessEndsAt: pbutils.Timestamp(time.Now().Add(-time.Hour)),
		})

		assert.Nil(t, w.doCheckRequest(ctx, req), "%s", status)

		reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
		assert.Equal(t, status, reqG.Status.State.Status)
		assert.Equal(t, req.Metadata.ResourceVersion, reqG.Metadata.ResourceVersion, "%s", status)
	}
}
