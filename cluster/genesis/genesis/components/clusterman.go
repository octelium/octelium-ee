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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getClusterManDeployment(o *CommonOpts) *appsv1.Deployment {

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentClusterMan),
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentClusterMan),
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      getComponentLabels(componentClusterMan),
					Annotations: getAnnotations(),
				},
				Spec: k8scorev1.PodSpec{
					NodeSelector:       getNodeSelectorControlPlane(o.ClusterConfig),
					ImagePullSecrets:   getImagePullSecrets(),
					ServiceAccountName: getComponentName(componentNocturne),
					Containers: []k8scorev1.Container{
						{
							Name:            componentClusterMan,
							Resources:       getDefaultResourceRequirements(),
							SecurityContext: oc.MainSecurityContext(),
							Image:           components.GetImage(components.ClusterMan, o.Version),
							ImagePullPolicy: k8sutils.GetImagePullPolicy(),
							Args:            []string{"run"},

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

func getClusterManNetworkPolicy(c *corev1.ClusterConfig) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentClusterMan),
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentClusterMan),
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}
}

func CreateClusterMan(ctx context.Context, o *CommonOpts) error {

	if _, err := k8sutils.CreateOrUpdateDeployment(ctx,
		o.K8sC, getClusterManDeployment(o)); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateNetworkPolicy(ctx,
		o.K8sC, getClusterManNetworkPolicy(o.ClusterConfig)); err != nil {
		return err
	}

	return nil
}
