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
	"math"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	pb "github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedPolicyPortalServiceServer
	c pb.PolicyPortalServiceClient
}

func NewServer(octeliumC octeliumc.ClientInterface) (*Server, error) {

	var host string

	{
		host = "octeliumee-policyportal.octelium.svc:8080"
	}

	grpcConn, err := grpc.Dial(
		host, getDialOpts()...,
	)
	if err != nil {
		return nil, err
	}

	return &Server{
		octeliumC: octeliumC,
		c:         pb.NewPolicyPortalServiceClient(grpcConn),
	}, nil
}

func (s *Server) IsAuthorized(ctx context.Context, req *enterprisev1.IsAuthorizedRequest) (*enterprisev1.IsAuthorizedResponse, error) {
	return s.c.IsAuthorized(ctx, req)
}

func getDialOpts() []grpc.DialOption {

	unaryTries := uint(32)

	if ldflags.IsTest() {
		unaryTries = 1
	}

	retryCodes := []codes.Code{
		codes.Unavailable,
		codes.ResourceExhausted,
		codes.Unknown,
		codes.Aborted,
		codes.DataLoss,
		codes.Internal,
		codes.DeadlineExceeded,
	}

	unaryMiddlewares := []grpc.UnaryClientInterceptor{
		grpc_retry.UnaryClientInterceptor(
			grpc_retry.WithMax(unaryTries),
			grpc_retry.WithBackoff(grpc_retry.BackoffLinear(1000*time.Millisecond)),
			grpc_retry.WithCodes(retryCodes...)),
	}

	streamMiddlewares := []grpc.StreamClientInterceptor{
		grpc_retry.StreamClientInterceptor(
			grpc_retry.WithMax(math.MaxUint32),
			grpc_retry.WithBackoff(grpc_retry.BackoffLinear(1000*time.Millisecond)),
			grpc_retry.WithCodes(retryCodes...)),
	}

	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(unaryMiddlewares...)),
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(streamMiddlewares...)),
	}
}
