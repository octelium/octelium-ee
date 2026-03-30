// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package collector

import (
	"context"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/watchers"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/cluster/common/healthcheck"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awss3exporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuredataexplorerexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuremonitorexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/influxdbexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logzioexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/splunkhecexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sumologicexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/bearertokenauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/intervalprocessor"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
	"go.uber.org/zap"
)

type Server struct {
	octeliumC    octeliumc.ClientInterface
	collector    *otelcol.Collector
	p            *provider
	ccController *ccController
	receiverPort int
}

func NewServer(ctx context.Context, octeliumC octeliumc.ClientInterface) (*Server, error) {

	ret := &Server{
		octeliumC:    octeliumC,
		receiverPort: 8080,
	}

	return ret, nil
}

func (s *Server) doRun(ctx context.Context) error {
	var err error

	factories := otelcol.Factories{}

	factories.Receivers, err = otelcol.MakeFactoryMap[receiver.Factory](
		otlpreceiver.NewFactory(),
	)
	if err != nil {
		return err
	}

	factories.Processors, err = otelcol.MakeFactoryMap[processor.Factory](
		memorylimiterprocessor.NewFactory(),
		batchprocessor.NewFactory(),
		attributesprocessor.NewFactory(),
		intervalprocessor.NewFactory(),
	)
	if err != nil {
		return err
	}

	factories.Extensions, err = otelcol.MakeFactoryMap(
		basicauthextension.NewFactory(),
		bearertokenauthextension.NewFactory(),
	)
	if err != nil {
		return err
	}

	factories.Exporters, err = otelcol.MakeFactoryMap[exporter.Factory](
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),

		elasticsearchexporter.NewFactory(),
		// datadogexporter.NewFactory(),
		prometheusremotewriteexporter.NewFactory(),
		clickhouseexporter.NewFactory(),
		// lokiexporter.NewFactory(),
		splunkhecexporter.NewFactory(),
		influxdbexporter.NewFactory(),
		kafkaexporter.NewFactory(),
		sumologicexporter.NewFactory(),
		azuremonitorexporter.NewFactory(),
		awss3exporter.NewFactory(),
		logzioexporter.NewFactory(),
		azuredataexplorerexporter.NewFactory(),
	)
	if err != nil {
		return err
	}

	factories.Telemetry = otelconftelemetry.NewFactory()

	s.p = &provider{
		octeliumC:  s.octeliumC,
		schemeName: "octelium-api",
		port:       s.receiverPort,
	}

	s.ccController = &ccController{
		p: s.p,
	}

	zap.L().Debug("Starting a new collector")

	s.collector, err = otelcol.NewCollector(otelcol.CollectorSettings{
		Factories: func() (otelcol.Factories, error) {
			return factories, nil
		},
		ConfigProviderSettings: s.getConfigProviderSettings(),
	})
	if err != nil {
		return err
	}

	go func(ctx context.Context) {

		for {
			select {
			case <-ctx.Done():
				return
			default:
				zap.L().Debug("Starting running the collector")
				err := s.collector.Run(ctx)
				if err == nil {
					return
				}

				zap.S().Errorf("Could not run collector. Trying again...: %+v", err)

				time.Sleep(1 * time.Second)
				s.collector, err = otelcol.NewCollector(otelcol.CollectorSettings{
					Factories: func() (otelcol.Factories, error) {
						return factories, nil
					},
					ConfigProviderSettings: s.getConfigProviderSettings(),
				})
				if err != nil {
					zap.L().Debug("Could not create a new collector. Exiting the loop.", zap.Error(err))
					return
				}
			}
		}

	}(ctx)

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	return s.doRun(ctx)
}

type ccController struct {
	p *provider
}

func (c *ccController) OnUpdate(ctx context.Context, new, old *enterprisev1.ClusterConfig) error {

	if !pbutils.IsEqual(new.Spec.Collector, old.Spec.Collector) {
		zap.S().Debugf("Collector spec changed. Instructing a config change...")
		c.p.sendUpdate()
	} else {
		zap.S().Debugf("Cluster config changed but not Collector spec. Nothing to be done...")
	}

	return nil
}

func (s *Server) getConfigProviderSettings() otelcol.ConfigProviderSettings {
	return otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{"octelium-api:default"},
			ProviderFactories: []confmap.ProviderFactory{
				yamlprovider.NewFactory(),
				s.p.newFactory(),
			},
			DefaultScheme: "octelium-api",
			ProviderSettings: confmap.ProviderSettings{
				Logger: zap.L(),
			},
		},
	}
}

func Run(ctx context.Context) error {

	octeliumC, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	srv, err := NewServer(ctx, octeliumC)
	if err != nil {
		return err
	}

	if err := srv.Run(ctx); err != nil {
		return err
	}

	watcher := watchers.NewEnterpriseV1(srv.octeliumC)
	if err := watcher.ClusterConfig(ctx, nil, srv.ccController.OnUpdate); err != nil {
		return err
	}

	if err := watcher.CollectorExporter(ctx, nil,
		func(ctx context.Context, item *enterprisev1.CollectorExporter) error {
			srv.p.sendUpdate()
			return nil
		}, func(ctx context.Context, new, old *enterprisev1.CollectorExporter) error {
			srv.p.sendUpdate()
			return nil
		}, func(ctx context.Context, item *enterprisev1.CollectorExporter) error {
			srv.p.sendUpdate()
			return nil
		}); err != nil {
		return err
	}

	healthcheck.Run(vutils.HealthCheckPortMain)
	zap.L().Debug("Collector is now running...")

	<-ctx.Done()

	return nil
}
