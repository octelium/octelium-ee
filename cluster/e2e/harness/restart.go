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
	"testing"

	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/pkg/errors"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const enterpriseSelector = "octelium.com/app=octeliumee,octelium.com/component=%s"

func (h *H) EnterprisePods(ctx context.Context, component string) ([]k8scorev1.Pod, error) {
	podList, err := h.K8sC().CoreV1().Pods(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{
		LabelSelector: fmt.Sprintf(enterpriseSelector, component),
	})
	if err != nil {
		return nil, err
	}

	if len(podList.Items) < 1 {
		return nil, errors.Errorf("No enterprise pods found for the component %q", component)
	}

	return podList.Items, nil
}

func (h *H) RestartEnterprise(t *testing.T, component string) {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	before, err := h.EnterprisePods(ctx, component)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	old := map[string]struct{}{}
	for _, pod := range before {
		old[pod.Name] = struct{}{}

		if err := h.K8sC().CoreV1().Pods(vutils.K8sNS).
			Delete(ctx, pod.Name, k8smetav1.DeleteOptions{}); err != nil {
			t.Fatalf("Could not delete the pod %s: %+v", pod.Name, err)
		}
	}

	h.Within(t, fmt.Sprintf("the enterprise %s pods to be replaced", component),
		harness.DeploymentBudget, func(ctx context.Context) error {
			pods, err := h.EnterprisePods(ctx, component)
			if err != nil {
				return err
			}

			for _, pod := range pods {
				if _, ok := old[pod.Name]; ok {
					return errors.Errorf("the pod %s is still present", pod.Name)
				}
				if pod.Status.Phase != k8scorev1.PodRunning {
					return errors.Errorf("the pod %s is %s", pod.Name, pod.Status.Phase)
				}
				for _, cs := range pod.Status.ContainerStatuses {
					if !cs.Ready {
						return errors.Errorf("the container %s of the pod %s is not ready",
							cs.Name, pod.Name)
					}
				}
			}

			return nil
		})

	h.MustWaitDeployment(t, "octeliumee-"+component)
}

func (h *H) EnterpriseRestarts(t *testing.T, component string) int32 {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	pods, err := h.EnterprisePods(ctx, component)
	if err != nil {
		return 0
	}

	var ret int32
	for _, pod := range pods {
		for _, cs := range pod.Status.ContainerStatuses {
			ret += cs.RestartCount
		}
	}

	return ret
}
