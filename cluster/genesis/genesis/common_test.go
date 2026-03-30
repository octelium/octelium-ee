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
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/genesis/genesis/components"
	gcomponents "github.com/octelium/octelium/cluster/genesis/genesis/components"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRscServerDeployment(t *testing.T) {

	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	assert.Nil(t, err)
	zap.ReplaceGlobals(logger)

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	cc, err := fakeC.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err)

	opts := &components.CommonOpts{
		CommonOpts: gcomponents.CommonOpts{
			K8sC:          fakeC.K8sC,
			ClusterConfig: cc,
		},
	}
	err = components.CreateRscServer(ctx, opts)
	assert.Nil(t, err)

	g := &Genesis{}
	g.k8sC = fakeC.K8sC

	/*
		err = g.setSecretManEnvVars(ctx)
		assert.Nil(t, err)

		depList, err := g.k8sC.AppsV1().Deployments(vutils.K8sNS).List(ctx, v1.ListOptions{
			LabelSelector: "octelium.com/component=rscserver",
		})
		assert.Nil(t, err)
		assert.True(t, len(depList.Items) > 0)

		for _, dep := range depList.Items {
			assert.True(t, slices.ContainsFunc(dep.Spec.Template.Spec.Containers, func(c k8scorev1.Container) bool {
				return slices.ContainsFunc(c.Env, func(e k8scorev1.EnvVar) bool {
					return e.Name == "OCTELIUM_USE_SECRETMAN"
				})
			}))

			assert.True(t, slices.ContainsFunc(dep.Spec.Template.Spec.Containers, func(c k8scorev1.Container) bool {
				return slices.ContainsFunc(c.Env, func(e k8scorev1.EnvVar) bool {
					return e.Name == "OCTELIUM_SECRETMAN_ADDR"
				})
			}))
		}
	*/
}
