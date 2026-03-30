// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package stores

import (
	"context"

	"github.com/octelium/octelium/apis/main/enterprisev1"
)

type Store interface {
	UID() string
	Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error)
	Close() error
	Initialize(ctx context.Context) error
}

type StoreOpts struct {
	SecretStore *enterprisev1.SecretStore
}
