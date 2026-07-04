// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package azurekeyvault

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
)

const (
	blobFormatVersion byte = 1

	algAzureRSAOAEP256 byte = 1
	algLocalTest       byte = 2

	wrappingKeySize = 32

	maxLenPrefixedField = 1<<16 - 1

	gcmNonceSize = 12
	gcmTagSize   = 16
	minGCMBlobSz = gcmNonceSize + gcmTagSize
)

var blobMagic = [4]byte{'O', 'A', 'K', 'V'}

var azTestKey = []byte("octelium-azure-keyvault-testkey!")

type store struct {
	store   *enterprisev1.SecretStore
	c       *azkeys.Client
	keyName string
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

	spec := opts.SecretStore.Spec.GetAzureKeyVault()
	if spec == nil {
		return nil, errors.Errorf("Not an Azure Key Vault store type")
	}

	ret := &store{
		store:   opts.SecretStore,
		keyName: spec.Key,
	}

	if ldflags.IsTest() {
		if ret.keyName == "" {
			ret.keyName = "octelium-test"
		}

		testKey, err := getTestKey()
		if err != nil {
			return nil, err
		}
		ret.testKey = testKey

		return ret, nil
	}

	if spec.VaultURL == "" {
		return nil, errors.Errorf("Empty Azure Key Vault vaultURL")
	}
	if spec.TenantID == "" {
		return nil, errors.Errorf("Empty Azure Key Vault tenantID")
	}
	if spec.ClientID == "" {
		return nil, errors.Errorf("Empty Azure Key Vault clientID")
	}
	if spec.Key == "" {
		return nil, errors.Errorf("Empty Azure Key Vault key")
	}

	cred, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		TenantID: spec.TenantID,
		ClientID: spec.ClientID,
	})
	if err != nil {
		return nil, err
	}

	c, err := azkeys.NewClient(spec.VaultURL, cred, nil)
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
		if len(valBytes) != wrappingKeySize {
			return nil, errors.Errorf("Invalid OCTELIUM_TEST_SS_SECRET length: got %d, want %d",
				len(valBytes), wrappingKeySize)
		}

		return append([]byte(nil), valBytes...), nil
	}

	return append([]byte(nil), azTestKey...), nil
}

func (s *store) Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error) {
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}
	if len(plaintext) == 0 {
		return nil, errors.Errorf("Empty plaintext")
	}

	wk, err := utilrand.GetRandomBytes(wrappingKeySize)
	if err != nil {
		return nil, err
	}
	defer zero(wk)

	var algID byte
	var keyVersion string
	var wrappedWK []byte

	if s.testKey != nil {
		algID = algLocalTest
		wrappedWK, err = gcmSeal(s.testKey, wk, s.wrapAAD(algID, s.keyName, keyVersion, uid))
		if err != nil {
			return nil, err
		}
	} else {
		if s.c == nil {
			return nil, errors.Errorf("Azure Key Vault client is not initialized")
		}

		algID = algAzureRSAOAEP256
		alg := azkeys.EncryptionAlgorithmRSAOAEP256

		resp, err := s.c.WrapKey(ctx, s.keyName, "", azkeys.KeyOperationParameters{
			Algorithm: &alg,
			Value:     wk,
		}, nil)
		if err != nil {
			return nil, err
		}
		if len(resp.Result) == 0 {
			return nil, errors.Errorf("Azure Key Vault returned an empty wrapped key")
		}
		if resp.KID == nil || resp.KID.Version() == "" {
			return nil, errors.Errorf("Azure Key Vault did not return a key version")
		}

		keyVersion = resp.KID.Version()
		wrappedWK = resp.Result
	}

	sealed, err := gcmSeal(wk, plaintext, s.dataAAD(algID, s.keyName, keyVersion, uid))
	if err != nil {
		return nil, err
	}

	return encodeBlob(algID, s.keyName, keyVersion, wrappedWK, sealed)
}

func (s *store) Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error) {
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}

	hdr, wrappedWK, sealed, err := decodeBlob(ciphertext)
	if err != nil {
		return nil, err
	}

	var wk []byte

	switch hdr.algID {
	case algLocalTest:
		if s.testKey == nil {
			return nil, errors.Errorf("Cannot unwrap a test-wrapped KEK blob outside of test mode")
		}

		wk, err = gcmOpen(s.testKey, wrappedWK, s.wrapAAD(hdr.algID, hdr.keyName, hdr.keyVersion, uid))
		if err != nil {
			return nil, err
		}

	case algAzureRSAOAEP256:
		if s.c == nil {
			return nil, errors.Errorf("Azure Key Vault client is not initialized")
		}
		if hdr.keyVersion == "" {
			return nil, errors.Errorf("Azure Key Vault blob is missing key version")
		}

		alg := azkeys.EncryptionAlgorithmRSAOAEP256

		resp, err := s.c.UnwrapKey(ctx, hdr.keyName, hdr.keyVersion, azkeys.KeyOperationParameters{
			Algorithm: &alg,
			Value:     wrappedWK,
		}, nil)
		if err != nil {
			return nil, errors.Wrapf(err,
				"Could not unwrap Azure Key Vault wrapping key: key=%s version=%s",
				hdr.keyName, hdr.keyVersion)
		}
		if len(resp.Result) == 0 {
			return nil, errors.Errorf("Azure Key Vault returned an empty unwrapped key")
		}

		wk = resp.Result

	default:
		return nil, errors.Errorf("Unsupported KEK wrap algorithm: %d", hdr.algID)
	}
	defer zero(wk)

	if len(wk) != wrappingKeySize {
		return nil, errors.Errorf("Unexpected unwrapped key size: got %d, want %d", len(wk), wrappingKeySize)
	}

	plaintext, err := gcmOpen(wk, sealed, s.dataAAD(hdr.algID, hdr.keyName, hdr.keyVersion, uid))
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (s *store) dataAAD(algID byte, keyName, keyVersion, uid string) []byte {
	return []byte(fmt.Sprintf(
		"octelium.secretman.kek.azurekeyvault.v1\x00alg=%d\x00key=%s\x00key_version=%s\x00uid=%s",
		algID, keyName, keyVersion, uid,
	))
}

func (s *store) wrapAAD(algID byte, keyName, keyVersion, uid string) []byte {
	return []byte(fmt.Sprintf(
		"octelium.secretman.kek.azurekeyvault.wrap.v1\x00alg=%d\x00key=%s\x00key_version=%s\x00uid=%s",
		algID, keyName, keyVersion, uid,
	))
}

func (s *store) UID() string {
	if s == nil || s.store == nil || s.store.Metadata == nil {
		return ""
	}

	return s.store.Metadata.Uid
}

func (s *store) Close() error {
	return nil
}

func (s *store) Initialize(ctx context.Context) error {
	if s == nil {
		return errors.Errorf("Nil Azure Key Vault store")
	}
	if s.store == nil {
		return errors.Errorf("Nil SecretStore")
	}
	if s.store.Metadata == nil {
		return errors.Errorf("Nil SecretStore metadata")
	}
	if s.store.Metadata.Uid == "" {
		return errors.Errorf("Empty SecretStore uid")
	}
	if s.keyName == "" {
		return errors.Errorf("Empty Azure Key Vault key")
	}

	if s.testKey != nil {
		if len(s.testKey) != wrappingKeySize {
			return errors.Errorf("Invalid test wrapping key size: got %d, want %d",
				len(s.testKey), wrappingKeySize)
		}
	} else if s.c == nil {
		return errors.Errorf("Azure Key Vault client is not initialized")
	}

	return s.selfTest(ctx)
}

func (s *store) selfTest(ctx context.Context) error {
	plaintext, err := utilrand.GetRandomBytes(wrappingKeySize)
	if err != nil {
		return err
	}
	defer zero(plaintext)

	blob, err := s.Encrypt(ctx, "self-test", plaintext)
	if err != nil {
		return err
	}

	out, err := s.Decrypt(ctx, "self-test", blob)
	if err != nil {
		return err
	}
	defer zero(out)

	if !bytes.Equal(out, plaintext) {
		return errors.Errorf("Azure Key Vault store self-test failed")
	}

	return nil
}

type blobHeader struct {
	algID      byte
	keyName    string
	keyVersion string
}

func encodeBlob(algID byte, keyName, keyVersion string, wrappedWK, sealed []byte) ([]byte, error) {
	if algID == 0 {
		return nil, errors.Errorf("Invalid KEK wrap algorithm")
	}
	if keyName == "" {
		return nil, errors.Errorf("Empty KEK key name")
	}
	if len(wrappedWK) == 0 {
		return nil, errors.Errorf("Empty wrapped key")
	}
	if len(sealed) < minGCMBlobSz {
		return nil, errors.Errorf("Sealed payload is too short")
	}

	keyNameBytes := []byte(keyName)
	keyVersionBytes := []byte(keyVersion)

	out := make([]byte, 0,
		len(blobMagic)+2+
			2+len(keyNameBytes)+
			2+len(keyVersionBytes)+
			2+len(wrappedWK)+
			len(sealed))

	out = append(out, blobMagic[:]...)
	out = append(out, blobFormatVersion)
	out = append(out, algID)

	var err error
	out, err = appendLenPrefixed(out, keyNameBytes)
	if err != nil {
		return nil, err
	}

	out, err = appendLenPrefixed(out, keyVersionBytes)
	if err != nil {
		return nil, err
	}

	out, err = appendLenPrefixed(out, wrappedWK)
	if err != nil {
		return nil, err
	}

	out = append(out, sealed...)

	return out, nil
}

func decodeBlob(blob []byte) (*blobHeader, []byte, []byte, error) {
	if len(blob) < len(blobMagic)+2 {
		return nil, nil, nil, errors.Errorf("KEK blob is too short")
	}
	if !bytes.Equal(blob[:len(blobMagic)], blobMagic[:]) {
		return nil, nil, nil, errors.Errorf("Invalid KEK blob magic")
	}

	buf := blob[len(blobMagic):]

	if buf[0] != blobFormatVersion {
		return nil, nil, nil, errors.Errorf("Unsupported KEK blob format version: %d", buf[0])
	}
	buf = buf[1:]

	hdr := &blobHeader{
		algID: buf[0],
	}
	buf = buf[1:]

	keyName, buf, err := readLenPrefixed(buf)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(keyName) == 0 {
		return nil, nil, nil, errors.Errorf("KEK blob has an empty key name")
	}
	hdr.keyName = string(keyName)

	keyVersion, buf, err := readLenPrefixed(buf)
	if err != nil {
		return nil, nil, nil, err
	}
	hdr.keyVersion = string(keyVersion)

	wrappedWK, buf, err := readLenPrefixed(buf)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(wrappedWK) == 0 {
		return nil, nil, nil, errors.Errorf("KEK blob has an empty wrapped key")
	}

	if len(buf) < minGCMBlobSz {
		return nil, nil, nil, errors.Errorf("KEK blob sealed payload is too short")
	}

	return hdr, wrappedWK, buf, nil
}

func appendLenPrefixed(dst, b []byte) ([]byte, error) {
	if len(b) > maxLenPrefixedField {
		return nil, errors.Errorf("KEK blob field is too large: %d", len(b))
	}

	var l [2]byte
	binary.BigEndian.PutUint16(l[:], uint16(len(b)))
	dst = append(dst, l[:]...)
	dst = append(dst, b...)

	return dst, nil
}

func readLenPrefixed(buf []byte) ([]byte, []byte, error) {
	if len(buf) < 2 {
		return nil, nil, errors.Errorf("KEK blob has a truncated length prefix")
	}

	n := int(binary.BigEndian.Uint16(buf[:2]))
	buf = buf[2:]

	if len(buf) < n {
		return nil, nil, errors.Errorf("KEK blob has a truncated field")
	}

	return buf[:n], buf[n:], nil
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
	if len(blob) < nonceSize+gcmTagSize {
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
