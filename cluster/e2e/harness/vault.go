// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"encoding/json"
	"fmt"
	"testing"

	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/enterprisev1"
)

type Vault struct {
	h *H

	Addr string
	Role string
	Key  string

	pod       string
	namespace string
	token     string
}

func (h *H) Vault(t *testing.T) *Vault {
	t.Helper()

	return &Vault{
		h:         h,
		Addr:      h.StateValue(t, eescenario.StateVaultAddr),
		Role:      h.StateValue(t, eescenario.StateVaultRole),
		Key:       h.StateValue(t, eescenario.StateVaultKey),
		pod:       h.StateValue(t, eescenario.StateVaultPod),
		namespace: h.StateValue(t, eescenario.StateVaultNS),
		token:     h.StateValue(t, eescenario.StateVaultToken),
	}
}

func (v *Vault) SecretStoreSpec() *enterprisev1.SecretStore_Spec {
	return &enterprisev1.SecretStore_Spec{
		Type: &enterprisev1.SecretStore_Spec_HashicorpVault_{
			HashicorpVault: &enterprisev1.SecretStore_Spec_HashicorpVault{
				Address: v.Addr,
				Role:    v.Role,
				Key:     v.Key,
			},
		},
	}
}

func (v *Vault) Exec(t *testing.T, cmd string) []byte {
	t.Helper()

	return v.h.MustOutput(t, fmt.Sprintf(
		`kubectl exec -n %s %s -- sh -c 'export VAULT_ADDR=http://127.0.0.1:8200; `+
			`export VAULT_TOKEN=%s; %s'`,
		v.namespace, v.pod, v.token, cmd))
}

type VaultKeyInfo struct {
	Data struct {
		Name           string         `json:"name"`
		LatestVersion  int            `json:"latest_version"`
		Type           string         `json:"type"`
		Keys           map[string]any `json:"keys"`
		MinDecryptions int            `json:"min_decryption_version"`
	} `json:"data"`
}

func (v *Vault) KeyInfo(t *testing.T) *VaultKeyInfo {
	t.Helper()

	out := v.Exec(t, fmt.Sprintf("vault read -format=json transit/keys/%s", v.Key))

	ret := &VaultKeyInfo{}
	if err := json.Unmarshal(out, ret); err != nil {
		t.Fatalf("Could not parse the Vault Transit key info: %+v\n%s", err, out)
	}

	return ret
}
