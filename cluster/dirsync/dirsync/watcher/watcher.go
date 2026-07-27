// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package watcher

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
)

const (
	pollInterval       = 30 * time.Second
	minPollingInterval = 5 * time.Minute
	maxBackoff         = 30 * time.Minute
	syncStaleAfter     = 30 * time.Minute
	pollListPageSize   = 200
)

type Watcher struct {
	octeliumC octeliumc.ClientInterface

	mu       sync.Mutex
	schedule map[string]*scheduleEntry
}

type scheduleEntry struct {
	nextRun  time.Time
	failures int
}

func New(octeliumC octeliumc.ClientInterface) *Watcher {
	return &Watcher{
		octeliumC: octeliumC,
		schedule:  map[string]*scheduleEntry{},
	}
}

func (s *Watcher) Run(ctx context.Context) {
	go func(ctx context.Context) {
		if err := s.doRun(ctx); err != nil {
			zap.L().Warn("Could not run watcher doRun", zap.Error(err))
		}
	}(ctx)
}

func (s *Watcher) doRun(ctx context.Context) error {
	t := time.NewTicker(pollInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := s.poll(ctx); err != nil {
				zap.L().Warn("Could not poll directoryProviders for polling", zap.Error(err))
			}
		}
	}
}

func (s *Watcher) poll(ctx context.Context) error {
	dps, err := s.listDirectoryProviders(ctx)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(dps))
	now := time.Now()

	for _, dp := range dps {
		seen[dp.Metadata.Uid] = struct{}{}
		s.reconcileOne(ctx, dp, now)
	}

	s.mu.Lock()
	for uid := range s.schedule {
		if _, ok := seen[uid]; !ok {
			delete(s.schedule, uid)
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *Watcher) reconcileOne(ctx context.Context, dp *enterprisev1.DirectoryProvider, now time.Time) {
	interval := pollingInterval(dp)
	if interval <= 0 || dp.Spec.GetIsDisabled() {
		s.forget(dp.Metadata.Uid)
		return
	}
	if interval < minPollingInterval {
		interval = minPollingInterval
	}

	switch syncState(dp) {
	case enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED,
		enterprisev1.DirectoryProvider_Status_Synchronization_SYNCING:
		started := dp.Status.Synchronization.GetCreatedAt()
		if started == nil || now.Sub(started.AsTime()) < syncStaleAfter {
			return
		}
	}

	s.mu.Lock()
	ent := s.schedule[dp.Metadata.Uid]
	if ent == nil {
		s.schedule[dp.Metadata.Uid] = &scheduleEntry{nextRun: now.Add(spread(interval))}
		s.mu.Unlock()
		return
	}
	if now.Before(ent.nextRun) {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	if err := s.requestSync(ctx, dp); err != nil {
		zap.L().Warn("Could not request directoryProvider sync",
			zap.String("directoryProvider", dp.Metadata.Name), zap.Error(err))
		s.mu.Lock()
		ent.failures++
		ent.nextRun = now.Add(backoff(interval, ent.failures))
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	ent.failures = 0
	ent.nextRun = now.Add(interval + jitter(interval))
	s.mu.Unlock()
}

func (s *Watcher) requestSync(ctx context.Context, dp *enterprisev1.DirectoryProvider) error {
	cur, err := s.octeliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{Uid: dp.Metadata.Uid})
	if err != nil {
		return err
	}

	switch syncState(cur) {
	case enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED,
		enterprisev1.DirectoryProvider_Status_Synchronization_SYNCING:
		return nil
	}

	if cur.Status == nil {
		cur.Status = &enterprisev1.DirectoryProvider_Status{}
	}
	cur.Status.Synchronization = &enterprisev1.DirectoryProvider_Status_Synchronization{
		State:     enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED,
		CreatedAt: pbutils.Now(),
	}

	_, err = s.octeliumC.EnterpriseC().UpdateDirectoryProvider(ctx, cur)
	return err
}

func (s *Watcher) listDirectoryProviders(ctx context.Context) ([]*enterprisev1.DirectoryProvider, error) {
	var out []*enterprisev1.DirectoryProvider
	var page uint32
	for {
		l, err := s.octeliumC.EnterpriseC().ListDirectoryProvider(ctx, &rmetav1.ListOptions{
			Paginate:     true,
			Page:         page,
			ItemsPerPage: pollListPageSize,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, l.Items...)
		if len(l.Items) < pollListPageSize {
			break
		}
		page++
	}
	return out, nil
}

func (s *Watcher) forget(uid string) {
	s.mu.Lock()
	delete(s.schedule, uid)
	s.mu.Unlock()
}

func pollingInterval(dp *enterprisev1.DirectoryProvider) time.Duration {
	if dp.Spec == nil {
		return 0
	}

	var d *metav1.Duration
	switch t := dp.Spec.Type.(type) {
	case *enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_:
		d = t.GoogleWorkspace.GetPolling().GetInterval()
	case *enterprisev1.DirectoryProvider_Spec_Keycloak_:
		d = t.Keycloak.GetPolling().GetInterval()
	}
	if d == nil {
		return 0
	}
	return umetav1.ToDuration(d).ToGo()
}

func syncState(dp *enterprisev1.DirectoryProvider) enterprisev1.DirectoryProvider_Status_Synchronization_State {
	if dp.Status == nil || dp.Status.Synchronization == nil {
		return enterprisev1.DirectoryProvider_Status_Synchronization_STATE_UNSET
	}
	return dp.Status.Synchronization.State
}

func spread(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(d)))
}

func jitter(d time.Duration) time.Duration {
	j := d / 10
	if j <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(j)))
}

func backoff(base time.Duration, failures int) time.Duration {
	d := base
	for i := 1; i < failures; i++ {
		d *= 2
		if d <= 0 || d >= maxBackoff {
			return maxBackoff
		}
	}
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}
