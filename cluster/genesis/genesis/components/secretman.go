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
	gc "github.com/octelium/octelium/cluster/genesis/genesis/components"
	oc "github.com/octelium/octelium/cluster/genesis/genesis/components"
	appsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getSecretManService() *k8scorev1.Service {

	ret := &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentSecretMan),
			Namespace: ns,
			Labels:    getComponentLabels(componentSecretMan),
		},
		Spec: k8scorev1.ServiceSpec{
			Type:     k8scorev1.ServiceTypeClusterIP,
			Selector: getComponentLabels(componentSecretMan),
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

func getSecretManDeployment(o *CommonOpts) *appsv1.Deployment {

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentSecretMan),
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentSecretMan),
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      getComponentLabels(componentSecretMan),
					Annotations: getAnnotations(),
				},
				Spec: k8scorev1.PodSpec{
					ServiceAccountName: getComponentName(componentSecretMan),
					NodeSelector:       getNodeSelectorControlPlane(o.ClusterConfig),
					ImagePullSecrets:   getImagePullSecrets(),

					Volumes: []k8scorev1.Volume{
						{
							Name: "sys-init-kek",
							VolumeSource: k8scorev1.VolumeSource{
								Secret: &k8scorev1.SecretVolumeSource{
									SecretName: "sys-init-kek",
								},
							},
						},
					},

					Containers: []k8scorev1.Container{
						{
							Name:            componentSecretMan,
							Image:           components.GetImage(components.SecretMan, ""),
							ImagePullPolicy: k8sutils.GetImagePullPolicy(),
							SecurityContext: oc.MainSecurityContext(),
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

								ret = append(ret, gc.GetPostgresEnv()...)

								return ret
							}(),
							VolumeMounts: []k8scorev1.VolumeMount{
								{
									Name:      "sys-init-kek",
									MountPath: "/octelium-kek",
									ReadOnly:  true,
									SubPath:   "data",
								},
							},
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

func getSecretManNetworkPolicy(c *corev1.ClusterConfig) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentSecretMan),
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentSecretMan),
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
									"app":                         "octelium",
									"octelium.com/component":      componentRscServer,
									"octelium.com/component-type": "cluster",
								},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": ns,
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

func getSecretManRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: getComponentName(componentSecretMan),
		},

		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "update", "watch", "list", "delete"},
			},
		},
	}
}

func getSecretManServiceAccount() *k8scorev1.ServiceAccount {
	return &k8scorev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentSecretMan),
			Namespace: ns,
		},
	}
}

func getSecretManRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: getComponentName(componentSecretMan),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     getComponentName(componentSecretMan),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      getComponentName(componentSecretMan),
				Namespace: ns,
			},
		},
	}
}

func CreateSecretMan(ctx context.Context, o *CommonOpts) error {

	if _, err := k8sutils.CreateOrUpdateServiceAccount(ctx, o.K8sC, getSecretManServiceAccount()); err != nil {
		return err
	}

	/*


		if _, err := k8sutils.CreateOrUpdateClusterRole(ctx, c, getSecretManRole()); err != nil {
			return err
		}

		if _, err := k8sutils.CreateOrUpdateClusterRoleBinding(ctx, c, getSecretManRoleBinding()); err != nil {
			return err
		}
	*/

	if _, err := k8sutils.CreateOrUpdateDeployment(ctx, o.K8sC, getSecretManDeployment(o)); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateService(ctx, o.K8sC, getSecretManService()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateNetworkPolicy(ctx, o.K8sC, getSecretManNetworkPolicy(o.ClusterConfig)); err != nil {
		return err
	}

	return nil
}
