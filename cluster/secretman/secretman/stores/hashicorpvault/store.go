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
	"encoding/base64"
	"fmt"

	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/identity"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
)

type store struct {
	c *vault.Client

	store   *enterprisev1.SecretStore
	keyName string
}

const testToken = "root"

func NewStore(ctx context.Context, opts *stores.StoreOpts) (*store, error) {
	ret := &store{
		store: opts.SecretStore,
	}
	var err error
	if opts.SecretStore.Spec.GetHashicorpVault() == nil {
		return nil, errors.Errorf("Not a hashicorp vault store type")
	}

	spec := opts.SecretStore.Spec.GetHashicorpVault()

	cfg := vault.DefaultConfig()
	cfg.Address = spec.Address

	ret.keyName = spec.Key
	ret.c, err = vault.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if ldflags.IsTest() {
		ret.c.SetToken(testToken)
	} else {

		tkn, err := identity.GetIdentityToken(ctx, opts.SecretStore)
		if err != nil {
			return nil, err
		}
		authK8s, err := kubernetes.NewKubernetesAuth(spec.Role, kubernetes.WithServiceAccountToken(tkn.Token))
		if err != nil {
			return nil, err
		}

		_, err = ret.c.Auth().Login(ctx, authK8s)
		if err != nil {
			return nil, err
		}
	}

	// ret.kvC = ret.c.KVv2(spec.MountPath)

	return ret, nil
}

func (s *store) Close() error {

	return nil
}

func (s *store) Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error) {

	encryptData := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}
	encryptResp, err := s.c.Logical().Write(fmt.Sprintf("transit/encrypt/%s", s.keyName), encryptData)
	if err != nil {
		return nil, err
	}
	ciphertext := encryptResp.Data["ciphertext"].(string)

	return []byte(ciphertext), nil
}

func (s *store) Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error) {

	decryptData := map[string]interface{}{
		"ciphertext": string(ciphertext),
	}

	decryptResp, err := s.c.Logical().Write(fmt.Sprintf("transit/decrypt/%s", s.keyName), decryptData)
	if err != nil {
		return nil, err
	}
	base64Plaintext := decryptResp.Data["plaintext"].(string)
	plaintext, err := base64.StdEncoding.DecodeString(base64Plaintext)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (s *store) UID() string {
	return s.store.Metadata.Uid
}

func (s *store) Initialize(ctx context.Context) error {
	return nil
}
