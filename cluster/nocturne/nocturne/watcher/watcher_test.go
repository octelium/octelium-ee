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
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/integrationbindings"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/controllers/requests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/raccessv1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func newWatcherTest(t *testing.T) (context.Context, *Watcher, octeliumc.ClientInterface) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	bindingCtl, err := integrationbindings.NewController(ctx, tst.C.OcteliumC)
	assert.Nil(t, err)

	return ctx, InitWatcher(tst.C.OcteliumC, bindingCtl), tst.C.OcteliumC
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

type watcherPagingConn struct {
	invoke func(context.Context, string, any, any) error
}

func (c *watcherPagingConn) Invoke(ctx context.Context, method string, req, reply any, opts ...grpc.CallOption) error {
	return c.invoke(ctx, method, req, reply)
}
func (c *watcherPagingConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("unexpected stream")
}

type watcherPagingClient struct {
	octeliumc.ClientInterface
	conn *watcherPagingConn
}

func (c *watcherPagingClient) CoreC() rcorev1.ResourceServiceClient {
	return rcorev1.NewResourceServiceClient(c.conn)
}
func (c *watcherPagingClient) AccessC() raccessv1.ResourceServiceClient {
	return raccessv1.NewResourceServiceClient(c.conn)
}

func watcherListNames(ctx context.Context, w *Watcher, kind string) ([]string, error) {
	var names []string
	switch kind {
	case "Request":
		items, err := w.listRequests(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			names = append(names, item.Metadata.Name)
		}
	case "PolicyTrigger":
		items, err := w.listPolicyTriggers(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			names = append(names, item.Metadata.Name)
		}
	case "IntegrationBinding":
		items, err := w.listIntegrationBindings(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			names = append(names, item.Metadata.Name)
		}
	}
	return names, nil
}

func TestWatcherListsEveryPage(t *testing.T) {
	for _, kind := range []string{"Request", "PolicyTrigger", "IntegrationBinding"} {
		for _, failAt := range []int{-1, 0, 1} {
			t.Run(fmt.Sprintf("%s/%d", kind, failAt), func(t *testing.T) {
				failure := errors.New("page unavailable")
				calls := 0
				conn := &watcherPagingConn{invoke: func(ctx context.Context, method string, req, reply any) error {
					opts := req.(*rmetav1.ListOptions)
					assert.True(t, opts.Paginate)
					assert.EqualValues(t, 500, opts.ItemsPerPage)
					assert.EqualValues(t, calls, opts.Page)
					page := calls
					calls++
					if page == failAt {
						return failure
					}
					md := &metav1.Metadata{Name: fmt.Sprintf("page-%d", page)}
					meta := &metav1.ListResponseMeta{HasMore: page == 0}
					switch list := reply.(type) {
					case *accessv1.RequestList:
						list.Items = []*accessv1.Request{{Metadata: md}}
						list.ListResponseMeta = meta
					case *corev1.PolicyTriggerList:
						list.Items = []*corev1.PolicyTrigger{{Metadata: md}}
						list.ListResponseMeta = meta
					case *accessv1.IntegrationBindingList:
						list.Items = []*accessv1.IntegrationBinding{{Metadata: md}}
						list.ListResponseMeta = meta
					}
					return nil
				}}
				w := InitWatcher(&watcherPagingClient{conn: conn}, nil)
				names, err := watcherListNames(t.Context(), w, kind)
				if failAt >= 0 {
					assert.ErrorIs(t, err, failure)
					assert.Nil(t, names)
					assert.Equal(t, failAt+1, calls)
				} else {
					require.NoError(t, err)
					assert.Equal(t, []string{"page-0", "page-1"}, names)
					assert.Equal(t, 2, calls)
				}
			})
		}
	}
}

func TestIntegrationBindingDueScheduling(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name    string
		binding *accessv1.IntegrationBinding
		due     bool
	}{
		{"Nil", nil, false},
		{"NoStatus", &accessv1.IntegrationBinding{}, false},
		{"Ready", &accessv1.IntegrationBinding{Status: &accessv1.IntegrationBinding_Status{State: accessv1.IntegrationBinding_Status_READY, AppliedRevision: "same", DesiredRevision: "same"}}, false},
		{"Closed", &accessv1.IntegrationBinding{Status: &accessv1.IntegrationBinding_Status{State: accessv1.IntegrationBinding_Status_CLOSED, AppliedRevision: "same", DesiredRevision: "same"}}, false},
		{"Changed", &accessv1.IntegrationBinding{Status: &accessv1.IntegrationBinding_Status{State: accessv1.IntegrationBinding_Status_READY, AppliedRevision: "old", DesiredRevision: "new"}}, true},
		{"Deferred", &accessv1.IntegrationBinding{Status: &accessv1.IntegrationBinding_Status{NextAttemptAt: pbutils.Timestamp(now.Add(time.Hour))}}, false},
		{"Retry", &accessv1.IntegrationBinding{Status: &accessv1.IntegrationBinding_Status{NextAttemptAt: pbutils.Timestamp(now.Add(-time.Hour))}}, true},
		{"New", &accessv1.IntegrationBinding{Status: &accessv1.IntegrationBinding_Status{}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) { assert.Equal(t, tc.due, isIntegrationBindingDue(tc.binding)) })
	}
}

func TestWatchIntervalEnvironmentValidation(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  time.Duration
	}{
		{"", time.Minute}, {"invalid", time.Minute}, {"0s", time.Minute}, {"-1s", time.Minute}, {"250ms", 250 * time.Millisecond}, {"24h", 24 * time.Hour},
	} {
		t.Setenv("OCTELIUM_TEST_WATCH_INTERVAL", tc.value)
		assert.Equal(t, tc.want, durationFromEnv("OCTELIUM_TEST_WATCH_INTERVAL", time.Minute))
	}
}

type watcherRetryReconciler struct{ calls atomic.Int32 }

func (r *watcherRetryReconciler) Reconcile(context.Context, *accessv1.IntegrationBinding) error {
	r.calls.Add(1)
	return errors.New("presentation temporarily unavailable")
}

func TestWatchersPollAndStopOnCancellation(t *testing.T) {
	originalAccess, originalBinding := accessWatchInterval, integrationBindingWatchInterval
	accessWatchInterval, integrationBindingWatchInterval = 10*time.Millisecond, 10*time.Millisecond
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{}, 3)
	t.Cleanup(func() {
		cancel()
		for range 3 {
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Error("watcher did not stop")
			}
		}
		accessWatchInterval, integrationBindingWatchInterval = originalAccess, originalBinding
	})
	var requestCalls, policyCalls atomic.Int32
	reconciler := &watcherRetryReconciler{}
	conn := &watcherPagingConn{invoke: func(ctx context.Context, method string, req, reply any) error {
		switch list := reply.(type) {
		case *accessv1.RequestList:
			requestCalls.Add(1)
			return errors.New("request page temporarily unavailable")
		case *corev1.PolicyTriggerList:
			policyCalls.Add(1)
		case *accessv1.IntegrationBindingList:
			list.Items = []*accessv1.IntegrationBinding{
				{Metadata: &metav1.Metadata{Name: "due"}, Status: &accessv1.IntegrationBinding_Status{}},
				{Metadata: &metav1.Metadata{Name: "ready"}, Status: &accessv1.IntegrationBinding_Status{State: accessv1.IntegrationBinding_Status_READY}},
			}
		}
		return nil
	}}
	w := InitWatcher(&watcherPagingClient{conn: conn}, reconciler)
	for _, run := range []func(context.Context){w.runRequests, w.runPolicyTriggers, w.runIntegrationBindings} {
		go func() { defer func() { done <- struct{}{} }(); run(ctx) }()
	}
	require.Eventually(t, func() bool { return requestCalls.Load() > 0 && policyCalls.Load() > 0 && reconciler.calls.Load() > 0 }, 2*time.Second, 5*time.Millisecond)
}
