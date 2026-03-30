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
	"os"

	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
)

type store struct {
	// c           kubernetes.Interface
	secretStore *enterprisev1.SecretStore
	secret      []byte
}

var secTest = utilrand.GetRandomBytesMust(32)

func NewStore(ctx context.Context, o *stores.StoreOpts,

// k8sC kubernetes.Interface
) (*store, error) {
	ret := &store{
		// c:           k8sC,
		secretStore: o.SecretStore,
	}

	var err error
	if ldflags.IsTest() {
		ret.secret = secTest
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
	/*
		secret, err := utilrand.GetRandomBytes(32)
		if err != nil {
			return err
		}
		_, err = k8sutils.CreateOrUpdateSecret(ctx, s.c, &k8scorev1.Secret{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name:      s.secretStore.Metadata.Name,
				Namespace: vutils.K8sNS,
			},
			Data: map[string][]byte{
				"data": secret,
			},
		})

		return err
	*/
}
