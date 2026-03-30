// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package secretman

import (
	"context"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
)

func (s *server) onSecretManUpdate(ctx context.Context, new, old *enterprisev1.SecretStore) error {

	if pbutils.IsEqual(new.Spec, old.Spec) && pbutils.IsEqual(new.Status, old.Status) {
		return nil
	}

	if new.Status.Type == old.Status.Type {
		return nil
	}

	switch {
	case new.Status.State == enterprisev1.SecretStore_Status_LOADING &&
		old.Status.State != enterprisev1.SecretStore_Status_LOADING:
	default:
		return nil
	}

	zap.L().Info("Starting rotating DEKs")

	store, err := s.getKEKFromSecretStore(ctx, new)
	if err != nil {
		return err
	}

	s.deks.RLock()
	defer s.deks.RUnlock()

	for _, dek := range s.deks.dekMap {
		enc, err := store.Encrypt(ctx, dek.uid, dek.key)
		if err != nil {
			return err
		}

		if err := s.doUpdateDEK(ctx, dek.uid, enc, "", store.UID()); err != nil {
			return err
		}
	}

	ss, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{
		Uid: new.Metadata.Uid,
	})
	if err != nil {
		return err
	}

	ss.Status.State = enterprisev1.SecretStore_Status_OK

	_, err = s.octeliumC.EnterpriseC().UpdateSecretStore(ctx, ss)
	if err != nil {
		return err
	}

	zap.L().Info("Successfully rotatd DEKs")

	return nil
}
