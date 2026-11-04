package genesis

import (
	"context"
	"time"

	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func waitGatewayAgentReady(ctx context.Context, k8sC kubernetes.Interface) error {
	if ldflags.IsTest() {
		return nil
	}
	return k8sutils.WaitReadiness(ctx, 2*time.Second, 20000*time.Second, func(ctx context.Context) (bool, error) {
		return gatewayAgentReadiness(ctx, k8sC)
	})
}

func gatewayAgentReadiness(ctx context.Context, k8sC kubernetes.Interface) (bool, error) {
	ds, err := k8sC.AppsV1().DaemonSets(vutils.K8sNS).Get(ctx, "octelium-gwagent", k8smetav1.GetOptions{})
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
		nodeList, err := k8sC.CoreV1().Nodes().List(ctx, k8smetav1.ListOptions{
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
