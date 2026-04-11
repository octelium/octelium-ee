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
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestDEKEncryption(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	db, err := postgresutils.NewDB()
	assert.Nil(t, err)

	err = migrations.Migrate(ctx, db)
	assert.Nil(t, err)

	dek := &dek{
		uid: vutils.UUIDv4(),
	}
	k, err := utilrand.GetRandomBytes(32)
	assert.Nil(t, err)
	dek.key = k

	plaintext, err := utilrand.GetRandomBytes(32)
	assert.Nil(t, err)

	res, err := dek.encrypt(plaintext)
	assert.Nil(t, err)

	out, err := dek.decrypt(res.Ciphertext)
	assert.Nil(t, err)
	assert.Equal(t, plaintext, out)
}

func TestDEK(t *testing.T) {
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

	assert.True(t, len(srv.deks.dekMap) == 1)

	dek, err := srv.chooseDEK(ctx)
	assert.Nil(t, err)
	assert.False(t, dek.createdAt.IsZero())
	assert.True(t, dek.createdAt.After(time.Now().Add(-1*time.Minute)))

	val, err := utilrand.GetRandomBytes(32)
	assert.Nil(t, err)

	ciphertext, err := dek.encrypt(val)
	assert.Nil(t, err)

	plaintext, err := dek.decrypt(ciphertext.Ciphertext)
	assert.Nil(t, err)

	assert.Equal(t, val, plaintext)

	{
		err = srv.doCreateDEK(ctx)
		assert.Nil(t, err)

		err = srv.setDEKMap(ctx)
		assert.Nil(t, err)
		assert.True(t, len(srv.deks.dekMap) == 2)

		dek2, err := srv.chooseDEK(ctx)
		assert.Nil(t, err)

		assert.True(t, dek.uid != dek2.uid)
		assert.True(t, dek2.createdAt.After(dek.createdAt))

		val, err := utilrand.GetRandomBytes(32)
		assert.Nil(t, err)

		ciphertext, err := dek2.encrypt(val)
		assert.Nil(t, err)

		plaintext, err := dek2.decrypt(ciphertext.Ciphertext)
		assert.Nil(t, err)
		assert.Equal(t, val, plaintext)
	}
}
