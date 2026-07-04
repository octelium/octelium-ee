// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package awskms

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/identity"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
)

const (
	testKeySize = 32

	defaultCredentialProbeTimeout = 5 * time.Second
	identityTokenTimeout          = 10 * time.Second

	maxRoleSessionNameLen = 64
)

var awsTestKey = sha256.Sum256([]byte("octelium-awskms-test-key-v1"))

type store struct {
	uid string

	c     *kms.Client
	keyID string

	testKey []byte
}

var _ stores.Store = (*store)(nil)

type tokenRetriever struct {
	ss *enterprisev1.SecretStore
}

var _ stscreds.IdentityTokenRetriever = (*tokenRetriever)(nil)

func (t *tokenRetriever) GetIdentityToken() ([]byte, error) {
	if t == nil || t.ss == nil {
		return nil, errors.Errorf("Nil SecretStore")
	}

	ctx, cancel := context.WithTimeout(context.Background(), identityTokenTimeout)
	defer cancel()

	tkn, err := identity.GetIdentityToken(ctx, t.ss)
	if err != nil {
		return nil, err
	}
	if tkn == nil {
		return nil, errors.Errorf("Nil Kubernetes identity token")
	}

	token := strings.TrimSpace(tkn.Token)
	if token == "" {
		return nil, errors.Errorf("Empty Kubernetes identity token")
	}

	return []byte(token), nil
}

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

	spec := opts.SecretStore.Spec.GetAwsKeyManagementService()
	if spec == nil {
		return nil, errors.Errorf("Not an AWS KMS store type")
	}

	ret := &store{
		uid:   opts.SecretStore.Metadata.Uid,
		keyID: spec.KeyID,
	}

	if ldflags.IsTest() {
		testKey, err := getTestKey()
		if err != nil {
			return nil, err
		}

		ret.testKey = testKey

		return ret, nil
	}

	if spec.KeyID == "" {
		return nil, errors.Errorf("Empty AWS KMS keyID")
	}
	if spec.Region == "" {
		return nil, errors.Errorf("Empty AWS KMS region")
	}

	cfg, err := buildAWSConfig(ctx, opts.SecretStore)
	if err != nil {
		return nil, err
	}

	ret.c = kms.NewFromConfig(cfg)

	return ret, nil
}

func buildAWSConfig(ctx context.Context, ss *enterprisev1.SecretStore) (aws.Config, error) {
	if ss == nil || ss.Spec == nil {
		return aws.Config{}, errors.Errorf("Invalid SecretStore")
	}

	spec := ss.Spec.GetAwsKeyManagementService()
	if spec == nil {
		return aws.Config{}, errors.Errorf("Not an AWS KMS store type")
	}

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(spec.Region),
	)
	if err != nil {
		return aws.Config{}, err
	}

	if cfg.Credentials != nil {
		probeCtx, cancel := context.WithTimeout(ctx, defaultCredentialProbeTimeout)
		_, probeErr := cfg.Credentials.Retrieve(probeCtx)
		cancel()

		if probeErr == nil {
			return cfg, nil
		}
	}

	if spec.RoleARN == "" {
		return aws.Config{}, errors.Errorf(
			"Could not retrieve AWS credentials from the default credential chain and no roleARN is configured",
		)
	}

	stsClient := sts.NewFromConfig(
		cfg,
		func(o *sts.Options) {
			o.Credentials = aws.AnonymousCredentials{}
		},
	)

	provider := stscreds.NewWebIdentityRoleProvider(
		stsClient,
		spec.RoleARN,
		&tokenRetriever{ss: ss},
		func(o *stscreds.WebIdentityRoleOptions) {
			o.RoleSessionName = roleSessionName(vutils.GetMyRegionName())
		},
	)

	creds := aws.NewCredentialsCache(provider)

	if _, err := creds.Retrieve(ctx); err != nil {
		return aws.Config{}, errors.Wrap(
			err,
			"Could not retrieve AWS credentials using Kubernetes web identity",
		)
	}

	cfg.Credentials = creds

	return cfg, nil
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

	return append([]byte(nil), awsTestKey[:]...), nil
}

func (s *store) Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error) {
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}
	if len(plaintext) == 0 {
		return nil, errors.Errorf("Empty plaintext")
	}

	if s.testKey != nil {
		return gcmSeal(s.testKey, plaintext, dataAAD(uid))
	}

	if s.c == nil {
		return nil, errors.Errorf("AWS KMS client is not initialized")
	}
	if s.keyID == "" {
		return nil, errors.Errorf("Empty AWS KMS keyID")
	}

	res, err := s.c.Encrypt(ctx, &kms.EncryptInput{
		KeyId:             aws.String(s.keyID),
		Plaintext:         plaintext,
		EncryptionContext: encryptionContext(uid),
	})
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"Could not encrypt with AWS KMS key %q",
			s.keyID,
		)
	}
	if len(res.CiphertextBlob) == 0 {
		return nil, errors.Errorf("AWS KMS returned empty ciphertext")
	}

	return res.CiphertextBlob, nil
}

func (s *store) Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error) {
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}
	if len(ciphertext) == 0 {
		return nil, errors.Errorf("Empty ciphertext")
	}

	if s.testKey != nil {
		return gcmOpen(s.testKey, ciphertext, dataAAD(uid))
	}

	if s.c == nil {
		return nil, errors.Errorf("AWS KMS client is not initialized")
	}
	if s.keyID == "" {
		return nil, errors.Errorf("Empty AWS KMS keyID")
	}

	res, err := s.c.Decrypt(ctx, &kms.DecryptInput{
		KeyId:             aws.String(s.keyID),
		CiphertextBlob:    ciphertext,
		EncryptionContext: encryptionContext(uid),
	})
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"Could not decrypt with AWS KMS key %q",
			s.keyID,
		)
	}
	if len(res.Plaintext) == 0 {
		return nil, errors.Errorf("AWS KMS returned empty plaintext")
	}

	return res.Plaintext, nil
}

func (s *store) UID() string {
	if s == nil {
		return ""
	}

	return s.uid
}

func (s *store) Close() error {
	return nil
}

func (s *store) Initialize(ctx context.Context) error {
	if s == nil {
		return errors.Errorf("Nil AWS KMS store")
	}
	if s.uid == "" {
		return errors.Errorf("Empty SecretStore uid")
	}
	if s.keyID == "" && s.testKey == nil {
		return errors.Errorf("Empty AWS KMS keyID")
	}

	if s.testKey != nil {
		if len(s.testKey) != testKeySize {
			return errors.Errorf(
				"Invalid test key length: got %d, want %d",
				len(s.testKey),
				testKeySize,
			)
		}
	} else if s.c == nil {
		return errors.Errorf("AWS KMS client is not initialized")
	}

	return s.selfTest(ctx)
}

func (s *store) selfTest(ctx context.Context) error {
	plaintext, err := utilrand.GetRandomBytes(testKeySize)
	if err != nil {
		return err
	}
	defer zero(plaintext)

	ciphertext, err := s.Encrypt(ctx, "self-test", plaintext)
	if err != nil {
		return errors.Wrap(err, "AWS KMS store self-test encryption failed")
	}

	out, err := s.Decrypt(ctx, "self-test", ciphertext)
	if err != nil {
		return errors.Wrap(err, "AWS KMS store self-test decryption failed")
	}
	defer zero(out)

	if !bytes.Equal(out, plaintext) {
		return errors.Errorf("AWS KMS store self-test failed")
	}

	return nil
}

func encryptionContext(uid string) map[string]string {
	return map[string]string{
		"octelium.secretman.kek.awskms": "v1",
		"uid":                           uid,
	}
}

func dataAAD(uid string) []byte {
	return []byte("octelium.secretman.kek.awskms.v1\x00uid=" + uid)
}

func roleSessionName(region string) string {
	const prefix = "octelium-secretman-"

	var b strings.Builder
	b.Grow(len(prefix) + len(region))
	b.WriteString(prefix)

	for _, r := range region {
		if isRoleSessionNameChar(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}

	ret := strings.TrimRight(b.String(), "-")
	if ret == strings.TrimSuffix(prefix, "-") {
		ret = prefix + "default"
	}

	if len(ret) > maxRoleSessionNameLen {
		ret = ret[:maxRoleSessionNameLen]
	}

	return ret
}

func isRoleSessionNameChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	}

	switch r {
	case '_', '+', '=', ',', '.', '@', '-':
		return true
	default:
		return false
	}
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
