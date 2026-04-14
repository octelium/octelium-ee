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
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

func (s *Server) validateCondition(ctx context.Context, c *enterprisev1.Condition) error {
	if c == nil {
		return grpcutils.InvalidArg("Nil Condition")
	}

	switch c.Type.(type) {
	case *enterprisev1.Condition_All_:
		for _, cond := range c.GetAll().Of {
			if err := s.validateCondition(ctx, cond); err != nil {
				return err
			}
		}
	case *enterprisev1.Condition_Any_:
		for _, cond := range c.GetAny().Of {
			if err := s.validateCondition(ctx, cond); err != nil {
				return err
			}
		}
	case *enterprisev1.Condition_MatchAny:
	case *enterprisev1.Condition_None_:
		for _, cond := range c.GetNone().Of {
			if err := s.validateCondition(ctx, cond); err != nil {
				return err
			}
		}
	case *enterprisev1.Condition_Not_:
		if err := s.validateCondition(ctx, c.GetNot().Condition); err != nil {
			return err
		}
	case *enterprisev1.Condition_Expression_:
		if err := s.validateExpression(ctx, c.GetExpression()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) validateExpression(ctx context.Context, p *enterprisev1.Condition_Expression) error {
	if p == nil {
		return grpcutils.InvalidArg("Nil Expression")
	}

	switch p.Type.(type) {
	case *enterprisev1.Condition_Expression_User_:
		if _, err := s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid:  p.GetUser().UserRef.Uid,
			Name: p.GetUser().UserRef.Name,
		}); err != nil {
			return err
		}
	case *enterprisev1.Condition_Expression_Group_:
		if _, err := s.octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
			Uid:  p.GetGroup().GroupRef.Uid,
			Name: p.GetGroup().GroupRef.Name,
		}); err != nil {
			return err
		}
	case *enterprisev1.Condition_Expression_Device_:
		if _, err := s.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{
			Uid:  p.GetDevice().DeviceRef.Uid,
			Name: p.GetDevice().DeviceRef.Name,
		}); err != nil {
			return err
		}
	case *enterprisev1.Condition_Expression_Session_:
		if _, err := s.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{
			Uid:  p.GetSession().SessionRef.Uid,
			Name: p.GetSession().SessionRef.Name,
		}); err != nil {
			return err
		}
	case *enterprisev1.Condition_Expression_Service_:
		if _, err := s.octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
			Uid:  p.GetService().ServiceRef.Uid,
			Name: p.GetService().ServiceRef.Name,
		}); err != nil {
			return err
		}
	case *enterprisev1.Condition_Expression_Namespace_:
		if _, err := s.octeliumC.CoreC().GetNamespace(ctx, &rmetav1.GetOptions{
			Uid:  p.GetNamespace().NamespaceRef.Uid,
			Name: p.GetNamespace().NamespaceRef.Name,
		}); err != nil {
			return err
		}

	case *enterprisev1.Condition_Expression_UserType_:
	case *enterprisev1.Condition_Expression_SessionType_:
	case *enterprisev1.Condition_Expression_SessionBrowser_:
	case *enterprisev1.Condition_Expression_SessionAuthenticationType_:
	case *enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider_:
	case *enterprisev1.Condition_Expression_SessionAuthenticationCredential_:
	case *enterprisev1.Condition_Expression_SessionAuthenticationAAL_:
	case *enterprisev1.Condition_Expression_ServicePublic_:
	case *enterprisev1.Condition_Expression_ServiceMode_:
	case *enterprisev1.Condition_Expression_DeviceOSType_:
	}

	return nil
}

func (s *Server) toCoreCondition(in *enterprisev1.Condition) *corev1.Condition {
	if in == nil {
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
		ret := &corev1.Condition{
			Type: &corev1.Condition_MatchAny{
				MatchAny: in.GetMatchAny(),
			},
		}
		return ret
	case *enterprisev1.Condition_Expression_:
		ret := &corev1.Condition{
			Type: &corev1.Condition_Match{
				Match: s.getExpression(in.GetExpression()),
			},
		}
		return ret
	case *enterprisev1.Condition_Not_:
		ret := &corev1.Condition{
			Type: &corev1.Condition_Not{
				Not: s.getExpression(in.GetExpression()),
			},
		}
		return ret
	default:
		return nil
	}
}

func (s *Server) getExpression(in *enterprisev1.Condition_Expression) string {

	if in == nil || in.Type == nil {
		return ""
	}

	getRefExpr := func(arg *metav1.ObjectReference) string {
		if arg == nil {
			return ""
		}
		return fmt.Sprintf(`ctx.%s.metadata.name == "%s"`, strings.ToLower(arg.Kind), arg.Name)
	}

	isAPIServer := `ctx.service.systemLabels["octelium-apiserver"] == "true" && ctx.service.status.namespaceRef.name == "octelium-api"`
	isAPServerWithAPI := func(arg string) string {
		return fmt.Sprintf(`ctx.request.grpc.package == "octelium.api.main.%s.v1"`, arg)
	}

	switch in.Type.(type) {
	case *enterprisev1.Condition_Expression_Device_:
		return getRefExpr(in.GetDevice().DeviceRef)
	case *enterprisev1.Condition_Expression_User_:
		return getRefExpr(in.GetUser().UserRef)
	case *enterprisev1.Condition_Expression_Session_:
		return getRefExpr(in.GetSession().SessionRef)
	case *enterprisev1.Condition_Expression_Service_:
		return getRefExpr(in.GetService().ServiceRef)
	case *enterprisev1.Condition_Expression_Namespace_:
		return getRefExpr(in.GetNamespace().NamespaceRef)
	case *enterprisev1.Condition_Expression_Group_:
		return fmt.Sprintf(`"%s" in ctx.user.spec.groups`, in.GetGroup().GroupRef.Name)
	case *enterprisev1.Condition_Expression_ServiceMode_:
		return fmt.Sprintf(`ctx.service.spec.mode == "%s"`, in.GetServiceMode().Mode.String())
	case *enterprisev1.Condition_Expression_DeviceOSType_:
		return fmt.Sprintf(`ctx.device.status.osType == "%s"`, in.GetDeviceOSType().OsType.String())
	case *enterprisev1.Condition_Expression_ServicePublic_:
		if in.GetServicePublic().IsPublic {
			return `ctx.service.spec.isPublic`
		} else {
			return `!ctx.service.spec.isPublic`
		}
	case *enterprisev1.Condition_Expression_SessionAuthenticationAAL_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.aal == "%s"`,
			in.GetSessionAuthenticationAAL().Aal.String())
	case *enterprisev1.Condition_Expression_SessionAuthenticationCredential_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.credential.credentialRef.uid == "%s"`,
			in.GetSessionAuthenticationCredential().CredentialRef.Uid)
	case *enterprisev1.Condition_Expression_SessionAuthenticationIdentityProvider_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.identityProvider.identityProviderRef.uid == "%s"`,
			in.GetSessionAuthenticationIdentityProvider().IdentityProviderRef.Uid)
	case *enterprisev1.Condition_Expression_SessionAuthenticationType_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.type == "%s"`,
			in.GetSessionAuthenticationType().Type.String())
	case *enterprisev1.Condition_Expression_SessionBrowser_:
		if in.GetSessionBrowser().IsBrowser {
			return `ctx.session.status.isBrowser`
		} else {
			return `!ctx.session.status.isBrowser`
		}
	case *enterprisev1.Condition_Expression_SessionType_:
		return fmt.Sprintf(`ctx.session.status.type == "%s"`,
			in.GetSessionType().Type.String())
	case *enterprisev1.Condition_Expression_UserType_:
		return fmt.Sprintf(`ctx.user.spec.type == "%s"`,
			in.GetUserType().Type.String())
	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorAAGUID_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.authenticator.info.fido.aaguid == "%s"`,
			in.GetSessionAuthenticationCredAuthenticatorAAGUID().Aaguid)
	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOPasskey_:
		if in.GetSessionAuthenticationCredAuthenticatorFIDOPasskey().IsPasskey {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isPasskey`
		} else {
			return `!ctx.session.status.authentication.info.authenticator.info.fido.isPasskey`
		}
	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOAttestationVerified_:
		if in.GetSessionAuthenticationCredAuthenticatorFIDOAttestationVerified().IsAttestationVerified {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isAttestationVerified`
		} else {
			return `!ctx.session.status.authentication.info.authenticator.info.fido.isAttestationVerified`
		}
	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOHardware_:
		if in.GetSessionAuthenticationCredAuthenticatorFIDOHardware().IsHardware {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isHardware`
		} else {
			return `!ctx.session.status.authentication.info.authenticator.info.fido.isHardware`
		}
	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserPresent_:
		if in.GetSessionAuthenticationCredAuthenticatorFIDOUserPresent().IsUserPresent {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isUserPresent`
		} else {
			return `!ctx.session.status.authentication.info.authenticator.info.fido.isUserPresent`
		}
	case *enterprisev1.Condition_Expression_SessionAuthenticationCredAuthenticatorFIDOUserVerified_:
		if in.GetSessionAuthenticationCredAuthenticatorFIDOUserVerified().IsUserVerified {
			return `ctx.session.status.authentication.info.authenticator.info.fido.isUserVerified`
		} else {
			return `!ctx.session.status.authentication.info.authenticator.info.fido.isUserVerified`
		}
	case *enterprisev1.Condition_Expression_SessionAuthenticationCredentialType_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.credential.type == "%s"`,
			in.GetSessionAuthenticationCredentialType().Type.String())
	case *enterprisev1.Condition_Expression_SessionAuthenticationGeoipCountryCode_:
		return fmt.Sprintf(`ctx.session.status.authentication.info.geoip.country.code == "%s"`,
			in.GetSessionAuthenticationGeoipCountryCode().Code)
	case *enterprisev1.Condition_Expression_TimeAfter_:
		return fmt.Sprintf(`now() > timestamp("%s")`, in.GetTimeAfter().Timestamp.AsTime().Format(time.RFC3339Nano))
	case *enterprisev1.Condition_Expression_TimeBefore_:
		return fmt.Sprintf(`now() < timestamp("%s")`, in.GetTimeBefore().Timestamp.AsTime().Format(time.RFC3339Nano))

	case *enterprisev1.Condition_Expression_RequestHTTPPathExact_:
		return fmt.Sprintf(`ctx.request.http.path == "%s"`, in.GetRequestHTTPPathExact().Value)
	case *enterprisev1.Condition_Expression_RequestHTTPPathPrefix_:
		return fmt.Sprintf(`ctx.request.http.path.startsWith("%s")`, in.GetRequestHTTPPathPrefix().Value)
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
		return fmt.Sprintf(`"%s" in ctx.request.http.headers`, in.GetRequestHTTPHasHeader().Value)
	case *enterprisev1.Condition_Expression_RequestHTTPHeaderValue_:
		return fmt.Sprintf(`ctx.request.http.headers["%s"] == "%s"`,
			in.GetRequestHTTPHeaderValue().Header, in.GetRequestHTTPHeaderValue().Value)
	case *enterprisev1.Condition_Expression_RequestHTTPMethod_:
		return fmt.Sprintf(`ctx.request.http.method == "%s"`, in.GetRequestHTTPMethod().Value)
	case *enterprisev1.Condition_Expression_RequestIP_:
		return fmt.Sprintf(`ctx.request.ip == "%s"`, in.GetRequestIP().Value)
	case *enterprisev1.Condition_Expression_RequestIPInRange_:
		return fmt.Sprintf(`net.isIPInRange(ctx.request.ip, "%s")`, in.GetRequestIPInRange().Value)
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
