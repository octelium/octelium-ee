// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package hashicorpvault

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/identity"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	testToken = "root"

	defaultTransitMountPath        = "transit"
	defaultKubernetesAuthMountPath = "kubernetes"

	transitMountPathEnv        = "OCTELIUM_VAULT_TRANSIT_MOUNT_PATH"
	kubernetesAuthMountPathEnv = "OCTELIUM_VAULT_KUBERNETES_AUTH_MOUNT_PATH"

	authRetryInitialBackoff = 1 * time.Second
	authRetryMaxBackoff     = 30 * time.Second
)

type store struct {
	c *vault.Client

	store *enterprisev1.SecretStore

	uid          string
	role         string
	keyName      string
	transitMount string
	authMount    string

	cancel context.CancelFunc
	done   chan struct{}

	closeOnce sync.Once
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

	spec := opts.SecretStore.Spec.GetHashicorpVault()
	if spec == nil {
		return nil, errors.Errorf("Not a HashiCorp Vault store type")
	}

	if spec.Address == "" {
		return nil, errors.Errorf("Empty HashiCorp Vault address")
	}
	if spec.Key == "" {
		return nil, errors.Errorf("Empty HashiCorp Vault Transit key")
	}
	if !ldflags.IsTest() && spec.Role == "" {
		return nil, errors.Errorf("Empty HashiCorp Vault Kubernetes auth role")
	}

	if err := validateKeyName(spec.Key); err != nil {
		return nil, err
	}

	transitMount, err := getMountPath(
		os.Getenv(transitMountPathEnv),
		defaultTransitMountPath,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid Vault Transit mount path")
	}

	authMount, err := getMountPath(
		os.Getenv(kubernetesAuthMountPathEnv),
		defaultKubernetesAuthMountPath,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid Vault Kubernetes auth mount path")
	}

	cfg := vault.DefaultConfig()
	if cfg == nil {
		return nil, errors.Errorf("Could not create HashiCorp Vault configuration")
	}
	if cfg.Error != nil {
		return nil, errors.Wrap(cfg.Error, "Could not initialize HashiCorp Vault configuration")
	}

	cfg.Address = spec.Address

	client, err := vault.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	ret := &store{
		c:            client,
		store:        opts.SecretStore,
		uid:          opts.SecretStore.Metadata.Uid,
		role:         spec.Role,
		keyName:      spec.Key,
		transitMount: transitMount,
		authMount:    authMount,
	}

	if ldflags.IsTest() {
		ret.c.SetToken(testToken)

		if err := ret.validateTransitKey(ctx); err != nil {
			return nil, err
		}

		return ret, nil
	}

	loginSecret, err := ret.login(ctx)
	if err != nil {
		return nil, err
	}

	if err := ret.validateTransitKey(ctx); err != nil {
		return nil, err
	}

	authCtx, cancel := context.WithCancel(context.Background())
	ret.cancel = cancel
	ret.done = make(chan struct{})

	go ret.runAuthLifecycle(authCtx, loginSecret)

	return ret, nil
}

func (s *store) Close() error {
	if s == nil {
		return nil
	}

	s.closeOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}

		if s.done != nil {
			<-s.done
		}
	})

	return nil
}

func (s *store) Encrypt(ctx context.Context, uid string, plaintext []byte) ([]byte, error) {
	if s == nil || s.c == nil {
		return nil, errors.Errorf("HashiCorp Vault client is not initialized")
	}
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}
	if len(plaintext) == 0 {
		return nil, errors.Errorf("Empty plaintext")
	}

	encryptData := map[string]interface{}{
		"plaintext":       base64.StdEncoding.EncodeToString(plaintext),
		"associated_data": base64.StdEncoding.EncodeToString(dataAAD(uid)),
	}

	encryptResp, err := s.c.Logical().WriteWithContext(
		ctx,
		s.encryptPath(),
		encryptData,
	)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"Could not encrypt with HashiCorp Vault Transit key %q",
			s.keyName,
		)
	}
	if encryptResp == nil {
		return nil, errors.Errorf("Vault Transit encrypt returned nil response")
	}
	if encryptResp.Data == nil {
		return nil, errors.Errorf("Vault Transit encrypt returned empty response data")
	}

	ciphertext, ok := encryptResp.Data["ciphertext"].(string)
	if !ok || ciphertext == "" {
		return nil, errors.Errorf("Vault Transit encrypt response has invalid ciphertext")
	}

	return []byte(ciphertext), nil
}

func (s *store) Decrypt(ctx context.Context, uid string, ciphertext []byte) ([]byte, error) {
	if s == nil || s.c == nil {
		return nil, errors.Errorf("HashiCorp Vault client is not initialized")
	}
	if uid == "" {
		return nil, errors.Errorf("Empty uid")
	}
	if len(ciphertext) == 0 {
		return nil, errors.Errorf("Empty ciphertext")
	}

	decryptData := map[string]interface{}{
		"ciphertext":      string(ciphertext),
		"associated_data": base64.StdEncoding.EncodeToString(dataAAD(uid)),
	}

	decryptResp, err := s.c.Logical().WriteWithContext(
		ctx,
		s.decryptPath(),
		decryptData,
	)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"Could not decrypt with HashiCorp Vault Transit key %q",
			s.keyName,
		)
	}
	if decryptResp == nil {
		return nil, errors.Errorf("Vault Transit decrypt returned nil response")
	}
	if decryptResp.Data == nil {
		return nil, errors.Errorf("Vault Transit decrypt returned empty response data")
	}

	base64Plaintext, ok := decryptResp.Data["plaintext"].(string)
	if !ok || base64Plaintext == "" {
		return nil, errors.Errorf("Vault Transit decrypt response has invalid plaintext")
	}

	plaintext, err := base64.StdEncoding.DecodeString(base64Plaintext)
	if err != nil {
		return nil, errors.Wrap(err, "Could not decode Vault Transit plaintext")
	}
	if len(plaintext) == 0 {
		return nil, errors.Errorf("Vault Transit returned empty plaintext")
	}

	return plaintext, nil
}

func (s *store) UID() string {
	if s == nil {
		return ""
	}

	return s.uid
}

func (s *store) Initialize(ctx context.Context) error {
	if s == nil {
		return errors.Errorf("Nil HashiCorp Vault store")
	}
	if s.c == nil {
		return errors.Errorf("HashiCorp Vault client is not initialized")
	}
	if s.uid == "" {
		return errors.Errorf("Empty SecretStore uid")
	}
	if s.keyName == "" {
		return errors.Errorf("Empty HashiCorp Vault Transit key")
	}
	if s.transitMount == "" {
		return errors.Errorf("Empty HashiCorp Vault Transit mount path")
	}

	if err := s.validateTransitKey(ctx); err != nil {
		return err
	}

	return s.selfTest(ctx)
}

func (s *store) validateTransitKey(ctx context.Context) error {
	resp, err := s.c.Logical().ReadWithContext(ctx, s.keyPath())
	if err != nil {
		return errors.Wrapf(
			err,
			"Could not read HashiCorp Vault Transit key %q",
			s.keyName,
		)
	}
	if resp == nil {
		return errors.Errorf(
			"HashiCorp Vault Transit key %q does not exist",
			s.keyName,
		)
	}
	if resp.Data == nil {
		return errors.Errorf(
			"HashiCorp Vault Transit key %q returned empty metadata",
			s.keyName,
		)
	}

	keyType, ok := resp.Data["type"].(string)
	if !ok || keyType == "" {
		return errors.Errorf(
			"HashiCorp Vault Transit key %q has invalid type metadata",
			s.keyName,
		)
	}

	switch keyType {
	case "aes128-gcm96", "aes256-gcm96", "chacha20-poly1305":
	default:
		return errors.Errorf(
			"HashiCorp Vault Transit key %q uses unsupported type %q; an AEAD key is required",
			s.keyName,
			keyType,
		)
	}

	supportsEncryption, ok := resp.Data["supports_encryption"].(bool)
	if !ok || !supportsEncryption {
		return errors.Errorf(
			"HashiCorp Vault Transit key %q does not support encryption",
			s.keyName,
		)
	}

	supportsDecryption, ok := resp.Data["supports_decryption"].(bool)
	if !ok || !supportsDecryption {
		return errors.Errorf(
			"HashiCorp Vault Transit key %q does not support decryption",
			s.keyName,
		)
	}

	derived, ok := resp.Data["derived"].(bool)
	if !ok {
		return errors.Errorf(
			"HashiCorp Vault Transit key %q has invalid derived metadata",
			s.keyName,
		)
	}
	if derived {
		return errors.Errorf(
			"HashiCorp Vault Transit key %q is derived; derived Transit keys are not supported",
			s.keyName,
		)
	}

	return nil
}

func (s *store) selfTest(ctx context.Context) error {
	plaintext, err := utilrand.GetRandomBytes(32)
	if err != nil {
		return err
	}
	defer zero(plaintext)

	ciphertext, err := s.Encrypt(ctx, "self-test", plaintext)
	if err != nil {
		return errors.Wrap(err, "HashiCorp Vault store self-test encryption failed")
	}

	out, err := s.Decrypt(ctx, "self-test", ciphertext)
	if err != nil {
		return errors.Wrap(err, "HashiCorp Vault store self-test decryption failed")
	}
	defer zero(out)

	if !bytes.Equal(out, plaintext) {
		return errors.Errorf("HashiCorp Vault store self-test failed")
	}

	return nil
}

func (s *store) login(ctx context.Context) (*vault.Secret, error) {
	tkn, err := identity.GetIdentityToken(ctx, s.store)
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

	authK8s, err := kubernetes.NewKubernetesAuth(
		s.role,
		kubernetes.WithMountPath(s.authMount),
		kubernetes.WithServiceAccountToken(token),
	)
	if err != nil {
		return nil, err
	}

	loginSecret, err := s.c.Auth().Login(ctx, authK8s)
	if err != nil {
		return nil, err
	}
	if loginSecret == nil {
		return nil, errors.Errorf("Vault Kubernetes auth returned nil login response")
	}
	if loginSecret.Auth == nil {
		return nil, errors.Errorf("Vault Kubernetes auth returned nil auth metadata")
	}
	if strings.TrimSpace(loginSecret.Auth.ClientToken) == "" {
		return nil, errors.Errorf("Vault Kubernetes auth returned empty client token")
	}
	if loginSecret.Auth.LeaseDuration <= 0 {
		return nil, errors.Errorf("Vault Kubernetes auth returned invalid token lease duration")
	}

	return loginSecret, nil
}

func (s *store) runAuthLifecycle(ctx context.Context, initialSecret *vault.Secret) {
	defer close(s.done)

	current := initialSecret

	for {
		if ctx.Err() != nil {
			return
		}

		watcher, err := s.c.NewLifetimeWatcher(&vault.LifetimeWatcherInput{
			Secret: current,
		})
		if err != nil {
			zap.L().Warn(
				"Could not initialize Vault token lifetime watcher",
				zap.Error(err),
			)

			current = s.loginWithRetry(ctx)
			if current == nil {
				return
			}

			continue
		}

		go watcher.Start()

		watcherDone := false

		for !watcherDone {
			select {
			case <-ctx.Done():
				watcher.Stop()
				return

			case renewal := <-watcher.RenewCh():
				if renewal != nil {
					zap.L().Debug(
						"Successfully renewed Vault token",
						zap.Time("renewedAt", renewal.RenewedAt),
					)
				}

			case err := <-watcher.DoneCh():
				watcher.Stop()

				if ctx.Err() != nil {
					return
				}

				if err != nil {
					zap.L().Warn(
						"Vault token renewal stopped",
						zap.Error(err),
					)
				} else {
					zap.L().Debug(
						"Vault token can no longer be renewed; re-authenticating",
					)
				}

				watcherDone = true
			}
		}

		current = s.loginWithRetry(ctx)
		if current == nil {
			return
		}
	}
}

func (s *store) loginWithRetry(ctx context.Context) *vault.Secret {
	backoff := authRetryInitialBackoff

	for {
		if ctx.Err() != nil {
			return nil
		}

		loginSecret, err := s.login(ctx)
		if err == nil {
			zap.L().Info("Successfully re-authenticated to HashiCorp Vault")
			return loginSecret
		}

		zap.L().Warn(
			"Could not re-authenticate to HashiCorp Vault",
			zap.Error(err),
			zap.Duration("retryAfter", backoff),
		)

		timer := time.NewTimer(backoff)

		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil

		case <-timer.C:
		}

		if backoff < authRetryMaxBackoff {
			backoff *= 2
			if backoff > authRetryMaxBackoff {
				backoff = authRetryMaxBackoff
			}
		}
	}
}

func (s *store) keyPath() string {
	return fmt.Sprintf("%s/keys/%s", s.transitMount, s.keyName)
}

func (s *store) encryptPath() string {
	return fmt.Sprintf("%s/encrypt/%s", s.transitMount, s.keyName)
}

func (s *store) decryptPath() string {
	return fmt.Sprintf("%s/decrypt/%s", s.transitMount, s.keyName)
}

func dataAAD(uid string) []byte {
	return []byte(
		"octelium.secretman.kek.hashicorpvault.v1\x00uid=" + uid,
	)
}

func getMountPath(value, defaultValue string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultValue
	}

	value = strings.Trim(value, "/")
	if value == "" {
		return "", errors.Errorf("Empty mount path")
	}

	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", errors.Errorf("Invalid mount path segment %q", segment)
		}
	}

	return value, nil
}

func validateKeyName(keyName string) error {
	if strings.TrimSpace(keyName) == "" {
		return errors.Errorf("Empty HashiCorp Vault Transit key")
	}

	if strings.Contains(keyName, "/") {
		return errors.Errorf("HashiCorp Vault Transit key name cannot contain '/'")
	}

	if keyName == "." || keyName == ".." {
		return errors.Errorf("Invalid HashiCorp Vault Transit key name")
	}

	return nil
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
