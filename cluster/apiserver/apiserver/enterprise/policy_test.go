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
					Expression: policyRequestHTTPPathPrefix("/internal"),
				},
			},
		})
		assert.NotNil(t, cond)
		assert.Equal(t, `ctx.request.http.path.startsWith("/internal")`, cond.GetNot())
	}
}

func TestPolicyGetExpression(t *testing.T) {
	srv := &Server{}
	ts := timestamppb.New(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC))
	apiServerExpr := `ctx.service.status.namespaceRef.name == "octelium-api" && ctx.service.spec.mode == "GRPC"`

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_User_{
				User: &enterprisev1.Condition_Expression_User{
					UserRef: policyNameRef("usr"),
				},
			},
		}, `ctx.user.metadata.name == "usr"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Device_{
				Device: &enterprisev1.Condition_Expression_Device{
					DeviceRef: policyNameRef("dev"),
				},
			},
		}, `ctx.device.metadata.name == "dev"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Session_{
				Session: &enterprisev1.Condition_Expression_Session{
					SessionRef: policyNameRef("sess"),
				},
			},
		}, `ctx.session.metadata.name == "sess"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Service_{
				Service: &enterprisev1.Condition_Expression_Service{
					ServiceRef: policyNameRef("svc"),
				},
			},
		}, `ctx.service.metadata.name == "svc"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Namespace_{
				Namespace: &enterprisev1.Condition_Expression_Namespace{
					NamespaceRef: policyNameRef("ns"),
				},
			},
		}, `ctx.namespace.metadata.name == "ns"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Group_{
				Group: &enterprisev1.Condition_Expression_Group{
					GroupRef: policyNameRef("devs"),
				},
			},
		}, `"devs" in ctx.user.spec.groups`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_UserType_{
				UserType: &enterprisev1.Condition_Expression_UserType{
					Type: corev1.User_Spec_HUMAN,
				},
			},
		}, `ctx.user.spec.type == "HUMAN"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionType_{
				SessionType: &enterprisev1.Condition_Expression_SessionType{
					Type: corev1.Session_Status_CLIENT,
				},
			},
		}, `ctx.session.status.type == "CLIENT"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionBrowser_{
				SessionBrowser: &enterprisev1.Condition_Expression_SessionBrowser{},
			},
		}, `ctx.session.status.isBrowser`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationType_{
				SessionAuthenticationType: &enterprisev1.Condition_Expression_SessionAuthenticationType{
					Type: corev1.Session_Status_Authentication_Info_AUTHENTICATOR,
				},
			},
		}, `ctx.session.status.authentication.info.type == "AUTHENTICATOR"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationAAL_{
				SessionAuthenticationAAL: &enterprisev1.Condition_Expression_SessionAuthenticationAAL{
					Aal: corev1.Session_Status_Authentication_Info_AAL2,
				},
			},
		}, `ctx.session.status.authentication.info.aal == "AAL2"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider_{
				SessionAuthenticationIdentityProvider: &enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider{
					IdentityProviderRef: policyUIDRef("idp-uid"),
				},
			},
		}, `ctx.session.status.authentication.info.identityProvider.identityProviderRef.uid == "idp-uid"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredential_{
				SessionAuthenticationCredential: &enterprisev1.Condition_Expression_SessionAuthenticationCredential{
					CredentialRef: policyUIDRef("cred-uid"),
				},
			},
		}, `ctx.session.status.authentication.info.credential.credentialRef.uid == "cred-uid"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType_{
				SessionAuthenticationCredentialType: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType{
					Type: corev1.Credential_Spec_AUTH_TOKEN,
				},
			},
		}, `ctx.session.status.authentication.info.credential.type == "AUTH_TOKEN"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_{
				SessionAuthenticationGeoipCountryCode: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode{
					Code: "US",
				},
			},
		}, `ctx.session.status.authentication.info.geoip.country.code == "US"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID_{
				SessionAuthenticationCredAuthenticatorAAGUID: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID{
					Aaguid: "00000000-0000-0000-0000-000000000000",
				},
			},
		}, `ctx.session.status.authentication.info.authenticator.info.fido.aaguid == "00000000-0000-0000-0000-000000000000"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey_{
				SessionAuthenticationCredAuthenticatorFIDOPasskey: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey{},
			},
		}, `ctx.session.status.authentication.info.authenticator.info.fido.isPasskey`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware_{
				SessionAuthenticationCredAuthenticatorFIDOHardware: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware{},
			},
		}, `ctx.session.status.authentication.info.authenticator.info.fido.isHardware`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified_{
				SessionAuthenticationCredAuthenticatorFIDOAttestationVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified{},
			},
		}, `ctx.session.status.authentication.info.authenticator.info.fido.isAttestationVerified`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent_{
				SessionAuthenticationCredAuthenticatorFIDOUserPresent: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent{},
			},
		}, `ctx.session.status.authentication.info.authenticator.info.fido.userPresent`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified_{
				SessionAuthenticationCredAuthenticatorFIDOUserVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified{},
			},
		}, `ctx.session.status.authentication.info.authenticator.info.fido.userVerified`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_DeviceOSType_{
				DeviceOSType: &enterprisev1.Condition_Expression_DeviceOSType{
					OsType: corev1.Device_Status_LINUX,
				},
			},
		}, `ctx.device.status.osType == "LINUX"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_ServiceMode_{
				ServiceMode: &enterprisev1.Condition_Expression_ServiceMode{
					Mode: corev1.Service_Spec_HTTP,
				},
			},
		}, `ctx.service.spec.mode == "HTTP"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_ServicePublic_{
				ServicePublic: &enterprisev1.Condition_Expression_ServicePublic{},
			},
		}, `ctx.service.spec.isPublic`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_TimeAfter_{
				TimeAfter: &enterprisev1.Condition_Expression_TimeAfter{
					Timestamp: ts,
				},
			},
		}, `now() > timestamp("2026-01-02T03:04:05Z")`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_TimeBefore_{
				TimeBefore: &enterprisev1.Condition_Expression_TimeBefore{
					Timestamp: ts,
				},
			},
		}, `now() < timestamp("2026-01-02T03:04:05Z")`)
	}

	{
		assertPolicyExpression(t, srv, policyRequestHTTPPathExact("/api"), `ctx.request.http.path == "/api"`)
	}

	{
		assertPolicyExpression(t, srv, policyRequestHTTPPathPrefix("/api"), `ctx.request.http.path.startsWith("/api")`)
	}

	{
		assertPolicyExpression(t, srv, policyRequestHTTPMethod("post"), `ctx.request.http.method == "POST"`)
	}

	{
		assertPolicyExpression(t, srv, policyRequestHTTPHasHeader("X-Request-ID"), `"x-request-id" in ctx.request.http.headers`)
	}

	{
		assertPolicyExpression(t, srv, policyRequestHTTPHeaderValue("X-Forwarded-For", "10.0.0.1"), `ctx.request.http.headers["x-forwarded-for"] == "10.0.0.1"`)
	}

	{
		assertPolicyExpression(t, srv, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_RequestIP_{
				RequestIP: &enterprisev1.Condition_Expression_RequestIP{
					Value: "10.0.0.1",
				},
			},
		}, `ctx.request.ip == "10.0.0.1"`)
	}

	{
		assertPolicyExpression(t, srv, policyRequestIPInRange("10.0.0.0/24"), `net.isIPInRange(ctx.request.ip, "10.0.0.0/24")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerReadOnlyMethods(), `(`+apiServerExpr+`) && (["Get", "List"].exists(x, ctx.request.grpc.method.startsWith(x)))`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerMethods("GetSecret", "ListSecret"), `(`+apiServerExpr+`) && (ctx.request.grpc.method in ["GetSecret", "ListSecret"])`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerServices("MainService", "ClusterService"), `(`+apiServerExpr+`) && (ctx.request.grpc.service in ["MainService", "ClusterService"])`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerCore(false), `(`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.core.v1")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerCore(true), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.core.v1")) && (["Get", "List"].exists(x, ctx.request.grpc.method.startsWith(x)))`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerUser(false), `(`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.user.v1")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerUser(true), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.user.v1")) && (["Get", "List"].exists(x, ctx.request.grpc.method.startsWith(x)))`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerEnterprise(enterprisev1.Condition_Expression_APIServerEnterprise_ANY), `(`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.enterprise.v1")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerEnterprise(enterprisev1.Condition_Expression_APIServerEnterprise_MAIN), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.enterprise.v1")) && (ctx.request.grpc.service == "MainService")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerEnterprise(enterprisev1.Condition_Expression_APIServerEnterprise_CLUSTER), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.enterprise.v1")) && (ctx.request.grpc.service == "ClusterService")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerEnterprise(enterprisev1.Condition_Expression_APIServerEnterprise_POLICY_PORTAL), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.enterprise.v1")) && (ctx.request.grpc.service == "PolicyPortalService")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerCordium(enterprisev1.Condition_Expression_APIServerCordium_MAIN), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.cordium.v1")) && (ctx.request.grpc.service == "MainService")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerCordium(enterprisev1.Condition_Expression_APIServerCordium_MANAGEMENT), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.cordium.v1")) && (ctx.request.grpc.service == "ManagementService")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerCordium(enterprisev1.Condition_Expression_APIServerCordium_WORKSPACE), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.cordium.v1")) && (ctx.request.grpc.service == "WorkspaceService")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerAccess(enterprisev1.Condition_Expression_APIServerAccess_ANY), `(`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.access.v1")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerAccess(enterprisev1.Condition_Expression_APIServerAccess_MAIN), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.access.v1")) && (ctx.request.grpc.service == "MainService")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerAccess(enterprisev1.Condition_Expression_APIServerAccess_USER), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.access.v1")) && (ctx.request.grpc.service == "UserService")`)
	}

	{
		assertPolicyExpression(t, srv, policyAPIServerAccess(enterprisev1.Condition_Expression_APIServerAccess_REVIEWER), `((`+apiServerExpr+`) && (ctx.request.grpc.package == "octelium.api.main.access.v1")) && (ctx.request.grpc.service == "ReviewerService")`)
	}
}

func TestPolicyGetRequestExpression(t *testing.T) {
	srv := &Server{}

	tests := []struct {
		name string
		expr *enterprisev1.Condition_Expression
		want string
	}{
		{
			name: "http method in",
			expr: policyRequestHTTPMethodMatchIn("get", "POST"),
			want: `ctx.request.http.method in ["GET", "POST"]`,
		},
		{
			name: "http path contains",
			expr: policyRequestHTTPPathMatch(&enterprisev1.Condition_Expression_StringMatch{
				Type: &enterprisev1.Condition_Expression_StringMatch_Contains{Contains: "/admin/"},
			}),
			want: `ctx.request.http.path.contains("/admin/")`,
		},
		{
			name: "http header value prefix",
			expr: policyRequestHTTPHeaderValueMatch(
				policyStringMatchExact("X-Auth"),
				&enterprisev1.Condition_Expression_StringMatch{Type: &enterprisev1.Condition_Expression_StringMatch_Prefix{Prefix: "Bearer "}},
			),
			want: `ctx.request.http.headers.exists(k, k == "x-auth" && ctx.request.http.headers[k].startsWith("Bearer "))`,
		},
		{
			name: "http query parameter",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestHTTPQueryParamValue_{
					RequestHTTPQueryParamValue: &enterprisev1.Condition_Expression_RequestHTTPQueryParamValue{
						Name:  "tenant",
						Match: &enterprisev1.Condition_Expression_StringMatch{Type: &enterprisev1.Condition_Expression_StringMatch_Suffix{Suffix: "-prod"}},
					},
				},
			},
			want: `("tenant" in ctx.request.http.queryParams) && (ctx.request.http.queryParams["tenant"].endsWith("-prod"))`,
		},
		{
			name: "ssh marker",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestSSH_{
					RequestSSH: &enterprisev1.Condition_Expression_RequestSSH{},
				},
			},
			want: `has(ctx.request.ssh)`,
		},
		{
			name: "kubernetes namespace",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestKubernetesNamespace_{
					RequestKubernetesNamespace: &enterprisev1.Condition_Expression_RequestKubernetesNamespace{
						Match: &enterprisev1.Condition_Expression_StringMatch{Type: &enterprisev1.Condition_Expression_StringMatch_Prefix{Prefix: "team-"}},
					},
				},
			},
			want: `(has(ctx.request.kubernetes)) && (ctx.request.kubernetes.namespace.startsWith("team-"))`,
		},
		{
			name: "grpc method",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestGRPCMethod_{
					RequestGRPCMethod: &enterprisev1.Condition_Expression_RequestGRPCMethod{
						Match: policyStringMatchIn("Get", "List"),
					},
				},
			},
			want: `(has(ctx.request.grpc)) && (ctx.request.grpc.method in ["Get", "List"])`,
		},
		{
			name: "postgres query",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestPostgresQueryText_{
					RequestPostgresQueryText: &enterprisev1.Condition_Expression_RequestPostgresQueryText{
						Match: &enterprisev1.Condition_Expression_StringMatch{Type: &enterprisev1.Condition_Expression_StringMatch_Contains{Contains: "SELECT"}},
					},
				},
			},
			want: `(has(ctx.request.postgres)) && (has(ctx.request.postgres.query)) && (ctx.request.postgres.query.query.contains("SELECT"))`,
		},
		{
			name: "dns type",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestDNSTypeID_{
					RequestDNSTypeID: &enterprisev1.Condition_Expression_RequestDNSTypeID{
						Match: &enterprisev1.Condition_Expression_IntMatch{Type: &enterprisev1.Condition_Expression_IntMatch_Exact{Exact: 16}},
					},
				},
			},
			want: `(has(ctx.request.dns)) && (ctx.request.dns.typeID == 16)`,
		},
		{
			name: "socks port",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestSOCKS5Port_{
					RequestSOCKS5Port: &enterprisev1.Condition_Expression_RequestSOCKS5Port{
						Match: &enterprisev1.Condition_Expression_UIntMatch{Type: &enterprisev1.Condition_Expression_UIntMatch_GreaterThan{GreaterThan: 1024}},
					},
				},
			},
			want: `(has(ctx.request.socks5)) && (has(ctx.request.socks5.connect)) && (ctx.request.socks5.connect.port > uint(1024))`,
		},
		{
			name: "mcp tool allowlist",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestMCPToolName{
					RequestMCPToolName: &enterprisev1.Condition_Expression_MCPToolName{
						Match: policyStringMatchIn("search", "read_db"),
					},
				},
			},
			want: `(has(ctx.request.mcp)) && (ctx.request.mcp.method == "tools/call") && (ctx.request.mcp.name in ["search", "read_db"])`,
		},
		{
			name: "llm input estimate",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestLLMEstimatedInputTokens{
					RequestLLMEstimatedInputTokens: &enterprisev1.Condition_Expression_LLMEstimatedInputTokens{
						Match:           &enterprisev1.Condition_Expression_UIntMatch{Type: &enterprisev1.Condition_Expression_UIntMatch_LessThanOrEqual{LessThanOrEqual: 1000}},
						RequireComplete: true,
					},
				},
			},
			want: `((has(ctx.request.llm)) && (ctx.request.llm.estimatedInputTokens <= uint(1000))) && (ctx.request.llm.estimateQuality == "COMPLETE")`,
		},
		{
			name: "llm tool name",
			expr: &enterprisev1.Condition_Expression{
				Type: &enterprisev1.Condition_Expression_RequestLLMToolName{
					RequestLLMToolName: &enterprisev1.Condition_Expression_LLMToolName{
						Match: &enterprisev1.Condition_Expression_StringMatch{Type: &enterprisev1.Condition_Expression_StringMatch_Prefix{Prefix: "mcp_"}},
					},
				},
			},
			want: `(has(ctx.request.llm)) && (ctx.request.llm.toolNames.exists(x, x.startsWith("mcp_")))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPolicyExpression(t, srv, tt.expr, tt.want)
		})
	}
}

func TestPolicyValidateConditionValid(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}

	{
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
									Expression: policyRequestIPInRange("10.0.0.0/24"),
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
}

func TestPolicyValidateConditionInvalid(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}

	{
		err := srv.validateCondition(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, &enterprisev1.Condition{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_All_{
				All: &enterprisev1.Condition_All{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_Any_{
				Any: &enterprisev1.Condition_Any{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_None_{
				None: &enterprisev1.Condition_None{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_Not_{
				Not: &enterprisev1.Condition_Not{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_All_{
				All: &enterprisev1.Condition_All{
					Of: policyMatchAnyConditions(maxConditionChildren + 1),
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_Any_{
				Any: &enterprisev1.Condition_Any{
					Of: policyMatchAnyConditions(maxConditionChildren + 1),
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_None_{
				None: &enterprisev1.Condition_None{
					Of: policyMatchAnyConditions(maxConditionChildren + 1),
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, policyNestedAll(maxConditionDepth+1))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_All_{
				All: &enterprisev1.Condition_All{
					Of: []*enterprisev1.Condition{{}},
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}
}

func TestPolicyValidateExpressionInvalid(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}
	longValue := strings.Repeat("a", maxConditionStringBytes+1)

	{
		err := srv.validateExpression(ctx, nil)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_User_{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_User_{
				User: &enterprisev1.Condition_Expression_User{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Group_{
				Group: &enterprisev1.Condition_Expression_Group{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Device_{
				Device: &enterprisev1.Condition_Expression_Device{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Session_{
				Session: &enterprisev1.Condition_Expression_Session{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Service_{
				Service: &enterprisev1.Condition_Expression_Service{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_Namespace_{
				Namespace: &enterprisev1.Condition_Expression_Namespace{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider_{
				SessionAuthenticationIdentityProvider: &enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredential_{
				SessionAuthenticationCredential: &enterprisev1.Condition_Expression_SessionAuthenticationCredential{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_UserType_{
				UserType: &enterprisev1.Condition_Expression_UserType{Type: corev1.User_Spec_TYPE_UNKNOWN},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionType_{
				SessionType: &enterprisev1.Condition_Expression_SessionType{Type: corev1.Session_Status_TYPE_UNKNOWN},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationType_{
				SessionAuthenticationType: &enterprisev1.Condition_Expression_SessionAuthenticationType{Type: corev1.Session_Status_Authentication_Info_TYPE_UNSET},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationAAL_{
				SessionAuthenticationAAL: &enterprisev1.Condition_Expression_SessionAuthenticationAAL{Aal: corev1.Session_Status_Authentication_Info_AAL_UNSET},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType_{
				SessionAuthenticationCredentialType: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType{Type: corev1.Credential_Spec_TYPE_UNKNOWN},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_ServiceMode_{
				ServiceMode: &enterprisev1.Condition_Expression_ServiceMode{Mode: corev1.Service_Spec_MODE_UNSET},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_DeviceOSType_{
				DeviceOSType: &enterprisev1.Condition_Expression_DeviceOSType{OsType: corev1.Device_Status_OS_TYPE_UNKNOWN},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID_{
				SessionAuthenticationCredAuthenticatorAAGUID: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID{Aaguid: "invalid"},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_{
				SessionAuthenticationGeoipCountryCode: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode{Code: "U"},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_{
				SessionAuthenticationGeoipCountryCode: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode{Code: "us"},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_TimeAfter_{
				TimeAfter: &enterprisev1.Condition_Expression_TimeAfter{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_TimeBefore_{
				TimeBefore: &enterprisev1.Condition_Expression_TimeBefore{},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyRequestHTTPPathExact(""))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyRequestHTTPPathPrefix("api"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyRequestHTTPPathPrefix("/api\nv1"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyRequestHTTPHasHeader(""))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyRequestHTTPHasHeader("Bad Header"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyRequestHTTPHeaderValue("Bad Header", "v"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyRequestHTTPHeaderValue("X-Test", longValue))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyRequestHTTPMethod("BREW"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_RequestIP_{
				RequestIP: &enterprisev1.Condition_Expression_RequestIP{Value: "10.0.0.999"},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyRequestIPInRange("10.0.0.0"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerMethods())
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerMethods("GetSecret", "GetSecret"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerMethods("Get Secret"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerMethods(longValue))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerServices())
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerEnterprise(enterprisev1.Condition_Expression_APIServerEnterprise_SERVICE_UNSET))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerEnterprise(enterprisev1.Condition_Expression_APIServerEnterprise_Service(-1)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerEnterprise(enterprisev1.Condition_Expression_APIServerEnterprise_Service(99)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerCordium(enterprisev1.Condition_Expression_APIServerCordium_SERVICE_UNSET))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerCordium(enterprisev1.Condition_Expression_APIServerCordium_Service(99)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerAccess(enterprisev1.Condition_Expression_APIServerAccess_SERVICE_UNSET))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}

	{
		err := srv.validateExpression(ctx, policyAPIServerAccess(enterprisev1.Condition_Expression_APIServerAccess_Service(99)))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	}
}

func TestPolicyValidateExpressionValidNoRefs(t *testing.T) {
	ctx := context.Background()
	srv := &Server{}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_UserType_{
				UserType: &enterprisev1.Condition_Expression_UserType{Type: corev1.User_Spec_WORKLOAD},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionType_{
				SessionType: &enterprisev1.Condition_Expression_SessionType{Type: corev1.Session_Status_CLIENTLESS},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionBrowser_{
				SessionBrowser: &enterprisev1.Condition_Expression_SessionBrowser{},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationType_{
				SessionAuthenticationType: &enterprisev1.Condition_Expression_SessionAuthenticationType{Type: corev1.Session_Status_Authentication_Info_CREDENTIAL},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationAAL_{
				SessionAuthenticationAAL: &enterprisev1.Condition_Expression_SessionAuthenticationAAL{Aal: corev1.Session_Status_Authentication_Info_AAL1},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID_{
				SessionAuthenticationCredAuthenticatorAAGUID: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID{Aaguid: "00000000-0000-0000-0000-000000000000"},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey_{
				SessionAuthenticationCredAuthenticatorFIDOPasskey: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey{},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware_{
				SessionAuthenticationCredAuthenticatorFIDOHardware: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware{},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent_{
				SessionAuthenticationCredAuthenticatorFIDOUserPresent: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent{},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified_{
				SessionAuthenticationCredAuthenticatorFIDOUserVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified{},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified_{
				SessionAuthenticationCredAuthenticatorFIDOAttestationVerified: &enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified{},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType_{
				SessionAuthenticationCredentialType: &enterprisev1.Condition_Expression_SessionAuthenticationCredentialType{Type: corev1.Credential_Spec_OAUTH2},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_{
				SessionAuthenticationGeoipCountryCode: &enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode{Code: "GB"},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_TimeAfter_{
				TimeAfter: &enterprisev1.Condition_Expression_TimeAfter{Timestamp: timestamppb.Now()},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_TimeBefore_{
				TimeBefore: &enterprisev1.Condition_Expression_TimeBefore{Timestamp: timestamppb.Now()},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyRequestHTTPPathExact("/api")))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyRequestHTTPPathPrefix("/api")))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyRequestHTTPMethod("GET")))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyRequestHTTPHasHeader("X-Request-ID")))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyRequestHTTPHeaderValue("X-Request-ID", "abc")))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, &enterprisev1.Condition_Expression{
			Type: &enterprisev1.Condition_Expression_RequestIP_{
				RequestIP: &enterprisev1.Condition_Expression_RequestIP{Value: "10.0.0.1"},
			},
		}))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyRequestIPInRange("10.0.0.0/24")))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerReadOnlyMethods()))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerMethods("GetSecret", "ListSecret")))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerServices("MainService", "ClusterService")))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerCore(false)))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerCore(true)))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerUser(false)))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerUser(true)))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerEnterprise(enterprisev1.Condition_Expression_APIServerEnterprise_ANY)))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerEnterprise(enterprisev1.Condition_Expression_APIServerEnterprise_MAIN)))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerCordium(enterprisev1.Condition_Expression_APIServerCordium_MAIN)))
	}

	{
		assert.Nil(t, srv.validateExpression(ctx, policyAPIServerAccess(enterprisev1.Condition_Expression_APIServerAccess_ANY)))
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

	{
		cond, err := srv.GetCoreCondition(ctx, &enterprisev1.Condition{
			Type: &enterprisev1.Condition_Not_{
				Not: &enterprisev1.Condition_Not{
					Expression: policyRequestHTTPMethod("delete"),
				},
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, cond)
		assert.Equal(t, `ctx.request.http.method == "DELETE"`, cond.GetNot())
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

func TestPolicyValidateAPIServerStringHelpers(t *testing.T) {
	{
		assert.Nil(t, validateAPIServerString("GetSecret", "API server method"))
		assert.Nil(t, validateAPIServerString("octelium.api.main.enterprise.v1.MainService", "API server service"))
		assert.NotNil(t, validateAPIServerString("", "API server method"))
		assert.NotNil(t, validateAPIServerString("Get Secret", "API server method"))
		assert.NotNil(t, validateAPIServerString("Get/Secret", "API server method"))
		assert.NotNil(t, validateAPIServerString("é", "API server method"))
		assert.NotNil(t, validateAPIServerString(strings.Repeat("a", maxConditionStringBytes+1), "API server method"))
	}

	{
		assert.Nil(t, validateAPIServerStringList([]string{"GetSecret", "ListSecret"}, "API server methods"))
		assert.NotNil(t, validateAPIServerStringList(nil, "API server methods"))
		assert.NotNil(t, validateAPIServerStringList([]string{"GetSecret", "GetSecret"}, "API server methods"))
		assert.NotNil(t, validateAPIServerStringList(policyStringList(maxConditionChildren+1), "API server methods"))
	}
}

func assertPolicyExpression(t *testing.T, srv *Server, expr *enterprisev1.Condition_Expression, expected string) {
	ret := srv.getExpression(expr)
	assert.Equal(t, expected, ret)
	assert.NotEmpty(t, ret)
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

func policyStringMatchExact(v string) *enterprisev1.Condition_Expression_StringMatch {
	return &enterprisev1.Condition_Expression_StringMatch{
		Type: &enterprisev1.Condition_Expression_StringMatch_Exact{Exact: v},
	}
}

func policyStringMatchPrefix(v string) *enterprisev1.Condition_Expression_StringMatch {
	return &enterprisev1.Condition_Expression_StringMatch{
		Type: &enterprisev1.Condition_Expression_StringMatch_Prefix{Prefix: v},
	}
}

func policyStringMatchIn(values ...string) *enterprisev1.Condition_Expression_StringMatch {
	return &enterprisev1.Condition_Expression_StringMatch{
		Type: &enterprisev1.Condition_Expression_StringMatch_In_{
			In: &enterprisev1.Condition_Expression_StringMatch_In{Values: values},
		},
	}
}

func policyRequestHTTPPathExact(v string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPPath_{
			RequestHTTPPath: &enterprisev1.Condition_Expression_RequestHTTPPath{
				Match: policyStringMatchExact(v),
			},
		},
	}
}

func policyRequestHTTPPathPrefix(v string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPPath_{
			RequestHTTPPath: &enterprisev1.Condition_Expression_RequestHTTPPath{
				Match: policyStringMatchPrefix(v),
			},
		},
	}
}

func policyRequestHTTPPathMatch(match *enterprisev1.Condition_Expression_StringMatch) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPPath_{
			RequestHTTPPath: &enterprisev1.Condition_Expression_RequestHTTPPath{
				Match: match,
			},
		},
	}
}

func policyRequestHTTPMethod(v string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPMethod_{
			RequestHTTPMethod: &enterprisev1.Condition_Expression_RequestHTTPMethod{
				Match: policyStringMatchExact(v),
			},
		},
	}
}

func policyRequestHTTPMethodMatchIn(values ...string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPMethod_{
			RequestHTTPMethod: &enterprisev1.Condition_Expression_RequestHTTPMethod{
				Match: policyStringMatchIn(values...),
			},
		},
	}
}

func policyRequestHTTPHasHeader(v string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPHasHeader_{
			RequestHTTPHasHeader: &enterprisev1.Condition_Expression_RequestHTTPHasHeader{
				Match: policyStringMatchExact(v),
			},
		},
	}
}

func policyRequestHTTPHeaderValue(header, value string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue_{
			RequestHTTPHeaderValue: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue{
				Header: policyStringMatchExact(header),
				Value:  policyStringMatchExact(value),
			},
		},
	}
}

func policyRequestHTTPHeaderValueMatch(header, value *enterprisev1.Condition_Expression_StringMatch) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue_{
			RequestHTTPHeaderValue: &enterprisev1.Condition_Expression_RequestHTTPHeaderValue{
				Header: header,
				Value:  value,
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

func policyAPIServerReadOnlyMethods() *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_ApiServerReadOnlyMethods{
			ApiServerReadOnlyMethods: &enterprisev1.Condition_Expression_APIServerReadOnlyMethods{},
		},
	}
}

func policyAPIServerMethods(methods ...string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_ApiServerMethods{
			ApiServerMethods: &enterprisev1.Condition_Expression_APIServerMethods{
				Methods: methods,
			},
		},
	}
}

func policyAPIServerServices(services ...string) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_ApiServerServices{
			ApiServerServices: &enterprisev1.Condition_Expression_APIServerServices{
				Services: services,
			},
		},
	}
}

func policyAPIServerCore(readOnlyMethods bool) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_ApiServerCore{
			ApiServerCore: &enterprisev1.Condition_Expression_APIServerCore{
				ReadOnlyMethods: readOnlyMethods,
			},
		},
	}
}

func policyAPIServerUser(readOnlyMethods bool) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_ApiServerUser{
			ApiServerUser: &enterprisev1.Condition_Expression_APIServerUser{
				ReadOnlyMethods: readOnlyMethods,
			},
		},
	}
}

func policyAPIServerEnterprise(service enterprisev1.Condition_Expression_APIServerEnterprise_Service) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_ApiServerEnterprise{
			ApiServerEnterprise: &enterprisev1.Condition_Expression_APIServerEnterprise{
				Service: service,
			},
		},
	}
}

func policyAPIServerCordium(service enterprisev1.Condition_Expression_APIServerCordium_Service) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_ApiServerCordium{
			ApiServerCordium: &enterprisev1.Condition_Expression_APIServerCordium{
				Service: service,
			},
		},
	}
}

func policyAPIServerAccess(service enterprisev1.Condition_Expression_APIServerAccess_Service) *enterprisev1.Condition_Expression {
	return &enterprisev1.Condition_Expression{
		Type: &enterprisev1.Condition_Expression_ApiServerAccess{
			ApiServerAccess: &enterprisev1.Condition_Expression_APIServerAccess{
				Service: service,
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

func policyNestedAll(depth int) *enterprisev1.Condition {
	ret := policyExprCond(policyRequestHTTPPathExact("/api"))
	for i := 0; i < depth; i++ {
		ret = &enterprisev1.Condition{
			Type: &enterprisev1.Condition_All_{
				All: &enterprisev1.Condition_All{
					Of: []*enterprisev1.Condition{ret},
				},
			},
		}
	}
	return ret
}

func policyStringList(n int) []string {
	ret := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ret = append(ret, "Method"+celString(string(rune('a'+i))))
	}
	return ret
}
