// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package collector

type exporterOTLP struct {
	Endpoint    string            `json:"endpoint,omitempty"`
	TLS         *otelTLS          `json:"tls,omitempty"`
	Compression string            `json:"compression,omitempty"`
	Auth        *otelAuth         `json:"auth,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

type exporterOTLPHTTP struct {
	Endpoint        string            `json:"endpoint,omitempty"`
	MetricsEndpoint string            `json:"metrics_endpoint,omitempty"`
	LogsEndpoint    string            `json:"logs_endpoint,omitempty"`
	TLS             *otelTLS          `json:"tls,omitempty"`
	Compression     string            `json:"compression,omitempty"`
	Encoding        string            `json:"encoding,omitempty"`
	Auth            *otelAuth         `json:"auth,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
}

type exporterPrometheusRemoteWriteRead struct {
	Endpoint  string            `json:"endpoint,omitempty"`
	Namespace string            `json:"namespace,omitempty"`
	TLS       *otelTLS          `json:"tls,omitempty"`
	Auth      *otelAuth         `json:"auth,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type exporterClickhouse struct {
	Endpoint string `json:"endpoint,omitempty"`
	Database string `json:"database,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
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
}

type exporterDatadogAPI struct {
	Key string `json:"key,omitempty"`
}

type exporterLogzio struct {
	Region       string `json:"region,omitempty"`
	Endpoint     string `json:"endpoint,omitempty"`
	AccountToken string `json:"account_token,omitempty"`
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

type otelAuth struct {
	Authenticator string `json:"authenticator,omitempty"`
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
