package visibility

import (
	"context"

	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcordiumv1"
)

func (s *ServerResourceCordium) GetWorkspaceSummary(ctx context.Context, req *vcordiumv1.GetWorkspaceSummaryRequest) (*vcordiumv1.GetWorkspaceSummaryResponse, error) {
	return s.cordiumC.GetWorkspaceSummary(ctx, req)
}

func (s *ServerResourceCordium) GetTemplateSummary(ctx context.Context, req *vcordiumv1.GetTemplateSummaryRequest) (*vcordiumv1.GetTemplateSummaryResponse, error) {
	return s.cordiumC.GetTemplateSummary(ctx, req)
}

func (s *ServerResourceCordium) GetSpaceSummary(ctx context.Context, req *vcordiumv1.GetSpaceSummaryRequest) (*vcordiumv1.GetSpaceSummaryResponse, error) {
	return s.cordiumC.GetSpaceSummary(ctx, req)
}

func (s *ServerResourceCordium) GetMembershipSummary(ctx context.Context, req *vcordiumv1.GetMembershipSummaryRequest) (*vcordiumv1.GetMembershipSummaryResponse, error) {
	return s.cordiumC.GetMembershipSummary(ctx, req)
}

func (s *ServerResourceCordium) GetGitProviderSummary(ctx context.Context, req *vcordiumv1.GetGitProviderSummaryRequest) (*vcordiumv1.GetGitProviderSummaryResponse, error) {
	return s.cordiumC.GetGitProviderSummary(ctx, req)
}

func (s *ServerResourceCordium) GetSecretSummary(ctx context.Context, req *vcordiumv1.GetSecretSummaryRequest) (*vcordiumv1.GetSecretSummaryResponse, error) {
	return s.cordiumC.GetSecretSummary(ctx, req)
}

func (s *ServerResourceCordium) GetUserSecretSummary(ctx context.Context, req *vcordiumv1.GetUserSecretSummaryRequest) (*vcordiumv1.GetUserSecretSummaryResponse, error) {
	return s.cordiumC.GetUserSecretSummary(ctx, req)
}

func (s *ServerResourceCordium) GetRegionSummary(ctx context.Context, req *vcordiumv1.GetRegionSummaryRequest) (*vcordiumv1.GetRegionSummaryResponse, error) {
	return s.cordiumC.GetRegionSummary(ctx, req)
}

func (s *ServerResourceCordium) ListWorkspace(ctx context.Context, req *vcordiumv1.ListWorkspaceOptions) (*cordiumv1.WorkspaceList, error) {
	return s.cordiumC.ListWorkspace(ctx, req)
}

func (s *ServerResourceCordium) ListTemplate(ctx context.Context, req *vcordiumv1.ListTemplateOptions) (*cordiumv1.TemplateList, error) {
	return s.cordiumC.ListTemplate(ctx, req)
}

func (s *ServerResourceCordium) ListSpace(ctx context.Context, req *vcordiumv1.ListSpaceOptions) (*cordiumv1.SpaceList, error) {
	return s.cordiumC.ListSpace(ctx, req)
}

func (s *ServerResourceCordium) ListMembership(ctx context.Context, req *vcordiumv1.ListMembershipOptions) (*cordiumv1.MembershipList, error) {
	return s.cordiumC.ListMembership(ctx, req)
}

func (s *ServerResourceCordium) ListGitProvider(ctx context.Context, req *vcordiumv1.ListGitProviderOptions) (*cordiumv1.GitProviderList, error) {
	return s.cordiumC.ListGitProvider(ctx, req)
}

func (s *ServerResourceCordium) ListSecret(ctx context.Context, req *vcordiumv1.ListSecretOptions) (*cordiumv1.SecretList, error) {
	return s.cordiumC.ListSecret(ctx, req)
}

func (s *ServerResourceCordium) ListUserSecret(ctx context.Context, req *vcordiumv1.ListUserSecretOptions) (*cordiumv1.UserSecretList, error) {
	return s.cordiumC.ListUserSecret(ctx, req)
}

func (s *ServerResourceCordium) ListRegion(ctx context.Context, req *vcordiumv1.ListRegionOptions) (*cordiumv1.RegionList, error) {
	return s.cordiumC.ListRegion(ctx, req)
}
