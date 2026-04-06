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

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	pb "github.com/octelium/octelium/apis/main/enterprisev1"
	oc "github.com/octelium/octelium/cluster/common/octeliumc"
	"google.golang.org/grpc"
)

type Server struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedPolicyPortalServiceServer
	c pb.PolicyPortalServiceClient
}

func NewServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {

	var host string

	{
		host = "octeliumee-policyportal.octelium.svc:8080"
	}

	grpcOpts, err := oc.DefaultDialOpts(ctx)
	if err != nil {
		return nil, err
	}

	grpcConn, err := grpc.NewClient(
		host, grpcOpts...,
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
