// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package azurekeyvault

import (
	"bytes"
	"context"
	"fmt"
	"strings"
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
				Name:           fmt.Sprintf("test-azure-key-vault-%s", vutils.GetMyRegionName()),
				IsSystem:       true,
				IsSystemHidden: true,
			},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_AzureKeyVault_{
					AzureKeyVault: &enterprisev1.SecretStore_Spec_AzureKeyVault{
						ClientID: "00000000-0000-0000-0000-000000000001",
						TenantID: "00000000-0000-0000-0000-000000000002",
						VaultURL: "https://octelium-test.vault.azure.net",
						Key:      "octelium-test-key",
					},
				},
			},
			Status: &enterprisev1.SecretStore_Status{
				State: enterprisev1.SecretStore_Status_OK,
				Type:  enterprisev1.SecretStore_Status_TYPE_AZURE_KEY_VAULT,
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

func TestAzureEnvelopeRoundTripAndTruncation(t *testing.T) {
	wrapped := bytes.Repeat([]byte{1}, 32)
	sealed := bytes.Repeat([]byte{2}, minGCMBlobSz)
	blob, err := encodeBlob(algAzureRSAOAEP256, "key-name", "version", wrapped, sealed)
	require.NoError(t, err)
	header, gotWrapped, gotSealed, err := decodeBlob(blob)
	require.NoError(t, err)
	assert.Equal(t, algAzureRSAOAEP256, header.algID)
	assert.Equal(t, "key-name", header.keyName)
	assert.Equal(t, "version", header.keyVersion)
	assert.Equal(t, wrapped, gotWrapped)
	assert.Equal(t, sealed, gotSealed)
	for size := range len(blob) {
		_, _, _, err := decodeBlob(blob[:size])
		assert.Error(t, err)
	}
	bad := bytes.Clone(blob)
	bad[0] ^= 1
	_, _, _, err = decodeBlob(bad)
	assert.Error(t, err)
	bad = bytes.Clone(blob)
	bad[len(blobMagic)]++
	_, _, _, err = decodeBlob(bad)
	assert.Error(t, err)
	for _, tc := range []struct {
		alg             byte
		name, version   string
		wrapped, sealed []byte
	}{
		{0, "name", "", wrapped, sealed}, {algLocalTest, "", "", wrapped, sealed}, {algLocalTest, "name", "", nil, sealed}, {algLocalTest, "name", "", wrapped, nil},
		{algLocalTest, strings.Repeat("x", maxLenPrefixedField+1), "", wrapped, sealed}, {algLocalTest, "name", strings.Repeat("x", maxLenPrefixedField+1), wrapped, sealed}, {algLocalTest, "name", "", bytes.Repeat([]byte{1}, maxLenPrefixedField+1), sealed},
	} {
		_, err := encodeBlob(tc.alg, tc.name, tc.version, tc.wrapped, tc.sealed)
		assert.Error(t, err)
	}
	for _, input := range [][]byte{nil, {0}, {0, 2, 1}} {
		_, _, err := readLenPrefixed(input)
		assert.Error(t, err)
	}
}

func TestAzureAADSeparatesWrappingAndData(t *testing.T) {
	s := &store{}
	assert.NotEqual(t, s.dataAAD(algLocalTest, "key", "version", "uid"), s.wrapAAD(algLocalTest, "key", "version", "uid"))
	assert.NotEqual(t, s.dataAAD(algLocalTest, "key", "version", "uid"), s.dataAAD(algLocalTest, "key", "other-version", "uid"))
}
