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
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func mins(n uint32) *metav1.Duration {
	return &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: n}}
}

func TestWatcherHelpers(t *testing.T) {
	assert.Equal(t, time.Duration(0), pollingInterval(&enterprisev1.DirectoryProvider{}))

	assert.Equal(t, time.Duration(0), pollingInterval(&enterprisev1.DirectoryProvider{
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{
				Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
			},
		},
	}))

	assert.Equal(t, 10*time.Minute, pollingInterval(&enterprisev1.DirectoryProvider{
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_{
				GoogleWorkspace: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace{
					Polling: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_Polling{Interval: mins(10)},
				},
			},
		},
	}))

	assert.Equal(t, time.Duration(0), pollingInterval(&enterprisev1.DirectoryProvider{
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_{
				GoogleWorkspace: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace{},
			},
		},
	}))

	assert.Equal(t, 15*time.Minute, pollingInterval(&enterprisev1.DirectoryProvider{
		Spec: &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Keycloak_{
				Keycloak: &enterprisev1.DirectoryProvider_Spec_Keycloak{
					Polling: &enterprisev1.DirectoryProvider_Spec_Keycloak_Polling{Interval: mins(15)},
				},
			},
		},
	}))

	assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_STATE_UNSET,
		syncState(&enterprisev1.DirectoryProvider{}))
	assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_STATE_UNSET,
		syncState(&enterprisev1.DirectoryProvider{Status: &enterprisev1.DirectoryProvider_Status{}}))
	assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS,
		syncState(&enterprisev1.DirectoryProvider{
			Status: &enterprisev1.DirectoryProvider_Status{
				Synchronization: &enterprisev1.DirectoryProvider_Status_Synchronization{
					State: enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS,
				},
			},
		}))

	base := 5 * time.Minute
	assert.Equal(t, base, backoff(base, 0))
	assert.Equal(t, base, backoff(base, 1))
	assert.Equal(t, 2*base, backoff(base, 2))
	assert.Equal(t, 4*base, backoff(base, 3))
	assert.Equal(t, maxBackoff, backoff(base, 4))
	assert.Equal(t, maxBackoff, backoff(base, 100))

	for range 200 {
		s := spread(10 * time.Minute)
		assert.True(t, s >= 0 && s < 10*time.Minute)

		j := jitter(10 * time.Minute)
		assert.True(t, j >= 0 && j < time.Minute)
	}

	assert.Equal(t, time.Duration(0), spread(0))
	assert.Equal(t, time.Duration(0), jitter(0))
	assert.Equal(t, time.Duration(0), jitter(time.Duration(5)))
}

func TestWatcher(t *testing.T) {
	ctx := context.Background()

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	googleSpec := func(minutes uint32, disabled bool) *enterprisev1.DirectoryProvider_Spec {
		return &enterprisev1.DirectoryProvider_Spec{
			IsDisabled: disabled,
			Type: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_{
				GoogleWorkspace: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace{
					Polling: &enterprisev1.DirectoryProvider_Spec_GoogleWorkspace_Polling{Interval: mins(minutes)},
				},
			},
		}
	}

	scimSpec := func() *enterprisev1.DirectoryProvider_Spec {
		return &enterprisev1.DirectoryProvider_Spec{
			Type: &enterprisev1.DirectoryProvider_Spec_Scim{
				Scim: &enterprisev1.DirectoryProvider_Spec_SCIM{},
			},
		}
	}

	createDP := func(spec *enterprisev1.DirectoryProvider_Spec) *enterprisev1.DirectoryProvider {
		dp, err := fakeC.OcteliumC.EnterpriseC().CreateDirectoryProvider(ctx, &enterprisev1.DirectoryProvider{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec:   spec,
			Status: &enterprisev1.DirectoryProvider_Status{},
		})
		assert.Nil(t, err, "%+v", err)
		return dp
	}

	setState := func(dp *enterprisev1.DirectoryProvider,
		state enterprisev1.DirectoryProvider_Status_Synchronization_State) *enterprisev1.DirectoryProvider {
		if dp.Status == nil {
			dp.Status = &enterprisev1.DirectoryProvider_Status{}
		}
		dp.Status.Synchronization = &enterprisev1.DirectoryProvider_Status_Synchronization{State: state}
		updated, err := fakeC.OcteliumC.EnterpriseC().UpdateDirectoryProvider(ctx, dp)
		assert.Nil(t, err, "%+v", err)
		return updated
	}

	getState := func(uid string) enterprisev1.DirectoryProvider_Status_Synchronization_State {
		cur, err := fakeC.OcteliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{Uid: uid})
		assert.Nil(t, err)
		return syncState(cur)
	}

	w := New(fakeC.OcteliumC)
	t0 := time.Now()

	{
		dp := createDP(googleSpec(10, false))

		w.reconcileOne(ctx, dp, t0)
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_STATE_UNSET,
			getState(dp.Metadata.Uid))

		w.reconcileOne(ctx, dp, t0.Add(11*time.Minute))
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED,
			getState(dp.Metadata.Uid))

		cur, err := fakeC.OcteliumC.EnterpriseC().GetDirectoryProvider(ctx, &rmetav1.GetOptions{
			Uid: dp.Metadata.Uid,
		})
		assert.Nil(t, err)
		assert.NotNil(t, cur.Status.Synchronization.CreatedAt)

		w.reconcileOne(ctx, cur, t0.Add(30*time.Minute))
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED,
			getState(dp.Metadata.Uid))
	}

	{
		dp := createDP(googleSpec(10, true))

		w.reconcileOne(ctx, dp, t0)
		w.reconcileOne(ctx, dp, t0.Add(11*time.Minute))
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_STATE_UNSET,
			getState(dp.Metadata.Uid))
	}

	{
		dp := createDP(scimSpec())

		w.reconcileOne(ctx, dp, t0)
		w.reconcileOne(ctx, dp, t0.Add(11*time.Minute))
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_STATE_UNSET,
			getState(dp.Metadata.Uid))
	}

	{
		dp := setState(createDP(scimSpec()),
			enterprisev1.DirectoryProvider_Status_Synchronization_SYNCING)

		err := w.requestSync(ctx, dp)
		assert.Nil(t, err)
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_SYNCING,
			getState(dp.Metadata.Uid))
	}

	{
		dp := setState(createDP(scimSpec()),
			enterprisev1.DirectoryProvider_Status_Synchronization_SUCCESS)

		err := w.requestSync(ctx, dp)
		assert.Nil(t, err)
		assert.Equal(t, enterprisev1.DirectoryProvider_Status_Synchronization_SYNC_REQUESTED,
			getState(dp.Metadata.Uid))
	}

	{
		w2 := New(fakeC.OcteliumC)
		dp := createDP(googleSpec(10, false))

		err := w2.poll(ctx)
		assert.Nil(t, err)
		_, ok := w2.schedule[dp.Metadata.Uid]
		assert.True(t, ok)

		_, err = fakeC.OcteliumC.EnterpriseC().DeleteDirectoryProvider(ctx, &rmetav1.DeleteOptions{
			Uid: dp.Metadata.Uid,
		})
		assert.Nil(t, err)

		err = w2.poll(ctx)
		assert.Nil(t, err)
		_, ok = w2.schedule[dp.Metadata.Uid]
		assert.False(t, ok)
	}
}
