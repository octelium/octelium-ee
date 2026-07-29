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
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium/apis/cluster/csecretmanv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestData(t *testing.T) {
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

	req := &csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
		},
		Data: utilrand.GetRandomBytesMust(32),
	}
	err = srv.doSetDataSecret(ctx, req)
	assert.Nil(t, err, "%+v", err)

	resp, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
		SecretRef: req.SecretRef,
	})
	assert.Nil(t, err, "%+v", err)

	assert.Equal(t, req.Data, resp.Data)
	assert.Equal(t, dek.uid, resp.KeyUID)

	req2 := &csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{
			Uid:             req.SecretRef.Uid,
			ResourceVersion: vutils.UUIDv7(),
		},
		Data: utilrand.GetRandomBytesMust(32),
	}

	err = srv.doSetDataSecret(ctx, req2)
	assert.Nil(t, err)

	{
		_, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: req.SecretRef,
		})
		assert.NotNil(t, err, "%+v", err)
		assert.True(t, grpcerr.IsNotFound(err))
	}
	{
		resp, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: req2.SecretRef,
		})
		assert.Nil(t, err)

		assert.Equal(t, req2.Data, resp.Data)
		assert.Equal(t, dek.uid, resp.KeyUID)
	}

	err = srv.doCreateDEK(ctx)
	assert.Nil(t, err)

	err = srv.setDEKMap(ctx)
	assert.Nil(t, err)

	assert.True(t, len(srv.deks.dekMap) == 2)

	dek2, err := srv.chooseDEK(ctx)
	assert.Nil(t, err)

	assert.NotEqual(t, dek.uid, dek2.uid)

	{
		resp, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: req2.SecretRef,
		})
		assert.Nil(t, err)

		assert.Equal(t, req2.Data, resp.Data)
		assert.Equal(t, dek.uid, resp.KeyUID)
	}

	req3 := &csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{
			Uid:             req.SecretRef.Uid,
			ResourceVersion: vutils.UUIDv7(),
		},
		Data: utilrand.GetRandomBytesMust(32),
	}

	err = srv.doSetDataSecret(ctx, req3)
	assert.Nil(t, err)

	{
		_, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: req2.SecretRef,
		})
		assert.NotNil(t, err, "%+v", err)
		assert.True(t, grpcerr.IsNotFound(err))
	}
	{
		resp, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: req3.SecretRef,
		})
		assert.Nil(t, err)

		assert.Equal(t, req3.Data, resp.Data)
		assert.Equal(t, dek2.uid, resp.KeyUID)

		itmList, err := srv.doListDataSecret(ctx, &csecretmanv1.ListSecretRequest{
			SecretRefs: []*metav1.ObjectReference{
				req3.SecretRef,
			},
		})
		assert.Nil(t, err)
		assert.True(t, len(itmList.Items) == 1)
		assert.Equal(t, resp.Data, itmList.Items[0].Data)
	}

	err = srv.doDeleteDataSecret(ctx, &csecretmanv1.DeleteSecretRequest{
		SecretRef: req3.SecretRef,
	})
	assert.Nil(t, err)

	{
		_, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: req3.SecretRef,
		})
		assert.NotNil(t, err, "%+v", err)
		assert.True(t, grpcerr.IsNotFound(err))
	}
}

func TestDataSecretAAD(t *testing.T) {
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

	getRow := func(uid string) ([]byte, string, int16) {
		ds := goqu.From(dataTable).Where(goqu.C("uid").Eq(uid)).
			Select("ciphertext", "key_uid", "aead_version")
		sqln, sqlargs, err := ds.ToSQL()
		assert.Nil(t, err)

		var ciphertext []byte
		var keyUID string
		var aeadVersion int16

		err = srv.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&ciphertext, &keyUID, &aeadVersion)
		assert.Nil(t, err)

		return ciphertext, keyUID, aeadVersion
	}

	req := &csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
		},
		Data: utilrand.GetRandomBytesMust(64),
	}

	err = srv.doSetDataSecret(ctx, req)
	assert.Nil(t, err, "%+v", err)

	ciphertext, keyUID, aeadVersion := getRow(req.SecretRef.Uid)
	assert.Equal(t, dataSecretAEADVersionV1, aeadVersion)
	assert.NotEqual(t, req.Data, ciphertext)

	{
		plaintext, err := srv.decryptData(req.SecretRef.Uid,
			req.SecretRef.ResourceVersion, keyUID, ciphertext, aeadVersion)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, req.Data, plaintext)
	}
	{
		_, err := srv.decryptData(vutils.UUIDv4(),
			req.SecretRef.ResourceVersion, keyUID, ciphertext, aeadVersion)
		assert.NotNil(t, err)
	}
	{
		_, err := srv.decryptData(req.SecretRef.Uid,
			vutils.UUIDv7(), keyUID, ciphertext, aeadVersion)
		assert.NotNil(t, err)
	}
	{
		_, err := srv.decryptData(req.SecretRef.Uid,
			req.SecretRef.ResourceVersion, vutils.UUIDv4(), ciphertext, aeadVersion)
		assert.NotNil(t, err)
	}
	{
		_, err := srv.decryptData(req.SecretRef.Uid,
			req.SecretRef.ResourceVersion, keyUID, ciphertext, dataSecretAEADVersionLegacy)
		assert.NotNil(t, err)
	}
	{
		_, err := srv.decryptData(req.SecretRef.Uid,
			req.SecretRef.ResourceVersion, keyUID, ciphertext, 99)
		assert.NotNil(t, err)
	}
	{
		tampered := append([]byte(nil), ciphertext...)
		tampered[len(tampered)-1] ^= 0x01

		_, err := srv.decryptData(req.SecretRef.Uid,
			req.SecretRef.ResourceVersion, keyUID, tampered, aeadVersion)
		assert.NotNil(t, err)
	}
}

func TestDataSecretLegacyAEAD(t *testing.T) {
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

	dek, err := srv.chooseDEK(ctx)
	assert.Nil(t, err)

	getAEADVersion := func(uid string) int16 {
		ds := goqu.From(dataTable).Where(goqu.C("uid").Eq(uid)).Select("aead_version")
		sqln, sqlargs, err := ds.ToSQL()
		assert.Nil(t, err)

		var aeadVersion int16
		err = srv.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&aeadVersion)
		assert.Nil(t, err)

		return aeadVersion
	}

	data := utilrand.GetRandomBytesMust(48)

	enc, err := dek.encrypt(data)
	assert.Nil(t, err)

	secretRef := &metav1.ObjectReference{
		Uid:             vutils.UUIDv4(),
		ResourceVersion: vutils.UUIDv7(),
	}

	now := pbutils.Now().AsTime()

	_, err = srv.db.ExecContext(ctx, `
INSERT INTO octelium_encrypted_resources
    (uid, resource_version, created_at, updated_at, key_uid, ciphertext, aead_version)
VALUES
    ($1, $2, $3, $4, $5, $6, $7)
`,
		secretRef.Uid,
		secretRef.ResourceVersion,
		now,
		now,
		enc.KeyUID,
		enc.Ciphertext,
		dataSecretAEADVersionLegacy,
	)
	assert.Nil(t, err, "%+v", err)

	assert.Equal(t, dataSecretAEADVersionLegacy, getAEADVersion(secretRef.Uid))

	{
		resp, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: secretRef,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, data, resp.Data)
		assert.Equal(t, dek.uid, resp.KeyUID)
	}
	{
		itmList, err := srv.doListDataSecret(ctx, &csecretmanv1.ListSecretRequest{
			SecretRefs: []*metav1.ObjectReference{secretRef},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(itmList.Items))
		assert.Equal(t, data, itmList.Items[0].Data)
	}

	newData := utilrand.GetRandomBytesMust(48)

	err = srv.doSetDataSecret(ctx, &csecretmanv1.SetSecretRequest{
		SecretRef: secretRef,
		Data:      newData,
	})
	assert.Nil(t, err, "%+v", err)

	assert.Equal(t, dataSecretAEADVersionV1, getAEADVersion(secretRef.Uid))

	{
		resp, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: secretRef,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, newData, resp.Data)
	}
}

func TestDataSecretUpsertSingleRow(t *testing.T) {
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

	uid := vutils.UUIDv4()

	var lastRef *metav1.ObjectReference
	var lastData []byte

	for range 4 {
		req := &csecretmanv1.SetSecretRequest{
			SecretRef: &metav1.ObjectReference{
				Uid:             uid,
				ResourceVersion: vutils.UUIDv7(),
			},
			Data: utilrand.GetRandomBytesMust(32),
		}

		err = srv.doSetDataSecret(ctx, req)
		assert.Nil(t, err, "%+v", err)

		lastRef = req.SecretRef
		lastData = req.Data
	}

	{
		var count int
		var resourceVersion string

		err := srv.db.QueryRowContext(ctx,
			`SELECT count(*), max(resource_version) FROM octelium_encrypted_resources WHERE uid = $1`,
			uid).Scan(&count, &resourceVersion)
		assert.Nil(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, lastRef.ResourceVersion, resourceVersion)
	}
	{
		resp, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: lastRef,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, lastData, resp.Data)
	}
}

func TestDataSecretDeleteStaleResourceVersion(t *testing.T) {
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

	uid := vutils.UUIDv4()

	staleRef := &metav1.ObjectReference{
		Uid:             uid,
		ResourceVersion: vutils.UUIDv7(),
	}

	err = srv.doSetDataSecret(ctx, &csecretmanv1.SetSecretRequest{
		SecretRef: staleRef,
		Data:      utilrand.GetRandomBytesMust(32),
	})
	assert.Nil(t, err)

	currentRef := &metav1.ObjectReference{
		Uid:             uid,
		ResourceVersion: vutils.UUIDv7(),
	}

	err = srv.doSetDataSecret(ctx, &csecretmanv1.SetSecretRequest{
		SecretRef: currentRef,
		Data:      utilrand.GetRandomBytesMust(32),
	})
	assert.Nil(t, err)

	err = srv.doDeleteDataSecret(ctx, &csecretmanv1.DeleteSecretRequest{
		SecretRef: staleRef,
	})
	assert.Nil(t, err)

	{
		var count int
		err := srv.db.QueryRowContext(ctx,
			`SELECT count(*) FROM octelium_encrypted_resources WHERE uid = $1`,
			uid).Scan(&count)
		assert.Nil(t, err)
		assert.Equal(t, 1, count)
	}

	err = srv.doDeleteDataSecret(ctx, &csecretmanv1.DeleteSecretRequest{
		SecretRef: currentRef,
	})
	assert.Nil(t, err)

	{
		var count int
		err := srv.db.QueryRowContext(ctx,
			`SELECT count(*) FROM octelium_encrypted_resources WHERE uid = $1`,
			uid).Scan(&count)
		assert.Nil(t, err)
		assert.Equal(t, 0, count)
	}
}

func TestDataSecretListEdgeCases(t *testing.T) {
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
		_, err := srv.doListDataSecret(ctx, nil)
		assert.NotNil(t, err)
	}
	{
		res, err := srv.doListDataSecret(ctx, &csecretmanv1.ListSecretRequest{})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(res.Items))
	}
	{
		res, err := srv.doListDataSecret(ctx, &csecretmanv1.ListSecretRequest{
			SecretRefs: []*metav1.ObjectReference{
				nil,
				{},
				{Uid: vutils.UUIDv4()},
				{ResourceVersion: vutils.UUIDv7()},
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(res.Items))
	}
	{
		res, err := srv.doListDataSecret(ctx, &csecretmanv1.ListSecretRequest{
			SecretRefs: []*metav1.ObjectReference{
				{
					Uid:             vutils.UUIDv4(),
					ResourceVersion: vutils.UUIDv7(),
				},
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(res.Items))
	}

	req1 := &csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
		},
		Data: utilrand.GetRandomBytesMust(32),
	}
	err = srv.doSetDataSecret(ctx, req1)
	assert.Nil(t, err)

	req2 := &csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
		},
		Data: utilrand.GetRandomBytesMust(32),
	}
	err = srv.doSetDataSecret(ctx, req2)
	assert.Nil(t, err)

	{
		res, err := srv.doListDataSecret(ctx, &csecretmanv1.ListSecretRequest{
			SecretRefs: []*metav1.ObjectReference{
				req1.SecretRef,
				{
					Uid:             vutils.UUIDv4(),
					ResourceVersion: vutils.UUIDv7(),
				},
				req2.SecretRef,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(res.Items))
		assert.Equal(t, req1.Data, res.Items[0].Data)
		assert.Equal(t, req2.Data, res.Items[1].Data)
	}
	{
		res, err := srv.doListDataSecret(ctx, &csecretmanv1.ListSecretRequest{
			SecretRefs: []*metav1.ObjectReference{
				req1.SecretRef,
				req1.SecretRef,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(res.Items))
		assert.Equal(t, req1.Data, res.Items[0].Data)
		assert.Equal(t, req1.Data, res.Items[1].Data)
	}
	{
		res, err := srv.doListDataSecret(ctx, &csecretmanv1.ListSecretRequest{
			SecretRefs: []*metav1.ObjectReference{
				{
					Uid:             req1.SecretRef.Uid,
					ResourceVersion: vutils.UUIDv7(),
				},
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(res.Items))
	}
}

func TestDataSecretWithoutDEK(t *testing.T) {
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

	assert.Equal(t, 0, len(srv.deks.dekMap))

	{
		_, err := srv.chooseDEK(ctx)
		assert.NotNil(t, err)
	}
	{
		_, err := srv.getDEKByUID(vutils.UUIDv4())
		assert.NotNil(t, err)
	}
	{
		err := srv.doSetDataSecret(ctx, &csecretmanv1.SetSecretRequest{
			SecretRef: &metav1.ObjectReference{
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
			},
			Data: utilrand.GetRandomBytesMust(32),
		})
		assert.NotNil(t, err)
	}
	{
		_, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: &metav1.ObjectReference{
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err))
	}
}

func TestValidateDataSecretRequests(t *testing.T) {
	uid := vutils.UUIDv4()
	resourceVersion := vutils.UUIDv7()

	assert.NotNil(t, validateGetSecretRequest(nil))
	assert.NotNil(t, validateGetSecretRequest(&csecretmanv1.GetSecretRequest{}))
	assert.NotNil(t, validateGetSecretRequest(&csecretmanv1.GetSecretRequest{
		SecretRef: &metav1.ObjectReference{ResourceVersion: resourceVersion},
	}))
	assert.NotNil(t, validateGetSecretRequest(&csecretmanv1.GetSecretRequest{
		SecretRef: &metav1.ObjectReference{Uid: uid},
	}))
	assert.Nil(t, validateGetSecretRequest(&csecretmanv1.GetSecretRequest{
		SecretRef: &metav1.ObjectReference{Uid: uid, ResourceVersion: resourceVersion},
	}))

	assert.NotNil(t, validateSetSecretRequest(nil))
	assert.NotNil(t, validateSetSecretRequest(&csecretmanv1.SetSecretRequest{}))
	assert.NotNil(t, validateSetSecretRequest(&csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{ResourceVersion: resourceVersion},
		Data:      []byte("octelium"),
	}))
	assert.NotNil(t, validateSetSecretRequest(&csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{Uid: uid},
		Data:      []byte("octelium"),
	}))
	assert.NotNil(t, validateSetSecretRequest(&csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{Uid: uid, ResourceVersion: resourceVersion},
	}))
	assert.NotNil(t, validateSetSecretRequest(&csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{Uid: uid, ResourceVersion: resourceVersion},
		Data:      []byte{},
	}))
	assert.Nil(t, validateSetSecretRequest(&csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{Uid: uid, ResourceVersion: resourceVersion},
		Data:      []byte("octelium"),
	}))

	assert.NotNil(t, validateDeleteSecretRequest(nil))
	assert.NotNil(t, validateDeleteSecretRequest(&csecretmanv1.DeleteSecretRequest{}))
	assert.NotNil(t, validateDeleteSecretRequest(&csecretmanv1.DeleteSecretRequest{
		SecretRef: &metav1.ObjectReference{ResourceVersion: resourceVersion},
	}))
	assert.NotNil(t, validateDeleteSecretRequest(&csecretmanv1.DeleteSecretRequest{
		SecretRef: &metav1.ObjectReference{Uid: uid},
	}))
	assert.Nil(t, validateDeleteSecretRequest(&csecretmanv1.DeleteSecretRequest{
		SecretRef: &metav1.ObjectReference{Uid: uid, ResourceVersion: resourceVersion},
	}))
}

func TestGetSecretRefKey(t *testing.T) {
	assert.Equal(t, getSecretRefKey("uid", "rv"), getSecretRefKey("uid", "rv"))
	assert.NotEqual(t, getSecretRefKey("uid", "rv"), getSecretRefKey("rv", "uid"))
	assert.NotEqual(t, getSecretRefKey("a", "bc"), getSecretRefKey("ab", "c"))
	assert.NotEqual(t, getSecretRefKey("", ""), getSecretRefKey("", "x"))
}

func TestGetDataSecretAAD(t *testing.T) {
	uid := vutils.UUIDv4()
	resourceVersion := vutils.UUIDv7()
	keyUID := vutils.UUIDv4()

	aad := getDataSecretAAD(uid, resourceVersion, keyUID)
	assert.NotEmpty(t, aad)

	assert.Equal(t, aad, getDataSecretAAD(uid, resourceVersion, keyUID))
	assert.NotEqual(t, aad, getDataSecretAAD(vutils.UUIDv4(), resourceVersion, keyUID))
	assert.NotEqual(t, aad, getDataSecretAAD(uid, vutils.UUIDv7(), keyUID))
	assert.NotEqual(t, aad, getDataSecretAAD(uid, resourceVersion, vutils.UUIDv4()))
	assert.NotEqual(t, getDataSecretAAD("a", "bc", keyUID), getDataSecretAAD("ab", "c", keyUID))
}
