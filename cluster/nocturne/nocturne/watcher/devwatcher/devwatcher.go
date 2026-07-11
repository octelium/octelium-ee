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
}

type Resolver interface {
	GetOwner(ownerUID string) (*devicemgrcommon.Owner, bool)
	ListOwners() []*devicemgrcommon.Owner
}

type DeviceLocker interface {
	LockDevice(uid string) func()
}

type Opts struct {
	OcteliumC  octeliumc.ClientInterface
	Resolver   Resolver
	Reconciler Reconciler
	Locker     DeviceLocker
	Interval   time.Duration
}

type Watcher struct {
	octeliumC  octeliumc.ClientInterface
	resolver   Resolver
	reconciler Reconciler
	locker     DeviceLocker
	interval   time.Duration
	nudgeCh    chan struct{}
}

func NewWatcher(opts *Opts) (*Watcher, error) {
	if opts == nil || opts.OcteliumC == nil || opts.Resolver == nil || opts.Locker == nil {
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
		locker:     opts.Locker,
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
	linked          uint32
	waitingApproval uint32
	ambiguous       uint32
	failedUpdates   uint32
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

	tallies := map[string]*linkTally{}

	for _, dev := range devices {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		w.sweepDevice(ctx, dev, tallies)
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

func (w *Watcher) sweepDevice(
	ctx context.Context,
	dev *corev1.Device,
	tallies map[string]*linkTally,
) {
	if dev == nil || dev.GetStatus() == nil {
		return
	}

	binding := dev.Status.GetBinding()

	switch binding.GetState() {
	case corev1.Device_Status_Binding_ACCEPTED:
		t := tallyFor(tallies, binding.GetOwnerRef().GetUid())
		t.linked++

		if err := w.refreshPosture(ctx, dev.GetMetadata().GetUid()); err != nil {
			t.failedUpdates++
			zap.L().Warn("Could not refresh Device posture",
				zap.String("device", dev.GetMetadata().GetName()),
				zap.Error(err))
		}

	case corev1.Device_Status_Binding_WAITING_APPROVAL:
		tallyFor(tallies, binding.GetOwnerRef().GetUid()).waitingApproval++
		w.delegate(ctx, dev)

	case corev1.Device_Status_Binding_AMBIGUOUS:
		if ownerUID := binding.GetOwnerRef().GetUid(); ownerUID != "" {
			tallyFor(tallies, ownerUID).ambiguous++
		}
		w.delegate(ctx, dev)

	case corev1.Device_Status_Binding_REJECTED:

	default:
		w.delegate(ctx, dev)
	}
}

func (w *Watcher) delegate(ctx context.Context, dev *corev1.Device) {
	if w.reconciler == nil {
		return
	}

	if err := w.reconciler.ReconcileDevice(ctx, dev); err != nil {
		zap.L().Warn("Could not reconcile Device binding",
			zap.String("device", dev.GetMetadata().GetName()),
			zap.Error(err))
	}
}

func (w *Watcher) refreshPosture(ctx context.Context, deviceUID string) error {
	if deviceUID == "" {
		return errors.New("Invalid Device uid")
	}

	unlock := w.locker.LockDevice(deviceUID)
	defer unlock()

	dev, err := w.octeliumC.CoreC().GetDevice(ctx, &rmetav1.GetOptions{
		Uid: deviceUID,
	})
	if err != nil {
		return errors.Wrap(err, "Could not get Device")
	}

	binding := dev.Status.GetBinding()
	if binding.GetState() != corev1.Device_Status_Binding_ACCEPTED {
		return nil
	}

	now := time.Now()

	owner, ok := w.resolver.GetOwner(binding.GetOwnerRef().GetUid())
	if !ok || owner == nil || !owner.Fresh(now) {
		return w.clearExpiredPosture(ctx, dev, now)
	}

	match := owner.Fleet.MatchExternalID(binding.GetExternalID())
	if match.State != devicemgrcommon.MatchStateUnique {
		return w.clearExpiredPosture(ctx, dev, now)
	}

	desired := devicemgrcommon.MaterializePosture(owner, match.Entry)
	current := dev.Status.GetPosture()

	if posturesEqualIgnoringTimestamps(current, desired) && !refreshDue(current, owner, now) {
		return nil
	}

	dev.Status.Posture = desired

	if _, err := w.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
		return errors.Wrap(err, "Could not update Device posture")
	}

	return nil
}

func (w *Watcher) clearExpiredPosture(
	ctx context.Context,
	dev *corev1.Device,
	now time.Time,
) error {
	posture := dev.Status.GetPosture()
	if posture == nil {
		return nil
	}

	expiresAt := posture.GetExpiresAt()
	if expiresAt == nil || !expiresAt.IsValid() || now.Before(expiresAt.AsTime()) {
		return nil
	}

	dev.Status.Posture = nil

	if _, err := w.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
		return errors.Wrap(err, "Could not clear expired Device posture")
	}

	return nil
}

func posturesEqualIgnoringTimestamps(a, b *corev1.Device_Status_Posture) bool {
	return pbutils.IsEqual(normalizePosture(a), normalizePosture(b))
}

func normalizePosture(posture *corev1.Device_Status_Posture) *corev1.Device_Status_Posture {
	if posture == nil {
		return nil
	}

	out := pbutils.Clone(posture).(*corev1.Device_Status_Posture)
	out.LastSyncAt = nil
	out.LastSeenAt = nil
	out.ExpiresAt = nil

	return out
}

func refreshDue(
	current *corev1.Device_Status_Posture,
	owner *devicemgrcommon.Owner,
	now time.Time,
) bool {
	if current == nil {
		return true
	}

	expiresAt := current.GetExpiresAt()
	if expiresAt == nil || !expiresAt.IsValid() {
		return true
	}

	staleAfter := owner.ExpiresAt.Sub(owner.CollectedAt)
	if staleAfter <= 0 {
		return true
	}

	return !now.Before(expiresAt.AsTime().Add(-staleAfter / 2))
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
				zap.String("deviceManager", owner.DM.GetMetadata().GetName()),
				zap.Error(err))
			continue
		}

		if dm.Status == nil {
			dm.Status = &enterprisev1.DeviceManager_Status{}
		}

		dm.Status.Linking = &enterprisev1.DeviceManager_Status_Linking{
			LastSweepAt:     sweepAt,
			LinkedDevices:   t.linked,
			WaitingApproval: t.waitingApproval,
			Ambiguous:       t.ambiguous,
			FailedUpdates:   t.failedUpdates,
		}

		if _, err := w.octeliumC.EnterpriseC().UpdateDeviceManager(ctx, dm); err != nil {
			zap.L().Warn("Could not update DeviceManager Linking status",
				zap.String("deviceManager", dm.GetMetadata().GetName()),
				zap.Error(err))
		}
	}
}
