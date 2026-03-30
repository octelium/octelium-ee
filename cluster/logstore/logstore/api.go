// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"context"

	"github.com/octelium/octelium/apis/main/visibilityv1"
)

type srvAccessLog struct {
	s *Server
	visibilityv1.UnimplementedAccessLogServiceServer
}

func (s *srvAccessLog) ListAccessLog(ctx context.Context, req *visibilityv1.ListAccessLogRequest) (*visibilityv1.ListAccessLogResponse, error) {
	return s.s.listAccessLog(ctx, req)
}

func (s *srvAccessLog) ListSSHSession(ctx context.Context, req *visibilityv1.ListSSHSessionRequest) (*visibilityv1.ListSSHSessionResponse, error) {
	return s.s.listSSHSession(ctx, req)
}

func (s *srvAccessLog) GetSSHSession(ctx context.Context, req *visibilityv1.GetSSHSessionRequest) (*visibilityv1.SSHSession, error) {
	return s.s.getSSHSession(ctx, req)
}

func (s *srvAccessLog) ListSSHSessionRecording(ctx context.Context, req *visibilityv1.ListSSHSessionRecordingRequest) (*visibilityv1.ListSSHSessionRecordingResponse, error) {
	return s.s.listSSHSessionRecording(ctx, req)
}

func (s *srvAccessLog) GetAccessLogSummary(ctx context.Context, req *visibilityv1.GetAccessLogSummaryRequest) (*visibilityv1.GetAccessLogSummaryResponse, error) {
	return s.s.getSummaryAccessLog(ctx, req)
}

func (s *srvAccessLog) GetAccessLogDataPoint(ctx context.Context, req *visibilityv1.GetAccessLogDataPointRequest) (*visibilityv1.GetAccessLogDataPointResponse, error) {
	return s.s.getAccessLogDataPoint(ctx, req)
}

func (s *srvAccessLog) ListAccessLogTopUser(ctx context.Context, req *visibilityv1.ListAccessLogTopUserRequest) (*visibilityv1.ListAccessLogTopUserResponse, error) {
	return s.s.listAccessLogTopUser(ctx, req)
}

func (s *srvAccessLog) ListAccessLogTopService(ctx context.Context, req *visibilityv1.ListAccessLogTopServiceRequest) (*visibilityv1.ListAccessLogTopServiceResponse, error) {
	return s.s.listAccessLogTopService(ctx, req)
}

func (s *srvAccessLog) ListAccessLogTopPolicy(ctx context.Context, req *visibilityv1.ListAccessLogTopPolicyRequest) (*visibilityv1.ListAccessLogTopPolicyResponse, error) {
	return s.s.listAccessLogTopPolicy(ctx, req)
}

func (s *srvAccessLog) ListAccessLogTopSession(ctx context.Context, req *visibilityv1.ListAccessLogTopSessionRequest) (*visibilityv1.ListAccessLogTopSessionResponse, error) {
	return s.s.listAccessLogTopSession(ctx, req)
}

type srvAuthenticationLog struct {
	s *Server
	visibilityv1.UnimplementedAuthenticationLogServiceServer
}

func (s *srvAuthenticationLog) ListAuthenticationLog(ctx context.Context, req *visibilityv1.ListAuthenticationLogRequest) (*visibilityv1.ListAuthenticationLogResponse, error) {
	return s.s.listAuthenticationLog(ctx, req)
}

func (s *srvAuthenticationLog) GetAuthenticationLogSummary(ctx context.Context, req *visibilityv1.GetAuthenticationLogSummaryRequest) (*visibilityv1.GetAuthenticationLogSummaryResponse, error) {
	return s.s.getSummaryAuthenticationLog(ctx, req)
}

func (s *srvAuthenticationLog) GetAuthenticationLogDataPoint(ctx context.Context, req *visibilityv1.GetAuthenticationLogDataPointRequest) (*visibilityv1.GetAuthenticationLogDataPointResponse, error) {
	return s.s.getAuthenticationLogDataPoint(ctx, req)
}

func (s *srvAuthenticationLog) ListAuthenticationLogTopUser(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopUserRequest) (*visibilityv1.ListAuthenticationLogTopUserResponse, error) {
	return s.s.listAuthenticationLogTopUser(ctx, req)
}

func (s *srvAuthenticationLog) ListAuthenticationLogTopCredential(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopCredentialRequest) (*visibilityv1.ListAuthenticationLogTopCredentialResponse, error) {
	return s.s.listAuthenticationLogTopCredential(ctx, req)
}

func (s *srvAuthenticationLog) ListAuthenticationLogTopIdentityProvider(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopIdentityProviderRequest) (*visibilityv1.ListAuthenticationLogTopIdentityProviderResponse, error) {
	return s.s.listAuthenticationLogTopIdentityProvider(ctx, req)
}

type srvAuditLog struct {
	s *Server
	visibilityv1.UnimplementedAuditLogServiceServer
}

func (s *srvAuditLog) ListAuditLog(ctx context.Context, req *visibilityv1.ListAuditLogRequest) (*visibilityv1.ListAuditLogResponse, error) {
	return s.s.listAuditLog(ctx, req)
}

func (s *srvAuditLog) GetAuditLogSummary(ctx context.Context, req *visibilityv1.GetAuditLogSummaryRequest) (*visibilityv1.GetAuditLogSummaryResponse, error) {
	return s.s.getSummaryAuditLog(ctx, req)
}

type srvComponentLog struct {
	s *Server
	visibilityv1.UnimplementedComponentLogServiceServer
}

func (s *srvComponentLog) ListComponentLog(ctx context.Context, req *visibilityv1.ListComponentLogRequest) (*visibilityv1.ListComponentLogResponse, error) {
	return s.s.listComponentLog(ctx, req)
}

func (s *srvComponentLog) GetComponentLogSummary(ctx context.Context, req *visibilityv1.GetComponentLogSummaryRequest) (*visibilityv1.GetComponentLogSummaryResponse, error) {
	return s.s.getSummaryComponentLog(ctx, req)
}
