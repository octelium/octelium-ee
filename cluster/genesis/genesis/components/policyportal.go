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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getPolicyPortalService() *k8scorev1.Service {

	ret := &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentPolicyPortal),
			Namespace: ns,
			Labels:    getComponentLabels(componentPolicyPortal),
		},
		Spec: k8scorev1.ServiceSpec{
			Type:     k8scorev1.ServiceTypeClusterIP,
			Selector: getComponentLabels(componentPolicyPortal),
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

func getPolicyPortalDeployment(o *CommonOpts) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentPolicyPortal),
			Namespace: ns,
			Labels:    getComponentLabels(componentPolicyPortal),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentPolicyPortal),
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      getComponentLabels(componentPolicyPortal),
					Annotations: getAnnotations(),
				},
				Spec: k8scorev1.PodSpec{
					AutomountServiceAccountToken: utils_types.BoolToPtr(false),
					NodeSelector:                 getNodeSelectorControlPlane(o.ClusterConfig),
					ImagePullSecrets:             getImagePullSecrets(),

					Containers: []k8scorev1.Container{
						{
							Name: componentPolicyPortal,

							Image:           components.GetImage(components.PolicyPortal, ""),
							ImagePullPolicy: k8sutils.GetImagePullPolicy(),
							SecurityContext: oc.MainSecurityContext(),

							Resources: k8scorev1.ResourceRequirements{
								Requests: getDefaultRequests(),
								Limits:   getDuckDBResourceLimit(),
							},

							LivenessProbe: getDefaultLivenessProbe(),
							VolumeMounts: []k8scorev1.VolumeMount{
								{
									MountPath: "/tmp",
									Name:      "tmpfs",
								},
							},
						},
					},

					Volumes: []k8scorev1.Volume{
						{
							Name: "tmpfs",
							VolumeSource: k8scorev1.VolumeSource{
								EmptyDir: &k8scorev1.EmptyDirVolumeSource{
									Medium: k8scorev1.StorageMediumMemory,
								},
							},
						},
					},
				},
			},
		},
	}
	oc.SetDeploymentSPIFFE(deployment, &o.CommonOpts)
	return deployment
}

func getPolicyPortalNetworkPolicy(c *corev1.ClusterConfig) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentPolicyPortal),
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentPolicyPortal),
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

func CreatePolicyPortal(ctx context.Context, o *CommonOpts) error {

	if _, err := k8sutils.CreateOrUpdateDeployment(ctx, o.K8sC, getPolicyPortalDeployment(o)); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateService(ctx, o.K8sC, getPolicyPortalService()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateNetworkPolicy(ctx, o.K8sC, getPolicyPortalNetworkPolicy(o.ClusterConfig)); err != nil {
		return err
	}

	return nil
}
