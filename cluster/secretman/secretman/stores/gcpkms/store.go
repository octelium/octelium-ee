// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package gcpkms

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"os"
	"strings"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	testKeySize = 32

	integrityRetryCount = 2
)

var gcpTestKey = sha256.Sum256([]byte("octelium-gcpkms-test-key-v1"))

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

type store struct {
	uid     string
	keyPath string

	c       *kms.KeyManagementClient
	testKey []byte
}

var _ stores.Store = (*store)(nil)

func NewStore(ctx context.Context, opts *stores.StoreOpts) (*store, error) {
	if opts == nil {
		return nil, errors.Errorf("Nil StoreOpts")
	}
	if opts.SecretStore == nil {
		return nil, errors.Errorf("Nil SecretStore")
	}
	if opts.SecretStore.Metadata == nil {
		return nil, errors.Errorf("Nil SecretStore metadata")
	}
	if opts.SecretStore.Metadata.Uid == "" {
		return nil, errors.Errorf("Empty SecretStore uid")
	}
	if opts.SecretStore.Spec == nil {
		return nil, errors.Errorf("Nil SecretStore spec")
	}

	spec := opts.SecretStore.Spec.GetGoogleCloudKeyManagementService()
	if spec == nil {
		return nil, errors.Errorf("Not a Google Cloud KMS store type")
	}

	ret := &store{
		uid: opts.SecretStore.Metadata.Uid,
	}

	if ldflags.IsTest() {
		testKey, err := getTestKey()
		if err != nil {
			return nil, err
		}

		ret.testKey = testKey

		return ret, nil
	}

	if spec.Project == "" {
		return nil, errors.Errorf("Empty Google Cloud KMS project")
	}
	if spec.Location == "" {
		return nil, errors.Errorf("Empty Google Cloud KMS location")
	}
	if spec.KeyRing == "" {
		return nil, errors.Errorf("Empty Google Cloud KMS keyRing")
	}
	if spec.Key == "" {
		return nil, errors.Errorf("Empty Google Cloud KMS key")
	}

	ret.keyPath = getKeyPath(
		spec.Project,
		spec.Location,
		spec.KeyRing,
		spec.Key,
	)

	c, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, err
	}
	ret.c = c

	return ret, nil
}

func getTestKey() ([]byte, error) {
	if val := strings.TrimSpace(os.Getenv("OCTELIUM_TEST_SS_SECRET")); val != "" {
		valBytes, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return nil, err
		}
		if len(valBytes) != testKeySize {
			return nil, errors.Errorf(
				"Invalid OCTELIUM_TEST_SS_SECRET length: got %d, want %d",
				len(valBytes),
				testKeySize,
			)
		}

		return append([]byte(nil), valBytes...), nil
	}

	return append([]byte(nil), gcpTestKey[:]...), nil
}

func (s *store) Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error) {
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}
	if len(plaintext) == 0 {
		return nil, errors.Errorf("Empty plaintext")
	}

	aad := dataAAD(uid)

	if s.testKey != nil {
		return gcmSeal(s.testKey, plaintext, aad)
	}

	if s.c == nil {
		return nil, errors.Errorf("Google Cloud KMS client is not initialized")
	}
	if s.keyPath == "" {
		return nil, errors.Errorf("Empty Google Cloud KMS key path")
	}

	req := &kmspb.EncryptRequest{
		Name:                              s.keyPath,
		Plaintext:                         plaintext,
		AdditionalAuthenticatedData:       aad,
		PlaintextCrc32C:                   wrapperspb.Int64(int64(crc32c(plaintext))),
		AdditionalAuthenticatedDataCrc32C: wrapperspb.Int64(int64(crc32c(aad))),
	}

	var lastErr error

	for attempt := 0; attempt <= integrityRetryCount; attempt++ {
		res, err := s.c.Encrypt(ctx, req)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"Could not encrypt with Google Cloud KMS key %q",
				s.keyPath,
			)
		}

		if !res.VerifiedPlaintextCrc32C {
			lastErr = errors.Errorf(
				"Google Cloud KMS did not verify request plaintext integrity",
			)
			continue
		}

		if !res.VerifiedAdditionalAuthenticatedDataCrc32C {
			lastErr = errors.Errorf(
				"Google Cloud KMS did not verify request AAD integrity",
			)
			continue
		}

		if len(res.Ciphertext) == 0 {
			lastErr = errors.Errorf(
				"Google Cloud KMS returned empty ciphertext",
			)
			continue
		}

		if res.CiphertextCrc32C == nil {
			lastErr = errors.Errorf(
				"Google Cloud KMS response is missing ciphertext CRC32C",
			)
			continue
		}

		if int64(crc32c(res.Ciphertext)) != res.CiphertextCrc32C.GetValue() {
			lastErr = errors.Errorf(
				"Google Cloud KMS response ciphertext integrity check failed",
			)
			continue
		}

		return res.Ciphertext, nil
	}

	return nil, lastErr
}

func (s *store) Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error) {
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}
	if len(ciphertext) == 0 {
		return nil, errors.Errorf("Empty ciphertext")
	}

	aad := dataAAD(uid)

	if s.testKey != nil {
		return gcmOpen(s.testKey, ciphertext, aad)
	}

	if s.c == nil {
		return nil, errors.Errorf("Google Cloud KMS client is not initialized")
	}
	if s.keyPath == "" {
		return nil, errors.Errorf("Empty Google Cloud KMS key path")
	}

	req := &kmspb.DecryptRequest{
		Name:                              s.keyPath,
		Ciphertext:                        ciphertext,
		AdditionalAuthenticatedData:       aad,
		CiphertextCrc32C:                  wrapperspb.Int64(int64(crc32c(ciphertext))),
		AdditionalAuthenticatedDataCrc32C: wrapperspb.Int64(int64(crc32c(aad))),
	}

	var lastErr error

	for attempt := 0; attempt <= integrityRetryCount; attempt++ {
		res, err := s.c.Decrypt(ctx, req)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"Could not decrypt with Google Cloud KMS key %q",
				s.keyPath,
			)
		}

		if res.PlaintextCrc32C == nil {
			lastErr = errors.Errorf(
				"Google Cloud KMS response is missing plaintext CRC32C",
			)
			continue
		}

		if int64(crc32c(res.Plaintext)) != res.PlaintextCrc32C.GetValue() {
			lastErr = errors.Errorf(
				"Google Cloud KMS response plaintext integrity check failed",
			)
			continue
		}

		return res.Plaintext, nil
	}

	return nil, lastErr
}

func (s *store) UID() string {
	if s == nil {
		return ""
	}

	return s.uid
}

func (s *store) Close() error {
	if s == nil || s.c == nil {
		return nil
	}

	return s.c.Close()
}

func (s *store) Initialize(ctx context.Context) error {
	if s == nil {
		return errors.Errorf("Nil Google Cloud KMS store")
	}
	if s.uid == "" {
		return errors.Errorf("Empty SecretStore uid")
	}

	if s.testKey != nil {
		if len(s.testKey) != testKeySize {
			return errors.Errorf(
				"Invalid test key length: got %d, want %d",
				len(s.testKey),
				testKeySize,
			)
		}
	} else {
		if s.c == nil {
			return errors.Errorf("Google Cloud KMS client is not initialized")
		}
		if s.keyPath == "" {
			return errors.Errorf("Empty Google Cloud KMS key path")
		}
	}

	return s.selfTest(ctx)
}

func (s *store) selfTest(ctx context.Context) error {
	plaintext, err := utilrand.GetRandomBytes(32)
	if err != nil {
		return err
	}
	defer zero(plaintext)

	ciphertext, err := s.Encrypt(ctx, "self-test", plaintext)
	if err != nil {
		return errors.Wrap(err, "Google Cloud KMS store self-test encryption failed")
	}

	out, err := s.Decrypt(ctx, "self-test", ciphertext)
	if err != nil {
		return errors.Wrap(err, "Google Cloud KMS store self-test decryption failed")
	}
	defer zero(out)

	if !bytes.Equal(out, plaintext) {
		return errors.Errorf("Google Cloud KMS store self-test failed")
	}

	return nil
}

func getKeyPath(project, location, keyRing, key string) string {
	return fmt.Sprintf(
		"projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		project,
		location,
		keyRing,
		key,
	)
}

func dataAAD(uid string) []byte {
	return []byte(
		"octelium.secretman.kek.gcpkms.v1\x00uid=" + uid,
	)
}

func crc32c(data []byte) uint32 {
	return crc32.Checksum(data, crc32cTable)
}

func gcmSeal(key, plaintext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
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

	return aesgcm.Seal(nonce, nonce, plaintext, aad), nil
}

func gcmOpen(key, blob, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()

	if len(blob) < nonceSize+aesgcm.Overhead() {
		return nil, errors.Errorf("Invalid AES-GCM blob length")
	}

	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]

	return aesgcm.Open(nil, nonce, ciphertext, aad)
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
