// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package collector

type exporterOTLP struct {
	Endpoint        string             `json:"endpoint,omitempty"`
	Headers         map[string]string  `json:"headers,omitempty"`
	Auth            *otelAuth          `json:"auth,omitempty"`
	TLS             *otelClientTLS     `json:"tls,omitempty"`
	Compression     string             `json:"compression,omitempty"`
	Timeout         string             `json:"timeout,omitempty"`
	WaitForReady    bool               `json:"wait_for_ready,omitempty"`
	Authority       string             `json:"authority,omitempty"`
	UserAgent       string             `json:"user_agent,omitempty"`
	BalancerName    string             `json:"balancer_name,omitempty"`
	ReadBufferSize  int32              `json:"read_buffer_size,omitempty"`
	WriteBufferSize int32              `json:"write_buffer_size,omitempty"`
	Keepalive       *otelGRPCKeepalive `json:"keepalive,omitempty"`
}

type otelClientTLS struct {
	Insecure           bool   `json:"insecure,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	ServerNameOverride string `json:"server_name_override,omitempty"`
	CAPEM              string `json:"ca_pem,omitempty"`
}

type otelGRPCKeepalive struct {
	Time                string `json:"time,omitempty"`
	Timeout             string `json:"timeout,omitempty"`
	PermitWithoutStream bool   `json:"permit_without_stream,omitempty"`
}

type otelAuth struct {
	Authenticator string `json:"authenticator,omitempty"`
}

type exporterOTLPHTTP struct {
	Endpoint        string             `json:"endpoint,omitempty"`
	MetricsEndpoint string             `json:"metrics_endpoint,omitempty"`
	LogsEndpoint    string             `json:"logs_endpoint,omitempty"`
	Headers         map[string]string  `json:"headers,omitempty"`
	Auth            *otelAuth          `json:"auth,omitempty"`
	TLS             *otelHTTPClientTLS `json:"tls,omitempty"`
	Compression     string             `json:"compression,omitempty"`
	Encoding        string             `json:"encoding,omitempty"`
	Timeout         string             `json:"timeout,omitempty"`
	ReadBufferSize  int32              `json:"read_buffer_size,omitempty"`
	WriteBufferSize int32              `json:"write_buffer_size,omitempty"`
}

type otelHTTPClientTLS struct {
	Insecure           bool   `json:"insecure,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	ServerNameOverride string `json:"server_name_override,omitempty"`
	CAPEM              string `json:"ca_pem,omitempty"`
}

type exporterPrometheusRemoteWrite struct {
	Endpoint                      string                                   `json:"endpoint,omitempty"`
	Namespace                     string                                   `json:"namespace,omitempty"`
	Headers                       map[string]string                        `json:"headers,omitempty"`
	ExternalLabels                map[string]string                        `json:"external_labels,omitempty"`
	TLS                           *prometheusRemoteWriteTLS                `json:"tls,omitempty"`
	Timeout                       string                                   `json:"timeout,omitempty"`
	RemoteWriteQueue              *prometheusRemoteWriteQueue              `json:"remote_write_queue,omitempty"`
	ResourceToTelemetryConversion *prometheusResourceToTelemetryConversion `json:"resource_to_telemetry_conversion,omitempty"`
	TargetInfo                    *prometheusTargetInfo                    `json:"target_info,omitempty"`
	DisableScopeInfo              bool                                     `json:"disable_scope_info,omitempty"`
	MaxBatchSizeBytes             int64                                    `json:"max_batch_size_bytes,omitempty"`
	MaxBatchRequestParallelism    *int32                                   `json:"max_batch_request_parallelism,omitempty"`
	TranslationStrategy           string                                   `json:"translation_strategy,omitempty"`
	SendMetadata                  bool                                     `json:"send_metadata,omitempty"`
}

type prometheusRemoteWriteTLS struct {
	Insecure           bool   `json:"insecure,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	ServerNameOverride string `json:"server_name_override,omitempty"`
	CAPEM              string `json:"ca_pem,omitempty"`
}

type prometheusRemoteWriteQueue struct {
	Enabled      bool  `json:"enabled"`
	QueueSize    int64 `json:"queue_size,omitempty"`
	NumConsumers int32 `json:"num_consumers,omitempty"`
}

type prometheusResourceToTelemetryConversion struct {
	Enabled                  bool `json:"enabled"`
	ExcludeServiceAttributes bool `json:"exclude_service_attributes,omitempty"`
}

type prometheusTargetInfo struct {
	Enabled bool `json:"enabled"`
}

type exporterClickhouse struct {
	Endpoint         string                   `json:"endpoint,omitempty"`
	Username         string                   `json:"username,omitempty"`
	Password         string                   `json:"password,omitempty"`
	Database         string                   `json:"database,omitempty"`
	TLS              *clickhouseTLS           `json:"tls,omitempty"`
	ConnectionParams map[string]string        `json:"connection_params,omitempty"`
	LogsTableName    string                   `json:"logs_table_name,omitempty"`
	MetricsTables    *clickhouseMetricsTables `json:"metrics_tables,omitempty"`
	TTL              string                   `json:"ttl,omitempty"`
	CreateSchema     *bool                    `json:"create_schema,omitempty"`
	Compress         string                   `json:"compress,omitempty"`
	AsyncInsert      *bool                    `json:"async_insert,omitempty"`
	JSON             bool                     `json:"json,omitempty"`
	ClusterName      string                   `json:"cluster_name,omitempty"`
	TableEngine      *clickhouseTableEngine   `json:"table_engine,omitempty"`
	Timeout          string                   `json:"timeout,omitempty"`
}

type clickhouseTLS struct {
	Insecure           bool   `json:"insecure,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	ServerNameOverride string `json:"server_name_override,omitempty"`
	CAPEM              string `json:"ca_pem,omitempty"`
}

type clickhouseTableEngine struct {
	Name   string `json:"name,omitempty"`
	Params string `json:"params,omitempty"`
}

type clickhouseMetricsTables struct {
	Gauge                *clickhouseMetricTable `json:"gauge,omitempty"`
	Sum                  *clickhouseMetricTable `json:"sum,omitempty"`
	Summary              *clickhouseMetricTable `json:"summary,omitempty"`
	Histogram            *clickhouseMetricTable `json:"histogram,omitempty"`
	ExponentialHistogram *clickhouseMetricTable `json:"exponential_histogram,omitempty"`
}

type clickhouseMetricTable struct {
	Name string `json:"name,omitempty"`
}

type exporterElasticsearch struct {
	Endpoint     string            `json:"endpoint,omitempty"`
	Endpoints    []string          `json:"endpoints,omitempty"`
	CloudID      string            `json:"cloudid,omitempty"`
	Pipeline     string            `json:"pipeline,omitempty"`
	LogsIndex    string            `json:"logs_index,omitempty"`
	MetricsIndex string            `json:"metrics_index,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	User         string            `json:"user,omitempty"`
	Password     string            `json:"password,omitempty"`
	APIKey       string            `json:"api_key,omitempty"`
	Compression  string            `json:"compression,omitempty"`
	Timeout      string            `json:"timeout,omitempty"`
	TLS          *elasticTLS       `json:"tls,omitempty"`
}

type elasticTLS struct {
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

type exporterDatadog struct {
	API *exporterDatadogAPI `json:"api,omitempty"`

	Hostname string `json:"hostname,omitempty"`

	Metrics *exporterDatadogMetrics `json:"metrics,omitempty"`
	Logs    *exporterDatadogLogs    `json:"logs,omitempty"`

	HostMetadata *exporterDatadogHostMetadata `json:"host_metadata,omitempty"`

	HostnameDetectionTimeout string `json:"hostname_detection_timeout,omitempty"`
}

type exporterDatadogAPI struct {
	Key              string `json:"key,omitempty"`
	Site             string `json:"site,omitempty"`
	FailOnInvalidKey bool   `json:"fail_on_invalid_key,omitempty"`
}

type exporterDatadogMetrics struct {
	Endpoint                           string `json:"endpoint,omitempty"`
	ResourceAttributesAsTags           bool   `json:"resource_attributes_as_tags,omitempty"`
	InstrumentationScopeMetadataAsTags bool   `json:"instrumentation_scope_metadata_as_tags,omitempty"`
}

type exporterDatadogLogs struct {
	Endpoint         string `json:"endpoint,omitempty"`
	UseCompression   *bool  `json:"use_compression,omitempty"`
	CompressionLevel int32  `json:"compression_level,omitempty"`
	BatchWait        string `json:"batch_wait,omitempty"`
}

type exporterDatadogHostMetadata struct {
	Enabled        *bool  `json:"enabled,omitempty"`
	ReporterPeriod string `json:"reporter_period,omitempty"`
}

type exporterLogzio struct {
	AccountToken string `json:"account_token,omitempty"`
	Region       string `json:"region,omitempty"`
	Endpoint     string `json:"endpoint,omitempty"`
	Timeout      string `json:"timeout,omitempty"`
}

type exporterKafka struct {
	Brokers         []string `json:"brokers,omitempty"`
	ProtocolVersion string   `json:"protocol_version,omitempty"`
	ClientID        string   `json:"client_id,omitempty"`

	Logs    *exporterKafkaSignal `json:"logs,omitempty"`
	Metrics *exporterKafkaSignal `json:"metrics,omitempty"`

	RecordHeaders []exporterKafkaHeader `json:"record_headers,omitempty"`

	Auth *exporterKafkaAuth `json:"auth,omitempty"`
	TLS  *exporterKafkaTLS  `json:"tls,omitempty"`

	Timeout         string `json:"timeout,omitempty"`
	ConnIdleTimeout string `json:"conn_idle_timeout,omitempty"`

	Producer *exporterKafkaProducer `json:"producer,omitempty"`

	PartitionLogsByResourceAttributes    bool `json:"partition_logs_by_resource_attributes,omitempty"`
	PartitionMetricsByResourceAttributes bool `json:"partition_metrics_by_resource_attributes,omitempty"`
}

type exporterKafkaSignal struct {
	Topic    string `json:"topic,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

type exporterKafkaHeader struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type exporterKafkaAuth struct {
	SASL *exporterKafkaSASL `json:"sasl,omitempty"`
}

type exporterKafkaSASL struct {
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	Mechanism string `json:"mechanism,omitempty"`
}

type exporterKafkaTLS struct {
	Insecure           bool `json:"insecure,omitempty"`
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

type exporterKafkaProducer struct {
	MaxMessageBytes        int64  `json:"max_message_bytes,omitempty"`
	RequiredAcks           int32  `json:"required_acks,omitempty"`
	Compression            string `json:"compression,omitempty"`
	FlushMaxMessages       int64  `json:"flush_max_messages,omitempty"`
	AllowAutoTopicCreation bool   `json:"allow_auto_topic_creation,omitempty"`
	Linger                 string `json:"linger,omitempty"`
}

type otelTLS struct {
	Insecure bool `json:"insecure,omitempty"`
}

type exporterSplunk struct {
	Token                   string     `json:"token,omitempty"`
	Endpoint                string     `json:"endpoint,omitempty"`
	Source                  string     `json:"source,omitempty"`
	SourceType              string     `json:"sourcetype,omitempty"`
	Index                   string     `json:"index,omitempty"`
	MaxIdleConns            int64      `json:"max_idle_conns,omitempty"`
	DisableCompression      bool       `json:"disable_compression,omitempty"`
	Timeout                 string     `json:"timeout,omitempty"`
	TLS                     *splunkTLS `json:"tls,omitempty"`
	SplunkAppName           string     `json:"splunk_app_name,omitempty"`
	SplunkAppVersion        string     `json:"splunk_app_version,omitempty"`
	UseMultiMetricFormat    bool       `json:"use_multi_metric_format,omitempty"`
	MaxContentLengthLogs    int64      `json:"max_content_length_logs,omitempty"`
	MaxContentLengthMetrics int64      `json:"max_content_length_metrics,omitempty"`
}

type splunkTLS struct {
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

type exporterAzureMonitor struct {
	ConnectionString       string `json:"connection_string,omitempty"`
	InstrumentationKey     string `json:"instrumentation_key,omitempty"`
	Endpoint               string `json:"endpoint,omitempty"`
	MaxBatchSize           int64  `json:"maxbatchsize,omitempty"`
	MaxBatchInterval       string `json:"maxbatchinterval,omitempty"`
	ShutdownTimeout        string `json:"shutdown_timeout,omitempty"`
	CustomEventsEnabled    bool   `json:"custom_events_enabled,omitempty"`
	ExceptionEventsEnabled bool   `json:"exception_events_enabled,omitempty"`
}

type exporterInfluxDB struct {
	Endpoint string `json:"endpoint,omitempty"`

	Org    string `json:"org,omitempty"`
	Bucket string `json:"bucket,omitempty"`
	Token  string `json:"token,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

	MetricsSchema string `json:"metrics_schema,omitempty"`
	Precision     string `json:"precision,omitempty"`

	PayloadMaxLines int64 `json:"payload_max_lines,omitempty"`
	PayloadMaxBytes int64 `json:"payload_max_bytes,omitempty"`

	Timeout string `json:"timeout,omitempty"`

	LogRecordDimensions []string `json:"log_record_dimensions,omitempty"`

	V1Compatibility *exporterInfluxDBV1Compatibility `json:"v1_compatibility,omitempty"`
}

type exporterInfluxDBV1Compatibility struct {
	Enabled  bool   `json:"enabled,omitempty"`
	DB       string `json:"db,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
