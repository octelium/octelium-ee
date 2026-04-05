// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package k8s

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"os"

	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
)

type store struct {
	secretStore *enterprisev1.SecretStore
	secret      []byte
}

var secTest = utilrand.GetRandomBytesMust(32)

func NewStore(ctx context.Context, o *stores.StoreOpts) (*store, error) {
	ret := &store{
		secretStore: o.SecretStore,
	}

	var err error
	if ldflags.IsTest() {
		if val := os.Getenv("OCTELIUM_TEST_SS_SECRET"); val != "" {
			valBytes, err := base64.StdEncoding.DecodeString(val)
			if err != nil {
				return nil, err
			}
			ret.secret = valBytes
		} else {
			ret.secret = secTest
		}

	} else {
		ret.secret, err = os.ReadFile("/octelium-kek")
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (s *store) Close() error {
	return nil
}

func (s *store) Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error) {

	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return nil, err
	}

	nonce, err := utilrand.GetRandomBytes(12)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

func (s *store) Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error) {

	block, err := aes.NewCipher(s.secret)
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

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (s *store) UID() string {
	return s.secretStore.Metadata.Uid
}

func (s *store) Initialize(ctx context.Context) error {
	return nil
}
