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

func (s *ServerComponentLog) ListComponentLog(ctx context.Context, req *visibilityv1.ListComponentLogRequest) (*visibilityv1.ListComponentLogResponse, error) {
	return s.c.ListComponentLog(ctx, req)
}

func (s *ServerComponentLog) GetComponentLogSummary(ctx context.Context, req *visibilityv1.GetComponentLogSummaryRequest) (*visibilityv1.GetComponentLogSummaryResponse, error) {
	return s.c.GetComponentLogSummary(ctx, req)
}

func (s *ServerComponentLog) GetComponentLogDataPoint(ctx context.Context, req *visibilityv1.GetComponentLogDataPointRequest) (*visibilityv1.GetComponentLogDataPointResponse, error) {
	return s.c.GetComponentLogDataPoint(ctx, req)
}
