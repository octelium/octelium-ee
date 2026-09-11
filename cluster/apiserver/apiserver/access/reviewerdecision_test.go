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
	"testing"

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
)

type decisionTest struct {
	ctx       context.Context
	srv       *ServerReviewer
	octeliumC octeliumc.ClientInterface

	adminSrv *admin.Server
	usrSrv   *user.Server
}

func newDecisionTest(t *testing.T) *decisionTest {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	return &decisionTest{
		ctx:       ctx,
		srv:       NewServerReviewer(tst.C.OcteliumC),
		octeliumC: tst.C.OcteliumC,
		adminSrv: admin.NewServer(&admin.Opts{
			OcteliumC:  tst.C.OcteliumC,
			IsEmbedded: true,
		}),
		usrSrv: user.NewServer(tst.C.OcteliumC),
	}
}

func (d *decisionTest) newUser(t *testing.T) *tstuser.User {
	usr, err := tstuser.NewUser(d.octeliumC, d.adminSrv, d.usrSrv, nil)
	assert.Nil(t, err, "%+v", err)

	return usr
}

func tstReviewStep(name string,
	sod *accessv1.Policy_Spec_Rule_SeparationOfDuties,
	reviewerRefs ...*metav1.ObjectReference) *accessv1.Policy_Spec_Rule_Action_Review_Step {
	ret := &accessv1.Policy_Spec_Rule_Action_Review_Step{
		Name:                name,
		ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
		SeparationOfDuties:  sod,
	}

	for _, ref := range reviewerRefs {
		ret.Reviewers = append(ret.Reviewers,
			&accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
				Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
					User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
						UserRef: ref,
					},
				},
			})
	}

	return ret
}

func (d *decisionTest) newRequest(t *testing.T, requesterRef *metav1.ObjectReference,
	steps ...*accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Request {
	svc, err := d.octeliumC.CoreC().CreateService(d.ctx, &corev1.Service{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   &corev1.Service_Spec{},
		Status: &corev1.Service_Status{},
	})
	assert.Nil(t, err, "%+v", err)

	req, err := d.octeliumC.AccessC().CreateRequest(d.ctx, &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_NORMAL,
			Resource: &accessv1.Request_Spec_Resource{
				Type: &accessv1.Request_Spec_Resource_ServiceRef{
					ServiceRef: umetav1.GetObjectReference(svc),
				},
			},
			Subject: &accessv1.Request_Spec_Subject{
				Type: &accessv1.Request_Spec_Subject_UserRef{
					UserRef: requesterRef,
				},
			},
		},
		Status: &accessv1.Request_Status{
			UserRef: requesterRef,
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_PENDING,
			},
			Rule: &accessv1.Policy_Spec_Rule{
				Name:   utilrand.GetRandomStringCanonical(6),
				Effect: accessv1.Policy_Spec_Rule_REVIEW,
				Action: &accessv1.Policy_Spec_Rule_Action{
					Type: &accessv1.Policy_Spec_Rule_Action_Review_{
						Review: &accessv1.Policy_Spec_Rule_Action_Review{
							Steps: steps,
						},
					},
				},
			},
			Review: &accessv1.Request_Status_Review{
				CurrentStep: 0,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return req
}

func (d *decisionTest) gotoStep(t *testing.T, req *accessv1.Request, stepIndex int32) *accessv1.Request {
	item, err := d.octeliumC.AccessC().GetRequest(d.ctx, &rmetav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	item.Status.Review.CurrentStep = stepIndex

	item, err = d.octeliumC.AccessC().UpdateRequest(d.ctx, item)
	assert.Nil(t, err, "%+v", err)

	return item
}

func TestSetReviewDecision(t *testing.T) {
	d := newDecisionTest(t)

	reviewer := d.newUser(t)
	requester := d.newUser(t)

	req := d.newRequest(t, umetav1.GetObjectReference(requester.Usr),
		tstReviewStep("first", nil, umetav1.GetObjectReference(reviewer.Usr)))

	review, err := d.srv.SetReviewDecision(reviewer.Ctx(), &accessv1.SetReviewDecisionRequest{
		RequestRef:    umetav1.GetObjectReference(req),
		Decision:      accessv1.Review_Spec_DECISION_APPROVE,
		Justification: utilrand.GetRandomString(16),
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, reviewer.Usr.Metadata.Uid, review.Status.UserRef.Uid)
	assert.Equal(t, "first", review.Status.StepName)
	assert.Equal(t, int32(0), review.Status.StepIndex)
	assert.Equal(t, accessv1.Origin_API, review.Status.Origin.Type)

	{
		ret, err := d.srv.SetReviewDecision(reviewer.Ctx(), &accessv1.SetReviewDecisionRequest{
			RequestRef:    umetav1.GetObjectReference(req),
			Decision:      accessv1.Review_Spec_DECISION_APPROVE,
			Justification: review.Spec.Justification,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, review.Metadata.Uid, ret.Metadata.Uid)
		assert.Equal(t, 0, len(ret.Status.LastRevisions))
	}

	{
		ret, err := d.srv.SetReviewDecision(reviewer.Ctx(), &accessv1.SetReviewDecisionRequest{
			RequestRef: umetav1.GetObjectReference(req),
			Decision:   accessv1.Review_Spec_DECISION_REJECT,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, review.Metadata.Uid, ret.Metadata.Uid)
		assert.Equal(t, accessv1.Review_Spec_DECISION_REJECT, ret.Spec.Decision)
		assert.Equal(t, 1, len(ret.Status.LastRevisions))
		assert.Equal(t, accessv1.Review_Spec_DECISION_APPROVE, ret.Status.LastRevisions[0].Spec.Decision)
	}

	{
		ret, err := d.srv.SetReviewDecision(reviewer.Ctx(), &accessv1.SetReviewDecisionRequest{
			RequestRef: umetav1.GetObjectReference(req),
			Decision:   accessv1.Review_Spec_DECISION_UNSET,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, accessv1.Review_Spec_DECISION_UNSET, ret.Spec.Decision)
	}

	{
		_, err := d.srv.SetReviewDecision(reviewer.Ctx(), &accessv1.SetReviewDecisionRequest{
			RequestRef:       umetav1.GetObjectReference(req),
			Decision:         accessv1.Review_Spec_DECISION_APPROVE,
			ExpectedStepName: "second",
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		other := d.newUser(t)
		_, err := d.srv.SetReviewDecision(other.Ctx(), &accessv1.SetReviewDecisionRequest{
			RequestRef: umetav1.GetObjectReference(req),
			Decision:   accessv1.Review_Spec_DECISION_APPROVE,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)
	}
}

func TestSetReviewDecisionSeparationOfDuties(t *testing.T) {
	d := newDecisionTest(t)

	requester := d.newUser(t)
	requesterRef := umetav1.GetObjectReference(requester.Usr)

	{
		req := d.newRequest(t, requesterRef, tstReviewStep("first", nil, requesterRef))

		_, err := d.srv.SetReviewDecision(requester.Ctx(), &accessv1.SetReviewDecisionRequest{
			RequestRef: umetav1.GetObjectReference(req),
			Decision:   accessv1.Review_Spec_DECISION_APPROVE,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)
	}

	{
		req := d.newRequest(t, requesterRef, tstReviewStep("first",
			&accessv1.Policy_Spec_Rule_SeparationOfDuties{
				AllowRequesterReview: true,
				AllowSubjectReview:   true,
			}, requesterRef))

		review, err := d.srv.SetReviewDecision(requester.Ctx(), &accessv1.SetReviewDecisionRequest{
			RequestRef: umetav1.GetObjectReference(req),
			Decision:   accessv1.Review_Spec_DECISION_APPROVE,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, requester.Usr.Metadata.Uid, review.Status.UserRef.Uid)
	}
}

func TestSetReviewDecisionMultipleSteps(t *testing.T) {
	d := newDecisionTest(t)

	reviewer := d.newUser(t)
	requester := d.newUser(t)
	reviewerRef := umetav1.GetObjectReference(reviewer.Usr)

	req := d.newRequest(t, umetav1.GetObjectReference(requester.Usr),
		tstReviewStep("first", nil, reviewerRef),
		tstReviewStep("second", nil, reviewerRef))

	first, err := d.srv.SetReviewDecision(reviewer.Ctx(), &accessv1.SetReviewDecisionRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Decision:   accessv1.Review_Spec_DECISION_APPROVE,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, "first", first.Status.StepName)

	d.gotoStep(t, req, 1)

	second, err := d.srv.SetReviewDecision(reviewer.Ctx(), &accessv1.SetReviewDecisionRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Decision:   accessv1.Review_Spec_DECISION_APPROVE,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, "second", second.Status.StepName)
	assert.Equal(t, int32(1), second.Status.StepIndex)
	assert.NotEqual(t, first.Metadata.Uid, second.Metadata.Uid)
}

func TestSetReviewDecisionDisabledReviewer(t *testing.T) {
	d := newDecisionTest(t)

	reviewer := d.newUser(t)
	requester := d.newUser(t)

	req := d.newRequest(t, umetav1.GetObjectReference(requester.Usr),
		tstReviewStep("first", nil, umetav1.GetObjectReference(reviewer.Usr)))

	usr, err := d.octeliumC.CoreC().GetUser(d.ctx, &rmetav1.GetOptions{
		Uid: reviewer.Usr.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	usr.Spec.IsDisabled = true
	_, err = d.octeliumC.CoreC().UpdateUser(d.ctx, usr)
	assert.Nil(t, err, "%+v", err)

	reviewer.Resync()

	_, err = d.srv.SetReviewDecision(reviewer.Ctx(), &accessv1.SetReviewDecisionRequest{
		RequestRef: umetav1.GetObjectReference(req),
		Decision:   accessv1.Review_Spec_DECISION_APPROVE,
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)
}
