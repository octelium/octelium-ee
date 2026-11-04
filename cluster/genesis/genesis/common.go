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
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
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

	if isInit && region.Metadata.Name == "default" {
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
	if err := ctx.Err(); err != nil {
		return err
	}
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
		if err := ctx.Err(); err != nil {
			return err
		}
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

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	if err := k8sutils.WaitReadinessDeployment(ctx, g.k8sC, oc.OcteliumEnterpriseComponent(oc.RscServer)); err != nil {
		return err
	}

	return nil
}

func waitGatewayAgentReady(ctx context.Context, k8sC kubernetes.Interface) error {
	if ldflags.IsTest() {
		return nil
	}
	return k8sutils.WaitReadiness(ctx, 2*time.Second, 20000*time.Second, func(ctx context.Context) (bool, error) {
		return gatewayAgentReadiness(ctx, k8sC)
	})
}

func gatewayAgentReadiness(ctx context.Context, k8sC kubernetes.Interface) (bool, error) {
	ds, err := k8sC.AppsV1().DaemonSets(vutils.K8sNS).Get(ctx, "octelium-gwagent", v1.GetOptions{})
	if err != nil {
		return false, err
	}
	return ds.Status.DesiredNumberScheduled > 0 && k8sutils.IsDaemonSetReady(ds), nil
}

func waitGatewayAgentsRegistered(ctx context.Context, k8sC kubernetes.Interface) error {
	if ldflags.IsTest() {
		return nil
	}

	zap.L().Debug("Checking that all Gateway Agents are registered")

	doCheck := func(ctx context.Context) (bool, error) {
		ready, err := gatewayAgentReadiness(ctx, k8sC)
		if err != nil || !ready {
			return false, err
		}
		nodeList, err := k8sC.CoreV1().Nodes().List(ctx, v1.ListOptions{
			LabelSelector: vutils.NodeLabelDataPlane + "=",
		})
		if err != nil {
			return false, err
		}

		if len(nodeList.Items) == 0 {
			return false, errors.Errorf(
				"No Kubernetes node is labeled as an Octelium data-plane node. At least one node must be labeled via: kubectl label nodes <NODE_NAME> %s=",
				vutils.NodeLabelDataPlane)
		}

		for _, node := range nodeList.Items {
			if node.Labels[vutils.NodeLabelGatewayRegistered] != "true" {
				zap.L().Debug("Gateway Agent is not registered yet on the data-plane node",
					zap.String("node", node.Name))
				return false, nil
			}
		}

		zap.L().Debug("All Gateway Agents are now registered",
			zap.Int("nodes", len(nodeList.Items)))

		return true, nil
	}

	return k8sutils.WaitReadiness(ctx, 2*time.Second, 1800*time.Second, doCheck)
}

func (g *Genesis) setRegionVersionMap(ctx context.Context, rgn *corev1.Region) error {
	info := &corev1.Region_Status_VersionInfo{
		Package: "octeliumee",
		SetAt:   pbutils.Now(),
		Version: ldflags.GetVersion(),
		Id:      os.Getenv("OCTELIUM_INSTALL_ID"),
	}
	err := retryRegionVersion(ctx, func(ctx context.Context) error {
		region, err := g.octeliumC.CoreC().GetRegion(ctx, apivalidation.ObjectToRGetOptions(rgn))
		if err != nil {
			return err
		}
		updated := proto.Clone(region).(*corev1.Region)
		if updated.Status == nil {
			updated.Status = &corev1.Region_Status{}
		}
		if updated.Status.VersionInfoMap == nil {
			updated.Status.VersionInfoMap = make(map[string]*corev1.Region_Status_VersionInfo)
		}
		updated.Status.VersionInfoMap["octeliumee"] = info
		if proto.Equal(region, updated) {
			return nil
		}
		_, err = g.octeliumC.CoreC().UpdateRegion(ctx, updated)
		return err
	})
	return errors.Wrap(err, "Could not publish octeliumee Region version")
}

func retryRegionVersion(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	err := wait.ExponentialBackoffWithContext(ctx, retry.DefaultRetry, func(ctx context.Context) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		lastErr = fn(ctx)
		if lastErr == nil {
			return true, nil
		}
		switch status.Code(lastErr) {
		case codes.OutOfRange, codes.Unavailable, codes.ResourceExhausted, codes.Aborted, codes.DeadlineExceeded:
			return false, nil
		}
		if errors.Is(lastErr, context.DeadlineExceeded) {
			return false, nil
		}
		return false, lastErr
	})
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil && lastErr != nil {
		return lastErr
	}
	return err
}
