// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package components

import (
	"context"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/cluster/common/vutils"
	gcomponents "github.com/octelium/octelium/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
)

const componentNocturne = components.Nocturne
const componentCloudMan = components.CloudMan
const componentCollector = components.Collector
const componentAPIServer = components.APIServer
const componentRscServer = components.RscServer
const componentSecretMan = components.SecretMan
const componentClusterMan = components.ClusterMan
const componentLogStore = components.LogStore
const componentRscStore = components.RscStore
const componentMetricStore = components.MetricStore
const componentPolicyPortal = components.PolicyPortal

const componentIngressDataPlane = "ingress-dataplane"

func defaultAnnotations() map[string]string {

	ret := make(map[string]string)
	ret["octelium.com/last-touched"] = time.Now().Format(time.RFC3339)

	return ret
}

func getNodeSelectorDataPlane(c *corev1.ClusterConfig) map[string]string {
	return map[string]string{
		"octelium.com/node-mode-dataplane": "",
	}
}

func getNodeSelectorControlPlane(c *corev1.ClusterConfig) map[string]string {
	return map[string]string{
		"octelium.com/node-mode-controlplane": "",
	}
}

var tcpProtocol = k8scorev1.ProtocolTCP
var udpProtocol = k8scorev1.ProtocolUDP

func getImagePullSecrets() []k8scorev1.LocalObjectReference {
	if ovutils.IsPrivateRegistry() {
		return []k8scorev1.LocalObjectReference{
			{
				Name: "octelium-regcred",
			},
		}
	} else {
		return nil
	}
}

func getComponentName(arg string) string {
	return components.OcteliumEnterpriseComponent(arg)
}

const ns = vutils.K8sNS

func getComponentLabels(arg string) map[string]string {
	return map[string]string{
		"app":                         "octelium",
		"octelium.com/app":            "octeliumee",
		"octelium.com/component":      arg,
		"octelium.com/component-type": "cluster",
	}
}

func getAnnotations() map[string]string {
	return map[string]string{
		"octelium.com/install-uid": utilrand.GetRandomStringLowercase(8),
	}
}

func InstallCommon(ctx context.Context, c kubernetes.Interface,
	clusterCfg *corev1.ClusterConfig, r *corev1.Region) error {

	return nil
}

func getDefaultRequests() k8scorev1.ResourceList {
	return k8scorev1.ResourceList{
		k8scorev1.ResourceMemory: resource.MustParse("5Mi"),
		k8scorev1.ResourceCPU:    resource.MustParse("10m"),
	}
}

func getDefaultLimits() k8scorev1.ResourceList {
	return k8scorev1.ResourceList{
		k8scorev1.ResourceMemory: resource.MustParse("700Mi"),
		k8scorev1.ResourceCPU:    resource.MustParse("1200m"),
	}
}

func getDefaultResourceRequirements() k8scorev1.ResourceRequirements {
	return k8scorev1.ResourceRequirements{
		Requests: getDefaultRequests(),
		Limits:   getDefaultLimits(),
	}
}

func getDefaultLivenessProbe() *k8scorev1.Probe {
	return gcomponents.MainLivenessProbe()
}

func getDuckDBResourceLimit() k8scorev1.ResourceList {
	return k8scorev1.ResourceList{
		k8scorev1.ResourceMemory: resource.MustParse("3000Mi"),
		k8scorev1.ResourceCPU:    resource.MustParse("4000m"),
	}
}

type CommonOpts struct {
	gcomponents.CommonOpts
	Version string
}
