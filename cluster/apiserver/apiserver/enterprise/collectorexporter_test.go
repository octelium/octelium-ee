// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"math"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

type collectorExporterCase[T proto.Message] struct {
	name   string
	mutate func(T)
}

func tstCollectorExporterValidation[T proto.Message](t *testing.T, spec T, validate func(context.Context, T) error, cases []collectorExporterCase[T]) {
	t.Helper()

	t.Run("Valid", func(t *testing.T) {
		assert.Nil(t, validate(t.Context(), proto.Clone(spec).(T)))
	})
	t.Run("Nil", func(t *testing.T) {
		var arg T
		err := validate(t.Context(), arg)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	})
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			arg := proto.Clone(spec).(T)
			tt.mutate(arg)
			err := validate(t.Context(), arg)
			assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
		})
	}
}

func TestCollectorExporterValidation(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)
	secretName := tstSecretName()
	_, err = srv.CreateSecret(t.Context(), tstSecret(secretName, tstSecretValue("credential")))
	require.Nil(t, err)

	for _, arg := range []*enterprisev1.CollectorExporter{nil, {}, {Spec: &enterprisev1.CollectorExporter_Spec{}}} {
		err := srv.validateCollectorExporter(t.Context(), arg)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	t.Run("OTLP", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_OTLP{Endpoint: "collector.example.com:4317"}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Otlp{Otlp: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Otlp{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterOTLP, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_OTLP]{
			{name: "Endpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.Endpoint = "" }},
			{name: "EndpointScheme", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.Endpoint = "ftp://collector.example.com:4317" }},
			{name: "Headers", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) {
				arg.Headers = []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "Authorization"}}
			}},
			{name: "TLS", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) {
				arg.Tls = &enterprisev1.CollectorExporter_Spec_OTLP_TLS{Insecure: true, InsecureSkipVerify: true}
			}},
			{name: "Compression", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.Compression = 99 }},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) {
				arg.Timeout = tstCollectorDuration(time.Nanosecond)
			}},
			{name: "Authority", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.Authority = strings.Repeat("a", 254) }},
			{name: "UserAgent", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.UserAgent = strings.Repeat("a", 513) }},
			{name: "Balancer", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.BalancerName = "unknown" }},
			{name: "ReadBufferNegative", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.ReadBufferSize = -1 }},
			{name: "WriteBufferNegative", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.WriteBufferSize = -1 }},
			{name: "ReadBufferLarge", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.ReadBufferSize = 64*1024*1024 + 1 }},
			{name: "WriteBufferLarge", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) { arg.WriteBufferSize = 64*1024*1024 + 1 }},
			{name: "KeepaliveTime", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) {
				arg.Keepalive = &enterprisev1.CollectorExporter_Spec_OTLP_Keepalive{Time: tstCollectorDuration(time.Millisecond)}
			}},
			{name: "KeepaliveTimeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLP) {
				arg.Keepalive = &enterprisev1.CollectorExporter_Spec_OTLP_Keepalive{Timeout: tstCollectorDuration(11 * time.Minute)}
			}},
		})
	})

	t.Run("OTLPHTTP", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_OTLPHTTP{Endpoint: "https://collector.example.com"}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_OtlpHTTP{OtlpHTTP: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_OtlpHTTP{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterOTLPHTTP, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_OTLPHTTP]{
			{name: "Endpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.Endpoint = "" }},
			{name: "EndpointScheme", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.Endpoint = "ftp://collector.example.com" }},
			{name: "LogsEndpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.LogsEndpoint = "/v1/logs" }},
			{name: "MetricsEndpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.MetricsEndpoint = "/v1/metrics" }},
			{name: "Headers", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) {
				arg.Headers = []*enterprisev1.CollectorExporter_Spec_OTLPHTTP_Header{{Key: "bad key"}}
			}},
			{name: "TLS", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) {
				arg.Tls = &enterprisev1.CollectorExporter_Spec_OTLPHTTP_TLS{Insecure: true, InsecureSkipVerify: true}
			}},
			{name: "Encoding", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.Encoding = 99 }},
			{name: "Compression", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.Compression = 99 }},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) {
				arg.Timeout = tstCollectorDuration(11 * time.Minute)
			}},
			{name: "ReadBufferNegative", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.ReadBufferSize = -1 }},
			{name: "WriteBufferNegative", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.WriteBufferSize = -1 }},
			{name: "ReadBufferLarge", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.ReadBufferSize = 64*1024*1024 + 1 }},
			{name: "WriteBufferLarge", mutate: func(arg *enterprisev1.CollectorExporter_Spec_OTLPHTTP) { arg.WriteBufferSize = 64*1024*1024 + 1 }},
		})
	})

	t.Run("Splunk", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_Splunk{Endpoint: "https://splunk.example.com:8088",
			Token: &enterprisev1.CollectorExporter_Spec_Splunk_Token{Type: &enterprisev1.CollectorExporter_Spec_Splunk_Token_FromSecret{FromSecret: secretName}}}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Splunk_{Splunk: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Splunk_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterSplunk, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_Splunk]{
			{name: "Endpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) { arg.Endpoint = "" }},
			{name: "EndpointScheme", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) { arg.Endpoint = "ftp://splunk.example.com" }},
			{name: "Token", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) { arg.Token = nil }},
			{name: "EmptyToken", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) {
				arg.Token = &enterprisev1.CollectorExporter_Spec_Splunk_Token{}
			}},
			{name: "MissingSecret", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) {
				arg.Token.Type = &enterprisev1.CollectorExporter_Spec_Splunk_Token_FromSecret{FromSecret: "missing-secret"}
			}},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) {
				arg.Timeout = tstCollectorDuration(time.Millisecond)
			}},
			{name: "MaxIdleConnsNegative", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) { arg.MaxIdleConns = -1 }},
			{name: "MaxIdleConnsLarge", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) { arg.MaxIdleConns = 10001 }},
			{name: "LogsLength", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) { arg.MaxContentLengthLogs = -1 }},
			{name: "MetricsLength", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) { arg.MaxContentLengthMetrics = -1 }},
			{name: "PlaintextTLS", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Splunk) {
				arg.Endpoint = "http://splunk.example.com"
				arg.Tls = &enterprisev1.CollectorExporter_Spec_Splunk_TLS{InsecureSkipVerify: true}
			}},
		})
	})

	t.Run("Logzio", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_Logzio{Token: &enterprisev1.CollectorExporter_Spec_Logzio_Token{Type: &enterprisev1.CollectorExporter_Spec_Logzio_Token_FromSecret{FromSecret: secretName}}}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Logzio_{Logzio: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Logzio_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterLogzio, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_Logzio]{
			{name: "Token", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Logzio) { arg.Token = nil }},
			{name: "EmptyToken", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Logzio) {
				arg.Token = &enterprisev1.CollectorExporter_Spec_Logzio_Token{}
			}},
			{name: "MissingSecret", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Logzio) {
				arg.Token.Type = &enterprisev1.CollectorExporter_Spec_Logzio_Token_FromSecret{FromSecret: "missing-secret"}
			}},
			{name: "BlankRegion", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Logzio) { arg.Region = "  " }},
			{name: "LongRegion", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Logzio) { arg.Region = strings.Repeat("a", 33) }},
			{name: "RegionCharacters", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Logzio) { arg.Region = "us/east" }},
			{name: "Endpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Logzio) { arg.Endpoint = "ftp://logzio.example.com" }},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Logzio) {
				arg.Timeout = tstCollectorDuration(6 * time.Minute)
			}},
		})
	})

	t.Run("Elasticsearch", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_Elasticsearch{Endpoint: "https://elastic.example.com"}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Elasticsearch_{Elasticsearch: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Elasticsearch_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterElasticsearch, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_Elasticsearch]{
			{name: "Destination", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) { arg.Endpoint = "" }},
			{name: "ConflictingEndpoints", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) {
				arg.Endpoints = []string{"https://other.example.com"}
			}},
			{name: "ConflictingCloudID", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) { arg.CloudID = "cloud-id" }},
			{name: "Endpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) {
				arg.Endpoint = "ftp://elastic.example.com"
			}},
			{name: "DuplicateEndpoints", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) {
				arg.Endpoint = ""
				arg.Endpoints = []string{"https://elastic.example.com", "https://elastic.example.com"}
			}},
			{name: "InvalidEndpointList", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) {
				arg.Endpoint = ""
				arg.Endpoints = []string{"/relative"}
			}},
			{name: "TooManyEndpoints", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) {
				arg.Endpoint = ""
				arg.Endpoints = make([]string, 33)
			}},
			{name: "CloudIDLength", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) {
				arg.Endpoint = ""
				arg.CloudID = strings.Repeat("a", 4097)
			}},
			{name: "Headers", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) {
				arg.Headers = []*enterprisev1.CollectorExporter_Spec_Elasticsearch_Header{{Key: "Authorization"}}
			}},
			{name: "Auth", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) {
				arg.Auth = &enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth{}
			}},
			{name: "Compression", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) { arg.Compression = 99 }},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) {
				arg.Timeout = tstCollectorDuration(11 * time.Minute)
			}},
			{name: "LogsIndex", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) { arg.LogsIndex = "bad index" }},
			{name: "MetricsIndex", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) { arg.MetricsIndex = "_metrics" }},
			{name: "Pipeline", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Elasticsearch) { arg.Pipeline = strings.Repeat("a", 257) }},
		})
	})

	t.Run("InfluxDB", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_InfluxDB{Endpoint: "https://influx.example.com", Org: "octelium", Bucket: "metrics"}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_InfluxDB_{InfluxDB: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_InfluxDB_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterInfluxDB, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_InfluxDB]{
			{name: "Endpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.Endpoint = "" }},
			{name: "EndpointScheme", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.Endpoint = "ftp://influx.example.com" }},
			{name: "Org", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.Org = " " }},
			{name: "Bucket", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.Bucket = "" }},
			{name: "LongOrg", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.Org = strings.Repeat("a", 257) }},
			{name: "LongBucket", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.Bucket = strings.Repeat("a", 257) }},
			{name: "Token", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.Token = &enterprisev1.CollectorExporter_Spec_InfluxDB_Token{}
			}},
			{name: "MissingSecret", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.Token = &enterprisev1.CollectorExporter_Spec_InfluxDB_Token{Type: &enterprisev1.CollectorExporter_Spec_InfluxDB_Token_FromSecret{FromSecret: "missing-secret"}}
			}},
			{name: "Headers", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.Headers = []*enterprisev1.CollectorExporter_Spec_InfluxDB_Header{{Key: "Authorization"}}
			}},
			{name: "MetricsSchema", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.MetricsSchema = 99 }},
			{name: "Precision", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.Precision = 99 }},
			{name: "NegativeLines", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.PayloadMaxLines = -1 }},
			{name: "NegativeBytes", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.PayloadMaxBytes = -1 }},
			{name: "LargeLines", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.PayloadMaxLines = 1000001 }},
			{name: "LargeBytes", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.PayloadMaxBytes = 256*1024*1024 + 1 }},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.Timeout = tstCollectorDuration(11 * time.Minute)
			}},
			{name: "TooManyDimensions", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.LogRecordDimensions = make([]string, 65) }},
			{name: "EmptyDimension", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) { arg.LogRecordDimensions = []string{" "} }},
			{name: "LongDimension", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.LogRecordDimensions = []string{strings.Repeat("a", 257)}
			}},
			{name: "DuplicateDimensions", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.LogRecordDimensions = []string{"host", " host "}
			}},
			{name: "V1Database", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.V1Compatibility = &enterprisev1.CollectorExporter_Spec_InfluxDB_V1Compatibility{Enabled: true}
			}},
			{name: "V1LongDatabase", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.V1Compatibility = &enterprisev1.CollectorExporter_Spec_InfluxDB_V1Compatibility{Db: strings.Repeat("a", 257)}
			}},
			{name: "V1LongUsername", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.V1Compatibility = &enterprisev1.CollectorExporter_Spec_InfluxDB_V1Compatibility{Username: strings.Repeat("a", 257)}
			}},
			{name: "V1Password", mutate: func(arg *enterprisev1.CollectorExporter_Spec_InfluxDB) {
				arg.V1Compatibility = &enterprisev1.CollectorExporter_Spec_InfluxDB_V1Compatibility{Password: &enterprisev1.CollectorExporter_Spec_InfluxDB_V1Compatibility_Password{}}
			}},
		})
	})

	t.Run("Kafka", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_Kafka{
			Brokers: []string{"kafka.example.com:9092"},
			RecordHeaders: []*enterprisev1.CollectorExporter_Spec_Kafka_Header{
				{Key: "X-Record", Value: "value\x00\r\n\x7f"},
			},
		}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Kafka_{Kafka: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Kafka_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterKafka, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_Kafka]{
			{name: "TooManyBrokers", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) { arg.Brokers = make([]string, 65) }},
			{name: "EmptyBroker", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) { arg.Brokers = []string{" "} }},
			{name: "LongBroker", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) { arg.Brokers = []string{strings.Repeat("a", 256)} }},
			{name: "DuplicateBroker", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) {
				arg.Brokers = []string{"host:9092", " host:9092 "}
			}},
			{name: "ProtocolVersion", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) { arg.ProtocolVersion = strings.Repeat("a", 33) }},
			{name: "ClientID", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) { arg.ClientID = strings.Repeat("a", 129) }},
			{name: "Logs", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) {
				arg.Logs = &enterprisev1.CollectorExporter_Spec_Kafka_Signal{Encoding: 99}
			}},
			{name: "Metrics", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) {
				arg.Metrics = &enterprisev1.CollectorExporter_Spec_Kafka_Signal{Encoding: enterprisev1.CollectorExporter_Spec_Kafka_RAW}
			}},
			{name: "Headers", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) {
				arg.RecordHeaders = []*enterprisev1.CollectorExporter_Spec_Kafka_Header{{Key: "bad key"}}
			}},
			{name: "Auth", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) {
				arg.Auth = &enterprisev1.CollectorExporter_Spec_Kafka_Auth{}
			}},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) {
				arg.Timeout = tstCollectorDuration(time.Millisecond)
			}},
			{name: "IdleTimeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) {
				arg.ConnIdleTimeout = tstCollectorDuration(3 * time.Hour)
			}},
			{name: "Producer", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Kafka) {
				arg.Producer = &enterprisev1.CollectorExporter_Spec_Kafka_Producer{RequiredAcks: 2}
			}},
		})
	})

	t.Run("Datadog", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_Datadog{Api: &enterprisev1.CollectorExporter_Spec_Datadog_API{Key: &enterprisev1.CollectorExporter_Spec_Datadog_APIKey{Type: &enterprisev1.CollectorExporter_Spec_Datadog_APIKey_FromSecret{FromSecret: secretName}}}}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Datadog_{Datadog: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Datadog_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterDatadog, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_Datadog]{
			{name: "API", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) { arg.Api = nil }},
			{name: "Key", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) { arg.Api.Key = nil }},
			{name: "EmptyKey", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) {
				arg.Api.Key = &enterprisev1.CollectorExporter_Spec_Datadog_APIKey{}
			}},
			{name: "MissingSecret", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) {
				arg.Api.Key.Type = &enterprisev1.CollectorExporter_Spec_Datadog_APIKey_FromSecret{FromSecret: "missing-secret"}
			}},
			{name: "Site", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) { arg.Api.Site = "https://datadoghq.com" }},
			{name: "LongSite", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) { arg.Api.Site = strings.Repeat("a", 254) }},
			{name: "Hostname", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) { arg.Hostname = "host\nname" }},
			{name: "LongHostname", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) { arg.Hostname = strings.Repeat("a", 256) }},
			{name: "MetricsEndpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) {
				arg.Metrics = &enterprisev1.CollectorExporter_Spec_Datadog_Metrics{Endpoint: "ftp://datadog.example.com"}
			}},
			{name: "LogsEndpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) {
				arg.Logs = &enterprisev1.CollectorExporter_Spec_Datadog_Logs{Endpoint: "/logs"}
			}},
			{name: "LowCompression", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) {
				arg.Logs = &enterprisev1.CollectorExporter_Spec_Datadog_Logs{CompressionLevel: -1}
			}},
			{name: "HighCompression", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) {
				arg.Logs = &enterprisev1.CollectorExporter_Spec_Datadog_Logs{CompressionLevel: 10}
			}},
			{name: "BatchWait", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) {
				arg.Logs = &enterprisev1.CollectorExporter_Spec_Datadog_Logs{BatchWait: tstCollectorDuration(6 * time.Minute)}
			}},
			{name: "ReporterPeriod", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) {
				arg.HostMetadata = &enterprisev1.CollectorExporter_Spec_Datadog_HostMetadata{ReporterPeriod: tstCollectorDuration(time.Minute)}
			}},
			{name: "DetectionTimeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Datadog) {
				arg.HostnameDetectionTimeout = tstCollectorDuration(6 * time.Minute)
			}},
		})
	})

	t.Run("AzureMonitor", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_AzureMonitor{ConnectionString: &enterprisev1.CollectorExporter_Spec_AzureMonitor_ConnectionString{Type: &enterprisev1.CollectorExporter_Spec_AzureMonitor_ConnectionString_FromSecret{FromSecret: secretName}}}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_AzureMonitor_{AzureMonitor: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_AzureMonitor_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterAzureMonitor, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_AzureMonitor]{
			{name: "Credentials", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureMonitor) { arg.ConnectionString = nil }},
			{name: "BothCredentials", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureMonitor) {
				arg.InstrumentationKey = &enterprisev1.CollectorExporter_Spec_AzureMonitor_InstrumentationKey{Type: &enterprisev1.CollectorExporter_Spec_AzureMonitor_InstrumentationKey_FromSecret{FromSecret: secretName}}
			}},
			{name: "MissingSecret", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureMonitor) {
				arg.ConnectionString.Type = &enterprisev1.CollectorExporter_Spec_AzureMonitor_ConnectionString_FromSecret{FromSecret: "missing-secret"}
			}},
			{name: "Endpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureMonitor) { arg.Endpoint = "/monitor" }},
			{name: "NegativeBatch", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureMonitor) { arg.MaxBatchSize = -1 }},
			{name: "LargeBatch", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureMonitor) { arg.MaxBatchSize = 100001 }},
			{name: "BatchInterval", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureMonitor) {
				arg.MaxBatchInterval = tstCollectorDuration(time.Millisecond)
			}},
			{name: "ShutdownTimeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureMonitor) {
				arg.ShutdownTimeout = tstCollectorDuration(6 * time.Minute)
			}},
		})
	})

	t.Run("AzureDataExplorer", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_AzureDataExplorer{ClusterURI: "https://cluster.kusto.windows.net",
			Auth: &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth{Type: &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth_AzureDefault{AzureDefault: &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_AzureDefaultAuth{}}}}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_{AzureDataExplorer: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterAzureDataExplorer, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_AzureDataExplorer]{
			{name: "ClusterURI", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) { arg.ClusterURI = "" }},
			{name: "InvalidURI", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) { arg.ClusterURI = "/cluster" }},
			{name: "HTTP", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) {
				arg.ClusterURI = "http://cluster.kusto.windows.net"
			}},
			{name: "Auth", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) { arg.Auth = nil }},
			{name: "AuthType", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) {
				arg.Auth = &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth{}
			}},
			{name: "EmptyManagedIdentity", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) {
				arg.Auth.Type = &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth_ManagedIdentity{ManagedIdentity: &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_ManagedIdentity{}}
			}},
			{name: "InvalidManagedIdentity", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) {
				arg.Auth.Type = &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth_ManagedIdentity{ManagedIdentity: &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_ManagedIdentity{Id: "bad-id"}}
			}},
			{name: "NilManagedIdentity", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) {
				arg.Auth.Type = &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth_ManagedIdentity{}
			}},
			{name: "NilPrincipal", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) {
				arg.Auth.Type = &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth_ServicePrincipal{}
			}},
			{name: "PrincipalID", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) {
				arg.Auth.Type = &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth_ServicePrincipal{ServicePrincipal: &enterprisev1.CollectorExporter_Spec_AzureDataExplorer_ServicePrincipal{ApplicationID: "bad-id"}}
			}},
			{name: "Database", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) { arg.Database = "bad-name" }},
			{name: "MetricsTable", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) { arg.MetricsTable = "bad.table" }},
			{name: "LogsTable", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) { arg.LogsTable = "bad table" }},
			{name: "MetricsMapping", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) {
				arg.MetricsTableMapping = "bad mapping"
			}},
			{name: "LogsMapping", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) { arg.LogsTableMapping = "bad/mapping" }},
			{name: "IngestionType", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) { arg.IngestionType = 99 }},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_AzureDataExplorer) {
				arg.Timeout = tstCollectorDuration(11 * time.Minute)
			}},
		})
	})

	t.Run("Clickhouse", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_Clickhouse{Endpoint: "tcp://clickhouse.example.com:9000"}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Clickhouse_{Clickhouse: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_Clickhouse_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterClickhouse, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_Clickhouse]{
			{name: "Endpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) { arg.Endpoint = "" }},
			{name: "EndpointScheme", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.Endpoint = "ftp://clickhouse.example.com"
			}},
			{name: "Username", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) { arg.Username = strings.Repeat("a", 257) }},
			{name: "Password", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.Password = &enterprisev1.CollectorExporter_Spec_Clickhouse_Password{}
			}},
			{name: "MissingSecret", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.Password = &enterprisev1.CollectorExporter_Spec_Clickhouse_Password{Type: &enterprisev1.CollectorExporter_Spec_Clickhouse_Password_FromSecret{FromSecret: "missing-secret"}}
			}},
			{name: "Database", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) { arg.Database = "bad-name" }},
			{name: "TLS", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.Tls = &enterprisev1.CollectorExporter_Spec_Clickhouse_TLS{Insecure: true, InsecureSkipVerify: true}
			}},
			{name: "TooManyParams", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.ConnectionParams = make([]*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam, 65)
			}},
			{name: "NilParam", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.ConnectionParams = []*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam{nil}
			}},
			{name: "EmptyParam", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.ConnectionParams = []*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam{{Key: " "}}
			}},
			{name: "LargeParam", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.ConnectionParams = []*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam{{Key: strings.Repeat("a", 129)}}
			}},
			{name: "LargeParamValue", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.ConnectionParams = []*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam{{Key: "setting", Value: strings.Repeat("a", 4097)}}
			}},
			{name: "DuplicateParams", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.ConnectionParams = []*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam{{Key: "Setting"}, {Key: "setting"}}
			}},
			{name: "PasswordParam", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.ConnectionParams = []*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam{{Key: "PASSWORD"}}
			}},
			{name: "UsernameParam", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.ConnectionParams = []*enterprisev1.CollectorExporter_Spec_Clickhouse_ConnectionParam{{Key: "username"}}
			}},
			{name: "LogsTable", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) { arg.LogsTableName = "bad.table" }},
			{name: "MetricsTable", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.MetricsTables = &enterprisev1.CollectorExporter_Spec_Clickhouse_MetricsTables{Gauge: "bad-name"}
			}},
			{name: "TTL", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.Ttl = tstCollectorDuration(366 * 24 * time.Hour)
			}},
			{name: "Compression", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) { arg.Compression = 99 }},
			{name: "ClusterName", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) { arg.ClusterName = "bad-name" }},
			{name: "EngineName", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.TableEngine = &enterprisev1.CollectorExporter_Spec_Clickhouse_TableEngine{Name: "bad-engine"}
			}},
			{name: "EngineParams", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.TableEngine = &enterprisev1.CollectorExporter_Spec_Clickhouse_TableEngine{Params: strings.Repeat("a", 4097)}
			}},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_Clickhouse) {
				arg.Timeout = tstCollectorDuration(11 * time.Minute)
			}},
		})
	})

	t.Run("PrometheusRemoteWrite", func(t *testing.T) {
		spec := &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite{Endpoint: "https://prometheus.example.com/api/v1/write"}
		err := srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_{PrometheusRemoteWrite: spec}},
		})
		require.Nil(t, err, "%+v", err)
		err = srv.validateCollectorExporter(t.Context(), &enterprisev1.CollectorExporter{
			Spec: &enterprisev1.CollectorExporter_Spec{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_{}},
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		tstCollectorExporterValidation(t, spec, srv.validateCollectorExporterPrometheusRemoteWrite, []collectorExporterCase[*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite]{
			{name: "Endpoint", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) { arg.Endpoint = "" }},
			{name: "EndpointScheme", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.Endpoint = "ftp://prometheus.example.com"
			}},
			{name: "Namespace", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) { arg.Namespace = "9metrics" }},
			{name: "LongNamespace", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.Namespace = strings.Repeat("a", 129)
			}},
			{name: "Headers", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.Headers = []*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Header{{Key: "Authorization"}}
			}},
			{name: "TLS", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.Tls = &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_TLS{Insecure: true, InsecureSkipVerify: true}
			}},
			{name: "TooManyLabels", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.ExternalLabels = make([]*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel, 65)
			}},
			{name: "NilLabel", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.ExternalLabels = []*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel{nil}
			}},
			{name: "EmptyLabel", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.ExternalLabels = []*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel{{Key: " "}}
			}},
			{name: "LongLabel", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.ExternalLabels = []*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel{{Key: strings.Repeat("a", 129)}}
			}},
			{name: "LongLabelValue", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.ExternalLabels = []*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel{{Key: "host", Value: strings.Repeat("a", 4097)}}
			}},
			{name: "InvalidLabel", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.ExternalLabels = []*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel{{Key: "bad-key"}}
			}},
			{name: "DuplicateLabels", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.ExternalLabels = []*enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel{{Key: "host"}, {Key: "host"}}
			}},
			{name: "Timeout", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.Timeout = tstCollectorDuration(11 * time.Minute)
			}},
			{name: "NegativeQueue", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.RemoteWriteQueue = &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_RemoteWriteQueue{QueueSize: -1}
			}},
			{name: "EmptyEnabledQueue", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.RemoteWriteQueue = &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_RemoteWriteQueue{Enabled: true}
			}},
			{name: "LargeQueue", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.RemoteWriteQueue = &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_RemoteWriteQueue{QueueSize: 10000001}
			}},
			{name: "Consumers", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.RemoteWriteQueue = &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_RemoteWriteQueue{NumConsumers: 1025}
			}},
			{name: "NegativeBatch", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) { arg.MaxBatchSizeBytes = -1 }},
			{name: "LargeBatch", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.MaxBatchSizeBytes = 512*1024*1024 + 1
			}},
			{name: "NegativeParallelism", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.MaxBatchRequestParallelism = -1
			}},
			{name: "LargeParallelism", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.MaxBatchRequestParallelism = 1025
			}},
			{name: "UTF8Strategy", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.TranslationStrategy = enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_NO_UTF8_ESCAPING_WITH_SUFFIXES
			}},
			{name: "NoTranslation", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) {
				arg.TranslationStrategy = enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_NO_TRANSLATION
			}},
			{name: "TranslationStrategy", mutate: func(arg *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite) { arg.TranslationStrategy = 99 }},
		})
	})
}

func tstCollectorDuration(d time.Duration) *metav1.Duration {
	if d >= time.Second {
		return &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: uint32(d / time.Second)}}
	}
	return &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: uint32(d / time.Millisecond)}}
}

func newEnterpriseAPI(t *testing.T) (enterprisev1.MainServiceClient, *tests.T) {
	t.Helper()

	tst, err := tests.Initialize(nil)
	require.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	enterprisev1.RegisterMainServiceServer(srv, NewServer(tst.C.OcteliumC))
	go srv.Serve(lis)

	conn, err := grpc.NewClient("passthrough:///enterprise-api",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}))
	require.Nil(t, err)
	t.Cleanup(func() {
		conn.Close()
		srv.Stop()
		lis.Close()
	})
	return enterprisev1.NewMainServiceClient(conn), tst
}

func tstCollectorExporter(name string) *enterprisev1.CollectorExporter {
	return &enterprisev1.CollectorExporter{
		Metadata: &metav1.Metadata{Name: name},
		Spec: &enterprisev1.CollectorExporter_Spec{
			Type: &enterprisev1.CollectorExporter_Spec_Otlp{
				Otlp: &enterprisev1.CollectorExporter_Spec_OTLP{Endpoint: "collector.example.com:4317"},
			},
		},
	}
}

func TestCollectorExporterAPI(t *testing.T) {
	client, tst := newEnterpriseAPI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	name := utilrand.GetRandomStringCanonical(8)
	arg := tstCollectorExporter(name)
	arg.Metadata.Labels = map[string]string{"team": "platform"}
	arg.Metadata.Description = "initial exporter"
	created, err := client.CreateCollectorExporter(ctx, arg)
	require.Nil(t, err)
	assert.NotEmpty(t, created.Metadata.Uid)
	assert.NotNil(t, created.Status)
	assert.Equal(t, "platform", created.Metadata.Labels["team"])
	assert.True(t, proto.Equal(arg.Spec, created.Spec))

	for _, opts := range []*metav1.GetOptions{{Name: name}, {Uid: created.Metadata.Uid}} {
		got, err := client.GetCollectorExporter(ctx, opts)
		require.Nil(t, err)
		assert.True(t, proto.Equal(created, got))
	}

	_, err = client.CreateCollectorExporter(ctx, arg)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	invalid := proto.Clone(created).(*enterprisev1.CollectorExporter)
	invalid.Spec.GetOtlp().Endpoint = "invalid"
	invalid.Metadata.Description = "must not be persisted"
	_, err = client.UpdateCollectorExporter(ctx, invalid)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	stored, err := client.GetCollectorExporter(ctx, &metav1.GetOptions{Uid: created.Metadata.Uid})
	require.Nil(t, err)
	assert.Equal(t, "initial exporter", stored.Metadata.Description)
	assert.Equal(t, "collector.example.com:4317", stored.Spec.GetOtlp().Endpoint)

	updatedArg := proto.Clone(created).(*enterprisev1.CollectorExporter)
	updatedArg.Metadata.Description = "updated exporter"
	updatedArg.Metadata.Labels = map[string]string{"team": "security"}
	updatedArg.Spec = &enterprisev1.CollectorExporter_Spec{
		Type: &enterprisev1.CollectorExporter_Spec_OtlpHTTP{
			OtlpHTTP: &enterprisev1.CollectorExporter_Spec_OTLPHTTP{LogsEndpoint: "https://collector.example.com/v1/logs"},
		},
	}
	updated, err := client.UpdateCollectorExporter(ctx, updatedArg)
	require.Nil(t, err)
	assert.Equal(t, created.Metadata.Uid, updated.Metadata.Uid)
	assert.Equal(t, created.Metadata.CreatedAt.AsTime(), updated.Metadata.CreatedAt.AsTime())
	assert.Equal(t, "security", updated.Metadata.Labels["team"])
	assert.Equal(t, "updated exporter", updated.Metadata.Description)
	assert.True(t, proto.Equal(updatedArg.Spec, updated.Spec))
	stored, err = tst.C.OcteliumC.EnterpriseC().GetCollectorExporter(ctx, &rmetav1.GetOptions{Uid: created.Metadata.Uid})
	require.Nil(t, err)
	assert.True(t, proto.Equal(updated.Spec, stored.Spec))

	listed, err := client.ListCollectorExporter(ctx, &enterprisev1.ListCollectorExporterOptions{})
	require.Nil(t, err)
	assert.True(t, slices.ContainsFunc(listed.Items, func(item *enterprisev1.CollectorExporter) bool { return item.Metadata.Uid == created.Metadata.Uid }))

	_, err = client.DeleteCollectorExporter(ctx, &metav1.DeleteOptions{Uid: created.Metadata.Uid})
	require.Nil(t, err)
	_, err = client.GetCollectorExporter(ctx, &metav1.GetOptions{Name: name})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	_, err = client.UpdateCollectorExporter(ctx, updatedArg)
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	_, err = client.DeleteCollectorExporter(ctx, &metav1.DeleteOptions{Name: name})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

	recreated, err := client.CreateCollectorExporter(ctx, arg)
	require.Nil(t, err)
	assert.NotEqual(t, created.Metadata.Uid, recreated.Metadata.Uid)
	_, err = client.GetCollectorExporter(ctx, &metav1.GetOptions{Uid: created.Metadata.Uid})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
}

func TestCollectorExporterAPIValidation(t *testing.T) {
	client, _ := newEnterpriseAPI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	for _, arg := range []*enterprisev1.CollectorExporter{
		{},
		{Metadata: &metav1.Metadata{Name: "Invalid Name"}, Spec: &enterprisev1.CollectorExporter_Spec{}},
		{Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)}},
		{Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)}, Spec: &enterprisev1.CollectorExporter_Spec{}},
	} {
		_, err := client.CreateCollectorExporter(ctx, arg)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
	for _, opts := range []*metav1.GetOptions{{}, {Name: "Invalid Name"}, {Uid: "bad-uid"}} {
		_, err := client.GetCollectorExporter(ctx, opts)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
	for _, opts := range []*metav1.DeleteOptions{{}, {Name: "Invalid Name"}, {Uid: "bad-uid"}} {
		_, err := client.DeleteCollectorExporter(ctx, opts)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}
	_, err := NewServer(nil).DeleteCollectorExporter(ctx, nil)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	_, err = client.UpdateCollectorExporter(ctx, &enterprisev1.CollectorExporter{})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	arg := tstCollectorExporter(utilrand.GetRandomStringCanonical(8))
	arg.Spec.GetOtlp().Endpoint = "invalid"
	_, err = client.CreateCollectorExporter(ctx, arg)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	_, err = client.GetCollectorExporter(ctx, &metav1.GetOptions{Name: arg.Metadata.Name})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	arg.Spec.GetOtlp().Endpoint = "collector.example.com:4317"
	_, err = client.CreateCollectorExporter(ctx, arg)
	assert.Nil(t, err, "%+v", err)
}

func TestCollectorExporterSystemAndHidden(t *testing.T) {
	client, tst := newEnterpriseAPI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	for _, tt := range []struct {
		name           string
		isSystem       bool
		isSystemHidden bool
		isUserHidden   bool
	}{
		{name: "Visible"},
		{name: "System", isSystem: true},
		{name: "SystemHidden", isSystemHidden: true},
		{name: "SystemAndHidden", isSystem: true, isSystemHidden: true, isUserHidden: true},
		{name: "UserHidden", isUserHidden: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			arg := tstCollectorExporter(utilrand.GetRandomStringCanonical(8))
			arg.Metadata.IsSystem = tt.isSystem
			arg.Metadata.IsUserHidden = tt.isUserHidden
			arg.Metadata.IsSystemHidden = tt.isSystemHidden
			arg.Status = &enterprisev1.CollectorExporter_Status{}
			created, err := tst.C.OcteliumC.EnterpriseC().CreateCollectorExporter(ctx, arg)
			require.Nil(t, err)

			for _, opts := range []*metav1.GetOptions{{Name: created.Metadata.Name}, {Uid: created.Metadata.Uid}} {
				got, err := client.GetCollectorExporter(ctx, opts)
				if tt.isSystemHidden {
					assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)
					assert.Nil(t, got)
					continue
				}
				require.Nil(t, err)
				assert.Equal(t, created.Metadata.Uid, got.Metadata.Uid)
			}

			listed, err := client.ListCollectorExporter(ctx, &enterprisev1.ListCollectorExporterOptions{})
			require.Nil(t, err)
			found := slices.ContainsFunc(listed.Items, func(item *enterprisev1.CollectorExporter) bool { return item.Metadata.Uid == created.Metadata.Uid })
			assert.Equal(t, !tt.isSystemHidden, found)

			if tt.isSystem {
				_, err = client.UpdateCollectorExporter(ctx, created)
				assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
				for _, opts := range []*metav1.DeleteOptions{{Name: created.Metadata.Name}, {Uid: created.Metadata.Uid}} {
					_, err := client.DeleteCollectorExporter(ctx, opts)
					assert.True(t, grpcerr.IsUnauthorized(err), "%+v", err)
				}
				stored, err := tst.C.OcteliumC.EnterpriseC().GetCollectorExporter(ctx, &rmetav1.GetOptions{Uid: created.Metadata.Uid})
				require.Nil(t, err)
				assert.True(t, proto.Equal(created.Spec, stored.Spec))
			}
		})
	}
}

func TestCollectorExporterAuth(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	srv := NewServer(tst.C.OcteliumC)
	secretName := tstSecretName()
	_, err = srv.CreateSecret(t.Context(), tstSecret(secretName, tstSecretValue("credential")))
	require.Nil(t, err)
	t.Run("OTLP", func(t *testing.T) {
		for _, tt := range []struct {
			name    string
			auth    *enterprisev1.CollectorExporter_Spec_OTLP_Auth
			invalid bool
		}{
			{name: "None", auth: nil, invalid: false},
			{name: "Empty", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{}, invalid: true},
			{name: "NilBearer", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_{}}, invalid: true},
			{name: "EmptyBearer", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_FromSecret{FromSecret: ""}}}}, invalid: true},
			{name: "MissingBearer", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_FromSecret{FromSecret: "missing-secret"}}}}, invalid: true},
			{name: "InvalidBearerName", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_FromSecret{FromSecret: "Invalid Secret"}}}}, invalid: true},
			{name: "Bearer", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_FromSecret{FromSecret: secretName}}}}, invalid: false},
			{name: "NilBasic", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_{}}, invalid: true},
			{name: "EmptyUsername", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic{Username: " ", Password: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_Password{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_Password_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "MissingPassword", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic{Username: "user", Password: nil}}}, invalid: true},
			{name: "EmptyPassword", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic{Username: "user", Password: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_Password{}}}}, invalid: true},
			{name: "MissingPasswordSecret", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic{Username: "user", Password: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_Password{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_Password_FromSecret{FromSecret: "missing-secret"}}}}}, invalid: true},
			{name: "Basic", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic{Username: "user", Password: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_Password{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_Password_FromSecret{FromSecret: secretName}}}}}, invalid: false},
			{name: "NilCustom", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_{}}, invalid: true},
			{name: "EmptyHeader", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom{Header: "", Value: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "AuthorizationHeader", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom{Header: "AUTHORIZATION", Value: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "InvalidHeader", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom{Header: "bad:key", Value: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "MissingValue", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom{Header: "X-Token", Value: nil}}}, invalid: true},
			{name: "EmptyValue", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom{Header: "X-Token", Value: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value{}}}}, invalid: true},
			{name: "MissingValueSecret", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom{Header: "X-Token", Value: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value_FromSecret{FromSecret: "missing-secret"}}}}}, invalid: true},
			{name: "Custom", auth: &enterprisev1.CollectorExporter_Spec_OTLP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom{Header: "X-Token", Value: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: false},
		} {
			t.Run(tt.name, func(t *testing.T) {
				err := srv.validateOTLPAuth(t.Context(), tt.auth)
				if tt.invalid {
					assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
					return
				}
				assert.Nil(t, err, "%+v", err)
			})
		}
	})
	t.Run("OTLPHTTP", func(t *testing.T) {
		for _, tt := range []struct {
			name    string
			auth    *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth
			invalid bool
		}{
			{name: "None", auth: nil, invalid: false},
			{name: "Empty", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{}, invalid: true},
			{name: "NilBearer", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_{}}, invalid: true},
			{name: "EmptyBearer", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_FromSecret{FromSecret: ""}}}}, invalid: true},
			{name: "MissingBearer", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_FromSecret{FromSecret: "missing-secret"}}}}, invalid: true},
			{name: "InvalidBearerName", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_FromSecret{FromSecret: "Invalid Secret"}}}}, invalid: true},
			{name: "Bearer", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_FromSecret{FromSecret: secretName}}}}, invalid: false},
			{name: "NilBasic", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_{}}, invalid: true},
			{name: "EmptyUsername", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic{Username: " ", Password: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "MissingPassword", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic{Username: "user", Password: nil}}}, invalid: true},
			{name: "EmptyPassword", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic{Username: "user", Password: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password{}}}}, invalid: true},
			{name: "MissingPasswordSecret", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic{Username: "user", Password: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password_FromSecret{FromSecret: "missing-secret"}}}}}, invalid: true},
			{name: "Basic", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic{Username: "user", Password: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_Password_FromSecret{FromSecret: secretName}}}}}, invalid: false},
			{name: "NilCustom", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_{}}, invalid: true},
			{name: "EmptyHeader", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom{Header: "", Value: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "AuthorizationHeader", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom{Header: "AUTHORIZATION", Value: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "InvalidHeader", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom{Header: "bad:key", Value: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "MissingValue", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom{Header: "X-Token", Value: nil}}}, invalid: true},
			{name: "EmptyValue", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom{Header: "X-Token", Value: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value{}}}}, invalid: true},
			{name: "MissingValueSecret", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom{Header: "X-Token", Value: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value_FromSecret{FromSecret: "missing-secret"}}}}}, invalid: true},
			{name: "Custom", auth: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom{Header: "X-Token", Value: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: false},
		} {
			t.Run(tt.name, func(t *testing.T) {
				err := srv.validateOTLPHTTPAuth(t.Context(), tt.auth)
				if tt.invalid {
					assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
					return
				}
				assert.Nil(t, err, "%+v", err)
			})
		}
	})
	t.Run("PrometheusRemoteWrite", func(t *testing.T) {
		for _, tt := range []struct {
			name    string
			auth    *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth
			invalid bool
		}{
			{name: "None", auth: nil, invalid: false},
			{name: "Empty", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{}, invalid: true},
			{name: "NilBearer", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_{}}, invalid: true},
			{name: "EmptyBearer", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_FromSecret{FromSecret: ""}}}}, invalid: true},
			{name: "MissingBearer", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_FromSecret{FromSecret: "missing-secret"}}}}, invalid: true},
			{name: "InvalidBearerName", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_FromSecret{FromSecret: "Invalid Secret"}}}}, invalid: true},
			{name: "Bearer", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_{Bearer: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_FromSecret{FromSecret: secretName}}}}, invalid: false},
			{name: "NilBasic", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_{}}, invalid: true},
			{name: "EmptyUsername", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic{Username: " ", Password: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_Password{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_Password_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "MissingPassword", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic{Username: "user", Password: nil}}}, invalid: true},
			{name: "EmptyPassword", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic{Username: "user", Password: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_Password{}}}}, invalid: true},
			{name: "MissingPasswordSecret", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic{Username: "user", Password: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_Password{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_Password_FromSecret{FromSecret: "missing-secret"}}}}}, invalid: true},
			{name: "Basic", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_{Basic: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic{Username: "user", Password: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_Password{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_Password_FromSecret{FromSecret: secretName}}}}}, invalid: false},
			{name: "NilCustom", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_{}}, invalid: true},
			{name: "EmptyHeader", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom{Header: "", Value: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "AuthorizationHeader", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom{Header: "AUTHORIZATION", Value: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "InvalidHeader", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom{Header: "bad:key", Value: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: true},
			{name: "MissingValue", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom{Header: "X-Token", Value: nil}}}, invalid: true},
			{name: "EmptyValue", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom{Header: "X-Token", Value: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value{}}}}, invalid: true},
			{name: "MissingValueSecret", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom{Header: "X-Token", Value: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value_FromSecret{FromSecret: "missing-secret"}}}}}, invalid: true},
			{name: "Custom", auth: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_{Custom: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom{Header: "X-Token", Value: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value{Type: &enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_Value_FromSecret{FromSecret: secretName}}}}}, invalid: false},
		} {
			t.Run(tt.name, func(t *testing.T) {
				err := srv.validatePrometheusRemoteWriteAuth(t.Context(), tt.auth)
				if tt.invalid {
					assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
					return
				}
				assert.Nil(t, err, "%+v", err)
			})
		}
	})
}

func TestCollectorExporterEndpoints(t *testing.T) {
	for _, tt := range []struct {
		name     string
		validate func(string) error
		valid    []string
		invalid  []string
	}{
		{
			name:     "HTTP",
			validate: func(val string) error { return validateHTTPURL("endpoint", val) },
			valid:    []string{"http://collector:4318", "https://collector.example.com/v1/logs", "https://[::1]:4318"},
			invalid:  []string{"", "/v1/logs", "collector:4318", "ftp://collector", "http:///path", "https://%invalid"},
		},
		{
			name:     "OTLP",
			validate: validateOTLPGRCPEndpoint,
			valid:    []string{"collector:4317", "https://collector:4317", "http://collector:4317", "dns:///collector:4317", "dns://collector:4317", "[::1]:4317", "unix:///tmp/collector.sock", "collector:1", "collector:65535"},
			invalid:  []string{"", " ", "unix://", "ftp://collector:4317", "collector", ":4317", "collector:abc", "https://collector:4317/path", "collector:0", "collector:-1", "collector:+4317", "collector:65536", "collector:4294967296", "collector:", "https://collector:65536", "http://collector:-1", "dns:///collector:0", "[::1]:65536"},
		},
		{
			name:     "ClickHouse",
			validate: validateClickhouseEndpoint,
			valid:    []string{"tcp://collector:9000", "clickhouse://collector:9000", "http://collector:8123", "https://collector:8443"},
			invalid:  []string{"", "collector:9000", "ftp://collector:9000", "tcp:///path", "tcp://%invalid"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, val := range tt.valid {
				assert.Nil(t, tt.validate(val), "%s", val)
			}
			for _, val := range tt.invalid {
				err := tt.validate(val)
				assert.True(t, grpcerr.IsInvalidArg(err), "%s: %+v", val, err)
			}
		})
	}
}

func TestCollectorExporterDurations(t *testing.T) {
	for _, tt := range []struct {
		name string
		arg  *metav1.Duration
		want time.Duration
	}{
		{name: "Milliseconds", arg: &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 2}}, want: 2 * time.Millisecond},
		{name: "Seconds", arg: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 2}}, want: 2 * time.Second},
		{name: "Minutes", arg: &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 2}}, want: 2 * time.Minute},
		{name: "Hours", arg: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}}, want: 2 * time.Hour},
		{name: "Days", arg: &metav1.Duration{Type: &metav1.Duration_Days{Days: 2}}, want: 48 * time.Hour},
		{name: "Weeks", arg: &metav1.Duration{Type: &metav1.Duration_Weeks{Weeks: 2}}, want: 14 * 24 * time.Hour},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := metav1DurationToTimeDuration(tt.arg)
			require.Nil(t, err)
			assert.Equal(t, tt.want, got)
			assert.Nil(t, validateCollectorDuration("timeout", tt.arg, tt.want, tt.want, false))
			assert.True(t, grpcerr.IsInvalidArg(validateCollectorDuration("timeout", tt.arg, tt.want+time.Millisecond, 365*24*time.Hour, false)))
			assert.True(t, grpcerr.IsInvalidArg(validateCollectorDuration("timeout", tt.arg, 0, tt.want-time.Millisecond, false)))
		})
	}
	for _, arg := range []*metav1.Duration{
		nil,
		{},
		{Type: &metav1.Duration_Milliseconds{}},
		{Type: &metav1.Duration_Seconds{}},
		{Type: &metav1.Duration_Minutes{}},
		{Type: &metav1.Duration_Hours{}},
		{Type: &metav1.Duration_Days{}},
		{Type: &metav1.Duration_Weeks{}},
		{Type: &metav1.Duration_Months{Months: 1}},
	} {
		_, err := metav1DurationToTimeDuration(arg)
		assert.NotNil(t, err, "%+v", arg)
		assert.True(t, grpcerr.IsInvalidArg(validateCollectorDuration("timeout", arg, time.Millisecond, time.Hour, false)))
	}
	assert.Nil(t, validateCollectorDuration("timeout", nil, time.Millisecond, time.Hour, true))
}

func TestCollectorExporterDurationOverflow(t *testing.T) {
	for _, tt := range []struct {
		name string
		unit time.Duration
		make func(uint32) *metav1.Duration
	}{
		{name: "Milliseconds", unit: time.Millisecond, make: func(val uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: val}}
		}},
		{name: "Seconds", unit: time.Second, make: func(val uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: val}}
		}},
		{name: "Minutes", unit: time.Minute, make: func(val uint32) *metav1.Duration {
			return &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: val}}
		}},
		{name: "Hours", unit: time.Hour, make: func(val uint32) *metav1.Duration { return &metav1.Duration{Type: &metav1.Duration_Hours{Hours: val}} }},
		{name: "Days", unit: 24 * time.Hour, make: func(val uint32) *metav1.Duration { return &metav1.Duration{Type: &metav1.Duration_Days{Days: val}} }},
		{name: "Weeks", unit: 7 * 24 * time.Hour, make: func(val uint32) *metav1.Duration { return &metav1.Duration{Type: &metav1.Duration_Weeks{Weeks: val}} }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			limit := min(uint64(math.MaxInt64)/uint64(tt.unit), uint64(math.MaxUint32))
			got, err := metav1DurationToTimeDuration(tt.make(uint32(limit)))
			require.Nil(t, err)
			assert.Equal(t, time.Duration(limit)*tt.unit, got)
			if limit < math.MaxUint32 {
				for _, val := range []uint32{uint32(limit + 1), math.MaxUint32} {
					got, err := metav1DurationToTimeDuration(tt.make(val))
					assert.NotNil(t, err)
					assert.Zero(t, got)
					assert.True(t, grpcerr.IsInvalidArg(validateCollectorDuration("timeout", tt.make(val), time.Millisecond, 10*time.Minute, false)))
				}
			}
		})
	}
	duration := &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 307445735}}
	_, err := metav1DurationToTimeDuration(duration)
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsInvalidArg(validateCollectorDuration("timeout", duration, time.Millisecond, 10*time.Minute, false)))
}

func TestCollectorExporterTransportValidation(t *testing.T) {
	client, _ := newEnterpriseAPI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	created, err := client.CreateCollectorExporter(ctx, tstCollectorExporter(utilrand.GetRandomStringCanonical(8)))
	require.Nil(t, err)
	for _, tt := range []struct {
		name   string
		mutate func(*enterprisev1.CollectorExporter)
	}{
		{name: "ZeroPort", mutate: func(arg *enterprisev1.CollectorExporter) { arg.Spec.GetOtlp().Endpoint = "collector.example.com:0" }},
		{name: "NegativePort", mutate: func(arg *enterprisev1.CollectorExporter) { arg.Spec.GetOtlp().Endpoint = "collector.example.com:-1" }},
		{name: "LargePort", mutate: func(arg *enterprisev1.CollectorExporter) { arg.Spec.GetOtlp().Endpoint = "collector.example.com:65536" }},
		{name: "DurationOverflow", mutate: func(arg *enterprisev1.CollectorExporter) {
			arg.Spec.GetOtlp().Timeout = &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: 307445735}}
		}},
		{name: "HeaderCRLF", mutate: func(arg *enterprisev1.CollectorExporter) {
			arg.Spec.GetOtlp().Headers = []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token", Value: "value\r\nInjected: true"}}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			arg := tstCollectorExporter(utilrand.GetRandomStringCanonical(8))
			tt.mutate(arg)
			_, err := client.CreateCollectorExporter(ctx, arg)
			assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
			_, err = client.GetCollectorExporter(ctx, &metav1.GetOptions{Name: arg.Metadata.Name})
			assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

			arg = proto.Clone(created).(*enterprisev1.CollectorExporter)
			arg.Metadata.Description = "must not persist"
			tt.mutate(arg)
			_, err = client.UpdateCollectorExporter(ctx, arg)
			assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
			got, err := client.GetCollectorExporter(ctx, &metav1.GetOptions{Uid: created.Metadata.Uid})
			require.Nil(t, err)
			assert.True(t, proto.Equal(created.Spec, got.Spec))
			assert.Equal(t, created.Metadata.Description, got.Metadata.Description)
		})
	}
}

func TestCollectorExporterHeaders(t *testing.T) {
	for _, tt := range []struct {
		name        string
		headers     []*enterprisev1.CollectorExporter_Spec_OTLP_Header
		httpHeaders bool
		invalid     bool
	}{
		{name: "Empty"},
		{name: "Valid", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Request-ID", Value: "request"}, {Key: "X-Tenant", Value: "tenant"}}},
		{name: "Limits", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: strings.Repeat("a", 128), Value: strings.Repeat("a", 4096)}}},
		{name: "WhitespaceValue", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token", Value: " value\tvalue "}}, httpHeaders: true},
		{name: "EmptyValue", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token"}}, httpHeaders: true},
		{name: "CarriageReturn", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token", Value: "value\rinjected"}}, httpHeaders: true, invalid: true},
		{name: "LineFeed", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token", Value: "value\ninjected"}}, httpHeaders: true, invalid: true},
		{name: "CRLF", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token", Value: "value\r\nInjected: true"}}, httpHeaders: true, invalid: true},
		{name: "NUL", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token", Value: "value\x00injected"}}, httpHeaders: true, invalid: true},
		{name: "Control", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token", Value: "value\x1finjected"}}, httpHeaders: true, invalid: true},
		{name: "DEL", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token", Value: "value\x7finjected"}}, httpHeaders: true, invalid: true},
		{name: "TooMany", headers: make([]*enterprisev1.CollectorExporter_Spec_OTLP_Header, 65), invalid: true},
		{name: "Nil", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{nil}, invalid: true},
		{name: "EmptyKey", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: " "}}, invalid: true},
		{name: "LongKey", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: strings.Repeat("a", 129)}}, invalid: true},
		{name: "LongValue", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "x-token", Value: strings.Repeat("a", 4097)}}, invalid: true},
		{name: "Duplicate", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Token"}, {Key: "x-token"}}, invalid: true},
		{name: "AuthorizationRejected", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "AUTHORIZATION"}}, httpHeaders: true, invalid: true},
		{name: "AuthorizationAllowed", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "Authorization"}}},
		{name: "RecordValue", headers: []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: "X-Record", Value: "value\x00\r\n\x7f"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeaders("exporter", tt.headers, tt.httpHeaders)
			if tt.invalid {
				assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
				return
			}
			assert.Nil(t, err)
		})
	}
	for _, val := range []string{"bad key", "bad:key", "bad/key", "bad\nkey", "bad\x7fkey", "badékey", "bad(key"} {
		err := validateHeaders("exporter", []*enterprisev1.CollectorExporter_Spec_OTLP_Header{{Key: val}}, false)
		assert.True(t, grpcerr.IsInvalidArg(err), "%s: %+v", val, err)
		assert.True(t, grpcerr.IsInvalidArg(validateCustomAuthHeader("exporter", val)), "%s", val)
	}
	for _, val := range []string{"", " ", "Authorization", "AUTHORIZATION", strings.Repeat("a", 129)} {
		assert.True(t, grpcerr.IsInvalidArg(validateCustomAuthHeader("exporter", val)), "%s", val)
	}
	assert.Nil(t, validateCustomAuthHeader("exporter", "X-Api-Key"))
}

func TestCollectorExporterTLS(t *testing.T) {
	for _, cfg := range []*enterprisev1.CollectorExporter_Spec_OTLP_TLS{
		nil,
		{},
		{Insecure: true},
		{InsecureSkipVerify: true},
		{ServerNameOverride: strings.Repeat("a", 253), CaPEM: strings.Repeat("a", 128*1024)},
	} {
		assert.Nil(t, validateClientTLS("exporter", cfg))
	}
	for _, cfg := range []*enterprisev1.CollectorExporter_Spec_OTLP_TLS{
		{Insecure: true, InsecureSkipVerify: true},
		{ServerNameOverride: strings.Repeat("a", 254)},
		{CaPEM: strings.Repeat("a", 128*1024+1)},
	} {
		assert.True(t, grpcerr.IsInvalidArg(validateClientTLS("exporter", cfg)))
	}
}

func TestCollectorExporterIdentifiers(t *testing.T) {
	for _, tt := range []struct {
		name     string
		validate func(string, string) error
		valid    []string
		invalid  []string
	}{
		{name: "ClickHouse", validate: validateClickhouseIdent, valid: []string{"database_01", strings.Repeat("a", 128)}, invalid: []string{"", " ", "bad-name", "bad.name", "é", strings.Repeat("a", 129)}},
		{name: "ADX", validate: validateADXIdentifier, valid: []string{"table_01", strings.Repeat("a", 128)}, invalid: []string{"", " ", "bad-name", "bad.name", strings.Repeat("a", 129)}},
		{name: "ADXMapping", validate: validateADXMappingName, valid: []string{"map.v1-name_01", strings.Repeat("a", 256)}, invalid: []string{"", " ", "bad/name", "bad name", strings.Repeat("a", 257)}},
		{name: "Elasticsearch", validate: validateElasticsearchIndexLike, valid: []string{"logs-2026.10.25", strings.Repeat("a", 255)}, invalid: []string{"", " ", "bad index", "bad*index", "_index", "+index", "-index", strings.Repeat("a", 256)}},
		{name: "PrometheusPrefix", validate: validatePrometheusMetricPrefix, valid: []string{"", "metrics_01", ":metrics", "_metrics"}, invalid: []string{"1metrics", "bad-metrics", "bad.metrics", "é"}},
		{name: "PrometheusLabel", validate: func(name, val string) error { return validatePrometheusLabelName(name, val, false) }, valid: []string{"label_01", "_label"}, invalid: []string{"", "__reserved", "1label", "bad-label", ":label", "é"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, val := range tt.valid {
				assert.Nil(t, tt.validate("field", val), "%s", val)
			}
			for _, val := range tt.invalid {
				assert.True(t, grpcerr.IsInvalidArg(tt.validate("field", val)), "%s", val)
			}
		})
	}
	assert.Nil(t, validatePrometheusLabelName("field", "__reserved", true))
	assert.Nil(t, validateClickhouseMetricsTables(nil))
	assert.Nil(t, validateClickhouseMetricsTables(&enterprisev1.CollectorExporter_Spec_Clickhouse_MetricsTables{Gauge: "gauge", Sum: "sum", Summary: "summary", Histogram: "histogram", ExponentialHistogram: "exponential"}))
	for _, arg := range []*enterprisev1.CollectorExporter_Spec_Clickhouse_MetricsTables{
		{Gauge: "bad-name"}, {Sum: "bad-name"}, {Summary: "bad-name"}, {Histogram: "bad-name"}, {ExponentialHistogram: "bad-name"},
	} {
		assert.True(t, grpcerr.IsInvalidArg(validateClickhouseMetricsTables(arg)))
	}
}

func TestCollectorExporterKafkaOptions(t *testing.T) {
	assert.Nil(t, validateKafkaSignal("logs", nil, true))
	for _, encoding := range []enterprisev1.CollectorExporter_Spec_Kafka_Encoding{
		enterprisev1.CollectorExporter_Spec_Kafka_ENCODING_UNSET,
		enterprisev1.CollectorExporter_Spec_Kafka_OTLP_PROTO,
		enterprisev1.CollectorExporter_Spec_Kafka_OTLP_JSON,
		enterprisev1.CollectorExporter_Spec_Kafka_RAW,
	} {
		assert.Nil(t, validateKafkaSignal("logs", &enterprisev1.CollectorExporter_Spec_Kafka_Signal{Topic: "logs", Encoding: encoding}, true))
	}
	for _, arg := range []*enterprisev1.CollectorExporter_Spec_Kafka_Signal{
		{Topic: strings.Repeat("a", 250)}, {Topic: "bad\nname"}, {Encoding: 99}, {Encoding: enterprisev1.CollectorExporter_Spec_Kafka_RAW},
	} {
		assert.True(t, grpcerr.IsInvalidArg(validateKafkaSignal("metrics", arg, false)))
	}
	for _, acks := range []int32{-1, 0, 1} {
		assert.Nil(t, validateCollectorExporterKafkaProducer(&enterprisev1.CollectorExporter_Spec_Kafka_Producer{RequiredAcks: acks, MaxMessageBytes: 100 * 1024 * 1024, Linger: tstCollectorDuration(time.Minute)}))
	}
	for _, arg := range []*enterprisev1.CollectorExporter_Spec_Kafka_Producer{
		{MaxMessageBytes: -1}, {MaxMessageBytes: 100*1024*1024 + 1}, {RequiredAcks: 2}, {Compression: 99}, {FlushMaxMessages: -1}, {Linger: tstCollectorDuration(time.Hour)},
	} {
		assert.True(t, grpcerr.IsInvalidArg(validateCollectorExporterKafkaProducer(arg)))
	}
}
