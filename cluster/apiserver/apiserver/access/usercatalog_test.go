// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/user"
	"github.com/octelium/octelium/cluster/common/tests/tstuser"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUserCatalog(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	adminSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  tst.C.OcteliumC,
		IsEmbedded: true,
	})
	usrSrv := user.NewServer(tst.C.OcteliumC)

	mainSrv := NewServerMain(tst.C.OcteliumC)
	srv, err := NewServerUser(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)

	usr, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

	svc, err := adminSrv.CreateService(ctx, &corev1.Service{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.Service_Spec{
			Mode: corev1.Service_Spec_HTTP,
			Config: &corev1.Service_Spec_Config{
				Upstream: &corev1.Service_Spec_Config_Upstream{
					Type: &corev1.Service_Spec_Config_Upstream_Url{
						Url: "https://example.com",
					},
				},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	cat, err := mainSrv.CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Namespaces: []string{"default"},
				},
			},
			AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{testFullLevel("full")},
		},
	})
	assert.Nil(t, err, "%+v", err)

	hiddenCat, err := mainSrv.CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Catalog_Spec{
			ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
				Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
					Namespaces: []string{"default"},
				},
			},
			AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{testFullLevel("full")},
			Audience: &accessv1.Catalog_Spec_Audience{
				Users: []string{"someone-else"},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	catList, err := srv.ListCatalog(usr.Ctx(), &accessv1.ListUserCatalogOptions{})
	assert.Nil(t, err, "%+v", err)
	assert.True(t, len(catList.Items) > 0)

	foundCatalog := false
	for _, item := range catList.Items {
		assert.NotEqual(t, hiddenCat.Metadata.Uid, item.Metadata.Uid)
		if item.Metadata.Uid == cat.Metadata.Uid {
			foundCatalog = true
			assert.Equal(t, "full", item.AccessLevels[0].Name)
			assert.Equal(t, accessv1.UserCatalog_AccessLevel_FULL, item.AccessLevels[0].Type)
			assert.NotNil(t, item.Viewer)
		}
	}
	assert.True(t, foundCatalog)

	_, err = srv.GetCatalog(usr.Ctx(), &metav1.GetOptions{Uid: hiddenCat.Metadata.Uid})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

	catG, err := srv.GetCatalog(usr.Ctx(), &metav1.GetOptions{Uid: cat.Metadata.Uid})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, cat.Metadata.Name, catG.Metadata.Name)

	_, err = srv.ListCatalogService(usr.Ctx(), &accessv1.ListUserCatalogServiceOptions{})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

	svcList, err := srv.ListCatalogService(usr.Ctx(), &accessv1.ListUserCatalogServiceOptions{
		CatalogRef: umetav1.GetObjectReference(cat),
	})
	assert.Nil(t, err, "%+v", err)

	foundService := false
	for _, item := range svcList.Items {
		if item.Metadata.Uid == svc.Metadata.Uid {
			foundService = true
		}
	}
	assert.True(t, foundService)
}

func TestUserCatalogServiceSelection(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	ctx := context.WithValue(t.Context(), "octelium-user-ctx", &userctx.UserCtx{
		User: &corev1.User{
			Metadata: &metav1.Metadata{Name: "requester", Uid: vutils.UUIDv4()},
			Spec:     &corev1.User_Spec{},
		},
	})
	ns, err := tst.C.OcteliumC.CoreC().CreateNamespace(ctx, &corev1.Namespace{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &corev1.Namespace_Spec{},
		Status:   &corev1.Namespace_Status{},
	})
	require.Nil(t, err)
	var services []*corev1.Service
	for _, name := range []string{"z-visible", "a-visible", "hidden"} {
		item, err := tst.C.OcteliumC.CoreC().CreateService(ctx, &corev1.Service{
			Metadata: &metav1.Metadata{Name: name + "." + ns.Metadata.Name, IsUserHidden: name == "hidden"},
			Spec:     &corev1.Service_Spec{Mode: corev1.Service_Spec_HTTP, Port: 8080},
			Status:   &corev1.Service_Status{NamespaceRef: umetav1.GetObjectReference(ns), Port: 8080},
		})
		require.Nil(t, err)
		services = append(services, item)
	}
	catalog, err := tst.C.OcteliumC.AccessC().CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &accessv1.Catalog_Spec{ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
			Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
				Services:   []string{"", services[0].Metadata.Name, services[0].Metadata.Name, services[2].Metadata.Name, "missing." + ns.Metadata.Name},
				Namespaces: []string{"", "missing-namespace", ns.Metadata.Name, ns.Metadata.Name},
			},
		},
			AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{testFullLevel("full")},
		},
		Status: &accessv1.Catalog_Status{},
	})
	require.Nil(t, err)
	srv := &ServerUser{octeliumC: tst.C.OcteliumC}
	res, err := srv.ListCatalogService(ctx, &accessv1.ListUserCatalogServiceOptions{
		CatalogRef: umetav1.GetObjectReference(catalog),
	})
	require.Nil(t, err)
	require.Len(t, res.Items, 2)
	assert.Equal(t, uint32(2), res.ListResponseMeta.TotalCount)
	assert.Equal(t, services[1].Metadata.Uid, res.Items[0].Metadata.Uid)
	assert.Equal(t, services[0].Metadata.Uid, res.Items[1].Metadata.Uid)
	assert.Equal(t, ns.Metadata.Name, res.Items[0].Status.Namespace)
	assert.Equal(t, uint32(8080), res.Items[0].Spec.Port)
	assert.Equal(t, "user/v1", res.ApiVersion)
	assert.Equal(t, "ServiceList", res.Kind)
}

func TestUserCatalogLookupFailure(t *testing.T) {
	srv := &ServerUser{octeliumC: &policyValidationClient{accessC: &policyValidationAccessClient{err: status.Error(codes.Unavailable, "store unavailable")}}}
	ctx := context.WithValue(t.Context(), "octelium-user-ctx", &userctx.UserCtx{
		User: &corev1.User{
			Metadata: &metav1.Metadata{Name: "requester", Uid: vutils.UUIDv4()},
			Spec:     &corev1.User_Spec{},
		},
	})
	_, err := srv.ListCatalogService(ctx, &accessv1.ListUserCatalogServiceOptions{
		CatalogRef: &metav1.ObjectReference{Name: "catalog"},
	})
	assert.True(t, grpcerr.IsInternal(err), "%+v", err)
}

func testFullLevel(name string) *accessv1.Catalog_Spec_AccessLevel {
	return &accessv1.Catalog_Spec_AccessLevel{
		Name: name,
		Type: &accessv1.Catalog_Spec_AccessLevel_Full_{
			Full: &accessv1.Catalog_Spec_AccessLevel_Full{},
		},
	}
}

func TestUserCatalogServiceMetadata(t *testing.T) {
	svc := &corev1.Service{
		Metadata: &metav1.Metadata{Name: "service.default", Labels: map[string]string{"team": "platform"}},
		Spec:     &corev1.Service_Spec{Mode: corev1.Service_Spec_HTTP},
		Status:   &corev1.Service_Status{},
	}
	res := toUserCatalogService(svc)
	assert.Empty(t, res.Status.Namespace)
	res.Metadata.Labels["team"] = "changed"
	assert.Equal(t, "platform", svc.Metadata.Labels["team"])
}
