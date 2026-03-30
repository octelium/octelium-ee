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

func getRscStoreService() *k8scorev1.Service {

	ret := &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentRscStore),
			Namespace: ns,
			Labels:    getComponentLabels(componentRscStore),
		},
		Spec: k8scorev1.ServiceSpec{
			Type:     k8scorev1.ServiceTypeClusterIP,
			Selector: getComponentLabels(componentRscStore),
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

func getRscStoreDeployment(o *CommonOpts) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentRscStore),
			Namespace: ns,
			Labels:    getComponentLabels(componentRscStore),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentRscStore),
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      getComponentLabels(componentRscStore),
					Annotations: getAnnotations(),
				},
				Spec: k8scorev1.PodSpec{
					AutomountServiceAccountToken: utils_types.BoolToPtr(false),
					NodeSelector:                 getNodeSelectorControlPlane(o.ClusterConfig),
					ImagePullSecrets:             getImagePullSecrets(),

					Containers: []k8scorev1.Container{
						{
							Name: componentRscStore,

							Image:           components.GetImage(components.RscStore, ""),
							ImagePullPolicy: k8sutils.GetImagePullPolicy(),

							Resources: k8scorev1.ResourceRequirements{
								Requests: getDefaultRequests(),
								Limits:   getDuckDBResourceLimit(),
							},
							VolumeMounts: []k8scorev1.VolumeMount{
								{
									Name:      "octelium",
									MountPath: "/octelium-data",
								},
							},
							LivenessProbe:   getDefaultLivenessProbe(),
							SecurityContext: oc.MainSecurityContext(),
						},
					},
					Volumes: []k8scorev1.Volume{
						{
							Name: "octelium",
							VolumeSource: k8scorev1.VolumeSource{
								PersistentVolumeClaim: &k8scorev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "octelium-rscstore",
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

func getRscStoreNetworkPolicy(c *corev1.ClusterConfig) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentRscStore),
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentRscStore),
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

func CreateRscStore(ctx context.Context, o *CommonOpts) error {

	if _, err := k8sutils.CreateOrUpdateDeployment(ctx, o.K8sC, getRscStoreDeployment(o)); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateService(ctx, o.K8sC, getRscStoreService()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateNetworkPolicy(ctx, o.K8sC, getRscStoreNetworkPolicy(o.ClusterConfig)); err != nil {
		return err
	}

	return nil
}
