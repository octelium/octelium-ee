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
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

const (
	maxConditionDepth       = 32
	maxConditionChildren    = 64
	maxConditionStringBytes = 512
)

func (s *Server) validateCondition(ctx context.Context, c *enterprisev1.Condition) error {
	return s.validateConditionDepth(ctx, c, 0)
}

func (s *Server) validateConditionDepth(ctx context.Context, c *enterprisev1.Condition, depth int) error {
	if c == nil {
		return grpcutils.InvalidArg("Nil Condition")
	}
	if c.Type == nil {
		return grpcutils.InvalidArg("Condition type must be set")
	}
	if depth > maxConditionDepth {
		return grpcutils.InvalidArg("Condition nesting is too deep")
	}

	switch c.Type.(type) {
	case *enterprisev1.Condition_All_:
		cond := c.GetAll()
		if cond == nil {
			return grpcutils.InvalidArg("Nil ALL condition")
		}
		if len(cond.Of) == 0 {
			return grpcutils.InvalidArg("ALL condition must contain at least one child")
		}
		if len(cond.Of) > maxConditionChildren {
			return grpcutils.InvalidArg("ALL condition has too many children")
		}
		for _, child := range cond.Of {
			if err := s.validateConditionDepth(ctx, child, depth+1); err != nil {
				return err
			}
		}

	case *enterprisev1.Condition_Any_:
		cond := c.GetAny()
		if cond == nil {
			return grpcutils.InvalidArg("Nil ANY condition")
		}
		if len(cond.Of) == 0 {
			return grpcutils.InvalidArg("ANY condition must contain at least one child")
		}
		if len(cond.Of) > maxConditionChildren {
			return grpcutils.InvalidArg("ANY condition has too many children")
		}
		for _, child := range cond.Of {
			if err := s.validateConditionDepth(ctx, child, depth+1); err != nil {
				return err
			}
		}

	case *enterprisev1.Condition_None_:
		cond := c.GetNone()
		if cond == nil {
			return grpcutils.InvalidArg("Nil NONE condition")
		}
		if len(cond.Of) == 0 {
			return grpcutils.InvalidArg("NONE condition must contain at least one child")
		}
		if len(cond.Of) > maxConditionChildren {
			return grpcutils.InvalidArg("NONE condition has too many children")
		}
		for _, child := range cond.Of {
			if err := s.validateConditionDepth(ctx, child, depth+1); err != nil {
				return err
			}
		}

	case *enterprisev1.Condition_Not_:
		cond := c.GetNot()
		if cond == nil || cond.Expression == nil {
			return grpcutils.InvalidArg("NOT condition must contain an expression")
		}
		if err := s.validateExpression(ctx, cond.Expression); err != nil {
			return err
		}

	case *enterprisev1.Condition_MatchAny:
	case *enterprisev1.Condition_Expression_:
		if err := s.validateExpression(ctx, c.GetExpression()); err != nil {
			return err
		}

	default:
		return grpcutils.InvalidArg("Unsupported Condition type")
	}

	return nil
}

func (s *Server) validateExpression(ctx context.Context, p *enterprisev1.Condition_Expression) error {
	if p == nil {
		return grpcutils.InvalidArg("Nil Expression")
	}
	if p.Type == nil {
		return grpcutils.InvalidArg("Expression type must be set")
	}

	switch p.Type.(type) {
	case *enterprisev1.Condition_Expression_User_:
		expr := p.GetUser()
		if expr == nil {
			return grpcutils.InvalidArg("Nil User expression")
		}
		return s.validateUserRef(ctx, expr.GetUserRef())

	case *enterprisev1.Condition_Expression_Group_:
		expr := p.GetGroup()
		if expr == nil {
			return grpcutils.InvalidArg("Nil Group expression")
		}
		return s.validateGroupRef(ctx, expr.GetGroupRef())

	case *enterprisev1.Condition_Expression_Device_:
		expr := p.GetDevice()
		if expr == nil {
			return grpcutils.InvalidArg("Nil Device expression")
		}
		return s.validateDeviceRef(ctx, expr.GetDeviceRef())

	case *enterprisev1.Condition_Expression_Session_:
		expr := p.GetSession()
		if expr == nil {
			return grpcutils.InvalidArg("Nil Session expression")
		}
		return s.validateSessionRef(ctx, expr.GetSessionRef())

	case *enterprisev1.Condition_Expression_Service_:
		expr := p.GetService()
		if expr == nil {
			return grpcutils.InvalidArg("Nil Service expression")
		}
		return s.validateServiceRef(ctx, expr.GetServiceRef())

	case *enterprisev1.Condition_Expression_Namespace_:
		expr := p.GetNamespace()
		if expr == nil {
			return grpcutils.InvalidArg("Nil Namespace expression")
		}
		return s.validateNamespaceRef(ctx, expr.GetNamespaceRef())

	case *enterprisev1.Condition_Expression_UserType_:
		if p.GetUserType() == nil || p.GetUserType().Type == corev1.User_Spec_TYPE_UNKNOWN {
			return grpcutils.InvalidArg("User type must be set")
		}

	case *enterprisev1.Condition_Expression_SessionType_:
		if p.GetSessionType() == nil || p.GetSessionType().Type == corev1.Session_Status_TYPE_UNKNOWN {
			return grpcutils.InvalidArg("Session type must be set")
		}

	case *enterprisev1.Condition_Expression_SessionBrowser_:
		if p.GetSessionBrowser() == nil {
			return grpcutils.InvalidArg("Nil SessionBrowser expression")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationType_:
		if p.GetSessionAuthenticationType() == nil ||
			p.GetSessionAuthenticationType().Type == corev1.Session_Status_Authentication_Info_TYPE_UNSET {
			return grpcutils.InvalidArg("Session authentication type must be set")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider_:
		expr := p.GetSessionAuthenticationIdentityProvider()
		if expr == nil {
			return grpcutils.InvalidArg("Nil SessionAuthenticationIdentityProvider expression")
		}
		return s.validateIdentityProviderRef(ctx, expr.GetIdentityProviderRef())

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredential_:
		expr := p.GetSessionAuthenticationCredential()
		if expr == nil {
			return grpcutils.InvalidArg("Nil SessionAuthenticationCredential expression")
		}
		return s.validateCredentialRef(ctx, expr.GetCredentialRef())

	case *enterprisev1.Condition_Expression_SessionAuthenticationAAL_:
		if p.GetSessionAuthenticationAAL() == nil ||
			p.GetSessionAuthenticationAAL().Aal == corev1.Session_Status_Authentication_Info_AAL_UNSET {
			return grpcutils.InvalidArg("Session authentication AAL must be set")
		}

	case *enterprisev1.Condition_Expression_ServicePublic_:
		if p.GetServicePublic() == nil {
			return grpcutils.InvalidArg("Nil ServicePublic expression")
		}

	case *enterprisev1.Condition_Expression_ServiceMode_:
		if p.GetServiceMode() == nil || p.GetServiceMode().Mode == corev1.Service_Spec_MODE_UNSET {
			return grpcutils.InvalidArg("Service mode must be set")
		}

	case *enterprisev1.Condition_Expression_DeviceOSType_:
		if p.GetDeviceOSType() == nil || p.GetDeviceOSType().OsType == corev1.Device_Status_OS_TYPE_UNKNOWN {
			return grpcutils.InvalidArg("Device OS type must be set")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID_:
		expr := p.GetSessionAuthenticationCredAuthenticatorAAGUID()
		if expr == nil {
			return grpcutils.InvalidArg("Nil Authenticator AAGUID expression")
		}
		if !govalidator.IsUUID(expr.Aaguid) {
			return grpcutils.InvalidArg("Invalid AAGUID")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey_:
		if p.GetSessionAuthenticationCredAuthenticatorFIDOPasskey() == nil {
			return grpcutils.InvalidArg("Nil FIDO passkey expression")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified_:
		if p.GetSessionAuthenticationCredAuthenticatorFIDOAttestationVerified() == nil {
			return grpcutils.InvalidArg("Nil FIDO attestation expression")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware_:
		if p.GetSessionAuthenticationCredAuthenticatorFIDOHardware() == nil {
			return grpcutils.InvalidArg("Nil FIDO hardware expression")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent_:
		if p.GetSessionAuthenticationCredAuthenticatorFIDOUserPresent() == nil {
			return grpcutils.InvalidArg("Nil FIDO user-present expression")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified_:
		if p.GetSessionAuthenticationCredAuthenticatorFIDOUserVerified() == nil {
			return grpcutils.InvalidArg("Nil FIDO user-verified expression")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredentialType_:
		if p.GetSessionAuthenticationCredentialType() == nil ||
			p.GetSessionAuthenticationCredentialType().Type == corev1.Credential_Spec_TYPE_UNKNOWN {
			return grpcutils.InvalidArg("Credential type must be set")
		}

	case *enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_:
		expr := p.GetSessionAuthenticationGeoipCountryCode()
		if expr == nil {
			return grpcutils.InvalidArg("Nil GeoIP country code expression")
		}
		if len(expr.Code) != 2 {
			return grpcutils.InvalidArg("GeoIP country code must be two characters")
		}
		for _, r := range expr.Code {
			if r < 'A' || r > 'Z' {
				return grpcutils.InvalidArg("GeoIP country code must be uppercase ISO-3166 alpha-2")
			}
		}

	case *enterprisev1.Condition_Expression_TimeAfter_:
		expr := p.GetTimeAfter()
		if expr == nil || expr.Timestamp == nil || !expr.Timestamp.IsValid() {
			return grpcutils.InvalidArg("Invalid TimeAfter timestamp")
		}

	case *enterprisev1.Condition_Expression_TimeBefore_:
		expr := p.GetTimeBefore()
		if expr == nil || expr.Timestamp == nil || !expr.Timestamp.IsValid() {
			return grpcutils.InvalidArg("Invalid TimeBefore timestamp")
		}

	case *enterprisev1.Condition_Expression_TimeDayType_:
		expr := p.GetTimeDayType()
		if expr == nil {
			return grpcutils.InvalidArg("Nil TimeDayType expression")
		}
		if expr.GetType() != enterprisev1.Condition_Expression_TimeDayType_WEEKDAY &&
			expr.GetType() != enterprisev1.Condition_Expression_TimeDayType_WEEKEND {
			return grpcutils.InvalidArg("TimeDayType type must be set")
		}
		if err := validateBoundedString(expr.GetTimezone(), false, "TimeDayType timezone"); err != nil {
			return err
		}
		if expr.GetTimezone() != "" {
			if _, err := time.LoadLocation(expr.GetTimezone()); err != nil {
				return grpcutils.InvalidArg("Invalid TimeDayType timezone")
			}
		}

	case *enterprisev1.Condition_Expression_RequestHTTPPath_:
		expr := p.GetRequestHTTPPath()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP path expression")
		}
		return validateHTTPPathMatch(expr.GetMatch())

	case *enterprisev1.Condition_Expression_RequestHTTPHasHeader_:
		expr := p.GetRequestHTTPHasHeader()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP header expression")
		}
		return validateStringMatch(expr.GetMatch(), "HTTP header name", validateHTTPHeaderName)

	case *enterprisev1.Condition_Expression_RequestHTTPHeaderValue_:
		expr := p.GetRequestHTTPHeaderValue()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP header value expression")
		}
		if err := validateStringMatch(expr.GetHeader(), "HTTP header name", validateHTTPHeaderName); err != nil {
			return err
		}
		return validateStringMatch(expr.GetValue(), "HTTP header value", nil)

	case *enterprisev1.Condition_Expression_RequestHTTPMethod_:
		expr := p.GetRequestHTTPMethod()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP method expression")
		}
		return validateHTTPMethodMatch(expr.GetMatch())

	case *enterprisev1.Condition_Expression_RequestHTTPHost_:
		return validateStringMatch(p.GetRequestHTTPHost().GetMatch(), "HTTP host", nil)

	case *enterprisev1.Condition_Expression_RequestHTTPProtocol_:
		return validateStringMatch(p.GetRequestHTTPProtocol().GetMatch(), "HTTP protocol", nil)

	case *enterprisev1.Condition_Expression_RequestHTTPScheme_:
		return validateStringMatch(p.GetRequestHTTPScheme().GetMatch(), "HTTP scheme", nil)

	case *enterprisev1.Condition_Expression_RequestHTTPURI_:
		return validateStringMatch(p.GetRequestHTTPURI().GetMatch(), "HTTP URI", nil)

	case *enterprisev1.Condition_Expression_RequestHTTPSize_:
		return validateIntMatch(p.GetRequestHTTPSize().GetMatch(), "HTTP request size")

	case *enterprisev1.Condition_Expression_RequestHTTPHasQueryParam_:
		expr := p.GetRequestHTTPHasQueryParam()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP query parameter expression")
		}
		return validateBoundedString(expr.GetName(), true, "HTTP query parameter name")

	case *enterprisev1.Condition_Expression_RequestHTTPQueryParamValue_:
		expr := p.GetRequestHTTPQueryParamValue()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP query parameter value expression")
		}
		if err := validateBoundedString(expr.GetName(), true, "HTTP query parameter name"); err != nil {
			return err
		}
		return validateStringMatch(expr.GetMatch(), "HTTP query parameter value", nil)

	case *enterprisev1.Condition_Expression_RequestSSH_:
		if p.GetRequestSSH() == nil {
			return grpcutils.InvalidArg("Nil SSH request expression")
		}

	case *enterprisev1.Condition_Expression_RequestSSHUser_:
		return validateStringMatch(p.GetRequestSSHUser().GetMatch(), "SSH user", nil)

	case *enterprisev1.Condition_Expression_RequestKubernetes_:
		if p.GetRequestKubernetes() == nil {
			return grpcutils.InvalidArg("Nil Kubernetes request expression")
		}

	case *enterprisev1.Condition_Expression_RequestKubernetesVerb_:
		return validateStringMatch(p.GetRequestKubernetesVerb().GetMatch(), "Kubernetes verb", nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesAPIPrefix_:
		return validateStringMatch(p.GetRequestKubernetesAPIPrefix().GetMatch(), "Kubernetes API prefix", nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesAPIGroup_:
		return validateStringMatch(p.GetRequestKubernetesAPIGroup().GetMatch(), "Kubernetes API group", nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesAPIVersion_:
		return validateStringMatch(p.GetRequestKubernetesAPIVersion().GetMatch(), "Kubernetes API version", nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesNamespace_:
		return validateStringMatch(p.GetRequestKubernetesNamespace().GetMatch(), "Kubernetes namespace", nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesResource_:
		return validateStringMatch(p.GetRequestKubernetesResource().GetMatch(), "Kubernetes resource", nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesSubresource_:
		return validateStringMatch(p.GetRequestKubernetesSubresource().GetMatch(), "Kubernetes subresource", nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesName_:
		return validateStringMatch(p.GetRequestKubernetesName().GetMatch(), "Kubernetes resource name", nil)

	case *enterprisev1.Condition_Expression_RequestGRPC_:
		if p.GetRequestGRPC() == nil {
			return grpcutils.InvalidArg("Nil gRPC request expression")
		}

	case *enterprisev1.Condition_Expression_RequestGRPCMethod_:
		return validateStringMatch(p.GetRequestGRPCMethod().GetMatch(), "gRPC method", nil)
	case *enterprisev1.Condition_Expression_RequestGRPCService_:
		return validateStringMatch(p.GetRequestGRPCService().GetMatch(), "gRPC service", nil)
	case *enterprisev1.Condition_Expression_RequestGRPCServiceFullName_:
		return validateStringMatch(p.GetRequestGRPCServiceFullName().GetMatch(), "gRPC full service name", nil)
	case *enterprisev1.Condition_Expression_RequestGRPCPackage_:
		return validateStringMatch(p.GetRequestGRPCPackage().GetMatch(), "gRPC package", nil)

	case *enterprisev1.Condition_Expression_RequestPostgresConnect_:
		if p.GetRequestPostgresConnect() == nil {
			return grpcutils.InvalidArg("Nil PostgreSQL connect expression")
		}
	case *enterprisev1.Condition_Expression_RequestPostgresConnectUser_:
		return validateStringMatch(p.GetRequestPostgresConnectUser().GetMatch(), "PostgreSQL user", nil)
	case *enterprisev1.Condition_Expression_RequestPostgresConnectDatabase_:
		return validateStringMatch(p.GetRequestPostgresConnectDatabase().GetMatch(), "PostgreSQL database", nil)
	case *enterprisev1.Condition_Expression_RequestPostgresConnectApplicationName_:
		return validateStringMatch(p.GetRequestPostgresConnectApplicationName().GetMatch(), "PostgreSQL application name", nil)
	case *enterprisev1.Condition_Expression_RequestPostgresQuery_:
		if p.GetRequestPostgresQuery() == nil {
			return grpcutils.InvalidArg("Nil PostgreSQL query expression")
		}
	case *enterprisev1.Condition_Expression_RequestPostgresQueryText_:
		return validateStringMatch(p.GetRequestPostgresQueryText().GetMatch(), "PostgreSQL query", nil)
	case *enterprisev1.Condition_Expression_RequestPostgresParse_:
		if p.GetRequestPostgresParse() == nil {
			return grpcutils.InvalidArg("Nil PostgreSQL parse expression")
		}
	case *enterprisev1.Condition_Expression_RequestPostgresParseName_:
		return validateStringMatch(p.GetRequestPostgresParseName().GetMatch(), "PostgreSQL parse name", nil)
	case *enterprisev1.Condition_Expression_RequestPostgresParseQuery_:
		return validateStringMatch(p.GetRequestPostgresParseQuery().GetMatch(), "PostgreSQL parse query", nil)

	case *enterprisev1.Condition_Expression_RequestDNS_:
		if p.GetRequestDNS() == nil {
			return grpcutils.InvalidArg("Nil DNS request expression")
		}
	case *enterprisev1.Condition_Expression_RequestDNSName_:
		return validateStringMatch(p.GetRequestDNSName().GetMatch(), "DNS name", nil)
	case *enterprisev1.Condition_Expression_RequestDNSTypeID_:
		return validateIntMatch(p.GetRequestDNSTypeID().GetMatch(), "DNS type ID")

	case *enterprisev1.Condition_Expression_RequestSOCKS5_:
		if p.GetRequestSOCKS5() == nil {
			return grpcutils.InvalidArg("Nil SOCKS5 request expression")
		}
	case *enterprisev1.Condition_Expression_RequestSOCKS5Host_:
		return validateStringMatch(p.GetRequestSOCKS5Host().GetMatch(), "SOCKS5 host", nil)
	case *enterprisev1.Condition_Expression_RequestSOCKS5Port_:
		return validateUIntMatch(p.GetRequestSOCKS5Port().GetMatch(), "SOCKS5 port")
	case *enterprisev1.Condition_Expression_RequestSOCKS5AddressType_:
		expr := p.GetRequestSOCKS5AddressType()
		if expr == nil || expr.GetAddressType() == corev1.RequestContext_Request_SOCKS5_Connect_ADDRESS_TYPE_UNSPECIFIED {
			return grpcutils.InvalidArg("SOCKS5 address type must be set")
		}
		if expr.GetAddressType() < corev1.RequestContext_Request_SOCKS5_Connect_ADDRESS_TYPE_UNSPECIFIED ||
			expr.GetAddressType() > corev1.RequestContext_Request_SOCKS5_Connect_IPV6 {
			return grpcutils.InvalidArg("Invalid SOCKS5 address type")
		}

	case *enterprisev1.Condition_Expression_RequestMCPProtocolVersion:
		return validateStringMatch(p.GetRequestMCPProtocolVersion().GetMatch(), "MCP protocol version", nil)
	case *enterprisev1.Condition_Expression_RequestMCPMethod:
		return validateStringMatch(p.GetRequestMCPMethod().GetMatch(), "MCP method", nil)
	case *enterprisev1.Condition_Expression_RequestMCPToolName:
		return validateStringMatch(p.GetRequestMCPToolName().GetMatch(), "MCP tool name", nil)
	case *enterprisev1.Condition_Expression_RequestMCPPromptName:
		return validateStringMatch(p.GetRequestMCPPromptName().GetMatch(), "MCP prompt name", nil)
	case *enterprisev1.Condition_Expression_RequestMCPResourceURI:
		return validateStringMatch(p.GetRequestMCPResourceURI().GetMatch(), "MCP resource URI", nil)
	case *enterprisev1.Condition_Expression_RequestMCPIsNotification:
		if p.GetRequestMCPIsNotification() == nil {
			return grpcutils.InvalidArg("Nil MCP notification expression")
		}

	case *enterprisev1.Condition_Expression_RequestLLMProtocol:
		expr := p.GetRequestLLMProtocol()
		if expr == nil || expr.GetProtocol() == corev1.Service_Spec_Config_LLM_PROTOCOL_UNSET {
			return grpcutils.InvalidArg("LLM protocol must be set")
		}
		if expr.GetProtocol() < corev1.Service_Spec_Config_LLM_PROTOCOL_UNSET ||
			expr.GetProtocol() > corev1.Service_Spec_Config_LLM_ANTHROPIC {
			return grpcutils.InvalidArg("Invalid LLM protocol")
		}
	case *enterprisev1.Condition_Expression_RequestLLMOperation:
		expr := p.GetRequestLLMOperation()
		if expr == nil || expr.GetOperation() == corev1.RequestContext_Request_LLM_OPERATION_UNSET {
			return grpcutils.InvalidArg("LLM operation must be set")
		}
		if expr.GetOperation() < corev1.RequestContext_Request_LLM_OPERATION_UNSET ||
			expr.GetOperation() > corev1.RequestContext_Request_LLM_COUNT_TOKENS {
			return grpcutils.InvalidArg("Invalid LLM operation")
		}
	case *enterprisev1.Condition_Expression_RequestLLMModel:
		return validateStringMatch(p.GetRequestLLMModel().GetMatch(), "LLM model", nil)
	case *enterprisev1.Condition_Expression_RequestLLMStream:
		if p.GetRequestLLMStream() == nil {
			return grpcutils.InvalidArg("Nil LLM stream expression")
		}
	case *enterprisev1.Condition_Expression_RequestLLMEstimatedInputTokens:
		expr := p.GetRequestLLMEstimatedInputTokens()
		if expr == nil {
			return grpcutils.InvalidArg("Nil LLM estimated input tokens expression")
		}
		return validateUIntMatch(expr.GetMatch(), "LLM estimated input tokens")
	case *enterprisev1.Condition_Expression_RequestLLMEstimateQuality:
		expr := p.GetRequestLLMEstimateQuality()
		if expr == nil || expr.GetQuality() == corev1.RequestContext_Request_LLM_ESTIMATE_QUALITY_UNSET {
			return grpcutils.InvalidArg("LLM estimate quality must be set")
		}
		if expr.GetQuality() < corev1.RequestContext_Request_LLM_ESTIMATE_QUALITY_UNSET ||
			expr.GetQuality() > corev1.RequestContext_Request_LLM_UNAVAILABLE {
			return grpcutils.InvalidArg("Invalid LLM estimate quality")
		}
	case *enterprisev1.Condition_Expression_RequestLLMMaxOutputTokens:
		return validateUIntMatch(p.GetRequestLLMMaxOutputTokens().GetMatch(), "LLM max output tokens")
	case *enterprisev1.Condition_Expression_RequestLLMHasTools:
		if p.GetRequestLLMHasTools() == nil {
			return grpcutils.InvalidArg("Nil LLM tools expression")
		}
	case *enterprisev1.Condition_Expression_RequestLLMToolCount:
		return validateUIntMatch(p.GetRequestLLMToolCount().GetMatch(), "LLM tool count")
	case *enterprisev1.Condition_Expression_RequestLLMToolName:
		return validateStringMatch(p.GetRequestLLMToolName().GetMatch(), "LLM tool name", nil)
	case *enterprisev1.Condition_Expression_RequestLLMInputItemCount:
		return validateUIntMatch(p.GetRequestLLMInputItemCount().GetMatch(), "LLM input item count")
	case *enterprisev1.Condition_Expression_RequestLLMHasImageInput:
		if p.GetRequestLLMHasImageInput() == nil {
			return grpcutils.InvalidArg("Nil LLM image input expression")
		}
	case *enterprisev1.Condition_Expression_RequestLLMHasAudioInput:
		if p.GetRequestLLMHasAudioInput() == nil {
			return grpcutils.InvalidArg("Nil LLM audio input expression")
		}

	case *enterprisev1.Condition_Expression_RequestIP_:
		expr := p.GetRequestIP()
		if expr == nil {
			return grpcutils.InvalidArg("Nil request IP expression")
		}
		if _, err := netip.ParseAddr(expr.Value); err != nil {
			return grpcutils.InvalidArg("Invalid IP address")
		}

	case *enterprisev1.Condition_Expression_RequestIPInRange_:
		expr := p.GetRequestIPInRange()
		if expr == nil {
			return grpcutils.InvalidArg("Nil request IP range expression")
		}
		if _, err := netip.ParsePrefix(expr.Value); err != nil {
			return grpcutils.InvalidArg("Invalid IP range")
		}

	case *enterprisev1.Condition_Expression_ApiServerReadOnlyMethods:
		if p.GetApiServerReadOnlyMethods() == nil {
			return grpcutils.InvalidArg("Nil API server read-only methods expression")
		}

	case *enterprisev1.Condition_Expression_ApiServerMethods:
		expr := p.GetApiServerMethods()
		if expr == nil {
			return grpcutils.InvalidArg("Nil API server methods expression")
		}
		if err := validateAPIServerStringList(expr.GetMethods(), "API server methods"); err != nil {
			return err
		}

	case *enterprisev1.Condition_Expression_ApiServerServices:
		expr := p.GetApiServerServices()
		if expr == nil {
			return grpcutils.InvalidArg("Nil API server services expression")
		}
		if err := validateAPIServerStringList(expr.GetServices(), "API server services"); err != nil {
			return err
		}

	case *enterprisev1.Condition_Expression_ApiServerCore:
		if p.GetApiServerCore() == nil {
			return grpcutils.InvalidArg("Nil API server core expression")
		}

	case *enterprisev1.Condition_Expression_ApiServerUser:
		if p.GetApiServerUser() == nil {
			return grpcutils.InvalidArg("Nil API server user expression")
		}

	case *enterprisev1.Condition_Expression_ApiServerEnterprise:
		expr := p.GetApiServerEnterprise()
		if expr == nil {
			return grpcutils.InvalidArg("Nil API server enterprise expression")
		}
		if expr.Service == 0 {
			return grpcutils.InvalidArg("API server enterprise service must be set")
		}
		if expr.Service < 0 || expr.Service > 4 {
			return grpcutils.InvalidArg("Invalid API server enterprise service")
		}

	case *enterprisev1.Condition_Expression_ApiServerCordium:
		expr := p.GetApiServerCordium()
		if expr == nil {
			return grpcutils.InvalidArg("Nil API server cordium expression")
		}
		if expr.Service == 0 {
			return grpcutils.InvalidArg("API server cordium service must be set")
		}
		if expr.Service < 0 || expr.Service > 3 {
			return grpcutils.InvalidArg("Invalid API server cordium service")
		}

	case *enterprisev1.Condition_Expression_ApiServerAccess:
		expr := p.GetApiServerAccess()
		if expr == nil {
			return grpcutils.InvalidArg("Nil API server access expression")
		}
		if expr.Service == 0 {
			return grpcutils.InvalidArg("API server access service must be set")
		}
		if expr.Service < 0 || expr.Service > 4 {
			return grpcutils.InvalidArg("Invalid API server access service")
		}

	default:
		return grpcutils.InvalidArg("Unsupported Expression type")
	}

	return nil
}

func (s *Server) toCoreCondition(in *enterprisev1.Condition) *corev1.Condition {
	if in == nil || in.Type == nil {
		return nil
	}

	switch in.Type.(type) {
	case *enterprisev1.Condition_All_:
		ret := &corev1.Condition{
			Type: &corev1.Condition_All_{
				All: &corev1.Condition_All{},
			},
		}
		for _, cond := range in.GetAll().Of {
			ret.GetAll().Of = append(ret.GetAll().Of, s.toCoreCondition(cond))
		}
		return ret

	case *enterprisev1.Condition_Any_:
		ret := &corev1.Condition{
			Type: &corev1.Condition_Any_{
				Any: &corev1.Condition_Any{},
			},
		}
		for _, cond := range in.GetAny().Of {
			ret.GetAny().Of = append(ret.GetAny().Of, s.toCoreCondition(cond))
		}
		return ret

	case *enterprisev1.Condition_None_:
		ret := &corev1.Condition{
			Type: &corev1.Condition_None_{
				None: &corev1.Condition_None{},
			},
		}
		for _, cond := range in.GetNone().Of {
			ret.GetNone().Of = append(ret.GetNone().Of, s.toCoreCondition(cond))
		}
		return ret

	case *enterprisev1.Condition_MatchAny:
		return &corev1.Condition{
			Type: &corev1.Condition_MatchAny{
				MatchAny: in.GetMatchAny(),
			},
		}

	case *enterprisev1.Condition_Expression_:
		return &corev1.Condition{
			Type: &corev1.Condition_Match{
				Match: s.getExpression(in.GetExpression()),
			},
		}

	case *enterprisev1.Condition_Not_:
		return &corev1.Condition{
			Type: &corev1.Condition_Not{
				Not: s.getExpression(in.GetNot().GetExpression()),
			},
		}

	default:
		return nil
	}
}

func (s *Server) getExpression(in *enterprisev1.Condition_Expression) string {
	if in == nil || in.Type == nil {
		return ""
	}

	getRefExpr := func(ctxField string, ref *metav1.ObjectReference) string {
		if ref == nil {
			return "false"
		}

		return fmt.Sprintf(`ctx.%s.metadata.name == %s`, ctxField, celString(ref.Name))
	}

	isAPIServer := `ctx.service.status.namespaceRef.name == "octelium-api" && ctx.service.spec.mode == "GRPC"`
	apiServerReadOnlyMethods := `["Get", "List"].exists(x, ctx.request.grpc.method.startsWith(x))`
	isAPIServerWithAPI := func(arg string) string {
		return andCEL(isAPIServer, fmt.Sprintf(`ctx.request.grpc.package == %s`, celString(fmt.Sprintf("octelium.api.main.%s.v1", arg))))
	}
	isAPIServerWithService := func(arg string) string {
		return fmt.Sprintf(`ctx.request.grpc.service == %s`, celString(arg))
	}

	switch in.Type.(type) {
	case *enterprisev1.Condition_Expression_Device_:
		return getRefExpr("device", in.GetDevice().GetDeviceRef())

	case *enterprisev1.Condition_Expression_User_:
		return getRefExpr("user", in.GetUser().GetUserRef())

	case *enterprisev1.Condition_Expression_Session_:
		return getRefExpr("session", in.GetSession().GetSessionRef())

	case *enterprisev1.Condition_Expression_Service_:
		return getRefExpr("service", in.GetService().GetServiceRef())

	case *enterprisev1.Condition_Expression_Namespace_:
		return getRefExpr("namespace", in.GetNamespace().GetNamespaceRef())

	case *enterprisev1.Condition_Expression_Group_:
		return fmt.Sprintf(`%s in ctx.user.spec.groups`, celString(in.GetGroup().GetGroupRef().Name))

	case *enterprisev1.Condition_Expression_ServiceMode_:
		return fmt.Sprintf(`ctx.service.spec.mode == %s`, celString(in.GetServiceMode().Mode.String()))

	case *enterprisev1.Condition_Expression_DeviceOSType_:
		return fmt.Sprintf(`ctx.device.status.osType == %s`, celString(in.GetDeviceOSType().OsType.String()))

	case *enterprisev1.Condition_Expression_ServicePublic_:
		return `ctx.service.spec.isPublic`

	case *enterprisev1.Condition_Expression_SessionAuthenticationAAL_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.aal == %s`,
			celString(in.GetSessionAuthenticationAAL().Aal.String()))

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredential_:
		ref := in.GetSessionAuthenticationCredential().GetCredentialRef()
		return fmt.Sprintf(`ctx.session.status.authentication.info.credential.credentialRef.uid == %s`,
			celString(ref.Uid))

	case *enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider_:
		ref := in.GetSessionAuthenticationIdentityProvider().GetIdentityProviderRef()
		return fmt.Sprintf(`ctx.session.status.authentication.info.identityProvider.identityProviderRef.uid == %s`,
			celString(ref.Uid))

	case *enterprisev1.Condition_Expression_SessionAuthenticationType_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.type == %s`,
			celString(in.GetSessionAuthenticationType().Type.String()))

	case *enterprisev1.Condition_Expression_SessionBrowser_:
		return `ctx.session.status.isBrowser`

	case *enterprisev1.Condition_Expression_SessionType_:
		return fmt.Sprintf(`ctx.session.status.type == %s`,
			celString(in.GetSessionType().Type.String()))

	case *enterprisev1.Condition_Expression_UserType_:
		return fmt.Sprintf(`ctx.user.spec.type == %s`,
			celString(in.GetUserType().Type.String()))

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.authenticator.info.fido.aaguid == %s`,
			celString(in.GetSessionAuthenticationCredAuthenticatorAAGUID().Aaguid))

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey_:
		return `ctx.session.status.authentication.info.authenticator.info.fido.isPasskey`

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified_:
		return `ctx.session.status.authentication.info.authenticator.info.fido.isAttestationVerified`

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware_:
		return `ctx.session.status.authentication.info.authenticator.info.fido.isHardware`

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent_:
		return `ctx.session.status.authentication.info.authenticator.info.fido.userPresent`

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified_:
		return `ctx.session.status.authentication.info.authenticator.info.fido.userVerified`

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredentialType_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.credential.type == %s`,
			celString(in.GetSessionAuthenticationCredentialType().Type.String()))

	case *enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.geoip.country.code == %s`,
			celString(in.GetSessionAuthenticationGeoipCountryCode().Code))

	case *enterprisev1.Condition_Expression_TimeAfter_:
		return fmt.Sprintf(`now() > timestamp(%s)`,
			celString(in.GetTimeAfter().Timestamp.AsTime().Format(time.RFC3339Nano)))

	case *enterprisev1.Condition_Expression_TimeBefore_:
		return fmt.Sprintf(`now() < timestamp(%s)`,
			celString(in.GetTimeBefore().Timestamp.AsTime().Format(time.RFC3339Nano)))

	case *enterprisev1.Condition_Expression_TimeDayType_:
		return timeDayTypeCEL(in.GetTimeDayType())

	case *enterprisev1.Condition_Expression_RequestHTTPPath_:
		return stringMatchCEL("ctx.request.http.path", in.GetRequestHTTPPath().GetMatch(), nil)

	case *enterprisev1.Condition_Expression_RequestHTTPHasHeader_:
		headerMatch := in.GetRequestHTTPHasHeader().GetMatch()
		if value, ok := exactStringMatchValue(headerMatch); ok {
			return fmt.Sprintf(`%s in ctx.request.http.headers`, celString(strings.ToLower(value)))
		}
		match := stringMatchCEL("k", headerMatch, strings.ToLower)
		return fmt.Sprintf(`ctx.request.http.headers.exists(k, %s)`, match)

	case *enterprisev1.Condition_Expression_RequestHTTPHeaderValue_:
		expr := in.GetRequestHTTPHeaderValue()
		if header, headerOK := exactStringMatchValue(expr.GetHeader()); headerOK {
			if value, valueOK := exactStringMatchValue(expr.GetValue()); valueOK {
				return fmt.Sprintf(`ctx.request.http.headers[%s] == %s`,
					celString(strings.ToLower(header)), celString(value))
			}
		}
		headerMatch := stringMatchCEL("k", expr.GetHeader(), strings.ToLower)
		valueMatch := stringMatchCEL("ctx.request.http.headers[k]", expr.GetValue(), nil)
		return fmt.Sprintf(`ctx.request.http.headers.exists(k, %s && %s)`, headerMatch, valueMatch)

	case *enterprisev1.Condition_Expression_RequestHTTPMethod_:
		return stringMatchCEL("ctx.request.http.method", in.GetRequestHTTPMethod().GetMatch(), strings.ToUpper)

	case *enterprisev1.Condition_Expression_RequestHTTPHost_:
		return stringMatchCEL("ctx.request.http.host", in.GetRequestHTTPHost().GetMatch(), nil)

	case *enterprisev1.Condition_Expression_RequestHTTPProtocol_:
		return stringMatchCEL("ctx.request.http.protocol", in.GetRequestHTTPProtocol().GetMatch(), nil)

	case *enterprisev1.Condition_Expression_RequestHTTPScheme_:
		return stringMatchCEL("ctx.request.http.scheme", in.GetRequestHTTPScheme().GetMatch(), strings.ToLower)

	case *enterprisev1.Condition_Expression_RequestHTTPURI_:
		return stringMatchCEL("ctx.request.http.uri", in.GetRequestHTTPURI().GetMatch(), nil)

	case *enterprisev1.Condition_Expression_RequestHTTPSize_:
		return intMatchCEL("ctx.request.http.size", in.GetRequestHTTPSize().GetMatch())

	case *enterprisev1.Condition_Expression_RequestHTTPHasQueryParam_:
		return fmt.Sprintf(`%s in ctx.request.http.queryParams`, celString(in.GetRequestHTTPHasQueryParam().GetName()))

	case *enterprisev1.Condition_Expression_RequestHTTPQueryParamValue_:
		expr := in.GetRequestHTTPQueryParamValue()
		valueMatch := stringMatchCEL(fmt.Sprintf(`ctx.request.http.queryParams[%s]`, celString(expr.GetName())), expr.GetMatch(), nil)
		return andCEL(fmt.Sprintf(`%s in ctx.request.http.queryParams`, celString(expr.GetName())), valueMatch)

	case *enterprisev1.Condition_Expression_RequestSSH_:
		return `has(ctx.request.ssh)`

	case *enterprisev1.Condition_Expression_RequestSSHUser_:
		return requestNestedStringMatchCEL("ssh", "connect", "user", in.GetRequestSSHUser().GetMatch(), nil)

	case *enterprisev1.Condition_Expression_RequestKubernetes_:
		return `has(ctx.request.kubernetes)`

	case *enterprisev1.Condition_Expression_RequestKubernetesVerb_:
		return requestStringMatchCEL("kubernetes", "verb", in.GetRequestKubernetesVerb().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesAPIPrefix_:
		return requestStringMatchCEL("kubernetes", "apiPrefix", in.GetRequestKubernetesAPIPrefix().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesAPIGroup_:
		return requestStringMatchCEL("kubernetes", "apiGroup", in.GetRequestKubernetesAPIGroup().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesAPIVersion_:
		return requestStringMatchCEL("kubernetes", "apiVersion", in.GetRequestKubernetesAPIVersion().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesNamespace_:
		return requestStringMatchCEL("kubernetes", "namespace", in.GetRequestKubernetesNamespace().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesResource_:
		return requestStringMatchCEL("kubernetes", "resource", in.GetRequestKubernetesResource().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesSubresource_:
		return requestStringMatchCEL("kubernetes", "subresource", in.GetRequestKubernetesSubresource().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestKubernetesName_:
		return requestStringMatchCEL("kubernetes", "name", in.GetRequestKubernetesName().GetMatch(), nil)

	case *enterprisev1.Condition_Expression_RequestGRPC_:
		return `has(ctx.request.grpc)`

	case *enterprisev1.Condition_Expression_RequestGRPCMethod_:
		return requestStringMatchCEL("grpc", "method", in.GetRequestGRPCMethod().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestGRPCService_:
		return requestStringMatchCEL("grpc", "service", in.GetRequestGRPCService().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestGRPCServiceFullName_:
		return requestStringMatchCEL("grpc", "serviceFullName", in.GetRequestGRPCServiceFullName().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestGRPCPackage_:
		return requestStringMatchCEL("grpc", "package", in.GetRequestGRPCPackage().GetMatch(), nil)

	case *enterprisev1.Condition_Expression_RequestPostgresConnect_:
		return `has(ctx.request.postgres.connect)`
	case *enterprisev1.Condition_Expression_RequestPostgresConnectUser_:
		return requestNestedStringMatchCEL("postgres", "connect", "user", in.GetRequestPostgresConnectUser().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestPostgresConnectDatabase_:
		return requestNestedStringMatchCEL("postgres", "connect", "database", in.GetRequestPostgresConnectDatabase().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestPostgresConnectApplicationName_:
		return requestNestedStringMatchCEL("postgres", "connect", "applicationName", in.GetRequestPostgresConnectApplicationName().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestPostgresQuery_:
		return `has(ctx.request.postgres.query)`
	case *enterprisev1.Condition_Expression_RequestPostgresQueryText_:
		return requestNestedStringMatchCEL("postgres", "query", "query", in.GetRequestPostgresQueryText().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestPostgresParse_:
		return `has(ctx.request.postgres.parse)`
	case *enterprisev1.Condition_Expression_RequestPostgresParseName_:
		return requestNestedStringMatchCEL("postgres", "parse", "name", in.GetRequestPostgresParseName().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestPostgresParseQuery_:
		return requestNestedStringMatchCEL("postgres", "parse", "query", in.GetRequestPostgresParseQuery().GetMatch(), nil)

	case *enterprisev1.Condition_Expression_RequestDNS_:
		return `has(ctx.request.dns)`
	case *enterprisev1.Condition_Expression_RequestDNSName_:
		return requestStringMatchCEL("dns", "name", in.GetRequestDNSName().GetMatch(), strings.ToLower)
	case *enterprisev1.Condition_Expression_RequestDNSTypeID_:
		return requestIntMatchCEL("dns", "typeID", in.GetRequestDNSTypeID().GetMatch())

	case *enterprisev1.Condition_Expression_RequestSOCKS5_:
		return `has(ctx.request.socks5)`
	case *enterprisev1.Condition_Expression_RequestSOCKS5Host_:
		return requestNestedStringMatchCEL("socks5", "connect", "host", in.GetRequestSOCKS5Host().GetMatch(), strings.ToLower)
	case *enterprisev1.Condition_Expression_RequestSOCKS5Port_:
		return requestNestedUIntMatchCEL("socks5", "connect", "port", in.GetRequestSOCKS5Port().GetMatch())
	case *enterprisev1.Condition_Expression_RequestSOCKS5AddressType_:
		return andCEL(`has(ctx.request.socks5.connect)`, fmt.Sprintf(`ctx.request.socks5.connect.addressType == %s`, celString(in.GetRequestSOCKS5AddressType().GetAddressType().String())))

	case *enterprisev1.Condition_Expression_RequestMCPProtocolVersion:
		return requestStringMatchCEL("mcp", "protocolVersion", in.GetRequestMCPProtocolVersion().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestMCPMethod:
		return requestStringMatchCEL("mcp", "method", in.GetRequestMCPMethod().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestMCPToolName:
		return andCEL(`has(ctx.request.mcp)`, `ctx.request.mcp.method == "tools/call"`, stringMatchCEL("ctx.request.mcp.name", in.GetRequestMCPToolName().GetMatch(), nil))
	case *enterprisev1.Condition_Expression_RequestMCPPromptName:
		return andCEL(`has(ctx.request.mcp)`, `ctx.request.mcp.method == "prompts/get"`, stringMatchCEL("ctx.request.mcp.name", in.GetRequestMCPPromptName().GetMatch(), nil))
	case *enterprisev1.Condition_Expression_RequestMCPResourceURI:
		return andCEL(`has(ctx.request.mcp)`, `ctx.request.mcp.method == "resources/read"`, stringMatchCEL("ctx.request.mcp.name", in.GetRequestMCPResourceURI().GetMatch(), nil))
	case *enterprisev1.Condition_Expression_RequestMCPIsNotification:
		return `has(ctx.request.mcp) && ctx.request.mcp.isNotification`

	case *enterprisev1.Condition_Expression_RequestLLMProtocol:
		return requestEnumCEL("llm", "protocol", in.GetRequestLLMProtocol().GetProtocol().String())
	case *enterprisev1.Condition_Expression_RequestLLMOperation:
		return requestEnumCEL("llm", "operation", in.GetRequestLLMOperation().GetOperation().String())
	case *enterprisev1.Condition_Expression_RequestLLMModel:
		return requestStringMatchCEL("llm", "model", in.GetRequestLLMModel().GetMatch(), nil)
	case *enterprisev1.Condition_Expression_RequestLLMStream:
		return `has(ctx.request.llm) && ctx.request.llm.stream`
	case *enterprisev1.Condition_Expression_RequestLLMEstimatedInputTokens:
		expr := in.GetRequestLLMEstimatedInputTokens()
		ret := requestUIntMatchCEL("llm", "estimatedInputTokens", expr.GetMatch())
		if expr.GetRequireComplete() {
			ret = andCEL(ret, `ctx.request.llm.estimateQuality == "COMPLETE"`)
		}
		return ret
	case *enterprisev1.Condition_Expression_RequestLLMEstimateQuality:
		return requestEnumCEL("llm", "estimateQuality", in.GetRequestLLMEstimateQuality().GetQuality().String())
	case *enterprisev1.Condition_Expression_RequestLLMMaxOutputTokens:
		return requestUIntMatchCEL("llm", "maxOutputTokens", in.GetRequestLLMMaxOutputTokens().GetMatch())
	case *enterprisev1.Condition_Expression_RequestLLMHasTools:
		return `has(ctx.request.llm) && ctx.request.llm.hasTools`
	case *enterprisev1.Condition_Expression_RequestLLMToolCount:
		return requestUIntMatchCEL("llm", "toolCount", in.GetRequestLLMToolCount().GetMatch())
	case *enterprisev1.Condition_Expression_RequestLLMToolName:
		return andCEL(`has(ctx.request.llm)`, fmt.Sprintf(`ctx.request.llm.toolNames.exists(x, %s)`, stringMatchCEL("x", in.GetRequestLLMToolName().GetMatch(), nil)))
	case *enterprisev1.Condition_Expression_RequestLLMInputItemCount:
		return requestUIntMatchCEL("llm", "inputItemCount", in.GetRequestLLMInputItemCount().GetMatch())
	case *enterprisev1.Condition_Expression_RequestLLMHasImageInput:
		return `has(ctx.request.llm) && ctx.request.llm.hasImageInput`
	case *enterprisev1.Condition_Expression_RequestLLMHasAudioInput:
		return `has(ctx.request.llm) && ctx.request.llm.hasAudioInput`

	case *enterprisev1.Condition_Expression_RequestIP_:
		return fmt.Sprintf(`ctx.request.ip == %s`, celString(in.GetRequestIP().Value))

	case *enterprisev1.Condition_Expression_RequestIPInRange_:
		return fmt.Sprintf(`net.isIPInRange(ctx.request.ip, %s)`, celString(in.GetRequestIPInRange().Value))

	case *enterprisev1.Condition_Expression_ApiServerReadOnlyMethods:
		return andCEL(isAPIServer, apiServerReadOnlyMethods)

	case *enterprisev1.Condition_Expression_ApiServerMethods:
		return andCEL(isAPIServer,
			fmt.Sprintf(`ctx.request.grpc.method in %s`, celStringList(in.GetApiServerMethods().GetMethods())))

	case *enterprisev1.Condition_Expression_ApiServerServices:
		return andCEL(isAPIServer,
			fmt.Sprintf(`ctx.request.grpc.service in %s`, celStringList(in.GetApiServerServices().GetServices())))

	case *enterprisev1.Condition_Expression_ApiServerCore:
		ret := isAPIServerWithAPI("core")
		if in.GetApiServerCore().GetReadOnlyMethods() {
			return andCEL(ret, apiServerReadOnlyMethods)
		}
		return ret

	case *enterprisev1.Condition_Expression_ApiServerUser:
		ret := isAPIServerWithAPI("user")
		if in.GetApiServerUser().GetReadOnlyMethods() {
			return andCEL(ret, apiServerReadOnlyMethods)
		}
		return ret

	case *enterprisev1.Condition_Expression_ApiServerEnterprise:
		ret := isAPIServerWithAPI("enterprise")
		switch int32(in.GetApiServerEnterprise().Service) {
		case 1:
			return ret
		case 2:
			return andCEL(ret, isAPIServerWithService("MainService"))
		case 3:
			return andCEL(ret, isAPIServerWithService("ClusterService"))
		case 4:
			return andCEL(ret, isAPIServerWithService("PolicyPortalService"))
		default:
			return "false"
		}

	case *enterprisev1.Condition_Expression_ApiServerCordium:
		ret := isAPIServerWithAPI("cordium")
		switch int32(in.GetApiServerCordium().Service) {
		case 1:
			return andCEL(ret, isAPIServerWithService("MainService"))
		case 2:
			return andCEL(ret, isAPIServerWithService("ManagementService"))
		case 3:
			return andCEL(ret, isAPIServerWithService("WorkspaceService"))
		default:
			return "false"
		}

	case *enterprisev1.Condition_Expression_ApiServerAccess:
		ret := isAPIServerWithAPI("access")
		switch int32(in.GetApiServerAccess().Service) {
		case 1:
			return ret
		case 2:
			return andCEL(ret, isAPIServerWithService("MainService"))
		case 3:
			return andCEL(ret, isAPIServerWithService("UserService"))
		case 4:
			return andCEL(ret, isAPIServerWithService("ReviewerService"))
		default:
			return "false"
		}

	default:
		return ""
	}
}

func (s *Server) GetCoreCondition(ctx context.Context, req *enterprisev1.Condition) (*corev1.Condition, error) {
	if err := s.validateCondition(ctx, req); err != nil {
		return nil, err
	}

	return s.toCoreCondition(req), nil
}

func (s *Server) validateUserRef(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}
	_, err := s.octeliumC.CoreC().GetUser(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	return err
}

func (s *Server) validateGroupRef(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}
	_, err := s.octeliumC.CoreC().GetGroup(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	return err
}

func (s *Server) validateDeviceRef(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}
	_, err := s.octeliumC.CoreC().GetDevice(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	return err
}

func (s *Server) validateSessionRef(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}
	_, err := s.octeliumC.CoreC().GetSession(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	return err
}

func (s *Server) validateServiceRef(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 2,
	}); err != nil {
		return err
	}
	_, err := s.octeliumC.CoreC().GetService(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	return err
}

func (s *Server) validateNamespaceRef(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}
	_, err := s.octeliumC.CoreC().GetNamespace(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	return err
}

func (s *Server) validateCredentialRef(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}
	_, err := s.octeliumC.CoreC().GetCredential(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	return err
}

func (s *Server) validateIdentityProviderRef(ctx context.Context, ref *metav1.ObjectReference) error {
	if err := apivalidation.CheckObjectRef(ref, &apivalidation.CheckGetOptionsOpts{}); err != nil {
		return err
	}
	_, err := s.octeliumC.CoreC().GetIdentityProvider(ctx, apivalidation.ObjectReferenceToRGetOptions(ref))
	return err
}

func celString(v string) string {
	return fmt.Sprintf("%q", v)
}

func celStringList(items []string) string {
	ret := make([]string, 0, len(items))
	for _, item := range items {
		ret = append(ret, celString(item))
	}
	return fmt.Sprintf("[%s]", strings.Join(ret, ", "))
}

func stringMatchCEL(field string, match *enterprisev1.Condition_Expression_StringMatch, normalize func(string) string) string {
	if match == nil || match.GetType() == nil {
		return "false"
	}

	normalizeValue := func(value string) string {
		if normalize != nil {
			return normalize(value)
		}
		return value
	}

	switch m := match.GetType().(type) {
	case *enterprisev1.Condition_Expression_StringMatch_Exact:
		return fmt.Sprintf(`%s == %s`, field, celString(normalizeValue(m.Exact)))
	case *enterprisev1.Condition_Expression_StringMatch_Prefix:
		return fmt.Sprintf(`%s.startsWith(%s)`, field, celString(normalizeValue(m.Prefix)))
	case *enterprisev1.Condition_Expression_StringMatch_Suffix:
		return fmt.Sprintf(`%s.endsWith(%s)`, field, celString(normalizeValue(m.Suffix)))
	case *enterprisev1.Condition_Expression_StringMatch_Contains:
		return fmt.Sprintf(`%s.contains(%s)`, field, celString(normalizeValue(m.Contains)))
	case *enterprisev1.Condition_Expression_StringMatch_In_:
		if m.In == nil {
			return "false"
		}
		values := make([]string, 0, len(m.In.Values))
		for _, value := range m.In.Values {
			values = append(values, normalizeValue(value))
		}
		return fmt.Sprintf(`%s in %s`, field, celStringList(values))
	default:
		return "false"
	}
}

func exactStringMatchValue(match *enterprisev1.Condition_Expression_StringMatch) (string, bool) {
	if match == nil {
		return "", false
	}
	value, ok := match.GetType().(*enterprisev1.Condition_Expression_StringMatch_Exact)
	if !ok {
		return "", false
	}
	return value.Exact, true
}

func uintMatchCEL(field string, match *enterprisev1.Condition_Expression_UIntMatch) string {
	if match == nil || match.GetType() == nil {
		return "false"
	}

	switch m := match.GetType().(type) {
	case *enterprisev1.Condition_Expression_UIntMatch_Exact:
		return fmt.Sprintf(`%s == uint(%d)`, field, m.Exact)
	case *enterprisev1.Condition_Expression_UIntMatch_LessThan:
		return fmt.Sprintf(`%s < uint(%d)`, field, m.LessThan)
	case *enterprisev1.Condition_Expression_UIntMatch_LessThanOrEqual:
		return fmt.Sprintf(`%s <= uint(%d)`, field, m.LessThanOrEqual)
	case *enterprisev1.Condition_Expression_UIntMatch_GreaterThan:
		return fmt.Sprintf(`%s > uint(%d)`, field, m.GreaterThan)
	case *enterprisev1.Condition_Expression_UIntMatch_GreaterThanOrEqual:
		return fmt.Sprintf(`%s >= uint(%d)`, field, m.GreaterThanOrEqual)
	default:
		return "false"
	}
}

func intMatchCEL(field string, match *enterprisev1.Condition_Expression_IntMatch) string {
	if match == nil || match.GetType() == nil {
		return "false"
	}

	switch m := match.GetType().(type) {
	case *enterprisev1.Condition_Expression_IntMatch_Exact:
		return fmt.Sprintf(`%s == %d`, field, m.Exact)
	case *enterprisev1.Condition_Expression_IntMatch_LessThan:
		return fmt.Sprintf(`%s < %d`, field, m.LessThan)
	case *enterprisev1.Condition_Expression_IntMatch_LessThanOrEqual:
		return fmt.Sprintf(`%s <= %d`, field, m.LessThanOrEqual)
	case *enterprisev1.Condition_Expression_IntMatch_GreaterThan:
		return fmt.Sprintf(`%s > %d`, field, m.GreaterThan)
	case *enterprisev1.Condition_Expression_IntMatch_GreaterThanOrEqual:
		return fmt.Sprintf(`%s >= %d`, field, m.GreaterThanOrEqual)
	default:
		return "false"
	}
}

func requestStringMatchCEL(requestType, field string, match *enterprisev1.Condition_Expression_StringMatch, normalize func(string) string) string {
	return andCEL(
		fmt.Sprintf(`has(ctx.request.%s)`, requestType),
		stringMatchCEL(fmt.Sprintf(`ctx.request.%s.%s`, requestType, field), match, normalize),
	)
}

func requestNestedStringMatchCEL(requestType, nestedType, field string, match *enterprisev1.Condition_Expression_StringMatch, normalize func(string) string) string {
	return andCEL(
		fmt.Sprintf(`has(ctx.request.%s)`, requestType),
		fmt.Sprintf(`has(ctx.request.%s.%s)`, requestType, nestedType),
		stringMatchCEL(fmt.Sprintf(`ctx.request.%s.%s.%s`, requestType, nestedType, field), match, normalize),
	)
}

func requestUIntMatchCEL(requestType, field string, match *enterprisev1.Condition_Expression_UIntMatch) string {
	return andCEL(
		fmt.Sprintf(`has(ctx.request.%s)`, requestType),
		uintMatchCEL(fmt.Sprintf(`ctx.request.%s.%s`, requestType, field), match),
	)
}

func requestNestedUIntMatchCEL(requestType, nestedType, field string, match *enterprisev1.Condition_Expression_UIntMatch) string {
	return andCEL(
		fmt.Sprintf(`has(ctx.request.%s)`, requestType),
		fmt.Sprintf(`has(ctx.request.%s.%s)`, requestType, nestedType),
		uintMatchCEL(fmt.Sprintf(`ctx.request.%s.%s.%s`, requestType, nestedType, field), match),
	)
}

func requestIntMatchCEL(requestType, field string, match *enterprisev1.Condition_Expression_IntMatch) string {
	return andCEL(
		fmt.Sprintf(`has(ctx.request.%s)`, requestType),
		intMatchCEL(fmt.Sprintf(`ctx.request.%s.%s`, requestType, field), match),
	)
}

func requestEnumCEL(requestType, field, value string) string {
	return andCEL(
		fmt.Sprintf(`has(ctx.request.%s)`, requestType),
		fmt.Sprintf(`ctx.request.%s.%s == %s`, requestType, field, celString(value)),
	)
}

func andCEL(items ...string) string {
	if len(items) == 0 {
		return "true"
	}

	ret := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		ret = append(ret, fmt.Sprintf("(%s)", item))
	}

	if len(ret) == 0 {
		return "true"
	}

	return strings.Join(ret, " && ")
}

func timeDayTypeCEL(expr *enterprisev1.Condition_Expression_TimeDayType) string {
	weekday := expr.GetType() == enterprisev1.Condition_Expression_TimeDayType_WEEKDAY
	baseFunc := "time.isWeekend"
	tzFunc := "time.isWeekendInTZ"
	if weekday {
		baseFunc = "time.isWeekday"
		tzFunc = "time.isWeekdayInTZ"
	}

	if expr.GetTimezone() == "" {
		return fmt.Sprintf(`%s(now())`, baseFunc)
	}
	return fmt.Sprintf(`%s(now(), %s)`, tzFunc, celString(expr.GetTimezone()))
}

func validateBoundedString(v string, required bool, field string) error {
	if v == "" {
		if required {
			return grpcutils.InvalidArg("%s is required", field)
		}
		return nil
	}

	if len(v) > maxConditionStringBytes {
		return grpcutils.InvalidArg("%s is too long", field)
	}

	return nil
}

func validateStringMatch(match *enterprisev1.Condition_Expression_StringMatch, field string, validate func(string) error) error {
	if match == nil || match.GetType() == nil {
		return grpcutils.InvalidArg("%s matcher must be set", field)
	}

	values := []string{}
	switch m := match.GetType().(type) {
	case *enterprisev1.Condition_Expression_StringMatch_Exact:
		values = append(values, m.Exact)
	case *enterprisev1.Condition_Expression_StringMatch_Prefix:
		values = append(values, m.Prefix)
	case *enterprisev1.Condition_Expression_StringMatch_Suffix:
		values = append(values, m.Suffix)
	case *enterprisev1.Condition_Expression_StringMatch_Contains:
		values = append(values, m.Contains)
	case *enterprisev1.Condition_Expression_StringMatch_In_:
		if m.In == nil || len(m.In.Values) == 0 {
			return grpcutils.InvalidArg("%s in matcher must contain at least one value", field)
		}
		if len(m.In.Values) > maxConditionChildren {
			return grpcutils.InvalidArg("%s in matcher has too many values", field)
		}
		values = append(values, m.In.Values...)
	default:
		return grpcutils.InvalidArg("Unsupported %s matcher", field)
	}

	seen := map[string]struct{}{}
	for _, value := range values {
		if err := validateBoundedString(value, false, field); err != nil {
			return err
		}
		if validate != nil {
			if err := validate(value); err != nil {
				return err
			}
		}
		if _, ok := seen[value]; ok {
			return grpcutils.InvalidArg("Duplicate %s matcher value: %s", field, value)
		}
		seen[value] = struct{}{}
	}

	return nil
}

func validateHTTPPathMatch(match *enterprisev1.Condition_Expression_StringMatch) error {
	if match == nil || match.GetType() == nil {
		return grpcutils.InvalidArg("HTTP path matcher must be set")
	}

	switch m := match.GetType().(type) {
	case *enterprisev1.Condition_Expression_StringMatch_Exact:
		return validateHTTPPath(m.Exact)
	case *enterprisev1.Condition_Expression_StringMatch_Prefix:
		return validateHTTPPath(m.Prefix)
	case *enterprisev1.Condition_Expression_StringMatch_Suffix:
		return validateHTTPPathFragment(m.Suffix)
	case *enterprisev1.Condition_Expression_StringMatch_Contains:
		return validateHTTPPathFragment(m.Contains)
	case *enterprisev1.Condition_Expression_StringMatch_In_:
		if err := validateStringMatch(match, "HTTP path", nil); err != nil {
			return err
		}
		for _, value := range m.In.Values {
			if err := validateHTTPPath(value); err != nil {
				return err
			}
		}
		return nil
	default:
		return grpcutils.InvalidArg("Unsupported HTTP path matcher")
	}
}

func validateHTTPPathFragment(v string) error {
	if err := validateBoundedString(v, true, "HTTP path"); err != nil {
		return err
	}
	if strings.ContainsAny(v, "\x00\r\n") {
		return grpcutils.InvalidArg("HTTP path contains invalid characters")
	}
	return nil
}

func validateHTTPMethod(v string) error {
	if err := validateBoundedString(v, true, "HTTP method"); err != nil {
		return err
	}
	if !isHTTPToken(v) {
		return grpcutils.InvalidArg("Invalid HTTP method")
	}
	return nil
}

func validateHTTPMethodMatch(match *enterprisev1.Condition_Expression_StringMatch) error {
	if err := validateStringMatch(match, "HTTP method", validateHTTPMethod); err != nil {
		return err
	}

	var values []string
	switch m := match.GetType().(type) {
	case *enterprisev1.Condition_Expression_StringMatch_Exact:
		values = []string{m.Exact}
	case *enterprisev1.Condition_Expression_StringMatch_In_:
		values = m.In.Values
	default:
		return nil
	}

	for _, value := range values {
		switch strings.ToUpper(value) {
		case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE":
		default:
			return grpcutils.InvalidArg("Unsupported HTTP method")
		}
	}
	return nil
}

func validateUIntMatch(match *enterprisev1.Condition_Expression_UIntMatch, field string) error {
	if match == nil || match.GetType() == nil {
		return grpcutils.InvalidArg("%s matcher must be set", field)
	}
	return nil
}

func validateIntMatch(match *enterprisev1.Condition_Expression_IntMatch, field string) error {
	if match == nil || match.GetType() == nil {
		return grpcutils.InvalidArg("%s matcher must be set", field)
	}
	return nil
}

func validateHTTPPath(v string) error {
	if err := validateBoundedString(v, true, "HTTP path"); err != nil {
		return err
	}

	if !strings.HasPrefix(v, "/") {
		return grpcutils.InvalidArg("HTTP path must start with /")
	}

	if strings.ContainsAny(v, "\x00\r\n") {
		return grpcutils.InvalidArg("HTTP path contains invalid characters")
	}

	return nil
}

func validateHTTPHeaderName(v string) error {
	if err := validateBoundedString(v, true, "HTTP header name"); err != nil {
		return err
	}

	if !isHTTPToken(v) {
		return grpcutils.InvalidArg("Invalid HTTP header name")
	}

	return nil
}

func validateAPIServerStringList(items []string, field string) error {
	if len(items) == 0 {
		return grpcutils.InvalidArg("%s must contain at least one value", field)
	}

	if len(items) > maxConditionChildren {
		return grpcutils.InvalidArg("%s has too many values", field)
	}

	seen := map[string]struct{}{}
	for _, item := range items {
		if err := validateAPIServerString(item, field); err != nil {
			return err
		}

		if _, ok := seen[item]; ok {
			return grpcutils.InvalidArg("Duplicate %s value: %s", field, item)
		}
		seen[item] = struct{}{}
	}

	return nil
}

func validateAPIServerString(v string, field string) error {
	if err := validateBoundedString(v, true, field); err != nil {
		return err
	}

	for _, r := range v {
		if r > 127 {
			return grpcutils.InvalidArg("%s contains invalid characters", field)
		}

		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '.' || r == '-':
		default:
			return grpcutils.InvalidArg("%s contains invalid characters", field)
		}
	}

	return nil
}

func isHTTPToken(v string) bool {
	if v == "" {
		return false
	}

	for _, r := range v {
		if r > 127 {
			return false
		}

		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", r):
		default:
			return false
		}
	}

	return true
}
