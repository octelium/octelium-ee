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

	"github.com/octelium/octelium/apis/main/visibilityv1"
)

func (s *ServerAccessLog) ListAccessLog(ctx context.Context, req *visibilityv1.ListAccessLogRequest) (*visibilityv1.ListAccessLogResponse, error) {
	return s.c.ListAccessLog(ctx, req)
}

func (s *ServerAccessLog) ListSSHSession(ctx context.Context, req *visibilityv1.ListSSHSessionRequest) (*visibilityv1.ListSSHSessionResponse, error) {
	return s.c.ListSSHSession(ctx, req)
}

func (s *ServerAccessLog) ListSSHSessionRecording(ctx context.Context, req *visibilityv1.ListSSHSessionRecordingRequest) (*visibilityv1.ListSSHSessionRecordingResponse, error) {
	return s.c.ListSSHSessionRecording(ctx, req)
}

func (s *ServerAccessLog) GetAccessLogSummary(ctx context.Context, req *visibilityv1.GetAccessLogSummaryRequest) (*visibilityv1.GetAccessLogSummaryResponse, error) {
	return s.c.GetAccessLogSummary(ctx, req)
}

func (s *ServerAccessLog) GetSSHSession(ctx context.Context, req *visibilityv1.GetSSHSessionRequest) (*visibilityv1.SSHSession, error) {
	return s.c.GetSSHSession(ctx, req)
}

func (s *ServerAccessLog) GetAccessLogDataPoint(ctx context.Context, req *visibilityv1.GetAccessLogDataPointRequest) (*visibilityv1.GetAccessLogDataPointResponse, error) {
	return s.c.GetAccessLogDataPoint(ctx, req)
}

func (s *ServerAccessLog) ListAccessLogTopUser(ctx context.Context, req *visibilityv1.ListAccessLogTopUserRequest) (*visibilityv1.ListAccessLogTopUserResponse, error) {
	return s.c.ListAccessLogTopUser(ctx, req)
}

func (s *ServerAccessLog) ListAccessLogTopService(ctx context.Context, req *visibilityv1.ListAccessLogTopServiceRequest) (*visibilityv1.ListAccessLogTopServiceResponse, error) {
	return s.c.ListAccessLogTopService(ctx, req)
}

func (s *ServerAccessLog) ListAccessLogTopPolicy(ctx context.Context, req *visibilityv1.ListAccessLogTopPolicyRequest) (*visibilityv1.ListAccessLogTopPolicyResponse, error) {
	return s.c.ListAccessLogTopPolicy(ctx, req)
}

func (s *ServerAccessLog) ListAccessLogTopSession(ctx context.Context, req *visibilityv1.ListAccessLogTopSessionRequest) (*visibilityv1.ListAccessLogTopSessionResponse, error) {
	return s.c.ListAccessLogTopSession(ctx, req)
}
