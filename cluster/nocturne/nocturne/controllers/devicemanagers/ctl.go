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
	"sort"
	"strings"
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
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultPollInterval = 5 * time.Minute
	minPollInterval     = 30 * time.Second
	defaultPollTimeout  = 2 * time.Minute
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
	return c.sync(ctx, dm)
}

func (c *Controller) OnUpdate(ctx context.Context, dm, old *enterprisev1.DeviceManager) error {
	if old != nil && pbutils.IsEqual(dm.GetSpec(), old.GetSpec()) {
		return nil
	}
	return c.sync(ctx, dm)
}

func (c *Controller) OnDelete(ctx context.Context, dm *enterprisev1.DeviceManager) error {
	uid := dm.GetMetadata().GetUid()

	old := c.removeWorker(uid)
	if old != nil {
		old.stop()
	}

	c.registry.DeleteOwner(uid)

	if c.resetter != nil {
		if err := c.resetter.ResetBindingsForOwner(ctx, uid); err != nil {
			return errors.Wrap(err, "Could not reset Device bindings for deleted DeviceManager")
		}
	}

	return c.publishProbeConfig(ctx)
}

func (c *Controller) sync(ctx context.Context, dm *enterprisev1.DeviceManager) error {
	if dm == nil || dm.GetMetadata().GetUid() == "" {
		return errors.New("Invalid DeviceManager")
	}

	if polling := dm.Spec.GetPolling(); polling != nil && polling.GetIsDisabled() {
		return c.disable(ctx, dm)
	}

	mgr, err := buildManager(c.ctx, c.octeliumC, dm)
	if err != nil {
		return errors.Wrap(err, "Could not build DeviceManager")
	}

	uid := dm.GetMetadata().GetUid()

	old := c.removeWorker(uid)
	if old != nil {
		old.stop()
	}

	replacement := newWorker(
		c.ctx,
		c.octeliumC,
		c.registry,
		c.nudger,
		dm,
		mgr,
	)

	go replacement.run()

	c.storeWorker(replacement)

	zap.L().Info("Started DeviceManager worker",
		zap.String("name", dm.GetMetadata().GetName()),
		zap.Duration("interval", replacement.interval))

	return c.publishProbeConfig(ctx)
}

func (c *Controller) disable(ctx context.Context, dm *enterprisev1.DeviceManager) error {
	uid := dm.GetMetadata().GetUid()

	old := c.removeWorker(uid)
	if old != nil {
		old.stop()
	}

	c.registry.DeleteOwner(uid)

	return c.publishProbeConfig(ctx)
}

func (c *Controller) removeWorker(uid string) *worker {
	c.mu.Lock()
	defer c.mu.Unlock()

	old := c.workers[uid]
	delete(c.workers, uid)
	return old
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

func (c *Controller) publishProbeConfig(ctx context.Context) error {
	workers := c.snapshotWorkers()

	deviceConfig := &corev1.ClusterConfig_Status_Device{}

	for _, w := range workers {
		if !devicemgrcommon.UsesProbe(w.dm) {
			continue
		}

		ownerRef := umetav1.GetObjectReference(w.dm)
		condition := w.dm.Spec.GetCondition()

		for _, probe := range w.mgr.IdentityProbes() {
			if probe == nil ||
				(probe.RunCommand == nil && probe.ReadFile == nil && probe.ReadRegistry == nil) {
				continue
			}
			deviceConfig.Probes = append(
				deviceConfig.Probes,
				toCoreProbe(probe, ownerRef, condition),
			)
		}
	}

	sort.Slice(deviceConfig.Probes, func(i, j int) bool {
		return probeSortKey(deviceConfig.Probes[i]) < probeSortKey(deviceConfig.Probes[j])
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

func probeSortKey(probe *corev1.ClusterConfig_Status_Device_Probe) string {
	if probe == nil {
		return ""
	}

	key := probe.GetOwnerRef().GetUid() + "|" + probe.GetOsType().String() + "|"

	switch t := probe.GetType().(type) {
	case *corev1.ClusterConfig_Status_Device_Probe_RunCommand_:
		key += "command|" + t.RunCommand.Command + "|" + strings.Join(t.RunCommand.Args, " ")
	case *corev1.ClusterConfig_Status_Device_Probe_ReadFile_:
		key += "file|" + t.ReadFile.Path
	case *corev1.ClusterConfig_Status_Device_Probe_ReadRegistry_:
		key += "registry|" + t.ReadRegistry.Key + "|" + t.ReadRegistry.Name
	default:
		key += "unknown"
	}

	return key
}

func toCoreProbe(
	probe *devicemgrcommon.Probe,
	ownerRef *metav1.ObjectReference,
	condition *corev1.Condition,
) *corev1.ClusterConfig_Status_Device_Probe {
	out := &corev1.ClusterConfig_Status_Device_Probe{
		OwnerRef:         ownerRef,
		OsType:           probe.OSType,
		RequireElevation: probe.RequireElevation,
		Condition:        condition,
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

	dm.Status.Type = w.mgr.Type()
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
