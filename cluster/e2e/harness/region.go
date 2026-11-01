// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Region struct {
	h *H

	Name       string
	Node       string
	NodeIP     string
	Kubeconfig string

	k8sC kubernetes.Interface
}

func (h *H) Region(t *testing.T) *Region {
	t.Helper()

	ret := &Region{
		h:          h,
		Name:       h.StateValue(t, eescenario.StateRegionName),
		Node:       h.StateValue(t, eescenario.StateRegionNode),
		NodeIP:     h.StateValue(t, eescenario.StateRegionNodeIP),
		Kubeconfig: h.StateValue(t, eescenario.StateRegionKubeconfig),
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", ret.Kubeconfig)
	if err != nil {
		t.Fatalf("Could not load the kubeconfig of the Region %s: %+v", ret.Name, err)
	}
	cfg.Timeout = 30 * time.Second

	k8sC, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Could not create the Kubernetes client of the Region %s: %+v", ret.Name, err)
	}
	ret.k8sC = k8sC

	return ret
}

func (r *Region) K8sC() kubernetes.Interface {
	return r.k8sC
}

func (r *Region) KubeconfigData(t *testing.T) string {
	t.Helper()

	ret, err := os.ReadFile(r.Kubeconfig)
	if err != nil {
		t.Fatalf("Could not read the kubeconfig of the Region %s: %+v", r.Name, err)
	}

	return string(ret)
}

func (r *Region) Kubectl(args string) string {
	return fmt.Sprintf("kubectl --kubeconfig %s %s", r.Kubeconfig, args)
}

func (r *Region) WaitDeployment(ctx context.Context, name string) error {
	return r.h.EventuallyErr(ctx, fmt.Sprintf("the deployment %s of the Region %s", name, r.Name),
		harness.DeploymentBudget, func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			return k8sutils.WaitReadinessDeploymentWithNS(ctx, r.k8sC, name, vutils.K8sNS)
		})
}

func (r *Region) MustWaitDeployment(t *testing.T, name string) {
	t.Helper()

	if err := r.WaitDeployment(t.Context(), name); err != nil {
		t.Fatalf("%+v", err)
	}
}

func (r *Region) MustWaitDaemonSet(t *testing.T, name string) {
	t.Helper()

	if err := r.h.EventuallyErr(t.Context(), fmt.Sprintf("the daemonset %s of the Region %s", name, r.Name),
		harness.DeploymentBudget, func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			return k8sutils.WaitReadinessDaemonsetWithNS(ctx, r.k8sC, name, vutils.K8sNS)
		}); err != nil {
		t.Fatalf("%+v", err)
	}
}

func (r *Region) SetGatewayHosts(t *testing.T, gws []*corev1.Gateway) {
	t.Helper()

	script, err := regionGatewayHostsScript(gws)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	r.h.MustRun(t, script)
}

func regionGatewayHostsScript(gws []*corev1.Gateway) (string, error) {
	if len(gws) == 0 {
		return "", errors.Errorf("No Gateway to publish")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "set -e\nsudo sed -i '/%s$/d' /etc/hosts\n", eescenario.RegionHostsMarker)

	for _, gw := range gws {
		if gw.Status == nil || gw.Status.Hostname == "" || len(gw.Status.PublicIPs) == 0 {
			return "", errors.Errorf("The Gateway %s has no hostname or public address yet",
				gw.GetMetadata().GetName())
		}

		fmt.Fprintf(&b, "echo %s | sudo tee -a /etc/hosts >/dev/null\n", shellQuote(
			fmt.Sprintf("%s %s %s", gw.Status.PublicIPs[0], gw.Status.Hostname, eescenario.RegionHostsMarker)))
	}

	fmt.Fprintf(&b, "grep -F '%s' /etc/hosts\n", eescenario.RegionHostsMarker)

	return b.String(), nil
}

func shellQuote(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}
