// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package syncprovider

import (
	"context"
	"testing"

	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/enterprise"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

type testSource struct {
	users  []*User
	groups []*Group
}

func (s *testSource) ListUsers(ctx context.Context) ([]*User, error) {
	return s.users, nil
}

func (s *testSource) ListGroups(ctx context.Context) ([]*Group, error) {
	return s.groups, nil
}

func TestSync(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	apiSrv := enterprise.NewServer(fakeC.OcteliumC)

	dp, err := apiSrv.CreateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{
				Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	dp, err = fakeC.OcteliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
		Uid: dp.Metadata.Uid,
	})
	assert.Nil(t, err)

	reconciler := NewReconciler(fakeC.OcteliumC, dp)

	uid1 := "87453e5a-1f78-4e3e-9932-345cee971fe8"
	uid2 := "1b2c3d4e-5f60-4a1b-8c2d-3e4f5a6b7c8d"
	gid1 := "9f8e7d6c-5b4a-4c3d-2e1f-0a9b8c7d6e5f"

	src := &testSource{
		users: []*User{
			{
				ExternalID:  uid1,
				Email:       "usr1@example.com",
				DisplayName: "User One",
				FirstName:   "User",
				LastName:    "One",
			},
			{
				ExternalID:  uid2,
				DisplayName: "Service Account",
			},
		},
		groups: []*Group{
			{
				ExternalID:        gid1,
				DisplayName:       "Engineering",
				MemberExternalIDs: []string{uid1},
			},
		},
	}

	err = reconciler.Sync(ctx, src)
	assert.Nil(t, err, "%+v", err)

	usr1Name := reconciler.genUserName(src.users[0])
	usr2Name := reconciler.genUserName(src.users[1])
	grp1Name := reconciler.genGroupName(src.groups[0])

	assert.True(t, len(usr1Name) <= maxNameLen)
	assert.True(t, len(usr2Name) <= maxNameLen)
	assert.True(t, len(grp1Name) <= maxNameLen)

	{
		usr, err := fakeC.OcteliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Name: usr1Name})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "usr1@example.com", usr.Spec.Email)
		assert.Equal(t, "User One", usr.Metadata.DisplayName)
		assert.Equal(t, "User", usr.Spec.Info.FirstName)
		assert.Equal(t, "One", usr.Spec.Info.LastName)
		assert.False(t, usr.Spec.IsDisabled)
		assert.Contains(t, usr.Spec.Groups, grp1Name)
	}

	{
		usr, err := fakeC.OcteliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Name: usr2Name})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "", usr.Spec.Email)
		assert.Equal(t, "Service Account", usr.Metadata.DisplayName)
	}

	{
		grp, err := fakeC.OcteliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{Name: grp1Name})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "Engineering", grp.Metadata.DisplayName)
	}

	{
		err = reconciler.Sync(ctx, src)
		assert.Nil(t, err, "%+v", err)

		dpUsrList, err := fakeC.OcteliumC.EnterpriseC().ListDirectoryProviderUser(ctx, &rmetav1.ListOptions{})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(dpUsrList.Items))

		dpGrpList, err := fakeC.OcteliumC.EnterpriseC().ListDirectoryProviderGroup(ctx, &rmetav1.ListOptions{})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(dpGrpList.Items))

		_, err = fakeC.OcteliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Name: usr1Name})
		assert.Nil(t, err)
	}

	{
		src2 := &testSource{
			users:  []*User{src.users[0]},
			groups: src.groups,
		}

		err = reconciler.Sync(ctx, src2)
		assert.Nil(t, err, "%+v", err)

		_, err = fakeC.OcteliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Name: usr2Name})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err))

		usr, err := fakeC.OcteliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{Name: usr1Name})
		assert.Nil(t, err, "%+v", err)
		assert.Contains(t, usr.Spec.Groups, grp1Name)
	}
}
