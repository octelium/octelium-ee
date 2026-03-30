// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cluster_config

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	utils_types "github.com/octelium/octelium/pkg/utils/types"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
	k8sC      kubernetes.Interface
}

func NewController(
	octeliumC octeliumc.ClientInterface,
	k8sC kubernetes.Interface,
) *Controller {
	return &Controller{
		octeliumC: octeliumC,
		k8sC:      k8sC,
	}
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *enterprisev1.ClusterConfig) error {

	zap.L().Debug("ee CC updated", zap.Any("new", new))
	if err := c.updateCollectorReplicas(ctx, new, old); err != nil {
		return err
	}

	if err := c.updateOctovigilReplicas(ctx, new, old); err != nil {
		return err
	}

	if err := c.updateIngressReplicas(ctx, new, old); err != nil {
		return err
	}

	return nil
}

func (c *Controller) updateCollectorReplicas(ctx context.Context, new, old *enterprisev1.ClusterConfig) error {
	if pbutils.IsEqual(new.Spec.Scaler, old.Spec.Scaler) {
		return nil
	}

	replicas := func() int32 {
		if new.Spec.Scaler == nil || new.Spec.Scaler.Collector == nil || new.Spec.Scaler.Collector.Replicas <= 1 {
			return 1
		}
		return int32(new.Spec.Scaler.Collector.Replicas)
	}()

	return c.updateReplicasDeployment(ctx, "octeliumee-collector", replicas)
}

func (c *Controller) updateReplicasDeployment(ctx context.Context, name string, replicas int32) error {
	if replicas < 0 {
		return nil
	}

	depl, err := c.k8sC.AppsV1().Deployments(vutils.K8sNS).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if depl.Spec.Replicas != nil {
		if *depl.Spec.Replicas == replicas {
			return nil
		}
	} else {
		if replicas == 1 {
			return nil
		}
	}

	depl.Spec.Replicas = utils_types.Int32ToPtr(replicas)

	zap.L().Debug("Updating deployment replicas",
		zap.String("name", name), zap.Int32("replicas", replicas))

	if _, err := c.k8sC.AppsV1().Deployments(vutils.K8sNS).Update(ctx, depl, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Controller) updateOctovigilReplicas(ctx context.Context, new, old *enterprisev1.ClusterConfig) error {
	if pbutils.IsEqual(new.Spec.Scaler, old.Spec.Scaler) {
		return nil
	}

	replicas := func() int32 {
		if new.Spec.Scaler == nil || new.Spec.Scaler.Octovigil == nil || new.Spec.Scaler.Octovigil.Replicas <= 1 {
			return 1
		}
		return int32(new.Spec.Scaler.Octovigil.Replicas)
	}()

	return c.updateReplicasDeployment(ctx, "octelium-octovigil", replicas)
}

func (c *Controller) updateIngressReplicas(ctx context.Context, new, old *enterprisev1.ClusterConfig) error {
	if pbutils.IsEqual(new.Spec.Scaler, old.Spec.Scaler) {
		return nil
	}

	replicas := func() int32 {
		if new.Spec.Scaler == nil || new.Spec.Scaler.Ingress == nil || new.Spec.Scaler.Ingress.Replicas <= 1 {
			return 1
		}
		return int32(new.Spec.Scaler.Ingress.Replicas)
	}()

	return c.updateReplicasDeployment(ctx, "octelium-ingress-dataplane", replicas)
}
