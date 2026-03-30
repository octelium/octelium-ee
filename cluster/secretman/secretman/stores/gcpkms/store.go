// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package gcpkms

import (
	"context"
	"fmt"

	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/pkg/errors"
	"google.golang.org/api/option"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
)

type store struct {
	c *kms.KeyManagementClient

	store *enterprisev1.SecretStore
}

func NewStore(ctx context.Context, opts *stores.StoreOpts) (*store, error) {
	ret := &store{
		store: opts.SecretStore,
	}

	spec := opts.SecretStore.Spec.GetGoogleCloudKeyManagementService()
	if spec == nil {
		return nil, errors.Errorf("Not a google secret manager store type")
	}

	c, err := kms.NewKeyManagementClient(ctx,

		option.WithCredentialsJSON(nil))
	if err != nil {
		return nil, err
	}

	ret.c = c

	return ret, nil
}

func (s *store) Close() error {
	s.c.Close()
	return nil
}

func (s *store) Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error) {
	res, err := s.c.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      s.getKeyPath(),
		Plaintext: plaintext,
	})
	if err != nil {
		return nil, err
	}

	return res.Ciphertext, nil
}

func (s *store) Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error) {
	res, err := s.c.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       s.getKeyPath(),
		Ciphertext: ciphertext,
	})
	if err != nil {
		return nil, err
	}

	return res.Plaintext, nil
}

func (s *store) UID() string {
	return s.store.Metadata.Uid
}

func (s *store) Initialize(ctx context.Context) error {
	return nil
}

func (s *store) getKeyPath() string {

	spec := s.store.Spec.GetGoogleCloudKeyManagementService()
	return fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		spec.Project, spec.Location, spec.KeyRing, spec.Key)
}
