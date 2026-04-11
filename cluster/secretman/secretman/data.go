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
	"slices"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/cluster/csecretmanv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
)

const dataTable = "octelium_encrypted_resources"

type dataSecret struct {
	Data   []byte
	Info   []byte
	KeyUID string
}

func (s *server) doGetDataSecret(ctx context.Context, req *csecretmanv1.GetSecretRequest) (*dataSecret, error) {

	filters := []exp.Expression{
		goqu.C("uid").Eq(req.SecretRef.Uid),
		goqu.C("resource_version").Eq(req.SecretRef.ResourceVersion),
	}

	ds := goqu.From(dataTable).Where(filters...).Select("ciphertext", "key_uid")
	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	var ciphertext []byte
	var keyUID string

	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&ciphertext, &keyUID); err != nil {
		if err == sql.ErrNoRows {
			return nil, grpcutils.NotFound("")
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	dek, err := s.getDEKByUID(keyUID)
	if err != nil {
		return nil, err
	}
	data, err := dek.decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	return &dataSecret{
		Data: data,
	}, nil
}

func (s *server) doSetDataSecret(ctx context.Context, req *csecretmanv1.SetSecretRequest) error {

	_, err := s.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
		SecretRef: req.SecretRef,
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return grpcutils.InternalWithErr(err)
		}

		return s.doCreateDataSecret(ctx, req)
	}

	return s.doUpdateDataSecret(ctx, req)
}

func (s *server) doCreateDataSecret(ctx context.Context, req *csecretmanv1.SetSecretRequest) error {

	enc, err := s.encryptData(ctx, req)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	ds := goqu.Insert(dataTable).
		Cols("uid", "resource_version", "created_at", "key_uid", "ciphertext").
		Vals(goqu.Vals{
			req.SecretRef.Uid,
			req.SecretRef.ResourceVersion,
			pbutils.Now().AsTime(),
			enc.KeyUID,
			fmt.Sprintf("\\x%s", hex.EncodeToString(enc.Ciphertext)),
		})

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	_, err = s.db.ExecContext(ctx, sqln, sqlargs...)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	return nil
}

func (s *server) doUpdateDataSecret(ctx context.Context, req *csecretmanv1.SetSecretRequest) error {

	enc, err := s.encryptData(ctx, req)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	ds := goqu.Update(dataTable).Where(goqu.C("uid").Eq(req.SecretRef.Uid)).Set(
		goqu.Record{
			"ciphertext":       fmt.Sprintf("\\x%s", hex.EncodeToString(enc.Ciphertext)),
			"key_uid":          enc.KeyUID,
			"resource_version": req.SecretRef.ResourceVersion,
			"updated_at":       pbutils.Now().AsTime(),
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

func (s *server) chooseDEK(ctx context.Context) (*dek, error) {
	s.deks.RLock()
	defer s.deks.RUnlock()

	for _, v := range s.deks.dekMap {
		return v, nil
	}

	return nil, errors.Errorf("Could not find a DEK")
}

func (s *server) encryptData(ctx context.Context, req *csecretmanv1.SetSecretRequest) (*dekEncryptionOutptut, error) {
	dek, err := s.chooseDEK(ctx)
	if err != nil {
		return nil, err
	}

	return dek.encrypt(req.Data)
}

func (s *server) doDeleteDataSecret(ctx context.Context, req *csecretmanv1.DeleteSecretRequest) error {
	_, err := s.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
		SecretRef: req.SecretRef,
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return grpcutils.InternalWithErr(err)
		}

		return nil
	}

	ds := goqu.Delete(dataTable).Where(goqu.C("uid").Eq(req.SecretRef.Uid))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	_, err = s.db.ExecContext(ctx, sqln, sqlargs...)
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	return nil
}

func (s *server) doListDataSecret(ctx context.Context, req *csecretmanv1.ListSecretRequest) (
	*csecretmanv1.ListSecretResponse, error) {

	type dbItem struct {
		uid             string
		resourceVersion string
		ciphertext      []byte
		keyUID          string
	}

	var filters []exp.Expression

	for _, itm := range req.SecretRefs {
		filters = append(filters, goqu.And(
			goqu.C("uid").Eq(itm.Uid),
			goqu.C("resource_version").Eq(itm.ResourceVersion),
		))
	}

	ds := goqu.From(dataTable).Where(goqu.Or(filters...)).
		Select("uid", "resource_version", "ciphertext", "key_uid")

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	dbItems := []*dbItem{}
	defer rows.Close()
	for rows.Next() {
		itm := &dbItem{}
		if err := rows.Scan(&itm.uid, &itm.resourceVersion, &itm.ciphertext, &itm.keyUID); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		dbItems = append(dbItems, itm)
	}

	ret := &csecretmanv1.ListSecretResponse{}

	for _, itm := range req.SecretRefs {
		if idx := slices.IndexFunc(dbItems, func(dbItem *dbItem) bool {
			return itm.Uid == dbItem.uid && itm.ResourceVersion == dbItem.resourceVersion
		}); idx >= 0 {
			dbItem := dbItems[idx]
			plaintext, err := s.decryptData(dbItem.ciphertext, dbItem.keyUID)
			if err != nil {
				return nil, grpcutils.InternalWithErr(err)
			}

			ret.Items = append(ret.Items, &csecretmanv1.ListSecretResponse_Item{
				SecretRef: itm,
				Data:      plaintext,
			})
		}
	}

	return ret, nil
}

func (s *server) decryptData(ciphertext []byte, keyUID string) ([]byte, error) {
	dek, err := s.getDEKByUID(keyUID)
	if err != nil {
		return nil, err
	}

	return dek.decrypt(ciphertext)
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
