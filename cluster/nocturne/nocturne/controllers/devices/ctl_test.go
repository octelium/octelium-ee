// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package devcontroller

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type tstManager struct{}

func (m *tstManager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_CROWDSTRIKE
}

func (m *tstManager) IdentityProbes() []*devicemgrcommon.Probe {
	return nil
}

func (m *tstManager) ParseExternalID(osType corev1.Device_Status_OSType, results []*devicemgrcommon.ProbeResult) (string, error) {
	for _, r := range results {
		if r.Text == "invalid" {
			return "", errors.New("invalid output")
		}
		if r.Text != "" {
			return strings.TrimSpace(r.Text), nil
		}
	}
	return "", nil
}

func (m *tstManager) Collect(ctx context.Context) (*devicemgrcommon.Fleet, error) {
	return nil, nil
}

func (m *tstManager) Close() error {
	return nil
}

type tstEvaluator struct{}

func (e *tstEvaluator) MatchesDevice(ctx context.Context, condition *corev1.Condition, dev *corev1.Device, usr *corev1.User) (bool, error) {
	switch condition.GetMatch() {
	case "error":
		return false, errors.New("condition error")
	case "false":
		return false, nil
	case "has-user":
		return usr != nil && usr.Metadata.Uid == dev.Status.UserRef.GetUid(), nil
	default:
		return true, nil
	}
}

type countingCore struct {
	rcorev1.ResourceServiceClient
	getDevice  atomic.Int32
	getUser    atomic.Int32
	getUserErr atomic.Int32
}

func (c *countingCore) GetDevice(ctx context.Context, req *rmetav1.GetOptions, opts ...grpc.CallOption) (*corev1.Device, error) {
	c.getDevice.Add(1)
	return c.ResourceServiceClient.GetDevice(ctx, req, opts...)
}

func (c *countingCore) GetUser(ctx context.Context, req *rmetav1.GetOptions, opts ...grpc.CallOption) (*corev1.User, error) {
	c.getUser.Add(1)
	if c.getUserErr.Load() > 0 {
		c.getUserErr.Add(-1)
		return nil, status.Error(codes.Unavailable, "unavailable")
	}
	return c.ResourceServiceClient.GetUser(ctx, req, opts...)
}

type countingEnterpriseUtils struct {
	octeliumc.EnterpriseV1Utils
	getClusterConfig atomic.Int32
}

func (c *countingEnterpriseUtils) GetClusterConfig(ctx context.Context) (*enterprisev1.ClusterConfig, error) {
	c.getClusterConfig.Add(1)
	return c.EnterpriseV1Utils.GetClusterConfig(ctx)
}

type countingClient struct {
	octeliumc.ClientInterface
	core            *countingCore
	enterpriseUtils *countingEnterpriseUtils
}

func (c *countingClient) CoreC() rcorev1.ResourceServiceClient {
	return c.core
}

func (c *countingClient) EnterpriseV1Utils() octeliumc.EnterpriseV1Utils {
	return c.enterpriseUtils
}

func (e *tstEnv) useCountingClient() *countingClient {
	ret := &countingClient{
		ClientInterface: e.octeliumC,
		core: &countingCore{
			ResourceServiceClient: e.octeliumC.CoreC(),
		},
		enterpriseUtils: &countingEnterpriseUtils{
			EnterpriseV1Utils: e.octeliumC.EnterpriseV1Utils(),
		},
	}
	e.ctrl.octeliumC = ret
	return ret
}

type tstEnv struct {
	ctx       context.Context
	ctrl      *Controller
	registry  *devicemgrcommon.Registry
	octeliumC octeliumc.ClientInterface
}

func newTstEnv(t *testing.T) *tstEnv {
	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	registry := devicemgrcommon.NewRegistry()

	return &tstEnv{
		ctx:       context.Background(),
		ctrl:      NewController(tst.C.OcteliumC, registry, &tstEvaluator{}),
		registry:  registry,
		octeliumC: tst.C.OcteliumC,
	}
}

func (e *tstEnv) setOwner(name string, spec *enterprisev1.DeviceManager_Spec, entries ...*devicemgrcommon.Entry) *devicemgrcommon.Owner {
	return e.setOwnerAt(name, spec, time.Now(), entries...)
}

func (e *tstEnv) setOwnerAt(name string, spec *enterprisev1.DeviceManager_Spec,
	collectedAt time.Time, entries ...*devicemgrcommon.Entry) *devicemgrcommon.Owner {
	if spec == nil {
		spec = &enterprisev1.DeviceManager_Spec{}
	}

	owner := devicemgrcommon.NewOwner(&tstManager{}, tstDeviceManager(name, spec),
		devicemgrcommon.NewFleet(entries), collectedAt, time.Hour)
	e.registry.SetOwner(owner)
	return owner
}

func (e *tstEnv) setDeviceManagers(t *testing.T, names ...string) {
	cc, err := e.octeliumC.EnterpriseV1Utils().GetClusterConfig(e.ctx)
	assert.Nil(t, err)
	cc.Spec.DeviceManagers = names
	_, err = e.octeliumC.EnterpriseC().UpdateClusterConfig(e.ctx, cc)
	assert.Nil(t, err)
}

func (e *tstEnv) createDevice(t *testing.T, serial, email string) *corev1.Device {
	usr, err := e.octeliumC.CoreC().CreateUser(e.ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type:  corev1.User_Spec_HUMAN,
			Email: email,
		},
		Status: &corev1.User_Status{},
	})
	assert.Nil(t, err)

	dev, err := e.octeliumC.CoreC().CreateDevice(e.ctx, &corev1.Device{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.Device_Spec{
			State: corev1.Device_Spec_ACTIVE,
		},
		Status: &corev1.Device_Status{
			UserRef:      umetav1.GetObjectReference(usr),
			OsType:       corev1.Device_Status_LINUX,
			SerialNumber: serial,
			MacAddresses: []string{"a4:bb:cc:dd:ee:ff"},
		},
	})
	assert.Nil(t, err)

	return dev
}

func (e *tstEnv) createUserDevice(t *testing.T, usr *corev1.User, serial string) *corev1.Device {
	dev, err := e.octeliumC.CoreC().CreateDevice(e.ctx, &corev1.Device{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.Device_Spec{
			State: corev1.Device_Spec_ACTIVE,
		},
		Status: &corev1.Device_Status{
			UserRef:      umetav1.GetObjectReference(usr),
			OsType:       corev1.Device_Status_LINUX,
			SerialNumber: serial,
		},
	})
	assert.Nil(t, err)

	return dev
}

func (e *tstEnv) createDeletedDevice(t *testing.T) *corev1.Device {
	dev := e.createDevice(t, "", "user@example.com")
	_, err := e.octeliumC.CoreC().DeleteDevice(e.ctx, &rmetav1.DeleteOptions{Uid: dev.Metadata.Uid})
	assert.Nil(t, err)
	return dev
}

func (e *tstEnv) getDevice(t *testing.T, dev *corev1.Device) *corev1.Device {
	ret, err := e.octeliumC.CoreC().GetDevice(e.ctx, &rmetav1.GetOptions{Uid: dev.Metadata.Uid})
	assert.Nil(t, err)
	return ret
}

func (e *tstEnv) updateDevice(t *testing.T, dev *corev1.Device, fn func(dev *corev1.Device)) *corev1.Device {
	dev = e.getDevice(t, dev)
	fn(dev)
	ret, err := e.octeliumC.CoreC().UpdateDevice(e.ctx, dev)
	assert.Nil(t, err)
	return ret
}

func (e *tstEnv) reconcile(t *testing.T, dev *corev1.Device) *corev1.Device {
	err := e.ctrl.ReconcileDevice(e.ctx, dev)
	assert.Nil(t, err, "%+v", err)
	return e.getDevice(t, dev)
}

func tstDeviceManager(name string, spec *enterprisev1.DeviceManager_Spec) *enterprisev1.DeviceManager {
	return &enterprisev1.DeviceManager{
		Metadata: &metav1.Metadata{
			Uid:  fmt.Sprintf("uid-%s", name),
			Name: name,
		},
		Spec: spec,
	}
}

func tstEntry(externalID, serial string, aliases ...string) *devicemgrcommon.Entry {
	return &devicemgrcommon.Entry{
		ExternalID:  externalID,
		Aliases:     aliases,
		Serial:      serial,
		OwnerEmails: []string{"owner@example.com"},
		Posture: &corev1.Device_Status_Posture{
			Compliant: corev1.Device_Status_Posture_PASS,
		},
	}
}

func tstAttempt(owner *devicemgrcommon.Owner, state corev1.Device_Status_ProbeAttempt_State,
	startedAt time.Time, results ...*corev1.Device_Status_ProbeAttempt_Result) *corev1.Device_Status_ProbeAttempt {
	return &corev1.Device_Status_ProbeAttempt{
		Uid:       utilrand.GetRandomStringCanonical(32),
		State:     state,
		StartedAt: pbutils.Timestamp(startedAt),
		ExpiresAt: pbutils.Timestamp(startedAt.Add(10 * time.Minute)),
		Probes: []*corev1.ClusterConfig_Status_Device_Probe{
			{
				Id:       fmt.Sprintf("%s.probe", owner.Name()),
				OwnerRef: owner.OwnerRef(),
				Type: &corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_{
					PlatformIdentifier: &corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier{
						Kind: corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
					},
				},
			},
		},
		Results: results,
	}
}

func tstResult(owner *devicemgrcommon.Owner, status corev1.Device_Status_ProbeAttempt_Result_Status, text string) *corev1.Device_Status_ProbeAttempt_Result {
	return &corev1.Device_Status_ProbeAttempt_Result{
		ProbeID: fmt.Sprintf("%s.probe", owner.Name()),
		Status:  status,
		Value: &corev1.Device_Status_ProbeAttempt_Result_Text{
			Text: text,
		},
	}
}

func TestReconcileDeviceSelection(t *testing.T) {
	env := newTstEnv(t)

	dev1 := env.createDevice(t, "SERIAL-1", "user1@example.com")
	dev2 := env.createDevice(t, "SERIAL-1", "user2@example.com")
	dev3 := env.createDevice(t, "SERIAL-3", "user3@example.com")

	{
		dev := env.reconcile(t, dev1)
		assert.Nil(t, dev.Status.Binding)
		assert.Nil(t, dev.Status.Posture)
	}

	ownerA := env.setOwner("dm-a", nil, tstEntry("ext-1", "serial-1"))

	{
		dev := env.reconcile(t, dev1)
		binding := dev.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, binding.State)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, binding.Validity)
		assert.Equal(t, "ext-1", binding.ExternalID)
		assert.Equal(t, ownerA.UID(), binding.OwnerRef.Uid)
		assert.NotEmpty(t, binding.Uid)
		assert.True(t, binding.AcceptedAt.IsValid())
		assert.True(t, binding.LastVerifiedAt.IsValid())
		assert.Nil(t, binding.NextVerificationAt)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, dev.Status.Posture.Compliant)
		assert.True(t, dev.Status.Posture.ExpiresAt.IsValid())

		assert.Equal(t, dev.Status.Binding.Uid, env.reconcile(t, dev1).Status.Binding.Uid)
	}

	var conflictUID string
	{
		dev := env.reconcile(t, dev2)
		binding := dev.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_CONFLICT, binding.State)
		assert.Equal(t, reasonConflict, binding.Reason)
		assert.Contains(t, binding.Message, dev1.Metadata.Name)
		assert.Equal(t, "ext-1", binding.ExternalID)
		assert.Equal(t, ownerA.UID(), binding.OwnerRef.Uid)
		assert.Nil(t, dev.Status.Posture)
		conflictUID = binding.Uid

		assert.Equal(t, conflictUID, env.reconcile(t, dev2).Status.Binding.Uid)
	}

	{
		dev := env.reconcile(t, dev3)
		assert.Nil(t, dev.Status.Binding)
	}

	{
		bindingUID := env.getDevice(t, dev1).Status.Binding.Uid

		assert.Nil(t, env.ctrl.ResetDeviceBinding(env.ctx, dev1.Metadata.Uid, utilrand.GetRandomStringCanonical(32)))
		assert.Equal(t, bindingUID, env.getDevice(t, dev1).Status.Binding.Uid)

		assert.Nil(t, env.ctrl.ResetDeviceBinding(env.ctx, dev1.Metadata.Uid, bindingUID))
		dev := env.getDevice(t, dev1)
		assert.Nil(t, dev.Status.Binding)
		assert.Nil(t, dev.Status.Posture)

		assert.Nil(t, env.ctrl.ResetDeviceBinding(env.ctx, dev1.Metadata.Uid, bindingUID))
		assert.Nil(t, env.ctrl.ResetDeviceBinding(env.ctx, env.createDeletedDevice(t).Metadata.Uid, ""))
		assert.NotNil(t, env.ctrl.ResetDeviceBinding(env.ctx, "", ""))
	}

	{
		dev := env.reconcile(t, dev2)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
		assert.NotEqual(t, conflictUID, dev.Status.Binding.Uid)

		dev = env.reconcile(t, dev1)
		assert.Equal(t, corev1.Device_Status_Binding_CONFLICT, dev.Status.Binding.State)
	}

	env.registry.DeleteOwner(ownerA.UID())

	{
		dev := env.reconcile(t, dev1)
		assert.Nil(t, dev.Status.Binding)

		dev = env.reconcile(t, dev2)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
	}

	{
		assert.Nil(t, env.ctrl.ReconcileDevice(env.ctx, nil))
		assert.NotNil(t, env.ctrl.ReconcileDevice(env.ctx, &corev1.Device{Metadata: &metav1.Metadata{}}))
		assert.Nil(t, env.ctrl.ReconcileDevice(env.ctx, env.createDeletedDevice(t)))
	}
}

func TestReconcileDeviceOrder(t *testing.T) {
	env := newTstEnv(t)

	ownerA := env.setOwner("dm-a", nil, tstEntry("a-1", "serial-1"), tstEntry("a-2", "serial-2"),
		tstEntry("a-3", "serial-3"), tstEntry("a-4", "serial-4"), tstEntry("a-5", "serial-5"))
	ownerB := env.setOwner("dm-b", nil, tstEntry("b-1", "serial-1"), tstEntry("b-2", "serial-2"),
		tstEntry("b-3", "serial-3"), tstEntry("b-4", "serial-4"), tstEntry("b-5", "serial-5"))

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user@example.com"))
		assert.Equal(t, ownerA.UID(), dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, "a-1", dev.Status.Binding.ExternalID)
	}

	env.setDeviceManagers(t, "dm-b", "dm-a")

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-2", "user@example.com"))
		assert.Equal(t, ownerB.UID(), dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, "b-2", dev.Status.Binding.ExternalID)
	}

	env.setDeviceManagers(t, "dm-a")

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-B-ONLY", "user@example.com"))
		assert.Nil(t, dev.Status.Binding)
	}

	env.setDeviceManagers(t, "dm-unknown", "dm-b")

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-3", "user@example.com"))
		assert.Equal(t, ownerB.UID(), dev.Status.Binding.OwnerRef.Uid)
	}

	env.setDeviceManagers(t)

	{
		env.registry.SetOwner(devicemgrcommon.NewPendingOwner(tstDeviceManager("dm-0", &enterprisev1.DeviceManager_Spec{})))

		dev := env.reconcile(t, env.createDevice(t, "SERIAL-4", "user@example.com"))
		assert.Nil(t, dev.Status.Binding)

		env.setOwnerAt("dm-0", nil, time.Now().Add(-2*time.Hour))
		dev = env.reconcile(t, dev)
		assert.Nil(t, dev.Status.Binding)

		env.setOwner("dm-0", &enterprisev1.DeviceManager_Spec{
			Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "error"}},
		})
		dev = env.reconcile(t, dev)
		assert.Nil(t, dev.Status.Binding)

		env.setOwner("dm-0", &enterprisev1.DeviceManager_Spec{
			Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "false"}},
		}, tstEntry("zero-4", "serial-4"))
		dev = env.reconcile(t, dev)
		assert.Equal(t, ownerA.UID(), dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, "a-4", dev.Status.Binding.ExternalID)
	}

	{
		inapplicable := &enterprisev1.DeviceManager_Spec{
			Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "false"}},
		}

		env.registry.SetOwner(devicemgrcommon.NewPendingOwner(tstDeviceManager("dm-0", inapplicable)))
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user@example.com"))
		assert.Equal(t, ownerA.UID(), dev.Status.Binding.OwnerRef.Uid)

		env.setOwnerAt("dm-0", inapplicable, time.Now().Add(-2*time.Hour), tstEntry("zero-2", "serial-2"))
		dev = env.reconcile(t, env.createDevice(t, "SERIAL-2", "user@example.com"))
		assert.Equal(t, ownerA.UID(), dev.Status.Binding.OwnerRef.Uid)

		env.registry.SetOwner(devicemgrcommon.NewPendingOwner(tstDeviceManager("dm-0", &enterprisev1.DeviceManager_Spec{
			Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "error"}},
		})))
		dev = env.reconcile(t, env.createDevice(t, "SERIAL-3", "user@example.com"))
		assert.Nil(t, dev.Status.Binding)
	}

	{
		env.setOwner("dm-0", nil, tstEntry("zero-5a", "serial-5"), tstEntry("zero-5b", "serial-5"))

		dev := env.reconcile(t, env.createDevice(t, "SERIAL-5", "user@example.com"))
		binding := dev.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_AMBIGUOUS, binding.State)
		assert.Equal(t, reasonAmbiguous, binding.Reason)
		assert.Equal(t, "uid-dm-0", binding.OwnerRef.Uid)
		assert.Empty(t, binding.ExternalID)

		ambiguousUID := binding.Uid
		assert.Equal(t, ambiguousUID, env.reconcile(t, dev).Status.Binding.Uid)

		env.setOwner("dm-0", nil, tstEntry("zero-5a", "serial-5"))
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
		assert.Equal(t, "zero-5a", dev.Status.Binding.ExternalID)
		assert.NotEqual(t, ambiguousUID, dev.Status.Binding.Uid)
	}
}

func TestReconcileDeviceOwnerMatch(t *testing.T) {
	env := newTstEnv(t)

	env.setOwner("dm-a", &enterprisev1.DeviceManager_Spec{
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			RequireOwnerMatch: true,
		},
	}, tstEntry("a-1", "serial-1"), tstEntry("a-2", "serial-2"))
	ownerB := env.setOwner("dm-b", nil, tstEntry("b-1", "serial-1"))

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user@example.com"))
		assert.Equal(t, ownerB.UID(), dev.Status.Binding.OwnerRef.Uid)
	}

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-2", "Owner@Example.com"))
		assert.Equal(t, "uid-dm-a", dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, "a-2", dev.Status.Binding.ExternalID)
	}
}

func TestReconcileDeviceProbe(t *testing.T) {
	env := newTstEnv(t)
	now := time.Now()

	owner := env.setOwner("dm-p", &enterprisev1.DeviceManager_Spec{
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			Strategy: enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY,
		},
	}, tstEntry("p-1", "serial-1", "hw-uuid-1"), tstEntry("p-2", "serial-2", "hw-uuid-2"))

	{
		dev := env.createDevice(t, "SERIAL-1", "user@example.com")
		dev = env.reconcile(t, dev)
		assert.Nil(t, dev.Status.Binding)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, now,
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_PERMISSION_REQUIRED, "hw-uuid-2"))
		})

		dev = env.reconcile(t, dev)
		assert.Nil(t, dev.Status.Binding)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, dev.Status.ProbeAttempt.State)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, now,
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, " HW-UUID-2 "))
		})

		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
		assert.Equal(t, "p-2", dev.Status.Binding.ExternalID)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, dev.Status.ProbeAttempt.State)
	}

	{
		dev := env.createDevice(t, "SERIAL-1", "user@example.com")
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			attempt := tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, now,
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-uuid-1"))
			attempt.Results[0].IsTruncated = true
			dev.Status.ProbeAttempt = attempt
		})

		dev = env.reconcile(t, dev)
		assert.Nil(t, dev.Status.Binding)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, now,
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "invalid"))
		})
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_AMBIGUOUS, dev.Status.Binding.State)
		assert.Equal(t, reasonSourcesDisagree, dev.Status.Binding.Reason)
		assert.Contains(t, dev.Status.Binding.Message, "invalid output")
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, dev.Status.ProbeAttempt.State)
	}
}

func TestReconcileDeviceDisabled(t *testing.T) {
	env := newTstEnv(t)

	disabledSpec := func() *enterprisev1.DeviceManager_Spec {
		return &enterprisev1.DeviceManager_Spec{
			Polling: &enterprisev1.DeviceManager_Spec_Polling{
				IsDisabled: true,
			},
		}
	}

	env.setOwner("dm-a", nil, tstEntry("a-1", "serial-1"), tstEntry("a-2", "serial-2"))
	ownerB := env.setOwner("dm-b", nil, tstEntry("b-2", "serial-2"))
	env.setDeviceManagers(t, "dm-a", "dm-b")

	dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user@example.com"))
	bindingUID := dev.Status.Binding.Uid
	assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)
	assert.NotNil(t, dev.Status.Posture)

	env.registry.SetOwner(devicemgrcommon.NewPendingOwner(tstDeviceManager("dm-a", disabledSpec())))

	{
		dev := env.reconcile(t, dev)
		binding := dev.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, binding.State)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, binding.Validity)
		assert.Equal(t, reasonDisabled, binding.Reason)
		assert.Equal(t, bindingUID, binding.Uid)
		assert.Nil(t, dev.Status.Posture)

		resourceVersion := dev.Metadata.ResourceVersion
		assert.Equal(t, resourceVersion, env.reconcile(t, dev).Metadata.ResourceVersion)
	}

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-2", "user@example.com"))
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
		assert.Equal(t, ownerB.UID(), dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, "b-2", dev.Status.Binding.ExternalID)
	}

	env.setOwner("dm-a", nil, tstEntry("a-1", "serial-1"), tstEntry("a-2", "serial-2"))

	{
		dev := env.reconcile(t, dev)
		binding := dev.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_VALID, binding.Validity)
		assert.Empty(t, binding.Reason)
		assert.Equal(t, bindingUID, binding.Uid)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, dev.Status.Posture.Compliant)
	}
}

func TestReconcileDeviceAgreement(t *testing.T) {
	env := newTstEnv(t)
	now := time.Now()

	owner := env.setOwner("dm-a", &enterprisev1.DeviceManager_Spec{
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			RequireAgreement: true,
		},
	}, tstEntry("a-1", "serial-1", "hw-uuid-1"), tstEntry("a-2", "serial-2", "hw-uuid-2"))
	ownerB := env.setOwner("dm-b", nil, tstEntry("b-1", "serial-1"))

	{
		dev := env.createDevice(t, "SERIAL-1", "user@example.com")

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_ISSUED, now)
		})
		dev = env.reconcile(t, dev)
		assert.Nil(t, dev.Status.Binding)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, now,
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-uuid-2"))
		})
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_AMBIGUOUS, dev.Status.Binding.State)
		assert.Equal(t, reasonSourcesDisagree, dev.Status.Binding.Reason)
		assert.Equal(t, owner.UID(), dev.Status.Binding.OwnerRef.Uid)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, now,
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-uuid-1"))
		})
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
		assert.Equal(t, "a-1", dev.Status.Binding.ExternalID)
	}

	{
		dev := env.createDevice(t, "SERIAL-1", "user@example.com")
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_ISSUED, now.Add(-time.Hour))
		})

		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
		assert.Equal(t, ownerB.UID(), dev.Status.Binding.OwnerRef.Uid)
	}
}

func TestReconcileDeviceAccepted(t *testing.T) {
	env := newTstEnv(t)

	spec := &enterprisev1.DeviceManager_Spec{
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			RequireOwnerMatch: true,
		},
	}

	env.setOwner("dm-a", spec, tstEntry("a-1", "serial-1"))

	dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "owner@example.com"))
	bindingUID := dev.Status.Binding.Uid
	assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)

	{
		env.registry.DeleteOwner("uid-dm-a")
		assert.Equal(t, corev1.Device_Status_Binding_VALID, env.reconcile(t, dev).Status.Binding.Validity)

		env.setOwnerAt("dm-a", spec, time.Now().Add(-2*time.Hour), tstEntry("a-1", "serial-1"))
		assert.Equal(t, corev1.Device_Status_Binding_VALID, env.reconcile(t, dev).Status.Binding.Validity)
	}

	{
		env.setOwner("dm-a", spec)
		dev := env.reconcile(t, dev)
		binding := dev.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, binding.State)
		assert.Equal(t, corev1.Device_Status_Binding_LOST, binding.Validity)
		assert.Equal(t, reasonEntryNotFound, binding.Reason)
		assert.Equal(t, bindingUID, binding.Uid)
		assert.Nil(t, dev.Status.Posture)
	}

	{
		env.setOwner("dm-a", spec, tstEntry("a-1", "serial-1"), tstEntry("a-1", "serial-9"))
		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, dev.Status.Binding.Validity)
		assert.Equal(t, reasonEntryAmbiguous, dev.Status.Binding.Reason)
	}

	{
		entry := tstEntry("a-1", "serial-1")
		entry.OwnerEmails = []string{"other@example.com"}
		env.setOwner("dm-a", spec, entry)
		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_LOST, dev.Status.Binding.Validity)
		assert.Equal(t, reasonOwnerMismatch, dev.Status.Binding.Reason)
	}

	{
		env.setOwner("dm-a", &enterprisev1.DeviceManager_Spec{
			Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "false"}},
		}, tstEntry("a-1", "serial-1"))
		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_LOST, dev.Status.Binding.Validity)
		assert.Equal(t, reasonNotApplicable, dev.Status.Binding.Reason)

		env.setOwner("dm-a", &enterprisev1.DeviceManager_Spec{
			Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "error"}},
		}, tstEntry("a-1", "serial-1"))
		assert.Equal(t, reasonNotApplicable, env.reconcile(t, dev).Status.Binding.Reason)
	}

	{
		env.setOwner("dm-a", spec, tstEntry("a-1", "serial-1"))
		dev := env.reconcile(t, dev)
		binding := dev.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_VALID, binding.Validity)
		assert.Empty(t, binding.Reason)
		assert.Empty(t, binding.Message)
		assert.Equal(t, bindingUID, binding.Uid)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, dev.Status.Posture.Compliant)

		lastSyncAt := dev.Status.Posture.LastSyncAt.AsTime()

		entry := tstEntry("a-1", "serial-1")
		entry.Posture.Compliant = corev1.Device_Status_Posture_FAIL
		entry.Posture.Firewall = corev1.Device_Status_Posture_PASS
		env.setOwner("dm-a", spec, entry)

		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, dev.Status.Posture.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, dev.Status.Posture.Firewall)
		assert.False(t, dev.Status.Posture.LastSyncAt.AsTime().Before(lastSyncAt))

		resourceVersion := dev.Metadata.ResourceVersion
		assert.Equal(t, resourceVersion, env.reconcile(t, dev).Metadata.ResourceVersion)
	}
}

func TestReconcileDeviceVerification(t *testing.T) {
	env := newTstEnv(t)

	spec := &enterprisev1.DeviceManager_Spec{
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			VerificationInterval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
		},
	}

	owner := env.setOwner("dm-v", spec, tstEntry("v-1", "serial-1", "hw-uuid-1"),
		tstEntry("v-2", "serial-2", "hw-uuid-2"))

	dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user@example.com"))
	binding := dev.Status.Binding
	assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, binding.State)
	assert.True(t, binding.NextVerificationAt.IsValid())
	assert.InDelta(t, time.Hour.Seconds(),
		binding.NextVerificationAt.AsTime().Sub(binding.AcceptedAt.AsTime()).Seconds(), 1)

	{
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-uuid-1"))
			dev.Status.Binding.LastVerifiedAt = pbutils.Timestamp(time.Now().Add(-time.Hour))
		})

		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)
		assert.WithinDuration(t, time.Now(), dev.Status.Binding.LastVerifiedAt.AsTime(), time.Minute)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, dev.Status.ProbeAttempt.State)
	}

	{
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-uuid-2"))
		})

		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, dev.Status.Binding.Validity)
		assert.Equal(t, reasonVerificationFailed, dev.Status.Binding.Reason)
		assert.Equal(t, binding.Uid, dev.Status.Binding.Uid)
		assert.NotNil(t, dev.Status.Posture)

		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, dev.Status.Binding.Validity)

		env.setOwner("dm-v", nil, tstEntry("v-1", "serial-1", "hw-uuid-1"))
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, env.reconcile(t, dev).Status.Binding.Validity)
		assert.Nil(t, env.getDevice(t, dev).Status.Binding.NextVerificationAt)
		owner = env.setOwner("dm-v", spec, tstEntry("v-1", "serial-1", "hw-uuid-1"),
			tstEntry("v-2", "serial-2", "hw-uuid-2"))
	}

	{
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-uuid-1"))
		})

		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)
		assert.Empty(t, dev.Status.Binding.Reason)
	}

	{
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_UNSUPPORTED, ""))
		})

		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, dev.Status.Binding.Validity)
		assert.Equal(t, reasonVerificationFailed, dev.Status.Binding.Reason)
	}

	{
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-uuid-1"))
		})
		assert.Equal(t, corev1.Device_Status_Binding_VALID, env.reconcile(t, dev).Status.Binding.Validity)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.Binding.NextVerificationAt = pbutils.Timestamp(time.Now().Add(-time.Minute))
		})
		assert.Equal(t, corev1.Device_Status_Binding_VALID, env.reconcile(t, dev).Status.Binding.Validity)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.Binding.NextVerificationAt = pbutils.Timestamp(time.Now().Add(-time.Hour))
		})

		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, dev.Status.Binding.Validity)
		assert.Equal(t, reasonVerificationOverdue, dev.Status.Binding.Reason)

		env.setOwner("dm-v", nil, tstEntry("v-1", "serial-1", "hw-uuid-1"))
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)
		assert.Nil(t, dev.Status.Binding.NextVerificationAt)
	}

	{
		owner = env.setOwner("dm-v", nil, tstEntry("v-1", "serial-1", "hw-uuid-1"),
			tstEntry("v-2", "serial-2", "hw-uuid-2"))

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED,
				dev.Status.Binding.AcceptedAt.AsTime().Add(-time.Minute),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_PERMISSION_REQUIRED, ""))
		})
		assert.Equal(t, corev1.Device_Status_Binding_VALID, env.reconcile(t, dev).Status.Binding.Validity)
	}
}

func TestReconcileDeviceLastSeen(t *testing.T) {
	env := newTstEnv(t)
	now := time.Now()

	getEntry := func(lastSeenAt time.Time) *devicemgrcommon.Entry {
		ret := tstEntry("a-1", "serial-1")
		ret.Posture.LastSeenAt = pbutils.Timestamp(lastSeenAt)
		return ret
	}

	env.setOwner("dm-a", nil, getEntry(now.Add(-time.Hour)))

	dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user@example.com"))
	assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
	assert.Equal(t, now.Add(-time.Hour).Unix(), dev.Status.Posture.LastSeenAt.AsTime().Unix())

	{
		env.setOwner("dm-a", nil, getEntry(now.Add(-time.Hour+time.Minute)))
		resourceVersion := dev.Metadata.ResourceVersion
		dev := env.reconcile(t, dev)
		assert.Equal(t, resourceVersion, dev.Metadata.ResourceVersion)
		assert.Equal(t, now.Add(-time.Hour).Unix(), dev.Status.Posture.LastSeenAt.AsTime().Unix())
	}

	{
		env.setOwner("dm-a", nil, getEntry(now.Add(-time.Minute)))
		dev := env.reconcile(t, dev)
		assert.Equal(t, now.Add(-time.Minute).Unix(), dev.Status.Posture.LastSeenAt.AsTime().Unix())
	}

	{
		env.setOwner("dm-a", nil, tstEntry("a-1", "serial-1"))
		dev := env.reconcile(t, dev)
		assert.Nil(t, dev.Status.Posture.LastSeenAt)
	}
}

func TestIsLastSeenRefreshDue(t *testing.T) {
	now := time.Now()

	getPosture := func(lastSeenAt time.Time) *corev1.Device_Status_Posture {
		return &corev1.Device_Status_Posture{
			LastSeenAt: pbutils.Timestamp(lastSeenAt),
		}
	}

	assert.False(t, isLastSeenRefreshDue(nil, &corev1.Device_Status_Posture{}))
	assert.False(t, isLastSeenRefreshDue(getPosture(now), getPosture(now)))
	assert.False(t, isLastSeenRefreshDue(getPosture(now), getPosture(now.Add(5*time.Minute))))
	assert.True(t, isLastSeenRefreshDue(getPosture(now), getPosture(now.Add(lastSeenRefreshThreshold))))
	assert.True(t, isLastSeenRefreshDue(getPosture(now), getPosture(now.Add(-time.Hour))))
	assert.True(t, isLastSeenRefreshDue(&corev1.Device_Status_Posture{}, getPosture(now)))
	assert.True(t, isLastSeenRefreshDue(getPosture(now), &corev1.Device_Status_Posture{}))
}

func TestIsReconcileRequired(t *testing.T) {
	getDevice := func(fn func(dev *corev1.Device)) *corev1.Device {
		ret := &corev1.Device{
			Status: &corev1.Device_Status{
				UserRef:      &metav1.ObjectReference{Uid: "user-1"},
				OsType:       corev1.Device_Status_LINUX,
				SerialNumber: "serial-1",
				MacAddresses: []string{"a4:bb:cc:dd:ee:01"},
				Binding: &corev1.Device_Status_Binding{
					Uid:   "binding-1",
					State: corev1.Device_Status_Binding_ACCEPTED,
				},
				ProbeAttempt: &corev1.Device_Status_ProbeAttempt{
					Uid:   "attempt-1",
					State: corev1.Device_Status_ProbeAttempt_PROCESSED,
				},
			},
		}
		if fn != nil {
			fn(ret)
		}
		return ret
	}

	old := getDevice(nil)

	assert.True(t, isReconcileRequired(getDevice(nil), nil))
	assert.True(t, isReconcileRequired(getDevice(nil), &corev1.Device{}))
	assert.False(t, isReconcileRequired(getDevice(nil), old))

	for _, fn := range []func(dev *corev1.Device){
		func(dev *corev1.Device) {
			dev.Status.Posture = &corev1.Device_Status_Posture{Compliant: corev1.Device_Status_Posture_PASS}
		},
		func(dev *corev1.Device) {
			dev.Status.Binding.Validity = corev1.Device_Status_Binding_SUSPENDED
		},
		func(dev *corev1.Device) {
			dev.Status.ProbeAttempt.Uid = "attempt-2"
			dev.Status.ProbeAttempt.State = corev1.Device_Status_ProbeAttempt_ISSUED
		},
		func(dev *corev1.Device) {
			dev.Status.Hostname = "host"
		},
	} {
		assert.False(t, isReconcileRequired(getDevice(fn), old))
	}

	for _, fn := range []func(dev *corev1.Device){
		func(dev *corev1.Device) {
			dev.Status.ProbeAttempt.Uid = "attempt-2"
			dev.Status.ProbeAttempt.State = corev1.Device_Status_ProbeAttempt_SUBMITTED
		},
		func(dev *corev1.Device) {
			dev.Status.ProbeAttempt.State = corev1.Device_Status_ProbeAttempt_SUBMITTED
		},
		func(dev *corev1.Device) {
			dev.Status.Binding = nil
		},
		func(dev *corev1.Device) {
			dev.Status.SerialNumber = "serial-2"
		},
		func(dev *corev1.Device) {
			dev.Status.MacAddresses = nil
		},
		func(dev *corev1.Device) {
			dev.Status.OsType = corev1.Device_Status_MAC
		},
		func(dev *corev1.Device) {
			dev.Status.UserRef = &metav1.ObjectReference{Uid: "user-2"}
		},
	} {
		assert.True(t, isReconcileRequired(getDevice(fn), old))
	}

	submitted := getDevice(func(dev *corev1.Device) {
		dev.Status.ProbeAttempt.State = corev1.Device_Status_ProbeAttempt_SUBMITTED
	})
	assert.False(t, isReconcileRequired(getDevice(func(dev *corev1.Device) {
		dev.Status.ProbeAttempt.State = corev1.Device_Status_ProbeAttempt_SUBMITTED
		dev.Status.Posture = &corev1.Device_Status_Posture{}
	}), submitted))
}

func TestControllerEvents(t *testing.T) {
	env := newTstEnv(t)

	env.setOwner("dm-a", nil, tstEntry("a-1", "serial-1"))

	dev := env.createDevice(t, "SERIAL-1", "user@example.com")
	assert.Nil(t, env.ctrl.OnAdd(env.ctx, dev))
	bound := env.getDevice(t, dev)
	assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, bound.Status.Binding.State)
	assert.Equal(t, corev1.Device_Status_Binding_VALID, bound.Status.Binding.Validity)

	env.setOwner("dm-a", nil)

	updated := env.updateDevice(t, bound, func(dev *corev1.Device) {
		dev.Status.Hostname = "host-2"
	})
	assert.Nil(t, env.ctrl.OnUpdate(env.ctx, updated, bound))
	assert.Equal(t, corev1.Device_Status_Binding_VALID, env.getDevice(t, dev).Status.Binding.Validity)

	changed := env.updateDevice(t, updated, func(dev *corev1.Device) {
		dev.Status.SerialNumber = "SERIAL-9"
	})
	assert.Nil(t, env.ctrl.OnUpdate(env.ctx, changed, updated))
	lost := env.getDevice(t, dev)
	assert.Equal(t, corev1.Device_Status_Binding_LOST, lost.Status.Binding.Validity)
	assert.Equal(t, reasonEntryNotFound, lost.Status.Binding.Reason)

	env.ctrl.LockDevice(dev.Metadata.Uid)()
	_, ok := env.ctrl.deviceLocks.Load(dev.Metadata.Uid)
	assert.True(t, ok)
	assert.Nil(t, env.ctrl.OnDelete(env.ctx, dev))
	_, ok = env.ctrl.deviceLocks.Load(dev.Metadata.Uid)
	assert.False(t, ok)

	assert.Nil(t, env.ctrl.OnDelete(env.ctx, &corev1.Device{}))
	env.ctrl.LockDevice("")()
}

func TestReconcileDeviceOwnerMatchMissingUser(t *testing.T) {
	env := newTstEnv(t)

	spec := &enterprisev1.DeviceManager_Spec{
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			RequireOwnerMatch: true,
		},
	}

	env.setOwner("dm-a", spec, tstEntry("a-1", "serial-1"), tstEntry("a-2", "serial-2"), tstEntry("a-3", "serial-3"))
	ownerB := env.setOwner("dm-b", nil, tstEntry("b-1", "serial-1"), tstEntry("b-2", "serial-2"))

	{
		dev := env.createDevice(t, "SERIAL-1", "owner@example.com")
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.UserRef = nil
		})
		dev = env.reconcile(t, dev)
		assert.Equal(t, ownerB.UID(), dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, "b-1", dev.Status.Binding.ExternalID)
	}

	{
		dev := env.createDevice(t, "SERIAL-2", "owner@example.com")
		_, err := env.octeliumC.CoreC().DeleteUser(env.ctx, &rmetav1.DeleteOptions{Uid: dev.Status.UserRef.Uid})
		assert.Nil(t, err)
		dev = env.reconcile(t, dev)
		assert.Equal(t, ownerB.UID(), dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, "b-2", dev.Status.Binding.ExternalID)
	}

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-3", "owner@example.com"))
		assert.Equal(t, "uid-dm-a", dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)

		_, err := env.octeliumC.CoreC().DeleteUser(env.ctx, &rmetav1.DeleteOptions{Uid: dev.Status.UserRef.Uid})
		assert.Nil(t, err)
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_LOST, dev.Status.Binding.Validity)
		assert.Equal(t, reasonOwnerMismatch, dev.Status.Binding.Reason)
		assert.Nil(t, dev.Status.Posture)
	}
}

func TestReconcileDevicesSharesLookups(t *testing.T) {
	env := newTstEnv(t)

	spec := &enterprisev1.DeviceManager_Spec{
		Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "has-user"}},
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			RequireOwnerMatch: true,
		},
	}

	env.setOwner("dm-a", spec, tstEntry("a-1", "serial-1"), tstEntry("a-2", "serial-2"),
		tstEntry("a-3", "serial-3"), tstEntry("a-4", "serial-4"))
	env.setOwner("dm-b", &enterprisev1.DeviceManager_Spec{
		Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "has-user"}},
	}, tstEntry("b-5", "serial-5"))

	var devices []*corev1.Device
	for i, email := range []string{"owner@example.com", "other@example.com"} {
		usr, err := env.octeliumC.CoreC().CreateUser(env.ctx, &corev1.User{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &corev1.User_Spec{
				Type:  corev1.User_Spec_HUMAN,
				Email: email,
			},
			Status: &corev1.User_Status{},
		})
		assert.Nil(t, err)

		for j := range 2 {
			devices = append(devices, env.createUserDevice(t, usr, fmt.Sprintf("SERIAL-%d", i*2+j+1)))
		}
	}
	devices = append(devices, env.createDevice(t, "SERIAL-5", "owner@example.com"))

	client := env.useCountingClient()

	for _, err := range env.ctrl.ReconcileDevices(env.ctx, devices) {
		assert.Nil(t, err, "%+v", err)
	}

	assert.Zero(t, client.core.getDevice.Load())
	assert.Equal(t, int32(3), client.core.getUser.Load())
	assert.Equal(t, int32(1), client.enterpriseUtils.getClusterConfig.Load())
	for _, dev := range devices {
		assert.Nil(t, dev.Status.Binding)
	}

	var bindingUIDs []string
	for i, dev := range devices {
		dev := env.getDevice(t, dev)
		switch i {
		case 0, 1:
			assert.Equal(t, "uid-dm-a", dev.Status.Binding.OwnerRef.Uid)
			assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
		case 2, 3:
			assert.Nil(t, dev.Status.Binding)
		case 4:
			assert.Equal(t, "uid-dm-b", dev.Status.Binding.OwnerRef.Uid)
		}
		bindingUIDs = append(bindingUIDs, dev.Status.GetBinding().GetUid())
	}

	for _, err := range env.ctrl.ReconcileDevices(env.ctx, devices) {
		assert.Nil(t, err, "%+v", err)
	}

	assert.Equal(t, int32(3), client.core.getDevice.Load())
	assert.Equal(t, int32(2), client.enterpriseUtils.getClusterConfig.Load())

	for i, dev := range devices {
		assert.Equal(t, bindingUIDs[i], env.getDevice(t, dev).Status.GetBinding().GetUid())
	}

	ctx, cancel := context.WithCancel(env.ctx)
	cancel()
	for _, err := range env.ctrl.ReconcileDevices(ctx, devices) {
		assert.ErrorIs(t, err, context.Canceled)
	}
	assert.Equal(t, int32(3), client.core.getDevice.Load())
}

func TestReconcileDevicesEdgeCases(t *testing.T) {
	env := newTstEnv(t)

	owner := env.setOwner("dm-a", nil, tstEntry("a-1", "serial-1"))

	deleted := env.createDeletedDevice(t)
	deleted.Status.Binding = &corev1.Device_Status_Binding{
		Uid:      utilrand.GetRandomStringCanonical(32),
		OwnerRef: owner.OwnerRef(),
		State:    corev1.Device_Status_Binding_AMBIGUOUS,
		Reason:   reasonAmbiguous,
	}

	errs := env.ctrl.ReconcileDevices(env.ctx, []*corev1.Device{
		nil,
		deleted,
		{Metadata: &metav1.Metadata{}},
	})
	assert.Len(t, errs, 3)
	assert.Nil(t, errs[0])
	assert.Nil(t, errs[1])
	assert.NotNil(t, errs[2])
	assert.NotNil(t, deleted.Status.Binding)

	assert.Empty(t, env.ctrl.ReconcileDevices(env.ctx, nil))
}

func TestReconcileDeviceUserLookup(t *testing.T) {
	env := newTstEnv(t)

	spec := &enterprisev1.DeviceManager_Spec{
		Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "has-user"}},
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			RequireOwnerMatch: true,
		},
	}

	env.setOwner("dm-a", spec, tstEntry("a-1", "serial-1"))
	env.setOwner("dm-b", spec, tstEntry("b-1", "serial-1"))
	env.setDeviceManagers(t, "dm-a", "dm-b")

	dev := env.createDevice(t, "SERIAL-1", "other@example.com")
	client := env.useCountingClient()

	client.core.getUserErr.Store(1)
	assert.Nil(t, env.ctrl.ReconcileDevice(env.ctx, dev))
	assert.Nil(t, env.getDevice(t, dev).Status.Binding)
	assert.Equal(t, int32(1), client.core.getUser.Load())

	assert.Nil(t, env.ctrl.ReconcileDevice(env.ctx, dev))
	assert.Nil(t, env.getDevice(t, dev).Status.Binding)
	assert.Equal(t, int32(2), client.core.getUser.Load())
	assert.Equal(t, int32(2), client.enterpriseUtils.getClusterConfig.Load())

	env.setOwner("dm-b", nil, tstEntry("b-1", "serial-1"))
	assert.Nil(t, env.ctrl.ReconcileDevice(env.ctx, dev))
	bound := env.getDevice(t, dev)
	assert.Equal(t, "uid-dm-b", bound.Status.Binding.OwnerRef.Uid)
	assert.Equal(t, int32(3), client.core.getUser.Load())
}

func TestResetBindingsForOwner(t *testing.T) {
	env := newTstEnv(t)

	env.setOwner("dm-a", nil, tstEntry("a-1", "serial-1"), tstEntry("a-2", "serial-2"))
	env.setOwner("dm-b", nil, tstEntry("b-3", "serial-3"))

	dev1 := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user@example.com"))
	dev2 := env.reconcile(t, env.createDevice(t, "SERIAL-2", "user@example.com"))
	dev3 := env.reconcile(t, env.createDevice(t, "SERIAL-3", "user@example.com"))

	assert.Equal(t, "uid-dm-a", dev1.Status.Binding.OwnerRef.Uid)
	assert.Equal(t, "uid-dm-a", dev2.Status.Binding.OwnerRef.Uid)
	assert.Equal(t, "uid-dm-b", dev3.Status.Binding.OwnerRef.Uid)

	assert.Nil(t, env.ctrl.ResetBindingsForOwner(env.ctx, "uid-dm-a"))
	assert.NotNil(t, env.ctrl.ResetBindingsForOwner(env.ctx, ""))

	assert.Nil(t, env.getDevice(t, dev1).Status.Binding)
	assert.Nil(t, env.getDevice(t, dev2).Status.Binding)
	assert.NotNil(t, env.getDevice(t, dev3).Status.Binding)
}

func TestIsPostureRefreshDue(t *testing.T) {
	now := time.Now()

	current := &corev1.Device_Status_Posture{
		LastSyncAt: pbutils.Timestamp(now.Add(-40 * time.Minute)),
		ExpiresAt:  pbutils.Timestamp(now.Add(20 * time.Minute)),
	}

	assert.True(t, isPostureRefreshDue(nil, &corev1.Device_Status_Posture{}, now))
	assert.True(t, isPostureRefreshDue(&corev1.Device_Status_Posture{}, &corev1.Device_Status_Posture{}, now))
	assert.True(t, isPostureRefreshDue(current, &corev1.Device_Status_Posture{
		ExpiresAt: pbutils.Timestamp(now.Add(time.Hour)),
	}, now))
	assert.False(t, isPostureRefreshDue(current, &corev1.Device_Status_Posture{
		ExpiresAt: current.ExpiresAt,
	}, now))
	assert.False(t, isPostureRefreshDue(current, &corev1.Device_Status_Posture{
		ExpiresAt: pbutils.Timestamp(now.Add(time.Hour)),
	}, now.Add(-15*time.Minute)))

	assert.True(t, posturesEqualIgnoringTimestamps(current, &corev1.Device_Status_Posture{
		LastSeenAt: pbutils.Now(),
	}))
	assert.False(t, posturesEqualIgnoringTimestamps(current, &corev1.Device_Status_Posture{
		Compliant: corev1.Device_Status_Posture_PASS,
	}))
}

func TestAttemptHelpers(t *testing.T) {
	now := time.Now()
	owner := devicemgrcommon.NewOwner(&tstManager{}, tstDeviceManager("dm-a", &enterprisev1.DeviceManager_Spec{}),
		devicemgrcommon.NewFleet(nil), now, time.Hour)

	dev := &corev1.Device{Status: &corev1.Device_Status{}}
	assert.Nil(t, getAttemptResults(dev))
	assert.False(t, isAttemptPending(dev, owner.UID(), now))
	assert.False(t, markAttemptProcessed(dev))

	dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_ISSUED, now)
	assert.Nil(t, getAttemptResults(dev))
	assert.True(t, isAttemptPending(dev, owner.UID(), now))
	assert.False(t, isAttemptPending(dev, "other", now))
	assert.False(t, isAttemptPending(dev, owner.UID(), now.Add(time.Hour)))
	assert.False(t, markAttemptProcessed(dev))

	dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, now,
		tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "a"),
		tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_NOT_FOUND, "b"),
		&corev1.Device_Status_ProbeAttempt_Result{
			ProbeID: "unknown",
			Status:  corev1.Device_Status_ProbeAttempt_Result_OK,
		},
		&corev1.Device_Status_ProbeAttempt_Result{
			ProbeID: fmt.Sprintf("%s.probe", owner.Name()),
			Status:  corev1.Device_Status_ProbeAttempt_Result_OK,
			Value: &corev1.Device_Status_ProbeAttempt_Result_List_{
				List: &corev1.Device_Status_ProbeAttempt_Result_List{
					Items: []string{"x", "y"},
				},
			},
		},
	)

	results := getAttemptResults(dev)
	assert.Len(t, results, 1)
	assert.Len(t, results[owner.UID()], 2)
	assert.Equal(t, "a", results[owner.UID()][0].Text)
	assert.Equal(t, []string{"x", "y"}, results[owner.UID()][1].Items)

	assert.True(t, markAttemptProcessed(dev))
	assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, dev.Status.ProbeAttempt.State)
	assert.Len(t, getAttemptResults(dev)[owner.UID()], 2)
}
