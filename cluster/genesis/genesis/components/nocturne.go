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
	appsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getNocturneDeployment(o *CommonOpts) *appsv1.Deployment {

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentNocturne),
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentNocturne),
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      getComponentLabels(componentNocturne),
					Annotations: getAnnotations(),
				},
				Spec: k8scorev1.PodSpec{
					NodeSelector:       getNodeSelectorControlPlane(o.ClusterConfig),
					ServiceAccountName: getComponentName(componentNocturneRuntime),
					ImagePullSecrets:   getImagePullSecrets(),

					Containers: []k8scorev1.Container{
						{
							Name:            componentNocturne,
							Resources:       getDefaultResourceRequirements(),
							Image:           components.GetImage(components.Nocturne, ""),
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

func getNocturneRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: getComponentName(componentNocturne),
		},

		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*", "*.*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}
}

func getNocturneServiceAccount() *k8scorev1.ServiceAccount {
	return &k8scorev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentNocturne),
			Namespace: ns,
		},
	}
}

func getNocturneRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: getComponentName(componentNocturne),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     getComponentName(componentNocturne),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      getComponentName(componentNocturne),
				Namespace: ns,
			},
		},
	}
}

func getNocturneRuntimeServiceAccount() *k8scorev1.ServiceAccount {
	return &k8scorev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentNocturneRuntime),
			Namespace: ns,
		},
	}
}

func getNocturneRuntimeRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentNocturneRuntime),
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments"},
				ResourceNames: []string{
					getComponentName(componentCollector),
					"octelium-octovigil",
					"octelium-ingress-dataplane",
				},
				Verbs: []string{"get", "update"},
			},
		},
	}
}

func getNocturneRuntimeRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentNocturneRuntime),
			Namespace: ns,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     getComponentName(componentNocturneRuntime),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      getComponentName(componentNocturneRuntime),
				Namespace: ns,
			},
		},
	}
}

func getNocturneNetworkPolicy(c *corev1.ClusterConfig) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentNocturne),
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentNocturne),
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}
}

func CreateNocturne(ctx context.Context, o *CommonOpts) error {

	if _, err := k8sutils.CreateOrUpdateServiceAccount(ctx, o.K8sC, getNocturneRuntimeServiceAccount()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateRole(ctx, o.K8sC, getNocturneRuntimeRole()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateRoleBinding(ctx, o.K8sC, getNocturneRuntimeRoleBinding()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateServiceAccount(ctx, o.K8sC, getNocturneServiceAccount()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateClusterRole(ctx, o.K8sC, getNocturneRole()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateClusterRoleBinding(ctx, o.K8sC, getNocturneRoleBinding()); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateDeployment(ctx, o.K8sC, getNocturneDeployment(o)); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateNetworkPolicy(ctx, o.K8sC, getNocturneNetworkPolicy(o.ClusterConfig)); err != nil {
		return err
	}

	return nil
}
