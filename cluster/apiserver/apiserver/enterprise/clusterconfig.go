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

	if req.Spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}

	if req.Spec.Collector == nil {
		return nil
	}

	const (
		maxPipelines            = 64
		maxExportersPerPipeline = 64
		maxPipelineNameLen      = 64
	)

	if len(req.Spec.Collector.Pipelines) > maxPipelines {
		return grpcutils.InvalidArg("Too many collector pipelines")
	}

	exporterList, err := s.ListCollectorExporter(ctx, &enterprisev1.ListCollectorExporterOptions{})
	if err != nil {
		return err
	}

	exporterSet := make(map[string]struct{}, len(exporterList.Items))
	for _, exporter := range exporterList.Items {
		exporterSet[exporter.Metadata.Name] = struct{}{}
	}

	seenPipelines := make(map[string]struct{}, len(req.Spec.Collector.Pipelines))

	for _, pipeline := range req.Spec.Collector.Pipelines {
		if pipeline == nil {
			return grpcutils.InvalidArg("Nil collector pipeline")
		}

		if err := apivalidation.ValidateName(pipeline.Name, 1, maxPipelineNameLen); err != nil {
			return err
		}

		if _, ok := seenPipelines[pipeline.Name]; ok {
			return grpcutils.InvalidArg("Duplicate collector pipeline name: %s", pipeline.Name)
		}
		seenPipelines[pipeline.Name] = struct{}{}

		switch pipeline.Type {
		case enterprisev1.ClusterConfig_Spec_Collector_Pipeline_LOGS,
			enterprisev1.ClusterConfig_Spec_Collector_Pipeline_METRICS:
		case enterprisev1.ClusterConfig_Spec_Collector_Pipeline_TYPE_UNSET:
			return grpcutils.InvalidArg("Pipeline type must be set to either LOGS or METRICS")
		default:
			return grpcutils.InvalidArg("Invalid collector pipeline type")
		}

		if len(pipeline.Exporters) == 0 {
			return grpcutils.InvalidArg("Collector pipeline %s must include at least one exporter", pipeline.Name)
		}

		if len(pipeline.Exporters) > maxExportersPerPipeline {
			return grpcutils.InvalidArg("Too many exporters in collector pipeline: %s", pipeline.Name)
		}

		seenExporters := make(map[string]struct{}, len(pipeline.Exporters))
		for _, exporterName := range pipeline.Exporters {
			if err := apivalidation.ValidateName(exporterName, 1, 0); err != nil {
				return err
			}

			if _, ok := seenExporters[exporterName]; ok {
				return grpcutils.InvalidArg(
					"Duplicate exporter %s in collector pipeline %s",
					exporterName,
					pipeline.Name,
				)
			}
			seenExporters[exporterName] = struct{}{}

			if _, ok := exporterSet[exporterName]; !ok {
				return grpcutils.InvalidArg("CollectorExporter does not exist: %s", exporterName)
			}
		}
	}

	return nil
}
