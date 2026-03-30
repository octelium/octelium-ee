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
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/postgresutils"
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
