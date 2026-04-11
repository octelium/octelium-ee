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

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/secretman/secretman/migrations"
	"github.com/octelium/octelium/apis/cluster/csecretmanv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
	"github.com/octelium/octelium/cluster/common/vutils"
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
	err = srv.doCreateDataSecret(ctx, req)
	assert.Nil(t, err)

	resp, err := srv.doGetDataSecret(ctx, &csecretmanv1.GetSecretRequest{
		SecretRef: req.SecretRef,
	})
	assert.Nil(t, err)

	assert.Equal(t, req.Data, resp.Data)
	assert.Equal(t, dek.uid, resp.KeyUID)

	req2 := &csecretmanv1.SetSecretRequest{
		SecretRef: &metav1.ObjectReference{
			Uid:             req.SecretRef.Uid,
			ResourceVersion: vutils.UUIDv7(),
		},
		Data: utilrand.GetRandomBytesMust(32),
	}

	err = srv.doUpdateDataSecret(ctx, req2)
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

	err = srv.doUpdateDataSecret(ctx, req3)
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
