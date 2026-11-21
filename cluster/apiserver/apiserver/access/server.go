// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/cluster/caccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/cluster/common/celengine"
	oc "github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"google.golang.org/grpc"
)

type celEngineHolder struct {
	once   sync.Once
	engine *celengine.CELEngine
	err    error
}

func (h *celEngineHolder) get(ctx context.Context) (*celengine.CELEngine, error) {
	h.once.Do(func() {
		h.engine, h.err = celengine.New(ctx, &celengine.Opts{})
	})

	return h.engine, h.err
}

type ServerMain struct {
	octeliumC octeliumc.ClientInterface
	accessv1.UnimplementedMainServiceServer

	cel celEngineHolder
}

func NewServerMain(octeliumC octeliumc.ClientInterface) *ServerMain {
	return &ServerMain{
		octeliumC: octeliumC,
	}
}

type ServerUser struct {
	octeliumC octeliumc.ClientInterface
	accessv1.UnimplementedUserServiceServer

	rscStoreC caccessv1.InternalServiceClient

	cel celEngineHolder
}

func NewServerUser(ctx context.Context, octeliumC octeliumc.ClientInterface) (*ServerUser, error) {

	var host string

	if ovutils.IsMockMode() {
		host = "localhost:40001"
	} else if ldflags.IsTest() {
		host = fmt.Sprintf("localhost:%s", os.Getenv("OCTELIUM_TEST_RSCSTORE_PORT"))
	} else {
		host = "octeliumee-rscstore.octelium.svc:8080"
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

	return &ServerUser{
		octeliumC: octeliumC,
		rscStoreC: caccessv1.NewInternalServiceClient(grpcConn),
	}, nil
}

type ServerReviewer struct {
	octeliumC octeliumc.ClientInterface
	accessv1.UnimplementedReviewerServiceServer

	cel celEngineHolder
}

func NewServerReviewer(octeliumC octeliumc.ClientInterface) *ServerReviewer {
	return &ServerReviewer{
		octeliumC: octeliumC,
	}
}
