// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package hashicorpvault

import (
	"context"
	"fmt"
	"testing"

	vault "github.com/hashicorp/vault/api"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	db, err := postgresutils.NewDB()
	assert.Nil(t, err)

	err = migrations.Migrate(ctx, db)
	assert.Nil(t, err)

	ss, err := fakeC.OcteliumC.EnterpriseC().CreateSecretStore(ctx, &enterprisev1.SecretStore{
		Metadata: &metav1.Metadata{
			Name:           utilrand.GetRandomStringCanonical(8),
			IsSystem:       true,
			IsSystemHidden: true,
		},
		Spec: &enterprisev1.SecretStore_Spec{
			Type: &enterprisev1.SecretStore_Spec_HashicorpVault_{
				HashicorpVault: &enterprisev1.SecretStore_Spec_HashicorpVault{
					Address: "http://localhost:8200",
					Key:     utilrand.GetRandomStringCanonical(6),
				},
			},
		},
		Status: &enterprisev1.SecretStore_Status{
			State: enterprisev1.SecretStore_Status_OK,
			Type:  enterprisev1.SecretStore_Status_TYPE_HASHICORP_VAULT,
		},
	})

	assert.Nil(t, err)

	{
		cfg := vault.DefaultConfig()
		cfg.Address = ss.Spec.GetHashicorpVault().Address

		c, err := vault.NewClient(cfg)
		assert.Nil(t, err)

		c.SetToken(testToken)

		_, _ = c.Logical().Write("sys/mounts/transit", map[string]interface{}{
			"type": "transit",
		})

		_, err = c.Logical().Write(fmt.Sprintf("transit/keys/%s", ss.Spec.GetHashicorpVault().Key), map[string]interface{}{
			"type": "aes256-gcm96",
		})
		assert.Nil(t, err)
	}

	store, err := NewStore(ctx, &stores.StoreOpts{
		SecretStore: ss,
	})
	assert.Nil(t, err)

	err = store.Initialize(ctx)
	assert.Nil(t, err)

	val := utilrand.GetRandomBytesMust(32)
	uid := vutils.UUIDv4()
	ciphertext, err := store.Encrypt(ctx, uid, val)
	assert.Nil(t, err)

	{
		val2, err := store.Decrypt(ctx, uid, ciphertext)
		assert.Nil(t, err)
		assert.Equal(t, val, val2)
	}

	{
		val2, err := store.Decrypt(ctx, uid, ciphertext)
		assert.Nil(t, err)
		assert.Equal(t, val, val2)
	}
}
