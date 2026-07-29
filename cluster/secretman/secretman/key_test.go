// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package secretman

import (
	"testing"

	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestDEKAAD(t *testing.T) {
	k := &dek{
		uid: vutils.UUIDv4(),
	}

	key, err := utilrand.GetRandomBytes(32)
	assert.Nil(t, err)
	k.key = key

	plaintext, err := utilrand.GetRandomBytes(64)
	assert.Nil(t, err)

	aad := []byte("octelium-test-aad")

	res, err := k.encryptWithAAD(plaintext, aad)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, k.uid, res.KeyUID)
	assert.NotEqual(t, plaintext, res.Ciphertext)

	out, err := k.decryptWithAAD(res.Ciphertext, aad)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, plaintext, out)

	{
		_, err := k.decryptWithAAD(res.Ciphertext, []byte("octelium-test-aa"))
		assert.NotNil(t, err)
	}
	{
		_, err := k.decryptWithAAD(res.Ciphertext, nil)
		assert.NotNil(t, err)
	}
	{
		res2, err := k.encrypt(plaintext)
		assert.Nil(t, err)

		_, err = k.decryptWithAAD(res2.Ciphertext, aad)
		assert.NotNil(t, err)

		out2, err := k.decryptWithAAD(res2.Ciphertext, nil)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, plaintext, out2)
	}
	{
		res2, err := k.encryptWithAAD(plaintext, aad)
		assert.Nil(t, err)
		assert.NotEqual(t, res.Ciphertext, res2.Ciphertext)

		out2, err := k.decryptWithAAD(res2.Ciphertext, aad)
		assert.Nil(t, err)
		assert.Equal(t, plaintext, out2)
	}
	{
		tampered := append([]byte(nil), res.Ciphertext...)
		tampered[len(tampered)-1] ^= 0x01

		_, err := k.decryptWithAAD(tampered, aad)
		assert.NotNil(t, err)
	}
	{
		tampered := append([]byte(nil), res.Ciphertext...)
		tampered[0] ^= 0x01

		_, err := k.decryptWithAAD(tampered, aad)
		assert.NotNil(t, err)
	}
	{
		_, err := k.decryptWithAAD(nil, aad)
		assert.NotNil(t, err)
	}
	{
		_, err := k.decryptWithAAD([]byte("short"), aad)
		assert.NotNil(t, err)
	}
	{
		_, err := k.decryptWithAAD(make([]byte, 16), aad)
		assert.NotNil(t, err)
	}
}

func TestDEKInvalidKey(t *testing.T) {
	for _, size := range []int{0, 1, 15, 17, 31, 33, 64} {
		k := &dek{
			uid: vutils.UUIDv4(),
			key: make([]byte, size),
		}

		_, err := k.encrypt([]byte("octelium"))
		assert.NotNil(t, err)

		_, err = k.decrypt(make([]byte, 64))
		assert.NotNil(t, err)
	}

	for _, size := range []int{16, 24, 32} {
		key, err := utilrand.GetRandomBytes(size)
		assert.Nil(t, err)

		k := &dek{
			uid: vutils.UUIDv4(),
			key: key,
		}

		res, err := k.encrypt([]byte("octelium"))
		assert.Nil(t, err, "%+v", err)

		out, err := k.decrypt(res.Ciphertext)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []byte("octelium"), out)
	}
}
