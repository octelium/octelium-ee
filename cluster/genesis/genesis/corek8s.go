// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesis

import (
	"context"

	"github.com/octelium/octelium/cluster/common/components"
	appsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
)

var coreImageComponents = []string{
	components.RscServer,
	components.GWAgent,
	components.NodeInit,
	components.Nocturne,
	components.Octovigil,
	components.Ingress,
}

type coreK8sC struct {
	kubernetes.Interface
	version string
}

func newCoreK8sC(c kubernetes.Interface, version string) kubernetes.Interface {
	return &coreK8sC{
		Interface: c,
		version:   version,
	}
}

func (c *coreK8sC) AppsV1() appsv1client.AppsV1Interface {
	return &coreAppsV1{
		AppsV1Interface: c.Interface.AppsV1(),
		version:         c.version,
	}
}

type coreAppsV1 struct {
	appsv1client.AppsV1Interface
	version string
}

func (c *coreAppsV1) Deployments(namespace string) appsv1client.DeploymentInterface {
	return &coreDeployments{
		DeploymentInterface: c.AppsV1Interface.Deployments(namespace),
		version:             c.version,
	}
}

func (c *coreAppsV1) DaemonSets(namespace string) appsv1client.DaemonSetInterface {
	return &coreDaemonSets{
		DaemonSetInterface: c.AppsV1Interface.DaemonSets(namespace),
		version:            c.version,
	}
}

type coreDeployments struct {
	appsv1client.DeploymentInterface
	version string
}

func (c *coreDeployments) Create(ctx context.Context,
	itm *appsv1.Deployment, opts k8smetav1.CreateOptions) (*appsv1.Deployment, error) {
	itm = itm.DeepCopy()
	setCoreImages(&itm.Spec.Template.Spec, c.version)
	return c.DeploymentInterface.Create(ctx, itm, opts)
}

func (c *coreDeployments) Update(ctx context.Context,
	itm *appsv1.Deployment, opts k8smetav1.UpdateOptions) (*appsv1.Deployment, error) {
	itm = itm.DeepCopy()
	setCoreImages(&itm.Spec.Template.Spec, c.version)
	return c.DeploymentInterface.Update(ctx, itm, opts)
}

type coreDaemonSets struct {
	appsv1client.DaemonSetInterface
	version string
}

func (c *coreDaemonSets) Create(ctx context.Context,
	itm *appsv1.DaemonSet, opts k8smetav1.CreateOptions) (*appsv1.DaemonSet, error) {
	itm = itm.DeepCopy()
	setCoreImages(&itm.Spec.Template.Spec, c.version)
	return c.DaemonSetInterface.Create(ctx, itm, opts)
}

func (c *coreDaemonSets) Update(ctx context.Context,
	itm *appsv1.DaemonSet, opts k8smetav1.UpdateOptions) (*appsv1.DaemonSet, error) {
	itm = itm.DeepCopy()
	setCoreImages(&itm.Spec.Template.Spec, c.version)
	return c.DaemonSetInterface.Update(ctx, itm, opts)
}

func setCoreImages(spec *k8scorev1.PodSpec, version string) {
	if version == "" {
		return
	}

	for i := range spec.InitContainers {
		spec.InitContainers[i].Image = getCoreImage(spec.InitContainers[i].Image, version)
	}

	for i := range spec.Containers {
		spec.Containers[i].Image = getCoreImage(spec.Containers[i].Image, version)
	}
}

func getCoreImage(image string, version string) string {
	for _, component := range coreImageComponents {
		if image == components.GetImage(component, "") {
			return components.GetImage(component, version)
		}
	}

	return image
}
