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
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium/cluster/common/postgresutils"
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
