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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type tstNudger struct {
	count atomic.Int32
}

func (n *tstNudger) Nudge() {
	n.count.Add(1)
}

type tstManager struct {
	fleet *devicemgrcommon.Fleet
	err   error
}

func (m *tstManager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_FLEETDM
}

func (m *tstManager) IdentityProbes() []*devicemgrcommon.Probe {
	return nil
}

func (m *tstManager) ParseExternalID(osType corev1.Device_Status_OSType,
	results []*devicemgrcommon.ProbeResult) (string, error) {
	return "", nil
}

func (m *tstManager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	return m.fleet, m.err
}

func (m *tstManager) Close() error {
	return nil
}

type tstResetter struct {
	mu   sync.Mutex
	uids []string
}

func (r *tstResetter) ResetBindingsForOwner(ctx context.Context, ownerUID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uids = append(r.uids, ownerUID)
	return nil
}

func (r *tstResetter) getUIDs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.uids...)
}

type tstEnv struct {
	ctx       context.Context
	ctrl      *Controller
	registry  *devicemgrcommon.Registry
	resetter  *tstResetter
	nudger    *tstNudger
	octeliumC octeliumc.ClientInterface
}

func newTstEnv(t *testing.T) *tstEnv {
	ctx, cancel := context.WithCancel(context.Background())

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)

	registry := devicemgrcommon.NewRegistry()
	resetter := &tstResetter{}
	nudger := &tstNudger{}

	ctrl, err := NewController(ctx, tst.C.OcteliumC, registry, nudger, resetter)
	assert.Nil(t, err)

	t.Cleanup(func() {
		for _, w := range ctrl.snapshotWorkers() {
			ctrl.stopWorker(w.uid)
		}
		cancel()
		tst.Destroy()
	})

	return &tstEnv{
		ctx:       ctx,
		ctrl:      ctrl,
		registry:  registry,
		resetter:  resetter,
		nudger:    nudger,
		octeliumC: tst.C.OcteliumC,
	}
}

func (e *tstEnv) createSecret(t *testing.T, name string) {
	_, err := e.octeliumC.EnterpriseC().CreateSecret(e.ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: name,
		},
		Spec:   &enterprisev1.Secret_Spec{},
		Status: &enterprisev1.Secret_Status{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		},
	})
	assert.Nil(t, err)
}

func (e *tstEnv) createDeviceManager(t *testing.T, secretName string) *enterprisev1.DeviceManager {
	dm, err := e.octeliumC.EnterpriseC().CreateDeviceManager(e.ctx, &enterprisev1.DeviceManager{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.DeviceManager_Spec{
			Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
				FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{
					BaseURL: "https://127.0.0.1:1",
					ApiToken: &enterprisev1.DeviceManager_Spec_SecretRef{
						Type: &enterprisev1.DeviceManager_Spec_SecretRef_FromSecret{
							FromSecret: secretName,
						},
					},
				},
			},
		},
		Status: &enterprisev1.DeviceManager_Status{},
	})
	assert.Nil(t, err)
	return dm
}

func (e *tstEnv) getDeviceManager(t *testing.T, dm *enterprisev1.DeviceManager) *enterprisev1.DeviceManager {
	ret, err := e.octeliumC.EnterpriseC().GetDeviceManager(e.ctx, &rmetav1.GetOptions{Uid: dm.Metadata.Uid})
	assert.Nil(t, err)
	return ret
}

func (e *tstEnv) updateSpec(t *testing.T, dm *enterprisev1.DeviceManager,
	fn func(spec *enterprisev1.DeviceManager_Spec)) (*enterprisev1.DeviceManager, *enterprisev1.DeviceManager) {
	old := e.getDeviceManager(t, dm)
	dm = pbutils.Clone(old).(*enterprisev1.DeviceManager)
	fn(dm.Spec)
	dm, err := e.octeliumC.EnterpriseC().UpdateDeviceManager(e.ctx, dm)
	assert.Nil(t, err)
	return dm, old
}

func (e *tstEnv) getProbeIDs(t *testing.T) []string {
	cc, err := e.octeliumC.CoreV1Utils().GetClusterConfig(e.ctx)
	assert.Nil(t, err)

	var ret []string
	for _, p := range cc.GetStatus().GetDevice().GetProbes() {
		ret = append(ret, p.Id)
	}
	return ret
}

func TestDeviceManagerLifecycle(t *testing.T) {
	env := newTstEnv(t)

	secretName := utilrand.GetRandomStringCanonical(8)
	env.createSecret(t, secretName)

	dm := env.createDeviceManager(t, secretName)
	uid := dm.Metadata.Uid

	{
		assert.Nil(t, env.ctrl.OnAdd(env.ctx, dm))

		w := env.ctrl.getWorker(uid)
		assert.NotNil(t, w)

		owner, ok := env.registry.GetOwner(uid)
		assert.True(t, ok)
		assert.Equal(t, dm.Metadata.Name, owner.Name())

		assert.Equal(t, []string{
			dm.Metadata.Name + ".hardware-uuid",
			dm.Metadata.Name + ".hardware-uuid-linux",
		}, env.getProbeIDs(t))

		assert.Nil(t, env.ctrl.OnAdd(env.ctx, dm))
		assert.True(t, w == env.ctrl.getWorker(uid))

		assert.Nil(t, env.ctrl.Resync(env.ctx))
		assert.True(t, w == env.ctrl.getWorker(uid))
	}

	{
		w := env.ctrl.getWorker(uid)

		updated, old := env.updateSpec(t, dm, func(spec *enterprisev1.DeviceManager_Spec) {
			spec.GetFleetDM().TeamID = 3
		})
		assert.Nil(t, env.ctrl.OnUpdate(env.ctx, updated, old))

		replacement := env.ctrl.getWorker(uid)
		assert.NotNil(t, replacement)
		assert.False(t, w == replacement)
		assert.Equal(t, uint32(3), replacement.dm.Spec.GetFleetDM().TeamID)
		assert.NotNil(t, w.ctx.Err())
	}

	{
		updated, old := env.updateSpec(t, dm, func(spec *enterprisev1.DeviceManager_Spec) {
			spec.Linking = &enterprisev1.DeviceManager_Spec_Linking{
				Strategy: enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY,
			}
		})
		assert.Nil(t, env.ctrl.OnUpdate(env.ctx, updated, old))
		assert.Empty(t, env.getProbeIDs(t))
	}

	{
		updated, old := env.updateSpec(t, dm, func(spec *enterprisev1.DeviceManager_Spec) {
			spec.Linking = nil
			spec.Polling = &enterprisev1.DeviceManager_Spec_Polling{
				IsDisabled: true,
			}
		})
		assert.Nil(t, env.ctrl.OnUpdate(env.ctx, updated, old))

		assert.Nil(t, env.ctrl.getWorker(uid))
		owner, ok := env.registry.GetOwner(uid)
		assert.True(t, ok)
		assert.True(t, devicemgrcommon.IsDisabled(owner.DM))
		assert.Nil(t, owner.Manager)
		assert.Empty(t, env.getProbeIDs(t))
		assert.Equal(t, int32(1), env.nudger.count.Load())

		dm := env.getDeviceManager(t, dm)
		assert.Equal(t, enterprisev1.DeviceManager_Status_DISABLED, dm.Status.State)
		assert.Equal(t, enterprisev1.DeviceManager_Status_FLEETDM, dm.Status.Type)

		assert.Nil(t, env.ctrl.Resync(env.ctx))
		assert.Equal(t, int32(1), env.nudger.count.Load())
	}

	{
		updated, old := env.updateSpec(t, dm, func(spec *enterprisev1.DeviceManager_Spec) {
			spec.Polling = nil
		})
		assert.Nil(t, env.ctrl.OnUpdate(env.ctx, updated, old))
		assert.NotNil(t, env.ctrl.getWorker(uid))
		assert.Len(t, env.getProbeIDs(t), 2)

		owner, ok := env.registry.GetOwner(uid)
		assert.True(t, ok)
		assert.False(t, devicemgrcommon.IsDisabled(owner.DM))
	}

	{
		_, err := env.octeliumC.EnterpriseC().DeleteDeviceManager(env.ctx, &rmetav1.DeleteOptions{Uid: uid})
		assert.Nil(t, err)

		assert.Nil(t, env.ctrl.Resync(env.ctx))

		assert.Nil(t, env.ctrl.getWorker(uid))
		_, ok := env.registry.GetOwner(uid)
		assert.False(t, ok)
		assert.Empty(t, env.getProbeIDs(t))
		assert.Equal(t, []string{uid}, env.resetter.getUIDs())
	}
}

func TestDeviceManagerBuildFailure(t *testing.T) {
	env := newTstEnv(t)

	secretName := utilrand.GetRandomStringCanonical(8)
	dm := env.createDeviceManager(t, secretName)
	uid := dm.Metadata.Uid

	{
		assert.Nil(t, env.ctrl.OnAdd(env.ctx, dm))
		assert.Nil(t, env.ctrl.getWorker(uid))

		owner, ok := env.registry.GetOwner(uid)
		assert.True(t, ok)
		assert.Nil(t, owner.Manager)
		assert.Nil(t, owner.Fleet)

		dm := env.getDeviceManager(t, dm)
		assert.Equal(t, enterprisev1.DeviceManager_Status_ERROR, dm.Status.State)
		assert.Equal(t, enterprisev1.DeviceManager_Status_FLEETDM, dm.Status.Type)
		assert.NotEmpty(t, dm.Status.Collection.LastError)

		assert.Empty(t, env.getProbeIDs(t))
	}

	env.createSecret(t, secretName)

	{
		assert.Nil(t, env.ctrl.Resync(env.ctx))
		assert.NotNil(t, env.ctrl.getWorker(uid))
		assert.Len(t, env.getProbeIDs(t), 2)
	}

	{
		assert.Nil(t, env.ctrl.OnDelete(env.ctx, dm))
		assert.Nil(t, env.ctrl.getWorker(uid))
		assert.Empty(t, env.getProbeIDs(t))
		assert.Equal(t, []string{uid}, env.resetter.getUIDs())
		assert.NotNil(t, env.ctrl.OnDelete(env.ctx, &enterprisev1.DeviceManager{}))
	}

	{
		assert.NotNil(t, env.ctrl.OnAdd(env.ctx, &enterprisev1.DeviceManager{}))
	}
}

func TestDeviceManagerSecretRotation(t *testing.T) {
	env := newTstEnv(t)

	secretName := utilrand.GetRandomStringCanonical(8)
	env.createSecret(t, secretName)

	dm := env.createDeviceManager(t, secretName)
	uid := dm.Metadata.Uid

	assert.Nil(t, env.ctrl.OnAdd(env.ctx, dm))
	w := env.ctrl.getWorker(uid)
	assert.NotNil(t, w)
	assert.NotEmpty(t, w.secretVersion)

	{
		assert.Nil(t, env.ctrl.Resync(env.ctx))
		assert.True(t, w == env.ctrl.getWorker(uid))
	}

	{
		sec, err := env.octeliumC.EnterpriseC().GetSecret(env.ctx, &rmetav1.GetOptions{Name: secretName})
		assert.Nil(t, err)
		sec.Data = &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		}
		_, err = env.octeliumC.EnterpriseC().UpdateSecret(env.ctx, sec)
		assert.Nil(t, err)

		assert.Nil(t, env.ctrl.Resync(env.ctx))

		replacement := env.ctrl.getWorker(uid)
		assert.NotNil(t, replacement)
		assert.False(t, w == replacement)
		assert.NotEqual(t, w.secretVersion, replacement.secretVersion)
		assert.NotNil(t, w.ctx.Err())

		assert.Nil(t, env.ctrl.Resync(env.ctx))
		assert.True(t, replacement == env.ctrl.getWorker(uid))
	}

	{
		_, err := env.octeliumC.EnterpriseC().DeleteSecret(env.ctx, &rmetav1.DeleteOptions{Name: secretName})
		assert.Nil(t, err)

		assert.Nil(t, env.ctrl.Resync(env.ctx))
		assert.Nil(t, env.ctrl.getWorker(uid))

		owner, ok := env.registry.GetOwner(uid)
		assert.True(t, ok)
		assert.Nil(t, owner.Manager)

		dm := env.getDeviceManager(t, dm)
		assert.Equal(t, enterprisev1.DeviceManager_Status_ERROR, dm.Status.State)
		assert.NotEmpty(t, dm.Status.Collection.LastError)
	}

	{
		env.createSecret(t, secretName)
		assert.Nil(t, env.ctrl.Resync(env.ctx))
		assert.NotNil(t, env.ctrl.getWorker(uid))
	}
}

func TestWorkerPoll(t *testing.T) {
	env := newTstEnv(t)

	dm := env.createDeviceManager(t, utilrand.GetRandomStringCanonical(8))
	uid := dm.Metadata.Uid

	fleet := devicemgrcommon.NewFleet([]*devicemgrcommon.Entry{
		{ExternalID: "id-1"},
		{ExternalID: "id-2"},
	})
	mgr := &tstManager{fleet: fleet}
	w := newWorker(env.ctx, env.octeliumC, env.registry, env.nudger, dm, mgr, "")

	{
		w.poll()

		dm := env.getDeviceManager(t, dm)
		assert.Equal(t, enterprisev1.DeviceManager_Status_OK, dm.Status.State)
		assert.Equal(t, enterprisev1.DeviceManager_Status_FLEETDM, dm.Status.Type)
		assert.Empty(t, dm.Status.Collection.LastError)
		assert.Equal(t, uint32(2), dm.Status.Collection.ManagedDevices)
		assert.True(t, dm.Status.Collection.LastSuccessAt.IsValid())
		assert.True(t, dm.Status.Collection.LastAttemptAt.IsValid())

		owner, ok := env.registry.GetOwner(uid)
		assert.True(t, ok)
		assert.True(t, owner.Fresh(time.Now()))
		assert.Equal(t, 2, owner.Fleet.Len())
		assert.Equal(t, int32(1), env.nudger.count.Load())
	}

	{
		fleet.SetDegraded("partial collection")
		w.poll()

		dm := env.getDeviceManager(t, dm)
		assert.Equal(t, enterprisev1.DeviceManager_Status_DEGRADED, dm.Status.State)
		assert.Equal(t, "partial collection", dm.Status.Collection.LastError)
		assert.Equal(t, uint32(2), dm.Status.Collection.ManagedDevices)
		assert.Equal(t, int32(2), env.nudger.count.Load())
	}

	{
		mgr.err = errors.New("collection error")
		w.poll()

		dm := env.getDeviceManager(t, dm)
		assert.Equal(t, enterprisev1.DeviceManager_Status_ERROR, dm.Status.State)
		assert.Equal(t, "collection error", dm.Status.Collection.LastError)
		assert.Equal(t, uint32(2), dm.Status.Collection.ManagedDevices)
		assert.Equal(t, int32(2), env.nudger.count.Load())

		owner, ok := env.registry.GetOwner(uid)
		assert.True(t, ok)
		assert.Equal(t, 2, owner.Fleet.Len())
	}

	{
		w.cancel()
		mgr.err = nil
		w.poll()
		assert.Equal(t, int32(2), env.nudger.count.Load())
	}
}

func TestToCoreProbe(t *testing.T) {
	dm := &enterprisev1.DeviceManager{
		Metadata: &metav1.Metadata{
			Uid:  "dm-uid",
			Name: "dm",
		},
		Spec: &enterprisev1.DeviceManager_Spec{
			Condition: &corev1.Condition{
				Type: &corev1.Condition_MatchAny{
					MatchAny: true,
				},
			},
		},
	}

	assert.Nil(t, toCoreProbe(dm, nil))
	assert.Nil(t, toCoreProbe(dm, &devicemgrcommon.Probe{ID: "probe"}))
	assert.Nil(t, toCoreProbe(dm, &devicemgrcommon.Probe{
		ReadFile: &devicemgrcommon.ReadFile{Path: "/etc/machine-id"},
	}))

	osTypes := []corev1.Device_Status_OSType{corev1.Device_Status_LINUX}

	{
		ret := toCoreProbe(dm, &devicemgrcommon.Probe{
			ID:               "command",
			OSTypes:          osTypes,
			RequireElevation: true,
			RunCommand: &devicemgrcommon.RunCommand{
				Command:        "/usr/bin/tool",
				Args:           []string{"--id"},
				TimeoutSeconds: 5,
				MaxOutputBytes: 1024,
			},
		})
		assert.Equal(t, "dm.command", ret.Id)
		assert.Equal(t, "dm-uid", ret.OwnerRef.Uid)
		assert.Equal(t, osTypes, ret.OsTypes)
		assert.True(t, ret.RequireElevation)
		assert.True(t, ret.Condition.GetMatchAny())
		assert.Equal(t, "/usr/bin/tool", ret.GetRunCommand().Command)
		assert.Equal(t, []string{"--id"}, ret.GetRunCommand().Args)
		assert.Equal(t, uint32(5), ret.GetRunCommand().TimeoutSeconds)
		assert.Equal(t, uint32(1024), ret.GetRunCommand().MaxOutputBytes)

		ret.OsTypes[0] = corev1.Device_Status_MAC
		assert.Equal(t, corev1.Device_Status_LINUX, osTypes[0])
	}

	{
		ret := toCoreProbe(dm, &devicemgrcommon.Probe{
			ID:       "file",
			ReadFile: &devicemgrcommon.ReadFile{Path: "/etc/machine-id", MaxBytes: 64},
		})
		assert.Equal(t, "/etc/machine-id", ret.GetReadFile().Path)
		assert.Equal(t, uint32(64), ret.GetReadFile().MaxBytes)
	}

	{
		ret := toCoreProbe(dm, &devicemgrcommon.Probe{
			ID:           "registry",
			ReadRegistry: &devicemgrcommon.ReadRegistry{Key: `HKLM\SOFTWARE\Vendor`, Name: "ID"},
		})
		assert.Equal(t, `HKLM\SOFTWARE\Vendor`, ret.GetReadRegistry().Key)
		assert.Equal(t, "ID", ret.GetReadRegistry().Name)
	}

	{
		ret := toCoreProbe(dm, &devicemgrcommon.Probe{
			ID: "platform",
			PlatformIdentifier: &devicemgrcommon.PlatformIdentifier{
				Kind: corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
			},
		})
		assert.Equal(t, corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
			ret.GetPlatformIdentifier().Kind)
	}
}

func TestGetStatusType(t *testing.T) {
	for _, tc := range []struct {
		spec *enterprisev1.DeviceManager_Spec
		typ  enterprisev1.DeviceManager_Status_Type
	}{
		{&enterprisev1.DeviceManager_Spec{}, enterprisev1.DeviceManager_Status_TYPE_UNKNOWN},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_CrowdStrike_{
			CrowdStrike: &enterprisev1.DeviceManager_Spec_CrowdStrike{},
		}}, enterprisev1.DeviceManager_Status_CROWDSTRIKE},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_SentinelOne_{
			SentinelOne: &enterprisev1.DeviceManager_Spec_SentinelOne{},
		}}, enterprisev1.DeviceManager_Status_SENTINELONE},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_MicrosoftIntune_{
			MicrosoftIntune: &enterprisev1.DeviceManager_Spec_MicrosoftIntune{},
		}}, enterprisev1.DeviceManager_Status_MICROSOFT_INTUNE},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_Jamf_{
			Jamf: &enterprisev1.DeviceManager_Spec_Jamf{},
		}}, enterprisev1.DeviceManager_Status_JAMF_PRO},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_OnePassword_{
			OnePassword: &enterprisev1.DeviceManager_Spec_OnePassword{},
		}}, enterprisev1.DeviceManager_Status_ONEPASSWORD},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
			FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{},
		}}, enterprisev1.DeviceManager_Status_FLEETDM},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_Huntress_{
			Huntress: &enterprisev1.DeviceManager_Spec_Huntress{},
		}}, enterprisev1.DeviceManager_Status_HUNTRESS},
		{&enterprisev1.DeviceManager_Spec{Type: &enterprisev1.DeviceManager_Spec_Iru_{
			Iru: &enterprisev1.DeviceManager_Spec_Iru{},
		}}, enterprisev1.DeviceManager_Status_IRU},
	} {
		assert.Equal(t, tc.typ, getStatusType(&enterprisev1.DeviceManager{Spec: tc.spec}))
	}
}

func TestPollOptions(t *testing.T) {
	dm := &enterprisev1.DeviceManager{
		Spec: &enterprisev1.DeviceManager_Spec{},
	}
	assert.Equal(t, defaultPollInterval, pollInterval(dm))
	assert.Equal(t, defaultPollTimeout, pollTimeout(dm))

	dm.Spec.Polling = &enterprisev1.DeviceManager_Spec_Polling{
		Interval: &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 5}},
		Timeout:  &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 45}},
	}
	assert.Equal(t, minPollInterval, pollInterval(dm))
	assert.Equal(t, 45*time.Second, pollTimeout(dm))
}
