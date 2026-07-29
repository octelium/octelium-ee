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
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/grpcerr"
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

func TestDEKStoreOperations(t *testing.T) {
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

	deks, err := srv.doListDEK(ctx)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, 1, len(deks))

	dek1 := deks[0]
	assert.NotEmpty(t, dek1.uid)
	assert.Equal(t, 32, len(dek1.key))
	assert.False(t, dek1.createdAt.IsZero())

	{
		got, err := srv.doGetDEK(ctx, dek1.uid)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, dek1.uid, got.uid)
		assert.Equal(t, dek1.key, got.key)
	}
	{
		_, err := srv.doGetDEK(ctx, vutils.UUIDv4())
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}

	{
		kek, err := srv.chooseKEK(ctx)
		assert.Nil(t, err, "%+v", err)

		enc, err := kek.Encrypt(ctx, dek1.uid, dek1.key)
		assert.Nil(t, err, "%+v", err)

		err = srv.doUpdateDEK(ctx, dek1.uid, enc, "", kek.UID())
		assert.Nil(t, err, "%+v", err)

		assert.Nil(t, kek.Close())

		got, err := srv.doGetDEK(ctx, dek1.uid)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, dek1.key, got.key)
	}

	{
		err := srv.doUpdateDEK(ctx, vutils.UUIDv4(), []byte("octelium"), "", "kek")
		assert.Nil(t, err)

		deks, err := srv.doListDEK(ctx)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(deks))
	}

	{
		err := srv.doCreateDEK(ctx)
		assert.Nil(t, err)

		err = srv.doCreateDEK(ctx)
		assert.Nil(t, err)

		deks, err := srv.doListDEK(ctx)
		assert.Nil(t, err)
		assert.Equal(t, 3, len(deks))

		seen := make(map[string]struct{})
		for idx, itm := range deks {
			assert.NotEmpty(t, itm.uid)
			assert.Equal(t, 32, len(itm.key))

			_, ok := seen[itm.uid]
			assert.False(t, ok)
			seen[itm.uid] = struct{}{}

			if idx > 0 {
				assert.False(t, itm.createdAt.Before(deks[idx-1].createdAt))
			}
		}

		err = srv.setDEKMap(ctx)
		assert.Nil(t, err)

		assert.Equal(t, 3, len(srv.deks.dekMap))
		assert.Equal(t, deks[len(deks)-1].uid, srv.deks.cur.uid)
	}
}

func TestKEKSelection(t *testing.T) {
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

	kek, err := srv.chooseKEK(ctx)
	assert.Nil(t, err, "%+v", err)
	assert.NotEmpty(t, kek.UID())

	{
		kek2, err := srv.getKEKByUID(ctx, kek.UID())
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, kek.UID(), kek2.UID())
		assert.Nil(t, kek2.Close())
	}
	{
		_, err := srv.getKEKByUID(ctx, vutils.UUIDv4())
		assert.NotNil(t, err)
	}

	assert.Nil(t, kek.Close())

	{
		store, err := srv.getKEKFromSecretStore(ctx, &enterprisev1.SecretStore{
			Metadata: &metav1.Metadata{
				Uid: vutils.UUIDv4(),
			},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
					Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
				},
			},
		})
		assert.Nil(t, err, "%+v", err)

		err = store.Initialize(ctx)
		assert.Nil(t, err, "%+v", err)

		assert.Nil(t, store.Close())
	}
	{
		_, err := srv.getKEKFromSecretStore(ctx, &enterprisev1.SecretStore{
			Metadata: &metav1.Metadata{
				Uid: vutils.UUIDv4(),
			},
			Spec: &enterprisev1.SecretStore_Spec{},
		})
		assert.NotNil(t, err)
	}
	{
		_, err := srv.getKEKFromSecretStore(ctx, &enterprisev1.SecretStore{
			Metadata: &metav1.Metadata{},
			Spec: &enterprisev1.SecretStore_Spec{
				Type: &enterprisev1.SecretStore_Spec_Kubernetes_{
					Kubernetes: &enterprisev1.SecretStore_Spec_Kubernetes{},
				},
			},
		})
		assert.NotNil(t, err)
	}
}
