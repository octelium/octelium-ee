// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package secretman

import (
	"crypto/aes"
	"crypto/cipher"
	"time"

	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
)

type dekEncryptionOutptut struct {
	Ciphertext []byte
	KeyUID     string
}

type dek struct {
	uid       string
	key       []byte
	createdAt time.Time
}

func (k *dek) encrypt(plaintext []byte) (*dekEncryptionOutptut, error) {
	return k.encryptWithAAD(plaintext, nil)
}

func (k *dek) encryptWithAAD(plaintext, aad []byte) (*dekEncryptionOutptut, error) {
	block, err := aes.NewCipher(k.key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := utilrand.GetRandomBytes(aesgcm.NonceSize())
	if err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, aad)

	return &dekEncryptionOutptut{
		Ciphertext: ciphertext,
		KeyUID:     k.uid,
	}, nil
}

func (k *dek) decrypt(ciphertext []byte) ([]byte, error) {
	return k.decryptWithAAD(ciphertext, nil)
}

func (k *dek) decryptWithAAD(ciphertext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(k.key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.Errorf("Invalid nonce length")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}