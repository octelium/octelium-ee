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

func (s *ServerAuthenticationLog) ListAuthenticationLog(ctx context.Context, req *visibilityv1.ListAuthenticationLogRequest) (*visibilityv1.ListAuthenticationLogResponse, error) {
	return s.c.ListAuthenticationLog(ctx, req)
}

func (s *ServerAuthenticationLog) GetAuthenticationLogSummary(ctx context.Context, req *visibilityv1.GetAuthenticationLogSummaryRequest) (*visibilityv1.GetAuthenticationLogSummaryResponse, error) {
	return s.c.GetAuthenticationLogSummary(ctx, req)
}

func (s *ServerAuthenticationLog) GetAuthenticationLogDataPoint(ctx context.Context, req *visibilityv1.GetAuthenticationLogDataPointRequest) (*visibilityv1.GetAuthenticationLogDataPointResponse, error) {
	return s.c.GetAuthenticationLogDataPoint(ctx, req)
}

func (s *ServerAuthenticationLog) ListAuthenticationLogTopUser(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopUserRequest) (*visibilityv1.ListAuthenticationLogTopUserResponse, error) {
	return s.c.ListAuthenticationLogTopUser(ctx, req)
}

func (s *ServerAuthenticationLog) ListAuthenticationLogTopCredential(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopCredentialRequest) (*visibilityv1.ListAuthenticationLogTopCredentialResponse, error) {
	return s.c.ListAuthenticationLogTopCredential(ctx, req)
}

func (s *ServerAuthenticationLog) ListAuthenticationLogTopIdentityProvider(ctx context.Context, req *visibilityv1.ListAuthenticationLogTopIdentityProviderRequest) (*visibilityv1.ListAuthenticationLogTopIdentityProviderResponse, error) {
	return s.c.ListAuthenticationLogTopIdentityProvider(ctx, req)
}
