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

	"github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	oc "github.com/octelium/octelium/cluster/genesis/genesis/components"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	appsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getCollectorService() *k8scorev1.Service {

	ret := &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "octelium-collector",
			Namespace: ns,
			Labels:    getComponentLabels(componentCollector),
		},
		Spec: k8scorev1.ServiceSpec{
			Type:     k8scorev1.ServiceTypeClusterIP,
			Selector: getComponentLabels(componentCollector),
			Ports: []k8scorev1.ServicePort{
				{
					Protocol: k8scorev1.ProtocolTCP,
					Port:     8080,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8080,
					},
				},
			},
		},
	}

	return ret
}

func getCollectorDeployment(o *CommonOpts) *appsv1.Deployment {

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentCollector),
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentCollector),
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      getComponentLabels(componentCollector),
					Annotations: getAnnotations(),
				},
				Spec: k8scorev1.PodSpec{
					AutomountServiceAccountToken: utils_types.BoolToPtr(false),
					NodeSelector:                 getNodeSelectorControlPlane(o.ClusterConfig),
					ImagePullSecrets:             getImagePullSecrets(),

					Containers: []k8scorev1.Container{
						{
							Name:            componentCollector,
							Image:           components.GetImage(components.Collector, ""),
							ImagePullPolicy: k8sutils.GetImagePullPolicy(),
							SecurityContext: oc.MainSecurityContext(),
							Resources: k8scorev1.ResourceRequirements{
								Requests: getDefaultRequests(),
								Limits: k8scorev1.ResourceList{
									k8scorev1.ResourceMemory: resource.MustParse("3000Mi"),
									k8scorev1.ResourceCPU:    resource.MustParse("4000m"),
								},
							},
							Env: func() []k8scorev1.EnvVar {
								ret := []k8scorev1.EnvVar{
									{
										Name:  "OCTELIUM_REGION_NAME",
										Value: o.Region.Metadata.Name,
									},
									{
										Name:  "OCTELIUM_REGION_UID",
										Value: o.Region.Metadata.Uid,
									},
								}

								return ret
							}(),
							LivenessProbe: getDefaultLivenessProbe(),
						},
					},
				},
			},
		},
	}

	oc.SetDeploymentSPIFFE(deployment, &o.CommonOpts)
	return deployment
}

func getCollectorNetworkPolicy(c *corev1.ClusterConfig) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentCollector),
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentCollector),
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &tcpProtocol,
							Port: &intstr.IntOrString{
								IntVal: 8080,
							},
						},
					},
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": "octelium",
								},
							},
						},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}
}

func CreateCollector(ctx context.Context, o *CommonOpts) error {

	if _, err := k8sutils.CreateOrUpdateDeployment(ctx, o.K8sC, getCollectorDeployment(o)); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateService(ctx, o.K8sC, getCollectorService()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateNetworkPolicy(ctx, o.K8sC, getCollectorNetworkPolicy(o.ClusterConfig)); err != nil {
		return err
	}

	return nil
}
