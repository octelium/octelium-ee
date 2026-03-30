// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	pb "github.com/octelium/octelium/apis/main/accessv1"
)

type ServerMain struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedMainServiceServer
}

func NewServerMain(octeliumC octeliumc.ClientInterface) *ServerMain {
	return &ServerMain{
		octeliumC: octeliumC,
	}
}

type ServerUser struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedUserServiceServer
}

func NewServerUser(octeliumC octeliumc.ClientInterface) *ServerUser {
	return &ServerUser{
		octeliumC: octeliumC,
	}
}

type ServerReviewer struct {
	octeliumC octeliumc.ClientInterface
	pb.UnimplementedReviewerServiceServer
}

func NewServerReviewer(octeliumC octeliumc.ClientInterface) *ServerReviewer {
	return &ServerReviewer{
		octeliumC: octeliumC,
	}
}
