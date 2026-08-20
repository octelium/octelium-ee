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
			if updateErr := s.markSyncFinished(ctx, new.Metadata.Uid, err); updateErr != nil {
				zap.L().Warn("Could not record synchronization failure",
					zap.Any("ss", new), zap.Error(updateErr))
			}
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
	defer store.Close()

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

	if err := func() error {
		s.deks.RLock()
		deks := make([]*dek, 0, len(s.deks.dekMap))
		for _, dek := range s.deks.dekMap {
			deks = append(deks, dek)
		}
		s.deks.RUnlock()

		for _, dek := range deks {
			enc, err := store.Encrypt(ctx, dek.uid, dek.key)
			if err != nil {
				return err
			}

			if err := s.doUpdateDEK(ctx, dek.uid, enc, "", store.UID()); err != nil {
				return err
			}
		}

		return nil
	}(); err != nil {
		return err
	}

	if err := s.markSyncFinished(ctx, new.Metadata.Uid, nil); err != nil {
		return err
	}

	zap.L().Info("Successfully rotated DEKs")

	return nil
}

func (s *server) markSyncFinished(ctx context.Context, uid string, syncErr error) error {
	ss, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{Uid: uid})
	if err != nil {
		return err
	}

	if ss.Status == nil {
		ss.Status = &enterprisev1.SecretStore_Status{}
	}
	if ss.Status.Synchronization == nil {
		ss.Status.Synchronization = &enterprisev1.SecretStore_Status_Synchronization{
			CreatedAt: pbutils.Now(),
		}
	}

	sync := ss.Status.Synchronization
	sync.State = enterprisev1.SecretStore_Status_Synchronization_SUCCESS
	if syncErr != nil {
		sync.State = enterprisev1.SecretStore_Status_Synchronization_FAILED
	} else {
		ss.Status.State = enterprisev1.SecretStore_Status_OK
	}
	sync.CompletedAt = pbutils.Now()

	ss.Status.LastSynchronizations = appendSecretStoreSync(
		ss.Status.LastSynchronizations, sync)

	_, err = s.octeliumC.EnterpriseC().UpdateSecretStore(ctx, ss)
	return err
}

const maxLastSecretStoreSynchronizations = 100

func appendSecretStoreSync(
	history []*enterprisev1.SecretStore_Status_Synchronization,
	rec *enterprisev1.SecretStore_Status_Synchronization,
) []*enterprisev1.SecretStore_Status_Synchronization {
	entry := &enterprisev1.SecretStore_Status_Synchronization{
		State:       rec.State,
		CreatedAt:   rec.CreatedAt,
		CompletedAt: rec.CompletedAt,
	}
	history = append([]*enterprisev1.SecretStore_Status_Synchronization{entry}, history...)
	if len(history) > maxLastSecretStoreSynchronizations {
		history = history[:maxLastSecretStoreSynchronizations]
	}
	return history
}
