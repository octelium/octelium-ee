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
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultSweepInterval = 1 * time.Minute
	minSweepInterval     = 10 * time.Second
	itemsPerPage         = 500
)

type Reconciler interface {
	ReconcileDevice(ctx context.Context, dev *corev1.Device) error
	ResetDeviceBinding(ctx context.Context, deviceUID, bindingUID string) error
}

type Resolver interface {
	ListOwners() []*devicemgrcommon.Owner
}

type Opts struct {
	OcteliumC  octeliumc.ClientInterface
	Resolver   Resolver
	Reconciler Reconciler
	Interval   time.Duration
}

type Watcher struct {
	octeliumC  octeliumc.ClientInterface
	resolver   Resolver
	reconciler Reconciler
	interval   time.Duration
	nudgeCh    chan struct{}
}

func NewWatcher(opts *Opts) (*Watcher, error) {
	if opts == nil || opts.OcteliumC == nil || opts.Resolver == nil || opts.Reconciler == nil {
		return nil, errors.New("Invalid devwatcher Opts")
	}

	interval := opts.Interval
	if interval <= 0 {
		interval = defaultSweepInterval
	}
	if interval < minSweepInterval {
		interval = minSweepInterval
	}

	return &Watcher{
		octeliumC:  opts.OcteliumC,
		resolver:   opts.Resolver,
		reconciler: opts.Reconciler,
		interval:   interval,
		nudgeCh:    make(chan struct{}, 1),
	}, nil
}

func (w *Watcher) Run(ctx context.Context) {
	go w.run(ctx)
}

func (w *Watcher) Nudge() {
	select {
	case w.nudgeCh <- struct{}{}:
	default:
	}
}

func (w *Watcher) run(ctx context.Context) {
	zap.L().Debug("Starting Device posture watcher")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	if err := w.doSweep(ctx); err != nil {
		zap.L().Error("Could not run Device posture sweep", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.doSweep(ctx); err != nil {
				zap.L().Error("Could not run Device posture sweep", zap.Error(err))
			}
		case <-w.nudgeCh:
			if err := w.doSweep(ctx); err != nil {
				zap.L().Error("Could not run Device posture sweep", zap.Error(err))
			}
		}
	}
}

type linkTally struct {
	linked        uint32
	ambiguous     uint32
	conflicts     uint32
	suspended     uint32
	failedUpdates uint32
}

func tallyFor(tallies map[string]*linkTally, ownerUID string) *linkTally {
	if ownerUID == "" {
		return &linkTally{}
	}

	t, ok := tallies[ownerUID]
	if !ok {
		t = &linkTally{}
		tallies[ownerUID] = t
	}
	return t
}

func (w *Watcher) doSweep(ctx context.Context) error {
	sweepAt := pbutils.Now()

	devices, err := w.listDevices(ctx)
	if err != nil {
		return errors.Wrap(err, "Could not list Devices")
	}

	dmUIDs, err := w.listDeviceManagerUIDs(ctx)
	if err != nil {
		zap.L().Warn("Could not list DeviceManagers during Device sweep", zap.Error(err))
	}

	tallies := map[string]*linkTally{}

	for _, dev := range devices {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		w.sweepDevice(ctx, dev, dmUIDs, tallies)
	}

	w.updateLinkingStatus(ctx, tallies, sweepAt)

	return nil
}

func (w *Watcher) listDevices(ctx context.Context) ([]*corev1.Device, error) {
	var ret []*corev1.Device
	var page uint32

	for {
		itmList, err := w.octeliumC.CoreC().ListDevice(ctx, &rmetav1.ListOptions{
			Paginate:     true,
			ItemsPerPage: itemsPerPage,
			Page:         page,
		})
		if err != nil {
			return nil, err
		}

		ret = append(ret, itmList.Items...)

		if itmList.ListResponseMeta == nil || !itmList.ListResponseMeta.HasMore {
			return ret, nil
		}

		page = page + 1
	}
}

func (w *Watcher) listDeviceManagerUIDs(ctx context.Context) (map[string]struct{}, error) {
	ret := map[string]struct{}{}
	var page uint32

	for {
		itmList, err := w.octeliumC.EnterpriseC().ListDeviceManager(ctx, &rmetav1.ListOptions{
			Paginate:     true,
			ItemsPerPage: itemsPerPage,
			Page:         page,
		})
		if err != nil {
			return nil, err
		}

		for _, itm := range itmList.Items {
			ret[itm.Metadata.Uid] = struct{}{}
		}

		if itmList.ListResponseMeta == nil || !itmList.ListResponseMeta.HasMore {
			return ret, nil
		}

		page = page + 1
	}
}

func (w *Watcher) sweepDevice(
	ctx context.Context,
	dev *corev1.Device,
	dmUIDs map[string]struct{},
	tallies map[string]*linkTally,
) {
	if dev == nil || dev.GetStatus() == nil {
		return
	}

	binding := dev.Status.GetBinding()
	ownerUID := binding.GetOwnerRef().GetUid()

	if dmUIDs != nil && ownerUID != "" {
		if _, ok := dmUIDs[ownerUID]; !ok {
			if err := w.reconciler.ResetDeviceBinding(ctx, dev.GetMetadata().GetUid(), binding.GetUid()); err != nil {
				zap.L().Warn("Could not reset the Device binding of a deleted DeviceManager",
					zap.String("device", dev.GetMetadata().GetName()),
					zap.Error(err))
			}
			return
		}
	}

	t := tallyFor(tallies, ownerUID)

	switch binding.GetState() {
	case corev1.Device_Status_Binding_ACCEPTED:
		t.linked++
		if binding.GetValidity() != corev1.Device_Status_Binding_VALID {
			t.suspended++
		}
	case corev1.Device_Status_Binding_AMBIGUOUS:
		t.ambiguous++
	case corev1.Device_Status_Binding_CONFLICT:
		t.conflicts++
	}

	if err := w.reconciler.ReconcileDevice(ctx, dev); err != nil {
		t.failedUpdates++
		zap.L().Warn("Could not reconcile Device binding",
			zap.String("device", dev.GetMetadata().GetName()),
			zap.Error(err))
	}
}

func (w *Watcher) updateLinkingStatus(
	ctx context.Context,
	tallies map[string]*linkTally,
	sweepAt *timestamppb.Timestamp,
) {
	for _, owner := range w.resolver.ListOwners() {
		ownerUID := owner.UID()
		if ownerUID == "" {
			continue
		}

		t := tallies[ownerUID]
		if t == nil {
			t = &linkTally{}
		}

		dm, err := w.octeliumC.EnterpriseC().GetDeviceManager(ctx, &rmetav1.GetOptions{
			Uid: ownerUID,
		})
		if err != nil {
			zap.L().Warn("Could not get DeviceManager while updating Linking status",
				zap.String("deviceManager", owner.Name()),
				zap.Error(err))
			continue
		}

		if dm.Status == nil {
			dm.Status = &enterprisev1.DeviceManager_Status{}
		}

		dm.Status.Linking = &enterprisev1.DeviceManager_Status_Linking{
			LastSweepAt:   sweepAt,
			LinkedDevices: t.linked,
			Ambiguous:     t.ambiguous,
			Conflicts:     t.conflicts,
			Suspended:     t.suspended,
			FailedUpdates: t.failedUpdates,
		}

		if _, err := w.octeliumC.EnterpriseC().UpdateDeviceManager(ctx, dm); err != nil {
			zap.L().Warn("Could not update DeviceManager Linking status",
				zap.String("deviceManager", dm.GetMetadata().GetName()),
				zap.Error(err))
		}
	}
}
