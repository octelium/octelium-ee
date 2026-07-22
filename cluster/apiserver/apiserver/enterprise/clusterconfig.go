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
	"strings"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

const (
	maxClusterConfigCollectorPipelines                  = 64
	maxClusterConfigCollectorExportersPerPipeline       = 64
	maxClusterConfigCollectorInlineConfigBytes          = 1024 * 1024
	maxClusterConfigScalerReplicas                int32 = 1024
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

	if err := s.validateClusterConfigCollector(ctx, req.Spec.Collector); err != nil {
		return err
	}

	if err := validateClusterConfigScaler(req.Spec.Scaler); err != nil {
		return err
	}

	if err := validateClusterConfigCertificate(req.Spec.Certificate); err != nil {
		return err
	}

	return nil
}

func (s *Server) validateClusterConfigCollector(ctx context.Context, collector *enterprisev1.ClusterConfig_Spec_Collector) error {
	if collector == nil {
		return nil
	}

	if len(collector.AdditionalInlineConfig) > maxClusterConfigCollectorInlineConfigBytes {
		return grpcutils.InvalidArg("Collector additional inline config is too long")
	}

	if strings.ContainsRune(collector.AdditionalInlineConfig, '\x00') {
		return grpcutils.InvalidArg("Collector additional inline config contains invalid characters")
	}

	if len(collector.Pipelines) > maxClusterConfigCollectorPipelines {
		return grpcutils.InvalidArg("Too many collector pipelines")
	}

	if len(collector.Pipelines) == 0 {
		return nil
	}

	exporterList, err := s.ListCollectorExporter(ctx, &enterprisev1.ListCollectorExporterOptions{})
	if err != nil {
		return err
	}

	exporterSet := make(map[string]struct{}, len(exporterList.Items))
	for _, exporter := range exporterList.Items {
		exporterSet[exporter.Metadata.Name] = struct{}{}
	}

	seenPipelines := make(map[string]struct{}, len(collector.Pipelines))

	for _, pipeline := range collector.Pipelines {
		if pipeline == nil {
			return grpcutils.InvalidArg("Nil collector pipeline")
		}

		if err := apivalidation.ValidateName(pipeline.Name, 0, 0); err != nil {
			return grpcutils.InvalidArg("Invalid pipeline name: %s", pipeline.Name)
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

		if len(pipeline.Exporters) > maxClusterConfigCollectorExportersPerPipeline {
			return grpcutils.InvalidArg("Too many exporters in collector pipeline: %s", pipeline.Name)
		}

		seenExporters := make(map[string]struct{}, len(pipeline.Exporters))
		for _, exporterName := range pipeline.Exporters {
			if err := apivalidation.ValidateName(exporterName, 0, 0); err != nil {
				return grpcutils.InvalidArg("Invalid exporter name: %s", exporterName)
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

func validateClusterConfigScaler(scaler *enterprisev1.ClusterConfig_Spec_Scaler) error {
	if scaler == nil {
		return nil
	}

	if scaler.Octovigil != nil {
		if err := validateClusterConfigScalerReplicas("octovigil replicas", scaler.Octovigil.Replicas); err != nil {
			return err
		}
	}

	if scaler.Ingress != nil {
		if err := validateClusterConfigScalerReplicas("ingress replicas", scaler.Ingress.Replicas); err != nil {
			return err
		}
	}

	if scaler.Collector != nil {
		if err := validateClusterConfigScalerReplicas("collector replicas", scaler.Collector.Replicas); err != nil {
			return err
		}
	}

	return nil
}

func validateClusterConfigScalerReplicas(field string, replicas int32) error {
	if replicas < 0 {
		return grpcutils.InvalidArg("%s must be greater than or equal to zero", field)
	}

	if replicas > maxClusterConfigScalerReplicas {
		return grpcutils.InvalidArg("%s is too large", field)
	}

	return nil
}

func validateClusterConfigCertificate(certificate *enterprisev1.ClusterConfig_Spec_Certificate) error {
	if certificate == nil {
		return nil
	}

	switch certificate.DefaultMode {
	case enterprisev1.Certificate_Spec_MODE_UNSET,
		enterprisev1.Certificate_Spec_MANAGED,
		enterprisev1.Certificate_Spec_MANUAL:
		return nil
	default:
		return grpcutils.InvalidArg("Invalid default certificate mode")
	}
}
