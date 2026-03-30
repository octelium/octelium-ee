// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package collector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

type tstSrvHTTP struct {
	port        int
	srv         *http.Server
	isHTTP2     bool
	crt         *tls.Certificate
	isWS        bool
	bearerToken string
	caPool      *x509.CertPool
	lis         net.Listener

	wait      time.Duration
	startedAt time.Time
}

type tstResp struct {
	Hello string `json:"hello"`
}

func (s *tstSrvHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	zap.L().Debug("__________COLL REQ_______",
		zap.Any("req", r.Header), zap.Any("path", r.URL.Path), zap.String("a", r.Header.Get("Authorization")))

	if s.wait > 0 && time.Since(s.startedAt) < s.wait {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		var req tstResp
		if err := json.Unmarshal(body, &req); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp, err := json.Marshal(&tstResp{
			Hello: req.Hello,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(resp)
		return
	}

	if s.bearerToken != "" {
		bearer := r.Header.Get("Authorization")
		tkn := strings.TrimPrefix(bearer, "Bearer ")
		if s.bearerToken != tkn {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	resp, err := json.Marshal(&tstResp{
		Hello: "world",
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(resp)
	return

}

func newSrvHTTP(t *testing.T, port int, isHTTP2 bool, crt *tls.Certificate) *tstSrvHTTP {
	return &tstSrvHTTP{
		port:    port,
		isHTTP2: isHTTP2,
		crt:     crt,
	}
}

func (s *tstSrvHTTP) run(t *testing.T) {
	addr := fmt.Sprintf("localhost:%d", s.port)
	var err error

	handler := http.AllowQuerySemicolons(s)
	if s.isHTTP2 {
		handler = h2c.NewHandler(handler, &http2.Server{})
	}
	s.srv = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	if s.crt != nil {
		zap.L().Debug("upstream listening over TLS")
		s.lis, err = func() (net.Listener, error) {
			for range 100 {
				ret, err := tls.Listen("tcp", addr, s.getTLSConfig())
				if err == nil {
					return ret, nil
				}
				time.Sleep(1 * time.Second)
			}
			return nil, errors.Errorf("Could not listen tstSrvHTTP")
		}()
		assert.Nil(t, err)
	} else {
		s.lis, err = func() (net.Listener, error) {
			for range 100 {
				ret, err := net.Listen("tcp", addr)
				if err == nil {
					return ret, nil
				}
				time.Sleep(1 * time.Second)
			}
			return nil, errors.Errorf("Could not listen tstSrvHTTP")
		}()
		assert.Nil(t, err)
	}

	s.startedAt = time.Now()
	go s.srv.Serve(s.lis)
}

func (s *tstSrvHTTP) getTLSConfig() *tls.Config {
	if s.crt == nil {
		return nil
	}

	return &tls.Config{
		Certificates: []tls.Certificate{*s.crt},
		NextProtos: func() []string {
			if s.isHTTP2 {
				return []string{"h2", "http/1.1"}
			} else {
				return []string{"http/1.1"}
			}
		}(),
		RootCAs: s.caPool,
	}

}

func (s *tstSrvHTTP) close() {
	if s.srv != nil {
		s.srv.Close()
	}
	if s.lis != nil {
		s.lis.Close()
	}

	time.Sleep(1 * time.Second)
}

func TestServer(t *testing.T) {
	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	assert.Nil(t, err)

	zap.ReplaceGlobals(logger)

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := &Server{
		octeliumC:    fakeC.OcteliumC,
		receiverPort: tests.GetPort(),
	}
	zap.S().Debugf("Running srv")

	ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()

	/*
		err = srv.initCollectorExporters(ctx)
		assert.Nil(t, err)
	*/
	err = srv.doRun(ctx)
	assert.Nil(t, err)
	zap.S().Debugf("Starting watch loop")

	srv.p.sendUpdate()
	time.Sleep(1 * time.Second)

	upstreamPort := tests.GetPort()
	upstreamSrv := newSrvHTTP(t, upstreamPort, false, nil)

	upstreamSrv.run(t)

	{
		cc, err := fakeC.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err)

		_, err = fakeC.OcteliumC.CoreC().UpdateClusterConfig(ctx, cc)
		assert.Nil(t, err)

		srv.p.sendUpdate()
		time.Sleep(1 * time.Second)
	}

	dummySecret, err := fakeC.OcteliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   &enterprisev1.Secret_Spec{},
		Status: &enterprisev1.Secret_Status{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: "octelium",
			},
		},
	})
	assert.Nil(t, err)

	otlpExporter, err := fakeC.OcteliumC.EnterpriseC().CreateCollectorExporter(ctx, &enterprisev1.CollectorExporter{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.CollectorExporter_Spec{
			Type: &enterprisev1.CollectorExporter_Spec_OtlpHTTP{
				OtlpHTTP: &enterprisev1.CollectorExporter_Spec_OTLPHTTP{
					Endpoint: fmt.Sprintf("http://localhost:%d", upstreamPort),
					Auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{
						Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_{
							Basic: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic{
								Username: "123456",
								Password: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password{
									Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password_FromSecret{
										FromSecret: dummySecret.Metadata.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Nil(t, err)

	{

		/*
			clickhouseCollector, err := fakeC.OcteliumC.EnterpriseC().CreateCollectorExporter(ctx, &enterprisev1.CollectorExporter{
				Metadata: &metav1.Metadata{
					Name: utilrand.GetRandomStringCanonical(6),
				},
				Spec: &enterprisev1.CollectorExporter_Spec{
					Type: &enterprisev1.CollectorExporter_Spec_Clickhouse_{
						Clickhouse: &enterprisev1.CollectorExporter_Spec_Clickhouse{
							Endpoint: "tcp://127.0.0.1:9000?dial_timeout=10s",
							Username: "octelium",
							Password: &enterprisev1.CollectorExporter_Spec_Clickhouse_Password{
								Type: &enterprisev1.CollectorExporter_Spec_Clickhouse_Password_FromSecret{
									FromSecret: dummySecret.Metadata.Name,
								},
							},
						},
					},
				},
				Status: &enterprisev1.CollectorExporter_Status{},
			})
			assert.Nil(t, err)
		*/

		/*
			logzioCollector, err := fakeC.OcteliumC.EnterpriseC().CreateCollectorExporter(ctx, &enterprisev1.CollectorExporter{
				Metadata: &metav1.Metadata{
					Name: utilrand.GetRandomStringCanonical(6),
				},
				Spec: &enterprisev1.CollectorExporter_Spec{
					Type: &enterprisev1.CollectorExporter_Spec_Logzio_{
						Logzio: &enterprisev1.CollectorExporter_Spec_Logzio{},
					},
				},
				Status: &enterprisev1.CollectorExporter_Status{},
			})
			assert.Nil(t, err)
		*/

		/*
			influxDBCollector, err := fakeC.OcteliumC.EnterpriseC().CreateCollectorExporter(ctx, &enterprisev1.CollectorExporter{
				Metadata: &metav1.Metadata{
					Name: utilrand.GetRandomStringCanonical(6),
				},
				Spec: &enterprisev1.CollectorExporter_Spec{
					Type: &enterprisev1.CollectorExporter_Spec_InfluxDB_{
						InfluxDB: &enterprisev1.CollectorExporter_Spec_InfluxDB{
							Endpoint:      "https://eu-central-1-1.aws.cloud2.influxdata.com",
							Bucket:        "bucket-01",
							Org:           "org-1",
							MetricsSchema: "telegraf-prometheus-",

							Token: &enterprisev1.CollectorExporter_Spec_InfluxDB_Token{
								Type: &enterprisev1.CollectorExporter_Spec_InfluxDB_Token_FromSecret{
									FromSecret: dummySecret.Metadata.Name,
								},
							},
						},
					},
				},
				Status: &enterprisev1.CollectorExporter_Status{},
			})
			assert.Nil(t, err)
		*/

		cc, err := fakeC.OcteliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
		assert.Nil(t, err)
		cc.Spec.Collector = &enterprisev1.ClusterConfig_Spec_Collector{

			Pipelines: []*enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
				{
					Name: "pipline1",
					Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS,
					Exporters: []string{
						otlpExporter.Metadata.Name,
						// clickhouseCollector.Metadata.Name,
						// logzioCollector.Metadata.Name,
						// influxDBCollector.Metadata.Name,
					},
				},
				{
					Name: "pipline2",
					Type: enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS,
					Exporters: []string{
						otlpExporter.Metadata.Name,
						// clickhouseCollector.Metadata.Name,
						// logzioCollector.Metadata.Name,
						// influxDBCollector.Metadata.Name,
					},
				},
			},

			/*
							AdditionalInlineConfig: `
				receivers:
				  otlp:
				    protocols:
				      grpc:
				      http:
				exporters:
				  otlp/gg01:
				    endpoint: otelcol.observability.svc.cluster.local:443

				service:
				  extensions: []
				  pipelines:
				    logs/a01:
				      receivers: [otlp]
				      exporters: [otlp/gg01]
				`,

			*/
		}

		cc, err = fakeC.OcteliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
		assert.Nil(t, err)

		srv.p.sendUpdate()
		time.Sleep(1 * time.Second)
	}

	{
		grpcOpts := []grpc.DialOption{
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff: backoff.DefaultConfig,
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}

		conn, err := grpc.NewClient(
			fmt.Sprintf("localhost:%d", srv.ccController.p.port), grpcOpts...)
		assert.Nil(t, err)
		client := plogotlp.NewGRPCClient(conn)

		curLogs := plog.NewLogs()
		curLogs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()

		logRecords := curLogs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
		lr := logRecords.AppendEmpty()
		convertLogRecord(&corev1.AccessLog{
			Metadata: &metav1.LogMetadata{
				Id:        utilrand.GetRandomStringCanonical(18),
				CreatedAt: pbutils.Now(),
			},
		}, lr)
		_, err = client.Export(ctx, plogotlp.NewExportRequestFromLogs(curLogs))
		assert.Nil(t, err, "%+v", err)
	}

	time.Sleep(3 * time.Second)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		srv.collector.Shutdown()
		cancel()
	}()

	select {
	case <-ctx.Done():
	case <-time.After(15 * time.Second):
		zap.S().Fatal("Could not shutdown server")
	}

}

func convertLogRecord(in *corev1.AccessLog, ret plog.LogRecord) {
	inMap := pbutils.MustConvertToMap(in)

	ret.SetTimestamp(pcommon.NewTimestampFromTime(in.Metadata.CreatedAt.AsTime()))
	ret.SetObservedTimestamp(pcommon.NewTimestampFromTime(in.Metadata.CreatedAt.AsTime()))
	ret.SetSeverityNumber(plog.SeverityNumberInfo)
	ret.SetSeverityText(plog.SeverityNumberInfo.String())
	ret.Body().SetEmptyMap().FromRaw(inMap)
}
