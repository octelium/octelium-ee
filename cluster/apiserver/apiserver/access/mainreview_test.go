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
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestMainReview(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	mainSrv := NewServerMain(tst.C.OcteliumC)

	review, err := tst.C.OcteliumC.AccessC().CreateReview(ctx, &accessv1.Review{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Review_Spec{
			Decision:      accessv1.Review_Spec_DECISION_APPROVE,
			Justification: utilrand.GetRandomString(16),
		},
		Status: &accessv1.Review_Status{},
	})
	assert.Nil(t, err, "%+v", err)

	reviewG, err := mainSrv.GetReview(ctx, &metav1.GetOptions{
		Uid: review.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, review.Metadata.Uid, reviewG.Metadata.Uid)

	reviewList, err := mainSrv.ListReview(ctx, &accessv1.ListReviewOptions{})
	assert.Nil(t, err)
	assert.True(t, len(reviewList.Items) > 0)

	found := false
	for _, item := range reviewList.Items {
		if item.Metadata.Uid == review.Metadata.Uid {
			found = true
		}
	}
	assert.True(t, found)

	_, err = mainSrv.DeleteReview(ctx, &metav1.DeleteOptions{
		Uid: review.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	_, err = mainSrv.GetReview(ctx, &metav1.GetOptions{
		Uid: review.Metadata.Uid,
	})
	assert.NotNil(t, err)
}

func TestValidateMainReview(t *testing.T) {
	srv := &ServerMain{}
	for _, decision := range []accessv1.Review_Spec_Decision{
		accessv1.Review_Spec_DECISION_UNSET,
		accessv1.Review_Spec_DECISION_APPROVE,
		accessv1.Review_Spec_DECISION_REJECT,
	} {
		arg := &accessv1.Review{Spec: &accessv1.Review_Spec{Decision: decision}, Status: &accessv1.Review_Status{}}
		assert.Nil(t, srv.validateReview(t.Context(), arg))
		arg.Status.UserRef = &metav1.ObjectReference{Name: "reviewer"}
		arg.Status.RequestRef = &metav1.ObjectReference{Name: "request"}
		assert.Nil(t, srv.validateReview(t.Context(), arg))
	}
	for _, arg := range []*accessv1.Review{
		{},
		{Spec: &accessv1.Review_Spec{}},
		{Spec: &accessv1.Review_Spec{Decision: 99}, Status: &accessv1.Review_Status{}},
		{Spec: &accessv1.Review_Spec{}, Status: &accessv1.Review_Status{UserRef: &metav1.ObjectReference{}}},
		{Spec: &accessv1.Review_Spec{}, Status: &accessv1.Review_Status{RequestRef: &metav1.ObjectReference{}}},
	} {
		err := srv.validateReview(t.Context(), arg)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}
