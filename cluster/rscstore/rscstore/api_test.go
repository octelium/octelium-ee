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
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/cluster/caccessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcordiumv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestListInvalidServiceReferences(t *testing.T) {
	ctx := context.Background()
	access := &srvAccess{}
	enterprise := &srvEnterprise{}

	for _, tst := range []struct {
		name string
		list func(*metav1.ObjectReference) error
	}{
		{
			name: "policy",
			list: func(ref *metav1.ObjectReference) error {
				_, err := access.ListPolicy(ctx, &vaccessv1.ListPolicyOptions{ServiceRef: ref})
				return err
			},
		},
		{
			name: "catalog",
			list: func(ref *metav1.ObjectReference) error {
				_, err := access.ListCatalog(ctx, &vaccessv1.ListCatalogOptions{ServiceRef: ref})
				return err
			},
		},
		{
			name: "request",
			list: func(ref *metav1.ObjectReference) error {
				_, err := access.ListRequest(ctx, &vaccessv1.ListRequestOptions{ServiceRef: ref})
				return err
			},
		},
		{
			name: "certificate",
			list: func(ref *metav1.ObjectReference) error {
				_, err := enterprise.ListCertificate(ctx, &venterprisev1.ListCertificateOptions{ServiceRef: ref})
				return err
			},
		},
	} {
		t.Run(tst.name, func(t *testing.T) {
			for _, ref := range []*metav1.ObjectReference{
				{Name: "invalid name"},
				{Name: ".default"},
				{Name: "service."},
				{Name: "service..default"},
				{Name: "service.namespace.parent"},
				{Name: "service.default", Uid: "not-a-uid"},
			} {
				err := tst.list(ref)
				assert.True(t, grpcerr.IsInvalidArg(err), "%+v: %+v", ref, err)
			}
		})
	}
}

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
		Metadata:   newRscStoreMetadata("public-http.namespace-one", now),
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
		assert.Equal(t, "public-http.namespace-one", resp.Items[0].Metadata.Name)
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
		Metadata:   newRscStoreMetadata("disabled-policy.parent.root", now.Add(time.Second)),
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
		assert.Equal(t, "disabled-policy.parent.root", resp.Items[0].Metadata.Name)
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

func newRscStoreRPCClient(t *testing.T, env *rscStoreTestEnv) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	vcorev1.RegisterResourceServiceServer(server, &srvCore{s: env.srv})
	venterprisev1.RegisterResourceServiceServer(server, &srvEnterprise{s: env.srv})
	vaccessv1.RegisterResourceServiceServer(server, &srvAccess{s: env.srv})
	vcordiumv1.RegisterResourceServiceServer(server, &srvCordium{s: env.srv})
	caccessv1.RegisterInternalServiceServer(server, &srvClusterAccess{s: env.srv})
	go server.Serve(listener)
	conn, err := grpc.NewClient("passthrough:///rscstore-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) { return listener.DialContext(ctx) }))
	require.Nil(t, err)
	t.Cleanup(func() {
		conn.Close()
		server.Stop()
		listener.Close()
	})
	return conn
}

type rscStoreSummaryRPC struct {
	name string
	api  string
	kind string
	call func(context.Context, *vmetav1.CommonSummaryOptions) (proto.Message, error)
}

func rscStoreSummaryRPCCases(conn *grpc.ClientConn) []rscStoreSummaryRPC {
	core := vcorev1.NewResourceServiceClient(conn)
	enterprise := venterprisev1.NewResourceServiceClient(conn)
	access := vaccessv1.NewResourceServiceClient(conn)
	cordium := vcordiumv1.NewResourceServiceClient(conn)
	return []rscStoreSummaryRPC{
		{name: "core/Service", api: "core", kind: "Service", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetServiceSummary(ctx, &vcorev1.GetServiceSummaryRequest{Common: common})
		}},
		{name: "core/Session", api: "core", kind: "Session", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetSessionSummary(ctx, &vcorev1.GetSessionSummaryRequest{Common: common})
		}},
		{name: "core/User", api: "core", kind: "User", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetUserSummary(ctx, &vcorev1.GetUserSummaryRequest{Common: common})
		}},
		{name: "core/Policy", api: "core", kind: "Policy", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetPolicySummary(ctx, &vcorev1.GetPolicySummaryRequest{Common: common})
		}},
		{name: "core/Credential", api: "core", kind: "Credential", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetCredentialSummary(ctx, &vcorev1.GetCredentialSummaryRequest{Common: common})
		}},
		{name: "core/Device", api: "core", kind: "Device", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetDeviceSummary(ctx, &vcorev1.GetDeviceSummaryRequest{Common: common})
		}},
		{name: "core/IdentityProvider", api: "core", kind: "IdentityProvider", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetIdentityProviderSummary(ctx, &vcorev1.GetIdentityProviderSummaryRequest{Common: common})
		}},
		{name: "core/Authenticator", api: "core", kind: "Authenticator", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetAuthenticatorSummary(ctx, &vcorev1.GetAuthenticatorSummaryRequest{Common: common})
		}},
		{name: "core/Group", api: "core", kind: "Group", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetGroupSummary(ctx, &vcorev1.GetGroupSummaryRequest{Common: common})
		}},
		{name: "core/Gateway", api: "core", kind: "Gateway", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetGatewaySummary(ctx, &vcorev1.GetGatewaySummaryRequest{Common: common})
		}},
		{name: "core/Region", api: "core", kind: "Region", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetRegionSummary(ctx, &vcorev1.GetRegionSummaryRequest{Common: common})
		}},
		{name: "core/Secret", api: "core", kind: "Secret", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetSecretSummary(ctx, &vcorev1.GetSecretSummaryRequest{Common: common})
		}},
		{name: "core/Namespace", api: "core", kind: "Namespace", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return core.GetNamespaceSummary(ctx, &vcorev1.GetNamespaceSummaryRequest{Common: common})
		}},
		{name: "enterprise/CollectorExporter", api: "enterprise", kind: "CollectorExporter", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetCollectorExporterSummary(ctx, &venterprisev1.GetCollectorExporterSummaryRequest{Common: common})
		}},
		{name: "enterprise/Secret", api: "enterprise", kind: "Secret", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetSecretSummary(ctx, &venterprisev1.GetSecretSummaryRequest{Common: common})
		}},
		{name: "enterprise/SecretStore", api: "enterprise", kind: "SecretStore", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetSecretStoreSummary(ctx, &venterprisev1.GetSecretStoreSummaryRequest{Common: common})
		}},
		{name: "enterprise/Certificate", api: "enterprise", kind: "Certificate", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetCertificateSummary(ctx, &venterprisev1.GetCertificateSummaryRequest{Common: common})
		}},
		{name: "enterprise/CertificateIssuer", api: "enterprise", kind: "CertificateIssuer", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetCertificateIssuerSummary(ctx, &venterprisev1.GetCertificateIssuerSummaryRequest{Common: common})
		}},
		{name: "enterprise/DNSProvider", api: "enterprise", kind: "DNSProvider", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetDNSProviderSummary(ctx, &venterprisev1.GetDNSProviderSummaryRequest{Common: common})
		}},
		{name: "enterprise/DirectoryProvider", api: "enterprise", kind: "DirectoryProvider", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetDirectoryProviderSummary(ctx, &venterprisev1.GetDirectoryProviderSummaryRequest{Common: common})
		}},
		{name: "enterprise/DirectoryProviderUser", api: "enterprise", kind: "DirectoryProviderUser", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetDirectoryProviderUserSummary(ctx, &venterprisev1.GetDirectoryProviderUserSummaryRequest{Common: common})
		}},
		{name: "enterprise/DirectoryProviderGroup", api: "enterprise", kind: "DirectoryProviderGroup", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetDirectoryProviderGroupSummary(ctx, &venterprisev1.GetDirectoryProviderGroupSummaryRequest{Common: common})
		}},
		{name: "enterprise/DeviceManager", api: "enterprise", kind: "DeviceManager", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return enterprise.GetDeviceManagerSummary(ctx, &venterprisev1.GetDeviceManagerSummaryRequest{Common: common})
		}},
		{name: "access/Policy", api: "access", kind: "Policy", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return access.GetPolicySummary(ctx, &vaccessv1.GetPolicySummaryRequest{Common: common})
		}},
		{name: "access/Catalog", api: "access", kind: "Catalog", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return access.GetCatalogSummary(ctx, &vaccessv1.GetCatalogSummaryRequest{Common: common})
		}},
		{name: "access/Request", api: "access", kind: "Request", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return access.GetRequestSummary(ctx, &vaccessv1.GetRequestSummaryRequest{Common: common})
		}},
		{name: "access/Review", api: "access", kind: "Review", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return access.GetReviewSummary(ctx, &vaccessv1.GetReviewSummaryRequest{Common: common})
		}},
		{name: "access/Secret", api: "access", kind: "Secret", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return access.GetSecretSummary(ctx, &vaccessv1.GetSecretSummaryRequest{Common: common})
		}},
		{name: "access/Integration", api: "access", kind: "Integration", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return access.GetIntegrationSummary(ctx, &vaccessv1.GetIntegrationSummaryRequest{Common: common})
		}},
		{name: "access/IntegrationIdentity", api: "access", kind: "IntegrationIdentity", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return access.GetIntegrationIdentitySummary(ctx, &vaccessv1.GetIntegrationIdentitySummaryRequest{Common: common})
		}},
		{name: "access/IntegrationBinding", api: "access", kind: "IntegrationBinding", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return access.GetIntegrationBindingSummary(ctx, &vaccessv1.GetIntegrationBindingSummaryRequest{Common: common})
		}},
		{name: "cordium/Workspace", api: "cordium", kind: "Workspace", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return cordium.GetWorkspaceSummary(ctx, &vcordiumv1.GetWorkspaceSummaryRequest{Common: common})
		}},
		{name: "cordium/Template", api: "cordium", kind: "Template", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return cordium.GetTemplateSummary(ctx, &vcordiumv1.GetTemplateSummaryRequest{Common: common})
		}},
		{name: "cordium/Space", api: "cordium", kind: "Space", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return cordium.GetSpaceSummary(ctx, &vcordiumv1.GetSpaceSummaryRequest{Common: common})
		}},
		{name: "cordium/Membership", api: "cordium", kind: "Membership", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return cordium.GetMembershipSummary(ctx, &vcordiumv1.GetMembershipSummaryRequest{Common: common})
		}},
		{name: "cordium/GitProvider", api: "cordium", kind: "GitProvider", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return cordium.GetGitProviderSummary(ctx, &vcordiumv1.GetGitProviderSummaryRequest{Common: common})
		}},
		{name: "cordium/Secret", api: "cordium", kind: "Secret", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return cordium.GetSecretSummary(ctx, &vcordiumv1.GetSecretSummaryRequest{Common: common})
		}},
		{name: "cordium/UserSecret", api: "cordium", kind: "UserSecret", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return cordium.GetUserSecretSummary(ctx, &vcordiumv1.GetUserSecretSummaryRequest{Common: common})
		}},
		{name: "cordium/Region", api: "core", kind: "Region", call: func(ctx context.Context, common *vmetav1.CommonSummaryOptions) (proto.Message, error) {
			return cordium.GetRegionSummary(ctx, &vcordiumv1.GetRegionSummaryRequest{Common: common})
		}},
	}
}

func TestSummaryRPCWindowsAndFailures(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	conn := newRscStoreRPCClient(t, env)
	cases := rscStoreSummaryRPCCases(conn)
	base := time.Date(2025, time.January, 2, 12, 0, 0, 0, time.UTC)
	common := &vmetav1.CommonSummaryOptions{
		From: timestamppb.New(base), To: timestamppb.New(base.Add(time.Hour)),
		CompareFrom: timestamppb.New(base.Add(-3 * time.Hour)), CompareTo: timestamppb.New(base.Add(-time.Hour)),
	}
	seeded := make(map[string]bool)
	for _, tt := range cases {
		key := tt.api + "/" + tt.kind
		if seeded[key] {
			continue
		}
		seeded[key] = true
		for idx, createdAt := range []time.Time{base.Add(time.Minute), base.Add(-2 * time.Hour), base.Add(-90 * time.Minute), base.Add(2 * time.Minute)} {
			item, err := ovutils.NewResourceObject(tt.api, "v1", tt.kind)
			require.Nil(t, err)
			md := newRscStoreMetadata(fmt.Sprintf("rpc-%s-%d", strings.ToLower(tt.kind), idx), createdAt)
			md.IsSystemHidden = idx == 3
			msg := item.ProtoReflect()
			fields := msg.Descriptor().Fields()
			msg.Set(fields.ByName("apiVersion"), protoreflect.ValueOfString(tt.api+"/v1"))
			msg.Set(fields.ByName("kind"), protoreflect.ValueOfString(tt.kind))
			msg.Set(fields.ByName("metadata"), protoreflect.ValueOfMessage(md.ProtoReflect()))
			insertRscStoreObject(t, env, item)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			original := proto.Clone(common)
			response, err := tt.call(ctx, common)
			require.Nil(t, err)
			msg := response.ProtoReflect()
			assert.Equal(t, uint64(1), msg.Get(msg.Descriptor().Fields().ByName("totalNumber")).Uint())
			previous := msg.Descriptor().Fields().ByName("previous")
			require.NotNil(t, previous)
			require.True(t, msg.Has(previous))
			prev := msg.Get(previous).Message()
			assert.Equal(t, uint64(2), prev.Get(prev.Descriptor().Fields().ByName("totalNumber")).Uint())
			assert.True(t, proto.Equal(original, common))
			for _, invalid := range []*vmetav1.CommonSummaryOptions{
				{From: &timestamppb.Timestamp{Seconds: 253402300800}},
				{To: &timestamppb.Timestamp{Nanos: -1}},
				{From: common.To, To: common.From},
				{CompareFrom: &timestamppb.Timestamp{Seconds: 253402300800}},
				{CompareTo: &timestamppb.Timestamp{Nanos: -1}},
				{CompareFrom: common.To, CompareTo: common.From},
			} {
				_, err := tt.call(ctx, invalid)
				assert.Equal(t, codes.InvalidArgument, status.Code(err), "%+v", err)
			}
		})
	}
	t.Run("DatabaseUnavailable", func(t *testing.T) {
		require.Nil(t, env.srv.db.Close())
		for _, tt := range cases {
			response, err := tt.call(ctx, nil)
			assert.NotNil(t, err, "%s", tt.name)
			assert.True(t, response == nil || !response.ProtoReflect().IsValid(), "%s", tt.name)
		}
	})
}

func TestListRPCDefaultsAndFailures(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)
	conn := newRscStoreRPCClient(t, env)
	core := vcorev1.NewResourceServiceClient(conn)
	enterprise := venterprisev1.NewResourceServiceClient(conn)
	access := vaccessv1.NewResourceServiceClient(conn)
	cordium := vcordiumv1.NewResourceServiceClient(conn)
	cases := []struct {
		name string
		api  string
		kind string
		call func(context.Context, *vmetav1.CommonListOptions) (proto.Message, error)
	}{
		{name: "core/Service", api: "core/v1", kind: "ServiceList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListService(ctx, &vcorev1.ListServiceOptions{Common: common})
		}},
		{name: "core/User", api: "core/v1", kind: "UserList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListUser(ctx, &vcorev1.ListUserOptions{Common: common})
		}},
		{name: "core/Group", api: "core/v1", kind: "GroupList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListGroup(ctx, &vcorev1.ListGroupOptions{Common: common})
		}},
		{name: "core/Namespace", api: "core/v1", kind: "NamespaceList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListNamespace(ctx, &vcorev1.ListNamespaceOptions{Common: common})
		}},
		{name: "core/Policy", api: "core/v1", kind: "PolicyList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListPolicy(ctx, &vcorev1.ListPolicyOptions{Common: common})
		}},
		{name: "core/Region", api: "core/v1", kind: "RegionList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListRegion(ctx, &vcorev1.ListRegionOptions{Common: common})
		}},
		{name: "core/Gateway", api: "core/v1", kind: "GatewayList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListGateway(ctx, &vcorev1.ListGatewayOptions{Common: common})
		}},
		{name: "core/Secret", api: "core/v1", kind: "SecretList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListSecret(ctx, &vcorev1.ListSecretOptions{Common: common})
		}},
		{name: "core/Credential", api: "core/v1", kind: "CredentialList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListCredential(ctx, &vcorev1.ListCredentialOptions{Common: common})
		}},
		{name: "core/IdentityProvider", api: "core/v1", kind: "IdentityProviderList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListIdentityProvider(ctx, &vcorev1.ListIdentityProviderOptions{Common: common})
		}},
		{name: "core/Session", api: "core/v1", kind: "SessionList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListSession(ctx, &vcorev1.ListSessionOptions{Common: common})
		}},
		{name: "core/Device", api: "core/v1", kind: "DeviceList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListDevice(ctx, &vcorev1.ListDeviceOptions{Common: common})
		}},
		{name: "core/Authenticator", api: "core/v1", kind: "AuthenticatorList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return core.ListAuthenticator(ctx, &vcorev1.ListAuthenticatorOptions{Common: common})
		}},
		{name: "enterprise/CollectorExporter", api: "enterprise/v1", kind: "CollectorExporterList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListCollectorExporter(ctx, &venterprisev1.ListCollectorExporterOptions{Common: common})
		}},
		{name: "enterprise/Secret", api: "enterprise/v1", kind: "SecretList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListSecret(ctx, &venterprisev1.ListSecretOptions{Common: common})
		}},
		{name: "enterprise/SecretStore", api: "enterprise/v1", kind: "SecretStoreList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListSecretStore(ctx, &venterprisev1.ListSecretStoreOptions{Common: common})
		}},
		{name: "enterprise/Certificate", api: "enterprise/v1", kind: "CertificateList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListCertificate(ctx, &venterprisev1.ListCertificateOptions{Common: common})
		}},
		{name: "enterprise/CertificateIssuer", api: "enterprise/v1", kind: "CertificateIssuerList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListCertificateIssuer(ctx, &venterprisev1.ListCertificateIssuerOptions{Common: common})
		}},
		{name: "enterprise/DNSProvider", api: "enterprise/v1", kind: "DNSProviderList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListDNSProvider(ctx, &venterprisev1.ListDNSProviderOptions{Common: common})
		}},
		{name: "enterprise/DirectoryProvider", api: "enterprise/v1", kind: "DirectoryProviderList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListDirectoryProvider(ctx, &venterprisev1.ListDirectoryProviderOptions{Common: common})
		}},
		{name: "enterprise/DirectoryProviderUser", api: "enterprise/v1", kind: "DirectoryProviderUserList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListDirectoryProviderUser(ctx, &venterprisev1.ListDirectoryProviderUserOptions{Common: common})
		}},
		{name: "enterprise/DirectoryProviderGroup", api: "enterprise/v1", kind: "DirectoryProviderGroupList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListDirectoryProviderGroup(ctx, &venterprisev1.ListDirectoryProviderGroupOptions{Common: common})
		}},
		{name: "enterprise/DeviceManager", api: "enterprise/v1", kind: "DeviceManagerList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return enterprise.ListDeviceManager(ctx, &venterprisev1.ListDeviceManagerOptions{Common: common})
		}},
		{name: "access/Policy", api: "access/v1", kind: "PolicyList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return access.ListPolicy(ctx, &vaccessv1.ListPolicyOptions{Common: common})
		}},
		{name: "access/Catalog", api: "access/v1", kind: "CatalogList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return access.ListCatalog(ctx, &vaccessv1.ListCatalogOptions{Common: common})
		}},
		{name: "access/Request", api: "access/v1", kind: "RequestList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return access.ListRequest(ctx, &vaccessv1.ListRequestOptions{Common: common})
		}},
		{name: "access/Review", api: "access/v1", kind: "ReviewList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return access.ListReview(ctx, &vaccessv1.ListReviewOptions{Common: common})
		}},
		{name: "access/Secret", api: "access/v1", kind: "SecretList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return access.ListSecret(ctx, &vaccessv1.ListSecretOptions{Common: common})
		}},
		{name: "access/Integration", api: "access/v1", kind: "IntegrationList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return access.ListIntegration(ctx, &vaccessv1.ListIntegrationOptions{Common: common})
		}},
		{name: "access/IntegrationIdentity", api: "access/v1", kind: "IntegrationIdentityList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return access.ListIntegrationIdentity(ctx, &vaccessv1.ListIntegrationIdentityOptions{Common: common})
		}},
		{name: "access/IntegrationBinding", api: "access/v1", kind: "IntegrationBindingList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return access.ListIntegrationBinding(ctx, &vaccessv1.ListIntegrationBindingOptions{Common: common})
		}},
		{name: "cordium/Workspace", api: "cordium/v1", kind: "WorkspaceList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return cordium.ListWorkspace(ctx, &vcordiumv1.ListWorkspaceOptions{Common: common})
		}},
		{name: "cordium/Template", api: "cordium/v1", kind: "TemplateList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return cordium.ListTemplate(ctx, &vcordiumv1.ListTemplateOptions{Common: common})
		}},
		{name: "cordium/Space", api: "cordium/v1", kind: "SpaceList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return cordium.ListSpace(ctx, &vcordiumv1.ListSpaceOptions{Common: common})
		}},
		{name: "cordium/Membership", api: "cordium/v1", kind: "MembershipList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return cordium.ListMembership(ctx, &vcordiumv1.ListMembershipOptions{Common: common})
		}},
		{name: "cordium/GitProvider", api: "cordium/v1", kind: "GitProviderList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return cordium.ListGitProvider(ctx, &vcordiumv1.ListGitProviderOptions{Common: common})
		}},
		{name: "cordium/Secret", api: "cordium/v1", kind: "SecretList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return cordium.ListSecret(ctx, &vcordiumv1.ListSecretOptions{Common: common})
		}},
		{name: "cordium/UserSecret", api: "cordium/v1", kind: "UserSecretList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return cordium.ListUserSecret(ctx, &vcordiumv1.ListUserSecretOptions{Common: common})
		}},
		{name: "cordium/Region", api: "cordium/v1", kind: "RegionList", call: func(ctx context.Context, common *vmetav1.CommonListOptions) (proto.Message, error) {
			return cordium.ListRegion(ctx, &vcordiumv1.ListRegionOptions{Common: common})
		}},
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.call(ctx, nil)
			require.Nil(t, err)
			msg := res.ProtoReflect()
			fields := msg.Descriptor().Fields()
			assert.Equal(t, tt.api, msg.Get(fields.ByName("apiVersion")).String())
			assert.Equal(t, tt.kind, msg.Get(fields.ByName("kind")).String())
			assert.Zero(t, msg.Get(fields.ByName("items")).List().Len())
			meta := msg.Get(fields.ByName("listResponseMeta")).Message()
			require.True(t, meta.IsValid())
			assert.Zero(t, meta.Get(meta.Descriptor().Fields().ByName("totalCount")).Uint())
			assert.Equal(t, uint64(defaultItemsPerPage), meta.Get(meta.Descriptor().Fields().ByName("itemsPerPage")).Uint())
			_, err = tt.call(ctx, &vmetav1.CommonListOptions{Page: 1})
			assert.Equal(t, codes.NotFound, status.Code(err))
			_, err = tt.call(ctx, &vmetav1.CommonListOptions{From: &timestamppb.Timestamp{Nanos: -1}})
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
	require.Nil(t, env.srv.db.Close())
	for _, tt := range cases {
		_, err := tt.call(ctx, nil)
		assert.Equal(t, codes.Internal, status.Code(err), "%s: %+v", tt.name, err)
	}
}
