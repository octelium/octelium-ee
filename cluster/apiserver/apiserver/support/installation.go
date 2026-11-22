// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package support

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"regexp"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	installationSecretName  = "sys-octelium-support-installation"
	installationSecretLabel = "octelium-support-installation"
)

var rgxInstallationKey = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func (s *Server) getInstallationKey(ctx context.Context) (string, error) {
	s.mu.Lock()
	key := s.installationKey
	s.mu.Unlock()
	if key != "" {
		return key, nil
	}

	key, err := s.loadInstallationKey(ctx)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.installationKey = key
	s.mu.Unlock()

	return key, nil
}

func (s *Server) loadInstallationKey(ctx context.Context) (string, error) {
	sec, err := s.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: installationSecretName,
	})
	if err == nil {
		return getInstallationKeyFromSecret(sec)
	}
	if !grpcerr.IsNotFound(err) {
		return "", err
	}

	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", err
	}
	key := base64.RawURLEncoding.EncodeToString(keyBytes)

	sec, err = s.octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name:           installationSecretName,
			IsSystem:       true,
			IsUserHidden:   true,
			IsSystemHidden: true,
			SystemLabels: map[string]string{
				installationSecretLabel: "true",
			},
		},
		Spec:   &enterprisev1.Secret_Spec{},
		Status: &enterprisev1.Secret_Status{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: key,
			},
		},
	})
	if err != nil {
		if !grpcerr.AlreadyExists(err) {
			return "", err
		}

		sec, err = s.octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
			Name: installationSecretName,
		})
		if err != nil {
			return "", err
		}
		return getInstallationKeyFromSecret(sec)
	}

	zap.L().Info("Created the support installation key of the Cluster")

	return getInstallationKeyFromSecret(sec)
}

func getInstallationKeyFromSecret(sec *enterprisev1.Secret) (string, error) {
	if sec == nil || sec.Metadata == nil {
		return "", errors.Errorf("Invalid support installation Secret")
	}
	if !sec.Metadata.IsSystem || !sec.Metadata.IsUserHidden || !sec.Metadata.IsSystemHidden ||
		sec.Metadata.SystemLabels[installationSecretLabel] != "true" {
		return "", errors.Errorf("The Secret is not owned by the support subsystem")
	}

	value := sec.GetData().GetValue()
	if !rgxInstallationKey.MatchString(value) {
		return "", errors.Errorf("The support installation Secret has an invalid value")
	}

	return value, nil
}
