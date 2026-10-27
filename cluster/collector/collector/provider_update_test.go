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
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestProviderRetrieveSuccessAndWatcherLifetime(t *testing.T) {
	p := newCollectorProviderTest(t)
	p.updateDebounce = 5 * time.Millisecond
	instance := newProviderInstance(p)
	t.Cleanup(func() { instance.Shutdown(context.Background()) })
	assert.Equal(t, "octelium-api", instance.Scheme())
	var calls atomic.Int32
	retrieved, err := instance.Retrieve(t.Context(), "octelium-api:default", func(*confmap.ChangeEvent) { calls.Add(1) })
	require.NoError(t, err)
	configuration, err := retrieved.AsConf()
	require.NoError(t, err)
	assert.NotEmpty(t, configuration.ToStringMap())
	p.sendUpdate()
	require.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)
	require.NoError(t, retrieved.Close(t.Context()))
	require.NoError(t, retrieved.Close(t.Context()))
	p.mu.RLock()
	assert.Empty(t, p.watchers)
	assert.False(t, p.updateLoopRunning)
	p.mu.RUnlock()
}

func TestProviderRetrieveWithoutWatcherAndAfterShutdown(t *testing.T) {
	p := newCollectorProviderTest(t)
	instance := p.newFactory().Create(confmap.ProviderSettings{}).(*providerInstance)
	_, err := instance.Retrieve(t.Context(), "yaml:unsupported", nil)
	require.Error(t, err)
	retrieved, err := instance.Retrieve(t.Context(), "octelium-api:default", nil)
	require.NoError(t, err)
	require.NoError(t, retrieved.Close(t.Context()))
	assert.Empty(t, p.watchers)
	require.NoError(t, instance.Shutdown(t.Context()))
	require.NoError(t, instance.Shutdown(t.Context()))
	_, err = instance.Retrieve(t.Context(), "octelium-api:default", nil)
	assert.ErrorContains(t, err, "shut down")
}

func TestProviderRetrieveFailureRemovesItsWatcher(t *testing.T) {
	p := newCollectorProviderTest(t)
	failure := errors.New("config unavailable")
	p.octeliumC = &collectorErrorClient{ClientInterface: p.octeliumC, resources: &collectorErrorResources{err: failure}}
	instance := newProviderInstance(p)
	t.Cleanup(func() { instance.Shutdown(context.Background()) })
	_, err := instance.Retrieve(t.Context(), "octelium-api:default", func(*confmap.ChangeEvent) { t.Error("failed retrieval notified a watcher") })
	assert.ErrorIs(t, err, failure)
	assert.Empty(t, instance.watcherIDs)
	p.mu.RLock()
	assert.Empty(t, p.watchers)
	assert.False(t, p.updateLoopRunning)
	p.mu.RUnlock()
}

func TestProviderWatcherRecoversPanicsAndRejectsNotificationsAfterClose(t *testing.T) {
	panicking := newProviderWatcher(func(*confmap.ChangeEvent) { panic("watcher failure") })
	require.NotPanics(t, func() { panicking.notify(&confmap.ChangeEvent{}) })
	require.NoError(t, panicking.close(t.Context()))
	assert.Zero(t, panicking.active)
	var calls atomic.Int32
	watcher := newProviderWatcher(func(*confmap.ChangeEvent) { calls.Add(1) })
	watcher.notify(&confmap.ChangeEvent{})
	require.NoError(t, watcher.close(t.Context()))
	watcher.notify(&confmap.ChangeEvent{})
	assert.EqualValues(t, 1, calls.Load())
	var absent *providerWatcher
	absent.notify(nil)
	require.NoError(t, absent.close(t.Context()))
}

func TestProviderWatcherCloseWaitsForActiveCallback(t *testing.T) {
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	finish := sync.OnceFunc(func() { close(release) })
	defer finish()
	watcher := newProviderWatcher(func(*confmap.ChangeEvent) { close(started); <-release })
	go func() { defer close(done); watcher.notify(&confmap.ChangeEvent{}) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("watcher did not start")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	assert.ErrorIs(t, watcher.close(ctx), context.DeadlineExceeded)
	finish()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not finish")
	}
	require.NoError(t, watcher.close(t.Context()))
	assert.Zero(t, watcher.active)
}

func TestProviderUpdateGenerationIgnoresOldLoops(t *testing.T) {
	p := &provider{}
	assert.Zero(t, p.registerWatcher(nil))
	p.sendUpdate()
	var calls atomic.Int32
	firstID := p.registerWatcher(func(*confmap.ChangeEvent) { calls.Add(1) })
	oldGeneration := p.updateLoopGeneration
	require.NoError(t, p.unregisterWatcher(t.Context(), firstID))
	secondID := p.registerWatcher(func(*confmap.ChangeEvent) { calls.Add(1) })
	t.Cleanup(func() { p.unregisterWatcher(context.Background(), secondID) })
	p.notifyWatchers(oldGeneration)
	assert.Zero(t, calls.Load())
	p.notifyWatchers(p.updateLoopGeneration)
	assert.EqualValues(t, 1, calls.Load())
	require.NoError(t, p.unregisterWatcher(t.Context(), 0))
	require.NoError(t, p.unregisterWatcher(t.Context(), firstID))
}

func TestCollectorControllerOnlyNotifiesForCollectorChanges(t *testing.T) {
	p := &provider{updateDebounce: 5 * time.Millisecond}
	var calls atomic.Int32
	id := p.registerWatcher(func(*confmap.ChangeEvent) { calls.Add(1) })
	t.Cleanup(func() { p.unregisterWatcher(context.Background(), id) })
	ctl := &ccController{p: p}
	old := &enterprisev1.ClusterConfig{Spec: &enterprisev1.ClusterConfig_Spec{}}
	unchanged := pbutils.Clone(old).(*enterprisev1.ClusterConfig)
	unchanged.Metadata = &metav1.Metadata{Description: "metadata-only"}
	require.NoError(t, ctl.OnUpdate(t.Context(), unchanged, old))
	p.mu.RLock()
	assert.Empty(t, p.updateCh)
	p.mu.RUnlock()
	changed := pbutils.Clone(old).(*enterprisev1.ClusterConfig)
	changed.Spec.Collector = &enterprisev1.ClusterConfig_Spec_Collector{AdditionalInlineConfig: "processors: {}"}
	require.NoError(t, ctl.OnUpdate(t.Context(), changed, old))
	require.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)
}
