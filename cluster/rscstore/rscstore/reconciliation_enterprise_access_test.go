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
	"testing"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestReconcileEnterpriseResourceKind(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	sec, err := fakeC.OcteliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		ApiVersion: "enterprise/v1",
		Kind:       "Secret",
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{
			Data: &enterprisev1.Secret_Spec_Data{
				Type: &enterprisev1.Secret_Spec_Data_Value{Value: "shh"},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.False(t, localExists(ctx, t, srv, sec.Metadata.Uid))

	err = srv.reconcileEnterpriseResourceKind(ctx, resourceKind{
		api: "enterprise", version: "v1", kind: "Secret",
	})
	assert.Nil(t, err, "%+v", err)
	assert.True(t, localExists(ctx, t, srv, sec.Metadata.Uid))

	_, err = fakeC.OcteliumC.EnterpriseC().DeleteSecret(ctx, &rmetav1.DeleteOptions{Uid: sec.Metadata.Uid})
	assert.Nil(t, err, "%+v", err)

	err = srv.reconcileEnterpriseResourceKind(ctx, resourceKind{
		api: "enterprise", version: "v1", kind: "Secret",
	})
	assert.Nil(t, err, "%+v", err)
	assert.False(t, localExists(ctx, t, srv, sec.Metadata.Uid))
}

func TestReconcileAccessResourceKind(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	pol, err := fakeC.OcteliumC.AccessC().CreatePolicy(ctx, &accessv1.Policy{
		ApiVersion: "access/v1",
		Kind:       "Policy",
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{},
	})
	assert.Nil(t, err, "%+v", err)
	assert.False(t, localExists(ctx, t, srv, pol.Metadata.Uid))

	err = srv.reconcileAccessResourceKind(ctx, resourceKind{
		api: "access", version: "v1", kind: "Policy",
	})
	assert.Nil(t, err, "%+v", err)
	assert.True(t, localExists(ctx, t, srv, pol.Metadata.Uid))

	_, err = fakeC.OcteliumC.AccessC().DeletePolicy(ctx, &rmetav1.DeleteOptions{Uid: pol.Metadata.Uid})
	assert.Nil(t, err, "%+v", err)

	err = srv.reconcileAccessResourceKind(ctx, resourceKind{
		api: "access", version: "v1", kind: "Policy",
	})
	assert.Nil(t, err, "%+v", err)
	assert.False(t, localExists(ctx, t, srv, pol.Metadata.Uid))
}

func TestReconcileAllResourcesCoversEnterpriseAndAccess(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	usr := createUser(ctx, t, fakeC.OcteliumC)

	sec, err := fakeC.OcteliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		ApiVersion: "enterprise/v1",
		Kind:       "Secret",
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{
			Data: &enterprisev1.Secret_Spec_Data{
				Type: &enterprisev1.Secret_Spec_Data_Value{Value: "shh"},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	pol, err := fakeC.OcteliumC.AccessC().CreatePolicy(ctx, &accessv1.Policy{
		ApiVersion: "access/v1",
		Kind:       "Policy",
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Policy_Spec{},
	})
	assert.Nil(t, err, "%+v", err)

	err = srv.reconcileAllResources(ctx)
	assert.Nil(t, err, "%+v", err)

	assert.True(t, localExists(ctx, t, srv, usr.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, sec.Metadata.Uid))
	assert.True(t, localExists(ctx, t, srv, pol.Metadata.Uid))
}

func TestValidateEnterpriseAndAccessReconcileKinds(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv := setupReconcileServer(ctx, t, fakeC.OcteliumC)

	assert.Nil(t, srv.validateEnterpriseReconcileKinds())
	assert.Nil(t, srv.validateAccessReconcileKinds())
}
