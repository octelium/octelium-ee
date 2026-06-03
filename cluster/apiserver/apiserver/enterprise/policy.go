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
	"net/http"
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
		if cond == nil || cond.Condition == nil {
			return grpcutils.InvalidArg("NOT condition must contain a child condition")
		}
		if err := s.validateConditionDepth(ctx, cond.Condition, depth+1); err != nil {
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

	case *enterprisev1.Condition_Expression_RequestHTTPPathExact_:
		expr := p.GetRequestHTTPPathExact()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP path exact expression")
		}
		if err := validateHTTPPath(expr.Value); err != nil {
			return err
		}

	case *enterprisev1.Condition_Expression_RequestHTTPPathPrefix_:
		expr := p.GetRequestHTTPPathPrefix()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP path prefix expression")
		}
		if err := validateHTTPPath(expr.Value); err != nil {
			return err
		}

	case *enterprisev1.Condition_Expression_RequestHTTPHasHeader_:
		expr := p.GetRequestHTTPHasHeader()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP header expression")
		}
		if err := validateHTTPHeaderName(expr.Value); err != nil {
			return err
		}

	case *enterprisev1.Condition_Expression_RequestHTTPHeaderValue_:
		expr := p.GetRequestHTTPHeaderValue()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP header value expression")
		}
		if err := validateHTTPHeaderName(expr.Header); err != nil {
			return err
		}
		if err := validateBoundedString(expr.Value, false, "HTTP header value"); err != nil {
			return err
		}

	case *enterprisev1.Condition_Expression_RequestHTTPMethod_:
		expr := p.GetRequestHTTPMethod()
		if expr == nil {
			return grpcutils.InvalidArg("Nil HTTP method expression")
		}

		switch strings.ToUpper(expr.Value) {
		case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE":
		default:
			return grpcutils.InvalidArg("Unsupported HTTP method")
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

	case *enterprisev1.Condition_Expression_ApiServer,
		*enterprisev1.Condition_Expression_ApiServerCore,
		*enterprisev1.Condition_Expression_ApiServerUser,
		*enterprisev1.Condition_Expression_ApiServerEnterprise,
		*enterprisev1.Condition_Expression_ApiServerCordium:
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

		switch in.GetNot().Condition.Type.(type) {
		case *enterprisev1.Condition_Expression_:
			return &corev1.Condition{
				Type: &corev1.Condition_Not{
					Not: s.getExpression(in.GetNot().GetCondition().GetExpression()),
				},
			}
		default:
			return nil
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

	isAPIServer := `ctx.service.systemLabels["octelium-apiserver"] == "true" && ctx.service.status.namespaceRef.name == "octelium-api"`
	isAPServerWithAPI := func(arg string) string {
		return fmt.Sprintf(`ctx.request.grpc.package == %s`, celString(fmt.Sprintf("octelium.api.main.%s.v1", arg)))
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
		if in.GetServicePublic().IsPublic {
			return `ctx.service.spec.isPublic`
		}
		return `!ctx.service.spec.isPublic`

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
		if in.GetSessionBrowser().IsBrowser {
			return `ctx.session.status.isBrowser`
		}
		return `!ctx.session.status.isBrowser`

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
		if in.GetSessionAuthenticationCredAuthenticatorFIDOPasskey().IsPasskey {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isPasskey`
		}
		return `!ctx.session.status.authentication.info.authenticator.info.fido.isPasskey`

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified_:
		if in.GetSessionAuthenticationCredAuthenticatorFIDOAttestationVerified().IsAttestationVerified {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isAttestationVerified`
		}
		return `!ctx.session.status.authentication.info.authenticator.info.fido.isAttestationVerified`

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware_:
		if in.GetSessionAuthenticationCredAuthenticatorFIDOHardware().IsHardware {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isHardware`
		}
		return `!ctx.session.status.authentication.info.authenticator.info.fido.isHardware`

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent_:
		if in.GetSessionAuthenticationCredAuthenticatorFIDOUserPresent().IsUserPresent {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isUserPresent`
		}
		return `!ctx.session.status.authentication.info.authenticator.info.fido.isUserPresent`

	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified_:
		if in.GetSessionAuthenticationCredAuthenticatorFIDOUserVerified().IsUserVerified {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isUserVerified`
		}
		return `!ctx.session.status.authentication.info.authenticator.info.fido.isUserVerified`

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

	case *enterprisev1.Condition_Expression_RequestHTTPPathExact_:
		return fmt.Sprintf(`ctx.request.http.path == %s`, celString(in.GetRequestHTTPPathExact().Value))

	case *enterprisev1.Condition_Expression_RequestHTTPPathPrefix_:
		return fmt.Sprintf(`ctx.request.http.path.startsWith(%s)`, celString(in.GetRequestHTTPPathPrefix().Value))

	case *enterprisev1.Condition_Expression_ApiServer:
		return isAPIServer

	case *enterprisev1.Condition_Expression_ApiServerCore:
		return fmt.Sprintf("%s && %s", isAPIServer, isAPServerWithAPI("core"))

	case *enterprisev1.Condition_Expression_ApiServerUser:
		return fmt.Sprintf("%s && %s", isAPIServer, isAPServerWithAPI("user"))

	case *enterprisev1.Condition_Expression_ApiServerEnterprise:
		return fmt.Sprintf("%s && %s", isAPIServer, isAPServerWithAPI("enterprise"))

	case *enterprisev1.Condition_Expression_ApiServerCordium:
		return fmt.Sprintf("%s && %s", isAPIServer, isAPServerWithAPI("cordium"))

	case *enterprisev1.Condition_Expression_RequestHTTPHasHeader_:
		return fmt.Sprintf(`%s in ctx.request.http.headers`,
			celString(strings.ToLower(in.GetRequestHTTPHasHeader().Value)))

	case *enterprisev1.Condition_Expression_RequestHTTPHeaderValue_:
		return fmt.Sprintf(`ctx.request.http.headers[%s] == %s`,
			celString(http.CanonicalHeaderKey(in.GetRequestHTTPHeaderValue().Header)),
			celString(in.GetRequestHTTPHeaderValue().Value))

	case *enterprisev1.Condition_Expression_RequestHTTPMethod_:
		return fmt.Sprintf(`ctx.request.http.method == %s`,
			celString(strings.ToUpper(in.GetRequestHTTPMethod().Value)))

	case *enterprisev1.Condition_Expression_RequestIP_:
		return fmt.Sprintf(`ctx.request.ip == %s`, celString(in.GetRequestIP().Value))

	case *enterprisev1.Condition_Expression_RequestIPInRange_:
		return fmt.Sprintf(`net.isIPInRange(ctx.request.ip, %s)`, celString(in.GetRequestIPInRange().Value))

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
