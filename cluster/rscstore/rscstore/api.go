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
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
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

type srvEnterprise struct {
	s *Server
	venterprisev1.UnimplementedResourceServiceServer
}

func (s *srvEnterprise) GetCollectorExporterSummary(ctx context.Context, req *venterprisev1.GetCollectorExporterSummaryRequest) (*venterprisev1.GetCollectorExporterSummaryResponse, error) {
	return s.s.getSummaryEnterpriseCollectorExporter(ctx, req)
}

func (s *srvEnterprise) GetSecretSummary(ctx context.Context, req *venterprisev1.GetSecretSummaryRequest) (*venterprisev1.GetSecretSummaryResponse, error) {
	return s.s.getSummaryEnterpriseSecret(ctx, req)
}

func (s *srvEnterprise) GetSecretStoreSummary(ctx context.Context, req *venterprisev1.GetSecretStoreSummaryRequest) (*venterprisev1.GetSecretStoreSummaryResponse, error) {
	return s.s.getSummaryEnterpriseSecretStore(ctx, req)
}

func (s *srvEnterprise) GetCertificateSummary(ctx context.Context, req *venterprisev1.GetCertificateSummaryRequest) (*venterprisev1.GetCertificateSummaryResponse, error) {
	return s.s.getSummaryEnterpriseCertificate(ctx, req)
}

func (s *srvEnterprise) GetCertificateIssuerSummary(ctx context.Context, req *venterprisev1.GetCertificateIssuerSummaryRequest) (*venterprisev1.GetCertificateIssuerSummaryResponse, error) {
	return s.s.getSummaryEnterpriseCertificateIssuer(ctx, req)
}

func (s *srvEnterprise) GetDNSProviderSummary(ctx context.Context, req *venterprisev1.GetDNSProviderSummaryRequest) (*venterprisev1.GetDNSProviderSummaryResponse, error) {
	return s.s.getSummaryEnterpriseDNSProvider(ctx, req)
}

func (s *srvEnterprise) GetDirectoryProviderSummary(ctx context.Context, req *venterprisev1.GetDirectoryProviderSummaryRequest) (*venterprisev1.GetDirectoryProviderSummaryResponse, error) {
	return s.s.getSummaryEnterpriseDirectoryProvider(ctx, req)
}

func (s *srvEnterprise) GetDirectoryProviderUserSummary(ctx context.Context, req *venterprisev1.GetDirectoryProviderUserSummaryRequest) (*venterprisev1.GetDirectoryProviderUserSummaryResponse, error) {
	return s.s.getSummaryEnterpriseDirectoryProviderUser(ctx, req)
}

func (s *srvEnterprise) GetDirectoryProviderGroupSummary(ctx context.Context, req *venterprisev1.GetDirectoryProviderGroupSummaryRequest) (*venterprisev1.GetDirectoryProviderGroupSummaryResponse, error) {
	return s.s.getSummaryEnterpriseDirectoryProviderGroup(ctx, req)
}

func (s *srvEnterprise) GetDeviceManagerSummary(ctx context.Context, req *venterprisev1.GetDeviceManagerSummaryRequest) (*venterprisev1.GetDeviceManagerSummaryResponse, error) {
	return s.s.getSummaryEnterpriseDeviceManager(ctx, req)
}

func collectorExporterTypeField(t venterprisev1.ListCollectorExporterOptions_Type) string {
	switch t {
	case venterprisev1.ListCollectorExporterOptions_OTLP:
		return "otlp"
	case venterprisev1.ListCollectorExporterOptions_OTLP_HTTP:
		return "otlpHTTP"
	case venterprisev1.ListCollectorExporterOptions_CLICKHOUSE:
		return "clickhouse"
	case venterprisev1.ListCollectorExporterOptions_ELASTICSEARCH:
		return "elasticsearch"
	case venterprisev1.ListCollectorExporterOptions_LOGZIO:
		return "logzio"
	case venterprisev1.ListCollectorExporterOptions_INFLUXDB:
		return "influxDB"
	case venterprisev1.ListCollectorExporterOptions_KAFKA:
		return "kafka"
	case venterprisev1.ListCollectorExporterOptions_DATADOG:
		return "datadog"
	case venterprisev1.ListCollectorExporterOptions_SPLUNK:
		return "splunk"
	case venterprisev1.ListCollectorExporterOptions_AZURE_MONITOR:
		return "azureMonitor"
	case venterprisev1.ListCollectorExporterOptions_AZURE_DATA_EXPLORER:
		return "azureDataExplorer"
	case venterprisev1.ListCollectorExporterOptions_PROMETHEUS_REMOTE_WRITE:
		return "prometheusRemoteWrite"
	default:
		return ""
	}
}

func (s *srvEnterprise) ListCollectorExporter(ctx context.Context, req *venterprisev1.ListCollectorExporterOptions) (*enterprisev1.CollectorExporterList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindCollectorExporter,
		common:  req.Common,
	}

	if field := collectorExporterTypeField(req.Type); field != "" {
		doListReq.filters = append(doListReq.filters,
			goqu.L(fmt.Sprintf(`json_extract(rsc, '$.spec.%s') IS NOT NULL`, field)))
	}

	if req.IsDisabled {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.isDisabled') = true`))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.CollectorExporterList), nil
}

func (s *srvEnterprise) ListSecret(ctx context.Context, req *venterprisev1.ListSecretOptions) (*enterprisev1.SecretList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindSecret,
		common:  req.Common,
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.SecretList), nil
}

func (s *srvEnterprise) ListSecretStore(ctx context.Context, req *venterprisev1.ListSecretStoreOptions) (*enterprisev1.SecretStoreList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindSecretStore,
		common:  req.Common,
	}

	if req.Type != enterprisev1.SecretStore_Status_TYPE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.type'`).Eq(req.Type.String()))
	}

	if req.State != enterprisev1.SecretStore_Status_STATE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.state'`).Eq(req.State.String()))
	}

	if req.SynchronizationState != enterprisev1.SecretStore_Status_Synchronization_STATE_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.synchronization.state'`).Eq(req.SynchronizationState.String()))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.SecretStoreList), nil
}

func (s *srvEnterprise) ListCertificate(ctx context.Context, req *venterprisev1.ListCertificateOptions) (*enterprisev1.CertificateList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindCertificate,
		common:  req.Common,
	}
	var err error

	if req.Mode != enterprisev1.Certificate_Spec_MODE_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.mode'`).Eq(req.Mode.String()))
	}

	if req.IssuanceState != enterprisev1.Certificate_Status_Issuance_STATE_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.issuance.state'`).Eq(req.IssuanceState.String()))
	}

	doListReq.filters, err = appendRefFilter(doListReq.filters, req.CertificateIssuerRef, nil, "status.certificateIssuerRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.NamespaceRef, nil, "status.namespaceRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.ServiceRef, nil, "status.serviceRef")
	if err != nil {
		return nil, err
	}

	if req.IsExpired || req.IsExpiringSoon {
		now := time.Now().UTC()

		if req.IsExpired {
			doListReq.filters = append(doListReq.filters,
				goqu.L(`(rsc->>'$.status.issuance.expiresAt') < ?`, now.Format(time.RFC3339Nano)))
		}

		if req.IsExpiringSoon {
			doListReq.filters = append(doListReq.filters,
				goqu.L(`(rsc->>'$.status.issuance.expiresAt' >= ?) AND (rsc->>'$.status.issuance.expiresAt' < ?)`,
					now.Format(time.RFC3339Nano), now.Add(30*24*time.Hour).Format(time.RFC3339Nano)))
		}
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.CertificateList), nil
}

func (s *srvEnterprise) ListCertificateIssuer(ctx context.Context, req *venterprisev1.ListCertificateIssuerOptions) (*enterprisev1.CertificateIssuerList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindCertificateIssuer,
		common:  req.Common,
	}

	if req.Type == venterprisev1.ListCertificateIssuerOptions_ACME {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.acme') IS NOT NULL`))
	}

	if req.State != enterprisev1.CertificateIssuer_Status_STATE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.state'`).Eq(req.State.String()))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.CertificateIssuerList), nil
}

func dnsProviderTypeField(t venterprisev1.ListDNSProviderOptions_Type) string {
	switch t {
	case venterprisev1.ListDNSProviderOptions_CLOUDFLARE:
		return "cloudflare"
	case venterprisev1.ListDNSProviderOptions_AWS:
		return "aws"
	case venterprisev1.ListDNSProviderOptions_DIGITALOCEAN:
		return "digitalocean"
	case venterprisev1.ListDNSProviderOptions_GOOGLE:
		return "google"
	case venterprisev1.ListDNSProviderOptions_AZURE:
		return "azure"
	case venterprisev1.ListDNSProviderOptions_LINODE:
		return "linode"
	case venterprisev1.ListDNSProviderOptions_OVH:
		return "ovh"
	default:
		return ""
	}
}

func (s *srvEnterprise) ListDNSProvider(ctx context.Context, req *venterprisev1.ListDNSProviderOptions) (*enterprisev1.DNSProviderList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindDNSProvider,
		common:  req.Common,
	}

	if field := dnsProviderTypeField(req.Type); field != "" {
		doListReq.filters = append(doListReq.filters,
			goqu.L(fmt.Sprintf(`json_extract(rsc, '$.spec.%s') IS NOT NULL`, field)))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.DNSProviderList), nil
}

func directoryProviderTypeField(t venterprisev1.ListDirectoryProviderOptions_Type) string {
	switch t {
	case venterprisev1.ListDirectoryProviderOptions_SCIM:
		return "scim"
	case venterprisev1.ListDirectoryProviderOptions_GOOGLE_WORKSPACE:
		return "googleWorkspace"
	case venterprisev1.ListDirectoryProviderOptions_KEYCLOAK:
		return "keycloak"
	default:
		return ""
	}
}

func (s *srvEnterprise) ListDirectoryProvider(ctx context.Context, req *venterprisev1.ListDirectoryProviderOptions) (*enterprisev1.DirectoryProviderList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindDirectoryProvider,
		common:  req.Common,
	}

	if field := directoryProviderTypeField(req.Type); field != "" {
		doListReq.filters = append(doListReq.filters,
			goqu.L(fmt.Sprintf(`json_extract(rsc, '$.spec.%s') IS NOT NULL`, field)))
	}

	if req.IsDisabled {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.isDisabled') = true`))
	}

	if req.SynchronizationState != enterprisev1.DirectoryProvider_Status_Synchronization_STATE_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.synchronization.state'`).Eq(req.SynchronizationState.String()))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.DirectoryProviderList), nil
}

func (s *srvEnterprise) ListDirectoryProviderUser(ctx context.Context, req *venterprisev1.ListDirectoryProviderUserOptions) (*enterprisev1.DirectoryProviderUserList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindDirectoryProviderUser,
		common:  req.Common,
	}
	var err error

	doListReq.filters, err = appendRefFilter(doListReq.filters, req.DirectoryProviderRef, nil, "status.directoryProviderRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.UserRef, nil, "status.userRef")
	if err != nil {
		return nil, err
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.DirectoryProviderUserList), nil
}

func (s *srvEnterprise) ListDirectoryProviderGroup(ctx context.Context, req *venterprisev1.ListDirectoryProviderGroupOptions) (*enterprisev1.DirectoryProviderGroupList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindDirectoryProviderGroup,
		common:  req.Common,
	}
	var err error

	doListReq.filters, err = appendRefFilter(doListReq.filters, req.DirectoryProviderRef, nil, "status.directoryProviderRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.GroupRef, nil, "status.groupRef")
	if err != nil {
		return nil, err
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.DirectoryProviderGroupList), nil
}

func (s *srvEnterprise) ListDeviceManager(ctx context.Context, req *venterprisev1.ListDeviceManagerOptions) (*enterprisev1.DeviceManagerList, error) {

	doListReq := &doListReq{
		api:     uenterprisev1.API,
		version: uenterprisev1.Version,
		kind:    uenterprisev1.KindDeviceManager,
		common:  req.Common,
	}

	if req.Type != enterprisev1.DeviceManager_Status_TYPE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.type'`).Eq(req.Type.String()))
	}

	if req.State != enterprisev1.DeviceManager_Status_STATE_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.state'`).Eq(req.State.String()))
	}

	if req.Strategy != enterprisev1.DeviceManager_Spec_Linking_STRATEGY_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.linking.strategy'`).Eq(req.Strategy.String()))
	}

	if req.ApprovalMode != enterprisev1.DeviceManager_Spec_Linking_APPROVAL_MODE_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.linking.approvalMode'`).Eq(req.ApprovalMode.String()))
	}

	if req.IsPollingDisabled {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.polling.isDisabled') = true`))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*enterprisev1.DeviceManagerList), nil
}

type srvAccess struct {
	s *Server
	vaccessv1.UnimplementedResourceServiceServer
}

func (s *srvAccess) GetPolicySummary(ctx context.Context, req *vaccessv1.GetPolicySummaryRequest) (*vaccessv1.GetPolicySummaryResponse, error) {
	return s.s.getSummaryAccessPolicy(ctx, req)
}

func (s *srvAccess) GetCatalogSummary(ctx context.Context, req *vaccessv1.GetCatalogSummaryRequest) (*vaccessv1.GetCatalogSummaryResponse, error) {
	return s.s.getSummaryAccessCatalog(ctx, req)
}

func (s *srvAccess) GetRequestSummary(ctx context.Context, req *vaccessv1.GetRequestSummaryRequest) (*vaccessv1.GetRequestSummaryResponse, error) {
	return s.s.getSummaryAccessRequest(ctx, req)
}

func (s *srvAccess) GetReviewSummary(ctx context.Context, req *vaccessv1.GetReviewSummaryRequest) (*vaccessv1.GetReviewSummaryResponse, error) {
	return s.s.getSummaryAccessReview(ctx, req)
}

func (s *srvAccess) ListPolicy(ctx context.Context, req *vaccessv1.ListPolicyOptions) (*accessv1.PolicyList, error) {

	doListReq := &doListReq{
		api:     uaccessv1.API,
		version: uaccessv1.Version,
		kind:    uaccessv1.KindPolicy,
		common:  req.Common,
	}

	if req.IsDisabled {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`json_extract(rsc, '$.spec.isDisabled') = true`))
	}

	if req.Effect != accessv1.Policy_Spec_Rule_EFFECT_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`len(list_filter(CAST(json_extract(rsc, '$.spec.rules') AS JSON[]), x -> json_extract_string(x, '$.effect') = ?)) > 0`,
				req.Effect.String()))
	}

	if req.UserRef != nil && (req.UserRef.Uid != "" || req.UserRef.Name != "") {
		if err := apivalidation.CheckObjectRef(req.UserRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		val := req.UserRef.Uid
		pth := "$.condition.subject.userRef.uid"
		if val == "" {
			val = req.UserRef.Name
			pth = "$.condition.subject.userRef.name"
		}

		doListReq.filters = append(doListReq.filters,
			goqu.L(fmt.Sprintf(`len(list_filter(CAST(json_extract(rsc, '$.spec.rules') AS JSON[]), x -> json_extract_string(x, '%s') = ?)) > 0`, pth), val))
	}

	if req.GroupRef != nil && (req.GroupRef.Uid != "" || req.GroupRef.Name != "") {
		if err := apivalidation.CheckObjectRef(req.GroupRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		val := req.GroupRef.Uid
		pth := "$.condition.subject.groupRef.uid"
		if val == "" {
			val = req.GroupRef.Name
			pth = "$.condition.subject.groupRef.name"
		}

		doListReq.filters = append(doListReq.filters,
			goqu.L(fmt.Sprintf(`len(list_filter(CAST(json_extract(rsc, '$.spec.rules') AS JSON[]), x -> json_extract_string(x, '%s') = ?)) > 0`, pth), val))
	}

	if req.ServiceRef != nil && (req.ServiceRef.Uid != "" || req.ServiceRef.Name != "") {
		if err := apivalidation.CheckObjectRef(req.ServiceRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		val := req.ServiceRef.Uid
		pth := "$.condition.resource.serviceRef.uid"
		if val == "" {
			val = req.ServiceRef.Name
			pth = "$.condition.resource.serviceRef.name"
		}

		doListReq.filters = append(doListReq.filters,
			goqu.L(fmt.Sprintf(`len(list_filter(CAST(json_extract(rsc, '$.spec.rules') AS JSON[]), x -> json_extract_string(x, '%s') = ?)) > 0`, pth), val))
	}

	if req.CatalogRef != nil && (req.CatalogRef.Uid != "" || req.CatalogRef.Name != "") {
		if err := apivalidation.CheckObjectRef(req.CatalogRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		val := req.CatalogRef.Uid
		pth := "$.condition.resource.catalogRef.uid"
		if val == "" {
			val = req.CatalogRef.Name
			pth = "$.condition.resource.catalogRef.name"
		}

		doListReq.filters = append(doListReq.filters,
			goqu.L(fmt.Sprintf(`len(list_filter(CAST(json_extract(rsc, '$.spec.rules') AS JSON[]), x -> json_extract_string(x, '%s') = ?)) > 0`, pth), val))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*accessv1.PolicyList), nil
}

func (s *srvAccess) ListCatalog(ctx context.Context, req *vaccessv1.ListCatalogOptions) (*accessv1.CatalogList, error) {

	doListReq := &doListReq{
		api:     uaccessv1.API,
		version: uaccessv1.Version,
		kind:    uaccessv1.KindCatalog,
		common:  req.Common,
	}

	if req.ServiceRef != nil && req.ServiceRef.Name != "" {
		if err := apivalidation.CheckObjectRef(req.ServiceRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}
		doListReq.filters = append(doListReq.filters,
			goqu.L(`list_contains(CAST(json_extract(rsc, '$.spec.resourceCollection.service.services') AS VARCHAR[]), ?)`, req.ServiceRef.Name))
	}

	if req.NamespaceRef != nil && req.NamespaceRef.Name != "" {
		if err := apivalidation.CheckObjectRef(req.NamespaceRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}
		doListReq.filters = append(doListReq.filters,
			goqu.L(`list_contains(CAST(json_extract(rsc, '$.spec.resourceCollection.service.namespaces') AS VARCHAR[]), ?)`, req.NamespaceRef.Name))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*accessv1.CatalogList), nil
}

func (s *srvAccess) ListRequest(ctx context.Context, req *vaccessv1.ListRequestOptions) (*accessv1.RequestList, error) {

	doListReq := &doListReq{
		api:     uaccessv1.API,
		version: uaccessv1.Version,
		kind:    uaccessv1.KindRequest,
		common:  req.Common,
	}
	var err error

	doListReq.filters, err = appendRefFilter(doListReq.filters, req.UserRef, nil, "status.userRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.SubjectUserRef, nil, "spec.subject.userRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.ServiceRef, nil, "spec.resource.serviceRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.CatalogRef, nil, "spec.resource.catalog.catalogRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.PolicyRef, nil, "status.policyRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.PolicyTriggerRef, nil, "status.policyTriggerRef")
	if err != nil {
		return nil, err
	}

	if req.ReviewerRef != nil && (req.ReviewerRef.Uid != "" || req.ReviewerRef.Name != "") {
		if err := apivalidation.CheckObjectRef(req.ReviewerRef, &apivalidation.CheckGetOptionsOpts{}); err != nil {
			return nil, err
		}

		val := req.ReviewerRef.Uid
		pth := "$.user.userRef.uid"
		if val == "" {
			val = req.ReviewerRef.Name
			pth = "$.user.userRef.name"
		}

		doListReq.filters = append(doListReq.filters,
			goqu.L(fmt.Sprintf(`len(list_filter(
				flatten(list_transform(
					COALESCE(CAST(json_extract(rsc, '$.status.rule.action.review.steps') AS JSON[]), []),
					s -> COALESCE(CAST(json_extract(s, '$.reviewers') AS JSON[]), [])
				)),
				r -> json_extract_string(r, '%s') = ?
			)) > 0`, pth), val))
	}

	if req.State != accessv1.Request_Status_State_STATUS_UNKNOWN {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.status.state.status'`).Eq(req.State.String()))
	}

	if req.Urgency != accessv1.Request_Spec_URGENCY_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.urgency'`).Eq(req.Urgency.String()))
	}

	if req.IsActive {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`(rsc->>'$.status.state.status' = ?) AND ((json_extract(rsc, '$.status.accessEndsAt') IS NULL) OR (rsc->>'$.status.accessEndsAt' > ?))`,
				accessv1.Request_Status_State_APPROVED.String(), time.Now().UTC().Format(time.RFC3339Nano)))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*accessv1.RequestList), nil
}

func (s *srvAccess) ListReview(ctx context.Context, req *vaccessv1.ListReviewOptions) (*accessv1.ReviewList, error) {

	doListReq := &doListReq{
		api:     uaccessv1.API,
		version: uaccessv1.Version,
		kind:    uaccessv1.KindReview,
		common:  req.Common,
	}
	var err error

	doListReq.filters, err = appendRefFilter(doListReq.filters, req.UserRef, nil, "status.userRef")
	if err != nil {
		return nil, err
	}
	doListReq.filters, err = appendRefFilter(doListReq.filters, req.RequestRef, nil, "status.requestRef")
	if err != nil {
		return nil, err
	}

	if req.Decision != accessv1.Review_Spec_DECISION_UNSET {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`rsc->>'$.spec.decision'`).Eq(req.Decision.String()))
	}

	if req.IsDecided {
		doListReq.filters = append(doListReq.filters,
			goqu.L(`(rsc->>'$.spec.decision' = 'DECISION_APPROVE') OR (rsc->>'$.spec.decision' = 'DECISION_REJECT')`))
	}

	ret, err := s.s.doList(ctx, doListReq)
	if err != nil {
		return nil, err
	}

	return ret.(*accessv1.ReviewList), nil
}
