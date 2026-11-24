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

	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/join"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	joinTimeout            = 55 * time.Minute
	maxLastJoinRequests    = 200
	maxStatusMessageLen    = 1024
	statusRetryInterval    = 200 * time.Millisecond
	maxStatusRetryInterval = 10 * time.Second
)

func (c *Controller) join(cluster *enterprisev1.ClusterConfig) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), joinTimeout)
		defer cancel()

		if err := c.doJoin(ctx, cluster); err != nil {
			zap.L().Warn("Could not doJoin", zap.Error(err))
		}
	}()
}

func (c *Controller) doJoin(ctx context.Context, cluster *enterprisev1.ClusterConfig) error {
	req := cluster.Status.JoinRequest
	if req == nil || req.State != enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED {
		return nil
	}

	if !c.acquire(taskJoin, req.CreatedAt) {
		zap.L().Debug("Another join request is being handled. Nothing to be done...")
		return nil
	}
	defer c.release(taskJoin)

	isStarted := false
	if err := c.updateStatus(ctx, func(cc *enterprisev1.ClusterConfig) bool {
		cur := cc.Status.JoinRequest
		if !isSameJoinRequest(cur, req) {
			isStarted = false
			return false
		}

		if isStarted && cur.State == enterprisev1.ClusterConfig_Status_JoinRequest_JOINING {
			return false
		}

		isStarted = cur.State == enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED
		if isStarted {
			cur.State = enterprisev1.ClusterConfig_Status_JoinRequest_JOINING
		}

		return isStarted
	}); err != nil {
		return err
	}

	if !isStarted {
		zap.L().Debug("The join request is already handled. Nothing to be done...")
		return nil
	}

	zap.L().Debug("Starting doJoin", zap.String("region", req.Request.GetName()))

	rgn, joinErr := join.JoinCluster(ctx, &join.JoinClusterOpts{
		OcteliumC:     c.octeliumC,
		K8sC:          c.k8sC,
		JoinRequest:   req,
		NewRegionK8sC: c.newRegionK8sC,
	})
	if joinErr != nil {
		zap.L().Warn("Could not join the Region", zap.Error(joinErr))
	}

	doneCtx, cancel := context.WithTimeout(context.Background(), recordTimeout)
	defer cancel()

	if joinErr != nil {
		if val, err := c.octeliumC.CoreC().GetRegion(doneCtx, &rmetav1.GetOptions{
			Name: req.Request.GetName(),
		}); err == nil {
			rgn = val
		}
	}

	if err := c.recordJoin(doneCtx, req, rgn, joinErr); err != nil {
		return err
	}

	if joinErr != nil {
		return joinErr
	}

	zap.L().Debug("Completed doJoin", zap.String("region", req.Request.GetName()))

	return nil
}

func (c *Controller) setUpgradeRegions(ctx context.Context,
	req *enterprisev1.ClusterConfig_Status_UpgradeRequest,
	regions []*enterprisev1.ClusterConfig_Status_UpgradeRequest_Region) {
	ctx, cancel := context.WithTimeout(ctx, reportTimeout)
	defer cancel()

	if err := c.updateStatus(ctx, func(cc *enterprisev1.ClusterConfig) bool {
		cur := cc.Status.UpgradeRequest
		if cur == nil || req == nil ||
			cur.State != enterprisev1.ClusterConfig_Status_UpgradeRequest_UPGRADING ||
			!isSameTimestamp(cur.CreatedAt, req.CreatedAt) {
			return false
		}

		cur.Regions = regions
		return true
	}); err != nil {
		zap.L().Warn("Could not set the upgrade Regions", zap.Error(err))
	}
}

func (c *Controller) recordJoin(ctx context.Context,
	req *enterprisev1.ClusterConfig_Status_JoinRequest, rgn *corev1.Region, joinErr error) error {
	return c.updateStatus(ctx, func(cc *enterprisev1.ClusterConfig) bool {
		cur := cc.Status.JoinRequest
		isCurrent := isSameJoinRequest(cur, req)
		if !isCurrent {
			cur = findJoinRequest(cc.Status.LastJoinRequests, req)
		}

		if cur == nil || isJoinDone(cur.State) {
			return false
		}

		cur.DoneAt = pbutils.Now()
		if rgn != nil {
			cur.RegionRef = umetav1.GetObjectReference(rgn)
		}

		if joinErr != nil {
			cur.State = enterprisev1.ClusterConfig_Status_JoinRequest_FAILED
			cur.Message = truncateStatusMessage(joinErr.Error())
		} else {
			cur.State = enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS
			cur.Message = ""
		}

		if isCurrent {
			cc.Status.LastJoinRequests = prependJoinHistory(cc.Status.LastJoinRequests, cur)
			cc.Status.JoinRequest = nil
		}

		return true
	})
}

func (c *Controller) reconcileJoin(ctx context.Context, cc *enterprisev1.ClusterConfig) error {
	for i, req := range cc.Status.LastJoinRequests {
		if isJoinDone(req.State) || c.isOwned(taskJoin, req.CreatedAt) {
			continue
		}

		before := cc.Status.JoinRequest.GetCreatedAt()
		if i > 0 {
			before = cc.Status.LastJoinRequests[i-1].CreatedAt
		}

		zap.L().Info("Recording the replaced join request", zap.Any("request", req))

		if err := c.recoverJoin(ctx, req, before, supersededRequestMessage); err != nil {
			return err
		}
	}

	req := cc.Status.JoinRequest
	if req == nil || c.isBusy(taskJoin) {
		return nil
	}

	switch req.State {
	case enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED:
		zap.L().Info("Starting the pending join request", zap.String("region", req.Request.GetName()))
		c.join(cc)
		return nil
	case enterprisev1.ClusterConfig_Status_JoinRequest_JOINING:
		zap.L().Info("Recording the interrupted join request", zap.String("region", req.Request.GetName()))
		return c.recoverJoin(ctx, req, nil, interruptedRequestMessage)
	default:
		return nil
	}
}

func (c *Controller) recoverJoin(ctx context.Context,
	req *enterprisev1.ClusterConfig_Status_JoinRequest, before *timestamppb.Timestamp, msg string) error {
	rgn, err := join.GetJoinedRegion(ctx, c.octeliumC, req, before)
	if err != nil {
		return err
	}

	var joinErr error
	if rgn == nil {
		joinErr = errors.New(msg)
	}

	return c.recordJoin(ctx, req, rgn, joinErr)
}

func isJoinDone(state enterprisev1.ClusterConfig_Status_JoinRequest_State) bool {
	switch state {
	case enterprisev1.ClusterConfig_Status_JoinRequest_SUCCESS,
		enterprisev1.ClusterConfig_Status_JoinRequest_FAILED:
		return true
	default:
		return false
	}
}

func findJoinRequest(history []*enterprisev1.ClusterConfig_Status_JoinRequest,
	req *enterprisev1.ClusterConfig_Status_JoinRequest) *enterprisev1.ClusterConfig_Status_JoinRequest {
	for _, itm := range history {
		if isSameJoinRequest(itm, req) {
			return itm
		}
	}

	return nil
}

func (c *Controller) updateStatus(ctx context.Context, fn func(cc *enterprisev1.ClusterConfig) bool) error {
	interval := statusRetryInterval

	for {
		err := c.doUpdateStatus(ctx, fn)
		if err == nil || !isTransientErr(err) {
			return err
		}

		zap.L().Debug("Could not update the ClusterConfig status. Trying again...", zap.Error(err))

		select {
		case <-ctx.Done():
			return err
		case <-time.After(interval):
		}

		interval = min(2*interval, maxStatusRetryInterval)
	}
}

func (c *Controller) doUpdateStatus(ctx context.Context, fn func(cc *enterprisev1.ClusterConfig) bool) error {
	cc, err := c.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	if cc.Status == nil {
		cc.Status = &enterprisev1.ClusterConfig_Status{}
	}

	if !fn(cc) {
		return nil
	}

	_, err = c.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
	return err
}

func isTransientErr(err error) bool {
	return grpcerr.IsResourceChanged(err) ||
		grpcerr.IsUnavailable(err) ||
		grpcerr.IsInternal(err) ||
		grpcerr.IsDeadlineExceeded(err) ||
		grpcerr.IsResourceExhausted(err)
}

func isSameJoinRequest(a, b *enterprisev1.ClusterConfig_Status_JoinRequest) bool {
	if a == nil || b == nil {
		return false
	}

	return a.Request.GetName() == b.Request.GetName() && isSameTimestamp(a.CreatedAt, b.CreatedAt)
}

func isSameUpgradeRequest(a, b *enterprisev1.ClusterConfig_Status_UpgradeRequest) bool {
	if a == nil || b == nil {
		return false
	}

	return isSameTimestamp(a.CreatedAt, b.CreatedAt)
}

func isSameTimestamp(a, b *timestamppb.Timestamp) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return a.Seconds == b.Seconds && a.Nanos == b.Nanos
}

func prependJoinHistory(
	history []*enterprisev1.ClusterConfig_Status_JoinRequest,
	req *enterprisev1.ClusterConfig_Status_JoinRequest,
) []*enterprisev1.ClusterConfig_Status_JoinRequest {
	if req == nil {
		return history
	}

	history = append([]*enterprisev1.ClusterConfig_Status_JoinRequest{req}, history...)
	if len(history) > maxLastJoinRequests {
		history = history[:maxLastJoinRequests]
	}

	return history
}

func truncateStatusMessage(msg string) string {
	if len(msg) > maxStatusMessageLen {
		return msg[:maxStatusMessageLen]
	}
	return msg
}
