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
	"google.golang.org/protobuf/proto"
)

type tstManager struct {
	isPlatformProbe bool
}

func (m *tstManager) Type() devicemgrcommon.ProviderType {
	return enterprisev1.DeviceManager_Status_CROWDSTRIKE
}

func (m *tstManager) IdentityProbes() []*devicemgrcommon.Probe {
	ret := &devicemgrcommon.Probe{
		ID:         "probe",
		OSType:     corev1.Device_Status_LINUX,
		RunCommand: &devicemgrcommon.RunCommand{Command: "/opt/agent"},
	}

	if m.isPlatformProbe {
		ret.RunCommand = nil
		ret.PlatformIdentifier = &devicemgrcommon.PlatformIdentifier{
			Kind: corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
		}
	}

	return []*devicemgrcommon.Probe{ret}
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
	getDevice        atomic.Int32
	getUser          atomic.Int32
	getUserErr       atomic.Int32
	updateSessionErr atomic.Int32
}

func (c *countingCore) UpdateSession(ctx context.Context, req *corev1.Session, opts ...grpc.CallOption) (*corev1.Session, error) {
	if c.updateSessionErr.Load() > 0 {
		c.updateSessionErr.Add(-1)
		return nil, status.Error(codes.Unavailable, "unavailable")
	}
	return c.ResourceServiceClient.UpdateSession(ctx, req, opts...)
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

func (e *tstEnv) createSession(t *testing.T, dev *corev1.Device) *corev1.Session {
	sess, err := e.octeliumC.CoreC().CreateSession(e.ctx, &corev1.Session{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.Session_Spec{
			State: corev1.Session_Spec_ACTIVE,
		},
		Status: &corev1.Session_Status{
			Type:      corev1.Session_Status_CLIENT,
			UserRef:   dev.Status.UserRef,
			DeviceRef: umetav1.GetObjectReference(dev),
		},
	})
	assert.Nil(t, err, "%+v", err)

	return sess
}

func (e *tstEnv) getDeviceProof(t *testing.T, sess *corev1.Session) *corev1.Session_Status_DeviceProof {
	ret, err := e.octeliumC.CoreC().GetSession(e.ctx, &rmetav1.GetOptions{Uid: sess.Metadata.Uid})
	assert.Nil(t, err)
	return ret.Status.DeviceProof
}

func tstAssertBinding(t *testing.T, binding *corev1.Device_Status_Binding,
	isOwnerMatched bool, assurance corev1.Device_Status_Binding_Assurance) {
	t.Helper()

	assert.Equal(t, isOwnerMatched, binding.IsOwnerMatched)
	assert.Equal(t, assurance, binding.Assurance)
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
		ExternalID: externalID,
		Aliases:    aliases,
		Serial:     serial,
		Posture: &corev1.Device_Status_Posture{
			Compliant: corev1.Device_Status_Posture_PASS,
		},
	}
}

func tstOwnedEntry(externalID, serial, email string, aliases ...string) *devicemgrcommon.Entry {
	ret := tstEntry(externalID, serial, aliases...)
	ret.OSType = corev1.Device_Status_LINUX
	ret.OwnerEmails = []string{email}
	ret.Posture.LastSeenAt = pbutils.Timestamp(time.Now().Add(-time.Hour))
	return ret
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
	}, tstEntry("a-1", "serial-1"), tstOwnedEntry("a-2", "serial-2", "owner@example.com"))
	ownerB := env.setOwner("dm-b", nil, tstEntry("b-1", "serial-1"), tstEntry("b-2", "serial-2"))

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user@example.com"))
		assert.Equal(t, ownerB.UID(), dev.Status.Binding.OwnerRef.Uid)
		tstAssertBinding(t, dev.Status.Binding, false, corev1.Device_Status_Binding_IDENTIFIER)
	}

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-2", "Owner@Example.com"))
		assert.Equal(t, "uid-dm-a", dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, "a-2", dev.Status.Binding.ExternalID)
		tstAssertBinding(t, dev.Status.Binding, true, corev1.Device_Status_Binding_IDENTIFIER)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_OWNER_EMAIL,
			corev1.Device_Status_Binding_SERIAL_NUMBER,
		}, dev.Status.Binding.Evidence)
	}

	{
		dev := env.reconcile(t, env.createDevice(t, "SERIAL-2", "user@example.com"))
		binding := dev.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_OWNER_MISMATCH, binding.State)
		assert.Equal(t, "uid-dm-a", binding.OwnerRef.Uid)
		assert.Equal(t, "a-2", binding.ExternalID)
		assert.Equal(t, reasonOwnerMismatch, binding.Reason)
		assert.Nil(t, dev.Status.Posture)

		resourceVersion := dev.Metadata.ResourceVersion
		assert.Equal(t, resourceVersion, env.reconcile(t, dev).Metadata.ResourceVersion)
	}
}

func TestReconcileDeviceOwnerLinking(t *testing.T) {
	env := newTstEnv(t)

	newOwner := func(entries ...*devicemgrcommon.Entry) *devicemgrcommon.Owner {
		return env.setOwner("dm-a", nil, entries...)
	}

	aliceEntry := func(email string) *devicemgrcommon.Entry {
		ret := tstOwnedEntry("alice-1", "serial-alice", email, "hw-alice")
		if email == "" {
			ret.OwnerEmails = nil
		}
		return ret
	}

	frank1 := tstOwnedEntry("frank-1", "", "frank@example.com")
	frank1.Hostname = "frank-desktop"
	frank2 := tstOwnedEntry("frank-2", "", "frank@example.com")
	frank2.Hostname = "frank-laptop"

	entries := func(alice *devicemgrcommon.Entry) []*devicemgrcommon.Entry {
		return []*devicemgrcommon.Entry{
			alice,
			tstOwnedEntry("bob-1", "serial-bob", "bob@example.com"),
			tstOwnedEntry("carol-1", "", "carol@example.com", "hw-carol"),
			frank1,
			frank2,
		}
	}

	owner := newOwner(entries(aliceEntry("alice@example.com"))...)
	env.setOwner("dm-b", nil, tstEntry("b-1", "serial-alice"))

	aliceUsr, err := env.octeliumC.CoreC().CreateUser(env.ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type:  corev1.User_Spec_HUMAN,
			Email: "alice@example.com",
		},
		Status: &corev1.User_Status{},
	})
	assert.Nil(t, err)

	medium := env.reconcile(t, env.createUserDevice(t, aliceUsr, ""))
	{
		binding := medium.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, binding.State)
		assert.Equal(t, "alice-1", binding.ExternalID)
		tstAssertBinding(t, binding, true, corev1.Device_Status_Binding_OWNER_ONLY)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_OWNER_EMAIL,
		}, binding.Evidence)
		assert.NotNil(t, medium.Status.Posture)
	}

	high := env.reconcile(t, env.createUserDevice(t, aliceUsr, "SERIAL-ALICE"))
	{
		binding := high.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, binding.State)
		assert.Equal(t, "alice-1", binding.ExternalID)
		tstAssertBinding(t, binding, true, corev1.Device_Status_Binding_IDENTIFIER)

		dev := env.getDevice(t, medium)
		assert.Equal(t, corev1.Device_Status_Binding_CONFLICT, dev.Status.Binding.State)
		assert.Equal(t, reasonSuperseded, dev.Status.Binding.Reason)
		assert.Contains(t, dev.Status.Binding.Message, high.Metadata.Name)
		assert.NotEqual(t, medium.Status.Binding.Uid, dev.Status.Binding.Uid)
		assert.Nil(t, dev.Status.Posture)

		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_CONFLICT, dev.Status.Binding.State)
		assert.Equal(t, reasonConflict, dev.Status.Binding.Reason)
		assert.Equal(t, high.Status.Binding.Uid, env.getDevice(t, high).Status.Binding.Uid)
	}

	{
		owner = newOwner(entries(aliceEntry(""))...)
		dev := env.reconcile(t, high)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)
		tstAssertBinding(t, dev.Status.Binding, false, corev1.Device_Status_Binding_IDENTIFIER)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_SERIAL_NUMBER,
		}, dev.Status.Binding.Evidence)

		owner = newOwner(entries(aliceEntry("alice@example.com"))...)
		dev = env.reconcile(t, dev)
		tstAssertBinding(t, dev.Status.Binding, true, corev1.Device_Status_Binding_IDENTIFIER)

		owner = newOwner(entries(aliceEntry("eve@example.com"))...)
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_LOST, dev.Status.Binding.Validity)
		assert.Equal(t, reasonOwnerMismatch, dev.Status.Binding.Reason)
		assert.Nil(t, dev.Status.Posture)

		other := env.reconcile(t, env.createUserDevice(t, aliceUsr, "SERIAL-ALICE"))
		assert.Equal(t, corev1.Device_Status_Binding_OWNER_MISMATCH, other.Status.Binding.State)
		assert.Equal(t, "uid-dm-a", other.Status.Binding.OwnerRef.Uid)

		owner = newOwner(entries(aliceEntry("alice@example.com"))...)
		other = env.reconcile(t, other)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, other.Status.Binding.State)
		tstAssertBinding(t, other.Status.Binding, true, corev1.Device_Status_Binding_IDENTIFIER)
		assert.Equal(t, corev1.Device_Status_Binding_CONFLICT, env.getDevice(t, high).Status.Binding.State)
	}

	{
		dev := env.reconcile(t, env.createDevice(t, "", "bob@example.com"))
		assert.Equal(t, "bob-1", dev.Status.Binding.ExternalID)
		tstAssertBinding(t, dev.Status.Binding, true, corev1.Device_Status_Binding_OWNER_ONLY)

		dev = env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.SerialNumber = "SERIAL-OTHER"
		})
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_LOST, dev.Status.Binding.Validity)
		assert.Equal(t, reasonIdentityMismatch, dev.Status.Binding.Reason)

		assert.Nil(t, env.reconcile(t, env.createDevice(t, "SERIAL-OTHER", "bob@example.com")).Status.Binding)
	}

	{
		dev := env.createDevice(t, "", "carol@example.com")
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_ISSUED, time.Now())
		})
		dev = env.reconcile(t, dev)
		assert.Nil(t, dev.Status.Binding)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "HW-CAROL"))
		})
		dev = env.reconcile(t, dev)
		assert.Equal(t, "carol-1", dev.Status.Binding.ExternalID)
		tstAssertBinding(t, dev.Status.Binding, true, corev1.Device_Status_Binding_PROVIDER_ID)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_OWNER_EMAIL,
			corev1.Device_Status_Binding_PROBE,
		}, dev.Status.Binding.Evidence)

		dev = env.reconcile(t, env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = nil
		}))
		tstAssertBinding(t, dev.Status.Binding, true, corev1.Device_Status_Binding_PROVIDER_ID)
		assert.Contains(t, dev.Status.Binding.Evidence, corev1.Device_Status_Binding_PROBE)
	}

	{
		dev := env.createDevice(t, "", "carol@example.com")
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-unknown"))
		})
		assert.Nil(t, env.reconcile(t, dev).Status.Binding)
	}

	{
		dev := env.createDevice(t, "", "frank@example.com")
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_AMBIGUOUS, dev.Status.Binding.State)
		assert.Equal(t, reasonOwnerAmbiguous, dev.Status.Binding.Reason)

		dev = env.reconcile(t, env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.Hostname = "FRANK-LAPTOP.local"
		}))
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
		assert.Equal(t, "frank-2", dev.Status.Binding.ExternalID)
		tstAssertBinding(t, dev.Status.Binding, true, corev1.Device_Status_Binding_OWNER_ONLY)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_OWNER_EMAIL,
			corev1.Device_Status_Binding_HOSTNAME,
		}, dev.Status.Binding.Evidence)
	}
}

func TestReconcileDeviceTakeOverLost(t *testing.T) {
	env := newTstEnv(t)

	env.setOwner("dm-a", nil, tstEntry("a-1", "serial-1"))

	holder := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user1@example.com"))
	assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, holder.Status.Binding.State)
	tstAssertBinding(t, holder.Status.Binding, false, corev1.Device_Status_Binding_IDENTIFIER)

	dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user2@example.com"))
	assert.Equal(t, corev1.Device_Status_Binding_CONFLICT, dev.Status.Binding.State)

	env.updateDevice(t, holder, func(dev *corev1.Device) {
		dev.Status.Binding.Validity = corev1.Device_Status_Binding_LOST
	})

	dev = env.reconcile(t, dev)
	assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
	assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)

	holder = env.getDevice(t, holder)
	assert.Equal(t, corev1.Device_Status_Binding_CONFLICT, holder.Status.Binding.State)
	assert.Equal(t, reasonSuperseded, holder.Status.Binding.Reason)

	assert.Nil(t, env.ctrl.supersede(env.ctx, holder, dev))
	assert.Equal(t, holder.Metadata.ResourceVersion, env.getDevice(t, holder).Metadata.ResourceVersion)
	assert.Nil(t, env.ctrl.supersede(env.ctx, env.createDeletedDevice(t), dev))
}

func TestReconcileDeviceProbe(t *testing.T) {
	env := newTstEnv(t)
	now := time.Now()

	owner := env.setOwner("dm-p", nil, tstEntry("p-1", "serial-1", "hw-uuid-1"), tstEntry("p-2", "serial-2", "hw-uuid-2"))

	{
		dev := env.createDevice(t, "", "user@example.com")
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
		tstAssertBinding(t, dev.Status.Binding, false, corev1.Device_Status_Binding_PROVIDER_ID)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_PROBE,
		}, dev.Status.Binding.Evidence)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, dev.Status.ProbeAttempt.State)
	}

	{
		dev := env.createDevice(t, "", "user@example.com")
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

func TestReconcileDeviceSourcesDisagree(t *testing.T) {
	env := newTstEnv(t)
	now := time.Now()

	owner := env.setOwner("dm-a", nil, tstEntry("a-1", "serial-1", "hw-uuid-1"),
		tstEntry("a-2", "serial-2", "hw-uuid-2"))
	env.setOwner("dm-b", nil, tstEntry("b-1", "serial-1"))

	dev := env.createDevice(t, "SERIAL-1", "user@example.com")

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
	assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
		corev1.Device_Status_Binding_PROBE,
		corev1.Device_Status_Binding_SERIAL_NUMBER,
	}, dev.Status.Binding.Evidence)
}

func TestReconcileDeviceAccepted(t *testing.T) {
	env := newTstEnv(t)

	spec := &enterprisev1.DeviceManager_Spec{
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			RequireOwnerMatch: true,
		},
	}

	env.setOwner("dm-a", spec, tstOwnedEntry("a-1", "serial-1", "owner@example.com"))

	dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "owner@example.com"))
	bindingUID := dev.Status.Binding.Uid
	assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
	tstAssertBinding(t, dev.Status.Binding, true, corev1.Device_Status_Binding_IDENTIFIER)

	{
		env.registry.DeleteOwner("uid-dm-a")
		assert.Equal(t, corev1.Device_Status_Binding_VALID, env.reconcile(t, dev).Status.Binding.Validity)

		env.setOwnerAt("dm-a", spec, time.Now().Add(-2*time.Hour), tstOwnedEntry("a-1", "serial-1", "owner@example.com"))
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
		env.setOwner("dm-a", spec, tstOwnedEntry("a-1", "serial-1", "owner@example.com"), tstEntry("a-1", "serial-9"))
		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, dev.Status.Binding.Validity)
		assert.Equal(t, reasonEntryAmbiguous, dev.Status.Binding.Reason)
	}

	{
		entry := tstOwnedEntry("a-1", "serial-1", "owner@example.com")
		entry.OwnerEmails = []string{"other@example.com"}
		env.setOwner("dm-a", spec, entry)
		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_LOST, dev.Status.Binding.Validity)
		assert.Equal(t, reasonOwnerMismatch, dev.Status.Binding.Reason)

		env.setOwner("dm-a", spec, tstEntry("a-1", "serial-1"))
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_LOST, dev.Status.Binding.Validity)
		assert.Equal(t, reasonOwnerMismatch, dev.Status.Binding.Reason)
	}

	{
		env.setOwner("dm-a", &enterprisev1.DeviceManager_Spec{
			Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "false"}},
		}, tstOwnedEntry("a-1", "serial-1", "owner@example.com"))
		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_LOST, dev.Status.Binding.Validity)
		assert.Equal(t, reasonNotApplicable, dev.Status.Binding.Reason)

		env.setOwner("dm-a", &enterprisev1.DeviceManager_Spec{
			Condition: &corev1.Condition{Type: &corev1.Condition_Match{Match: "error"}},
		}, tstOwnedEntry("a-1", "serial-1", "owner@example.com"))
		assert.Equal(t, reasonNotApplicable, env.reconcile(t, dev).Status.Binding.Reason)
	}

	{
		env.setOwner("dm-a", spec, tstOwnedEntry("a-1", "serial-1", "owner@example.com"))
		dev := env.reconcile(t, dev)
		binding := dev.Status.Binding
		assert.Equal(t, corev1.Device_Status_Binding_VALID, binding.Validity)
		assert.Empty(t, binding.Reason)
		assert.Empty(t, binding.Message)
		assert.Equal(t, bindingUID, binding.Uid)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, dev.Status.Posture.Compliant)

		lastSyncAt := dev.Status.Posture.LastSyncAt.AsTime()

		entry := tstOwnedEntry("a-1", "serial-1", "owner@example.com")
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
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_PROBE,
			corev1.Device_Status_Binding_SERIAL_NUMBER,
		}, dev.Status.Binding.Evidence)
	}

	{
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-uuid-2"))
		})

		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, dev.Status.Binding.Validity)
		assert.Equal(t, reasonVerificationContradicted, dev.Status.Binding.Reason)
		assert.Equal(t, binding.Uid, dev.Status.Binding.Uid)
		assert.NotNil(t, dev.Status.Posture)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_SERIAL_NUMBER,
		}, dev.Status.Binding.Evidence)

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
		assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity,
			"a Device that produces no evidence must not be suspended")
		assert.Empty(t, dev.Status.Binding.Reason)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, dev.Status.ProbeAttempt.State)
		tstAssertBinding(t, dev.Status.Binding, false, corev1.Device_Status_Binding_PROVIDER_ID)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "invalid"))
		})

		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)
		tstAssertBinding(t, dev.Status.Binding, false, corev1.Device_Status_Binding_PROVIDER_ID)
	}

	{
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "unknown-id"))
		})

		dev := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, dev.Status.Binding.Validity,
			"an identifier that is not the bound entry's contradicts the Binding")
		assert.Equal(t, reasonVerificationContradicted, dev.Status.Binding.Reason)
		tstAssertBinding(t, dev.Status.Binding, false, corev1.Device_Status_Binding_IDENTIFIER)
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
		assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity,
			"an overdue verification lowers the Assurance and does not suspend the Binding")
		assert.Equal(t, reasonVerificationOverdue, dev.Status.Binding.Reason)
		assert.NotNil(t, dev.Status.Posture)
		tstAssertBinding(t, dev.Status.Binding, false, corev1.Device_Status_Binding_IDENTIFIER)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_SERIAL_NUMBER,
		}, dev.Status.Binding.Evidence)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "hw-uuid-1"))
		})

		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)
		assert.Empty(t, dev.Status.Binding.Reason)
		tstAssertBinding(t, dev.Status.Binding, false, corev1.Device_Status_Binding_PROVIDER_ID)
		assert.True(t, dev.Status.Binding.NextVerificationAt.AsTime().After(time.Now()))

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

func TestReconcileDevicePendingProbe(t *testing.T) {
	env := newTstEnv(t)

	owner := env.setOwner("dm-q", nil, tstEntry("q-1", "serial-1"),
		tstOwnedEntry("q-2", "", "owner@example.com"))

	{
		dev := env.createDevice(t, "SERIAL-1", "user@example.com")
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_ISSUED, time.Now())
		})

		res := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, res.Status.Binding.GetState(),
			"an identifier must not wait for a pending probe")
		tstAssertBinding(t, res.Status.Binding, false, corev1.Device_Status_Binding_IDENTIFIER)
	}

	{
		dev := env.createDevice(t, "", "owner@example.com")
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_ISSUED, time.Now())
		})

		res := env.reconcile(t, dev)
		assert.Nil(t, res.Status.Binding, "the owner alone must wait for a pending probe")
	}
}

func TestReconcileDeviceVerificationWithoutProbes(t *testing.T) {
	env := newTstEnv(t)

	env.setOwner("dm-v", &enterprisev1.DeviceManager_Spec{
		Linking: &enterprisev1.DeviceManager_Spec_Linking{
			VerificationInterval: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
		},
	}, tstEntry("v-1", "serial-1"))

	dev := env.createDevice(t, "SERIAL-1", "user@example.com")
	dev = env.updateDevice(t, dev, func(dev *corev1.Device) {
		dev.Status.OsType = corev1.Device_Status_MAC
	})

	dev = env.reconcile(t, dev)
	assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, dev.Status.Binding.State)
	assert.Nil(t, dev.Status.Binding.NextVerificationAt,
		"a Device of an OS without Probes has nothing to verify")

	env.updateDevice(t, dev, func(dev *corev1.Device) {
		dev.Status.Binding.NextVerificationAt = pbutils.Timestamp(time.Now().Add(-time.Hour))
	})

	dev = env.reconcile(t, dev)
	assert.Equal(t, corev1.Device_Status_Binding_VALID, dev.Status.Binding.Validity)
	assert.Empty(t, dev.Status.Binding.Reason)
	assert.Nil(t, dev.Status.Binding.NextVerificationAt)
}

func TestReconcileDeviceSessionProof(t *testing.T) {
	env := newTstEnv(t)

	owner := env.setOwner("dm-p", nil, tstEntry("p-1", "serial-1", "agent-1"),
		tstEntry("p-2", "serial-2", "agent-2"))

	dev := env.reconcile(t, env.createDevice(t, "SERIAL-1", "user@example.com"))
	tstAssertBinding(t, dev.Status.Binding, false, corev1.Device_Status_Binding_IDENTIFIER)

	submit := func(t *testing.T, sess *corev1.Session, status corev1.Device_Status_ProbeAttempt_Result_Status, text string) {
		t.Helper()

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, status, text))
			dev.Status.ProbeAttempt.SessionRef = umetav1.GetObjectReference(sess)
		})
	}

	t.Run("ProbeResolvesToTheEntry", func(t *testing.T) {
		sess := env.createSession(t, dev)
		submit(t, sess, corev1.Device_Status_ProbeAttempt_Result_OK, "agent-1")

		res := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, res.Status.Binding.Validity)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, res.Status.ProbeAttempt.State)
		tstAssertBinding(t, res.Status.Binding, false, corev1.Device_Status_Binding_PROVIDER_ID)

		proof := env.getDeviceProof(t, sess)
		assert.Equal(t, corev1.Session_Status_DeviceProof_PROBE, proof.GetType())
		assert.Equal(t, res.Status.Binding.Uid, proof.GetBindingUID())
		assert.True(t, proof.GetProvedAt().IsValid())
	})

	t.Run("NoUsableResult", func(t *testing.T) {
		sess := env.createSession(t, dev)
		submit(t, sess, corev1.Device_Status_ProbeAttempt_Result_PERMISSION_REQUIRED, "")

		res := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_VALID, res.Status.Binding.Validity,
			"a Session that cannot probe must not suspend the Binding")
		tstAssertBinding(t, res.Status.Binding, false, corev1.Device_Status_Binding_PROVIDER_ID)
		assert.Nil(t, env.getDeviceProof(t, sess))
	})

	t.Run("HardwareKeyIsKept", func(t *testing.T) {
		sess := env.createSession(t, dev)
		sess.Status.DeviceProof = &corev1.Session_Status_DeviceProof{
			Type: corev1.Session_Status_DeviceProof_HARDWARE_KEY,
		}
		_, err := env.octeliumC.CoreC().UpdateSession(env.ctx, sess)
		assert.Nil(t, err)

		submit(t, sess, corev1.Device_Status_ProbeAttempt_Result_OK, "agent-1")
		env.reconcile(t, dev)

		assert.Equal(t, corev1.Session_Status_DeviceProof_HARDWARE_KEY, env.getDeviceProof(t, sess).GetType())
	})

	t.Run("SessionOfAnotherDevice", func(t *testing.T) {
		other := env.createDevice(t, "SERIAL-2", "other@example.com")
		sess := env.createSession(t, other)

		submit(t, sess, corev1.Device_Status_ProbeAttempt_Result_OK, "agent-1")
		env.reconcile(t, dev)

		assert.Nil(t, env.getDeviceProof(t, sess))
	})

	t.Run("DeletedSession", func(t *testing.T) {
		sess := env.createSession(t, dev)
		_, err := env.octeliumC.CoreC().DeleteSession(env.ctx, &rmetav1.DeleteOptions{Uid: sess.Metadata.Uid})
		assert.Nil(t, err)

		submit(t, sess, corev1.Device_Status_ProbeAttempt_Result_OK, "agent-1")

		res := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, res.Status.ProbeAttempt.State)
	})

	t.Run("FailedUpdateIsRetried", func(t *testing.T) {
		sess := env.createSession(t, dev)
		submit(t, sess, corev1.Device_Status_ProbeAttempt_Result_OK, "agent-1")

		client := env.useCountingClient()
		t.Cleanup(func() {
			env.ctrl.octeliumC = env.octeliumC
		})

		client.core.updateSessionErr.Store(1)

		res := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_SUBMITTED, res.Status.ProbeAttempt.State,
			"the attempt must stay pending so that the Session can still get its proof")
		assert.Nil(t, env.getDeviceProof(t, sess))

		res = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_PROCESSED, res.Status.ProbeAttempt.State)
		assert.Equal(t, corev1.Session_Status_DeviceProof_PROBE, env.getDeviceProof(t, sess).GetType())
	})

	t.Run("Contradiction", func(t *testing.T) {
		sess := env.createSession(t, dev)
		submit(t, sess, corev1.Device_Status_ProbeAttempt_Result_OK, "agent-2")

		res := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_SUSPENDED, res.Status.Binding.Validity)
		assert.Equal(t, reasonVerificationContradicted, res.Status.Binding.Reason)
		assert.Nil(t, env.getDeviceProof(t, sess))
	})

	t.Run("NewBinding", func(t *testing.T) {
		owner := env.setOwner("dm-n", nil, tstEntry("n-1", "serial-n", "agent-n"))

		dev := env.createDevice(t, "", "new@example.com")
		sess := env.createSession(t, dev)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(owner, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(owner, corev1.Device_Status_ProbeAttempt_Result_OK, "agent-n"))
			dev.Status.ProbeAttempt.SessionRef = umetav1.GetObjectReference(sess)
		})

		res := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, res.Status.Binding.State)
		tstAssertBinding(t, res.Status.Binding, false, corev1.Device_Status_Binding_PROVIDER_ID)

		proof := env.getDeviceProof(t, sess)
		assert.Equal(t, corev1.Session_Status_DeviceProof_PROBE, proof.GetType())
		assert.Equal(t, res.Status.Binding.Uid, proof.GetBindingUID())
	})

	t.Run("HardwareProbeIsNoProof", func(t *testing.T) {
		platform := devicemgrcommon.NewOwner(&tstManager{isPlatformProbe: true},
			tstDeviceManager("dm-h", &enterprisev1.DeviceManager_Spec{}),
			devicemgrcommon.NewFleet([]*devicemgrcommon.Entry{tstEntry("h-1", "serial-h", "uuid-h")}),
			time.Now(), time.Hour)
		env.registry.SetOwner(platform)

		dev := env.createDevice(t, "", "hardware@example.com")
		sess := env.createSession(t, dev)

		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.ProbeAttempt = tstAttempt(platform, corev1.Device_Status_ProbeAttempt_SUBMITTED, time.Now(),
				tstResult(platform, corev1.Device_Status_ProbeAttempt_Result_OK, "uuid-h"))
			dev.Status.ProbeAttempt.SessionRef = umetav1.GetObjectReference(sess)
		})

		res := env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_ACCEPTED, res.Status.Binding.State)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{
			corev1.Device_Status_Binding_PROBE,
		}, res.Status.Binding.Evidence)
		tstAssertBinding(t, res.Status.Binding, false, corev1.Device_Status_Binding_IDENTIFIER)
		assert.Nil(t, env.getDeviceProof(t, sess),
			"a hardware UUID is not an identifier that the provider issued")
	})
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
			dev.Status.IsLocked = true
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
			dev.Status.Hostname = "host"
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
		dev.Status.IsLocked = true
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

	env.setOwner("dm-a", spec, tstOwnedEntry("a-1", "serial-1", "owner@example.com"),
		tstEntry("a-2", "serial-2"), tstOwnedEntry("a-3", "serial-3", "owner@example.com"))
	ownerB := env.setOwner("dm-b", nil, tstEntry("b-1", "serial-1"), tstEntry("b-2", "serial-2"))

	{
		dev := env.createDevice(t, "SERIAL-1", "owner@example.com")
		env.updateDevice(t, dev, func(dev *corev1.Device) {
			dev.Status.UserRef = nil
		})
		dev = env.reconcile(t, dev)
		assert.Equal(t, corev1.Device_Status_Binding_OWNER_MISMATCH, dev.Status.Binding.State)
		assert.Equal(t, "uid-dm-a", dev.Status.Binding.OwnerRef.Uid)
		assert.Equal(t, "a-1", dev.Status.Binding.ExternalID)
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

	env.setOwner("dm-a", spec, tstOwnedEntry("a-1", "serial-1", "owner@example.com"),
		tstOwnedEntry("a-2", "serial-2", "owner@example.com"),
		tstOwnedEntry("a-3", "serial-3", "owner@example.com"),
		tstOwnedEntry("a-4", "serial-4", "owner@example.com"))
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
			assert.Equal(t, corev1.Device_Status_Binding_OWNER_MISMATCH, dev.Status.Binding.State)
		case 4:
			assert.Equal(t, "uid-dm-b", dev.Status.Binding.OwnerRef.Uid)
		}
		bindingUIDs = append(bindingUIDs, dev.Status.GetBinding().GetUid())
	}

	for _, err := range env.ctrl.ReconcileDevices(env.ctx, devices) {
		assert.Nil(t, err, "%+v", err)
	}

	assert.Equal(t, int32(5), client.core.getDevice.Load())
	assert.Equal(t, int32(2), client.enterpriseUtils.getClusterConfig.Load())

	for i, dev := range devices {
		assert.Equal(t, bindingUIDs[i], env.getDevice(t, dev).Status.GetBinding().GetUid())
	}

	ctx, cancel := context.WithCancel(env.ctx)
	cancel()
	for _, err := range env.ctrl.ReconcileDevices(ctx, devices) {
		assert.ErrorIs(t, err, context.Canceled)
	}
	assert.Equal(t, int32(5), client.core.getDevice.Load())
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

	assert.True(t, posturesEqualIgnoringTimestamps(
		&corev1.Device_Status_Posture{AssessedAt: pbutils.Timestamp(time.Now())},
		&corev1.Device_Status_Posture{AssessedAt: pbutils.Timestamp(time.Now().Add(time.Hour))}),
		"a new assessment time alone must not write the Device")
	assert.False(t, posturesEqualIgnoringTimestamps(
		&corev1.Device_Status_Posture{Score: proto.Uint32(0)},
		&corev1.Device_Status_Posture{}),
		"a reported zero is different from an unreported score")
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
