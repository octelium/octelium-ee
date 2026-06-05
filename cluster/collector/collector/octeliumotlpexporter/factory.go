// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package octeliumotlpexporter

import (
	"context"
	"net"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

var componentType = component.MustNewType("octelium_otlp")

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		componentType,
		createDefaultConfig,
		exporter.WithLogs(createLogs, component.StabilityLevelStable),
		exporter.WithMetrics(createMetrics, component.StabilityLevelStable),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		TimeoutConfig:         exporterhelper.NewDefaultTimeoutConfig(),
		RetryConfig:           configretry.NewDefaultBackOffConfig(),
		QueueConfig:           configoptional.Some(exporterhelper.NewDefaultQueueConfig()),
		Compression:           string(configcompression.TypeGzip),
		MaxCallRecvMsgSizeMiB: 16,
		MaxCallSendMsgSizeMiB: 16,
	}
}

func endpointAttributes(cfg *Config) []attribute.KeyValue {
	endpoint := cfg.sanitizedEndpoint()

	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return []attribute.KeyValue{semconv.ServerAddress(endpoint)}
	}

	out := []attribute.KeyValue{
		semconv.ServerAddress(host),
	}

	if portNumber, err := strconv.Atoi(port); err == nil {
		out = append(out, semconv.ServerPort(portNumber))
	}

	return out
}

func createLogs(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	oCfg := cfg.(*Config)
	oce := newExporter(oCfg, set)

	return exporterhelper.NewLogs(
		ctx,
		set,
		cfg,
		oce.pushLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
		exporterhelper.WithAttrs(endpointAttributes(oCfg)...),
	)
}

func createMetrics(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Metrics, error) {
	oCfg := cfg.(*Config)
	oce := newExporter(oCfg, set)

	return exporterhelper.NewMetrics(
		ctx,
		set,
		cfg,
		oce.pushMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(oCfg.TimeoutConfig),
		exporterhelper.WithRetry(oCfg.RetryConfig),
		exporterhelper.WithQueue(oCfg.QueueConfig),
		exporterhelper.WithStart(oce.start),
		exporterhelper.WithShutdown(oce.shutdown),
		exporterhelper.WithAttrs(endpointAttributes(oCfg)...),
	)
}