// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package awskms

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
				Name:           fmt.Sprintf("test-aws-kms-%s", vutils.GetMyRegionName()),
				IsSystem:       true,
				IsSystemHidden: true,
			},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_AwsKeyManagementService{
					AwsKeyManagementService: &enterprisev1.SecretStore_Spec_AWSKeyManagementService{
						KeyID:   "test-key",
						Region:  "us-east-1",
						RoleARN: "arn:aws:iam::123456789012:role/octelium-secretman",
					},
				},
			},
			Status: &enterprisev1.SecretStore_Status{
				State: enterprisev1.SecretStore_Status_OK,
				Type:  enterprisev1.SecretStore_Status_TYPE_AWS_KMS,
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

	ciphertext, err := store.Encrypt(ctx, uid, val)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}
	assert.NotEmpty(t, ciphertext)

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

	_, err = store.Decrypt(ctx, "different-dek-uid", ciphertext)
	assert.NotNil(t, err)
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

func TestAWSRoleSessionNameBoundaries(t *testing.T) {
	for _, value := range []string{"", "---", "us-east-1", "region with spaces / ?", strings.Repeat("a", 200), "地区"} {
		name := roleSessionName(value)
		assert.NotEmpty(t, name)
		assert.LessOrEqual(t, len(name), maxRoleSessionNameLen)
		for _, char := range name {
			assert.True(t, isRoleSessionNameChar(char))
		}
	}
	assert.Equal(t, "octelium-secretman-default", roleSessionName(""))
	assert.NotEqual(t, dataAAD("one"), dataAAD("two"))
	assert.Equal(t, "uid", encryptionContext("uid")["uid"])
}
