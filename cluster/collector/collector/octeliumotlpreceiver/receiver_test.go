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
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func receiverTestLogs() plog.Logs {
	ret := plog.NewLogs()
	ret.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("receiver-test")
	return ret
}

func receiverTestMetrics() pmetric.Metrics {
	ret := pmetric.NewMetrics()
	metric := ret.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("receiver.test")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(7)
	return ret
}

func newReceiverTest(t *testing.T) *octeliumOTLPReceiver {
	t.Helper()
	t.Setenv("OCTELIUM_ENABLE_SPIFFE_CSI", "false")
	t.Setenv("SPIFFE_ENDPOINT_SOCKET", "")
	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = "127.0.0.1:0"
	r, err := loadOrCreateReceiver(receivertest.NewNopSettings(componentType), cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, r.Shutdown(ctx))
	})
	return r
}

func receiverTestConn(t *testing.T, r *octeliumOTLPReceiver) *grpc.ClientConn {
	t.Helper()
	r.mu.RLock()
	endpoint := r.listener.Addr().String()
	r.mu.RUnlock()
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestReceiverConfigValidation(t *testing.T) {
	require.NoError(t, createDefaultConfig().(*Config).Validate())
	for _, tc := range []struct {
		name string
		set  func(*Config)
	}{
		{"Endpoint", func(c *Config) { c.Endpoint = "" }},
		{"ReceiveSize", func(c *Config) { c.MaxRecvMsgSizeMiB = -1 }},
		{"Streams", func(c *Config) { c.MaxConcurrentStreams = 0 }},
		{"ReadBuffer", func(c *Config) { c.ReadBufferSize = -1 }},
		{"WriteBuffer", func(c *Config) { c.WriteBufferSize = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := createDefaultConfig().(*Config)
			tc.set(cfg)
			assert.Error(t, cfg.Validate())
		})
	}
	cfg := createDefaultConfig().(*Config)
	cfg.MaxRecvMsgSizeMiB, cfg.ReadBufferSize, cfg.WriteBufferSize = 0, 0, 0
	require.NoError(t, cfg.Validate())
	assert.Equal(t, 16<<20, (&octeliumOTLPReceiver{cfg: cfg}).maxRecvMsgSize())
	cfg.MaxConcurrentStreams = 0
	assert.EqualValues(t, 1024, (&octeliumOTLPReceiver{cfg: cfg}).maxConcurrentStreams())
	cfg.MaxRecvMsgSizeMiB, cfg.MaxConcurrentStreams = 2, 17
	assert.Equal(t, 2<<20, (&octeliumOTLPReceiver{cfg: cfg}).maxRecvMsgSize())
	assert.EqualValues(t, 17, (&octeliumOTLPReceiver{cfg: cfg}).maxConcurrentStreams())
}

func TestReceiverRegistryIsolationAndValidation(t *testing.T) {
	settings := receivertest.NewNopSettings(componentType)
	_, err := loadOrCreateReceiver(settings, &struct{}{})
	require.Error(t, err)
	_, err = loadOrCreateReceiver(settings, &Config{})
	require.Error(t, err)
	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = "127.0.0.1:0"
	first, err := loadOrCreateReceiver(settings, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { first.Shutdown(t.Context()) })
	second, err := loadOrCreateReceiver(settings, cfg)
	require.NoError(t, err)
	assert.Same(t, first, second)
	other, err := loadOrCreateReceiver(receivertest.NewNopSettings(componentType), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { other.Shutdown(t.Context()) })
	assert.NotSame(t, first, other)
	require.NoError(t, first.Shutdown(t.Context()))
	replacement, err := loadOrCreateReceiver(settings, cfg)
	require.NoError(t, err)
	t.Cleanup(func() { replacement.Shutdown(t.Context()) })
	assert.NotSame(t, first, replacement)
	require.NoError(t, first.Shutdown(t.Context()))
	stored, found := receiverRegistry.Load(replacement.registryKey)
	require.True(t, found)
	assert.Same(t, replacement, stored)
	assert.False(t, replacement.Capabilities().MutatesData)
}

func TestReceiverFactorySharesSignalsAndStopsOnce(t *testing.T) {
	t.Setenv("OCTELIUM_ENABLE_SPIFFE_CSI", "false")
	t.Setenv("SPIFFE_ENDPOINT_SOCKET", "")
	factory := NewFactory()
	settings := receivertest.NewNopSettings(componentType)
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Endpoint = "127.0.0.1:0"
	cfg.WriteBufferSize = 1024
	logs := &consumertest.LogsSink{}
	metrics := &consumertest.MetricsSink{}
	lr, err := factory.CreateLogs(t.Context(), settings, cfg, logs)
	require.NoError(t, err)
	mr, err := factory.CreateMetrics(t.Context(), settings, cfg, metrics)
	require.NoError(t, err)
	require.Same(t, lr, mr)
	r := lr.(*octeliumOTLPReceiver)
	t.Cleanup(func() { r.Shutdown(context.Background()) })
	require.NoError(t, lr.Start(t.Context(), nil))
	listener := r.listener
	require.NoError(t, mr.Start(t.Context(), nil))
	assert.Same(t, listener, r.listener)
	conn := receiverTestConn(t, r)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	_, err = plogotlp.NewGRPCClient(conn).Export(ctx, plogotlp.NewExportRequestFromLogs(receiverTestLogs()))
	require.NoError(t, err)
	_, err = pmetricotlp.NewGRPCClient(conn).Export(ctx, pmetricotlp.NewExportRequestFromMetrics(receiverTestMetrics()))
	require.NoError(t, err)
	assert.Equal(t, 1, logs.LogRecordCount())
	assert.Equal(t, 1, metrics.DataPointCount())
	require.NoError(t, lr.Shutdown(ctx))
	assert.False(t, r.started)
	_, err = pmetricotlp.NewGRPCClient(conn).Export(ctx, pmetricotlp.NewExportRequestFromMetrics(receiverTestMetrics()), grpc.WaitForReady(false))
	require.Error(t, err)
	assert.Equal(t, 1, metrics.DataPointCount())
	require.NoError(t, mr.Shutdown(ctx))
	assert.False(t, r.started)
	assert.Nil(t, r.getLogsConsumer())
	assert.Nil(t, r.getMetricsConsumer())
	_, found := receiverRegistry.Load(r.registryKey)
	assert.False(t, found)
}

func TestReceiverBindFailureCanBeRetried(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	r := newReceiverTest(t)
	r.cfg.Endpoint = listener.Addr().String()
	require.Error(t, r.Start(t.Context(), nil))
	assert.False(t, r.started)
	require.NoError(t, listener.Close())
	require.NoError(t, r.Start(t.Context(), nil))
	assert.True(t, r.started)
}

func TestReceiverEmptyAndMissingPipelines(t *testing.T) {
	r := newReceiverTest(t)
	_, err := (&logsServer{parent: r}).Export(t.Context(), plogotlp.NewExportRequest())
	require.NoError(t, err)
	_, err = (&metricsServer{parent: r}).Export(t.Context(), pmetricotlp.NewExportRequest())
	require.NoError(t, err)
	_, err = (&logsServer{parent: r}).Export(t.Context(), plogotlp.NewExportRequestFromLogs(receiverTestLogs()))
	assert.Equal(t, codes.Unavailable, status.Code(err))
	_, err = (&metricsServer{parent: r}).Export(t.Context(), pmetricotlp.NewExportRequestFromMetrics(receiverTestMetrics()))
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestReceiverConsumerErrorsAndUnchangedPayloads(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		code codes.Code
	}{
		{"Success", nil, codes.OK},
		{"Transient", errors.New("queue full"), codes.Unavailable},
		{"Permanent", consumererror.NewPermanent(errors.New("invalid payload")), codes.Internal},
		{"ExplicitStatus", status.Error(codes.ResourceExhausted, "retry later"), codes.ResourceExhausted},
		{"WrappedStatus", fmt.Errorf("wrapped: %w", status.Error(codes.InvalidArgument, "invalid")), codes.InvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newReceiverTest(t)
			var logsCount, metricsCount int
			logs, err := consumer.NewLogs(func(ctx context.Context, data plog.Logs) error { logsCount += data.LogRecordCount(); return tc.err })
			require.NoError(t, err)
			metrics, err := consumer.NewMetrics(func(ctx context.Context, data pmetric.Metrics) error {
				metricsCount += data.DataPointCount()
				return tc.err
			})
			require.NoError(t, err)
			r.registerLogsConsumer(logs)
			r.registerMetricsConsumer(metrics)
			ld, md := receiverTestLogs(), receiverTestMetrics()
			_, err = (&logsServer{parent: r}).Export(t.Context(), plogotlp.NewExportRequestFromLogs(ld))
			assert.Equal(t, tc.code, status.Code(err))
			_, err = (&metricsServer{parent: r}).Export(t.Context(), pmetricotlp.NewExportRequestFromMetrics(md))
			assert.Equal(t, tc.code, status.Code(err))
			assert.Equal(t, 1, logsCount)
			assert.Equal(t, 1, metricsCount)
			assert.Equal(t, "receiver-test", ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Str())
			assert.EqualValues(t, 7, md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Gauge().DataPoints().At(0).IntValue())
		})
	}
	assert.Nil(t, grpcStatusFromConsumerError(nil))
}

func TestReceiverRejectsOversizedGRPCMessages(t *testing.T) {
	r := newReceiverTest(t)
	r.cfg.MaxRecvMsgSizeMiB = 1
	var calls atomic.Int32
	logs, err := consumer.NewLogs(func(context.Context, plog.Logs) error { calls.Add(1); return nil })
	require.NoError(t, err)
	r.registerLogsConsumer(logs)
	require.NoError(t, r.Start(t.Context(), nil))
	data := receiverTestLogs()
	data.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().SetStr(strings.Repeat("x", (1<<20)+1024))
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	_, err = plogotlp.NewGRPCClient(receiverTestConn(t, r)).Export(ctx, plogotlp.NewExportRequestFromLogs(data))
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.Zero(t, calls.Load())
}

func TestReceiverShutdownCancelsActiveRPC(t *testing.T) {
	r := newReceiverTest(t)
	started := make(chan struct{})
	logs, err := consumer.NewLogs(func(ctx context.Context, _ plog.Logs) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	require.NoError(t, err)
	r.registerLogsConsumer(logs)
	require.NoError(t, r.Start(t.Context(), nil))
	conn := receiverTestConn(t, r)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := plogotlp.NewGRPCClient(conn).Export(ctx, plogotlp.NewExportRequestFromLogs(receiverTestLogs()))
		done <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("receiver did not start the RPC")
	}
	shutdownCtx, stop := context.WithCancel(t.Context())
	stop()
	require.NoError(t, r.Shutdown(shutdownCtx))
	select {
	case err := <-done:
		require.Error(t, err)
	case <-ctx.Done():
		t.Fatal("receiver did not stop the RPC")
	}
}

func TestReceiverShutdownStopsIntakeBeforeDownstreamProcessors(t *testing.T) {
	r := newReceiverTest(t)
	var stopped atomic.Bool
	entered, release := make(chan struct{}), make(chan struct{})
	finish := sync.OnceFunc(func() { close(release) })
	defer finish()
	metrics, err := consumer.NewMetrics(func(context.Context, pmetric.Metrics) error {
		if stopped.Load() {
			close(entered)
			<-release
		}
		return nil
	})
	require.NoError(t, err)
	r.registerLogsConsumer(&consumertest.LogsSink{})
	r.registerMetricsConsumer(metrics)
	require.NoError(t, r.Start(t.Context(), nil))
	require.NoError(t, r.Start(t.Context(), nil))
	conn := receiverTestConn(t, r)
	client := pmetricotlp.NewGRPCClient(conn)
	_, err = client.Export(t.Context(), pmetricotlp.NewExportRequestFromMetrics(receiverTestMetrics()))
	require.NoError(t, err)

	require.NoError(t, r.Shutdown(t.Context()))
	stopped.Store(true)
	requestDone := make(chan error, 1)
	go func() {
		_, err := client.Export(t.Context(), pmetricotlp.NewExportRequestFromMetrics(receiverTestMetrics()), grpc.WaitForReady(false))
		requestDone <- err
	}()
	select {
	case <-entered:
	case err := <-requestDone:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("request did not finish")
	}

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- r.Shutdown(context.Background()) }()
	select {
	case err := <-shutdownDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		assert.Fail(t, "shared receiver waited for a request sent to a stopped downstream processor")
		finish()
		require.NoError(t, <-shutdownDone)
	}
	select {
	case <-entered:
		t.Error("shared receiver accepted a request after downstream shutdown")
	default:
	}
}

func TestReceiverCredentialFailureDoesNotRetainStartReference(t *testing.T) {
	r := newReceiverTest(t)
	t.Setenv("OCTELIUM_ENABLE_SPIFFE_CSI", "true")
	t.Setenv("SPIFFE_ENDPOINT_SOCKET", "unix://"+filepath.Join(t.TempDir(), "missing.sock"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.Error(t, r.Start(ctx, nil))
	assert.False(t, r.started)
	assert.Nil(t, r.serverGRPC)
}
