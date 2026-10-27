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
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/apiserver/apiserver/enterprise"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	"github.com/octelium/octelium/apis/rsc/rlockv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	cocteliumc "github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
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

type failingDirectorySource struct {
	usersErr, groupsErr     error
	usersCalls, groupsCalls int
}

func (s *failingDirectorySource) ListUsers(context.Context) ([]*User, error) {
	s.usersCalls++
	return nil, s.usersErr
}
func (s *failingDirectorySource) ListGroups(context.Context) ([]*Group, error) {
	s.groupsCalls++
	return nil, s.groupsErr
}

func TestSyncStopsBeforeWritesOnSourceFailure(t *testing.T) {
	failure := errors.New("provider unavailable")
	for _, source := range []*failingDirectorySource{{usersErr: failure}, {groupsErr: failure}} {
		err := (&Reconciler{}).Sync(t.Context(), source)
		assert.ErrorIs(t, err, failure)
		assert.Equal(t, 1, source.usersCalls)
		if source.usersErr != nil {
			assert.Zero(t, source.groupsCalls)
		} else {
			assert.Equal(t, 1, source.groupsCalls)
		}
	}
}

func TestDirectoryGeneratedNameBoundaries(t *testing.T) {
	r := &Reconciler{
		dp: &enterprisev1.DirectoryProvider{Status: &enterprisev1.DirectoryProvider_Status{Id: "provider"}},
	}
	for _, external := range []string{"", "short", "12345678-0000-4000-8000-000000000000", "UPPER Case", "你好", strings.Repeat("id", 100)} {
		for _, readable := range []string{"", "person@example.com", "á ü 你好", strings.Repeat("a", 200), "---"} {
			name := r.genReadableName(readable, external)
			assert.NoError(t, apivalidation.ValidateName(name, 0, 0), name)
			assert.LessOrEqual(t, len(name), maxNameLen)
			assert.Equal(t, name, r.genReadableName(readable, external))
		}
		assert.Len(t, externalIDTag(external), userTagLen)
	}
}

type directorySyncTestClient struct {
	octeliumc.ClientInterface
	conn *grpc.ClientConn
}

func (c *directorySyncTestClient) CoreC() rcorev1.ResourceServiceClient {
	return rcorev1.NewResourceServiceClient(c.conn)
}

func (c *directorySyncTestClient) EnterpriseC() renterprisev1.ResourceServiceClient {
	return renterprisev1.NewResourceServiceClient(c.conn)
}

func (c *directorySyncTestClient) LockC() rlockv1.MainServiceClient {
	return rlockv1.NewMainServiceClient(c.conn)
}

func newDirectorySyncTestReconciler(t *testing.T, fail func(string, any) error) *Reconciler {
	t.Helper()
	tst, err := otests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	dp, err := enterprise.NewServer(tst.C.OcteliumC).CreateDirectoryProvider(t.Context(), &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{}},
		},
	})
	require.NoError(t, err)
	opts, err := cocteliumc.DefaultDialOpts(t.Context())
	require.NoError(t, err)
	opts = append(opts, grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if fail != nil {
			if err := fail(method, req); err != nil {
				return err
			}
		}
		return invoke(ctx, method, req, reply, cc, opts...)
	}))
	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSERVER_PORT")), opts...)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return NewReconciler(&directorySyncTestClient{ClientInterface: tst.C.OcteliumC, conn: conn}, dp)
}

func TestSyncUpsertFailuresPreserveExistingResources(t *testing.T) {
	failure := errors.New("group update unavailable")
	fail := false
	r := newDirectorySyncTestReconciler(t, func(method string, req any) error {
		if fail && method == rcorev1.ResourceService_UpdateGroup_FullMethodName {
			return failure
		}
		return nil
	})
	usr := &User{ExternalID: "active-user", Email: "active@example.com"}
	staleUsr := &User{ExternalID: "stale-user", Email: "stale@example.com"}
	grp := &Group{ExternalID: "active-group", DisplayName: "Active", MemberExternalIDs: []string{usr.ExternalID}}
	staleGrp := &Group{ExternalID: "stale-group", DisplayName: "Stale"}
	require.NoError(t, r.Sync(t.Context(), &testSource{users: []*User{usr, staleUsr}, groups: []*Group{grp, staleGrp}}))

	valid := &User{ExternalID: "new-user", Email: "new@example.com"}
	invalid := &User{ExternalID: "invalid-user", Email: "invalid-email"}
	newGrp := &Group{ExternalID: "new-group", DisplayName: "New"}
	grp.MemberExternalIDs = nil
	src := &testSource{users: []*User{usr, invalid, valid}, groups: []*Group{grp, newGrp}}
	fail = true
	err := r.Sync(t.Context(), src)
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid-email")
	assert.ErrorContains(t, err, "Could not upsert Group "+grp.ExternalID)
	for _, u := range []*User{usr, staleUsr, valid} {
		coreUsr, err := r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(u)})
		require.NoError(t, err)
		if u == usr {
			assert.Contains(t, coreUsr.Spec.Groups, r.genGroupName(grp))
		}
	}
	for _, g := range []*Group{grp, staleGrp, newGrp} {
		_, err := r.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(g)})
		require.NoError(t, err)
	}

	fail = false
	src.users = []*User{usr, valid}
	require.NoError(t, r.Sync(t.Context(), src))
	_, err = r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(staleUsr)})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	_, err = r.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(staleGrp)})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	coreUsr, err := r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
	require.NoError(t, err)
	assert.Empty(t, coreUsr.Spec.Groups)
}

func TestSyncGroupListingFailurePreservesUpsertErrors(t *testing.T) {
	userErr := errors.New("user lookup unavailable")
	groupErr := errors.New("group listing unavailable")
	r := newDirectorySyncTestReconciler(t, func(method string, req any) error {
		switch method {
		case renterprisev1.ResourceService_GetDirectoryProviderUser_FullMethodName:
			return userErr
		case renterprisev1.ResourceService_ListDirectoryProviderGroup_FullMethodName:
			return groupErr
		}
		return nil
	})
	err := r.Sync(t.Context(), &testSource{users: []*User{{ExternalID: "user", Email: "user@example.com"}}})
	assert.ErrorIs(t, err, userErr)
	assert.ErrorIs(t, err, groupErr)
}

func TestSyncMembershipFailureSkipsPruning(t *testing.T) {
	failure := errors.New("membership update unavailable")
	fail := false
	r := newDirectorySyncTestReconciler(t, func(method string, req any) error {
		if fail && method == rcorev1.ResourceService_UpdateUser_FullMethodName && len(req.(*corev1.User).Spec.Groups) == 0 {
			return failure
		}
		return nil
	})
	usr := &User{ExternalID: "active-user", Email: "active@example.com"}
	staleUsr := &User{ExternalID: "stale-user", Email: "stale@example.com"}
	grp := &Group{ExternalID: "active-group", DisplayName: "Active", MemberExternalIDs: []string{usr.ExternalID}}
	staleGrp := &Group{ExternalID: "stale-group", DisplayName: "Stale"}
	require.NoError(t, r.Sync(t.Context(), &testSource{users: []*User{usr, staleUsr}, groups: []*Group{grp, staleGrp}}))
	grp.MemberExternalIDs = nil
	src := &testSource{users: []*User{usr}, groups: []*Group{grp}}
	fail = true
	err := r.Sync(t.Context(), src)
	assert.ErrorContains(t, err, "Could not reconcile User groups for "+r.genUserName(usr))
	_, err = r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(staleUsr)})
	require.NoError(t, err)
	_, err = r.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(staleGrp)})
	require.NoError(t, err)
	coreUsr, err := r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
	require.NoError(t, err)
	assert.Contains(t, coreUsr.Spec.Groups, r.genGroupName(grp))

	fail = false
	require.NoError(t, r.Sync(t.Context(), src))
	_, err = r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(staleUsr)})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	_, err = r.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(staleGrp)})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
}

func TestSyncPruneFailuresAreReturnedAndRetryable(t *testing.T) {
	for _, tc := range []struct {
		name, method string
		user         bool
		listCall     int
	}{
		{"ListUsers", renterprisev1.ResourceService_ListDirectoryProviderUser_FullMethodName, true, 1},
		{"ListGroups", renterprisev1.ResourceService_ListDirectoryProviderGroup_FullMethodName, false, 2},
		{"CheckUserReferences", renterprisev1.ResourceService_ListDirectoryProviderUser_FullMethodName, true, 2},
		{"CheckGroupReferences", renterprisev1.ResourceService_ListDirectoryProviderGroup_FullMethodName, false, 3},
		{"DeleteUser", rcorev1.ResourceService_DeleteUser_FullMethodName, true, 0},
		{"DeleteGroup", rcorev1.ResourceService_DeleteGroup_FullMethodName, false, 0},
		{"DeleteUserLink", renterprisev1.ResourceService_DeleteDirectoryProviderUser_FullMethodName, true, 0},
		{"DeleteGroupLink", renterprisev1.ResourceService_DeleteDirectoryProviderGroup_FullMethodName, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failure := errors.New(tc.name + " unavailable")
			fail := false
			calls := 0
			r := newDirectorySyncTestReconciler(t, func(method string, req any) error {
				if fail && method == tc.method {
					calls++
					if tc.listCall == 0 || calls == tc.listCall {
						return failure
					}
				}
				return nil
			})
			usr := &User{ExternalID: "user", Email: "user@example.com"}
			grp := &Group{ExternalID: "group", DisplayName: "Group"}
			require.NoError(t, r.Sync(t.Context(), &testSource{users: []*User{usr}, groups: []*Group{grp}}))
			fail = true
			err := r.Sync(t.Context(), &testSource{})
			assert.ErrorIs(t, err, failure)
			if tc.user {
				_, err = r.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
			} else {
				_, err = r.octeliumC.EnterpriseC().GetDirectoryProviderGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(grp)})
			}
			require.NoError(t, err)

			fail = false
			require.NoError(t, r.Sync(t.Context(), &testSource{}))
			_, err = r.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
			assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
			_, err = r.octeliumC.EnterpriseC().GetDirectoryProviderGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(grp)})
			assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
			_, err = r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
			assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
			_, err = r.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(grp)})
			assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
		})
	}
}

func TestSyncAggregatesPruneFailures(t *testing.T) {
	userErr := errors.New("user deletion unavailable")
	groupErr := errors.New("group deletion unavailable")
	fail := false
	r := newDirectorySyncTestReconciler(t, func(method string, req any) error {
		if fail {
			switch method {
			case rcorev1.ResourceService_DeleteUser_FullMethodName:
				return userErr
			case rcorev1.ResourceService_DeleteGroup_FullMethodName:
				return groupErr
			}
		}
		return nil
	})
	require.NoError(t, r.Sync(t.Context(), &testSource{
		users:  []*User{{ExternalID: "user", Email: "user@example.com"}},
		groups: []*Group{{ExternalID: "group", DisplayName: "Group"}},
	}))
	fail = true
	err := r.Sync(t.Context(), &testSource{})
	assert.ErrorIs(t, err, userErr)
	assert.ErrorIs(t, err, groupErr)
}

func TestSyncPruningPreservesSharedResources(t *testing.T) {
	r := newDirectorySyncTestReconciler(t, nil)
	usr := &User{ExternalID: "user", Email: "user@example.com"}
	grp := &Group{ExternalID: "group", DisplayName: "Group"}
	require.NoError(t, r.Sync(t.Context(), &testSource{users: []*User{usr}, groups: []*Group{grp}}))
	coreUsr, err := r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
	require.NoError(t, err)
	coreGrp, err := r.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(grp)})
	require.NoError(t, err)
	otherDP, err := enterprise.NewServer(r.octeliumC).CreateDirectoryProvider(t.Context(), &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{}},
		},
	})
	require.NoError(t, err)
	otherUser, err := r.octeliumC.EnterpriseC().CreateDirectoryProviderUser(t.Context(), &enterprisev1.DirectoryProviderUser{
		Metadata: &metav1.Metadata{Name: "shared-user", Annotations: map[string]string{directoryOwnedAnnotation: "true"}},
		Spec:     &enterprisev1.DirectoryProviderUser_Spec{},
		Status: &enterprisev1.DirectoryProviderUser_Status{
			DirectoryProviderRef: &metav1.ObjectReference{Uid: otherDP.Metadata.Uid},
			UserRef:              &metav1.ObjectReference{Uid: coreUsr.Metadata.Uid},
		},
	})
	require.NoError(t, err)
	otherGroup, err := r.octeliumC.EnterpriseC().CreateDirectoryProviderGroup(t.Context(), &enterprisev1.DirectoryProviderGroup{
		Metadata: &metav1.Metadata{Name: "shared-group"},
		Spec:     &enterprisev1.DirectoryProviderGroup_Spec{},
		Status: &enterprisev1.DirectoryProviderGroup_Status{
			DirectoryProviderRef: &metav1.ObjectReference{Uid: otherDP.Metadata.Uid},
			GroupRef:             &metav1.ObjectReference{Uid: coreGrp.Metadata.Uid},
		},
	})
	require.NoError(t, err)
	require.NoError(t, r.Sync(t.Context(), &testSource{}))
	_, err = r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: coreUsr.Metadata.Uid})
	require.NoError(t, err)
	_, err = r.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Uid: coreGrp.Metadata.Uid})
	require.NoError(t, err)
	_, err = r.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Uid: otherUser.Metadata.Uid})
	require.NoError(t, err)
	_, err = r.octeliumC.EnterpriseC().GetDirectoryProviderGroup(t.Context(), &rmetav1.GetOptions{Uid: otherGroup.Metadata.Uid})
	require.NoError(t, err)
	_, err = r.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	_, err = r.octeliumC.EnterpriseC().GetDirectoryProviderGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(grp)})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)

	require.NoError(t, NewReconciler(r.octeliumC, otherDP).Sync(t.Context(), &testSource{}))
	_, err = r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: coreUsr.Metadata.Uid})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	_, err = r.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Uid: coreGrp.Metadata.Uid})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
}

func TestSyncLookupFailuresAbortReconciliation(t *testing.T) {
	for _, tc := range []struct {
		name, method string
	}{
		{"ManagedGroup", rcorev1.ResourceService_GetGroup_FullMethodName},
		{"AdoptedUserOwnership", renterprisev1.ResourceService_ListDirectoryProviderUser_FullMethodName},
		{"PrunedUserOwnership", rcorev1.ResourceService_GetUser_FullMethodName},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failure := errors.New(tc.name + " unavailable")
			fail := false
			r := newDirectorySyncTestReconciler(t, func(method string, req any) error {
				if fail && method == tc.method {
					return failure
				}
				return nil
			})
			usr := &User{ExternalID: "user", Email: "user@example.com"}
			grp := &Group{ExternalID: "group", DisplayName: "Group"}
			src := &testSource{}
			if tc.name == "AdoptedUserOwnership" {
				_, err := r.octeliumC.CoreC().CreateUser(t.Context(), &corev1.User{
					Metadata: &metav1.Metadata{Name: "local-user"},
					Spec:     &corev1.User_Spec{Type: corev1.User_Spec_HUMAN, Email: usr.Email},
					Status:   &corev1.User_Status{},
				})
				require.NoError(t, err)
				src.users = []*User{usr}
			} else {
				require.NoError(t, r.Sync(t.Context(), &testSource{users: []*User{usr}, groups: []*Group{grp}}))
				if tc.name == "ManagedGroup" {
					link, err := r.octeliumC.EnterpriseC().GetDirectoryProviderGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(grp)})
					require.NoError(t, err)
					link.Status.GroupRef.Name = ""
					_, err = r.octeliumC.EnterpriseC().UpdateDirectoryProviderGroup(t.Context(), link)
					require.NoError(t, err)
				} else {
					link, err := r.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
					require.NoError(t, err)
					link.Metadata.Annotations = nil
					_, err = r.octeliumC.EnterpriseC().UpdateDirectoryProviderUser(t.Context(), link)
					require.NoError(t, err)
				}
			}
			fail = true
			assert.ErrorIs(t, r.Sync(t.Context(), src), failure)
			fail = false
			switch tc.name {
			case "ManagedGroup":
				_, err := r.octeliumC.EnterpriseC().GetDirectoryProviderGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(grp)})
				require.NoError(t, err)
			case "PrunedUserOwnership":
				_, err := r.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
				require.NoError(t, err)
			case "AdoptedUserOwnership":
				_, err := r.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
				assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
			}
			require.NoError(t, r.Sync(t.Context(), src))
			require.NoError(t, r.Sync(t.Context(), &testSource{}))
			if tc.name == "AdoptedUserOwnership" {
				_, err := r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: "local-user"})
				require.NoError(t, err)
			} else {
				_, err := r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
				assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
			}
		})
	}
}

func TestSyncCleanupLockFailuresPreserveResources(t *testing.T) {
	failure := errors.New("cleanup lock unavailable")
	fail := false
	r := newDirectorySyncTestReconciler(t, func(method string, req any) error {
		if fail && method == rlockv1.MainService_Lock_FullMethodName {
			return failure
		}
		return nil
	})
	usr := &User{ExternalID: "user", Email: "user@example.com"}
	grp := &Group{ExternalID: "group", DisplayName: "Group"}
	require.NoError(t, r.Sync(t.Context(), &testSource{users: []*User{usr}, groups: []*Group{grp}}))
	fail = true
	assert.ErrorIs(t, r.Sync(t.Context(), &testSource{}), failure)
	_, err := r.octeliumC.EnterpriseC().GetDirectoryProviderUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
	require.NoError(t, err)
	_, err = r.octeliumC.EnterpriseC().GetDirectoryProviderGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(grp)})
	require.NoError(t, err)
	_, err = r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
	require.NoError(t, err)
	_, err = r.octeliumC.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Name: r.genGroupName(grp)})
	require.NoError(t, err)
	fail = false
	require.NoError(t, r.Sync(t.Context(), &testSource{}))
}

func TestDirectoryDeletionLockCancellationAndContention(t *testing.T) {
	r := newDirectorySyncTestReconciler(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	lockCtx, unlock, err := r.lockResourceDeletion(ctx, "user/test")
	require.NoError(t, err)

	waitCtx, cancelWait := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancelWait()
	_, _, err = r.lockResourceDeletion(waitCtx, "user/test")
	require.Error(t, err)
	cancel()
	assert.ErrorIs(t, lockCtx.Err(), context.Canceled)
	require.NoError(t, unlock())

	_, unlock, err = r.lockResourceDeletion(t.Context(), "user/test")
	require.NoError(t, err)
	require.NoError(t, unlock())
}

func TestDirectoryDeletionUnlockFailure(t *testing.T) {
	failure := errors.New("unlock response unavailable")
	var r *Reconciler
	r = newDirectorySyncTestReconciler(t, func(method string, req any) error {
		if method == rlockv1.MainService_Unlock_FullMethodName {
			client := r.octeliumC.(*directorySyncTestClient)
			_, err := client.ClientInterface.LockC().Unlock(t.Context(), req.(*rlockv1.UnlockRequest))
			require.NoError(t, err)
			return failure
		}
		return nil
	})
	usr := &User{ExternalID: "user", Email: "user@example.com"}
	require.NoError(t, r.Sync(t.Context(), &testSource{users: []*User{usr}}))
	assert.ErrorIs(t, r.Sync(t.Context(), &testSource{}), failure)
	_, err := r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
}

func TestSyncConcurrentPruningOfSharedUser(t *testing.T) {
	r := newDirectorySyncTestReconciler(t, nil)
	otherDP, err := enterprise.NewServer(r.octeliumC).CreateDirectoryProvider(t.Context(), &enterprisev1.DirectoryProvider{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{}},
		},
	})
	require.NoError(t, err)
	other := NewReconciler(r.octeliumC, otherDP)
	for i := range 5 {
		usr := &User{ExternalID: fmt.Sprintf("user-%d", i), Email: fmt.Sprintf("user%d@example.com", i)}
		src := &testSource{users: []*User{usr}}
		require.NoError(t, r.Sync(t.Context(), src))
		require.NoError(t, other.Sync(t.Context(), src))
		coreUsr, err := r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Name: r.genUserName(usr)})
		require.NoError(t, err)
		start := make(chan struct{})
		results := make(chan error, 2)
		for _, reconciler := range []*Reconciler{r, other} {
			go func() {
				<-start
				results <- reconciler.Sync(t.Context(), &testSource{})
			}()
		}
		close(start)
		for range 2 {
			assert.NoError(t, <-results)
		}
		_, err = r.octeliumC.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: coreUsr.Metadata.Uid})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
		links, err := r.octeliumC.EnterpriseC().ListDirectoryProviderUser(t.Context(), &rmetav1.ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, links.Items)
	}
}
