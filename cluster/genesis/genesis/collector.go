// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesis

import (
	"fmt"

	"github.com/octelium/octelium-ee/cluster/common/collectorutils"
	oc "github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
)

const collectorMetricStoreConfigName = "metricstore"

func getCollectorExportCondition() string {
	return fmt.Sprintf(
		`ctx.user.metadata.name == "octelium" && ctx.request.grpc.method == "%s" && ctx.request.grpc.serviceFullName in ["%s", "%s"]`,
		collectorutils.ExportMethod,
		collectorutils.LogsServiceFullName,
		collectorutils.MetricsServiceFullName)
}

func getCollectorUpstream(component string) *corev1.Service_Spec_Config_Upstream {
	return &corev1.Service_Spec_Config_Upstream{
		Type: &corev1.Service_Spec_Config_Upstream_Url{
			Url: collectorutils.GetUpstreamURL(
				fmt.Sprintf("%s.%s.svc", oc.OcteliumEnterpriseComponent(component), vutils.K8sNS)),
		},
	}
}

func getCollectorService() *corev1.Service {
	return &corev1.Service{
		Metadata: &metav1.Metadata{
			Name:         collectorutils.ServiceName,
			IsSystem:     true,
			IsUserHidden: true,
			DisplayName:  "Octelium Collector",
			Description:  "This Service receives the logs and metrics of the collectors of the non-default Regions",
		},
		Spec: &corev1.Service_Spec{
			Port:     8080,
			IsPublic: true,
			Mode:     corev1.Service_Spec_GRPC,
			Config: &corev1.Service_Spec_Config{
				Upstream: getCollectorUpstream(oc.LogStore),
			},
			DynamicConfig: &corev1.Service_Spec_DynamicConfig{
				Configs: []*corev1.Service_Spec_Config{
					{
						Name:     collectorMetricStoreConfigName,
						Upstream: getCollectorUpstream(oc.MetricStore),
					},
				},
				Rules: []*corev1.Service_Spec_DynamicConfig_Rule{
					{
						Condition: &corev1.Condition{
							Type: &corev1.Condition_Match{
								Match: fmt.Sprintf(`ctx.request.grpc.serviceFullName == "%s"`,
									collectorutils.MetricsServiceFullName),
							},
						},
						Type: &corev1.Service_Spec_DynamicConfig_Rule_ConfigName{
							ConfigName: collectorMetricStoreConfigName,
						},
					},
				},
			},
			Authorization: &corev1.Service_Spec_Authorization{
				InlinePolicies: []*corev1.InlinePolicy{
					{
						Spec: &corev1.Policy_Spec{
							Rules: []*corev1.Policy_Spec_Rule{
								{
									Effect:   corev1.Policy_Spec_Rule_DENY,
									Priority: -4,
									Condition: &corev1.Condition{
										Type: &corev1.Condition_Not{
											Not: getCollectorExportCondition(),
										},
									},
								},
								{
									Effect: corev1.Policy_Spec_Rule_ALLOW,
									Condition: &corev1.Condition{
										Type: &corev1.Condition_Match{
											Match: getCollectorExportCondition(),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
