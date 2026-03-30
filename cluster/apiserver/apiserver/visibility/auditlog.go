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
