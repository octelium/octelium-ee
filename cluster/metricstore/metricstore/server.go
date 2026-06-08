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
	"net/url"
	"os"
	"strings"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
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
		grpc.UnaryInterceptor(s.metricstoreAuthUnaryInterceptor()),
		grpc.StreamInterceptor(s.metricstoreAuthStreamInterceptor()),
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

func (s *Server) metricstoreAuthUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if err := s.authorizeMetricstoreMethod(ctx, info.FullMethod); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func (s *Server) metricstoreAuthStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if err := s.authorizeMetricstoreMethod(ss.Context(), info.FullMethod); err != nil {
			return err
		}

		return handler(srv, ss)
	}
}

func (s *Server) authorizeMetricstoreMethod(ctx context.Context, fullMethod string) error {
	if ovutils.IsMockMode() || ldflags.IsTest() {
		return nil
	}

	spiffeID := getPeerSPIFFEID(ctx)

	switch {
	case strings.HasSuffix(fullMethod, "/Export"):
		if !s.isCollectorSPIFFEID(spiffeID) {
			return status.Error(codes.PermissionDenied, "not authorized to export metrics")
		}

	default:
		if !s.isMetricsQuerySPIFFEID(spiffeID) {
			return status.Error(codes.PermissionDenied, "not authorized to query metrics")
		}
	}

	return nil
}

func getPeerSPIFFEID(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		return ""
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ""
	}

	for _, cert := range tlsInfo.State.PeerCertificates {
		for _, uri := range cert.URIs {
			if uri.Scheme == "spiffe" {
				return uri.String()
			}
		}
	}

	return ""
}

func (s *Server) isCollectorSPIFFEID(id string) bool {
	return s.isAllowedSPIFFEID(id, map[string]struct{}{
		"/ns/octelium/sa/octelium-collector": {},
		"/ns/octelium/sa/collector":          {},
	})
}

func (s *Server) isMetricsQuerySPIFFEID(id string) bool {
	return s.isAllowedSPIFFEID(id, map[string]struct{}{
		"/ns/octelium/sa/octelium-apiserver": {},
		"/ns/octelium/sa/apiserver":          {},
		"/ns/octelium/sa/octelium-console":   {},
		"/ns/octelium/sa/console":            {},
	})
}

func (s *Server) isAllowedSPIFFEID(id string, allowedPaths map[string]struct{}) bool {
	trustDomain, path, ok := parseSPIFFEID(id)
	if !ok {
		return false
	}

	if trustDomain != s.expectedSPIFFETrustDomain() {
		return false
	}

	_, ok = allowedPaths[path]
	return ok
}

func (s *Server) expectedSPIFFETrustDomain() string {
	if val := strings.TrimSpace(os.Getenv("OCTELIUM_SPIFFE_TRUST_DOMAIN")); val != "" {
		return val
	}

	return s.clusterDomain
}

func parseSPIFFEID(id string) (trustDomain string, path string, ok bool) {
	u, err := url.Parse(id)
	if err != nil {
		return "", "", false
	}

	if u.Scheme != "spiffe" || u.Host == "" || u.Path == "" {
		return "", "", false
	}

	return u.Host, u.Path, true
}
