// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package devicemanager

import (
	"context"
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
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultPollInterval = 5 * time.Minute
	minPollInterval     = 30 * time.Second
	defaultPollTimeout  = 2 * time.Minute
)

type Owner struct {
	Manager devicemgrcommon.Manager
	DM      *enterprisev1.DeviceManager
}

type Registry struct {
	mu sync.RWMutex
	m  map[string]*Owner
}

func NewRegistry() *Registry {
	return &Registry{m: map[string]*Owner{}}
}

func (r *Registry) set(uid string, o *Owner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[uid] = o
}

func (r *Registry) delete(uid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.m, uid)
}

func (r *Registry) Get(uid string) (*Owner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.m[uid]
	return o, ok
}

func (r *Registry) snapshot() []*Owner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Owner, 0, len(r.m))
	for _, o := range r.m {
		out = append(out, o)
	}
	return out
}

type Controller struct {
	octeliumC octeliumc.ClientInterface
	ctx       context.Context
	registry  *Registry

	mu      sync.Mutex
	workers map[string]*worker
}

func NewController(ctx context.Context, octeliumC octeliumc.ClientInterface, registry *Registry) (*Controller, error) {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Controller{
		octeliumC: octeliumC,
		ctx:       ctx,
		registry:  registry,
		workers:   map[string]*worker{},
	}, nil
}

func (c *Controller) OnAdd(ctx context.Context, dm *enterprisev1.DeviceManager) error {
	return c.sync(ctx, dm)
}

func (c *Controller) OnUpdate(ctx context.Context, dm, old *enterprisev1.DeviceManager) error {
	return c.sync(ctx, dm)
}

func (c *Controller) OnDelete(ctx context.Context, dm *enterprisev1.DeviceManager) error {
	c.mu.Lock()
	if w, ok := c.workers[dm.Metadata.Uid]; ok {
		w.stop()
		delete(c.workers, dm.Metadata.Uid)
	}
	c.mu.Unlock()
	c.registry.delete(dm.Metadata.Uid)
	return c.publishProbeConfig(ctx)
}

func (c *Controller) sync(ctx context.Context, dm *enterprisev1.DeviceManager) error {
	if p := dm.Spec.GetPolling(); p != nil && p.IsDisabled {
		return c.OnDelete(ctx, dm)
	}

	key := dm.Metadata.Uid

	c.mu.Lock()
	if w, ok := c.workers[key]; ok {
		if proto.Equal(w.dm.Spec, dm.Spec) {
			w.dm = dm
			c.mu.Unlock()
			c.registry.set(key, &Owner{Manager: w.mgr, DM: dm})
			return c.publishProbeConfig(ctx)
		}
		w.stop()
		delete(c.workers, key)
	}
	c.mu.Unlock()

	mgr, err := buildManager(c.ctx, c.octeliumC, dm)
	if err != nil {
		return errors.Wrap(err, "Could not build DeviceManager")
	}
	c.registry.set(key, &Owner{Manager: mgr, DM: dm})

	w := newWorker(c.ctx, c.octeliumC, dm, mgr)
	c.mu.Lock()
	c.workers[key] = w
	c.mu.Unlock()
	go w.run()

	zap.L().Info("Started DeviceManager worker",
		zap.String("name", dm.Metadata.Name), zap.Duration("interval", w.interval))

	return c.publishProbeConfig(ctx)
}

func buildManager(ctx context.Context, octeliumC octeliumc.ClientInterface, dm *enterprisev1.DeviceManager) (devicemgrcommon.Manager, error) {
	opts := &devicemgrcommon.ManagerOpts{DeviceManager: dm, OcteliumC: octeliumC}
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
		return nil, errors.Errorf("Unsupported DeviceManager type: %s", dm.Metadata.Name)
	}
}

func (c *Controller) publishProbeConfig(ctx context.Context) error {
	dev := &corev1.ClusterConfig_Status_Device{}
	for _, o := range c.registry.snapshot() {
		ownerRef := umetav1.GetObjectReference(o.DM)
		condition := o.DM.Spec.GetCondition()
		for _, p := range o.Manager.IdentityProbes() {
			dev.Probes = append(dev.Probes, toCoreProbe(p, ownerRef, condition))
		}
	}

	cc, err := c.octeliumC.CoreV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return err
	}
	if cc.Status == nil {
		cc.Status = &corev1.ClusterConfig_Status{}
	}
	if proto.Equal(cc.Status.Device, dev) {
		return nil
	}
	cc.Status.Device = dev
	if _, err := c.octeliumC.CoreC().UpdateClusterConfig(ctx, cc); err != nil {
		return errors.Wrap(err, "Could not publish device probe config")
	}
	return nil
}

func toCoreProbe(p *devicemgrcommon.Probe, ownerRef *metav1.ObjectReference, condition *corev1.Condition) *corev1.ClusterConfig_Status_Device_Probe {
	cp := &corev1.ClusterConfig_Status_Device_Probe{
		OwnerRef:         ownerRef,
		OsType:           p.OSType,
		RequireElevation: p.RequireElevation,
		Condition:        condition,
	}
	switch {
	case p.RunCommand != nil:
		cp.Type = &corev1.ClusterConfig_Status_Device_Probe_RunCommand_{
			RunCommand: &corev1.ClusterConfig_Status_Device_Probe_RunCommand{
				Command:        p.RunCommand.Command,
				Args:           p.RunCommand.Args,
				TimeoutSeconds: p.RunCommand.TimeoutSeconds,
				MaxOutputBytes: p.RunCommand.MaxOutputBytes,
			},
		}
	case p.ReadFile != nil:
		cp.Type = &corev1.ClusterConfig_Status_Device_Probe_ReadFile_{
			ReadFile: &corev1.ClusterConfig_Status_Device_Probe_ReadFile{
				Path:     p.ReadFile.Path,
				MaxBytes: p.ReadFile.MaxBytes,
			},
		}
	case p.ReadRegistry != nil:
		cp.Type = &corev1.ClusterConfig_Status_Device_Probe_ReadRegistry_{
			ReadRegistry: &corev1.ClusterConfig_Status_Device_Probe_ReadRegistry{
				Key:  p.ReadRegistry.Key,
				Name: p.ReadRegistry.Name,
			},
		}
	}
	return cp
}

type worker struct {
	ctx       context.Context
	cancel    context.CancelFunc
	octeliumC octeliumc.ClientInterface
	dm        *enterprisev1.DeviceManager
	mgr       devicemgrcommon.Manager
	interval  time.Duration
	timeout   time.Duration
}

func newWorker(ctx context.Context, octeliumC octeliumc.ClientInterface, dm *enterprisev1.DeviceManager, mgr devicemgrcommon.Manager) *worker {
	wctx, cancel := context.WithCancel(ctx)
	return &worker{
		ctx:       wctx,
		cancel:    cancel,
		octeliumC: octeliumC,
		dm:        dm,
		mgr:       mgr,
		interval:  pollInterval(dm),
		timeout:   pollTimeout(dm),
	}
}

func (w *worker) stop() {
	w.cancel()
	_ = w.mgr.Close()
}

func (w *worker) run() {
	t := time.NewTicker(w.interval)
	defer t.Stop()

	w.poll()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-t.C:
			w.poll()
		}
	}
}

func (w *worker) poll() {
	ctx, cancel := context.WithTimeout(w.ctx, w.timeout)
	defer cancel()

	now := timestamppb.Now()

	fleet, err := w.mgr.Collect(ctx)
	if err != nil {
		zap.L().Warn("Could not collect from DeviceManager",
			zap.String("name", w.dm.Metadata.Name), zap.Error(err))
		w.setStatus(enterprisev1.DeviceManager_Status_ERROR, now, 0, 0, err.Error())
		return
	}

	devList, err := w.octeliumC.CoreC().ListDevice(ctx, &rmetav1.ListOptions{})
	if err != nil {
		zap.L().Warn("Could not list Devices", zap.Error(err))
		return
	}

	ownerRef := umetav1.GetObjectReference(w.dm)
	linked := 0
	for _, dev := range devList.Items {
		if devicemgrcommon.UpsertPosture(dev, w.dm, fleet, now) {
			if _, err := w.octeliumC.CoreC().UpdateDevice(ctx, dev); err != nil {
				zap.L().Warn("Could not update Device",
					zap.String("device", dev.Metadata.Name), zap.Error(err))
				continue
			}
		}
		if p := dev.Status.GetPosture(); p != nil && devicemgrcommon.RefUIDEqual(p.OwnerRef, ownerRef) {
			linked++
		}
	}

	w.setStatus(enterprisev1.DeviceManager_Status_OK, now, uint32(len(fleet.Entries)), uint32(linked), "")
}

func (w *worker) setStatus(state enterprisev1.DeviceManager_Status_State, now *timestamppb.Timestamp, managed, linked uint32, errMsg string) {
	dm, err := w.octeliumC.EnterpriseC().GetDeviceManager(w.ctx, &rmetav1.GetOptions{Uid: w.dm.Metadata.Uid})
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
	col := dm.Status.Collection
	col.LastAttemptAt = now
	col.State = state
	col.LastError = errMsg
	if state == enterprisev1.DeviceManager_Status_OK {
		col.LastSuccessAt = now
		col.ManagedDevices = managed
		col.LinkedDevices = linked
	}
	if _, err := w.octeliumC.EnterpriseC().UpdateDeviceManager(w.ctx, dm); err != nil {
		zap.L().Warn("Could not update DeviceManager status", zap.Error(err))
	}
}

func pollInterval(dm *enterprisev1.DeviceManager) time.Duration {
	d := defaultPollInterval
	if p := dm.Spec.GetPolling(); p != nil && p.Interval != nil {
		if v := umetav1.ToDuration(p.Interval).ToGo(); v > 0 {
			d = v
		}
	}
	if d < minPollInterval {
		d = minPollInterval
	}
	return d
}

func pollTimeout(dm *enterprisev1.DeviceManager) time.Duration {
	if p := dm.Spec.GetPolling(); p != nil && p.Timeout != nil {
		if v := umetav1.ToDuration(p.Timeout).ToGo(); v > 0 {
			return v
		}
	}
	return defaultPollTimeout
}
