// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type richAccessLogOptions struct {
	CreatedAt   time.Time
	Duration    time.Duration
	Status      corev1.AccessLog_Entry_Common_Status
	Mode        corev1.Service_Spec_Mode
	Reason      corev1.AccessLog_Entry_Common_Reason_Type
	PolicyRef   *metav1.ObjectReference
	IsPublic    bool
	IsAnonymous bool
	SentBytes   uint64
	RecvBytes   uint64
}

func newRichAccessLog(opts *richAccessLogOptions) *corev1.AccessLog {
	common := &corev1.AccessLog_Entry_Common{
		StartedAt:   pbutils.Timestamp(opts.CreatedAt.Add(-opts.Duration)),
		EndedAt:     pbutils.Timestamp(opts.CreatedAt),
		Status:      opts.Status,
		Mode:        opts.Mode,
		IsPublic:    opts.IsPublic,
		IsAnonymous: opts.IsAnonymous,
	}

	if opts.Reason != corev1.AccessLog_Entry_Common_Reason_TYPE_UNKNOWN_REASON {
		common.Reason = &corev1.AccessLog_Entry_Common_Reason{
			Type: opts.Reason,
		}

		if opts.PolicyRef != nil {
			common.Reason.Details = &corev1.AccessLog_Entry_Common_Reason_Details{
				Type: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch_{
					PolicyMatch: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch{
						Type: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch_Policy_{
							Policy: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch_Policy{
								PolicyRef: opts.PolicyRef,
							},
						},
					},
				},
			}
		}
	}

	return &corev1.AccessLog{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindAccessLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: pbutils.Timestamp(opts.CreatedAt),
		},
		Entry: &corev1.AccessLog_Entry{
			Common: common,
			Info: &corev1.AccessLog_Entry_Info{
				Type: &corev1.AccessLog_Entry_Info_Tcp{
					Tcp: &corev1.AccessLog_Entry_Info_TCP{
						Type:          corev1.AccessLog_Entry_Info_TCP_END,
						SentBytes:     opts.SentBytes,
						ReceivedBytes: opts.RecvBytes,
					},
				},
			},
		},
	}
}

func newComponentLogFor(createdAt time.Time, level corev1.ComponentLog_Entry_Level,
	namespace, componentType string) *corev1.ComponentLog {
	return &corev1.ComponentLog{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindComponentLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: pbutils.Timestamp(createdAt),
		},
		Entry: &corev1.ComponentLog_Entry{
			Level:   level,
			Message: fmt.Sprintf("%s/%s", namespace, componentType),
			Time:    pbutils.Timestamp(createdAt),
			Component: &corev1.ComponentLog_Entry_Component{
				Namespace: namespace,
				Type:      componentType,
				Uid:       vutils.UUIDv4(),
			},
		},
	}
}

func newAuditLogFor(createdAt time.Time, method, kind string) *enterprisev1.AuditLog {
	return &enterprisev1.AuditLog{
		ApiVersion: uenterprisev1.APIVersion,
		Kind:       uenterprisev1.KindAuditLog,
		Metadata: &metav1.LogMetadata{
			Id:        vutils.GenerateLogID(),
			CreatedAt: pbutils.Timestamp(createdAt),
		},
		Entry: &enterprisev1.AuditLog_Entry{
			Method:      method,
			Service:     "MainService",
			Package:     "octelium.api.main.core.v1",
			Operation:   fmt.Sprintf("octelium.api.main.core.v1.MainService/%s", method),
			ResourceRef: &metav1.ObjectReference{Uid: vutils.UUIDv4(), Kind: kind},
		},
	}
}

func TestAccessLogSummaryComparisonAndBreakdowns(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	current := now.Add(-10 * time.Minute)
	previous := now.Add(-90 * time.Minute)

	policyRef := randomObjectReference()

	// 10 current entries: 6 allowed HTTP, 4 denied SSH.
	for idx := range 10 {
		opts := &richAccessLogOptions{
			CreatedAt: current.Add(time.Duration(idx) * time.Second),
			Duration:  time.Duration(idx+1) * 10 * time.Millisecond,
			Status:    corev1.AccessLog_Entry_Common_ALLOWED,
			Mode:      corev1.Service_Spec_HTTP,
			Reason:    corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH,
			PolicyRef: policyRef,
			SentBytes: 1000,
			RecvBytes: 500,
		}

		if idx >= 6 {
			opts.Status = corev1.AccessLog_Entry_Common_DENIED
			opts.Mode = corev1.Service_Spec_SSH
			opts.Reason = corev1.AccessLog_Entry_Common_Reason_NO_POLICY_MATCH
			opts.PolicyRef = nil
			opts.IsPublic = true
		}

		insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newRichAccessLog(opts)))
	}

	// 3 entries in the preceding window.
	for idx := range 3 {
		insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newRichAccessLog(&richAccessLogOptions{
			CreatedAt: previous.Add(time.Duration(idx) * time.Second),
			Duration:  time.Second,
			Status:    corev1.AccessLog_Entry_Common_ALLOWED,
			Mode:      corev1.Service_Spec_HTTP,
		})))
	}

	{
		resp, err := ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
			From: pbutils.Timestamp(now.Add(-60 * time.Minute)),
			To:   pbutils.Timestamp(now),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(10), resp.TotalNumber)
		assert.Equal(t, uint64(6), resp.TotalAllowed)
		assert.Equal(t, uint64(4), resp.TotalDenied)
		assert.Equal(t, uint64(4), resp.TotalPublic)
		assert.Equal(t, uint64(0), resp.TotalAnonymous)
		assert.Equal(t, uint64(10*1000), resp.TotalBytesSent)
		assert.Equal(t, uint64(10*500), resp.TotalBytesReceived)
		assert.Nil(t, resp.Previous)

		assert.NotNil(t, resp.Latency)
		assert.Equal(t, uint64(10), resp.Latency.Count)
		assert.Equal(t, float64(100), resp.Latency.MaxMilliseconds)
		assert.True(t, resp.Latency.P95Milliseconds >= resp.Latency.P50Milliseconds)

		assert.Equal(t, uint64(6), resp.TotalByMode["HTTP"])
		assert.Equal(t, uint64(4), resp.TotalByMode["SSH"])
	}

	{
		resp, err := ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
			From:        pbutils.Timestamp(now.Add(-60 * time.Minute)),
			To:          pbutils.Timestamp(now),
			CompareFrom: pbutils.Timestamp(now.Add(-120 * time.Minute)),
			CompareTo:   pbutils.Timestamp(now.Add(-60 * time.Minute)),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(10), resp.TotalNumber)
		assert.NotNil(t, resp.Previous)
		assert.Equal(t, uint64(3), resp.Previous.TotalNumber)
		assert.Nil(t, resp.Previous.Previous)
	}

	{
		resp, err := ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
			From:     pbutils.Timestamp(now.Add(-60 * time.Minute)),
			To:       pbutils.Timestamp(now),
			IsPublic: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(4), resp.TotalNumber)
	}

	{
		resp, err := ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
			From:   pbutils.Timestamp(now.Add(-60 * time.Minute)),
			To:     pbutils.Timestamp(now),
			Status: corev1.AccessLog_Entry_Common_DENIED,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(4), resp.TotalNumber)
		assert.Equal(t, uint64(4), resp.TotalDenied)
		assert.Equal(t, uint64(0), resp.TotalAllowed)
	}

	{
		resp, err := ts.srv.getAccessLogDataPoint(ts.ctx, &visibilityv1.GetAccessLogDataPointRequest{
			From:     pbutils.Timestamp(now.Add(-60 * time.Minute)),
			To:       pbutils.Timestamp(now),
			Interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 5}},
			GroupBy:  visibilityv1.GetAccessLogDataPointRequest_STATUS,
		})
		assert.Nil(t, err, "%+v", err)
		assert.True(t, len(resp.Datapoints) > 0)
		assert.Equal(t, 2, len(resp.Series))

		byKey := make(map[string]int64)
		for _, series := range resp.Series {
			byKey[series.Key] = series.Total
			assert.Equal(t, len(resp.Datapoints), len(series.Datapoints))
		}
		assert.Equal(t, int64(6), byKey["ALLOWED"])
		assert.Equal(t, int64(4), byKey["DENIED"])
	}

	{
		resp, err := ts.srv.getAccessLogDataPoint(ts.ctx, &visibilityv1.GetAccessLogDataPointRequest{
			From:     pbutils.Timestamp(now.Add(-60 * time.Minute)),
			To:       pbutils.Timestamp(now),
			Interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 5}},
			GroupBy:  visibilityv1.GetAccessLogDataPointRequest_MODE,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Series))
	}

	{
		resp, err := ts.srv.listAccessLogTopDenyReason(ts.ctx,
			&visibilityv1.ListAccessLogTopDenyReasonRequest{
				From: pbutils.Timestamp(now.Add(-60 * time.Minute)),
				To:   pbutils.Timestamp(now),
			})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, corev1.AccessLog_Entry_Common_Reason_NO_POLICY_MATCH, resp.Items[0].Reason)
		assert.Equal(t, uint64(4), resp.Items[0].Count)
		assert.Equal(t, uint64(1), resp.TotalCount)
		assert.Equal(t, uint64(0), resp.TotalOther)
	}

	{
		resp, err := ts.srv.listAccessLogTopUser(ts.ctx, &visibilityv1.ListAccessLogTopUserRequest{
			From:  pbutils.Timestamp(now.Add(-60 * time.Minute)),
			To:    pbutils.Timestamp(now),
			Limit: 1,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(0), resp.TotalCount)
	}
}

func TestComponentAndAuditLogBreakdowns(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	current := now.Add(-10 * time.Minute)

	for idx := range 5 {
		insertLogJSON(t, ts.srv, "component_logs", marshalLog(t,
			newComponentLogFor(current.Add(time.Duration(idx)*time.Second),
				corev1.ComponentLog_Entry_ERROR, "octelium", "vigil")))
	}
	for idx := range 3 {
		insertLogJSON(t, ts.srv, "component_logs", marshalLog(t,
			newComponentLogFor(current.Add(time.Duration(idx)*time.Second),
				corev1.ComponentLog_Entry_INFO, "octelium", "apiserver")))
	}

	{
		resp, err := ts.srv.getSummaryComponentLog(ts.ctx,
			&visibilityv1.GetComponentLogSummaryRequest{
				From: pbutils.Timestamp(now.Add(-60 * time.Minute)),
				To:   pbutils.Timestamp(now),
			})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(8), resp.TotalNumber)
		assert.Equal(t, uint64(5), resp.TotalError)
		assert.Equal(t, uint64(2), resp.TotalComponent)
	}

	{
		resp, err := ts.srv.getSummaryComponentLog(ts.ctx,
			&visibilityv1.GetComponentLogSummaryRequest{
				From:      pbutils.Timestamp(now.Add(-60 * time.Minute)),
				To:        pbutils.Timestamp(now),
				Component: &visibilityv1.ComponentSelector{Type: "vigil"},
			})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(5), resp.TotalNumber)
		assert.Equal(t, uint64(1), resp.TotalComponent)
	}

	{
		resp, err := ts.srv.listComponentLogTopComponent(ts.ctx,
			&visibilityv1.ListComponentLogTopComponentRequest{
				From: pbutils.Timestamp(now.Add(-60 * time.Minute)),
				To:   pbutils.Timestamp(now),
			})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Items))
		assert.Equal(t, "vigil", resp.Items[0].Component.Type)
		assert.Equal(t, "octelium", resp.Items[0].Component.Namespace)
		assert.Equal(t, uint64(5), resp.Items[0].Count)
		assert.Equal(t, uint64(5), resp.Items[0].CountError)
		assert.Equal(t, uint64(2), resp.TotalCount)
	}

	{
		resp, err := ts.srv.getComponentLogDataPoint(ts.ctx,
			&visibilityv1.GetComponentLogDataPointRequest{
				From:     pbutils.Timestamp(now.Add(-60 * time.Minute)),
				To:       pbutils.Timestamp(now),
				Interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 5}},
				GroupBy:  visibilityv1.GetComponentLogDataPointRequest_COMPONENT_TYPE,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Series))
	}

	for _, method := range []string{"CreateUser", "CreateUser", "UpdateService", "DeleteService", "GetUser"} {
		kind := "User"
		if method == "UpdateService" || method == "DeleteService" {
			kind = "Service"
		}
		insertLogJSON(t, ts.srv, "audit_logs", marshalLog(t,
			newAuditLogFor(current, method, kind)))
	}

	{
		resp, err := ts.srv.getSummaryAuditLog(ts.ctx, &visibilityv1.GetAuditLogSummaryRequest{
			From: pbutils.Timestamp(now.Add(-60 * time.Minute)),
			To:   pbutils.Timestamp(now),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(5), resp.TotalNumber)
		assert.Equal(t, uint64(2), resp.TotalCreate)
		assert.Equal(t, uint64(1), resp.TotalUpdate)
		assert.Equal(t, uint64(1), resp.TotalDelete)
		assert.Equal(t, uint64(1), resp.TotalOther)
		assert.Equal(t, uint64(3), resp.TotalByResourceKind["User"])
		assert.Equal(t, uint64(2), resp.TotalByResourceKind["Service"])
	}

	{
		resp, err := ts.srv.getAuditLogDataPoint(ts.ctx, &visibilityv1.GetAuditLogDataPointRequest{
			From:     pbutils.Timestamp(now.Add(-60 * time.Minute)),
			To:       pbutils.Timestamp(now),
			Interval: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 5}},
			GroupBy:  visibilityv1.GetAuditLogDataPointRequest_ACTION,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 4, len(resp.Series))
	}
}

func TestComponentLogTopComponentOrderBy(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	current := now.Add(-10 * time.Minute)

	for idx := range 6 {
		insertLogJSON(t, ts.srv, "component_logs", marshalLog(t,
			newComponentLogFor(current.Add(time.Duration(idx)*time.Second),
				corev1.ComponentLog_Entry_INFO, "octelium", "apiserver")))
	}
	for idx := range 2 {
		insertLogJSON(t, ts.srv, "component_logs", marshalLog(t,
			newComponentLogFor(current.Add(time.Duration(idx)*time.Second),
				corev1.ComponentLog_Entry_WARN, "octelium", "nocturne")))
	}
	insertLogJSON(t, ts.srv, "component_logs", marshalLog(t,
		newComponentLogFor(current, corev1.ComponentLog_Entry_FATAL, "octelium", "vigil")))

	types := func(resp *visibilityv1.ListComponentLogTopComponentResponse) []string {
		var ret []string
		for _, item := range resp.Items {
			ret = append(ret, item.Component.Type)
		}
		return ret
	}

	{
		resp, err := ts.srv.listComponentLogTopComponent(ts.ctx,
			&visibilityv1.ListComponentLogTopComponentRequest{
				From: pbutils.Timestamp(now.Add(-60 * time.Minute)),
				To:   pbutils.Timestamp(now),
			})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"apiserver", "nocturne", "vigil"}, types(resp))
	}

	{
		resp, err := ts.srv.listComponentLogTopComponent(ts.ctx,
			&visibilityv1.ListComponentLogTopComponentRequest{
				From:    pbutils.Timestamp(now.Add(-60 * time.Minute)),
				To:      pbutils.Timestamp(now),
				OrderBy: visibilityv1.ListComponentLogTopComponentRequest_SEVERITY,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"vigil", "nocturne", "apiserver"}, types(resp))
	}

	{
		resp, err := ts.srv.listComponentLogTopComponent(ts.ctx,
			&visibilityv1.ListComponentLogTopComponentRequest{
				From:    pbutils.Timestamp(now.Add(-60 * time.Minute)),
				To:      pbutils.Timestamp(now),
				OrderBy: visibilityv1.ListComponentLogTopComponentRequest_SEVERITY,
				Limit:   1,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"vigil"}, types(resp))
		assert.Equal(t, uint64(1), resp.Items[0].CountFatal)
		assert.Equal(t, uint64(3), resp.TotalCount)
		assert.Equal(t, uint64(8), resp.TotalOther)
	}
}

func TestAccessLogReasonAndModeFilters(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	current := now.Add(-10 * time.Minute)

	for _, arg := range []struct {
		count  int
		status corev1.AccessLog_Entry_Common_Status
		reason corev1.AccessLog_Entry_Common_Reason_Type
		mode   corev1.Service_Spec_Mode
	}{
		{3, corev1.AccessLog_Entry_Common_DENIED, corev1.AccessLog_Entry_Common_Reason_NO_POLICY_MATCH, corev1.Service_Spec_SSH},
		{2, corev1.AccessLog_Entry_Common_DENIED, corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH, corev1.Service_Spec_HTTP},
		{4, corev1.AccessLog_Entry_Common_ALLOWED, corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH, corev1.Service_Spec_HTTP},
	} {
		for idx := range arg.count {
			insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newRichAccessLog(&richAccessLogOptions{
				CreatedAt: current.Add(time.Duration(idx) * time.Second),
				Duration:  time.Second,
				Status:    arg.status,
				Reason:    arg.reason,
				Mode:      arg.mode,
			})))
		}
	}

	from := pbutils.Timestamp(now.Add(-60 * time.Minute))
	to := pbutils.Timestamp(now)

	for _, tc := range []struct {
		reason corev1.AccessLog_Entry_Common_Reason_Type
		mode   corev1.Service_Spec_Mode
		status corev1.AccessLog_Entry_Common_Status
		total  uint32
	}{
		{total: 9},
		{reason: corev1.AccessLog_Entry_Common_Reason_NO_POLICY_MATCH, total: 3},
		{reason: corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH, total: 6},
		{reason: corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH, status: corev1.AccessLog_Entry_Common_DENIED, total: 2},
		{mode: corev1.Service_Spec_HTTP, total: 6},
		{mode: corev1.Service_Spec_SSH, reason: corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH, total: 0},
	} {
		{
			resp, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
				From:   from,
				To:     to,
				Status: tc.status,
				Reason: tc.reason,
				Mode:   tc.mode,
			})
			if assert.Nil(t, err, "%+v", err) {
				assert.Equal(t, tc.total, resp.ListResponseMeta.TotalCount)
				assert.Len(t, resp.Items, int(tc.total))
			}
		}

		{
			resp, err := ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
				From:   from,
				To:     to,
				Status: tc.status,
				Reason: tc.reason,
				Mode:   tc.mode,
			})
			if assert.Nil(t, err, "%+v", err) {
				assert.Equal(t, uint64(tc.total), resp.TotalNumber)
			}
		}

		{
			resp, err := ts.srv.getAccessLogDataPoint(ts.ctx, &visibilityv1.GetAccessLogDataPointRequest{
				From:   from,
				To:     to,
				Status: tc.status,
				Reason: tc.reason,
				Mode:   tc.mode,
			})
			if assert.Nil(t, err, "%+v", err) {
				var total int64
				for _, dp := range resp.Datapoints {
					total += dp.Count
				}
				assert.Equal(t, int64(tc.total), total)
			}
		}
	}

	{
		resp, err := ts.srv.listAccessLogTopDenyReason(ts.ctx, &visibilityv1.ListAccessLogTopDenyReasonRequest{
			From: from,
			To:   to,
			Mode: corev1.Service_Spec_SSH,
		})
		assert.Nil(t, err, "%+v", err)
		if assert.Len(t, resp.Items, 1) {
			assert.Equal(t, corev1.AccessLog_Entry_Common_Reason_NO_POLICY_MATCH, resp.Items[0].Reason)
			assert.Equal(t, uint64(3), resp.Items[0].Count)
		}
	}
}

func TestComponentLogLevelsFilter(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)
	current := now.Add(-10 * time.Minute)

	for _, arg := range []struct {
		count         int
		level         corev1.ComponentLog_Entry_Level
		componentType string
	}{
		{5, corev1.ComponentLog_Entry_INFO, "apiserver"},
		{2, corev1.ComponentLog_Entry_ERROR, "apiserver"},
		{1, corev1.ComponentLog_Entry_PANIC, "nocturne"},
		{1, corev1.ComponentLog_Entry_FATAL, "vigil"},
	} {
		for idx := range arg.count {
			insertLogJSON(t, ts.srv, "component_logs", marshalLog(t,
				newComponentLogFor(current.Add(time.Duration(idx)*time.Second),
					arg.level, "octelium", arg.componentType)))
		}
	}

	from := pbutils.Timestamp(now.Add(-60 * time.Minute))
	to := pbutils.Timestamp(now)

	severe := []corev1.ComponentLog_Entry_Level{
		corev1.ComponentLog_Entry_ERROR,
		corev1.ComponentLog_Entry_PANIC,
		corev1.ComponentLog_Entry_FATAL,
	}

	for _, tc := range []struct {
		level  corev1.ComponentLog_Entry_Level
		levels []corev1.ComponentLog_Entry_Level
		total  uint32
		types  []string
	}{
		{total: 9, types: []string{"apiserver", "nocturne", "vigil"}},
		{levels: severe, total: 4, types: []string{"apiserver", "nocturne", "vigil"}},
		{level: corev1.ComponentLog_Entry_PANIC, levels: []corev1.ComponentLog_Entry_Level{
			corev1.ComponentLog_Entry_FATAL,
		}, total: 2, types: []string{"nocturne", "vigil"}},
		{level: corev1.ComponentLog_Entry_ERROR, total: 2, types: []string{"apiserver"}},
		{levels: []corev1.ComponentLog_Entry_Level{corev1.ComponentLog_Entry_WARN}, total: 0},
	} {
		{
			resp, err := ts.srv.listComponentLog(ts.ctx, &visibilityv1.ListComponentLogRequest{
				From:   from,
				To:     to,
				Level:  tc.level,
				Levels: tc.levels,
			})
			if assert.Nil(t, err, "%+v", err) {
				assert.Equal(t, tc.total, resp.ListResponseMeta.TotalCount)
			}
		}

		{
			resp, err := ts.srv.getComponentLogDataPoint(ts.ctx, &visibilityv1.GetComponentLogDataPointRequest{
				From:   from,
				To:     to,
				Level:  tc.level,
				Levels: tc.levels,
			})
			if assert.Nil(t, err, "%+v", err) {
				var total int64
				for _, dp := range resp.Datapoints {
					total += dp.Count
				}
				assert.Equal(t, int64(tc.total), total)
			}
		}

		{
			resp, err := ts.srv.listComponentLogTopComponent(ts.ctx, &visibilityv1.ListComponentLogTopComponentRequest{
				From:    from,
				To:      to,
				Level:   tc.level,
				Levels:  tc.levels,
				OrderBy: visibilityv1.ListComponentLogTopComponentRequest_SEVERITY,
			})
			if assert.Nil(t, err, "%+v", err) {
				var types []string
				for _, item := range resp.Items {
					types = append(types, item.Component.Type)
				}
				assert.ElementsMatch(t, tc.types, types)
			}
		}
	}

	{
		_, err := ts.srv.listComponentLog(ts.ctx, &visibilityv1.ListComponentLogRequest{
			Levels: make([]corev1.ComponentLog_Entry_Level, maxComponentLevels+1),
		})
		assert.NotNil(t, err)
	}
}

func TestAccessLogRPCFlagAndInlinePolicyFilters(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	client := visibilityv1.NewAccessLogServiceClient(newLogStoreRPCClient(t, ts))
	ref := randomObjectReference()
	ref.Name = "namespace.service"
	otherRef := randomObjectReference()
	now := time.Now().UTC()

	var logs []*corev1.AccessLog
	for i, opts := range []struct {
		public    bool
		anonymous bool
		inline    *metav1.ObjectReference
		policy    *metav1.ObjectReference
	}{
		{},
		{public: true, inline: ref},
		{anonymous: true, inline: otherRef},
		{public: true, anonymous: true, inline: ref},
		{public: true, anonymous: true, policy: ref},
	} {
		log := newAccessLog(&accessLogOptions{CreatedAt: now.Add(time.Duration(i) * time.Second), PolicyRef: opts.policy})
		log.Entry.Common.IsPublic = opts.public
		log.Entry.Common.IsAnonymous = opts.anonymous
		if opts.inline != nil {
			log.Entry.Common.Reason = &corev1.AccessLog_Entry_Common_Reason{
				Type: corev1.AccessLog_Entry_Common_Reason_POLICY_MATCH,
				Details: &corev1.AccessLog_Entry_Common_Reason_Details{
					Type: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch_{
						PolicyMatch: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch{
							Type: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch_InlinePolicy_{
								InlinePolicy: &corev1.AccessLog_Entry_Common_Reason_Details_PolicyMatch_InlinePolicy{
									ResourceRef: opts.inline, Name: "allow",
								},
							},
						},
					},
				},
			}
		}
		insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, log))
		logs = append(logs, log)
	}

	for _, tc := range []struct {
		name string
		req  *visibilityv1.ListAccessLogRequest
		want []int
	}{
		{name: "All", req: &visibilityv1.ListAccessLogRequest{}, want: []int{0, 1, 2, 3, 4}},
		{name: "Public", req: &visibilityv1.ListAccessLogRequest{IsPublic: true}, want: []int{1, 3, 4}},
		{name: "Anonymous", req: &visibilityv1.ListAccessLogRequest{IsAnonymous: true}, want: []int{2, 3, 4}},
		{name: "BothFlags", req: &visibilityv1.ListAccessLogRequest{IsPublic: true, IsAnonymous: true}, want: []int{3, 4}},
		{name: "InlineUID", req: &visibilityv1.ListAccessLogRequest{MatchInlinePolicyRef: &metav1.ObjectReference{Uid: ref.Uid}}, want: []int{1, 3}},
		{name: "InlineName", req: &visibilityv1.ListAccessLogRequest{MatchInlinePolicyRef: &metav1.ObjectReference{Name: ref.Name}}, want: []int{1, 3}},
		{name: "InlineBoth", req: &visibilityv1.ListAccessLogRequest{MatchInlinePolicyRef: ref}, want: []int{1, 3}},
		{name: "InlineMismatch", req: &visibilityv1.ListAccessLogRequest{MatchInlinePolicyRef: &metav1.ObjectReference{Uid: ref.Uid, Name: otherRef.Name}}},
		{name: "InlineAndFlags", req: &visibilityv1.ListAccessLogRequest{MatchInlinePolicyRef: ref, IsPublic: true, IsAnonymous: true}, want: []int{3}},
		{name: "InlineAndTime", req: &visibilityv1.ListAccessLogRequest{MatchInlinePolicyRef: ref, From: pbutils.Timestamp(now.Add(2 * time.Second))}, want: []int{3}},
		{name: "StandalonePolicy", req: &visibilityv1.ListAccessLogRequest{PolicyRef: ref}, want: []int{4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := client.ListAccessLog(ts.ctx, tc.req)
			require.NoError(t, err)
			assert.EqualValues(t, len(tc.want), res.ListResponseMeta.TotalCount)
			var got, want []string
			for _, item := range res.Items {
				got = append(got, item.Metadata.Id)
			}
			for _, i := range tc.want {
				want = append(want, logs[i].Metadata.Id)
			}
			assert.ElementsMatch(t, want, got)
		})
	}

	res, err := client.ListAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
		IsPublic: true, Common: &vmetav1.CommonListOptions{ItemsPerPage: 2, Page: 1},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, res.ListResponseMeta.TotalCount)
	require.Len(t, res.Items, 1)
	assert.Equal(t, logs[1].Metadata.Id, res.Items[0].Metadata.Id)

	for _, ref := range []*metav1.ObjectReference{
		{},
		{Uid: "invalid"},
		{Name: "invalid/name"},
		{Name: strings.Repeat("parent.", 9) + "resource"},
	} {
		_, err := client.ListAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{MatchInlinePolicyRef: ref})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
}

func TestAuditLogRPCSummaryResourceFilter(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	client := visibilityv1.NewAuditLogServiceClient(newLogStoreRPCClient(t, ts))
	ref := randomObjectReference()
	ref.Name = "namespace.service"
	ref.Kind = "Service"
	otherRef := randomObjectReference()
	otherRef.Kind = "User"
	userRef, deviceRef, sessionRef := randomObjectReference(), randomObjectReference(), randomObjectReference()
	base := time.Now().UTC().Truncate(time.Hour)

	for _, entry := range []struct {
		at     time.Time
		ref    *metav1.ObjectReference
		method string
	}{
		{base.Add(10 * time.Minute), ref, "CreateService"},
		{base.Add(20 * time.Minute), ref, "UpdateService"},
		{base.Add(-90 * time.Minute), ref, "DeleteService"},
		{base.Add(30 * time.Minute), otherRef, "DeleteUser"},
		{base.Add(-100 * time.Minute), otherRef, "CreateUser"},
	} {
		log := newAuditLog(&auditLogOptions{
			CreatedAt: entry.at, ResourceRef: entry.ref,
			UserRef: userRef, DeviceRef: deviceRef, SessionRef: sessionRef,
		})
		log.Entry.Method = entry.method
		insertLogJSON(t, ts.srv, "audit_logs", marshalLog(t, log))
	}

	for _, ref := range []*metav1.ObjectReference{
		{Uid: ref.Uid},
		{Name: ref.Name},
		ref,
	} {
		res, err := client.GetAuditLogSummary(ts.ctx, &visibilityv1.GetAuditLogSummaryRequest{
			ResourceRef: ref, From: pbutils.Timestamp(base), To: pbutils.Timestamp(base.Add(time.Hour)),
			CompareFrom: pbutils.Timestamp(base.Add(-2 * time.Hour)), CompareTo: pbutils.Timestamp(base.Add(-time.Hour)),
		})
		require.NoError(t, err)
		assert.EqualValues(t, 2, res.TotalNumber)
		assert.EqualValues(t, 1, res.TotalResource)
		assert.EqualValues(t, 1, res.TotalUser)
		assert.EqualValues(t, 1, res.TotalDevice)
		assert.EqualValues(t, 1, res.TotalSession)
		assert.EqualValues(t, 1, res.TotalCreate)
		assert.EqualValues(t, 1, res.TotalUpdate)
		assert.Zero(t, res.TotalDelete)
		assert.Zero(t, res.TotalOther)
		assert.Equal(t, map[string]uint64{"Service": 2}, res.TotalByResourceKind)
		require.NotNil(t, res.Previous)
		assert.EqualValues(t, 1, res.Previous.TotalNumber)
		assert.EqualValues(t, 1, res.Previous.TotalDelete)
		assert.Equal(t, map[string]uint64{"Service": 1}, res.Previous.TotalByResourceKind)
	}

	res, err := client.GetAuditLogSummary(ts.ctx, &visibilityv1.GetAuditLogSummaryRequest{
		ResourceRef: &metav1.ObjectReference{Uid: ref.Uid, Name: otherRef.Name},
	})
	require.NoError(t, err)
	assert.Zero(t, res.TotalNumber)
	assert.Zero(t, res.TotalResource)
	assert.Empty(t, res.TotalByResourceKind)

	for _, ref := range []*metav1.ObjectReference{
		{},
		{Uid: "invalid"},
		{Name: "invalid/name"},
		{Name: strings.Repeat("parent.", 9) + "resource"},
	} {
		_, err := client.GetAuditLogSummary(ts.ctx, &visibilityv1.GetAuditLogSummaryRequest{ResourceRef: ref})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
}
