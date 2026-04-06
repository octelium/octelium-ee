// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package visibility

import (
	"context"
	"fmt"
	"os"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	pb "github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	oc "github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"google.golang.org/grpc"
)

type ServerAccessLog struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedAccessLogServiceServer

	c pb.AccessLogServiceClient
}

func NewServerMain(ctx context.Context, octeliumC octeliumc.ClientInterface) (*ServerAccessLog, error) {

	var host string

	if ldflags.IsTest() {
		host = fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSTORE_PORT"))
	} else {
		host = "octeliumee-logstore.octelium.svc:8080"
	}

	grpcOpts, err := oc.DefaultDialOpts(ctx)
	if err != nil {
		return nil, err
	}
	grpcConn, err := grpc.NewClient(
		host, grpcOpts...,
	)
	if err != nil {
		return nil, err
	}

	return &ServerAccessLog{
		octeliumC: octeliumC,
		c:         pb.NewAccessLogServiceClient(grpcConn),
	}, nil
}

type ServerResource struct {
	octeliumC octeliumc.ClientInterface
	vcorev1.UnimplementedResourceServiceServer

	coreC vcorev1.ResourceServiceClient
}

func NewServerResource(ctx context.Context, octeliumC octeliumc.ClientInterface) (*ServerResource, error) {

	var host string

	if ovutils.IsMockMode() {
		host = "localhost:40001"
	} else if ldflags.IsTest() {
		host = fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSTORE_PORT"))
	} else {
		host = "octeliumee-rscstore.octelium.svc:8080"
	}

	grpcOpts, err := oc.DefaultDialOpts(ctx)
	if err != nil {
		return nil, err
	}
	grpcConn, err := grpc.NewClient(
		host, grpcOpts...,
	)
	if err != nil {
		return nil, err
	}

	return &ServerResource{
		octeliumC: octeliumC,
		coreC:     vcorev1.NewResourceServiceClient(grpcConn),
	}, nil
}

type ServerAuthenticationLog struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedAuthenticationLogServiceServer

	c pb.AuthenticationLogServiceClient
}

type ServerComponentLog struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedComponentLogServiceServer

	c pb.ComponentLogServiceClient
}

type ServerMetric struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedMetricsServiceServer

	c pb.MetricsServiceClient
}

func NewServerAuthenticationLog(ctx context.Context, octeliumC octeliumc.ClientInterface) (*ServerAuthenticationLog, error) {

	var host string

	if ldflags.IsTest() {
		host = fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSTORE_PORT"))
	} else {
		host = "octeliumee-logstore.octelium.svc:8080"
	}

	grpcOpts, err := oc.DefaultDialOpts(ctx)
	if err != nil {
		return nil, err
	}
	grpcConn, err := grpc.NewClient(
		host, grpcOpts...,
	)
	if err != nil {
		return nil, err
	}

	return &ServerAuthenticationLog{
		octeliumC: octeliumC,
		c:         pb.NewAuthenticationLogServiceClient(grpcConn),
	}, nil
}

func NewServerComponentLog(ctx context.Context, octeliumC octeliumc.ClientInterface) (*ServerComponentLog, error) {

	var host string

	if ldflags.IsTest() {
		host = fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSTORE_PORT"))
	} else {
		host = "octeliumee-logstore.octelium.svc:8080"
	}

	grpcOpts, err := oc.DefaultDialOpts(ctx)
	if err != nil {
		return nil, err
	}
	grpcConn, err := grpc.NewClient(
		host, grpcOpts...,
	)
	if err != nil {
		return nil, err
	}

	return &ServerComponentLog{
		octeliumC: octeliumC,
		c:         pb.NewComponentLogServiceClient(grpcConn),
	}, nil
}

type ServerAuditLog struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedAuditLogServiceServer

	c pb.AuditLogServiceClient
}

func NewServerAuditLog(ctx context.Context, octeliumC octeliumc.ClientInterface) (*ServerAuditLog, error) {

	var host string

	if ldflags.IsTest() {
		host = fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSTORE_PORT"))
	} else {
		host = "octeliumee-logstore.octelium.svc:8080"
	}

	grpcOpts, err := oc.DefaultDialOpts(ctx)
	if err != nil {
		return nil, err
	}
	grpcConn, err := grpc.NewClient(
		host, grpcOpts...,
	)
	if err != nil {
		return nil, err
	}

	return &ServerAuditLog{
		octeliumC: octeliumC,
		c:         pb.NewAuditLogServiceClient(grpcConn),
	}, nil
}

func NewServerMetric(ctx context.Context, octeliumC octeliumc.ClientInterface) (*ServerMetric, error) {

	var host string

	if ldflags.IsTest() {
		host = fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSTORE_PORT"))
	} else {
		host = "octeliumee-metricstore.octelium.svc:8080"
	}

	grpcOpts, err := oc.DefaultDialOpts(ctx)
	if err != nil {
		return nil, err
	}
	grpcConn, err := grpc.NewClient(
		host, grpcOpts...,
	)
	if err != nil {
		return nil, err
	}

	return &ServerMetric{
		octeliumC: octeliumC,
		c:         pb.NewMetricsServiceClient(grpcConn),
	}, nil
}
