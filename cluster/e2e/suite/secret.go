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
	"net/http"
	"testing"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type secretConsumer struct {
	probe func(t *testing.T)
}

func newSecretConsumer(t *testing.T, h *eeharness.H) *secretConsumer {
	t.Helper()

	token := utilrand.GetRandomStringCanonical(32)
	sec := h.CreateCoreSecret(t, token)

	upstream := h.StartNodeUpstream(t, token)

	h.Eventually(t, "the node upstream to answer", eeharness.PropagationBudget,
		upstream.Reachable)

	usr := h.CreateWorkloadUser(t, &corev1.User_Spec_Authorization{
		InlinePolicies: harness.InlineAllowAny("allow"),
	})

	svc := h.CreateService(t, &corev1.Service{
		Metadata: &metav1.Metadata{Name: h.Name()},
		Spec: &corev1.Service_Spec{
			Mode:     corev1.Service_Spec_HTTP,
			IsPublic: true,
			Config: &corev1.Service_Spec_Config{
				Upstream: &corev1.Service_Spec_Config_Upstream{
					Type: &corev1.Service_Spec_Config_Upstream_Url{
						Url: upstream.URL,
					},
				},
				Type: &corev1.Service_Spec_Config_Http{
					Http: &corev1.Service_Spec_Config_HTTP{
						Auth: &corev1.Service_Spec_Config_HTTP_Auth{
							Type: &corev1.Service_Spec_Config_HTTP_Auth_Bearer_{
								Bearer: &corev1.Service_Spec_Config_HTTP_Auth_Bearer{
									Type: &corev1.Service_Spec_Config_HTTP_Auth_Bearer_FromSecret{
										FromSecret: sec.Metadata.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	})

	h.MustWaitService(t, svc.Metadata.Name)

	probe := h.Probe(t, usr, svc)
	probe.MustBeAllowed(t)

	return &secretConsumer{
		probe: func(t *testing.T) { probe.MustBeAllowed(t) },
	}
}

func testSecretDurability(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	consumer := newSecretConsumer(t, h)

	t.Run("AfterSecretManRestart", func(t *testing.T) {
		h.RestartEnterprise(t, "secretman")
		consumer.probe(t)
	})

	t.Run("AfterRscServerRestart", func(t *testing.T) {
		h.RestartEnterprise(t, "rscserver")
		consumer.probe(t)
	})

	t.Run("EnterpriseSecretShapes", func(t *testing.T) {
		ctx := t.Context()

		for _, data := range []*enterprisev1.Secret_Data{
			{Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomStringCanonical(32)}},
			{Type: &enterprisev1.Secret_Data_ValueBytes{
				ValueBytes: []byte(utilrand.GetRandomStringCanonical(48))}},
		} {
			sec, err := h.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
				Metadata: &metav1.Metadata{Name: h.Name()},
				Spec:     &enterprisev1.Secret_Spec{},
				Data:     data,
			})
			require.Nil(t, err)

			got, err := h.EnterpriseC().GetSecret(ctx,
				&metav1.GetOptions{Uid: sec.Metadata.Uid})
			require.Nil(t, err)
			assert.Nil(t, got.Data)

			_, err = h.EnterpriseC().DeleteSecret(ctx,
				&metav1.DeleteOptions{Uid: sec.Metadata.Uid})
			assert.Nil(t, err)
		}
	})
}

func testSecretStoreVault(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapVault)

	vault := h.Vault(t)
	consumer := newSecretConsumer(t, h)

	before := vault.KeyInfo(t)

	t.Cleanup(func() {
		h.SetSecretStoreSpec(t, "default", &enterprisev1.SecretStore_Spec{
			Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
				Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
			},
		})
	})

	ss := h.SetSecretStoreSpec(t, "default", vault.SecretStoreSpec())
	h.SynchronizeSecretStore(t, ss)

	h.WaitSecretStoreSynchronized(t, "default",
		enterprisev1.SecretStore_Status_TYPE_HASHICORP_VAULT, eeharness.SyncBudget)

	t.Run("VaultKeyIsUsed", func(t *testing.T) {
		after := vault.KeyInfo(t)
		assert.Equal(t, vault.Key, after.Data.Name)
		assert.True(t, after.Data.LatestVersion >= before.Data.LatestVersion)
	})

	t.Run("ExistingSecretsStillResolve", func(t *testing.T) {
		consumer.probe(t)
	})

	t.Run("NewSecretsResolve", func(t *testing.T) {
		newSecretConsumer(t, h)
	})

	t.Run("SecretManHasNotRestarted", func(t *testing.T) {
		assert.Zero(t, h.EnterpriseRestarts(t, "secretman"))
	})

	t.Run("AfterSecretManRestart", func(t *testing.T) {
		h.RestartEnterprise(t, "secretman")
		consumer.probe(t)
	})

	t.Run("BackToKubernetes", func(t *testing.T) {
		ss := h.SetSecretStoreSpec(t, "default", &enterprisev1.SecretStore_Spec{
			Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
				Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
			},
		})
		h.SynchronizeSecretStore(t, ss)

		h.WaitSecretStoreSynchronized(t, "default",
			enterprisev1.SecretStore_Status_KUBERNETES, eeharness.SyncBudget)

		consumer.probe(t)
	})
}

func testSecretStoreVaultFailure(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapVault)

	vault := h.Vault(t)
	consumer := newSecretConsumer(t, h)

	t.Cleanup(func() {
		h.SetSecretStoreSpec(t, "default", &enterprisev1.SecretStore_Spec{
			Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
				Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
			},
		})
	})

	ss := h.SetSecretStoreSpec(t, "default", &enterprisev1.SecretStore_Spec{
		Type: &enterprisev1.SecretStore_Spec_HashicorpVault_{
			HashicorpVault: &enterprisev1.SecretStore_Spec_HashicorpVault{
				Address: vault.Addr,
				Role:    "octelium-e2e-nonexistent",
				Key:     vault.Key,
			},
		},
	})
	h.SynchronizeSecretStore(t, ss)

	t.Run("SecretsSurviveAFailedRotation", func(t *testing.T) {
		consumer.probe(t)
	})

	t.Run("SynchronizationDoesNotHang", func(t *testing.T) {
		h.Eventually(t, "the SecretStore to leave the SYNCING state",
			eeharness.SyncBudget, func(ctx context.Context) error {
				cur, err := h.EnterpriseC().GetSecretStore(ctx,
					&metav1.GetOptions{Name: "default"})
				if err != nil {
					return err
				}
				if cur.Status.Synchronization == nil {
					return nil
				}
				if cur.Status.Synchronization.State ==
					enterprisev1.SecretStore_Status_Synchronization_FAILED {
					return nil
				}
				return errUnexpectedSyncState(cur.Status.Synchronization.State.String())
			})
	})
}

func testSecretConsumersAfterRotation(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapVault, eescenario.CapOTLPSink)

	vault := h.Vault(t)
	sink := h.Sink(t)

	token := utilrand.GetRandomStringCanonical(40)
	sec := h.CreateEnterpriseSecret(t, token)

	exp := h.CreateCollectorExporter(t, otlpExporter(sink.GRPCEndpoint, sec.Metadata.Name))
	h.SetCollectorPipelines(t,
		logsPipeline(h.Name(), exp.Metadata.Name),
		metricsPipeline(h.Name(), exp.Metadata.Name))

	sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)

	t.Cleanup(func() {
		h.SetSecretStoreSpec(t, "default", &enterprisev1.SecretStore_Spec{
			Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
				Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
			},
		})
	})

	ss := h.SetSecretStoreSpec(t, "default", vault.SecretStoreSpec())
	h.SynchronizeSecretStore(t, ss)
	h.WaitSecretStoreSynchronized(t, "default",
		enterprisev1.SecretStore_Status_TYPE_HASHICORP_VAULT, eeharness.SyncBudget)

	t.Run("ExporterStillAuthenticates", func(t *testing.T) {
		sink.Truncate(t)
		h.RestartEnterprise(t, "collector")

		h.GetStatus(t, h.HTTPPublic("demo-nginx"), "/", http.StatusUnauthorized)
		sink.WaitLogs(t, "resourceLogs", eeharness.IngestionBudget)
	})
}
