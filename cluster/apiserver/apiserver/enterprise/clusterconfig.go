// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package enterprise

import (
	"context"
	"slices"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

func (s *Server) GetClusterConfig(ctx context.Context, req *enterprisev1.GetClusterConfigRequest) (*enterprisev1.ClusterConfig, error) {
	cc, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return cc, nil
}

func (s *Server) UpdateClusterConfig(ctx context.Context, req *enterprisev1.ClusterConfig) (*enterprisev1.ClusterConfig, error) {

	if err := s.validateClusterConfig(ctx, req); err != nil {
		return nil, err
	}

	cfg, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	apisrvcommon.MetadataUpdate(cfg.Metadata, req.Metadata)
	cfg.Spec = req.Spec

	ccOut, err := s.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cfg)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return ccOut, nil
}

func (s *Server) validateClusterConfig(ctx context.Context, req *enterprisev1.ClusterConfig) error {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return err
	}

	if req.Spec.Collector != nil {

		exporterList, err := s.ListCollectorExporter(ctx, &enterprisev1.ListCollectorExporterOptions{})
		if err != nil {
			return err
		}

		for _, pipeline := range req.Spec.Collector.Pipelines {
			if err := apivalidation.ValidateName(pipeline.Name, 0, 0); err != nil {
				return err
			}

			switch pipeline.Type {
			case enterprisev1.ClusterConfig_Spec_Collector_Pipeline_TYPE_UNSET:
				return grpcutils.InvalidArg("Pipeline type must be set to either LOGS or METRICS")
			}

			if len(pipeline.Exporters) > 64 {
				return grpcutils.InvalidArg("Too many exporters")
			}

			for _, exporter := range pipeline.Exporters {
				if !slices.ContainsFunc(exporterList.Items, func(exp *enterprisev1.CollectorExporter) bool {
					return exp.Metadata.Name == exporter
				}) {
					return grpcutils.InvalidArg("This CollectorExport does not exist: %s", exporter)
				}
			}
		}
	}
	return nil
}
