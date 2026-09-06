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

	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
)

func (s *ServerLLM) GetSummary(ctx context.Context, req *vllmv1.GetSummaryRequest) (*vllmv1.GetSummaryResponse, error) {
	return s.c.GetSummary(ctx, req)
}

func (s *ServerLLM) GetDataPoint(ctx context.Context, req *vllmv1.GetDataPointRequest) (*vllmv1.GetDataPointResponse, error) {
	return s.c.GetDataPoint(ctx, req)
}

func (s *ServerLLM) ListTopDimension(ctx context.Context, req *vllmv1.ListTopDimensionRequest) (*vllmv1.ListTopDimensionResponse, error) {
	return s.c.ListTopDimension(ctx, req)
}

func (s *ServerLLM) ListTopModel(ctx context.Context, req *vllmv1.ListTopModelRequest) (*vllmv1.ListTopModelResponse, error) {
	return s.c.ListTopModel(ctx, req)
}

func (s *ServerLLM) ListTopTool(ctx context.Context, req *vllmv1.ListTopToolRequest) (*vllmv1.ListTopToolResponse, error) {
	return s.c.ListTopTool(ctx, req)
}

func (s *ServerLLM) ListTopUser(ctx context.Context, req *vllmv1.ListTopUserRequest) (*vllmv1.ListTopUserResponse, error) {
	return s.c.ListTopUser(ctx, req)
}

func (s *ServerLLM) ListTopSession(ctx context.Context, req *vllmv1.ListTopSessionRequest) (*vllmv1.ListTopSessionResponse, error) {
	return s.c.ListTopSession(ctx, req)
}

func (s *ServerLLM) ListTopService(ctx context.Context, req *vllmv1.ListTopServiceRequest) (*vllmv1.ListTopServiceResponse, error) {
	return s.c.ListTopService(ctx, req)
}

func (s *ServerLLM) ListTopPolicy(ctx context.Context, req *vllmv1.ListTopPolicyRequest) (*vllmv1.ListTopPolicyResponse, error) {
	return s.c.ListTopPolicy(ctx, req)
}

func (s *ServerLLM) ListAccessLog(ctx context.Context, req *vllmv1.ListAccessLogRequest) (*vllmv1.ListAccessLogResponse, error) {
	return s.c.ListAccessLog(ctx, req)
}
