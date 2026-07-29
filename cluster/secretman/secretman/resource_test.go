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
	"encoding/json"
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium/apis/cluster/csecretmanv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestResource(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C
	db, err := postgresutils.NewDB()
	assert.Nil(t, err)
	err = migrations.Migrate(ctx, db)
	assert.Nil(t, err)

	srv, err := newServer(ctx, fakeC.OcteliumC, db)
	assert.Nil(t, err)

	err = srv.initRootDEK(ctx)
	assert.Nil(t, err)

	err = srv.setDEKMap(ctx)
	assert.Nil(t, err)

	{
		var filters []exp.Expression

		filters = append(filters, goqu.C("kind").Like("%Secret"))
		filters = append(filters, goqu.L(`jsonb_path_exists(resource, '$.data ? (@ != null)')`))

		ds := goqu.From(resourceTableName).Where(filters...).
			Select(goqu.L(`count(*) OVER() AS full_count`))

		ds = ds.OrderAppend(goqu.I(`created_at`).Desc())

		sqln, sqlargs, err := ds.ToSQL()
		assert.Nil(t, err)

		rows, err := srv.db.QueryContext(ctx, sqln, sqlargs...)
		assert.Nil(t, err)

		var count int
		for rows.Next() {
			err := rows.Scan(&count)
			assert.Nil(t, err)
		}

		assert.True(t, count > 0)
	}

	err = srv.setStaleSecretResources(ctx)
	assert.Nil(t, err)

	{
		var filters []exp.Expression

		filters = append(filters, goqu.C("kind").Like("%Secret"))
		filters = append(filters, goqu.L(`jsonb_path_exists(resource, '$.data ? (@ != null)')`))

		ds := goqu.From(resourceTableName).Where(filters...).
			Select(goqu.L(`count(*) OVER() AS full_count`))

		ds = ds.OrderAppend(goqu.I(`created_at`).Desc())

		sqln, sqlargs, err := ds.ToSQL()
		assert.Nil(t, err)

		rows, err := srv.db.QueryContext(ctx, sqln, sqlargs...)
		assert.Nil(t, err)

		var count int
		for rows.Next() {
			err := rows.Scan(&count)
			assert.Nil(t, err)
		}
		assert.True(t, count == 0)
	}

	zap.L().Debug("setStaleSecretResources again...")
	err = srv.setStaleSecretResources(ctx)
	assert.Nil(t, err)

	{
		var filters []exp.Expression

		filters = append(filters, goqu.C("kind").Like("%Secret"))
		filters = append(filters, goqu.L(`jsonb_path_exists(resource, '$.data ? (@ != null)')`))

		ds := goqu.From(resourceTableName).Where(filters...).
			Select(goqu.L(`count(*) OVER() AS full_count`))

		ds = ds.OrderAppend(goqu.I(`created_at`).Desc())

		sqln, sqlargs, err := ds.ToSQL()
		assert.Nil(t, err)

		rows, err := srv.db.QueryContext(ctx, sqln, sqlargs...)
		assert.Nil(t, err)

		var count int
		for rows.Next() {
			err := rows.Scan(&count)
			assert.Nil(t, err)
		}
		assert.True(t, count == 0)
	}
}

func TestResourceSecretDataPreserved(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C
	db, err := postgresutils.NewDB()
	assert.Nil(t, err)
	err = migrations.Migrate(ctx, db)
	assert.Nil(t, err)

	srv, err := newServer(ctx, fakeC.OcteliumC, db)
	assert.Nil(t, err)

	err = srv.initRootDEK(ctx)
	assert.Nil(t, err)

	err = srv.setDEKMap(ctx)
	assert.Nil(t, err)

	getResourceMap := func(uid string) map[string]any {
		ds := goqu.From(resourceTableName).Where(goqu.C("uid").Eq(uid)).Select("resource")
		sqln, sqlargs, err := ds.ToSQL()
		assert.Nil(t, err)

		var data []byte
		err = srv.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&data)
		assert.Nil(t, err)

		rscMap := make(map[string]any)
		err = json.Unmarshal(data, &rscMap)
		assert.Nil(t, err)

		return rscMap
	}

	sec, err := fakeC.OcteliumC.CoreC().CreateSecret(ctx, &corev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec:   &corev1.Secret_Spec{},
		Status: &corev1.Secret_Status{},
		Data: &corev1.Secret_Data{
			Type: &corev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		},
	})
	assert.Nil(t, err)

	{
		rscMap := getResourceMap(sec.Metadata.Uid)
		assert.NotNil(t, rscMap["data"])
	}

	err = srv.setStaleSecretResources(ctx)
	assert.Nil(t, err, "%+v", err)

	{
		rscMap := getResourceMap(sec.Metadata.Uid)
		assert.Nil(t, rscMap["data"])

		mdMap, ok := rscMap["metadata"].(map[string]any)
		assert.True(t, ok)

		md := &metav1.Metadata{}
		err := pbutils.UnmarshalFromMap(mdMap, md)
		assert.Nil(t, err)

		assert.Equal(t, sec.Metadata.Uid, md.Uid)
		assert.Equal(t, sec.Metadata.ResourceVersion, md.ResourceVersion)
	}

	resp, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
		SecretRef: &metav1.ObjectReference{
			Uid:             sec.Metadata.Uid,
			ResourceVersion: sec.Metadata.ResourceVersion,
		},
	})
	assert.Nil(t, err, "%+v", err)

	dataR := &corev1.Secret_Data{}
	err = pbutils.UnmarshalJSON(resp.Data, dataR)
	assert.Nil(t, err, "%+v", err)

	assert.True(t, pbutils.IsEqual(dataR, sec.Data))

	err = srv.setStaleSecretResources(ctx)
	assert.Nil(t, err)

	{
		resp2, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: &metav1.ObjectReference{
				Uid:             sec.Metadata.Uid,
				ResourceVersion: sec.Metadata.ResourceVersion,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, resp.Data, resp2.Data)
	}
}

func TestCheckAndUpdateSecretResource(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C
	db, err := postgresutils.NewDB()
	assert.Nil(t, err)
	err = migrations.Migrate(ctx, db)
	assert.Nil(t, err)

	srv, err := newServer(ctx, fakeC.OcteliumC, db)
	assert.Nil(t, err)

	err = srv.initRootDEK(ctx)
	assert.Nil(t, err)

	err = srv.setDEKMap(ctx)
	assert.Nil(t, err)

	{
		err := srv.checkAndUpdateSecretResource(ctx, []byte("octelium"))
		assert.NotNil(t, err)
	}
	{
		err := srv.checkAndUpdateSecretResource(ctx, nil)
		assert.NotNil(t, err)
	}
	{
		err := srv.checkAndUpdateSecretResource(ctx,
			[]byte(`{"apiVersion":"core/v1","kind":"Secret"}`))
		assert.Nil(t, err)
	}
	{
		err := srv.checkAndUpdateSecretResource(ctx,
			[]byte(`{"apiVersion":"core/v1","kind":"Secret","data":null}`))
		assert.Nil(t, err)
	}
	{
		err := srv.checkAndUpdateSecretResource(ctx,
			[]byte(`{"kind":"Secret","data":{"value":"octelium"}}`))
		assert.NotNil(t, err)
	}
	{
		err := srv.checkAndUpdateSecretResource(ctx,
			[]byte(`{"apiVersion":"core/v1","data":{"value":"octelium"}}`))
		assert.NotNil(t, err)
	}
	{
		err := srv.checkAndUpdateSecretResource(ctx,
			[]byte(`{"apiVersion":"core/v1","kind":"Secret","data":{"value":"octelium"}}`))
		assert.NotNil(t, err)
	}
	{
		err := srv.checkAndUpdateSecretResource(ctx,
			[]byte(`{"apiVersion":"core/v1","kind":"Secret","metadata":"octelium","data":{"value":"octelium"}}`))
		assert.NotNil(t, err)
	}
}
