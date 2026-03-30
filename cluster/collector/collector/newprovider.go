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

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/pkg/errors"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
)

type provider struct {
	// c          *configProvider
	schemeName string
	waitFn     confmap.WatcherFunc
	octeliumC  octeliumc.ClientInterface
	port       int
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
	if p.waitFn == nil {
		p.waitFn = waitFn
	}

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

func (p *provider) setConfig(ctx context.Context) error {

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
			Headers:         spec.Headers,
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

		spec := exp.Spec.GetElasticsearch()
		c := &exporterElasticsearch{
			Endpoints: spec.Endpoints,
			Headers:   spec.Headers,
			CloudID:   spec.CloudID,
		}
		ret.exp = c

		if spec.Auth != nil {
			if spec.Headers == nil {
				spec.Headers = make(map[string]string)
			}
			switch spec.Auth.Type.(type) {
			case *enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_ApiKey:
				if spec.Auth.GetApiKey().GetFromSecret() != "" {
					sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
						Name: spec.Auth.GetApiKey().GetFromSecret(),
					})
					if err != nil {
						return nil, err
					}

					c.APIKey = (uenterprisev1.ToSecret(sec).GetValueStr())
				}
			case *enterprisev1.CollectorExporter_Spec_Elasticsearch_Auth_Basic_:
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
			}
		}
	case *enterprisev1.CollectorExporter_Spec_Kafka_:

		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetKafka()

		c := &exporterKafka{
			Brokers: spec.Brokers,
		}
		ret.exp = c

	case *enterprisev1.CollectorExporter_Spec_Datadog_:
		ret.HasLogs = true
		ret.HasMetrics = true

		spec := exp.Spec.GetDatadog()

		c := &exporterDatadog{}
		ret.exp = c

		if spec.GetApiKey() != nil && spec.GetApiKey().GetFromSecret() != "" {
			sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
				Name: spec.GetApiKey().GetFromSecret(),
			})
			if err != nil {
				return nil, err
			}

			c.API = &exporterDatadogAPI{
				Key: uenterprisev1.ToSecret(sec).GetValueStr(),
			}
		}

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
			"receivers":  []string{"otlp/octelium"},
			"exporters":  exporters,
			"processors": []string{"batch", "memory_limiter"},
		}

		pipelinesMap[fmt.Sprintf("%s/%s", pipelineType, pipeline.Name)] = pipelineMap
	}

	zap.L().Debug("Successfully obtained provider config", zap.Any("cfg", ret))

	return ret, nil
}

func (c *provider) getInitConfig(ctx context.Context) (map[string]any, error) {

	return mt{
		"exporters": mt{
			"otlp/octelium-logs": mt{
				"endpoint": "octeliumee-logstore.octelium.svc:8080",
				"tls": mt{
					"insecure": true,
				},
			},
			"otlp/octelium-metrics": mt{
				"endpoint": "octeliumee-metricstore.octelium.svc:8080",
				"tls": mt{
					"insecure": true,
				},
			},
		},
		"extensions": mt{},
		"receivers": mt{
			"otlp/octelium": mt{
				"protocols": mt{
					"grpc": mt{
						"endpoint": fmt.Sprintf(":%d", c.port),
					},
				},
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
					"receivers":  []string{"otlp/octelium"},
					"processors": []string{"batch", "memory_limiter"},
					"exporters":  []string{"otlp/octelium-logs"},
				},
				"metrics": mt{
					"receivers":  []string{"otlp/octelium"},
					"processors": []string{"batch", "memory_limiter"},
					"exporters":  []string{"otlp/octelium-metrics"},
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

func (cm *provider) sendUpdate() {

	if cm.waitFn != nil {
		zap.L().Debug("Config provider sending update...")
		cm.waitFn(&confmap.ChangeEvent{})
	}
}
