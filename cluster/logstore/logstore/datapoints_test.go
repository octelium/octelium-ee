// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
)

func TestDataPointIntervalAndGapHelpers(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	assert.Equal(t, &intervalDataPoint{Unit: "minute", Value: 1}, ts.srv.getDataPointInterval(nil))
	assert.Equal(t, &intervalDataPoint{Unit: "second", Value: 15}, ts.srv.getDataPointInterval(&metav1.Duration{
		Type: &metav1.Duration_Seconds{Seconds: 15},
	}))
	assert.Equal(t, &intervalDataPoint{Unit: "minute", Value: 5}, ts.srv.getDataPointInterval(&metav1.Duration{
		Type: &metav1.Duration_Minutes{Minutes: 5},
	}))
	assert.Equal(t, &intervalDataPoint{Unit: "hour", Value: 2}, ts.srv.getDataPointInterval(&metav1.Duration{
		Type: &metav1.Duration_Hours{Hours: 2},
	}))
	assert.Equal(t, &intervalDataPoint{Unit: "day", Value: 3}, ts.srv.getDataPointInterval(&metav1.Duration{
		Type: &metav1.Duration_Days{Days: 3},
	}))
	assert.Equal(t, &intervalDataPoint{Unit: "day", Value: 14}, ts.srv.getDataPointInterval(&metav1.Duration{
		Type: &metav1.Duration_Weeks{Weeks: 2},
	}))
	assert.Equal(t, &intervalDataPoint{Unit: "day", Value: 60}, ts.srv.getDataPointInterval(&metav1.Duration{
		Type: &metav1.Duration_Months{Months: 2},
	}))

	from := time.Date(2026, time.July, 29, 10, 7, 0, 0, time.UTC)
	to := from.Add(30 * time.Minute)
	interval := &intervalDataPoint{Unit: "minute", Value: 10}

	ret := fillGaps([]DataPoint{
		{
			Timestamp: time.Date(2026, time.July, 29, 10, 10, 0, 0, time.UTC).Format(time.RFC3339),
			Count:     3,
		},
		{
			Timestamp: time.Date(2026, time.July, 29, 10, 30, 0, 0, time.UTC).Format(time.RFC3339),
			Count:     2,
		},
	}, from, to, interval)

	assert.Len(t, ret, 4)
	assert.Equal(t, int64(0), ret[0].Count)
	assert.Equal(t, int64(3), ret[1].Count)
	assert.Equal(t, int64(0), ret[2].Count)
	assert.Equal(t, int64(2), ret[3].Count)
}

func TestAllLogDataPointQueries(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	base := time.Now().UTC().Add(-2 * time.Hour).Truncate(30 * time.Minute)
	userRef := randomObjectReference()
	otherUserRef := randomObjectReference()

	for idx := range 12 {
		createdAt := base.Add(time.Duration(idx) * 10 * time.Minute)
		currentUserRef := userRef
		if idx%2 == 0 {
			currentUserRef = otherUserRef
		}

		insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{
			CreatedAt: createdAt,
			Status:    corev1.AccessLog_Entry_Common_ALLOWED,
			UserRef:   currentUserRef,
		})))

		insertLogJSON(t, ts.srv, "authentication_logs", marshalLog(t, newAuthenticationLog(&authenticationLogOptions{
			CreatedAt: createdAt,
			UserRef:   currentUserRef,
			Type:      corev1.Session_Status_Authentication_Info_CREDENTIAL,
			AAL:       corev1.Session_Status_Authentication_Info_AAL1,
		})))

		insertLogJSON(t, ts.srv, "audit_logs", marshalLog(t, newAuditLog(&auditLogOptions{
			CreatedAt: createdAt,
			UserRef:   currentUserRef,
		})))

		insertLogJSON(t, ts.srv, "component_logs", marshalLog(t, newComponentLog(createdAt, corev1.ComponentLog_Entry_INFO, "test")))
	}

	interval := &metav1.Duration{
		Type: &metav1.Duration_Minutes{Minutes: 30},
	}
	from := pbutils.Timestamp(base)
	to := pbutils.Timestamp(base.Add(2 * time.Hour))

	{
		resp, err := ts.srv.getAccessLogDataPoint(ts.ctx, &visibilityv1.GetAccessLogDataPointRequest{
			From:     from,
			To:       to,
			Interval: interval,
			UserRef:  userRef,
			Status:   corev1.AccessLog_Entry_Common_ALLOWED,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Datapoints, 4)

		var total int64
		for _, item := range resp.Datapoints {
			total += item.Count
		}
		assert.Equal(t, int64(6), total)
	}

	{
		resp, err := ts.srv.getAuthenticationLogDataPoint(ts.ctx, &visibilityv1.GetAuthenticationLogDataPointRequest{
			From:     from,
			To:       to,
			Interval: interval,
			UserRef:  userRef,
		})
		assert.Nil(t, err, "%+v", err)

		var total int64
		for _, item := range resp.Datapoints {
			total += item.Count
		}
		assert.Equal(t, int64(6), total)
	}

	{
		resp, err := ts.srv.getAuditLogDataPoint(ts.ctx, &visibilityv1.GetAuditLogDataPointRequest{
			From:     from,
			To:       to,
			Interval: interval,
			UserRef:  userRef,
		})
		assert.Nil(t, err, "%+v", err)

		var total int64
		for _, item := range resp.Datapoints {
			total += item.Count
		}
		assert.Equal(t, int64(6), total)
	}

	{
		resp, err := ts.srv.getComponentLogDataPoint(ts.ctx, &visibilityv1.GetComponentLogDataPointRequest{
			From:     from,
			To:       to,
			Interval: interval,
			Level:    corev1.ComponentLog_Entry_INFO,
		})
		assert.Nil(t, err, "%+v", err)

		var total int64
		for _, item := range resp.Datapoints {
			total += item.Count
		}
		assert.Equal(t, int64(12), total)
	}

	{
		_, err := ts.srv.getDataPoints(ts.ctx, "access_logs", base.Add(time.Hour), base, &intervalDataPoint{Unit: "minute", Value: 1}, nil)
		assert.NotNil(t, err)
	}
}
