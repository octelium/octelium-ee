// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package directoryproviders

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
	srvI      srvI
}

func NewController(
	octeliumC octeliumc.ClientInterface,
	srvI srvI,
) *Controller {
	return &Controller{
		octeliumC: octeliumC,
		srvI:      srvI,
	}
}

type srvI interface {
	SetDirectoryProvider(dp *enterprisev1.DirectoryProvider)
	DeleteDirectoryProvider(dp *enterprisev1.DirectoryProvider)
	Synchronize(ctx context.Context, dp *enterprisev1.DirectoryProvider) error
}

func (c *Controller) OnAdd(ctx context.Context, dp *enterprisev1.DirectoryProvider) error {

	c.srvI.SetDirectoryProvider(dp)

	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *enterprisev1.DirectoryProvider) error {
	c.srvI.SetDirectoryProvider(new)

	if !pbutils.IsEqual(new.Status.Synchronization, old.Status.Synchronization) {
		if new.Status.Synchronization != nil &&
			new.Status.Synchronization.State == enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED &&
			(old == nil ||
				old.Status.Synchronization == nil ||
				old.Status.Synchronization.State != enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED) {
			if err := c.srvI.Synchronize(ctx, new); err != nil {
				zap.L().Warn("Could not synchronize directoryProvider", zap.Any("dp", new))
			}
		}
	}

	return nil
}

func (c *Controller) OnDelete(ctx context.Context, dp *enterprisev1.DirectoryProvider) error {

	c.srvI.DeleteDirectoryProvider(dp)

	return nil
}
