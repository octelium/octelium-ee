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
	"fmt"
	"net"
	"sync"

	"github.com/octelium/octelium/cluster/common/spiffec"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const dataFormatProtobuf = "protobuf"

var receiverRegistry sync.Map

type octeliumOTLPReceiver struct {
	cfg      *Config
	settings receiver.Settings

	mu sync.RWMutex

	started    bool
	startCount int

	serverGRPC *grpc.Server
	listener   net.Listener

	shutdownWG sync.WaitGroup

	nextLogs    consumer.Logs
	nextMetrics consumer.Metrics

	obsrepGRPC *receiverhelper.ObsReport

	registryKey string
}

var (
	_ receiver.Logs    = (*octeliumOTLPReceiver)(nil)
	_ receiver.Metrics = (*octeliumOTLPReceiver)(nil)
)

func loadOrCreateReceiver(settings receiver.Settings, cfg component.Config) (*octeliumOTLPReceiver, error) {
	typedCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("invalid octelium_otlp receiver config type")
	}

	if err := typedCfg.Validate(); err != nil {
		return nil, err
	}

	key := receiverRegistryKey(settings.ID.String(), typedCfg)

	if val, ok := receiverRegistry.Load(key); ok {
		return val.(*octeliumOTLPReceiver), nil
	}

	r, err := newReceiver(settings, typedCfg, key)
	if err != nil {
		return nil, err
	}

	actual, loaded := receiverRegistry.LoadOrStore(key, r)
	if loaded {
		return actual.(*octeliumOTLPReceiver), nil
	}

	return r, nil
}

func receiverRegistryKey(id string, cfg *Config) string {
	return fmt.Sprintf("%s|%s", id, cfg.Endpoint)
}

func newReceiver(settings receiver.Settings, cfg *Config, registryKey string) (*octeliumOTLPReceiver, error) {
	obsrep, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             settings.ID,
		Transport:              "grpc",
		ReceiverCreateSettings: settings,
	})
	if err != nil {
		return nil, err
	}

	return &octeliumOTLPReceiver{
		cfg:         cfg,
		settings:    settings,
		obsrepGRPC:  obsrep,
		registryKey: registryKey,
	}, nil
}

func (r *octeliumOTLPReceiver) registerLogsConsumer(next consumer.Logs) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextLogs = next
}

func (r *octeliumOTLPReceiver) registerMetricsConsumer(next consumer.Metrics) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextMetrics = next
}

func (r *octeliumOTLPReceiver) Start(ctx context.Context, host component.Host) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.startCount++

	if r.started {
		return nil
	}

	cred, err := spiffec.GetGRPCServerCred(ctx, nil)
	if err != nil {
		r.startCount--
		return err
	}

	opts := []grpc.ServerOption{
		cred,
		grpc.MaxRecvMsgSize(r.maxRecvMsgSize()),
		grpc.MaxConcurrentStreams(r.maxConcurrentStreams()),
	}

	if r.cfg.ReadBufferSize > 0 {
		opts = append(opts, grpc.ReadBufferSize(r.cfg.ReadBufferSize))
	}

	if r.cfg.WriteBufferSize > 0 {
		opts = append(opts, grpc.WriteBufferSize(r.cfg.WriteBufferSize))
	}

	serverGRPC := grpc.NewServer(opts...)

	plogotlp.RegisterGRPCServer(serverGRPC, &logsServer{parent: r})
	pmetricotlp.RegisterGRPCServer(serverGRPC, &metricsServer{parent: r})

	listener, err := net.Listen("tcp", r.cfg.Endpoint)
	if err != nil {
		r.startCount--
		return err
	}

	r.serverGRPC = serverGRPC
	r.listener = listener
	r.started = true

	r.shutdownWG.Add(1)
	go r.serve(host, serverGRPC, listener)

	r.settings.Logger.Info("octelium OTLP receiver started",
		zap.String("endpoint", listener.Addr().String()),
		zap.String("id", r.settings.ID.String()),
	)

	return nil
}

func (r *octeliumOTLPReceiver) serve(host component.Host, serverGRPC *grpc.Server, listener net.Listener) {
	defer r.shutdownWG.Done()

	err := serverGRPC.Serve(listener)
	if err == nil {
		return
	}

	if errors.Is(err, grpc.ErrServerStopped) || errors.Is(err, net.ErrClosed) {
		r.settings.Logger.Debug("octelium OTLP receiver stopped",
			zap.String("id", r.settings.ID.String()),
		)
		return
	}

	r.settings.Logger.Error("octelium OTLP receiver exited",
		zap.String("id", r.settings.ID.String()),
		zap.Error(err),
	)

	componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(err))
}

func (r *octeliumOTLPReceiver) Shutdown(ctx context.Context) error {
	r.mu.Lock()

	if r.startCount > 0 {
		r.startCount--
	}

	if r.startCount > 0 {
		r.mu.Unlock()
		return nil
	}

	if !r.started {
		receiverRegistry.Delete(r.registryKey)
		r.mu.Unlock()
		return nil
	}

	serverGRPC := r.serverGRPC
	listener := r.listener

	r.started = false
	r.serverGRPC = nil
	r.listener = nil
	r.nextLogs = nil
	r.nextMetrics = nil

	receiverRegistry.Delete(r.registryKey)

	r.mu.Unlock()

	if serverGRPC != nil {
		done := make(chan struct{})
		go func() {
			serverGRPC.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			serverGRPC.Stop()
			<-done
		}
	}

	if listener != nil {
		_ = listener.Close()
	}

	r.shutdownWG.Wait()

	return nil
}

func (r *octeliumOTLPReceiver) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{
		MutatesData: false,
	}
}

func (r *octeliumOTLPReceiver) maxRecvMsgSize() int {
	if r.cfg.MaxRecvMsgSizeMiB > 0 {
		return r.cfg.MaxRecvMsgSizeMiB << 20
	}

	return 16 << 20
}

func (r *octeliumOTLPReceiver) maxConcurrentStreams() uint32 {
	if r.cfg.MaxConcurrentStreams > 0 {
		return r.cfg.MaxConcurrentStreams
	}

	return 1024
}

func (r *octeliumOTLPReceiver) getLogsConsumer() consumer.Logs {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.nextLogs
}

func (r *octeliumOTLPReceiver) getMetricsConsumer() consumer.Metrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.nextMetrics
}

type logsServer struct {
	plogotlp.UnimplementedGRPCServer
	parent *octeliumOTLPReceiver
}

func (s *logsServer) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	ld := req.Logs()
	count := ld.LogRecordCount()
	if count == 0 {
		return plogotlp.NewExportResponse(), nil
	}

	next := s.parent.getLogsConsumer()
	if next == nil {
		return plogotlp.NewExportResponse(),
			status.Error(codes.Unavailable, "logs pipeline is not ready")
	}

	ctx = s.parent.obsrepGRPC.StartLogsOp(ctx)

	err := next.ConsumeLogs(ctx, ld)

	s.parent.obsrepGRPC.EndLogsOp(ctx, dataFormatProtobuf, count, err)

	if err != nil {
		return plogotlp.NewExportResponse(), grpcStatusFromConsumerError(err)
	}

	return plogotlp.NewExportResponse(), nil
}

type metricsServer struct {
	pmetricotlp.UnimplementedGRPCServer
	parent *octeliumOTLPReceiver
}

func (s *metricsServer) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	md := req.Metrics()
	count := md.DataPointCount()
	if count == 0 {
		return pmetricotlp.NewExportResponse(), nil
	}

	next := s.parent.getMetricsConsumer()
	if next == nil {
		return pmetricotlp.NewExportResponse(),
			status.Error(codes.Unavailable, "metrics pipeline is not ready")
	}

	ctx = s.parent.obsrepGRPC.StartMetricsOp(ctx)

	err := next.ConsumeMetrics(ctx, md)

	s.parent.obsrepGRPC.EndMetricsOp(ctx, dataFormatProtobuf, count, err)

	if err != nil {
		return pmetricotlp.NewExportResponse(), grpcStatusFromConsumerError(err)
	}

	return pmetricotlp.NewExportResponse(), nil
}

func grpcStatusFromConsumerError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		code := codes.Unavailable
		if consumererror.IsPermanent(err) {
			code = codes.Internal
		}

		st = status.New(code, err.Error())
	}

	return st.Err()
}
