// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package devwatcher

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type tstReconciler struct {
	mu         sync.Mutex
	failUIDs   map[string]struct{}
	reconciled []string
	resets     map[string]string
}

func (r *tstReconciler) ReconcileDevice(ctx context.Context, dev *corev1.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reconciled = append(r.reconciled, dev.Metadata.Uid)
	if _, ok := r.failUIDs[dev.Metadata.Uid]; ok {
		return errors.New("reconcile error")
	}
	return nil
}

func (r *tstReconciler) ResetDeviceBinding(ctx context.Context, deviceUID, bindingUID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.resets[deviceUID] = bindingUID
	return nil
}

func TestNewWatcher(t *testing.T) {
	{
		_, err := NewWatcher(nil)
		assert.NotNil(t, err)
	}

	{
		_, err := NewWatcher(&Opts{})
		assert.NotNil(t, err)
	}
}

func TestSweep(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	octeliumC := tst.C.OcteliumC

	dm, err := octeliumC.EnterpriseC().CreateDeviceManager(ctx, &enterprisev1.DeviceManager{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.DeviceManager_Spec{
			Type: &enterprisev1.DeviceManager_Spec_Iru_{
				Iru: &enterprisev1.DeviceManager_Spec_Iru{},
			},
		},
		Status: &enterprisev1.DeviceManager_Status{},
	})
	assert.Nil(t, err)

	registry := devicemgrcommon.NewRegistry()
	registry.SetOwner(devicemgrcommon.NewPendingOwner(dm))

	createDevice := func(binding *corev1.Device_Status_Binding) *corev1.Device {
		dev, err := octeliumC.CoreC().CreateDevice(ctx, &corev1.Device{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &corev1.Device_Spec{},
			Status: &corev1.Device_Status{
				Binding: binding,
			},
		})
		assert.Nil(t, err)
		return dev
	}

	newBinding := func(ownerRef *metav1.ObjectReference,
		state corev1.Device_Status_Binding_State,
		validity corev1.Device_Status_Binding_Validity) *corev1.Device_Status_Binding {
		return &corev1.Device_Status_Binding{
			Uid:        utilrand.GetRandomStringCanonical(32),
			OwnerRef:   ownerRef,
			ExternalID: utilrand.GetRandomStringCanonical(8),
			State:      state,
			Validity:   validity,
		}
	}

	ownerRef := umetav1.GetObjectReference(dm)
	orphanRef := &metav1.ObjectReference{
		Name: "deleted",
		Uid:  "00000000-0000-0000-0000-000000000001",
	}

	valid := createDevice(newBinding(ownerRef, corev1.Device_Status_Binding_ACCEPTED, corev1.Device_Status_Binding_VALID))
	suspended := createDevice(newBinding(ownerRef, corev1.Device_Status_Binding_ACCEPTED, corev1.Device_Status_Binding_SUSPENDED))
	lost := createDevice(newBinding(ownerRef, corev1.Device_Status_Binding_ACCEPTED, corev1.Device_Status_Binding_LOST))
	ambiguous := createDevice(newBinding(ownerRef, corev1.Device_Status_Binding_AMBIGUOUS, corev1.Device_Status_Binding_VALIDITY_UNKNOWN))
	conflict := createDevice(newBinding(ownerRef, corev1.Device_Status_Binding_CONFLICT, corev1.Device_Status_Binding_VALIDITY_UNKNOWN))
	orphan := createDevice(newBinding(orphanRef, corev1.Device_Status_Binding_ACCEPTED, corev1.Device_Status_Binding_VALID))
	unbound := createDevice(nil)

	reconciler := &tstReconciler{
		failUIDs: map[string]struct{}{
			suspended.Metadata.Uid: {},
			unbound.Metadata.Uid:   {},
		},
		resets: map[string]string{},
	}

	w, err := NewWatcher(&Opts{
		OcteliumC:  octeliumC,
		Resolver:   registry,
		Reconciler: reconciler,
		Interval:   time.Second,
	})
	assert.Nil(t, err)
	assert.Equal(t, minSweepInterval, w.interval)

	w.Nudge()
	w.Nudge()

	assert.Nil(t, w.doSweep(ctx))

	{
		expected := []string{
			valid.Metadata.Uid,
			suspended.Metadata.Uid,
			lost.Metadata.Uid,
			ambiguous.Metadata.Uid,
			conflict.Metadata.Uid,
			unbound.Metadata.Uid,
		}
		sort.Strings(expected)

		reconciled := append([]string(nil), reconciler.reconciled...)
		sort.Strings(reconciled)

		assert.Equal(t, expected, reconciled)
		assert.Equal(t, map[string]string{
			orphan.Metadata.Uid: orphan.Status.Binding.Uid,
		}, reconciler.resets)
	}

	{
		dm, err := octeliumC.EnterpriseC().GetDeviceManager(ctx, &rmetav1.GetOptions{Uid: dm.Metadata.Uid})
		assert.Nil(t, err)

		linking := dm.Status.Linking
		assert.True(t, linking.LastSweepAt.IsValid())
		assert.Equal(t, uint32(3), linking.LinkedDevices)
		assert.Equal(t, uint32(2), linking.Suspended)
		assert.Equal(t, uint32(1), linking.Ambiguous)
		assert.Equal(t, uint32(1), linking.Conflicts)
		assert.Equal(t, uint32(1), linking.FailedUpdates)
	}
}
