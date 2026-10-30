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
	"path/filepath"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/raccessv1"
	"github.com/octelium/octelium/apis/rsc/rcordiumv1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func TestReconcileRepopulatesAfterDuckDBFormatReset(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	ctx := env.ctx
	usr := createUser(ctx, t, env.srv.octeliumC)
	grp := createGroup(ctx, t, env.srv.octeliumC)
	require.NoError(t, env.srv.db.Close())

	database := filepath.Join(t.TempDir(), "store.db")
	restoreRscStoreTestDatabase(t, database, true)
	db, err := openRscStoreDB(ctx, database)
	require.NoError(t, err)
	env.srv.db = db
	require.NoError(t, env.srv.initDB(ctx))
	assert.False(t, localExists(ctx, t, env.srv, "legacy-user"))
	assert.False(t, localExists(ctx, t, env.srv, "wal-user"))
	assert.False(t, localExists(ctx, t, env.srv, usr.Metadata.Uid))

	require.NoError(t, env.srv.reconcileAllResources(ctx))
	assert.True(t, localExists(ctx, t, env.srv, usr.Metadata.Uid))
	assert.True(t, localExists(ctx, t, env.srv, grp.Metadata.Uid))
	assert.Equal(t, usr.Metadata.ResourceVersion, localResourceVersion(ctx, t, env.srv, usr.Metadata.Uid))

	usr.Spec.Email = "updated@example.com"
	usr, err = env.srv.octeliumC.CoreC().UpdateUser(ctx, usr)
	require.NoError(t, err)
	require.NoError(t, env.srv.reconcileAllResources(ctx))
	assert.Equal(t, usr.Metadata.ResourceVersion, localResourceVersion(ctx, t, env.srv, usr.Metadata.Uid))

	require.NoError(t, db.Close())
	db, err = openRscStoreDB(ctx, database)
	require.NoError(t, err)
	env.srv.db = db
	require.NoError(t, env.srv.initDB(ctx))
	assert.True(t, localExists(ctx, t, env.srv, grp.Metadata.Uid))
	assert.Equal(t, usr.Metadata.ResourceVersion, localResourceVersion(ctx, t, env.srv, usr.Metadata.Uid))
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

func TestReconcileDoesNotResurrectRemovedResource(t *testing.T) {
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

	reconcileKind(ctx, t, srv, ucorev1.KindUser)
	assert.True(t, localExists(ctx, t, srv, u1.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, u2.Metadata.Uid))

	err = srv.removeResource(ctx, u2)
	assert.Nil(t, err, "%+v", err)
	assert.False(t, localExists(ctx, t, srv, u2.Metadata.Uid))

	err = srv.insertResource(ctx, u2)
	assert.Nil(t, err, "%+v", err)
	assert.False(t, localExists(ctx, t, srv, u2.Metadata.Uid))

	reconcileKind(ctx, t, srv, ucorev1.KindUser)

	assert.False(t, localExists(ctx, t, srv, u2.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, u1.Metadata.Uid))
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

type rscStoreReconcileConn struct {
	invoke func(context.Context, string, any, any) error
}

func (c *rscStoreReconcileConn) Invoke(ctx context.Context, method string, req, res any, opts ...grpc.CallOption) error {
	return c.invoke(ctx, method, req, res)
}

func (c *rscStoreReconcileConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Error(codes.Unimplemented, "unexpected stream")
}

type rscStoreReconcileClient struct {
	octeliumc.ClientInterface
	conn *rscStoreReconcileConn
}

func (c *rscStoreReconcileClient) CoreC() rcorev1.ResourceServiceClient {
	return rcorev1.NewResourceServiceClient(c.conn)
}

func (c *rscStoreReconcileClient) EnterpriseC() renterprisev1.ResourceServiceClient {
	return renterprisev1.NewResourceServiceClient(c.conn)
}

func (c *rscStoreReconcileClient) AccessC() raccessv1.ResourceServiceClient {
	return raccessv1.NewResourceServiceClient(c.conn)
}

func (c *rscStoreReconcileClient) CordiumC() rcordiumv1.ResourceServiceClient {
	return rcordiumv1.NewResourceServiceClient(c.conn)
}

type rscStoreReconcileTest struct {
	name      string
	kind      resourceKind
	method    string
	list      func(*Server, context.Context, string) ([]umetav1.ResourceObjectI, error)
	reconcile func(*Server, context.Context, resourceKind) error
}

func rscStoreReconcileCases() []rscStoreReconcileTest {
	return []rscStoreReconcileTest{
		{name: "Core", kind: resourceKind{api: "core", version: "v1", kind: "User"}, method: rcorev1.ResourceService_ListUser_FullMethodName, list: (*Server).listAllCoreResourcesByKind, reconcile: (*Server).reconcileCoreResourceKind},
		{name: "Enterprise", kind: resourceKind{api: "enterprise", version: "v1", kind: "CollectorExporter"}, method: renterprisev1.ResourceService_ListCollectorExporter_FullMethodName, list: (*Server).listAllEnterpriseResourcesByKind, reconcile: (*Server).reconcileEnterpriseResourceKind},
		{name: "Access", kind: resourceKind{api: "access", version: "v1", kind: "Policy"}, method: raccessv1.ResourceService_ListPolicy_FullMethodName, list: (*Server).listAllAccessResourcesByKind, reconcile: (*Server).reconcileAccessResourceKind},
		{name: "Cordium", kind: resourceKind{api: "cordium", version: "v1", kind: "Space"}, method: rcordiumv1.ResourceService_ListSpace_FullMethodName, list: (*Server).listAllCordiumResourcesByKind, reconcile: (*Server).reconcileCordiumResourceKind},
	}
}

func newRscStoreReconcileResource(t *testing.T, kind resourceKind, name string) umetav1.ResourceObjectI {
	t.Helper()
	ret, err := ovutils.NewResourceObject(kind.api, kind.version, kind.kind)
	require.Nil(t, err)
	msg := ret.ProtoReflect()
	fields := msg.Descriptor().Fields()
	msg.Set(fields.ByName("apiVersion"), protoreflect.ValueOfString(kind.api+"/"+kind.version))
	msg.Set(fields.ByName("kind"), protoreflect.ValueOfString(kind.kind))
	msg.Set(fields.ByName("metadata"), protoreflect.ValueOfMessage(newRscStoreMetadata(name, time.Now()).ProtoReflect()))
	msg.Mutable(fields.ByName("spec"))
	msg.Mutable(fields.ByName("status"))
	return ret
}

func setRscStoreReconcilePage(t *testing.T, res any, kind resourceKind, items []umetav1.ResourceObjectI, hasMore bool) {
	t.Helper()
	list, err := ovutils.NewResourceObjectList(kind.api, kind.version, kind.kind)
	require.Nil(t, err)
	msg := list.ProtoReflect()
	fields := msg.Descriptor().Fields()
	msg.Set(fields.ByName("listResponseMeta"), protoreflect.ValueOfMessage((&metav1.ListResponseMeta{HasMore: hasMore}).ProtoReflect()))
	values := msg.Mutable(fields.ByName("items")).List()
	for _, item := range items {
		values.Append(protoreflect.ValueOfMessage(item.ProtoReflect()))
	}
	proto.Merge(res.(proto.Message), list)
}

func TestReconcileSourcePaginationAndFailures(t *testing.T) {
	for _, tt := range rscStoreReconcileCases() {
		t.Run(tt.name, func(t *testing.T) {
			env := newRscStoreTestEnv(t)
			require.NotNil(t, env)
			first := newRscStoreReconcileResource(t, tt.kind, "first")
			second := newRscStoreReconcileResource(t, tt.kind, "second")
			conn := &rscStoreReconcileConn{}
			env.srv.octeliumC = &rscStoreReconcileClient{ClientInterface: env.srv.octeliumC, conn: conn}
			calls := 0
			conn.invoke = func(ctx context.Context, method string, req, res any) error {
				assert.Equal(t, tt.method, method)
				opts := req.(*rmetav1.ListOptions)
				assert.True(t, opts.Paginate)
				assert.Equal(t, uint32(500), opts.ItemsPerPage)
				assert.Equal(t, uint32(calls), opts.Page)
				calls++
				item := first
				if opts.Page == 1 {
					item = second
				}
				setRscStoreReconcilePage(t, res, tt.kind, []umetav1.ResourceObjectI{item}, opts.Page == 0)
				return nil
			}
			items, err := tt.list(env.srv, env.ctx, tt.kind.kind)
			require.Nil(t, err)
			require.Len(t, items, 2)
			assert.Equal(t, 2, calls)
			assert.Equal(t, first.GetMetadata().Uid, items[0].GetMetadata().Uid)
			assert.Equal(t, second.GetMetadata().Uid, items[1].GetMetadata().Uid)
			_, err = tt.list(env.srv, env.ctx, "UnsupportedKind")
			assert.ErrorContains(t, err, "unsupported")

			insertRscStoreObject(t, env, first)
			failure := status.Error(codes.Unavailable, "source unavailable")
			calls = 0
			conn.invoke = func(ctx context.Context, method string, req, res any) error {
				calls++
				if req.(*rmetav1.ListOptions).Page == 0 {
					setRscStoreReconcilePage(t, res, tt.kind, []umetav1.ResourceObjectI{second}, true)
					return nil
				}
				return failure
			}
			err = tt.reconcile(env.srv, env.ctx, tt.kind)
			assert.Equal(t, failure, err)
			assert.Equal(t, 2, calls)
			assert.True(t, localExists(env.ctx, t, env.srv, first.GetMetadata().Uid))
			assert.False(t, localExists(env.ctx, t, env.srv, second.GetMetadata().Uid))

			ctx, cancel := context.WithCancel(env.ctx)
			defer cancel()
			conn.invoke = func(ctx context.Context, method string, req, res any) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				setRscStoreReconcilePage(t, res, tt.kind, []umetav1.ResourceObjectI{second}, true)
				cancel()
				return nil
			}
			items, err = tt.list(env.srv, ctx, tt.kind.kind)
			assert.ErrorIs(t, err, context.Canceled)
			assert.Empty(t, items)
		})
	}
}

func TestReconcileTransactionRollbackAndRecovery(t *testing.T) {
	for _, tt := range rscStoreReconcileCases() {
		t.Run(tt.name, func(t *testing.T) {
			env := newRscStoreTestEnv(t)
			require.NotNil(t, env)
			original := newRscStoreReconcileResource(t, tt.kind, "original")
			orphan := newRscStoreReconcileResource(t, tt.kind, "orphan")
			insertRscStoreObject(t, env, original)
			insertRscStoreObject(t, env, orphan)
			_, err := env.srv.db.ExecContext(env.ctx, `ALTER TABLE resources ALTER COLUMN name SET NOT NULL`)
			require.Nil(t, err)
			updated := proto.Clone(original).(umetav1.ResourceObjectI)
			updated.GetMetadata().ResourceVersion = vutils.UUIDv7()
			updated.GetMetadata().Description = "new description"
			invalid := newRscStoreReconcileResource(t, tt.kind, "invalid")
			invalid.GetMetadata().Name = ""
			conn := &rscStoreReconcileConn{invoke: func(ctx context.Context, method string, req, res any) error {
				setRscStoreReconcilePage(t, res, tt.kind, []umetav1.ResourceObjectI{updated, invalid}, false)
				return nil
			}}
			env.srv.octeliumC = &rscStoreReconcileClient{ClientInterface: env.srv.octeliumC, conn: conn}
			err = tt.reconcile(env.srv, env.ctx, tt.kind)
			require.NotNil(t, err)
			assert.Equal(t, original.GetMetadata().ResourceVersion, localResourceVersion(env.ctx, t, env.srv, original.GetMetadata().Uid))
			assert.True(t, localExists(env.ctx, t, env.srv, orphan.GetMetadata().Uid))
			assert.False(t, localExists(env.ctx, t, env.srv, invalid.GetMetadata().Uid))
			invalid.GetMetadata().Name = "repaired"
			require.Nil(t, tt.reconcile(env.srv, env.ctx, tt.kind))
			assert.Equal(t, updated.GetMetadata().ResourceVersion, localResourceVersion(env.ctx, t, env.srv, original.GetMetadata().Uid))
			assert.False(t, localExists(env.ctx, t, env.srv, orphan.GetMetadata().Uid))
			assert.True(t, localExists(env.ctx, t, env.srv, invalid.GetMetadata().Uid))
		})
	}
}

func TestReconciliationLoopCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan struct{})
	go func() {
		defer close(done)
		(&Server{}).startReconciliationLoop(ctx)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Reconciliation loop did not stop")
	}
}
