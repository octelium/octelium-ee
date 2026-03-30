// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package run

import (
	"context"
	"os"
	"os/signal"

	"github.com/octelium/octelium-ee/cluster/clusterman/clusterman/controllers/clusterconfig"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/watchers"
	"github.com/octelium/octelium/cluster/common/commoninit"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var Cmd = &cobra.Command{
	Use: "run",

	RunE: func(cmd *cobra.Command, args []string) error {
		return doCmd(cmd, args)
	},
}

func doCmd(cmd *cobra.Command, args []string) error {

	ctx, cancelFn := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFn()

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}

	k8sC, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	if err := commoninit.Run(ctx, nil); err != nil {
		return err
	}

	{
		watcher := watchers.NewEnterpriseV1(octeliumC)

		ccCtl := clusterconfig.NewController(octeliumC, k8sC)
		if err := watcher.ClusterConfig(ctx, nil, ccCtl.OnUpdate); err != nil {
			return err
		}
	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.L().Info("Cluster Manager is running...")

	<-ctx.Done()

	return nil
}
