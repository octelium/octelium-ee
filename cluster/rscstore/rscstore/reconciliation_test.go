// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"context"
	"fmt"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func setupReconcileServer(ctx context.Context, t *testing.T, octeliumC octeliumc.ClientInterface) *Server {
	srv, err := newServer(ctx, octeliumC)
	assert.Nil(t, err, "%+v", err)

	err = srv.initDB(ctx)
	assert.Nil(t, err, "%+v", err)

	return srv
}

func createUser(ctx context.Context, t *testing.T, octeliumC octeliumc.ClientInterface) *corev1.User {
	usr, err := octeliumC.CoreC().CreateUser(ctx, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type:  corev1.User_Spec_HUMAN,
			Email: fmt.Sprintf("%s@example.com", utilrand.GetRandomStringCanonical(9)),
		},
	})
	assert.Nil(t, err, "%+v", err)

	return usr
}

func createGroup(ctx context.Context, t *testing.T, octeliumC octeliumc.ClientInterface) *corev1.Group {
	grp, err := octeliumC.CoreC().CreateGroup(ctx, &corev1.Group{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindGroup,
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.Group_Spec{},
	})
	assert.Nil(t, err, "%+v", err)

	return grp
}

func localExists(ctx context.Context, t *testing.T, srv *Server, uid string) bool {
	var n int
	err := srv.db.QueryRowContext(ctx,
		`SELECT count(*) FROM resources WHERE uid = ?`, uid).Scan(&n)
	assert.Nil(t, err, "%+v", err)

	return n > 0
}

func localResourceVersion(ctx context.Context, t *testing.T, srv *Server, uid string) string {
	var rv string
	err := srv.db.QueryRowContext(ctx,
		`SELECT resource_version FROM resources WHERE uid = ?`, uid).Scan(&rv)
	assert.Nil(t, err, "%+v", err)

	return rv
}

func reconcileKind(ctx context.Context, t *testing.T, srv *Server, kind string) {
	err := srv.reconcileCoreResourceKind(ctx, resourceKind{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    kind,
	})
	assert.Nil(t, err, "%+v", err)
}

func TestReconcileCreatesMirror(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	var uids []string
	for i := 0; i < 5; i++ {
		usr := createUser(ctx, t, fakeC.OcteliumC)
		uids = append(uids, usr.Metadata.Uid)
	}

	for _, uid := range uids {
		assert.False(t, localExists(ctx, t, srv, uid))
	}

	reconcileKind(ctx, t, srv, ucorev1.KindUser)

	for _, uid := range uids {
		assert.True(t, localExists(ctx, t, srv, uid))
	}
}

func TestReconcileRemovesOrphan(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	usr := createUser(ctx, t, fakeC.OcteliumC)
	reconcileKind(ctx, t, srv, ucorev1.KindUser)
	assert.True(t, localExists(ctx, t, srv, usr.Metadata.Uid))

	orphan := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name:            utilrand.GetRandomStringCanonical(8),
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_HUMAN,
		},
		Status: &corev1.User_Status{},
	}

	err = srv.upsertResource(ctx, orphan)
	assert.Nil(t, err, "%+v", err)
	assert.True(t, localExists(ctx, t, srv, orphan.Metadata.Uid))

	reconcileKind(ctx, t, srv, ucorev1.KindUser)

	assert.False(t, localExists(ctx, t, srv, orphan.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, usr.Metadata.Uid))
}

func TestReconcileDeletesRemovedResource(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	u1 := createUser(ctx, t, fakeC.OcteliumC)
	u2 := createUser(ctx, t, fakeC.OcteliumC)
	u3 := createUser(ctx, t, fakeC.OcteliumC)

	reconcileKind(ctx, t, srv, ucorev1.KindUser)
	assert.True(t, localExists(ctx, t, srv, u1.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, u2.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, u3.Metadata.Uid))

	_, err = fakeC.OcteliumC.CoreC().DeleteUser(ctx, &rmetav1.DeleteOptions{
		Uid: u2.Metadata.Uid,
	})
	assert.Nil(t, err, "%+v", err)

	reconcileKind(ctx, t, srv, ucorev1.KindUser)

	assert.False(t, localExists(ctx, t, srv, u2.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, u1.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, u3.Metadata.Uid))
}

func TestReconcileReflectsUpdate(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	usr := createUser(ctx, t, fakeC.OcteliumC)

	reconcileKind(ctx, t, srv, ucorev1.KindUser)
	rv1 := localResourceVersion(ctx, t, srv, usr.Metadata.Uid)
	assert.Equal(t, usr.Metadata.ResourceVersion, rv1)

	usr.Spec.Email = fmt.Sprintf("%s@example.com", utilrand.GetRandomStringCanonical(10))
	usr2, err := fakeC.OcteliumC.CoreC().UpdateUser(ctx, usr)
	assert.Nil(t, err, "%+v", err)

	reconcileKind(ctx, t, srv, ucorev1.KindUser)
	rv2 := localResourceVersion(ctx, t, srv, usr.Metadata.Uid)

	assert.Equal(t, usr2.Metadata.ResourceVersion, rv2)
	assert.NotEqual(t, rv1, rv2)
}

func TestReconcileScopedByKind(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	orphan := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name:            utilrand.GetRandomStringCanonical(8),
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_HUMAN,
		},
		Status: &corev1.User_Status{},
	}

	err = srv.upsertResource(ctx, orphan)
	assert.Nil(t, err, "%+v", err)
	assert.True(t, localExists(ctx, t, srv, orphan.Metadata.Uid))

	reconcileKind(ctx, t, srv, ucorev1.KindGroup)

	assert.True(t, localExists(ctx, t, srv, orphan.Metadata.Uid))

	reconcileKind(ctx, t, srv, ucorev1.KindUser)

	assert.False(t, localExists(ctx, t, srv, orphan.Metadata.Uid))
}

func TestReconcileAllResourcesMultipleKinds(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	usr := createUser(ctx, t, fakeC.OcteliumC)
	grp := createGroup(ctx, t, fakeC.OcteliumC)

	err = srv.reconcileAllResources(ctx)
	assert.Nil(t, err, "%+v", err)

	assert.True(t, localExists(ctx, t, srv, usr.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, grp.Metadata.Uid))
}

func TestReconcileIdempotent(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	usr := createUser(ctx, t, fakeC.OcteliumC)

	reconcileKind(ctx, t, srv, ucorev1.KindUser)
	rv1 := localResourceVersion(ctx, t, srv, usr.Metadata.Uid)

	reconcileKind(ctx, t, srv, ucorev1.KindUser)
	rv2 := localResourceVersion(ctx, t, srv, usr.Metadata.Uid)

	assert.True(t, localExists(ctx, t, srv, usr.Metadata.Uid))
	assert.Equal(t, rv1, rv2)
}
