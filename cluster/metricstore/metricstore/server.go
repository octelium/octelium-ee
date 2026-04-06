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
	"net"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/spiffec"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

const tstAddr = "localhost:32123"

type Server struct {
	octeliumC octeliumc.ClientInterface

	clusterDomain string
	genCache      *cache.Cache
	db            *sql.DB
}

func newServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {

	var err error
	ret := &Server{
		octeliumC: octeliumC,
		genCache:  cache.New(cache.NoExpiration, 1*time.Minute),
	}

	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	ret.clusterDomain = cc.Status.Domain

	ret.db, err = sql.Open("duckdb", ovutils.GetDuckDBDSN())
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *Server) Run(ctx context.Context) error {

	if err := s.initDB(ctx); err != nil {
		return err
	}

	if err := s.initGRPC(ctx); err != nil {
		return err
	}

	return nil
}

func DoRun(ctx context.Context, octeliumC octeliumc.ClientInterface) error {
	s, err := newServer(ctx, octeliumC)
	if err != nil {
		return err
	}

	if err := s.Run(ctx); err != nil {
		return err
	}

	return nil
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
	zap.S().Infof("MetricStore is running...")

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
		grpc.ReadBufferSize(32*1024),
		grpc.MaxConcurrentStreams(1000000),
	)

	srv := s.newSrvMetric()

	go srv.startProcessLoop(ctx)

	pmetricotlp.RegisterGRPCServer(grpcSrv, srv)
	visibilityv1.RegisterMetricsServiceServer(grpcSrv, srv)

	zap.L().Debug("Starting gRPC sever", zap.Bool("mockMode", ovutils.IsMockMode()))

	go func() {

		lis, err := net.Listen("tcp", func() string {

			if ovutils.IsMockMode() {
				return "localhost:40001"
			}

			if ldflags.IsTest() {
				return tstAddr
			}

			return ":8080"
		}())
		if err != nil {
			return
		}
		grpcSrv.Serve(lis)
	}()

	return nil
}

func (s *Server) initDB(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS metrics (
    timestamp      TIMESTAMP,
    name           TEXT,
    unit           TEXT,
    metric_type    TEXT,
    resource       JSON,
    scope          JSON,
    attributes     JSON,

    -- Number data points (Gauge/Sum)
    value          DOUBLE,

    -- Histogram
    histogram_count      BIGINT,
    histogram_sum        DOUBLE,
    histogram_bounds     JSON,
    histogram_bucket_counts JSON,

    -- Summary
    summary_count        BIGINT,
    summary_sum          DOUBLE,
    summary_quantiles    JSON,

	-- Component
	component_type TEXT,
	component_namespace TEXT
)`)
	if err != nil {
		return err
	}

	return nil
}
