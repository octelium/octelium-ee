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
	Endpoints []string          `json:"endpoints,omitempty"`
	CloudID   string            `json:"cloudid,omitempty"`
	APIKey    string            `json:"api_key,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
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
	Brokers []string                   `json:"brokers,omitempty"`
	Logs    *exporterKafkaSignalConfig `json:"logs,omitempty"`
	Metrics *exporterKafkaSignalConfig `json:"metrics,omitempty"`
}

type exporterKafkaAuth struct {
}

type exporterKafkaSignalConfig struct {
	Topic    string `json:"topic,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

type otelTLS struct {
	Insecure bool `json:"insecure,omitempty"`
}

type otelAuth struct {
	Authenticator string `json:"authenticator,omitempty"`
}
