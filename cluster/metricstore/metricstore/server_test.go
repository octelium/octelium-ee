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
	"net"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func newTestGRPCServer(t *testing.T) *Server {
	t.Helper()
	testMode := ldflags.TestMode
	ldflags.TestMode = "true"
	t.Cleanup(func() { ldflags.TestMode = testMode })

	dbConfig, err := getMetricStoreDBConfig()
	require.NoError(t, err)
	db, err := openMetricStoreDB(dbConfig)
	require.NoError(t, err)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	return &Server{
		db:           db,
		dbConfig:     dbConfig,
		dbRecoverCh:  make(chan struct{}, 1),
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
		shutdownDone: make(chan struct{}),
	}
}

func isTestAddrFree(t *testing.T, addr string) bool {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	assert.NoError(t, listener.Close())
	return true
}

func TestMetricStoreIngestListenerOnlyServesOTLP(t *testing.T) {
	s := newTestGRPCServer(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.NoError(t, s.Run(ctx))

	ingestConn, err := grpc.NewClient(tstIngestAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, ingestConn.Close()) })
	mainConn, err := grpc.NewClient(tstAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, mainConn.Close()) })

	rpcCtx, rpcCancel := context.WithTimeout(ctx, 5*time.Second)
	defer rpcCancel()
	metrics, _ := newVigilResourceMetrics()
	_, err = pmetricotlp.NewGRPCClient(ingestConn).Export(rpcCtx, pmetricotlp.NewExportRequestFromMetrics(metrics))
	require.NoError(t, err)

	_, err = vmetricsv1.NewMetricsServiceClient(ingestConn).GetMetricsCapabilities(rpcCtx,
		&vmetricsv1.GetMetricsCapabilitiesRequest{})
	assert.Equal(t, codes.Unimplemented, status.Code(err))
	_, err = vmetricsv1.NewMetricsServiceClient(mainConn).GetMetricsCapabilities(rpcCtx,
		&vmetricsv1.GetMetricsCapabilitiesRequest{})
	require.NoError(t, err)

	cancel()
	s.WaitForShutdown()
	assert.True(t, isTestAddrFree(t, tstAddr))
	assert.True(t, isTestAddrFree(t, tstIngestAddr))
}

func TestMetricStoreIngestListenerBindFailure(t *testing.T) {
	s := newTestGRPCServer(t)
	listener, err := net.Listen("tcp", tstIngestAddr)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, listener.Close()) })

	err = s.Run(t.Context())
	require.Error(t, err)
	var netErr *net.OpError
	require.ErrorAs(t, err, &netErr)
	assert.Equal(t, "listen", netErr.Op)
	assert.True(t, isTestAddrFree(t, tstAddr))
}
