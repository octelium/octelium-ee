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
	"sync"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func TestKeyedLocker(t *testing.T) {
	ctx := context.Background()
	locker := newKeyedLocker()

	unlock, err := locker.Lock(ctx, "a")
	assert.Nil(t, err)

	{
		unlockB, err := locker.Lock(ctx, "b")
		assert.Nil(t, err)
		unlockB()
	}

	{
		lockCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		_, err := locker.Lock(lockCtx, "a")
		cancel()
		assert.NotNil(t, err)
	}

	unlock()
	unlock()

	locker.mu.Lock()
	assert.Equal(t, 0, len(locker.locks))
	locker.mu.Unlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	counter := 0
	inside := 0

	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			unlock, err := locker.Lock(ctx, "shared")
			if err != nil {
				return
			}
			defer unlock()

			mu.Lock()
			inside = inside + 1
			assert.Equal(t, 1, inside)
			counter = counter + 1
			mu.Unlock()

			time.Sleep(time.Millisecond)

			mu.Lock()
			inside = inside - 1
			mu.Unlock()
		}()
	}

	wg.Wait()

	assert.Equal(t, 16, counter)

	locker.mu.Lock()
	assert.Equal(t, 0, len(locker.locks))
	locker.mu.Unlock()
}

func TestToDurationProto(t *testing.T) {
	assert.Equal(t, uint32(180), toDurationProto(3*time.Minute).GetSeconds())
	assert.Equal(t, uint32(1), toDurationProto(time.Millisecond).GetSeconds())
	assert.Equal(t, uint32(1), toDurationProto(0).GetSeconds())
}

func TestDistributedLock(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	c := NewController(tst.C.OcteliumC)

	key := utilrand.GetRandomStringCanonical(8)

	lockCtx, unlock, err := c.acquireLock(ctx, key)
	assert.Nil(t, err, "%+v", err)
	assert.Nil(t, lockCtx.Err())

	{
		other := NewController(tst.C.OcteliumC)
		_, _, err := other.acquireLock(ctx, key)
		assert.NotNil(t, err)
	}

	unlock()
	assert.NotNil(t, lockCtx.Err())

	{
		other := NewController(tst.C.OcteliumC)
		lockCtx, unlock, err := other.acquireLock(ctx, key)
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, lockCtx.Err())
		unlock()
	}
}
