// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDoListCommonOptionsComprehensive(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

	for idx := range 12 {
		tags := []string{"all"}
		if idx%2 == 0 {
			tags = append(tags, "even")
		}

		email := fmt.Sprintf("user-%02d@example.com", idx)
		if idx == 7 {
			email = "needle@example.com"
		}

		insertRscStoreObject(t, env, &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata: &metav1.Metadata{
				Name:            fmt.Sprintf("user-%02d", idx),
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
				CreatedAt:       pbutils.Timestamp(base.Add(time.Duration(idx) * time.Minute)),
				Tags:            tags,
			},
			Spec: &corev1.User_Spec{
				Type:  corev1.User_Spec_HUMAN,
				Email: email,
			},
			Status: &corev1.User_Status{},
		})
	}

	insertRscStoreObject(t, env, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name:            "hidden-user",
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
			CreatedAt:       pbutils.Timestamp(base),
			IsSystemHidden:  true,
		},
		Spec:   &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
		Status: &corev1.User_Status{},
	})

	err := env.srv.recreateFTSIndex(env.ctx)
	assert.Nil(t, err, "%+v", err)

	srvCore := &srvCore{s: env.srv}

	{
		resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, defaultItemsPerPage)
		assert.Equal(t, uint32(12), resp.ListResponseMeta.TotalCount)
		assert.Equal(t, uint32(defaultItemsPerPage), resp.ListResponseMeta.ItemsPerPage)
		assert.True(t, resp.ListResponseMeta.HasMore)
		assert.Equal(t, "user-11", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{
				Page:         2,
				ItemsPerPage: 5,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 2)
		assert.Equal(t, uint32(12), resp.ListResponseMeta.TotalCount)
		assert.False(t, resp.ListResponseMeta.HasMore)
	}

	{
		resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
				OrderBy: &vmetav1.CommonListOptions_OrderBy{
					Type: vmetav1.CommonListOptions_OrderBy_NAME,
					Mode: vmetav1.CommonListOptions_OrderBy_ASC,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 12)
		assert.Equal(t, "user-00", resp.Items[0].Metadata.Name)
		assert.Equal(t, "user-11", resp.Items[len(resp.Items)-1].Metadata.Name)
	}

	{
		resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
				Tag:          "even",
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 6)
		assert.Equal(t, uint32(6), resp.ListResponseMeta.TotalCount)
	}

	{
		resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{
				ItemsPerPage: 1000,
				From:         pbutils.Timestamp(base.Add(3 * time.Minute)),
				To:           pbutils.Timestamp(base.Add(5 * time.Minute)),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 3)
	}

	{
		resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{
				Query: "needle",
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "user-07", resp.Items[0].Metadata.Name)
	}

	{
		_, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{
				Query: strings.Repeat("a", 101),
			},
		})
		assert.NotNil(t, err)
	}

	{
		_, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{
				Tag: "invalid tag",
			},
		})
		assert.NotNil(t, err)
	}

	{
		_, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{
				Page: 10001,
			},
		})
		assert.NotNil(t, err)
	}

	{
		_, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Common: &vmetav1.CommonListOptions{
				Page:  1,
				Query: "does-not-exist",
			},
		})
		assert.NotNil(t, err)
	}
}

func TestCoreListResourceSpecificFilters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srvCore := &srvCore{s: env.srv}
	now := time.Now().UTC()
	userRef1 := &metav1.ObjectReference{Name: "user-one", Uid: vutils.UUIDv4()}
	userRef2 := &metav1.ObjectReference{Name: "user-two", Uid: vutils.UUIDv4()}
	deviceRef := &metav1.ObjectReference{Name: "device-one", Uid: vutils.UUIDv4()}

	credentialRef := &metav1.ObjectReference{Name: "credential-one", Uid: vutils.UUIDv4()}
	namespaceRef := &metav1.ObjectReference{Name: "namespace-one", Uid: vutils.UUIDv4()}
	regionRef := &metav1.ObjectReference{Name: "region-one", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("disabled-human", now),
		Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN, IsDisabled: true},
		Status:     &corev1.User_Status{},
	})
	insertRscStoreObject(t, env, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("enabled-workload", now.Add(time.Second)),
		Spec:       &corev1.User_Spec{Type: corev1.User_Spec_WORKLOAD},
		Status:     &corev1.User_Status{},
	})

	insertRscStoreObject(t, env, &corev1.Service{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindService,
		Metadata:   newRscStoreMetadata("public-http", now),
		Spec: &corev1.Service_Spec{
			Mode:        corev1.Service_Spec_HTTP,
			IsPublic:    true,
			IsAnonymous: true,
			IsDisabled:  true,
		},
		Status: &corev1.Service_Status{
			NamespaceRef: namespaceRef,
			RegionRef:    regionRef,
		},
	})
	insertRscStoreObject(t, env, &corev1.Service{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindService,
		Metadata:   newRscStoreMetadata("private-tcp", now.Add(time.Second)),
		Spec:       &corev1.Service_Spec{Mode: corev1.Service_Spec_TCP},
		Status:     &corev1.Service_Status{},
	})

	insertRscStoreObject(t, env, &corev1.Session{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindSession,
		Metadata:   newRscStoreMetadata("connected-browser", now),
		Spec:       &corev1.Session_Spec{State: corev1.Session_Spec_ACTIVE},
		Status: &corev1.Session_Status{
			Type:          corev1.Session_Status_CLIENT,
			IsBrowser:     true,
			IsConnected:   true,
			UserRef:       userRef1,
			DeviceRef:     deviceRef,
			CredentialRef: credentialRef,
		},
	})
	insertRscStoreObject(t, env, &corev1.Session{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindSession,
		Metadata:   newRscStoreMetadata("pending-clientless", now.Add(time.Second)),
		Spec:       &corev1.Session_Spec{State: corev1.Session_Spec_PENDING},
		Status: &corev1.Session_Status{
			Type:    corev1.Session_Status_CLIENTLESS,
			UserRef: userRef2,
		},
	})

	insertRscStoreObject(t, env, &corev1.Device{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindDevice,
		Metadata:   newRscStoreMetadata("linux-device", now),
		Spec:       &corev1.Device_Spec{State: corev1.Device_Spec_ACTIVE},
		Status:     &corev1.Device_Status{UserRef: userRef1, OsType: corev1.Device_Status_LINUX},
	})
	insertRscStoreObject(t, env, &corev1.Device{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindDevice,
		Metadata:   newRscStoreMetadata("android-device", now.Add(time.Second)),
		Spec:       &corev1.Device_Spec{State: corev1.Device_Spec_PENDING},
		Status:     &corev1.Device_Status{UserRef: userRef2, OsType: corev1.Device_Status_ANDROID},
	})

	insertRscStoreObject(t, env, &corev1.Credential{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindCredential,
		Metadata:   newRscStoreMetadata("oauth-credential", now),
		Spec:       &corev1.Credential_Spec{Type: corev1.Credential_Spec_OAUTH2, IsDisabled: true},
		Status:     &corev1.Credential_Status{UserRef: userRef1},
	})
	insertRscStoreObject(t, env, &corev1.Credential{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindCredential,
		Metadata:   newRscStoreMetadata("auth-token", now.Add(time.Second)),
		Spec:       &corev1.Credential_Spec{Type: corev1.Credential_Spec_AUTH_TOKEN},
		Status:     &corev1.Credential_Status{UserRef: userRef2},
	})

	insertRscStoreObject(t, env, &corev1.IdentityProvider{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindIdentityProvider,
		Metadata:   newRscStoreMetadata("oidc-idp", now),
		Spec:       &corev1.IdentityProvider_Spec{IsDisabled: true},
		Status:     &corev1.IdentityProvider_Status{Type: corev1.IdentityProvider_Status_OIDC},
	})
	insertRscStoreObject(t, env, &corev1.IdentityProvider{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindIdentityProvider,
		Metadata:   newRscStoreMetadata("github-idp", now.Add(time.Second)),
		Spec:       &corev1.IdentityProvider_Spec{},
		Status:     &corev1.IdentityProvider_Status{Type: corev1.IdentityProvider_Status_GITHUB},
	})

	insertRscStoreObject(t, env, &corev1.Authenticator{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindAuthenticator,
		Metadata:   newRscStoreMetadata("registered-fido", now),
		Spec:       &corev1.Authenticator_Spec{State: corev1.Authenticator_Spec_ACTIVE},
		Status: &corev1.Authenticator_Status{
			Type:         corev1.Authenticator_Status_FIDO,
			UserRef:      userRef1,
			DeviceRef:    deviceRef,
			IsRegistered: true,
		},
	})
	insertRscStoreObject(t, env, &corev1.Authenticator{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindAuthenticator,
		Metadata:   newRscStoreMetadata("pending-totp", now.Add(time.Second)),
		Spec:       &corev1.Authenticator_Spec{State: corev1.Authenticator_Spec_PENDING},
		Status:     &corev1.Authenticator_Status{Type: corev1.Authenticator_Status_TOTP, UserRef: userRef2},
	})

	{
		resp, err := srvCore.ListUser(env.ctx, &vcorev1.ListUserOptions{
			Type:       corev1.User_Spec_HUMAN,
			IsDisabled: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "disabled-human", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListService(env.ctx, &vcorev1.ListServiceOptions{
			NamespaceRef: namespaceRef,
			RegionRef:    regionRef,
			Mode:         corev1.Service_Spec_HTTP,
			IsPublic:     true,
			IsAnonymous:  true,
			IsDisabled:   true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "public-http", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListSession(env.ctx, &vcorev1.ListSessionOptions{
			UserRef:       userRef1,
			DeviceRef:     deviceRef,
			CredentialRef: credentialRef,
			State:         corev1.Session_Spec_ACTIVE,
			Type:          corev1.Session_Status_CLIENT,
			IsBrowser:     true,
			IsConnected:   true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "connected-browser", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListDevice(env.ctx, &vcorev1.ListDeviceOptions{
			UserRef: userRef2,
			State:   corev1.Device_Spec_PENDING,
			OsType:  corev1.Device_Status_ANDROID,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "android-device", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListCredential(env.ctx, &vcorev1.ListCredentialOptions{
			UserRef:    userRef1,
			Type:       corev1.Credential_Spec_OAUTH2,
			IsDisabled: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "oauth-credential", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListIdentityProvider(env.ctx, &vcorev1.ListIdentityProviderOptions{
			Type:       corev1.IdentityProvider_Status_OIDC,
			IsDisabled: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "oidc-idp", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListAuthenticator(env.ctx, &vcorev1.ListAuthenticatorOptions{
			UserRef:      userRef1,
			DeviceRef:    deviceRef,
			State:        corev1.Authenticator_Spec_ACTIVE,
			Type:         corev1.Authenticator_Status_FIDO,
			IsRegistered: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "registered-fido", resp.Items[0].Metadata.Name)
	}
}

func TestCoreSimpleListMethodsAndPolicyFilter(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC()
	srvCore := &srvCore{s: env.srv}
	regionRef := &metav1.ObjectReference{Name: "region-one", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &corev1.Group{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindGroup,
		Metadata:   newRscStoreMetadata("visible-group", now),
		Spec:       &corev1.Group_Spec{},
		Status:     &corev1.Group_Status{},
	})
	insertRscStoreObject(t, env, &corev1.Namespace{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindNamespace,
		Metadata:   newRscStoreMetadata("visible-namespace", now),
		Spec:       &corev1.Namespace_Spec{},
		Status:     &corev1.Namespace_Status{},
	})
	insertRscStoreObject(t, env, &corev1.Region{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindRegion,
		Metadata:   newRscStoreMetadata("visible-region", now),
		Spec:       &corev1.Region_Spec{},
		Status:     &corev1.Region_Status{},
	})
	insertRscStoreObject(t, env, &corev1.Gateway{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindGateway,
		Metadata:   newRscStoreMetadata("visible-gateway", now),
		Spec:       &corev1.Gateway_Spec{},
		Status:     &corev1.Gateway_Status{RegionRef: regionRef},
	})
	insertRscStoreObject(t, env, &corev1.Gateway{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindGateway,
		Metadata:   newRscStoreMetadata("other-gateway", now.Add(time.Second)),
		Spec:       &corev1.Gateway_Spec{},
		Status: &corev1.Gateway_Status{RegionRef: &metav1.ObjectReference{
			Name: "region-two",
			Uid:  vutils.UUIDv4(),
		}},
	})
	insertRscStoreObject(t, env, &corev1.Secret{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindSecret,
		Metadata:   newRscStoreMetadata("visible-secret", now),
		Spec:       &corev1.Secret_Spec{},
		Status:     &corev1.Secret_Status{},
	})
	insertRscStoreObject(t, env, &corev1.Policy{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindPolicy,
		Metadata:   newRscStoreMetadata("enabled-policy", now),
		Spec:       &corev1.Policy_Spec{},
		Status:     &corev1.Policy_Status{},
	})
	insertRscStoreObject(t, env, &corev1.Policy{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindPolicy,
		Metadata:   newRscStoreMetadata("disabled-policy", now.Add(time.Second)),
		Spec:       &corev1.Policy_Spec{IsDisabled: true},
		Status:     &corev1.Policy_Status{},
	})

	{
		resp, err := srvCore.ListGroup(env.ctx, &vcorev1.ListGroupOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "visible-group", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListNamespace(env.ctx, &vcorev1.ListNamespaceOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "visible-namespace", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListRegion(env.ctx, &vcorev1.ListRegionOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "visible-region", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListGateway(env.ctx, &vcorev1.ListGatewayOptions{RegionRef: regionRef})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "visible-gateway", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListSecret(env.ctx, &vcorev1.ListSecretOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "visible-secret", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srvCore.ListPolicy(env.ctx, &vcorev1.ListPolicyOptions{IsDisabled: true})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "disabled-policy", resp.Items[0].Metadata.Name)
	}
}

func TestDoListRejectsInvertedTimeRange(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC()
	insertRscStoreObject(t, env, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("time-range-user", now),
		Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
		Status:     &corev1.User_Status{},
	})

	_, err := (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{
			From: pbutils.Timestamp(now.Add(time.Hour)),
			To:   pbutils.Timestamp(now),
		},
	})
	assert.NotNil(t, err)
}

func TestListUserGroupUIDFilterIsNotIgnored(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	group := &corev1.Group{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindGroup,
		Metadata:   newRscStoreMetadata("engineering", time.Now()),
		Spec:       &corev1.Group_Spec{},
		Status:     &corev1.Group_Status{},
	}
	insertRscStoreObject(t, env, group)

	insertRscStoreObject(t, env, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("member", time.Now()),
		Spec: &corev1.User_Spec{
			Type:   corev1.User_Spec_HUMAN,
			Groups: []string{group.Metadata.Name},
		},
		Status: &corev1.User_Status{},
	})
	insertRscStoreObject(t, env, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("non-member", time.Now().Add(time.Second)),
		Spec: &corev1.User_Spec{
			Type:   corev1.User_Spec_HUMAN,
			Groups: []string{"other-group"},
		},
		Status: &corev1.User_Status{},
	})

	resp, err := (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		GroupRef: &metav1.ObjectReference{Uid: group.Metadata.Uid, Name: group.Metadata.Name},
		Common:   &vmetav1.CommonListOptions{ItemsPerPage: 1000},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "member", resp.Items[0].Metadata.Name)
}

func TestDoListAcceptsEqualTimeRange(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC().Truncate(time.Second)

	insertRscStoreObject(t, env, &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata: &metav1.Metadata{
			Name:            "equal-time-user",
			Uid:             vutils.UUIDv4(),
			ResourceVersion: vutils.UUIDv7(),
			CreatedAt:       pbutils.Timestamp(now),
		},
		Spec:   &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
		Status: &corev1.User_Status{},
	})

	resp, err := (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{
			From: pbutils.Timestamp(now),
			To:   pbutils.Timestamp(now),
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "equal-time-user", resp.Items[0].Metadata.Name)
}

func TestDoListRejectsInvalidFromTimestamp(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	_, err := (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{
			From: &timestamppb.Timestamp{
				Seconds: 253402300800,
			},
		},
	})
	assert.NotNil(t, err)
}

func TestDoListRejectsInvalidToTimestamp(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	_, err := (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{
			To: &timestamppb.Timestamp{
				Nanos: -1,
			},
		},
	})
	assert.NotNil(t, err)
}
