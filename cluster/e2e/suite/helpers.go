// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/pkg/errors"
)

func errNotProvisioned(arg string) error {
	return errors.Errorf("%s has not been provisioned yet", arg)
}

func errUnexpectedStatus(got, want int) error {
	return errors.Errorf("got status %d, want %d", got, want)
}

func errUnexpectedSyncState(state string) error {
	return errors.Errorf("the synchronization is %s", state)
}

func otlpExporter(endpoint, secretName string) *enterprisev1.CollectorExporter {
	spec := &enterprisev1.CollectorExporter_Spec_OTLP{
		Endpoint: endpoint,
		Tls: &enterprisev1.CollectorExporter_Spec_OTLP_TLS{
			Insecure: true,
		},
	}

	if secretName != "" {
		spec.Auth = &enterprisev1.CollectorExporter_Spec_OTLP_Auth{
			Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_{
				Bearer: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer{
					Type: &enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_FromSecret{
						FromSecret: secretName,
					},
				},
			},
		}
	}

	return &enterprisev1.CollectorExporter{
		Spec: &enterprisev1.CollectorExporter_Spec{
			Type: &enterprisev1.CollectorExporter_Spec_Otlp{Otlp: spec},
		},
	}
}

func otlpHTTPExporter(endpoint, secretName string) *enterprisev1.CollectorExporter {
	spec := &enterprisev1.CollectorExporter_Spec_OTLPHTTP{
		Endpoint: endpoint,
		Encoding: enterprisev1.CollectorExporter_Spec_OTLPHTTP_JSON,
		Tls: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_TLS{
			Insecure: true,
		},
	}

	if secretName != "" {
		spec.Auth = &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth{
			Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_{
				Bearer: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer{
					Type: &enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_FromSecret{
						FromSecret: secretName,
					},
				},
			},
		}
	}

	return &enterprisev1.CollectorExporter{
		Spec: &enterprisev1.CollectorExporter_Spec{
			Type: &enterprisev1.CollectorExporter_Spec_OtlpHTTP{OtlpHTTP: spec},
		},
	}
}

func logsPipeline(name string, exporters ...string) *enterprisev1.ClusterConfig_Spec_Collector_Pipeline {
	return &enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		Name:      name,
		Type:      enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS,
		Exporters: exporters,
	}
}

func metricsPipeline(name string, exporters ...string) *enterprisev1.ClusterConfig_Spec_Collector_Pipeline {
	return &enterprisev1.ClusterConfig_Spec_Collector_Pipeline{
		Name:      name,
		Type:      enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS,
		Exporters: exporters,
	}
}
