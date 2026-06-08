// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package vmetricsv1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	MetricsService_QueryMetrics_FullMethodName          = "/octelium.api.main.visibility.metrics.v1.MetricsService/QueryMetrics"
	MetricsService_ListMetricDescriptors_FullMethodName = "/octelium.api.main.visibility.metrics.v1.MetricsService/ListMetricDescriptors"
	MetricsService_ListMetricCatalog_FullMethodName     = "/octelium.api.main.visibility.metrics.v1.MetricsService/ListMetricCatalog"
)

// MetricsServiceClient is the client API for MetricsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MetricsServiceClient interface {
	QueryMetrics(ctx context.Context, in *QueryMetricsRequest, opts ...grpc.CallOption) (*QueryMetricsResponse, error)
	ListMetricDescriptors(ctx context.Context, in *ListMetricDescriptorsRequest, opts ...grpc.CallOption) (*ListMetricDescriptorsResponse, error)
	ListMetricCatalog(ctx context.Context, in *ListMetricCatalogRequest, opts ...grpc.CallOption) (*ListMetricCatalogResponse, error)
}

type metricsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMetricsServiceClient(cc grpc.ClientConnInterface) MetricsServiceClient {
	return &metricsServiceClient{cc}
}

func (c *metricsServiceClient) QueryMetrics(ctx context.Context, in *QueryMetricsRequest, opts ...grpc.CallOption) (*QueryMetricsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(QueryMetricsResponse)
	err := c.cc.Invoke(ctx, MetricsService_QueryMetrics_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *metricsServiceClient) ListMetricDescriptors(ctx context.Context, in *ListMetricDescriptorsRequest, opts ...grpc.CallOption) (*ListMetricDescriptorsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListMetricDescriptorsResponse)
	err := c.cc.Invoke(ctx, MetricsService_ListMetricDescriptors_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *metricsServiceClient) ListMetricCatalog(ctx context.Context, in *ListMetricCatalogRequest, opts ...grpc.CallOption) (*ListMetricCatalogResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListMetricCatalogResponse)
	err := c.cc.Invoke(ctx, MetricsService_ListMetricCatalog_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MetricsServiceServer is the server API for MetricsService service.
// All implementations must embed UnimplementedMetricsServiceServer
// for forward compatibility.
type MetricsServiceServer interface {
	QueryMetrics(context.Context, *QueryMetricsRequest) (*QueryMetricsResponse, error)
	ListMetricDescriptors(context.Context, *ListMetricDescriptorsRequest) (*ListMetricDescriptorsResponse, error)
	ListMetricCatalog(context.Context, *ListMetricCatalogRequest) (*ListMetricCatalogResponse, error)
	mustEmbedUnimplementedMetricsServiceServer()
}

// UnimplementedMetricsServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedMetricsServiceServer struct{}

func (UnimplementedMetricsServiceServer) QueryMetrics(context.Context, *QueryMetricsRequest) (*QueryMetricsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryMetrics not implemented")
}
func (UnimplementedMetricsServiceServer) ListMetricDescriptors(context.Context, *ListMetricDescriptorsRequest) (*ListMetricDescriptorsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListMetricDescriptors not implemented")
}
func (UnimplementedMetricsServiceServer) ListMetricCatalog(context.Context, *ListMetricCatalogRequest) (*ListMetricCatalogResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListMetricCatalog not implemented")
}
func (UnimplementedMetricsServiceServer) mustEmbedUnimplementedMetricsServiceServer() {}
func (UnimplementedMetricsServiceServer) testEmbeddedByValue()                        {}

// UnsafeMetricsServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MetricsServiceServer will
// result in compilation errors.
type UnsafeMetricsServiceServer interface {
	mustEmbedUnimplementedMetricsServiceServer()
}

func RegisterMetricsServiceServer(s grpc.ServiceRegistrar, srv MetricsServiceServer) {
	// If the following call pancis, it indicates UnimplementedMetricsServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&MetricsService_ServiceDesc, srv)
}

func _MetricsService_QueryMetrics_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryMetricsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetricsServiceServer).QueryMetrics(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MetricsService_QueryMetrics_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetricsServiceServer).QueryMetrics(ctx, req.(*QueryMetricsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MetricsService_ListMetricDescriptors_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListMetricDescriptorsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetricsServiceServer).ListMetricDescriptors(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MetricsService_ListMetricDescriptors_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetricsServiceServer).ListMetricDescriptors(ctx, req.(*ListMetricDescriptorsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MetricsService_ListMetricCatalog_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListMetricCatalogRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MetricsServiceServer).ListMetricCatalog(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MetricsService_ListMetricCatalog_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MetricsServiceServer).ListMetricCatalog(ctx, req.(*ListMetricCatalogRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// MetricsService_ServiceDesc is the grpc.ServiceDesc for MetricsService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MetricsService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "octelium.api.main.visibility.metrics.v1.MetricsService",
	HandlerType: (*MetricsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "QueryMetrics",
			Handler:    _MetricsService_QueryMetrics_Handler,
		},
		{
			MethodName: "ListMetricDescriptors",
			Handler:    _MetricsService_ListMetricDescriptors_Handler,
		},
		{
			MethodName: "ListMetricCatalog",
			Handler:    _MetricsService_ListMetricCatalog_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "vmetricsv1.proto",
}
