// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package clusterconfig

import (
	"context"
	"slices"
	"time"

	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/upgrade"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	gcomponents "github.com/octelium/octelium/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
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

	if !pbutils.IsEqual(new.Status, old.Status) && new.Status.UpgradeRequest != nil {
		c.update(new)
	}

	return nil
}

func (c *Controller) update(cluster *enterprisev1.ClusterConfig) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
		defer cancel()
		if err := c.doUpgrade(ctx, cluster); err != nil {
			zap.L().Warn("Could not doUpdate", zap.Error(err))
			if cluster.Status.UpgradeRequest != nil &&
				cluster.Status.UpgradeRequest.State == enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING {
				cluster.Status.UpgradeRequest.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
				cluster.Status.UpgradeRequest.DoneAt = pbutils.Now()

				cluster.Status.LastUpgradeRequests = slices.Insert(
					cluster.Status.LastUpgradeRequests, 0, cluster.Status.UpgradeRequest)
				cluster.Status.UpgradeRequest = nil
				cluster.Status.TotalFailedUpgrades += 1
				_, err := c.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cluster)
				if err != nil {
					zap.L().Warn("Could not updateCluster after update failure", zap.Error(err))
				}
			}
		}
	}()
}

func (c *Controller) doUpgrade(ctx context.Context, cluster *enterprisev1.ClusterConfig) error {

	if cluster.Status.UpgradeRequest == nil {
		return nil
	}

	req := cluster.Status.UpgradeRequest
	if req.State != enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED {
		zap.L().Debug("Not an UPGRADE_REQUESTED. Nothing to be done....", zap.Any("cluster", cluster))
		return nil
	}

	zap.L().Debug("Starting doUpgrade", zap.Any("cluster", cluster))

	cluster.Status.UpgradeRequest.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING
	cluster, err := c.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cluster)
	if err != nil {
		return err
	}

	{

		ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()

		if err := upgrade.UpgradeCluster(ctx, &upgrade.UpgradeClusterOpts{
			OcteliumC:      c.octeliumC,
			K8sC:           c.k8sC,
			UpgradeRequest: cluster.Status.UpgradeRequest,
		}); err != nil {
			zap.L().Warn("Could not upgradeCluster", zap.Error(err))
			cluster, err = c.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
			if err != nil {
				return err
			}
			cluster.Status.UpgradeRequest.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
			cluster.Status.UpgradeRequest.DoneAt = pbutils.Now()

			cluster.Status.LastUpgradeRequests = slices.Insert(
				cluster.Status.LastUpgradeRequests, 0, cluster.Status.UpgradeRequest)
			cluster.Status.UpgradeRequest = nil
			cluster.Status.TotalFailedUpgrades += 1
			_, err = c.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cluster)
			if err != nil {
				return err
			}

			return err
		}
	}

	// TODO: Check upgrade completed
	cluster, err = c.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	cluster.Status.UpgradeRequest.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS
	cluster.Status.UpgradeRequest.DoneAt = pbutils.Now()
	upgradeReq := cluster.Status.UpgradeRequest
	cluster.Status.LastUpgradeRequests = slices.Insert(
		cluster.Status.LastUpgradeRequests, 0, cluster.Status.UpgradeRequest)
	cluster.Status.UpgradeRequest = nil

	cluster.Status.TotalSuccessfulUpgrades += 1

	cluster, err = c.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cluster)
	if err != nil {
		return err
	}

	if upgradeReq.Request != nil && upgradeReq.Request.PackageEnterprise != nil {

		cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
		if err != nil {
			return err
		}

		rgn, err := c.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
			Name: vutils.GetMyRegionName(),
		})
		if err != nil {
			return err
		}
		if err := components.CreateClusterMan(ctx, &components.CommonOpts{
			CommonOpts: gcomponents.CommonOpts{
				K8sC:          c.k8sC,
				ClusterConfig: cc,
				Region:        rgn,
			},
			Version: upgradeReq.Request.PackageEnterprise.Version,
		}); err != nil {
			zap.L().Warn("Could not CreateClusterMan", zap.Error(err))
		}
	}

	zap.L().Debug("Completed doUpgrade", zap.Any("cluster", cluster))

	return nil
}
