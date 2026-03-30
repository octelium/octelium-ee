// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package genesis

import (
	"context"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Genesis) setInitKEK(ctx context.Context) error {
	if !vutils.IsDefaultRegion() {
		return nil
	}

	_, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}

		zap.L().Debug("Creating default SecretStore")
		_, err := s.octeliumC.EnterpriseC().CreateSecretStore(ctx, &enterprisev1.SecretStore{
			Metadata: &metav1.Metadata{
				Name: "default",
				// IsSystem:    true,
				DisplayName: "The root SecretStore",
				Description: `The initial SecretStore powered by the Kubernetes Cluster of the default Region`,
			},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
					Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
				},
			},
			Status: &enterprisev1.SecretStore_Status{
				State: enterprisev1.SecretStore_Status_OK,
				Type:  enterprisev1.SecretStore_Status_KUBERNETES,
			},
		})
		if err != nil {
			return err
		}
	}

	_, err = s.k8sC.CoreV1().Secrets(vutils.K8sNS).Get(ctx, "sys-init-kek", k8smetav1.GetOptions{})
	if err == nil {
		zap.L().Debug("sys-init-kek k8s Secret is already created...")
		return nil
	}
	if !k8serr.IsNotFound(err) {
		return err
	}

	secret, err := utilrand.GetRandomBytes(32)
	if err != nil {
		return err
	}

	zap.L().Debug("Creating sys-init-kek")
	if _, err := k8sutils.CreateOrUpdateSecret(ctx, s.k8sC, &k8scorev1.Secret{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      "sys-init-kek",
			Namespace: vutils.K8sNS,
		},
		Data: map[string][]byte{
			"data": secret,
		},
	}); err != nil {
		return err
	}

	return nil
}
