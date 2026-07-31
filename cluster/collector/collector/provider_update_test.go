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
	"sync/atomic"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/confmap"
)

func TestProviderSendUpdateDebouncesBursts(t *testing.T) {
	p := &provider{
		updateDebounce: 25 * time.Millisecond,
	}
	instance := newProviderInstance(p)

	var calls atomic.Int64
	watcherID := p.registerWatcher(func(*confmap.ChangeEvent) {
		calls.Add(1)
	})
	if !assert.NotZero(t, watcherID) {
		return
	}
	instance.watcherIDs[watcherID] = struct{}{}

	for idx := 0; idx < 100; idx++ {
		p.sendUpdate()
	}

	assert.Eventually(t, func() bool {
		return calls.Load() == 1
	}, time.Second, 10*time.Millisecond)

	time.Sleep(75 * time.Millisecond)
	assert.Equal(t, int64(1), calls.Load())

	assert.NoError(t, instance.Shutdown(context.Background()))
}

func TestProviderRetrievedCloseRemovesOnlyItsWatcher(t *testing.T) {
	p := &provider{
		updateDebounce: 10 * time.Millisecond,
	}
	first := newProviderInstance(p)
	second := newProviderInstance(p)

	var firstCalls atomic.Int64
	var secondCalls atomic.Int64

	firstID := p.registerWatcher(func(*confmap.ChangeEvent) {
		firstCalls.Add(1)
	})
	first.watcherIDs[firstID] = struct{}{}

	secondID := p.registerWatcher(func(*confmap.ChangeEvent) {
		secondCalls.Add(1)
	})
	second.watcherIDs[secondID] = struct{}{}

	assert.NoError(t, first.closeWatcher(context.Background(), firstID))

	p.sendUpdate()

	assert.Eventually(t, func() bool {
		return secondCalls.Load() == 1
	}, time.Second, 10*time.Millisecond)

	assert.Zero(t, firstCalls.Load())
	assert.NoError(t, second.Shutdown(context.Background()))
}

func TestProviderShutdownStopsPendingNotifications(t *testing.T) {
	p := &provider{
		updateDebounce: 100 * time.Millisecond,
	}
	instance := newProviderInstance(p)

	var calls atomic.Int64
	watcherID := p.registerWatcher(func(*confmap.ChangeEvent) {
		calls.Add(1)
	})
	instance.watcherIDs[watcherID] = struct{}{}

	p.sendUpdate()
	assert.NoError(t, instance.Shutdown(context.Background()))

	time.Sleep(150 * time.Millisecond)
	assert.Zero(t, calls.Load())
}

func TestProviderFactoryCreatesFreshInstances(t *testing.T) {
	p := &provider{
		schemeName: "octelium-api",
	}

	first := newProviderInstance(p)
	second := newProviderInstance(p)

	assert.NotSame(t, first, second)
	assert.Same(t, p, first.p)
	assert.Same(t, p, second.p)

	assert.NoError(t, first.Shutdown(context.Background()))
	assert.NoError(t, second.Shutdown(context.Background()))
}

func TestInternalExporterConfigIsBounded(t *testing.T) {
	p := &provider{}

	cfg := p.getInternalExporterConfig("metricstore.example:8080", 8)

	assert.Equal(t, "metricstore.example:8080", cfg["endpoint"])
	assert.Equal(t, "10s", cfg["timeout"])
	assert.Equal(t, 8, cfg["max_call_send_msg_size_mib"])

	queue, ok := cfg["sending_queue"].(map[string]any)
	if !assert.True(t, ok) {
		return
	}
	assert.Equal(t, true, queue["enabled"])
	assert.Equal(t, 1, queue["num_consumers"])
	assert.Equal(t, false, queue["block_on_overflow"])
	assert.Equal(t, "bytes", queue["sizer"])
	assert.Equal(t, 32<<20, queue["queue_size"])

	batch, ok := queue["batch"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, "bytes", batch["sizer"])
		assert.Equal(t, 4<<20, batch["max_size"])
	}

	retry, ok := cfg["retry_on_failure"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, true, retry["enabled"])
		assert.Equal(t, "2m", retry["max_elapsed_time"])
	}
}

func TestKafkaExporterSignalCapabilitiesUseUpstreamDefaults(t *testing.T) {
	hasLogs, hasMetrics := kafkaExporterSignalCapabilities(
		&enterprisev1.CollectorExporter_Spec_Kafka{},
	)

	assert.True(t, hasLogs)
	assert.True(t, hasMetrics)
}
