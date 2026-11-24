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
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/genesisjob"
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
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/client-go/kubernetes"
)

const maxUpgradeDuration = 24 * time.Hour

const (
	reconcileInterval      = 30 * time.Second
	reconcileRetryInterval = 2 * time.Second
	recordTimeout          = time.Hour
	reportTimeout          = time.Minute
	clusterManAttempts     = 8

	interruptedRequestMessage = "Cluster Manager restarted before this request finished. " +
		"Its genesis Job may still be running. Check the Cluster before requesting it again"
	supersededRequestMessage = "A newer request replaced this request before it finished. " +
		"Check the Cluster before requesting it again"
)

const (
	taskUpgrade = "upgrade"
	taskJoin    = "join"
)

type Controller struct {
	octeliumC     octeliumc.ClientInterface
	k8sC          kubernetes.Interface
	newRegionK8sC regionk8s.NewClientFn

	mu    sync.Mutex
	tasks map[string]*timestamppb.Timestamp
}

func NewController(
	octeliumC octeliumc.ClientInterface,
	k8sC kubernetes.Interface,
) *Controller {
	return &Controller{
		octeliumC:     octeliumC,
		k8sC:          k8sC,
		newRegionK8sC: regionk8s.NewClientFactory(octeliumC),
		tasks:         make(map[string]*timestamppb.Timestamp),
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

func (c *Controller) RunReconciler(ctx context.Context) {
	go func() {
		retryInterval := reconcileRetryInterval

		for {
			interval := reconcileInterval
			if err := c.doReconcile(ctx); err != nil {
				zap.L().Warn("Could not reconcile the Cluster requests", zap.Error(err))
				interval = retryInterval
				retryInterval = min(2*retryInterval, reconcileInterval)
			} else {
				retryInterval = reconcileRetryInterval
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}()
}

func (c *Controller) doReconcile(ctx context.Context) error {
	cc, err := c.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	if cc.Status == nil {
		return nil
	}

	upgradeErr := c.reconcileUpgrade(ctx, cc)
	joinErr := c.reconcileJoin(ctx, cc)

	if upgradeErr != nil {
		return upgradeErr
	}

	return joinErr
}

func (c *Controller) reconcileUpgrade(ctx context.Context, cc *enterprisev1.ClusterConfig) error {
	for i, req := range cc.Status.LastUpgradeRequests {
		if upgrade.IsDone(req.State) || c.isOwned(taskUpgrade, req.CreatedAt) {
			continue
		}

		before := cc.Status.UpgradeRequest.GetCreatedAt()
		if i > 0 {
			before = cc.Status.LastUpgradeRequests[i-1].CreatedAt
		}

		zap.L().Info("Recording the replaced upgrade request", zap.Any("request", req))

		regions, err := upgrade.GetRegionResults(ctx, c.octeliumC, req, before)
		if err != nil {
			return err
		}

		if _, err := c.recordUpgrade(ctx, req,
			completeUpgradeRegions(regions, supersededRequestMessage), isUpgradeSucceeded(regions)); err != nil {
			return err
		}
	}

	req := cc.Status.UpgradeRequest
	if req == nil || c.isBusy(taskUpgrade) {
		return nil
	}

	switch req.State {
	case enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED:
		zap.L().Info("Starting the pending upgrade request", zap.Any("request", req))
		c.update(cc)
		return nil
	case enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING:
		return c.recoverUpgrade(ctx, req)
	default:
		return nil
	}
}

func (c *Controller) recoverUpgrade(ctx context.Context, req *enterprisev1.ClusterConfig_Status_UpgradeRequest) error {
	regions, err := upgrade.GetRegionResults(ctx, c.octeliumC, req, nil)
	if err != nil {
		return err
	}

	isSuccess := isUpgradeSucceeded(regions)
	if !isSuccess {
		isActive, err := genesisjob.HasActive(ctx, c.k8sC)
		if err != nil {
			return err
		}

		if isActive {
			zap.L().Debug("Waiting for the running genesis Job before recording the interrupted upgrade")
			return nil
		}
	}

	zap.L().Info("Recording the interrupted upgrade request",
		zap.Any("request", req), zap.Bool("success", isSuccess))

	_, err = c.recordUpgrade(ctx, req, completeUpgradeRegions(regions, interruptedRequestMessage), isSuccess)
	return err
}

func isUpgradeSucceeded(regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region) bool {
	if len(regions) == 0 {
		return false
	}

	for _, rgn := range regions {
		if rgn.State != enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS {
			return false
		}
	}

	return true
}

func completeUpgradeRegions(regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region,
	msg string) []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region {
	now := pbutils.Now()
	for _, rgn := range regions {
		if upgrade.IsDone(rgn.State) {
			continue
		}

		rgn.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
		rgn.DoneAt = now
		rgn.Message = msg
	}

	return regions
}

func (c *Controller) update(cluster *enterprisev1.ClusterConfig) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), maxUpgradeDuration)
		defer cancel()
		if err := c.doUpgrade(ctx, cluster); err != nil {
			zap.L().Warn("Could not doUpdate", zap.Error(err))
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

	if !c.acquire(taskUpgrade, req.CreatedAt) {
		zap.L().Debug("Another upgrade request is being handled. Nothing to be done...")
		return nil
	}
	defer c.release(taskUpgrade)

	isStarted := false
	if err := c.updateStatus(ctx, func(cc *enterprisev1.ClusterConfig) bool {
		cur := cc.Status.UpgradeRequest
		if !isSameUpgradeRequest(cur, req) {
			isStarted = false
			return false
		}

		if isStarted && cur.State == enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING {
			return false
		}

		isStarted = cur.State == enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADE_REQUESTED
		if isStarted {
			cur.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING
		}

		return isStarted
	}); err != nil {
		return err
	}

	if !isStarted {
		zap.L().Debug("The upgrade request is already handled. Nothing to be done...")
		return nil
	}

	zap.L().Debug("Starting doUpgrade", zap.Any("request", req))

	var regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region
	upgradeErr := upgrade.UpgradeCluster(ctx, &upgrade.UpgradeClusterOpts{
		OcteliumC:      c.octeliumC,
		K8sC:           c.k8sC,
		UpgradeRequest: req,
		NewRegionK8sC:  c.newRegionK8sC,
		OnRegions: func(ctx context.Context, rgns []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region) {
			regions = rgns
			c.setUpgradeRegions(ctx, req, rgns)
		},
	})
	if upgradeErr != nil {
		zap.L().Warn("Could not upgradeCluster", zap.Error(upgradeErr))
	}

	recordCtx, cancel := context.WithTimeout(context.Background(), recordTimeout)
	defer cancel()

	isRecorded, err := c.recordUpgrade(recordCtx, req, regions, upgradeErr == nil)
	if err != nil {
		return err
	}

	if !isRecorded {
		zap.L().Debug("The upgrade request was already recorded. Nothing to be done...")
		return nil
	}

	if upgradeErr == nil && req.Request.GetPackageEnterprise() != nil {
		if err := c.upgradeClusterMan(recordCtx, req.Request.PackageEnterprise.Version); err != nil {
			return err
		}
	}

	zap.L().Debug("Completed doUpgrade", zap.Any("request", req), zap.Bool("success", upgradeErr == nil))

	return nil
}

func (c *Controller) recordUpgrade(ctx context.Context,
	req *enterprisev1.ClusterConfig_Status_UpgradeRequest,
	regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region, isSuccess bool) (bool, error) {
	isRecorded := false

	err := c.updateStatus(ctx, func(cc *enterprisev1.ClusterConfig) bool {
		isRecorded = false

		cur := cc.Status.UpgradeRequest
		isCurrent := isSameUpgradeRequest(cur, req)
		if !isCurrent {
			cur = findUpgradeRequest(cc.Status.LastUpgradeRequests, req)
		}

		if cur == nil || upgrade.IsDone(cur.State) {
			return false
		}

		if len(regions) > 0 {
			cur.Regions = regions
		}

		cur.DoneAt = pbutils.Now()
		if isSuccess {
			cur.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS
			cc.Status.TotalSuccessfulUpgrades += 1
		} else {
			cur.State = enterprisev1.ClusterConfig_Status_UpgradeRequest_FAILED
			cc.Status.TotalFailedUpgrades += 1
		}

		if isCurrent {
			cc.Status.LastUpgradeRequests = prependUpgradeHistory(cc.Status.LastUpgradeRequests, cur)
			cc.Status.UpgradeRequest = nil
		}

		isRecorded = true
		return true
	})

	return isRecorded, err
}

func findUpgradeRequest(history []*enterprisev1.ClusterConfig_Status_UpgradeRequest,
	req *enterprisev1.ClusterConfig_Status_UpgradeRequest) *enterprisev1.ClusterConfig_Status_UpgradeRequest {
	for _, itm := range history {
		if isSameUpgradeRequest(itm, req) {
			return itm
		}
	}

	return nil
}

func (c *Controller) upgradeClusterMan(ctx context.Context, version string) error {
	var err error
	interval := statusRetryInterval

	for range clusterManAttempts {
		if err = c.doUpgradeClusterMan(ctx, version); err == nil {
			return nil
		}

		zap.L().Warn("Could not CreateClusterMan. Trying again...", zap.Error(err))

		select {
		case <-ctx.Done():
			return err
		case <-time.After(interval):
		}

		interval = min(2*interval, maxStatusRetryInterval)
	}

	return err
}

func (c *Controller) doUpgradeClusterMan(ctx context.Context, version string) error {
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

	return components.CreateClusterMan(ctx, &components.CommonOpts{
		CommonOpts: gcomponents.CommonOpts{
			K8sC:          c.k8sC,
			ClusterConfig: cc,
			Region:        rgn,
		},
		Version: version,
	})
}

func (c *Controller) acquire(task string, createdAt *timestamppb.Timestamp) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.tasks[task]; ok {
		return false
	}

	c.tasks[task] = createdAt
	return true
}

func (c *Controller) release(task string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.tasks, task)
}

func (c *Controller) isBusy(task string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.tasks[task]
	return ok
}

func (c *Controller) isOwned(task string, createdAt *timestamppb.Timestamp) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	cur, ok := c.tasks[task]
	return ok && isSameTimestamp(cur, createdAt)
}
