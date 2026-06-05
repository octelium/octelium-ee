// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package octeliumotlpreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var componentType = component.MustNewType("octelium_otlp")

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		componentType,
		createDefaultConfig,
		receiver.WithLogs(createLogs, component.StabilityLevelStable),
		receiver.WithMetrics(createMetrics, component.StabilityLevelStable),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Endpoint:             ":8080",
		MaxRecvMsgSizeMiB:    16,
		MaxConcurrentStreams: 1024,
		ReadBufferSize:       512 * 1024,
		WriteBufferSize:      0,
	}
}

func createLogs(
	_ context.Context,
	settings receiver.Settings,
	cfg component.Config,
	next consumer.Logs,
) (receiver.Logs, error) {
	r, err := loadOrCreateReceiver(settings, cfg)
	if err != nil {
		return nil, err
	}

	r.registerLogsConsumer(next)

	return r, nil
}

func createMetrics(
	_ context.Context,
	settings receiver.Settings,
	cfg component.Config,
	next consumer.Metrics,
) (receiver.Metrics, error) {
	r, err := loadOrCreateReceiver(settings, cfg)
	if err != nil {
		return nil, err
	}

	r.registerMetricsConsumer(next)

	return r, nil
}
