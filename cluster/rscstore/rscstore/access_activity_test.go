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
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
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
