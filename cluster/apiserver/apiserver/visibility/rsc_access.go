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

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
)

func (s *ServerResourceAccess) GetPolicySummary(ctx context.Context, req *vaccessv1.GetPolicySummaryRequest) (*vaccessv1.GetPolicySummaryResponse, error) {
	return s.accessC.GetPolicySummary(ctx, req)
}

func (s *ServerResourceAccess) GetCatalogSummary(ctx context.Context, req *vaccessv1.GetCatalogSummaryRequest) (*vaccessv1.GetCatalogSummaryResponse, error) {
	return s.accessC.GetCatalogSummary(ctx, req)
}

func (s *ServerResourceAccess) GetRequestSummary(ctx context.Context, req *vaccessv1.GetRequestSummaryRequest) (*vaccessv1.GetRequestSummaryResponse, error) {
	return s.accessC.GetRequestSummary(ctx, req)
}

func (s *ServerResourceAccess) GetReviewSummary(ctx context.Context, req *vaccessv1.GetReviewSummaryRequest) (*vaccessv1.GetReviewSummaryResponse, error) {
	return s.accessC.GetReviewSummary(ctx, req)
}

func (s *ServerResourceAccess) ListPolicy(ctx context.Context, req *vaccessv1.ListPolicyOptions) (*accessv1.PolicyList, error) {
	return s.accessC.ListPolicy(ctx, req)
}

func (s *ServerResourceAccess) ListCatalog(ctx context.Context, req *vaccessv1.ListCatalogOptions) (*accessv1.CatalogList, error) {
	return s.accessC.ListCatalog(ctx, req)
}

func (s *ServerResourceAccess) ListRequest(ctx context.Context, req *vaccessv1.ListRequestOptions) (*accessv1.RequestList, error) {
	return s.accessC.ListRequest(ctx, req)
}

func (s *ServerResourceAccess) ListReview(ctx context.Context, req *vaccessv1.ListReviewOptions) (*accessv1.ReviewList, error) {
	return s.accessC.ListReview(ctx, req)
}
