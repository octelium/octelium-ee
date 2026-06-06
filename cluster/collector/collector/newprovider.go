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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/pkg/errors"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
)

type provider struct {
	schemeName string
	waitFn     confmap.WatcherFunc
	octeliumC  octeliumc.ClientInterface
	port       int
	mu         sync.RWMutex

	internalLogstoreEndpoint    string
	internalMetricstoreEndpoint string
}

type mt = map[string]any

func (c *provider) newFactory() confmap.ProviderFactory {
	return confmap.NewProviderFactory(func(_ confmap.ProviderSettings) confmap.Provider {
		return c
	})
}

func (p *provider) Retrieve(ctx context.Context, uri string, waitFn confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, p.schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, p.schemeName)
	}

	zap.L().Debug("Retrieving provider config")
	p.setWaitFn(waitFn)

	cfg, err := p.getConfig(ctx)
	if err != nil {
		return nil, err
	}

	return confmap.NewRetrieved(cfg)
}

func (p *provider) Scheme() string {
	return p.schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}

func (p *provider) getExporter(ctx context.Context, exp *enterprisev1.CollectorExporter) (*exporterInfo, error) {

	octeliumC := p.octeliumC
	ret := &exporterInfo{
		Exporter: exp,
	}

	switch exp.Spec.Type.(type) {
	case *enterprisev1.CollectorExporter_Spec_Otlp:

		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetOtlp()

		c := &exporterOTLP{
			Endpoint: spec.Endpoint,
			TLS: func() *otelTLS {
				if spec.NoTLS {
					return &otelTLS{
						Insecure: true,
					}
				}
				return nil
			}(),
			Headers: func() map[string]string {
				if len(spec.Headers) < 1 {
					return nil
				}

				ret := make(map[string]string)

				for _, hdr := range spec.Headers {
					ret[hdr.Key] = hdr.Value
				}

				return ret
			}(),
		}
		ret.exp = c

		if spec.Auth != nil {
			if c.Headers == nil {
				c.Headers = make(map[string]string)
			}
			switch spec.Auth.Type.(type) {
			case *enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_:
				if spec.Auth.GetBearer().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetBearer().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					c.Headers["Authorization"] = (fmt.Sprintf("Bearer %s", uenterprisev1.ToSecret(sec).GetValueStr()))
				}
			case *enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_:
				if spec.Auth.GetBasic().GetPassword() != nil && spec.Auth.GetBasic().GetPassword().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetBasic().GetPassword().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					authVal := base64.StdEncoding.EncodeToString(
						[]byte(fmt.Sprintf("%s:%s",
							spec.Auth.GetBasic().Username, uenterprisev1.ToSecret(sec).GetValueStr())))

					c.Headers["Authorization"] = (fmt.Sprintf("Basic %s", authVal))
				}
			case *enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_:
				if spec.Auth.GetCustom().GetHeader() != "" &&
					spec.Auth.GetCustom().GetValue() != nil &&
					spec.Auth.GetCustom().GetValue().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetCustom().GetValue().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					c.Headers[spec.Auth.GetCustom().GetHeader()] = (uenterprisev1.ToSecret(sec).GetValueStr())
				}
			}
		}

	case *enterprisev1.CollectorExporter_Spec_OtlpHTTP:

		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetOtlpHTTP()

		c := &exporterOTLPHTTP{
			Endpoint:        spec.Endpoint,
			MetricsEndpoint: spec.MetricsEndpoint,
			LogsEndpoint:    spec.LogsEndpoint,
			Headers: func() map[string]string {
				if len(spec.Headers) < 1 {
					return nil
				}

				ret := make(map[string]string)

				for _, hdr := range spec.Headers {
					ret[hdr.Key] = hdr.Value
				}

				return ret
			}(),
		}
		ret.exp = c

		// cfg.Encoding = otlphttpexporter.EncodingJSON
		switch spec.Mode {
		case enterprisev1.CollectorExporter_Spec_OTLPHTTP_JSON:
			c.Encoding = "json"
		case enterprisev1.CollectorExporter_Spec_OTLPHTTP_PROTO:
			c.Encoding = "proto"
		}

		switch spec.Compression {
		case enterprisev1.CollectorExporter_Spec_OTLPHTTP_NONE:
			c.Compression = "none"
		default:
			c.Compression = "gzip"
		}

		if spec.Auth != nil {
			if c.Headers == nil {
				c.Headers = make(map[string]string)
			}
			switch spec.Auth.Type.(type) {
			case *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_:
				if spec.Auth.GetBearer().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetBearer().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					c.Headers["Authorization"] = (fmt.Sprintf("Bearer %s", uenterprisev1.ToSecret(sec).GetValueStr()))
				}
			case *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_:
				if spec.Auth.GetBasic().GetPassword() != nil && spec.Auth.GetBasic().GetPassword().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetBasic().GetPassword().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					authVal := base64.StdEncoding.EncodeToString(
						[]byte(fmt.Sprintf("%s:%s",
							spec.Auth.GetBasic().Username, uenterprisev1.ToSecret(sec).GetValueStr())))

					c.Headers["Authorization"] = (fmt.Sprintf("Basic %s", authVal))

				}
			case *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_:
				if spec.Auth.GetCustom().GetHeader() != "" &&
					spec.Auth.GetCustom().GetValue() != nil &&
					spec.Auth.GetCustom().GetValue().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetCustom().GetValue().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					c.Headers[spec.Auth.GetCustom().GetHeader()] = (uenterprisev1.ToSecret(sec).GetValueStr())
				}
			}
		}

	case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_:

		ret.HasLogs = false
		ret.HasMetrics = true

		spec := exp.Spec.GetPrometheusRemoteWrite()
		c := &exporterPrometheusRemoteWriteRead{
			Endpoint: spec.Endpoint,
			Headers:  spec.Headers,
		}
		ret.exp = c

		c.Namespace = spec.Namespace

		if spec.Auth != nil {
			if c.Headers == nil {
				c.Headers = make(map[string]string)
			}
			switch spec.Auth.Type.(type) {
			case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Bearer_:
				if spec.Auth.GetBearer().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetBearer().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					c.Headers["Authorization"] = (fmt.Sprintf("Bearer %s", uenterprisev1.ToSecret(sec).GetValueStr()))
				}

			case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic_:
				if spec.Auth.GetBasic().GetUser() != "" &&
					spec.Auth.GetBasic().GetPassword() != nil &&
					spec.Auth.GetBasic().GetPassword().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetBasic().GetPassword().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					authVal := base64.StdEncoding.EncodeToString(
						[]byte(fmt.Sprintf("%s:%s",
							spec.Auth.GetBasic().GetUser(), uenterprisev1.ToSecret(sec).GetValueStr())))

					c.Headers["Authorization"] = (fmt.Sprintf("Basic %s", authVal))
				}
			case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom_:
				if spec.Auth.GetCustom().GetHeader() != "" &&
					spec.Auth.GetCustom().GetValue() != nil &&
					spec.Auth.GetCustom().GetValue().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetCustom().GetValue().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					c.Headers[spec.Auth.GetCustom().GetHeader()] = (uenterprisev1.ToSecret(sec).GetValueStr())
				}
			}
		}

	case *enterprisev1.CollectorExporter_Spec_Clickhouse_:

		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetClickhouse()
		c := &exporterClickhouse{
			Endpoint: spec.Endpoint,
			Database: spec.Database,
			Username: spec.Username,
		}
		ret.exp = c

		if spec.GetPassword() != nil && spec.GetPassword().GetFromSecret() != "" {
			sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
				Name: spec.GetPassword().GetFromSecret(),
			})
			if err != nil {
				return nil, err
			}

			c.Password = (uenterprisev1.ToSecret(sec).GetValueStr())

		}

	case *enterprisev1.CollectorExporter_Spec_Elasticsearch_:
		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetElasticsearch()
		if spec == nil {
			return nil, errors.Errorf("nil Elasticsearch exporter spec")
		}

		timeout, err := durationToCollectorString(spec.Timeout)
		if err != nil {
			return nil, err
		}

		c := &exporterElasticsearch{
			Endpoint:     spec.Endpoint,
			Endpoints:    append([]string(nil), spec.Endpoints...),
			CloudID:      spec.CloudID,
			Pipeline:     spec.Pipeline,
			LogsIndex:    spec.LogsIndex,
			MetricsIndex: spec.MetricsIndex,
			Headers:      elasticHeadersToMap(spec.Headers),
			Timeout:      timeout,
		}
		ret.exp = c

		switch spec.Compression {
		case enterprisev1.CollectorExporter_Spec_Elasticsearch_GZIP,
			enterprisev1.CollectorExporter_Spec_Elasticsearch_COMPRESSION_UNSET:
		case enterprisev1.CollectorExporter_Spec_Elasticsearch_NONE:
			c.Compression = "none"
		default:
			return nil, errors.Errorf("invalid Elasticsearch compression")
		}

		if spec.Tls != nil && spec.Tls.InsecureSkipVerify {
			c.TLS = &elasticTLS{
				InsecureSkipVerify: true,
			}
		}

		if spec.Auth != nil {
			switch spec.Auth.Type.(type) {
			case *enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_ApiKey:
				if spec.Auth.GetApiKey().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetApiKey().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					c.APIKey = uenterprisev1.ToSecret(sec).GetValueStr()
				}

			case *enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_Basic_:
				basic := spec.Auth.GetBasic()
				if basic != nil {
					c.User = basic.User

					if basic.GetPassword() != nil && basic.GetPassword().GetFromSecret() != "" {
						sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
							Name: basic.GetPassword().GetFromSecret(),
						})
						if err != nil {
							return nil, err
						}

						c.Password = uenterprisev1.ToSecret(sec).GetValueStr()
					}
				}
			}
		}

	case *enterprisev1.CollectorExporter_Spec_Kafka_:
		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetKafka()
		if spec == nil {
			return nil, errors.Errorf("nil Kafka exporter spec")
		}

		timeout, err := durationToCollectorString(spec.Timeout)
		if err != nil {
			return nil, err
		}

		connIdleTimeout, err := durationToCollectorString(spec.ConnIdleTimeout)
		if err != nil {
			return nil, err
		}

		c := &exporterKafka{
			Brokers:                              append([]string(nil), spec.Brokers...),
			ProtocolVersion:                      spec.ProtocolVersion,
			ClientID:                             spec.ClientID,
			RecordHeaders:                        kafkaHeadersToCollector(spec.RecordHeaders),
			Timeout:                              timeout,
			ConnIdleTimeout:                      connIdleTimeout,
			PartitionLogsByResourceAttributes:    spec.PartitionLogsByResourceAttributes,
			PartitionMetricsByResourceAttributes: spec.PartitionMetricsByResourceAttributes,
		}
		ret.exp = c

		if spec.Logs != nil {
			enc, err := kafkaEncodingToCollectorString(spec.Logs.Encoding, true)
			if err != nil {
				return nil, err
			}

			c.Logs = &exporterKafkaSignal{
				Topic:    spec.Logs.Topic,
				Encoding: enc,
			}
		}

		if spec.Metrics != nil {
			enc, err := kafkaEncodingToCollectorString(spec.Metrics.Encoding, false)
			if err != nil {
				return nil, err
			}

			c.Metrics = &exporterKafkaSignal{
				Topic:    spec.Metrics.Topic,
				Encoding: enc,
			}
		}

		if spec.Tls != nil {
			c.TLS = &exporterKafkaTLS{
				Insecure:           spec.Tls.Insecure,
				InsecureSkipVerify: spec.Tls.InsecureSkipVerify,
			}
		}

		if spec.Auth != nil {
			switch spec.Auth.Type.(type) {
			case *enterprisev1.CollectorExporter_Spec_Kafka_Auth_Sasl:
				sasl := spec.Auth.GetSasl()
				if sasl != nil {
					mech, err := kafkaSASLMechanismToCollectorString(sasl.Mechanism)
					if err != nil {
						return nil, err
					}

					c.Auth = &exporterKafkaAuth{
						SASL: &exporterKafkaSASL{
							Username:  sasl.Username,
							Mechanism: mech,
						},
					}

					if sasl.GetPassword() != nil && sasl.GetPassword().GetFromSecret() != "" {
						sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
							Name: sasl.GetPassword().GetFromSecret(),
						})
						if err != nil {
							return nil, err
						}

						c.Auth.SASL.Password = uenterprisev1.ToSecret(sec).GetValueStr()
					}
				}
			}
		}

		if spec.Producer != nil {
			compression, err := kafkaProducerCompressionToCollectorString(spec.Producer.Compression)
			if err != nil {
				return nil, err
			}

			linger, err := durationToCollectorString(spec.Producer.Linger)
			if err != nil {
				return nil, err
			}

			c.Producer = &exporterKafkaProducer{
				MaxMessageBytes:        spec.Producer.MaxMessageBytes,
				RequiredAcks:           spec.Producer.RequiredAcks,
				Compression:            compression,
				FlushMaxMessages:       spec.Producer.FlushMaxMessages,
				AllowAutoTopicCreation: spec.Producer.AllowAutoTopicCreation,
				Linger:                 linger,
			}
		}

	case *enterprisev1.CollectorExporter_Spec_Datadog_:
		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetDatadog()
		if spec == nil {
			return nil, errors.Errorf("nil Datadog exporter spec")
		}

		c := &exporterDatadog{
			API: &exporterDatadogAPI{},
		}
		ret.exp = c

		if spec.Api != nil {
			c.API.Site = spec.Api.Site
			c.API.FailOnInvalidKey = spec.Api.FailOnInvalidKey

			if spec.Api.GetKey() != nil && spec.Api.GetKey().GetFromSecret() != "" {
				sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
					Name: spec.Api.GetKey().GetFromSecret(),
				})
				if err != nil {
					return nil, err
				}

				c.API.Key = uenterprisev1.ToSecret(sec).GetValueStr()
			}
		}

		c.Hostname = spec.Hostname

		if spec.Metrics != nil {
			c.Metrics = &exporterDatadogMetrics{
				Endpoint:                           spec.Metrics.Endpoint,
				ResourceAttributesAsTags:           spec.Metrics.ResourceAttributesAsTags,
				InstrumentationScopeMetadataAsTags: spec.Metrics.InstrumentationScopeMetadataAsTags,
			}
		}

		if spec.Logs != nil {
			batchWait, err := durationToCollectorString(spec.Logs.BatchWait)
			if err != nil {
				return nil, err
			}

			c.Logs = &exporterDatadogLogs{
				Endpoint:         spec.Logs.Endpoint,
				CompressionLevel: spec.Logs.CompressionLevel,
				BatchWait:        batchWait,
			}

			c.Logs.UseCompression = &spec.Logs.UseCompression
		}

		if spec.HostMetadata != nil {
			reporterPeriod, err := durationToCollectorString(spec.HostMetadata.ReporterPeriod)
			if err != nil {
				return nil, err
			}

			c.HostMetadata = &exporterDatadogHostMetadata{
				ReporterPeriod: reporterPeriod,
			}

			c.HostMetadata.Enabled = &spec.HostMetadata.Enabled
		}

		hostnameDetectionTimeout, err := durationToCollectorString(spec.HostnameDetectionTimeout)
		if err != nil {
			return nil, err
		}
		c.HostnameDetectionTimeout = hostnameDetectionTimeout

	case *enterprisev1.CollectorExporter_Spec_Logzio_:
		ret.HasLogs = true

		spec := exp.Spec.GetLogzio()

		c := &exporterLogzio{
			Region:   spec.Region,
			Endpoint: spec.Endpoint,
		}
		ret.exp = c

		if spec.GetToken() != nil && spec.GetToken().GetFromSecret() != "" {
			sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
				Name: spec.GetToken().GetFromSecret(),
			})
			if err != nil {
				return nil, err
			}

			c.AccountToken = uenterprisev1.ToSecret(sec).GetValueStr()
		}

	case *enterprisev1.CollectorExporter_Spec_Splunk_:
		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetSplunk()
		if spec == nil {
			return nil, errors.Errorf("nil Splunk exporter spec")
		}

		timeout, err := durationToCollectorString(spec.Timeout)
		if err != nil {
			return nil, err
		}

		c := &exporterSplunk{
			Endpoint:                spec.Endpoint,
			Source:                  spec.Source,
			SourceType:              spec.SourceType,
			Index:                   spec.Index,
			UseMultiMetricFormat:    spec.UseMultiMetricFormat,
			SplunkAppName:           spec.AppName,
			SplunkAppVersion:        spec.AppVersion,
			MaxContentLengthLogs:    spec.MaxContentLengthLogs,
			MaxContentLengthMetrics: spec.MaxContentLengthMetrics,
			DisableCompression:      spec.DisableCompression,
			Timeout:                 timeout,
			MaxIdleConns:            spec.MaxIdleConns,
		}
		ret.exp = c

		if spec.Tls != nil && spec.Tls.InsecureSkipVerify {
			c.TLS = &splunkTLS{
				InsecureSkipVerify: true,
			}
		}

		if spec.GetToken() != nil && spec.GetToken().GetFromSecret() != "" {
			sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
				Name: spec.GetToken().GetFromSecret(),
			})
			if err != nil {
				return nil, err
			}

			c.Token = uenterprisev1.ToSecret(sec).GetValueStr()
		}

	case *enterprisev1.CollectorExporter_Spec_AzureMonitor_:
		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetAzureMonitor()
		if spec == nil {
			return nil, errors.Errorf("nil AzureMonitor exporter spec")
		}

		maxBatchInterval, err := durationToCollectorString(spec.MaxBatchInterval)
		if err != nil {
			return nil, err
		}

		shutdownTimeout, err := durationToCollectorString(spec.ShutdownTimeout)
		if err != nil {
			return nil, err
		}

		c := &exporterAzureMonitor{
			Endpoint:               spec.Endpoint,
			MaxBatchSize:           spec.MaxBatchSize,
			MaxBatchInterval:       maxBatchInterval,
			ShutdownTimeout:        shutdownTimeout,
			CustomEventsEnabled:    spec.CustomEventsEnabled,
			ExceptionEventsEnabled: spec.ExceptionEventsEnabled,
		}
		ret.exp = c

		if spec.GetConnectionString() != nil && spec.GetConnectionString().GetFromSecret() != "" {
			sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
				Name: spec.GetConnectionString().GetFromSecret(),
			})
			if err != nil {
				return nil, err
			}

			c.ConnectionString = uenterprisev1.ToSecret(sec).GetValueStr()
		}

		if spec.GetInstrumentationKey() != nil && spec.GetInstrumentationKey().GetFromSecret() != "" {
			sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
				Name: spec.GetInstrumentationKey().GetFromSecret(),
			})
			if err != nil {
				return nil, err
			}

			c.InstrumentationKey = uenterprisev1.ToSecret(sec).GetValueStr()
		}

		if spec.MaxBatchInterval != nil {
			if err := validateCollectorDuration(
				"AzureMonitor maxBatchInterval",
				spec.MaxBatchInterval,
				time.Second,
				10*time.Minute,
				true,
			); err != nil {
				return nil, err
			}
		}

		if spec.ShutdownTimeout != nil {
			if err := validateCollectorDuration(
				"AzureMonitor shutdownTimeout",
				spec.ShutdownTimeout,
				time.Second,
				5*time.Minute,
				true,
			); err != nil {
				return nil, err
			}
		}

	case *enterprisev1.CollectorExporter_Spec_InfluxDB_:
		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetInfluxDB()
		if spec == nil {
			return nil, errors.Errorf("nil InfluxDB exporter spec")
		}

		timeout, err := durationToCollectorString(spec.Timeout)
		if err != nil {
			return nil, err
		}

		metricsSchema, err := influxMetricsSchemaToCollectorString(spec.MetricsSchema)
		if err != nil {
			return nil, err
		}

		precision, err := influxPrecisionToCollectorString(spec.Precision)
		if err != nil {
			return nil, err
		}

		c := &exporterInfluxDB{
			Endpoint:            spec.Endpoint,
			Org:                 spec.Org,
			Bucket:              spec.Bucket,
			Headers:             influxHeadersToMap(spec.Headers),
			MetricsSchema:       metricsSchema,
			Precision:           precision,
			PayloadMaxLines:     spec.PayloadMaxLines,
			PayloadMaxBytes:     spec.PayloadMaxBytes,
			Timeout:             timeout,
			LogRecordDimensions: append([]string(nil), spec.LogRecordDimensions...),
		}
		ret.exp = c

		if spec.GetToken() != nil && spec.GetToken().GetFromSecret() != "" {
			sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
				Name: spec.GetToken().GetFromSecret(),
			})
			if err != nil {
				return nil, err
			}

			c.Token = uenterprisev1.ToSecret(sec).GetValueStr()
		}

		if spec.V1Compatibility != nil {
			c.V1Compatibility = &exporterInfluxDBV1Compatibility{
				Enabled:  spec.V1Compatibility.Enabled,
				DB:       spec.V1Compatibility.Db,
				Username: spec.V1Compatibility.Username,
			}

			if spec.V1Compatibility.GetPassword() != nil &&
				spec.V1Compatibility.GetPassword().GetFromSecret() != "" {
				sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
					Name: spec.V1Compatibility.GetPassword().GetFromSecret(),
				})
				if err != nil {
					return nil, err
				}

				c.V1Compatibility.Password = uenterprisev1.ToSecret(sec).GetValueStr()
			}
		}
	default:
		return nil, errors.Errorf("Unsupported exporter type: %+v", exp)
	}

	if err := ret.toExporterMapAny(); err != nil {
		return nil, err
	}

	return ret, nil

}

func (e *exporterInfo) toExporterMapAny() error {
	jsn, err := json.Marshal(e.exp)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jsn, &e.exporterMap); err != nil {
		return err
	}
	return nil
}

func (p *provider) getType(exp *enterprisev1.CollectorExporter) string {

	switch exp.Spec.Type.(type) {
	case *enterprisev1.CollectorExporter_Spec_AzureDataExplorer_:
		return "azuredataexplorer"
	case *enterprisev1.CollectorExporter_Spec_AzureMonitor_:
		return "azuremonitor"
	case *enterprisev1.CollectorExporter_Spec_Datadog_:
		return "datadog"
	case *enterprisev1.CollectorExporter_Spec_Elasticsearch_:
		return "elasticsearch"
	case *enterprisev1.CollectorExporter_Spec_InfluxDB_:
		return "influxdb"
	case *enterprisev1.CollectorExporter_Spec_Kafka_:
		return "kafka"
	case *enterprisev1.CollectorExporter_Spec_Logzio_:
		return "logzio"
	case *enterprisev1.CollectorExporter_Spec_Otlp:
		return "otlp"
	case *enterprisev1.CollectorExporter_Spec_OtlpHTTP:
		return "otlphttp"
	case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_:
		return "prometheusremotewrite"
	case *enterprisev1.CollectorExporter_Spec_Splunk_:
		return "splunk"
	case *enterprisev1.CollectorExporter_Spec_Clickhouse_:
		return "clickhouse"
	default:
		return ""
	}
}

func (p *provider) getTypeName(exp *enterprisev1.CollectorExporter) string {
	return fmt.Sprintf("%s/%s", p.getType(exp), exp.Metadata.Uid)
}

func (c *provider) getConfig(ctx context.Context) (map[string]any, error) {

	octeliumC := c.octeliumC
	cc, err := octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, err
	}

	exporterList, err := octeliumC.EnterpriseC().ListCollectorExporter(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret, err := c.getInitConfig(ctx)
	if err != nil {
		return nil, errors.Errorf("Could not get init config: %+v", err)
	}
	exportersMap := ret["exporters"].(map[string]any)
	extensionsMap := ret["extensions"].(map[string]any)
	serviceMap := ret["service"].(map[string]any)
	serviceExtensions := serviceMap["extensions"].([]string)
	pipelinesMap := serviceMap["pipelines"].(map[string]any)

	if cc.Spec.Collector == nil || len(exporterList.Items) == 0 {
		zap.L().Debug("No exporters in the ClusterConfig. Nothing to be done...")
		return ret, nil
	}

	var exporterInfoList []*exporterInfo

	getExporterInfo := func(name string) *exporterInfo {
		for _, itm := range exporterInfoList {
			if itm.Exporter.Metadata.Name == name {
				return itm
			}
		}
		return nil
	}

	for _, exp := range exporterList.Items {
		if exp.Spec.IsDisabled {
			continue
		}

		info, err := c.getExporter(ctx, exp)
		if err != nil {
			zap.L().Warn("Could not get exporter info. Ignoring it...",
				zap.String("name", exp.Metadata.Name),
				zap.Error(err),
			)
			continue
		}

		exporterInfoList = append(exporterInfoList, info)
		zap.L().Debug("Exporter added", zap.String("name", info.Exporter.Metadata.Name))
		exportersMap[c.getTypeName(info.Exporter)] = info.exporterMap
		for k, v := range info.extensionMap {
			extensionsMap[k] = v
			serviceExtensions = append(serviceExtensions, k)
		}
	}

	for _, pipeline := range cc.Spec.Collector.Pipelines {
		if pipeline.IsDisabled {
			continue
		}
		if len(pipeline.Exporters) < 1 {
			continue
		}

		pipelineType := func() string {
			switch pipeline.Type {
			case enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS:
				return "logs"
			case enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS:
				return "metrics"
			default:
				return ""
			}
		}()
		if pipelineType == "" {
			continue
		}

		exporters := func() []string {
			var ret []string
			for _, exp := range pipeline.Exporters {
				info := getExporterInfo(exp)
				if info == nil {
					continue
				}
				switch {
				case pipeline.Type == enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS && !info.HasLogs:
					continue
				case pipeline.Type == enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS && !info.HasMetrics:
					continue
				}
				ret = append(ret, c.getTypeName(info.Exporter))
			}
			return ret
		}()

		if len(exporters) < 1 {
			continue
		}

		pipelineMap := mt{
			"receivers":  []string{"octelium_otlp"},
			"exporters":  exporters,
			"processors": []string{"memory_limiter", "batch"},
		}

		pipelinesMap[fmt.Sprintf("%s/%s", pipelineType, pipeline.Name)] = pipelineMap
	}

	zap.L().Debug("Successfully obtained provider config", zap.Any("cfg", ret))

	return ret, nil
}

func (c *provider) getInitConfig(ctx context.Context) (map[string]any, error) {

	return mt{
		"exporters": mt{
			"octelium_otlp/logstore": mt{
				"endpoint":                   c.getInternalLogstoreEndpoint(),
				"compression":                "gzip",
				"wait_for_ready":             true,
				"max_call_recv_msg_size_mib": 16,
				"max_call_send_msg_size_mib": 16,
			},
			"octelium_otlp/metricstore": mt{
				"endpoint":                   c.getInternalMetricstoreEndpoint(),
				"compression":                "gzip",
				"wait_for_ready":             true,
				"max_call_recv_msg_size_mib": 16,
				"max_call_send_msg_size_mib": 16,
			},
		},
		"extensions": mt{},
		"receivers": mt{
			"octelium_otlp": mt{
				"endpoint":               fmt.Sprintf(":%d", c.port),
				"max_recv_msg_size_mib":  16,
				"max_concurrent_streams": 1024,
				"read_buffer_size":       512 * 1024,
			},
		},
		"processors": mt{
			"batch": mt{},
			"memory_limiter": mt{
				"check_interval": "1s",
				"limit_mib":      800,
			},
		},
		"service": mt{
			"extensions": []string{},
			"pipelines": mt{
				"logs": mt{
					"receivers":  []string{"octelium_otlp"},
					"processors": []string{"memory_limiter", "batch"},
					"exporters":  []string{"octelium_otlp/logstore"},
				},
				"metrics": mt{
					"receivers":  []string{"octelium_otlp"},
					"processors": []string{"memory_limiter", "batch"},
					"exporters":  []string{"octelium_otlp/metricstore"},
				},
			},
		},
	}, nil
}

type exporterInfo struct {
	Exporter     *enterprisev1.CollectorExporter
	exp          any
	HasLogs      bool
	HasMetrics   bool
	exporterMap  map[string]any
	extensionMap map[string]any
}

func (p *provider) setWaitFn(fn confmap.WatcherFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.waitFn = fn
}

func (p *provider) sendUpdate() {
	p.mu.RLock()
	fn := p.waitFn
	p.mu.RUnlock()

	if fn != nil {
		zap.L().Debug("Config provider sending update...")
		fn(&confmap.ChangeEvent{})
	}
}

func (p *provider) getInternalLogstoreEndpoint() string {
	if p.internalLogstoreEndpoint != "" {
		return p.internalLogstoreEndpoint
	}
	return defaultInternalLogstoreEndpoint
}

func (p *provider) getInternalMetricstoreEndpoint() string {
	if p.internalMetricstoreEndpoint != "" {
		return p.internalMetricstoreEndpoint
	}
	return defaultInternalMetricstoreEndpoint
}

func durationToCollectorString(d *metav1.Duration) (string, error) {
	if d == nil {
		return "", nil
	}

	switch d.Type.(type) {
	case *metav1.Duration_Milliseconds:
		if d.GetMilliseconds() == 0 {
			return "", errors.Errorf("duration milliseconds cannot be zero")
		}
		return fmt.Sprintf("%dms", d.GetMilliseconds()), nil

	case *metav1.Duration_Seconds:
		if d.GetSeconds() == 0 {
			return "", errors.Errorf("duration seconds cannot be zero")
		}
		return fmt.Sprintf("%ds", d.GetSeconds()), nil

	case *metav1.Duration_Minutes:
		if d.GetMinutes() == 0 {
			return "", errors.Errorf("duration minutes cannot be zero")
		}
		return fmt.Sprintf("%dm", d.GetMinutes()), nil

	case *metav1.Duration_Hours:
		if d.GetHours() == 0 {
			return "", errors.Errorf("duration hours cannot be zero")
		}
		return fmt.Sprintf("%dh", d.GetHours()), nil

	case *metav1.Duration_Days:
		if d.GetDays() == 0 {
			return "", errors.Errorf("duration days cannot be zero")
		}
		return fmt.Sprintf("%dh", d.GetDays()*24), nil

	case *metav1.Duration_Weeks:
		if d.GetWeeks() == 0 {
			return "", errors.Errorf("duration weeks cannot be zero")
		}
		return fmt.Sprintf("%dh", d.GetWeeks()*7*24), nil

	case *metav1.Duration_Months:
		return "", errors.Errorf("months are not supported for Collector duration fields")

	default:
		return "", errors.Errorf("invalid duration")
	}
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

func elasticHeadersToMap(headers []*enterprisev1.CollectorExporter_Spec_Elasticsearch_Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	ret := make(map[string]string, len(headers))
	for _, h := range headers {
		if h == nil || h.Key == "" {
			continue
		}

		ret[h.Key] = h.Value
	}

	if len(ret) == 0 {
		return nil
	}

	return ret
}

func kafkaEncodingToCollectorString(enc enterprisev1.CollectorExporter_Spec_Kafka_Encoding, allowRaw bool) (string, error) {
	switch enc {
	case enterprisev1.CollectorExporter_Spec_Kafka_ENCODING_UNSET:
		return "", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_OTLP_PROTO:
		return "otlp_proto", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_OTLP_JSON:
		return "otlp_json", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_RAW:
		if !allowRaw {
			return "", errors.Errorf("raw Kafka encoding is only valid for logs")
		}
		return "raw", nil
	default:
		return "", errors.Errorf("invalid Kafka encoding")
	}
}

func kafkaSASLMechanismToCollectorString(mech enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism) (string, error) {
	switch mech {
	case enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_MECHANISM_UNSET,
		enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_PLAIN:
		return "PLAIN", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_SCRAM_SHA_256:
		return "SCRAM-SHA-256", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_Auth_SASL_SCRAM_SHA_512:
		return "SCRAM-SHA-512", nil
	default:
		return "", errors.Errorf("invalid Kafka SASL mechanism")
	}
}

func kafkaProducerCompressionToCollectorString(
	compression enterprisev1.CollectorExporter_Spec_Kafka_ProducerCompression,
) (string, error) {
	switch compression {
	case enterprisev1.CollectorExporter_Spec_Kafka_PRODUCER_COMPRESSION_UNSET:
		return "", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_NONE:
		return "none", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_GZIP:
		return "gzip", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_SNAPPY:
		return "snappy", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_LZ4:
		return "lz4", nil
	case enterprisev1.CollectorExporter_Spec_Kafka_ZSTD:
		return "zstd", nil
	default:
		return "", errors.Errorf("invalid Kafka producer compression")
	}
}

func kafkaHeadersToCollector(headers []*enterprisev1.CollectorExporter_Spec_Kafka_Header) []exporterKafkaHeader {
	if len(headers) == 0 {
		return nil
	}

	ret := make([]exporterKafkaHeader, 0, len(headers))
	for _, h := range headers {
		if h == nil || h.Key == "" {
			continue
		}
		ret = append(ret, exporterKafkaHeader{
			Key:   h.Key,
			Value: h.Value,
		})
	}

	if len(ret) == 0 {
		return nil
	}

	return ret
}

func influxHeadersToMap(headers []*enterprisev1.CollectorExporter_Spec_InfluxDB_Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	ret := make(map[string]string, len(headers))
	for _, h := range headers {
		if h == nil || h.Key == "" {
			continue
		}

		ret[h.Key] = h.Value
	}

	if len(ret) == 0 {
		return nil
	}

	return ret
}

func influxMetricsSchemaToCollectorString(
	schema enterprisev1.CollectorExporter_Spec_InfluxDB_MetricsSchema,
) (string, error) {
	switch schema {
	case enterprisev1.CollectorExporter_Spec_InfluxDB_METRICS_SCHEMA_UNSET:
		return "", nil
	case enterprisev1.CollectorExporter_Spec_InfluxDB_TELEGRAF_PROMETHEUS_V1:
		return "telegraf-prometheus-v1", nil
	case enterprisev1.CollectorExporter_Spec_InfluxDB_TELEGRAF_PROMETHEUS_V2:
		return "telegraf-prometheus-v2", nil
	default:
		return "", errors.Errorf("invalid InfluxDB metrics schema")
	}
}

func influxPrecisionToCollectorString(
	precision enterprisev1.CollectorExporter_Spec_InfluxDB_Precision,
) (string, error) {
	switch precision {
	case enterprisev1.CollectorExporter_Spec_InfluxDB_PRECISION_UNSET:
		return "", nil
	case enterprisev1.CollectorExporter_Spec_InfluxDB_NS:
		return "ns", nil
	case enterprisev1.CollectorExporter_Spec_InfluxDB_US:
		return "us", nil
	case enterprisev1.CollectorExporter_Spec_InfluxDB_MS:
		return "ms", nil
	case enterprisev1.CollectorExporter_Spec_InfluxDB_S:
		return "s", nil
	default:
		return "", errors.Errorf("invalid InfluxDB precision")
	}
}
