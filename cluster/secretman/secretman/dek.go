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
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	_ "github.com/lib/pq"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores/awskms"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores/azurekeyvault"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores/gcpkms"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores/hashicorpvault"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/stores/k8s"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const dekTable = "octelium_data_encryption_keys"

func (s *server) doCreateDEK(ctx context.Context) error {

	k, err := utilrand.GetRandomBytes(32)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}
	uid := vutils.UUIDv4()

	kek, err := s.chooseKEK(ctx)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	ciphertext, err := kek.Encrypt(ctx, uid, k)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	zap.L().Debug("Creating a new DEK", zap.String("uid", uid), zap.String("kek", kek.UID()))

	ds := goqu.Insert(dekTable).
		Cols("uid", "key_uid", "key_version", "created_at", "ciphertext").
		Vals(goqu.Vals{
			uid,
			kek.UID(),
			"",
			pbutils.Now().AsTime(),
			fmt.Sprintf("\\x%s", hex.EncodeToString(ciphertext)),
		})

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	_, err = s.db.ExecContext(ctx, sqln, sqlargs...)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	zap.L().Debug("Creating DEK done...", zap.String("uid", uid), zap.String("kek", kek.UID()))

	return nil
}

func (s *server) doUpdateDEK(ctx context.Context, uid string, ciphertext []byte, keyVersion, kekUID string) error {

	ds := goqu.Update(dekTable).Where(goqu.C("uid").Eq(uid)).Set(
		goqu.Record{
			"ciphertext":  fmt.Sprintf("\\x%s", hex.EncodeToString(ciphertext)),
			"key_uid":     kekUID,
			"key_version": keyVersion,
			"updated_at":  pbutils.Now().AsTime(),
		},
	)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	if _, err := s.db.ExecContext(ctx, sqln, sqlargs...); err != nil {
		return grpcutils.InternalWithErr(err)
	}

	return nil
}

func (s *server) doGetDEK(ctx context.Context, uid string) (*dek, error) {

	filters := []exp.Expression{
		goqu.C("uid").Eq(uid),
	}
	ds := goqu.From(dekTable).Where(filters...).Select("key_uid", "key_version", "ciphertext")
	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	var ciphertext []byte
	var keyUID string
	var keyVersion string

	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&keyUID, &keyVersion, &ciphertext); err != nil {
		if err == sql.ErrNoRows {
			return nil, grpcutils.NotFound("")
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	kek, err := s.getKEKByUID(ctx, keyUID)
	if err != nil {
		return nil, err
	}

	plaintext, err := kek.Decrypt(ctx, uid, ciphertext)
	if err != nil {
		return nil, err
	}

	return &dek{
		key: plaintext,
		uid: uid,
	}, nil
}

func (s *server) doListDEK(ctx context.Context) ([]*dek, error) {

	ds := goqu.From(dekTable).
		Select("uid", "ciphertext", "key_uid", "key_version")

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.Internal("Could not generate SQL: %+v", err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(errors.Errorf("Could not list db DEKs: %+v", err))
	}

	var ret []*dek

	for rows.Next() {
		var uid string
		var keyUID string
		var keyVersion string
		var ciphertext []byte
		if err := rows.Scan(&uid, &ciphertext, &keyUID, &keyVersion); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		kek, err := s.getKEKByUID(ctx, keyUID)
		if err != nil {
			return nil, err
		}

		plaintext, err := kek.Decrypt(ctx, uid, ciphertext)
		if err != nil {
			return nil, err
		}

		ret = append(ret, &dek{
			key: plaintext,
			uid: uid,
		})
	}

	return ret, nil
}

func (s *server) chooseKEK(ctx context.Context) (stores.Store, error) {
	ss, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	if err != nil {
		return nil, err
	}
	return s.getKEKFromSecretStore(ctx, ss)
}

func (s *server) getKEKByUID(ctx context.Context, uid string) (stores.Store, error) {
	ss, err := s.octeliumC.EnterpriseC().GetSecretStore(ctx, &rmetav1.GetOptions{
		Uid: uid,
	})
	if err != nil {
		return nil, err
	}
	return s.getKEKFromSecretStore(ctx, ss)
}

func (s *server) getKEKFromSecretStore(ctx context.Context, ss *enterprisev1.SecretStore) (stores.Store, error) {

	opts := &stores.StoreOpts{
		SecretStore: ss,
	}

	switch ss.Spec.Type.(type) {
	case *enterprisev1.SecretStore_Spec_HashicorpVault_:
		return hashicorpvault.NewStore(ctx, opts)
	case *enterprisev1.SecretStore_Spec_GoogleCloudKeyManagementService_:
		return gcpkms.NewStore(ctx, opts)
	case *enterprisev1.SecretStore_Spec_AwsKeyManagementService:
		return awskms.NewStore(ctx, opts)
	case *enterprisev1.SecretStore_Spec_AzureKeyVault_:
		return azurekeyvault.NewStore(ctx, opts)
	case *enterprisev1.SecretStore_Spec_Kubernetes_:
		return k8s.NewStore(ctx, opts)
	default:
		return k8s.NewStore(ctx, opts)
	}
}
