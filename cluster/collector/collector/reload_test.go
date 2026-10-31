package collector

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc"
)

func TestServerReloadWithConcurrentTrafficAndRecoveringMetricstore(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logs := newOTLPGRPCSink(t)
	t.Cleanup(logs.close)
	metrics := &otlpGRPCSink{port: tests.GetPort()}
	t.Cleanup(metrics.close)
	srv := &Server{
		octeliumC:                   tst.C.OcteliumC,
		receiverPort:                tests.GetPort(),
		internalLogstoreEndpoint:    logs.endpoint(),
		internalMetricstoreEndpoint: metrics.endpoint(),
	}
	require.NoError(t, srv.Run(ctx))
	producerCtx, stop := context.WithCancel(ctx)
	var producers sync.WaitGroup
	t.Cleanup(func() {
		stop()
		producers.Wait()
		cancel()
		shutdownCollector(t, srv)
	})
	require.Eventually(t, func() bool {
		return srv.collector.GetState() == otelcol.StateRunning
	}, 5*time.Second, 10*time.Millisecond)
	for _, export := range []func(context.Context, int) error{exportTestLog, exportTestMetric} {
		producers.Go(func() {
			for producerCtx.Err() == nil {
				requestCtx, done := context.WithTimeout(producerCtx, time.Second)
				_ = export(requestCtx, srv.receiverPort)
				done()
				select {
				case <-producerCtx.Done():
					return
				case <-time.After(5 * time.Millisecond):
				}
			}
		})
	}
	require.Eventually(t, func() bool {
		return logs.logRecordCount() > 0
	}, 5*time.Second, 10*time.Millisecond)
	srv.p.sendUpdate()
	require.Eventually(t, func() bool {
		return srv.collector.GetState() == otelcol.StateClosing
	}, 3*time.Second, 10*time.Millisecond)
	time.Sleep(3 * time.Second)
	listener, err := net.Listen("tcp", metrics.endpoint())
	require.NoError(t, err)
	metrics.lis = listener
	metrics.srv = grpc.NewServer()
	pmetricotlp.RegisterGRPCServer(metrics.srv, &otlpGRPCMetricsSink{sink: metrics})
	go metrics.srv.Serve(listener)
	require.Eventually(t, func() bool {
		return srv.collector.GetState() == otelcol.StateRunning
	}, 20*time.Second, 10*time.Millisecond)
	before := logs.logRecordCount()
	require.Eventually(t, func() bool {
		return logs.logRecordCount() > before && metrics.metricPointCount() > 0
	}, 5*time.Second, 10*time.Millisecond)
}
