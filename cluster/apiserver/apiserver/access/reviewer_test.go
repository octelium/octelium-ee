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

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/user"
	"github.com/octelium/octelium/cluster/common/tests/tstuser"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func newReviewerTest(t *testing.T) (*ServerReviewer, *tstuser.User, *tstuser.User, *accessv1.Request) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	adminSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  tst.C.OcteliumC,
		IsEmbedded: true,
	})
	usrSrv := user.NewServer(tst.C.OcteliumC)

	mainSrv := NewServerMain(tst.C.OcteliumC)
	srv := NewServerReviewer(tst.C.OcteliumC)

	reviewer, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)
	requester, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)
	other, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

	cat, err := mainSrv.CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services: []string{utilrand.GetRandomStringCanonical(6)},
				},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	rule := &accessv1.Policy_Spec_Rule{
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
					Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
						{
							Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
								{
									Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
										User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
											UserRef: umetav1.GetObjectReference(reviewer.Usr),
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
	}

	req, err := tst.C.OcteliumC.AccessC().CreateRequest(ctx, &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_NORMAL,
			Resource: &accessv1.Request_Spec_Resource{
				Type: &accessv1.Request_Spec_Resource_Catalog_{
					Catalog: &accessv1.Request_Spec_Resource_Catalog{
						CatalogRef: umetav1.GetObjectReference(cat),
					},
				},
			},
			Subject: &accessv1.Request_Spec_Subject{
				Type: &accessv1.Request_Spec_Subject_UserRef{
					UserRef: umetav1.GetObjectReference(requester.Usr),
				},
			},
		},
		Status: &accessv1.Request_Status{
			UserRef: umetav1.GetObjectReference(requester.Usr),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_PENDING,
			},
			Rule: rule,
			Review: &accessv1.Request_Status_Review{
				CurrentStep: 0,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return srv, reviewer, other, req
}

func TestReviewerRequest(t *testing.T) {
	srv, reviewer, other, req := newReviewerTest(t)

	reqG, err := srv.GetRequest(reviewer.Ctx(), &metav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, req.Metadata.Uid, reqG.Metadata.Uid)

	reqList, err := srv.ListRequest(reviewer.Ctx(), &accessv1.ListReviewerRequestOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(reqList.Items))
	assert.Equal(t, req.Metadata.Uid, reqList.Items[0].Metadata.Uid)

	_, err = srv.GetRequest(other.Ctx(), &metav1.GetOptions{
		Uid: req.Metadata.Uid,
	})
	assert.NotNil(t, err)

	reqListOther, err := srv.ListRequest(other.Ctx(), &accessv1.ListReviewerRequestOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(reqListOther.Items))
}

func TestReviewerReview(t *testing.T) {
	srv, reviewer, other, req := newReviewerTest(t)

	_, err := srv.CreateReview(other.Ctx(), &accessv1.Review{
		Metadata: &metav1.Metadata{},
		Spec: &accessv1.Review_Spec{
			Decision: accessv1.Review_Spec_DECISION_APPROVE,
		},
		Status: &accessv1.Review_Status{
			RequestRef: umetav1.GetObjectReference(req),
		},
	})
	assert.NotNil(t, err)

	review, err := srv.CreateReview(reviewer.Ctx(), &accessv1.Review{
		Metadata: &metav1.Metadata{},
		Spec: &accessv1.Review_Spec{
			Decision:      accessv1.Review_Spec_DECISION_APPROVE,
			Justification: utilrand.GetRandomString(16),
		},
		Status: &accessv1.Review_Status{
			RequestRef: umetav1.GetObjectReference(req),
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, reviewer.Usr.Metadata.Uid, review.Status.UserRef.Uid)
	assert.Equal(t, req.Metadata.Uid, review.Status.RequestRef.Uid)
	assert.NotNil(t, review.Status.SetAt)

	_, err = srv.CreateReview(reviewer.Ctx(), &accessv1.Review{
		Spec: &accessv1.Review_Spec{
			Decision: accessv1.Review_Spec_DECISION_APPROVE,
		},
		Status: &accessv1.Review_Status{
			RequestRef: umetav1.GetObjectReference(req),
		},
	})
	assert.NotNil(t, err)

	reviewG, err := srv.GetReview(reviewer.Ctx(), &metav1.GetOptions{
		Uid: review.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, review.Metadata.Uid, reviewG.Metadata.Uid)

	_, err = srv.GetReview(other.Ctx(), &metav1.GetOptions{
		Uid: review.Metadata.Uid,
	})
	assert.NotNil(t, err)

	reviewList, err := srv.ListReview(reviewer.Ctx(), &accessv1.ListReviewerReviewOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(reviewList.Items))
	assert.Equal(t, review.Metadata.Uid, reviewList.Items[0].Metadata.Uid)

	reviewListOther, err := srv.ListReview(other.Ctx(), &accessv1.ListReviewerReviewOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(reviewListOther.Items))

	review.Spec.Decision = accessv1.Review_Spec_DECISION_REJECT
	review.Spec.Justification = utilrand.GetRandomString(20)
	reviewU, err := srv.UpdateReview(reviewer.Ctx(), review)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, accessv1.Review_Spec_DECISION_REJECT, reviewU.Spec.Decision)

	_, err = srv.CancelReview(reviewer.Ctx(), &accessv1.CancelReviewRequest{
		ReviewRef: umetav1.GetObjectReference(reviewU),
	})
	assert.Nil(t, err, "%+v", err)

	reviewC, err := srv.GetReview(reviewer.Ctx(), &metav1.GetOptions{
		Uid: review.Metadata.Uid,
	})
	assert.Nil(t, err)
	assert.Equal(t, accessv1.Review_Spec_DECISION_UNSET, reviewC.Spec.Decision)
}
