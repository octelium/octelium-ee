// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type keycloakDirectoryClient struct {
	enterprisev1.MainServiceClient
	t      *testing.T
	ctx    context.Context
	items  []*enterprisev1.DirectoryProviderUser
	err    error
	onList func()
}

func (c *keycloakDirectoryClient) ListDirectoryProviderUser(ctx context.Context,
	req *enterprisev1.ListDirectoryProviderUserOptions,
	opts ...grpc.CallOption) (*enterprisev1.DirectoryProviderUserList, error) {
	require.Equal(c.t, c.ctx, ctx)
	assert.Equal(c.t, "provider", req.GetDirectoryProviderRef().GetUid())
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.onList != nil {
		c.onList()
	}
	return &enterprisev1.DirectoryProviderUserList{Items: c.items}, nil
}

type keycloakCoreClient struct {
	corev1.MainServiceClient
	t     *testing.T
	ctx   context.Context
	users map[string]*corev1.User
	err   error
}

func (c *keycloakCoreClient) GetUser(ctx context.Context,
	req *metav1.GetOptions, opts ...grpc.CallOption) (*corev1.User, error) {
	require.Equal(c.t, c.ctx, ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.err != nil {
		return nil, c.err
	}
	usr := c.users[req.Uid]
	require.NotNil(c.t, usr)
	return usr, nil
}

func TestKeycloakUserProvisioned(t *testing.T) {
	for _, tc := range []struct {
		name      string
		email     string
		listError error
		userError error
		want      error
	}{
		{name: "Provisioned", email: "polled@octelium.com"},
		{name: "Missing", email: "missing@octelium.com", want: errNotProvisioned("missing@octelium.com")},
		{name: "ListError", listError: context.DeadlineExceeded, want: context.DeadlineExceeded},
		{name: "UserError", userError: context.DeadlineExceeded, want: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			c := &keycloakDirectoryClient{
				t: t, ctx: ctx, err: tc.listError,
				items: []*enterprisev1.DirectoryProviderUser{
					{Status: &enterprisev1.DirectoryProviderUser_Status{UserRef: &metav1.ObjectReference{Uid: "existing"}}},
					{Status: &enterprisev1.DirectoryProviderUser_Status{UserRef: &metav1.ObjectReference{Uid: "polled"}}},
				},
			}
			coreC := &keycloakCoreClient{
				t: t, ctx: ctx, err: tc.userError,
				users: map[string]*corev1.User{
					"existing": {Spec: &corev1.User_Spec{Email: "existing@octelium.com"}},
					"polled":   {Spec: &corev1.User_Spec{Email: "polled@octelium.com"}},
				},
			}
			dp := &enterprisev1.DirectoryProvider{Metadata: &metav1.Metadata{Uid: "provider"}}
			err := keycloakUserProvisioned(ctx, c, coreC, dp, tc.email)
			if tc.want == nil {
				require.Nil(t, err)
			} else {
				require.EqualError(t, err, tc.want.Error())
			}
		})
	}
}

func TestKeycloakUserProvisionedContextCancellation(t *testing.T) {
	for _, name := range []string{"List", "GetUser"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			c := &keycloakDirectoryClient{
				t: t, ctx: ctx,
				items: []*enterprisev1.DirectoryProviderUser{
					{Status: &enterprisev1.DirectoryProviderUser_Status{UserRef: &metav1.ObjectReference{Uid: "polled"}}},
				},
			}
			if name == "List" {
				cancel()
			} else {
				c.onList = cancel
			}
			coreC := &keycloakCoreClient{t: t, ctx: ctx}
			dp := &enterprisev1.DirectoryProvider{Metadata: &metav1.Metadata{Uid: "provider"}}
			require.ErrorIs(t, keycloakUserProvisioned(ctx, c, coreC, dp,
				"polled@octelium.com"), context.Canceled)
		})
	}
}
