// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package gwcontroller

import (
	"context"
	"reflect"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/cloudmanutils"
	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/dnsctl"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
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

func (c *Controller) OnAdd(ctx context.Context, gw *corev1.Gateway) error {
	zap.S().Debugf("New Gateway: %s", gw.Metadata.Name)
	if err := doSetDNS(ctx, c.octeliumC, gw); err != nil {
		return err
	}
	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *corev1.Gateway) error {

	if reflect.DeepEqual(new.Status.PublicIPs, old.Status.PublicIPs) {
		zap.S().Debugf("No need to update Gateway DNS entries: %s", new.Metadata.Name)
		return nil
	}

	if err := doSetDNS(ctx, c.octeliumC, new); err != nil {
		return err
	}

	return nil
}

func (c *Controller) OnDelete(ctx context.Context, gw *corev1.Gateway) error {

	return nil
}

func doSetDNS(ctx context.Context, octeliumC octeliumc.ClientInterface, gw *corev1.Gateway) error {

	if gw.Status.Hostname == "" {
		return nil
	}

	ccc, err := octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	zap.S().Debugf("Setting the Gateway %s FQDN %s public DNS to the IPs: %+v", gw.Metadata.Name, gw.Status.Hostname, gw.Status.PublicIPs)

	provider, err := cloudmanutils.GetDefaultDNSProvider(ctx, octeliumC)
	if err != nil {
		return err
	}

	if err := dnsctl.SetAddresses(ctx, octeliumC, provider, ccc, gw.Status.PublicIPs, gw.Status.Hostname); err != nil {
		zap.L().Warn("Could not do SetAddresses for Cluster domain", zap.Error(err))
	} else {
		zap.S().Debugf("Successfully set the DNS entries of the Gateway %s", gw.Metadata.Name)
	}

	return nil
}
