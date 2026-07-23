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
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
)

func (s *Server) CreateCollectorExporter(ctx context.Context, req *enterprisev1.CollectorExporter) (*enterprisev1.CollectorExporter, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.EnterpriseC().GetCollectorExporter(ctx, apivalidation.ObjectToRGetOptions(req))
	if err == nil {
		return nil, serr.InvalidArg("The CollectorExporter %s already exists", req.Metadata.Name)
	}

	if !grpcerr.IsNotFound(err) {
		return nil, serr.K8sInternal(err)
	}

	item := &enterprisev1.CollectorExporter{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &enterprisev1.CollectorExporter_Status{},
	}

	if err := s.validateCollectorExporter(ctx, item); err != nil {
		return nil, err
	}

	item, err = s.octeliumC.EnterpriseC().CreateCollectorExporter(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *Server) GetCollectorExporter(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.CollectorExporter, error) {
	if err := apivalidation.CheckGetOptions(req, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetCollectorExporter(ctx, apivalidation.GetOptionsToRGetOptions(req))
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) ListCollectorExporter(ctx context.Context, req *enterprisev1.ListCollectorExporterOptions) (*enterprisev1.CollectorExporterList, error) {

	itemList, err := s.octeliumC.EnterpriseC().ListCollectorExporter(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) DeleteCollectorExporter(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {

	g, err := s.octeliumC.EnterpriseC().GetCollectorExporter(ctx, apivalidation.DeleteOptionsToRGetOptions(req))
	if err != nil {
		return nil, err
	}

	_, err = s.octeliumC.EnterpriseC().DeleteCollectorExporter(ctx, &rmetav1.DeleteOptions{Uid: g.Metadata.Uid})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *Server) UpdateCollectorExporter(ctx context.Context, req *enterprisev1.CollectorExporter) (*enterprisev1.CollectorExporter, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetCollectorExporter(ctx, apivalidation.ObjectToRGetOptions(req))
	if err != nil {
		return nil, err
	}

	if item.Metadata.IsSystem {
		return nil, serr.InvalidArg("Cannot update the Group %s since it's a system object", item.Metadata.Name)
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	if err := s.validateCollectorExporter(ctx, item); err != nil {
		return nil, err
	}

	item, err = s.octeliumC.EnterpriseC().UpdateCollectorExporter(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *Server) validateCollectorExporter(ctx context.Context, req *enterprisev1.CollectorExporter) error {
	if req == nil {
		return grpcutils.InvalidArg("Nil CollectorExporter")
	}

	spec := req.Spec
	if spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	switch spec.Type.(type) {
	case *enterprisev1.CollectorExporter_Spec_AzureDataExplorer_:
		return s.validateCollectorExporterAzureDataExplorer(ctx, spec.GetAzureDataExplorer())

	case *enterprisev1.CollectorExporter_Spec_AzureMonitor_:
		return s.validateCollectorExporterAzureMonitor(ctx, spec.GetAzureMonitor())

	case *enterprisev1.CollectorExporter_Spec_Clickhouse_:
		return s.validateCollectorExporterClickhouse(ctx, spec.GetClickhouse())

	case *enterprisev1.CollectorExporter_Spec_Datadog_:
		return s.validateCollectorExporterDatadog(ctx, spec.GetDatadog())

	case *enterprisev1.CollectorExporter_Spec_Elasticsearch_:
		return s.validateCollectorExporterElasticsearch(ctx, spec.GetElasticsearch())

	case *enterprisev1.CollectorExporter_Spec_InfluxDB_:
		return s.validateCollectorExporterInfluxDB(ctx, spec.GetInfluxDB())

	case *enterprisev1.CollectorExporter_Spec_Kafka_:
		return s.validateCollectorExporterKafka(ctx, spec.GetKafka())

	case *enterprisev1.CollectorExporter_Spec_Logzio_:
		return s.validateCollectorExporterLogzio(ctx, spec.GetLogzio())

	case *enterprisev1.CollectorExporter_Spec_Otlp:
		return s.validateCollectorExporterOTLP(ctx, spec.GetOtlp())

	case *enterprisev1.CollectorExporter_Spec_OtlpHTTP:
		return s.validateCollectorExporterOTLPHTTP(ctx, spec.GetOtlpHTTP())

	case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_:
		return s.validateCollectorExporterPrometheusRemoteWrite(ctx, spec.GetPrometheusRemoteWrite())

	case *enterprisev1.CollectorExporter_Spec_Splunk_:
		return s.validateCollectorExporterSplunk(ctx, spec.GetSplunk())

	default:
		return grpcutils.InvalidArg("Invalid CollectorExporter type")
	}
}

func (s *Server) validateCollectorExporterOTLP(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_OTLP,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil OTLP exporter spec")
	}

	if strings.TrimSpace(spec.Endpoint) == "" {
		return grpcutils.InvalidArg("OTLP endpoint must be set")
	}

	if err := validateOTLPGRCPEndpoint(spec.Endpoint); err != nil {
		return err
	}

	if err := validateHeaders("OTLP", spec.Headers, true); err != nil {
		return err
	}

	if err := s.validateOTLPAuth(ctx, spec.Auth); err != nil {
		return err
	}

	if err := validateClientTLS("OTLP", spec.Tls); err != nil {
		return err
	}

	switch spec.Compression {
	case enterprisev1.CollectorExporter_Spec_OTLP_COMPRESSION_UNSET,
		enterprisev1.CollectorExporter_Spec_OTLP_GZIP,
		enterprisev1.CollectorExporter_Spec_OTLP_NONE,
		enterprisev1.CollectorExporter_Spec_OTLP_SNAPPY,
		enterprisev1.CollectorExporter_Spec_OTLP_ZSTD:
	default:
		return grpcutils.InvalidArg("Invalid OTLP compression")
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("OTLP timeout", spec.Timeout, time.Millisecond, 10*time.Minute, true); err != nil {
			return err
		}
	}

	if spec.Authority != "" && len(spec.Authority) > 253 {
		return grpcutils.InvalidArg("OTLP authority is too long")
	}

	if spec.UserAgent != "" && len(spec.UserAgent) > 512 {
		return grpcutils.InvalidArg("OTLP userAgent is too long")
	}

	if spec.BalancerName != "" {
		switch spec.BalancerName {
		case "round_robin", "pick_first":
		default:
			return grpcutils.InvalidArg("Unsupported OTLP balancerName")
		}
	}

	if spec.ReadBufferSize < 0 || spec.WriteBufferSize < 0 {
		return grpcutils.InvalidArg("OTLP buffer sizes cannot be negative")
	}
	if spec.ReadBufferSize > 64*1024*1024 || spec.WriteBufferSize > 64*1024*1024 {
		return grpcutils.InvalidArg("OTLP buffer size is too high")
	}

	if spec.Keepalive != nil {
		if spec.Keepalive.Time != nil {
			if err := validateCollectorDuration("OTLP keepalive.time", spec.Keepalive.Time, time.Second, 24*time.Hour, true); err != nil {
				return err
			}
		}
		if spec.Keepalive.Timeout != nil {
			if err := validateCollectorDuration("OTLP keepalive.timeout", spec.Keepalive.Timeout, time.Second, 10*time.Minute, true); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Server) validateCollectorExporterOTLPHTTP(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_OTLPHTTP,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil OTLPHTTP exporter spec")
	}

	if strings.TrimSpace(spec.Endpoint) == "" &&
		strings.TrimSpace(spec.LogsEndpoint) == "" &&
		strings.TrimSpace(spec.MetricsEndpoint) == "" {
		return grpcutils.InvalidArg("OTLPHTTP requires endpoint, logsEndpoint, or metricsEndpoint")
	}

	if spec.Endpoint != "" {
		if err := validateHTTPURL("OTLPHTTP endpoint", spec.Endpoint); err != nil {
			return err
		}
	}
	if spec.LogsEndpoint != "" {
		if err := validateHTTPURL("OTLPHTTP logsEndpoint", spec.LogsEndpoint); err != nil {
			return err
		}
	}
	if spec.MetricsEndpoint != "" {
		if err := validateHTTPURL("OTLPHTTP metricsEndpoint", spec.MetricsEndpoint); err != nil {
			return err
		}
	}

	if err := validateHeaders("OTLPHTTP", spec.Headers, true); err != nil {
		return err
	}

	if err := s.validateOTLPHTTPAuth(ctx, spec.Auth); err != nil {
		return err
	}

	if err := validateClientTLS("OTLPHTTP", spec.Tls); err != nil {
		return err
	}

	switch spec.Encoding {
	case enterprisev1.CollectorExporter_Spec_OTLPHTTP_ENCODING_UNSET,
		enterprisev1.CollectorExporter_Spec_OTLPHTTP_PROTO,
		enterprisev1.CollectorExporter_Spec_OTLPHTTP_JSON:
	default:
		return grpcutils.InvalidArg("Invalid OTLPHTTP encoding")
	}

	switch spec.Compression {
	case enterprisev1.CollectorExporter_Spec_OTLPHTTP_COMPRESSION_UNSET,
		enterprisev1.CollectorExporter_Spec_OTLPHTTP_GZIP,
		enterprisev1.CollectorExporter_Spec_OTLPHTTP_NONE:
	default:
		return grpcutils.InvalidArg("Invalid OTLPHTTP compression")
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("OTLPHTTP timeout", spec.Timeout, time.Millisecond, 10*time.Minute, true); err != nil {
			return err
		}
	}

	if spec.ReadBufferSize < 0 || spec.WriteBufferSize < 0 {
		return grpcutils.InvalidArg("OTLPHTTP buffer sizes cannot be negative")
	}
	if spec.ReadBufferSize > 64*1024*1024 || spec.WriteBufferSize > 64*1024*1024 {
		return grpcutils.InvalidArg("OTLPHTTP buffer size is too high")
	}

	return nil
}

func (s *Server) validateCollectorExporterSplunk(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_Splunk,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil Splunk exporter spec")
	}

	if strings.TrimSpace(spec.Endpoint) == "" {
		return grpcutils.InvalidArg("Splunk endpoint must be set")
	}
	if err := validateHTTPURL("Splunk endpoint", spec.Endpoint); err != nil {
		return err
	}

	if spec.Token == nil || spec.Token.GetFromSecret() == "" {
		return grpcutils.InvalidArg("Splunk token secret must be set")
	}
	if err := s.validateSecretOwner(ctx, spec.Token); err != nil {
		return err
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("Splunk timeout", spec.Timeout, time.Second, 5*time.Minute, true); err != nil {
			return err
		}
	}

	if spec.MaxIdleConns < 0 || spec.MaxIdleConns > 10000 {
		return grpcutils.InvalidArg("Invalid Splunk maxIdleConns")
	}

	if spec.MaxContentLengthLogs < 0 || spec.MaxContentLengthMetrics < 0 {
		return grpcutils.InvalidArg("Splunk max content length cannot be negative")
	}

	if spec.Tls != nil && spec.Tls.InsecureSkipVerify && strings.HasPrefix(strings.ToLower(spec.Endpoint), "http://") {
		return grpcutils.InvalidArg("Splunk tls.insecureSkipVerify is only valid for https endpoints")
	}

	return nil
}

func (s *Server) validateCollectorExporterLogzio(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_Logzio,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil Logzio exporter spec")
	}

	if spec.Token == nil || spec.Token.GetFromSecret() == "" {
		return grpcutils.InvalidArg("Logzio token secret must be set")
	}
	if err := s.validateSecretOwner(ctx, spec.Token); err != nil {
		return err
	}

	if spec.Region != "" {
		region := strings.TrimSpace(spec.Region)
		if region == "" || len(region) > 32 {
			return grpcutils.InvalidArg("Invalid Logzio region")
		}
		for _, r := range region {
			switch {
			case r >= 'a' && r <= 'z':
			case r >= 'A' && r <= 'Z':
			case r >= '0' && r <= '9':
			case r == '-':
			default:
				return grpcutils.InvalidArg("Invalid Logzio region")
			}
		}
	}

	if spec.Endpoint != "" {
		if err := validateHTTPURL("Logzio endpoint", spec.Endpoint); err != nil {
			return err
		}
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("Logzio timeout", spec.Timeout, time.Second, 5*time.Minute, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) validateCollectorExporterElasticsearch(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_Elasticsearch,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil Elasticsearch exporter spec")
	}

	numDestinations := 0
	if strings.TrimSpace(spec.Endpoint) != "" {
		numDestinations++
	}
	if len(spec.Endpoints) > 0 {
		numDestinations++
	}
	if strings.TrimSpace(spec.CloudID) != "" {
		numDestinations++
	}

	if numDestinations != 1 {
		return grpcutils.InvalidArg("Elasticsearch requires exactly one of endpoint, endpoints, or cloudID")
	}

	if spec.Endpoint != "" {
		if err := validateHTTPURL("Elasticsearch endpoint", spec.Endpoint); err != nil {
			return err
		}
	}

	if len(spec.Endpoints) > 0 {
		if len(spec.Endpoints) > 32 {
			return grpcutils.InvalidArg("Elasticsearch has too many endpoints")
		}

		seen := map[string]struct{}{}
		for _, endpoint := range spec.Endpoints {
			if err := validateHTTPURL("Elasticsearch endpoint", endpoint); err != nil {
				return err
			}
			if _, ok := seen[endpoint]; ok {
				return grpcutils.InvalidArg("Duplicate Elasticsearch endpoint: %s", endpoint)
			}
			seen[endpoint] = struct{}{}
		}
	}

	if spec.CloudID != "" && len(spec.CloudID) > 4096 {
		return grpcutils.InvalidArg("Elasticsearch cloudID is too long")
	}

	if err := validateHeaders("Elasticsearch", spec.Headers, true); err != nil {
		return err
	}

	if spec.Auth != nil {
		switch spec.Auth.Type.(type) {
		case *enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_ApiKey:
			if spec.Auth.GetApiKey().GetFromSecret() == "" {
				return grpcutils.InvalidArg("Elasticsearch apiKey secret must be set")
			}
			if err := s.validateSecretOwner(ctx, spec.Auth.GetApiKey()); err != nil {
				return err
			}

		case *enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_Basic_:
			basic := spec.Auth.GetBasic()
			if basic == nil {
				return grpcutils.InvalidArg("Invalid Elasticsearch basic auth")
			}
			if strings.TrimSpace(basic.User) == "" {
				return grpcutils.InvalidArg("Elasticsearch basic auth user must be set")
			}
			if basic.Password == nil || basic.Password.GetFromSecret() == "" {
				return grpcutils.InvalidArg("Elasticsearch basic auth password secret must be set")
			}
			if err := s.validateSecretOwner(ctx, basic.Password); err != nil {
				return err
			}

		default:
			return grpcutils.InvalidArg("Invalid Elasticsearch auth type")
		}
	}

	switch spec.Compression {
	case enterprisev1.CollectorExporter_Spec_Elasticsearch_COMPRESSION_UNSET,
		enterprisev1.CollectorExporter_Spec_Elasticsearch_GZIP,
		enterprisev1.CollectorExporter_Spec_Elasticsearch_NONE:
	default:
		return grpcutils.InvalidArg("Invalid Elasticsearch compression")
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("Elasticsearch timeout", spec.Timeout, time.Second, 10*time.Minute, true); err != nil {
			return err
		}
	}

	if spec.LogsIndex != "" {
		if err := validateElasticsearchIndexLike("Elasticsearch logsIndex", spec.LogsIndex); err != nil {
			return err
		}
	}

	if spec.MetricsIndex != "" {
		if err := validateElasticsearchIndexLike("Elasticsearch metricsIndex", spec.MetricsIndex); err != nil {
			return err
		}
	}

	if spec.Pipeline != "" && len(spec.Pipeline) > 256 {
		return grpcutils.InvalidArg("Elasticsearch pipeline is too long")
	}

	return nil
}

func (s *Server) validateCollectorExporterInfluxDB(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_InfluxDB,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil InfluxDB exporter spec")
	}

	if strings.TrimSpace(spec.Endpoint) == "" {
		return grpcutils.InvalidArg("InfluxDB endpoint must be set")
	}
	if err := validateHTTPURL("InfluxDB endpoint", spec.Endpoint); err != nil {
		return err
	}

	if strings.TrimSpace(spec.Org) == "" {
		return grpcutils.InvalidArg("InfluxDB org must be set")
	}
	if strings.TrimSpace(spec.Bucket) == "" {
		return grpcutils.InvalidArg("InfluxDB bucket must be set")
	}
	if len(spec.Org) > 256 || len(spec.Bucket) > 256 {
		return grpcutils.InvalidArg("InfluxDB org/bucket is too long")
	}

	if spec.Token != nil {
		if spec.Token.GetFromSecret() == "" {
			return grpcutils.InvalidArg("InfluxDB token secret must be set")
		}
		if err := s.validateSecretOwner(ctx, spec.Token); err != nil {
			return err
		}
	}

	if err := validateHeaders("InfluxDB", spec.Headers, true); err != nil {
		return err
	}

	switch spec.MetricsSchema {
	case enterprisev1.CollectorExporter_Spec_InfluxDB_METRICS_SCHEMA_UNSET,
		enterprisev1.CollectorExporter_Spec_InfluxDB_TELEGRAF_PROMETHEUS_V1,
		enterprisev1.CollectorExporter_Spec_InfluxDB_TELEGRAF_PROMETHEUS_V2:
	default:
		return grpcutils.InvalidArg("Invalid InfluxDB metricsSchema")
	}

	switch spec.Precision {
	case enterprisev1.CollectorExporter_Spec_InfluxDB_PRECISION_UNSET,
		enterprisev1.CollectorExporter_Spec_InfluxDB_NS,
		enterprisev1.CollectorExporter_Spec_InfluxDB_US,
		enterprisev1.CollectorExporter_Spec_InfluxDB_MS,
		enterprisev1.CollectorExporter_Spec_InfluxDB_S:
	default:
		return grpcutils.InvalidArg("Invalid InfluxDB precision")
	}

	if spec.PayloadMaxLines < 0 || spec.PayloadMaxBytes < 0 {
		return grpcutils.InvalidArg("InfluxDB payload limits cannot be negative")
	}
	if spec.PayloadMaxLines > 1_000_000 {
		return grpcutils.InvalidArg("InfluxDB payloadMaxLines is too high")
	}
	if spec.PayloadMaxBytes > 256*1024*1024 {
		return grpcutils.InvalidArg("InfluxDB payloadMaxBytes is too high")
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("InfluxDB timeout", spec.Timeout, time.Millisecond, 10*time.Minute, true); err != nil {
			return err
		}
	}

	if len(spec.LogRecordDimensions) > 64 {
		return grpcutils.InvalidArg("InfluxDB has too many logRecordDimensions")
	}

	seenDims := map[string]struct{}{}
	for _, dim := range spec.LogRecordDimensions {
		dim = strings.TrimSpace(dim)
		if dim == "" {
			return grpcutils.InvalidArg("InfluxDB logRecordDimension cannot be empty")
		}
		if len(dim) > 256 {
			return grpcutils.InvalidArg("InfluxDB logRecordDimension is too long")
		}
		if _, ok := seenDims[dim]; ok {
			return grpcutils.InvalidArg("Duplicate InfluxDB logRecordDimension: %s", dim)
		}
		seenDims[dim] = struct{}{}
	}

	if spec.V1Compatibility != nil {
		if spec.V1Compatibility.Enabled && strings.TrimSpace(spec.V1Compatibility.Db) == "" {
			return grpcutils.InvalidArg("InfluxDB v1Compatibility.db must be set when v1Compatibility is enabled")
		}
		if len(spec.V1Compatibility.Db) > 256 {
			return grpcutils.InvalidArg("InfluxDB v1Compatibility.db is too long")
		}
		if len(spec.V1Compatibility.Username) > 256 {
			return grpcutils.InvalidArg("InfluxDB v1Compatibility.username is too long")
		}
		if spec.V1Compatibility.Password != nil {
			if spec.V1Compatibility.Password.GetFromSecret() == "" {
				return grpcutils.InvalidArg("InfluxDB v1Compatibility password secret must be set")
			}
			if err := s.validateSecretOwner(ctx, spec.V1Compatibility.Password); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Server) validateCollectorExporterKafka(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_Kafka,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil Kafka exporter spec")
	}

	if len(spec.Brokers) > 64 {
		return grpcutils.InvalidArg("Kafka has too many brokers")
	}

	seenBrokers := map[string]struct{}{}
	for _, broker := range spec.Brokers {
		broker = strings.TrimSpace(broker)
		if broker == "" {
			return grpcutils.InvalidArg("Kafka broker cannot be empty")
		}
		if len(broker) > 255 {
			return grpcutils.InvalidArg("Kafka broker is too long")
		}
		if _, ok := seenBrokers[broker]; ok {
			return grpcutils.InvalidArg("Duplicate Kafka broker: %s", broker)
		}
		seenBrokers[broker] = struct{}{}
	}

	if spec.ProtocolVersion != "" && len(spec.ProtocolVersion) > 32 {
		return grpcutils.InvalidArg("Kafka protocolVersion is too long")
	}

	if spec.ClientID != "" && len(spec.ClientID) > 128 {
		return grpcutils.InvalidArg("Kafka clientID is too long")
	}

	if spec.Logs != nil {
		if err := validateKafkaSignal("Kafka logs", spec.Logs, true); err != nil {
			return err
		}
	}
	if spec.Metrics != nil {
		if err := validateKafkaSignal("Kafka metrics", spec.Metrics, false); err != nil {
			return err
		}
	}

	if err := validateHeaders("Kafka record", spec.RecordHeaders, false); err != nil {
		return err
	}

	if spec.Auth != nil {
		switch spec.Auth.Type.(type) {
		case *enterprisev1.CollectorExporter_Spec_Kafka_Auth_Sasl:
			sasl := spec.Auth.GetSasl()
			if sasl == nil {
				return grpcutils.InvalidArg("Invalid Kafka SASL auth")
			}
			if strings.TrimSpace(sasl.Username) == "" {
				return grpcutils.InvalidArg("Kafka SASL username must be set")
			}
			if sasl.Password == nil || sasl.Password.GetFromSecret() == "" {
				return grpcutils.InvalidArg("Kafka SASL password secret must be set")
			}
			if err := s.validateSecretOwner(ctx, sasl.Password); err != nil {
				return err
			}

			switch sasl.Mechanism {
			case enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_MECHANISM_UNSET,
				enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_PLAIN,
				enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_SCRAM_SHA_256,
				enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_SCRAM_SHA_512:
			default:
				return grpcutils.InvalidArg("Invalid Kafka SASL mechanism")
			}

		default:
			return grpcutils.InvalidArg("Invalid Kafka auth type")
		}
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("Kafka timeout", spec.Timeout, time.Second, 5*time.Minute, true); err != nil {
			return err
		}
	}
	if spec.ConnIdleTimeout != nil {
		if err := validateCollectorDuration("Kafka connIdleTimeout", spec.ConnIdleTimeout, time.Second, 2*time.Hour, true); err != nil {
			return err
		}
	}

	if spec.Producer != nil {
		if err := validateCollectorExporterKafkaProducer(spec.Producer); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) validateCollectorExporterDatadog(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_Datadog,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil Datadog exporter spec")
	}

	if spec.Api == nil {
		return grpcutils.InvalidArg("Datadog api must be set")
	}

	if spec.Api.Key == nil || spec.Api.Key.GetFromSecret() == "" {
		return grpcutils.InvalidArg("Datadog api.key secret must be set")
	}
	if err := s.validateSecretOwner(ctx, spec.Api.Key); err != nil {
		return err
	}

	if spec.Api.Site != "" {
		if len(spec.Api.Site) > 253 || strings.ContainsAny(spec.Api.Site, "/: \t\r\n") {
			return grpcutils.InvalidArg("Invalid Datadog api.site")
		}
	}

	if spec.Hostname != "" {
		if len(spec.Hostname) > 255 || strings.ContainsAny(spec.Hostname, "\x00\r\n") {
			return grpcutils.InvalidArg("Invalid Datadog hostname")
		}
	}

	if spec.Metrics != nil && spec.Metrics.Endpoint != "" {
		if err := validateHTTPURL("Datadog metrics endpoint", spec.Metrics.Endpoint); err != nil {
			return err
		}
	}

	if spec.Logs != nil {
		if spec.Logs.Endpoint != "" {
			if err := validateHTTPURL("Datadog logs endpoint", spec.Logs.Endpoint); err != nil {
				return err
			}
		}
		if spec.Logs.CompressionLevel < 0 || spec.Logs.CompressionLevel > 9 {
			return grpcutils.InvalidArg("Datadog logs compressionLevel must be between 0 and 9")
		}
		if spec.Logs.BatchWait != nil {
			if err := validateCollectorDuration("Datadog logs batchWait", spec.Logs.BatchWait, time.Millisecond, 5*time.Minute, true); err != nil {
				return err
			}
		}
	}

	if spec.HostMetadata != nil && spec.HostMetadata.ReporterPeriod != nil {
		if err := validateCollectorDuration("Datadog hostMetadata reporterPeriod", spec.HostMetadata.ReporterPeriod, 5*time.Minute, 24*time.Hour, true); err != nil {
			return err
		}
	}

	if spec.HostnameDetectionTimeout != nil {
		if err := validateCollectorDuration("Datadog hostnameDetectionTimeout", spec.HostnameDetectionTimeout, time.Second, 5*time.Minute, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) validateCollectorExporterAzureMonitor(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_AzureMonitor,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil AzureMonitor exporter spec")
	}

	hasConnectionString := spec.ConnectionString != nil && spec.ConnectionString.GetFromSecret() != ""
	hasInstrumentationKey := spec.InstrumentationKey != nil && spec.InstrumentationKey.GetFromSecret() != ""

	if !hasConnectionString && !hasInstrumentationKey {
		return grpcutils.InvalidArg("AzureMonitor requires either connectionString or instrumentationKey")
	}

	if hasConnectionString && hasInstrumentationKey {
		return grpcutils.InvalidArg("AzureMonitor must not set both connectionString and instrumentationKey")
	}

	if hasConnectionString {
		if err := s.validateSecretOwner(ctx, spec.ConnectionString); err != nil {
			return err
		}
	}
	if hasInstrumentationKey {
		if err := s.validateSecretOwner(ctx, spec.InstrumentationKey); err != nil {
			return err
		}
	}

	if spec.Endpoint != "" {
		if err := validateHTTPURL("AzureMonitor endpoint", spec.Endpoint); err != nil {
			return err
		}
	}

	if spec.MaxBatchSize < 0 || spec.MaxBatchSize > 100000 {
		return grpcutils.InvalidArg("Invalid AzureMonitor maxBatchSize")
	}

	if spec.MaxBatchInterval != nil {
		if err := validateCollectorDuration("AzureMonitor maxBatchInterval", spec.MaxBatchInterval, time.Second, 10*time.Minute, true); err != nil {
			return err
		}
	}

	if spec.ShutdownTimeout != nil {
		if err := validateCollectorDuration("AzureMonitor shutdownTimeout", spec.ShutdownTimeout, time.Second, 5*time.Minute, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) validateCollectorExporterAzureDataExplorer(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_AzureDataExplorer,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil AzureDataExplorer exporter spec")
	}

	if strings.TrimSpace(spec.ClusterURI) == "" {
		return grpcutils.InvalidArg("AzureDataExplorer clusterURI must be set")
	}

	u, err := url.Parse(spec.ClusterURI)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return grpcutils.InvalidArg("Invalid AzureDataExplorer clusterURI")
	}
	if u.Scheme != "https" {
		return grpcutils.InvalidArg("AzureDataExplorer clusterURI must use https")
	}

	if spec.Auth == nil {
		return grpcutils.InvalidArg("AzureDataExplorer auth must be set")
	}

	switch spec.Auth.Type.(type) {
	case *enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth_ServicePrincipal:
		sp := spec.Auth.GetServicePrincipal()
		if sp == nil {
			return grpcutils.InvalidArg("Invalid AzureDataExplorer servicePrincipal auth")
		}

		if _, err := uuid.Parse(strings.TrimSpace(sp.ApplicationID)); err != nil {
			return grpcutils.InvalidArg("AzureDataExplorer servicePrincipal.applicationID must be a UUID")
		}

		if sp.ApplicationKey == nil || sp.ApplicationKey.GetFromSecret() == "" {
			return grpcutils.InvalidArg("AzureDataExplorer servicePrincipal.applicationKey secret must be set")
		}
		if err := s.validateSecretOwner(ctx, sp.ApplicationKey); err != nil {
			return err
		}

		if _, err := uuid.Parse(strings.TrimSpace(sp.TenantID)); err != nil {
			return grpcutils.InvalidArg("AzureDataExplorer servicePrincipal.tenantID must be a UUID")
		}

	case *enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth_ManagedIdentity:
		mi := spec.Auth.GetManagedIdentity()
		if mi == nil {
			return grpcutils.InvalidArg("Invalid AzureDataExplorer managedIdentity auth")
		}

		id := strings.TrimSpace(mi.Id)
		if id == "" {
			return grpcutils.InvalidArg("AzureDataExplorer managedIdentity.id must be set")
		}
		if !strings.EqualFold(id, "system") {
			if _, err := uuid.Parse(id); err != nil {
				return grpcutils.InvalidArg("AzureDataExplorer managedIdentity.id must be \"system\" or a UUID")
			}
		}

	case *enterprisev1.CollectorExporter_Spec_AzureDataExplorer_Auth_AzureDefault:
	default:
		return grpcutils.InvalidArg("Invalid AzureDataExplorer auth type")
	}

	if spec.Database != "" {
		if err := validateADXIdentifier("AzureDataExplorer database", spec.Database); err != nil {
			return err
		}
	}
	if spec.MetricsTable != "" {
		if err := validateADXIdentifier("AzureDataExplorer metricsTable", spec.MetricsTable); err != nil {
			return err
		}
	}
	if spec.LogsTable != "" {
		if err := validateADXIdentifier("AzureDataExplorer logsTable", spec.LogsTable); err != nil {
			return err
		}
	}
	if spec.MetricsTableMapping != "" {
		if err := validateADXMappingName("AzureDataExplorer metricsTableMapping", spec.MetricsTableMapping); err != nil {
			return err
		}
	}
	if spec.LogsTableMapping != "" {
		if err := validateADXMappingName("AzureDataExplorer logsTableMapping", spec.LogsTableMapping); err != nil {
			return err
		}
	}

	switch spec.IngestionType {
	case enterprisev1.CollectorExporter_Spec_AzureDataExplorer_INGESTION_TYPE_UNSET,
		enterprisev1.CollectorExporter_Spec_AzureDataExplorer_QUEUED,
		enterprisev1.CollectorExporter_Spec_AzureDataExplorer_MANAGED:
	default:
		return grpcutils.InvalidArg("Invalid AzureDataExplorer ingestionType")
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("AzureDataExplorer timeout", spec.Timeout, time.Second, 10*time.Minute, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) validateCollectorExporterClickhouse(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_Clickhouse,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil ClickHouse exporter spec")
	}

	if strings.TrimSpace(spec.Endpoint) == "" {
		return grpcutils.InvalidArg("ClickHouse endpoint must be set")
	}

	if err := validateClickhouseEndpoint(spec.Endpoint); err != nil {
		return err
	}

	if spec.Username != "" && len(spec.Username) > 256 {
		return grpcutils.InvalidArg("ClickHouse username is too long")
	}

	if spec.Password != nil {
		if spec.Password.GetFromSecret() == "" {
			return grpcutils.InvalidArg("ClickHouse password secret must be set")
		}
		if err := s.validateSecretOwner(ctx, spec.Password); err != nil {
			return err
		}
	}

	if spec.Database != "" {
		if err := validateClickhouseIdent("ClickHouse database", spec.Database); err != nil {
			return err
		}
	}

	if err := validateClientTLS("ClickHouse", spec.Tls); err != nil {
		return err
	}

	if len(spec.ConnectionParams) > 64 {
		return grpcutils.InvalidArg("ClickHouse has too many connectionParams")
	}

	seenParams := map[string]struct{}{}
	for _, p := range spec.ConnectionParams {
		if p == nil {
			return grpcutils.InvalidArg("ClickHouse connectionParam cannot be nil")
		}

		key := strings.TrimSpace(p.Key)
		if key == "" {
			return grpcutils.InvalidArg("ClickHouse connectionParam key cannot be empty")
		}
		if len(key) > 128 || len(p.Value) > 4096 {
			return grpcutils.InvalidArg("ClickHouse connectionParam is too large")
		}

		keyLower := strings.ToLower(key)
		if _, ok := seenParams[keyLower]; ok {
			return grpcutils.InvalidArg("Duplicate ClickHouse connectionParam: %s", key)
		}
		seenParams[keyLower] = struct{}{}

		switch keyLower {
		case "username", "password":
			return grpcutils.InvalidArg("ClickHouse username/password must be configured through dedicated fields")
		}
	}

	if spec.LogsTableName != "" {
		if err := validateClickhouseIdent("ClickHouse logsTableName", spec.LogsTableName); err != nil {
			return err
		}
	}

	if spec.MetricsTables != nil {
		if err := validateClickhouseMetricsTables(spec.MetricsTables); err != nil {
			return err
		}
	}

	if spec.Ttl != nil {
		if err := validateCollectorDuration("ClickHouse ttl", spec.Ttl, time.Second, 365*24*time.Hour, true); err != nil {
			return err
		}
	}

	switch spec.Compression {
	case enterprisev1.CollectorExporter_Spec_Clickhouse_COMPRESSION_UNSET,
		enterprisev1.CollectorExporter_Spec_Clickhouse_LZ4,
		enterprisev1.CollectorExporter_Spec_Clickhouse_NONE,
		enterprisev1.CollectorExporter_Spec_Clickhouse_ZSTD,
		enterprisev1.CollectorExporter_Spec_Clickhouse_GZIP,
		enterprisev1.CollectorExporter_Spec_Clickhouse_DEFLATE,
		enterprisev1.CollectorExporter_Spec_Clickhouse_BR:
	default:
		return grpcutils.InvalidArg("Invalid ClickHouse compression")
	}

	if spec.ClusterName != "" {
		if err := validateClickhouseIdent("ClickHouse clusterName", spec.ClusterName); err != nil {
			return err
		}
	}

	if spec.TableEngine != nil {
		if spec.TableEngine.Name != "" {
			if err := validateClickhouseIdent("ClickHouse tableEngine.name", spec.TableEngine.Name); err != nil {
				return err
			}
		}
		if len(spec.TableEngine.Params) > 4096 {
			return grpcutils.InvalidArg("ClickHouse tableEngine.params is too long")
		}
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("ClickHouse timeout", spec.Timeout, time.Millisecond, 10*time.Minute, true); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) validateCollectorExporterPrometheusRemoteWrite(
	ctx context.Context,
	spec *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite,
) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil PrometheusRemoteWrite exporter spec")
	}

	if strings.TrimSpace(spec.Endpoint) == "" {
		return grpcutils.InvalidArg("PrometheusRemoteWrite endpoint must be set")
	}

	if err := validateHTTPURL("PrometheusRemoteWrite endpoint", spec.Endpoint); err != nil {
		return err
	}

	if spec.Namespace != "" {
		if len(spec.Namespace) > 128 {
			return grpcutils.InvalidArg("PrometheusRemoteWrite namespace is too long")
		}
		if err := validatePrometheusMetricPrefix("PrometheusRemoteWrite namespace", spec.Namespace); err != nil {
			return err
		}
	}

	if err := validateHeaders("PrometheusRemoteWrite", spec.Headers, true); err != nil {
		return err
	}

	if err := s.validatePrometheusRemoteWriteAuth(ctx, spec.Auth); err != nil {
		return err
	}

	if err := validateClientTLS("PrometheusRemoteWrite", spec.Tls); err != nil {
		return err
	}

	if len(spec.ExternalLabels) > 64 {
		return grpcutils.InvalidArg("PrometheusRemoteWrite has too many externalLabels")
	}

	seenLabels := map[string]struct{}{}
	for _, l := range spec.ExternalLabels {
		if l == nil {
			return grpcutils.InvalidArg("PrometheusRemoteWrite externalLabel cannot be nil")
		}

		key := strings.TrimSpace(l.Key)
		if key == "" {
			return grpcutils.InvalidArg("PrometheusRemoteWrite externalLabel key cannot be empty")
		}

		if len(key) > 128 || len(l.Value) > 4096 {
			return grpcutils.InvalidArg("PrometheusRemoteWrite externalLabel is too large")
		}

		if err := validatePrometheusLabelName("PrometheusRemoteWrite externalLabel", key, true); err != nil {
			return err
		}

		if _, ok := seenLabels[key]; ok {
			return grpcutils.InvalidArg("Duplicate PrometheusRemoteWrite externalLabel: %s", key)
		}
		seenLabels[key] = struct{}{}
	}

	if spec.Timeout != nil {
		if err := validateCollectorDuration("PrometheusRemoteWrite timeout", spec.Timeout, time.Millisecond, 10*time.Minute, true); err != nil {
			return err
		}
	}

	if spec.RemoteWriteQueue != nil {
		if spec.RemoteWriteQueue.QueueSize < 0 {
			return grpcutils.InvalidArg("PrometheusRemoteWrite remoteWriteQueue.queueSize cannot be negative")
		}
		if spec.RemoteWriteQueue.Enabled && spec.RemoteWriteQueue.QueueSize == 0 {
			return grpcutils.InvalidArg("PrometheusRemoteWrite remoteWriteQueue.queueSize cannot be zero when enabled")
		}
		if spec.RemoteWriteQueue.QueueSize > 10_000_000 {
			return grpcutils.InvalidArg("PrometheusRemoteWrite remoteWriteQueue.queueSize is too high")
		}
		if spec.RemoteWriteQueue.NumConsumers < 0 || spec.RemoteWriteQueue.NumConsumers > 1024 {
			return grpcutils.InvalidArg("Invalid PrometheusRemoteWrite remoteWriteQueue.numConsumers")
		}
	}

	if spec.MaxBatchSizeBytes < 0 || spec.MaxBatchSizeBytes > 512*1024*1024 {
		return grpcutils.InvalidArg("Invalid PrometheusRemoteWrite maxBatchSizeBytes")
	}

	if spec.MaxBatchRequestParallelism < 0 || spec.MaxBatchRequestParallelism > 1024 {
		return grpcutils.InvalidArg("Invalid PrometheusRemoteWrite maxBatchRequestParallelism")
	}

	switch spec.TranslationStrategy {
	case enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_TRANSLATION_STRATEGY_UNSET,
		enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_UNDERSCORE_ESCAPING_WITH_SUFFIXES,
		enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_UNDERSCORE_ESCAPING_WITHOUT_SUFFIXES:
		return nil

	case enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_NO_UTF8_ESCAPING_WITH_SUFFIXES,
		enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_NO_TRANSLATION:
		return grpcutils.InvalidArg("PrometheusRemoteWrite UTF-8/no-translation strategies require Remote Write 2.0 and are not supported yet")

	default:
		return grpcutils.InvalidArg("Invalid PrometheusRemoteWrite translationStrategy")
	}
}

func validateHTTPURL(name, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return grpcutils.InvalidArg("Invalid %s", name)
	}

	switch u.Scheme {
	case "http", "https":
		return nil
	default:
		return grpcutils.InvalidArg("%s must use http or https", name)
	}
}

func validateOTLPGRCPEndpoint(endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return grpcutils.InvalidArg("OTLP endpoint must be set")
	}

	if strings.HasPrefix(endpoint, "unix://") {
		if strings.TrimPrefix(endpoint, "unix://") == "" {
			return grpcutils.InvalidArg("OTLP unix socket path cannot be empty")
		}
		return nil
	}

	sanitized := endpoint

	switch {
	case strings.HasPrefix(endpoint, "http://"):
		sanitized = strings.TrimPrefix(endpoint, "http://")
	case strings.HasPrefix(endpoint, "https://"):
		sanitized = strings.TrimPrefix(endpoint, "https://")
	case strings.HasPrefix(endpoint, "dns://"):
		sanitized = strings.TrimPrefix(endpoint, "dns://")
		sanitized = strings.TrimPrefix(sanitized, "/")
	case strings.Contains(endpoint, "://"):
		return grpcutils.InvalidArg("Unsupported OTLP endpoint scheme")
	}

	host, port, err := net.SplitHostPort(sanitized)
	if err != nil {
		return grpcutils.InvalidArg("OTLP endpoint must be in host:port form")
	}
	if strings.TrimSpace(host) == "" {
		return grpcutils.InvalidArg("OTLP endpoint host cannot be empty")
	}
	if _, err := strconv.Atoi(port); err != nil {
		return grpcutils.InvalidArg("Invalid OTLP endpoint port")
	}

	return nil
}

func validateCollectorDuration(
	name string,
	d *metav1.Duration,
	min time.Duration,
	max time.Duration,
	allowNil bool,
) error {
	if d == nil {
		if allowNil {
			return nil
		}
		return grpcutils.InvalidArg("%s must be set", name)
	}

	td, err := metav1DurationToTimeDuration(d)
	if err != nil {
		return grpcutils.InvalidArg("Invalid %s: %+v", name, err)
	}

	if td < min || td > max {
		return grpcutils.InvalidArg("%s must be between %s and %s", name, min, max)
	}

	return nil
}

func metav1DurationToTimeDuration(d *metav1.Duration) (time.Duration, error) {
	if d == nil {
		return 0, errors.Errorf("nil duration")
	}

	switch d.Type.(type) {
	case *metav1.Duration_Milliseconds:
		if d.GetMilliseconds() == 0 {
			return 0, errors.Errorf("milliseconds cannot be zero")
		}
		return time.Duration(d.GetMilliseconds()) * time.Millisecond, nil

	case *metav1.Duration_Seconds:
		if d.GetSeconds() == 0 {
			return 0, errors.Errorf("seconds cannot be zero")
		}
		return time.Duration(d.GetSeconds()) * time.Second, nil

	case *metav1.Duration_Minutes:
		if d.GetMinutes() == 0 {
			return 0, errors.Errorf("minutes cannot be zero")
		}
		return time.Duration(d.GetMinutes()) * time.Minute, nil

	case *metav1.Duration_Hours:
		if d.GetHours() == 0 {
			return 0, errors.Errorf("hours cannot be zero")
		}
		return time.Duration(d.GetHours()) * time.Hour, nil

	case *metav1.Duration_Days:
		if d.GetDays() == 0 {
			return 0, errors.Errorf("days cannot be zero")
		}
		return time.Duration(d.GetDays()) * 24 * time.Hour, nil

	case *metav1.Duration_Weeks:
		if d.GetWeeks() == 0 {
			return 0, errors.Errorf("weeks cannot be zero")
		}
		return time.Duration(d.GetWeeks()) * 7 * 24 * time.Hour, nil

	case *metav1.Duration_Months:
		return 0, errors.Errorf("months are not supported for fixed-duration collector fields")

	default:
		return 0, errors.Errorf("invalid duration type")
	}
}

type headerLike interface {
	GetKey() string
	GetValue() string
}

func validateHeaders[T headerLike](name string, headers []T, rejectAuthorization bool) error {
	if len(headers) > 64 {
		return grpcutils.InvalidArg("%s has too many headers", name)
	}

	seen := map[string]struct{}{}

	for _, h := range headers {
		if any(h) == nil {
			return grpcutils.InvalidArg("%s header cannot be nil", name)
		}

		key := strings.TrimSpace(h.GetKey())
		if key == "" {
			return grpcutils.InvalidArg("%s header key cannot be empty", name)
		}

		if len(key) > 128 {
			return grpcutils.InvalidArg("%s header key is too long", name)
		}

		if len(h.GetValue()) > 4096 {
			return grpcutils.InvalidArg("%s header value is too long", name)
		}

		for _, r := range key {
			if r <= 32 || r >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={}", r) {
				return grpcutils.InvalidArg("%s header key contains invalid character", name)
			}
		}

		keyLower := strings.ToLower(key)
		if _, ok := seen[keyLower]; ok {
			return grpcutils.InvalidArg("Duplicate %s header: %s", name, key)
		}
		seen[keyLower] = struct{}{}

		if rejectAuthorization && keyLower == "authorization" {
			return grpcutils.InvalidArg("%s Authorization header must be configured through auth", name)
		}
	}

	return nil
}

type clientTLSLike interface {
	GetInsecure() bool
	GetInsecureSkipVerify() bool
	GetServerNameOverride() string
	GetCaPEM() string
}

func validateClientTLS[T clientTLSLike](name string, tlsCfg T) error {
	if any(tlsCfg) == nil {
		return nil
	}

	if tlsCfg.GetInsecure() && tlsCfg.GetInsecureSkipVerify() {
		return grpcutils.InvalidArg("%s tls.insecure and tls.insecureSkipVerify cannot both be true", name)
	}

	if len(tlsCfg.GetServerNameOverride()) > 253 {
		return grpcutils.InvalidArg("%s tls.serverNameOverride is too long", name)
	}

	if len(tlsCfg.GetCaPEM()) > 128*1024 {
		return grpcutils.InvalidArg("%s tls.caPEM is too large", name)
	}

	return nil
}

func (s *Server) validateOTLPAuth(
	ctx context.Context,
	auth *enterprisev1.CollectorExporter_Spec_OTLP_Auth,
) error {
	if auth == nil {
		return nil
	}

	switch auth.Type.(type) {
	case *enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_:
		if auth.GetBearer().GetFromSecret() == "" {
			return grpcutils.InvalidArg("OTLP bearer token secret must be set")
		}
		return s.validateSecretOwner(ctx, auth.GetBearer())

	case *enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_:
		basic := auth.GetBasic()
		if basic == nil {
			return grpcutils.InvalidArg("Invalid OTLP basic auth")
		}
		if strings.TrimSpace(basic.Username) == "" {
			return grpcutils.InvalidArg("OTLP basic auth username must be set")
		}
		if basic.Password == nil || basic.Password.GetFromSecret() == "" {
			return grpcutils.InvalidArg("OTLP basic auth password secret must be set")
		}
		return s.validateSecretOwner(ctx, basic.Password)

	case *enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_:
		custom := auth.GetCustom()
		if custom == nil {
			return grpcutils.InvalidArg("Invalid OTLP custom auth")
		}
		if err := validateCustomAuthHeader("OTLP", custom.Header); err != nil {
			return err
		}
		if custom.Value == nil || custom.Value.GetFromSecret() == "" {
			return grpcutils.InvalidArg("OTLP custom auth value secret must be set")
		}
		return s.validateSecretOwner(ctx, custom.Value)

	default:
		return grpcutils.InvalidArg("Invalid OTLP auth type")
	}
}

func (s *Server) validateOTLPHTTPAuth(
	ctx context.Context,
	auth *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth,
) error {
	if auth == nil {
		return nil
	}

	switch auth.Type.(type) {
	case *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_:
		if auth.GetBearer().GetFromSecret() == "" {
			return grpcutils.InvalidArg("OTLPHTTP bearer token secret must be set")
		}
		return s.validateSecretOwner(ctx, auth.GetBearer())

	case *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_:
		basic := auth.GetBasic()
		if basic == nil {
			return grpcutils.InvalidArg("Invalid OTLPHTTP basic auth")
		}
		if strings.TrimSpace(basic.Username) == "" {
			return grpcutils.InvalidArg("OTLPHTTP basic auth username must be set")
		}
		if basic.Password == nil || basic.Password.GetFromSecret() == "" {
			return grpcutils.InvalidArg("OTLPHTTP basic auth password secret must be set")
		}
		return s.validateSecretOwner(ctx, basic.Password)

	case *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_:
		custom := auth.GetCustom()
		if custom == nil {
			return grpcutils.InvalidArg("Invalid OTLPHTTP custom auth")
		}
		if err := validateCustomAuthHeader("OTLPHTTP", custom.Header); err != nil {
			return err
		}
		if custom.Value == nil || custom.Value.GetFromSecret() == "" {
			return grpcutils.InvalidArg("OTLPHTTP custom auth value secret must be set")
		}
		return s.validateSecretOwner(ctx, custom.Value)

	default:
		return grpcutils.InvalidArg("Invalid OTLPHTTP auth type")
	}
}

func (s *Server) validatePrometheusRemoteWriteAuth(
	ctx context.Context,
	auth *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth,
) error {
	if auth == nil {
		return nil
	}

	switch auth.Type.(type) {
	case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_:
		if auth.GetBearer().GetFromSecret() == "" {
			return grpcutils.InvalidArg("PrometheusRemoteWrite bearer token secret must be set")
		}
		return s.validateSecretOwner(ctx, auth.GetBearer())

	case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_:
		basic := auth.GetBasic()
		if basic == nil {
			return grpcutils.InvalidArg("Invalid PrometheusRemoteWrite basic auth")
		}
		if strings.TrimSpace(basic.Username) == "" {
			return grpcutils.InvalidArg("PrometheusRemoteWrite basic auth username must be set")
		}
		if basic.Password == nil || basic.Password.GetFromSecret() == "" {
			return grpcutils.InvalidArg("PrometheusRemoteWrite basic auth password secret must be set")
		}
		return s.validateSecretOwner(ctx, basic.Password)

	case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_:
		custom := auth.GetCustom()
		if custom == nil {
			return grpcutils.InvalidArg("Invalid PrometheusRemoteWrite custom auth")
		}
		if err := validateCustomAuthHeader("PrometheusRemoteWrite", custom.Header); err != nil {
			return err
		}
		if custom.Value == nil || custom.Value.GetFromSecret() == "" {
			return grpcutils.InvalidArg("PrometheusRemoteWrite custom auth value secret must be set")
		}
		return s.validateSecretOwner(ctx, custom.Value)

	default:
		return grpcutils.InvalidArg("Invalid PrometheusRemoteWrite auth type")
	}
}

func validateCustomAuthHeader(name, header string) error {
	header = strings.TrimSpace(header)
	if header == "" {
		return grpcutils.InvalidArg("%s custom auth header must be set", name)
	}

	if strings.EqualFold(header, "authorization") {
		return grpcutils.InvalidArg("%s custom auth must not use Authorization header", name)
	}

	if len(header) > 128 {
		return grpcutils.InvalidArg("%s custom auth header is too long", name)
	}

	for _, r := range header {
		if r <= 32 || r >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={}", r) {
			return grpcutils.InvalidArg("%s custom auth header contains invalid character", name)
		}
	}

	return nil
}

func validateKafkaSignal(
	name string,
	sig *enterprisev1.CollectorExporter_Spec_Kafka_Signal,
	allowRaw bool,
) error {
	if sig == nil {
		return nil
	}

	if sig.Topic != "" {
		if len(sig.Topic) > 249 {
			return grpcutils.InvalidArg("%s topic is too long", name)
		}
		if strings.ContainsAny(sig.Topic, "\x00\r\n") {
			return grpcutils.InvalidArg("%s topic contains invalid characters", name)
		}
	}

	switch sig.Encoding {
	case enterprisev1.CollectorExporter_Spec_Kafka_ENCODING_UNSET,
		enterprisev1.CollectorExporter_Spec_Kafka_OTLP_PROTO,
		enterprisev1.CollectorExporter_Spec_Kafka_OTLP_JSON:
		return nil

	case enterprisev1.CollectorExporter_Spec_Kafka_RAW:
		if allowRaw {
			return nil
		}
		return grpcutils.InvalidArg("Kafka raw encoding is only valid for logs")

	default:
		return grpcutils.InvalidArg("Invalid %s encoding", name)
	}
}

func validateCollectorExporterKafkaProducer(
	producer *enterprisev1.CollectorExporter_Spec_Kafka_Producer,
) error {
	if producer.MaxMessageBytes < 0 || producer.MaxMessageBytes > 100*1024*1024 {
		return grpcutils.InvalidArg("Invalid Kafka producer maxMessageBytes")
	}

	switch producer.RequiredAcks {
	case 0, 1, -1:
	default:
		return grpcutils.InvalidArg("Kafka producer requiredAcks must be 0, 1, or -1")
	}

	switch producer.Compression {
	case enterprisev1.CollectorExporter_Spec_Kafka_PRODUCER_COMPRESSION_UNSET,
		enterprisev1.CollectorExporter_Spec_Kafka_NONE,
		enterprisev1.CollectorExporter_Spec_Kafka_GZIP,
		enterprisev1.CollectorExporter_Spec_Kafka_SNAPPY,
		enterprisev1.CollectorExporter_Spec_Kafka_LZ4,
		enterprisev1.CollectorExporter_Spec_Kafka_ZSTD:
	default:
		return grpcutils.InvalidArg("Invalid Kafka producer compression")
	}

	if producer.FlushMaxMessages < 0 {
		return grpcutils.InvalidArg("Kafka producer flushMaxMessages cannot be negative")
	}

	if producer.Linger != nil {
		if err := validateCollectorDuration("Kafka producer linger", producer.Linger, time.Millisecond, time.Minute, true); err != nil {
			return err
		}
	}

	return nil
}

func validateElasticsearchIndexLike(name, val string) error {
	if len(val) > 255 {
		return grpcutils.InvalidArg("%s is too long", name)
	}

	if strings.TrimSpace(val) == "" {
		return grpcutils.InvalidArg("%s cannot be empty", name)
	}

	if strings.ContainsAny(val, `\/*?"<>| ,#`) {
		return grpcutils.InvalidArg("%s contains invalid characters", name)
	}

	if strings.HasPrefix(val, "-") || strings.HasPrefix(val, "_") || strings.HasPrefix(val, "+") {
		return grpcutils.InvalidArg("%s must not start with '-', '_' or '+'", name)
	}

	return nil
}

func validateClickhouseEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return grpcutils.InvalidArg("Invalid ClickHouse endpoint")
	}

	switch u.Scheme {
	case "tcp", "http", "https", "clickhouse":
		return nil
	default:
		return grpcutils.InvalidArg("ClickHouse endpoint must use tcp, http, https, or clickhouse scheme")
	}
}

func validateClickhouseIdent(name, val string) error {
	val = strings.TrimSpace(val)
	if val == "" {
		return grpcutils.InvalidArg("%s cannot be empty", name)
	}

	if len(val) > 128 {
		return grpcutils.InvalidArg("%s is too long", name)
	}

	for _, r := range val {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return grpcutils.InvalidArg("%s contains invalid character", name)
		}
	}

	return nil
}

func validateClickhouseMetricsTables(tables *enterprisev1.CollectorExporter_Spec_Clickhouse_MetricsTables) error {
	if tables == nil {
		return nil
	}

	check := func(name, val string) error {
		if val == "" {
			return nil
		}
		return validateClickhouseIdent(name, val)
	}

	if err := check("ClickHouse metricsTables.gauge", tables.Gauge); err != nil {
		return err
	}
	if err := check("ClickHouse metricsTables.sum", tables.Sum); err != nil {
		return err
	}
	if err := check("ClickHouse metricsTables.summary", tables.Summary); err != nil {
		return err
	}
	if err := check("ClickHouse metricsTables.histogram", tables.Histogram); err != nil {
		return err
	}
	if err := check("ClickHouse metricsTables.exponentialHistogram", tables.ExponentialHistogram); err != nil {
		return err
	}

	return nil
}

func validatePrometheusMetricPrefix(name, val string) error {
	if val == "" {
		return nil
	}

	for i, r := range val {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '_':
		case r == ':':
		case r >= '0' && r <= '9':
			if i == 0 {
				return grpcutils.InvalidArg("%s must not start with a digit", name)
			}
		default:
			return grpcutils.InvalidArg("%s contains invalid character", name)
		}
	}

	return nil
}

func validatePrometheusLabelName(name, val string, allowReservedPrefix bool) error {
	if val == "" {
		return grpcutils.InvalidArg("%s cannot be empty", name)
	}

	if strings.HasPrefix(val, "__") && !allowReservedPrefix {
		return grpcutils.InvalidArg("%s must not start with reserved prefix __", name)
	}

	for i, r := range val {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '_':
		case r >= '0' && r <= '9':
			if i == 0 {
				return grpcutils.InvalidArg("%s must not start with a digit", name)
			}
		default:
			return grpcutils.InvalidArg("%s contains invalid character", name)
		}
	}

	return nil
}

func validateADXIdentifier(name, val string) error {
	val = strings.TrimSpace(val)
	if val == "" {
		return grpcutils.InvalidArg("%s cannot be empty", name)
	}

	if len(val) > 128 {
		return grpcutils.InvalidArg("%s is too long", name)
	}

	for _, r := range val {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return grpcutils.InvalidArg("%s contains invalid character", name)
		}
	}

	return nil
}

func validateADXMappingName(name, val string) error {
	val = strings.TrimSpace(val)
	if val == "" {
		return grpcutils.InvalidArg("%s cannot be empty", name)
	}

	if len(val) > 256 {
		return grpcutils.InvalidArg("%s is too long", name)
	}

	for _, r := range val {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		case r == '-':
		case r == '.':
		default:
			return grpcutils.InvalidArg("%s contains invalid character", name)
		}
	}

	return nil
}
