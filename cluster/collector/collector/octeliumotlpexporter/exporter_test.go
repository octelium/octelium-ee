// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package octeliumotlpexporter

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

type exporterTestLogsClient struct {
	plogotlp.GRPCClient
	export func(context.Context, plogotlp.ExportRequest) (plogotlp.ExportResponse, error)
}

func (c *exporterTestLogsClient) Export(ctx context.Context, req plogotlp.ExportRequest, _ ...grpc.CallOption) (plogotlp.ExportResponse, error) {
	return c.export(ctx, req)
}

type exporterTestMetricsClient struct {
	pmetricotlp.GRPCClient
	export func(context.Context, pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error)
}

func (c *exporterTestMetricsClient) Export(ctx context.Context, req pmetricotlp.ExportRequest, _ ...grpc.CallOption) (pmetricotlp.ExportResponse, error) {
	return c.export(ctx, req)
}

type exporterTestLogsServer struct {
	plogotlp.UnimplementedGRPCServer
	received chan metadata.MD
	count    atomic.Int32
}

func (s *exporterTestLogsServer) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	s.count.Add(int32(req.Logs().LogRecordCount()))
	md, _ := metadata.FromIncomingContext(ctx)
	s.received <- md.Copy()
	return plogotlp.NewExportResponse(), nil
}

type exporterTestMetricsServer struct {
	pmetricotlp.UnimplementedGRPCServer
	received chan metadata.MD
	count    atomic.Int32
}

func (s *exporterTestMetricsServer) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	s.count.Add(int32(req.Metrics().DataPointCount()))
	md, _ := metadata.FromIncomingContext(ctx)
	s.received <- md.Copy()
	return pmetricotlp.NewExportResponse(), nil
}

func newExporterTestServer(t *testing.T) (string, *exporterTestLogsServer, *exporterTestMetricsServer) {
	t.Helper()
	t.Setenv("OCTELIUM_ENABLE_SPIFFE_CSI", "false")
	t.Setenv("SPIFFE_ENDPOINT_SOCKET", "")
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	logs := &exporterTestLogsServer{received: make(chan metadata.MD, 4)}
	metrics := &exporterTestMetricsServer{received: make(chan metadata.MD, 4)}
	srv := grpc.NewServer()
	plogotlp.RegisterGRPCServer(srv, logs)
	pmetricotlp.RegisterGRPCServer(srv, metrics)
	go srv.Serve(lis)
	t.Cleanup(func() { srv.Stop(); lis.Close() })
	return lis.Addr().String(), logs, metrics
}

func exporterTestLogs() plog.Logs {
	ret := plog.NewLogs()
	ret.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("exporter-test")
	return ret
}

func exporterTestMetrics() pmetric.Metrics {
	ret := pmetric.NewMetrics()
	m := ret.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	m.SetName("exporter.test")
	m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(9)
	return ret
}

func TestExporterConfigValidationAndEndpointSanitization(t *testing.T) {
	for _, tc := range []struct{ endpoint, want string }{
		{"localhost:4317", "localhost:4317"},
		{"http://localhost:4317", "localhost:4317"},
		{"https://localhost:4317", "localhost:4317"},
		{"dns://localhost:4317", "localhost:4317"},
		{"dns:///localhost:4317", "localhost:4317"},
		{"[::1]:4317", "[::1]:4317"},
	} {
		t.Run(tc.endpoint, func(t *testing.T) {
			cfg := createDefaultConfig().(*Config)
			cfg.Endpoint = tc.endpoint
			assert.Equal(t, tc.want, cfg.sanitizedEndpoint())
			require.NoError(t, cfg.Validate())
		})
	}
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Endpoint = "" },
		func(c *Config) { c.Endpoint = "https://" },
		func(c *Config) { c.MaxCallRecvMsgSizeMiB = -1 },
		func(c *Config) { c.MaxCallSendMsgSizeMiB = -1 },
		func(c *Config) { c.Compression = "snappy" },
	} {
		cfg := createDefaultConfig().(*Config)
		cfg.Endpoint = "localhost:4317"
		mutate(cfg)
		assert.Error(t, cfg.Validate())
	}
	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint, cfg.Compression = "localhost:4317", ""
	cfg.MaxCallRecvMsgSizeMiB, cfg.MaxCallSendMsgSizeMiB = 0, 0
	require.NoError(t, cfg.Validate())
}

func TestExporterEndpointTelemetryAttributes(t *testing.T) {
	for _, tc := range []struct {
		endpoint, address string
		port              int
	}{
		{"https://localhost:4317", "localhost", 4317},
		{"dns:///[::1]:4317", "::1", 4317},
		{"localhost", "localhost", 0},
		{"localhost:invalid", "localhost", 0},
	} {
		attributes := endpointAttributes(&Config{Endpoint: tc.endpoint})
		assert.Equal(t, "server.address", string(attributes[0].Key))
		assert.Equal(t, tc.address, attributes[0].Value.AsString())
		if tc.port > 0 {
			require.Len(t, attributes, 2)
			assert.Equal(t, "server.port", string(attributes[1].Key))
			assert.EqualValues(t, tc.port, attributes[1].Value.AsInt64())
		} else {
			assert.Len(t, attributes, 1)
		}
	}
}

func TestExporterStartHeadersCompressionAndBothSignals(t *testing.T) {
	endpoint, logs, metrics := newExporterTestServer(t)
	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint, cfg.WaitForReady = "http://"+endpoint, true
	cfg.Headers = map[string]string{"X-Test": "before", "Authorization": "Bearer test-token"}
	set := exportertest.NewNopSettings(componentType)
	set.BuildInfo = component.BuildInfo{Description: "collector-test", Version: "1.2.3"}
	e := newExporter(cfg, set)
	require.NoError(t, e.start(t.Context(), nil))
	t.Cleanup(func() { e.shutdown(context.Background()) })
	cfg.Headers["X-Test"] = "after"
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	require.NoError(t, e.pushLogs(ctx, exporterTestLogs()))
	require.NoError(t, e.pushMetrics(ctx, exporterTestMetrics()))
	assert.EqualValues(t, 1, logs.count.Load())
	assert.EqualValues(t, 1, metrics.count.Load())
	for _, channel := range []chan metadata.MD{logs.received, metrics.received} {
		md := <-channel
		assert.Equal(t, []string{"before"}, md.Get("x-test"))
		assert.Equal(t, []string{"Bearer test-token"}, md.Get("authorization"))
		assert.Contains(t, strings.Join(md.Get("user-agent"), " "), "collector-test/1.2.3")
	}
	assert.Equal(t, "after", cfg.Headers["X-Test"])
}

func TestExporterPushRequiresStart(t *testing.T) {
	e := newExporter(createDefaultConfig().(*Config), exportertest.NewNopSettings(componentType))
	assert.ErrorContains(t, e.pushLogs(t.Context(), exporterTestLogs()), "not started")
	assert.ErrorContains(t, e.pushMetrics(t.Context(), exporterTestMetrics()), "not started")
	require.NoError(t, e.shutdown(t.Context()))
}

func TestExporterRetryClassification(t *testing.T) {
	retryable := map[codes.Code]bool{
		codes.Canceled: true, codes.DeadlineExceeded: true, codes.Aborted: true,
		codes.OutOfRange: true, codes.Unavailable: true, codes.DataLoss: true,
	}
	for code := codes.Canceled; code <= codes.Unauthenticated; code++ {
		t.Run(code.String(), func(t *testing.T) {
			original := status.Error(code, "upstream failure")
			err := processError(original)
			require.Error(t, err)
			assert.Equal(t, !retryable[code], consumererror.IsPermanent(err))
			assert.ErrorIs(t, err, original)
			assert.Equal(t, retryable[code], shouldRetry(code, nil))
		})
	}
	assert.Nil(t, processError(nil))
	assert.True(t, consumererror.IsPermanent(processError(errors.New("unclassified failure"))))
}

func TestExporterRetryInfoAndThrottle(t *testing.T) {
	retry := &errdetails.RetryInfo{RetryDelay: durationpb.New(2 * time.Second)}
	st, err := status.New(codes.ResourceExhausted, "overloaded").WithDetails(&errdetails.ErrorInfo{Reason: "busy"}, retry)
	require.NoError(t, err)
	assert.Equal(t, 2*time.Second, getRetryInfo(st).RetryDelay.AsDuration())
	original := st.Err()
	err = processError(original)
	require.Error(t, err)
	assert.False(t, consumererror.IsPermanent(err))
	assert.ErrorIs(t, err, original)
	assert.ErrorContains(t, err, "Throttle (2s)")
	st, err = status.New(codes.ResourceExhausted, "overloaded").WithDetails(&errdetails.RetryInfo{})
	require.NoError(t, err)
	assert.Equal(t, st.Err(), processError(st.Err()))
	assert.Nil(t, getRetryInfo(nil))
	assert.Nil(t, getRetryInfo(status.New(codes.Unavailable, "no details")))
}

func TestExporterPartialSuccessReportsRejectedItems(t *testing.T) {
	core, observed := observer.New(zap.WarnLevel)
	settings := exportertest.NewNopSettings(componentType)
	settings.Logger = zap.New(core)
	e := newExporter(createDefaultConfig().(*Config), settings)
	e.logExporter = &exporterTestLogsClient{export: func(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
		assert.Equal(t, 1, req.Logs().LogRecordCount())
		resp := plogotlp.NewExportResponse()
		resp.PartialSuccess().SetRejectedLogRecords(1)
		resp.PartialSuccess().SetErrorMessage("rejected log")
		return resp, nil
	}}
	e.metricExporter = &exporterTestMetricsClient{export: func(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
		assert.Equal(t, 1, req.Metrics().DataPointCount())
		resp := pmetricotlp.NewExportResponse()
		resp.PartialSuccess().SetRejectedDataPoints(1)
		resp.PartialSuccess().SetErrorMessage("rejected point")
		return resp, nil
	}}
	require.NoError(t, e.pushLogs(t.Context(), exporterTestLogs()))
	require.NoError(t, e.pushMetrics(t.Context(), exporterTestMetrics()))
	entries := observed.All()
	require.Len(t, entries, 2)
	assert.EqualValues(t, 1, entries[0].ContextMap()["dropped_log_records"])
	assert.EqualValues(t, 1, entries[1].ContextMap()["dropped_data_points"])
}

func TestExporterPushPropagatesUpstreamErrors(t *testing.T) {
	for _, code := range []codes.Code{codes.Unavailable, codes.InvalidArgument} {
		failure := status.Error(code, "upstream failed")
		e := newExporter(createDefaultConfig().(*Config), exportertest.NewNopSettings(componentType))
		e.logExporter = &exporterTestLogsClient{export: func(context.Context, plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
			return plogotlp.ExportResponse{}, failure
		}}
		e.metricExporter = &exporterTestMetricsClient{export: func(context.Context, pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
			return pmetricotlp.ExportResponse{}, failure
		}}
		for _, err := range []error{e.pushLogs(t.Context(), exporterTestLogs()), e.pushMetrics(t.Context(), exporterTestMetrics())} {
			assert.ErrorIs(t, err, failure)
			assert.Equal(t, code == codes.InvalidArgument, consumererror.IsPermanent(err))
		}
	}
}

func TestExporterFactoriesDeliverWithoutQueues(t *testing.T) {
	endpoint, logs, metrics := newExporterTestServer(t)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Endpoint, cfg.WaitForReady = endpoint, true
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.RetryConfig.Enabled = false
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	lr, err := factory.CreateLogs(ctx, exportertest.NewNopSettings(componentType), cfg)
	require.NoError(t, err)
	mr, err := factory.CreateMetrics(ctx, exportertest.NewNopSettings(componentType), cfg)
	require.NoError(t, err)
	require.NoError(t, lr.Start(ctx, nil))
	t.Cleanup(func() { lr.Shutdown(context.Background()) })
	require.NoError(t, mr.Start(ctx, nil))
	t.Cleanup(func() { mr.Shutdown(context.Background()) })
	assert.False(t, lr.Capabilities().MutatesData)
	assert.False(t, mr.Capabilities().MutatesData)
	require.NoError(t, lr.ConsumeLogs(ctx, exporterTestLogs()))
	require.NoError(t, mr.ConsumeMetrics(ctx, exporterTestMetrics()))
	assert.EqualValues(t, 1, logs.count.Load())
	assert.EqualValues(t, 1, metrics.count.Load())
}

func TestExporterStartReturnsCredentialFailure(t *testing.T) {
	t.Setenv("OCTELIUM_ENABLE_SPIFFE_CSI", "true")
	t.Setenv("SPIFFE_ENDPOINT_SOCKET", "unix://"+filepath.Join(t.TempDir(), "missing.sock"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = "localhost:4317"
	e := newExporter(cfg, exportertest.NewNopSettings(componentType))
	require.Error(t, e.start(ctx, nil))
	assert.Nil(t, e.conn)
	assert.Nil(t, e.logExporter)
	assert.Nil(t, e.metricExporter)
}
