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
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/tests/tstuser"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestCoreUser(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	/*
		rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
			Name: vutils.GetMyRegionName(),
		})
		assert.Nil(t, err)
	*/

	for range utilrand.GetRandomRangeMath(100, 400) {
		rsc := &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata: &metav1.Metadata{
				Name:            utilrand.GetRandomStringCanonical(8),
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
				CreatedAt:       pbutils.Timestamp(time.Now().UTC().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 500) * int(time.Minute)))),
			},
			Spec: &corev1.User_Spec{
				Type: func() corev1.User_Spec_Type {
					if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
						return corev1.User_Spec_HUMAN
					}
					return corev1.User_Spec_WORKLOAD
				}(),
			},
		}

		err = srv.insertResource(ctx, rsc)
		assert.Nil(t, err)

		err = srv.insertResource(ctx, rsc)
		assert.Nil(t, err)
	}

	_, err = srv.getSummaryCoreUser(ctx, &vcorev1.GetUserSummaryRequest{})
	assert.Nil(t, err)

}

func TestCoreSession(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	/*
		rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
			Name: vutils.GetMyRegionName(),
		})
		assert.Nil(t, err)
	*/

	coreSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  srv.octeliumC,
		IsEmbedded: true,
	})

	// uid := vutils.UUIDv4()
	for range 100 {
		usrT, err := tstuser.NewUserWithType(tst.C.OcteliumC, coreSrv, nil, nil,
			corev1.User_Spec_HUMAN,
			func() corev1.Session_Status_Type {
				if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
					return corev1.Session_Status_CLIENT
				}
				return corev1.Session_Status_CLIENTLESS
			}())
		assert.Nil(t, err)

		/*
			rsc := &corev1.Session{
				ApiVersion: ucorev1.APIVersion,
				Kind:       ucorev1.KindSession,
				Metadata: &metav1.Metadata{
					Name:            utilrand.GetRandomStringCanonical(8),
					Uid:             vutils.UUIDv4(),
					ResourceVersion: vutils.UUIDv7(),
					CreatedAt:       pbutils.Timestamp(time.Now().UTC().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 500) * int(time.Minute)))),
				},
				Spec: &corev1.Session_Spec{
					State: func() corev1.Session_Spec_State {
						if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
							return corev1.Session_Spec_ACTIVE
						}
						return corev1.Session_Spec_REJECTED
					}(),
				},
				Status: &corev1.Session_Status{
					Type: func() corev1.Session_Status_Type {
						if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
							return corev1.Session_Status_CLIENT
						}
						return corev1.Session_Status_CLIENTLESS
					}(),

					IsConnected: getRandomBool(),
					UserRef: &metav1.ObjectReference{
						Uid: uid,
					},
				},
			}
		*/

		err = srv.insertResource(ctx, usrT.Session)
		assert.Nil(t, err)
	}

	{
		_, err := srv.getSummaryCoreSession(ctx, &vcorev1.GetSessionSummaryRequest{})
		assert.Nil(t, err)
	}

	sessList, err := srv.octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{
		OrderBy: []*rmetav1.ListOptions_OrderBy{
			{
				Type: rmetav1.ListOptions_OrderBy_TYPE_CREATED_AT,
				Mode: rmetav1.ListOptions_OrderBy_MODE_DESC,
			},
		},
	})
	assert.Nil(t, err)
	sess := sessList.Items[0]

	sessI, err := srv.getSessionFromActorRef(ctx, umetav1.GetObjectReference(sess))
	assert.Nil(t, err)
	assert.True(t, pbutils.IsEqual(sess, sessI))

}

func TestCorePolicy(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	err = srv.initDB(ctx)
	assert.Nil(t, err)

	/*
		rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
			Name: vutils.GetMyRegionName(),
		})
		assert.Nil(t, err)
	*/

	// uid := vutils.UUIDv4()
	for range utilrand.GetRandomRangeMath(1000, 4000) {
		rsc := &corev1.Policy{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindPolicy,
			Metadata: &metav1.Metadata{
				Name:            utilrand.GetRandomStringCanonical(8),
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
				CreatedAt:       pbutils.Timestamp(time.Now().UTC().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 500) * int(time.Minute)))),
			},
			Spec: &corev1.Policy_Spec{
				IsDisabled: getRandomBool(),
				Rules: []*corev1.Policy_Spec_Rule{
					{
						Effect: corev1.Policy_Spec_Rule_ALLOW,
						Condition: &corev1.Condition{
							Type: &corev1.Condition_MatchAny{
								MatchAny: true,
							},
						},
					},
					{
						Effect: corev1.Policy_Spec_Rule_DENY,
						Condition: &corev1.Condition{
							Type: &corev1.Condition_MatchAny{
								MatchAny: true,
							},
						},
					},
				},
			},
			Status: &corev1.Policy_Status{},
		}

		err = srv.insertResource(ctx, rsc)
		assert.Nil(t, err)
	}

	{
		_, err := srv.getSummaryCorePolicy(ctx, &vcorev1.GetPolicySummaryRequest{})
		assert.Nil(t, err)
	}
}

func getRandomBool() bool {
	return utilrand.GetRandomRangeMath(1, 500)%2 == 0
}

func TestCoreSummaryCountsComprehensive(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	user1 := "user-1"
	user2 := "user-2"
	device1 := "device-1"

	insertRawRscStoreResource(t, env, ucorev1.KindUser, newRawRscStoreResource(ucorev1.KindUser, "human-enabled", false,
		map[string]any{"type": "HUMAN", "isDisabled": false}, nil))
	insertRawRscStoreResource(t, env, ucorev1.KindUser, newRawRscStoreResource(ucorev1.KindUser, "human-disabled", false,
		map[string]any{"type": "HUMAN", "isDisabled": true}, nil))
	insertRawRscStoreResource(t, env, ucorev1.KindUser, newRawRscStoreResource(ucorev1.KindUser, "workload", false,
		map[string]any{"type": "WORKLOAD", "isDisabled": false}, nil))
	insertRawRscStoreResource(t, env, ucorev1.KindUser, newRawRscStoreResource(ucorev1.KindUser, "hidden-user", true,
		map[string]any{"type": "HUMAN", "isDisabled": true}, nil))

	insertRawRscStoreResource(t, env, ucorev1.KindSession, newRawRscStoreResource(ucorev1.KindSession, "client-active", false,
		map[string]any{"state": "ACTIVE"},
		map[string]any{"type": "CLIENT", "isConnected": true, "userRef": map[string]any{"uid": user1}, "deviceRef": map[string]any{"uid": device1}}))
	insertRawRscStoreResource(t, env, ucorev1.KindSession, newRawRscStoreResource(ucorev1.KindSession, "browser-active", false,
		map[string]any{"state": "ACTIVE"},
		map[string]any{"type": "CLIENTLESS", "isBrowser": true, "userRef": map[string]any{"uid": user1}}))
	insertRawRscStoreResource(t, env, ucorev1.KindSession, newRawRscStoreResource(ucorev1.KindSession, "clientless-pending", false,
		map[string]any{"state": "PENDING"},
		map[string]any{"type": "CLIENTLESS", "userRef": map[string]any{"uid": user2}}))
	insertRawRscStoreResource(t, env, ucorev1.KindSession, newRawRscStoreResource(ucorev1.KindSession, "client-rejected", false,
		map[string]any{"state": "REJECTED"},
		map[string]any{"type": "CLIENT", "userRef": map[string]any{"uid": user2}}))

	serviceModes := []string{"TCP", "UDP", "HTTP", "SSH", "KUBERNETES", "POSTGRES", "MYSQL", "DNS", "GRPC", "WEB", "SOCKS5", "RDP_WEB", "MCP", "LLM"}
	for idx, mode := range serviceModes {
		insertRawRscStoreResource(t, env, ucorev1.KindService, newRawRscStoreResource(ucorev1.KindService,
			fmt.Sprintf("service-%02d", idx), false,
			map[string]any{"mode": mode, "isPublic": idx < 3, "isAnonymous": idx == 0, "isDisabled": idx == 1}, nil))
	}
	insertRawRscStoreResource(t, env, ucorev1.KindService, newRawRscStoreResource(ucorev1.KindService, "hidden-service", true,
		map[string]any{"mode": "HTTP", "isPublic": true, "isAnonymous": true, "isDisabled": true}, nil))

	insertRawRscStoreResource(t, env, ucorev1.KindPolicy, newRawRscStoreResource(ucorev1.KindPolicy, "policy-one", false,
		map[string]any{
			"isDisabled": false,
			"rules": []any{
				map[string]any{"effect": "ALLOW"},
				map[string]any{"effect": "ALLOW"},
				map[string]any{"effect": "DENY"},
			},
		}, nil))
	insertRawRscStoreResource(t, env, ucorev1.KindPolicy, newRawRscStoreResource(ucorev1.KindPolicy, "policy-two", false,
		map[string]any{
			"isDisabled": true,
			"rules": []any{
				map[string]any{"effect": "DENY"},
			},
		}, nil))

	credentialTypes := []string{"AUTH_TOKEN", "OAUTH2", "ACCESS_TOKEN"}
	for idx, credentialType := range credentialTypes {
		userUID := user1
		if idx == 2 {
			userUID = user2
		}
		insertRawRscStoreResource(t, env, ucorev1.KindCredential, newRawRscStoreResource(ucorev1.KindCredential,
			fmt.Sprintf("credential-%d", idx), false,
			map[string]any{"type": credentialType, "isDisabled": idx == 1},
			map[string]any{"userRef": map[string]any{"uid": userUID}}))
	}

	idpTypes := []string{"GITHUB", "OIDC", "SAML", "OIDC_IDENTITY_TOKEN"}
	for idx, idpType := range idpTypes {
		insertRawRscStoreResource(t, env, ucorev1.KindIdentityProvider, newRawRscStoreResource(ucorev1.KindIdentityProvider,
			fmt.Sprintf("idp-%d", idx), false,
			map[string]any{"isDisabled": idx == 2},
			map[string]any{"type": idpType}))
	}

	deviceOSTypes := []string{"LINUX", "WINDOWS", "MAC", "ANDROID", "IOS"}
	deviceStates := []string{"ACTIVE", "PENDING", "REJECTED", "ACTIVE", "ACTIVE"}
	for idx, osType := range deviceOSTypes {
		userUID := user1
		if idx >= 3 {
			userUID = user2
		}
		insertRawRscStoreResource(t, env, ucorev1.KindDevice, newRawRscStoreResource(ucorev1.KindDevice,
			fmt.Sprintf("device-%d", idx), false,
			map[string]any{"state": deviceStates[idx]},
			map[string]any{"osType": osType, "userRef": map[string]any{"uid": userUID}}))
	}

	insertRawRscStoreResource(t, env, ucorev1.KindAuthenticator, newRawRscStoreResource(ucorev1.KindAuthenticator, "fido-platform", false,
		map[string]any{"state": "ACTIVE"},
		map[string]any{
			"type":      "FIDO",
			"userRef":   map[string]any{"uid": user1},
			"deviceRef": map[string]any{"uid": device1},
			"info": map[string]any{
				"fido": map[string]any{"type": "PLATFORM", "isPasskey": true, "isHardware": true},
			},
		}))
	insertRawRscStoreResource(t, env, ucorev1.KindAuthenticator, newRawRscStoreResource(ucorev1.KindAuthenticator, "fido-roaming", false,
		map[string]any{"state": "PENDING"},
		map[string]any{
			"type":    "FIDO",
			"userRef": map[string]any{"uid": user2},
			"info": map[string]any{
				"fido": map[string]any{"type": "ROAMING"},
			},
		}))
	insertRawRscStoreResource(t, env, ucorev1.KindAuthenticator, newRawRscStoreResource(ucorev1.KindAuthenticator, "totp", false,
		map[string]any{"state": "REJECTED"}, map[string]any{"type": "TOTP", "userRef": map[string]any{"uid": user2}}))
	insertRawRscStoreResource(t, env, ucorev1.KindAuthenticator, newRawRscStoreResource(ucorev1.KindAuthenticator, "tpm", false,
		map[string]any{"state": "ACTIVE"}, map[string]any{"type": "TPM", "userRef": map[string]any{"uid": user1}}))

	for _, kind := range []string{ucorev1.KindGroup, ucorev1.KindGateway, ucorev1.KindRegion, ucorev1.KindSecret, ucorev1.KindNamespace} {
		insertRawRscStoreResource(t, env, kind, newRawRscStoreResource(kind, kind+"-visible", false, nil, nil))
		insertRawRscStoreResource(t, env, kind, newRawRscStoreResource(kind, kind+"-hidden", true, nil, nil))
	}

	{
		resp, err := env.srv.getSummaryCoreUser(env.ctx, &vcorev1.GetUserSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(3), resp.TotalNumber)
		assert.Equal(t, uint32(2), resp.TotalHuman)
		assert.Equal(t, uint32(1), resp.TotalWorkload)
		assert.Equal(t, uint32(1), resp.TotalDisabled)
	}

	{
		resp, err := env.srv.getSummaryCoreSession(env.ctx, &vcorev1.GetSessionSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(4), resp.TotalNumber)
		assert.Equal(t, uint32(2), resp.TotalClient)
		assert.Equal(t, uint32(2), resp.TotalClientless)
		assert.Equal(t, uint32(1), resp.TotalConnected)
		assert.Equal(t, uint32(2), resp.TotalUser)
		assert.Equal(t, uint32(1), resp.TotalDevice)
		assert.Equal(t, uint32(1), resp.TotalClientlessBrowser)
		assert.Equal(t, uint32(2), resp.TotalActive)
		assert.Equal(t, uint32(1), resp.TotalPending)
		assert.Equal(t, uint32(1), resp.TotalRejected)
	}

	{
		resp, err := env.srv.getSummaryCoreService(env.ctx, &vcorev1.GetServiceSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(14), resp.TotalNumber)
		assert.Equal(t, uint32(3), resp.TotalPublic)
		assert.Equal(t, uint32(1), resp.TotalAnonymous)
		assert.Equal(t, uint32(1), resp.TotalTCP)
		assert.Equal(t, uint32(1), resp.TotalUDP)
		assert.Equal(t, uint32(1), resp.TotalHTTP)
		assert.Equal(t, uint32(1), resp.TotalSSH)
		assert.Equal(t, uint32(1), resp.TotalKubernetes)
		assert.Equal(t, uint32(1), resp.TotalPostgres)
		assert.Equal(t, uint32(1), resp.TotalMysql)
		assert.Equal(t, uint32(1), resp.TotalDNS)
		assert.Equal(t, uint32(1), resp.TotalGRPC)
		assert.Equal(t, uint32(1), resp.TotalWeb)
		assert.Equal(t, uint32(1), resp.TotalSOCKS5)
		assert.Equal(t, uint32(1), resp.TotalRDPWeb)
		assert.Equal(t, uint32(1), resp.TotalMCP)
		assert.Equal(t, uint32(1), resp.TotalLLM)
		assert.Equal(t, uint32(1), resp.TotalDisabled)
	}

	{
		resp, err := env.srv.getSummaryCorePolicy(env.ctx, &vcorev1.GetPolicySummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(2), resp.TotalNumber)
		assert.Equal(t, uint32(1), resp.TotalDisabled)
		assert.Equal(t, uint32(4), resp.TotalRule)
		assert.Equal(t, uint32(2), resp.TotalRuleAllow)
		assert.Equal(t, uint32(2), resp.TotalRuleDenied)
	}

	{
		resp, err := env.srv.getSummaryCoreCredential(env.ctx, &vcorev1.GetCredentialSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(3), resp.TotalNumber)
		assert.Equal(t, uint32(2), resp.TotalUser)
		assert.Equal(t, uint32(1), resp.TotalDisabled)
		assert.Equal(t, uint32(1), resp.TotalAuthenticationToken)
		assert.Equal(t, uint32(1), resp.TotalOAuth2)
		assert.Equal(t, uint32(1), resp.TotalAccessToken)
	}

	{
		resp, err := env.srv.getSummaryCoreIdentityProvider(env.ctx, &vcorev1.GetIdentityProviderSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(4), resp.TotalNumber)
		assert.Equal(t, uint32(1), resp.TotalDisabled)
		assert.Equal(t, uint32(1), resp.TotalGithub)
		assert.Equal(t, uint32(1), resp.TotalOIDC)
		assert.Equal(t, uint32(1), resp.TotalSAML)
		assert.Equal(t, uint32(1), resp.TotalOIDCIdentityToken)
	}

	{
		resp, err := env.srv.getSummaryCoreDevice(env.ctx, &vcorev1.GetDeviceSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(5), resp.TotalNumber)
		assert.Equal(t, uint32(2), resp.TotalUser)
		assert.Equal(t, uint32(3), resp.TotalActive)
		assert.Equal(t, uint32(1), resp.TotalPending)
		assert.Equal(t, uint32(1), resp.TotalRejected)
		assert.Equal(t, uint32(1), resp.TotalLinux)
		assert.Equal(t, uint32(1), resp.TotalWindows)
		assert.Equal(t, uint32(1), resp.TotalMac)
		assert.Equal(t, uint32(1), resp.TotalAndroid)
		assert.Equal(t, uint32(1), resp.TotalIOS)
	}

	{
		resp, err := env.srv.getSummaryCoreAuthenticator(env.ctx, &vcorev1.GetAuthenticatorSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(4), resp.TotalNumber)
		assert.Equal(t, uint32(2), resp.TotalFIDO)
		assert.Equal(t, uint32(1), resp.TotalTOTP)
		assert.Equal(t, uint32(1), resp.TotalTPM)
		assert.Equal(t, uint32(2), resp.TotalUser)
		assert.Equal(t, uint32(1), resp.TotalDevice)
		assert.Equal(t, uint32(2), resp.TotalActive)
		assert.Equal(t, uint32(1), resp.TotalPending)
		assert.Equal(t, uint32(1), resp.TotalRejected)
		assert.Equal(t, uint32(1), resp.TotalFIDOPlatform)
		assert.Equal(t, uint32(1), resp.TotalFIDORoaming)
		assert.Equal(t, uint32(1), resp.TotalFIDOIsPasskey)
		assert.Equal(t, uint32(1), resp.TotalFIDOIsHardware)
	}

	{
		resp, err := env.srv.getSummaryCoreGroup(env.ctx, &vcorev1.GetGroupSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(1), resp.TotalNumber)
	}

	{
		resp, err := env.srv.getSummaryCoreGateway(env.ctx, &vcorev1.GetGatewaySummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(1), resp.TotalNumber)
	}

	{
		resp, err := env.srv.getSummaryCoreRegion(env.ctx, &vcorev1.GetRegionSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(1), resp.TotalNumber)
	}

	{
		resp, err := env.srv.getSummaryCoreSecret(env.ctx, &vcorev1.GetSecretSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(1), resp.TotalNumber)
	}

	{
		resp, err := env.srv.getSummaryCoreNamespace(env.ctx, &vcorev1.GetNamespaceSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(1), resp.TotalNumber)
	}
}
