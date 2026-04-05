// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package azurekeyvault

import (
	"context"
	"fmt"

	"github.com/octelium/octelium-ee/cluster/secretman/secretman/identity"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type store struct {
	c       *azkeys.Client
	keyName string
	store   *enterprisev1.SecretStore
}

type tknStore struct {
	result *confidential.AuthResult
}

func (c *tknStore) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {

	return azcore.AccessToken{
		Token:     c.result.AccessToken,
		ExpiresOn: c.result.ExpiresOn,
	}, nil
}

var _ azcore.TokenCredential = (*tknStore)(nil)

func NewStore(ctx context.Context, opts *stores.StoreOpts) (*store, error) {
	ret := &store{
		store: opts.SecretStore,
	}

	spec := opts.SecretStore.Spec.GetAzureKeyVault()
	/*
		cred, err := azidentity.NewClientSecretCredential(
			spec.TenantID, spec.ClientID, "",
			&azidentity.ClientSecretCredentialOptions{})
		if err != nil {
			return nil, err
		}
	*/

	cred := confidential.NewCredFromAssertionCallback(
		func(ctx context.Context, o confidential.AssertionRequestOptions) (string, error) {
			tkn, err := identity.GetIdentityToken(ctx, opts.SecretStore)
			if err != nil {
				return "", err
			}
			return tkn.Token, nil
		})

	client, err := confidential.New(
		fmt.Sprintf("https://login.microsoftonline.com/%s", spec.TenantID),
		spec.ClientID,
		cred,
	)
	if err != nil {
		return nil, err
	}

	result, err := client.AcquireTokenByCredential(ctx, []string{"https://vault.azure.net/.default"})
	if err != nil {
		return nil, err
	}

	ret.c, err = azkeys.NewClient(spec.VaultURL, &tknStore{result: &result}, nil)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *store) Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error) {
	alg := azkeys.EncryptionAlgorithmA256GCM
	res, err := s.c.WrapKey(ctx, s.keyName, "", azkeys.KeyOperationParameters{
		Algorithm: &alg,
		Value:     plaintext,
	}, &azkeys.WrapKeyOptions{})
	if err != nil {
		return nil, err
	}

	return res.Result, nil
}

func (s *store) Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error) {
	alg := azkeys.EncryptionAlgorithmA256GCM
	res, err := s.c.UnwrapKey(ctx, s.keyName, "", azkeys.KeyOperationParameters{
		Algorithm: &alg,
		Value:     ciphertext,
	}, &azkeys.UnwrapKeyOptions{})
	if err != nil {
		return nil, err
	}

	return res.Result, nil
}

func (s *store) Close() error {
	return nil
}

func (s *store) UID() string {
	return s.store.Metadata.Uid
}

func (s *store) Initialize(ctx context.Context) error {
	return nil
}
