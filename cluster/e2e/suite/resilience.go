// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"testing"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var enterpriseComponents = []string{
	"nocturne",
	"secretman",
	"collector",
	"policyportal",
	"cloudman",
	"clusterman",
	"logstore",
	"rscstore",
	"metricstore",
}

func testEnterpriseTopology(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	t.Run("TheCollectorServiceSelectsTheEnterpriseCollector", func(t *testing.T) {
		svc, err := h.K8sC().CoreV1().Services(vutils.K8sNS).
			Get(ctx, "octelium-collector", k8smetav1.GetOptions{})
		require.Nil(t, err)

		assert.Equal(t, "octeliumee", svc.Spec.Selector["octelium.com/app"])
		assert.Equal(t, "collector", svc.Spec.Selector["octelium.com/component"])
	})

	t.Run("TheSecretManagerAddressIsSet", func(t *testing.T) {
		cc, err := h.CoreC().GetClusterConfig(ctx, &corev1.GetClusterConfigRequest{})
		require.Nil(t, err)
		require.NotNil(t, cc.Status.SecretManager)

		assert.Equal(t, "octeliumee-secretman.octelium.svc:8080",
			cc.Status.SecretManager.Address)
	})

	t.Run("SecretManRunsWithItsServiceAccount", func(t *testing.T) {
		dep := h.MustK8sDeployment(t, "octeliumee-secretman")
		assert.Equal(t, "octeliumee-secretman",
			dep.Spec.Template.Spec.ServiceAccountName)
	})

	t.Run("EveryEnterpriseDeploymentIsReady", func(t *testing.T) {
		for _, name := range eescenario.Deployments {
			h.MustWaitDeployment(t, name)
		}
	})

	t.Run("TheEnterpriseServicesExist", func(t *testing.T) {
		for _, name := range []string{
			"enterprise.octelium-api",
			"public.octelium",
			"dirsync.octelium",
			"console.octelium",
			"access.octelium",
		} {
			_, err := h.CoreC().GetService(ctx, &metav1.GetOptions{Name: name})
			assert.Nil(t, err, "the Service %s does not exist", name)
		}
	})

	t.Run("TheDefaultEnterpriseResourcesExist", func(t *testing.T) {
		_, err := h.EnterpriseC().GetDNSProvider(ctx, &metav1.GetOptions{Name: "default"})
		assert.Nil(t, err)

		_, err = h.EnterpriseC().GetCertificateIssuer(ctx, &metav1.GetOptions{Name: "default"})
		assert.Nil(t, err)

		ss, err := h.EnterpriseC().GetSecretStore(ctx, &metav1.GetOptions{Name: "default"})
		require.Nil(t, err)
		assert.Equal(t, enterprisev1.SecretStore_Status_OK, ss.Status.State)
	})
}

func testEnterpriseRollingRestart(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	for _, component := range enterpriseComponents {
		t.Run(component, func(t *testing.T) {
			h.RestartEnterprise(t, component)

			enterpriseSmoke(t, h)

			for _, other := range enterpriseComponents {
				assert.Zero(t, h.EnterpriseRestarts(t, other),
					"the component %s restarted while %s was replaced", other, component)
			}
		})
	}
}

func testRscServerFailover(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	sec := h.CreateEnterpriseSecret(t, utilrand.GetRandomStringCanonical(24))

	h.RestartEnterprise(t, "rscserver")

	t.Run("TheAPIRecovers", func(t *testing.T) {
		h.Eventually(t, "the enterprise API to answer after the rscserver restart",
			eeharness.PropagationBudget, func(ctx context.Context) error {
				_, err := h.EnterpriseC().GetSecret(ctx,
					&metav1.GetOptions{Uid: sec.Metadata.Uid})
				return err
			})
	})

	t.Run("TheWatchersReconnect", func(t *testing.T) {
		dp := h.CreateDirectoryProvider(t, &enterprisev1.DirectoryProvider{
			Spec: &enterprisev1.DirectoryProvider_Spec{
				Type: &enterprisev1.DirectoryProvider_Spec_Scim{
					Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
				},
			},
		})

		h.Eventually(t, "dirsync to react to the new DirectoryProvider",
			eeharness.PropagationBudget, func(ctx context.Context) error {
				cur, err := h.EnterpriseC().GetDirectoryProvider(ctx,
					&metav1.GetOptions{Uid: dp.Metadata.Uid})
				if err != nil {
					return err
				}
				if cur.Status.Id == "" {
					return errors.Errorf("the DirectoryProvider has no id yet")
				}
				return nil
			})
	})

	t.Run("TheAccessControllerStillReconciles", func(t *testing.T) {
		c := newAccessCast(t, h)
		h.CreateAccessPolicy(t, c.autoApproveRule("failover"))

		req := h.CreateRequest(t, c.alice,
			eeharness.CatalogRequest(c.alphaCatalog, eeharness.Minutes(5)))

		h.WaitRequestState(t, req,
			accessv1.Request_Status_State_APPROVED, eeharness.RequestBudget)
	})
}

func testStorePersistence(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	driveTraffic(t, h)
	waitAccessLogGrows(t, h, 1)

	ctx, cancel := h.Ctx(t)
	before, err := accessLogCount(ctx, h)
	cancel()
	require.Nil(t, err)

	for _, component := range []string{"logstore", "rscstore", "metricstore"} {
		h.RestartEnterprise(t, component)
	}

	h.Eventually(t, "the stores to answer with their previous data",
		eeharness.IngestionBudget, func(ctx context.Context) error {
			cur, err := accessLogCount(ctx, h)
			if err != nil {
				return err
			}
			if cur < before {
				return errors.Errorf("the access log lost entries: %d, was %d", cur, before)
			}
			return nil
		})

	waitAccessLogGrows(t, h, 1)
}

func enterpriseSmoke(t *testing.T, h *eeharness.H) {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	sec, err := h.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: h.Name()},
		Spec:     &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomStringCanonical(24),
			},
		},
	})
	require.Nil(t, err)

	_, err = h.EnterpriseC().DeleteSecret(ctx, &metav1.DeleteOptions{Uid: sec.Metadata.Uid})
	assert.Nil(t, err)

	_, err = h.EnterpriseC().GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
	assert.Nil(t, err)
}
