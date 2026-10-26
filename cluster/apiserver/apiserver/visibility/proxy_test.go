package visibility

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcordiumv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type visibilityClientConn struct {
	invoke func(context.Context, string, any, any) error
}

func (c *visibilityClientConn) Invoke(ctx context.Context, method string, req, res any, opts ...grpc.CallOption) error {
	return c.invoke(ctx, method, req, res)
}

func (c *visibilityClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Error(codes.Unimplemented, "unexpected stream")
}

type visibilityProxyTest struct {
	method   string
	request  proto.Message
	response proto.Message
	call     func(context.Context) (proto.Message, error)
}

func newVisibilityProxyTest[Q, R proto.Message](method string, call func(context.Context, Q) (R, error), req Q, res R) visibilityProxyTest {
	req.ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 0x07})
	res.ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 0x09})
	return visibilityProxyTest{
		method:   method,
		request:  req,
		response: res,
		call: func(ctx context.Context) (proto.Message, error) {
			return call(ctx, req)
		},
	}
}

func TestVisibilityProxy(t *testing.T) {
	conn := &visibilityClientConn{}
	accessLog := &ServerAccessLog{c: visibilityv1.NewAccessLogServiceClient(conn)}
	authenticationLog := &ServerAuthenticationLog{c: visibilityv1.NewAuthenticationLogServiceClient(conn)}
	auditLog := &ServerAuditLog{c: visibilityv1.NewAuditLogServiceClient(conn)}
	componentLog := &ServerComponentLog{c: visibilityv1.NewComponentLogServiceClient(conn)}
	llm := &ServerLLM{c: vllmv1.NewLLMServiceClient(conn)}
	metric := &ServerMetric{c: vmetricsv1.NewMetricsServiceClient(conn)}
	resource := &ServerResource{coreC: vcorev1.NewResourceServiceClient(conn)}
	resourceEnterprise := &ServerResourceEnterprise{enterpriseC: venterprisev1.NewResourceServiceClient(conn)}
	resourceAccess := &ServerResourceAccess{accessC: vaccessv1.NewResourceServiceClient(conn)}
	resourceCordium := &ServerResourceCordium{cordiumC: vcordiumv1.NewResourceServiceClient(conn)}

	cases := []visibilityProxyTest{
		newVisibilityProxyTest(visibilityv1.AccessLogService_ListAccessLog_FullMethodName, accessLog.ListAccessLog, &visibilityv1.ListAccessLogRequest{}, &visibilityv1.ListAccessLogResponse{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_ListSSHSession_FullMethodName, accessLog.ListSSHSession, &visibilityv1.ListSSHSessionRequest{}, &visibilityv1.ListSSHSessionResponse{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_ListSSHSessionRecording_FullMethodName, accessLog.ListSSHSessionRecording, &visibilityv1.ListSSHSessionRecordingRequest{}, &visibilityv1.ListSSHSessionRecordingResponse{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_GetAccessLogSummary_FullMethodName, accessLog.GetAccessLogSummary, &visibilityv1.GetAccessLogSummaryRequest{}, &visibilityv1.GetAccessLogSummaryResponse{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_GetSSHSession_FullMethodName, accessLog.GetSSHSession, &visibilityv1.GetSSHSessionRequest{}, &visibilityv1.SSHSession{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_GetAccessLogDataPoint_FullMethodName, accessLog.GetAccessLogDataPoint, &visibilityv1.GetAccessLogDataPointRequest{}, &visibilityv1.GetAccessLogDataPointResponse{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_ListAccessLogTopUser_FullMethodName, accessLog.ListAccessLogTopUser, &visibilityv1.ListAccessLogTopUserRequest{}, &visibilityv1.ListAccessLogTopUserResponse{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_ListAccessLogTopService_FullMethodName, accessLog.ListAccessLogTopService, &visibilityv1.ListAccessLogTopServiceRequest{}, &visibilityv1.ListAccessLogTopServiceResponse{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_ListAccessLogTopPolicy_FullMethodName, accessLog.ListAccessLogTopPolicy, &visibilityv1.ListAccessLogTopPolicyRequest{}, &visibilityv1.ListAccessLogTopPolicyResponse{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_ListAccessLogTopSession_FullMethodName, accessLog.ListAccessLogTopSession, &visibilityv1.ListAccessLogTopSessionRequest{}, &visibilityv1.ListAccessLogTopSessionResponse{}),
		newVisibilityProxyTest(visibilityv1.AccessLogService_ListAccessLogTopDenyReason_FullMethodName, accessLog.ListAccessLogTopDenyReason, &visibilityv1.ListAccessLogTopDenyReasonRequest{}, &visibilityv1.ListAccessLogTopDenyReasonResponse{}),
		newVisibilityProxyTest(visibilityv1.AuditLogService_ListAuditLog_FullMethodName, auditLog.ListAuditLog, &visibilityv1.ListAuditLogRequest{}, &visibilityv1.ListAuditLogResponse{}),
		newVisibilityProxyTest(visibilityv1.AuditLogService_GetAuditLogSummary_FullMethodName, auditLog.GetAuditLogSummary, &visibilityv1.GetAuditLogSummaryRequest{}, &visibilityv1.GetAuditLogSummaryResponse{}),
		newVisibilityProxyTest(visibilityv1.AuditLogService_GetAuditLogDataPoint_FullMethodName, auditLog.GetAuditLogDataPoint, &visibilityv1.GetAuditLogDataPointRequest{}, &visibilityv1.GetAuditLogDataPointResponse{}),
		newVisibilityProxyTest(visibilityv1.AuditLogService_ListAuditLogTopUser_FullMethodName, auditLog.ListAuditLogTopUser, &visibilityv1.ListAuditLogTopUserRequest{}, &visibilityv1.ListAuditLogTopUserResponse{}),
		newVisibilityProxyTest(visibilityv1.AuditLogService_ListAuditLogTopSession_FullMethodName, auditLog.ListAuditLogTopSession, &visibilityv1.ListAuditLogTopSessionRequest{}, &visibilityv1.ListAuditLogTopSessionResponse{}),
		newVisibilityProxyTest(visibilityv1.AuthenticationLogService_ListAuthenticationLog_FullMethodName, authenticationLog.ListAuthenticationLog, &visibilityv1.ListAuthenticationLogRequest{}, &visibilityv1.ListAuthenticationLogResponse{}),
		newVisibilityProxyTest(visibilityv1.AuthenticationLogService_GetAuthenticationLogSummary_FullMethodName, authenticationLog.GetAuthenticationLogSummary, &visibilityv1.GetAuthenticationLogSummaryRequest{}, &visibilityv1.GetAuthenticationLogSummaryResponse{}),
		newVisibilityProxyTest(visibilityv1.AuthenticationLogService_GetAuthenticationLogDataPoint_FullMethodName, authenticationLog.GetAuthenticationLogDataPoint, &visibilityv1.GetAuthenticationLogDataPointRequest{}, &visibilityv1.GetAuthenticationLogDataPointResponse{}),
		newVisibilityProxyTest(visibilityv1.AuthenticationLogService_ListAuthenticationLogTopUser_FullMethodName, authenticationLog.ListAuthenticationLogTopUser, &visibilityv1.ListAuthenticationLogTopUserRequest{}, &visibilityv1.ListAuthenticationLogTopUserResponse{}),
		newVisibilityProxyTest(visibilityv1.AuthenticationLogService_ListAuthenticationLogTopCredential_FullMethodName, authenticationLog.ListAuthenticationLogTopCredential, &visibilityv1.ListAuthenticationLogTopCredentialRequest{}, &visibilityv1.ListAuthenticationLogTopCredentialResponse{}),
		newVisibilityProxyTest(visibilityv1.AuthenticationLogService_ListAuthenticationLogTopIdentityProvider_FullMethodName, authenticationLog.ListAuthenticationLogTopIdentityProvider, &visibilityv1.ListAuthenticationLogTopIdentityProviderRequest{}, &visibilityv1.ListAuthenticationLogTopIdentityProviderResponse{}),
		newVisibilityProxyTest(visibilityv1.ComponentLogService_ListComponentLog_FullMethodName, componentLog.ListComponentLog, &visibilityv1.ListComponentLogRequest{}, &visibilityv1.ListComponentLogResponse{}),
		newVisibilityProxyTest(visibilityv1.ComponentLogService_GetComponentLogSummary_FullMethodName, componentLog.GetComponentLogSummary, &visibilityv1.GetComponentLogSummaryRequest{}, &visibilityv1.GetComponentLogSummaryResponse{}),
		newVisibilityProxyTest(visibilityv1.ComponentLogService_GetComponentLogDataPoint_FullMethodName, componentLog.GetComponentLogDataPoint, &visibilityv1.GetComponentLogDataPointRequest{}, &visibilityv1.GetComponentLogDataPointResponse{}),
		newVisibilityProxyTest(visibilityv1.ComponentLogService_ListComponentLogTopComponent_FullMethodName, componentLog.ListComponentLogTopComponent, &visibilityv1.ListComponentLogTopComponentRequest{}, &visibilityv1.ListComponentLogTopComponentResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_GetSummary_FullMethodName, llm.GetSummary, &vllmv1.GetSummaryRequest{}, &vllmv1.GetSummaryResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_GetDataPoint_FullMethodName, llm.GetDataPoint, &vllmv1.GetDataPointRequest{}, &vllmv1.GetDataPointResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_ListTopDimension_FullMethodName, llm.ListTopDimension, &vllmv1.ListTopDimensionRequest{}, &vllmv1.ListTopDimensionResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_ListTopModel_FullMethodName, llm.ListTopModel, &vllmv1.ListTopModelRequest{}, &vllmv1.ListTopModelResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_ListTopTool_FullMethodName, llm.ListTopTool, &vllmv1.ListTopToolRequest{}, &vllmv1.ListTopToolResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_ListTopUser_FullMethodName, llm.ListTopUser, &vllmv1.ListTopUserRequest{}, &vllmv1.ListTopUserResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_ListTopSession_FullMethodName, llm.ListTopSession, &vllmv1.ListTopSessionRequest{}, &vllmv1.ListTopSessionResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_ListTopService_FullMethodName, llm.ListTopService, &vllmv1.ListTopServiceRequest{}, &vllmv1.ListTopServiceResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_ListTopPolicy_FullMethodName, llm.ListTopPolicy, &vllmv1.ListTopPolicyRequest{}, &vllmv1.ListTopPolicyResponse{}),
		newVisibilityProxyTest(vllmv1.LLMService_ListAccessLog_FullMethodName, llm.ListAccessLog, &vllmv1.ListAccessLogRequest{}, &vllmv1.ListAccessLogResponse{}),
		newVisibilityProxyTest(vmetricsv1.MetricsService_QueryMetrics_FullMethodName, metric.QueryMetrics, &vmetricsv1.QueryMetricsRequest{}, &vmetricsv1.QueryMetricsResponse{}),
		newVisibilityProxyTest(vmetricsv1.MetricsService_ListMetricDescriptors_FullMethodName, metric.ListMetricDescriptors, &vmetricsv1.ListMetricDescriptorsRequest{}, &vmetricsv1.ListMetricDescriptorsResponse{}),
		newVisibilityProxyTest(vmetricsv1.MetricsService_ListMetricCatalog_FullMethodName, metric.ListMetricCatalog, &vmetricsv1.ListMetricCatalogRequest{}, &vmetricsv1.ListMetricCatalogResponse{}),
		newVisibilityProxyTest(vmetricsv1.MetricsService_GetMetricsCapabilities_FullMethodName, metric.GetMetricsCapabilities, &vmetricsv1.GetMetricsCapabilitiesRequest{}, &vmetricsv1.GetMetricsCapabilitiesResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetServiceSummary_FullMethodName, resource.GetServiceSummary, &vcorev1.GetServiceSummaryRequest{}, &vcorev1.GetServiceSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetSessionSummary_FullMethodName, resource.GetSessionSummary, &vcorev1.GetSessionSummaryRequest{}, &vcorev1.GetSessionSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetUserSummary_FullMethodName, resource.GetUserSummary, &vcorev1.GetUserSummaryRequest{}, &vcorev1.GetUserSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetPolicySummary_FullMethodName, resource.GetPolicySummary, &vcorev1.GetPolicySummaryRequest{}, &vcorev1.GetPolicySummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetCredentialSummary_FullMethodName, resource.GetCredentialSummary, &vcorev1.GetCredentialSummaryRequest{}, &vcorev1.GetCredentialSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetDeviceSummary_FullMethodName, resource.GetDeviceSummary, &vcorev1.GetDeviceSummaryRequest{}, &vcorev1.GetDeviceSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetIdentityProviderSummary_FullMethodName, resource.GetIdentityProviderSummary, &vcorev1.GetIdentityProviderSummaryRequest{}, &vcorev1.GetIdentityProviderSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetAuthenticatorSummary_FullMethodName, resource.GetAuthenticatorSummary, &vcorev1.GetAuthenticatorSummaryRequest{}, &vcorev1.GetAuthenticatorSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetGroupSummary_FullMethodName, resource.GetGroupSummary, &vcorev1.GetGroupSummaryRequest{}, &vcorev1.GetGroupSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetGatewaySummary_FullMethodName, resource.GetGatewaySummary, &vcorev1.GetGatewaySummaryRequest{}, &vcorev1.GetGatewaySummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetSecretSummary_FullMethodName, resource.GetSecretSummary, &vcorev1.GetSecretSummaryRequest{}, &vcorev1.GetSecretSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetRegionSummary_FullMethodName, resource.GetRegionSummary, &vcorev1.GetRegionSummaryRequest{}, &vcorev1.GetRegionSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_GetNamespaceSummary_FullMethodName, resource.GetNamespaceSummary, &vcorev1.GetNamespaceSummaryRequest{}, &vcorev1.GetNamespaceSummaryResponse{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListUser_FullMethodName, resource.ListUser, &vcorev1.ListUserOptions{}, &corev1.UserList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListSession_FullMethodName, resource.ListSession, &vcorev1.ListSessionOptions{}, &corev1.SessionList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListService_FullMethodName, resource.ListService, &vcorev1.ListServiceOptions{}, &corev1.ServiceList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListNamespace_FullMethodName, resource.ListNamespace, &vcorev1.ListNamespaceOptions{}, &corev1.NamespaceList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListSecret_FullMethodName, resource.ListSecret, &vcorev1.ListSecretOptions{}, &corev1.SecretList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListGroup_FullMethodName, resource.ListGroup, &vcorev1.ListGroupOptions{}, &corev1.GroupList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListRegion_FullMethodName, resource.ListRegion, &vcorev1.ListRegionOptions{}, &corev1.RegionList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListGateway_FullMethodName, resource.ListGateway, &vcorev1.ListGatewayOptions{}, &corev1.GatewayList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListCredential_FullMethodName, resource.ListCredential, &vcorev1.ListCredentialOptions{}, &corev1.CredentialList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListPolicy_FullMethodName, resource.ListPolicy, &vcorev1.ListPolicyOptions{}, &corev1.PolicyList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListIdentityProvider_FullMethodName, resource.ListIdentityProvider, &vcorev1.ListIdentityProviderOptions{}, &corev1.IdentityProviderList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListDevice_FullMethodName, resource.ListDevice, &vcorev1.ListDeviceOptions{}, &corev1.DeviceList{}),
		newVisibilityProxyTest(vcorev1.ResourceService_ListAuthenticator_FullMethodName, resource.ListAuthenticator, &vcorev1.ListAuthenticatorOptions{}, &corev1.AuthenticatorList{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_GetPolicySummary_FullMethodName, resourceAccess.GetPolicySummary, &vaccessv1.GetPolicySummaryRequest{}, &vaccessv1.GetPolicySummaryResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_GetCatalogSummary_FullMethodName, resourceAccess.GetCatalogSummary, &vaccessv1.GetCatalogSummaryRequest{}, &vaccessv1.GetCatalogSummaryResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_GetRequestSummary_FullMethodName, resourceAccess.GetRequestSummary, &vaccessv1.GetRequestSummaryRequest{}, &vaccessv1.GetRequestSummaryResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_GetReviewSummary_FullMethodName, resourceAccess.GetReviewSummary, &vaccessv1.GetReviewSummaryRequest{}, &vaccessv1.GetReviewSummaryResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_GetSecretSummary_FullMethodName, resourceAccess.GetSecretSummary, &vaccessv1.GetSecretSummaryRequest{}, &vaccessv1.GetSecretSummaryResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_GetIntegrationSummary_FullMethodName, resourceAccess.GetIntegrationSummary, &vaccessv1.GetIntegrationSummaryRequest{}, &vaccessv1.GetIntegrationSummaryResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_GetIntegrationIdentitySummary_FullMethodName, resourceAccess.GetIntegrationIdentitySummary, &vaccessv1.GetIntegrationIdentitySummaryRequest{}, &vaccessv1.GetIntegrationIdentitySummaryResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_GetIntegrationBindingSummary_FullMethodName, resourceAccess.GetIntegrationBindingSummary, &vaccessv1.GetIntegrationBindingSummaryRequest{}, &vaccessv1.GetIntegrationBindingSummaryResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListPolicy_FullMethodName, resourceAccess.ListPolicy, &vaccessv1.ListPolicyOptions{}, &accessv1.PolicyList{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListCatalog_FullMethodName, resourceAccess.ListCatalog, &vaccessv1.ListCatalogOptions{}, &accessv1.CatalogList{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListRequest_FullMethodName, resourceAccess.ListRequest, &vaccessv1.ListRequestOptions{}, &accessv1.RequestList{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListReview_FullMethodName, resourceAccess.ListReview, &vaccessv1.ListReviewOptions{}, &accessv1.ReviewList{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListSecret_FullMethodName, resourceAccess.ListSecret, &vaccessv1.ListSecretOptions{}, &accessv1.SecretList{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListIntegration_FullMethodName, resourceAccess.ListIntegration, &vaccessv1.ListIntegrationOptions{}, &accessv1.IntegrationList{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListIntegrationIdentity_FullMethodName, resourceAccess.ListIntegrationIdentity, &vaccessv1.ListIntegrationIdentityOptions{}, &accessv1.IntegrationIdentityList{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListIntegrationBinding_FullMethodName, resourceAccess.ListIntegrationBinding, &vaccessv1.ListIntegrationBindingOptions{}, &accessv1.IntegrationBindingList{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_GetRequestDataPoint_FullMethodName, resourceAccess.GetRequestDataPoint, &vaccessv1.GetRequestDataPointRequest{}, &vaccessv1.GetRequestDataPointResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListRequestTop_FullMethodName, resourceAccess.ListRequestTop, &vaccessv1.ListRequestTopRequest{}, &vaccessv1.ListTopResponse{}),
		newVisibilityProxyTest(vaccessv1.ResourceService_ListReviewTop_FullMethodName, resourceAccess.ListReviewTop, &vaccessv1.ListReviewTopRequest{}, &vaccessv1.ListTopResponse{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_GetWorkspaceSummary_FullMethodName, resourceCordium.GetWorkspaceSummary, &vcordiumv1.GetWorkspaceSummaryRequest{}, &vcordiumv1.GetWorkspaceSummaryResponse{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_GetTemplateSummary_FullMethodName, resourceCordium.GetTemplateSummary, &vcordiumv1.GetTemplateSummaryRequest{}, &vcordiumv1.GetTemplateSummaryResponse{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_GetSpaceSummary_FullMethodName, resourceCordium.GetSpaceSummary, &vcordiumv1.GetSpaceSummaryRequest{}, &vcordiumv1.GetSpaceSummaryResponse{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_GetMembershipSummary_FullMethodName, resourceCordium.GetMembershipSummary, &vcordiumv1.GetMembershipSummaryRequest{}, &vcordiumv1.GetMembershipSummaryResponse{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_GetGitProviderSummary_FullMethodName, resourceCordium.GetGitProviderSummary, &vcordiumv1.GetGitProviderSummaryRequest{}, &vcordiumv1.GetGitProviderSummaryResponse{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_GetSecretSummary_FullMethodName, resourceCordium.GetSecretSummary, &vcordiumv1.GetSecretSummaryRequest{}, &vcordiumv1.GetSecretSummaryResponse{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_GetUserSecretSummary_FullMethodName, resourceCordium.GetUserSecretSummary, &vcordiumv1.GetUserSecretSummaryRequest{}, &vcordiumv1.GetUserSecretSummaryResponse{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_GetRegionSummary_FullMethodName, resourceCordium.GetRegionSummary, &vcordiumv1.GetRegionSummaryRequest{}, &vcordiumv1.GetRegionSummaryResponse{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_ListWorkspace_FullMethodName, resourceCordium.ListWorkspace, &vcordiumv1.ListWorkspaceOptions{}, &cordiumv1.WorkspaceList{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_ListTemplate_FullMethodName, resourceCordium.ListTemplate, &vcordiumv1.ListTemplateOptions{}, &cordiumv1.TemplateList{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_ListSpace_FullMethodName, resourceCordium.ListSpace, &vcordiumv1.ListSpaceOptions{}, &cordiumv1.SpaceList{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_ListMembership_FullMethodName, resourceCordium.ListMembership, &vcordiumv1.ListMembershipOptions{}, &cordiumv1.MembershipList{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_ListGitProvider_FullMethodName, resourceCordium.ListGitProvider, &vcordiumv1.ListGitProviderOptions{}, &cordiumv1.GitProviderList{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_ListSecret_FullMethodName, resourceCordium.ListSecret, &vcordiumv1.ListSecretOptions{}, &cordiumv1.SecretList{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_ListUserSecret_FullMethodName, resourceCordium.ListUserSecret, &vcordiumv1.ListUserSecretOptions{}, &cordiumv1.UserSecretList{}),
		newVisibilityProxyTest(vcordiumv1.ResourceService_ListRegion_FullMethodName, resourceCordium.ListRegion, &vcordiumv1.ListRegionOptions{}, &cordiumv1.RegionList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetCollectorExporterSummary_FullMethodName, resourceEnterprise.GetCollectorExporterSummary, &venterprisev1.GetCollectorExporterSummaryRequest{}, &venterprisev1.GetCollectorExporterSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetSecretSummary_FullMethodName, resourceEnterprise.GetSecretSummary, &venterprisev1.GetSecretSummaryRequest{}, &venterprisev1.GetSecretSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetSecretStoreSummary_FullMethodName, resourceEnterprise.GetSecretStoreSummary, &venterprisev1.GetSecretStoreSummaryRequest{}, &venterprisev1.GetSecretStoreSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetCertificateSummary_FullMethodName, resourceEnterprise.GetCertificateSummary, &venterprisev1.GetCertificateSummaryRequest{}, &venterprisev1.GetCertificateSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetCertificateIssuerSummary_FullMethodName, resourceEnterprise.GetCertificateIssuerSummary, &venterprisev1.GetCertificateIssuerSummaryRequest{}, &venterprisev1.GetCertificateIssuerSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetDNSProviderSummary_FullMethodName, resourceEnterprise.GetDNSProviderSummary, &venterprisev1.GetDNSProviderSummaryRequest{}, &venterprisev1.GetDNSProviderSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetDirectoryProviderSummary_FullMethodName, resourceEnterprise.GetDirectoryProviderSummary, &venterprisev1.GetDirectoryProviderSummaryRequest{}, &venterprisev1.GetDirectoryProviderSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetDirectoryProviderUserSummary_FullMethodName, resourceEnterprise.GetDirectoryProviderUserSummary, &venterprisev1.GetDirectoryProviderUserSummaryRequest{}, &venterprisev1.GetDirectoryProviderUserSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetDirectoryProviderGroupSummary_FullMethodName, resourceEnterprise.GetDirectoryProviderGroupSummary, &venterprisev1.GetDirectoryProviderGroupSummaryRequest{}, &venterprisev1.GetDirectoryProviderGroupSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_GetDeviceManagerSummary_FullMethodName, resourceEnterprise.GetDeviceManagerSummary, &venterprisev1.GetDeviceManagerSummaryRequest{}, &venterprisev1.GetDeviceManagerSummaryResponse{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListCollectorExporter_FullMethodName, resourceEnterprise.ListCollectorExporter, &venterprisev1.ListCollectorExporterOptions{}, &enterprisev1.CollectorExporterList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListSecret_FullMethodName, resourceEnterprise.ListSecret, &venterprisev1.ListSecretOptions{}, &enterprisev1.SecretList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListSecretStore_FullMethodName, resourceEnterprise.ListSecretStore, &venterprisev1.ListSecretStoreOptions{}, &enterprisev1.SecretStoreList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListCertificate_FullMethodName, resourceEnterprise.ListCertificate, &venterprisev1.ListCertificateOptions{}, &enterprisev1.CertificateList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListCertificateIssuer_FullMethodName, resourceEnterprise.ListCertificateIssuer, &venterprisev1.ListCertificateIssuerOptions{}, &enterprisev1.CertificateIssuerList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListDNSProvider_FullMethodName, resourceEnterprise.ListDNSProvider, &venterprisev1.ListDNSProviderOptions{}, &enterprisev1.DNSProviderList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListDirectoryProvider_FullMethodName, resourceEnterprise.ListDirectoryProvider, &venterprisev1.ListDirectoryProviderOptions{}, &enterprisev1.DirectoryProviderList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListDirectoryProviderUser_FullMethodName, resourceEnterprise.ListDirectoryProviderUser, &venterprisev1.ListDirectoryProviderUserOptions{}, &enterprisev1.DirectoryProviderUserList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListDirectoryProviderGroup_FullMethodName, resourceEnterprise.ListDirectoryProviderGroup, &venterprisev1.ListDirectoryProviderGroupOptions{}, &enterprisev1.DirectoryProviderGroupList{}),
		newVisibilityProxyTest(venterprisev1.ResourceService_ListDeviceManager_FullMethodName, resourceEnterprise.ListDeviceManager, &venterprisev1.ListDeviceManagerOptions{}, &enterprisev1.DeviceManagerList{}),
	}
	for _, tt := range cases {
		t.Run(tt.method, func(t *testing.T) {
			original := proto.Clone(tt.request)
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()
			ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-request-id", "visibility-test"))
			for _, failure := range []error{nil, status.Error(codes.Unavailable, "store unavailable"), status.Error(codes.PermissionDenied, "forbidden"), status.Error(codes.DeadlineExceeded, "deadline exceeded")} {
				calls := 0
				conn.invoke = func(gotCtx context.Context, method string, req, res any) error {
					calls++
					assert.Same(t, ctx, gotCtx)
					assert.Equal(t, tt.method, method)
					assert.Same(t, tt.request, req)
					assert.IsType(t, tt.response, res)
					if failure != nil {
						return failure
					}
					proto.Merge(res.(proto.Message), tt.response)
					return nil
				}
				res, err := tt.call(ctx)
				assert.True(t, proto.Equal(original, tt.request))
				assert.Equal(t, 1, calls)
				if failure != nil {
					assert.Equal(t, failure, err)
					assert.True(t, res == nil || !res.ProtoReflect().IsValid())
					continue
				}
				require.Nil(t, err)
				assert.True(t, proto.Equal(tt.response, res), "%+v", res)
			}
		})
	}
}

func TestVisibilityProxyErrorDetails(t *testing.T) {
	original, err := status.New(codes.ResourceExhausted, "query limit").WithDetails(wrapperspb.String("retry later"))
	require.Nil(t, err)
	conn := &visibilityClientConn{invoke: func(ctx context.Context, method string, req, res any) error {
		return original.Err()
	}}
	srv := &ServerMetric{c: vmetricsv1.NewMetricsServiceClient(conn)}
	_, err = srv.QueryMetrics(t.Context(), &vmetricsv1.QueryMetricsRequest{})
	assert.Equal(t, original.Proto(), status.Convert(err).Proto())
}
