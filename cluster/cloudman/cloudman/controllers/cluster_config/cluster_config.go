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

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/cloudmanutils"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnss"
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
	octeliumC octeliumc.ClientInterface,
) *Controller {
	return &Controller{
		octeliumC: octeliumC,
	}
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *enterprisev1.ClusterConfig) error {

	/*
		if new.Spec.Dns != nil && !pbutils.IsEqual(new.Spec.Dns, old.Spec.Dns) {
			if err := c.setAllServicesCNAME(ctx); err != nil {
				return err
			}
		}
	*/

	return nil
}

func (c *Controller) setAllServicesCNAME(ctx context.Context) error {

	zap.L().Info("Starting re-setting CNAME entries for public Services")
	svcList, err := c.octeliumC.CoreC().ListService(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldBooleanTrue("spec.isPublic"),
		},
	})
	if err != nil {
		return err
	}

	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	provider, err := cloudmanutils.GetDefaultDNSProvider(ctx, c.octeliumC)
	if err != nil {
		return err
	}

	for _, svc := range svcList.Items {
		if err := dnss.SetServiceCNAME(ctx, c.octeliumC, svc, cc, provider); err != nil {
			zap.L().Error("Could not set CNAME for Service",
				zap.String("name", svc.Metadata.Name),
				zap.Error(err))
		}
	}

	return nil
}
