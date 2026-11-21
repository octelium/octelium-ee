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
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func createUser(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface, groups ...string) *corev1.User {
	usr, err := octeliumC.CoreC().CreateUser(ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type:   corev1.User_Spec_HUMAN,
			Groups: groups,
		},
		Status: &corev1.User_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	return usr
}

func createUserRef(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface) *metav1.ObjectReference {
	return umetav1.GetObjectReference(createUser(t, ctx, octeliumC))
}

func createService(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface) *corev1.Service {
	svc, err := octeliumC.CoreC().CreateService(ctx, &corev1.Service{
		Metadata: &metav1.Metadata{
			Name: fmt.Sprintf("%s.default", utilrand.GetRandomStringCanonical(8)),
		},
		Spec:   &corev1.Service_Spec{},
		Status: &corev1.Service_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	return svc
}

func fullLevel(name string, maxDuration *metav1.Duration) *accessv1.Catalog_Spec_AccessLevel {
	return &accessv1.Catalog_Spec_AccessLevel{
		Name: name,
		Type: &accessv1.Catalog_Spec_AccessLevel_Full_{
			Full: &accessv1.Catalog_Spec_AccessLevel_Full{},
		},
		MaxDuration: maxDuration,
	}
}

func catalogSpec(levels []*accessv1.Catalog_Spec_AccessLevel, services ...*corev1.Service) *accessv1.Catalog_Spec {
	if len(levels) == 0 {
		levels = []*accessv1.Catalog_Spec_AccessLevel{fullLevel("full", nil)}
	}

	var names []string
	for _, svc := range services {
		names = append(names, svc.Metadata.Name)
	}

	return &accessv1.Catalog_Spec{
		ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
			Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
				Services: names,
			},
		},
		AccessLevels: levels,
	}
}

func createCatalog(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface,
	levels []*accessv1.Catalog_Spec_AccessLevel, services ...*corev1.Service) *accessv1.Catalog {
	if len(services) == 0 {
		services = []*corev1.Service{createService(t, ctx, octeliumC)}
	}

	catalog, err := octeliumC.AccessC().CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   catalogSpec(levels, services...),
		Status: &accessv1.Catalog_Status{},
	})
	assert.Nil(t, err, "%+v", err)
	return catalog
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

func resourceCondition(resource *accessv1.Policy_Spec_Rule_Condition_Resource) *accessv1.Policy_Spec_Rule_Condition {
	return &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{
			Resource: resource,
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

func baseRequest(uref *metav1.ObjectReference, catalog *accessv1.Catalog) *accessv1.Request {
	ret := &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_NORMAL,
		},
		Status: &accessv1.Request_Status{
			UserRef:      uref,
			SpecRevision: 1,
		},
	}

	if catalog != nil {
		ret.Spec.Resource = &accessv1.Request_Spec_Resource{
			Type: &accessv1.Request_Spec_Resource_Catalog_{
				Catalog: &accessv1.Request_Spec_Resource_Catalog{
					CatalogRef: umetav1.GetObjectReference(catalog),
				},
			},
		}
	}

	return ret
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

func getPolicyTrigger(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface,
	req *accessv1.Request) *corev1.PolicyTrigger {
	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{
		Name: policyTriggerName(req),
	})
	require.NoError(t, err)
	return pt
}

func setClusterConfig(t *testing.T, ctx context.Context, octeliumC octeliumc.ClientInterface,
	fn func(spec *accessv1.ClusterConfig_Spec)) {
	cc, err := octeliumC.AccessV1Utils().GetClusterConfig(ctx)
	require.NoError(t, err)

	if cc.Spec == nil {
		cc.Spec = &accessv1.ClusterConfig_Spec{}
	}
	fn(cc.Spec)

	_, err = octeliumC.AccessC().UpdateClusterConfig(ctx, cc)
	require.NoError(t, err)
}

func TestRequestAutoApprove(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	uref := createUserRef(t, ctx, octeliumC)
	svc := createService(t, ctx, octeliumC)
	catalog := createCatalog(t, ctx, octeliumC,
		[]*accessv1.Catalog_Spec_AccessLevel{fullLevel("full", durationHours(1))}, svc)

	pol := createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
		Message:   "approved for everyone",
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(uref, catalog))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_AUTO_APPROVED, reqG.Status.State.Reason)
	assert.Equal(t, "approved for everyone", reqG.Status.State.Message)
	assert.NotNil(t, reqG.Status.Rule)
	assert.Equal(t, accessv1.Policy_Spec_Rule_AUTO_APPROVE, reqG.Status.Rule.Effect)
	assert.Equal(t, pol.Metadata.Uid, reqG.Status.PolicyRef.Uid)
	assert.Equal(t, accessv1.Policy_Spec_Rule_AUTO_APPROVE, reqG.Status.Workflow.Effect)
	assert.Equal(t, accessv1.Request_Status_Evaluation_EVALUATED, reqG.Status.Evaluation.State)
	assert.NotNil(t, reqG.Status.ApprovalStartAt)
	assert.NotNil(t, reqG.Status.ApprovalEndAt)
	assert.NotNil(t, reqG.Status.PolicyTriggerRef)
	assert.Equal(t, uref.Uid, reqG.Status.SubjectInfo.UserRef.Uid)
	assert.Equal(t, uref.Uid, reqG.Status.RequesterInfo.UserRef.Uid)

	scope := reqG.Status.ResolvedScope
	require.NotNil(t, scope)
	assert.Equal(t, catalog.Metadata.Uid, scope.CatalogRef.Uid)
	assert.Equal(t, "full", scope.AccessLevel.Name)
	require.Len(t, scope.ServiceRefs, 1)
	assert.Equal(t, svc.Metadata.Uid, scope.ServiceRefs[0].Uid)
	assert.Equal(t, int32(0), scope.Priority)
	assert.False(t, scope.OverridesGuardrails)

	assert.Equal(t, accessv1.Request_Status_Grant_ACTIVE, reqG.Status.Grant.State)
	assert.NotNil(t, reqG.Status.Grant.ActivatedAt)
	assert.Equal(t, reqG.Status.ApprovalEndAt.AsTime(), reqG.Status.AccessStartsAt.AsTime())
	assert.Equal(t, reqG.Status.AccessStartsAt.AsTime().Add(time.Hour), reqG.Status.AccessEndsAt.AsTime())

	pt := getPolicyTrigger(t, ctx, octeliumC, reqG)
	assert.Equal(t, fmt.Sprintf("access-request-%s", req.Metadata.Uid), pt.Metadata.Name)
	assert.True(t, pt.Metadata.IsSystem)
	assert.Equal(t, req.Metadata.Uid, pt.Status.OwnerRef.Uid)
	assert.Empty(t, pt.Status.Policies)
	require.Len(t, pt.Status.InlinePolicies, 1)
	assert.Equal(t, "access-request-default", pt.Status.InlinePolicies[0].Name)
	assert.Equal(t, int32(0), pt.Status.InlinePolicies[0].Spec.Rules[0].Priority)

	all := pt.Status.PreCondition.GetAll()
	require.NotNil(t, all)
	require.Len(t, all.Of, 4)
	assert.Equal(t, uref.Uid, all.Of[0].GetUserRef().Uid)
	assert.Equal(t, svc.Metadata.Uid, all.Of[1].GetServiceRef().Uid)
	assert.Equal(t, reqG.Status.AccessStartsAt.AsTime(), all.Of[2].GetNotBefore().AsTime())
	assert.Equal(t, reqG.Status.AccessEndsAt.AsTime(), all.Of[3].GetNotAfter().AsTime())
}

func TestRequestRestrictedLevelHasNoImplicitAllow(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	corePolicy := utilrand.GetRandomStringCanonical(6)
	catalog := createCatalog(t, ctx, octeliumC, []*accessv1.Catalog_Spec_AccessLevel{
		{
			Name: "read",
			Type: &accessv1.Catalog_Spec_AccessLevel_Restricted_{
				Restricted: &accessv1.Catalog_Spec_AccessLevel_Restricted{
					Policies: []string{corePolicy},
					InlinePolicies: []*corev1.InlinePolicy{
						{
							Name: "read-only",
							Spec: &corev1.Policy_Spec{
								Rules: []*corev1.Policy_Spec_Rule{
									{
										Effect: corev1.Policy_Spec_Rule_ALLOW,
										Condition: &corev1.Condition{
											Type: &corev1.Condition_Match{
												Match: `ctx.request.http.method == "GET"`,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), catalog))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)

	pt := getPolicyTrigger(t, ctx, octeliumC, reqG)
	assert.Equal(t, []string{corePolicy}, pt.Status.Policies)
	require.Len(t, pt.Status.InlinePolicies, 1)
	assert.Equal(t, "read-only", pt.Status.InlinePolicies[0].Name)
	for _, pol := range pt.Status.InlinePolicies {
		assert.NotEqual(t, "access-request-default", pol.Name)
	}
}

func TestRequestAutoApproveNoDuration(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, reqG.Status.AccessStartsAt.AsTime().Add(accesscmd.DefaultMaxAccessDuration),
		reqG.Status.AccessEndsAt.AsTime())

	all := getPolicyTrigger(t, ctx, octeliumC, reqG).Status.PreCondition.GetAll()
	assert.Equal(t, reqG.Status.AccessEndsAt.AsTime(), all.Of[3].GetNotAfter().AsTime())
}

func TestRequestEffectiveDuration(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	level := fullLevel("full", durationHours(8))
	level.DefaultDuration = durationHours(2)
	catalog := createCatalog(t, ctx, octeliumC, []*accessv1.Catalog_Spec_AccessLevel{level})

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:        utilrand.GetRandomStringCanonical(6),
		Effect:      accessv1.Policy_Spec_Rule_REVIEW,
		Condition:   matchAny(),
		Action:      reviewAction(anyStep(userReviewer(createUserRef(t, ctx, octeliumC)))),
		MaxDuration: durationHours(4),
	})

	for _, tc := range []struct {
		requested *metav1.Duration
		want      uint32
	}{
		{nil, 2 * 3600},
		{durationHours(3), 3 * 3600},
		{durationHours(6), 4 * 3600},
		{durationHours(20), 4 * 3600},
	} {
		in := baseRequest(createUserRef(t, ctx, octeliumC), catalog)
		in.Spec.Duration = tc.requested
		req := createRequest(t, ctx, octeliumC, in)
		reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

		assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
		assert.Equal(t, tc.want, reqG.Status.EffectiveDuration.GetSeconds())
	}
}

func TestRequestPendingDurationEditIsApplied(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	catalog := createCatalog(t, ctx, octeliumC,
		[]*accessv1.Catalog_Spec_AccessLevel{fullLevel("full", durationHours(8))})

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(anyStep(userReviewer(createUserRef(t, ctx, octeliumC)))),
	})

	in := baseRequest(createUserRef(t, ctx, octeliumC), catalog)
	in.Spec.Duration = durationHours(1)
	req := createRequest(t, ctx, octeliumC, in)
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	assert.Equal(t, uint32(3600), reqG.Status.EffectiveDuration.GetSeconds())

	reqG.Spec.Duration = durationHours(5)
	_, err := octeliumC.AccessC().UpdateRequest(ctx, reqG)
	require.NoError(t, err)

	reqG = converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	assert.Equal(t, uint32(5*3600), reqG.Status.EffectiveDuration.GetSeconds())
}

func TestRequestReview(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reviewer := createUser(t, ctx, octeliumC)
	step := anyStep(userReviewer(umetav1.GetObjectReference(reviewer)))
	step.Name = "lead"

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(step),
	})

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_AWAITING_REVIEW, reqG.Status.State.Reason)
	assert.Equal(t, accessv1.Policy_Spec_Rule_REVIEW, reqG.Status.Rule.Effect)
	assert.Equal(t, int32(0), reqG.Status.Review.CurrentStep)
	assert.NotNil(t, reqG.Status.ApprovalStartAt)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
	assert.Nil(t, reqG.Status.AccessEndsAt)
	assert.Nil(t, reqG.Status.Grant)

	require.Len(t, reqG.Status.Workflow.Steps, 1)
	assert.Equal(t, "lead", reqG.Status.Workflow.Steps[0].Name)
	require.Len(t, reqG.Status.Workflow.Steps[0].Reviewers, 1)
	assert.Equal(t, reviewer.Metadata.Name, reqG.Status.Workflow.Steps[0].Reviewers[0].DisplayName)

	require.Len(t, reqG.Status.Review.Steps, 1)
	stepState := reqG.Status.Review.Steps[0]
	assert.Equal(t, "lead", stepState.Name)
	assert.Equal(t, uint32(1), stepState.TotalEligibleReviewer)
	assert.Equal(t, uint32(1), stepState.RequiredApprovals)
	assert.False(t, stepState.IsUnroutable)
	require.Len(t, stepState.EligibleReviewers, 1)
	assert.Equal(t, reviewer.Metadata.Uid, stepState.EligibleReviewers[0].Uid)
}

func TestRequestReviewWithoutEligibleReviewers(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	requester := createUserRef(t, ctx, octeliumC)
	escalation := createUser(t, ctx, octeliumC)
	fallback := createUser(t, ctx, octeliumC)
	catalog := createCatalog(t, ctx, octeliumC, nil)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(anyStep(userReviewer(requester))),
	})

	req := createRequest(t, ctx, octeliumC, baseRequest(requester, catalog))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	assert.True(t, reqG.Status.Review.Steps[0].IsUnroutable)
	assert.Equal(t, uint32(0), reqG.Status.Review.Steps[0].TotalEligibleReviewer)

	setClusterConfig(t, ctx, octeliumC, func(spec *accessv1.ClusterConfig_Spec) {
		spec.Review = &accessv1.ClusterConfig_Spec_Review{
			FallbackReviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				userReviewer(umetav1.GetObjectReference(fallback)),
			},
		}
	})

	req = createRequest(t, ctx, octeliumC, baseRequest(requester, catalog))
	reqG = converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	stepState := reqG.Status.Review.Steps[0]
	assert.False(t, stepState.IsUnroutable)
	require.Len(t, stepState.EligibleReviewers, 1)
	assert.Equal(t, fallback.Metadata.Uid, stepState.EligibleReviewers[0].Uid)

	step := anyStep(userReviewer(requester))
	step.Escalation = &accessv1.Policy_Spec_Rule_Action_Review_Step_Escalation{
		Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
			userReviewer(umetav1.GetObjectReference(escalation)),
		},
	}
	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Priority:  -1,
		Condition: matchAny(),
		Action:    reviewAction(step),
	})

	req = createRequest(t, ctx, octeliumC, baseRequest(requester, catalog))
	reqG = converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	stepState = reqG.Status.Review.Steps[0]
	assert.True(t, stepState.IsEscalated)
	assert.NotNil(t, stepState.EscalatedAt)
	require.Len(t, stepState.EligibleReviewers, 1)
	assert.Equal(t, escalation.Metadata.Uid, stepState.EligibleReviewers[0].Uid)
}

func TestRequestDeny(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_DENY,
		Condition: matchAny(),
		Message:   "Ask the security team",
	})

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_DENIED_BY_RULE, reqG.Status.State.Reason)
	assert.Equal(t, "Ask the security team", reqG.Status.State.Message)
	assert.Equal(t, accessv1.Policy_Spec_Rule_DENY, reqG.Status.Rule.Effect)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
}

func TestRequestNoMatch(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	setClusterConfig(t, ctx, octeliumC, func(spec *accessv1.ClusterConfig_Spec) {
		spec.Request = &accessv1.ClusterConfig_Spec_Request{
			NoMatchMessage: "No access Policy covers this Catalog",
		}
	})

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_NO_MATCHING_RULE, reqG.Status.State.Reason)
	assert.Equal(t, "No access Policy covers this Catalog", reqG.Status.State.Message)
	assert.Nil(t, reqG.Status.Rule)
	assert.Nil(t, reqG.Status.PolicyRef)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
	assert.NotNil(t, reqG.Status.ResolvedScope)
}

func TestRequestNotRequestable(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	unknownUser := createRequest(t, ctx, octeliumC,
		baseRequest(objRef("User"), createCatalog(t, ctx, octeliumC, nil)))
	reqG := converge(t, ctx, ctrl, octeliumC, unknownUser.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_NOT_REQUESTABLE, reqG.Status.State.Reason)

	serviceRequest := baseRequest(createUserRef(t, ctx, octeliumC), nil)
	serviceRequest.Spec.Resource = &accessv1.Request_Spec_Resource{
		Type: &accessv1.Request_Spec_Resource_ServiceRef{
			ServiceRef: umetav1.GetObjectReference(createService(t, ctx, octeliumC)),
		},
	}
	reqG = converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC, serviceRequest).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_NOT_REQUESTABLE, reqG.Status.State.Reason)

	disabled := createCatalog(t, ctx, octeliumC, nil)
	disabled.Spec.IsDisabled = true
	disabled, err := octeliumC.AccessC().UpdateCatalog(ctx, disabled)
	require.NoError(t, err)
	reqG = converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), disabled)).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_NOT_REQUESTABLE, reqG.Status.State.Reason)

	restricted := createCatalog(t, ctx, octeliumC, nil)
	restricted.Spec.Audience = &accessv1.Catalog_Spec_Audience{
		Users: []string{"someone-else"},
	}
	restricted, err = octeliumC.AccessC().UpdateCatalog(ctx, restricted)
	require.NoError(t, err)
	reqG = converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), restricted)).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_NOT_REQUESTABLE, reqG.Status.State.Reason)

	noLevels := createCatalog(t, ctx, octeliumC, nil)
	noLevels.Spec.AccessLevels = nil
	noLevels, err = octeliumC.AccessC().UpdateCatalog(ctx, noLevels)
	require.NoError(t, err)
	reqG = converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), noLevels)).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_NOT_REQUESTABLE, reqG.Status.State.Reason)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)
}

func TestRequestDisabledPolicySkipped(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, true, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Nil(t, reqG.Status.Rule)
}

func TestRequestPriorityOrder(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false,
		&accessv1.Policy_Spec_Rule{
			Name:      utilrand.GetRandomStringCanonical(6),
			Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
			Priority:  1,
			Condition: matchAny(),
		},
		&accessv1.Policy_Spec_Rule{
			Name:      utilrand.GetRandomStringCanonical(6),
			Effect:    accessv1.Policy_Spec_Rule_DENY,
			Priority:  10,
			Condition: matchAny(),
		},
	)

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Policy_Spec_Rule_AUTO_APPROVE, reqG.Status.Rule.Effect)
}

func TestRequestPriorityAcrossPolicies(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	catalog := createCatalog(t, ctx, octeliumC, nil)
	requester := createUserRef(t, ctx, octeliumC)

	defaultPol := createNamedPolicy(t, ctx, octeliumC,
		fmt.Sprintf("a-%s", utilrand.GetRandomStringCanonical(6)), false,
		&accessv1.Policy_Spec_Rule{
			Name:      "default",
			Effect:    accessv1.Policy_Spec_Rule_REVIEW,
			Condition: matchAny(),
			Action:    reviewAction(anyStep(userReviewer(createUserRef(t, ctx, octeliumC)))),
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

	reqMatch := createRequest(t, ctx, octeliumC, baseRequest(requester, catalog))
	reqMatchG := converge(t, ctx, ctrl, octeliumC, reqMatch.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)
	assert.Equal(t, autoPol.Metadata.Uid, reqMatchG.Status.PolicyRef.Uid)
	assert.Equal(t, autoPol.Spec.Rules[0].Name, reqMatchG.Status.Rule.Name)
	assert.Nil(t, reqMatchG.Status.Review)

	reqMiss := createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), catalog))
	reqMissG := converge(t, ctx, ctrl, octeliumC, reqMiss.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_PENDING, reqMissG.Status.State.Status)
	assert.Equal(t, defaultPol.Metadata.Uid, reqMissG.Status.PolicyRef.Uid)
	assert.Equal(t, "default", reqMissG.Status.Rule.Name)
}

func TestRequestPriorityTieBreak(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

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

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
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

	rules := accesscmd.GetSortedPolicyRules([]*accessv1.Policy{
		newPol("default", false, newRule("default", 0)),
		nil,
		newPol("usr1-weekday-auto-approve", false,
			newRule("usr1-weekdays-cairo", -1), nil, newRule("usr1-late", 5)),
		newPol("disabled", true, newRule("disabled", -100)),
		newPol("break-glass", false, newRule("deny-all", 0), newRule("deny-first", -10)),
	})

	var names []string
	for _, itm := range rules {
		names = append(names, fmt.Sprintf("%s/%s", itm.Policy.Metadata.Name, itm.Rule.Name))
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

	requester := createUser(t, ctx, octeliumC)
	subjRef := createUserRef(t, ctx, octeliumC)
	catalog := createCatalog(t, ctx, octeliumC, nil)
	catalog.Spec.Audience = &accessv1.Catalog_Spec_Audience{
		OnBehalf: &accessv1.Catalog_Spec_Audience_OnBehalf{
			Users: []string{requester.Metadata.Name},
		},
	}
	catalog, err := octeliumC.AccessC().UpdateCatalog(ctx, catalog)
	require.NoError(t, err)

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
	})

	reqMatch := baseRequest(umetav1.GetObjectReference(requester), catalog)
	reqMatch.Spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: subjRef,
		},
	}
	reqMatch = createRequest(t, ctx, octeliumC, reqMatch)
	reqMatchG := converge(t, ctx, ctrl, octeliumC, reqMatch.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)

	all := getPolicyTrigger(t, ctx, octeliumC, reqMatchG).Status.PreCondition.GetAll()
	assert.Equal(t, subjRef.Uid, all.Of[0].GetUserRef().Uid)

	reqMiss := baseRequest(umetav1.GetObjectReference(requester), catalog)
	reqMiss.Spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: createUserRef(t, ctx, octeliumC),
		},
	}
	reqMiss = createRequest(t, ctx, octeliumC, reqMiss)
	reqMissG := converge(t, ctx, ctrl, octeliumC, reqMiss.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqMissG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_NO_MATCHING_RULE, reqMissG.Status.State.Reason)

	notAllowed := baseRequest(createUserRef(t, ctx, octeliumC), catalog)
	notAllowed.Spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: subjRef,
		},
	}
	notAllowedG := converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC, notAllowed).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_NOT_REQUESTABLE, notAllowedG.Status.State.Reason)
}

func TestRequestUserRefCondition(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	catalog := createCatalog(t, ctx, octeliumC, nil)
	requester := createUserRef(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: requesterCondition(requester),
	})

	reqMatchG := converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(requester, catalog)).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)

	reqMissG := converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), catalog)).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqMissG.Status.State.Status)
}

func TestRequestInvalidReviewActions(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	for _, action := range []*accessv1.Policy_Spec_Rule_Action{
		reviewAction(),
		reviewAction(&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				userReviewer(objRef("User")),
			},
		}),
		reviewAction(&accessv1.Policy_Spec_Rule_Action_Review_Step{
			Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				userReviewer(objRef("User")),
			},
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT,
		}),
	} {
		catalog := createCatalog(t, ctx, octeliumC, nil)
		createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
			Name:   utilrand.GetRandomStringCanonical(6),
			Effect: accessv1.Policy_Spec_Rule_REVIEW,
			Condition: resourceCondition(&accessv1.Policy_Spec_Rule_Condition_Resource{
				Type: &accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef{
					CatalogRef: umetav1.GetObjectReference(catalog),
				},
			}),
			Action: action,
		})

		req := createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), catalog))
		assert.Nil(t, ctrl.OnAdd(ctx, req))

		reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
		assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
		assert.Equal(t, accessv1.Request_Status_State_EVALUATION_FAILED, reqG.Status.State.Reason)
	}
}

func TestRequestCELMatch(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	pol := createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: celMatch("true"),
	})

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
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

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
	reqG := converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)

	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Policy_Spec_Rule_DENY, reqG.Status.Rule.Effect)
}

func TestRequestCELMatchContext(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	svc := createService(t, ctx, octeliumC)
	catalog := createCatalog(t, ctx, octeliumC, []*accessv1.Catalog_Spec_AccessLevel{fullLevel("operate", nil)}, svc)
	userRef := createUserRef(t, ctx, octeliumC)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:   utilrand.GetRandomStringCanonical(6),
		Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: celMatch(fmt.Sprintf(
			`ctx.user.metadata.name == "%s" && ctx.requester.metadata.name == "%s" && `+
				`ctx.subject.metadata.name == "%s" && ctx.catalog.metadata.name == "%s" && `+
				`ctx.accessLevel.name == "operate" && ctx.services[0].metadata.name == "%s" && `+
				`ctx.request.spec.resource.catalog.catalogRef.uid == "%s"`,
			userRef.Name, userRef.Name, userRef.Name, catalog.Metadata.Name,
			svc.Metadata.Name, catalog.Metadata.Uid)),
	})

	reqMatchG := converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(userRef, catalog)).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqMatchG.Status.State.Status)

	reqOtherCatalogG := converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(userRef, createCatalog(t, ctx, octeliumC, nil))).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqOtherCatalogG.Status.State.Status)

	reqOtherUserG := converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), catalog)).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqOtherUserG.Status.State.Status)
	assert.Nil(t, reqOtherUserG.Status.Rule)
}

func TestRequestCELMatchErrorSkipsTheRule(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	broken := createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      "broken",
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Priority:  -1,
		Condition: celMatch("ctx.request.("),
	})

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      "fallback",
		Effect:    accessv1.Policy_Spec_Rule_DENY,
		Condition: matchAny(),
	})

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))
	assert.Nil(t, ctrl.OnAdd(ctx, req))

	reqG := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Equal(t, "fallback", reqG.Status.Rule.Name)
	require.Len(t, reqG.Status.Evaluation.SkippedRules, 1)
	assert.Equal(t, "broken", reqG.Status.Evaluation.SkippedRules[0].RuleName)
	assert.Equal(t, broken.Metadata.Uid, reqG.Status.Evaluation.SkippedRules[0].PolicyRef.Uid)
	assert.NotEmpty(t, reqG.Status.Evaluation.SkippedRules[0].Error)
}

func TestRequestFailedGuardrailBlocksApproval(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	guardrail := createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      "guardrail",
		Effect:    accessv1.Policy_Spec_Rule_DENY,
		Priority:  -1,
		Condition: celMatch("1 / 0 == 1"),
	})

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      "fallback",
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	req := createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil)))

	reqG := req
	for i := 0; i < maxEvaluationAttempts+1; i++ {
		reqG = converge(t, ctx, ctrl, octeliumC, req.Metadata.Uid)
	}

	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_EVALUATION_FAILED, reqG.Status.State.Reason)
	assert.Nil(t, reqG.Status.Rule)
	assert.Contains(t, reqG.Status.Evaluation.LastError, "guardrail")
	require.Len(t, reqG.Status.Evaluation.SkippedRules, 1)
	assert.Equal(t, guardrail.Metadata.Uid, reqG.Status.Evaluation.SkippedRules[0].PolicyRef.Uid)
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, reqG))
}

func TestRequestExpiredDeadlineIsNeverApproved(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	in := baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil))
	in.Spec.Deadline = pbutils.Timestamp(time.Now().Add(-time.Second))

	reqG := converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC, in).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_DEADLINE_PASSED, reqG.Status.State.Reason)
	assert.Nil(t, reqG.Status.Rule)
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, reqG))
}

func TestRequestEvaluationFailureRetries(t *testing.T) {
	ctrl := &Controller{}

	req := baseRequest(objRef("User"), nil)
	req.Status.Evaluation = &accessv1.Request_Status_Evaluation{}

	for i := 1; i < maxEvaluationAttempts; i++ {
		ctrl.setEvaluationFailure(req, errors.New("unavailable"))
		assert.Equal(t, accessv1.Request_Status_Evaluation_RETRYING, req.Status.Evaluation.State)
		assert.Equal(t, uint32(i), req.Status.Evaluation.Attempts)
		assert.True(t, needsEvaluation(withState(req, accessv1.Request_Status_State_PENDING)))
	}

	ctrl.setEvaluationFailure(req, errors.New("unavailable"))
	assert.Equal(t, accessv1.Request_Status_Evaluation_FAILED, req.Status.Evaluation.State)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, req.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_EVALUATION_FAILED, req.Status.State.Reason)
	assert.Equal(t, "unavailable", req.Status.Evaluation.LastError)
}

func withState(req *accessv1.Request, status accessv1.Request_Status_State_Status) *accessv1.Request {
	ret := pbutils.Clone(req).(*accessv1.Request)
	ret.Status.State = &accessv1.Request_Status_State{
		Status: status,
	}
	return ret
}

func autoApprovedRequest(t *testing.T, ctx context.Context, ctrl *Controller,
	octeliumC octeliumc.ClientInterface) *accessv1.Request {
	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	catalog := createCatalog(t, ctx, octeliumC,
		[]*accessv1.Catalog_Spec_AccessLevel{fullLevel("full", durationHours(1))})

	req := createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), catalog))
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

func TestRequestExpire(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqApproved := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	ptUid := reqApproved.Status.PolicyTriggerRef.Uid

	reqApproved.Status.AccessEndsAt = pbutils.Timestamp(time.Now().Add(-2 * time.Hour))
	_, err := octeliumC.AccessC().UpdateRequest(ctx, reqApproved)
	assert.Nil(t, err, "%+v", err)

	reqG := converge(t, ctx, ctrl, octeliumC, reqApproved.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_ACCESS_ENDED, reqG.Status.State.Reason)
	assert.Equal(t, accessv1.Request_Status_Grant_ENDED, reqG.Status.Grant.State)
	assert.NotNil(t, reqG.Status.Grant.EndedAt)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)

	_, err = octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{Uid: ptUid})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestRequestRevoke(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqApproved := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	ptUid := reqApproved.Status.PolicyTriggerRef.Uid

	accesscmd.SetRequestState(reqApproved, &accesscmd.StateChange{
		Status: accessv1.Request_Status_State_REVOKED,
		Reason: accessv1.Request_Status_State_ACCESS_REVOKED,
	})
	_, err := octeliumC.AccessC().UpdateRequest(ctx, reqApproved)
	assert.Nil(t, err, "%+v", err)

	reqG := converge(t, ctx, ctrl, octeliumC, reqApproved.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REVOKED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_Grant_ENDED, reqG.Status.Grant.State)
	assert.Nil(t, reqG.Status.PolicyTriggerRef)

	_, err = octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{Uid: ptUid})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestRequestLateReconcileDoesNotRecreateARevokedGrant(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqApproved := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	stale := pbutils.Clone(reqApproved).(*accessv1.Request)

	accesscmd.SetRequestState(reqApproved, &accesscmd.StateChange{
		Status: accessv1.Request_Status_State_REVOKED,
		Reason: accessv1.Request_Status_State_ACCESS_REVOKED,
	})
	_, err := octeliumC.AccessC().UpdateRequest(ctx, reqApproved)
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, accesscmd.DeletePolicyTriggerOf(ctx, octeliumC, reqApproved))
	require.False(t, policyTriggerExists(t, ctx, octeliumC, reqApproved))

	_ = ctrl.OnAdd(ctx, stale)
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, reqApproved))
	assert.Equal(t, accessv1.Request_Status_State_REVOKED,
		getRequest(t, ctx, octeliumC, reqApproved.Metadata.Uid).Status.State.Status)
}

func TestRequestOnDelete(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqApproved := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	ptUid := reqApproved.Status.PolicyTriggerRef.Uid

	assert.Nil(t, ctrl.OnDelete(ctx, reqApproved))

	_, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{Uid: ptUid})
	assert.True(t, grpcerr.IsNotFound(err))
}

func TestRequestAnchorNoSliding(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqA := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	ptUidA := reqA.Status.PolicyTriggerRef.Uid
	accessEndsA := reqA.Status.AccessEndsAt.AsTime()

	reqB := converge(t, ctx, ctrl, octeliumC, reqA.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqB.Status.State.Status)
	assert.Equal(t, ptUidA, reqB.Status.PolicyTriggerRef.Uid)
	assert.Equal(t, accessEndsA, reqB.Status.AccessEndsAt.AsTime())

	pt, err := octeliumC.CoreC().GetPolicyTrigger(ctx, &rmetav1.GetOptions{Uid: ptUidA})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, accessEndsA, pt.Status.PreCondition.GetAll().Of[3].GetNotAfter().AsTime())
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
		ptBefore := getPolicyTrigger(t, ctx, octeliumC, before)

		assert.Nil(t, ctrl.OnAdd(ctx, before), "pass %d", i)

		after := getRequest(t, ctx, octeliumC, req.Metadata.Uid)
		ptAfter := getPolicyTrigger(t, ctx, octeliumC, before)

		assert.Equal(t, before.Metadata.ResourceVersion, after.Metadata.ResourceVersion, "pass %d", i)
		assert.Equal(t, ptBefore.Metadata.ResourceVersion, ptAfter.Metadata.ResourceVersion, "pass %d", i)
	}
}

func TestRequestRefsHaveNoResourceVersion(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	req := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	assert.Empty(t, req.Status.PolicyTriggerRef.ResourceVersion)
	assert.Empty(t, req.Status.PolicyRef.ResourceVersion)
	assert.Empty(t, req.Status.ResolvedScope.CatalogRef.ResourceVersion)

	pt := getPolicyTrigger(t, ctx, octeliumC, req)
	assert.Empty(t, pt.Status.OwnerRef.ResourceVersion)

	all := pt.Status.PreCondition.GetAll()
	require.NotNil(t, all)
	assert.Empty(t, all.Of[0].GetUserRef().ResourceVersion)
	assert.Empty(t, all.Of[1].GetServiceRef().ResourceVersion)
}

func policyTriggerAllowPriority(t *testing.T, ctx context.Context,
	octeliumC octeliumc.ClientInterface, req *accessv1.Request) int32 {
	pt := getPolicyTrigger(t, ctx, octeliumC, req)
	assert.Equal(t, "access-request-default", pt.Status.InlinePolicies[0].Name)
	assert.Equal(t, 1, len(pt.Status.InlinePolicies[0].Spec.Rules))

	return pt.Status.InlinePolicies[0].Spec.Rules[0].Priority
}

func TestRequestPolicyTriggerDefaultPriority(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	assert.Equal(t, int32(0), policyTriggerAllowPriority(t, ctx, octeliumC, reqG))
	assert.False(t, reqG.Status.ResolvedScope.OverridesGuardrails)
}

func TestRequestPolicyTriggerPriorityIsFrozen(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	priority := int32(-3)
	setClusterConfig(t, ctx, octeliumC, func(spec *accessv1.ClusterConfig_Spec) {
		spec.Authorization = &accessv1.ClusterConfig_Spec_Authorization{
			Priority: &priority,
		}
	})

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	assert.Equal(t, int32(-3), policyTriggerAllowPriority(t, ctx, octeliumC, reqG))
	assert.True(t, reqG.Status.ResolvedScope.OverridesGuardrails)

	setClusterConfig(t, ctx, octeliumC, func(spec *accessv1.ClusterConfig_Spec) {
		spec.Authorization = nil
	})

	reqG = converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, int32(-3), policyTriggerAllowPriority(t, ctx, octeliumC, reqG))

	explicit := int32(-2)
	level := fullLevel("break-glass", nil)
	level.GetFull().Priority = &explicit
	catalog := createCatalog(t, ctx, octeliumC, []*accessv1.Catalog_Spec_AccessLevel{level})

	reqG = converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), catalog)).Metadata.Uid)
	assert.Equal(t, int32(-2), policyTriggerAllowPriority(t, ctx, octeliumC, reqG))
	assert.True(t, reqG.Status.ResolvedScope.OverridesGuardrails)
}

func TestGetDefaultAccessPriority(t *testing.T) {
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
		{"nil", nil, 0},
		{"no spec", &accessv1.ClusterConfig{}, 0},
		{"no authorization", &accessv1.ClusterConfig{Spec: &accessv1.ClusterConfig_Spec{}}, 0},
		{"no priority", &accessv1.ClusterConfig{
			Spec: &accessv1.ClusterConfig_Spec{
				Authorization: &accessv1.ClusterConfig_Spec_Authorization{},
			},
		}, 0},
		{"zero", priority(0), 0},
		{"highest", priority(-4), -4},
		{"middle", priority(-2), -2},
		{"below range", priority(-9), -4},
		{"above range", priority(3), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, accesscmd.GetDefaultAccessPriority(tt.cc))
		})
	}
}

func TestRequestGroupConditionsUseTheDelegatedSubject(t *testing.T) {
	ctx, ctl, client := newControllerTest(t)

	group, err := client.CoreC().CreateGroup(ctx, &corev1.Group{Metadata: &metav1.Metadata{Name: "request-group"}, Spec: &corev1.Group_Spec{}, Status: &corev1.Group_Status{}})
	require.NoError(t, err)
	member := createUser(t, ctx, client, group.Metadata.Name)
	requester := createUser(t, ctx, client)

	condition := &accessv1.Policy_Spec_Rule_Condition{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{
		Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{Type: &accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef{GroupRef: umetav1.GetObjectReference(group)}},
	}}

	createPolicy(t, ctx, client, false, &accessv1.Policy_Spec_Rule{
		Name:      "group",
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: condition,
	})

	res, err := ctl.evaluator.Evaluate(ctx, &accesscmd.EvalInput{
		Requester: requester,
		Subject:   member,
	}, false)
	require.NoError(t, err)
	require.NotNil(t, res.Rule)
	assert.Equal(t, "group", res.Rule.Name)

	res, err = ctl.evaluator.Evaluate(ctx, &accesscmd.EvalInput{
		Requester: member,
		Subject:   requester,
	}, false)
	require.NoError(t, err)
	assert.Nil(t, res.Rule)
}

func TestRequestResourceConditionsMatchTheResolvedScope(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	ns, err := octeliumC.CoreC().CreateNamespace(ctx, &corev1.Namespace{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &corev1.Namespace_Spec{},
		Status:   &corev1.Namespace_Status{},
	})
	require.NoError(t, err)

	nsSvc, err := octeliumC.CoreC().CreateService(ctx, &corev1.Service{
		Metadata: &metav1.Metadata{Name: fmt.Sprintf("svc.%s", ns.Metadata.Name)},
		Spec:     &corev1.Service_Spec{},
		Status: &corev1.Service_Status{
			NamespaceRef: umetav1.GetObjectReference(ns),
		},
	})
	require.NoError(t, err)

	svc := createService(t, ctx, octeliumC)
	catalog := createCatalog(t, ctx, octeliumC,
		[]*accessv1.Catalog_Spec_AccessLevel{fullLevel("read", nil), fullLevel("write", nil)}, svc)
	catalog.Spec.ResourceCollection.Service.Namespaces = []string{ns.Metadata.Name}
	catalog, err = octeliumC.AccessC().UpdateCatalog(ctx, catalog)
	require.NoError(t, err)

	requester := createUser(t, ctx, octeliumC)
	subject := createUser(t, ctx, octeliumC)
	catalog.Spec.Audience = &accessv1.Catalog_Spec_Audience{
		OnBehalf: &accessv1.Catalog_Spec_Audience_OnBehalf{Users: []string{requester.Metadata.Name}},
	}
	catalog, err = octeliumC.AccessC().UpdateCatalog(ctx, catalog)
	require.NoError(t, err)

	for _, tc := range []struct {
		condition *accessv1.Policy_Spec_Rule_Condition
		level     string
		onBehalf  bool
		want      bool
	}{
		{resourceCondition(&accessv1.Policy_Spec_Rule_Condition_Resource{
			Type: &accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef{ServiceRef: umetav1.GetObjectReference(svc)},
		}), "", false, true},
		{resourceCondition(&accessv1.Policy_Spec_Rule_Condition_Resource{
			Type: &accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef{ServiceRef: umetav1.GetObjectReference(nsSvc)},
		}), "", false, true},
		{resourceCondition(&accessv1.Policy_Spec_Rule_Condition_Resource{
			Type: &accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef{ServiceRef: umetav1.GetObjectReference(createService(t, ctx, octeliumC))},
		}), "", false, false},
		{resourceCondition(&accessv1.Policy_Spec_Rule_Condition_Resource{
			Type: &accessv1.Policy_Spec_Rule_Condition_Resource_NamespaceRef{NamespaceRef: umetav1.GetObjectReference(ns)},
		}), "", false, true},
		{resourceCondition(&accessv1.Policy_Spec_Rule_Condition_Resource{
			Type: &accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef{CatalogRef: umetav1.GetObjectReference(catalog)},
		}), "", false, true},
		{&accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_AccessLevel{AccessLevel: "write"},
		}, "write", false, true},
		{&accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_AccessLevel{AccessLevel: "write"},
		}, "read", false, false},
		{&accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_IsOnBehalf{IsOnBehalf: true},
		}, "", true, true},
		{&accessv1.Policy_Spec_Rule_Condition{
			Type: &accessv1.Policy_Spec_Rule_Condition_IsOnBehalf{IsOnBehalf: true},
		}, "", false, false},
	} {
		pol := createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
			Name:      utilrand.GetRandomStringCanonical(6),
			Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
			Condition: tc.condition,
		})

		in := baseRequest(umetav1.GetObjectReference(requester), catalog)
		in.Spec.Resource.GetCatalog().AccessLevel = tc.level
		if tc.onBehalf {
			in.Spec.Subject = &accessv1.Request_Spec_Subject{
				Type: &accessv1.Request_Spec_Subject_UserRef{UserRef: umetav1.GetObjectReference(subject)},
			}
		}

		reqG := converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC, in).Metadata.Uid)
		if tc.want {
			assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
		} else {
			assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
		}

		_, err := octeliumC.AccessC().DeletePolicy(ctx, &rmetav1.DeleteOptions{Uid: pol.Metadata.Uid})
		require.NoError(t, err)
	}
}

func TestCatalogRequestScopeServicesAndNamespaces(t *testing.T) {
	ctx, ctrl, client := newControllerTest(t)

	createPolicy(t, ctx, client, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	svc, err := client.CoreC().CreateService(ctx, &corev1.Service{Metadata: &metav1.Metadata{Name: "catalog-service.default"}, Spec: &corev1.Service_Spec{}, Status: &corev1.Service_Status{}})
	require.NoError(t, err)
	ns, err := client.CoreC().CreateNamespace(ctx, &corev1.Namespace{Metadata: &metav1.Metadata{Name: "catalog-namespace"}, Spec: &corev1.Namespace_Spec{}, Status: &corev1.Namespace_Status{}})
	require.NoError(t, err)
	nsSvc, err := client.CoreC().CreateService(ctx, &corev1.Service{
		Metadata: &metav1.Metadata{Name: "member.catalog-namespace"},
		Spec:     &corev1.Service_Spec{},
		Status:   &corev1.Service_Status{NamespaceRef: umetav1.GetObjectReference(ns)},
	})
	require.NoError(t, err)

	newCatalog := func(membership accessv1.Catalog_Spec_Membership) *accessv1.Catalog {
		catalog, err := client.AccessC().CreateCatalog(ctx, &accessv1.Catalog{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec: &accessv1.Catalog_Spec{
				ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services:   []string{"", svc.Metadata.Name, "missing.default"},
					Namespaces: []string{"", ns.Metadata.Name, "missing"},
				}},
				AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{fullLevel("full", nil)},
				Membership:   membership,
			},
		})
		require.NoError(t, err)
		return catalog
	}

	fixed := converge(t, ctx, ctrl, client,
		createRequest(t, ctx, client, baseRequest(createUserRef(t, ctx, client), newCatalog(accessv1.Catalog_Spec_MEMBERSHIP_UNSET))).Metadata.Uid)
	require.Equal(t, accessv1.Request_Status_State_APPROVED, fixed.Status.State.Status)
	assert.Equal(t, accessv1.Catalog_Spec_FIXED, fixed.Status.ResolvedScope.Membership)
	require.Len(t, fixed.Status.ResolvedScope.ServiceRefs, 2)
	assert.Equal(t, svc.Metadata.Uid, fixed.Status.ResolvedScope.ServiceRefs[0].Uid)
	assert.Equal(t, nsSvc.Metadata.Uid, fixed.Status.ResolvedScope.ServiceRefs[1].Uid)
	assert.Empty(t, fixed.Status.ResolvedScope.NamespaceRefs)

	target := getPolicyTrigger(t, ctx, client, fixed).Status.PreCondition.GetAll().Of[1]
	require.NotNil(t, target.GetAny())
	require.Len(t, target.GetAny().Of, 2)
	assert.Equal(t, svc.Metadata.Uid, target.GetAny().Of[0].GetServiceRef().Uid)
	assert.Equal(t, nsSvc.Metadata.Uid, target.GetAny().Of[1].GetServiceRef().Uid)

	dynamic := converge(t, ctx, ctrl, client,
		createRequest(t, ctx, client, baseRequest(createUserRef(t, ctx, client), newCatalog(accessv1.Catalog_Spec_DYNAMIC))).Metadata.Uid)
	require.Equal(t, accessv1.Request_Status_State_APPROVED, dynamic.Status.State.Status)
	require.Len(t, dynamic.Status.ResolvedScope.NamespaceRefs, 1)
	assert.Equal(t, ns.Metadata.Uid, dynamic.Status.ResolvedScope.NamespaceRefs[0].Uid)

	target = getPolicyTrigger(t, ctx, client, dynamic).Status.PreCondition.GetAll().Of[1]
	require.Len(t, target.GetAny().Of, 3)
	nsTarget := target.GetAny().Of[2].GetAll()
	require.Len(t, nsTarget.Of, 2)
	assert.Equal(t, ns.Metadata.Uid, nsTarget.Of[0].GetNamespaceRef().Uid)
	assert.NotEmpty(t, nsTarget.Of[1].GetCondition().GetMatch())
}

func TestCatalogRequestPartialServices(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	svcA := createService(t, ctx, octeliumC)
	svcB := createService(t, ctx, octeliumC)
	catalog := createCatalog(t, ctx, octeliumC, nil, svcA, svcB)

	in := baseRequest(createUserRef(t, ctx, octeliumC), catalog)
	in.Spec.Resource.GetCatalog().ServiceRefs = []*metav1.ObjectReference{umetav1.GetObjectReference(svcB)}

	reqG := converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC, in).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_NOT_REQUESTABLE, reqG.Status.State.Reason)

	catalog.Spec.AllowPartialRequests = true
	catalog, err := octeliumC.AccessC().UpdateCatalog(ctx, catalog)
	require.NoError(t, err)

	in = baseRequest(createUserRef(t, ctx, octeliumC), catalog)
	in.Spec.Resource.GetCatalog().ServiceRefs = []*metav1.ObjectReference{umetav1.GetObjectReference(svcB)}

	reqG = converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC, in).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	require.Len(t, reqG.Status.ResolvedScope.ServiceRefs, 1)
	assert.Equal(t, svcB.Metadata.Uid, reqG.Status.ResolvedScope.ServiceRefs[0].Uid)
	assert.Equal(t, svcB.Metadata.Uid, getPolicyTrigger(t, ctx, octeliumC, reqG).Status.PreCondition.GetAll().Of[1].GetServiceRef().Uid)
}

func TestCatalogExcludesSystemServices(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	svc := createService(t, ctx, octeliumC)
	hidden, err := octeliumC.CoreC().CreateService(ctx, &corev1.Service{
		Metadata: &metav1.Metadata{
			Name:         fmt.Sprintf("%s.default", utilrand.GetRandomStringCanonical(8)),
			IsSystem:     true,
			IsUserHidden: true,
		},
		Spec:   &corev1.Service_Spec{},
		Status: &corev1.Service_Status{},
	})
	require.NoError(t, err)

	catalog := createCatalog(t, ctx, octeliumC, nil, svc, hidden)

	reqG := converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), catalog)).Metadata.Uid)
	require.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	require.Len(t, reqG.Status.ResolvedScope.ServiceRefs, 1)
	assert.Equal(t, svc.Metadata.Uid, reqG.Status.ResolvedScope.ServiceRefs[0].Uid)
	assert.Equal(t, uint32(1), reqG.Status.ResolvedScope.TotalExcludedService)

	hiddenOnly := createCatalog(t, ctx, octeliumC, nil, hidden)
	reqG = converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), hiddenOnly)).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_NOT_REQUESTABLE, reqG.Status.State.Reason)

	hiddenOnly.Spec.AllowSystemResources = true
	hiddenOnly, err = octeliumC.AccessC().UpdateCatalog(ctx, hiddenOnly)
	require.NoError(t, err)

	reqG = converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), hiddenOnly)).Metadata.Uid)
	require.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, hidden.Metadata.Uid, reqG.Status.ResolvedScope.ServiceRefs[0].Uid)
}

func TestRequestScopeIsFrozenOnceApproved(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	ptBefore := getPolicyTrigger(t, ctx, octeliumC, reqG)

	catalog, err := octeliumC.AccessC().GetCatalog(ctx, &rmetav1.GetOptions{Uid: reqG.Status.ResolvedScope.CatalogRef.Uid})
	require.NoError(t, err)
	catalog.Spec.ResourceCollection.Service.Services = append(catalog.Spec.ResourceCollection.Service.Services,
		createService(t, ctx, octeliumC).Metadata.Name)
	_, err = octeliumC.AccessC().UpdateCatalog(ctx, catalog)
	require.NoError(t, err)

	reqAfter := converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Len(t, reqAfter.Status.ResolvedScope.ServiceRefs, 1)
	assert.Equal(t, ptBefore.Metadata.ResourceVersion, getPolicyTrigger(t, ctx, octeliumC, reqAfter).Metadata.ResourceVersion)

	_, err = octeliumC.AccessC().DeleteCatalog(ctx, &rmetav1.DeleteOptions{Uid: catalog.Metadata.Uid})
	require.NoError(t, err)

	reqAfter = converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqAfter.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_Grant_ACTIVE, reqAfter.Status.Grant.State)
	assert.True(t, policyTriggerExists(t, ctx, octeliumC, reqAfter))
}

func TestRequestPendingScopeFollowsTheCatalog(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(anyStep(userReviewer(createUserRef(t, ctx, octeliumC)))),
	})

	catalog := createCatalog(t, ctx, octeliumC, nil)
	reqG := converge(t, ctx, ctrl, octeliumC,
		createRequest(t, ctx, octeliumC, baseRequest(createUserRef(t, ctx, octeliumC), catalog)).Metadata.Uid)
	require.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)
	require.Len(t, reqG.Status.ResolvedScope.ServiceRefs, 1)

	catalog.Spec.ResourceCollection.Service.Services = append(catalog.Spec.ResourceCollection.Service.Services,
		createService(t, ctx, octeliumC).Metadata.Name)
	catalog, err := octeliumC.AccessC().UpdateCatalog(ctx, catalog)
	require.NoError(t, err)

	reqG = converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Len(t, reqG.Status.ResolvedScope.ServiceRefs, 2)
	assert.Equal(t, catalog.Metadata.ResourceVersion, reqG.Status.ResolvedScope.CatalogResourceVersion)
	assert.Equal(t, uint32(2), reqG.Status.SpecRevision)
	require.Len(t, reqG.Status.Review.Steps, 2)
	assert.Equal(t, accessv1.Request_Status_Review_StepState_INTERRUPTED, reqG.Status.Review.Steps[0].Outcome)
	assert.Equal(t, accessv1.Request_Status_Review_StepState_OUTCOME_UNSET, reqG.Status.Review.Steps[1].Outcome)
	assert.NotNil(t, reqG.Status.Rule)

	catalog.Metadata.DisplayName = "Renamed"
	catalog, err = octeliumC.AccessC().UpdateCatalog(ctx, catalog)
	require.NoError(t, err)

	reqG = converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Equal(t, "Renamed", reqG.Status.ResolvedScope.CatalogDisplayName)
	assert.Equal(t, uint32(2), reqG.Status.SpecRevision)
	assert.Len(t, reqG.Status.Review.Steps, 2)

	_, err = octeliumC.AccessC().DeleteCatalog(ctx, &rmetav1.DeleteOptions{Uid: catalog.Metadata.Uid})
	require.NoError(t, err)

	reqG = converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_REJECTED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_NOT_REQUESTABLE, reqG.Status.State.Reason)
	assert.Equal(t, accessv1.Request_Status_Review_StepState_INTERRUPTED, reqG.Status.Review.Steps[1].Outcome)
}

func TestRequestScheduledStart(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	startAt := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	in := baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC,
		[]*accessv1.Catalog_Spec_AccessLevel{fullLevel("full", durationHours(1))}))
	in.Spec.StartAt = pbutils.Timestamp(startAt)

	reqG := converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC, in).Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_Grant_SCHEDULED, reqG.Status.Grant.State)
	assert.Nil(t, reqG.Status.Grant.ActivatedAt)
	assert.Equal(t, startAt, reqG.Status.AccessStartsAt.AsTime())
	assert.Equal(t, startAt.Add(time.Hour), reqG.Status.AccessEndsAt.AsTime())

	all := getPolicyTrigger(t, ctx, octeliumC, reqG).Status.PreCondition.GetAll()
	assert.Equal(t, startAt, all.Of[2].GetNotBefore().AsTime())
}

func TestRequestVerificationPreCondition(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
		Condition: matchAny(),
	})

	level := fullLevel("admin", nil)
	level.Verification = &accessv1.Catalog_Spec_AccessLevel_Verification{
		MinAAL: corev1.Session_Status_Authentication_Info_AAL2,
	}

	reqG := converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC,
			[]*accessv1.Catalog_Spec_AccessLevel{level}))).Metadata.Uid)
	require.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)

	all := getPolicyTrigger(t, ctx, octeliumC, reqG).Status.PreCondition.GetAll()
	require.Len(t, all.Of, 5)
	match := all.Of[4].GetCondition().GetMatch()
	assert.True(t, strings.Contains(match, `"AAL2"`))
	assert.True(t, strings.Contains(match, `"AAL3"`))
	assert.False(t, strings.Contains(match, `"AAL1"`))
}

func TestRequestExtensionSupersedesTheExtendedRequest(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	original := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	in := baseRequest(original.Status.UserRef, nil)
	in.Spec.Resource = pbutils.Clone(original.Spec.Resource).(*accessv1.Request_Spec_Resource)
	in.Spec.ExtendsRef = umetav1.GetObjectReference(original)

	extension := converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC, in).Metadata.Uid)
	require.Equal(t, accessv1.Request_Status_State_APPROVED, extension.Status.State.Status)
	require.Equal(t, accessv1.Request_Status_Grant_ACTIVE, extension.Status.Grant.State)

	original = converge(t, ctx, ctrl, octeliumC, original.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED, original.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_SUPERSEDED, original.Status.State.Reason)
	assert.Equal(t, extension.Metadata.Uid, original.Status.SupersededByRef.Uid)
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, original))
	assert.True(t, policyTriggerExists(t, ctx, octeliumC, extension))
}

func TestRequestExtensionNeverShortensTheExtendedRequest(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	original := autoApprovedRequest(t, ctx, ctrl, octeliumC)

	in := baseRequest(original.Status.UserRef, nil)
	in.Spec.Resource = pbutils.Clone(original.Spec.Resource).(*accessv1.Request_Spec_Resource)
	in.Spec.ExtendsRef = umetav1.GetObjectReference(original)
	in.Spec.Duration = &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 10}}

	extension := converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC, in).Metadata.Uid)
	require.Equal(t, accessv1.Request_Status_State_APPROVED, extension.Status.State.Status)
	require.Equal(t, accessv1.Request_Status_Grant_ACTIVE, extension.Status.Grant.State)

	original = converge(t, ctx, ctrl, octeliumC, original.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, original.Status.State.Status)
	assert.Nil(t, original.Status.SupersededByRef)
	assert.True(t, policyTriggerExists(t, ctx, octeliumC, original))
	assert.True(t, policyTriggerExists(t, ctx, octeliumC, extension))
}

func TestRequestStaleApprovalsRestartTheWorkflow(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	first := anyStep(userReviewer(createUserRef(t, ctx, octeliumC)))
	first.Name = "first"
	second := anyStep(userReviewer(createUserRef(t, ctx, octeliumC)))
	second.Name = "second"

	createPolicy(t, ctx, octeliumC, false, &accessv1.Policy_Spec_Rule{
		Name:      utilrand.GetRandomStringCanonical(6),
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: matchAny(),
		Action:    reviewAction(first, second),
	})

	reqG := converge(t, ctx, ctrl, octeliumC, createRequest(t, ctx, octeliumC,
		baseRequest(createUserRef(t, ctx, octeliumC), createCatalog(t, ctx, octeliumC, nil))).Metadata.Uid)
	require.Equal(t, accessv1.Request_Status_State_PENDING, reqG.Status.State.Status)

	accesscmd.EndCurrentStep(reqG, accessv1.Request_Status_Review_StepState_SATISFIED)
	reqG.Status.Review.LastSteps = append(reqG.Status.Review.LastSteps, &accessv1.Request_Status_Review_Step{
		StepIndex:    0,
		StepName:     "first",
		Decision:     accessv1.Review_Spec_DECISION_APPROVE,
		SpecRevision: 1,
	})
	require.NoError(t, accesscmd.StartReviewStep(ctx, &accesscmd.StartReviewStepOpts{
		OcteliumC: octeliumC,
		Request:   reqG,
		Index:     1,
	}))
	reqG, err := octeliumC.AccessC().UpdateRequest(ctx, reqG)
	require.NoError(t, err)

	reqG = converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Equal(t, int32(1), reqG.Status.Review.CurrentStep)

	reqG.Spec.Justification = "changed after the first approval"
	reqG.Status.SpecRevision = 2
	_, err = octeliumC.AccessC().UpdateRequest(ctx, reqG)
	require.NoError(t, err)

	reqG = converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Equal(t, int32(0), reqG.Status.Review.CurrentStep)
	require.Len(t, reqG.Status.Review.Steps, 3)
	assert.Equal(t, accessv1.Request_Status_Review_StepState_INTERRUPTED, reqG.Status.Review.Steps[1].Outcome)
	assert.Equal(t, "first", reqG.Status.Review.Steps[2].Name)
	assert.Equal(t, accessv1.Request_Status_Review_StepState_OUTCOME_UNSET, reqG.Status.Review.Steps[2].Outcome)
}

func TestRequestProvisioningFailure(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	require.NoError(t, accesscmd.DeletePolicyTriggerOf(ctx, octeliumC, reqG))

	reqG.Status.ResolvedScope.ServiceRefs = nil
	reqG.Status.Grant.State = accessv1.Request_Status_Grant_PROVISIONING
	reqG, err := octeliumC.AccessC().UpdateRequest(ctx, reqG)
	require.NoError(t, err)

	for i := 1; i < maxProvisioningAttempts; i++ {
		require.NoError(t, ctrl.OnAdd(ctx, getRequest(t, ctx, octeliumC, reqG.Metadata.Uid)))
		cur := getRequest(t, ctx, octeliumC, reqG.Metadata.Uid)
		assert.Equal(t, accessv1.Request_Status_State_APPROVED, cur.Status.State.Status)
		assert.Equal(t, accessv1.Request_Status_Grant_PROVISIONING, cur.Status.Grant.State)
		assert.Equal(t, uint32(i), cur.Status.Grant.Attempts)
		assert.NotEmpty(t, cur.Status.Grant.LastError)
	}

	reqG = converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_PROVISIONING_FAILED, reqG.Status.State.Reason)
	assert.Equal(t, accessv1.Request_Status_Grant_FAILED, reqG.Status.Grant.State)
	assert.False(t, policyTriggerExists(t, ctx, octeliumC, reqG))
}

func TestRequestWithoutScopeKeepsItsExistingPolicyTrigger(t *testing.T) {
	ctx, ctrl, octeliumC := newControllerTest(t)

	reqG := autoApprovedRequest(t, ctx, ctrl, octeliumC)
	reqG.Status.ResolvedScope = nil
	reqG.Status.Grant = nil
	reqG, err := octeliumC.AccessC().UpdateRequest(ctx, reqG)
	require.NoError(t, err)

	reqG = converge(t, ctx, ctrl, octeliumC, reqG.Metadata.Uid)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_Grant_ACTIVE, reqG.Status.Grant.State)
	assert.True(t, policyTriggerExists(t, ctx, octeliumC, reqG))
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

func TestPolicyTriggerNameAndRefCopies(t *testing.T) {
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
