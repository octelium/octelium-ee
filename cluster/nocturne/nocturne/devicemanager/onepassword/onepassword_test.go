// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package onepassword

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		results []*devicemgrcommon.ProbeResult
		want    string
	}{
		{nil, ""},
		{[]*devicemgrcommon.ProbeResult{nil, {Text: "invalid"}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Text: " 0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D\n"}},
			"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
	} {
		ret, err := m.ParseExternalID(corev1.Device_Status_LINUX, tc.results)
		assert.Nil(t, err)
		assert.Equal(t, tc.want, ret)
	}

	osTypes := map[string]corev1.Device_Status_OSType{}
	for _, probe := range m.IdentityProbes() {
		assert.NotEmpty(t, probe.ID)
		assert.NotNil(t, probe.PlatformIdentifier)
		assert.Equal(t, corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
			probe.PlatformIdentifier.Kind)
		assert.Equal(t, probe.ID == "hardware-uuid-linux", probe.RequireElevation)
		osTypes[probe.ID] = probe.OSType
	}
	assert.Equal(t, map[string]corev1.Device_Status_OSType{
		"hardware-uuid-linux":   corev1.Device_Status_LINUX,
		"hardware-uuid-macos":   corev1.Device_Status_MAC,
		"hardware-uuid-windows": corev1.Device_Status_WINDOWS,
	}, osTypes)
}

func TestToEntry(t *testing.T) {
	now := time.Now()

	getDevice := func(arg string) *kolideDevice {
		ret := &kolideDevice{}
		assert.Nil(t, json.Unmarshal([]byte(arg), ret))
		return ret
	}

	emails := map[string]string{
		"person-1": "user@example.com",
		"person-2": "invalid",
	}

	{
		entry := toEntry(getDevice(fmt.Sprintf(`{
			"id": "DEVICE-1",
			"name": "alice-mbp",
			"serial": "C02XYZ",
			"hardware_uuid": "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D",
			"auth_state": "Good",
			"last_seen_at": %q,
			"device_type": "Mac",
			"form_factor": "Computer",
			"operating_system": "macOS 14.5",
			"registered_owner_info": {"identifier": "person-1", "link": "https://api.kolide.com/people/person-1"}
		}`, now.Add(-time.Hour).UTC().Format(time.RFC3339))), emails, now)

		assert.Equal(t, "device-1", entry.ExternalID)
		assert.Equal(t, []string{"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"}, entry.Aliases)
		assert.Equal(t, "C02XYZ", entry.Serial)
		assert.Equal(t, []string{"user@example.com"}, entry.OwnerEmails)
		assert.Equal(t, "alice-mbp", entry.Hostname)
		assert.Equal(t, corev1.Device_Status_MAC, entry.OSType)
		assert.Equal(t, "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D", entry.MachineID)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_WORKSTATION, p.DeviceClass)
		assert.EqualValues(t, 14, p.Os.GetMajor())
		assert.EqualValues(t, 5, p.Os.GetMinor())
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.Equal(t, "Mac", p.Attrs.Fields["deviceType"].GetStringValue())
		assert.Equal(t, "person-1", p.Attrs.Fields["registeredOwnerId"].GetStringValue())
	}

	{
		entry := toEntry(getDevice(`{
			"id": "device-2",
			"auth_state": "Blocked",
			"form_factor": "Phone",
			"will_block_at": "2026-09-28T10:00:00Z",
			"operating_system": "Ubuntu 22.04.3 LTS",
			"hardware_uuid": "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D",
			"registered_owner_info": {"identifier": "person-2"}
		}`), emails, now)

		assert.Equal(t, []string{"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"}, entry.Aliases)
		assert.Empty(t, entry.OwnerEmails)
		assert.Equal(t, corev1.Device_Status_LINUX, entry.OSType)
		assert.Empty(t, entry.MachineID)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, entry.Posture.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, entry.Posture.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_PHONE, entry.Posture.DeviceClass)
		assert.EqualValues(t, 22, entry.Posture.Os.GetMajor())
		assert.EqualValues(t, 4, entry.Posture.Os.GetMinor())
		assert.Equal(t, "2026-09-28T10:00:00Z", entry.Posture.Attrs.Fields["willBlockAt"].GetStringValue())
	}

	for _, tc := range []struct {
		state string
		want  corev1.Device_Status_Posture_SignalState
	}{
		{"Good", corev1.Device_Status_Posture_PASS},
		{"Notified", corev1.Device_Status_Posture_GRACE_PERIOD},
		{"Will Block", corev1.Device_Status_Posture_GRACE_PERIOD},
		{"Blocked", corev1.Device_Status_Posture_FAIL},
		{"", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
	} {
		assert.Equal(t, tc.want, authStateCompliant(tc.state), tc.state)
	}
}

func TestCollect(t *testing.T) {
	requests := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, apiVersion, r.Header.Get("X-Kolide-Api-Version"))
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path + "|" + r.URL.Query().Get("cursor") {
		case devicesPath + "|":
			fmt.Fprint(w, `{
				"data": [{"id": "1", "registered_owner_info": {"identifier": "p1"}}],
				"pagination": {"next": "https://api.kolide.com/devices?cursor=NCw0", "next_cursor": "NCw0", "count": 1}
			}`)
		case devicesPath + "|NCw0":
			fmt.Fprint(w, `{
				"data": [{"id": "2", "hardware_uuid": "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"}, {"id": ""}],
				"pagination": {"next": "", "next_cursor": "", "count": 2}
			}`)
		case peoplePath + "|":
			fmt.Fprint(w, `{
				"data": [{"id": "p1", "email": "user@example.com"}],
				"pagination": {"next_cursor": "Miwy"}
			}`)
		case peoplePath + "|Miwy":
			fmt.Fprint(w, `{
				"data": [{"id": "p2", "email": "user2@example.com"}],
				"pagination": {"next_cursor": ""}
			}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	m := &Manager{
		api: &apiClient{
			rc: resty.New().SetBaseURL(srv.URL).SetHeader("X-Kolide-Api-Version", apiVersion),
		},
	}

	fleet, err := m.Collect(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 2, fleet.Len())
	assert.Equal(t, 4, requests)

	res := fleet.MatchExternalID("1")
	assert.Equal(t, devicemgrcommon.MatchStateUnique, res.State)
	assert.Equal(t, []string{"user@example.com"}, res.Entry.OwnerEmails)

	assert.Equal(t, "2", fleet.MatchProbeID("0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D").Entry.ExternalID)
}

func TestRepeatedCursor(t *testing.T) {
	for _, pth := range []string{devicesPath, peoplePath} {
		requests := 0

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.Header().Set("Content-Type", "application/json")

			switch {
			case r.URL.Path != pth:
				fmt.Fprint(w, `{"data": [{"id": "1"}], "pagination": {"next_cursor": ""}}`)
			case r.URL.Query().Get("cursor") == "Mywz":
				fmt.Fprint(w, `{"data": [{"id": "3"}], "pagination": {"next_cursor": "Miwy"}}`)
			default:
				fmt.Fprint(w, `{"data": [{"id": "2"}], "pagination": {"next_cursor": "Mywz"}}`)
			}
		}))

		m := &Manager{
			api: &apiClient{rc: resty.New().SetBaseURL(srv.URL)},
		}

		_, err := m.Collect(context.Background())
		assert.NotNil(t, err, pth)
		assert.Contains(t, err.Error(), "repeated pagination cursor")
		assert.LessOrEqual(t, requests, 5)

		srv.Close()
	}
}

func TestCollectError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == peoplePath {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data": [], "pagination": {}}`)
	}))
	t.Cleanup(srv.Close)

	m := &Manager{
		api: &apiClient{
			rc: resty.New().SetBaseURL(srv.URL),
		},
	}

	_, err := m.Collect(context.Background())
	assert.NotNil(t, err)
}

func TestInvalidResponse(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"text/html; charset=utf-8", `<html><body>Sign in</body></html>`},
		{"", `{"data": [], "pagination": {}}`},
		{"application/json", `{"pagination": {}}`},
		{"application/json", `{"data": null}`},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tc.contentType != "" {
				w.Header().Set("Content-Type", tc.contentType)
			}
			fmt.Fprint(w, tc.body)
		}))

		m := &Manager{
			api: &apiClient{rc: resty.New().SetBaseURL(srv.URL)},
		}
		_, err := m.Collect(context.Background())
		assert.NotNil(t, err, tc.body)

		srv.Close()
	}
}

func TestParseExternalIDDisagreement(t *testing.T) {
	m := &Manager{}

	ret, err := m.ParseExternalID(corev1.Device_Status_LINUX, []*devicemgrcommon.ProbeResult{
		{Text: "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D"},
		{Text: "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
	})
	assert.Nil(t, err)
	assert.Equal(t, "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", ret)

	_, err = m.ParseExternalID(corev1.Device_Status_LINUX, []*devicemgrcommon.ProbeResult{
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
		require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"onePassword\": {\"baseURL\": \"https://api.example.com\", \"apiToken\": {\"fromSecret\": \"provider-credential\"}}}}"), ret))
		return ret
	}
	dm := configuration(t)
	manager, err := New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: dm})
	require.NoError(t, err)
	require.NotNil(t, manager)
	assert.Equal(t, enterprisev1.DeviceManager_Status_ONEPASSWORD, manager.Type())
	require.NoError(t, manager.Close())
	wrong := &enterprisev1.DeviceManager{Metadata: &metav1.Metadata{Name: "wrong"}, Spec: &enterprisev1.DeviceManager_Spec{}}
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: wrong})
	require.Error(t, err)
	for _, field := range []string{"apiToken"} {
		t.Run("Missing/"+field, func(t *testing.T) {
			encoded, err := protojson.Marshal(configuration(t))
			require.NoError(t, err)
			var document map[string]any
			require.NoError(t, json.Unmarshal(encoded, &document))
			delete(document["spec"].(map[string]any)["onePassword"].(map[string]any), field)
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
	document["spec"].(map[string]any)["onePassword"].(map[string]any)["apiToken"] = map[string]any{"fromSecret": "missing-credential"}
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
	document["spec"].(map[string]any)["onePassword"].(map[string]any)["baseURL"] = "http://insecure.example.com"
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
	require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"onePassword\": {\"baseURL\": \"https://api.example.com\", \"apiToken\": {\"fromSecret\": \"provider-credential\"}}}}"), dm))
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
			fmt.Fprint(w, `{"data": [{"id": "1"}], "pagination": {}}`)
		}
	}))
	t.Cleanup(srv.Close)

	manager.api.rc.SetBaseURL(srv.URL).SetTransport(srv.Client().Transport).
		SetRetryWaitTime(time.Millisecond).SetRetryMaxWaitTime(time.Millisecond)

	fleet, err := manager.Collect(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, fleet.Len())
	assert.Equal(t, int32(4), requests.Load())

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
		fmt.Fprint(w, "{\"data\": []}")
	}))
	t.Cleanup(srv.Close)
	rc := resty.New().SetBaseURL(srv.URL).SetRetryCount(0)
	manager := &Manager{api: &apiClient{rc: rc}}
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
