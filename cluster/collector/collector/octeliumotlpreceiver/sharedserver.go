// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package octeliumotlpreceiver

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/octelium/octelium/cluster/common/spiffec"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var globalServers sync.Map

func getOrCreateSharedServer(settings receiver.Settings, cfg *Config) *sharedOTLPServer {
	key := settings.ID.String()

	if val, ok := globalServers.Load(key); ok {
		return val.(*sharedOTLPServer)
	}

	srv := &sharedOTLPServer{
		key:      key,
		cfg:      cfg,
		settings: settings,
	}

	actual, _ := globalServers.LoadOrStore(key, srv)
	return actual.(*sharedOTLPServer)
}

type sharedOTLPServer struct {
	key      string
	cfg      *Config
	settings receiver.Settings

	mu       sync.Mutex
	started  bool
	refCount int

	srv *grpc.Server
	lis net.Listener

	nextLogs    consumer.Logs
	nextMetrics consumer.Metrics
}

func (s *sharedOTLPServer) setLogsConsumer(next consumer.Logs) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextLogs = next
}

func (s *sharedOTLPServer) setMetricsConsumer(next consumer.Metrics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextMetrics = next
}

func (s *sharedOTLPServer) Start(ctx context.Context, _ component.Host) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.refCount++

	if s.started {
		return nil
	}

	cred, err := spiffec.GetGRPCServerCred(ctx, nil)
	if err != nil {
		s.refCount--
		return err
	}

	lis, err := net.Listen("tcp", s.cfg.Endpoint)
	if err != nil {
		s.refCount--
		return err
	}

	maxRecv := 16 << 20
	if s.cfg.MaxRecvMsgSizeMiB > 0 {
		maxRecv = s.cfg.MaxRecvMsgSizeMiB << 20
	}

	maxStreams := uint32(1024)
	if s.cfg.MaxConcurrentStreams > 0 {
		maxStreams = s.cfg.MaxConcurrentStreams
	}

	grpcSrv := grpc.NewServer(
		cred,
		grpc.MaxRecvMsgSize(maxRecv),
		grpc.MaxConcurrentStreams(maxStreams),
	)

	plogotlp.RegisterGRPCServer(grpcSrv, &logsServer{parent: s})
	pmetricotlp.RegisterGRPCServer(grpcSrv, &metricsServer{parent: s})

	s.srv = grpcSrv
	s.lis = lis
	s.started = true

	go s.serve(grpcSrv, lis)

	s.settings.Logger.Info("octelium OTLP receiver started",
		zap.String("endpoint", s.cfg.Endpoint),
		zap.String("id", s.key),
	)

	return nil
}

func (s *sharedOTLPServer) serve(grpcSrv *grpc.Server, lis net.Listener) {
	if err := grpcSrv.Serve(lis); err != nil {
		if errors.Is(err, grpc.ErrServerStopped) || errors.Is(err, net.ErrClosed) {
			s.settings.Logger.Debug("octelium OTLP receiver stopped",
				zap.String("id", s.key),
			)
			return
		}

		s.settings.Logger.Error("octelium OTLP receiver exited",
			zap.String("id", s.key),
			zap.Error(err),
		)
	}
}

func (s *sharedOTLPServer) Release(ctx context.Context) error {
	s.mu.Lock()

	if s.refCount > 0 {
		s.refCount--
	}

	if s.refCount > 0 || !s.started {
		s.mu.Unlock()
		return nil
	}

	grpcSrv := s.srv
	lis := s.lis

	s.started = false
	s.srv = nil
	s.lis = nil
	s.nextLogs = nil
	s.nextMetrics = nil

	globalServers.Delete(s.key)

	s.mu.Unlock()

	if grpcSrv != nil {
		done := make(chan struct{})
		go func() {
			grpcSrv.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			grpcSrv.Stop()
			<-done
		}
	}

	if lis != nil {
		_ = lis.Close()
	}

	return nil
}

func (s *sharedOTLPServer) consumeLogs(ctx context.Context, logs plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	s.mu.Lock()
	next := s.nextLogs
	s.mu.Unlock()

	if next == nil {
		return plogotlp.NewExportResponse(), nil
	}

	if err := next.ConsumeLogs(ctx, logs.Logs()); err != nil {
		return plogotlp.NewExportResponse(), err
	}

	return plogotlp.NewExportResponse(), nil
}

func (s *sharedOTLPServer) consumeMetrics(ctx context.Context, metrics pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	s.mu.Lock()
	next := s.nextMetrics
	s.mu.Unlock()

	if next == nil {
		return pmetricotlp.NewExportResponse(), nil
	}

	if err := next.ConsumeMetrics(ctx, metrics.Metrics()); err != nil {
		return pmetricotlp.NewExportResponse(), err
	}

	return pmetricotlp.NewExportResponse(), nil
}

type logsServer struct {
	plogotlp.UnimplementedGRPCServer
	parent *sharedOTLPServer
}

func (s *logsServer) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	return s.parent.consumeLogs(ctx, req)
}

type metricsServer struct {
	pmetricotlp.UnimplementedGRPCServer
	parent *sharedOTLPServer
}

func (s *metricsServer) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	return s.parent.consumeMetrics(ctx, req)
}
