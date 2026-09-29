// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package devicemanagers

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/crowdstrike"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/fleetdm"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/huntress"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/intune"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/iru"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/jamf"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/onepassword"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/sentinelone"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultPollInterval = 5 * time.Minute
	minPollInterval     = 30 * time.Second
	defaultPollTimeout  = 2 * time.Minute
	resyncInterval      = 1 * time.Minute
	itemsPerPage        = 500
)

type Nudger interface {
	Nudge()
}

type Resetter interface {
	ResetBindingsForOwner(ctx context.Context, ownerUID string) error
}

type Controller struct {
	octeliumC octeliumc.ClientInterface
	ctx       context.Context
	registry  *devicemgrcommon.Registry
	nudger    Nudger
	resetter  Resetter

	locks     sync.Map
	publishMu sync.Mutex

	mu      sync.Mutex
	workers map[string]*worker
}

func NewController(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	registry *devicemgrcommon.Registry,
	nudger Nudger,
	resetter Resetter,
) (*Controller, error) {
	if registry == nil {
		registry = devicemgrcommon.NewRegistry()
	}

	return &Controller{
		octeliumC: octeliumC,
		ctx:       ctx,
		registry:  registry,
		nudger:    nudger,
		resetter:  resetter,
		workers:   map[string]*worker{},
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, dm *enterprisev1.DeviceManager) error {
	return c.sync(ctx, dm.GetMetadata().GetUid())
}

func (c *Controller) OnUpdate(ctx context.Context, dm, old *enterprisev1.DeviceManager) error {
	if old != nil && pbutils.IsEqual(dm.GetSpec(), old.GetSpec()) {
		return nil
	}
	return c.sync(ctx, dm.GetMetadata().GetUid())
}

func (c *Controller) OnDelete(ctx context.Context, dm *enterprisev1.DeviceManager) error {
	uid := dm.GetMetadata().GetUid()
	if uid == "" {
		return errors.New("Invalid DeviceManager")
	}

	unlock := c.lock(uid)
	defer unlock()

	return c.doRemove(ctx, uid)
}

func (c *Controller) Run(ctx context.Context) {
	go c.run(ctx)
}

func (c *Controller) run(ctx context.Context) {
	ticker := time.NewTicker(resyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Resync(ctx); err != nil {
				zap.L().Warn("Could not resync DeviceManagers", zap.Error(err))
			}
		}
	}
}

func (c *Controller) Resync(ctx context.Context) error {
	dms, err := c.listDeviceManagers(ctx)
	if err != nil {
		return errors.Wrap(err, "Could not list DeviceManagers")
	}

	existing := make(map[string]struct{}, len(dms))
	for _, dm := range dms {
		uid := dm.GetMetadata().GetUid()
		existing[uid] = struct{}{}

		if err := c.sync(ctx, uid); err != nil {
			zap.L().Warn("Could not sync DeviceManager",
				zap.String("name", dm.GetMetadata().GetName()), zap.Error(err))
		}
	}

	for _, uid := range c.knownUIDs() {
		if _, ok := existing[uid]; ok {
			continue
		}

		unlock := c.lock(uid)
		err := c.doRemove(ctx, uid)
		unlock()
		if err != nil {
			zap.L().Warn("Could not remove deleted DeviceManager",
				zap.String("uid", uid), zap.Error(err))
		}
	}

	return c.publishProbeConfig(ctx)
}

func (c *Controller) lock(uid string) func() {
	value, _ := c.locks.LoadOrStore(uid, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()

	return mu.Unlock
}

func (c *Controller) sync(ctx context.Context, uid string) error {
	if uid == "" {
		return errors.New("Invalid DeviceManager")
	}

	unlock := c.lock(uid)
	defer unlock()

	dm, err := c.octeliumC.EnterpriseC().GetDeviceManager(ctx, &rmetav1.GetOptions{Uid: uid})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return c.doRemove(ctx, uid)
		}
		return errors.Wrap(err, "Could not get DeviceManager")
	}

	if dm.Spec.GetPolling().GetIsDisabled() {
		c.stopWorker(uid)
		c.registry.DeleteOwner(uid)
		c.setStatus(ctx, dm, enterprisev1.DeviceManager_Status_DISABLED, "")
		return c.publishProbeConfig(ctx)
	}

	if w := c.getWorker(uid); w != nil && pbutils.IsEqual(w.dm.GetSpec(), dm.GetSpec()) {
		return nil
	}

	c.stopWorker(uid)
	c.registry.SetOwner(devicemgrcommon.NewPendingOwner(dm))

	mgr, err := buildManager(c.ctx, c.octeliumC, dm)
	if err != nil {
		zap.L().Warn("Could not build DeviceManager",
			zap.String("name", dm.GetMetadata().GetName()), zap.Error(err))
		c.setStatus(ctx, dm, enterprisev1.DeviceManager_Status_ERROR, err.Error())
		return c.publishProbeConfig(ctx)
	}

	replacement := newWorker(
		c.ctx,
		c.octeliumC,
		c.registry,
		c.nudger,
		dm,
		mgr,
	)

	c.storeWorker(replacement)

	go replacement.run()

	zap.L().Info("Started DeviceManager worker",
		zap.String("name", dm.GetMetadata().GetName()),
		zap.Duration("interval", replacement.interval))

	return c.publishProbeConfig(ctx)
}

func (c *Controller) doRemove(ctx context.Context, uid string) error {
	c.stopWorker(uid)
	c.registry.DeleteOwner(uid)

	var resetErr error
	if c.resetter != nil {
		resetErr = c.resetter.ResetBindingsForOwner(ctx, uid)
	}

	if err := c.publishProbeConfig(ctx); err != nil {
		return err
	}

	if resetErr != nil {
		return errors.Wrap(resetErr, "Could not reset Device bindings for deleted DeviceManager")
	}

	return nil
}

func (c *Controller) setStatus(
	ctx context.Context,
	dm *enterprisev1.DeviceManager,
	state enterprisev1.DeviceManager_Status_State,
	errMsg string,
) {
	if dm.Status == nil {
		dm.Status = &enterprisev1.DeviceManager_Status{}
	}

	if dm.Status.Type == getStatusType(dm) &&
		dm.Status.State == state &&
		dm.Status.GetCollection().GetLastError() == errMsg {
		return
	}

	dm.Status.Type = getStatusType(dm)
	dm.Status.State = state

	if dm.Status.Collection == nil {
		dm.Status.Collection = &enterprisev1.DeviceManager_Status_Collection{}
	}
	dm.Status.Collection.LastError = errMsg

	if _, err := c.octeliumC.EnterpriseC().UpdateDeviceManager(ctx, dm); err != nil {
		zap.L().Warn("Could not update DeviceManager status",
			zap.String("deviceManager", dm.GetMetadata().GetName()),
			zap.Error(err))
	}
}

func (c *Controller) listDeviceManagers(ctx context.Context) ([]*enterprisev1.DeviceManager, error) {
	var ret []*enterprisev1.DeviceManager
	var page uint32

	for {
		itmList, err := c.octeliumC.EnterpriseC().ListDeviceManager(ctx, &rmetav1.ListOptions{
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

func (c *Controller) knownUIDs() []string {
	ret := []string{}
	seen := map[string]struct{}{}

	add := func(uid string) {
		if _, ok := seen[uid]; ok || uid == "" {
			return
		}
		seen[uid] = struct{}{}
		ret = append(ret, uid)
	}

	for _, owner := range c.registry.ListOwners() {
		add(owner.UID())
	}

	for _, w := range c.snapshotWorkers() {
		add(w.uid)
	}

	return ret
}

func (c *Controller) getWorker(uid string) *worker {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.workers[uid]
}

func (c *Controller) stopWorker(uid string) {
	c.mu.Lock()
	old := c.workers[uid]
	delete(c.workers, uid)
	c.mu.Unlock()

	if old != nil {
		old.stop()
	}
}

func (c *Controller) storeWorker(w *worker) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.workers[w.uid] = w
}

func buildManager(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	dm *enterprisev1.DeviceManager,
) (devicemgrcommon.Manager, error) {
	opts := &devicemgrcommon.ManagerOpts{
		DeviceManager: dm,
		OcteliumC:     octeliumC,
	}

	switch {
	case dm.Spec.GetCrowdStrike() != nil:
		return crowdstrike.New(ctx, octeliumC, opts)
	case dm.Spec.GetSentinelOne() != nil:
		return sentinelone.New(ctx, octeliumC, opts)
	case dm.Spec.GetMicrosoftIntune() != nil:
		return intune.New(ctx, octeliumC, opts)
	case dm.Spec.GetJamf() != nil:
		return jamf.New(ctx, octeliumC, opts)
	case dm.Spec.GetOnePassword() != nil:
		return onepassword.New(ctx, octeliumC, opts)
	case dm.Spec.GetFleetDM() != nil:
		return fleetdm.New(ctx, octeliumC, opts)
	case dm.Spec.GetHuntress() != nil:
		return huntress.New(ctx, octeliumC, opts)
	case dm.Spec.GetIru() != nil:
		return iru.New(ctx, octeliumC, opts)
	default:
		return nil, errors.Errorf(
			"Unsupported DeviceManager type: %s",
			dm.GetMetadata().GetName(),
		)
	}
}

func getStatusType(dm *enterprisev1.DeviceManager) enterprisev1.DeviceManager_Status_Type {
	switch {
	case dm.Spec.GetCrowdStrike() != nil:
		return enterprisev1.DeviceManager_Status_CROWDSTRIKE
	case dm.Spec.GetSentinelOne() != nil:
		return enterprisev1.DeviceManager_Status_SENTINELONE
	case dm.Spec.GetMicrosoftIntune() != nil:
		return enterprisev1.DeviceManager_Status_MICROSOFT_INTUNE
	case dm.Spec.GetJamf() != nil:
		return enterprisev1.DeviceManager_Status_JAMF_PRO
	case dm.Spec.GetOnePassword() != nil:
		return enterprisev1.DeviceManager_Status_ONEPASSWORD
	case dm.Spec.GetFleetDM() != nil:
		return enterprisev1.DeviceManager_Status_FLEETDM
	case dm.Spec.GetHuntress() != nil:
		return enterprisev1.DeviceManager_Status_HUNTRESS
	case dm.Spec.GetIru() != nil:
		return enterprisev1.DeviceManager_Status_IRU
	default:
		return enterprisev1.DeviceManager_Status_TYPE_UNKNOWN
	}
}

func (c *Controller) publishProbeConfig(ctx context.Context) error {
	c.publishMu.Lock()
	defer c.publishMu.Unlock()

	deviceConfig := &corev1.ClusterConfig_Status_Device{}

	for _, w := range c.snapshotWorkers() {
		if !devicemgrcommon.UsesProbe(w.dm) {
			continue
		}

		for _, probe := range w.mgr.IdentityProbes() {
			if coreProbe := toCoreProbe(w.dm, probe); coreProbe != nil {
				deviceConfig.Probes = append(deviceConfig.Probes, coreProbe)
			}
		}
	}

	sort.Slice(deviceConfig.Probes, func(i, j int) bool {
		return deviceConfig.Probes[i].Id < deviceConfig.Probes[j].Id
	})

	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}

	if cc.Status == nil {
		cc.Status = &corev1.ClusterConfig_Status{}
	}

	if pbutils.IsEqual(cc.Status.Device, deviceConfig) {
		return nil
	}

	cc.Status.Device = deviceConfig

	if _, err := c.octeliumC.CoreC().UpdateClusterConfig(ctx, cc); err != nil {
		return errors.Wrap(err, "Could not publish Device probe config")
	}

	return nil
}

func (c *Controller) snapshotWorkers() []*worker {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]*worker, 0, len(c.workers))
	for _, w := range c.workers {
		out = append(out, w)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].uid < out[j].uid
	})

	return out
}

func toCoreProbe(
	dm *enterprisev1.DeviceManager,
	probe *devicemgrcommon.Probe,
) *corev1.ClusterConfig_Status_Device_Probe {
	if probe == nil || probe.ID == "" {
		return nil
	}

	out := &corev1.ClusterConfig_Status_Device_Probe{
		Id:               fmt.Sprintf("%s.%s", dm.GetMetadata().GetName(), probe.ID),
		OwnerRef:         umetav1.GetObjectReference(dm),
		OsTypes:          append([]corev1.Device_Status_OSType(nil), probe.OSTypes...),
		RequireElevation: probe.RequireElevation,
		Condition:        dm.Spec.GetCondition(),
	}

	switch {
	case probe.RunCommand != nil:
		out.Type = &corev1.ClusterConfig_Status_Device_Probe_RunCommand_{
			RunCommand: &corev1.ClusterConfig_Status_Device_Probe_RunCommand{
				Command:        probe.RunCommand.Command,
				Args:           append([]string(nil), probe.RunCommand.Args...),
				TimeoutSeconds: probe.RunCommand.TimeoutSeconds,
				MaxOutputBytes: probe.RunCommand.MaxOutputBytes,
			},
		}
	case probe.ReadFile != nil:
		out.Type = &corev1.ClusterConfig_Status_Device_Probe_ReadFile_{
			ReadFile: &corev1.ClusterConfig_Status_Device_Probe_ReadFile{
				Path:     probe.ReadFile.Path,
				MaxBytes: probe.ReadFile.MaxBytes,
			},
		}
	case probe.ReadRegistry != nil:
		out.Type = &corev1.ClusterConfig_Status_Device_Probe_ReadRegistry_{
			ReadRegistry: &corev1.ClusterConfig_Status_Device_Probe_ReadRegistry{
				Key:  probe.ReadRegistry.Key,
				Name: probe.ReadRegistry.Name,
			},
		}
	case probe.PlatformIdentifier != nil:
		out.Type = &corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_{
			PlatformIdentifier: &corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier{
				Kind: probe.PlatformIdentifier.Kind,
			},
		}
	default:
		return nil
	}

	return out
}

type worker struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	octeliumC octeliumc.ClientInterface
	registry  *devicemgrcommon.Registry
	nudger    Nudger

	uid  string
	name string
	dm   *enterprisev1.DeviceManager
	mgr  devicemgrcommon.Manager

	interval time.Duration
	timeout  time.Duration
}

func newWorker(
	ctx context.Context,
	octeliumC octeliumc.ClientInterface,
	registry *devicemgrcommon.Registry,
	nudger Nudger,
	dm *enterprisev1.DeviceManager,
	mgr devicemgrcommon.Manager,
) *worker {
	workerCtx, cancel := context.WithCancel(ctx)
	clonedDM := pbutils.Clone(dm).(*enterprisev1.DeviceManager)

	return &worker{
		ctx:       workerCtx,
		cancel:    cancel,
		done:      make(chan struct{}),
		octeliumC: octeliumC,
		registry:  registry,
		nudger:    nudger,
		uid:       clonedDM.GetMetadata().GetUid(),
		name:      clonedDM.GetMetadata().GetName(),
		dm:        clonedDM,
		mgr:       mgr,
		interval:  pollInterval(clonedDM),
		timeout:   pollTimeout(clonedDM),
	}
}

func (w *worker) stop() {
	w.cancel()
	<-w.done
	_ = w.mgr.Close()
}

func (w *worker) run() {
	defer close(w.done)

	w.setStatus(
		enterprisev1.DeviceManager_Status_LOADING,
		pbutils.Now(),
		nil,
		0,
		"",
	)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.poll()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.poll()
		}
	}
}

func (w *worker) poll() {
	pollCtx, cancel := context.WithTimeout(w.ctx, w.timeout)
	defer cancel()

	attemptAt := pbutils.Now()

	fleet, err := w.mgr.Collect(pollCtx)
	if err != nil {
		if w.ctx.Err() != nil {
			return
		}

		zap.L().Warn("Could not collect from DeviceManager",
			zap.String("name", w.name),
			zap.Error(err))

		w.setStatus(
			enterprisev1.DeviceManager_Status_ERROR,
			attemptAt,
			nil,
			0,
			err.Error(),
		)
		return
	}

	if w.ctx.Err() != nil {
		return
	}

	collectedAt := time.Now()

	w.registry.SetOwner(devicemgrcommon.NewOwner(
		w.mgr,
		w.dm,
		fleet,
		collectedAt,
		devicemgrcommon.StaleAfter(w.dm),
	))

	if w.nudger != nil {
		w.nudger.Nudge()
	}

	w.setStatus(
		enterprisev1.DeviceManager_Status_OK,
		attemptAt,
		pbutils.Timestamp(collectedAt),
		uint32(fleet.Len()),
		"",
	)
}

func (w *worker) setStatus(
	state enterprisev1.DeviceManager_Status_State,
	attemptAt *timestamppb.Timestamp,
	successAt *timestamppb.Timestamp,
	managed uint32,
	errMsg string,
) {
	dm, err := w.octeliumC.EnterpriseC().GetDeviceManager(
		w.ctx,
		&rmetav1.GetOptions{Uid: w.uid},
	)
	if err != nil {
		return
	}

	if dm.Status == nil {
		dm.Status = &enterprisev1.DeviceManager_Status{}
	}

	dm.Status.Type = getStatusType(dm)
	dm.Status.State = state

	if dm.Status.Collection == nil {
		dm.Status.Collection = &enterprisev1.DeviceManager_Status_Collection{}
	}

	collection := dm.Status.Collection
	collection.LastAttemptAt = attemptAt
	collection.LastError = errMsg

	if successAt != nil {
		collection.LastSuccessAt = successAt
		collection.ManagedDevices = managed
	}

	if _, err := w.octeliumC.EnterpriseC().UpdateDeviceManager(w.ctx, dm); err != nil {
		zap.L().Warn("Could not update DeviceManager status",
			zap.String("deviceManager", w.name),
			zap.Error(err))
	}
}

func pollInterval(dm *enterprisev1.DeviceManager) time.Duration {
	interval := defaultPollInterval

	if polling := dm.Spec.GetPolling(); polling != nil && polling.GetInterval() != nil {
		if value := umetav1.ToDuration(polling.GetInterval()).ToGo(); value > 0 {
			interval = value
		}
	}

	if interval < minPollInterval {
		interval = minPollInterval
	}

	return interval
}

func pollTimeout(dm *enterprisev1.DeviceManager) time.Duration {
	if polling := dm.Spec.GetPolling(); polling != nil && polling.GetTimeout() != nil {
		if value := umetav1.ToDuration(polling.GetTimeout()).ToGo(); value > 0 {
			return value
		}
	}

	return defaultPollTimeout
}
