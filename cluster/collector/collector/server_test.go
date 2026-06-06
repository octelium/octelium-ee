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
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
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

func requireEventuallyExportLog(ctx context.Context, t *testing.T, port int) {
	t.Helper()

	require.Eventually(t, func() bool {
		err := exportTestLog(ctx, port)
		if err != nil {
			zap.L().Debug("Could not export test log yet", zap.Error(err))
			return false
		}
		return true
	}, 15*time.Second, 200*time.Millisecond)
}

func requireEventuallyExportMetric(ctx context.Context, t *testing.T, port int) {
	t.Helper()

	require.Eventually(t, func() bool {
		err := exportTestMetric(ctx, port)
		if err != nil {
			zap.L().Debug("Could not export test metric yet", zap.Error(err))
			return false
		}
		return true
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

	requireEventuallyExportLog(ctx, t, srv.p.port)
	requireEventuallyExportMetric(ctx, t, srv.p.port)

	require.Eventually(t, func() bool {
		return sink.logRecordCount() >= 1
	}, 15*time.Second, 200*time.Millisecond)

	require.Eventually(t, func() bool {
		return sink.metricPointCount() >= 1
	}, 15*time.Second, 200*time.Millisecond)

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

	requireEventuallyExportLog(ctx, t, srv.p.port)
	require.Eventually(t, func() bool {
		return sinkA.logRecordCount() >= 1
	}, 15*time.Second, 200*time.Millisecond)

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
