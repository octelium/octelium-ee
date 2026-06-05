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
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

type signalReceiver struct {
	settings receiver.Settings
	cfg      *Config

	server *sharedOTLPServer

	nextLogs    consumer.Logs
	nextMetrics consumer.Metrics
}

var (
	_ receiver.Logs    = (*signalReceiver)(nil)
	_ receiver.Metrics = (*signalReceiver)(nil)
)

func newSignalReceiver(
	settings receiver.Settings,
	cfg component.Config,
	nextLogs consumer.Logs,
	nextMetrics consumer.Metrics,
) (*signalReceiver, error) {
	typedCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("invalid octelium_otlp receiver config type")
	}
	if err := typedCfg.Validate(); err != nil {
		return nil, err
	}

	srv := getOrCreateSharedServer(settings, typedCfg)

	r := &signalReceiver{
		settings:    settings,
		cfg:         typedCfg,
		server:      srv,
		nextLogs:    nextLogs,
		nextMetrics: nextMetrics,
	}

	return r, nil
}

func (r *signalReceiver) Start(ctx context.Context, host component.Host) error {
	if r.nextLogs != nil {
		r.server.setLogsConsumer(r.nextLogs)
	}
	if r.nextMetrics != nil {
		r.server.setMetricsConsumer(r.nextMetrics)
	}

	return r.server.Start(ctx, host)
}

func (r *signalReceiver) Shutdown(ctx context.Context) error {
	return r.server.Release(ctx)
}

func (r *signalReceiver) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}