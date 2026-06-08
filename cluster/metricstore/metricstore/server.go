// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/spiffec"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

const tstAddr = "localhost:32123"

type Server struct {
	octeliumC octeliumc.ClientInterface

	clusterDomain string

	db *sql.DB

	grpcStopped chan struct{}
}

func newServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {
	ret := &Server{
		octeliumC:   octeliumC,
		grpcStopped: make(chan struct{}),
	}

	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret.clusterDomain = cc.Status.Domain

	dsn := ovutils.GetDuckDBDSN()

	ret.db, err = sql.Open("duckdb", dsn)
	if err != nil {
		return nil, err
	}

	ret.db.SetMaxOpenConns(5)
	ret.db.SetMaxIdleConns(5)
	ret.db.SetConnMaxLifetime(0)

	return ret, nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.initDB(ctx); err != nil {
		return err
	}

	if err := s.initGRPC(ctx); err != nil {
		return err
	}

	go s.runRetentionLoop(ctx)

	go func() {
		<-s.grpcStopped

		if s.db != nil {
			_ = s.db.Close()
		}
	}()

	return nil
}

func DoRun(ctx context.Context, octeliumC octeliumc.ClientInterface) error {
	s, err := newServer(ctx, octeliumC)
	if err != nil {
		return err
	}

	return s.Run(ctx)
}

func Run(ctx context.Context) error {
	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	if err := DoRun(ctx, octeliumC); err != nil {
		return err
	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.L().Info("MetricStore is running")

	<-ctx.Done()
	return nil
}

func (s *Server) initGRPC(ctx context.Context) error {
	cred, err := spiffec.GetGRPCServerCred(ctx, nil)
	if err != nil {
		return err
	}

	grpcSrv := grpc.NewServer(
		cred,
		grpc.ReadBufferSize(512*1024),
		grpc.WriteBufferSize(512*1024),
		grpc.MaxRecvMsgSize(16<<20),
		grpc.MaxConcurrentStreams(1024),
	)

	srv := s.newSrvMetric()
	go srv.startProcessLoop(ctx)

	pmetricotlp.RegisterGRPCServer(grpcSrv, srv)
	vmetricsv1.RegisterMetricsServiceServer(grpcSrv, srv)

	addr := func() string {
		if ovutils.IsMockMode() {
			return "localhost:40001"
		}
		if ldflags.IsTest() {
			return tstAddr
		}
		return ":8080"
	}()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	zap.L().Debug("Starting MetricStore gRPC server",
		zap.String("addr", addr),
		zap.Bool("mockMode", ovutils.IsMockMode()),
		zap.Bool("testMode", ldflags.IsTest()))

	go func() {
		if err := grpcSrv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			zap.L().Fatal("MetricStore gRPC server exited", zap.Error(err))
		}
	}()

	go func() {
		defer close(s.grpcStopped)

		<-ctx.Done()

		done := make(chan struct{})
		go func() {
			grpcSrv.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			grpcSrv.Stop()
			<-done
		}
	}()

	return nil
}
