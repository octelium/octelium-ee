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

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
)

func (s *ServerResourceEnterprise) GetCollectorExporterSummary(ctx context.Context, req *venterprisev1.GetCollectorExporterSummaryRequest) (*venterprisev1.GetCollectorExporterSummaryResponse, error) {
	return s.enterpriseC.GetCollectorExporterSummary(ctx, req)
}

func (s *ServerResourceEnterprise) GetSecretSummary(ctx context.Context, req *venterprisev1.GetSecretSummaryRequest) (*venterprisev1.GetSecretSummaryResponse, error) {
	return s.enterpriseC.GetSecretSummary(ctx, req)
}

func (s *ServerResourceEnterprise) GetSecretStoreSummary(ctx context.Context, req *venterprisev1.GetSecretStoreSummaryRequest) (*venterprisev1.GetSecretStoreSummaryResponse, error) {
	return s.enterpriseC.GetSecretStoreSummary(ctx, req)
}

func (s *ServerResourceEnterprise) GetCertificateSummary(ctx context.Context, req *venterprisev1.GetCertificateSummaryRequest) (*venterprisev1.GetCertificateSummaryResponse, error) {
	return s.enterpriseC.GetCertificateSummary(ctx, req)
}

func (s *ServerResourceEnterprise) GetCertificateIssuerSummary(ctx context.Context, req *venterprisev1.GetCertificateIssuerSummaryRequest) (*venterprisev1.GetCertificateIssuerSummaryResponse, error) {
	return s.enterpriseC.GetCertificateIssuerSummary(ctx, req)
}

func (s *ServerResourceEnterprise) GetDNSProviderSummary(ctx context.Context, req *venterprisev1.GetDNSProviderSummaryRequest) (*venterprisev1.GetDNSProviderSummaryResponse, error) {
	return s.enterpriseC.GetDNSProviderSummary(ctx, req)
}

func (s *ServerResourceEnterprise) GetDirectoryProviderSummary(ctx context.Context, req *venterprisev1.GetDirectoryProviderSummaryRequest) (*venterprisev1.GetDirectoryProviderSummaryResponse, error) {
	return s.enterpriseC.GetDirectoryProviderSummary(ctx, req)
}

func (s *ServerResourceEnterprise) GetDirectoryProviderUserSummary(ctx context.Context, req *venterprisev1.GetDirectoryProviderUserSummaryRequest) (*venterprisev1.GetDirectoryProviderUserSummaryResponse, error) {
	return s.enterpriseC.GetDirectoryProviderUserSummary(ctx, req)
}

func (s *ServerResourceEnterprise) GetDirectoryProviderGroupSummary(ctx context.Context, req *venterprisev1.GetDirectoryProviderGroupSummaryRequest) (*venterprisev1.GetDirectoryProviderGroupSummaryResponse, error) {
	return s.enterpriseC.GetDirectoryProviderGroupSummary(ctx, req)
}

func (s *ServerResourceEnterprise) GetDeviceManagerSummary(ctx context.Context, req *venterprisev1.GetDeviceManagerSummaryRequest) (*venterprisev1.GetDeviceManagerSummaryResponse, error) {
	return s.enterpriseC.GetDeviceManagerSummary(ctx, req)
}

func (s *ServerResourceEnterprise) ListCollectorExporter(ctx context.Context, req *venterprisev1.ListCollectorExporterOptions) (*enterprisev1.CollectorExporterList, error) {
	return s.enterpriseC.ListCollectorExporter(ctx, req)
}

func (s *ServerResourceEnterprise) ListSecret(ctx context.Context, req *venterprisev1.ListSecretOptions) (*enterprisev1.SecretList, error) {
	return s.enterpriseC.ListSecret(ctx, req)
}

func (s *ServerResourceEnterprise) ListSecretStore(ctx context.Context, req *venterprisev1.ListSecretStoreOptions) (*enterprisev1.SecretStoreList, error) {
	return s.enterpriseC.ListSecretStore(ctx, req)
}

func (s *ServerResourceEnterprise) ListCertificate(ctx context.Context, req *venterprisev1.ListCertificateOptions) (*enterprisev1.CertificateList, error) {
	return s.enterpriseC.ListCertificate(ctx, req)
}

func (s *ServerResourceEnterprise) ListCertificateIssuer(ctx context.Context, req *venterprisev1.ListCertificateIssuerOptions) (*enterprisev1.CertificateIssuerList, error) {
	return s.enterpriseC.ListCertificateIssuer(ctx, req)
}

func (s *ServerResourceEnterprise) ListDNSProvider(ctx context.Context, req *venterprisev1.ListDNSProviderOptions) (*enterprisev1.DNSProviderList, error) {
	return s.enterpriseC.ListDNSProvider(ctx, req)
}

func (s *ServerResourceEnterprise) ListDirectoryProvider(ctx context.Context, req *venterprisev1.ListDirectoryProviderOptions) (*enterprisev1.DirectoryProviderList, error) {
	return s.enterpriseC.ListDirectoryProvider(ctx, req)
}

func (s *ServerResourceEnterprise) ListDirectoryProviderUser(ctx context.Context, req *venterprisev1.ListDirectoryProviderUserOptions) (*enterprisev1.DirectoryProviderUserList, error) {
	return s.enterpriseC.ListDirectoryProviderUser(ctx, req)
}

func (s *ServerResourceEnterprise) ListDirectoryProviderGroup(ctx context.Context, req *venterprisev1.ListDirectoryProviderGroupOptions) (*enterprisev1.DirectoryProviderGroupList, error) {
	return s.enterpriseC.ListDirectoryProviderGroup(ctx, req)
}

func (s *ServerResourceEnterprise) ListDeviceManager(ctx context.Context, req *venterprisev1.ListDeviceManagerOptions) (*enterprisev1.DeviceManagerList, error) {
	return s.enterpriseC.ListDeviceManager(ctx, req)
}
