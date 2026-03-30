// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscserver

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/apis/rsc/renterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/rscserver/rscserver"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	inner *rscserver.Server
}

func NewServer(ctx context.Context) (*Server, error) {
	ret := &Server{}

	opts := &rscserver.Opts{
		RegisterResourceFn: func(s grpc.ServiceRegistrar) error {
			rcorev1.RegisterResourceServiceServer(s, &struct {
				rcorev1.UnimplementedResourceServiceServer
			}{})

			renterprisev1.RegisterResourceServiceServer(s, &struct {
				renterprisev1.UnimplementedResourceServiceServer
			}{})

			return nil
		},

		NewResourceObject:     ovutils.NewResourceObject,
		NewResourceObjectList: ovutils.NewResourceObjectList,
	}

	inner, err := rscserver.NewServer(ctx, opts)
	if err != nil {
		return nil, err
	}
	ret.inner = inner

	return ret, nil
}

func (s *Server) getClusterConfig(ctx context.Context) (*enterprisev1.ClusterConfig, error) {
	ccI, err := s.inner.GetResource(ctx, &rmetav1.GetOptions{
		Name: "default",
	}, uenterprisev1.API, uenterprisev1.Version, uenterprisev1.KindClusterConfig)
	if err != nil {
		return nil, err
	}

	return ccI.(*enterprisev1.ClusterConfig), nil
}

func (s *Server) CreateResource(ctx context.Context, req umetav1.ResourceObjectI, api, version, kind string) (umetav1.ResourceObjectI, error) {
	return s.inner.CreateResource(ctx, req, api, version, kind)
}

func (s *Server) GetResource(ctx context.Context, req *rmetav1.GetOptions, api, version, kind string) (umetav1.ResourceObjectI, error) {
	return s.inner.GetResource(ctx, req, api, version, kind)
}

func (s *Server) Run(ctx context.Context) error {
	return s.inner.Run(ctx)
}

func (s *Server) setInitResources(ctx context.Context) error {
	if _, err := s.GetResource(ctx, &rmetav1.GetOptions{
		Name: "default",
	}, uenterprisev1.API, uenterprisev1.Version, uenterprisev1.KindClusterConfig); err != nil {
		if !grpcerr.IsNotFound(err) {
			return err
		}

		zap.L().Debug("Initializing a default cluster-config for enterprise")

		if _, err := s.CreateResource(ctx, &enterprisev1.ClusterConfig{
			ApiVersion: uenterprisev1.APIVersion,
			Kind:       uenterprisev1.KindClusterConfig,
			Metadata: &metav1.Metadata{
				Name: "default",
			},
			Spec:   &enterprisev1.ClusterConfig_Spec{},
			Status: &enterprisev1.ClusterConfig_Status{},
		},
			uenterprisev1.API, uenterprisev1.Version, uenterprisev1.KindClusterConfig); err != nil {
			return err
		}

	}
	return nil
}

func Run(ctx context.Context) error {

	zap.S().Debug("Starting Enterprise Resource server")

	srv, err := NewServer(ctx)
	if err != nil {
		return err
	}
	if err := srv.setInitResources(ctx); err != nil {
		return errors.Errorf("Could not set init resources: %+v", err)
	}

	zap.S().Debug("starting gRPC server...")

	if err := srv.Run(ctx); err != nil {
		return err
	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.S().Infof("Resource Server is now running...")

	<-ctx.Done()

	return nil
}
