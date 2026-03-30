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
	"fmt"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"go.uber.org/zap"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
}

func NewController(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
) (*Controller, error) {
	return &Controller{
		octeliumC: octeliumC,
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, itm *enterprisev1.DirectoryProvider) error {

	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *enterprisev1.DirectoryProvider) error {

	return nil
}

func (c *Controller) OnDelete(ctx context.Context, itm *enterprisev1.DirectoryProvider) error {

	{
		userList, err := c.octeliumC.EnterpriseC().ListDirectoryProviderUser(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("status.directoryProviderRef.uid", itm.Metadata.Uid),
			},
		})
		if err != nil {
			return err
		}

		for _, itm := range userList.Items {
			if _, err := c.octeliumC.EnterpriseC().DeleteDirectoryProviderUser(ctx, &rmetav1.DeleteOptions{
				Uid: itm.Metadata.Uid,
			}); err != nil {
				zap.L().Warn("Could not deleteDirectoryProviderUser", zap.Error(err))
			}
		}
	}

	{
		grpList, err := c.octeliumC.EnterpriseC().ListDirectoryProviderGroup(ctx, &rmetav1.ListOptions{
			Filters: []*rmetav1.ListOptions_Filter{
				urscsrv.FilterFieldEQValStr("status.directoryProviderRef.uid", itm.Metadata.Uid),
			},
		})
		if err != nil {
			return err
		}

		for _, itm := range grpList.Items {
			if _, err := c.octeliumC.EnterpriseC().DeleteDirectoryProviderGroup(ctx, &rmetav1.DeleteOptions{
				Uid: itm.Metadata.Uid,
			}); err != nil {
				zap.L().Warn("Could not deleteDirectoryProviderGroup", zap.Error(err))
			}
		}
	}

	{
		if _, err := c.octeliumC.CoreC().DeleteUser(ctx, &rmetav1.DeleteOptions{
			Name: fmt.Sprintf("sys:dp-%s", itm.Status.Id),
		}); err != nil {
			zap.L().Debug("Could not delete the directoryProvider User", zap.Error(err))
		}
	}

	return nil
}
