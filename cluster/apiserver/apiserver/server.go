// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"context"
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/cluster"
	eesrv "github.com/octelium/octelium-ee/cluster/apiserver/apiserver/enterprise"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/policyportal"
	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/visibility"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/cluster/common/vutils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func Run(ctx context.Context) error {
	zap.S().Debug("Starting Enterprise octelium API server...")

	if err := commoninit.Run(ctx, nil); err != nil {
		return err
	}

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", vutils.ManagedServiceAddr)
	if err != nil {
		return err
	}

	srv := eesrv.NewServer(octeliumC)

	zap.S().Debug("starting gRPC server....")

	mdlwr, err := userctx.New(ctx, octeliumC)
	if err != nil {
		return err
	}

	s := grpc.NewServer(
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(mdlwr.StreamServerInterceptor())),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(mdlwr.UnaryServerInterceptor())),
	)
	enterprisev1.RegisterMainServiceServer(s, srv)

	srvVisibilityAccessLog, err := visibility.NewServerMain(ctx, octeliumC)
	if err != nil {
		return err
	}

	srvVisibilityAuthenticationLog, err := visibility.NewServerAuthenticationLog(ctx, octeliumC)
	if err != nil {
		return err
	}

	srvVisibilityAuditLog, err := visibility.NewServerAuditLog(ctx, octeliumC)
	if err != nil {
		return err
	}

	srvVisibilityComponentLog, err := visibility.NewServerComponentLog(ctx, octeliumC)
	if err != nil {
		return err
	}

	srvVisibilityResource, err := visibility.NewServerResource(ctx, octeliumC)
	if err != nil {
		return err
	}

	srvVisibilityMetric, err := visibility.NewServerMetric(ctx, octeliumC)
	if err != nil {
		return err
	}

	policyPortalSrv, err := policyportal.NewServer(octeliumC)
	if err != nil {
		return err
	}

	clusterSrv, err := cluster.NewServer(octeliumC)
	if err != nil {
		return err
	}

	visibilityv1.RegisterAccessLogServiceServer(s, srvVisibilityAccessLog)
	visibilityv1.RegisterAuthenticationLogServiceServer(s, srvVisibilityAuthenticationLog)
	visibilityv1.RegisterAuditLogServiceServer(s, srvVisibilityAuditLog)
	visibilityv1.RegisterMetricsServiceServer(s, srvVisibilityMetric)
	visibilityv1.RegisterComponentLogServiceServer(s, srvVisibilityComponentLog)
	vcorev1.RegisterResourceServiceServer(s, srvVisibilityResource)
	enterprisev1.RegisterPolicyPortalServiceServer(s, policyPortalSrv)
	enterprisev1.RegisterClusterServiceServer(s, clusterSrv)

	go func() {
		zap.S().Debug("running gRPC server.")
		if err := s.Serve(lis); err != nil {
			zap.S().Infof("gRPC server closed: %+v", err)
		}
	}()

	healthcheck.Run(vutils.HealthCheckPortManagedService)
	zap.L().Info("Enterprise APIServer is now running")
	<-ctx.Done()
	zap.L().Debug("Shutting down gRPC server")
	s.Stop()

	return nil
}
