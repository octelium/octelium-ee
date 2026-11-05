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
	"time"

	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/regionk8s"
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

const maxUpgradeDuration = 24 * time.Hour

const (
	recoverTimeout       = 5 * time.Minute
	recoverRetryInterval = 2 * time.Second

	interruptedRequestMessage = "Cluster Manager restarted before this request finished. " +
		"Its genesis Job may still be running. Check the Cluster before requesting it again"
)

type Controller struct {
	octeliumC     octeliumc.ClientInterface
	k8sC          kubernetes.Interface
	newRegionK8sC regionk8s.NewClientFn
}

func NewController(
	octeliumC octeliumc.ClientInterface,
	k8sC kubernetes.Interface,
) *Controller {
	return &Controller{
		octeliumC:     octeliumC,
		k8sC:          k8sC,
		newRegionK8sC: regionk8s.NewClientFactory(octeliumC),
	}
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *enterprisev1.ClusterConfig) error {

	if !pbutils.IsEqual(new.Status, old.Status) && new.Status.UpgradeRequest != nil {
		c.update(new)
	}

	if !pbutils.IsEqual(new.Status, old.Status) && new.Status.JoinRequest != nil {
		c.join(new)
	}

	return nil
}

func (c *Controller) RecoverInterrupted(ctx context.Context) {
	go func() {
		ctx, cancel := context.WithTimeout(ctx, recoverTimeout)
		defer cancel()

		for {
			err := c.doRecoverInterrupted(ctx)
			if err == nil {
				return
			}

			zap.L().Warn("Could not record the interrupted requests", zap.Error(err))

			select {
			case <-ctx.Done():
				return
			case <-time.After(recoverRetryInterval):
			}
		}
	}()
}

func (c *Controller) doRecoverInterrupted(ctx context.Context) error {
	return c.updateStatus(ctx, func(cc *enterprisev1.ClusterConfig) bool {
		isUpgrade := failInterruptedUpgrade(cc)
		isJoin := failInterruptedJoin(cc)
		return isUpgrade || isJoin
	})
}

func failInterruptedUpgrade(cc *enterprisev1.ClusterConfig) bool {
	req := cc.Status.UpgradeRequest
	if req == nil || req.State != enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING {
		return false
	}

	zap.L().Warn("Recording the upgrade interrupted by a restart as failed", zap.Any("request", req))

	now := pbutils.Now()
	req.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
	req.DoneAt = now

	for _, rgn := range req.Regions {
		switch rgn.State {
		case enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS,
			enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED:
		default:
			rgn.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
			rgn.DoneAt = now
			rgn.Message = interruptedRequestMessage
		}
	}

	cc.Status.LastUpgradeRequests = prependUpgradeHistory(cc.Status.LastUpgradeRequests, req)
	cc.Status.UpgradeRequest = nil
	cc.Status.TotalFailedUpgrades += 1

	return true
}

func failInterruptedJoin(cc *enterprisev1.ClusterConfig) bool {
	req := cc.Status.JoinRequest
	if req == nil || req.State != enterprisev1.ClusterConfig_Status_JoinRequest_JOINING {
		return false
	}

	zap.L().Warn("Recording the join interrupted by a restart as failed", zap.Any("request", req))

	req.State = enterprisev1.ClusterConfig_Status_JoinRequest_FAILED
	req.DoneAt = pbutils.Now()
	req.Message = interruptedRequestMessage

	cc.Status.LastJoinRequests = prependJoinHistory(cc.Status.LastJoinRequests, req)
	cc.Status.JoinRequest = nil

	return true
}

func (c *Controller) update(cluster *enterprisev1.ClusterConfig) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), maxUpgradeDuration)
		defer cancel()
		if err := c.doUpgrade(ctx, cluster); err != nil {
			zap.L().Warn("Could not doUpdate", zap.Error(err))
			cluster, err = c.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
			if err != nil {
				zap.L().Warn("Could not get ClusterConfig after update failure", zap.Error(err))
				return
			}

			if cluster.Status.UpgradeRequest != nil &&
				cluster.Status.UpgradeRequest.State == enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING {
				cluster.Status.UpgradeRequest.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
				cluster.Status.UpgradeRequest.DoneAt = pbutils.Now()

				cluster.Status.LastUpgradeRequests = prependUpgradeHistory(
					cluster.Status.LastUpgradeRequests, cluster.Status.UpgradeRequest)

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

const maxLastUpgradeRequests = 200

func prependUpgradeHistory(
	history []*enterprisev1.ClusterConfig_Status_UpgradeRequest,
	req *enterprisev1.ClusterConfig_Status_UpgradeRequest,
) []*enterprisev1.ClusterConfig_Status_UpgradeRequest {
	if req == nil {
		return history
	}

	history = append([]*enterprisev1.ClusterConfig_Status_UpgradeRequest{req}, history...)
	if len(history) > maxLastUpgradeRequests {
		history = history[:maxLastUpgradeRequests]
	}

	return history
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

	started := cluster.Status.UpgradeRequest

	{
		if err := upgrade.UpgradeCluster(ctx, &upgrade.UpgradeClusterOpts{
			OcteliumC:      c.octeliumC,
			K8sC:           c.k8sC,
			UpgradeRequest: started,
			NewRegionK8sC:  c.newRegionK8sC,
			OnRegions: func(ctx context.Context, regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region) {
				c.setUpgradeRegions(ctx, started, regions)
			},
		}); err != nil {
			zap.L().Warn("Could not upgradeCluster", zap.Error(err))
			cluster, err = c.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
			if err != nil {
				return err
			}
			if !isSameUpgradeRequest(cluster.Status.UpgradeRequest, started) {
				zap.L().Debug("The upgrade request was already recorded. Nothing to be done...")
				return nil
			}
			cluster.Status.UpgradeRequest.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
			cluster.Status.UpgradeRequest.DoneAt = pbutils.Now()

			cluster.Status.LastUpgradeRequests = prependUpgradeHistory(
				cluster.Status.LastUpgradeRequests, cluster.Status.UpgradeRequest)
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

	if !isSameUpgradeRequest(cluster.Status.UpgradeRequest, started) {
		zap.L().Debug("The upgrade request was already recorded. Nothing to be done...")
		return nil
	}

	cluster.Status.UpgradeRequest.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS
	cluster.Status.UpgradeRequest.DoneAt = pbutils.Now()
	upgradeReq := cluster.Status.UpgradeRequest
	cluster.Status.LastUpgradeRequests = prependUpgradeHistory(
		cluster.Status.LastUpgradeRequests, cluster.Status.UpgradeRequest)
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
