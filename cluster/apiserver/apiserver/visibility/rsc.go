// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package visibility

import (
	"context"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
)

func (s *ServerResource) GetServiceSummary(ctx context.Context, req *vcorev1.GetServiceSummaryRequest) (*vcorev1.GetServiceSummaryResponse, error) {
	return s.coreC.GetServiceSummary(ctx, req)
}

func (s *ServerResource) GetSessionSummary(ctx context.Context, req *vcorev1.GetSessionSummaryRequest) (*vcorev1.GetSessionSummaryResponse, error) {
	return s.coreC.GetSessionSummary(ctx, req)
}

func (s *ServerResource) GetUserSummary(ctx context.Context, req *vcorev1.GetUserSummaryRequest) (*vcorev1.GetUserSummaryResponse, error) {
	return s.coreC.GetUserSummary(ctx, req)
}

func (s *ServerResource) GetPolicySummary(ctx context.Context, req *vcorev1.GetPolicySummaryRequest) (*vcorev1.GetPolicySummaryResponse, error) {
	return s.coreC.GetPolicySummary(ctx, req)
}

func (s *ServerResource) GetCredentialSummary(ctx context.Context, req *vcorev1.GetCredentialSummaryRequest) (*vcorev1.GetCredentialSummaryResponse, error) {
	return s.coreC.GetCredentialSummary(ctx, req)
}

func (s *ServerResource) GetDeviceSummary(ctx context.Context, req *vcorev1.GetDeviceSummaryRequest) (*vcorev1.GetDeviceSummaryResponse, error) {
	return s.coreC.GetDeviceSummary(ctx, req)
}

func (s *ServerResource) GetIdentityProviderSummary(ctx context.Context, req *vcorev1.GetIdentityProviderSummaryRequest) (*vcorev1.GetIdentityProviderSummaryResponse, error) {
	return s.coreC.GetIdentityProviderSummary(ctx, req)
}

func (s *ServerResource) GetAuthenticatorSummary(ctx context.Context, req *vcorev1.GetAuthenticatorSummaryRequest) (*vcorev1.GetAuthenticatorSummaryResponse, error) {
	return s.coreC.GetAuthenticatorSummary(ctx, req)
}

func (s *ServerResource) GetGroupSummary(ctx context.Context, req *vcorev1.GetGroupSummaryRequest) (*vcorev1.GetGroupSummaryResponse, error) {
	return s.coreC.GetGroupSummary(ctx, req)
}

func (s *ServerResource) GetGatewaySummary(ctx context.Context, req *vcorev1.GetGatewaySummaryRequest) (*vcorev1.GetGatewaySummaryResponse, error) {
	return s.coreC.GetGatewaySummary(ctx, req)
}

func (s *ServerResource) GetSecretSummary(ctx context.Context, req *vcorev1.GetSecretSummaryRequest) (*vcorev1.GetSecretSummaryResponse, error) {
	return s.coreC.GetSecretSummary(ctx, req)
}

func (s *ServerResource) GetRegionSummary(ctx context.Context, req *vcorev1.GetRegionSummaryRequest) (*vcorev1.GetRegionSummaryResponse, error) {
	return s.coreC.GetRegionSummary(ctx, req)
}

func (s *ServerResource) GetNamespaceSummary(ctx context.Context, req *vcorev1.GetNamespaceSummaryRequest) (*vcorev1.GetNamespaceSummaryResponse, error) {
	return s.coreC.GetNamespaceSummary(ctx, req)
}

func (s *ServerResource) ListUser(ctx context.Context, req *vcorev1.ListUserOptions) (*corev1.UserList, error) {
	return s.coreC.ListUser(ctx, req)
}

func (s *ServerResource) ListSession(ctx context.Context, req *vcorev1.ListSessionOptions) (*corev1.SessionList, error) {
	return s.coreC.ListSession(ctx, req)
}

func (s *ServerResource) ListService(ctx context.Context, req *vcorev1.ListServiceOptions) (*corev1.ServiceList, error) {
	return s.coreC.ListService(ctx, req)
}

func (s *ServerResource) ListNamespace(ctx context.Context, req *vcorev1.ListNamespaceOptions) (*corev1.NamespaceList, error) {
	return s.coreC.ListNamespace(ctx, req)
}

func (s *ServerResource) ListSecret(ctx context.Context, req *vcorev1.ListSecretOptions) (*corev1.SecretList, error) {
	return s.coreC.ListSecret(ctx, req)
}

func (s *ServerResource) ListGroup(ctx context.Context, req *vcorev1.ListGroupOptions) (*corev1.GroupList, error) {
	return s.coreC.ListGroup(ctx, req)
}

func (s *ServerResource) ListRegion(ctx context.Context, req *vcorev1.ListRegionOptions) (*corev1.RegionList, error) {
	return s.coreC.ListRegion(ctx, req)
}

func (s *ServerResource) ListGateway(ctx context.Context, req *vcorev1.ListGatewayOptions) (*corev1.GatewayList, error) {
	return s.coreC.ListGateway(ctx, req)
}

func (s *ServerResource) ListCredential(ctx context.Context, req *vcorev1.ListCredentialOptions) (*corev1.CredentialList, error) {
	return s.coreC.ListCredential(ctx, req)
}

func (s *ServerResource) ListPolicy(ctx context.Context, req *vcorev1.ListPolicyOptions) (*corev1.PolicyList, error) {
	return s.coreC.ListPolicy(ctx, req)
}

func (s *ServerResource) ListIdentityProvider(ctx context.Context, req *vcorev1.ListIdentityProviderOptions) (*corev1.IdentityProviderList, error) {
	return s.coreC.ListIdentityProvider(ctx, req)
}

func (s *ServerResource) ListDevice(ctx context.Context, req *vcorev1.ListDeviceOptions) (*corev1.DeviceList, error) {
	return s.coreC.ListDevice(ctx, req)
}

func (s *ServerResource) ListAuthenticator(ctx context.Context, req *vcorev1.ListAuthenticatorOptions) (*corev1.AuthenticatorList, error) {
	return s.coreC.ListAuthenticator(ctx, req)
}
