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

	"github.com/asaskevich/govalidator"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	apisrvcommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/grpcerr"
)

func (s *Server) CreateCollectorExporter(ctx context.Context, req *enterprisev1.CollectorExporter) (*enterprisev1.CollectorExporter, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	_, err := s.octeliumC.EnterpriseC().GetCollectorExporter(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
	if err == nil {
		return nil, serr.InvalidArg("The CollectorExporter %s already exists", req.Metadata.Name)
	}

	if !grpcerr.IsNotFound(err) {
		return nil, serr.K8sInternal(err)
	}

	item := &enterprisev1.CollectorExporter{
		Metadata: apisrvcommon.MetadataFrom(req.Metadata),
		Spec:     req.Spec,
		Status:   &enterprisev1.CollectorExporter_Status{},
	}

	item, err = s.octeliumC.EnterpriseC().CreateCollectorExporter(ctx, item)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return item, nil
}

func (s *Server) GetCollectorExporter(ctx context.Context, req *metav1.GetOptions) (*enterprisev1.CollectorExporter, error) {
	if err := apisrvcommon.CheckGetOrDeleteOptions(req); err != nil {
		return nil, err
	}

	ret, err := s.octeliumC.EnterpriseC().GetCollectorExporter(ctx, &rmetav1.GetOptions{
		Uid:  req.Uid,
		Name: req.Name,
	})
	if err != nil {
		return nil, serr.K8sNotFoundOrInternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) ListCollectorExporter(ctx context.Context, req *enterprisev1.ListCollectorExporterOptions) (*enterprisev1.CollectorExporterList, error) {

	itemList, err := s.octeliumC.EnterpriseC().ListCollectorExporter(ctx, urscsrv.GetPublicListOptions(req))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	return itemList, nil
}

func (s *Server) DeleteCollectorExporter(ctx context.Context, req *metav1.DeleteOptions) (*metav1.OperationResult, error) {

	g, err := s.octeliumC.EnterpriseC().GetCollectorExporter(ctx, &rmetav1.GetOptions{Name: req.Name, Uid: req.Uid})
	if err != nil {
		return nil, err
	}

	_, err = s.octeliumC.EnterpriseC().DeleteCollectorExporter(ctx, &rmetav1.DeleteOptions{Uid: g.Metadata.Uid})
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return &metav1.OperationResult{}, nil
}

func (s *Server) UpdateCollectorExporter(ctx context.Context, req *enterprisev1.CollectorExporter) (*enterprisev1.CollectorExporter, error) {

	if err := apivalidation.ValidateCommon(req, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{
			RequireName: true,
		},
	}); err != nil {
		return nil, err
	}

	item, err := s.octeliumC.EnterpriseC().GetCollectorExporter(ctx, &rmetav1.GetOptions{Name: req.Metadata.Name})
	if err != nil {
		return nil, err
	}

	if item.Metadata.IsSystem {
		return nil, serr.InvalidArg("Cannot update the Group %s since it's a system object", item.Metadata.Name)
	}

	apisrvcommon.MetadataUpdate(item.Metadata, req.Metadata)
	item.Spec = req.Spec

	item, err = s.octeliumC.EnterpriseC().UpdateCollectorExporter(ctx, item)
	if err != nil {
		return nil, serr.K8sInternal(err)
	}

	return item, nil
}

func (s *Server) validateCollectorExporter(ctx context.Context, req *enterprisev1.CollectorExporter) error {
	spec := req.Spec
	if spec == nil {
		return grpcutils.InvalidArg("Nil spec")
	}
	switch spec.Type.(type) {
	case *enterprisev1.CollectorExporter_Spec_AzureDataExplorer_:
	case *enterprisev1.CollectorExporter_Spec_AzureMonitor_:
	case *enterprisev1.CollectorExporter_Spec_Clickhouse_:

		spec := spec.GetClickhouse()
		if spec.Endpoint == "" {
			return grpcutils.InvalidArg("Empty endpoint")
		}

		if len(spec.Endpoint) > 150 {
			return grpcutils.InvalidArg("Endpoint is too long")
		}

		switch {
		case govalidator.IsDNSName(spec.Endpoint) || govalidator.IsHost(spec.Endpoint) || govalidator.IsURL(spec.Endpoint):
		default:
			return grpcutils.InvalidArg("Invalid endpoint: %s", spec.Endpoint)
		}

	case *enterprisev1.CollectorExporter_Spec_Datadog_:
	case *enterprisev1.CollectorExporter_Spec_Elasticsearch_:
	case *enterprisev1.CollectorExporter_Spec_InfluxDB_:
	case *enterprisev1.CollectorExporter_Spec_Kafka_:
	case *enterprisev1.CollectorExporter_Spec_Logzio_:
	case *enterprisev1.CollectorExporter_Spec_Otlp:
		spec := spec.GetOtlp()
		if spec.Endpoint == "" {
			return grpcutils.InvalidArg("Empty endpoint")
		}

		if len(spec.Endpoint) > 150 {
			return grpcutils.InvalidArg("Endpoint is too long")
		}

		switch {
		case govalidator.IsDNSName(spec.Endpoint) || govalidator.IsHost(spec.Endpoint) || govalidator.IsURL(spec.Endpoint):
		default:
			return grpcutils.InvalidArg("Invalid endpoint: %s", spec.Endpoint)
		}

		if spec.GetAuth() == nil {
			return grpcutils.InvalidArg("Auth must be set")
		}

		switch spec.Auth.Type.(type) {
		case *enterprisev1.CollectorExporter_Spec_OTLP_Auth_Basic_:
			if spec.Auth.GetBasic().Username == "" {
				return grpcutils.InvalidArg("Empty basic auth username")
			}
			if len(spec.Auth.GetBasic().Username) > 64 {
				return grpcutils.InvalidArg("Username is too long")
			}
			if err := s.validateSecretOwner(ctx, spec.Auth.GetBasic().GetPassword()); err != nil {
				return err
			}
		case *enterprisev1.CollectorExporter_Spec_OTLP_Auth_Bearer_:
			if err := s.validateSecretOwner(ctx, spec.Auth.GetBearer()); err != nil {
				return err
			}
		case *enterprisev1.CollectorExporter_Spec_OTLP_Auth_Custom_:
			if spec.Auth.GetCustom().Header == "" {
				return grpcutils.InvalidArg("Empty header")
			}
			if !govalidator.IsASCII(spec.Auth.GetCustom().Header) {
				return grpcutils.InvalidArg("Invalid header")
			}
			if len(spec.Auth.GetCustom().Header) > 128 {
				return grpcutils.InvalidArg("Header is too long")
			}
			if err := s.validateSecretOwner(ctx, spec.Auth.GetCustom().GetValue()); err != nil {
				return err
			}
		default:
			return grpcutils.InvalidArg("Auth must be set")
		}

	case *enterprisev1.CollectorExporter_Spec_OtlpHTTP:
		spec := spec.GetOtlpHTTP()
		if spec.Endpoint == "" {
			return grpcutils.InvalidArg("Empty endpoint")
		}

		if len(spec.Endpoint) > 150 {
			return grpcutils.InvalidArg("Endpoint is too long")
		}

		switch {
		case govalidator.IsDNSName(spec.Endpoint) || govalidator.IsHost(spec.Endpoint) || govalidator.IsURL(spec.Endpoint):
		default:
			return grpcutils.InvalidArg("Invalid endpoint: %s", spec.Endpoint)
		}

		if spec.GetAuth() == nil {
			return grpcutils.InvalidArg("Auth must be set")
		}

		switch spec.Auth.Type.(type) {
		case *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Basic_:
			if spec.Auth.GetBasic().Username == "" {
				return grpcutils.InvalidArg("Empty basic auth username")
			}
			if len(spec.Auth.GetBasic().Username) > 64 {
				return grpcutils.InvalidArg("Username is too long")
			}
			if err := s.validateSecretOwner(ctx, spec.Auth.GetBasic().GetPassword()); err != nil {
				return err
			}
		case *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer_:
			if err := s.validateSecretOwner(ctx, spec.Auth.GetBearer()); err != nil {
				return err
			}
		case *enterprisev1.CollectorExporter_Spec_OTLPHTTP_Auth_Custom_:
			if spec.Auth.GetCustom().Header == "" {
				return grpcutils.InvalidArg("Empty header")
			}
			if !govalidator.IsASCII(spec.Auth.GetCustom().Header) {
				return grpcutils.InvalidArg("Invalid header")
			}
			if len(spec.Auth.GetCustom().Header) > 128 {
				return grpcutils.InvalidArg("Header is too long")
			}
			if err := s.validateSecretOwner(ctx, spec.Auth.GetCustom().GetValue()); err != nil {
				return err
			}
		default:
			return grpcutils.InvalidArg("Auth must be set")
		}

	case *enterprisev1.CollectorExporter_Spec_PrometheusRemoteWrite_:
	case *enterprisev1.CollectorExporter_Spec_Splunk_:
	default:
		return grpcutils.InvalidArg("Invalid CollectorExporter type")
	}

	return nil
}
