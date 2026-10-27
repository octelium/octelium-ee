// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type tstAccessRequestOpts struct {
	name         string
	createdAt    time.Time
	state        accessv1.Request_Status_State_Status
	lastStates   []accessv1.Request_Status_State_Status
	effect       accessv1.Policy_Spec_Rule_Effect
	requester    *metav1.ObjectReference
	subject      *metav1.ObjectReference
	serviceRef   *metav1.ObjectReference
	catalogRef   *metav1.ObjectReference
	policyRef    *metav1.ObjectReference
	origin       accessv1.Origin_Type
	urgency      accessv1.Request_Spec_Urgency
	approvalAt   time.Time
	decidedAt    time.Time
	accessEndsAt time.Time
	deadline     time.Time
}

func newTstAccessRequest(o *tstAccessRequestOpts) *accessv1.Request {
	ret := &accessv1.Request{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindRequest,
		Metadata:   newRscStoreMetadata(o.name, o.createdAt),
		Spec: &accessv1.Request_Spec{
			Urgency:  o.urgency,
			Resource: &accessv1.Request_Spec_Resource{},
		},
		Status: &accessv1.Request_Status{
			State:     &accessv1.Request_Status_State{Status: o.state},
			UserRef:   o.requester,
			PolicyRef: o.policyRef,
		},
	}

	if o.subject != nil {
		ret.Spec.Subject = &accessv1.Request_Spec_Subject{
			Type: &accessv1.Request_Spec_Subject_UserRef{UserRef: o.subject},
		}
	}

	if o.serviceRef != nil {
		ret.Spec.Resource.Type = &accessv1.Request_Spec_Resource_ServiceRef{ServiceRef: o.serviceRef}
	}

	if o.catalogRef != nil {
		ret.Spec.Resource.Type = &accessv1.Request_Spec_Resource_Catalog_{
			Catalog: &accessv1.Request_Spec_Resource_Catalog{CatalogRef: o.catalogRef},
		}
	}

	for _, state := range o.lastStates {
		ret.Status.LastStates = append(ret.Status.LastStates, &accessv1.Request_Status_State{Status: state})
	}

	if o.effect != accessv1.Policy_Spec_Rule_EFFECT_UNSET {
		ret.Status.Rule = &accessv1.Policy_Spec_Rule{Effect: o.effect}
	}

	if o.origin != accessv1.Origin_TYPE_UNSET {
		ret.Status.Origin = &accessv1.Origin{Type: o.origin}
	}

	if !o.approvalAt.IsZero() {
		ret.Status.ApprovalStartAt = timestamppb.New(o.approvalAt)
	}

	if !o.decidedAt.IsZero() {
		ret.Status.ApprovalEndAt = timestamppb.New(o.decidedAt)
	}

	if !o.accessEndsAt.IsZero() {
		ret.Status.AccessEndsAt = timestamppb.New(o.accessEndsAt)
	}

	if !o.deadline.IsZero() {
		ret.Spec.Deadline = timestamppb.New(o.deadline)
	}

	return ret
}

type tstAccessActivity struct {
	now       time.Time
	alice     *metav1.ObjectReference
	bob       *metav1.ObjectReference
	svc       *metav1.ObjectReference
	catalog   *metav1.ObjectReference
	policyA   *metav1.ObjectReference
	policyB   *metav1.ObjectReference
	requests  map[string]*accessv1.Request
	reviewer  *metav1.ObjectReference
	reviewer2 *metav1.ObjectReference
}

func insertTstAccessActivity(t *testing.T, env *rscStoreTestEnv) *tstAccessActivity {
	now := time.Now().UTC().Truncate(time.Second)

	ret := &tstAccessActivity{
		now:       now,
		alice:     &metav1.ObjectReference{Name: "alice", Uid: vutils.UUIDv4()},
		bob:       &metav1.ObjectReference{Name: "bob", Uid: vutils.UUIDv4()},
		svc:       &metav1.ObjectReference{Name: "prod-db.default", Uid: vutils.UUIDv4()},
		catalog:   &metav1.ObjectReference{Name: "support", Uid: vutils.UUIDv4()},
		policyA:   &metav1.ObjectReference{Name: "policy-a", Uid: vutils.UUIDv4()},
		policyB:   &metav1.ObjectReference{Name: "policy-b", Uid: vutils.UUIDv4()},
		reviewer:  &metav1.ObjectReference{Name: "carol", Uid: vutils.UUIDv4()},
		reviewer2: &metav1.ObjectReference{Name: "dave", Uid: vutils.UUIDv4()},
		requests:  make(map[string]*accessv1.Request),
	}

	insertRscStoreObject(t, env, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name:            ret.alice.Name,
			Uid:             ret.alice.Uid,
			DisplayName:     "Alice Smith",
			PicURL:          "https://example.com/alice.png",
			ResourceVersion: vutils.UUIDv7(),
			CreatedAt:       timestamppb.New(now.Add(-48 * time.Hour)),
		},
		Spec:   &corev1.User_Spec{},
		Status: &corev1.User_Status{},
	})

	items := []*tstAccessRequestOpts{
		{
			name: "auto-approved", createdAt: now.Add(-50 * time.Minute),
			state: accessv1.Request_Status_State_APPROVED, effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
			requester: ret.alice, subject: ret.alice, serviceRef: ret.svc, policyRef: ret.policyA,
			origin: accessv1.Origin_API, urgency: accessv1.Request_Spec_NORMAL,
			approvalAt: now.Add(-50 * time.Minute), decidedAt: now.Add(-50 * time.Minute),
			accessEndsAt: now.Add(30 * time.Minute),
		},
		{
			name: "reviewed-approved", createdAt: now.Add(-40 * time.Minute),
			state: accessv1.Request_Status_State_APPROVED, effect: accessv1.Policy_Spec_Rule_REVIEW,
			lastStates: []accessv1.Request_Status_State_Status{accessv1.Request_Status_State_PENDING},
			requester:  ret.alice, subject: ret.bob, serviceRef: ret.svc, policyRef: ret.policyA,
			origin: accessv1.Origin_INTEGRATION, urgency: accessv1.Request_Spec_HIGH,
			approvalAt: now.Add(-40 * time.Minute), decidedAt: now.Add(-30 * time.Minute),
			accessEndsAt: now.Add(5 * time.Hour),
		},
		{
			name: "reviewed-rejected", createdAt: now.Add(-30 * time.Minute),
			state: accessv1.Request_Status_State_REJECTED, effect: accessv1.Policy_Spec_Rule_REVIEW,
			lastStates: []accessv1.Request_Status_State_Status{accessv1.Request_Status_State_PENDING},
			requester:  ret.bob, subject: ret.bob, catalogRef: ret.catalog, policyRef: ret.policyB,
			origin: accessv1.Origin_API, urgency: accessv1.Request_Spec_NORMAL,
			approvalAt: now.Add(-30 * time.Minute), decidedAt: now.Add(-20 * time.Minute),
		},
		{
			name: "approved-then-expired", createdAt: now.Add(-25 * time.Minute),
			state: accessv1.Request_Status_State_EXPIRED, effect: accessv1.Policy_Spec_Rule_REVIEW,
			lastStates: []accessv1.Request_Status_State_Status{
				accessv1.Request_Status_State_APPROVED, accessv1.Request_Status_State_PENDING,
			},
			requester: ret.bob, subject: ret.bob, catalogRef: ret.catalog, policyRef: ret.policyB,
			approvalAt: now.Add(-25 * time.Minute), decidedAt: now.Add(-15 * time.Minute),
		},
		{
			name: "expired-while-pending", createdAt: now.Add(-20 * time.Minute),
			state: accessv1.Request_Status_State_EXPIRED, effect: accessv1.Policy_Spec_Rule_REVIEW,
			lastStates: []accessv1.Request_Status_State_Status{accessv1.Request_Status_State_PENDING},
			requester:  ret.bob, subject: ret.bob, catalogRef: ret.catalog, policyRef: ret.policyB,
			approvalAt: now.Add(-20 * time.Minute), decidedAt: now.Add(-1 * time.Minute),
		},
		{
			name: "unmatched", createdAt: now.Add(-15 * time.Minute),
			state: accessv1.Request_Status_State_REJECTED,
			requester: ret.bob, subject: ret.bob, serviceRef: ret.svc,
		},
		{
			name: "denied", createdAt: now.Add(-10 * time.Minute),
			state: accessv1.Request_Status_State_REJECTED, effect: accessv1.Policy_Spec_Rule_DENY,
			requester: ret.bob, subject: ret.bob, serviceRef: ret.svc, policyRef: ret.policyB,
		},
		{
			name: "pending-stale", createdAt: now.Add(-5 * time.Minute),
			state: accessv1.Request_Status_State_PENDING, effect: accessv1.Policy_Spec_Rule_REVIEW,
			requester: ret.alice, subject: ret.alice, serviceRef: ret.svc, policyRef: ret.policyA,
			approvalAt: now.Add(-30 * time.Hour), urgency: accessv1.Request_Spec_HIGHEST,
			deadline: now.Add(-time.Hour),
		},
		{
			name: "pending-fresh", createdAt: now.Add(-1 * time.Minute),
			state: accessv1.Request_Status_State_PENDING, effect: accessv1.Policy_Spec_Rule_REVIEW,
			requester: ret.alice, subject: ret.alice, serviceRef: ret.svc, policyRef: ret.policyA,
			approvalAt: now.Add(-1 * time.Minute),
		},
	}

	for _, itm := range items {
		req := newTstAccessRequest(itm)
		insertRscStoreObject(t, env, req)
		ret.requests[itm.name] = req
	}

	reviews := []struct {
		name     string
		reviewer *metav1.ObjectReference
		request  string
		decision accessv1.Review_Spec_Decision
		origin   accessv1.Origin_Type
	}{
		{"review-1", ret.reviewer, "reviewed-approved", accessv1.Review_Spec_DECISION_APPROVE, accessv1.Origin_INTEGRATION},
		{"review-2", ret.reviewer, "reviewed-rejected", accessv1.Review_Spec_DECISION_REJECT, accessv1.Origin_API},
		{"review-3", ret.reviewer, "approved-then-expired", accessv1.Review_Spec_DECISION_APPROVE, accessv1.Origin_API},
		{"review-4", ret.reviewer2, "approved-then-expired", accessv1.Review_Spec_DECISION_APPROVE, accessv1.Origin_INTEGRATION},
	}

	for _, itm := range reviews {
		req := ret.requests[itm.request]
		insertRscStoreObject(t, env, &accessv1.Review{
			ApiVersion: uaccessv1.APIVersion,
			Kind:       uaccessv1.KindReview,
			Metadata:   newRscStoreMetadata(itm.name, now.Add(-10*time.Minute)),
			Spec:       &accessv1.Review_Spec{Decision: itm.decision},
			Status: &accessv1.Review_Status{
				UserRef: itm.reviewer,
				RequestRef: &metav1.ObjectReference{
					Name: req.Metadata.Name,
					Uid:  req.Metadata.Uid,
				},
				Origin: &accessv1.Origin{Type: itm.origin},
			},
		})
	}

	return ret
}

func TestAccessRequestSummaryActivity(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	a := insertTstAccessActivity(t, env)

	{
		resp, err := env.srv.getSummaryAccessRequest(env.ctx, &vaccessv1.GetRequestSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 9, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalAutoApproved)
		assert.EqualValues(t, 6, resp.TotalReviewed)
		assert.EqualValues(t, 1, resp.TotalDenied)
		assert.EqualValues(t, 1, resp.TotalUnmatched)
		assert.EqualValues(t, 1, resp.TotalOnBehalf)
		assert.EqualValues(t, 1, resp.TotalFromIntegration)
		assert.EqualValues(t, 1, resp.TotalExpiringSoon)
		assert.EqualValues(t, 1, resp.TotalPendingStale)
		assert.EqualValues(t, 1, resp.TotalDeadlinePassed)
		assert.EqualValues(t, 1, resp.TotalPendingDeadlinePassed)
		assert.EqualValues(t, 3, resp.TotalGranted)
		assert.NotNil(t, resp.OldestPendingAt)
		assert.Equal(t, a.now.Add(-30*time.Hour).Unix(), resp.OldestPendingAt.AsTime().Unix())

		assert.NotNil(t, resp.DecisionTime)
		assert.EqualValues(t, 3, resp.DecisionTime.Count)
		assert.InDelta(t, 600, resp.DecisionTime.AverageSeconds, 1)
		assert.InDelta(t, 600, resp.DecisionTime.P50Seconds, 1)
		assert.InDelta(t, 600, resp.DecisionTime.MaxSeconds, 1)
	}

	{
		resp, err := env.srv.getSummaryAccessRequest(env.ctx, &vaccessv1.GetRequestSummaryRequest{
			PolicyRef: a.policyB,
		})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 4, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalDenied)
		assert.EqualValues(t, 0, resp.TotalUnmatched)
		assert.EqualValues(t, 2, resp.DecisionTime.Count)
	}

	{
		resp, err := env.srv.getSummaryAccessRequest(env.ctx, &vaccessv1.GetRequestSummaryRequest{
			CatalogRef: &metav1.ObjectReference{Name: a.catalog.Name},
		})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 3, resp.TotalNumber)
	}

	{
		resp, err := env.srv.getSummaryAccessRequest(env.ctx, &vaccessv1.GetRequestSummaryRequest{
			ServiceRef:     a.svc,
			SubjectUserRef: a.alice,
		})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 3, resp.TotalNumber)
		assert.EqualValues(t, 2, resp.TotalPending)
		assert.Nil(t, resp.DecisionTime)
	}

	{
		resp, err := env.srv.getSummaryAccessRequest(env.ctx, &vaccessv1.GetRequestSummaryRequest{
			UserRef: a.alice,
			Common: &vmetav1.CommonSummaryOptions{
				From:        timestamppb.New(a.now.Add(-10 * time.Minute)),
				To:          timestamppb.New(a.now),
				CompareFrom: timestamppb.New(a.now.Add(-time.Hour)),
				CompareTo:   timestamppb.New(a.now.Add(-10 * time.Minute)),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.NotNil(t, resp.Previous)
		assert.EqualValues(t, 2, resp.Previous.TotalNumber)
	}

	{
		_, err := env.srv.getSummaryAccessRequest(env.ctx, &vaccessv1.GetRequestSummaryRequest{
			PolicyRef: &metav1.ObjectReference{Name: "INVALID NAME"},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		resp, err := env.srv.getSummaryAccessReview(env.ctx, &vaccessv1.GetReviewSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 4, resp.TotalNumber)
		assert.EqualValues(t, 2, resp.TotalFromIntegration)
		assert.EqualValues(t, 2, resp.TotalUser)
	}

	{
		resp, err := env.srv.getSummaryAccessReview(env.ctx, &vaccessv1.GetReviewSummaryRequest{
			UserRef: a.reviewer,
		})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 3, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalFromIntegration)
		assert.EqualValues(t, 1, resp.TotalRejected)
	}

	{
		req := a.requests["approved-then-expired"]
		resp, err := env.srv.getSummaryAccessReview(env.ctx, &vaccessv1.GetReviewSummaryRequest{
			RequestRef: &metav1.ObjectReference{Uid: req.Metadata.Uid},
		})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 2, resp.TotalApproved)
	}
}

func TestAccessRequestListActivityFilters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	a := insertTstAccessActivity(t, env)
	srv := &srvAccess{s: env.srv}

	names := func(itemList *accessv1.RequestList) []string {
		var ret []string
		for _, itm := range itemList.Items {
			ret = append(ret, itm.Metadata.Name)
		}
		return ret
	}

	list := func(req *vaccessv1.ListRequestOptions) []string {
		if req.Common == nil {
			req.Common = &vmetav1.CommonListOptions{ItemsPerPage: 100}
		}
		itemList, err := srv.ListRequest(env.ctx, req)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return nil
		}
		return names(itemList)
	}

	assert.ElementsMatch(t, []string{"auto-approved"},
		list(&vaccessv1.ListRequestOptions{Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE}))
	assert.ElementsMatch(t, []string{"denied"},
		list(&vaccessv1.ListRequestOptions{Effect: accessv1.Policy_Spec_Rule_DENY}))
	assert.ElementsMatch(t, []string{"unmatched"},
		list(&vaccessv1.ListRequestOptions{IsUnmatched: true}))
	assert.ElementsMatch(t, []string{"auto-approved"},
		list(&vaccessv1.ListRequestOptions{IsExpiringSoon: true}))
	assert.ElementsMatch(t, []string{"pending-stale"},
		list(&vaccessv1.ListRequestOptions{IsPendingStale: true}))
	assert.ElementsMatch(t, []string{"reviewed-approved"},
		list(&vaccessv1.ListRequestOptions{IsOnBehalf: true}))
	assert.ElementsMatch(t, []string{"reviewed-approved"},
		list(&vaccessv1.ListRequestOptions{OriginType: accessv1.Origin_INTEGRATION}))

	assert.Equal(t, []string{"auto-approved", "reviewed-approved"},
		list(&vaccessv1.ListRequestOptions{
			IsActive: true,
			OrderBy: &vaccessv1.ListRequestOptions_OrderBy{
				Type: vaccessv1.ListRequestOptions_OrderBy_ACCESS_ENDS_AT,
				Mode: vmetav1.CommonListOptions_OrderBy_ASC,
			},
		}))

	assert.Equal(t, []string{"pending-stale", "pending-fresh"},
		list(&vaccessv1.ListRequestOptions{
			State: accessv1.Request_Status_State_PENDING,
			OrderBy: &vaccessv1.ListRequestOptions_OrderBy{
				Type: vaccessv1.ListRequestOptions_OrderBy_APPROVAL_START_AT,
				Mode: vmetav1.CommonListOptions_OrderBy_ASC,
			},
		}))

	{
		ret := list(&vaccessv1.ListRequestOptions{
			OrderBy: &vaccessv1.ListRequestOptions_OrderBy{
				Type: vaccessv1.ListRequestOptions_OrderBy_DEADLINE,
				Mode: vmetav1.CommonListOptions_OrderBy_DESC,
			},
		})
		assert.Equal(t, 9, len(ret))
		assert.Equal(t, "pending-stale", ret[0])
	}

	{
		itemList, err := srv.ListReview(env.ctx, &vaccessv1.ListReviewOptions{
			OriginType: accessv1.Origin_INTEGRATION,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(itemList.Items))
	}

	assert.NotNil(t, a)
}

func TestAccessRequestDataPoint(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	a := insertTstAccessActivity(t, env)

	from := a.now.Add(-time.Hour)
	to := a.now

	{
		resp, err := env.srv.getRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
			From:     timestamppb.New(from),
			To:       timestamppb.New(to),
			Interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 10}},
		})
		assert.Nil(t, err, "%+v", err)
		assert.True(t, len(resp.Datapoints) >= 6 && len(resp.Datapoints) <= 8, "%d", len(resp.Datapoints))
		assert.Equal(t, 0, len(resp.Series))

		var total int64
		for i, dp := range resp.Datapoints {
			total += dp.Count
			if i > 0 {
				assert.Equal(t, int64(600),
					dp.Timestamp.AsTime().Unix()-resp.Datapoints[i-1].Timestamp.AsTime().Unix())
			}
		}
		assert.EqualValues(t, 9, total)
	}

	{
		resp, err := env.srv.getRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
			From:    timestamppb.New(from),
			To:      timestamppb.New(to),
			GroupBy: vaccessv1.GetRequestDataPointRequest_STATE,
		})
		assert.Nil(t, err, "%+v", err)

		totals := make(map[string]int64)
		for _, series := range resp.Series {
			totals[series.Key] = series.Total
			assert.Equal(t, len(resp.Datapoints), len(series.Datapoints))
		}
		assert.EqualValues(t, map[string]int64{
			"APPROVED": 2,
			"REJECTED": 3,
			"EXPIRED":  2,
			"PENDING":  2,
		}, totals)
		assert.Equal(t, "REJECTED", resp.Series[0].Key)
	}

	{
		resp, err := env.srv.getRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
			From:        timestamppb.New(from),
			To:          timestamppb.New(to),
			GroupBy:     vaccessv1.GetRequestDataPointRequest_POLICY,
			LimitSeries: 1,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Series))
		assert.EqualValues(t, 4, resp.Series[0].Total)
		switch resp.Series[0].Key {
		case a.policyA.Uid:
			assert.Equal(t, a.policyA.Name, resp.Series[0].DisplayName)
		case a.policyB.Uid:
			assert.Equal(t, a.policyB.Name, resp.Series[0].DisplayName)
		default:
			t.Fatalf("Unexpected series key: %s", resp.Series[0].Key)
		}
	}

	{
		resp, err := env.srv.getRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
			From:      timestamppb.New(from),
			To:        timestamppb.New(to),
			GroupBy:   vaccessv1.GetRequestDataPointRequest_EFFECT,
			PolicyRef: a.policyB,
		})
		assert.Nil(t, err, "%+v", err)

		totals := make(map[string]int64)
		for _, series := range resp.Series {
			totals[series.Key] = series.Total
		}
		assert.EqualValues(t, map[string]int64{
			"REVIEW": 3,
			"DENY":   1,
		}, totals)
	}

	{
		_, err := env.srv.getRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
			From:     timestamppb.New(a.now.Add(-30 * 24 * time.Hour)),
			To:       timestamppb.New(to),
			Interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 10}},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := env.srv.getRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
			From: timestamppb.New(to),
			To:   timestamppb.New(from),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
}

func TestGetAccessDataPointInterval(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		window time.Duration
		want   time.Duration
	}{
		{time.Hour, time.Minute},
		{6 * time.Hour, 5 * time.Minute},
		{24 * time.Hour, 15 * time.Minute},
		{7 * 24 * time.Hour, 3 * time.Hour},
		{30 * 24 * time.Hour, 6 * time.Hour},
		{365 * 24 * time.Hour, 7 * 24 * time.Hour},
	}

	for _, tt := range tests {
		ret, err := getAccessDataPointInterval(now.Add(-tt.window), now, nil)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, tt.want, ret, "%s", tt.window)
	}

	{
		ret, err := getAccessDataPointInterval(now.Add(-time.Hour), now,
			&metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 5}})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 5*time.Minute, ret)
	}

	{
		_, err := getAccessDataPointInterval(now.Add(-time.Hour), now,
			&metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 10}})
		assert.NotNil(t, err)
	}
}

func TestAccessRequestTop(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	a := insertTstAccessActivity(t, env)

	{
		resp, err := env.srv.listRequestTop(env.ctx, &vaccessv1.ListRequestTopRequest{
			Dimension: vaccessv1.ListRequestTopRequest_REQUESTER,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Items))
		assert.EqualValues(t, 2, resp.TotalCount)
		assert.EqualValues(t, 0, resp.TotalOther)

		assert.Equal(t, a.bob.Uid, resp.Items[0].Ref.Uid)
		assert.EqualValues(t, 5, resp.Items[0].Count)
		assert.Equal(t, a.alice.Uid, resp.Items[1].Ref.Uid)
		assert.EqualValues(t, 4, resp.Items[1].Count)
		assert.Equal(t, "User", resp.Items[1].Ref.Kind)
		assert.Equal(t, "core/v1", resp.Items[1].Ref.ApiVersion)
		assert.Equal(t, "alice", resp.Items[1].Ref.Name)
		assert.Equal(t, "Alice Smith", resp.Items[1].DisplayName)
		assert.Equal(t, "https://example.com/alice.png", resp.Items[1].PicURL)
	}

	{
		resp, err := env.srv.listRequestTop(env.ctx, &vaccessv1.ListRequestTopRequest{
			Dimension: vaccessv1.ListRequestTopRequest_RESOURCE,
			Limit:     1,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, a.svc.Uid, resp.Items[0].Ref.Uid)
		assert.Equal(t, "Service", resp.Items[0].Ref.Kind)
		assert.EqualValues(t, 6, resp.Items[0].Count)
		assert.EqualValues(t, 2, resp.TotalCount)
		assert.EqualValues(t, 3, resp.TotalOther)
	}

	{
		resp, err := env.srv.listRequestTop(env.ctx, &vaccessv1.ListRequestTopRequest{
			Dimension:  vaccessv1.ListRequestTopRequest_RESOURCE,
			CatalogRef: a.catalog,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, "Catalog", resp.Items[0].Ref.Kind)
		assert.Equal(t, "access/v1", resp.Items[0].Ref.ApiVersion)
	}

	{
		resp, err := env.srv.listRequestTop(env.ctx, &vaccessv1.ListRequestTopRequest{
			Dimension: vaccessv1.ListRequestTopRequest_POLICY,
			State:     accessv1.Request_Status_State_PENDING,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, a.policyA.Uid, resp.Items[0].Ref.Uid)
		assert.EqualValues(t, 2, resp.Items[0].Count)
	}

	{
		resp, err := env.srv.listRequestTop(env.ctx, &vaccessv1.ListRequestTopRequest{
			Dimension: vaccessv1.ListRequestTopRequest_SUBJECT,
			From:      timestamppb.New(a.now.Add(-6 * time.Minute)),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, a.alice.Uid, resp.Items[0].Ref.Uid)
		assert.EqualValues(t, 2, resp.Items[0].Count)
	}

	{
		_, err := env.srv.listRequestTop(env.ctx, &vaccessv1.ListRequestTopRequest{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		resp, err := env.srv.listReviewTop(env.ctx, &vaccessv1.ListReviewTopRequest{
			Dimension: vaccessv1.ListReviewTopRequest_REVIEWER,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Items))
		assert.Equal(t, a.reviewer.Uid, resp.Items[0].Ref.Uid)
		assert.EqualValues(t, 3, resp.Items[0].Count)
	}

	{
		resp, err := env.srv.listReviewTop(env.ctx, &vaccessv1.ListReviewTopRequest{
			Dimension: vaccessv1.ListReviewTopRequest_REVIEWER,
			Decision:  accessv1.Review_Spec_DECISION_REJECT,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.EqualValues(t, 1, resp.Items[0].Count)
	}
}

func TestAccessPolicySummarySurfaces(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	integrationRef := &metav1.ObjectReference{Name: "slack", Uid: vutils.UUIDv4()}
	surface := func(mode accessv1.Policy_Spec_Rule_Surface_InteractionMode) *accessv1.Policy_Spec_Rule_Surface {
		return &accessv1.Policy_Spec_Rule_Surface{
			Destination: &accessv1.Policy_Spec_Rule_Surface_Destination{
				IntegrationRef: integrationRef,
				Audience:       accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
			},
			InteractionMode: mode,
		}
	}

	insertRscStoreObject(t, env, &accessv1.Policy{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindPolicy,
		Metadata:   newRscStoreMetadata("surfaces", time.Now().UTC()),
		Spec: &accessv1.Policy_Spec{
			Rules: []*accessv1.Policy_Spec_Rule{
				{
					Effect: accessv1.Policy_Spec_Rule_REVIEW,
					Action: &accessv1.Policy_Spec_Rule_Action{
						Type: &accessv1.Policy_Spec_Rule_Action_Review_{
							Review: &accessv1.Policy_Spec_Rule_Action_Review{
								Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
									{
										Surfaces: []*accessv1.Policy_Spec_Rule_Surface{
											surface(accessv1.Policy_Spec_Rule_Surface_INTERACTIVE),
											surface(accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY),
										},
									},
									{
										Surfaces: []*accessv1.Policy_Spec_Rule_Surface{
											surface(accessv1.Policy_Spec_Rule_Surface_INTERACTIVE),
										},
									},
								},
							},
						},
					},
					Notifications: []*accessv1.Policy_Spec_Rule_Surface{
						surface(accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET),
					},
				},
				{
					Effect: accessv1.Policy_Spec_Rule_AUTO_APPROVE,
					Notifications: []*accessv1.Policy_Spec_Rule_Surface{
						surface(accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET),
						surface(accessv1.Policy_Spec_Rule_Surface_INTERACTION_MODE_UNSET),
					},
				},
				{
					Effect: accessv1.Policy_Spec_Rule_DENY,
				},
			},
		},
		Status: &accessv1.Policy_Status{},
	})

	resp, err := env.srv.getSummaryAccessPolicy(env.ctx, &vaccessv1.GetPolicySummaryRequest{})
	assert.Nil(t, err, "%+v", err)
	assert.EqualValues(t, 3, resp.TotalRule)
	assert.EqualValues(t, 2, resp.TotalRuleNotification)
	assert.EqualValues(t, 3, resp.TotalReviewSurface)
	assert.EqualValues(t, 2, resp.TotalInteractiveSurface)
}

func TestAccessActivityWindowBoundaries(t *testing.T) {
	before := time.Now().UTC()
	from, to, err := getAccessWindow(nil, nil)
	require.Nil(t, err)
	assert.Equal(t, accessDataPointDefaultWindow, to.Sub(from))
	assert.WithinDuration(t, before, to, time.Second)
	fixed := timestamppb.New(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC))
	from, to, err = getAccessWindow(nil, fixed)
	require.Nil(t, err)
	assert.Equal(t, fixed.AsTime(), to)
	assert.Equal(t, to.Add(-accessDataPointDefaultWindow), from)
	from, to, err = getAccessWindow(fixed, fixed)
	require.Nil(t, err)
	assert.Equal(t, from, to)
	for _, tt := range []struct{ from, to *timestamppb.Timestamp }{
		{from: &timestamppb.Timestamp{Seconds: 253402300800}},
		{to: &timestamppb.Timestamp{Nanos: -1}},
		{from: timestamppb.New(fixed.AsTime().Add(time.Second)), to: fixed},
	} {
		from, to, err := getAccessWindow(tt.from, tt.to)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
		assert.True(t, from.IsZero())
		assert.True(t, to.IsZero())
	}
	for _, tt := range []struct {
		interval *metav1.Duration
		want     time.Duration
	}{
		{interval: &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 1000}}, want: time.Second},
		{interval: &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 1500}}, want: time.Second},
		{interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 2}}, want: 2 * time.Second},
		{interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 2}}, want: 2 * time.Minute},
		{interval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}}, want: 2 * time.Hour},
		{interval: &metav1.Duration{Type: &metav1.Duration_Days{Days: 2}}, want: 48 * time.Hour},
		{interval: &metav1.Duration{Type: &metav1.Duration_Weeks{Weeks: 2}}, want: 14 * 24 * time.Hour},
	} {
		got, err := getAccessDataPointInterval(fixed.AsTime(), fixed.AsTime().Add(time.Minute), tt.interval)
		require.Nil(t, err)
		assert.Equal(t, tt.want, got)
	}
	for _, interval := range []*metav1.Duration{
		{Type: &metav1.Duration_Milliseconds{}},
		{Type: &metav1.Duration_Milliseconds{Milliseconds: 999}},
		{Type: &metav1.Duration_Seconds{}},
		{Type: &metav1.Duration_Minutes{}},
	} {
		_, err := getAccessDataPointInterval(fixed.AsTime(), fixed.AsTime().Add(time.Minute), interval)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
	_, err = getAccessDataPointInterval(fixed.AsTime(), fixed.AsTime().Add(20*365*24*time.Hour), nil)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestAccessActivityRPCGroupsAndValidation(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	activity := insertTstAccessActivity(t, env)
	conn := newRscStoreRPCClient(t, env)
	client := vaccessv1.NewResourceServiceClient(conn)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	request := &vaccessv1.GetRequestDataPointRequest{
		From:     timestamppb.New(activity.now.Add(-time.Hour)),
		To:       timestamppb.New(activity.now),
		Interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 5}},
	}
	for _, group := range []vaccessv1.GetRequestDataPointRequest_GroupBy{
		vaccessv1.GetRequestDataPointRequest_GROUP_BY_UNSET,
		vaccessv1.GetRequestDataPointRequest_STATE,
		vaccessv1.GetRequestDataPointRequest_URGENCY,
		vaccessv1.GetRequestDataPointRequest_EFFECT,
		vaccessv1.GetRequestDataPointRequest_ORIGIN,
		vaccessv1.GetRequestDataPointRequest_POLICY,
	} {
		t.Run(group.String(), func(t *testing.T) {
			arg := proto.Clone(request).(*vaccessv1.GetRequestDataPointRequest)
			arg.GroupBy = group
			res, err := client.GetRequestDataPoint(ctx, arg)
			require.Nil(t, err)
			var total int64
			for idx, point := range res.Datapoints {
				total += point.Count
				if idx > 0 {
					assert.Equal(t, 5*time.Minute, point.Timestamp.AsTime().Sub(res.Datapoints[idx-1].Timestamp.AsTime()))
				}
			}
			assert.Equal(t, int64(9), total)
			for _, series := range res.Series {
				var count int64
				require.Len(t, series.Datapoints, len(res.Datapoints))
				for idx, point := range series.Datapoints {
					count += point.Count
					assert.True(t, proto.Equal(res.Datapoints[idx].Timestamp, point.Timestamp))
				}
				assert.Equal(t, count, series.Total)
			}
		})
	}
	for _, mutate := range []func(*vaccessv1.GetRequestDataPointRequest){
		func(arg *vaccessv1.GetRequestDataPointRequest) { arg.GroupBy = 99 },
		func(arg *vaccessv1.GetRequestDataPointRequest) {
			arg.Interval = &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 1}}
		},
		func(arg *vaccessv1.GetRequestDataPointRequest) { arg.From, arg.To = arg.To, arg.From },
		func(arg *vaccessv1.GetRequestDataPointRequest) { arg.From = &timestamppb.Timestamp{Nanos: -1} },
		func(arg *vaccessv1.GetRequestDataPointRequest) {
			arg.To = &timestamppb.Timestamp{Seconds: 253402300800}
		},
		func(arg *vaccessv1.GetRequestDataPointRequest) { arg.PolicyRef = &metav1.ObjectReference{} },
		func(arg *vaccessv1.GetRequestDataPointRequest) { arg.CatalogRef = &metav1.ObjectReference{} },
		func(arg *vaccessv1.GetRequestDataPointRequest) { arg.ServiceRef = &metav1.ObjectReference{} },
		func(arg *vaccessv1.GetRequestDataPointRequest) { arg.UserRef = &metav1.ObjectReference{} },
	} {
		arg := proto.Clone(request).(*vaccessv1.GetRequestDataPointRequest)
		mutate(arg)
		_, err := client.GetRequestDataPoint(ctx, arg)
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "%+v", err)
	}
	for _, dim := range []vaccessv1.ListReviewTopRequest_Dimension{vaccessv1.ListReviewTopRequest_DIMENSION_UNSET, 99} {
		_, err := client.ListReviewTop(ctx, &vaccessv1.ListReviewTopRequest{Dimension: dim})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
	_, err := client.ListRequestTop(ctx, &vaccessv1.ListRequestTopRequest{Dimension: 99})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = client.ListReviewTop(ctx, &vaccessv1.ListReviewTopRequest{Dimension: vaccessv1.ListReviewTopRequest_REVIEWER, From: &timestamppb.Timestamp{Nanos: -1}})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Nil(t, env.srv.db.Close())
	_, err = client.GetRequestDataPoint(ctx, request)
	assert.Equal(t, codes.Internal, status.Code(err))
	_, err = client.ListRequestTop(ctx, &vaccessv1.ListRequestTopRequest{Dimension: vaccessv1.ListRequestTopRequest_REQUESTER})
	assert.Equal(t, codes.Internal, status.Code(err))
	_, err = client.ListReviewTop(ctx, &vaccessv1.ListReviewTopRequest{Dimension: vaccessv1.ListReviewTopRequest_REVIEWER})
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestAccessActivityRankingLimitsAndTies(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	now := time.Now().UTC().Truncate(time.Second)
	for idx := 1; idx <= 30; idx++ {
		ref := &metav1.ObjectReference{Name: fmt.Sprintf("rank-%02d", idx), Uid: fmt.Sprintf("00000000-0000-4000-8000-%012d", idx)}
		insertRscStoreObject(t, env, newTstAccessRequest(&tstAccessRequestOpts{name: fmt.Sprintf("request-%d", idx), createdAt: now, policyRef: ref}))
		insertRscStoreObject(t, env, &accessv1.Review{
			ApiVersion: "access/v1", Kind: "Review",
			Metadata: newRscStoreMetadata(fmt.Sprintf("review-%d", idx), now),
			Spec:     &accessv1.Review_Spec{Decision: accessv1.Review_Spec_DECISION_APPROVE},
			Status:   &accessv1.Review_Status{UserRef: ref},
		})
	}
	insertRscStoreObject(t, env, newTstAccessRequest(&tstAccessRequestOpts{name: "unbound-request", createdAt: now}))
	client := vaccessv1.NewResourceServiceClient(newRscStoreRPCClient(t, env))
	for _, tt := range []struct {
		limit uint32
		want  int
	}{
		{limit: 0, want: 10}, {limit: 1, want: 1}, {limit: 100, want: 30}, {limit: math.MaxUint32, want: 30},
	} {
		request, err := client.ListRequestTop(env.ctx, &vaccessv1.ListRequestTopRequest{Dimension: vaccessv1.ListRequestTopRequest_POLICY, Limit: tt.limit})
		require.Nil(t, err)
		review, err := client.ListReviewTop(env.ctx, &vaccessv1.ListReviewTopRequest{Dimension: vaccessv1.ListReviewTopRequest_REVIEWER, Limit: tt.limit})
		require.Nil(t, err)
		for _, result := range []*vaccessv1.ListTopResponse{request, review} {
			require.Len(t, result.Items, tt.want)
			assert.Equal(t, uint64(30), result.TotalCount)
			assert.Equal(t, uint64(30-tt.want), result.TotalOther)
			for idx, item := range result.Items {
				assert.Equal(t, fmt.Sprintf("00000000-0000-4000-8000-%012d", idx+1), item.Ref.Uid)
				assert.Equal(t, fmt.Sprintf("rank-%02d", idx+1), item.Ref.Name)
				assert.Equal(t, uint64(1), item.Count)
			}
		}
	}
	for _, tt := range []struct {
		limit uint32
		want  int
	}{
		{limit: 0, want: accessDataPointDefaultSeries}, {limit: 1, want: 1}, {limit: math.MaxUint32, want: accessDataPointMaxSeries},
	} {
		result, err := client.GetRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
			From: timestamppb.New(now.Add(-time.Minute)), To: timestamppb.New(now.Add(time.Minute)),
			Interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 1}},
			GroupBy:  vaccessv1.GetRequestDataPointRequest_POLICY, LimitSeries: tt.limit,
		})
		require.Nil(t, err)
		require.Len(t, result.Series, tt.want)
		for idx, series := range result.Series {
			assert.Equal(t, fmt.Sprintf("00000000-0000-4000-8000-%012d", idx+1), series.Key)
			assert.Equal(t, int64(1), series.Total)
		}
		var total int64
		for _, point := range result.Datapoints {
			total += point.Count
		}
		assert.Equal(t, int64(31), total)
	}
}

func TestAccessActivityAutomaticIntervals(t *testing.T) {
	from := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, interval := range accessDataPointAutoIntervals {
		t.Run(interval.String(), func(t *testing.T) {
			to := from.Add(accessDataPointAutoBuckets * interval)
			got, err := getAccessDataPointInterval(from, to, nil)
			require.NoError(t, err)
			assert.Equal(t, interval, got)
			buckets := getAccessDataPointBuckets(from, to, got)
			assert.LessOrEqual(t, len(buckets), accessDataPointAutoBuckets+1)
			assert.LessOrEqual(t, buckets[0], from.Unix())
			assert.LessOrEqual(t, buckets[len(buckets)-1], to.Unix())
		})
	}
	interval, err := getAccessDataPointInterval(from, from.AddDate(10, 0, 0), &metav1.Duration{})
	require.NoError(t, err)
	assert.Equal(t, 7*24*time.Hour, interval)
}

func TestAccessDataPointIntervalOverflow(t *testing.T) {
	from := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name string
		unit time.Duration
		make func(uint32) *metav1.Duration
	}{
		{name: "Milliseconds", unit: time.Millisecond, make: func(value uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: value}}
		}},
		{name: "Seconds", unit: time.Second, make: func(value uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: value}}
		}},
		{name: "Minutes", unit: time.Minute, make: func(value uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: value}}
		}},
		{name: "Hours", unit: time.Hour, make: func(value uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Hours{Hours: value}}
		}},
		{name: "Days", unit: 24 * time.Hour, make: func(value uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Days{Days: value}}
		}},
		{name: "Weeks", unit: 7 * 24 * time.Hour, make: func(value uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Weeks{Weeks: value}}
		}},
		{name: "Months", unit: 30 * 24 * time.Hour, make: func(value uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Months{Months: value}}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			limit := min(uint64(math.MaxUint32), uint64(math.MaxInt64)/uint64(tt.unit))
			got, err := getAccessDataPointInterval(from, from.Add(time.Minute), tt.make(uint32(limit)))
			require.Nil(t, err)
			assert.Equal(t, (time.Duration(limit) * tt.unit).Truncate(time.Second), got)
			if limit < math.MaxUint32 {
				for _, value := range []uint32{uint32(limit + 1), math.MaxUint32} {
					got, err := getAccessDataPointInterval(from, from.Add(time.Minute), tt.make(value))
					assert.Equal(t, codes.InvalidArgument, status.Code(err))
					assert.Zero(t, got)
				}
			}
		})
	}
	got, err := getAccessDataPointInterval(from, from.Add(time.Minute),
		&metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 307445735}})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Zero(t, got)
}

func TestAccessDataPointBucketLimit(t *testing.T) {
	for _, tt := range []struct {
		name     string
		from     time.Time
		to       time.Time
		interval *metav1.Duration
		count    int
	}{
		{name: "TruncatedInterval", from: time.Unix(0, 0), to: time.Unix(1500, 0),
			interval: &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 1999}}},
		{name: "InclusiveLimit", from: time.Unix(0, 0), to: time.Unix(1000, 0),
			interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 1}}},
		{name: "UnalignedLimit", from: time.Unix(0, 900000000), to: time.Unix(1000, 100000000),
			interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 1}}},
		{name: "NegativeLimit", from: time.Unix(-1000, 0), to: time.Unix(0, 0),
			interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 1}}},
		{name: "AutomaticLimit", from: time.Unix(0, 0), to: time.Unix(1000*7*24*60*60, 0)},
		{name: "MaximumBuckets", from: time.Unix(0, 0), to: time.Unix(999, 0), count: 1000,
			interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 1}}},
		{name: "MaximumTruncatedBuckets", from: time.Unix(0, 0), to: time.Unix(999, 0), count: 1000,
			interval: &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 1999}}},
		{name: "MaximumUnalignedBuckets", from: time.Unix(0, 900000000), to: time.Unix(999, 900000000), count: 1000,
			interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 1}}},
		{name: "MaximumNegativeBuckets", from: time.Unix(-999, 0), to: time.Unix(0, 0), count: 1000,
			interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 1}}},
		{name: "MaximumAutomaticBuckets", from: time.Unix(0, 0), to: time.Unix(999*7*24*60*60, 0), count: 1000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			interval, err := getAccessDataPointInterval(tt.from, tt.to, tt.interval)
			if tt.count == 0 {
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
				assert.Zero(t, interval)
				return
			}
			require.Nil(t, err)
			assert.Len(t, getAccessDataPointBuckets(tt.from, tt.to, interval), tt.count)
		})
	}
}

func TestAccessDataPointBucketAlignment(t *testing.T) {
	for _, tt := range []struct {
		name string
		from time.Time
		to   time.Time
		want []int64
	}{
		{name: "Negative", from: time.Unix(-61, 0), to: time.Unix(-1, 0), want: []int64{-120, -60}},
		{name: "NegativeFraction", from: time.Unix(-1, 500000000), to: time.Unix(-1, 900000000), want: []int64{-60}},
		{name: "CrossingEpoch", from: time.Unix(-1, 0), to: time.Unix(1, 0), want: []int64{-60, 0}},
		{name: "ExactNegativeBoundary", from: time.Unix(-60, 0), to: time.Unix(-60, 0), want: []int64{-60}},
		{name: "ExactEpoch", from: time.Unix(0, 0), to: time.Unix(0, 0), want: []int64{0}},
		{name: "PositiveFraction", from: time.Unix(0, 500000000), to: time.Unix(60, 500000000), want: []int64{0, 60}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getAccessDataPointBuckets(tt.from, tt.to, time.Minute))
		})
	}
}

func TestAccessActivityRPCIntervalLimits(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	client := vaccessv1.NewResourceServiceClient(newRscStoreRPCClient(t, env))
	for _, tt := range []struct {
		name     string
		to       int64
		interval *metav1.Duration
	}{
		{name: "Overflow", to: 60, interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 307445735}}},
		{name: "Truncation", to: 1500, interval: &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 1999}}},
		{name: "InclusiveBuckets", to: 1000, interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 1}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res, err := client.GetRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
				From: timestamppb.New(time.Unix(0, 0)), To: timestamppb.New(time.Unix(tt.to, 0)), Interval: tt.interval,
			})
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, res)
		})
	}
}

func TestAccessActivityPreEpochRequests(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	for _, timestamp := range []int64{-120, -119, -61, -1, 0, 60, 61} {
		insertRscStoreObject(t, env, newTstAccessRequest(&tstAccessRequestOpts{
			name: fmt.Sprintf("epoch-%d", timestamp), createdAt: time.Unix(timestamp, 0).UTC(),
			state: accessv1.Request_Status_State_APPROVED,
		}))
	}
	client := vaccessv1.NewResourceServiceClient(newRscStoreRPCClient(t, env))
	res, err := client.GetRequestDataPoint(env.ctx, &vaccessv1.GetRequestDataPointRequest{
		From: timestamppb.New(time.Unix(-119, 0)), To: timestamppb.New(time.Unix(60, 0)),
		Interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 1}},
		GroupBy:  vaccessv1.GetRequestDataPointRequest_STATE,
	})
	require.Nil(t, err)
	require.Len(t, res.Datapoints, 4)
	for i, count := range []int64{2, 1, 1, 1} {
		assert.Equal(t, count, res.Datapoints[i].Count)
		assert.Equal(t, int64(-120+i*60), res.Datapoints[i].Timestamp.Seconds)
	}
	require.Len(t, res.Series, 1)
	assert.Equal(t, "APPROVED", res.Series[0].Key)
	assert.EqualValues(t, 5, res.Series[0].Total)
	require.Len(t, res.Series[0].Datapoints, len(res.Datapoints))
	for i, point := range res.Datapoints {
		assert.True(t, proto.Equal(point, res.Series[0].Datapoints[i]))
	}
}
