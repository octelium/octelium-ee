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

func (s *ServerAuditLog) ListAuditLog(ctx context.Context, req *visibilityv1.ListAuditLogRequest) (*visibilityv1.ListAuditLogResponse, error) {
	return s.c.ListAuditLog(ctx, req)
}

func (s *ServerAuditLog) GetAuditLogSummary(ctx context.Context, req *visibilityv1.GetAuditLogSummaryRequest) (*visibilityv1.GetAuditLogSummaryResponse, error) {
	return s.c.GetAuditLogSummary(ctx, req)
}

func (s *ServerAuditLog) GetAuditLogDataPoint(ctx context.Context, req *visibilityv1.GetAuditLogDataPointRequest) (*visibilityv1.GetAuditLogDataPointResponse, error) {
	return s.c.GetAuditLogDataPoint(ctx, req)
}

func (s *ServerAuditLog) ListAuditLogTopUser(ctx context.Context, req *visibilityv1.ListAuditLogTopUserRequest) (*visibilityv1.ListAuditLogTopUserResponse, error) {
	return s.c.ListAuditLogTopUser(ctx, req)
}

func (s *ServerAuditLog) ListAuditLogTopSession(ctx context.Context, req *visibilityv1.ListAuditLogTopSessionRequest) (*visibilityv1.ListAuditLogTopSessionResponse, error) {
	return s.c.ListAuditLogTopSession(ctx, req)
}
