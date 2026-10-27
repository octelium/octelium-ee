// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package collector

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuredataexplorerexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuremonitorexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/influxdbexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logzioexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/splunkhecexporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

type otlpSink struct {
	port int
	srv  *http.Server
	lis  net.Listener

	mu sync.Mutex

	logReqs int
	logRecs int
	logAuth string

	metricReqs int
	metricPts  int
	metricAuth string
}

func newOTLPSink(t *testing.T) *otlpSink {
	t.Helper()

	s := &otlpSink{
		port: tests.GetPort(),
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	require.NoError(t, err)
	s.lis = lis

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/logs", s.handleLogs)
	mux.HandleFunc("/v1/metrics", s.handleMetrics)

	s.srv = &http.Server{
		Handler: mux,
	}

	go func() {
		err := s.srv.Serve(lis)
		if err != nil && err != http.ErrServerClosed && !strings.Contains(err.Error(), "use of closed network connection") {
			zap.L().Warn("OTLP test sink exited", zap.Error(err))
		}
	}()

	return s
}

func (s *otlpSink) endpoint() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

func (s *otlpSink) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	body, err := readOTLPBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req := plogotlp.NewExportRequest()
	if err := req.UnmarshalProto(body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.logReqs++
	s.logRecs += req.Logs().LogRecordCount()
	s.logAuth = r.Header.Get("Authorization")
	s.mu.Unlock()

	resp, err := plogotlp.NewExportResponse().MarshalProto()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

func (s *otlpSink) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	body, err := readOTLPBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req := pmetricotlp.NewExportRequest()
	if err := req.UnmarshalProto(body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.metricReqs++
	s.metricPts += req.Metrics().DataPointCount()
	s.metricAuth = r.Header.Get("Authorization")
	s.mu.Unlock()

	resp, err := pmetricotlp.NewExportResponse().MarshalProto()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

func (s *otlpSink) logRecordCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logRecs
}

func (s *otlpSink) metricPointCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.metricPts
}

func (s *otlpSink) logAuthHeader() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logAuth
}

func (s *otlpSink) metricAuthHeader() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.metricAuth
}

func (s *otlpSink) close() {
	if s.srv != nil {
		_ = s.srv.Close()
	}
	if s.lis != nil {
		_ = s.lis.Close()
	}
}

func basicAuthHeader(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

func createSecret(
	ctx context.Context,
	t *testing.T,
	octeliumC octeliumc.ClientInterface,
	value string,
) *enterprisev1.Secret {
	t.Helper()

	sec, err := octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   &enterprisev1.Secret_Spec{},
		Status: &enterprisev1.Secret_Status{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: value,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, sec)
	require.NotNil(t, sec.Metadata)
	require.NotEmpty(t, sec.Metadata.Name)

	return sec
}

func otlpHTTPBasicAuth(username, secretName string) *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth {
	return &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{
		Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_{
			Basic: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic{
				Username: username,
				Password: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password{
					Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password_FromSecret{
						FromSecret: secretName,
					},
				},
			},
		},
	}
}

func createOTLPHTTPExporter(
	ctx context.Context,
	t *testing.T,
	octeliumC octeliumc.ClientInterface,
	endpoint string,
	auth *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth,
	disabled bool,
) *enterprisev1.CollectorExporter {
	t.Helper()

	exp, err := octeliumC.EnterpriseC().CreateCollectorExporter(ctx, &enterprisev1.CollectorExporter{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.CollectorExporter_Spec{
			IsDisabled: disabled,
			Type: &enterprisev1.CollectorExporter_Spec_OtlpHTTP{
				OtlpHTTP: &enterprisev1.CollectorExporter_Spec_OTLPHTTP{
					Endpoint: endpoint,
					Auth:     auth,
				},
			},
		},
		Status: &enterprisev1.CollectorExporter_Status{},
	})
	require.NoError(t, err)
	require.NotNil(t, exp)
	require.NotNil(t, exp.Metadata)
	require.NotEmpty(t, exp.Metadata.Name)

	return exp
}

func setCollectorPipelines(
	ctx context.Context,
	t *testing.T,
	octeliumC octeliumc.ClientInterface,
	pipelines []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline,
) {
	t.Helper()

	cc, err := octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, cc)
	require.NotNil(t, cc.Spec)

	cc.Spec.Collector = &enterprisev1.ClusterConfig_Spec_Collector{
		Pipelines: pipelines,
	}

	_, err = octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
	require.NoError(t, err)
}

func newTestCollectorServer(
	ctx context.Context,
	t *testing.T,
	octeliumC octeliumc.ClientInterface,
) *Server {
	t.Helper()

	internalLogstore := newOTLPGRPCSink(t)
	internalMetricstore := newOTLPGRPCSink(t)

	srv := &Server{
		octeliumC:                   octeliumC,
		receiverPort:                tests.GetPort(),
		internalLogstoreEndpoint:    internalLogstore.endpoint(),
		internalMetricstoreEndpoint: internalMetricstore.endpoint(),
	}

	require.NoError(t, srv.doRun(ctx))
	require.NotNil(t, srv.p)
	require.NotNil(t, srv.ccController)

	return srv
}

func newOTLPGRPCConn(ctx context.Context, port int) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		fmt.Sprintf("localhost:%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.DefaultConfig}),
	)
}

func exportTestLog(ctx context.Context, port int) error {
	conn, err := newOTLPGRPCConn(ctx, port)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := plogotlp.NewGRPCClient(conn)

	curLogs := plog.NewLogs()
	curLogs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	lr := curLogs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty()

	convertLogRecord(&corev1.AccessLog{
		Metadata: &metav1.LogMetadata{
			Id:        utilrand.GetRandomStringCanonical(18),
			CreatedAt: pbutils.Now(),
		},
	}, lr)

	_, err = client.Export(ctx, plogotlp.NewExportRequestFromLogs(curLogs))
	return err
}

func exportTestMetric(ctx context.Context, port int) error {
	conn, err := newOTLPGRPCConn(ctx, port)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pmetricotlp.NewGRPCClient(conn)

	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()

	metric := sm.Metrics().AppendEmpty()
	metric.SetName("octelium.collector.test.metric")

	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now().UTC()))
	dp.SetIntValue(1)

	_, err = client.Export(ctx, pmetricotlp.NewExportRequestFromMetrics(md))
	return err
}

func requireEventuallyExportLogToSink(
	ctx context.Context,
	t *testing.T,
	port int,
	sink *otlpSink,
) {
	t.Helper()

	require.Eventually(t, func() bool {
		if err := exportTestLog(ctx, port); err != nil {
			zap.L().Debug("Could not export test log yet", zap.Error(err))
			return false
		}

		return sink.logRecordCount() >= 1
	}, 15*time.Second, 200*time.Millisecond)
}

func requireEventuallyExportMetricToSink(
	ctx context.Context,
	t *testing.T,
	port int,
	sink *otlpSink,
) {
	t.Helper()

	require.Eventually(t, func() bool {
		if err := exportTestMetric(ctx, port); err != nil {
			zap.L().Debug("Could not export test metric yet", zap.Error(err))
			return false
		}

		return sink.metricPointCount() >= 1
	}, 15*time.Second, 200*time.Millisecond)
}

func shutdownCollector(t *testing.T, srv *Server) {
	t.Helper()

	if srv == nil || srv.collector == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		srv.collector.Shutdown()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("could not shutdown collector")
	}
}

func TestServerExportsLogsAndMetricsThroughSameReceiver(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	zap.ReplaceGlobals(logger)

	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC

	ctx, cancel := context.WithCancel(context.Background())

	internalLogstore := newOTLPGRPCSink(t)
	t.Cleanup(func() {
		internalLogstore.close()
	})

	internalMetricstore := newOTLPGRPCSink(t)
	t.Cleanup(func() {
		internalMetricstore.close()
	})

	srv := &Server{
		octeliumC:                   octeliumC,
		receiverPort:                tests.GetPort(),
		internalLogstoreEndpoint:    internalLogstore.endpoint(),
		internalMetricstoreEndpoint: internalMetricstore.endpoint(),
	}

	require.NoError(t, srv.doRun(ctx))
	require.NotNil(t, srv.p)
	require.NotNil(t, srv.ccController)

	t.Cleanup(func() {
		cancel()
		shutdownCollector(t, srv)
	})

	sink := newOTLPSink(t)
	t.Cleanup(func() {
		sink.close()
	})

	username := "123456"
	password := "octelium"

	sec := createSecret(ctx, t, octeliumC, password)

	exp := createOTLPHTTPExporter(
		ctx,
		t,
		octeliumC,
		sink.endpoint(),
		otlpHTTPBasicAuth(username, sec.Metadata.Name),
		false,
	)

	setCollectorPipelines(ctx, t, octeliumC, []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		{
			Name:      "pipeline-logs",
			Type:      enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS,
			Exporters: []string{exp.Metadata.Name},
		},
		{
			Name:      "pipeline-metrics",
			Type:      enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS,
			Exporters: []string{exp.Metadata.Name},
		},
	})

	srv.p.sendUpdate()

	requireEventuallyExportLogToSink(ctx, t, srv.p.port, sink)
	requireEventuallyExportMetricToSink(ctx, t, srv.p.port, sink)

	assert.Equal(t, basicAuthHeader(username, password), sink.logAuthHeader())
	assert.Equal(t, basicAuthHeader(username, password), sink.metricAuthHeader())

	require.Eventually(t, func() bool {
		return internalLogstore.logRecordCount() >= 1
	}, 15*time.Second, 200*time.Millisecond)

	require.Eventually(t, func() bool {
		return internalMetricstore.metricPointCount() >= 1
	}, 15*time.Second, 200*time.Millisecond)
}

func TestServerReloadRoutesToUpdatedExporter(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	zap.ReplaceGlobals(logger)

	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := newTestCollectorServer(ctx, t, octeliumC)
	t.Cleanup(func() {
		cancel()
		shutdownCollector(t, srv)
	})

	sinkA := newOTLPSink(t)
	t.Cleanup(func() {
		sinkA.close()
	})

	sinkB := newOTLPSink(t)
	t.Cleanup(func() {
		sinkB.close()
	})

	expA := createOTLPHTTPExporter(ctx, t, octeliumC, sinkA.endpoint(), nil, false)
	expB := createOTLPHTTPExporter(ctx, t, octeliumC, sinkB.endpoint(), nil, false)

	setCollectorPipelines(ctx, t, octeliumC, []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		{
			Name:      "reload-logs",
			Type:      enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS,
			Exporters: []string{expA.Metadata.Name},
		},
	})

	srv.p.sendUpdate()

	requireEventuallyExportLogToSink(ctx, t, srv.p.port, sinkA)

	sinkABefore := sinkA.logRecordCount()

	setCollectorPipelines(ctx, t, octeliumC, []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		{
			Name:      "reload-logs",
			Type:      enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS,
			Exporters: []string{expB.Metadata.Name},
		},
	})

	srv.p.sendUpdate()

	require.Eventually(t, func() bool {
		err := exportTestLog(ctx, srv.p.port)
		if err != nil {
			zap.L().Debug("Could not export test log after reload yet", zap.Error(err))
			return false
		}
		return sinkB.logRecordCount() >= 1
	}, 15*time.Second, 200*time.Millisecond)

	assert.GreaterOrEqual(t, sinkA.logRecordCount(), sinkABefore)
	assert.GreaterOrEqual(t, sinkB.logRecordCount(), 1)
}

func TestGetExporterElasticsearchBasicAuthDoesNotPanic(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC

	p := &provider{
		octeliumC:  octeliumC,
		schemeName: "octelium-api",
		port:       tests.GetPort(),
	}

	password := "es-pass-" + utilrand.GetRandomStringCanonical(6)
	username := "es-user"
	sec := createSecret(ctx, t, octeliumC, password)

	esExp := &enterprisev1.CollectorExporter{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.CollectorExporter_Spec{
			Type: &enterprisev1.CollectorExporter_Spec_Elasticsearch_{
				Elasticsearch: &enterprisev1.CollectorExporter_Spec_Elasticsearch{
					Endpoint: "http://localhost:9200",
					Auth: &enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth{
						Type: &enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_Basic_{
							Basic: &enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_Basic{
								User: username,
								Password: &enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_Basic_Password{
									Type: &enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_Basic_Password_FromSecret{
										FromSecret: sec.Metadata.Name,
									},
								},
							},
						},
					},
				},
			},
		},
		Status: &enterprisev1.CollectorExporter_Status{},
	}

	expInfo, err := p.getExporter(ctx, esExp)
	require.NoError(t, err)
	require.NotNil(t, expInfo)

	require.True(t, expInfo.HasLogs)
	require.True(t, expInfo.HasMetrics)

	cfg := expInfo.exporterMap
	require.NotNil(t, cfg)

	assert.Equal(t, "http://localhost:9200", cfg["endpoint"])
	assert.Equal(t, username, cfg["user"])
	assert.Equal(t, password, cfg["password"])

	_, hasHeaders := cfg["headers"]
	assert.False(t, hasHeaders)
}

func TestGetConfigSkipsDisabledAndUnresolvableExporters(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC

	p := &provider{
		octeliumC:  octeliumC,
		schemeName: "octelium-api",
		port:       tests.GetPort(),
	}

	working := createOTLPHTTPExporter(ctx, t, octeliumC, "http://localhost:1", nil, false)
	disabled := createOTLPHTTPExporter(ctx, t, octeliumC, "http://localhost:1", nil, true)
	broken := createOTLPHTTPExporter(ctx, t, octeliumC, "http://localhost:1",
		otlpHTTPBasicAuth("u", "nonexistent-secret-"+utilrand.GetRandomStringCanonical(6)), false)

	setCollectorPipelines(ctx, t, octeliumC, []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		{
			Name: "skiptest",
			Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS,
			Exporters: []string{
				working.Metadata.Name,
				disabled.Metadata.Name,
				broken.Metadata.Name,
			},
		},
	})

	cfg, err := p.getConfig(ctx)
	require.NoError(t, err)

	exporters, ok := cfg["exporters"].(map[string]any)
	require.True(t, ok)

	assert.Contains(t, exporters, p.getTypeName(working))
	assert.NotContains(t, exporters, p.getTypeName(disabled))
	assert.NotContains(t, exporters, p.getTypeName(broken))

	service, ok := cfg["service"].(map[string]any)
	require.True(t, ok)

	pipelines, ok := service["pipelines"].(map[string]any)
	require.True(t, ok)

	lp, ok := pipelines["logs/skiptest"].(map[string]any)
	require.True(t, ok)

	exporterNames, ok := lp["exporters"].([]string)
	require.True(t, ok)

	assert.Equal(t, []string{p.getTypeName(working)}, exporterNames)
}

func TestGetConfigExporterInBothPipelines(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC

	p := &provider{
		octeliumC:  octeliumC,
		schemeName: "octelium-api",
		port:       tests.GetPort(),
	}

	exp := createOTLPHTTPExporter(ctx, t, octeliumC, "http://localhost:1", nil, false)

	setCollectorPipelines(ctx, t, octeliumC, []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		{
			Name:      "lp",
			Type:      enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS,
			Exporters: []string{exp.Metadata.Name},
		},
		{
			Name:      "mp",
			Type:      enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS,
			Exporters: []string{exp.Metadata.Name},
		},
	})

	cfg, err := p.getConfig(ctx)
	require.NoError(t, err)

	service, ok := cfg["service"].(map[string]any)
	require.True(t, ok)

	pipelines, ok := service["pipelines"].(map[string]any)
	require.True(t, ok)

	assert.Contains(t, pipelines, "logs/lp")
	assert.Contains(t, pipelines, "metrics/mp")
	assert.Contains(t, pipelines, "logs")
	assert.Contains(t, pipelines, "metrics")

	lp, ok := pipelines["logs/lp"].(map[string]any)
	require.True(t, ok)

	lpExporters, ok := lp["exporters"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{p.getTypeName(exp)}, lpExporters)

	mp, ok := pipelines["metrics/mp"].(map[string]any)
	require.True(t, ok)

	mpExporters, ok := mp["exporters"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{p.getTypeName(exp)}, mpExporters)
}

func TestReadOTLPBodyRejectsInvalidGzip(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader("not-gzip"))
	require.NoError(t, err)
	req.Header.Set("Content-Encoding", "gzip")

	_, err = readOTLPBody(req)
	assert.Error(t, err)
}

func convertLogRecord(in *corev1.AccessLog, ret plog.LogRecord) {
	inMap := pbutils.MustConvertToMap(in)

	ret.SetTimestamp(pcommon.NewTimestampFromTime(in.Metadata.CreatedAt.AsTime()))
	ret.SetObservedTimestamp(pcommon.NewTimestampFromTime(in.Metadata.CreatedAt.AsTime()))
	ret.SetSeverityNumber(plog.SeverityNumberInfo)
	ret.SetSeverityText(plog.SeverityNumberInfo.String())
	ret.Body().SetEmptyMap().FromRaw(inMap)
}

func readOTLPBody(r *http.Request) ([]byte, error) {
	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		defer gz.Close()

		return io.ReadAll(gz)
	}

	return io.ReadAll(r.Body)
}

type otlpGRPCSink struct {
	port int
	lis  net.Listener
	srv  *grpc.Server

	mu sync.Mutex

	logReqs int
	logRecs int

	metricReqs int
	metricPts  int
}

func newOTLPGRPCSink(t *testing.T) *otlpGRPCSink {
	t.Helper()

	s := &otlpGRPCSink{
		port: tests.GetPort(),
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	require.NoError(t, err)

	s.lis = lis
	s.srv = grpc.NewServer(
		grpc.MaxRecvMsgSize(16<<20),
		grpc.MaxSendMsgSize(16<<20),
	)

	plogotlp.RegisterGRPCServer(s.srv, &otlpGRPCLogsSink{sink: s})
	pmetricotlp.RegisterGRPCServer(s.srv, &otlpGRPCMetricsSink{sink: s})

	go func() {
		err := s.srv.Serve(lis)
		if err != nil &&
			!errors.Is(err, grpc.ErrServerStopped) &&
			!errors.Is(err, net.ErrClosed) {
			zap.L().Warn("OTLP gRPC test sink exited", zap.Error(err))
		}
	}()

	return s
}

func (s *otlpGRPCSink) endpoint() string {
	return fmt.Sprintf("localhost:%d", s.port)
}

func (s *otlpGRPCSink) logRecordCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.logRecs
}

func (s *otlpGRPCSink) metricPointCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.metricPts
}

func (s *otlpGRPCSink) close() {
	if s.srv != nil {
		s.srv.Stop()
	}

	if s.lis != nil {
		_ = s.lis.Close()
	}
}

type otlpGRPCLogsSink struct {
	plogotlp.UnimplementedGRPCServer
	sink *otlpGRPCSink
}

func (s *otlpGRPCLogsSink) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	logs := req.Logs()

	s.sink.mu.Lock()
	s.sink.logReqs++
	s.sink.logRecs += logs.LogRecordCount()
	s.sink.mu.Unlock()

	return plogotlp.NewExportResponse(), nil
}

type otlpGRPCMetricsSink struct {
	pmetricotlp.UnimplementedGRPCServer
	sink *otlpGRPCSink
}

func (s *otlpGRPCMetricsSink) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	metrics := req.Metrics()

	s.sink.mu.Lock()
	s.sink.metricReqs++
	s.sink.metricPts += metrics.DataPointCount()
	s.sink.mu.Unlock()

	return pmetricotlp.NewExportResponse(), nil
}

func collectorExporterFromJSON(t *testing.T, spec string) *enterprisev1.CollectorExporter {
	t.Helper()
	ret := &enterprisev1.CollectorExporter{}
	require.NoError(t, protojson.Unmarshal([]byte(`{"metadata":{"name":"mapping-exporter","uid":"mapping-uid"},"spec":`+spec+`}`), ret))
	return ret
}

func newCollectorProviderTest(t *testing.T) *provider {
	t.Helper()
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	_, err = tst.C.OcteliumC.EnterpriseC().CreateSecret(t.Context(), &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: "mapping-credential"}, Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: "credential-value"}},
	})
	require.NoError(t, err)
	return &provider{octeliumC: tst.C.OcteliumC, schemeName: "octelium-api", port: tests.GetPort()}
}

func collectorConfigValue(t *testing.T, configuration map[string]any, path string) any {
	t.Helper()
	var value any = configuration
	for _, field := range strings.Split(path, ".") {
		object, ok := value.(map[string]any)
		require.True(t, ok, path)
		value, ok = object[field]
		require.True(t, ok, path)
	}
	return value
}

func TestGetExporterOTLPCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"otlp":{"endpoint":"otel.example:4317","headers":[{"key":"X-Tenant","value":"tenant"}],"auth":{"bearer":{"fromSecret":"mapping-credential"}},"tls":{"insecure":true,"insecureSkipVerify":true,"serverNameOverride":"otel.example","caPEM":"test-ca"},"compression":"ZSTD","timeout":{"seconds":3},"waitForReady":true,"authority":"authority.example","userAgent":"test-agent","balancerName":"round_robin","readBufferSize":1024,"writeBufferSize":2048,"keepalive":{"time":{"minutes":1},"timeout":{"seconds":2},"permitWithoutStream":true}}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "otel.example:4317", collectorConfigValue(t, info.exporterMap, "endpoint"))
	assert.EqualValues(t, "tenant", collectorConfigValue(t, info.exporterMap, "headers.X-Tenant"))
	assert.EqualValues(t, "Bearer credential-value", collectorConfigValue(t, info.exporterMap, "headers.Authorization"))
	assert.EqualValues(t, "test-ca", collectorConfigValue(t, info.exporterMap, "tls.ca_pem"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "tls.insecure"))
	assert.EqualValues(t, "otel.example", collectorConfigValue(t, info.exporterMap, "tls.server_name_override"))
	assert.EqualValues(t, "zstd", collectorConfigValue(t, info.exporterMap, "compression"))
	assert.EqualValues(t, "3s", collectorConfigValue(t, info.exporterMap, "timeout"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "wait_for_ready"))
	assert.EqualValues(t, "authority.example", collectorConfigValue(t, info.exporterMap, "authority"))
	assert.EqualValues(t, "test-agent", collectorConfigValue(t, info.exporterMap, "user_agent"))
	assert.EqualValues(t, "round_robin", collectorConfigValue(t, info.exporterMap, "balancer_name"))
	assert.EqualValues(t, 1024, collectorConfigValue(t, info.exporterMap, "read_buffer_size"))
	assert.EqualValues(t, 2048, collectorConfigValue(t, info.exporterMap, "write_buffer_size"))
	assert.EqualValues(t, "1m", collectorConfigValue(t, info.exporterMap, "keepalive.time"))
	assert.EqualValues(t, "2s", collectorConfigValue(t, info.exporterMap, "keepalive.timeout"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "keepalive.permit_without_stream"))
}

func TestGetExporterOTLPHTTPCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"otlpHTTP":{"endpoint":"https://otel.example","logsEndpoint":"https://logs.example/v1/logs","metricsEndpoint":"https://metrics.example/v1/metrics","headers":[{"key":"X-Tenant","value":"tenant"}],"auth":{"custom":{"header":"X-Auth","value":{"fromSecret":"mapping-credential"}}},"tls":{"insecureSkipVerify":true,"serverNameOverride":"otel.example","caPEM":"test-ca"},"encoding":"JSON","compression":"NONE","timeout":{"seconds":4},"readBufferSize":1024,"writeBufferSize":2048}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "https://otel.example", collectorConfigValue(t, info.exporterMap, "endpoint"))
	assert.EqualValues(t, "https://logs.example/v1/logs", collectorConfigValue(t, info.exporterMap, "logs_endpoint"))
	assert.EqualValues(t, "https://metrics.example/v1/metrics", collectorConfigValue(t, info.exporterMap, "metrics_endpoint"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "headers.X-Auth"))
	assert.EqualValues(t, "tenant", collectorConfigValue(t, info.exporterMap, "headers.X-Tenant"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "tls.insecure_skip_verify"))
	assert.EqualValues(t, "test-ca", collectorConfigValue(t, info.exporterMap, "tls.ca_pem"))
	assert.EqualValues(t, "json", collectorConfigValue(t, info.exporterMap, "encoding"))
	assert.EqualValues(t, "none", collectorConfigValue(t, info.exporterMap, "compression"))
	assert.EqualValues(t, "4s", collectorConfigValue(t, info.exporterMap, "timeout"))
	assert.EqualValues(t, 1024, collectorConfigValue(t, info.exporterMap, "read_buffer_size"))
	assert.EqualValues(t, 2048, collectorConfigValue(t, info.exporterMap, "write_buffer_size"))
}

func TestGetExporterPrometheusRemoteWriteCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"prometheusRemoteWrite":{"endpoint":"https://prometheus.example/api/v1/write","namespace":"octelium","headers":[{"key":"X-Tenant","value":"tenant"}],"externalLabels":[{"key":"cluster","value":"test"}],"auth":{"basic":{"username":"test-user","password":{"fromSecret":"mapping-credential"}}},"tls":{"insecureSkipVerify":true},"timeout":{"seconds":5},"remoteWriteQueue":{"enabled":false,"queueSize":100,"numConsumers":2},"resourceToTelemetryConversion":{"enabled":true,"excludeServiceAttributes":true},"targetInfo":{"enabled":false},"disableScopeInfo":true,"maxBatchSizeBytes":10000,"maxBatchRequestParallelism":2,"translationStrategy":"NO_TRANSLATION","sendMetadata":true}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, false, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "https://prometheus.example/api/v1/write", collectorConfigValue(t, info.exporterMap, "endpoint"))
	assert.EqualValues(t, "octelium", collectorConfigValue(t, info.exporterMap, "namespace"))
	assert.EqualValues(t, "Basic dGVzdC11c2VyOmNyZWRlbnRpYWwtdmFsdWU=", collectorConfigValue(t, info.exporterMap, "headers.Authorization"))
	assert.EqualValues(t, "test", collectorConfigValue(t, info.exporterMap, "external_labels.cluster"))
	assert.EqualValues(t, false, collectorConfigValue(t, info.exporterMap, "remote_write_queue.enabled"))
	assert.EqualValues(t, 100, collectorConfigValue(t, info.exporterMap, "remote_write_queue.queue_size"))
	assert.EqualValues(t, 2, collectorConfigValue(t, info.exporterMap, "remote_write_queue.num_consumers"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "resource_to_telemetry_conversion.enabled"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "resource_to_telemetry_conversion.exclude_service_attributes"))
	assert.EqualValues(t, false, collectorConfigValue(t, info.exporterMap, "target_info.enabled"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "disable_scope_info"))
	assert.EqualValues(t, 10000, collectorConfigValue(t, info.exporterMap, "max_batch_size_bytes"))
	assert.EqualValues(t, 2, collectorConfigValue(t, info.exporterMap, "max_batch_request_parallelism"))
	assert.EqualValues(t, "NoTranslation", collectorConfigValue(t, info.exporterMap, "translation_strategy"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "send_metadata"))
	assert.EqualValues(t, "5s", collectorConfigValue(t, info.exporterMap, "timeout"))
}

func TestGetExporterClickhouseCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"clickhouse":{"endpoint":"tcp://clickhouse.example:9000","username":"test-user","password":{"fromSecret":"mapping-credential"},"database":"telemetry","tls":{"insecureSkipVerify":true},"connectionParams":[{"key":"dial_timeout","value":"10s"}],"logsTableName":"logs","metricsTables":{"gauge":"gauges","sum":"sums","summary":"summaries","histogram":"histograms","exponentialHistogram":"exponential"},"ttl":{"days":2},"createSchema":false,"compression":"ZSTD","asyncInsert":false,"json":true,"clusterName":"test-cluster","tableEngine":{"name":"MergeTree","params":"()"},"timeout":{"seconds":6}}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "tcp://clickhouse.example:9000", collectorConfigValue(t, info.exporterMap, "endpoint"))
	assert.EqualValues(t, "test-user", collectorConfigValue(t, info.exporterMap, "username"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "password"))
	assert.EqualValues(t, "telemetry", collectorConfigValue(t, info.exporterMap, "database"))
	assert.EqualValues(t, "10s", collectorConfigValue(t, info.exporterMap, "connection_params.dial_timeout"))
	assert.EqualValues(t, "logs", collectorConfigValue(t, info.exporterMap, "logs_table_name"))
	assert.EqualValues(t, "gauges", collectorConfigValue(t, info.exporterMap, "metrics_tables.gauge.name"))
	assert.EqualValues(t, "sums", collectorConfigValue(t, info.exporterMap, "metrics_tables.sum.name"))
	assert.EqualValues(t, "summaries", collectorConfigValue(t, info.exporterMap, "metrics_tables.summary.name"))
	assert.EqualValues(t, "histograms", collectorConfigValue(t, info.exporterMap, "metrics_tables.histogram.name"))
	assert.EqualValues(t, "exponential", collectorConfigValue(t, info.exporterMap, "metrics_tables.exponential_histogram.name"))
	assert.EqualValues(t, "48h", collectorConfigValue(t, info.exporterMap, "ttl"))
	assert.EqualValues(t, false, collectorConfigValue(t, info.exporterMap, "create_schema"))
	assert.EqualValues(t, "zstd", collectorConfigValue(t, info.exporterMap, "compress"))
	assert.EqualValues(t, false, collectorConfigValue(t, info.exporterMap, "async_insert"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "json"))
	assert.EqualValues(t, "test-cluster", collectorConfigValue(t, info.exporterMap, "cluster_name"))
	assert.EqualValues(t, "MergeTree", collectorConfigValue(t, info.exporterMap, "table_engine.name"))
	assert.EqualValues(t, "()", collectorConfigValue(t, info.exporterMap, "table_engine.params"))
	assert.EqualValues(t, "6s", collectorConfigValue(t, info.exporterMap, "timeout"))
}

func TestGetExporterElasticsearchCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"elasticsearch":{"endpoint":"https://elastic.example","endpoints":["https://elastic.example","https://elastic2.example"],"cloudID":"test-cloud","pipeline":"ingest","logsIndex":"logs-test","metricsIndex":"metrics-test","headers":[{"key":"X-Tenant","value":"tenant"}],"timeout":{"seconds":7},"compression":"NONE","tls":{"insecureSkipVerify":true},"auth":{"apiKey":{"fromSecret":"mapping-credential"}}}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "https://elastic.example", collectorConfigValue(t, info.exporterMap, "endpoint"))
	assert.EqualValues(t, "test-cloud", collectorConfigValue(t, info.exporterMap, "cloudid"))
	assert.EqualValues(t, "ingest", collectorConfigValue(t, info.exporterMap, "pipeline"))
	assert.EqualValues(t, "logs-test", collectorConfigValue(t, info.exporterMap, "logs_index"))
	assert.EqualValues(t, "metrics-test", collectorConfigValue(t, info.exporterMap, "metrics_index"))
	assert.EqualValues(t, "tenant", collectorConfigValue(t, info.exporterMap, "headers.X-Tenant"))
	assert.EqualValues(t, "7s", collectorConfigValue(t, info.exporterMap, "timeout"))
	assert.EqualValues(t, "none", collectorConfigValue(t, info.exporterMap, "compression"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "tls.insecure_skip_verify"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "api_key"))
}

func TestGetExporterKafkaCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"kafka":{"brokers":["kafka.example:9092"],"protocolVersion":"2.8.0","clientID":"test-client","recordHeaders":[{"key":"tenant","value":"test"}],"timeout":{"seconds":8},"connIdleTimeout":{"minutes":1},"partitionLogsByResourceAttributes":true,"partitionMetricsByResourceAttributes":true,"logs":{"topic":"logs","encoding":"RAW"},"metrics":{"topic":"metrics","encoding":"OTLP_JSON"},"tls":{"insecure":true,"insecureSkipVerify":true},"auth":{"sasl":{"username":"test-user","password":{"fromSecret":"mapping-credential"},"mechanism":"SCRAM_SHA_512"}},"producer":{"maxMessageBytes":100000,"requiredAcks":-1,"compression":"ZSTD","flushMaxMessages":100,"allowAutoTopicCreation":true,"linger":{"milliseconds":10}}}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "2.8.0", collectorConfigValue(t, info.exporterMap, "protocol_version"))
	assert.EqualValues(t, "test-client", collectorConfigValue(t, info.exporterMap, "client_id"))
	assert.EqualValues(t, "8s", collectorConfigValue(t, info.exporterMap, "timeout"))
	assert.EqualValues(t, "1m", collectorConfigValue(t, info.exporterMap, "conn_idle_timeout"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "partition_logs_by_resource_attributes"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "partition_metrics_by_resource_attributes"))
	assert.EqualValues(t, "logs", collectorConfigValue(t, info.exporterMap, "logs.topic"))
	assert.EqualValues(t, "raw", collectorConfigValue(t, info.exporterMap, "logs.encoding"))
	assert.EqualValues(t, "metrics", collectorConfigValue(t, info.exporterMap, "metrics.topic"))
	assert.EqualValues(t, "otlp_json", collectorConfigValue(t, info.exporterMap, "metrics.encoding"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "tls.insecure"))
	assert.EqualValues(t, "test-user", collectorConfigValue(t, info.exporterMap, "auth.sasl.username"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "auth.sasl.password"))
	assert.EqualValues(t, "SCRAM-SHA-512", collectorConfigValue(t, info.exporterMap, "auth.sasl.mechanism"))
	assert.EqualValues(t, 100000, collectorConfigValue(t, info.exporterMap, "producer.max_message_bytes"))
	assert.EqualValues(t, -1, collectorConfigValue(t, info.exporterMap, "producer.required_acks"))
	assert.EqualValues(t, "zstd", collectorConfigValue(t, info.exporterMap, "producer.compression"))
	assert.EqualValues(t, 100, collectorConfigValue(t, info.exporterMap, "producer.flush_max_messages"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "producer.allow_auto_topic_creation"))
	assert.EqualValues(t, "10ms", collectorConfigValue(t, info.exporterMap, "producer.linger"))
}

func TestGetExporterDatadogCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"datadog":{"api":{"site":"datadoghq.eu","key":{"fromSecret":"mapping-credential"},"failOnInvalidKey":true},"hostname":"test-host","metrics":{"endpoint":"https://metrics.example","resourceAttributesAsTags":true,"instrumentationScopeMetadataAsTags":true},"logs":{"endpoint":"https://logs.example","useCompression":false,"compressionLevel":3,"batchWait":{"seconds":1}},"hostMetadata":{"enabled":false,"reporterPeriod":{"minutes":5}},"hostnameDetectionTimeout":{"seconds":2}}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "datadoghq.eu", collectorConfigValue(t, info.exporterMap, "api.site"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "api.key"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "api.fail_on_invalid_key"))
	assert.EqualValues(t, "test-host", collectorConfigValue(t, info.exporterMap, "hostname"))
	assert.EqualValues(t, "https://metrics.example", collectorConfigValue(t, info.exporterMap, "metrics.endpoint"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "metrics.resource_attributes_as_tags"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "metrics.instrumentation_scope_metadata_as_tags"))
	assert.EqualValues(t, "https://logs.example", collectorConfigValue(t, info.exporterMap, "logs.endpoint"))
	assert.EqualValues(t, false, collectorConfigValue(t, info.exporterMap, "logs.use_compression"))
	assert.EqualValues(t, 3, collectorConfigValue(t, info.exporterMap, "logs.compression_level"))
	assert.EqualValues(t, false, collectorConfigValue(t, info.exporterMap, "host_metadata.enabled"))
	assert.EqualValues(t, "5m", collectorConfigValue(t, info.exporterMap, "host_metadata.reporter_period"))
	assert.EqualValues(t, "2s", collectorConfigValue(t, info.exporterMap, "hostname_detection_timeout"))
}

func TestGetExporterLogzioCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"logzio":{"endpoint":"https://logzio.example","region":"us","token":{"fromSecret":"mapping-credential"},"timeout":{"seconds":9}}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, false, info.HasMetrics)
	assert.EqualValues(t, "https://logzio.example", collectorConfigValue(t, info.exporterMap, "endpoint"))
	assert.EqualValues(t, "us", collectorConfigValue(t, info.exporterMap, "region"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "account_token"))
	assert.EqualValues(t, "9s", collectorConfigValue(t, info.exporterMap, "timeout"))
}

func TestGetExporterSplunkCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"splunk":{"endpoint":"https://splunk.example/services/collector","token":{"fromSecret":"mapping-credential"},"source":"test-source","sourceType":"test-type","index":"test-index","useMultiMetricFormat":true,"appName":"test-app","appVersion":"1.2.3","maxContentLengthLogs":10000,"maxContentLengthMetrics":20000,"disableCompression":true,"timeout":{"seconds":10},"maxIdleConns":3,"tls":{"insecureSkipVerify":true}}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "https://splunk.example/services/collector", collectorConfigValue(t, info.exporterMap, "endpoint"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "token"))
	assert.EqualValues(t, "test-source", collectorConfigValue(t, info.exporterMap, "source"))
	assert.EqualValues(t, "test-type", collectorConfigValue(t, info.exporterMap, "sourcetype"))
	assert.EqualValues(t, "test-index", collectorConfigValue(t, info.exporterMap, "index"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "use_multi_metric_format"))
	assert.EqualValues(t, "test-app", collectorConfigValue(t, info.exporterMap, "splunk_app_name"))
	assert.EqualValues(t, "1.2.3", collectorConfigValue(t, info.exporterMap, "splunk_app_version"))
	assert.EqualValues(t, 10000, collectorConfigValue(t, info.exporterMap, "max_content_length_logs"))
	assert.EqualValues(t, 20000, collectorConfigValue(t, info.exporterMap, "max_content_length_metrics"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "disable_compression"))
	assert.EqualValues(t, "10s", collectorConfigValue(t, info.exporterMap, "timeout"))
	assert.EqualValues(t, 3, collectorConfigValue(t, info.exporterMap, "max_idle_conns"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "tls.insecure_skip_verify"))
}

func TestGetExporterAzureMonitorCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"azureMonitor":{"connectionString":{"fromSecret":"mapping-credential"},"endpoint":"https://monitor.example","maxBatchSize":50,"maxBatchInterval":{"seconds":2},"shutdownTimeout":{"seconds":3},"customEventsEnabled":true,"exceptionEventsEnabled":true}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "connection_string"))
	assert.EqualValues(t, "https://monitor.example", collectorConfigValue(t, info.exporterMap, "endpoint"))
	assert.EqualValues(t, 50, collectorConfigValue(t, info.exporterMap, "maxbatchsize"))
	assert.EqualValues(t, "2s", collectorConfigValue(t, info.exporterMap, "maxbatchinterval"))
	assert.EqualValues(t, "3s", collectorConfigValue(t, info.exporterMap, "shutdown_timeout"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "custom_events_enabled"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "exception_events_enabled"))
}

func TestGetExporterInfluxDBCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"influxDB":{"endpoint":"https://influx.example","org":"test-org","bucket":"test-bucket","token":{"fromSecret":"mapping-credential"},"headers":[{"key":"X-Tenant","value":"tenant"}],"metricsSchema":"TELEGRAF_PROMETHEUS_V2","precision":"MS","payloadMaxLines":100,"payloadMaxBytes":20000,"timeout":{"seconds":11},"logRecordDimensions":["level","service"],"v1Compatibility":{"enabled":true,"db":"test-db","username":"test-user","password":{"fromSecret":"mapping-credential"}}}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "https://influx.example", collectorConfigValue(t, info.exporterMap, "endpoint"))
	assert.EqualValues(t, "test-org", collectorConfigValue(t, info.exporterMap, "org"))
	assert.EqualValues(t, "test-bucket", collectorConfigValue(t, info.exporterMap, "bucket"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "token"))
	assert.EqualValues(t, "tenant", collectorConfigValue(t, info.exporterMap, "headers.X-Tenant"))
	assert.EqualValues(t, "telegraf-prometheus-v2", collectorConfigValue(t, info.exporterMap, "metrics_schema"))
	assert.EqualValues(t, "ms", collectorConfigValue(t, info.exporterMap, "precision"))
	assert.EqualValues(t, 100, collectorConfigValue(t, info.exporterMap, "payload_max_lines"))
	assert.EqualValues(t, 20000, collectorConfigValue(t, info.exporterMap, "payload_max_bytes"))
	assert.EqualValues(t, "11s", collectorConfigValue(t, info.exporterMap, "timeout"))
	assert.EqualValues(t, true, collectorConfigValue(t, info.exporterMap, "v1_compatibility.enabled"))
	assert.EqualValues(t, "test-db", collectorConfigValue(t, info.exporterMap, "v1_compatibility.db"))
	assert.EqualValues(t, "test-user", collectorConfigValue(t, info.exporterMap, "v1_compatibility.username"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "v1_compatibility.password"))
}

func TestGetExporterAzureDataExplorerCompleteMapping(t *testing.T) {
	p := newCollectorProviderTest(t)
	exp := collectorExporterFromJSON(t, `{"azureDataExplorer":{"clusterURI":"https://test.kusto.windows.net","database":"test-db","metricsTable":"metrics","logsTable":"logs","metricsTableMapping":"metric-map","logsTableMapping":"log-map","ingestionType":"MANAGED","timeout":{"seconds":12},"auth":{"servicePrincipal":{"applicationID":"test-app","tenantID":"test-tenant","applicationKey":{"fromSecret":"mapping-credential"}}}}}`)
	info, err := p.getExporter(t.Context(), exp)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Same(t, exp, info.Exporter)
	assert.Equal(t, true, info.HasLogs)
	assert.Equal(t, true, info.HasMetrics)
	assert.EqualValues(t, "https://test.kusto.windows.net", collectorConfigValue(t, info.exporterMap, "cluster_uri"))
	assert.EqualValues(t, "test-db", collectorConfigValue(t, info.exporterMap, "db_name"))
	assert.EqualValues(t, "metrics", collectorConfigValue(t, info.exporterMap, "metrics_table_name"))
	assert.EqualValues(t, "logs", collectorConfigValue(t, info.exporterMap, "logs_table_name"))
	assert.EqualValues(t, "metric-map", collectorConfigValue(t, info.exporterMap, "metrics_table_json_mapping"))
	assert.EqualValues(t, "log-map", collectorConfigValue(t, info.exporterMap, "logs_table_json_mapping"))
	assert.EqualValues(t, "managed", collectorConfigValue(t, info.exporterMap, "ingestion_type"))
	assert.EqualValues(t, "12s", collectorConfigValue(t, info.exporterMap, "timeout"))
	assert.EqualValues(t, "test-app", collectorConfigValue(t, info.exporterMap, "application_id"))
	assert.EqualValues(t, "test-tenant", collectorConfigValue(t, info.exporterMap, "tenant_id"))
	assert.EqualValues(t, "credential-value", collectorConfigValue(t, info.exporterMap, "application_key"))
}

func TestCollectorDurationConversionsAndValidation(t *testing.T) {
	for _, tc := range []struct {
		name, raw, formatted string
		want                 time.Duration
	}{
		{"Milliseconds", `{"milliseconds":125}`, "125ms", 125 * time.Millisecond},
		{"Seconds", `{"seconds":3}`, "3s", 3 * time.Second},
		{"Minutes", `{"minutes":2}`, "2m", 2 * time.Minute},
		{"Hours", `{"hours":2}`, "2h", 2 * time.Hour},
		{"Days", `{"days":2}`, "48h", 48 * time.Hour},
		{"Weeks", `{"weeks":2}`, "336h", 14 * 24 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &metav1.Duration{}
			require.NoError(t, protojson.Unmarshal([]byte(tc.raw), d))
			formatted, err := durationToCollectorString(d)
			require.NoError(t, err)
			assert.Equal(t, tc.formatted, formatted)
			converted, err := metav1DurationToTimeDuration(d)
			require.NoError(t, err)
			assert.Equal(t, tc.want, converted)
		})
	}
	for _, raw := range []string{`{}`, `{"milliseconds":0}`, `{"seconds":0}`, `{"minutes":0}`, `{"hours":0}`, `{"days":0}`, `{"weeks":0}`, `{"months":1}`} {
		d := &metav1.Duration{}
		require.NoError(t, protojson.Unmarshal([]byte(raw), d))
		_, err := durationToCollectorString(d)
		require.Error(t, err)
		_, err = metav1DurationToTimeDuration(d)
		require.Error(t, err)
	}
	formatted, err := durationToCollectorString(nil)
	require.NoError(t, err)
	assert.Empty(t, formatted)
	_, err = metav1DurationToTimeDuration(nil)
	require.Error(t, err)
	require.NoError(t, validateCollectorDuration("timeout", nil, time.Second, time.Minute, true))
	require.Error(t, validateCollectorDuration("timeout", nil, time.Second, time.Minute, false))
	for _, tc := range []struct {
		seconds uint32
		valid   bool
	}{{1, true}, {60, true}, {61, false}, {0, false}} {
		err := validateCollectorDuration("timeout", &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: tc.seconds}}, time.Second, time.Minute, false)
		assert.Equal(t, tc.valid, err == nil)
	}
}

func TestCollectorKeepaliveAndClickhouseEmptyOptions(t *testing.T) {
	for _, ka := range []*enterprisev1.CollectorExporter_Spec_OTLP_Keepalive{nil, {}} {
		value, err := otlpKeepaliveToCollector(ka)
		require.NoError(t, err)
		assert.Nil(t, value)
	}
	value, err := otlpKeepaliveToCollector(&enterprisev1.CollectorExporter_Spec_OTLP_Keepalive{PermitWithoutStream: true})
	require.NoError(t, err)
	assert.True(t, value.PermitWithoutStream)
	for _, ka := range []*enterprisev1.CollectorExporter_Spec_OTLP_Keepalive{
		{Time: &metav1.Duration{}}, {Timeout: &metav1.Duration{}},
	} {
		_, err := otlpKeepaliveToCollector(ka)
		require.Error(t, err)
	}
	assert.Nil(t, clickhouseTableEngineToCollector(nil))
	assert.Nil(t, clickhouseTableEngineToCollector(&enterprisev1.CollectorExporter_Spec_Clickhouse_TableEngine{}))
	assert.Nil(t, clickhouseMetricsTablesToCollector(nil))
	assert.Nil(t, clickhouseMetricsTablesToCollector(&enterprisev1.CollectorExporter_Spec_Clickhouse_MetricsTables{}))
}

func TestCollectorInitialConfigIsIndependentAndKeepsInternalRoutes(t *testing.T) {
	p := &provider{port: 12345, internalLogstoreEndpoint: "logs.internal:8080", internalMetricstoreEndpoint: "metrics.internal:8080"}
	first, err := p.getInitConfig(t.Context())
	require.NoError(t, err)
	assert.Equal(t, ":12345", collectorConfigValue(t, first, "receivers.octelium_otlp.endpoint"))
	assert.Equal(t, "logs.internal:8080", collectorConfigValue(t, first, "exporters.octelium_otlp/logstore.endpoint"))
	assert.Equal(t, "metrics.internal:8080", collectorConfigValue(t, first, "exporters.octelium_otlp/metricstore.endpoint"))
	first["exporters"].(map[string]any)["injected"] = true
	first["service"].(map[string]any)["pipelines"].(map[string]any)["injected"] = true
	second, err := p.getInitConfig(t.Context())
	require.NoError(t, err)
	assert.NotContains(t, second["exporters"], "injected")
	assert.NotContains(t, collectorConfigValue(t, second, "service.pipelines"), "injected")
	assert.Equal(t, []string{"octelium_otlp/logstore"}, collectorConfigValue(t, second, "service.pipelines.logs.exporters"))
	assert.Equal(t, []string{"octelium_otlp/metricstore"}, collectorConfigValue(t, second, "service.pipelines.metrics.exporters"))
	assert.Equal(t, defaultInternalLogstoreEndpoint, (&provider{}).getInternalLogstoreEndpoint())
	assert.Equal(t, defaultInternalMetricstoreEndpoint, (&provider{}).getInternalMetricstoreEndpoint())
	s, err := NewServer(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, 8080, s.receiverPort)
	assert.Equal(t, defaultInternalLogstoreEndpoint, s.getInternalLogstoreEndpoint())
	assert.Equal(t, defaultInternalMetricstoreEndpoint, s.getInternalMetricstoreEndpoint())
	s.internalLogstoreEndpoint, s.internalMetricstoreEndpoint = "custom-logs:1", "custom-metrics:2"
	assert.Equal(t, "custom-logs:1", s.getInternalLogstoreEndpoint())
	assert.Equal(t, "custom-metrics:2", s.getInternalMetricstoreEndpoint())
}

func TestExporterMapSerializationErrors(t *testing.T) {
	for _, exp := range []any{make(chan int), []string{"invalid-object"}} {
		assert.Error(t, (&exporterInfo{exp: exp}).toExporterMapAny())
	}
}

func TestGetExporterRejectsUnsupportedAndNilPayloads(t *testing.T) {
	for _, spec := range []*enterprisev1.CollectorExporter_Spec{
		{},
		{Type: &enterprisev1.CollectorExporter_Spec_Otlp{}},
		{Type: &enterprisev1.CollectorExporter_Spec_OtlpHTTP{}},
		{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_{}},
		{Type: &enterprisev1.CollectorExporter_Spec_Clickhouse_{}},
		{Type: &enterprisev1.CollectorExporter_Spec_Elasticsearch_{}},
		{Type: &enterprisev1.CollectorExporter_Spec_Kafka_{}},
		{Type: &enterprisev1.CollectorExporter_Spec_Datadog_{}},
		{Type: &enterprisev1.CollectorExporter_Spec_Logzio_{}},
		{Type: &enterprisev1.CollectorExporter_Spec_Splunk_{}},
		{Type: &enterprisev1.CollectorExporter_Spec_AzureMonitor_{}},
		{Type: &enterprisev1.CollectorExporter_Spec_InfluxDB_{}},
		{Type: &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_{}},
	} {
		info, err := (&provider{}).getExporter(t.Context(), &enterprisev1.CollectorExporter{Spec: spec})
		require.Error(t, err)
		assert.Nil(t, info)
	}
	assert.Empty(t, (&provider{}).getType(&enterprisev1.CollectorExporter{Spec: &enterprisev1.CollectorExporter_Spec{}}))
}

func TestCollectorPipelineFilteringByStateAndSignal(t *testing.T) {
	p := newCollectorProviderTest(t)
	for _, tc := range []struct{ name, spec string }{
		{"metrics-only", `{"prometheusRemoteWrite":{"endpoint":"https://metrics.example/api/v1/write"}}`},
		{"logs-only", `{"logzio":{"token":{"fromSecret":"mapping-credential"},"region":"us"}}`},
	} {
		exp := collectorExporterFromJSON(t, tc.spec)
		exp.Metadata = &metav1.Metadata{Name: tc.name}
		_, err := p.octeliumC.EnterpriseC().CreateCollectorExporter(t.Context(), exp)
		require.NoError(t, err)
	}
	cc, err := p.octeliumC.EnterpriseV1Utils().GetClusterConfig(t.Context())
	require.NoError(t, err)
	cc.Spec.Collector = &enterprisev1.ClusterConfig_Spec_Collector{Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		{Name: "disabled", Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, IsDisabled: true, Exporters: []string{"logs-only"}},
		{Name: "empty", Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS},
		{Name: "unknown", Type: 99, Exporters: []string{"logs-only"}},
		{Name: "missing", Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, Exporters: []string{"missing"}},
		{Name: "wrong-logs", Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, Exporters: []string{"metrics-only"}},
		{Name: "wrong-metrics", Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS, Exporters: []string{"logs-only"}},
		{Name: "valid-logs", Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, Exporters: []string{"metrics-only", "logs-only"}},
		{Name: "valid-metrics", Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS, Exporters: []string{"metrics-only", "logs-only"}},
	}}
	_, err = p.octeliumC.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
	require.NoError(t, err)
	configuration, err := p.getConfig(t.Context())
	require.NoError(t, err)
	pipelines := collectorConfigValue(t, configuration, "service.pipelines").(map[string]any)
	require.Len(t, pipelines, 4)
	for _, name := range []string{"logs", "metrics", "logs/valid-logs", "metrics/valid-metrics"} {
		assert.Contains(t, pipelines, name)
	}
	assert.Len(t, pipelines["logs/valid-logs"].(map[string]any)["exporters"], 1)
	assert.Len(t, pipelines["metrics/valid-metrics"].(map[string]any)["exporters"], 1)
}

type collectorErrorResources struct {
	renterprisev1.ResourceServiceClient
	err error
}

func (c *collectorErrorResources) ListCollectorExporter(context.Context, *rmetav1.ListOptions, ...grpc.CallOption) (*enterprisev1.CollectorExporterList, error) {
	return nil, c.err
}

type collectorErrorClient struct {
	octeliumc.ClientInterface
	resources renterprisev1.ResourceServiceClient
}

func (c *collectorErrorClient) EnterpriseC() renterprisev1.ResourceServiceClient { return c.resources }

func TestCollectorConfigReturnsResourceFailures(t *testing.T) {
	p := newCollectorProviderTest(t)
	failure := errors.New("exporter listing unavailable")
	p.octeliumC = &collectorErrorClient{ClientInterface: p.octeliumC, resources: &collectorErrorResources{err: failure}}
	_, err := p.getConfig(t.Context())
	assert.ErrorIs(t, err, failure)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = p.getConfig(ctx)
	require.Error(t, err)
}

func TestCollectorExporterEnumConversions(t *testing.T) {
	t.Run("OTLPCompression", func(t *testing.T) {
		for value, want := range []string{"", "", "none", "snappy", "zstd"} {
			got, err := otlpCompressionToCollectorString(enterprisev1.CollectorExporter_Spec_OTLP_Compression(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := otlpCompressionToCollectorString(enterprisev1.CollectorExporter_Spec_OTLP_Compression(99))
		require.Error(t, err)
	})
	t.Run("OTLPHTTPCompression", func(t *testing.T) {
		for value, want := range []string{"", "", "none"} {
			got, err := otlpHTTPCompressionToCollectorString(enterprisev1.CollectorExporter_Spec_OTLPHTTP_Compression(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := otlpHTTPCompressionToCollectorString(enterprisev1.CollectorExporter_Spec_OTLPHTTP_Compression(99))
		require.Error(t, err)
	})
	t.Run("OTLPHTTPEncoding", func(t *testing.T) {
		for value, want := range []string{"", "", "json"} {
			got, err := otlpHTTPEncodingToCollectorString(enterprisev1.CollectorExporter_Spec_OTLPHTTP_Encoding(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := otlpHTTPEncodingToCollectorString(enterprisev1.CollectorExporter_Spec_OTLPHTTP_Encoding(99))
		require.Error(t, err)
	})
	t.Run("KafkaEncoding", func(t *testing.T) {
		for value, want := range []string{"", "otlp_proto", "otlp_json", "raw"} {
			got, err := kafkaEncodingToCollectorString(enterprisev1.CollectorExporter_Spec_Kafka_Encoding(value), true)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := kafkaEncodingToCollectorString(enterprisev1.CollectorExporter_Spec_Kafka_Encoding(99), true)
		require.Error(t, err)
	})
	t.Run("KafkaSASL", func(t *testing.T) {
		for value, want := range []string{"PLAIN", "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512"} {
			got, err := kafkaSASLMechanismToCollectorString(enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := kafkaSASLMechanismToCollectorString(enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism(99))
		require.Error(t, err)
	})
	t.Run("KafkaCompression", func(t *testing.T) {
		for value, want := range []string{"", "none", "gzip", "snappy", "lz4", "zstd"} {
			got, err := kafkaProducerCompressionToCollectorString(enterprisev1.CollectorExporter_Spec_Kafka_ProducerCompression(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := kafkaProducerCompressionToCollectorString(enterprisev1.CollectorExporter_Spec_Kafka_ProducerCompression(99))
		require.Error(t, err)
	})
	t.Run("InfluxSchema", func(t *testing.T) {
		for value, want := range []string{"", "telegraf-prometheus-v1", "telegraf-prometheus-v2"} {
			got, err := influxMetricsSchemaToCollectorString(enterprisev1.CollectorExporter_Spec_InfluxDB_MetricsSchema(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := influxMetricsSchemaToCollectorString(enterprisev1.CollectorExporter_Spec_InfluxDB_MetricsSchema(99))
		require.Error(t, err)
	})
	t.Run("InfluxPrecision", func(t *testing.T) {
		for value, want := range []string{"", "ns", "us", "ms", "s"} {
			got, err := influxPrecisionToCollectorString(enterprisev1.CollectorExporter_Spec_InfluxDB_Precision(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := influxPrecisionToCollectorString(enterprisev1.CollectorExporter_Spec_InfluxDB_Precision(99))
		require.Error(t, err)
	})
	t.Run("ClickhouseCompression", func(t *testing.T) {
		for value, want := range []string{"", "lz4", "none", "zstd", "gzip", "deflate", "br"} {
			got, err := clickhouseCompressionToCollectorString(enterprisev1.CollectorExporter_Spec_Clickhouse_Compression(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := clickhouseCompressionToCollectorString(enterprisev1.CollectorExporter_Spec_Clickhouse_Compression(99))
		require.Error(t, err)
	})
	t.Run("PrometheusTranslation", func(t *testing.T) {
		for value, want := range []string{"", "UnderscoreEscapingWithSuffixes", "UnderscoreEscapingWithoutSuffixes", "NoUTF8EscapingWithSuffixes", "NoTranslation"} {
			got, err := prometheusTranslationStrategyToCollectorString(enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_TranslationStrategy(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := prometheusTranslationStrategyToCollectorString(enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_TranslationStrategy(99))
		require.Error(t, err)
	})
	t.Run("AzureIngestion", func(t *testing.T) {
		for value, want := range []string{"", "", "managed"} {
			got, err := azureDataExplorerIngestionTypeToCollectorString(enterprisev1.CollectorExporter_Spec_AzureDataExplorer_IngestionType(value))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		}
		_, err := azureDataExplorerIngestionTypeToCollectorString(enterprisev1.CollectorExporter_Spec_AzureDataExplorer_IngestionType(99))
		require.Error(t, err)
	})
	_, err := kafkaEncodingToCollectorString(enterprisev1.CollectorExporter_Spec_Kafka_RAW, false)
	require.Error(t, err)
}

func TestCollectorHeaderAndLabelConversions(t *testing.T) {
	t.Run("elasticHeadersToMap", func(t *testing.T) {
		assert.Nil(t, elasticHeadersToMap(nil))
		assert.Nil(t, elasticHeadersToMap([]*enterprisev1.CollectorExporter_Spec_Elasticsearch_Header{nil, {}}))
		input := []*enterprisev1.CollectorExporter_Spec_Elasticsearch_Header{nil, {}, {Key: "key", Value: "first"}, {Key: "key", Value: "last"}}
		got := elasticHeadersToMap(input)
		assert.Equal(t, map[string]string{"key": "last"}, got)
		got["key"] = "changed"
		assert.Equal(t, "last", input[3].Value)
	})
	t.Run("influxHeadersToMap", func(t *testing.T) {
		assert.Nil(t, influxHeadersToMap(nil))
		assert.Nil(t, influxHeadersToMap([]*enterprisev1.CollectorExporter_Spec_InfluxDB_Header{nil, {}}))
		input := []*enterprisev1.CollectorExporter_Spec_InfluxDB_Header{nil, {}, {Key: "key", Value: "first"}, {Key: "key", Value: "last"}}
		got := influxHeadersToMap(input)
		assert.Equal(t, map[string]string{"key": "last"}, got)
		got["key"] = "changed"
		assert.Equal(t, "last", input[3].Value)
	})
	t.Run("otlpHeadersToMap", func(t *testing.T) {
		assert.Nil(t, otlpHeadersToMap(nil))
		assert.Nil(t, otlpHeadersToMap([]*enterprisev1.CollectorExporter_Spec_OTLP_Header{nil, {}}))
		input := []*enterprisev1.CollectorExporter_Spec_OTLP_Header{nil, {}, {Key: "key", Value: "first"}, {Key: "key", Value: "last"}}
		got := otlpHeadersToMap(input)
		assert.Equal(t, map[string]string{"key": "last"}, got)
		got["key"] = "changed"
		assert.Equal(t, "last", input[3].Value)
	})
	t.Run("otlpHTTPHeadersToMap", func(t *testing.T) {
		assert.Nil(t, otlpHTTPHeadersToMap(nil))
		assert.Nil(t, otlpHTTPHeadersToMap([]*enterprisev1.CollectorExporter_Spec_OTLPHTTP_Header{nil, {}}))
		input := []*enterprisev1.CollectorExporter_Spec_OTLPHTTP_Header{nil, {}, {Key: "key", Value: "first"}, {Key: "key", Value: "last"}}
		got := otlpHTTPHeadersToMap(input)
		assert.Equal(t, map[string]string{"key": "last"}, got)
		got["key"] = "changed"
		assert.Equal(t, "last", input[3].Value)
	})
	t.Run("prometheusHeadersToMap", func(t *testing.T) {
		assert.Nil(t, prometheusHeadersToMap(nil))
		assert.Nil(t, prometheusHeadersToMap([]*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Header{nil, {}}))
		input := []*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Header{nil, {}, {Key: "key", Value: "first"}, {Key: "key", Value: "last"}}
		got := prometheusHeadersToMap(input)
		assert.Equal(t, map[string]string{"key": "last"}, got)
		got["key"] = "changed"
		assert.Equal(t, "last", input[3].Value)
	})
	t.Run("prometheusExternalLabelsToMap", func(t *testing.T) {
		assert.Nil(t, prometheusExternalLabelsToMap(nil))
		assert.Nil(t, prometheusExternalLabelsToMap([]*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel{nil, {}}))
		input := []*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel{nil, {}, {Key: "key", Value: "first"}, {Key: "key", Value: "last"}}
		got := prometheusExternalLabelsToMap(input)
		assert.Equal(t, map[string]string{"key": "last"}, got)
		got["key"] = "changed"
		assert.Equal(t, "last", input[3].Value)
	})
	t.Run("clickhouseConnectionParamsToMap", func(t *testing.T) {
		assert.Nil(t, clickhouseConnectionParamsToMap(nil))
		assert.Nil(t, clickhouseConnectionParamsToMap([]*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam{nil, {}}))
		input := []*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam{nil, {}, {Key: "key", Value: "first"}, {Key: "key", Value: "last"}}
		got := clickhouseConnectionParamsToMap(input)
		assert.Equal(t, map[string]string{"key": "last"}, got)
		got["key"] = "changed"
		assert.Equal(t, "last", input[3].Value)
	})
	assert.Nil(t, kafkaHeadersToCollector(nil))
	assert.Nil(t, kafkaHeadersToCollector([]*enterprisev1.CollectorExporter_Spec_Kafka_Header{nil, {}}))
	assert.Equal(t, []exporterKafkaHeader{{Key: "key", Value: "value"}}, kafkaHeadersToCollector([]*enterprisev1.CollectorExporter_Spec_Kafka_Header{nil, {}, {Key: "key", Value: "value"}}))
}

func TestGetExporterHeaderAuthenticationModes(t *testing.T) {
	p := newCollectorProviderTest(t)
	for _, kind := range []string{"otlp", "otlpHTTP", "prometheusRemoteWrite"} {
		for _, tc := range []struct{ name, auth, header, want string }{
			{"Bearer", `{"bearer":{"fromSecret":"mapping-credential"}}`, "Authorization", "Bearer credential-value"},
			{"Basic", `{"basic":{"username":"user","password":{"fromSecret":"mapping-credential"}}}`, "Authorization", basicAuthHeader("user", "credential-value")},
			{"Custom", `{"custom":{"header":"X-Auth","value":{"fromSecret":"mapping-credential"}}}`, "X-Auth", "credential-value"},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				endpoint := "https://export.example"
				if kind == "otlp" {
					endpoint = "export.example:4317"
				}
				document := map[string]any{kind: map[string]any{"endpoint": endpoint, "auth": json.RawMessage(tc.auth), "headers": []map[string]string{{"key": "X-Keep", "value": "keep"}, {"key": tc.header, "value": "old"}}}}
				encoded, err := json.Marshal(document)
				require.NoError(t, err)
				exp := collectorExporterFromJSON(t, string(encoded))
				original, err := protojson.Marshal(exp)
				require.NoError(t, err)
				info, err := p.getExporter(t.Context(), exp)
				require.NoError(t, err)
				assert.Equal(t, tc.want, collectorConfigValue(t, info.exporterMap, "headers."+tc.header))
				assert.Equal(t, "keep", collectorConfigValue(t, info.exporterMap, "headers.X-Keep"))
				after, err := protojson.Marshal(exp)
				require.NoError(t, err)
				assert.JSONEq(t, string(original), string(after))
			})
		}
	}
}

func TestGetExporterIncompleteHeaderAuthDoesNotReadSecrets(t *testing.T) {
	for _, kind := range []string{"otlp", "otlpHTTP", "prometheusRemoteWrite"} {
		for _, auth := range []string{`{}`, `{"bearer":{}}`, `{"basic":{}}`, `{"basic":{"password":{}}}`, `{"custom":{}}`, `{"custom":{"header":"X-Auth"}}`, `{"custom":{"header":"X-Auth","value":{}}}`} {
			document := fmt.Sprintf(`{%q:{"endpoint":"export.example:4317","auth":%s}}`, kind, auth)
			_, err := (&provider{}).getExporter(t.Context(), collectorExporterFromJSON(t, document))
			require.NoError(t, err)
		}
	}
}

func TestGetExporterAlternativeAzureAndInfluxCredentials(t *testing.T) {
	p := newCollectorProviderTest(t)
	for _, tc := range []struct {
		spec, key string
		want      any
	}{
		{`{"azureMonitor":{"instrumentationKey":{"fromSecret":"mapping-credential"}}}`, "instrumentation_key", "credential-value"},
		{`{"azureDataExplorer":{"auth":{"managedIdentity":{"id":"managed-id"}}}}`, "managed_identity_id", "managed-id"},
		{`{"azureDataExplorer":{"auth":{"azureDefault":{}}}}`, "use_azure_auth", true},
		{`{"influxDB":{"v1Compatibility":{"enabled":true,"password":{"fromSecret":"mapping-credential"}}}}`, "v1_compatibility.password", "credential-value"},
	} {
		info, err := p.getExporter(t.Context(), collectorExporterFromJSON(t, tc.spec))
		require.NoError(t, err)
		assert.Equal(t, tc.want, collectorConfigValue(t, info.exporterMap, tc.key))
	}
	for _, spec := range []string{
		`{"azureMonitor":{"instrumentationKey":{"fromSecret":"missing"}}}`,
		`{"influxDB":{"v1Compatibility":{"password":{"fromSecret":"missing"}}}}`,
	} {
		_, err := p.getExporter(t.Context(), collectorExporterFromJSON(t, spec))
		require.Error(t, err)
	}
}

func TestGetExporterRejectsInvalidNestedOptions(t *testing.T) {
	for _, spec := range []string{
		`{"otlp":{"timeout":{}}}`, `{"otlp":{"compression":99}}`,
		`{"otlp":{"keepalive":{"time":{}}}}`, `{"otlp":{"keepalive":{"timeout":{"months":1}}}}`,
		`{"otlpHTTP":{"timeout":{}}}`, `{"otlpHTTP":{"encoding":99}}`, `{"otlpHTTP":{"compression":99}}`,
		`{"prometheusRemoteWrite":{"timeout":{}}}`, `{"prometheusRemoteWrite":{"translationStrategy":99}}`,
		`{"clickhouse":{"timeout":{}}}`, `{"clickhouse":{"ttl":{}}}`, `{"clickhouse":{"compression":99}}`,
		`{"elasticsearch":{"timeout":{}}}`, `{"elasticsearch":{"compression":99}}`,
		`{"kafka":{"timeout":{}}}`, `{"kafka":{"connIdleTimeout":{}}}`,
		`{"kafka":{"logs":{"encoding":99}}}`, `{"kafka":{"metrics":{"encoding":"RAW"}}}`,
		`{"kafka":{"auth":{"sasl":{"mechanism":99}}}}`,
		`{"kafka":{"producer":{"compression":99}}}`, `{"kafka":{"producer":{"linger":{}}}}`,
		`{"datadog":{"logs":{"batchWait":{}}}}`, `{"datadog":{"hostMetadata":{"reporterPeriod":{}}}}`, `{"datadog":{"hostnameDetectionTimeout":{}}}`,
		`{"logzio":{"timeout":{}}}`, `{"splunk":{"timeout":{}}}`,
		`{"azureMonitor":{"maxBatchInterval":{}}}`, `{"azureMonitor":{"shutdownTimeout":{}}}`,
		`{"azureMonitor":{"maxBatchInterval":{"milliseconds":1}}}`, `{"azureMonitor":{"shutdownTimeout":{"minutes":6}}}`,
		`{"influxDB":{"timeout":{}}}`, `{"influxDB":{"metricsSchema":99}}`, `{"influxDB":{"precision":99}}`,
		`{"azureDataExplorer":{"timeout":{}}}`, `{"azureDataExplorer":{"ingestionType":99}}`,
	} {
		_, err := (&provider{}).getExporter(t.Context(), collectorExporterFromJSON(t, spec))
		require.Error(t, err, spec)
	}
}

func TestGetExporterCredentialSourceErrors(t *testing.T) {
	p := newCollectorProviderTest(t)
	for _, spec := range []string{
		`{"otlp":{"endpoint":"otel.example:4317","headers":[{"key":"X-Tenant","value":"tenant"}],"auth":{"bearer":{"fromSecret":"missing-credential"}},"tls":{"insecure":true,"insecureSkipVerify":true,"serverNameOverride":"otel.example","caPEM":"test-ca"},"compression":"ZSTD","timeout":{"seconds":3},"waitForReady":true,"authority":"authority.example","userAgent":"test-agent","balancerName":"round_robin","readBufferSize":1024,"writeBufferSize":2048,"keepalive":{"time":{"minutes":1},"timeout":{"seconds":2},"permitWithoutStream":true}}}`,
		`{"otlpHTTP":{"endpoint":"https://otel.example","logsEndpoint":"https://logs.example/v1/logs","metricsEndpoint":"https://metrics.example/v1/metrics","headers":[{"key":"X-Tenant","value":"tenant"}],"auth":{"custom":{"header":"X-Auth","value":{"fromSecret":"missing-credential"}}},"tls":{"insecureSkipVerify":true,"serverNameOverride":"otel.example","caPEM":"test-ca"},"encoding":"JSON","compression":"NONE","timeout":{"seconds":4},"readBufferSize":1024,"writeBufferSize":2048}}`,
		`{"prometheusRemoteWrite":{"endpoint":"https://prometheus.example/api/v1/write","namespace":"octelium","headers":[{"key":"X-Tenant","value":"tenant"}],"externalLabels":[{"key":"cluster","value":"test"}],"auth":{"basic":{"username":"test-user","password":{"fromSecret":"missing-credential"}}},"tls":{"insecureSkipVerify":true},"timeout":{"seconds":5},"remoteWriteQueue":{"enabled":false,"queueSize":100,"numConsumers":2},"resourceToTelemetryConversion":{"enabled":true,"excludeServiceAttributes":true},"targetInfo":{"enabled":false},"disableScopeInfo":true,"maxBatchSizeBytes":10000,"maxBatchRequestParallelism":2,"translationStrategy":"NO_TRANSLATION","sendMetadata":true}}`,
		`{"clickhouse":{"endpoint":"tcp://clickhouse.example:9000","username":"test-user","password":{"fromSecret":"missing-credential"},"database":"telemetry","tls":{"insecureSkipVerify":true},"connectionParams":[{"key":"dial_timeout","value":"10s"}],"logsTableName":"logs","metricsTables":{"gauge":"gauges","sum":"sums","summary":"summaries","histogram":"histograms","exponentialHistogram":"exponential"},"ttl":{"days":2},"createSchema":false,"compression":"ZSTD","asyncInsert":false,"json":true,"clusterName":"test-cluster","tableEngine":{"name":"MergeTree","params":"()"},"timeout":{"seconds":6}}}`,
		`{"elasticsearch":{"endpoint":"https://elastic.example","endpoints":["https://elastic.example","https://elastic2.example"],"cloudID":"test-cloud","pipeline":"ingest","logsIndex":"logs-test","metricsIndex":"metrics-test","headers":[{"key":"X-Tenant","value":"tenant"}],"timeout":{"seconds":7},"compression":"NONE","tls":{"insecureSkipVerify":true},"auth":{"apiKey":{"fromSecret":"missing-credential"}}}}`,
		`{"kafka":{"brokers":["kafka.example:9092"],"protocolVersion":"2.8.0","clientID":"test-client","recordHeaders":[{"key":"tenant","value":"test"}],"timeout":{"seconds":8},"connIdleTimeout":{"minutes":1},"partitionLogsByResourceAttributes":true,"partitionMetricsByResourceAttributes":true,"logs":{"topic":"logs","encoding":"RAW"},"metrics":{"topic":"metrics","encoding":"OTLP_JSON"},"tls":{"insecure":true,"insecureSkipVerify":true},"auth":{"sasl":{"username":"test-user","password":{"fromSecret":"missing-credential"},"mechanism":"SCRAM_SHA_512"}},"producer":{"maxMessageBytes":100000,"requiredAcks":-1,"compression":"ZSTD","flushMaxMessages":100,"allowAutoTopicCreation":true,"linger":{"milliseconds":10}}}}`,
		`{"datadog":{"api":{"site":"datadoghq.eu","key":{"fromSecret":"missing-credential"},"failOnInvalidKey":true},"hostname":"test-host","metrics":{"endpoint":"https://metrics.example","resourceAttributesAsTags":true,"instrumentationScopeMetadataAsTags":true},"logs":{"endpoint":"https://logs.example","useCompression":false,"compressionLevel":3,"batchWait":{"seconds":1}},"hostMetadata":{"enabled":false,"reporterPeriod":{"minutes":5}},"hostnameDetectionTimeout":{"seconds":2}}}`,
		`{"logzio":{"endpoint":"https://logzio.example","region":"us","token":{"fromSecret":"missing-credential"},"timeout":{"seconds":9}}}`,
		`{"splunk":{"endpoint":"https://splunk.example/services/collector","token":{"fromSecret":"missing-credential"},"source":"test-source","sourceType":"test-type","index":"test-index","useMultiMetricFormat":true,"appName":"test-app","appVersion":"1.2.3","maxContentLengthLogs":10000,"maxContentLengthMetrics":20000,"disableCompression":true,"timeout":{"seconds":10},"maxIdleConns":3,"tls":{"insecureSkipVerify":true}}}`,
		`{"azureMonitor":{"connectionString":{"fromSecret":"missing-credential"},"endpoint":"https://monitor.example","maxBatchSize":50,"maxBatchInterval":{"seconds":2},"shutdownTimeout":{"seconds":3},"customEventsEnabled":true,"exceptionEventsEnabled":true}}`,
		`{"influxDB":{"endpoint":"https://influx.example","org":"test-org","bucket":"test-bucket","token":{"fromSecret":"missing-credential"},"headers":[{"key":"X-Tenant","value":"tenant"}],"metricsSchema":"TELEGRAF_PROMETHEUS_V2","precision":"MS","payloadMaxLines":100,"payloadMaxBytes":20000,"timeout":{"seconds":11},"logRecordDimensions":["level","service"],"v1Compatibility":{"enabled":true,"db":"test-db","username":"test-user","password":{"fromSecret":"missing-credential"}}}}`,
		`{"azureDataExplorer":{"clusterURI":"https://test.kusto.windows.net","database":"test-db","metricsTable":"metrics","logsTable":"logs","metricsTableMapping":"metric-map","logsTableMapping":"log-map","ingestionType":"MANAGED","timeout":{"seconds":12},"auth":{"servicePrincipal":{"applicationID":"test-app","tenantID":"test-tenant","applicationKey":{"fromSecret":"missing-credential"}}}}}`,
	} {
		_, err := p.getExporter(t.Context(), collectorExporterFromJSON(t, spec))
		require.Error(t, err)
	}
}

func TestServerFanOutAcrossProtocolsAndCredentialReload(t *testing.T) {
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	logs, metrics, external := newOTLPGRPCSink(t), newOTLPGRPCSink(t), newOTLPGRPCSink(t)
	for _, sink := range []*otlpGRPCSink{logs, metrics, external} {
		t.Cleanup(sink.close)
	}
	httpSink := newOTLPSink(t)
	t.Cleanup(httpSink.close)
	srv := &Server{octeliumC: tst.C.OcteliumC, receiverPort: tests.GetPort(), internalLogstoreEndpoint: logs.endpoint(), internalMetricstoreEndpoint: metrics.endpoint()}
	secret := createSecret(ctx, t, tst.C.OcteliumC, "old-password")
	grpcExporter := collectorExporterFromJSON(t, fmt.Sprintf(`{"otlp":{"endpoint":%q,"tls":{"insecure":true}}}`, external.endpoint()))
	grpcExporter.Metadata = &metav1.Metadata{Name: "grpc-fanout"}
	grpcExporter, err = tst.C.OcteliumC.EnterpriseC().CreateCollectorExporter(ctx, grpcExporter)
	require.NoError(t, err)
	httpExporter := createOTLPHTTPExporter(ctx, t, tst.C.OcteliumC, httpSink.endpoint(), otlpHTTPBasicAuth("user", secret.Metadata.Name), false)
	setCollectorPipelines(ctx, t, tst.C.OcteliumC, []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		{Name: "fanout-logs", Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS, Exporters: []string{grpcExporter.Metadata.Name, httpExporter.Metadata.Name}},
		{Name: "fanout-metrics", Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS, Exporters: []string{grpcExporter.Metadata.Name, httpExporter.Metadata.Name}},
	})
	require.NoError(t, srv.Run(ctx))
	t.Cleanup(func() { cancel(); shutdownCollector(t, srv) })
	requireEventuallyExportLogToSink(ctx, t, srv.receiverPort, httpSink)
	requireEventuallyExportMetricToSink(ctx, t, srv.receiverPort, httpSink)
	require.Eventually(t, func() bool {
		return external.logRecordCount() > 0 && external.metricPointCount() > 0 && logs.logRecordCount() > 0 && metrics.metricPointCount() > 0
	}, 10*time.Second, 20*time.Millisecond)
	assert.Equal(t, basicAuthHeader("user", "old-password"), httpSink.logAuthHeader())
	secret.Data = &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: "new-password"}}
	_, err = tst.C.OcteliumC.EnterpriseC().UpdateSecret(ctx, secret)
	require.NoError(t, err)
	srv.p.sendUpdate()
	require.Eventually(t, func() bool {
		if exportTestLog(ctx, srv.receiverPort) != nil || exportTestMetric(ctx, srv.receiverPort) != nil {
			return false
		}
		return httpSink.logAuthHeader() == basicAuthHeader("user", "new-password") && httpSink.metricAuthHeader() == basicAuthHeader("user", "new-password")
	}, 15*time.Second, 100*time.Millisecond)
}

func TestCollectorVendorSchemaCompatibility(t *testing.T) {
	p := newCollectorProviderTest(t)
	for _, tc := range []struct {
		name, spec string
		factory    exporter.Factory
	}{
		{"OTLP", `{"otlp":{"endpoint":"otel.example:4317","headers":[{"key":"X-Tenant","value":"tenant"}],"auth":{"bearer":{"fromSecret":"mapping-credential"}},"tls":{"insecure":true,"insecureSkipVerify":true,"serverNameOverride":"otel.example","caPEM":"test-ca"},"compression":"ZSTD","timeout":{"seconds":3},"waitForReady":true,"authority":"authority.example","userAgent":"test-agent","balancerName":"round_robin","readBufferSize":1024,"writeBufferSize":2048,"keepalive":{"time":{"minutes":1},"timeout":{"seconds":2},"permitWithoutStream":true}}}`, otlpexporter.NewFactory()},
		{"OTLPHTTP", `{"otlpHTTP":{"endpoint":"https://otel.example","logsEndpoint":"https://logs.example/v1/logs","metricsEndpoint":"https://metrics.example/v1/metrics","headers":[{"key":"X-Tenant","value":"tenant"}],"auth":{"custom":{"header":"X-Auth","value":{"fromSecret":"mapping-credential"}}},"tls":{"insecureSkipVerify":true,"serverNameOverride":"otel.example","caPEM":"test-ca"},"encoding":"JSON","compression":"NONE","timeout":{"seconds":4},"readBufferSize":1024,"writeBufferSize":2048}}`, otlphttpexporter.NewFactory()},
		{"PrometheusRemoteWrite", `{"prometheusRemoteWrite":{"endpoint":"https://prometheus.example/api/v1/write","namespace":"octelium","headers":[{"key":"X-Tenant","value":"tenant"}],"externalLabels":[{"key":"cluster","value":"test"}],"auth":{"basic":{"username":"test-user","password":{"fromSecret":"mapping-credential"}}},"tls":{"insecureSkipVerify":true},"timeout":{"seconds":5},"remoteWriteQueue":{"enabled":false,"queueSize":100,"numConsumers":2},"resourceToTelemetryConversion":{"enabled":true,"excludeServiceAttributes":true},"targetInfo":{"enabled":false},"disableScopeInfo":true,"maxBatchSizeBytes":10000,"maxBatchRequestParallelism":2,"translationStrategy":"NO_TRANSLATION","sendMetadata":true}}`, prometheusremotewriteexporter.NewFactory()},
		{"Clickhouse", `{"clickhouse":{"endpoint":"tcp://clickhouse.example:9000","username":"test-user","password":{"fromSecret":"mapping-credential"},"database":"telemetry","tls":{"insecureSkipVerify":true},"connectionParams":[{"key":"dial_timeout","value":"10s"}],"logsTableName":"logs","metricsTables":{"gauge":"gauges","sum":"sums","summary":"summaries","histogram":"histograms","exponentialHistogram":"exponential"},"ttl":{"days":2},"createSchema":false,"compression":"ZSTD","asyncInsert":false,"json":true,"clusterName":"test-cluster","tableEngine":{"name":"MergeTree","params":"()"},"timeout":{"seconds":6}}}`, clickhouseexporter.NewFactory()},
		{"Elasticsearch", `{"elasticsearch":{"endpoint":"https://elastic.example","endpoints":["https://elastic.example","https://elastic2.example"],"cloudID":"test-cloud","pipeline":"ingest","logsIndex":"logs-test","metricsIndex":"metrics-test","headers":[{"key":"X-Tenant","value":"tenant"}],"timeout":{"seconds":7},"compression":"NONE","tls":{"insecureSkipVerify":true},"auth":{"apiKey":{"fromSecret":"mapping-credential"}}}}`, elasticsearchexporter.NewFactory()},
		{"Kafka", `{"kafka":{"brokers":["kafka.example:9092"],"protocolVersion":"2.8.0","clientID":"test-client","timeout":{"seconds":8},"connIdleTimeout":{"minutes":1},"partitionLogsByResourceAttributes":true,"partitionMetricsByResourceAttributes":true,"logs":{"topic":"logs","encoding":"RAW"},"metrics":{"topic":"metrics","encoding":"OTLP_JSON"},"tls":{"insecure":true,"insecureSkipVerify":true},"auth":{"sasl":{"username":"test-user","password":{"fromSecret":"mapping-credential"},"mechanism":"SCRAM_SHA_512"}},"producer":{"maxMessageBytes":100000,"requiredAcks":-1,"compression":"ZSTD","flushMaxMessages":100,"allowAutoTopicCreation":true,"linger":{"milliseconds":10}}}}`, kafkaexporter.NewFactory()},
		{"Datadog", `{"datadog":{"api":{"site":"datadoghq.eu","key":{"fromSecret":"mapping-credential"},"failOnInvalidKey":true},"hostname":"test-host","metrics":{"endpoint":"https://metrics.example","resourceAttributesAsTags":true,"instrumentationScopeMetadataAsTags":true},"logs":{"endpoint":"https://logs.example","useCompression":false,"compressionLevel":3},"hostMetadata":{"enabled":false,"reporterPeriod":{"minutes":5}},"hostnameDetectionTimeout":{"seconds":2}}}`, datadogexporter.NewFactory()},
		{"Logzio", `{"logzio":{"endpoint":"https://logzio.example","region":"us","token":{"fromSecret":"mapping-credential"},"timeout":{"seconds":9}}}`, logzioexporter.NewFactory()},
		{"Splunk", `{"splunk":{"endpoint":"https://splunk.example/services/collector","token":{"fromSecret":"mapping-credential"},"source":"test-source","sourceType":"test-type","index":"test-index","useMultiMetricFormat":true,"appName":"test-app","appVersion":"1.2.3","maxContentLengthLogs":10000,"maxContentLengthMetrics":20000,"disableCompression":true,"timeout":{"seconds":10},"maxIdleConns":3,"tls":{"insecureSkipVerify":true}}}`, splunkhecexporter.NewFactory()},
		{"AzureMonitor", `{"azureMonitor":{"connectionString":{"fromSecret":"mapping-credential"},"endpoint":"https://monitor.example","maxBatchSize":50,"maxBatchInterval":{"seconds":2},"shutdownTimeout":{"seconds":3},"customEventsEnabled":true,"exceptionEventsEnabled":true}}`, azuremonitorexporter.NewFactory()},
		{"InfluxDB", `{"influxDB":{"endpoint":"https://influx.example","org":"test-org","bucket":"test-bucket","token":{"fromSecret":"mapping-credential"},"headers":[{"key":"X-Tenant","value":"tenant"}],"metricsSchema":"TELEGRAF_PROMETHEUS_V2","precision":"MS","payloadMaxLines":100,"payloadMaxBytes":20000,"timeout":{"seconds":11},"logRecordDimensions":["level","service"],"v1Compatibility":{"enabled":true,"db":"test-db","username":"test-user","password":{"fromSecret":"mapping-credential"}}}}`, influxdbexporter.NewFactory()},
		{"AzureDataExplorer", `{"azureDataExplorer":{"clusterURI":"https://test.kusto.windows.net","database":"test-db","metricsTable":"metrics","logsTable":"logs","metricsTableMapping":"metric-map","logsTableMapping":"log-map","ingestionType":"MANAGED","timeout":{"seconds":12},"auth":{"servicePrincipal":{"applicationID":"test-app","tenantID":"test-tenant","applicationKey":{"fromSecret":"mapping-credential"}}}}}`, azuredataexplorerexporter.NewFactory()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info, err := p.getExporter(t.Context(), collectorExporterFromJSON(t, tc.spec))
			require.NoError(t, err)
			require.NoError(t, confmap.NewFromStringMap(info.exporterMap).Unmarshal(tc.factory.CreateDefaultConfig()))
		})
	}
}
