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
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/cluster/csecretmanv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
)

const dataTable = "octelium_encrypted_resources"

const (
	dataSecretAEADVersionLegacy int16 = 0
	dataSecretAEADVersionV1     int16 = 1
)

type dataSecret struct {
	Data   []byte
	Info   []byte
	KeyUID string
}

func (s *server) doGetDataSecret(ctx context.Context, req *csecretmanv1.GetSecretRequest) (*dataSecret, error) {
	if err := validateGetSecretRequest(req); err != nil {
		return nil, err
	}

	ds := goqu.From(dataTable).
		Where(
			goqu.C("uid").Eq(req.SecretRef.Uid),
			goqu.C("resource_version").Eq(req.SecretRef.ResourceVersion),
		).
		Select("ciphertext", "key_uid", "aead_version")

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	var ciphertext []byte
	var keyUID string
	var aeadVersion int16

	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&ciphertext, &keyUID, &aeadVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, grpcutils.NotFound("")
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	plaintext, err := s.decryptData(
		req.SecretRef.Uid,
		req.SecretRef.ResourceVersion,
		keyUID,
		ciphertext,
		aeadVersion,
	)
	if err != nil {
		return nil, err
	}

	return &dataSecret{
		Data:   plaintext,
		KeyUID: keyUID,
	}, nil
}

func (s *server) doSetDataSecret(ctx context.Context, req *csecretmanv1.SetSecretRequest) error {
	if err := validateSetSecretRequest(req); err != nil {
		return err
	}

	enc, err := s.encryptData(ctx, req)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	now := pbutils.Now().AsTime()

	_, err = s.db.ExecContext(ctx, `
INSERT INTO octelium_encrypted_resources
    (uid, resource_version, created_at, updated_at, key_uid, ciphertext, aead_version)
VALUES
    ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (uid)
DO UPDATE SET
    resource_version = EXCLUDED.resource_version,
    updated_at = EXCLUDED.updated_at,
    key_uid = EXCLUDED.key_uid,
    ciphertext = EXCLUDED.ciphertext,
    aead_version = EXCLUDED.aead_version
`,
		req.SecretRef.Uid,
		req.SecretRef.ResourceVersion,
		now,
		now,
		enc.KeyUID,
		enc.Ciphertext,
		dataSecretAEADVersionV1,
	)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	return nil
}

func (s *server) chooseDEK(_ context.Context) (*dek, error) {
	s.deks.RLock()
	defer s.deks.RUnlock()

	if s.deks.cur != nil {
		return s.deks.cur, nil
	}

	return nil, errors.Errorf("Could not find a DEK")
}

func (s *server) encryptData(ctx context.Context, req *csecretmanv1.SetSecretRequest) (*dekEncryptionOutptut, error) {
	dek, err := s.chooseDEK(ctx)
	if err != nil {
		return nil, err
	}

	aad := getDataSecretAAD(req.SecretRef.Uid, req.SecretRef.ResourceVersion, dek.uid)

	return dek.encryptWithAAD(req.Data, aad)
}

func (s *server) doDeleteDataSecret(ctx context.Context, req *csecretmanv1.DeleteSecretRequest) error {
	if err := validateDeleteSecretRequest(req); err != nil {
		return err
	}

	ds := goqu.Delete(dataTable).
		Where(
			goqu.C("uid").Eq(req.SecretRef.Uid),
			goqu.C("resource_version").Eq(req.SecretRef.ResourceVersion),
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

func (s *server) doListDataSecret(ctx context.Context, req *csecretmanv1.ListSecretRequest) (
	*csecretmanv1.ListSecretResponse, error,
) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	ret := &csecretmanv1.ListSecretResponse{}

	if len(req.SecretRefs) == 0 {
		return ret, nil
	}

	type dbItem struct {
		uid             string
		resourceVersion string
		ciphertext      []byte
		keyUID          string
		aeadVersion     int16
	}

	var filters []exp.Expression
	for _, ref := range req.SecretRefs {
		if ref == nil {
			continue
		}
		if ref.Uid == "" || ref.ResourceVersion == "" {
			continue
		}

		filters = append(filters, goqu.And(
			goqu.C("uid").Eq(ref.Uid),
			goqu.C("resource_version").Eq(ref.ResourceVersion),
		))
	}

	if len(filters) == 0 {
		return ret, nil
	}

	ds := goqu.From(dataTable).
		Where(goqu.Or(filters...)).
		Select("uid", "resource_version", "ciphertext", "key_uid", "aead_version")

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	dbItems := make(map[string]*dbItem)

	for rows.Next() {
		itm := &dbItem{}
		if err := rows.Scan(
			&itm.uid,
			&itm.resourceVersion,
			&itm.ciphertext,
			&itm.keyUID,
			&itm.aeadVersion,
		); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		dbItems[getSecretRefKey(itm.uid, itm.resourceVersion)] = itm
	}

	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	for _, ref := range req.SecretRefs {
		if ref == nil {
			continue
		}

		dbItem, ok := dbItems[getSecretRefKey(ref.Uid, ref.ResourceVersion)]
		if !ok {
			continue
		}

		plaintext, err := s.decryptData(
			dbItem.uid,
			dbItem.resourceVersion,
			dbItem.keyUID,
			dbItem.ciphertext,
			dbItem.aeadVersion,
		)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.Items = append(ret.Items, &csecretmanv1.ListSecretResponse_Item{
			SecretRef: ref,
			Data:      plaintext,
		})
	}

	return ret, nil
}

func (s *server) decryptData(uid, resourceVersion, keyUID string, ciphertext []byte, aeadVersion int16) ([]byte, error) {
	dek, err := s.getDEKByUID(keyUID)
	if err != nil {
		return nil, err
	}

	switch aeadVersion {
	case dataSecretAEADVersionLegacy:
		return dek.decryptWithAAD(ciphertext, nil)
	case dataSecretAEADVersionV1:
		return dek.decryptWithAAD(ciphertext, getDataSecretAAD(uid, resourceVersion, keyUID))
	default:
		return nil, errors.Errorf("Unsupported data secret AEAD version: %d", aeadVersion)
	}
}

func (s *server) getDEKByUID(uid string) (*dek, error) {
	s.deks.RLock()
	defer s.deks.RUnlock()

	dek, ok := s.deks.dekMap[uid]
	if !ok {
		return nil, errors.Errorf("Could not find dek for uid: %s", uid)
	}

	return dek, nil
}

func validateGetSecretRequest(req *csecretmanv1.GetSecretRequest) error {
	if req == nil {
		return grpcutils.InvalidArg("Nil request")
	}
	if req.SecretRef == nil {
		return grpcutils.InvalidArg("Nil SecretRef")
	}
	if req.SecretRef.Uid == "" {
		return grpcutils.InvalidArg("Empty SecretRef uid")
	}
	if req.SecretRef.ResourceVersion == "" {
		return grpcutils.InvalidArg("Empty SecretRef resourceVersion")
	}

	return nil
}

func validateSetSecretRequest(req *csecretmanv1.SetSecretRequest) error {
	if req == nil {
		return grpcutils.InvalidArg("Nil request")
	}
	if req.SecretRef == nil {
		return grpcutils.InvalidArg("Nil SecretRef")
	}
	if req.SecretRef.Uid == "" {
		return grpcutils.InvalidArg("Empty SecretRef uid")
	}
	if req.SecretRef.ResourceVersion == "" {
		return grpcutils.InvalidArg("Empty SecretRef resourceVersion")
	}
	if len(req.Data) == 0 {
		return grpcutils.InvalidArg("Empty Secret data")
	}

	return nil
}

func validateDeleteSecretRequest(req *csecretmanv1.DeleteSecretRequest) error {
	if req == nil {
		return grpcutils.InvalidArg("Nil request")
	}
	if req.SecretRef == nil {
		return grpcutils.InvalidArg("Nil SecretRef")
	}
	if req.SecretRef.Uid == "" {
		return grpcutils.InvalidArg("Empty SecretRef uid")
	}
	if req.SecretRef.ResourceVersion == "" {
		return grpcutils.InvalidArg("Empty SecretRef resourceVersion")
	}

	return nil
}

func getSecretRefKey(uid, resourceVersion string) string {
	return uid + "\x00" + resourceVersion
}

func getDataSecretAAD(uid, resourceVersion, keyUID string) []byte {
	return []byte(fmt.Sprintf(
		"octelium.secretman.encrypted_resource.v1\x00uid=%s\x00resource_version=%s\x00key_uid=%s",
		uid,
		resourceVersion,
		keyUID,
	))
}
