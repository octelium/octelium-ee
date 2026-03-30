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
)

func getCloudManDeployment(o *CommonOpts) *appsv1.Deployment {

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentCloudMan),
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentCloudMan),
			},
			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      getComponentLabels(componentCloudMan),
					Annotations: getAnnotations(),
				},
				Spec: k8scorev1.PodSpec{
					AutomountServiceAccountToken: utils_types.BoolToPtr(false),
					NodeSelector:                 getNodeSelectorControlPlane(o.ClusterConfig),
					ImagePullSecrets:             getImagePullSecrets(),

					Containers: []k8scorev1.Container{
						{
							Name:            componentCloudMan,
							Resources:       getDefaultResourceRequirements(),
							SecurityContext: oc.MainSecurityContext(),
							Image:           components.GetImage(components.CloudMan, ""),
							ImagePullPolicy: k8sutils.GetImagePullPolicy(),
							LivenessProbe:   getDefaultLivenessProbe(),
						},
					},
				},
			},
		},
	}
	oc.SetDeploymentSPIFFE(deployment, &o.CommonOpts)
	return deployment
}

func getCloudManNetworkPolicy(c *corev1.ClusterConfig) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getComponentName(componentCloudMan),
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: getComponentLabels(componentCloudMan),
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}
}

func CreateCloudMan(ctx context.Context, o *CommonOpts) error {

	if _, err := k8sutils.CreateOrUpdateDeployment(ctx, o.K8sC, getCloudManDeployment(o)); err != nil {
		return err
	}

	if _, err := k8sutils.CreateOrUpdateNetworkPolicy(ctx, o.K8sC, getCloudManNetworkPolicy(o.ClusterConfig)); err != nil {
		return err
	}

	return nil
}
