// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAccessSummaries(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC()
	userRef := &metav1.ObjectReference{Name: "user-one", Uid: vutils.UUIDv4()}
	groupRef := &metav1.ObjectReference{Name: "group-one", Uid: vutils.UUIDv4()}
	serviceRef := &metav1.ObjectReference{Name: "svc-one", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &accessv1.Policy{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindPolicy,
		Metadata:   newRscStoreMetadata("disabled-policy", now),
		Spec: &accessv1.Policy_Spec{
			IsDisabled: true,
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Effect: accessv1.Policy_Spec_Rule_DENY,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{
							Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{
								Type: &accessv1.Policy_Spec_Rule_Condition_Subject_UserRef{UserRef: userRef},
							},
						},
					},
				},
				{
					Effect: accessv1.Policy_Spec_Rule_REVIEW,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{
							Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{
								Type: &accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef{GroupRef: groupRef},
							},
						},
					},
					Action: &accessv1.Policy_Spec_Rule_Action{
						Type: &accessv1.Policy_Spec_Rule_Action_Review_{
							Review: &accessv1.Policy_Spec_Rule_Action_Review{
								Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
									{
										Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
											{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
												User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{UserRef: userRef},
											}},
											{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_{
												Group: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group{GroupRef: groupRef},
											}},
										},
									},
								},
							},
						},
					},
				},
				{
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{
							Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{
								Type: &accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef{ServiceRef: serviceRef},
							},
						},
					},
					Authorization: &accessv1.Policy_Spec_Rule_Authorization{
						Policies:          []string{"p1"},
						MaxAccessDuration: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 3600}},
					},
				},
			},
		},
		Status: &accessv1.Policy_Status{},
	})

	{
		resp, err := env.srv.getSummaryAccessPolicy(env.ctx, &vaccessv1.GetPolicySummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalDisabled)
		assert.EqualValues(t, 3, resp.TotalRule)
		assert.EqualValues(t, 1, resp.TotalRuleDeny)
		assert.EqualValues(t, 1, resp.TotalRuleReview)
		assert.EqualValues(t, 1, resp.TotalRuleAutoApprove)
		assert.EqualValues(t, 1, resp.TotalRuleAuthorization)
		assert.EqualValues(t, 1, resp.TotalRuleMaxAccessDuration)
		assert.EqualValues(t, 1, resp.TotalReviewStep)
		assert.EqualValues(t, 2, resp.TotalReviewer)
	}

	insertRscStoreObject(t, env, &accessv1.Catalog{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindCatalog,
		Metadata:   newRscStoreMetadata("catalog-one", now),
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services:   []string{"svc-a", "svc-b"},
					Namespaces: []string{"ns-a"},
				},
			},
		},
		Status: &accessv1.Catalog_Status{},
	})

	{
		resp, err := env.srv.getSummaryAccessCatalog(env.ctx, &vaccessv1.GetCatalogSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
		assert.EqualValues(t, 2, resp.TotalService)
		assert.EqualValues(t, 1, resp.TotalNamespace)
	}

	insertRscStoreObject(t, env, &accessv1.Request{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindRequest,
		Metadata:   newRscStoreMetadata("active-request", now),
		Spec: &accessv1.Request_Spec{
			Urgency:  accessv1.Request_Spec_HIGH,
			Deadline: timestamppb.New(now.Add(-time.Hour)),
		},
		Status: &accessv1.Request_Status{
			State:        &accessv1.Request_Status_State{Status: accessv1.Request_Status_State_APPROVED},
			UserRef:      userRef,
			AccessEndsAt: timestamppb.New(now.Add(24 * time.Hour)),
		},
	})
	insertRscStoreObject(t, env, &accessv1.Request{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindRequest,
		Metadata:   newRscStoreMetadata("pending-request", now.Add(time.Second)),
		Spec:       &accessv1.Request_Spec{Urgency: accessv1.Request_Spec_LOW},
		Status: &accessv1.Request_Status{
			State: &accessv1.Request_Status_State{Status: accessv1.Request_Status_State_PENDING},
		},
	})

	{
		resp, err := env.srv.getSummaryAccessRequest(env.ctx, &vaccessv1.GetRequestSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalPending)
		assert.EqualValues(t, 1, resp.TotalApproved)
		assert.EqualValues(t, 1, resp.TotalActive)
		assert.EqualValues(t, 1, resp.TotalUrgencyHigh)
		assert.EqualValues(t, 1, resp.TotalUrgencyLow)
		assert.EqualValues(t, 1, resp.TotalWithDeadline)
		assert.EqualValues(t, 1, resp.TotalDeadlinePassed)
	}

	requestRef := &metav1.ObjectReference{Name: "active-request", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &accessv1.Review{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindReview,
		Metadata:   newRscStoreMetadata("approved-review", now),
		Spec:       &accessv1.Review_Spec{Decision: accessv1.Review_Spec_DECISION_APPROVE},
		Status: &accessv1.Review_Status{
			UserRef:       userRef,
			RequestRef:    requestRef,
			LastRevisions: []*accessv1.Review_Status_Revision{{SetAt: timestamppb.New(now)}},
		},
	})
	insertRscStoreObject(t, env, &accessv1.Review{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindReview,
		Metadata:   newRscStoreMetadata("pending-review", now.Add(time.Second)),
		Spec:       &accessv1.Review_Spec{},
		Status: &accessv1.Review_Status{
			UserRef:    userRef,
			RequestRef: requestRef,
		},
	})

	{
		resp, err := env.srv.getSummaryAccessReview(env.ctx, &vaccessv1.GetReviewSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalApproved)
		assert.EqualValues(t, 1, resp.TotalPending)
		assert.EqualValues(t, 1, resp.TotalRevised)
		assert.EqualValues(t, 1, resp.TotalUser)
		assert.EqualValues(t, 1, resp.TotalRequest)
	}
}
