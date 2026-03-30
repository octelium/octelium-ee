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

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
)

type srvCore struct {
	s *Server
	vcorev1.UnimplementedResourceServiceServer
}

func (s *srvCore) GetServiceSummary(ctx context.Context, req *vcorev1.GetServiceSummaryRequest) (*vcorev1.GetServiceSummaryResponse, error) {
	return s.s.getSummaryCoreService(ctx, req)
}

func (s *srvCore) GetSessionSummary(ctx context.Context, req *vcorev1.GetSessionSummaryRequest) (*vcorev1.GetSessionSummaryResponse, error) {
	return s.s.getSummaryCoreSession(ctx, req)
}

func (s *srvCore) GetUserSummary(ctx context.Context, req *vcorev1.GetUserSummaryRequest) (*vcorev1.GetUserSummaryResponse, error) {
	return s.s.getSummaryCoreUser(ctx, req)
}

func (s *srvCore) GetPolicySummary(ctx context.Context, req *vcorev1.GetPolicySummaryRequest) (*vcorev1.GetPolicySummaryResponse, error) {
	return s.s.getSummaryCorePolicy(ctx, req)
}

func (s *srvCore) GetCredentialSummary(ctx context.Context, req *vcorev1.GetCredentialSummaryRequest) (*vcorev1.GetCredentialSummaryResponse, error) {
	return s.s.getSummaryCoreCredential(ctx, req)
}

func (s *srvCore) GetDeviceSummary(ctx context.Context, req *vcorev1.GetDeviceSummaryRequest) (*vcorev1.GetDeviceSummaryResponse, error) {
	return s.s.getSummaryCoreDevice(ctx, req)
}

func (s *srvCore) GetIdentityProviderSummary(ctx context.Context, req *vcorev1.GetIdentityProviderSummaryRequest) (*vcorev1.GetIdentityProviderSummaryResponse, error) {
	return s.s.getSummaryCoreIdentityProvider(ctx, req)
}

func (s *srvCore) GetAuthenticatorSummary(ctx context.Context, req *vcorev1.GetAuthenticatorSummaryRequest) (*vcorev1.GetAuthenticatorSummaryResponse, error) {
	return s.s.getSummaryCoreAuthenticator(ctx, req)
}

func (s *srvCore) GetGroupSummary(ctx context.Context, req *vcorev1.GetGroupSummaryRequest) (*vcorev1.GetGroupSummaryResponse, error) {
	return s.s.getSummaryCoreGroup(ctx, req)
}

func (s *srvCore) GetGatewaySummary(ctx context.Context, req *vcorev1.GetGatewaySummaryRequest) (*vcorev1.GetGatewaySummaryResponse, error) {
	return s.s.getSummaryCoreGateway(ctx, req)
}

func (s *srvCore) GetRegionSummary(ctx context.Context, req *vcorev1.GetRegionSummaryRequest) (*vcorev1.GetRegionSummaryResponse, error) {
	return s.s.getSummaryCoreRegion(ctx, req)
}

func (s *srvCore) GetSecretSummary(ctx context.Context, req *vcorev1.GetSecretSummaryRequest) (*vcorev1.GetSecretSummaryResponse, error) {
	return s.s.getSummaryCoreSecret(ctx, req)
}

func (s *srvCore) GetNamespaceSummary(ctx context.Context, req *vcorev1.GetNamespaceSummaryRequest) (*vcorev1.GetNamespaceSummaryResponse, error) {
	return s.s.getSummaryCoreNamespace(ctx, req)
}

/*
func (s *srvCore) GetSummary(ctx context.Context, req *vcorev1.GetSummaryRequest) (*vcorev1.GetSummaryResponse, error) {
	ret := &vcorev1.GetSummaryResponse{}
	var err error

	ret.User, err = s.s.getSummaryCoreUser(ctx, &vcorev1.GetUserSummaryRequest{})
	if err != nil {
		return nil, err
	}

	ret.Session, err = s.s.getSummaryCoreSession(ctx, &vcorev1.GetSessionSummaryRequest{})
	if err != nil {
		return nil, err
	}

	ret.Device, err = s.s.getSummaryCoreDevice(ctx, &vcorev1.GetDeviceSummaryRequest{})
	if err != nil {
		return nil, err
	}

	ret.Service, err = s.s.getSummaryCoreService(ctx, &vcorev1.GetServiceSummaryRequest{})
	if err != nil {
		return nil, err
	}

	ret.Authenticator, err = s.s.getSummaryCoreAuthenticator(ctx, &vcorev1.GetAuthenticatorSummaryRequest{})
	if err != nil {
		return nil, err
	}

	return ret, nil
}
*/

func (s *srvCore) ListService(ctx context.Context, req *vcorev1.ListServiceOptions) (*corev1.ServiceList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindService,
		common:  req.Common,
	}

	if req.NamespaceRef != nil {
		if err := apivalidation.CheckObjectRef(req.NamespaceRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		switch {
		case req.NamespaceRef.Name != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.namespaceRef.name'`).Eq(req.NamespaceRef.Name))
		case req.NamespaceRef.Uid != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.namespaceRef.uid'`).Eq(req.NamespaceRef.Uid))
		}
	}

	if req.RegionRef != nil {
		if err := apivalidation.CheckObjectRef(req.RegionRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		switch {
		case req.RegionRef.Name != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.regionRef.name'`).Eq(req.RegionRef.Name))
		case req.RegionRef.Uid != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.regionRef.uid'`).Eq(req.RegionRef.Uid))
		}
	}

	if req.Mode != corev1.Service_Spec_MODE_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.mode'`).Eq(req.Mode.String()))
	}

	if req.IsPublic {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.isPublic') = true`))
	}

	if req.IsAnonymous {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.isAnonymous') = true`))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.ServiceList), nil
}

func (s *srvCore) ListUser(ctx context.Context, req *vcorev1.ListUserOptions) (*corev1.UserList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindUser,
		common:  req.Common,
	}

	if req.Type != corev1.User_Spec_TYPE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.type'`).Eq(req.Type.String()))
	}

	if req.IsDisabled {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.isDisabled') = true`))
	}

	if req.GroupRef != nil && req.GroupRef.Name != "" {
		if err := apivalidation.CheckObjectRef(req.GroupRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}
		doListReq.filters = append(doListReq.filters,
			goqu.L(fmt.Sprintf(`list_contains(CAST(json_extract(rsc, '$.spec.groups') AS VARCHAR[]),'%s')`, req.GroupRef.Name)))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.UserList), nil
}

func (s *srvCore) ListGroup(ctx context.Context, req *vcorev1.ListGroupOptions) (*corev1.GroupList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindGroup,
		common:  req.Common,
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.GroupList), nil
}

func (s *srvCore) ListNamespace(ctx context.Context, req *vcorev1.ListNamespaceOptions) (*corev1.NamespaceList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindNamespace,
		common:  req.Common,
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.NamespaceList), nil
}

func (s *srvCore) ListPolicy(ctx context.Context, req *vcorev1.ListPolicyOptions) (*corev1.PolicyList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindPolicy,
		common:  req.Common,
	}

	if req.IsDisabled {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.isDisabled') = true`))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.PolicyList), nil
}

func (s *srvCore) ListRegion(ctx context.Context, req *vcorev1.ListRegionOptions) (*corev1.RegionList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindRegion,
		common:  req.Common,
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.RegionList), nil
}

func (s *srvCore) ListGateway(ctx context.Context, req *vcorev1.ListGatewayOptions) (*corev1.GatewayList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindGateway,
		common:  req.Common,
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.GatewayList), nil
}

func (s *srvCore) ListSecret(ctx context.Context, req *vcorev1.ListSecretOptions) (*corev1.SecretList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindSecret,
		common:  req.Common,
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.SecretList), nil
}

func (s *srvCore) ListCredential(ctx context.Context, req *vcorev1.ListCredentialOptions) (*corev1.CredentialList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindCredential,
		common:  req.Common,
	}

	if req.UserRef != nil {
		if err := apivalidation.CheckObjectRef(req.UserRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		switch {
		case req.UserRef.Name != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.userRef.name'`).Eq(req.UserRef.Name))
		case req.UserRef.Uid != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.userRef.uid'`).Eq(req.UserRef.Uid))
		}
	}

	if req.Type != corev1.Credential_Spec_TYPE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.type'`).Eq(req.Type.String()))
	}

	if req.IsDisabled {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.isDisabled') = true`))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.CredentialList), nil
}

func (s *srvCore) ListIdentityProvider(ctx context.Context, req *vcorev1.ListIdentityProviderOptions) (*corev1.IdentityProviderList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindIdentityProvider,
		common:  req.Common,
	}

	if req.Type != corev1.IdentityProvider_Status_TYPE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.type'`).Eq(req.Type.String()))
	}

	if req.IsDisabled {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.isDisabled') = true`))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.IdentityProviderList), nil
}

func (s *srvCore) ListSession(ctx context.Context, req *vcorev1.ListSessionOptions) (*corev1.SessionList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindSession,
		common:  req.Common,
	}

	if req.Type != corev1.Session_Status_TYPE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.type'`).Eq(req.Type.String()))
	}

	if req.IsBrowser {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.status.isBrowser') = true`))
	}

	if req.IsConnected {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.status.isConnected') = true`))
	}

	if req.State != corev1.Session_Spec_STATE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.state'`).Eq(req.State.String()))
	}

	if req.UserRef != nil {
		if err := apivalidation.CheckObjectRef(req.UserRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		switch {
		case req.UserRef.Name != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.userRef.name'`).Eq(req.UserRef.Name))
		case req.UserRef.Uid != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.userRef.uid'`).Eq(req.UserRef.Uid))
		}
	}

	if req.DeviceRef != nil {
		if err := apivalidation.CheckObjectRef(req.DeviceRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		switch {
		case req.DeviceRef.Name != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.deviceRef.name'`).Eq(req.DeviceRef.Name))
		case req.DeviceRef.Uid != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.deviceRef.uid'`).Eq(req.DeviceRef.Uid))
		}
	}

	if req.CredentialRef != nil {
		if err := apivalidation.CheckObjectRef(req.CredentialRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		switch {
		case req.CredentialRef.Name != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.credentialRef.name'`).Eq(req.CredentialRef.Name))
		case req.CredentialRef.Uid != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.credentialRef.uid'`).Eq(req.CredentialRef.Uid))
		}
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.SessionList), nil
}

func (s *srvCore) ListDevice(ctx context.Context, req *vcorev1.ListDeviceOptions) (*corev1.DeviceList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindDevice,
		common:  req.Common,
	}

	if req.UserRef != nil {
		if err := apivalidation.CheckObjectRef(req.UserRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		switch {
		case req.UserRef.Name != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.userRef.name'`).Eq(req.UserRef.Name))
		case req.UserRef.Uid != "":
			doListReq.filters = append(doListReq.filters,
				goqu.L(`rsc->>'$.status.userRef.uid'`).Eq(req.UserRef.Uid))
		}
	}

	if req.State != corev1.Device_Spec_STATE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.state'`).Eq(req.State.String()))
	}

	if req.OsType != corev1.Device_Status_OS_TYPE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.osType'`).Eq(req.OsType.String()))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.DeviceList), nil
}

func (s *srvCore) ListAuthenticator(ctx context.Context, req *vcorev1.ListAuthenticatorOptions) (*corev1.AuthenticatorList, error) {

	doListReq := &doListReq{
		api:     ucorev1.API,
		version: ucorev1.Version,
		kind:    ucorev1.KindAuthenticator,
		common:  req.Common,
	}
	var err error

	doListReq.filters, err = appendRefFilter(doListReq.filters, req.UserRef, nil, "status.userRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.DeviceRef, nil, "status.deviceRef")
	if err != nil {
		return nil, err
	}

	if req.State != corev1.Authenticator_Spec_STATE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.state'`).Eq(req.State.String()))
	}

	if req.Type != corev1.Authenticator_Status_TYPE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.type'`).Eq(req.Type.String()))
	}

	if req.IsRegistered {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.status.isRegistered') = true`))
	}

	/*
		if req.UserRef != nil {
			if err := apivalidation.CheckObjectRef(req.UserRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
				return nil, err
			}

			switch {
			case req.UserRef.Name != "":
				doListReq.filters = append(doListReq.filters,
					goqu.L(`rsc->>'$.status.userRef.name'`).Eq(req.UserRef.Name))
			case req.UserRef.Uid != "":
				doListReq.filters = append(doListReq.filters,
					goqu.L(`rsc->>'$.status.userRef.uid'`).Eq(req.UserRef.Uid))
			}
		}
	*/

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*corev1.AuthenticatorList), nil
}

func appendRefFilter(filters []exp.Expression, ref *metav1.ObjectReference, o *apivalidation.CheckGetOptionsOpts, pth string) ([]exp.Expression, error) {
	if ref == nil {
		return filters, nil
	}
	if o == nil {
		o = &apivalidation.CheckGetOptionsOpts{}
	}

	if err := apivalidation.CheckObjectRef(ref, o); err != nil {
		return nil, err
	}

	if ref.Uid != "" {
		filterName := fmt.Sprintf(`rsc->>'$.%s.uid'`, pth)
		filters = append(filters, goqu.L(filterName).Eq(ref.Uid))
	}

	if ref.Name != "" {
		filterName := fmt.Sprintf(`rsc->>'$.%s.name'`, pth)
		filters = append(filters, goqu.L(filterName).Eq(ref.Name))
	}

	return filters, nil
}
