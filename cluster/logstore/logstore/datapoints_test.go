// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
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

func TestLogDataPointRPCAllGroups(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	conn := newLogStoreRPCClient(t, ts)
	base := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	ref := randomObjectReference()
	for i := range 2 {
		at := base.Add(time.Duration(i)*time.Minute + 10*time.Second)
		insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{
			CreatedAt: at, UserRef: ref, DeviceRef: ref, SessionRef: ref, ServiceRef: ref, NamespaceRef: ref, RegionRef: ref, PolicyRef: ref,
		})))
		insertLogJSON(t, ts.srv, "authentication_logs", marshalLog(t, newAuthenticationLog(&authenticationLogOptions{
			CreatedAt: at, UserRef: ref, DeviceRef: ref, SessionRef: ref, CredentialRef: ref,
			Type: corev1.Session_Status_Authentication_Info_CREDENTIAL, AAL: corev1.Session_Status_Authentication_Info_AAL1,
		})))
		insertLogJSON(t, ts.srv, "audit_logs", marshalLog(t, newAuditLog(&auditLogOptions{
			CreatedAt: at, UserRef: ref, DeviceRef: ref, SessionRef: ref, ResourceRef: ref, Operation: "core/MainService/CreateUser",
		})))
		insertLogJSON(t, ts.srv, "component_logs", marshalLog(t, newComponentLog(at, corev1.ComponentLog_Entry_INFO, "grouped")))
	}
	for _, method := range logStoreRPCMethods() {
		if method.Name() != "GetAccessLogDataPoint" && method.Name() != "GetAuthenticationLogDataPoint" &&
			method.Name() != "GetAuditLogDataPoint" && method.Name() != "GetComponentLogDataPoint" {
			continue
		}
		request := newLogStoreRPCRequest(method)
		fields := request.Descriptor().Fields()
		request.Set(fields.ByName("from"), protoreflect.ValueOfMessage(pbutils.Timestamp(base).ProtoReflect()))
		request.Set(fields.ByName("to"), protoreflect.ValueOfMessage(pbutils.Timestamp(base.Add(2*time.Minute)).ProtoReflect()))
		request.Set(fields.ByName("interval"), protoreflect.ValueOfMessage((&metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 1}}).ProtoReflect()))
		group := fields.ByName("groupBy")
		for i := range group.Enum().Values().Len() {
			value := group.Enum().Values().Get(i)
			t.Run(string(method.Name())+"/"+string(value.Name()), func(t *testing.T) {
				request.Set(group, protoreflect.ValueOfEnum(value.Number()))
				res := dynamicpb.NewMessage(method.Output())
				require.Nil(t, conn.Invoke(ts.ctx, logStoreRPCName(method), request, res))
				points := res.Get(res.Descriptor().Fields().ByName("datapoints")).List()
				require.Equal(t, 2, points.Len())
				for j := range points.Len() {
					point := points.Get(j).Message()
					assert.Equal(t, int64(1), point.Get(point.Descriptor().Fields().ByName("count")).Int())
				}
				series := res.Get(res.Descriptor().Fields().ByName("series")).List()
				if value.Number() == 0 {
					assert.Zero(t, series.Len())
				}
				for j := range series.Len() {
					item := series.Get(j).Message()
					assert.NotEmpty(t, item.Get(item.Descriptor().Fields().ByName("key")).String())
					assert.Equal(t, int64(2), item.Get(item.Descriptor().Fields().ByName("total")).Int())
					assert.Equal(t, 2, item.Get(item.Descriptor().Fields().ByName("datapoints")).List().Len())
				}
			})
		}
	}
}

func TestDataPointGapFillingBoundaries(t *testing.T) {
	base := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		interval *intervalDataPoint
		width    time.Duration
	}{
		{interval: &intervalDataPoint{Unit: "second", Value: 15}, width: 15 * time.Second},
		{interval: &intervalDataPoint{Unit: "minute", Value: 5}, width: 5 * time.Minute},
		{interval: &intervalDataPoint{Unit: "hour", Value: 2}, width: 2 * time.Hour},
		{interval: &intervalDataPoint{Unit: "day", Value: 1}, width: 24 * time.Hour},
	} {
		t.Run(tt.interval.Unit, func(t *testing.T) {
			from := alignTimeToBucket(base, tt.interval)
			to := from.Add(3 * tt.width)
			points := []DataPoint{{Timestamp: from.Add(tt.width).Format(time.RFC3339), Count: 7}}
			filled := fillGaps(points, from, to, tt.interval)
			require.Len(t, filled, 3)
			for i, count := range []int64{0, 7, 0} {
				assert.Equal(t, count, filled[i].Count)
				assert.Equal(t, from.Add(time.Duration(i)*tt.width).Format(time.RFC3339), filled[i].Timestamp)
			}
			assert.Len(t, generateEmptyDataPoints(from, to, tt.interval), 3)
			assert.Empty(t, fillGaps(nil, from, from, tt.interval))
			assert.Empty(t, fillGaps(nil, to, from, tt.interval))
			assert.Equal(t, int64(7), points[0].Count)
		})
	}
}

func TestLogSeriesDisplayNameResolution(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)
	client := reflect.ValueOf(ts.srv.octeliumC.CoreC())
	for _, kind := range []string{"User", "Service", "Session", "Device", "Namespace", "Region", "Policy", "IdentityProvider", "Credential", "Authenticator"} {
		t.Run(kind, func(t *testing.T) {
			item, err := ucorev1.NewObject(kind)
			require.Nil(t, err)
			msg := item.ProtoReflect()
			fields := msg.Descriptor().Fields()
			msg.Set(fields.ByName("apiVersion"), protoreflect.ValueOfString(ucorev1.APIVersion))
			msg.Set(fields.ByName("kind"), protoreflect.ValueOfString(kind))
			msg.Set(fields.ByName("metadata"), protoreflect.ValueOfMessage((&metav1.Metadata{Name: "series-" + strings.ToLower(kind) + "-" + utilrand.GetRandomStringCanonical(8)}).ProtoReflect()))
			msg.Mutable(fields.ByName("spec"))
			msg.Mutable(fields.ByName("status"))
			result := client.MethodByName("Create" + kind).Call([]reflect.Value{reflect.ValueOf(ts.ctx), reflect.ValueOf(item)})
			require.True(t, result[1].IsNil(), "%v", result[1].Interface())
			created := result[0].Interface().(umetav1.ResourceObjectI)
			assert.Equal(t, created.GetMetadata().Name, ts.srv.resolveDisplayName(ts.ctx, kind, created.GetMetadata().Uid))
			result = client.MethodByName("Delete" + kind).Call([]reflect.Value{reflect.ValueOf(ts.ctx), reflect.ValueOf(&rmetav1.DeleteOptions{Uid: created.GetMetadata().Uid})})
			require.True(t, result[1].IsNil(), "%v", result[1].Interface())
			assert.Empty(t, ts.srv.resolveDisplayName(ts.ctx, kind, created.GetMetadata().Uid))
		})
	}
	assert.Empty(t, ts.srv.resolveDisplayName(ts.ctx, "", "uid"))
	assert.Empty(t, ts.srv.resolveDisplayName(ts.ctx, "User", ""))
	assert.Empty(t, ts.srv.resolveDisplayName(ts.ctx, "Unknown", "uid"))
}
