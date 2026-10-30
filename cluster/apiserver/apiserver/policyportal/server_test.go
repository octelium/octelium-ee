// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package policyportal

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type policyPortalClient struct {
	enterprisev1.PolicyPortalServiceClient
	call func(context.Context, *enterprisev1.IsAuthorizedRequest) (*enterprisev1.IsAuthorizedResponse, error)
}

func (c *policyPortalClient) IsAuthorized(ctx context.Context, req *enterprisev1.IsAuthorizedRequest, opts ...grpc.CallOption) (*enterprisev1.IsAuthorizedResponse, error) {
	return c.call(ctx, req)
}

func TestIsAuthorized(t *testing.T) {
	for _, tt := range []struct {
		name     string
		response *enterprisev1.IsAuthorizedResponse
		err      error
	}{
		{name: "Allowed", response: &enterprisev1.IsAuthorizedResponse{IsAuthorized: true}},
		{name: "Denied", response: &enterprisev1.IsAuthorizedResponse{Reason: &corev1.AccessLog_Entry_Common_Reason{}}},
		{name: "InvalidArgument", err: status.Error(codes.InvalidArgument, "invalid service reference")},
		{name: "Unavailable", err: status.Error(codes.Unavailable, "policy portal unavailable")},
		{name: "Cancelled", err: status.Error(codes.Canceled, "request cancelled")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()
			ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-request-id", "policy-portal-test"))
			req := &enterprisev1.IsAuthorizedRequest{
				Downstream: &enterprisev1.IsAuthorizedRequest_UserRef{UserRef: &metav1.ObjectReference{Name: "user"}},
				Upstream:   &enterprisev1.IsAuthorizedRequest_ServiceRef{ServiceRef: &metav1.ObjectReference{Name: "service.default"}},
			}
			calls := 0
			srv := &Server{c: &policyPortalClient{call: func(gotCtx context.Context, got *enterprisev1.IsAuthorizedRequest) (*enterprisev1.IsAuthorizedResponse, error) {
				calls++
				assert.Same(t, ctx, gotCtx)
				assert.Same(t, req, got)
				return tt.response, tt.err
			}}}
			res, err := srv.IsAuthorized(ctx, req)
			assert.Equal(t, 1, calls)
			assert.Equal(t, tt.err, err)
			assert.Same(t, tt.response, res)
		})
	}
}
