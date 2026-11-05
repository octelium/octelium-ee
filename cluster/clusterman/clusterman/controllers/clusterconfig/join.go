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
	joinTimeout          = 55 * time.Minute
	maxLastJoinRequests  = 200
	maxStatusMessageLen  = 1024
	statusUpdateAttempts = 8
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

	isStarted := false
	if err := c.updateJoinRequest(ctx, req, func(cur *enterprisev1.ClusterConfig_Status_JoinRequest) bool {
		if cur.State != enterprisev1.ClusterConfig_Status_JoinRequest_JOIN_REQUESTED {
			return false
		}

		cur.State = enterprisev1.ClusterConfig_Status_JoinRequest_JOINING
		isStarted = true
		return true
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

	doneCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if joinErr != nil {
		if val, err := c.octeliumC.CoreC().GetRegion(doneCtx, &rmetav1.GetOptions{
			Name: req.Request.GetName(),
		}); err == nil {
			rgn = val
		}
	}

	if err := c.updateStatus(doneCtx, func(cc *enterprisev1.ClusterConfig) bool {
		cur := cc.Status.JoinRequest
		if !isSameJoinRequest(cur, req) ||
			cur.State != enterprisev1.ClusterConfig_Status_JoinRequest_JOINING {
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
		}

		cc.Status.LastJoinRequests = prependJoinHistory(cc.Status.LastJoinRequests, cur)
		cc.Status.JoinRequest = nil

		return true
	}); err != nil {
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

func (c *Controller) updateJoinRequest(ctx context.Context,
	req *enterprisev1.ClusterConfig_Status_JoinRequest,
	fn func(cur *enterprisev1.ClusterConfig_Status_JoinRequest) bool) error {
	return c.updateStatus(ctx, func(cc *enterprisev1.ClusterConfig) bool {
		if !isSameJoinRequest(cc.Status.JoinRequest, req) {
			return false
		}

		return fn(cc.Status.JoinRequest)
	})
}

func (c *Controller) updateStatus(ctx context.Context, fn func(cc *enterprisev1.ClusterConfig) bool) error {
	for range statusUpdateAttempts {
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

		if _, err := c.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc); err == nil {
			return nil
		} else if !grpcerr.IsResourceChanged(err) {
			return err
		}

		zap.L().Debug("The ClusterConfig has changed. Trying again...")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	return errors.Errorf("Could not update the ClusterConfig status after %d attempts", statusUpdateAttempts)
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
