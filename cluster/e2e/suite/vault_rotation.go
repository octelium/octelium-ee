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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func testSecretStoreVaultKeyRotation(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapVault)

	vault := h.Vault(t)
	original := h.SecretStore(t, "default")
	keyBefore := readVaultKeyInfo(t, vault)
	consumer := newSecretConsumer(t, h)

	t.Cleanup(func() {
		vault.ExecWithin(t, fmt.Sprintf("vault write transit/keys/%s/config min_decryption_version=%d",
			vault.Key, keyBefore.Data.MinDecryptions), 30*time.Second)
		ss := h.SetSecretStoreSpec(t, "default", original.Spec)
		synchronizeCurrentSecretStore(t, h, ss)
	})

	ss := h.SetSecretStoreSpec(t, "default", vault.SecretStoreSpec())
	synchronizeCurrentSecretStore(t, h, ss)

	vault.ExecWithin(t, fmt.Sprintf("vault write -f transit/keys/%s/rotate", vault.Key), 30*time.Second)
	keyAfter := readVaultKeyInfo(t, vault)
	require.Equal(t, keyBefore.Data.LatestVersion+1, keyAfter.Data.LatestVersion)

	t.Run("OldCiphertextSurvivesTransitKeyRotation", func(t *testing.T) {
		h.RestartEnterprise(t, "secretman")
		probeStoredSecret(t, h, consumer)
	})

	t.Run("SynchronizationRewrapsEveryDataEncryptionKey", func(t *testing.T) {
		synchronizeCurrentSecretStore(t, h, h.SecretStore(t, "default"))
		vault.ExecWithin(t, fmt.Sprintf("vault write transit/keys/%s/config min_decryption_version=%d",
			vault.Key, keyAfter.Data.LatestVersion), 30*time.Second)
		require.Equal(t, keyAfter.Data.LatestVersion, readVaultKeyInfo(t, vault).Data.MinDecryptions)

		h.RestartEnterprise(t, "secretman")
		probeStoredSecret(t, h, consumer)
		newSecretConsumer(t, h)
	})

	t.Run("RewrappedSecretsMigrateBackToKubernetes", func(t *testing.T) {
		ss := h.SetSecretStoreSpec(t, "default", &enterprisev1.SecretStore_Spec{
			Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
				Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
			},
		})
		synchronizeCurrentSecretStore(t, h, ss)
		h.RestartEnterprise(t, "secretman")
		probeStoredSecret(t, h, consumer)
	})
}

func readVaultKeyInfo(t *testing.T, vault *eeharness.Vault) *eeharness.VaultKeyInfo {
	t.Helper()

	out := vault.ExecWithin(t, fmt.Sprintf("vault read -format=json transit/keys/%s", vault.Key),
		30*time.Second)
	ret := &eeharness.VaultKeyInfo{}
	require.Nil(t, json.Unmarshal(out, ret))
	return ret
}

func synchronizeCurrentSecretStore(t *testing.T, h *eeharness.H, ss *enterprisev1.SecretStore) {
	t.Helper()

	before := ss.GetStatus().GetSynchronization().GetCreatedAt().AsTime()
	ctx, cancel := h.Ctx(t)
	defer cancel()
	callCtx, callCancel := context.WithTimeout(ctx, 30*time.Second)
	_, err := h.EnterpriseC().SynchronizeSecretStore(callCtx, &enterprisev1.SynchronizeSecretStoreRequest{
		SecretStoreRef: &metav1.ObjectReference{Uid: ss.Metadata.Uid},
	})
	callCancel()
	require.Nil(t, err)
	err = h.EventuallyErr(ctx, "a new SecretStore synchronization to complete", eeharness.SyncBudget,
		func(ctx context.Context) error {
			cur, err := h.EnterpriseC().GetSecretStore(ctx, &metav1.GetOptions{Uid: ss.Metadata.Uid})
			if err != nil {
				return err
			}
			sync := cur.GetStatus().GetSynchronization()
			if sync == nil || !sync.GetCreatedAt().AsTime().After(before) ||
				sync.State != enterprisev1.SecretStore_Status_Synchronization_SUCCESS ||
				!sync.CompletedAt.IsValid() {
				return errors.Errorf("the new SecretStore synchronization has not succeeded")
			}
			if cur.Status.Type != ss.Status.Type || cur.Status.State != enterprisev1.SecretStore_Status_OK {
				return errors.Errorf("the SecretStore is not ready with the requested backend")
			}
			return nil
		})
	require.Nil(t, err)
}

func probeStoredSecret(t *testing.T, h *eeharness.H, consumer *secretConsumer) {
	t.Helper()

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
					Type: &corev1.Service_Spec_Config_Upstream_Url{Url: consumer.upstream.URL},
				},
				Type: &corev1.Service_Spec_Config_Http{
					Http: &corev1.Service_Spec_Config_HTTP{
						Auth: &corev1.Service_Spec_Config_HTTP_Auth{
							Type: &corev1.Service_Spec_Config_HTTP_Auth_Bearer_{
								Bearer: &corev1.Service_Spec_Config_HTTP_Auth_Bearer{
									Type: &corev1.Service_Spec_Config_HTTP_Auth_Bearer_FromSecret{
										FromSecret: consumer.secret.Metadata.Name,
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
	h.Probe(t, usr, svc).MustBeAllowed(t)
}
