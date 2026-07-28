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
	"os"
	"strings"
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

const defaultTestVaultAddr = "http://127.0.0.1:8200"

func testVaultAddr() string {
	if addr := strings.TrimSpace(os.Getenv("VAULT_ADDR")); addr != "" {
		return addr
	}

	return defaultTestVaultAddr
}

func TestServer(t *testing.T) {
	ctx := context.Background()

	t.Setenv(transitMountPathEnv, defaultTransitMountPath)
	t.Setenv(kubernetesAuthMountPathEnv, defaultKubernetesAuthMountPath)

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	t.Cleanup(func() {
		tst.Destroy()
	})

	fakeC := tst.C

	db, err := postgresutils.NewDB()
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	err = migrations.Migrate(ctx, db)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	vaultAddr := testVaultAddr()

	vaultCfg := vault.DefaultConfig()
	assert.NotNil(t, vaultCfg)
	if vaultCfg == nil {
		return
	}

	assert.Nil(t, vaultCfg.Error, "%+v", vaultCfg.Error)
	if vaultCfg.Error != nil {
		return
	}

	vaultCfg.Address = vaultAddr

	vaultC, err := vault.NewClient(vaultCfg)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	vaultC.SetToken(testToken)

	mounts, err := vaultC.Sys().ListMountsWithContext(ctx)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	if _, ok := mounts[defaultTransitMountPath+"/"]; !ok {
		err = vaultC.Sys().MountWithContext(
			ctx,
			defaultTransitMountPath,
			&vault.MountInput{
				Type: "transit",
			},
		)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}
	}

	keyName := fmt.Sprintf("octelium-test-%s", vutils.UUIDv4())
	keyPath := fmt.Sprintf(
		"%s/keys/%s",
		defaultTransitMountPath,
		keyName,
	)

	_, err = vaultC.Logical().WriteWithContext(
		ctx,
		keyPath,
		map[string]interface{}{
			"type": "aes256-gcm96",
		},
	)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()

		_, err := vaultC.Logical().WriteWithContext(
			cleanupCtx,
			keyPath+"/config",
			map[string]interface{}{
				"deletion_allowed": true,
			},
		)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}

		_, err = vaultC.Logical().DeleteWithContext(
			cleanupCtx,
			keyPath,
		)
		assert.Nil(t, err, "%+v", err)
	})

	ss, err := fakeC.OcteliumC.EnterpriseC().CreateSecretStore(
		ctx,
		&enterprisev1.SecretStore{
			Metadata: &metav1.Metadata{
				Name:           fmt.Sprintf("test-hashicorp-vault-%s", vutils.GetMyRegionName()),
				IsSystem:       true,
				IsSystemHidden: true,
			},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_HashicorpVault_{
					HashicorpVault: &enterprisev1.SecretStore_Spec_HashicorpVault{
						Address: vaultAddr,
						Role:    "octelium-test",
						Key:     keyName,
					},
				},
			},
			Status: &enterprisev1.SecretStore_Status{
				State: enterprisev1.SecretStore_Status_OK,
				Type:  enterprisev1.SecretStore_Status_TYPE_HASHICORP_VAULT,
			},
		},
	)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	store, err := NewStore(ctx, &stores.StoreOpts{
		SecretStore: ss,
	})
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	t.Cleanup(func() {
		assert.Nil(t, store.Close())
	})

	assert.Equal(t, ss.Metadata.Uid, store.UID())

	err = store.Initialize(ctx)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	val, err := utilrand.GetRandomBytes(32)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	const uid = "test-dek-uid"

	{
		ciphertext, err := store.Encrypt(ctx, uid, val)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}

		assert.NotEmpty(t, ciphertext)
		assert.NotEqual(t, val, ciphertext)

		plaintext, err := store.Decrypt(ctx, uid, ciphertext)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}

		assert.Equal(t, val, plaintext)

		p2, err := store.Decrypt(ctx, uid, ciphertext)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}

		assert.Equal(t, plaintext, p2)

		_, err = store.Decrypt(
			ctx,
			"different-dek-uid",
			ciphertext,
		)
		assert.NotNil(t, err)
	}

	{
		ciphertext1, err := store.Encrypt(ctx, uid, val)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}

		ciphertext2, err := store.Encrypt(ctx, uid, val)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}

		assert.NotEqual(t, ciphertext1, ciphertext2)

		plaintext1, err := store.Decrypt(ctx, uid, ciphertext1)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}

		plaintext2, err := store.Decrypt(ctx, uid, ciphertext2)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}

		assert.Equal(t, val, plaintext1)
		assert.Equal(t, val, plaintext2)
	}

	{
		ciphertext, err := store.Encrypt(ctx, uid, val)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return
		}

		tampered := append([]byte(nil), ciphertext...)
		tampered[len(tampered)-1] ^= 0x01

		_, err = store.Decrypt(ctx, uid, tampered)
		assert.NotNil(t, err)
	}

	{
		_, err := store.Encrypt(ctx, "", val)
		assert.NotNil(t, err)

		_, err = store.Encrypt(ctx, uid, nil)
		assert.NotNil(t, err)

		_, err = store.Encrypt(ctx, uid, []byte{})
		assert.NotNil(t, err)

		_, err = store.Decrypt(ctx, "", []byte("invalid"))
		assert.NotNil(t, err)

		_, err = store.Decrypt(ctx, uid, nil)
		assert.NotNil(t, err)

		_, err = store.Decrypt(ctx, uid, []byte{})
		assert.NotNil(t, err)

		_, err = store.Decrypt(ctx, uid, []byte("invalid"))
		assert.NotNil(t, err)
	}

	assert.Nil(t, store.Close())
	assert.Nil(t, store.Close())
}

func TestGetMountPath(t *testing.T) {
	{
		ret, err := getMountPath("", "transit")
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "transit", ret)
	}

	{
		ret, err := getMountPath("/custom/transit/", "transit")
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "custom/transit", ret)
	}

	{
		_, err := getMountPath("/", "")
		assert.NotNil(t, err)
	}

	{
		_, err := getMountPath("../transit", "transit")
		assert.NotNil(t, err)
	}

	{
		_, err := getMountPath("transit/../other", "transit")
		assert.NotNil(t, err)
	}
}

func TestValidateKeyName(t *testing.T) {
	assert.Nil(t, validateKeyName("octelium-key"))
	assert.Nil(t, validateKeyName("octelium_key-01"))

	assert.NotNil(t, validateKeyName(""))
	assert.NotNil(t, validateKeyName(" "))
	assert.NotNil(t, validateKeyName("."))
	assert.NotNil(t, validateKeyName(".."))
	assert.NotNil(t, validateKeyName("path/to/key"))
}
