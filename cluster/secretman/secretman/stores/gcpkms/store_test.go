// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package gcpkms

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	ctx := context.Background()

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

	ss, err := fakeC.OcteliumC.EnterpriseC().CreateSecretStore(
		ctx,
		&enterprisev1.SecretStore{
			Metadata: &metav1.Metadata{
				Name:           fmt.Sprintf("test-gcp-kms-%s", vutils.GetMyRegionName()),
				IsSystem:       true,
				IsSystemHidden: true,
			},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_GoogleCloudKeyManagementService_{
					GoogleCloudKeyManagementService: &enterprisev1.SecretStore_Spec_GoogleCloudKeyManagementService{
						Project:  "octelium-test",
						Location: "global",
						KeyRing:  "octelium-test",
						Key:      "octelium-test-key",
					},
				},
			},
			Status: &enterprisev1.SecretStore_Status{
				State: enterprisev1.SecretStore_Status_OK,
				Type:  enterprisev1.SecretStore_Status_TYPE_GCP_KMS,
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
}

func TestStoreAEADPayloadBoundaries(t *testing.T) {
	for _, size := range []int{16, 24, 32} {
		key := bytes.Repeat([]byte{byte(size)}, size)
		for _, plaintext := range [][]byte{nil, {0}, bytes.Repeat([]byte{0, 255, 128, 1}, 16384)} {
			blob, err := gcmSeal(key, plaintext, []byte("resource-one"))
			require.NoError(t, err)
			got, err := gcmOpen(key, blob, []byte("resource-one"))
			require.NoError(t, err)
			assert.True(t, bytes.Equal(plaintext, got))
			_, err = gcmOpen(key, blob, []byte("resource-two"))
			assert.Error(t, err)
			for _, length := range []int{0, 1, 11, len(blob) - 1} {
				_, err = gcmOpen(key, blob[:length], []byte("resource-one"))
				assert.Error(t, err)
			}
			blob[len(blob)-1] ^= 1
			_, err = gcmOpen(key, blob, []byte("resource-one"))
			assert.Error(t, err)
		}
	}
	_, err := gcmSeal([]byte("invalid"), []byte("value"), nil)
	assert.Error(t, err)
	_, err = gcmOpen([]byte("invalid"), nil, nil)
	assert.Error(t, err)
}

func TestStoreRejectsMissingConfiguration(t *testing.T) {
	for _, opts := range []*stores.StoreOpts{nil, {}, {SecretStore: &enterprisev1.SecretStore{}}, {SecretStore: &enterprisev1.SecretStore{Metadata: &metav1.Metadata{Uid: "uid"}}}} {
		_, err := NewStore(t.Context(), opts)
		assert.Error(t, err)
	}
}

func TestGCPKeyPathChecksumAndAAD(t *testing.T) {
	assert.Equal(t, "projects/project/locations/global/keyRings/ring/cryptoKeys/key", getKeyPath("project", "global", "ring", "key"))
	assert.Equal(t, uint32(0xe3069283), crc32c([]byte("123456789")))
	assert.Zero(t, crc32c(nil))
	assert.NotEqual(t, dataAAD("one"), dataAAD("two"))
}
