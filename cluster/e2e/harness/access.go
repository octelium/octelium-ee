// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
)

const RequestBudget = 90 * time.Second

func Seconds(arg uint32) *metav1.Duration {
	return &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: arg}}
}

func Minutes(arg uint32) *metav1.Duration {
	return &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: arg}}
}

func UserReviewer(usr *corev1.User) *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
		Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
			User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
				UserRef: umetav1.GetObjectReference(usr),
			},
		},
	}
}

func GroupReviewer(grp *corev1.Group) *accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer {
	return &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
		Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_{
			Group: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group{
				GroupRef: umetav1.GetObjectReference(grp),
			},
		},
	}
}

func CatalogResource(cat *accessv1.Catalog) *accessv1.Policy_Spec_Rule_Condition {
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

func GroupSubject(grp *corev1.Group) *accessv1.Policy_Spec_Rule_Condition {
	return &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_Subject_{
			Subject: &accessv1.Policy_Spec_Rule_Condition_Subject{
				Type: &accessv1.Policy_Spec_Rule_Condition_Subject_GroupRef{
					GroupRef: umetav1.GetObjectReference(grp),
				},
			},
		},
	}
}

func AllOf(conds ...*accessv1.Policy_Spec_Rule_Condition) *accessv1.Policy_Spec_Rule_Condition {
	return &accessv1.Policy_Spec_Rule_Condition{
		Type: &accessv1.Policy_Spec_Rule_Condition_All_{
			All: &accessv1.Policy_Spec_Rule_Condition_All{Of: conds},
		},
	}
}

func CatalogRequest(cat *accessv1.Catalog, duration *metav1.Duration) *accessv1.Request {
	return &accessv1.Request{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &accessv1.Request_Spec{
			Urgency:       accessv1.Request_Spec_NORMAL,
			Justification: "e2e",
			Resource: &accessv1.Request_Spec_Resource{
				Type: &accessv1.Request_Spec_Resource_Catalog_{
					Catalog: &accessv1.Request_Spec_Resource_Catalog{
						CatalogRef: umetav1.GetObjectReference(cat),
					},
				},
			},
			Duration: duration,
		},
	}
}

func ReviewOf(req *accessv1.Request,
	decision accessv1.Review_Spec_Decision) *accessv1.Review {
	return &accessv1.Review{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &accessv1.Review_Spec{
			Decision:      decision,
			Justification: "e2e",
		},
		Status: &accessv1.Review_Status{
			RequestRef: umetav1.GetObjectReference(req),
		},
	}
}

func (h *H) CreateCatalog(t *testing.T, services []string, namespaces []string) *accessv1.Catalog {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.AccessC().CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{Name: h.Name()},
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Services:   services,
					Namespaces: namespaces,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Could not create the Catalog: %+v", err)
	}

	t.Cleanup(func() {
		h.deleteQuietly(t, "Catalog", ret.Metadata.Name, func(ctx context.Context) error {
			_, err := h.AccessC().DeleteCatalog(ctx,
				&metav1.DeleteOptions{Uid: ret.Metadata.Uid})
			return err
		})
	})

	return ret
}

func (h *H) CreateAccessPolicy(t *testing.T,
	rules ...*accessv1.Policy_Spec_Rule) *accessv1.Policy {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.AccessC().CreatePolicy(ctx, &accessv1.Policy{
		Metadata: &metav1.Metadata{Name: h.Name()},
		Spec:     &accessv1.Policy_Spec{Rules: rules},
	})
	if err != nil {
		t.Fatalf("Could not create the access Policy: %+v", err)
	}

	t.Cleanup(func() {
		h.deleteQuietly(t, "Policy", ret.Metadata.Name, func(ctx context.Context) error {
			_, err := h.AccessC().DeletePolicy(ctx,
				&metav1.DeleteOptions{Uid: ret.Metadata.Uid})
			return err
		})
	})

	return ret
}

func (h *H) CreateRequest(t *testing.T, a *Actor, req *accessv1.Request) *accessv1.Request {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.AccessUserC(a.Conn).CreateRequest(ctx, req)
	if err != nil {
		t.Fatalf("Could not create the Request as %s: %+v", a.User.Metadata.Name, err)
	}

	t.Cleanup(func() {
		h.deleteQuietly(t, "Request", ret.Metadata.Name, func(ctx context.Context) error {
			_, err := h.AccessC().DeleteRequest(ctx,
				&metav1.DeleteOptions{Uid: ret.Metadata.Uid})
			return err
		})
	})

	return ret
}

func (h *H) Review(t *testing.T, a *Actor,
	req *accessv1.Request, decision accessv1.Review_Spec_Decision) *accessv1.Review {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.AccessReviewerC(a.Conn).CreateReview(ctx, ReviewOf(req, decision))
	if err != nil {
		t.Fatalf("Could not review the Request %s as %s: %+v",
			req.Metadata.Name, a.User.Metadata.Name, err)
	}

	return ret
}

func (h *H) GetRequest(t *testing.T, req *accessv1.Request) *accessv1.Request {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.AccessC().GetRequest(ctx, &metav1.GetOptions{Uid: req.Metadata.Uid})
	if err != nil {
		t.Fatalf("Could not get the Request %s: %+v", req.Metadata.Name, err)
	}

	return ret
}

func (h *H) WaitRequestState(t *testing.T, req *accessv1.Request,
	want accessv1.Request_Status_State_Status, budget time.Duration) *accessv1.Request {
	t.Helper()

	var ret *accessv1.Request

	h.Eventually(t, "the Request "+req.Metadata.Name+" to become "+want.String(), budget,
		func(ctx context.Context) error {
			cur, err := h.AccessC().GetRequest(ctx,
				&metav1.GetOptions{Uid: req.Metadata.Uid})
			if err != nil {
				return err
			}

			if cur.Status.State == nil {
				return errors.Errorf("the Request has no state yet")
			}
			if cur.Status.State.Status != want {
				return errors.Errorf("the Request is %s, want %s",
					cur.Status.State.Status, want)
			}

			ret = cur
			return nil
		})

	return ret
}

func (h *H) WaitRequestStep(t *testing.T,
	req *accessv1.Request, step int32, budget time.Duration) *accessv1.Request {
	t.Helper()

	var ret *accessv1.Request

	h.Eventually(t, "the Request to reach the review step", budget,
		func(ctx context.Context) error {
			cur, err := h.AccessC().GetRequest(ctx,
				&metav1.GetOptions{Uid: req.Metadata.Uid})
			if err != nil {
				return err
			}

			if cur.Status.Review == nil {
				return errors.Errorf("the Request has no review status yet")
			}
			if cur.Status.Review.CurrentStep != step {
				return errors.Errorf("the Request is at step %d, want %d",
					cur.Status.Review.CurrentStep, step)
			}

			ret = cur
			return nil
		})

	return ret
}
