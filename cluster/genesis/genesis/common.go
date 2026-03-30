// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesis

import (
	"context"
	"os"
	"time"

	oc "github.com/octelium/octelium-ee/cluster/common/components"
	"github.com/octelium/octelium-ee/cluster/genesis/genesis/components"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (g *Genesis) installComponents(ctx context.Context, opts *components.CommonOpts, isInit bool) error {
	zap.S().Debugf("Installing components...")
	clusterCfg, err := g.octeliumCInit.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	if err := g.setClusterConfigSecretManager(ctx); err != nil {
		return err
	}

	// zap.S().Debugf("Got region: %s", region.Metadata.Name)
	if opts == nil {
		opts = &components.CommonOpts{}
	}

	opts.ClusterConfig = clusterCfg
	if opts.K8sC == nil {
		opts.K8sC = g.k8sC
	}

	{
		regionName := func() string {
			if os.Getenv("OCTELIUM_REGION_NAME") != "" {
				return os.Getenv("OCTELIUM_REGION_NAME")
			}
			return "default"
		}()

		if g.octeliumC != nil {
			opts.OcteliumC = g.octeliumC

			rgn, err := g.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
				Name: regionName,
			})
			if err != nil {
				return err
			}
			opts.Region = rgn
		} else if g.octeliumCInit != nil {
			opts.OcteliumC = g.octeliumCInit

			rgn, err := g.octeliumCInit.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
				Name: regionName,
			})
			if err != nil {
				return err
			}
			opts.Region = rgn
		}
	}

	region := opts.Region

	if err := components.CreateRscServer(ctx, opts); err != nil {
		return err
	}

	if err := k8sutils.WaitReadinessDeployment(ctx, g.k8sC, oc.OcteliumEnterpriseComponent(oc.RscServer)); err != nil {
		return err
	}

	/*
		if err := g.setSecretManEnvVars(ctx); err != nil {
			zap.L().Warn("Could not set secretManager env vars", zap.Error(err))
		}
	*/

	if err := components.CreateNocturne(ctx, opts); err != nil {
		return err
	}

	{
		if err := components.CreateSecretMan(ctx, opts); err != nil {
			return err
		}

		if err := g.rolloutRestartRscServers(ctx); err != nil {
			return err
		}
	}

	if region.Metadata.Name == "default" {
		if err := components.CreateCloudMan(ctx, opts); err != nil {
			return err
		}
	}

	if isInit {
		if err := components.CreateClusterMan(ctx, opts); err != nil {
			return err
		}
	}

	if err := components.CreateCollector(ctx, opts); err != nil {
		return err
	}

	if region.Metadata.Name == "default" {
		if err := components.CreateLogStore(ctx, opts); err != nil {
			return err
		}

		if err := components.CreateRscStore(ctx, opts); err != nil {
			return err
		}

		if err := components.CreateMetricStore(ctx, opts); err != nil {
			return err
		}

		if err := components.CreatePolicyPortal(ctx, opts); err != nil {
			return err
		}
	}

	return nil
}

func (g *Genesis) setClusterConfigSecretManager(ctx context.Context) error {
	cc, err := g.octeliumCInit.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	cc.Status.SecretManager = &corev1.ClusterConfig_Status_SecretManager{
		Address: "octeliumee-secretman.octelium.svc:8080",
	}

	if _, err := g.octeliumCInit.CoreC().UpdateClusterConfig(ctx, cc); err != nil {
		return err
	}

	return nil
}

func (g *Genesis) rolloutRestartRscServers(ctx context.Context) error {
	depList, err := g.k8sC.AppsV1().Deployments(vutils.K8sNS).List(ctx, v1.ListOptions{
		LabelSelector: "octelium.com/component=rscserver",
	})
	if err != nil {
		return err
	}

	cc, err := g.octeliumCInit.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	secretManagerBytes, _ := pbutils.Marshal(cc.Status.SecretManager)

	zap.L().Debug("Found pods to set SecretMan env vars",
		zap.Int("count", len(depList.Items)))

	for i := range depList.Items {
		dep := depList.Items[i].DeepCopy()
		if dep.Spec.Template.Annotations == nil {
			dep.Spec.Template.Annotations = make(map[string]string)
		}
		dep.Spec.Template.Annotations["octelium.com/secret-manager-config"] = vutils.Sha256SumHex(secretManagerBytes)
		zap.L().Debug("Updating rscserver pod", zap.Any("dep", dep))
		_, err = g.k8sC.AppsV1().Deployments(vutils.K8sNS).Update(ctx, dep, v1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	time.Sleep(5 * time.Second)
	if err := k8sutils.WaitReadinessDeployment(ctx, g.k8sC, oc.OcteliumEnterpriseComponent(oc.RscServer)); err != nil {
		return err
	}

	return nil
}
