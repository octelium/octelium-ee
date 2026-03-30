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

func (s *ServerMetric) GetCounter(ctx context.Context, req *visibilityv1.GetCounterRequest) (*visibilityv1.GetCounterResponse, error) {
	return s.c.GetCounter(ctx, req)
}

func (s *ServerMetric) GetGauge(ctx context.Context, req *visibilityv1.GetGaugeRequest) (*visibilityv1.GetGaugeResponse, error) {
	return s.c.GetGauge(ctx, req)
}

func (s *ServerMetric) GetHistogram(ctx context.Context, req *visibilityv1.GetHistogramRequest) (*visibilityv1.GetHistogramResponse, error) {
	return s.c.GetHistogram(ctx, req)
}
