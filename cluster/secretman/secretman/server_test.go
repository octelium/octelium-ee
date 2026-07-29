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
	"fmt"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium/apis/cluster/csecretmanv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestServer(t *testing.T) {
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
	assert.Nil(t, err, "%+v", err)

	err = srv.setDEKMap(ctx)
	assert.Nil(t, err)

	grpcSrv := grpc.NewServer()
	csecretmanv1.RegisterMainServiceServer(grpcSrv, srv)

	/*
		addr := fmt.Sprintf("localhost:%d", tests.GetPort())
		lis, err := net.Listen("tcp", addr)
		assert.Nil(t, err)

		go func() {
			grpcSrv.Serve(lis)
		}()
		time.Sleep(1 * time.Second)

		grpcConn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		assert.Nil(t, err)
		c := csecretmanv1.NewMainServiceClient(grpcConn)
	*/

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

	sec2, err := fakeC.OcteliumC.CoreC().CreateSecret(ctx, &corev1.Secret{
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

	_, err = srv.GetSecret(ctx, &csecretmanv1.GetSecretRequest{
		SecretRef: umetav1.GetObjectReference(sec),
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

	data, err := pbutils.Marshal(sec.Data)
	assert.Nil(t, err)
	_, err = srv.SetSecret(ctx, &csecretmanv1.SetSecretRequest{
		SecretRef: umetav1.GetObjectReference(sec),
		Data:      data,
	})
	assert.Nil(t, err)

	_, err = srv.SetSecret(ctx, &csecretmanv1.SetSecretRequest{
		SecretRef: umetav1.GetObjectReference(sec2),
		Data:      data,
	})
	assert.Nil(t, err, "%+v", err)

	secR, err := srv.GetSecret(ctx, &csecretmanv1.GetSecretRequest{
		SecretRef: umetav1.GetObjectReference(sec),
	})
	assert.Nil(t, err)

	dataR := &corev1.Secret_Data{}
	err = pbutils.Unmarshal(secR.Data, dataR)
	assert.Nil(t, err)

	assert.True(t, pbutils.IsEqual(dataR, sec.Data))

	itmList, err := srv.ListSecret(ctx, &csecretmanv1.ListSecretRequest{
		SecretRefs: []*metav1.ObjectReference{
			umetav1.GetObjectReference(sec),
			umetav1.GetObjectReference(sec2),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(itmList.Items))
	assert.Equal(t, sec.Metadata.Uid, itmList.Items[0].SecretRef.Uid)

	{

		itmList, err := srv.ListSecret(ctx, &csecretmanv1.ListSecretRequest{
			SecretRefs: []*metav1.ObjectReference{
				umetav1.GetObjectReference(sec2),
				umetav1.GetObjectReference(sec),
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(itmList.Items))
		assert.Equal(t, sec2.Metadata.Uid, itmList.Items[0].SecretRef.Uid)
		assert.Equal(t, sec.Metadata.Uid, itmList.Items[1].SecretRef.Uid)
	}

	_, err = srv.DeleteSecret(ctx, &csecretmanv1.DeleteSecretRequest{
		SecretRef: umetav1.GetObjectReference(sec),
	})
	assert.Nil(t, err)

	_, err = srv.GetSecret(ctx, &csecretmanv1.GetSecretRequest{
		SecretRef: umetav1.GetObjectReference(sec),
	})
	assert.NotNil(t, err)
	assert.True(t, grpcerr.IsNotFound(err))

	{
		itmList, err := srv.ListSecret(ctx, &csecretmanv1.ListSecretRequest{
			SecretRefs: []*metav1.ObjectReference{
				umetav1.GetObjectReference(sec2),
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(itmList.Items))
		assert.Equal(t, sec2.Metadata.Uid, itmList.Items[0].SecretRef.Uid)
	}
}

func TestSetDEKMap(t *testing.T) {
	srv := &server{}
	srv.deks.dekMap = make(map[string]*dek)

	now := time.Now()

	dek1 := &dek{
		uid:       vutils.UUIDv4(),
		key:       utilrand.GetRandomBytesMust(32),
		createdAt: now.Add(-2 * time.Hour),
	}
	dek2 := &dek{
		uid:       vutils.UUIDv4(),
		key:       utilrand.GetRandomBytesMust(32),
		createdAt: now.Add(-1 * time.Hour),
	}
	dek3 := &dek{
		uid:       vutils.UUIDv4(),
		key:       utilrand.GetRandomBytesMust(32),
		createdAt: now,
	}

	srv.doSetDEKMap([]*dek{dek2, dek3, dek1})

	assert.Equal(t, 3, len(srv.deks.dekMap))
	assert.Equal(t, dek3.uid, srv.deks.cur.uid)

	for _, itm := range []*dek{dek1, dek2, dek3} {
		got, err := srv.getDEKByUID(itm.uid)
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, itm.uid, got.uid)
		assert.Equal(t, itm.key, got.key)
	}

	{
		cur, err := srv.chooseDEK(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, dek3.uid, cur.uid)
	}

	srv.doSetDEKMap([]*dek{dek1})

	assert.Equal(t, 1, len(srv.deks.dekMap))
	assert.Equal(t, dek1.uid, srv.deks.cur.uid)

	{
		_, err := srv.getDEKByUID(dek3.uid)
		assert.NotNil(t, err)
	}

	srv.doSetDEKMap(nil)

	assert.Equal(t, 0, len(srv.deks.dekMap))
	assert.Equal(t, dek1.uid, srv.deks.cur.uid)
}

func TestInitRootDEKIdempotency(t *testing.T) {
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
	assert.Nil(t, err, "%+v", err)

	deks, err := srv.doListDEK(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(deks))

	for range 3 {
		err = srv.initRootDEK(ctx)
		assert.Nil(t, err, "%+v", err)
	}

	deks2, err := srv.doListDEK(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(deks2))
	assert.Equal(t, deks[0].uid, deks2[0].uid)
	assert.Equal(t, deks[0].key, deks2[0].key)

	srv2, err := newServer(ctx, fakeC.OcteliumC, db)
	assert.Nil(t, err)

	err = srv2.initRootDEK(ctx)
	assert.Nil(t, err)

	err = srv2.setDEKMap(ctx)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(srv2.deks.dekMap))
	assert.Equal(t, deks[0].uid, srv2.deks.cur.uid)
}

func TestAPIRequestValidation(t *testing.T) {
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

	validRef := &metav1.ObjectReference{
		Uid:             vutils.UUIDv4(),
		ResourceVersion: vutils.UUIDv7(),
	}

	{
		_, err := srv.GetSecret(ctx, nil)
		assert.NotNil(t, err)

		_, err = srv.GetSecret(ctx, &csecretmanv1.GetSecretRequest{})
		assert.NotNil(t, err)

		_, err = srv.GetSecret(ctx, &csecretmanv1.GetSecretRequest{
			SecretRef: &metav1.ObjectReference{Uid: validRef.Uid},
		})
		assert.NotNil(t, err)
	}
	{
		_, err := srv.SetSecret(ctx, nil)
		assert.NotNil(t, err)

		_, err = srv.SetSecret(ctx, &csecretmanv1.SetSecretRequest{})
		assert.NotNil(t, err)

		_, err = srv.SetSecret(ctx, &csecretmanv1.SetSecretRequest{
			SecretRef: validRef,
		})
		assert.NotNil(t, err)
	}
	{
		_, err := srv.DeleteSecret(ctx, nil)
		assert.NotNil(t, err)

		_, err = srv.DeleteSecret(ctx, &csecretmanv1.DeleteSecretRequest{})
		assert.NotNil(t, err)

		_, err = srv.DeleteSecret(ctx, &csecretmanv1.DeleteSecretRequest{
			SecretRef: &metav1.ObjectReference{ResourceVersion: validRef.ResourceVersion},
		})
		assert.NotNil(t, err)
	}
	{
		_, err := srv.ListSecret(ctx, nil)
		assert.NotNil(t, err)

		res, err := srv.ListSecret(ctx, &csecretmanv1.ListSecretRequest{})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(res.Items))
	}
	{
		_, err := srv.SetSecret(ctx, &csecretmanv1.SetSecretRequest{
			SecretRef: validRef,
			Data:      utilrand.GetRandomBytesMust(32),
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.DeleteSecret(ctx, &csecretmanv1.DeleteSecretRequest{
			SecretRef: validRef,
		})
		assert.Nil(t, err)

		_, err = srv.DeleteSecret(ctx, &csecretmanv1.DeleteSecretRequest{
			SecretRef: validRef,
		})
		assert.Nil(t, err)
	}
}

func TestMigrations(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	db, err := postgresutils.NewDB()
	assert.Nil(t, err)

	for range 3 {
		err = migrations.Migrate(ctx, db)
		assert.Nil(t, err, "%+v", err)
	}

	uid := vutils.UUIDv4()
	now := pbutils.Now().AsTime()

	_, err = db.ExecContext(ctx,
		`DROP INDEX IF EXISTS idx_octelium_encrypted_resources_uid_unique`)
	assert.Nil(t, err, "%+v", err)

	for i := range 3 {
		_, err = db.ExecContext(ctx, `
INSERT INTO octelium_encrypted_resources
    (uid, resource_version, created_at, updated_at, key_uid, ciphertext, aead_version)
VALUES
    ($1, $2, $3, $4, $5, $6, $7)
`,
			uid,
			fmt.Sprintf("rv-%d", i),
			now.Add(time.Duration(i)*time.Second),
			now.Add(time.Duration(i)*time.Second),
			"kek",
			utilrand.GetRandomBytesMust(32),
			dataSecretAEADVersionV1,
		)
		assert.Nil(t, err, "%+v", err)
	}

	{
		var count int
		err := db.QueryRowContext(ctx,
			`SELECT count(*) FROM octelium_encrypted_resources WHERE uid = $1`, uid).Scan(&count)
		assert.Nil(t, err)
		assert.Equal(t, 3, count)
	}

	err = migrations.Migrate(ctx, db)
	assert.Nil(t, err, "%+v", err)

	{
		var count int
		var resourceVersion string

		err := db.QueryRowContext(ctx,
			`SELECT count(*), max(resource_version) FROM octelium_encrypted_resources WHERE uid = $1`,
			uid).Scan(&count, &resourceVersion)
		assert.Nil(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, "rv-2", resourceVersion)
	}

	{
		_, err := db.ExecContext(ctx, `
INSERT INTO octelium_encrypted_resources
    (uid, resource_version, created_at, updated_at, key_uid, ciphertext, aead_version)
VALUES
    ($1, $2, $3, $4, $5, $6, $7)
`,
			uid,
			"rv-3",
			now,
			now,
			"kek",
			utilrand.GetRandomBytesMust(32),
			dataSecretAEADVersionV1,
		)
		assert.NotNil(t, err)
	}
}
