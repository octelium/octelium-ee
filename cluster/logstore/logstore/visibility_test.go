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
	"testing"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
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
