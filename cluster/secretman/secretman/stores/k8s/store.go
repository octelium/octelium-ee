// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package k8s

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"strings"

	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
)

const kekPath = "/octelium-kek"

var secTest = sha256.Sum256(
	[]byte("octelium-kubernetes-kek-test-key-v1"),
)

type store struct {
	uid    string
	secret []byte
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
	if opts.SecretStore.Spec.GetKubernetes() == nil {
		return nil, errors.Errorf("Not a Kubernetes store type")
	}

	var key []byte
	var err error

	if ldflags.IsTest() {
		key, err = getTestKey()
		if err != nil {
			return nil, err
		}
	} else {
		key, err = os.ReadFile(kekPath)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"Could not read Kubernetes KEK from %q",
				kekPath,
			)
		}
	}

	if err := validateKey(key); err != nil {
		zero(key)
		return nil, err
	}

	ret := &store{
		uid:    opts.SecretStore.Metadata.Uid,
		secret: append([]byte(nil), key...),
	}

	zero(key)

	return ret, nil
}

func getTestKey() ([]byte, error) {
	if val := strings.TrimSpace(os.Getenv("OCTELIUM_TEST_SS_SECRET")); val != "" {
		key, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return nil, err
		}

		if err := validateKey(key); err != nil {
			zero(key)
			return nil, err
		}

		return key, nil
	}

	return append([]byte(nil), secTest[:]...), nil
}

func validateKey(key []byte) error {
	switch len(key) {
	case 16, 24, 32:
		return nil
	default:
		return errors.Errorf(
			"Invalid Kubernetes KEK size: got %d bytes, want 16, 24, or 32",
			len(key),
		)
	}
}

func (s *store) Close() error {
	if s == nil {
		return nil
	}

	zero(s.secret)
	s.secret = nil

	return nil
}

func (s *store) Encrypt(
	ctx context.Context,
	uid string,
	plaintext []byte,
) ([]byte, error) {
	if s == nil {
		return nil, errors.Errorf("Nil Kubernetes store")
	}
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}
	if len(plaintext) == 0 {
		return nil, errors.Errorf("Empty plaintext")
	}
	if err := validateKey(s.secret); err != nil {
		return nil, err
	}

	return gcmSeal(
		s.secret,
		plaintext,
		dataAAD(uid),
	)
}

func (s *store) Decrypt(
	ctx context.Context,
	uid string,
	ciphertext []byte,
) ([]byte, error) {
	if s == nil {
		return nil, errors.Errorf("Nil Kubernetes store")
	}
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}
	if len(ciphertext) == 0 {
		return nil, errors.Errorf("Empty ciphertext")
	}
	if err := validateKey(s.secret); err != nil {
		return nil, err
	}

	// Current format: ciphertext is authenticated with AAD bound to the DEK UID.
	plaintext, err := gcmOpen(
		s.secret,
		ciphertext,
		dataAAD(uid),
	)
	if err == nil {
		return plaintext, nil
	}

	// Legacy format: ciphertext written by older Octelium versions used nil AAD.
	// A ciphertext written with the current AAD-bound format cannot successfully
	// decrypt through this path because the GCM authentication tag also binds AAD.
	plaintext, legacyErr := gcmOpen(
		s.secret,
		ciphertext,
		nil,
	)
	if legacyErr == nil {
		return plaintext, nil
	}

	return nil, errors.Wrap(
		err,
		"Could not decrypt Kubernetes KEK ciphertext using current or legacy authentication mode",
	)
}

func (s *store) UID() string {
	if s == nil {
		return ""
	}

	return s.uid
}

func (s *store) Initialize(ctx context.Context) error {
	if s == nil {
		return errors.Errorf("Nil Kubernetes store")
	}
	if s.uid == "" {
		return errors.Errorf("Empty SecretStore uid")
	}
	if err := validateKey(s.secret); err != nil {
		return err
	}

	return s.selfTest(ctx)
}

func (s *store) selfTest(ctx context.Context) error {
	plaintext, err := utilrand.GetRandomBytes(32)
	if err != nil {
		return err
	}
	defer zero(plaintext)

	const uid = "self-test"

	ciphertext, err := s.Encrypt(
		ctx,
		uid,
		plaintext,
	)
	if err != nil {
		return errors.Wrap(
			err,
			"Kubernetes store self-test encryption failed",
		)
	}

	out, err := s.Decrypt(
		ctx,
		uid,
		ciphertext,
	)
	if err != nil {
		return errors.Wrap(
			err,
			"Kubernetes store self-test decryption failed",
		)
	}
	defer zero(out)

	if !bytes.Equal(out, plaintext) {
		return errors.Errorf(
			"Kubernetes store self-test round trip failed",
		)
	}

	if _, err := s.Decrypt(
		ctx,
		"self-test-wrong-uid",
		ciphertext,
	); err == nil {
		return errors.Errorf(
			"Kubernetes store self-test failed: AAD-bound ciphertext was accepted with the wrong uid",
		)
	}

	legacyCiphertext, err := gcmSeal(
		s.secret,
		plaintext,
		nil,
	)
	if err != nil {
		return errors.Wrap(
			err,
			"Kubernetes store self-test legacy encryption failed",
		)
	}

	legacyOut, err := s.Decrypt(
		ctx,
		uid,
		legacyCiphertext,
	)
	if err != nil {
		return errors.Wrap(
			err,
			"Kubernetes store self-test legacy decryption failed",
		)
	}
	defer zero(legacyOut)

	if !bytes.Equal(legacyOut, plaintext) {
		return errors.Errorf(
			"Kubernetes store self-test legacy compatibility failed",
		)
	}

	return nil
}

func dataAAD(uid string) []byte {
	return []byte(
		"octelium.secretman.kek.kubernetes.v1\x00uid=" + uid,
	)
}

func gcmSeal(
	key []byte,
	plaintext []byte,
	aad []byte,
) ([]byte, error) {
	aesgcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	nonce, err := utilrand.GetRandomBytes(
		aesgcm.NonceSize(),
	)
	if err != nil {
		return nil, err
	}

	return aesgcm.Seal(
		nonce,
		nonce,
		plaintext,
		aad,
	), nil
}

func gcmOpen(
	key []byte,
	blob []byte,
	aad []byte,
) ([]byte, error) {
	aesgcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(blob) < nonceSize+aesgcm.Overhead() {
		return nil, errors.Errorf(
			"Invalid AES-GCM blob length",
		)
	}

	nonce := blob[:nonceSize]
	ciphertext := blob[nonceSize:]

	return aesgcm.Open(
		nil,
		nonce,
		ciphertext,
		aad,
	)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
