package rscstore

import (
	"context"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/pkg/apiutils/ucordiumv1"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func createCordiumSpace(ctx context.Context, t *testing.T, fakeC *otests.FakeClient) *cordiumv1.Space {
	spc, err := fakeC.OcteliumC.CordiumC().CreateSpace(ctx, &cordiumv1.Space{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindSpace,
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &cordiumv1.Space_Spec{},
		Status: &cordiumv1.Space_Status{
			Type: cordiumv1.Space_Status_ORGANIZATION,
		},
	})
	assert.Nil(t, err, "%+v", err)

	return spc
}

func setCordiumInstalled(ctx context.Context, t *testing.T, fakeC *otests.FakeClient) {
	rgn, err := fakeC.OcteliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: "default"})
	assert.Nil(t, err, "%+v", err)

	if rgn.Status.VersionInfoMap == nil {
		rgn.Status.VersionInfoMap = make(map[string]*corev1.Region_Status_VersionInfo)
	}

	rgn.Status.VersionInfoMap[versionInfoKeyCordium] = &corev1.Region_Status_VersionInfo{
		Package: "cordium",
		Version: "v0.1.0",
	}

	_, err = fakeC.OcteliumC.CoreC().UpdateRegion(ctx, rgn)
	assert.Nil(t, err, "%+v", err)
}

func TestReconcileCordiumResourceKind(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	spc := createCordiumSpace(ctx, t, fakeC)

	mem, err := fakeC.OcteliumC.CordiumC().CreateMembership(ctx, &cordiumv1.Membership{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindMembership,
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &cordiumv1.Membership_Spec{
			Role: cordiumv1.Membership_Spec_OWNER,
		},
		Status: &cordiumv1.Membership_Status{
			SpaceRef: &metav1.ObjectReference{
				Name: spc.Metadata.Name,
				Uid:  spc.Metadata.Uid,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	assert.False(t, localExists(ctx, t, srv, spc.Metadata.Uid))
	assert.False(t, localExists(ctx, t, srv, mem.Metadata.Uid))

	err = srv.reconcileCordiumResourceKind(ctx, resourceKind{
		api: ucordiumv1.API, version: ucordiumv1.Version, kind: ucordiumv1.KindSpace,
	})
	assert.Nil(t, err, "%+v", err)
	assert.True(t, localExists(ctx, t, srv, spc.Metadata.Uid))

	err = srv.reconcileCordiumResourceKind(ctx, resourceKind{
		api: ucordiumv1.API, version: ucordiumv1.Version, kind: ucordiumv1.KindMembership,
	})
	assert.Nil(t, err, "%+v", err)
	assert.True(t, localExists(ctx, t, srv, mem.Metadata.Uid))

	_, err = fakeC.OcteliumC.CordiumC().DeleteMembership(ctx, &rmetav1.DeleteOptions{Uid: mem.Metadata.Uid})
	assert.Nil(t, err, "%+v", err)

	err = srv.reconcileCordiumResourceKind(ctx, resourceKind{
		api: ucordiumv1.API, version: ucordiumv1.Version, kind: ucordiumv1.KindMembership,
	})
	assert.Nil(t, err, "%+v", err)
	assert.False(t, localExists(ctx, t, srv, mem.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, spc.Metadata.Uid))
}

func TestValidateCordiumReconcileKinds(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	assert.Nil(t, srv.validateCordiumReconcileKinds())
}

func TestReconcileCordiumResourcesRequiresCordium(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		cancel()
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	spc := createCordiumSpace(ctx, t, fakeC)

	{
		isInstalled, err := srv.isCordiumInstalled(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.False(t, isInstalled)
	}

	assert.Nil(t, srv.setCordiumResources(ctx))
	assert.False(t, srv.isWatchingCordium)

	err = srv.reconcileAllResources(ctx)
	assert.Nil(t, err, "%+v", err)
	assert.False(t, localExists(ctx, t, srv, spc.Metadata.Uid))
	assert.False(t, srv.isWatchingCordium)

	setCordiumInstalled(ctx, t, fakeC)

	{
		isInstalled, err := srv.isCordiumInstalled(ctx)
		assert.Nil(t, err, "%+v", err)
		assert.True(t, isInstalled)
	}

	err = srv.reconcileAllResources(ctx)
	assert.Nil(t, err, "%+v", err)
	assert.True(t, localExists(ctx, t, srv, spc.Metadata.Uid))
	assert.True(t, srv.isWatchingCordium)

	assert.Nil(t, srv.setCordiumResources(ctx))
	assert.True(t, srv.isWatchingCordium)
}

func TestWatchCordiumResources(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		cancel()
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	setCordiumInstalled(ctx, t, fakeC)

	err = srv.setCordiumResources(ctx)
	assert.Nil(t, err, "%+v", err)
	assert.True(t, srv.isWatchingCordium)

	time.Sleep(2 * time.Second)

	ws, err := fakeC.OcteliumC.CordiumC().CreateWorkspace(ctx, &cordiumv1.Workspace{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindWorkspace,
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &cordiumv1.Workspace_Spec{},
		Status: &cordiumv1.Workspace_Status{
			State: cordiumv1.Workspace_Status_STOPPED,
		},
	})
	assert.Nil(t, err, "%+v", err)

	assert.Eventually(t, func() bool {
		return localExists(ctx, t, srv, ws.Metadata.Uid)
	}, 10*time.Second, 100*time.Millisecond)

	ws.Status.State = cordiumv1.Workspace_Status_RUNNING
	ws, err = fakeC.OcteliumC.CordiumC().UpdateWorkspace(ctx, ws)
	assert.Nil(t, err, "%+v", err)

	assert.Eventually(t, func() bool {
		return localResourceVersion(ctx, t, srv, ws.Metadata.Uid) == ws.Metadata.ResourceVersion
	}, 10*time.Second, 100*time.Millisecond)

	_, err = fakeC.OcteliumC.CordiumC().DeleteWorkspace(ctx, &rmetav1.DeleteOptions{Uid: ws.Metadata.Uid})
	assert.Nil(t, err, "%+v", err)

	assert.Eventually(t, func() bool {
		return !localExists(ctx, t, srv, ws.Metadata.Uid)
	}, 10*time.Second, 100*time.Millisecond)
}
