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
	"slices"
	"sort"
	"strings"
	"testing"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/authv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	unreachableFleetURL  = "https://fleet.octelium-e2e.invalid"
	fleetCollectionError = "FleetDM request"
	fleetHardwareUUID    = "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"
)

func deviceManagerSecretRef(secretName string) *enterprisev1.DeviceManager_Spec_SecretRef {
	return &enterprisev1.DeviceManager_Spec_SecretRef{
		Type: &enterprisev1.DeviceManager_Spec_SecretRef_FromSecret{
			FromSecret: secretName,
		},
	}
}

func fleetDeviceManager(secretName string) *enterprisev1.DeviceManager {
	return &enterprisev1.DeviceManager{
		Spec: &enterprisev1.DeviceManager_Spec{
			Type: &enterprisev1.DeviceManager_Spec_FleetDM_{
				FleetDM: &enterprisev1.DeviceManager_Spec_FleetDM{
					BaseURL:  unreachableFleetURL,
					ApiToken: deviceManagerSecretRef(secretName),
				},
			},
			Polling: &enterprisev1.DeviceManager_Spec_Polling{
				Interval: eeharness.Seconds(30),
				Timeout:  eeharness.Seconds(10),
			},
		},
	}
}

func deviceManagerProbeIDs(ctx context.Context, h *eeharness.H,
	dm *enterprisev1.DeviceManager) ([]string, error) {
	cc, err := h.CoreC().GetClusterConfig(ctx, &corev1.GetClusterConfigRequest{})
	if err != nil {
		return nil, err
	}

	var ret []string
	for _, p := range cc.GetStatus().GetDevice().GetProbes() {
		if p.GetOwnerRef().GetUid() != dm.Metadata.Uid {
			continue
		}
		if !strings.HasPrefix(p.Id, dm.Metadata.Name+".") {
			return nil, errors.Errorf("the probe %s does not belong to the DeviceManager %s",
				p.Id, dm.Metadata.Name)
		}
		ret = append(ret, p.Id)
	}

	sort.Strings(ret)
	return ret, nil
}

func waitDeviceManagerProbes(t *testing.T, h *eeharness.H,
	dm *enterprisev1.DeviceManager, want ...string) {
	t.Helper()

	h.Eventually(t, "the Device probe plan to be published", eeharness.PropagationBudget,
		func(ctx context.Context) error {
			ids, err := deviceManagerProbeIDs(ctx, h, dm)
			if err != nil {
				return err
			}
			if !slices.Equal(ids, want) {
				return errors.Errorf("the probe plan of the DeviceManager is %v, want %v", ids, want)
			}
			return nil
		})
}

func waitDeviceManagerState(t *testing.T, h *eeharness.H, dm *enterprisev1.DeviceManager,
	state enterprisev1.DeviceManager_Status_State,
	check func(dm *enterprisev1.DeviceManager) error) *enterprisev1.DeviceManager {
	t.Helper()

	var ret *enterprisev1.DeviceManager
	h.Eventually(t, "the DeviceManager to reach its state", eeharness.SyncBudget,
		func(ctx context.Context) error {
			cur, err := h.EnterpriseC().GetDeviceManager(ctx, &metav1.GetOptions{Uid: dm.Metadata.Uid})
			if err != nil {
				return err
			}
			if cur.GetStatus().GetState() != state {
				return errors.Errorf("the DeviceManager state is %s, want %s",
					cur.GetStatus().GetState(), state)
			}
			if check != nil {
				if err := check(cur); err != nil {
					return err
				}
			}
			ret = cur
			return nil
		})

	return ret
}

func hasCollectionError(want bool) func(dm *enterprisev1.DeviceManager) error {
	return func(dm *enterprisev1.DeviceManager) error {
		lastError := dm.GetStatus().GetCollection().GetLastError()
		if lastError == "" {
			return errors.Errorf("the DeviceManager has no collection error")
		}
		if strings.Contains(lastError, fleetCollectionError) != want {
			return errors.Errorf("unexpected DeviceManager collection error: %s", lastError)
		}
		return nil
	}
}

func testDeviceManagerAPI(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	sec := h.CreateEnterpriseSecret(t, utilrand.GetRandomString(32))

	t.Run("InvalidSpecsAreRejected", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			fn   func(spec *enterprisev1.DeviceManager_Spec)
		}{
			{"PlainHTTPBaseURL", func(spec *enterprisev1.DeviceManager_Spec) {
				spec.GetFleetDM().BaseURL = "http://fleet.example.com"
			}},
			{"BaseURLWithCredentials", func(spec *enterprisev1.DeviceManager_Spec) {
				spec.GetFleetDM().BaseURL = "https://user:password@fleet.example.com"
			}},
			{"UnknownSecret", func(spec *enterprisev1.DeviceManager_Spec) {
				spec.GetFleetDM().ApiToken = deviceManagerSecretRef(utilrand.GetRandomStringCanonical(10))
			}},
			{"ProbeOnlyWithoutProbes", func(spec *enterprisev1.DeviceManager_Spec) {
				spec.Type = &enterprisev1.DeviceManager_Spec_Huntress_{
					Huntress: &enterprisev1.DeviceManager_Spec_Huntress{
						ApiKey:    "key",
						ApiSecret: deviceManagerSecretRef(sec.Metadata.Name),
					},
				}
				spec.Linking = &enterprisev1.DeviceManager_Spec_Linking{
					Strategy: enterprisev1.DeviceManager_Spec_Linking_PROBE_ONLY,
				}
			}},
			{"OwnerMatchWithoutOwners", func(spec *enterprisev1.DeviceManager_Spec) {
				spec.Linking = &enterprisev1.DeviceManager_Spec_Linking{
					RequireOwnerMatch: true,
				}
			}},
			{"AgreementWithoutProbes", func(spec *enterprisev1.DeviceManager_Spec) {
				spec.Linking = &enterprisev1.DeviceManager_Spec_Linking{
					Strategy:         enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY,
					RequireAgreement: true,
				}
			}},
			{"VerificationWithoutProbes", func(spec *enterprisev1.DeviceManager_Spec) {
				spec.Linking = &enterprisev1.DeviceManager_Spec_Linking{
					Strategy:             enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY,
					VerificationInterval: eeharness.Minutes(60),
				}
			}},
			{"StaleAfterShorterThanTheInterval", func(spec *enterprisev1.DeviceManager_Spec) {
				spec.Polling = &enterprisev1.DeviceManager_Spec_Polling{
					Interval:   eeharness.Minutes(10),
					StaleAfter: eeharness.Minutes(5),
				}
			}},
			{"InvalidCondition", func(spec *enterprisev1.DeviceManager_Spec) {
				spec.Condition = &corev1.Condition{
					Type: &corev1.Condition_Match{Match: "ctx.device.status.osType =="},
				}
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				dm := fleetDeviceManager(sec.Metadata.Name)
				dm.Metadata = &metav1.Metadata{Name: h.Name()}
				tc.fn(dm.Spec)

				_, err := h.EnterpriseC().CreateDeviceManager(t.Context(), dm)
				require.NotNil(t, err, "an invalid DeviceManager was accepted")
				assert.True(t, grpcerr.IsInvalidArg(err),
					"an invalid DeviceManager returned an unexpected error: %+v", err)
			})
		}
	})

	dm := h.CreateDeviceManager(t, fleetDeviceManager(sec.Metadata.Name))

	t.Run("TheStatusTypeFollowsTheSpec", func(t *testing.T) {
		assert.Equal(t, enterprisev1.DeviceManager_Status_FLEETDM, dm.Status.Type)

		cur, err := h.EnterpriseC().GetDeviceManager(t.Context(),
			&metav1.GetOptions{Name: dm.Metadata.Name})
		require.Nil(t, err)
		assert.Equal(t, dm.Metadata.Uid, cur.Metadata.Uid)

		list, err := h.EnterpriseC().ListDeviceManager(t.Context(),
			&enterprisev1.ListDeviceManagerOptions{})
		require.Nil(t, err)
		assert.True(t, slices.ContainsFunc(list.Items, func(itm *enterprisev1.DeviceManager) bool {
			return itm.Metadata.Uid == dm.Metadata.Uid
		}))
	})

	t.Run("AnUpdateKeepsTheStatus", func(t *testing.T) {
		cur, err := h.EnterpriseC().GetDeviceManager(t.Context(), &metav1.GetOptions{Uid: dm.Metadata.Uid})
		require.Nil(t, err)

		cur.Spec.GetFleetDM().TeamID = 7
		updated := h.UpdateDeviceManager(t, cur)
		assert.Equal(t, uint32(7), updated.Spec.GetFleetDM().TeamID)
		assert.Equal(t, enterprisev1.DeviceManager_Status_FLEETDM, updated.Status.Type)

		cur.Spec.GetFleetDM().BaseURL = "http://fleet.example.com"
		_, err = h.EnterpriseC().UpdateDeviceManager(t.Context(), cur)
		require.NotNil(t, err, "an invalid DeviceManager update was accepted")
		assert.True(t, grpcerr.IsInvalidArg(err))
	})

	t.Run("TheClusterConfigOrderIsValidated", func(t *testing.T) {
		before := h.EnterpriseClusterConfig(t).Spec.DeviceManagers
		t.Cleanup(func() {
			cur := h.EnterpriseClusterConfig(t)
			cur.Spec.DeviceManagers = before
			h.UpdateEnterpriseClusterConfig(t, cur)
		})

		for _, names := range [][]string{
			{utilrand.GetRandomStringCanonical(10)},
			{dm.Metadata.Name, dm.Metadata.Name},
		} {
			cc := h.EnterpriseClusterConfig(t)
			cc.Spec.DeviceManagers = names

			_, err := h.EnterpriseC().UpdateClusterConfig(t.Context(), cc)
			require.NotNil(t, err, "the invalid deviceManagers %v were accepted", names)
			assert.True(t, grpcerr.IsInvalidArg(err))
		}

		cc := h.EnterpriseClusterConfig(t)
		cc.Spec.DeviceManagers = append(slices.Clone(before), dm.Metadata.Name)
		updated := h.UpdateEnterpriseClusterConfig(t, cc)
		assert.Contains(t, updated.Spec.DeviceManagers, dm.Metadata.Name)

		_, err := h.EnterpriseC().DeleteDeviceManager(t.Context(),
			&metav1.DeleteOptions{Uid: dm.Metadata.Uid})
		require.NotNil(t, err, "a DeviceManager used by the ClusterConfig was deleted")
		assert.True(t, grpcerr.IsInvalidArg(err))
	})

	t.Run("ResetDeviceBindingIsValidated", func(t *testing.T) {
		_, err := h.EnterpriseC().ResetDeviceBinding(t.Context(), &enterprisev1.ResetDeviceBindingRequest{
			BindingUID: utilrand.GetRandomStringCanonical(32),
		})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))

		_, err = h.EnterpriseC().ResetDeviceBinding(t.Context(), &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef:  &metav1.ObjectReference{Name: utilrand.GetRandomStringCanonical(10)},
			BindingUID: "Invalid UID",
		})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))

		_, err = h.EnterpriseC().ResetDeviceBinding(t.Context(), &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef:  &metav1.ObjectReference{Name: utilrand.GetRandomStringCanonical(10)},
			BindingUID: utilrand.GetRandomStringCanonical(32),
		})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err))
	})

	t.Run("AnUnusedDeviceManagerIsDeleted", func(t *testing.T) {
		_, err := h.EnterpriseC().DeleteDeviceManager(t.Context(),
			&metav1.DeleteOptions{Uid: dm.Metadata.Uid})
		require.Nil(t, err)

		_, err = h.EnterpriseC().GetDeviceManager(t.Context(), &metav1.GetOptions{Uid: dm.Metadata.Uid})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsNotFound(err))
	})
}

func testDeviceManagerProbePlan(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	sec := h.CreateEnterpriseSecret(t, utilrand.GetRandomString(32))
	dm := h.CreateDeviceManager(t, fleetDeviceManager(sec.Metadata.Name))

	linuxProbeID := dm.Metadata.Name + ".hardware-uuid-linux"

	t.Run("ThePlanIsPublished", func(t *testing.T) {
		waitDeviceManagerProbes(t, h, dm, dm.Metadata.Name+".hardware-uuid", linuxProbeID)
	})

	t.Run("AFailingCollectionIsReported", func(t *testing.T) {
		cur := waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_ERROR,
			func(dm *enterprisev1.DeviceManager) error {
				if err := hasCollectionError(true)(dm); err != nil {
					return err
				}
				if dm.GetStatus().GetLinking().GetLastSweepAt() == nil {
					return errors.Errorf("the Devices have not been swept yet")
				}
				return nil
			})

		assert.Equal(t, enterprisev1.DeviceManager_Status_FLEETDM, cur.Status.Type)
		assert.NotNil(t, cur.Status.Collection.LastAttemptAt)
		assert.Nil(t, cur.Status.Collection.LastSuccessAt)
		assert.Zero(t, cur.Status.Linking.LinkedDevices)
	})

	t.Run("AClientRunsTheProbes", func(t *testing.T) {
		sess := h.NewAuthSession(t, harness.AuthSessionOpts{
			SessionType: corev1.Session_Status_CLIENT,
		})

		_, err := sess.C().RegisterDevice(sess.Ctx(t.Context()), &authv1.RegisterDeviceRequest{
			Info: &authv1.RegisterDeviceRequest_Info{
				OsType:       authv1.RegisterDeviceRequest_Info_LINUX,
				Hostname:     utilrand.GetRandomStringCanonical(8),
				Id:           utilrand.GetRandomStringHex(64),
				SerialNumber: utilrand.GetRandomStringCanonical(12),
			},
		})
		require.Nil(t, err, "could not register the Device")

		deviceRef := sess.Session(t).Status.DeviceRef
		require.NotNil(t, deviceRef)

		var begin *authv1.RunDeviceProbeBeginResponse
		h.Eventually(t, "the DeviceManager probes to be issued", eeharness.PropagationBudget,
			func(ctx context.Context) error {
				resp, err := sess.C().RunDeviceProbeBegin(sess.Ctx(ctx), &authv1.RunDeviceProbeBeginRequest{})
				if err != nil {
					return err
				}
				if !slices.ContainsFunc(resp.Probes, func(p *authv1.DeviceProbe) bool {
					return p.ProbeID == linuxProbeID
				}) {
					return errors.Errorf("the probe %s has not been issued", linuxProbeID)
				}
				begin = resp
				return nil
			})

		require.NotEmpty(t, begin.AttemptUID)

		var results []*authv1.DeviceProbeResult
		for _, p := range begin.Probes {
			assert.NotEqual(t, dm.Metadata.Name+".hardware-uuid", p.ProbeID,
				"a probe for another OS type was issued")

			if p.ProbeID != linuxProbeID {
				results = append(results, &authv1.DeviceProbeResult{
					ProbeID: p.ProbeID,
					Status:  authv1.DeviceProbeResult_NOT_FOUND,
				})
				continue
			}

			assert.True(t, p.RequireElevation)
			assert.Equal(t, authv1.DeviceProbe_PlatformIdentifier_HARDWARE_UUID, p.GetPlatformIdentifier().GetKind())

			results = append(results, &authv1.DeviceProbeResult{
				ProbeID: p.ProbeID,
				Status:  authv1.DeviceProbeResult_OK,
				Value:   &authv1.DeviceProbeResult_Text{Text: fleetHardwareUUID},
			})
		}

		finish := &authv1.RunDeviceProbeFinishRequest{
			AttemptUID: begin.AttemptUID,
			Results:    results,
		}

		_, err = sess.C().RunDeviceProbeFinish(sess.Ctx(t.Context()), finish)
		require.Nil(t, err, "could not finish the Device probing")

		_, err = sess.C().RunDeviceProbeFinish(sess.Ctx(t.Context()), finish)
		require.Nil(t, err, "retrying an identical probe submission must succeed")

		retry := pbutils.Clone(finish).(*authv1.RunDeviceProbeFinishRequest)
		for _, r := range retry.Results {
			if r.ProbeID == linuxProbeID {
				r.Value = &authv1.DeviceProbeResult_Text{Text: strings.ToUpper(fleetHardwareUUID)}
			}
		}
		_, err = sess.C().RunDeviceProbeFinish(sess.Ctx(t.Context()), retry)
		require.NotNil(t, err, "a different probe submission for the same attempt must be rejected")
		assert.True(t, grpcerr.IsInvalidArg(err))

		dev, err := h.CoreC().GetDevice(t.Context(), &metav1.GetOptions{Uid: deviceRef.Uid})
		require.Nil(t, err)

		attempt := dev.Status.ProbeAttempt
		require.NotNil(t, attempt)
		assert.Equal(t, begin.AttemptUID, attempt.Uid)
		assert.Equal(t, corev1.Device_Status_ProbeAttempt_SUBMITTED, attempt.State)
		assert.True(t, slices.ContainsFunc(attempt.Results, func(r *corev1.Device_Status_ProbeAttempt_Result) bool {
			return r.ProbeID == linuxProbeID && r.GetText() == fleetHardwareUUID
		}))
		assert.Nil(t, dev.Status.Binding,
			"a Device must not be bound to a DeviceManager that has no inventory")

		resp, err := sess.C().RunDeviceProbeBegin(sess.Ctx(t.Context()), &authv1.RunDeviceProbeBeginRequest{})
		require.Nil(t, err)
		assert.Empty(t, resp.AttemptUID, "a submitted attempt must not be issued again")

		_, err = h.EnterpriseC().ResetDeviceBinding(t.Context(), &enterprisev1.ResetDeviceBindingRequest{
			DeviceRef:  deviceRef,
			BindingUID: utilrand.GetRandomStringCanonical(32),
		})
		assert.Nil(t, err, "resetting an unbound Device must succeed")
	})

	t.Run("ADisabledDeviceManagerIsWithdrawn", func(t *testing.T) {
		cur, err := h.EnterpriseC().GetDeviceManager(t.Context(), &metav1.GetOptions{Uid: dm.Metadata.Uid})
		require.Nil(t, err)

		cur.Spec.Polling.IsDisabled = true
		h.UpdateDeviceManager(t, cur)

		waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_DISABLED, nil)
		waitDeviceManagerProbes(t, h, dm)
	})

	t.Run("AReEnabledDeviceManagerIsRepublished", func(t *testing.T) {
		cur, err := h.EnterpriseC().GetDeviceManager(t.Context(), &metav1.GetOptions{Uid: dm.Metadata.Uid})
		require.Nil(t, err)

		cur.Spec.Polling.IsDisabled = false
		h.UpdateDeviceManager(t, cur)

		waitDeviceManagerProbes(t, h, dm, dm.Metadata.Name+".hardware-uuid", linuxProbeID)
		waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_ERROR, hasCollectionError(true))
	})

	t.Run("AnIdentityOnlyDeviceManagerHasNoProbes", func(t *testing.T) {
		cur, err := h.EnterpriseC().GetDeviceManager(t.Context(), &metav1.GetOptions{Uid: dm.Metadata.Uid})
		require.Nil(t, err)

		cur.Spec.Linking = &enterprisev1.DeviceManager_Spec_Linking{
			Strategy: enterprisev1.DeviceManager_Spec_Linking_IDENTITY_ONLY,
		}
		h.UpdateDeviceManager(t, cur)

		waitDeviceManagerProbes(t, h, dm)
	})

	t.Run("ADeletedDeviceManagerIsRemovedFromThePlan", func(t *testing.T) {
		cur, err := h.EnterpriseC().GetDeviceManager(t.Context(), &metav1.GetOptions{Uid: dm.Metadata.Uid})
		require.Nil(t, err)

		cur.Spec.Linking = nil
		h.UpdateDeviceManager(t, cur)
		waitDeviceManagerProbes(t, h, dm, dm.Metadata.Name+".hardware-uuid", linuxProbeID)

		_, err = h.EnterpriseC().DeleteDeviceManager(t.Context(), &metav1.DeleteOptions{Uid: dm.Metadata.Uid})
		require.Nil(t, err)

		waitDeviceManagerProbes(t, h, dm)
	})
}

func testDeviceManagerSecretRotation(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	sec := h.CreateEnterpriseSecret(t, utilrand.GetRandomString(32))
	dm := h.CreateDeviceManager(t, fleetDeviceManager(sec.Metadata.Name))

	t.Run("TheCollectionRuns", func(t *testing.T) {
		waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_ERROR, hasCollectionError(true))
	})

	t.Run("ADeletedSecretStopsTheCollection", func(t *testing.T) {
		_, err := h.EnterpriseC().DeleteSecret(t.Context(), &metav1.DeleteOptions{Uid: sec.Metadata.Uid})
		require.Nil(t, err)

		waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_ERROR, hasCollectionError(false))
	})

	t.Run("ARecreatedSecretResumesTheCollection", func(t *testing.T) {
		created, err := h.EnterpriseC().CreateSecret(t.Context(), &enterprisev1.Secret{
			Metadata: &metav1.Metadata{Name: sec.Metadata.Name},
			Spec:     &enterprisev1.Secret_Spec{},
			Data: &enterprisev1.Secret_Data{
				Type: &enterprisev1.Secret_Data_Value{Value: utilrand.GetRandomString(32)},
			},
		})
		require.Nil(t, err)

		t.Cleanup(func() {
			if _, err := h.EnterpriseC().DeleteSecret(context.Background(),
				&metav1.DeleteOptions{Uid: created.Metadata.Uid}); err != nil && !grpcerr.IsNotFound(err) {
				t.Logf("Could not delete the Secret %s: %+v", created.Metadata.Name, err)
			}
		})

		waitDeviceManagerState(t, h, dm, enterprisev1.DeviceManager_Status_ERROR, hasCollectionError(true))
	})
}
