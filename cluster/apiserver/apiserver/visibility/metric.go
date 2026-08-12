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

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
)

func (s *ServerMetric) QueryMetrics(ctx context.Context, req *vmetricsv1.QueryMetricsRequest) (*vmetricsv1.QueryMetricsResponse, error) {
	return s.c.QueryMetrics(ctx, req)
}

func (s *ServerMetric) ListMetricDescriptors(ctx context.Context, req *vmetricsv1.ListMetricDescriptorsRequest) (*vmetricsv1.ListMetricDescriptorsResponse, error) {
	return s.c.ListMetricDescriptors(ctx, req)
}

func (s *ServerMetric) ListMetricCatalog(ctx context.Context, req *vmetricsv1.ListMetricCatalogRequest) (*vmetricsv1.ListMetricCatalogResponse, error) {
	return s.c.ListMetricCatalog(ctx, req)
}

func (s *ServerMetric) GetMetricsCapabilities(ctx context.Context, req *vmetricsv1.GetMetricsCapabilitiesRequest) (*vmetricsv1.GetMetricsCapabilitiesResponse, error) {
	return s.c.GetMetricsCapabilities(ctx, req)
}
