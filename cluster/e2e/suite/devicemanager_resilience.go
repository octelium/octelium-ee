// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
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

const fleetPollInterval = 30 * time.Second

func testDeviceManagerResilience(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	hosts := []eeharness.FleetHost{
		{
			ID:                    1,
			UUID:                  vutils.UUIDv4(),
			HardwareSerial:        h.Name(),
			Platform:              "linux",
			Status:                "online",
			DiskEncryptionEnabled: true,
		},
	}
	token := utilrand.GetRandomStringCanonical(32)
	sec := h.CreateEnterpriseSecret(t, token)
	fleet := h.Fleet(t, token, hosts)
	fleet.SetFailures(http.StatusServiceUnavailable, http.StatusTooManyRequests, http.StatusBadGateway)

	dmReq := fleetDeviceManager(sec.Metadata.Name)
	dmReq.Spec.GetFleetDM().BaseURL = fleet.URL
	dmReq.Spec.Polling.Interval = eeharness.Seconds(uint32(fleetPollInterval / time.Second))
	dmReq.Spec.Polling.Timeout = eeharness.Seconds(25)
	dm := h.CreateDeviceManager(t, dmReq)

	unavailableReq := fleetDeviceManager(sec.Metadata.Name)
	unavailableReq.Spec.Condition = &corev1.Condition{
		Type: &corev1.Condition_Match{
			Match: `ctx.device.status.osType == "WINDOWS"`,
		},
	}
	unavailable := h.CreateDeviceManager(t, unavailableReq)

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
	cc.Spec.DeviceManagers = []string{unavailable.Metadata.Name, dm.Metadata.Name}
	h.UpdateEnterpriseClusterConfig(t, cc)

	t.Run("TransientProviderErrorsAreRetried", func(t *testing.T) {
		cur := waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_OK,
			func(cur *enterprisev1.DeviceManager) error {
				if cur.GetStatus().GetCollection().GetManagedDevices() != uint32(len(hosts)) {
					return errors.Errorf("the Fleet inventory has not been collected")
				}
				return nil
			})
		assert.Empty(t, cur.Status.Collection.LastError)
		assert.Zero(t, fleet.PendingFailures())
		assert.Less(t, cur.Status.Collection.LastSuccessAt.AsTime().Sub(dm.Metadata.CreatedAt.AsTime()),
			fleetPollInterval, "the transient errors were not retried within the first collection")

		waitDeviceManagerState(t, h, unavailable, enterprisev1.DeviceManager_Status_ERROR, hasCollectionError(true))
	})

	sess := h.NewAuthSession(t, harness.AuthSessionOpts{
		UserType:    corev1.User_Spec_HUMAN,
		SessionType: corev1.Session_Status_CLIENT,
	})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	_, err := sess.C().RegisterDevice(sess.Ctx(ctx), &authv1.RegisterDeviceRequest{
		Info: &authv1.RegisterDeviceRequest_Info{
			OsType:       authv1.RegisterDeviceRequest_Info_LINUX,
			Hostname:     h.Name(),
			Id:           utilrand.GetRandomStringHex(64),
			SerialNumber: hosts[0].HardwareSerial,
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

	var bindingUID string

	t.Run("AnInapplicableUnavailableManagerDoesNotBlockSelection", func(t *testing.T) {
		bound := waitFleetDevice(t, h, deviceRef, dm,
			corev1.Device_Status_Binding_VALID, corev1.Device_Status_Posture_PASS)
		assert.Equal(t, strings.ToLower(hosts[0].UUID), bound.Status.Binding.ExternalID)
		assert.False(t, bound.Status.Binding.IsOwnerMatched)
		assert.Equal(t, corev1.Device_Status_Binding_IDENTIFIER, bound.Status.Binding.Assurance)
		assert.Equal(t, []corev1.Device_Status_Binding_Evidence{corev1.Device_Status_Binding_SERIAL_NUMBER},
			bound.Status.Binding.Evidence)
		bindingUID = bound.Status.Binding.Uid
	})

	t.Run("ResetDeviceBindingRebindsTheDevice", func(t *testing.T) {
		require.NotEmpty(t, bindingUID)

		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		_, err := h.EnterpriseC().ResetDeviceBinding(ctx, &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef:  deviceRef,
			BindingUID: bindingUID,
		})
		cancel()
		require.Nil(t, err)

		h.Eventually(t, "the reset Device to be bound again", eeharness.SyncBudget,
			func(ctx context.Context) error {
				cur, err := h.CoreC().GetDevice(ctx, &metav1.GetOptions{Uid: deviceRef.Uid})
				if err != nil {
					return err
				}
				binding := cur.GetStatus().GetBinding()
				if binding.GetUid() == "" || binding.GetUid() == bindingUID {
					return errors.Errorf("the Device has not been bound again")
				}
				if binding.GetOwnerRef().GetUid() != dm.Metadata.Uid ||
					binding.GetState() != corev1.Device_Status_Binding_ACCEPTED ||
					binding.GetValidity() != corev1.Device_Status_Binding_VALID {
					return errors.Errorf("the replacement Device binding is not valid")
				}
				return nil
			})

		ctx, cancel = context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		_, err = h.EnterpriseC().ResetDeviceBinding(ctx, &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef:  deviceRef,
			BindingUID: bindingUID,
		})
		require.NotNil(t, err, "a stale Binding UID must not reset the replacement Binding")
		assert.True(t, grpcerr.IsInvalidArg(err))
	})

	t.Run("ACrossOriginRedirectIsRejected", func(t *testing.T) {
		fleet.SetRedirect(true)
		waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_ERROR,
			func(cur *enterprisev1.DeviceManager) error {
				if !strings.Contains(cur.GetStatus().GetCollection().GetLastError(), "another origin") {
					return errors.Errorf("the redirect was not rejected: %s",
						cur.GetStatus().GetCollection().GetLastError())
				}
				return nil
			})

		fleet.SetRedirect(false)
		waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_OK, nil)
		waitFleetDevice(t, h, deviceRef, dm,
			corev1.Device_Status_Binding_VALID, corev1.Device_Status_Posture_PASS)
	})
}
