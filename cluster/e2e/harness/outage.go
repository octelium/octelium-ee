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
	"sync"
	"testing"
	"time"

	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *H) StopEnterpriseWithin(t *testing.T, component string, budget time.Duration) func() {
	t.Helper()

	name := "octeliumee-" + component
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	scale, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).GetScale(ctx,
		name, k8smetav1.GetOptions{})
	cancel()
	require.Nil(t, err)
	replicas := scale.Spec.Replicas

	var once sync.Once
	restore := func() {
		once.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), harness.DeploymentBudget)
			defer cancel()

			err := h.EventuallyErr(ctx, "the "+component+" replicas to be restored", 30*time.Second,
				func(ctx context.Context) error {
					cur, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).GetScale(ctx,
						name, k8smetav1.GetOptions{})
					if err != nil {
						return err
					}
					cur.Spec.Replicas = replicas
					_, err = h.K8sC().AppsV1().Deployments(vutils.K8sNS).UpdateScale(ctx,
						name, cur, k8smetav1.UpdateOptions{})
					return err
				})
			require.Nil(t, err)
			if replicas > 0 {
				require.Nil(t, h.WaitDeployment(ctx, name))
			}
		})
	}
	t.Cleanup(restore)

	h.Eventually(t, "the "+component+" replicas to be stopped", 30*time.Second,
		func(ctx context.Context) error {
			cur, err := h.K8sC().AppsV1().Deployments(vutils.K8sNS).GetScale(ctx,
				name, k8smetav1.GetOptions{})
			if err != nil {
				return err
			}
			cur.Spec.Replicas = 0
			_, err = h.K8sC().AppsV1().Deployments(vutils.K8sNS).UpdateScale(ctx,
				name, cur, k8smetav1.UpdateOptions{})
			return err
		})
	h.Eventually(t, "all "+component+" pods to exit", budget,
		func(ctx context.Context) error {
			pods, err := h.K8sC().CoreV1().Pods(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{
				LabelSelector: fmt.Sprintf(enterpriseSelector, component),
			})
			if err != nil {
				return err
			}
			if len(pods.Items) != 0 {
				return errors.Errorf("the component %s still has %d pods", component, len(pods.Items))
			}
			return nil
		})
	return restore
}
