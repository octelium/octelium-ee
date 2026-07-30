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
	"crypto/rand"
	"database/sql"
	"errors"
	"net"
	"os"
	"strings"
	"sync"
	"time"

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
	db            *sql.DB
	dbConfig      *metricStoreDBConfig

	grpcSrv   *grpc.Server
	listener  net.Listener
	metricSrv *srvMetric

	pageTokenKey []byte

	workerCtx    context.Context
	workerCancel context.CancelFunc
	workerWG     sync.WaitGroup

	shutdownOnce sync.Once
	shutdownDone chan struct{}
}

func newServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {
	cc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	dbConfig, err := getMetricStoreDBConfig()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("duckdb", dbConfig.dsn)
	if err != nil {
		if dbConfig.cleanupFn != nil {
			dbConfig.cleanupFn()
		}
		return nil, err
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(0)

	workerCtx, workerCancel := context.WithCancel(context.Background())

	pageTokenKey, err := getPageTokenKey()
	if err != nil {
		_ = db.Close()
		if dbConfig.cleanupFn != nil {
			dbConfig.cleanupFn()
		}
		return nil, err
	}

	ret := &Server{
		octeliumC:     octeliumC,
		clusterDomain: cc.Status.Domain,
		db:            db,
		dbConfig:      dbConfig,
		pageTokenKey:  pageTokenKey,
		workerCtx:     workerCtx,
		workerCancel:  workerCancel,
		shutdownDone:  make(chan struct{}),
	}

	return ret, nil
}

func getPageTokenKey() ([]byte, error) {
	if val := strings.TrimSpace(os.Getenv("OCTELIUM_METRICSTORE_PAGE_TOKEN_SECRET")); val != "" {
		if len(val) < 32 {
			return nil, errors.New("OCTELIUM_METRICSTORE_PAGE_TOKEN_SECRET must contain at least 32 bytes")
		}
		return []byte(val), nil
	}

	if !ldflags.IsTest() {
		return nil, errors.New("OCTELIUM_METRICSTORE_PAGE_TOKEN_SECRET must be set")
	}

	ret := make([]byte, 32)
	if _, err := rand.Read(ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.initDB(ctx); err != nil {
		s.closeStorage()
		return err
	}

	if err := s.applyRetention(ctx); err != nil {
		s.closeStorage()
		return err
	}

	if err := s.initGRPC(ctx); err != nil {
		s.workerCancel()
		if s.metricSrv != nil {
			s.metricSrv.stopAccepting()
			s.metricSrv.closeQueue()
		}
		s.workerWG.Wait()
		s.closeStorage()
		return err
	}

	s.workerWG.Add(1)
	go func() {
		defer s.workerWG.Done()
		s.runRetentionLoop(s.workerCtx)
	}()

	s.workerWG.Add(1)
	go func() {
		defer s.workerWG.Done()
		s.runDiagnosticsLoop(s.workerCtx)
	}()

	go func() {
		<-ctx.Done()
		s.shutdown()
	}()

	return nil
}

func (s *Server) initGRPC(ctx context.Context) error {
	cred, err := spiffec.GetGRPCServerCred(ctx, nil)
	if err != nil {
		return err
	}

	s.grpcSrv = grpc.NewServer(
		cred,
		grpc.ReadBufferSize(64*1024),
		grpc.WriteBufferSize(64*1024),
		grpc.MaxRecvMsgSize(8<<20),
		grpc.MaxSendMsgSize(32<<20),
		grpc.MaxConcurrentStreams(128),
	)

	s.metricSrv = s.newSrvMetric()
	s.workerWG.Add(1)
	go func() {
		defer s.workerWG.Done()
		s.metricSrv.startProcessLoop(s.workerCtx)
	}()

	pmetricotlp.RegisterGRPCServer(s.grpcSrv, s.metricSrv)
	vmetricsv1.RegisterMetricsServiceServer(s.grpcSrv, s.metricSrv)

	addr := func() string {
		if ovutils.IsMockMode() {
			return "localhost:40001"
		}
		if ldflags.IsTest() {
			return tstAddr
		}
		return ":8080"
	}()

	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	zap.L().Debug("Starting MetricStore gRPC server",
		zap.String("addr", addr),
		zap.Bool("mockMode", ovutils.IsMockMode()),
		zap.Bool("testMode", ldflags.IsTest()))

	go func() {
		if err := s.grpcSrv.Serve(s.listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			zap.L().Error("MetricStore gRPC server exited", zap.Error(err))
		}
	}()

	return nil
}

func (s *Server) closeStorage() {
	if s.db != nil {
		_ = s.db.Close()
		s.db = nil
	}
	if s.dbConfig != nil && s.dbConfig.cleanupFn != nil {
		s.dbConfig.cleanupFn()
		s.dbConfig.cleanupFn = nil
	}
}

func (s *Server) shutdown() {
	s.shutdownOnce.Do(func() {
		defer close(s.shutdownDone)

		if s.metricSrv != nil {
			s.metricSrv.stopAccepting()
		}

		if s.grpcSrv != nil {
			done := make(chan struct{})
			go func() {
				s.grpcSrv.GracefulStop()
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(10 * time.Second):
				s.grpcSrv.Stop()
				<-done
			}
		}

		if s.metricSrv != nil {
			s.metricSrv.closeQueue()
		}

		s.workerCancel()
		s.workerWG.Wait()

		if s.listener != nil {
			_ = s.listener.Close()
		}
		s.closeStorage()
	})
}

func (s *Server) WaitForShutdown() {
	<-s.shutdownDone
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

	s, err := newServer(ctx, octeliumC)
	if err != nil {
		return err
	}

	if err := s.Run(ctx); err != nil {
		return err
	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.L().Info("MetricStore is running")

	<-ctx.Done()
	s.WaitForShutdown()

	return nil
}
