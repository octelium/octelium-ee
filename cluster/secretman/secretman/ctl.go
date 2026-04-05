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

	if s.shouldSync(new, old) {
		if err := s.doSync(ctx, new); err != nil {
			zap.L().Warn("Could not synchronize", zap.Any("ss", new), zap.Error(err))
		}
	}

	return nil
}

func (s *server) shouldSync(new, old *enterprisev1.SecretStore) bool {
	if pbutils.IsEqual(new.Status.Synchronization, old.Status.Synchronization) {
		return false
	}

	return new.Status.Synchronization != nil &&
		new.Status.Synchronization.State == enterprisev1.SecretStore_Status_Synchronization_SYNC_REQUESTED &&
		(old == nil ||
			old.Status.Synchronization == nil ||
			old.Status.Synchronization.State != enterprisev1.SecretStore_Status_Synchronization_SYNC_REQUESTED)
}

func (s *server) doSync(ctx context.Context, new *enterprisev1.SecretStore) error {
	zap.L().Info("Starting rotating DEKs")

	store, err := s.getKEKFromSecretStore(ctx, new)
	if err != nil {
		return err
	}

	s.deks.RLock()
	defer s.deks.RUnlock()

	{
		ss, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{
			Uid: new.Metadata.Uid,
		})
		if err != nil {
			return err
		}

		ss.Status.Synchronization.State = enterprisev1.SecretStore_Status_Synchronization_SYNCING

		if _, err := s.octeliumC.EnterpriseC().UpdateSecretStore(ctx, ss); err != nil {
			return err
		}
	}

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

	ss.Status.Synchronization.State = enterprisev1.SecretStore_Status_Synchronization_SUCCESS
	ss.Status.Synchronization.CompletedAt = pbutils.Now()

	ss.Status.LastSynchronizations = append([]*enterprisev1.SecretStore_Status_Synchronization{
		ss.Status.Synchronization,
	}, ss.Status.LastSynchronizations...)

	ss.Status.Synchronization = nil

	_, err = s.octeliumC.EnterpriseC().UpdateSecretStore(ctx, ss)
	if err != nil {
		return err
	}

	zap.L().Info("Successfully rotated DEKs")

	return nil
}
