package visibility

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	pb "github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcordiumv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type visibilityOcteliumClient struct {
	octeliumc.ClientInterface
	coreC rcorev1.ResourceServiceClient
}

func (c *visibilityOcteliumClient) CoreC() rcorev1.ResourceServiceClient {
	return c.coreC
}

func newTestClusterServer(conn *visibilityClientConn) *ServerCluster {
	return &ServerCluster{
		octeliumC:     &visibilityOcteliumClient{coreC: rcorev1.NewResourceServiceClient(conn)},
		coreC:         vcorev1.NewResourceServiceClient(conn),
		accessC:       vaccessv1.NewResourceServiceClient(conn),
		enterpriseC:   venterprisev1.NewResourceServiceClient(conn),
		cordiumC:      vcordiumv1.NewResourceServiceClient(conn),
		componentLogC: pb.NewComponentLogServiceClient(conn),
		accessLogC:    pb.NewAccessLogServiceClient(conn),
	}
}

func TestClusterSummaryFanOut(t *testing.T) {
	common := &vmetav1.CommonSummaryOptions{}
	common.ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 0x07})
	var mu sync.Mutex
	calls := make(map[string]int)
	conn := &visibilityClientConn{invoke: func(ctx context.Context, method string, req, res any) error {
		assert.Same(t, t.Context(), ctx)
		arg := req.(interface {
			GetCommon() *vmetav1.CommonSummaryOptions
		})
		assert.Same(t, common, arg.GetCommon())
		mu.Lock()
		calls[method]++
		mu.Unlock()
		msg := res.(proto.Message).ProtoReflect()
		field := msg.Descriptor().Fields().ByName("totalNumber")
		if !assert.NotNil(t, field) {
			return status.Error(codes.Internal, "missing totalNumber field")
		}
		msg.Set(field, protoreflect.ValueOfUint64(7))
		return nil
	}}

	srv := newTestClusterServer(conn)
	res, err := srv.GetClusterSummary(t.Context(), &pb.GetClusterSummaryRequest{Common: common})
	require.Nil(t, err)
	assert.Empty(t, res.Unavailables)
	assert.Len(t, calls, len(pb.GetClusterSummaryRequest_Kind_name)-1)
	for method, count := range calls {
		assert.Equal(t, 1, count, "%s", method)
	}
	assert.Equal(t, uint64(7), res.Core.User.TotalNumber)
	assert.Equal(t, uint64(7), res.Access.Request.TotalNumber)
	assert.Equal(t, uint64(7), res.Enterprise.DeviceManager.TotalNumber)
	assert.Equal(t, uint64(7), res.Cordium.Workspace.TotalNumber)

	for _, group := range []proto.Message{res.Core, res.Access, res.Enterprise, res.Cordium} {
		fields := group.ProtoReflect().Descriptor().Fields()
		for idx := 0; idx < fields.Len(); idx++ {
			field := fields.Get(idx)
			assert.True(t, group.ProtoReflect().Has(field), "%s", field.FullName())
		}
	}
}

func TestClusterSummarySelectors(t *testing.T) {
	for _, tt := range []struct {
		name  string
		kinds []pb.GetClusterSummaryRequest_Kind
		calls int
	}{
		{name: "Selected", kinds: []pb.GetClusterSummaryRequest_Kind{pb.GetClusterSummaryRequest_CORE_USER, pb.GetClusterSummaryRequest_ACCESS_REQUEST}, calls: 2},
		{name: "Duplicate", kinds: []pb.GetClusterSummaryRequest_Kind{pb.GetClusterSummaryRequest_CORE_USER, pb.GetClusterSummaryRequest_CORE_USER}, calls: 1},
		{name: "Unknown", kinds: []pb.GetClusterSummaryRequest_Kind{999}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var calls []string
			conn := &visibilityClientConn{invoke: func(ctx context.Context, method string, req, res any) error {
				mu.Lock()
				calls = append(calls, method)
				mu.Unlock()
				return nil
			}}
			res, err := newTestClusterServer(conn).GetClusterSummary(t.Context(), &pb.GetClusterSummaryRequest{Kinds: tt.kinds})
			require.Nil(t, err)
			assert.Len(t, calls, tt.calls)
			assert.Nil(t, res.Enterprise.Secret)
			assert.Nil(t, res.Cordium.Workspace)
			assert.Nil(t, res.Core.Service)
			assert.Empty(t, res.Unavailables)
		})
	}
}

func TestClusterSummaryPartialFailure(t *testing.T) {
	for _, failure := range []error{status.Error(codes.Unavailable, "store unavailable"), status.Error(codes.Canceled, "cancelled")} {
		t.Run(status.Code(failure).String(), func(t *testing.T) {
			conn := &visibilityClientConn{invoke: func(ctx context.Context, method string, req, res any) error {
				if method == vcorev1.ResourceService_GetUserSummary_FullMethodName {
					proto.Merge(res.(proto.Message), &vcorev1.GetUserSummaryResponse{TotalNumber: 3})
					return nil
				}
				return failure
			}}
			res, err := newTestClusterServer(conn).GetClusterSummary(t.Context(), nil)
			require.Nil(t, err)
			assert.Equal(t, uint64(3), res.Core.User.TotalNumber)
			require.Len(t, res.Unavailables, len(pb.GetClusterSummaryRequest_Kind_name)-2)
			for idx, item := range res.Unavailables {
				assert.Equal(t, failure.Error(), item.Message)
				if idx > 0 {
					assert.Less(t, res.Unavailables[idx-1].Kind, item.Kind)
				}
			}
		})
	}
}

func TestClusterHealthStatus(t *testing.T) {
	for _, tt := range []struct {
		critical uint64
		degraded uint64
		want     pb.GetClusterHealthResponse_Status
	}{
		{want: pb.GetClusterHealthResponse_OK},
		{degraded: 1, want: pb.GetClusterHealthResponse_DEGRADED},
		{critical: 1, want: pb.GetClusterHealthResponse_CRITICAL},
		{critical: 1, degraded: 1, want: pb.GetClusterHealthResponse_CRITICAL},
	} {
		assert.Equal(t, tt.want, healthStatus(tt.critical, tt.degraded))
	}
	for _, tt := range []struct {
		values []pb.GetClusterHealthResponse_Status
		want   pb.GetClusterHealthResponse_Status
	}{
		{want: pb.GetClusterHealthResponse_OK},
		{values: []pb.GetClusterHealthResponse_Status{pb.GetClusterHealthResponse_UNKNOWN}, want: pb.GetClusterHealthResponse_UNKNOWN},
		{values: []pb.GetClusterHealthResponse_Status{pb.GetClusterHealthResponse_OK, pb.GetClusterHealthResponse_UNKNOWN, pb.GetClusterHealthResponse_DEGRADED}, want: pb.GetClusterHealthResponse_DEGRADED},
		{values: []pb.GetClusterHealthResponse_Status{pb.GetClusterHealthResponse_CRITICAL, pb.GetClusterHealthResponse_DEGRADED, pb.GetClusterHealthResponse_UNKNOWN}, want: pb.GetClusterHealthResponse_CRITICAL},
	} {
		assert.Equal(t, tt.want, worstStatus(tt.values...))
		slices.Reverse(tt.values)
		assert.Equal(t, tt.want, worstStatus(tt.values...))
	}
}

func TestClusterHealth(t *testing.T) {
	from := timestamppb.New(time.Now().Add(-2 * time.Hour))
	to := timestamppb.New(time.Now().Add(-time.Hour))
	responses := map[string]proto.Message{
		pb.ComponentLogService_ListComponentLogTopComponent_FullMethodName: &pb.ListComponentLogTopComponentResponse{
			TotalCount: 4,
			Items: []*pb.ListComponentLogTopComponentResponse_Item{
				{Component: &pb.ComponentSelector{Type: "quiet"}, Count: 10},
				{Component: &pb.ComponentSelector{Type: "warning"}, Count: 5, CountWarn: 2},
				{Component: &pb.ComponentSelector{Type: "error"}, Count: 4, CountError: 1},
				{Component: &pb.ComponentSelector{Type: "fatal"}, Count: 3, CountPanic: 1, CountFatal: 1},
			},
		},
		pb.AccessLogService_GetAccessLogSummary_FullMethodName:                   &pb.GetAccessLogSummaryResponse{TotalNumber: 10, TotalDenied: 6},
		venterprisev1.ResourceService_GetCertificateSummary_FullMethodName:       &venterprisev1.GetCertificateSummaryResponse{TotalNumber: 5, TotalExpired: 1, TotalExpiringSoon: 2, TotalIssuanceFailed: 1},
		venterprisev1.ResourceService_GetCertificateIssuerSummary_FullMethodName: &venterprisev1.GetCertificateIssuerSummaryResponse{TotalNumber: 2, TotalNotReady: 1},
		venterprisev1.ResourceService_GetDirectoryProviderSummary_FullMethodName: &venterprisev1.GetDirectoryProviderSummaryResponse{TotalNumber: 2, TotalSynchronizationFailed: 1},
		venterprisev1.ResourceService_GetSecretStoreSummary_FullMethodName:       &venterprisev1.GetSecretStoreSummaryResponse{TotalNumber: 2, TotalSynchronizationFailed: 1},
		venterprisev1.ResourceService_GetDeviceManagerSummary_FullMethodName:     &venterprisev1.GetDeviceManagerSummaryResponse{TotalNumber: 4, TotalError: 1, TotalDegraded: 1, TotalFailedUpdates: 1},
		vcorev1.ResourceService_GetSessionSummary_FullMethodName:                 &vcorev1.GetSessionSummaryResponse{TotalNumber: 5, TotalPending: 1},
		vcorev1.ResourceService_GetDeviceSummary_FullMethodName:                  &vcorev1.GetDeviceSummaryResponse{TotalNumber: 4, TotalPending: 1},
		vcorev1.ResourceService_GetAuthenticatorSummary_FullMethodName:           &vcorev1.GetAuthenticatorSummaryResponse{TotalNumber: 3, TotalPending: 1},
		vaccessv1.ResourceService_GetRequestSummary_FullMethodName:               &vaccessv1.GetRequestSummaryResponse{TotalNumber: 5, TotalPending: 2, TotalPendingDeadlinePassed: 1},
		rcorev1.ResourceService_ListRegion_FullMethodName: &corev1.RegionList{Items: []*corev1.Region{
			{Metadata: &metav1.Metadata{Name: "primary", Uid: "r1"}, Status: &corev1.Region_Status{Version: "1.2.3"}},
			{Metadata: &metav1.Metadata{Name: "empty", Uid: "r2"}, Status: &corev1.Region_Status{}},
		}},
		rcorev1.ResourceService_ListGateway_FullMethodName: &corev1.GatewayList{Items: []*corev1.Gateway{
			{Status: &corev1.Gateway_Status{RegionRef: &metav1.ObjectReference{Uid: "r1"}}},
			{Status: &corev1.Gateway_Status{RegionRef: &metav1.ObjectReference{Uid: "r1"}}},
		}},
	}
	conn := &visibilityClientConn{invoke: func(ctx context.Context, method string, req, res any) error {
		switch arg := req.(type) {
		case *pb.ListComponentLogTopComponentRequest:
			assert.True(t, proto.Equal(from, arg.From))
			assert.True(t, proto.Equal(to, arg.To))
			assert.Equal(t, uint32(maxUnhealthyComponents), arg.Limit)
			assert.Equal(t, pb.ListComponentLogTopComponentRequest_SEVERITY, arg.OrderBy)
		case *pb.GetAccessLogSummaryRequest:
			assert.True(t, proto.Equal(from, arg.From))
			assert.True(t, proto.Equal(to, arg.To))
		}
		if response := responses[method]; response != nil {
			proto.Merge(res.(proto.Message), response)
		}
		return nil
	}}
	res, err := newTestClusterServer(conn).GetClusterHealth(t.Context(), &pb.GetClusterHealthRequest{From: from, To: to})
	require.Nil(t, err)
	assert.True(t, proto.Equal(from, res.From))
	assert.True(t, proto.Equal(to, res.To))
	assert.Equal(t, pb.GetClusterHealthResponse_CRITICAL, res.Status)
	require.Len(t, res.Subsystems, 9)
	for idx := 1; idx < len(res.Subsystems); idx++ {
		assert.Less(t, res.Subsystems[idx-1].Type, res.Subsystems[idx].Type)
	}
	for _, tt := range []struct {
		typ    pb.GetClusterHealthResponse_Subsystem_Type
		total  uint64
		status pb.GetClusterHealthResponse_Status
	}{
		{typ: pb.GetClusterHealthResponse_Subsystem_COMPONENTS, total: 4, status: pb.GetClusterHealthResponse_CRITICAL},
		{typ: pb.GetClusterHealthResponse_Subsystem_AUTHORIZATION, total: 10, status: pb.GetClusterHealthResponse_DEGRADED},
		{typ: pb.GetClusterHealthResponse_Subsystem_CERTIFICATES, total: 7, status: pb.GetClusterHealthResponse_CRITICAL},
		{typ: pb.GetClusterHealthResponse_Subsystem_DIRECTORY_PROVIDERS, total: 2, status: pb.GetClusterHealthResponse_CRITICAL},
		{typ: pb.GetClusterHealthResponse_Subsystem_SECRET_STORES, total: 2, status: pb.GetClusterHealthResponse_CRITICAL},
		{typ: pb.GetClusterHealthResponse_Subsystem_DEVICE_MANAGERS, total: 4, status: pb.GetClusterHealthResponse_CRITICAL},
		{typ: pb.GetClusterHealthResponse_Subsystem_ENROLLMENT, total: 12, status: pb.GetClusterHealthResponse_DEGRADED},
		{typ: pb.GetClusterHealthResponse_Subsystem_ACCESS_REQUESTS, total: 5, status: pb.GetClusterHealthResponse_CRITICAL},
		{typ: pb.GetClusterHealthResponse_Subsystem_DATA_PLANE, total: 2, status: pb.GetClusterHealthResponse_CRITICAL},
	} {
		idx := slices.IndexFunc(res.Subsystems, func(item *pb.GetClusterHealthResponse_Subsystem) bool { return item.Type == tt.typ })
		require.GreaterOrEqual(t, idx, 0)
		assert.Equal(t, tt.total, res.Subsystems[idx].Total)
		assert.Equal(t, tt.status, res.Subsystems[idx].Status)
		assert.NotZero(t, res.Subsystems[idx].Unhealthy)
	}
	require.Len(t, res.Components, 3)
	assert.Equal(t, "fatal", res.Components[0].Component.Type)
	assert.Equal(t, "error", res.Components[1].Component.Type)
	assert.Equal(t, "warning", res.Components[2].Component.Type)
	require.Len(t, res.Regions, 2)
	assert.Equal(t, uint64(2), res.Regions[0].TotalGateway)
	assert.Equal(t, "1.2.3", res.Regions[0].Version)
	assert.Equal(t, pb.GetClusterHealthResponse_OK, res.Regions[0].Status)
	assert.Equal(t, pb.GetClusterHealthResponse_CRITICAL, res.Regions[1].Status)
}

func TestClusterHealthUnavailable(t *testing.T) {
	methods := []string{
		pb.ComponentLogService_ListComponentLogTopComponent_FullMethodName,
		pb.AccessLogService_GetAccessLogSummary_FullMethodName,
		venterprisev1.ResourceService_GetCertificateSummary_FullMethodName,
		venterprisev1.ResourceService_GetCertificateIssuerSummary_FullMethodName,
		venterprisev1.ResourceService_GetDirectoryProviderSummary_FullMethodName,
		venterprisev1.ResourceService_GetSecretStoreSummary_FullMethodName,
		venterprisev1.ResourceService_GetDeviceManagerSummary_FullMethodName,
		vcorev1.ResourceService_GetSessionSummary_FullMethodName,
		vcorev1.ResourceService_GetDeviceSummary_FullMethodName,
		vcorev1.ResourceService_GetAuthenticatorSummary_FullMethodName,
		vaccessv1.ResourceService_GetRequestSummary_FullMethodName,
		rcorev1.ResourceService_ListRegion_FullMethodName,
		rcorev1.ResourceService_ListGateway_FullMethodName,
	}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			conn := &visibilityClientConn{invoke: func(ctx context.Context, got string, req, res any) error {
				if got == method {
					return status.Error(codes.Unavailable, "store unavailable")
				}
				return nil
			}}
			res, err := newTestClusterServer(conn).GetClusterHealth(t.Context(), nil)
			require.Nil(t, err)
			assert.Equal(t, pb.GetClusterHealthResponse_UNKNOWN, res.Status)
			assert.Len(t, res.Subsystems, 9)
			unknown := 0
			for _, item := range res.Subsystems {
				if item.Status == pb.GetClusterHealthResponse_UNKNOWN {
					unknown++
					assert.Contains(t, item.Message, "store unavailable")
				}
			}
			assert.Equal(t, 1, unknown)
			assert.WithinDuration(t, time.Now(), res.To.AsTime(), time.Second)
			assert.Equal(t, clusterHealthDefaultWindow, res.To.AsTime().Sub(res.From.AsTime()))
		})
	}
}
