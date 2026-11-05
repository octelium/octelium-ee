// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package jamf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestParseExternalID(t *testing.T) {
	m := &Manager{}

	for _, tc := range []struct {
		osType  corev1.Device_Status_OSType
		results []*devicemgrcommon.ProbeResult
		want    string
	}{
		{corev1.Device_Status_MAC, []*devicemgrcommon.ProbeResult{{Text: "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D\n"}},
			"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
		{corev1.Device_Status_MAC, []*devicemgrcommon.ProbeResult{nil, {Text: "invalid"}}, ""},
		{corev1.Device_Status_WINDOWS, []*devicemgrcommon.ProbeResult{{Text: "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"}}, ""},
	} {
		ret, err := m.ParseExternalID(tc.osType, tc.results)
		assert.Nil(t, err)
		assert.Equal(t, tc.want, ret)
	}

	probes := m.IdentityProbes()
	assert.Len(t, probes, 1)
	assert.Equal(t, corev1.Device_Status_MAC, probes[0].OSType)
	assert.Equal(t, corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
		probes[0].PlatformIdentifier.Kind)
}

func TestToEntry(t *testing.T) {
	now := time.Now()

	getComputer := func(arg string) *computer {
		ret := &computer{}
		assert.Nil(t, json.Unmarshal([]byte(arg), ret))
		return ret
	}

	m := &Manager{
		compliantGroups: []string{"Compliant Macs"},
	}

	{
		entry := m.toEntry(getComputer(fmt.Sprintf(`{
			"id": "1",
			"udid": "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D",
			"general": {
				"name": "Alice MacBook Pro",
				"lastContactTime": %q,
				"remoteManagement": {"managed": true}
			},
			"hardware": {
				"serialNumber": "C02XYZ",
				"macAddress": "a4:bb:cc:dd:ee:01",
				"altMacAddress": "A4:BB:CC:DD:EE:01"
			},
			"userAndLocation": {"email": "user@example.com"},
			"security": {
				"sipStatus": "ENABLED",
				"gatekeeperStatus": "APP_STORE_AND_IDENTIFIED_DEVELOPERS",
				"firewallEnabled": true,
				"secureBootLevel": "FULL_SECURITY",
				"externalBootLevel": "DISALLOW_BOOTING_FROM_EXTERNAL_MEDIA",
				"autoLoginDisabled": true,
				"remoteDesktopEnabled": false
			},
			"diskEncryption": {
				"bootPartitionEncryptionDetails": {"partitionFileVault2State": "ENCRYPTED"}
			},
			"groupMemberships": [
				{"groupId": "1", "groupName": "All Managed Clients", "smartGroup": true},
				{"groupId": "2", "groupName": "Compliant Macs", "smartGroup": true}
			]
		}`, now.Add(-time.Hour).UTC().Format(time.RFC3339))), now)

		assert.Equal(t, "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", entry.ExternalID)
		assert.Equal(t, "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D", entry.MachineID)
		assert.Equal(t, "C02XYZ", entry.Serial)
		assert.Equal(t, []string{"a4:bb:cc:dd:ee:01"}, entry.MACs)
		assert.Equal(t, []string{"user@example.com"}, entry.OwnerEmails)
		assert.Equal(t, "Alice MacBook Pro", entry.Hostname)
		assert.Equal(t, corev1.Device_Status_MAC, entry.OSType)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Firewall)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.SecureBoot)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Enrolled)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.OsIntegrity)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AppControl)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.Equal(t, corev1.Device_Status_MAC, p.Os.Type)
		assert.Equal(t, corev1.Device_Status_Posture_WORKSTATION, p.DeviceClass)
		assert.Equal(t, "DISALLOW_BOOTING_FROM_EXTERNAL_MEDIA", p.Attrs.Fields["externalBootLevel"].GetStringValue())
		assert.True(t, p.Attrs.Fields["autoLoginDisabled"].GetBoolValue())
		assert.False(t, p.Attrs.Fields["remoteDesktopEnabled"].GetBoolValue())
		assert.NotNil(t, p.Attrs.Fields["remoteDesktopEnabled"])
	}

	{
		p := m.toEntry(getComputer(`{
			"udid": "u",
			"general": {"enrolledViaAutomatedDeviceEnrollment": true},
			"hardware": {"model": "Apple Virtual Machine 1"},
			"operatingSystem": {"version": "14.5.1"}
		}`), now).Posture

		assert.Equal(t, corev1.Device_Status_Posture_COMPANY, p.Ownership)
		assert.Equal(t, corev1.Device_Status_Posture_VIRTUAL_MACHINE, p.DeviceClass)
		assert.EqualValues(t, 14, p.Os.GetMajor())
		assert.EqualValues(t, 5, p.Os.GetMinor())
		assert.EqualValues(t, 1, p.Os.GetPatch())
	}

	{
		entry := m.toEntry(getComputer(`{
			"udid": "u",
			"groupMemberships": [
				{"groupId": "2", "groupName": "Compliant Macs", "smartGroup": false}
			]
		}`), now)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Firewall)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.SecureBoot)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Enrolled)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.OsIntegrity)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AppControl)
		assert.Equal(t, corev1.Device_Status_Posture_OWNERSHIP_UNKNOWN, p.Ownership)
		assert.Empty(t, entry.OwnerEmails)
	}

	{
		entry := (&Manager{}).toEntry(getComputer(`{
			"udid": "u",
			"general": {"remoteManagement": {"managed": false}},
			"groupMemberships": [{"groupName": "Compliant Macs", "smartGroup": true}]
		}`), now)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, entry.Posture.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, entry.Posture.Enrolled)
	}

	{
		entry := m.toEntry(getComputer(fmt.Sprintf(`{
			"udid": "u",
			"general": {"lastContactTime": %q},
			"userAndLocation": {"username": "user2@example.com"}
		}`, now.Add(-72*time.Hour).UTC().Format(time.RFC3339))), now)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, entry.Posture.AgentHealthy)
		assert.Equal(t, []string{"user2@example.com"}, entry.OwnerEmails)
	}

	{
		lastContact := now.Add(-time.Hour).UTC().Truncate(time.Second)
		entry := m.toEntry(getComputer(fmt.Sprintf(`{
			"udid": "u",
			"general": {"lastContact": %q, "lastCheckIn": %q, "lastContactTime": %q}
		}`, lastContact.Format(time.RFC3339),
			now.Add(-72*time.Hour).UTC().Format(time.RFC3339),
			now.Add(-72*time.Hour).UTC().Format(time.RFC3339))), now)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, entry.Posture.AgentHealthy)
		assert.Equal(t, lastContact, entry.Posture.LastSeenAt.AsTime())
	}

	{
		entry := m.toEntry(getComputer(`{"udid": "u", "general": {"lastCheckIn": "2026-01-01T00:00:00Z"}}`), now)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, entry.Posture.AgentHealthy)
		assert.Nil(t, entry.Posture.LastSeenAt)
	}

	for _, tc := range []struct {
		state string
		want  corev1.Device_Status_Posture_SignalState
	}{
		{"ENCRYPTED", corev1.Device_Status_Posture_PASS},
		{"UNENCRYPTED", corev1.Device_Status_Posture_FAIL},
		{"DECRYPTING", corev1.Device_Status_Posture_FAIL},
		{"ENCRYPTING", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
		{"UNKNOWN", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
	} {
		assert.Equal(t, tc.want, fileVaultSignal(&diskEncryption{
			BootPartitionEncryptionDetails: &bootPartitionEncryptionDetails{
				PartitionFileVault2State: tc.state,
			},
		}), tc.state)
	}

	for _, tc := range []struct {
		level string
		want  corev1.Device_Status_Posture_SignalState
	}{
		{"FULL_SECURITY", corev1.Device_Status_Posture_PASS},
		{"MEDIUM_SECURITY", corev1.Device_Status_Posture_FAIL},
		{"NO_SECURITY", corev1.Device_Status_Posture_FAIL},
		{"NOT_SUPPORTED", corev1.Device_Status_Posture_NOT_APPLICABLE},
		{"UNKNOWN", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
	} {
		assert.Equal(t, tc.want, secureBootSignal(tc.level), tc.level)
	}

	assert.Equal(t, corev1.Device_Status_Posture_FAIL, sipSignal("DISABLED"))
	assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, sipSignal("NOT_COLLECTED"))
	assert.Equal(t, corev1.Device_Status_Posture_PASS, gatekeeperSignal("APP_STORE"))
	assert.Equal(t, corev1.Device_Status_Posture_FAIL, gatekeeperSignal("DISABLED"))
	assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, gatekeeperSignal("NOT_COLLECTED"))
}

func TestListComputers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, inventoryPath, r.URL.Path)
		assert.Contains(t, r.URL.Query()["section"], "GROUP_MEMBERSHIPS")
		assert.Equal(t, "general.platform==Mac", r.URL.Query().Get("filter"))

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "0":
			fmt.Fprint(w, `{"totalCount": 3, "results": [{"udid": "a"}, {"udid": "b"}]}`)
		case "1":
			fmt.Fprint(w, `{"totalCount": 3, "results": [{"udid": "c"}]}`)
		default:
			fmt.Fprint(w, `{"totalCount": 3, "results": []}`)
		}
	}))
	t.Cleanup(srv.Close)

	m := &Manager{
		jamf:   &jamfClient{rc: resty.New().SetBaseURL(srv.URL)},
		filter: "general.platform==Mac",
	}

	fleet, err := m.Collect(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 3, fleet.Len())
}

func TestListComputersLegacyFallback(t *testing.T) {
	var paths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?page="+r.URL.Query().Get("page"))
		assert.Equal(t, "id:asc", r.URL.Query().Get("sort"))
		assert.Equal(t, inventorySections, r.URL.Query()["section"])

		if r.URL.Path != legacyInventoryPath {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"httpStatus": 404, "errors": []}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "0":
			fmt.Fprint(w, `{"totalCount": 2, "results": [{"udid": "a", "general": {"lastContactTime": "2026-10-31T10:00:00Z"}}]}`)
		default:
			fmt.Fprint(w, `{"totalCount": 2, "results": [{"udid": "b"}]}`)
		}
	}))
	t.Cleanup(srv.Close)

	m := &Manager{
		jamf: &jamfClient{rc: resty.New().SetBaseURL(srv.URL)},
	}

	fleet, err := m.Collect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, fleet.Len())
	assert.Equal(t, []string{
		inventoryPath + "?page=0",
		legacyInventoryPath + "?page=0",
		legacyInventoryPath + "?page=1",
	}, paths)

	res := fleet.MatchExternalID("a")
	assert.Equal(t, devicemgrcommon.MatchStateUnique, res.State)
	assert.True(t, res.Entry.Posture.LastSeenAt.IsValid())
}

func TestListComputersNoFallback(t *testing.T) {
	for _, code := range []int{http.StatusInternalServerError, http.StatusForbidden} {
		var paths []string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			w.WriteHeader(code)
		}))

		m := &Manager{
			jamf: &jamfClient{rc: resty.New().SetBaseURL(srv.URL)},
		}

		_, err := m.Collect(context.Background())
		assert.NotNil(t, err)
		assert.Equal(t, []string{inventoryPath}, paths, code)

		srv.Close()
	}

	{
		var paths []string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)

		m := &Manager{
			jamf: &jamfClient{rc: resty.New().SetBaseURL(srv.URL)},
		}

		_, err := m.Collect(context.Background())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "Jamf status 404")
		assert.Equal(t, []string{inventoryPath, legacyInventoryPath}, paths)
	}
}

func TestInvalidResponse(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"text/html; charset=utf-8", `<html><body>Sign in</body></html>`},
		{"", `{"totalCount": 0, "results": []}`},
		{"application/json", `{"totalCount": 0}`},
		{"application/json", `{"results": null}`},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tc.contentType != "" {
				w.Header().Set("Content-Type", tc.contentType)
			}
			fmt.Fprint(w, tc.body)
		}))

		m := &Manager{
			jamf: &jamfClient{rc: resty.New().SetBaseURL(srv.URL)},
		}
		_, err := m.Collect(context.Background())
		assert.NotNil(t, err, tc.body)

		srv.Close()
	}
}

func TestParseExternalIDDisagreement(t *testing.T) {
	m := &Manager{}

	ret, err := m.ParseExternalID(corev1.Device_Status_MAC, []*devicemgrcommon.ProbeResult{
		{Text: "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D"},
		{Text: "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
	})
	assert.Nil(t, err)
	assert.Equal(t, "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", ret)

	_, err = m.ParseExternalID(corev1.Device_Status_MAC, []*devicemgrcommon.ProbeResult{
		{Text: "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
		{Text: "1a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
	})
	assert.NotNil(t, err)
}

func TestConstructorCredentialsAndConfiguration(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	_, err = tst.C.OcteliumC.EnterpriseC().CreateSecret(t.Context(), &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: "provider-credential"},
		Spec:     &enterprisev1.Secret_Spec{},
		Data:     &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: "test-token"}},
	})
	require.NoError(t, err)
	configuration := func(t *testing.T) *enterprisev1.DeviceManager {
		t.Helper()
		ret := &enterprisev1.DeviceManager{}
		require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"jamf\": {\"baseURL\": \"https://api.example.com\", \"clientID\": \"client\", \"clientSecret\": {\"fromSecret\": \"provider-credential\"}}}}"), ret))
		return ret
	}
	dm := configuration(t)
	manager, err := New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: dm})
	require.NoError(t, err)
	require.NotNil(t, manager)
	assert.Equal(t, enterprisev1.DeviceManager_Status_JAMF_PRO, manager.Type())
	require.NoError(t, manager.Close())
	wrong := &enterprisev1.DeviceManager{Metadata: &metav1.Metadata{Name: "wrong"}, Spec: &enterprisev1.DeviceManager_Spec{}}
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: wrong})
	require.Error(t, err)
	for _, field := range []string{"clientSecret", "clientID"} {
		t.Run("Missing/"+field, func(t *testing.T) {
			encoded, err := protojson.Marshal(configuration(t))
			require.NoError(t, err)
			var document map[string]any
			require.NoError(t, json.Unmarshal(encoded, &document))
			delete(document["spec"].(map[string]any)["jamf"].(map[string]any), field)
			encoded, err = json.Marshal(document)
			require.NoError(t, err)
			bad := &enterprisev1.DeviceManager{}
			require.NoError(t, protojson.Unmarshal(encoded, bad))
			_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: bad})
			assert.Error(t, err)
		})
	}
	encoded, err := protojson.Marshal(configuration(t))
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(encoded, &document))
	document["spec"].(map[string]any)["jamf"].(map[string]any)["clientSecret"] = map[string]any{"fromSecret": "missing-credential"}
	encoded, err = json.Marshal(document)
	require.NoError(t, err)
	missing := &enterprisev1.DeviceManager{}
	require.NoError(t, protojson.Unmarshal(encoded, missing))
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: missing})
	assert.Error(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = New(ctx, tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: configuration(t)})
	assert.Error(t, err)

	encoded, err = protojson.Marshal(configuration(t))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, &document))
	document["spec"].(map[string]any)["jamf"].(map[string]any)["baseURL"] = "http://insecure.example.com"
	encoded, err = json.Marshal(document)
	require.NoError(t, err)
	badURL := &enterprisev1.DeviceManager{}
	require.NoError(t, protojson.Unmarshal(encoded, badURL))
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: badURL})
	assert.Error(t, err)

}

func TestTransientErrorsAndRedirects(t *testing.T) {
	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() { tst.Destroy() })
	_, err = tst.C.OcteliumC.EnterpriseC().CreateSecret(t.Context(), &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: "provider-credential"},
		Spec:     &enterprisev1.Secret_Spec{},
		Data:     &enterprisev1.Secret_Data{Type: &enterprisev1.Secret_Data_Value{Value: "test-token"}},
	})
	require.NoError(t, err)
	dm := &enterprisev1.DeviceManager{}
	require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"jamf\": {\"baseURL\": \"https://api.example.com\", \"clientID\": \"client\", \"clientSecret\": {\"fromSecret\": \"provider-credential\"}}}}"), dm))
	manager, err := New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: dm})
	require.NoError(t, err)
	t.Cleanup(func() { manager.Close() })

	var foreignRequests atomic.Int32
	foreign := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foreignRequests.Add(1)
	}))
	t.Cleanup(foreign.Close)

	var requests atomic.Int32
	var redirect atomic.Bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch n := requests.Add(1); {
		case redirect.Load():
			http.Redirect(w, r, foreign.URL+r.URL.RequestURI(), http.StatusFound)
		case n == 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case n == 2:
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"totalCount": 1, "results": [{"id": "1", "udid": "udid-1"}]}`)
		}
	}))
	t.Cleanup(srv.Close)

	manager.jamf.rc.SetBaseURL(srv.URL).SetTransport(srv.Client().Transport).
		SetRetryWaitTime(time.Millisecond).SetRetryMaxWaitTime(time.Millisecond)

	fleet, err := manager.Collect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, fleet.Len())
	assert.Equal(t, int32(3), requests.Load())

	redirect.Store(true)
	requests.Store(0)
	_, err = manager.Collect(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, devicemgrcommon.ErrRedirectNotAllowed)
	assert.Equal(t, int32(1), requests.Load())
	assert.Zero(t, foreignRequests.Load())
}

func TestEmptyCollectionAndCanceledRequest(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{\"totalCount\": 0, \"results\": []}")
	}))
	t.Cleanup(srv.Close)
	rc := resty.New().SetBaseURL(srv.URL).SetRetryCount(0)
	manager := &Manager{jamf: &jamfClient{rc: rc}}
	fleet, err := manager.Collect(t.Context())
	require.NoError(t, err)
	assert.Zero(t, fleet.Len())
	before := requests.Load()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = manager.Collect(ctx)
	require.Error(t, err)
	assert.Equal(t, before, requests.Load())
}

func TestErrorStatus(t *testing.T) {
	for _, tc := range []struct {
		code int
		want string
	}{
		{http.StatusUnauthorized, "check the API Role has Read Computers"},
		{http.StatusForbidden, "check the API Role has Read Computers"},
		{http.StatusNotFound, "Jamf status 404"},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.code)
			fmt.Fprint(w, strings.Repeat("x", 400))
		}))

		m := &Manager{jamf: &jamfClient{rc: resty.New().SetBaseURL(srv.URL)}}
		_, err := m.Collect(context.Background())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), tc.want)
		assert.Contains(t, err.Error(), strings.Repeat("x", 300)+"...")
		assert.NotContains(t, err.Error(), strings.Repeat("x", 301))

		srv.Close()
	}
}
