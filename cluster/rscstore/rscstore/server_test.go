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
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type rscStoreTestEnv struct {
	ctx context.Context
	srv *Server
}

func newRscStoreTestEnv(t *testing.T) *rscStoreTestEnv {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		cancel()
		return nil
	}

	srv, err := newServer(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		cancel()
		tst.Destroy()
		return nil
	}

	err = srv.initDB(ctx)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		cancel()
		_ = srv.db.Close()
		tst.Destroy()
		return nil
	}

	t.Cleanup(func() {
		cancel()

		assert.Nil(t, srv.db.Close())
		tst.Destroy()
	})

	return &rscStoreTestEnv{
		ctx: ctx,
		srv: srv,
	}
}

func newRscStoreMetadata(name string, createdAt time.Time) *metav1.Metadata {
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return &metav1.Metadata{
		Name:            name,
		Uid:             vutils.UUIDv4(),
		ResourceVersion: vutils.UUIDv7(),
		CreatedAt:       pbutils.Timestamp(createdAt),
	}
}

func insertRscStoreObject(t *testing.T, env *rscStoreTestEnv, rsc umetav1.ResourceObjectI) {
	t.Helper()

	err := env.srv.upsertResource(env.ctx, rsc)
	assert.Nil(t, err, "%+v", err)
}

func insertRawRscStoreResource(t *testing.T, env *rscStoreTestEnv, kind string, rsc map[string]any) {
	t.Helper()

	metadata, ok := rsc["metadata"].(map[string]any)
	assert.True(t, ok)
	if !ok {
		return
	}

	uid, _ := metadata["uid"].(string)
	resourceVersion, _ := metadata["resourceVersion"].(string)

	rscJSON, err := json.Marshal(rsc)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	rscStr, err := env.srv.getRSCStr(rscJSON)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	_, err = env.srv.db.ExecContext(env.ctx, `DELETE FROM resources WHERE uid = ?`, uid)
	assert.Nil(t, err, "%+v", err)

	_, err = env.srv.db.ExecContext(env.ctx, getResourceInsertQuery(),
		getResourceInsertArgs(ucorev1.API, ucorev1.Version, kind, uid, resourceVersion,
			string(rscJSON), rscStr)...)
	assert.Nil(t, err, "%+v", err)
}

func newRawRscStoreResource(kind, name string, hidden bool, spec, status map[string]any) map[string]any {
	if spec == nil {
		spec = map[string]any{}
	}
	if status == nil {
		status = map[string]any{}
	}

	return map[string]any{
		"apiVersion": ucorev1.APIVersion,
		"kind":       kind,
		"metadata": map[string]any{
			"name":            name,
			"uid":             vutils.UUIDv4(),
			"resourceVersion": vutils.UUIDv7(),
			"createdAt":       time.Now().UTC().Format(time.RFC3339Nano),
			"isSystemHidden":  hidden,
		},
		"spec":   spec,
		"status": status,
	}
}

func getStoredRscStoreResource(t *testing.T, env *rscStoreTestEnv, uid string) map[string]any {
	t.Helper()

	ret := make(map[string]any)
	err := env.srv.db.QueryRowContext(env.ctx, `SELECT rsc FROM resources WHERE uid = ?`, uid).Scan(&ret)
	assert.Nil(t, err, "%+v", err)

	return ret
}

func getStoredRscStoreResourceVersion(t *testing.T, env *rscStoreTestEnv, uid string) string {
	t.Helper()

	var ret string
	err := env.srv.db.QueryRowContext(env.ctx, `SELECT resource_version FROM resources WHERE uid = ?`, uid).Scan(&ret)
	assert.Nil(t, err, "%+v", err)

	return ret
}

func getRscStoreResourceCount(t *testing.T, env *rscStoreTestEnv, kind string) int {
	t.Helper()

	var ret int
	err := env.srv.db.QueryRowContext(env.ctx, `SELECT COUNT(*) FROM resources WHERE api = ? AND version = ? AND kind = ?`,
		ucorev1.API, ucorev1.Version, kind).Scan(&ret)
	assert.Nil(t, err, "%+v", err)

	return ret
}

func TestTransformRsc(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	{
		rsc := &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata: &metav1.Metadata{
				Name:            utilrand.GetRandomStringCanonical(8),
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
				CreatedAt:       pbutils.Timestamp(time.Now().UTC().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 500) * int(time.Minute)))),
			},
			Spec: &corev1.User_Spec{
				Type: func() corev1.User_Spec_Type {
					if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
						return corev1.User_Spec_HUMAN
					}
					return corev1.User_Spec_WORKLOAD
				}(),
				Email: fmt.Sprintf("%s@example.com", utilrand.GetRandomStringCanonical(9)),
			},
			Status: &corev1.User_Status{},
		}

		rscJSON, err := pbutils.MarshalJSON(rsc, false)
		assert.Nil(t, err)
		_, err = srv.getRSCStr(rscJSON)
		assert.Nil(t, err)

	}

}

type rscStoreWatchTest struct {
	name   string
	create func(context.Context) (umetav1.ResourceObjectI, error)
	update func(context.Context, umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error)
	remove func(context.Context, string) error
}

func newRscStoreWatchTest[T umetav1.ResourceObjectI](api, kind string, item T,
	create func(context.Context, T, ...grpc.CallOption) (T, error),
	update func(context.Context, T, ...grpc.CallOption) (T, error),
	remove func(context.Context, *rmetav1.DeleteOptions, ...grpc.CallOption) (*rmetav1.OperationResult, error)) rscStoreWatchTest {
	ret := rscStoreWatchTest{
		name: api + "/" + kind,
		create: func(ctx context.Context) (umetav1.ResourceObjectI, error) {
			obj := proto.Clone(item).(T)
			msg := obj.ProtoReflect()
			fields := msg.Descriptor().Fields()
			msg.Set(fields.ByName("apiVersion"), protoreflect.ValueOfString(api+"/v1"))
			msg.Set(fields.ByName("kind"), protoreflect.ValueOfString(kind))
			msg.Set(fields.ByName("metadata"), protoreflect.ValueOfMessage((&metav1.Metadata{Name: "watch-" + utilrand.GetRandomStringCanonical(8)}).ProtoReflect()))
			msg.Mutable(fields.ByName("spec"))
			msg.Mutable(fields.ByName("status"))
			return create(ctx, obj)
		},
		update: func(ctx context.Context, obj umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
			arg := proto.Clone(obj).(T)
			arg.GetMetadata().Description = "updated-watch-projection"
			return update(ctx, arg)
		},
		remove: func(ctx context.Context, uid string) error {
			_, err := remove(ctx, &rmetav1.DeleteOptions{Uid: uid})
			return err
		},
	}
	if update == nil {
		ret.update = nil
	}
	return ret
}

func TestResourceWatchLifecycles(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	ctx, cancel := context.WithCancel(env.ctx)
	defer cancel()
	require.Nil(t, env.srv.validateCoreReconcileKinds())
	require.Nil(t, env.srv.setResources(ctx))
	require.Nil(t, env.srv.watchCordiumResources(ctx))
	require.Nil(t, env.srv.watchCordiumResources(ctx))
	core := env.srv.octeliumC.CoreC()
	enterprise := env.srv.octeliumC.EnterpriseC()
	access := env.srv.octeliumC.AccessC()
	cordium := env.srv.octeliumC.CordiumC()
	cases := []rscStoreWatchTest{
		newRscStoreWatchTest("core", "User", &corev1.User{}, core.CreateUser, core.UpdateUser, core.DeleteUser),
		newRscStoreWatchTest("core", "Session", &corev1.Session{}, core.CreateSession, core.UpdateSession, core.DeleteSession),
		newRscStoreWatchTest("core", "Device", &corev1.Device{}, core.CreateDevice, core.UpdateDevice, core.DeleteDevice),
		newRscStoreWatchTest("core", "Group", &corev1.Group{}, core.CreateGroup, core.UpdateGroup, core.DeleteGroup),
		newRscStoreWatchTest("core", "Service", &corev1.Service{}, core.CreateService, core.UpdateService, core.DeleteService),
		newRscStoreWatchTest("core", "Namespace", &corev1.Namespace{}, core.CreateNamespace, core.UpdateNamespace, core.DeleteNamespace),
		newRscStoreWatchTest("core", "Secret", &corev1.Secret{}, core.CreateSecret, core.UpdateSecret, core.DeleteSecret),
		newRscStoreWatchTest("core", "Policy", &corev1.Policy{}, core.CreatePolicy, core.UpdatePolicy, core.DeletePolicy),
		newRscStoreWatchTest("core", "IdentityProvider", &corev1.IdentityProvider{}, core.CreateIdentityProvider, core.UpdateIdentityProvider, core.DeleteIdentityProvider),
		newRscStoreWatchTest("core", "Authenticator", &corev1.Authenticator{}, core.CreateAuthenticator, core.UpdateAuthenticator, core.DeleteAuthenticator),
		newRscStoreWatchTest("core", "Region", &corev1.Region{}, core.CreateRegion, core.UpdateRegion, core.DeleteRegion),
		newRscStoreWatchTest("core", "Credential", &corev1.Credential{}, core.CreateCredential, core.UpdateCredential, core.DeleteCredential),
		newRscStoreWatchTest("core", "Gateway", &corev1.Gateway{}, core.CreateGateway, core.UpdateGateway, core.DeleteGateway),
		newRscStoreWatchTest("enterprise", "CollectorExporter", &enterprisev1.CollectorExporter{}, enterprise.CreateCollectorExporter, enterprise.UpdateCollectorExporter, enterprise.DeleteCollectorExporter),
		newRscStoreWatchTest("enterprise", "Secret", &enterprisev1.Secret{}, enterprise.CreateSecret, enterprise.UpdateSecret, enterprise.DeleteSecret),
		newRscStoreWatchTest("enterprise", "SecretStore", &enterprisev1.SecretStore{}, enterprise.CreateSecretStore, enterprise.UpdateSecretStore, enterprise.DeleteSecretStore),
		newRscStoreWatchTest("enterprise", "Certificate", &enterprisev1.Certificate{}, enterprise.CreateCertificate, enterprise.UpdateCertificate, enterprise.DeleteCertificate),
		newRscStoreWatchTest("enterprise", "CertificateIssuer", &enterprisev1.CertificateIssuer{}, enterprise.CreateCertificateIssuer, enterprise.UpdateCertificateIssuer, enterprise.DeleteCertificateIssuer),
		newRscStoreWatchTest("enterprise", "DNSProvider", &enterprisev1.DNSProvider{}, enterprise.CreateDNSProvider, enterprise.UpdateDNSProvider, enterprise.DeleteDNSProvider),
		newRscStoreWatchTest("enterprise", "DirectoryProvider", &enterprisev1.DirectoryProvider{}, enterprise.CreateDirectoryProvider, enterprise.UpdateDirectoryProvider, enterprise.DeleteDirectoryProvider),
		newRscStoreWatchTest("enterprise", "DirectoryProviderUser", &enterprisev1.DirectoryProviderUser{}, enterprise.CreateDirectoryProviderUser, enterprise.UpdateDirectoryProviderUser, enterprise.DeleteDirectoryProviderUser),
		newRscStoreWatchTest("enterprise", "DirectoryProviderGroup", &enterprisev1.DirectoryProviderGroup{}, enterprise.CreateDirectoryProviderGroup, enterprise.UpdateDirectoryProviderGroup, enterprise.DeleteDirectoryProviderGroup),
		newRscStoreWatchTest("enterprise", "DeviceManager", &enterprisev1.DeviceManager{}, enterprise.CreateDeviceManager, enterprise.UpdateDeviceManager, enterprise.DeleteDeviceManager),
		newRscStoreWatchTest("access", "Catalog", &accessv1.Catalog{}, access.CreateCatalog, access.UpdateCatalog, access.DeleteCatalog),
		newRscStoreWatchTest("access", "Policy", &accessv1.Policy{}, access.CreatePolicy, access.UpdatePolicy, access.DeletePolicy),
		newRscStoreWatchTest("access", "Request", &accessv1.Request{}, access.CreateRequest, access.UpdateRequest, access.DeleteRequest),
		newRscStoreWatchTest("access", "Review", &accessv1.Review{}, access.CreateReview, access.UpdateReview, access.DeleteReview),
		newRscStoreWatchTest("access", "Secret", &accessv1.Secret{}, access.CreateSecret, access.UpdateSecret, access.DeleteSecret),
		newRscStoreWatchTest("access", "Integration", &accessv1.Integration{}, access.CreateIntegration, access.UpdateIntegration, access.DeleteIntegration),
		newRscStoreWatchTest("access", "IntegrationIdentity", &accessv1.IntegrationIdentity{}, access.CreateIntegrationIdentity, access.UpdateIntegrationIdentity, access.DeleteIntegrationIdentity),
		newRscStoreWatchTest("access", "IntegrationBinding", &accessv1.IntegrationBinding{}, access.CreateIntegrationBinding, access.UpdateIntegrationBinding, access.DeleteIntegrationBinding),
		newRscStoreWatchTest("cordium", "Workspace", &cordiumv1.Workspace{}, cordium.CreateWorkspace, cordium.UpdateWorkspace, cordium.DeleteWorkspace),
		newRscStoreWatchTest("cordium", "Template", &cordiumv1.Template{}, cordium.CreateTemplate, cordium.UpdateTemplate, cordium.DeleteTemplate),
		newRscStoreWatchTest("cordium", "Space", &cordiumv1.Space{}, cordium.CreateSpace, cordium.UpdateSpace, cordium.DeleteSpace),
		newRscStoreWatchTest("cordium", "GitProvider", &cordiumv1.GitProvider{}, cordium.CreateGitProvider, cordium.UpdateGitProvider, cordium.DeleteGitProvider),
		newRscStoreWatchTest("cordium", "Secret", &cordiumv1.Secret{}, cordium.CreateSecret, nil, cordium.DeleteSecret),
		newRscStoreWatchTest("cordium", "UserSecret", &cordiumv1.UserSecret{}, cordium.CreateUserSecret, cordium.UpdateUserSecret, cordium.DeleteUserSecret),
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			item, err := tt.create(ctx)
			require.Nil(t, err, "%+v", err)
			uid := item.GetMetadata().Uid
			waitVersion := func(version string) {
				t.Helper()
				require.Eventually(t, func() bool {
					var got string
					err := env.srv.db.QueryRowContext(ctx, `SELECT resource_version FROM resources WHERE uid = ?`, uid).Scan(&got)
					return err == nil && got == version
				}, 5*time.Second, 10*time.Millisecond)
			}
			waitVersion(item.GetMetadata().ResourceVersion)
			updated := item
			if tt.update != nil {
				updated, err = tt.update(ctx, item)
				require.Nil(t, err, "%+v", err)
				require.NotEqual(t, item.GetMetadata().ResourceVersion, updated.GetMetadata().ResourceVersion)
				waitVersion(updated.GetMetadata().ResourceVersion)
				var search string
				require.Nil(t, env.srv.db.QueryRowContext(ctx, `SELECT rsc_str FROM resources WHERE uid = ?`, uid).Scan(&search))
				assert.Contains(t, search, "updated-watch-projection")
			}
			require.Nil(t, tt.remove(ctx, uid))
			require.Eventually(t, func() bool {
				var count int
				err := env.srv.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM resources WHERE uid = ?`, uid).Scan(&count)
				return err == nil && count == 0
			}, 5*time.Second, 10*time.Millisecond)
			assert.True(t, env.srv.isRemovedResource(updated))
			require.Nil(t, env.srv.insertResource(ctx, item))
			var count int
			require.Nil(t, env.srv.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM resources WHERE uid = ?`, uid).Scan(&count))
			assert.Zero(t, count)
		})
	}
}

func TestServerStartupFailures(t *testing.T) {
	t.Run("AddressInUse", func(t *testing.T) {
		env := newRscStoreTestEnv(t)
		require.NotNil(t, env)
		addr := ":8080"
		if ovutils.IsMockMode() {
			addr = "localhost:40001"
		} else if ldflags.IsTest() {
			addr = tstAddr
		}
		listener, err := net.Listen("tcp", addr)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, listener.Close()) })
		assert.Error(t, env.srv.initGRPC(env.ctx))
		assert.Error(t, env.srv.Run(env.ctx))
	})

	t.Run("ClosedDatabase", func(t *testing.T) {
		env := newRscStoreTestEnv(t)
		require.NotNil(t, env)
		require.NoError(t, env.srv.db.Close())
		assert.Error(t, env.srv.Run(env.ctx))
	})
}
