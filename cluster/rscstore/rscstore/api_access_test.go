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

func TestAccessListFilters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvAccess{s: env.srv}
	now := time.Now().UTC()

	userRef := &metav1.ObjectReference{Name: "user-one", Uid: vutils.UUIDv4()}
	otherUserRef := &metav1.ObjectReference{Name: "user-two", Uid: vutils.UUIDv4()}
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
			},
		},
		Status: &accessv1.Policy_Status{},
	})
	insertRscStoreObject(t, env, &accessv1.Policy{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindPolicy,
		Metadata:   newRscStoreMetadata("service-policy", now.Add(time.Second)),
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Condition: &accessv1.Policy_Spec_Rule_Condition{
						Type: &accessv1.Policy_Spec_Rule_Condition_Resource_{
							Resource: &accessv1.Policy_Spec_Rule_Condition_Resource{
								Type: &accessv1.Policy_Spec_Rule_Condition_Resource_ServiceRef{ServiceRef: serviceRef},
							},
						},
					},
				},
			},
		},
		Status: &accessv1.Policy_Status{},
	})

	{
		resp, err := srv.ListPolicy(env.ctx, &vaccessv1.ListPolicyOptions{IsDisabled: true})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "disabled-policy", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListPolicy(env.ctx, &vaccessv1.ListPolicyOptions{Effect: accessv1.Policy_Spec_Rule_DENY})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "disabled-policy", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListPolicy(env.ctx, &vaccessv1.ListPolicyOptions{UserRef: userRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "disabled-policy", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListPolicy(env.ctx, &vaccessv1.ListPolicyOptions{ServiceRef: serviceRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "service-policy", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListPolicy(env.ctx, &vaccessv1.ListPolicyOptions{GroupRef: groupRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 0)
	}

	insertRscStoreObject(t, env, &accessv1.Catalog{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindCatalog,
		Metadata:   newRscStoreMetadata("catalog-one", now),
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services:   []string{"svc-alpha"},
					Namespaces: []string{"ns-alpha"},
				},
			},
		},
		Status: &accessv1.Catalog_Status{},
	})
	insertRscStoreObject(t, env, &accessv1.Catalog{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindCatalog,
		Metadata:   newRscStoreMetadata("catalog-two", now.Add(time.Second)),
		Spec:       &accessv1.Catalog_Spec{},
		Status:     &accessv1.Catalog_Status{},
	})

	{
		resp, err := srv.ListCatalog(env.ctx, &vaccessv1.ListCatalogOptions{
			ServiceRef: &metav1.ObjectReference{Name: "svc-alpha"},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "catalog-one", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListCatalog(env.ctx, &vaccessv1.ListCatalogOptions{
			NamespaceRef: &metav1.ObjectReference{Name: "ns-alpha"},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "catalog-one", resp.Items[0].Metadata.Name)
	}

	policyRef := &metav1.ObjectReference{Name: "policy-one", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &accessv1.Request{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindRequest,
		Metadata:   newRscStoreMetadata("active-request", now),
		Spec: &accessv1.Request_Spec{
			Urgency: accessv1.Request_Spec_HIGH,
			Subject: &accessv1.Request_Spec_Subject{
				Type: &accessv1.Request_Spec_Subject_UserRef{UserRef: otherUserRef},
			},
			Resource: &accessv1.Request_Spec_Resource{
				Type: &accessv1.Request_Spec_Resource_ServiceRef{ServiceRef: serviceRef},
			},
		},
		Status: &accessv1.Request_Status{
			State:        &accessv1.Request_Status_State{Status: accessv1.Request_Status_State_APPROVED},
			UserRef:      userRef,
			PolicyRef:    policyRef,
			AccessEndsAt: timestamppb.New(now.Add(24 * time.Hour)),
			Rule: &accessv1.Policy_Spec_Rule{
				Action: &accessv1.Policy_Spec_Rule_Action{
					Type: &accessv1.Policy_Spec_Rule_Action_Review_{
						Review: &accessv1.Policy_Spec_Rule_Action_Review{
							Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
								{
									Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
										{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
											User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{UserRef: otherUserRef},
										}},
									},
								},
							},
						},
					},
				},
			},
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
		resp, err := srv.ListRequest(env.ctx, &vaccessv1.ListRequestOptions{UserRef: userRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "active-request", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListRequest(env.ctx, &vaccessv1.ListRequestOptions{SubjectUserRef: otherUserRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "active-request", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListRequest(env.ctx, &vaccessv1.ListRequestOptions{ServiceRef: serviceRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "active-request", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListRequest(env.ctx, &vaccessv1.ListRequestOptions{PolicyRef: policyRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "active-request", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListRequest(env.ctx, &vaccessv1.ListRequestOptions{ReviewerRef: otherUserRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "active-request", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListRequest(env.ctx, &vaccessv1.ListRequestOptions{State: accessv1.Request_Status_State_PENDING})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "pending-request", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListRequest(env.ctx, &vaccessv1.ListRequestOptions{Urgency: accessv1.Request_Spec_LOW})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "pending-request", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListRequest(env.ctx, &vaccessv1.ListRequestOptions{IsActive: true})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "active-request", resp.Items[0].Metadata.Name)
	}

	requestRef := &metav1.ObjectReference{Name: "active-request", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &accessv1.Review{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindReview,
		Metadata:   newRscStoreMetadata("approved-review", now),
		Spec:       &accessv1.Review_Spec{Decision: accessv1.Review_Spec_DECISION_APPROVE},
		Status: &accessv1.Review_Status{
			UserRef:    userRef,
			RequestRef: requestRef,
		},
	})
	insertRscStoreObject(t, env, &accessv1.Review{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindReview,
		Metadata:   newRscStoreMetadata("pending-review", now.Add(time.Second)),
		Spec:       &accessv1.Review_Spec{},
		Status: &accessv1.Review_Status{
			UserRef: otherUserRef,
		},
	})

	{
		resp, err := srv.ListReview(env.ctx, &vaccessv1.ListReviewOptions{UserRef: userRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "approved-review", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListReview(env.ctx, &vaccessv1.ListReviewOptions{RequestRef: requestRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "approved-review", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListReview(env.ctx, &vaccessv1.ListReviewOptions{Decision: accessv1.Review_Spec_DECISION_APPROVE})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "approved-review", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListReview(env.ctx, &vaccessv1.ListReviewOptions{IsDecided: true})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "approved-review", resp.Items[0].Metadata.Name)
	}
}
