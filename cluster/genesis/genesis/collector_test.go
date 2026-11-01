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

	"github.com/octelium/octelium-ee/cluster/common/collectorutils"
	oc "github.com/octelium/octelium-ee/cluster/common/components"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/genesis/genesis/genesisutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCollectorService(t *testing.T) {
	ctx := context.Background()
	t.Setenv("OCTELIUM_REGION_NAME", "")

	tst, err := otests.Initialize(nil)
	require.NoError(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	for _, name := range []string{"octelium", "octelium-api"} {
		require.NoError(t, genesisutils.CreateOrUpdateNamespace(ctx, tst.C.OcteliumC, &corev1.Namespace{
			Metadata: &metav1.Metadata{
				Name:     name,
				IsSystem: true,
			},
		}))
	}

	k8sC := fake.NewClientset()
	g := &Genesis{
		k8sC: k8sC,
	}

	assertCollectorService := func() {
		svc, err := tst.C.OcteliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{Name: collectorutils.ServiceName})
		require.NoError(t, err)
		assert.True(t, svc.Metadata.IsSystem)
		assert.True(t, svc.Metadata.IsUserHidden)
		assert.True(t, svc.Spec.IsPublic)
		assert.False(t, svc.Spec.IsAnonymous)
		assert.Equal(t, corev1.Service_Spec_GRPC, svc.Spec.Mode)
		assert.Empty(t, svc.Spec.Region)
		assert.Nil(t, svc.Status.ManagedService)
		assert.Equal(t, "octelium", svc.Status.NamespaceRef.Name)
		assert.Equal(t, "grpc://octeliumee-logstore.octelium.svc:8081", svc.Spec.Config.Upstream.GetUrl())

		require.Len(t, svc.Spec.DynamicConfig.Configs, 1)
		assert.Equal(t, "grpc://octeliumee-metricstore.octelium.svc:8081",
			svc.Spec.DynamicConfig.Configs[0].Upstream.GetUrl())
		require.Len(t, svc.Spec.DynamicConfig.Rules, 1)
		assert.Equal(t, `ctx.request.grpc.serviceFullName == "opentelemetry.proto.collector.metrics.v1.MetricsService"`,
			svc.Spec.DynamicConfig.Rules[0].Condition.GetMatch())
		assert.Equal(t, svc.Spec.DynamicConfig.Configs[0].Name, svc.Spec.DynamicConfig.Rules[0].GetConfigName())

		require.Len(t, svc.Spec.Authorization.InlinePolicies, 1)
		rules := svc.Spec.Authorization.InlinePolicies[0].Spec.Rules
		require.Len(t, rules, 2)
		assert.Equal(t, corev1.Policy_Spec_Rule_DENY, rules[0].Effect)
		assert.EqualValues(t, -4, rules[0].Priority)
		assert.Equal(t, getCollectorExportCondition(), rules[0].Condition.GetNot())
		assert.Equal(t, corev1.Policy_Spec_Rule_ALLOW, rules[1].Effect)
		assert.Equal(t, getCollectorExportCondition(), rules[1].Condition.GetMatch())
	}

	assertStoreIngest := func() {
		for _, component := range []string{oc.LogStore, oc.MetricStore} {
			name := oc.OcteliumEnterpriseComponent(component)

			svc, err := k8sC.CoreV1().Services(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
			require.NoError(t, err)
			require.Len(t, svc.Spec.Ports, 2)
			assert.Equal(t, "grpc", svc.Spec.Ports[0].Name)
			assert.EqualValues(t, 8080, svc.Spec.Ports[0].Port)
			assert.Equal(t, "ingest", svc.Spec.Ports[1].Name)
			assert.EqualValues(t, collectorutils.IngestPort, svc.Spec.Ports[1].Port)
			assert.EqualValues(t, collectorutils.IngestPort, svc.Spec.Ports[1].TargetPort.IntVal)

			pol, err := k8sC.NetworkingV1().NetworkPolicies(vutils.K8sNS).Get(ctx, name, k8smetav1.GetOptions{})
			require.NoError(t, err)
			require.Len(t, pol.Spec.Ingress, 2)
			assert.EqualValues(t, 8080, pol.Spec.Ingress[0].Ports[0].Port.IntVal)
			assert.Equal(t, map[string]string{"app": "octelium"}, pol.Spec.Ingress[0].From[0].PodSelector.MatchLabels)
			assert.EqualValues(t, collectorutils.IngestPort, pol.Spec.Ingress[1].Ports[0].Port.IntVal)
			assert.Equal(t, map[string]string{
				"octelium.com/component": "svc",
				"octelium.com/svc":       collectorutils.ServiceName,
			}, pol.Spec.Ingress[1].From[0].PodSelector.MatchLabels)
		}
	}

	require.NoError(t, g.RunInit(ctx, nil))
	assertCollectorService()
	assertStoreIngest()

	svc, err := tst.C.OcteliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{Name: collectorutils.ServiceName})
	require.NoError(t, err)
	svc.Spec.Authorization = nil
	svc.Spec.DynamicConfig = nil
	_, err = tst.C.OcteliumC.CoreC().UpdateService(ctx, svc)
	require.NoError(t, err)

	t.Setenv("OCTELIUM_REGION_NAME", "default")
	require.NoError(t, g.RunUpgrade(ctx, nil))
	assertCollectorService()
	assertStoreIngest()

	upgraded, err := tst.C.OcteliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{Name: collectorutils.ServiceName})
	require.NoError(t, err)
	assert.Equal(t, svc.Metadata.Uid, upgraded.Metadata.Uid)
}
