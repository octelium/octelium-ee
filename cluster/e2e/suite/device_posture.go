package suite

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/authv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fleetStaleAfter = 90 * time.Second

func testDeviceManagerPostureLifecycle(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	var hosts []eeharness.FleetHost
	for i := range 101 {
		hosts = append(hosts, eeharness.FleetHost{
			ID:                    int64(i + 1),
			UUID:                  vutils.UUIDv4(),
			HardwareSerial:        fmt.Sprintf("%s-%d", h.Name(), i),
			Platform:              "linux",
			Status:                "online",
			DiskEncryptionEnabled: true,
		})
	}
	hosts[100].UUID = fleetHardwareUUID
	token := utilrand.GetRandomStringCanonical(32)
	sec := h.CreateEnterpriseSecret(t, token)
	fleet := h.Fleet(t, token, hosts)

	dmReq := fleetDeviceManager(sec.Metadata.Name)
	dmReq.Spec.GetFleetDM().BaseURL = fleet.URL
	dmReq.Spec.Polling.StaleAfter = eeharness.Seconds(uint32(fleetStaleAfter / time.Second))
	dmReq.Spec.Linking = &enterprisev1.DeviceManager_Spec_Linking{
		Strategy: enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY,
	}
	dm := h.CreateDeviceManager(t, dmReq)
	cc := h.EnterpriseClusterConfig(t)
	before := slices.Clone(cc.Spec.DeviceManagers)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cur, err := h.EnterpriseC().GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
		require.Nil(t, err)
		cur.Spec.DeviceManagers = before
		_, err = h.EnterpriseC().UpdateClusterConfig(ctx, cur)
		require.Nil(t, err)
	})
	cc.Spec.DeviceManagers = []string{dm.Metadata.Name}
	h.UpdateEnterpriseClusterConfig(t, cc)

	linuxProbeID := dm.Metadata.Name + ".hardware-uuid-linux"
	waitDeviceManagerProbes(t, h, dm, dm.Metadata.Name+".hardware-uuid", linuxProbeID)
	waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_OK,
		func(cur *enterprisev1.DeviceManager) error {
			if cur.GetStatus().GetCollection().GetManagedDevices() != uint32(len(hosts)) {
				return errors.Errorf("the Fleet inventory is incomplete")
			}
			return nil
		})

	svc := h.NewPublicService(t, "default")
	svc.Spec.Authorization = &corev1.Service_Spec_Authorization{
		InlinePolicies: []*corev1.InlinePolicy{
			{
				Name: "device-posture",
				Spec: &corev1.Policy_Spec{
					Rules: []*corev1.Policy_Spec_Rule{
						{
							Name:   "encrypted-device",
							Effect: corev1.Policy_Spec_Rule_ALLOW,
							Condition: &corev1.Condition{
								Type: &corev1.Condition_Match{
									Match: `has(ctx.device.status.posture) && ` +
										`ctx.device.status.posture.diskEncryption == "PASS" && ` +
										`ctx.device.status.posture.agentHealthy == "PASS"`,
								},
							},
						},
					},
				},
			},
		},
	}
	svc = h.UpdateService(t, svc)

	sess := h.NewAuthSession(t, harness.AuthSessionOpts{
		SessionType: corev1.Session_Status_CLIENT,
	})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	_, err := sess.C().RegisterDevice(sess.Ctx(ctx), &authv1.RegisterDeviceRequest{
		Info: &authv1.RegisterDeviceRequest_Info{
			OsType:       authv1.RegisterDeviceRequest_Info_LINUX,
			Hostname:     h.Name(),
			Id:           utilrand.GetRandomStringHex(64),
			SerialNumber: hosts[100].HardwareSerial,
		},
	})
	cancel()
	require.Nil(t, err)
	deviceRef := sess.Session(t).Status.DeviceRef
	require.NotNil(t, deviceRef)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := h.CoreC().DeleteDevice(ctx, &metav1.DeleteOptions{Uid: deviceRef.Uid})
		if err != nil && !grpcerr.IsNotFound(err) {
			t.Errorf("Could not delete the Fleet Device: %+v", err)
		}
	})

	client := h.ServiceClient(svc, sess.Token().AccessToken)
	h.WaitDenied(t, client)
	submitFleetProbe(t, h, sess, linuxProbeID)
	bound := waitFleetDevice(t, h, deviceRef, dm,
		corev1.Device_Status_Binding_VALID, corev1.Device_Status_Posture_PASS)
	bindingUID := bound.Status.Binding.Uid
	require.Equal(t, fleetHardwareUUID, bound.Status.Binding.ExternalID)
	h.WaitAllowed(t, client)

	t.Run("AClientlessSessionCannotUseTheDevicePosture", func(t *testing.T) {
		h.Probe(t, sess.User, svc).MustBeDenied(t)
	})

	t.Run("APostureDowngradeRevokesAccess", func(t *testing.T) {
		hosts[100].DiskEncryptionEnabled = false
		fleet.SetHosts(hosts)
		cur := waitFleetDevice(t, h, deviceRef, dm,
			corev1.Device_Status_Binding_VALID, corev1.Device_Status_Posture_FAIL)
		assert.Equal(t, bindingUID, cur.Status.Binding.Uid)
		h.WaitDenied(t, client)

		hosts[100].DiskEncryptionEnabled = true
		fleet.SetHosts(hosts)
		waitFleetDevice(t, h, deviceRef, dm,
			corev1.Device_Status_Binding_VALID, corev1.Device_Status_Posture_PASS)
		h.WaitAllowed(t, client)
	})

	t.Run("AProviderCredentialOutageCannotExtendThePosture", func(t *testing.T) {
		next := utilrand.GetRandomStringCanonical(32)
		fleet.SetBearer(next)
		failed := waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_ERROR,
			func(cur *enterprisev1.DeviceManager) error {
				if !strings.Contains(cur.GetStatus().GetCollection().GetLastError(), "401") {
					return errors.Errorf("the collection did not fail with an authentication error")
				}
				return nil
			})
		require.NotNil(t, failed.Status.Collection.LastSuccessAt)

		h.Eventually(t, "the last successful Fleet inventory to expire", eeharness.IngestionBudget,
			func(ctx context.Context) error {
				if time.Now().Before(failed.Status.Collection.LastSuccessAt.AsTime().Add(fleetStaleAfter)) {
					return errors.Errorf("the Fleet inventory has not expired")
				}
				return nil
			})
		h.WaitDenied(t, client)
		h.Consistently(t, "the stale Device posture to stay denied", 5*time.Second,
			func(ctx context.Context) error {
				got, err := h.StatusOf(ctx, client, "/")
				if err != nil {
					return err
				}
				if got != http.StatusForbidden {
					return errUnexpectedStatus(got, http.StatusForbidden)
				}
				return nil
			})

		sec = h.UpdateEnterpriseSecret(t, sec, next)
		waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_OK, nil)
		cur := waitFleetDevice(t, h, deviceRef, dm,
			corev1.Device_Status_Binding_VALID, corev1.Device_Status_Posture_PASS)
		assert.Equal(t, bindingUID, cur.Status.Binding.Uid)
		h.WaitAllowed(t, client)
	})

	t.Run("InventoryLossAndRestartPreserveTheBinding", func(t *testing.T) {
		fleet.SetHosts(hosts[:100])
		lost := waitFleetDevice(t, h, deviceRef, dm,
			corev1.Device_Status_Binding_LOST, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN)
		assert.Equal(t, bindingUID, lost.Status.Binding.Uid)
		h.WaitDenied(t, client)

		restartedAt := time.Now()
		h.RestartEnterprise(t, "nocturne")
		waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_OK,
			func(cur *enterprisev1.DeviceManager) error {
				collection := cur.GetStatus().GetCollection()
				if collection.GetManagedDevices() != 100 ||
					!collection.GetLastSuccessAt().AsTime().After(restartedAt) {
					return errors.Errorf("the restarted DeviceManager has not collected the reduced inventory")
				}
				return nil
			})
		cur := waitFleetDevice(t, h, deviceRef, dm,
			corev1.Device_Status_Binding_LOST, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN)
		assert.Equal(t, bindingUID, cur.Status.Binding.Uid)
		h.WaitDenied(t, client)

		fleet.SetHosts(hosts)
		cur = waitFleetDevice(t, h, deviceRef, dm,
			corev1.Device_Status_Binding_VALID, corev1.Device_Status_Posture_PASS)
		assert.Equal(t, bindingUID, cur.Status.Binding.Uid)
		h.WaitAllowed(t, client)
	})
}

func submitFleetProbe(t *testing.T, h *eeharness.H, sess *harness.AuthSession, probeID string) {
	t.Helper()

	var begin *authv1.RunDeviceProbeBeginResponse
	h.Eventually(t, "the Fleet identity probe to be issued", eeharness.PropagationBudget,
		func(ctx context.Context) error {
			cur, err := sess.C().RunDeviceProbeBegin(sess.Ctx(ctx), &authv1.RunDeviceProbeBeginRequest{})
			if err != nil {
				return err
			}
			if cur.AttemptUID == "" || !slices.ContainsFunc(cur.Probes, func(p *authv1.DeviceProbe) bool {
				return p.ProbeID == probeID
			}) {
				return errors.Errorf("the Fleet identity probe is missing")
			}
			begin = cur
			return nil
		})

	finish := &authv1.RunDeviceProbeFinishRequest{AttemptUID: begin.AttemptUID}
	for _, p := range begin.Probes {
		r := &authv1.DeviceProbeResult{ProbeID: p.ProbeID, Status: authv1.DeviceProbeResult_NOT_FOUND}
		if p.ProbeID == probeID {
			r.Status = authv1.DeviceProbeResult_OK
			r.Value = &authv1.DeviceProbeResult_Text{Text: fleetHardwareUUID}
		}
		finish.Results = append(finish.Results, r)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	_, err := sess.C().RunDeviceProbeFinish(sess.Ctx(ctx), finish)
	require.Nil(t, err)
}

func waitFleetDevice(t *testing.T, h *eeharness.H, ref *metav1.ObjectReference,
	dm *enterprisev1.DeviceManager, validity corev1.Device_Status_Binding_Validity,
	signal corev1.Device_Status_Posture_SignalState) *corev1.Device {
	t.Helper()

	var ret *corev1.Device
	h.Eventually(t, "the Fleet Device binding and posture to converge", eeharness.SyncBudget,
		func(ctx context.Context) error {
			cur, err := h.CoreC().GetDevice(ctx, &metav1.GetOptions{Uid: ref.Uid})
			if err != nil {
				return err
			}
			binding := cur.GetStatus().GetBinding()
			if binding.GetOwnerRef().GetUid() != dm.Metadata.Uid ||
				binding.GetState() != corev1.Device_Status_Binding_ACCEPTED || binding.GetValidity() != validity {
				return errors.Errorf("the Fleet Device binding is not %s", validity)
			}
			posture := cur.GetStatus().GetPosture()
			if signal == corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN {
				if posture != nil {
					return errors.Errorf("the lost Device binding still has a posture")
				}
			} else if posture == nil || posture.DiskEncryption != signal ||
				!posture.GetExpiresAt().AsTime().After(time.Now()) {
				return errors.Errorf("the Fleet Device has no fresh %s disk encryption posture", signal)
			}
			ret = cur
			return nil
		})
	return ret
}
