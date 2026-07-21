// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestPolicyToCoreCondition(t *testing.T) {
	srv := &Server{}

	{
		assert.Nil(t, srv.toCoreCondition(nil))
		assert.Nil(t, srv.toCoreCondition(&enterprisev1.Condition{}))
	}

	{
		cond := srv.toCoreCondition(&enterprisev1.Condition{
			Type: &enterprisev1.Condition_MatchAny{
				MatchAny: true,
			},
		})
		assert.NotNil(t, cond)
		assert.True(t, cond.GetMatchAny())
	}

	{
		cond := srv.toCoreCondition(policyExprCond(policyRequestHTTPPathExact("/api")))
		assert.NotNil(t, cond)
		assert.Equal(t, `ctx.request.http.path == "/api"`, cond.GetMatch())
	}

	{
		cond := srv.toCoreCondition(&enterprisev1.Condition{
			Type: &enterprisev1.Condition_All_{
				All: &enterprisev1.Condition_All{
					Of: []*enterprisev1.Condition{
						policyExprCond(policyRequestHTTPPathExact("/api")),
						policyExprCond(policyRequestHTTPMethod("GET")),
					},
				},
			},
		})
		assert.NotNil(t, cond)
		assert.NotNil(t, cond.GetAll())
		assert.Len(t, cond.GetAll().Of, 2)
		assert.Equal(t, `ctx.request.http.path == "/api"`, cond.GetAll().Of[0].GetMatch())
		assert.Equal(t, `ctx.request.http.method == "GET"`, cond.GetAll().Of[1].GetMatch())
	}

	{
		cond := srv.toCoreCondition(&enterprisev1.Condition{
			Type: &enterprisev1.Condition_Any_{
				Any: &enterprisev1.Condition_Any{
					Of: []*enterprisev1.Condition{
						policyExprCond(policyRequestHTTPPathExact("/api")),
						policyExprCond(policyRequestHTTPPathExact("/healthz")),
					},
				},
			},
		})
		assert.NotNil(t, cond)
		assert.NotNil(t, cond.GetAny())
		assert.Len(t, cond.GetAny().Of, 2)
		assert.Equal(t, `ctx.request.http.path == "/api"`, cond.GetAny().Of[0].GetMatch())
		assert.Equal(t, `ctx.request.http.path == "/healthz"`, cond.GetAny().Of[1].GetMatch())
	}

	{
		cond := srv.toCoreCondition(&enterprisev1.Condition{
			Type: &enterprisev1.Condition_None_{
				None: &enterprisev1.Condition_None{
					Of: []*enterprisev1.Condition{
						policyExprCond(policyRequestHTTPPathPrefix("/internal")),
						policyExprCond(policyRequestHTTPPathPrefix("/admin")),
					},
				},
			},
		})
		assert.NotNil(t, cond)
		assert.NotNil(t, cond.GetNone())
		assert.Len(t, cond.GetNone().Of, 2)
		assert.Equal(t, `ctx.request.http.path.startsWith("/internal")`, cond.GetNone().Of[0].GetMatch())
		assert.Equal(t, `ctx.request.http.path.startsWith("/admin")`, cond.GetNone().Of[1].GetMatch())
	}

	{
		cond := srv.toCoreCondition(&enterprisev1.Condition{
			Type: &enterprisev1.Condition_Not_{
				Not: &enterprisev1.Condition_Not{
					Condition: policyExprCond(policyRequestHTTPPathPrefix("/internal")),
				},
			},
		})
		assert.NotNil(t, cond)
		assert.Equal(t, `ctx.request.http.path.startsWith("/internal")`, cond.GetNot())
	}
}

func TestPolicyToCoreConditionCompositeNot(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}

	{
		cond, err := srv.GetCoreCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_Not_{
				Not: &enterprisev1.Condition_Not{
					Condition: &enterprisev1.Condition{
						Type: &enterprisev1.Condition_Any_{
							Any: &enterprisev1.Condition_Any{
								Of: []*enterprisev1.Condition{
									policyExprCond(policyRequestHTTPPathExact("/admin")),
									policyExprCond(policyRequestHTTPPathExact("/internal")),
								},
							},
						},
					},
				},
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, cond)
		assert.NotNil(t, cond.GetNone())
		assert.Len(t, cond.GetNone().Of, 2)
		assert.Equal(t, `ctx.request.http.path == "/admin"`, cond.GetNone().Of[0].GetMatch())
		assert.Equal(t, `ctx.request.http.path == "/internal"`, cond.GetNone().Of[1].GetMatch())
	}

	{
		cond, err := srv.GetCoreCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_Not_{
				Not: &enterprisev1.Condition_Not{
					Condition: &enterprisev1.Condition{
						Type: &enterprisev1.Condition_All_{
							All: &enterprisev1.Condition_All{
								Of: []*enterprisev1.Condition{
									policyExprCond(policyRequestHTTPPathPrefix("/api")),
									policyExprCond(policyRequestHTTPMethod("POST")),
								},
							},
						},
					},
				},
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, cond)
		assert.NotNil(t, cond.GetAny())
		assert.Len(t, cond.GetAny().Of, 2)
		assert.Equal(t, `ctx.request.http.path.startsWith("/api")`, cond.GetAny().Of[0].GetNot())
		assert.Equal(t, `ctx.request.http.method == "POST"`, cond.GetAny().Of[1].GetNot())
	}

	{
		cond, err := srv.GetCoreCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_Not_{
				Not: &enterprisev1.Condition_Not{
					Condition: &enterprisev1.Condition{
						Type: &enterprisev1.Condition_None_{
							None: &enterprisev1.Condition_None{
								Of: []*enterprisev1.Condition{
									policyExprCond(policyRequestHTTPPathExact("/metrics")),
									policyExprCond(policyRequestHTTPPathExact("/debug")),
								},
							},
						},
					},
				},
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, cond)
		assert.NotNil(t, cond.GetAny())
		assert.Len(t, cond.GetAny().Of, 2)
		assert.Equal(t, `ctx.request.http.path == "/metrics"`, cond.GetAny().Of[0].GetMatch())
		assert.Equal(t, `ctx.request.http.path == "/debug"`, cond.GetAny().Of[1].GetMatch())
	}
}

func TestPolicyGetExpression(t *testing.T) {
	srv := &Server{}

	ts := timestamppb.New(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC))
	apiServerExpr := `ctx.service.spec.mode == "GRPC" && ctx.service.status.namespaceRef.name == "octelium-api"`

	tcs := []struct {
		name string
		expr *enterprisev1.Condition_Expression
		ret  string
	}{
		{
			name: "User",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_User_{
					User: &enterprisev1.Condition_Expression_User{
						UserRef: policyNameRef("usr"),
					},
				},
			},
			ret: `ctx.user.metadata.name == "usr"`,
		},
		{
			name: "Device",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Device_{
					Device: &enterprisev1.Condition_Expression_Device{
						DeviceRef: policyNameRef("dev"),
					},
				},
			},
			ret: `ctx.device.metadata.name == "dev"`,
		},
		{
			name: "Session",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Session_{
					Session: &enterprisev1.Condition_Expression_Session{
						SessionRef: policyNameRef("sess"),
					},
				},
			},
			ret: `ctx.session.metadata.name == "sess"`,
		},
		{
			name: "Service",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Service_{
					Service: &enterprisev1.Condition_Expression_Service{
						ServiceRef: policyNameRef("svc"),
					},
				},
			},
			ret: `ctx.service.metadata.name == "svc"`,
		},
		{
			name: "Namespace",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Namespace_{
					Namespace: &enterprisev1.Condition_Expression_Namespace{
						NamespaceRef: policyNameRef("ns"),
					},
				},
			},
			ret: `ctx.namespace.metadata.name == "ns"`,
		},
		{
			name: "Group",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Group_{
					Group: &enterprisev1.Condition_Expression_Group{
						GroupRef: policyNameRef("devs"),
					},
				},
			},
			ret: `"devs" in ctx.user.spec.groups`,
		},
		{
			name: "UserType",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_UserType_{
					UserType: &enterprisev1.Condition_Expression_UserType{
						Type: corev1.User_Spec_HUMAN,
					},
				},
			},
			ret: `ctx.user.spec.type == "HUMAN"`,
		},
		{
			name: "SessionType",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionType_{
					SessionType: &enterprisev1.Condition_Expression_SessionType{
						Type: corev1.Session_Status_CLIENT,
					},
				},
			},
			ret: `ctx.session.status.type == "CLIENT"`,
		},
		{
			name: "SessionBrowserTrue",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionBrowser_{
					SessionBrowser: &enterprisev1.Condition_Expression_SessionBrowser{
						IsBrowser: true,
					},
				},
			},
			ret: `ctx.session.status.isBrowser`,
		},
		{
			name: "SessionBrowserFalse",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionBrowser_{
					SessionBrowser: &enterprisev1.Condition_Expression_SessionBrowser{
						IsBrowser: false,
					},
				},
			},
			ret: `!ctx.session.status.isBrowser`,
		},
		{
			name: "SessionAuthenticationType",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationType_{
					SessionAuthenticationType: &enterprisev1.Condition_Expression_SessionAuthenticationType{
						Type: corev1.Session_Status_Authentication_Info_AUTHENTICATOR,
					},
				},
			},
			ret: `ctx.session.status.authentication.info.type == "AUTHENTICATOR"`,
		},
		{
			name: "SessionAuthenticationAAL",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationAAL_{
					SessionAuthenticationAAL: &enterprisev1.Condition_Expression_SessionAuthenticationAAL{
						Aal: corev1.Session_Status_Authentication_Info_AAL2,
					},
				},
			},
			ret: `ctx.session.status.authentication.info.aal == "AAL2"`,
		},
		{
			name: "SessionAuthenticationIdentityProvider",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider_{
					SessionAuthenticationIdentityProvider: &enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider{
						IdentityProviderRef: policyUIDRef("idp-uid"),
					},
				},
			},
			ret: `ctx.session.status.authentication.info.identityProvider.identityProviderRef.uid == "idp-uid"`,
		},
		{
			name: "SessionAuthenticationCredential",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredential_{
					SessionAuthenticationCredential: &enterprisev1.Condition_Expression_SessionAuthenticationCredential{
						CredentialRef: policyUIDRef("cred-uid"),
					},
				},
			},
			ret: `ctx.session.status.authentication.info.credential.credentialRef.uid == "cred-uid"`,
		},
		{
			name: "SessionAuthenticationCredentialType",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType_{
					SessionAuthenticationCredentialType: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType{
						Type: corev1.Credential_Spec_AUTH_TOKEN,
					},
				},
			},
			ret: `ctx.session.status.authentication.info.credential.type == "AUTH_TOKEN"`,
		},
		{
			name: "SessionAuthenticationGeoipCountryCode",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_{
					SessionAuthenticationGeoipCountryCode: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode{
						Code: "US",
					},
				},
			},
			ret: `ctx.session.status.authentication.info.geoip.country.code == "US"`,
		},
		{
			name: "FIDOAAGUID",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID_{
					SessionAuthenticationCredAuthenticatorAAGUID: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID{
						Aaguid: "00000000-0000-0000-0000-000000000000",
					},
				},
			},
			ret: `ctx.session.status.authentication.info.authenticator.info.fido.aaguid == "00000000-0000-0000-0000-000000000000"`,
		},
		{
			name: "FIDOPasskeyTrue",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey_{
					SessionAuthenticationCredAuthenticatorFIDOPasskey: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey{
						IsPasskey: true,
					},
				},
			},
			ret: `ctx.session.status.authentication.info.authenticator.info.fido.isPasskey`,
		},
		{
			name: "FIDOPasskeyFalse",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey_{
					SessionAuthenticationCredAuthenticatorFIDOPasskey: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey{
						IsPasskey: false,
					},
				},
			},
			ret: `!ctx.session.status.authentication.info.authenticator.info.fido.isPasskey`,
		},
		{
			name: "FIDOHardwareTrue",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware_{
					SessionAuthenticationCredAuthenticatorFIDOHardware: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware{
						IsHardware: true,
					},
				},
			},
			ret: `ctx.session.status.authentication.info.authenticator.info.fido.isHardware`,
		},
		{
			name: "FIDOHardwareFalse",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware_{
					SessionAuthenticationCredAuthenticatorFIDOHardware: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware{
						IsHardware: false,
					},
				},
			},
			ret: `!ctx.session.status.authentication.info.authenticator.info.fido.isHardware`,
		},
		{
			name: "FIDOAttestationVerifiedTrue",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified_{
					SessionAuthenticationCredAuthenticatorFIDOAttestationVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified{
						IsAttestationVerified: true,
					},
				},
			},
			ret: `ctx.session.status.authentication.info.authenticator.info.fido.isAttestationVerified`,
		},
		{
			name: "FIDOAttestationVerifiedFalse",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified_{
					SessionAuthenticationCredAuthenticatorFIDOAttestationVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified{
						IsAttestationVerified: false,
					},
				},
			},
			ret: `!ctx.session.status.authentication.info.authenticator.info.fido.isAttestationVerified`,
		},
		{
			name: "FIDOUserPresentTrue",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent_{
					SessionAuthenticationCredAuthenticatorFIDOUserPresent: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent{
						IsUserPresent: true,
					},
				},
			},
			ret: `ctx.session.status.authentication.info.authenticator.info.fido.userPresent`,
		},
		{
			name: "FIDOUserPresentFalse",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent_{
					SessionAuthenticationCredAuthenticatorFIDOUserPresent: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent{
						IsUserPresent: false,
					},
				},
			},
			ret: `!ctx.session.status.authentication.info.authenticator.info.fido.userPresent`,
		},
		{
			name: "FIDOUserVerifiedTrue",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified_{
					SessionAuthenticationCredAuthenticatorFIDOUserVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified{
						IsUserVerified: true,
					},
				},
			},
			ret: `ctx.session.status.authentication.info.authenticator.info.fido.userVerified`,
		},
		{
			name: "FIDOUserVerifiedFalse",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified_{
					SessionAuthenticationCredAuthenticatorFIDOUserVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified{
						IsUserVerified: false,
					},
				},
			},
			ret: `!ctx.session.status.authentication.info.authenticator.info.fido.userVerified`,
		},
		{
			name: "DeviceOSType",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_DeviceOSType_{
					DeviceOSType: &enterprisev1.Condition_Expression_DeviceOSType{
						OsType: corev1.Device_Status_LINUX,
					},
				},
			},
			ret: `ctx.device.status.osType == "LINUX"`,
		},
		{
			name: "ServiceMode",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_ServiceMode_{
					ServiceMode: &enterprisev1.Condition_Expression_ServiceMode{
						Mode: corev1.Service_Spec_HTTP,
					},
				},
			},
			ret: `ctx.service.spec.mode == "HTTP"`,
		},
		{
			name: "ServicePublicTrue",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_ServicePublic_{
					ServicePublic: &enterprisev1.Condition_Expression_ServicePublic{
						IsPublic: true,
					},
				},
			},
			ret: `ctx.service.spec.isPublic`,
		},
		{
			name: "ServicePublicFalse",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_ServicePublic_{
					ServicePublic: &enterprisev1.Condition_Expression_ServicePublic{
						IsPublic: false,
					},
				},
			},
			ret: `!ctx.service.spec.isPublic`,
		},
		{
			name: "TimeAfter",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_TimeAfter_{
					TimeAfter: &enterprisev1.Condition_Expression_TimeAfter{
						Timestamp: ts,
					},
				},
			},
			ret: `now() > timestamp("2026-01-02T03:04:05Z")`,
		},
		{
			name: "TimeBefore",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_TimeBefore_{
					TimeBefore: &enterprisev1.Condition_Expression_TimeBefore{
						Timestamp: ts,
					},
				},
			},
			ret: `now() < timestamp("2026-01-02T03:04:05Z")`,
		},
		{
			name: "RequestHTTPPathExact",
			expr: policyRequestHTTPPathExact("/api"),
			ret:  `ctx.request.http.path == "/api"`,
		},
		{
			name: "RequestHTTPPathPrefix",
			expr: policyRequestHTTPPathPrefix("/api"),
			ret:  `ctx.request.http.path.startsWith("/api")`,
		},
		{
			name: "RequestHTTPMethod",
			expr: policyRequestHTTPMethod("post"),
			ret:  `ctx.request.http.method == "POST"`,
		},
		{
			name: "RequestHTTPHasHeader",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestHTTPHasHeader_{
					RequestHTTPHasHeader: &enterprisev1.Condition_Expression_RequestHTTPHasHeader{
						Value: "X-Request-ID",
					},
				},
			},
			ret: `"x-request-id" in ctx.request.http.headers`,
		},
		{
			name: "RequestHTTPHeaderValue",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue_{
					RequestHTTPHeaderValue: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue{
						Header: "X-Forwarded-For",
						Value:  "10.0.0.1",
					},
				},
			},
			ret: `ctx.request.http.headers["x-forwarded-for"] == "10.0.0.1"`,
		},
		{
			name: "RequestIP",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestIP_{
					RequestIP: &enterprisev1.Condition_Expression_RequestIP{
						Value: "10.0.0.1",
					},
				},
			},
			ret: `ctx.request.ip == "10.0.0.1"`,
		},
		{
			name: "RequestIPInRange",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestIPInRange_{
					RequestIPInRange: &enterprisev1.Condition_Expression_RequestIPInRange{
						Value: "10.0.0.0/24",
					},
				},
			},
			ret: `net.isIPInRange(ctx.request.ip, "10.0.0.0/24")`,
		},
		{
			name: "APIServer",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_ApiServer{
					ApiServer: &enterprisev1.Condition_Expression_APIServer{
						IsAPIServer: true,
					},
				},
			},
			ret: apiServerExpr,
		},
		{
			name: "APIServerCore",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_ApiServerCore{
					ApiServerCore: &enterprisev1.Condition_Expression_APIServerCore{
						IsAPIServerCore: true,
					},
				},
			},
			ret: apiServerExpr + ` && ctx.request.grpc.package == "octelium.api.main.core.v1"`,
		},
		{
			name: "APIServerUser",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_ApiServerUser{
					ApiServerUser: &enterprisev1.Condition_Expression_APIServerUser{
						IsAPIServerUser: true,
					},
				},
			},
			ret: apiServerExpr + ` && ctx.request.grpc.package == "octelium.api.main.user.v1"`,
		},
		{
			name: "APIServerEnterprise",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_ApiServerEnterprise{
					ApiServerEnterprise: &enterprisev1.Condition_Expression_APIServerEnterprise{
						IsAPIServerEnterprise: true,
					},
				},
			},
			ret: apiServerExpr + ` && ctx.request.grpc.package == "octelium.api.main.enterprise.v1"`,
		},
		{
			name: "APIServerCordium",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_ApiServerCordium{
					ApiServerCordium: &enterprisev1.Condition_Expression_APIServerCordium{
						IsAPIServerCordium: true,
					},
				},
			},
			ret: apiServerExpr + ` && ctx.request.grpc.package == "octelium.api.main.cordium.v1"`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.ret, srv.getExpression(tc.expr))
			assert.NotEmpty(t, srv.getExpression(tc.expr))
		})
	}
}

func TestPolicyGetExpressionAPIServerFalse(t *testing.T) {
	srv := &Server{}

	tcs := []struct {
		name string
		t    func(bool) *enterprisev1.Condition_Expression
	}{
		{
			name: "APIServer",
			t: func(v bool) *enterprisev1.Condition_Expression {
				return &enterprisev1.Condition_Expression{
					Type: &enterprisev1.Condition_Expression_ApiServer{
						ApiServer: &enterprisev1.Condition_Expression_APIServer{
							IsAPIServer: v,
						},
					},
				}
			},
		},
		{
			name: "APIServerCore",
			t: func(v bool) *enterprisev1.Condition_Expression {
				return &enterprisev1.Condition_Expression{
					Type: &enterprisev1.Condition_Expression_ApiServerCore{
						ApiServerCore: &enterprisev1.Condition_Expression_APIServerCore{
							IsAPIServerCore: v,
						},
					},
				}
			},
		},
		{
			name: "APIServerUser",
			t: func(v bool) *enterprisev1.Condition_Expression {
				return &enterprisev1.Condition_Expression{
					Type: &enterprisev1.Condition_Expression_ApiServerUser{
						ApiServerUser: &enterprisev1.Condition_Expression_APIServerUser{
							IsAPIServerUser: v,
						},
					},
				}
			},
		},
		{
			name: "APIServerEnterprise",
			t: func(v bool) *enterprisev1.Condition_Expression {
				return &enterprisev1.Condition_Expression{
					Type: &enterprisev1.Condition_Expression_ApiServerEnterprise{
						ApiServerEnterprise: &enterprisev1.Condition_Expression_APIServerEnterprise{
							IsAPIServerEnterprise: v,
						},
					},
				}
			},
		},
		{
			name: "APIServerCordium",
			t: func(v bool) *enterprisev1.Condition_Expression {
				return &enterprisev1.Condition_Expression{
					Type: &enterprisev1.Condition_Expression_ApiServerCordium{
						ApiServerCordium: &enterprisev1.Condition_Expression_APIServerCordium{
							IsAPIServerCordium: v,
						},
					},
				}
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			trueExpr := srv.getExpression(tc.t(true))
			falseExpr := srv.getExpression(tc.t(false))
			assert.NotEmpty(t, trueExpr)
			assert.NotEmpty(t, falseExpr)
			assert.NotEqual(t, trueExpr, falseExpr)
			assert.Contains(t, falseExpr, "!")
		})
	}
}

func TestPolicyValidateConditionValid(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}

	err := srv.validateCondition(ctx, &enterprisev1.Condition{
		Type: &enterprisev1.Condition_All_{
			All: &enterprisev1.Condition_All{
				Of: []*enterprisev1.Condition{
					{
						Type: &enterprisev1.Condition_Any_{
							Any: &enterprisev1.Condition_Any{
								Of: []*enterprisev1.Condition{
									policyExprCond(policyRequestHTTPPathExact("/api")),
									policyExprCond(policyRequestHTTPMethod("GET")),
								},
							},
						},
					},
					{
						Type: &enterprisev1.Condition_None_{
							None: &enterprisev1.Condition_None{
								Of: []*enterprisev1.Condition{
									policyExprCond(policyRequestHTTPPathPrefix("/internal")),
								},
							},
						},
					},
					{
						Type: &enterprisev1.Condition_Not_{
							Not: &enterprisev1.Condition_Not{
								Condition: policyExprCond(policyRequestIPInRange("10.0.0.0/24")),
							},
						},
					},
					{
						Type: &enterprisev1.Condition_MatchAny{
							MatchAny: true,
						},
					},
				},
			},
		},
	})
	assert.Nil(t, err)
}

func TestPolicyValidateConditionInvalid(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}

	tcs := []struct {
		name string
		cond *enterprisev1.Condition
	}{
		{
			name: "Nil",
			cond: nil,
		},
		{
			name: "Empty",
			cond: &enterprisev1.Condition{},
		},
		{
			name: "AllEmpty",
			cond: &enterprisev1.Condition{
				Type: &enterprisev1.Condition_All_{
					All: &enterprisev1.Condition_All{},
				},
			},
		},
		{
			name: "AnyEmpty",
			cond: &enterprisev1.Condition{
				Type: &enterprisev1.Condition_Any_{
					Any: &enterprisev1.Condition_Any{},
				},
			},
		},
		{
			name: "NoneEmpty",
			cond: &enterprisev1.Condition{
				Type: &enterprisev1.Condition_None_{
					None: &enterprisev1.Condition_None{},
				},
			},
		},
		{
			name: "NotNil",
			cond: &enterprisev1.Condition{
				Type: &enterprisev1.Condition_Not_{
					Not: &enterprisev1.Condition_Not{},
				},
			},
		},
		{
			name: "AllTooManyChildren",
			cond: &enterprisev1.Condition{
				Type: &enterprisev1.Condition_All_{
					All: &enterprisev1.Condition_All{
						Of: policyMatchAnyConditions(maxConditionChildren + 1),
					},
				},
			},
		},
		{
			name: "AnyTooManyChildren",
			cond: &enterprisev1.Condition{
				Type: &enterprisev1.Condition_Any_{
					Any: &enterprisev1.Condition_Any{
						Of: policyMatchAnyConditions(maxConditionChildren + 1),
					},
				},
			},
		},
		{
			name: "NoneTooManyChildren",
			cond: &enterprisev1.Condition{
				Type: &enterprisev1.Condition_None_{
					None: &enterprisev1.Condition_None{
						Of: policyMatchAnyConditions(maxConditionChildren + 1),
					},
				},
			},
		},
		{
			name: "TooDeep",
			cond: policyNestedNot(maxConditionDepth + 1),
		},
		{
			name: "InvalidChild",
			cond: &enterprisev1.Condition{
				Type: &enterprisev1.Condition_All_{
					All: &enterprisev1.Condition_All{
						Of: []*enterprisev1.Condition{{}},
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := srv.validateCondition(ctx, tc.cond)
			assert.NotNil(t, err)
			assert.True(t, grpcerr.IsInvalidArg(err))
		})
	}
}

func TestPolicyValidateExpressionInvalid(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}
	longValue := strings.Repeat("a", maxConditionStringBytes+1)

	tcs := []struct {
		name string
		expr *enterprisev1.Condition_Expression
	}{
		{
			name: "Nil",
			expr: nil,
		},
		{
			name: "Empty",
			expr: &enterprisev1.Condition_Expression{},
		},
		{
			name: "UserNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_User_{},
			},
		},
		{
			name: "UserRefNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_User_{
					User: &enterprisev1.Condition_Expression_User{},
				},
			},
		},
		{
			name: "GroupRefNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Group_{
					Group: &enterprisev1.Condition_Expression_Group{},
				},
			},
		},
		{
			name: "DeviceRefNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Device_{
					Device: &enterprisev1.Condition_Expression_Device{},
				},
			},
		},
		{
			name: "SessionRefNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Session_{
					Session: &enterprisev1.Condition_Expression_Session{},
				},
			},
		},
		{
			name: "ServiceRefNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Service_{
					Service: &enterprisev1.Condition_Expression_Service{},
				},
			},
		},
		{
			name: "NamespaceRefNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_Namespace_{
					Namespace: &enterprisev1.Condition_Expression_Namespace{},
				},
			},
		},
		{
			name: "IdentityProviderRefNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider_{
					SessionAuthenticationIdentityProvider: &enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider{},
				},
			},
		},
		{
			name: "CredentialRefNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredential_{
					SessionAuthenticationCredential: &enterprisev1.Condition_Expression_SessionAuthenticationCredential{},
				},
			},
		},
		{
			name: "UserTypeUnknown",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_UserType_{
					UserType: &enterprisev1.Condition_Expression_UserType{
						Type: corev1.User_Spec_TYPE_UNKNOWN,
					},
				},
			},
		},
		{
			name: "SessionTypeUnknown",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionType_{
					SessionType: &enterprisev1.Condition_Expression_SessionType{
						Type: corev1.Session_Status_TYPE_UNKNOWN,
					},
				},
			},
		},
		{
			name: "SessionAuthenticationTypeUnset",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationType_{
					SessionAuthenticationType: &enterprisev1.Condition_Expression_SessionAuthenticationType{
						Type: corev1.Session_Status_Authentication_Info_TYPE_UNSET,
					},
				},
			},
		},
		{
			name: "SessionAuthenticationAALUnset",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationAAL_{
					SessionAuthenticationAAL: &enterprisev1.Condition_Expression_SessionAuthenticationAAL{
						Aal: corev1.Session_Status_Authentication_Info_AAL_UNSET,
					},
				},
			},
		},
		{
			name: "SessionAuthenticationCredentialTypeUnknown",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType_{
					SessionAuthenticationCredentialType: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType{
						Type: corev1.Credential_Spec_TYPE_UNKNOWN,
					},
				},
			},
		},
		{
			name: "ServiceModeUnset",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_ServiceMode_{
					ServiceMode: &enterprisev1.Condition_Expression_ServiceMode{
						Mode: corev1.Service_Spec_MODE_UNSET,
					},
				},
			},
		},
		{
			name: "DeviceOSTypeUnknown",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_DeviceOSType_{
					DeviceOSType: &enterprisev1.Condition_Expression_DeviceOSType{
						OsType: corev1.Device_Status_OS_TYPE_UNKNOWN,
					},
				},
			},
		},
		{
			name: "FIDOAAGUIDInvalid",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID_{
					SessionAuthenticationCredAuthenticatorAAGUID: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID{
						Aaguid: "invalid",
					},
				},
			},
		},
		{
			name: "GeoIPCountryCodeTooShort",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_{
					SessionAuthenticationGeoipCountryCode: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode{
						Code: "U",
					},
				},
			},
		},
		{
			name: "GeoIPCountryCodeLowercase",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_{
					SessionAuthenticationGeoipCountryCode: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode{
						Code: "us",
					},
				},
			},
		},
		{
			name: "TimeAfterNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_TimeAfter_{
					TimeAfter: &enterprisev1.Condition_Expression_TimeAfter{},
				},
			},
		},
		{
			name: "TimeBeforeNil",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_TimeBefore_{
					TimeBefore: &enterprisev1.Condition_Expression_TimeBefore{},
				},
			},
		},
		{
			name: "RequestHTTPPathExactEmpty",
			expr: policyRequestHTTPPathExact(""),
		},
		{
			name: "RequestHTTPPathPrefixRelative",
			expr: policyRequestHTTPPathPrefix("api"),
		},
		{
			name: "RequestHTTPPathPrefixInvalidChar",
			expr: policyRequestHTTPPathPrefix("/api\nv1"),
		},
		{
			name: "RequestHTTPHasHeaderEmpty",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestHTTPHasHeader_{
					RequestHTTPHasHeader: &enterprisev1.Condition_Expression_RequestHTTPHasHeader{},
				},
			},
		},
		{
			name: "RequestHTTPHasHeaderInvalid",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestHTTPHasHeader_{
					RequestHTTPHasHeader: &enterprisev1.Condition_Expression_RequestHTTPHasHeader{
						Value: "Bad Header",
					},
				},
			},
		},
		{
			name: "RequestHTTPHeaderValueInvalidHeader",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue_{
					RequestHTTPHeaderValue: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue{
						Header: "Bad Header",
						Value:  "v",
					},
				},
			},
		},
		{
			name: "RequestHTTPHeaderValueTooLong",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue_{
					RequestHTTPHeaderValue: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue{
						Header: "X-Test",
						Value:  longValue,
					},
				},
			},
		},
		{
			name: "RequestHTTPMethodUnsupported",
			expr: policyRequestHTTPMethod("BREW"),
		},
		{
			name: "RequestIPInvalid",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestIP_{
					RequestIP: &enterprisev1.Condition_Expression_RequestIP{
						Value: "10.0.0.999",
					},
				},
			},
		},
		{
			name: "RequestIPInRangeInvalid",
			expr: policyRequestIPInRange("10.0.0.0"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := srv.validateExpression(ctx, tc.expr)
			assert.NotNil(t, err)
			assert.True(t, grpcerr.IsInvalidArg(err))
		})
	}
}

func TestPolicyValidateExpressionValidNoRefs(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}

	tcs := []*enterprisev1.Condition_Expression{
		{
			Type: &enterprisev1.Condition_Expression_UserType_{
				UserType: &enterprisev1.Condition_Expression_UserType{
					Type: corev1.User_Spec_WORKLOAD,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionType_{
				SessionType: &enterprisev1.Condition_Expression_SessionType{
					Type: corev1.Session_Status_CLIENTLESS,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionBrowser_{
				SessionBrowser: &enterprisev1.Condition_Expression_SessionBrowser{
					IsBrowser: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationType_{
				SessionAuthenticationType: &enterprisev1.Condition_Expression_SessionAuthenticationType{
					Type: corev1.Session_Status_Authentication_Info_CREDENTIAL,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationAAL_{
				SessionAuthenticationAAL: &enterprisev1.Condition_Expression_SessionAuthenticationAAL{
					Aal: corev1.Session_Status_Authentication_Info_AAL1,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID_{
				SessionAuthenticationCredAuthenticatorAAGUID: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID{
					Aaguid: "00000000-0000-0000-0000-000000000000",
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey_{
				SessionAuthenticationCredAuthenticatorFIDOPasskey: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey{
					IsPasskey: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware_{
				SessionAuthenticationCredAuthenticatorFIDOHardware: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware{
					IsHardware: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent_{
				SessionAuthenticationCredAuthenticatorFIDOUserPresent: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent{
					IsUserPresent: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified_{
				SessionAuthenticationCredAuthenticatorFIDOUserVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified{
					IsUserVerified: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified_{
				SessionAuthenticationCredAuthenticatorFIDOAttestationVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified{
					IsAttestationVerified: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType_{
				SessionAuthenticationCredentialType: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType{
					Type: corev1.Credential_Spec_OAUTH2,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_{
				SessionAuthenticationGeoipCountryCode: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode{
					Code: "GB",
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_TimeAfter_{
				TimeAfter: &enterprisev1.Condition_Expression_TimeAfter{
					Timestamp: timestamppb.Now(),
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_TimeBefore_{
				TimeBefore: &enterprisev1.Condition_Expression_TimeBefore{
					Timestamp: timestamppb.Now(),
				},
			},
		},
		policyRequestHTTPPathExact("/api"),
		policyRequestHTTPPathPrefix("/api"),
		policyRequestHTTPMethod("GET"),
		{
			Type: &enterprisev1.Condition_Expression_RequestHTTPHasHeader_{
				RequestHTTPHasHeader: &enterprisev1.Condition_Expression_RequestHTTPHasHeader{
					Value: "X-Request-ID",
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue_{
				RequestHTTPHeaderValue: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue{
					Header: "X-Request-ID",
					Value:  "abc",
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_RequestIP_{
				RequestIP: &enterprisev1.Condition_Expression_RequestIP{
					Value: "10.0.0.1",
				},
			},
		},
		policyRequestIPInRange("10.0.0.0/24"),
		{
			Type: &enterprisev1.Condition_Expression_ApiServer{
				ApiServer: &enterprisev1.Condition_Expression_APIServer{
					IsAPIServer: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_ApiServerCore{
				ApiServerCore: &enterprisev1.Condition_Expression_APIServerCore{
					IsAPIServerCore: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_ApiServerUser{
				ApiServerUser: &enterprisev1.Condition_Expression_APIServerUser{
					IsAPIServerUser: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_ApiServerEnterprise{
				ApiServerEnterprise: &enterprisev1.Condition_Expression_APIServerEnterprise{
					IsAPIServerEnterprise: true,
				},
			},
		},
		{
			Type: &enterprisev1.Condition_Expression_ApiServerCordium{
				ApiServerCordium: &enterprisev1.Condition_Expression_APIServerCordium{
					IsAPIServerCordium: true,
				},
			},
		},
	}

	for _, tc := range tcs {
		err := srv.validateExpression(ctx, tc)
		assert.Nil(t, err, "%+v", err)
	}
}

func TestPolicyGetCoreConditionValidation(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}

	{
		_, err := srv.GetCoreCondition(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		_, err := srv.GetCoreCondition(ctx, policyExprCond(policyRequestHTTPPathExact("api")))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		cond, err := srv.GetCoreCondition(ctx, policyExprCond(policyRequestHTTPPathExact("/api")))
		assert.Nil(t, err)
		assert.NotNil(t, cond)
		assert.Equal(t, `ctx.request.http.path == "/api"`, cond.GetMatch())
	}
}

func TestPolicyValidateHTTPHelpers(t *testing.T) {
	{
		assert.Nil(t, validateHTTPPath("/"))
		assert.Nil(t, validateHTTPPath("/api/v1"))
		assert.NotNil(t, validateHTTPPath(""))
		assert.NotNil(t, validateHTTPPath("api"))
		assert.NotNil(t, validateHTTPPath("/api\nv1"))
		assert.NotNil(t, validateHTTPPath("/api\rv1"))
		assert.NotNil(t, validateHTTPPath("/api\x00v1"))
		assert.NotNil(t, validateHTTPPath("/"+strings.Repeat("a", maxConditionStringBytes+1)))
	}

	{
		assert.Nil(t, validateHTTPHeaderName("X-Request-ID"))
		assert.Nil(t, validateHTTPHeaderName("x_custom-header.1"))
		assert.NotNil(t, validateHTTPHeaderName(""))
		assert.NotNil(t, validateHTTPHeaderName("Bad Header"))
		assert.NotNil(t, validateHTTPHeaderName("Bad:Header"))
		assert.NotNil(t, validateHTTPHeaderName("é"))
		assert.NotNil(t, validateHTTPHeaderName(strings.Repeat("a", maxConditionStringBytes+1)))
	}

	{
		assert.True(t, isHTTPToken("X-Request-ID"))
		assert.True(t, isHTTPToken("!#$%&'*+-.^_`|~"))
		assert.False(t, isHTTPToken(""))
		assert.False(t, isHTTPToken("Bad Header"))
		assert.False(t, isHTTPToken("Bad:Header"))
		assert.False(t, isHTTPToken("é"))
	}
}

func policyNameRef(name string) *metav1.ObjectReference {
	return &metav1.ObjectReference{
		Name: name,
	}
}

func policyUIDRef(uid string) *metav1.ObjectReference {
	return &metav1.ObjectReference{
		Uid: uid,
	}
}

func policyExprCond(expr *enterprisev1.Condition_Expression) *enterprisev1.Condition {
	return &enterprisev1.Condition{
		Type: &enterprisev1.Condition_Expression_{
			Expression: expr,
		},
	}
}

func policyRequestHTTPPathExact(v string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPPathExact_{
			RequestHTTPPathExact: &enterprisev1.Condition_Expression_RequestHTTPPathExact{
				Value: v,
			},
		},
	}
}

func policyRequestHTTPPathPrefix(v string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPPathPrefix_{
			RequestHTTPPathPrefix: &enterprisev1.Condition_Expression_RequestHTTPPathPrefix{
				Value: v,
			},
		},
	}
}

func policyRequestHTTPMethod(v string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPMethod_{
			RequestHTTPMethod: &enterprisev1.Condition_Expression_RequestHTTPMethod{
				Value: v,
			},
		},
	}
}

func policyRequestIPInRange(v string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestIPInRange_{
			RequestIPInRange: &enterprisev1.Condition_Expression_RequestIPInRange{
				Value: v,
			},
		},
	}
}

func policyMatchAnyConditions(n int) []*enterprisev1.Condition {
	ret := make([]*enterprisev1.Condition, 0, n)
	for i := 0; i < n; i++ {
		ret = append(ret, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_MatchAny{
				MatchAny: true,
			},
		})
	}
	return ret
}

func policyNestedNot(depth int) *enterprisev1.Condition {
	ret := policyExprCond(policyRequestHTTPPathExact("/api"))
	for i := 0; i < depth; i++ {
		ret = &enterprisev1.Condition{
			Type: &enterprisev1.Condition_Not_{
				Not: &enterprisev1.Condition_Not{
					Condition: ret,
				},
			},
		}
	}
	return ret
}
