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
	"fmt"
	"slices"
	"testing"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const keycloakSeedUsers = 3

func keycloakProvider(h *eeharness.H, k *eeharness.Keycloak,
	secretName string) *enterprisev1.DirectoryProvider {
	return &enterprisev1.DirectoryProvider{
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Keycloak_{
				Keycloak: &enterprisev1.DirectoryProvider_Spec_Keycloak{
					Url:      k.URL,
					Realm:    k.Realm,
					ClientID: k.ClientID,
					ClientSecret: &enterprisev1.DirectoryProvider_Spec_Keycloak_ClientSecret{
						Type: &enterprisev1.DirectoryProvider_Spec_Keycloak_ClientSecret_FromSecret{
							FromSecret: secretName,
						},
					},
					Polling: &enterprisev1.DirectoryProvider_Spec_Keycloak_Polling{
						Interval: eeharness.Minutes(1),
					},
				},
			},
		},
	}
}

func testKeycloakDirectoryProvider(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapKeycloak)

	k := h.Keycloak(t)
	k.SeedRealm(t)

	users := map[string]string{}
	for i := range keycloakSeedUsers {
		name := fmt.Sprintf("kc-user-%d", i)
		users[name] = k.CreateUser(t, name,
			fmt.Sprintf("%s@octelium.com", name), "Key", "Cloak")
	}

	engineering := k.CreateGroup(t, "engineering")
	security := k.CreateGroup(t, "security")

	k.AddMember(t, users["kc-user-0"], engineering)
	k.AddMember(t, users["kc-user-1"], engineering)
	k.AddMember(t, users["kc-user-2"], security)

	sec := h.CreateEnterpriseSecret(t, k.Secret)
	dp := h.CreateDirectoryProvider(t, keycloakProvider(h, k, sec.Metadata.Name))

	h.SynchronizeDirectoryProvider(t, dp)
	h.WaitDirectorySynchronized(t, dp,
		enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS, eeharness.SyncBudget)

	t.Run("UsersAndGroupsAreProvisioned", func(t *testing.T) {
		assert.Equal(t, keycloakSeedUsers, len(h.DirectoryUsers(t, dp)))
		assert.Equal(t, 2, len(h.DirectoryGroups(t, dp)))

		usr := keycloakUser(t, h, dp, "kc-user-0@octelium.com")
		assert.Equal(t, "kc-user-0@octelium.com", usr.Spec.Email)
		assert.True(t, len(usr.Spec.Groups) > 0)
	})

	t.Run("MembershipMovesBetweenGroups", func(t *testing.T) {
		k.RemoveMember(t, users["kc-user-0"], engineering)
		k.AddMember(t, users["kc-user-0"], security)

		h.SynchronizeDirectoryProvider(t, dp)
		h.WaitDirectorySynchronized(t, dp,
			enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS, eeharness.SyncBudget)

		before := keycloakUser(t, h, dp, "kc-user-0@octelium.com")
		engineeringName := octeliumGroupName(t, h, dp, "engineering")
		securityName := octeliumGroupName(t, h, dp, "security")

		assert.False(t, slices.Contains(before.Spec.Groups, engineeringName))
		assert.True(t, slices.Contains(before.Spec.Groups, securityName))
	})

	t.Run("DisabledUserIsDisabled", func(t *testing.T) {
		k.SetUserEnabled(t, users["kc-user-1"], false)

		h.SynchronizeDirectoryProvider(t, dp)
		h.WaitDirectorySynchronized(t, dp,
			enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS, eeharness.SyncBudget)

		assert.True(t, keycloakUser(t, h, dp, "kc-user-1@octelium.com").Spec.IsDisabled)
	})

	t.Run("PrunesRemovedPrincipals", func(t *testing.T) {
		k.DeleteUser(t, users["kc-user-2"])
		k.DeleteGroup(t, engineering)

		h.SynchronizeDirectoryProvider(t, dp)
		h.WaitDirectorySynchronized(t, dp,
			enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS, eeharness.SyncBudget)

		assert.Equal(t, keycloakSeedUsers-1, len(h.DirectoryUsers(t, dp)))
		assert.Equal(t, 1, len(h.DirectoryGroups(t, dp)))
	})

	t.Run("LeavesUnmanagedPrincipalsAlone", func(t *testing.T) {
		local := h.CreateWorkloadUser(t, nil)

		h.SynchronizeDirectoryProvider(t, dp)
		h.WaitDirectorySynchronized(t, dp,
			enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS, eeharness.SyncBudget)

		ctx, cancel := h.Ctx(t)
		defer cancel()

		_, err := h.CoreC().GetUser(ctx, &metav1.GetOptions{Uid: local.Metadata.Uid})
		assert.Nil(t, err)
	})

	t.Run("PollingSynchronizesWithoutAnRPC", func(t *testing.T) {
		k.CreateUser(t, "kc-polled", "kc-polled@octelium.com", "Polled", "User")

		h.Eventually(t, "the polled synchronization to provision the new User",
			eeharness.SyncBudget, func(ctx context.Context) error {
				for _, itm := range h.DirectoryUsers(t, dp) {
					usr, err := h.CoreC().GetUser(ctx,
						&metav1.GetOptions{Uid: itm.Status.UserRef.Uid})
					if err != nil {
						return err
					}
					if usr.Spec.Email == "kc-polled@octelium.com" {
						return nil
					}
				}
				return errNotProvisioned("kc-polled@octelium.com")
			})
	})
}

func testKeycloakBadCredentials(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapKeycloak)

	k := h.Keycloak(t)
	k.SeedRealm(t)

	k.CreateUser(t, "kc-bad", "kc-bad@octelium.com", "Bad", "Credentials")

	sec := h.CreateEnterpriseSecret(t, h.Name()+h.Name())
	dp := h.CreateDirectoryProvider(t, keycloakProvider(h, k, sec.Metadata.Name))

	h.SynchronizeDirectoryProvider(t, dp)
	h.WaitDirectorySynchronized(t, dp,
		enterprisev1.DirectoryProvider_Status_Synchronization_FAILED, eeharness.SyncBudget)

	assert.Equal(t, 0, len(h.DirectoryUsers(t, dp)))
	assert.Equal(t, 0, len(h.DirectoryGroups(t, dp)))
}

func testDirectoryProviderIsolation(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, eescenario.CapKeycloak)

	k := h.Keycloak(t)
	k.SeedRealm(t)
	k.CreateUser(t, "shared", "shared@octelium.com", "Shared", "Name")

	sec := h.CreateEnterpriseSecret(t, k.Secret)
	kcDP := h.CreateDirectoryProvider(t, keycloakProvider(h, k, sec.Metadata.Name))

	h.SynchronizeDirectoryProvider(t, kcDP)
	h.WaitDirectorySynchronized(t, kcDP,
		enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS, eeharness.SyncBudget)

	s := newSCIMCtx(t, h)
	s.createUser(t, "shared@octelium.com", "shared@octelium.com")

	require.Equal(t, 1, len(h.DirectoryUsers(t, s.dp)))

	h.SynchronizeDirectoryProvider(t, kcDP)
	h.WaitDirectorySynchronized(t, kcDP,
		enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS, eeharness.SyncBudget)

	assert.Equal(t, 1, len(h.DirectoryUsers(t, s.dp)))
	assert.Equal(t, 1, len(h.DirectoryUsers(t, kcDP)))
}

func keycloakUser(t *testing.T, h *eeharness.H,
	dp *enterprisev1.DirectoryProvider, email string) *corev1.User {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	for _, itm := range h.DirectoryUsers(t, dp) {
		usr, err := h.CoreC().GetUser(ctx, &metav1.GetOptions{Uid: itm.Status.UserRef.Uid})
		require.Nil(t, err)

		if usr.Spec.Email == email {
			return usr
		}
	}

	t.Fatalf("Could not find a provisioned User with the email %q", email)
	return nil
}

func octeliumGroupName(t *testing.T, h *eeharness.H,
	dp *enterprisev1.DirectoryProvider, displayName string) string {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	for _, itm := range h.DirectoryGroups(t, dp) {
		grp, err := h.CoreC().GetGroup(ctx, &metav1.GetOptions{Uid: itm.Status.GroupRef.Uid})
		if err != nil {
			continue
		}
		if grp.Metadata.DisplayName == displayName {
			return grp.Metadata.Name
		}
	}

	return displayName
}
