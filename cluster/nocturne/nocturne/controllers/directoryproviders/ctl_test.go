package directoryproviders

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestDirectoryDeletionCleansOnlyOwnedProviderLinks(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	client := tst.C.OcteliumC
	actor, err := client.CoreC().CreateUser(t.Context(), &corev1.User{
		Metadata: &metav1.Metadata{Name: "directory-actor"}, Spec: &corev1.User_Spec{Type: corev1.User_Spec_WORKLOAD}, Status: &corev1.User_Status{},
	})
	require.NoError(t, err)
	protected, err := client.CoreC().CreateUser(t.Context(), &corev1.User{
		Metadata: &metav1.Metadata{Name: "local-user"}, Spec: &corev1.User_Spec{Type: corev1.User_Spec_HUMAN}, Status: &corev1.User_Status{},
	})
	require.NoError(t, err)
	group, err := client.CoreC().CreateGroup(t.Context(), &corev1.Group{
		Metadata: &metav1.Metadata{Name: "local-group"}, Spec: &corev1.Group_Spec{}, Status: &corev1.Group_Status{},
	})
	require.NoError(t, err)
	provider := &enterprisev1.DirectoryProvider{Metadata: &metav1.Metadata{Uid: vutils.UUIDv4()}, Status: &enterprisev1.DirectoryProvider_Status{UserRef: umetav1.GetObjectReference(actor)}}
	other := &metav1.ObjectReference{Uid: vutils.UUIDv4()}
	for i := range 4 {
		ref := umetav1.GetObjectReference(provider)
		if i == 3 {
			ref = other
		}
		_, err = client.EnterpriseC().CreateDirectoryProviderUser(t.Context(), &enterprisev1.DirectoryProviderUser{
			Metadata: &metav1.Metadata{Name: fmt.Sprintf("user-link-%d", i)}, Spec: &enterprisev1.DirectoryProviderUser_Spec{},
			Status: &enterprisev1.DirectoryProviderUser_Status{DirectoryProviderRef: ref, UserRef: umetav1.GetObjectReference(protected)},
		})
		require.NoError(t, err)
		_, err = client.EnterpriseC().CreateDirectoryProviderGroup(t.Context(), &enterprisev1.DirectoryProviderGroup{
			Metadata: &metav1.Metadata{Name: fmt.Sprintf("group-link-%d", i)}, Spec: &enterprisev1.DirectoryProviderGroup_Spec{},
			Status: &enterprisev1.DirectoryProviderGroup_Status{DirectoryProviderRef: ref, GroupRef: umetav1.GetObjectReference(group)},
		})
		require.NoError(t, err)
	}
	ctl, err := NewController(t.Context(), client)
	require.NoError(t, err)
	require.NoError(t, ctl.OnAdd(t.Context(), provider))
	require.NoError(t, ctl.OnUpdate(t.Context(), provider, provider))
	require.NoError(t, ctl.OnDelete(t.Context(), provider))
	users, err := client.EnterpriseC().ListDirectoryProviderUser(t.Context(), &rmetav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, users.Items, 1)
	assert.Equal(t, other.Uid, users.Items[0].Status.DirectoryProviderRef.Uid)
	groups, err := client.EnterpriseC().ListDirectoryProviderGroup(t.Context(), &rmetav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, groups.Items, 1)
	assert.Equal(t, other.Uid, groups.Items[0].Status.DirectoryProviderRef.Uid)
	_, err = client.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: actor.Metadata.Uid})
	assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	_, err = client.CoreC().GetUser(t.Context(), &rmetav1.GetOptions{Uid: protected.Metadata.Uid})
	require.NoError(t, err)
	_, err = client.CoreC().GetGroup(t.Context(), &rmetav1.GetOptions{Uid: group.Metadata.Uid})
	require.NoError(t, err)
	require.NoError(t, ctl.OnDelete(t.Context(), provider))
}

type cleanupListTestClient struct {
	octeliumc.ClientInterface
	resources *cleanupListTestResources
}

func (c *cleanupListTestClient) EnterpriseC() renterprisev1.ResourceServiceClient { return c.resources }

type cleanupListTestResources struct {
	renterprisev1.ResourceServiceClient
	usersErr, groupsErr error
	groupCalls          int
}

func (c *cleanupListTestResources) ListDirectoryProviderUser(context.Context, *rmetav1.ListOptions, ...grpc.CallOption) (*enterprisev1.DirectoryProviderUserList, error) {
	return &enterprisev1.DirectoryProviderUserList{}, c.usersErr
}
func (c *cleanupListTestResources) ListDirectoryProviderGroup(context.Context, *rmetav1.ListOptions, ...grpc.CallOption) (*enterprisev1.DirectoryProviderGroupList, error) {
	c.groupCalls++
	return &enterprisev1.DirectoryProviderGroupList{}, c.groupsErr
}
func TestDirectoryCleanupListFailuresAreReturned(t *testing.T) {
	failure := errors.New("directory list unavailable")
	for _, resources := range []*cleanupListTestResources{{usersErr: failure}, {groupsErr: failure}} {
		ctl, err := NewController(t.Context(), &cleanupListTestClient{resources: resources})
		require.NoError(t, err)
		err = ctl.OnDelete(t.Context(), &enterprisev1.DirectoryProvider{Metadata: &metav1.Metadata{Uid: vutils.UUIDv4()}, Status: &enterprisev1.DirectoryProvider_Status{}})
		assert.ErrorIs(t, err, failure)
		if resources.usersErr != nil {
			assert.Zero(t, resources.groupCalls)
		} else {
			assert.Equal(t, 1, resources.groupCalls)
		}
	}
}
