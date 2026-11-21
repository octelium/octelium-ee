// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"
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
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/user"
	"github.com/octelium/octelium/cluster/common/tests/tstuser"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lifecycleTest struct {
	ctx         context.Context
	octeliumC   octeliumc.ClientInterface
	adminSrv    *admin.Server
	usrSrv      *user.Server
	mainSrv     *ServerMain
	userSrv     *ServerUser
	reviewerSrv *ServerReviewer
}

func newLifecycleTest(t *testing.T) *lifecycleTest {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	userSrv, err := NewServerUser(ctx, tst.C.OcteliumC)
	require.NoError(t, err)

	return &lifecycleTest{
		ctx:       ctx,
		octeliumC: tst.C.OcteliumC,
		adminSrv: admin.NewServer(&admin.Opts{
			OcteliumC:  tst.C.OcteliumC,
			IsEmbedded: true,
		}),
		usrSrv:      user.NewServer(tst.C.OcteliumC),
		mainSrv:     NewServerMain(tst.C.OcteliumC),
		userSrv:     userSrv,
		reviewerSrv: NewServerReviewer(tst.C.OcteliumC),
	}
}

func (l *lifecycleTest) newUser(t *testing.T) *tstuser.User {
	usr, err := tstuser.NewUser(l.octeliumC, l.adminSrv, l.usrSrv, nil)
	require.NoError(t, err)
	return usr
}

func (l *lifecycleTest) newCatalog(t *testing.T, fn func(spec *accessv1.Catalog_Spec)) *accessv1.Catalog {
	spec := &accessv1.Catalog_Spec{
		ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
			Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
				Services: []string{newTestServiceName(t, l.octeliumC)},
			},
		},
		AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{testFullLevel("full")},
	}

	if fn != nil {
		fn(spec)
	}

	cat, err := l.mainSrv.CreateCatalog(l.ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     spec,
	})
	require.NoError(t, err)
	return cat
}

func catalogCondition(cat *accessv1.Catalog) *accessv1.Policy_Spec_Rule_Condition {
	return &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{
			Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{
				Type: &accessv1.Policy_Spec_Rule_Condition_Resource_CatalogRef{
					CatalogRef: umetav1.GetObjectReference(cat),
				},
			},
		},
	}
}

func userReviewerStep(name string, users ...*tstuser.User) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	ret := &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Name:                name,
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
	}

	for _, usr := range users {
		ret.Reviewers = append(ret.Reviewers, &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
				User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
					UserRef: umetav1.GetObjectReference(usr.Usr),
				},
			},
		})
	}

	return ret
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

func (l *lifecycleTest) newPolicy(t *testing.T, rules ...*accessv1.Policy_Spec_Rule) *accessv1.Policy {
	pol, err := l.mainSrv.CreatePolicy(l.ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &accessv1.Policy_Spec{Rules: rules},
	})
	require.NoError(t, err)
	return pol
}

func catalogRequestSpec(ref *metav1.ObjectReference) *accessv1.Request_Spec {
	return &accessv1.Request_Spec{
		Urgency: accessv1.Request_Spec_NORMAL,
		Resource: &accessv1.Request_Spec_Resource{
			Type: &accessv1.Request_Spec_Resource_Catalog_{
				Catalog: &accessv1.Request_Spec_Resource_Catalog{
					CatalogRef: ref,
				},
			},
		},
	}
}

func (l *lifecycleTest) setPendingReview(t *testing.T, req *accessv1.Request,
	step *accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Request {
	req, err := l.octeliumC.AccessC().GetRequest(l.ctx, &rmetav1.GetOptions{Uid: req.Metadata.Uid})
	require.NoError(t, err)

	req.Status.Rule = &accessv1.Policy_Spec_Rule{
		Name:   "review",
		Effect: accessv1.Policy_Spec_Rule_REVIEW,
		Action: reviewAction(step),
	}
	req.Status.State.Reason = accessv1.Request_Status_State_AWAITING_REVIEW
	require.NoError(t, accesscmd.StartReviewStep(l.ctx, &accesscmd.StartReviewStepOpts{
		OcteliumC: l.octeliumC,
		Request:   req,
		Index:     0,
	}))

	req, err = l.octeliumC.AccessC().UpdateRequest(l.ctx, req)
	require.NoError(t, err)
	return req
}

func (l *lifecycleTest) setApproved(t *testing.T, req *accessv1.Request) *accessv1.Request {
	req, err := l.octeliumC.AccessC().GetRequest(l.ctx, &rmetav1.GetOptions{Uid: req.Metadata.Uid})
	require.NoError(t, err)

	accesscmd.SetRequestState(req, &accesscmd.StateChange{
		Status: accessv1.Request_Status_State_APPROVED,
		Reason: accessv1.Request_Status_State_AUTO_APPROVED,
	})
	req.Status.AccessStartsAt = pbutils.Now()
	req.Status.AccessEndsAt = pbutils.Timestamp(time.Now().Add(time.Hour))
	req.Status.Grant = &accessv1.Request_Status_Grant{State: accessv1.Request_Status_Grant_ACTIVE}

	req, err = l.octeliumC.AccessC().UpdateRequest(l.ctx, req)
	require.NoError(t, err)

	pt, err := accesscmd.BuildPolicyTrigger(req)
	require.NoError(t, err)
	_, err = l.octeliumC.CoreC().CreatePolicyTrigger(l.ctx, pt)
	require.NoError(t, err)

	return req
}

func (l *lifecycleTest) policyTriggerExists(t *testing.T, req *accessv1.Request) bool {
	_, err := l.octeliumC.CoreC().GetPolicyTrigger(l.ctx, &rmetav1.GetOptions{
		Name: accesscmd.PolicyTriggerName(req),
	})
	if err != nil {
		require.True(t, grpcerr.IsNotFound(err), "%+v", err)
		return false
	}
	return true
}

func TestUserPreviewAndCreateRequest(t *testing.T) {
	l := newLifecycleTest(t)

	requester := l.newUser(t)
	reviewer := l.newUser(t)

	reviewed := l.newCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.Requirements = &accessv1.Catalog_Spec_Requirements{
			Justification: &accessv1.Catalog_Spec_Requirements_Justification{IsRequired: true},
		}
	})
	auto := l.newCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.AccessLevels[0].MaxDuration = &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}}
	})
	denied := l.newCatalog(t, nil)
	unmatched := l.newCatalog(t, nil)

	l.newPolicy(t,
		&accessv1.Policy_Spec_Rule{
			Name:      "review",
			Effect:    accessv1.Policy_Spec_Rule_REVIEW,
			Condition: catalogCondition(reviewed),
			Action:    reviewAction(userReviewerStep("lead", reviewer)),
		},
		&accessv1.Policy_Spec_Rule{
			Name:      "auto",
			Effect:    accessv1.Policy_Spec_Rule_AUTO_APPROVE,
			Condition: catalogCondition(auto),
		},
		&accessv1.Policy_Spec_Rule{
			Name:      "deny",
			Effect:    accessv1.Policy_Spec_Rule_DENY,
			Condition: catalogCondition(denied),
			Message:   "Ask the security team",
		},
	)

	preview := func(spec *accessv1.Request_Spec) *accessv1.RequestPreview {
		ret, err := l.userSrv.PreviewRequest(requester.Ctx(), &accessv1.Request{Spec: spec})
		require.NoError(t, err)
		return ret
	}

	assert.Equal(t, accessv1.RequestPreview_NO_MATCH, preview(catalogRequestSpec(umetav1.GetObjectReference(unmatched))).Outcome)

	deniedPreview := preview(catalogRequestSpec(umetav1.GetObjectReference(denied)))
	assert.Equal(t, accessv1.RequestPreview_DENIED, deniedPreview.Outcome)
	assert.Equal(t, "Ask the security team", deniedPreview.Message)

	autoPreview := preview(catalogRequestSpec(&metav1.ObjectReference{Name: auto.Metadata.Name}))
	assert.Equal(t, accessv1.RequestPreview_AUTO_APPROVED, autoPreview.Outcome)
	assert.Equal(t, 2*time.Hour, umetav1.ToDuration(autoPreview.EffectiveDuration).ToGo())
	assert.Equal(t, 2*time.Hour, umetav1.ToDuration(autoPreview.MaxDuration).ToGo())
	assert.Len(t, autoPreview.ResolvedScope.ServiceRefs, 1)
	assert.Nil(t, autoPreview.Request.Status.Rule)

	incomplete := preview(catalogRequestSpec(umetav1.GetObjectReference(reviewed)))
	assert.Equal(t, accessv1.RequestPreview_INCOMPLETE, incomplete.Outcome)
	require.Len(t, incomplete.Problems, 1)
	assert.Equal(t, "spec.justification", incomplete.Problems[0].Field)
	assert.True(t, incomplete.Requirements.Justification.IsRequired)

	spec := catalogRequestSpec(umetav1.GetObjectReference(reviewed))
	spec.Justification = "on-call investigation"
	reviewPreview := preview(spec)
	assert.Equal(t, accessv1.RequestPreview_REVIEW, reviewPreview.Outcome)
	require.Len(t, reviewPreview.Workflow.Steps, 1)
	assert.Equal(t, "lead", reviewPreview.Workflow.Steps[0].Name)
	assert.Empty(t, reviewPreview.ExistingRequests)

	notFound := preview(catalogRequestSpec(&metav1.ObjectReference{Name: "missing"}))
	assert.Equal(t, accessv1.RequestPreview_NOT_REQUESTABLE, notFound.Outcome)
	assert.NotEmpty(t, notFound.Message)

	svcName := reviewed.Spec.ResourceCollection.Service.Services[0]
	shorthand := preview(&accessv1.Request_Spec{
		Justification: "on-call investigation",
		Resource: &accessv1.Request_Spec_Resource{
			Type: &accessv1.Request_Spec_Resource_ServiceRef{
				ServiceRef: &metav1.ObjectReference{Name: svcName},
			},
		},
	})
	assert.Equal(t, accessv1.RequestPreview_REVIEW, shorthand.Outcome)
	assert.Equal(t, reviewed.Metadata.Uid, shorthand.Request.Spec.Resource.GetCatalog().CatalogRef.Uid)

	created, err := l.userSrv.CreateRequest(requester.Ctx(), &accessv1.Request{
		Metadata: &metav1.Metadata{},
		Spec: &accessv1.Request_Spec{
			Justification: "on-call investigation",
			Resource: &accessv1.Request_Spec_Resource{
				Type: &accessv1.Request_Spec_Resource_Catalog_{
					Catalog: &accessv1.Request_Spec_Resource_Catalog{
						CatalogRef: &metav1.ObjectReference{Name: reviewed.Metadata.Name},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, reviewed.Metadata.Uid, created.Spec.Resource.GetCatalog().CatalogRef.Uid)
	assert.Equal(t, "full", created.Spec.Resource.GetCatalog().AccessLevel)
	assert.Equal(t, requester.Usr.Metadata.Uid, created.Spec.Subject.GetUserRef().Uid)
	assert.Equal(t, uint32(1), created.Status.SpecRevision)
	assert.Equal(t, accessv1.Request_Status_State_SUBMITTED, created.Status.State.Reason)
	assert.Equal(t, requester.Usr.Metadata.Uid, created.Status.RequesterInfo.UserRef.Uid)
	assert.NotNil(t, created.Spec.Deadline)
	assert.Len(t, created.Status.ResolvedScope.ServiceRefs, 1)

	reviewPreview = preview(spec)
	require.Len(t, reviewPreview.ExistingRequests, 1)
	assert.Equal(t, created.Metadata.Uid, reviewPreview.ExistingRequests[0].RequestRef.Uid)

	_, err = l.userSrv.CreateRequest(requester.Ctx(), &accessv1.Request{Metadata: &metav1.Metadata{}, Spec: catalogRequestSpec(umetav1.GetObjectReference(reviewed))})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	hidden := newTestService(t, l.octeliumC, &metav1.Metadata{IsSystem: true, IsUserHidden: true})
	l.newCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.ResourceCollection.Service.Services = []string{hidden.Metadata.Name}
	})

	_, err = l.userSrv.CreateRequest(requester.Ctx(), &accessv1.Request{
		Metadata: &metav1.Metadata{},
		Spec: &accessv1.Request_Spec{
			Resource: &accessv1.Request_Spec_Resource{
				Type: &accessv1.Request_Spec_Resource_ServiceRef{
					ServiceRef: umetav1.GetObjectReference(hidden),
				},
			},
		},
	})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestUserRequestLifecycle(t *testing.T) {
	l := newLifecycleTest(t)

	requester := l.newUser(t)
	reviewer := l.newUser(t)
	other := l.newUser(t)
	catalog := l.newCatalog(t, nil)

	l.newPolicy(t, &accessv1.Policy_Spec_Rule{
		Name:      "review",
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: catalogCondition(catalog),
		Action:    reviewAction(userReviewerStep("lead", reviewer)),
	})

	req, err := l.userSrv.CreateRequest(requester.Ctx(), &accessv1.Request{
		Metadata: &metav1.Metadata{},
		Spec:     catalogRequestSpec(umetav1.GetObjectReference(catalog)),
	})
	require.NoError(t, err)

	update := func(fn func(spec *accessv1.Request_Spec)) *accessv1.Request {
		cur, err := l.userSrv.GetRequest(requester.Ctx(), &metav1.GetOptions{Uid: req.Metadata.Uid})
		require.NoError(t, err)
		fn(cur.Spec)
		ret, err := l.userSrv.UpdateRequest(requester.Ctx(), cur)
		require.NoError(t, err)
		return ret
	}

	assert.Equal(t, uint32(1), update(func(spec *accessv1.Request_Spec) {
		spec.Urgency = accessv1.Request_Spec_HIGH
	}).Status.SpecRevision)
	assert.Equal(t, uint32(1), update(func(spec *accessv1.Request_Spec) {
		spec.Duration = &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}}
	}).Status.SpecRevision)
	assert.Equal(t, uint32(2), update(func(spec *accessv1.Request_Spec) {
		spec.Duration = &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 3}}
	}).Status.SpecRevision)
	assert.Equal(t, uint32(3), update(func(spec *accessv1.Request_Spec) {
		spec.Justification = "now with a reason"
	}).Status.SpecRevision)

	req = l.setPendingReview(t, req, userReviewerStep("lead", reviewer))

	inbox, err := l.userSrv.GetInbox(reviewer.Ctx(), &accessv1.GetInboxRequest{})
	require.NoError(t, err)
	assert.Equal(t, uint32(1), inbox.TotalNeedsMyDecision)
	assert.True(t, inbox.IsReviewer)

	_, err = l.reviewerSrv.RequestInformation(other.Ctx(), &accessv1.RequestInformationRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Message:    "why?",
	})
	assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)

	asked, err := l.reviewerSrv.RequestInformation(reviewer.Ctx(), &accessv1.RequestInformationRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Message:    "Which tables?",
	})
	require.NoError(t, err)
	require.Len(t, asked.Status.Review.InformationRequests, 1)

	_, err = l.reviewerSrv.RequestInformation(reviewer.Ctx(), &accessv1.RequestInformationRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Message:    "And?",
	})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	inbox, err = l.userSrv.GetInbox(requester.Ctx(), &accessv1.GetInboxRequest{})
	require.NoError(t, err)
	assert.Equal(t, uint32(1), inbox.TotalMyPending)
	assert.Equal(t, uint32(1), inbox.TotalAwaitingMyResponse)
	assert.False(t, inbox.IsReviewer)

	responded, err := l.userSrv.RespondToRequest(requester.Ctx(), &accessv1.RespondToRequestRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Response:   "Only the invoices table",
	})
	require.NoError(t, err)
	assert.Equal(t, "Only the invoices table", responded.Status.Review.InformationRequests[0].Response)
	assert.NotNil(t, responded.Status.Review.InformationRequests[0].RespondedAt)

	_, err = l.userSrv.RespondToRequest(requester.Ctx(), &accessv1.RespondToRequestRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Response:   "again",
	})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	_, err = l.userSrv.EndAccess(requester.Ctx(), &accessv1.EndAccessRequest{RequestRef: umetav1.GetObjectReference(req)})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	req = l.setApproved(t, req)
	require.True(t, l.policyTriggerExists(t, req))

	inbox, err = l.userSrv.GetInbox(requester.Ctx(), &accessv1.GetInboxRequest{})
	require.NoError(t, err)
	assert.Equal(t, uint32(1), inbox.TotalActiveAccess)
	assert.Equal(t, uint32(1), inbox.TotalExpiringSoon)

	active, err := l.userSrv.ListRequest(requester.Ctx(), &accessv1.ListUserRequestOptions{IsActive: true})
	require.NoError(t, err)
	require.Len(t, active.Items, 1)

	_, err = l.userSrv.EndAccess(other.Ctx(), &accessv1.EndAccessRequest{RequestRef: umetav1.GetObjectReference(req)})
	assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)

	ended, err := l.userSrv.EndAccess(requester.Ctx(), &accessv1.EndAccessRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Reason:     "done early",
	})
	require.NoError(t, err)
	assert.Equal(t, accessv1.Request_Status_State_EXPIRED, ended.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_ENDED_EARLY, ended.Status.State.Reason)
	assert.Equal(t, "done early", ended.Status.State.Message)
	assert.Equal(t, requester.Usr.Metadata.Uid, ended.Status.State.ActorRef.Uid)
	assert.Equal(t, accessv1.Request_Status_Grant_ENDED, ended.Status.Grant.State)
	assert.False(t, l.policyTriggerExists(t, req))

	found, err := l.userSrv.ListRequest(requester.Ctx(), &accessv1.ListUserRequestOptions{Query: "invoices"})
	require.NoError(t, err)
	assert.Len(t, found.Items, 0)

	found, err = l.userSrv.ListRequest(requester.Ctx(), &accessv1.ListUserRequestOptions{Query: "WITH A REASON"})
	require.NoError(t, err)
	assert.Len(t, found.Items, 1)

	found, err = l.userSrv.ListRequest(requester.Ctx(), &accessv1.ListUserRequestOptions{
		ServiceRef: &metav1.ObjectReference{Name: catalog.Spec.ResourceCollection.Service.Services[0]},
	})
	require.NoError(t, err)
	assert.Len(t, found.Items, 1)
}

func TestReviewerCaseAndQueues(t *testing.T) {
	l := newLifecycleTest(t)

	requester := l.newUser(t)
	reviewerA := l.newUser(t)
	reviewerB := l.newUser(t)
	other := l.newUser(t)
	catalog := l.newCatalog(t, nil)

	step := userReviewerStep("lead", reviewerA, reviewerB)

	l.newPolicy(t, &accessv1.Policy_Spec_Rule{
		Name:      "review",
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: catalogCondition(catalog),
		Action:    reviewAction(step),
	})

	req, err := l.userSrv.CreateRequest(requester.Ctx(), &accessv1.Request{
		Metadata: &metav1.Metadata{},
		Spec:     catalogRequestSpec(umetav1.GetObjectReference(catalog)),
	})
	require.NoError(t, err)
	req = l.setPendingReview(t, req, step)

	list := func(usr *tstuser.User, filter accessv1.ListReviewerRequestOptions_Filter) []string {
		ret, err := l.reviewerSrv.ListRequest(usr.Ctx(), &accessv1.ListReviewerRequestOptions{Filter: filter})
		require.NoError(t, err)
		var names []string
		for _, itm := range ret.Items {
			assert.Nil(t, itm.Status.Rule)
			names = append(names, itm.Metadata.Name)
		}
		return names
	}

	assert.Equal(t, []string{req.Metadata.Name}, list(reviewerA, accessv1.ListReviewerRequestOptions_NEEDS_MY_DECISION))
	assert.Empty(t, list(reviewerA, accessv1.ListReviewerRequestOptions_WAITING_ON_OTHERS))
	assert.Empty(t, list(other, accessv1.ListReviewerRequestOptions_NEEDS_MY_DECISION))

	reviewCase, err := l.reviewerSrv.GetReviewCase(reviewerA.Ctx(), &accessv1.GetReviewCaseRequest{
		RequestRef: umetav1.GetObjectReference(req),
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []accessv1.ReviewCase_Action{
		accessv1.ReviewCase_APPROVE, accessv1.ReviewCase_REJECT, accessv1.ReviewCase_REQUEST_INFORMATION,
	}, reviewCase.AllowedActions)
	assert.True(t, reviewCase.Flags.IsFullAccess)
	assert.True(t, reviewCase.Flags.IsFirstRequest)
	assert.False(t, reviewCase.Flags.IsOnBehalf)
	assert.Nil(t, reviewCase.MyReview)
	assert.Nil(t, reviewCase.Request.Status.Rule)

	_, err = l.reviewerSrv.GetReviewCase(other.Ctx(), &accessv1.GetReviewCaseRequest{
		RequestRef: umetav1.GetObjectReference(req),
	})
	assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)

	_, err = l.reviewerSrv.SetReviewDecision(reviewerA.Ctx(), &accessv1.SetReviewDecisionRequest{
		RequestRef:           umetav1.GetObjectReference(req),
		Decision:             accessv1.Review_Spec_DECISION_APPROVE,
		ExpectedSpecRevision: 1,
	})
	require.NoError(t, err)

	assert.Empty(t, list(reviewerA, accessv1.ListReviewerRequestOptions_NEEDS_MY_DECISION))
	assert.Equal(t, []string{req.Metadata.Name}, list(reviewerA, accessv1.ListReviewerRequestOptions_WAITING_ON_OTHERS))
	assert.Equal(t, []string{req.Metadata.Name}, list(reviewerA, accessv1.ListReviewerRequestOptions_DECIDED_BY_ME))
	assert.Equal(t, []string{req.Metadata.Name}, list(reviewerB, accessv1.ListReviewerRequestOptions_NEEDS_MY_DECISION))

	reviewCase, err = l.reviewerSrv.GetReviewCase(reviewerA.Ctx(), &accessv1.GetReviewCaseRequest{
		RequestRef: umetav1.GetObjectReference(req),
	})
	require.NoError(t, err)
	require.NotNil(t, reviewCase.MyReview)
	assert.Contains(t, reviewCase.AllowedActions, accessv1.ReviewCase_WITHDRAW)
	assert.False(t, reviewCase.Flags.IsSpecChanged)
	assert.Len(t, reviewCase.Reviews, 1)

	cur, err := l.octeliumC.AccessC().GetRequest(l.ctx, &rmetav1.GetOptions{Uid: req.Metadata.Uid})
	require.NoError(t, err)
	cur.Status.SpecRevision = 2
	_, err = l.octeliumC.AccessC().UpdateRequest(l.ctx, cur)
	require.NoError(t, err)

	reviewCase, err = l.reviewerSrv.GetReviewCase(reviewerA.Ctx(), &accessv1.GetReviewCaseRequest{
		RequestRef: umetav1.GetObjectReference(req),
	})
	require.NoError(t, err)
	assert.True(t, reviewCase.Flags.IsSpecChanged)
	assert.Equal(t, []string{req.Metadata.Name}, list(reviewerA, accessv1.ListReviewerRequestOptions_NEEDS_MY_DECISION))

	_, err = l.reviewerSrv.SetReviewDecision(reviewerA.Ctx(), &accessv1.SetReviewDecisionRequest{
		RequestRef:           umetav1.GetObjectReference(req),
		Decision:             accessv1.Review_Spec_DECISION_APPROVE,
		ExpectedSpecRevision: 1,
	})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	l.setApproved(t, req)

	reqG, err := l.reviewerSrv.GetRequest(reviewerA.Ctx(), &metav1.GetOptions{Uid: req.Metadata.Uid})
	require.NoError(t, err)
	assert.Equal(t, accessv1.Request_Status_State_APPROVED, reqG.Status.State.Status)

	reqG, err = l.reviewerSrv.GetRequest(reviewerB.Ctx(), &metav1.GetOptions{Uid: req.Metadata.Uid})
	require.NoError(t, err)
	assert.Equal(t, req.Metadata.Uid, reqG.Metadata.Uid)

	_, err = l.reviewerSrv.GetRequest(other.Ctx(), &metav1.GetOptions{Uid: req.Metadata.Uid})
	assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)
}

func TestMainRevokeSimulateAndCatalogRules(t *testing.T) {
	l := newLifecycleTest(t)

	requester := l.newUser(t)
	reviewer := l.newUser(t)
	catalog := l.newCatalog(t, nil)

	pol := l.newPolicy(t, &accessv1.Policy_Spec_Rule{
		Name:      "review",
		Effect:    accessv1.Policy_Spec_Rule_REVIEW,
		Condition: catalogCondition(catalog),
		Action:    reviewAction(userReviewerStep("lead", reviewer)),
	})

	_, err := l.octeliumC.AccessC().CreatePolicy(l.ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{Name: "a-broken"},
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Name:     "broken",
					Priority: -1,
					Effect:   accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_Match{Match: "ctx.request.("},
					},
				},
			},
		},
		Status: &accessv1.Policy_Status{},
	})
	require.NoError(t, err)

	sim, err := l.mainSrv.SimulateRequest(l.ctx, &accessv1.SimulateRequestRequest{
		Spec: &accessv1.Request_Spec{
			Subject: &accessv1.Request_Spec_Subject{
				Type: &accessv1.Request_Spec_Subject_UserRef{UserRef: umetav1.GetObjectReference(requester.Usr)},
			},
			Resource: catalogRequestSpec(umetav1.GetObjectReference(catalog)).Resource,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, accessv1.RequestPreview_REVIEW, sim.Preview.Outcome)
	assert.Equal(t, "review", sim.Rule.Name)
	assert.Equal(t, pol.Metadata.Uid, sim.PolicyRef.Uid)
	require.Len(t, sim.Rules, 2)
	assert.Equal(t, "broken", sim.Rules[0].RuleName)
	assert.Equal(t, accessv1.SimulateRequestResponse_RuleEvaluation_ERROR, sim.Rules[0].Result)
	assert.NotEmpty(t, sim.Rules[0].Error)
	assert.Equal(t, accessv1.SimulateRequestResponse_RuleEvaluation_MATCHED, sim.Rules[1].Result)
	require.Len(t, sim.Reviewers, 1)
	assert.Equal(t, reviewer.Usr.Metadata.Uid, sim.Reviewers[0].EligibleReviewers[0].Uid)
	require.NotNil(t, sim.PolicyTrigger)
	assert.Equal(t, requester.Usr.Metadata.Uid, sim.PolicyTrigger.Status.PreCondition.GetAll().Of[0].GetUserRef().Uid)

	req, err := l.userSrv.CreateRequest(requester.Ctx(), &accessv1.Request{
		Metadata: &metav1.Metadata{},
		Spec:     catalogRequestSpec(umetav1.GetObjectReference(catalog)),
	})
	require.NoError(t, err)
	req = l.setApproved(t, req)

	_, err = l.mainSrv.RevokeRequest(l.ctx, &accessv1.RevokeRequestRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Reason:     strings.Repeat("x", accesscmd.MaxJustificationLen+1),
	})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	_, err = l.mainSrv.RevokeRequest(l.ctx, &accessv1.RevokeRequestRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Reason:     "incident closed",
	})
	require.NoError(t, err)

	revoked, err := l.mainSrv.GetRequest(l.ctx, &metav1.GetOptions{Uid: req.Metadata.Uid})
	require.NoError(t, err)
	assert.Equal(t, accessv1.Request_Status_State_REVOKED, revoked.Status.State.Status)
	assert.Equal(t, accessv1.Request_Status_State_ACCESS_REVOKED, revoked.Status.State.Reason)
	assert.Equal(t, "incident closed", revoked.Status.State.Message)
	assert.False(t, l.policyTriggerExists(t, req))

	_, err = l.mainSrv.DeleteCatalog(l.ctx, &metav1.DeleteOptions{Uid: catalog.Metadata.Uid})
	assert.ErrorContains(t, err, pol.Metadata.Name)

	_, err = l.mainSrv.DeletePolicy(l.ctx, &metav1.DeleteOptions{Uid: pol.Metadata.Uid})
	require.NoError(t, err)

	_, err = l.mainSrv.DeleteCatalog(l.ctx, &metav1.DeleteOptions{Uid: catalog.Metadata.Uid})
	require.NoError(t, err)
}

func TestMainCatalogValidation(t *testing.T) {
	l := newLifecycleTest(t)

	priority := func(v int32) *int32 { return &v }
	hours := func(h uint32) *metav1.Duration { return &metav1.Duration{Type: &metav1.Duration_Hours{Hours: h}} }

	valid := func() *accessv1.Catalog_Spec {
		return &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services: []string{"svc"},
				},
			},
			AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{testFullLevel("full")},
		}
	}

	restricted := func(pol *corev1.InlinePolicy, names ...string) *accessv1.Catalog_Spec_AccessLevel {
		ret := &accessv1.Catalog_Spec_AccessLevel{
			Name: "read",
			Type: &accessv1.Catalog_Spec_AccessLevel_Restricted_{
				Restricted: &accessv1.Catalog_Spec_AccessLevel_Restricted{Policies: names},
			},
		}
		if pol != nil {
			ret.GetRestricted().InlinePolicies = []*corev1.InlinePolicy{pol}
		}
		return ret
	}

	inline := func(match string, prio int32) *corev1.InlinePolicy {
		return &corev1.InlinePolicy{
			Spec: &corev1.Policy_Spec{
				Rules: []*corev1.Policy_Spec_Rule{
					{
						Effect:   corev1.Policy_Spec_Rule_ALLOW,
						Priority: prio,
						Condition: &corev1.Condition{
							Type: &corev1.Condition_Match{Match: match},
						},
					},
				},
			},
		}
	}

	for _, tc := range []struct {
		name   string
		mutate func(spec *accessv1.Catalog_Spec)
		valid  bool
	}{
		{"valid", func(spec *accessv1.Catalog_Spec) {}, true},
		{"no access levels", func(spec *accessv1.Catalog_Spec) { spec.AccessLevels = nil }, false},
		{"duplicate levels", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels = append(spec.AccessLevels, testFullLevel("full"))
		}, false},
		{"invalid level name", func(spec *accessv1.Catalog_Spec) { spec.AccessLevels[0].Name = "Not Valid" }, false},
		{"no level type", func(spec *accessv1.Catalog_Spec) { spec.AccessLevels[0].Type = nil }, false},
		{"priority out of range", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels[0].GetFull().Priority = priority(-5)
		}, false},
		{"negative priority", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels[0].GetFull().Priority = priority(-3)
		}, true},
		{"default above max", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels[0].DefaultDuration = hours(4)
			spec.AccessLevels[0].MaxDuration = hours(2)
		}, false},
		{"empty restricted", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels = []*accessv1.Catalog_Spec_AccessLevel{restricted(nil)}
		}, false},
		{"missing core policy", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels = []*accessv1.Catalog_Spec_AccessLevel{restricted(nil, "missing")}
		}, false},
		{"restricted inline", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels = []*accessv1.Catalog_Spec_AccessLevel{restricted(inline(`ctx.request.http.method == "GET"`, 0))}
		}, true},
		{"invalid inline expression", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels = []*accessv1.Catalog_Spec_AccessLevel{restricted(inline("ctx.request.(", 0))}
		}, false},
		{"inline priority out of range", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels = []*accessv1.Catalog_Spec_AccessLevel{restricted(inline("true", 9))}
		}, false},
		{"invalid audience expression", func(spec *accessv1.Catalog_Spec) {
			spec.Audience = &accessv1.Catalog_Spec_Audience{Match: "ctx.user.("}
		}, false},
		{"invalid audience user", func(spec *accessv1.Catalog_Spec) {
			spec.Audience = &accessv1.Catalog_Spec_Audience{Users: []string{"Not Valid"}}
		}, false},
		{"invalid owner group", func(spec *accessv1.Catalog_Spec) {
			spec.Owners = &accessv1.Catalog_Spec_Owners{Groups: []string{"Not Valid"}}
		}, false},
		{"invalid ticket pattern", func(spec *accessv1.Catalog_Spec) {
			spec.Requirements = &accessv1.Catalog_Spec_Requirements{
				Ticket: &accessv1.Catalog_Spec_Requirements_Ticket{Pattern: "("},
			}
		}, false},
		{"minimum length too long", func(spec *accessv1.Catalog_Spec) {
			spec.Requirements = &accessv1.Catalog_Spec_Requirements{
				Justification: &accessv1.Catalog_Spec_Requirements_Justification{MinLength: accesscmd.MaxJustificationLen + 1},
			}
		}, false},
		{"invalid membership", func(spec *accessv1.Catalog_Spec) { spec.Membership = 9 }, false},
		{"invalid min aal", func(spec *accessv1.Catalog_Spec) {
			spec.AccessLevels[0].Verification = &accessv1.Catalog_Spec_AccessLevel_Verification{MinAAL: 9}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := valid()
			tc.mutate(spec)
			_, err := l.mainSrv.CreateCatalog(l.ctx, &accessv1.Catalog{
				Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
				Spec:     spec,
			})
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
			}
		})
	}
}

func TestMainClusterConfigReviewAndRequestSettings(t *testing.T) {
	l := newLifecycleTest(t)

	cc, err := l.mainSrv.GetClusterConfig(l.ctx, &accessv1.GetClusterConfigRequest{})
	require.NoError(t, err)

	reviewer := l.newUser(t)

	cc.Spec.Request = &accessv1.ClusterConfig_Spec_Request{
		DefaultDeadline: &metav1.Duration{Type: &metav1.Duration_Days{Days: 3}},
		NoMatchMessage:  "Ask #access",
	}
	cc.Spec.Review = &accessv1.ClusterConfig_Spec_Review{
		FallbackReviewers: userReviewerStep("fallback", reviewer).Reviewers,
	}

	ret, err := l.mainSrv.UpdateClusterConfig(l.ctx, cc)
	require.NoError(t, err)
	assert.Equal(t, "Ask #access", ret.Spec.Request.NoMatchMessage)

	cc.Spec.Request.NoMatchMessage = strings.Repeat("x", accesscmd.MaxMessageLen+1)
	_, err = l.mainSrv.UpdateClusterConfig(l.ctx, cc)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	cc.Spec.Request.NoMatchMessage = ""
	cc.Spec.Review.FallbackReviewers = append(cc.Spec.Review.FallbackReviewers,
		&accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners_{
				CatalogOwners: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners{},
			},
		})
	_, err = l.mainSrv.UpdateClusterConfig(l.ctx, cc)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}
