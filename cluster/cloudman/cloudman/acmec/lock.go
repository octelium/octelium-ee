// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package acmec

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rlockv1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	distributedLockPrefix  = "cloudman/acme/"
	distributedLockTTL     = 3 * time.Minute
	distributedLockRefresh = 45 * time.Second

	distributedLockRefreshRetries = 3
	distributedLockWait           = 5 * time.Second
	distributedLockTimeout        = 20 * time.Second
)

type keyedLocker struct {
	mu    sync.Mutex
	locks map[string]*keyedLock
}

type keyedLock struct {
	ch   chan struct{}
	refs uint32
}

func newKeyedLocker() *keyedLocker {
	return &keyedLocker{
		locks: make(map[string]*keyedLock),
	}
}

func (l *keyedLocker) Lock(ctx context.Context, key string) (func(), error) {
	l.mu.Lock()
	entry := l.locks[key]
	if entry == nil {
		entry = &keyedLock{
			ch: make(chan struct{}, 1),
		}
		l.locks[key] = entry
	}
	entry.refs = entry.refs + 1
	l.mu.Unlock()

	select {
	case <-ctx.Done():
		l.releaseRef(key)
		return nil, ctx.Err()
	case entry.ch <- struct{}{}:
	}

	var once sync.Once

	return func() {
		once.Do(func() {
			<-entry.ch
			l.releaseRef(key)
		})
	}, nil
}

func (l *keyedLocker) releaseRef(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.locks[key]
	if entry == nil {
		return
	}

	entry.refs = entry.refs - 1
	if entry.refs == 0 {
		delete(l.locks, key)
	}
}

func (c *Controller) acquireLock(ctx context.Context, key string) (
	context.Context, func(), error) {

	localUnlock, err := c.locks.Lock(ctx, key)
	if err != nil {
		return nil, nil, err
	}

	lockCtx, unlock, err := acquireDistributedLock(ctx, c.octeliumC.LockC(), key)
	if err != nil {
		localUnlock()
		return nil, nil, err
	}

	return lockCtx, func() {
		unlock()
		localUnlock()
	}, nil
}

func acquireDistributedLock(ctx context.Context,
	lockC rlockv1.MainServiceClient, key string) (context.Context, func(), error) {

	if lockC == nil {
		return nil, nil, errors.Errorf("Nil distributed lock client")
	}

	lockKey := []byte(distributedLockPrefix + key)

	rpcCtx, cancelRPC := context.WithTimeout(ctx, distributedLockTimeout)
	resp, err := lockC.Lock(rpcCtx, &rlockv1.LockRequest{
		Key:  lockKey,
		Ttl:  toDurationProto(distributedLockTTL),
		Wait: toDurationProto(distributedLockWait),
	})
	cancelRPC()
	if err != nil {
		return nil, nil, err
	}

	if !resp.GetAcquired() || len(resp.GetLeaseID()) == 0 {
		return nil, nil, &deferredError{
			err:   errors.Errorf("Could not acquire the distributed ACME lock: %s", key),
			delay: minRetryDelay,
		}
	}

	leaseID := resp.GetLeaseID()

	lockCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(distributedLockRefresh)
		defer ticker.Stop()

		failures := 0

		for {
			select {
			case <-lockCtx.Done():
				return
			case <-ticker.C:
			}

			rpcCtx, cancelRPC := context.WithTimeout(lockCtx, distributedLockTimeout)
			resp, err := lockC.Refresh(rpcCtx, &rlockv1.RefreshRequest{
				Key:     lockKey,
				LeaseID: leaseID,
				Ttl:     toDurationProto(distributedLockTTL),
			})
			cancelRPC()

			switch {
			case err != nil:
				failures = failures + 1
				if lockCtx.Err() == nil {
					zap.L().Warn("Could not refresh the distributed ACME lock",
						zap.String("key", key),
						zap.Int("failures", failures),
						zap.Error(err))
				}
				if failures < distributedLockRefreshRetries {
					continue
				}
				cancel()
				return
			case !resp.GetRefreshed():
				zap.L().Warn("The distributed ACME lock is no longer held",
					zap.String("key", key))
				cancel()
				return
			default:
				failures = 0
			}
		}
	}()

	var once sync.Once

	return lockCtx, func() {
		once.Do(func() {
			cancel()
			<-done

			rpcCtx, cancelRPC := context.WithTimeout(
				context.WithoutCancel(ctx), distributedLockTimeout)
			defer cancelRPC()

			if _, err := lockC.Unlock(rpcCtx, &rlockv1.UnlockRequest{
				Key:     lockKey,
				LeaseID: leaseID,
			}); err != nil {
				zap.L().Warn("Could not release the distributed ACME lock",
					zap.String("key", key), zap.Error(err))
			}
		})
	}, nil
}

func toDurationProto(arg time.Duration) *metav1.Duration {
	seconds := uint64(math.Ceil(arg.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	if seconds > math.MaxUint32 {
		seconds = math.MaxUint32
	}

	return &metav1.Duration{
		Type: &metav1.Duration_Seconds{
			Seconds: uint32(seconds),
		},
	}
}
