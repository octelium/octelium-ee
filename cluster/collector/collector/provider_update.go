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
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
)

const defaultProviderUpdateDebounce = 350 * time.Millisecond

type providerInstance struct {
	p *provider

	mu         sync.Mutex
	watcherIDs map[uint64]struct{}
	shutdown   bool
}

func newProviderInstance(p *provider) *providerInstance {
	return &providerInstance{
		p:          p,
		watcherIDs: make(map[uint64]struct{}),
	}
}

func (p *providerInstance) Retrieve(
	ctx context.Context,
	uri string,
	waitFn confmap.WatcherFunc,
) (*confmap.Retrieved, error) {
	p.mu.Lock()
	if p.shutdown {
		p.mu.Unlock()
		return nil, fmt.Errorf("collector config provider is shut down")
	}
	p.mu.Unlock()

	if !strings.HasPrefix(uri, p.p.schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, p.p.schemeName)
	}

	watcherID := p.p.registerWatcher(waitFn)
	if watcherID != 0 {
		p.mu.Lock()
		if p.shutdown {
			p.mu.Unlock()
			_ = p.p.unregisterWatcher(context.Background(), watcherID)
			return nil, fmt.Errorf("collector config provider is shut down")
		}
		p.watcherIDs[watcherID] = struct{}{}
		p.mu.Unlock()
	}

	zap.L().Debug("Retrieving provider config")

	cfg, err := p.p.getConfig(ctx)
	if err != nil {
		if watcherID != 0 {
			_ = p.closeWatcher(context.Background(), watcherID)
		}
		return nil, err
	}

	if watcherID == 0 {
		return confmap.NewRetrieved(cfg)
	}

	return confmap.NewRetrieved(
		cfg,
		confmap.WithRetrievedClose(func(closeCtx context.Context) error {
			return p.closeWatcher(closeCtx, watcherID)
		}),
	)
}

func (p *providerInstance) Scheme() string {
	return p.p.schemeName
}

func (p *providerInstance) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	if p.shutdown {
		p.mu.Unlock()
		return nil
	}

	p.shutdown = true

	watcherIDs := make([]uint64, 0, len(p.watcherIDs))
	for watcherID := range p.watcherIDs {
		watcherIDs = append(watcherIDs, watcherID)
	}
	p.watcherIDs = nil
	p.mu.Unlock()

	for _, watcherID := range watcherIDs {
		if err := p.p.unregisterWatcher(ctx, watcherID); err != nil {
			return err
		}
	}

	return nil
}

func (p *providerInstance) closeWatcher(ctx context.Context, watcherID uint64) error {
	p.mu.Lock()
	delete(p.watcherIDs, watcherID)
	p.mu.Unlock()

	return p.p.unregisterWatcher(ctx, watcherID)
}

type providerWatcher struct {
	mu sync.Mutex

	fn     confmap.WatcherFunc
	closed bool
	active int

	done     chan struct{}
	doneOnce sync.Once
}

func newProviderWatcher(fn confmap.WatcherFunc) *providerWatcher {
	return &providerWatcher{
		fn:   fn,
		done: make(chan struct{}),
	}
}

func (w *providerWatcher) notify(event *confmap.ChangeEvent) {
	if w == nil {
		return
	}

	w.mu.Lock()
	if w.closed || w.fn == nil {
		w.mu.Unlock()
		return
	}

	fn := w.fn
	w.active++
	w.mu.Unlock()

	defer func() {
		if recovered := recover(); recovered != nil {
			zap.L().Error("Collector config watcher panicked",
				zap.Any("panic", recovered),
			)
		}

		w.mu.Lock()
		w.active--
		if w.closed && w.active == 0 {
			w.doneOnce.Do(func() {
				close(w.done)
			})
		}
		w.mu.Unlock()
	}()

	fn(event)
}

func (w *providerWatcher) close(ctx context.Context) error {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	if !w.closed {
		w.closed = true
		w.fn = nil
		if w.active == 0 {
			w.doneOnce.Do(func() {
				close(w.done)
			})
		}
	}
	done := w.done
	w.mu.Unlock()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *provider) registerWatcher(fn confmap.WatcherFunc) uint64 {
	if fn == nil {
		return 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.ensureUpdateLoopLocked()

	p.nextWatcherID++
	watcherID := p.nextWatcherID

	if p.watchers == nil {
		p.watchers = make(map[uint64]*providerWatcher)
	}
	p.watchers[watcherID] = newProviderWatcher(fn)

	return watcherID
}

func (p *provider) unregisterWatcher(ctx context.Context, watcherID uint64) error {
	if watcherID == 0 {
		return nil
	}

	p.mu.Lock()
	watcher := p.watchers[watcherID]
	delete(p.watchers, watcherID)

	var stopCh chan struct{}
	var doneCh chan struct{}

	if len(p.watchers) == 0 && p.updateLoopRunning {
		stopCh = p.updateStopCh
		doneCh = p.updateDoneCh

		p.updateCh = nil
		p.updateStopCh = nil
		p.updateDoneCh = nil
		p.updateLoopRunning = false
	}
	p.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}

	if watcher != nil {
		if err := watcher.close(ctx); err != nil {
			return err
		}
	}

	if doneCh == nil {
		return nil
	}

	select {
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *provider) ensureUpdateLoopLocked() {
	if p.updateLoopRunning {
		return
	}

	debounce := p.updateDebounce
	if debounce <= 0 {
		debounce = defaultProviderUpdateDebounce
	}

	p.updateLoopGeneration++
	generation := p.updateLoopGeneration

	updateCh := make(chan struct{}, 1)
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	p.updateCh = updateCh
	p.updateStopCh = stopCh
	p.updateDoneCh = doneCh
	p.updateLoopRunning = true

	go p.runUpdateLoop(generation, debounce, updateCh, stopCh, doneCh)
}

func (p *provider) runUpdateLoop(
	generation uint64,
	debounce time.Duration,
	updateCh <-chan struct{},
	stopCh <-chan struct{},
	doneCh chan<- struct{},
) {
	defer close(doneCh)

	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		<-timer.C
	}

	var timerCh <-chan time.Time

	for {
		select {
		case <-updateCh:
			if timerCh != nil && !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			timer.Reset(debounce)
			timerCh = timer.C

		case <-timerCh:
			timerCh = nil
			p.notifyWatchers(generation)

		case <-stopCh:
			if timerCh != nil && !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

func (p *provider) notifyWatchers(generation uint64) {
	p.mu.RLock()
	if generation != p.updateLoopGeneration {
		p.mu.RUnlock()
		return
	}

	watchers := make([]*providerWatcher, 0, len(p.watchers))
	for _, watcher := range p.watchers {
		watchers = append(watchers, watcher)
	}
	p.mu.RUnlock()

	if len(watchers) == 0 {
		return
	}

	zap.L().Debug("Config provider sending a debounced update",
		zap.Int("watchers", len(watchers)),
	)

	event := &confmap.ChangeEvent{}
	for _, watcher := range watchers {
		watcher.notify(event)
	}
}

func (p *provider) sendUpdate() {
	p.mu.RLock()
	updateCh := p.updateCh
	running := p.updateLoopRunning
	p.mu.RUnlock()

	if !running || updateCh == nil {
		return
	}

	select {
	case updateCh <- struct{}{}:
	default:
	}
}
