// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package cloudman

import (
	"context"
	"time"

	isscontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/certificateissuers"
	crtcontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/certificates"
	cccontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/cluster_config"
	dnspcontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/dnsproviders"
	gwcontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/gateways"
	nscontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/namespaces"
	regioncontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/regions"
	svccontroller "github.com/octelium/octelium-ee/cluster/cloudman/cloudman/controllers/services"
	eewatchers "github.com/octelium/octelium-ee/cluster/common/watchers"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/common/watchers"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"

	"github.com/octelium/octelium-ee/cluster/cloudman/cloudman/watcher"
)

func Run(ctx context.Context) error {
	if err := commoninit.Run(ctx, nil); err != nil {
		return err
	}

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	watcher.InitWatcher(octeliumC).Run(ctx)

	svcCtl := svccontroller.NewController(octeliumC)
	ccCtl := cccontroller.NewController(octeliumC)
	regionCtl := regioncontroller.NewController(octeliumC)
	gwCtl := gwcontroller.NewController(octeliumC)
	crtCtl := crtcontroller.NewController(octeliumC)
	issCtl := isscontroller.NewController(octeliumC)
	nsCtl := nscontroller.NewController(octeliumC)
	dnsPCtl := dnspcontroller.NewController(octeliumC)

	if err := waitForDefaultCertificateIssuer(ctx, octeliumC); err != nil {
		zap.L().Warn("Could not waitForDefaultCertificateIssuer", zap.Error(err))
	}

	{
		watcher := watchers.NewCoreV1(octeliumC)
		if err := watcher.Service(ctx, nil, svcCtl.OnAdd, svcCtl.OnUpdate, svcCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.Region(ctx, nil, regionCtl.OnAdd, regionCtl.OnUpdate, regionCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.Gateway(ctx, nil, gwCtl.OnAdd, gwCtl.OnUpdate, gwCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.Namespace(ctx, nil, nsCtl.OnAdd, nsCtl.OnUpdate, nsCtl.OnDelete); err != nil {
			return err
		}
	}

	{
		watcher := eewatchers.NewEnterpriseV1(octeliumC)
		if err := watcher.ClusterConfig(ctx, nil, ccCtl.OnUpdate); err != nil {
			return err
		}

		if err := watcher.DNSProvider(ctx, nil, dnsPCtl.OnAdd, dnsPCtl.OnUpdate, dnsPCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.CertificateIssuer(ctx, nil, issCtl.OnAdd, issCtl.OnUpdate, issCtl.OnDelete); err != nil {
			return err
		}

		if err := watcher.Certificate(ctx, nil, crtCtl.OnAdd, crtCtl.OnUpdate, crtCtl.OnDelete); err != nil {
			return err
		}

	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.L().Info("Cloud Manager is running...")

	<-ctx.Done()

	return nil
}

func waitForDefaultCertificateIssuer(ctx context.Context, octeliumC octeliumc.ClientInterface) error {
	for range 150 {
		if _, err := octeliumC.EnterpriseC().GetCertificateIssuer(ctx, &rmetav1.GetOptions{
			Name: "default",
		}); err == nil {
			return nil
		} else if grpcerr.IsNotFound(err) {
			zap.L().Debug("Could not find default certIssuer. Trying again...")
			time.Sleep(1 * time.Second)
		} else {
			return err
		}
	}

	return nil
}
