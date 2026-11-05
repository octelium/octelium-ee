// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package fleetdm

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
		results []*devicemgrcommon.ProbeResult
		want    string
	}{
		{nil, ""},
		{[]*devicemgrcommon.ProbeResult{nil, {Text: "UUID: 0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Text: "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D\r\n"}},
			"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
	} {
		ret, err := m.ParseExternalID(corev1.Device_Status_WINDOWS, tc.results)
		assert.Nil(t, err)
		assert.Equal(t, tc.want, ret)
	}

	osTypes := map[string]corev1.Device_Status_OSType{}
	for _, probe := range m.IdentityProbes() {
		assert.Nil(t, probe.RunCommand)
		assert.NotNil(t, probe.PlatformIdentifier)
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

	getHost := func(arg string) *fleetHost {
		ret := &fleetHost{}
		assert.Nil(t, json.Unmarshal([]byte(arg), ret))
		return ret
	}

	{
		entry := toEntry(getHost(`{
			"id": 10,
			"uuid": "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D",
			"hostname": "alice-mbp.local",
			"platform": "darwin",
			"hardware_serial": "C02XYZ",
			"primary_mac": "a4:bb:cc:dd:ee:01",
			"status": "online",
			"seen_time": "2026-09-28T10:00:00Z",
			"policy_updated_at": "2026-09-28T10:00:00Z",
			"os_version": "macOS 14.5",
			"hardware_model": "MacBookPro18,3",
			"disk_encryption_enabled": true,
			"issues": {"failing_policies_count": 0, "critical_vulnerabilities_count": 2, "total_issues_count": 2},
			"mdm": {"enrollment_status": "On (automatic)", "name": "Fleet"}
		}`), now)

		assert.Equal(t, "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", entry.ExternalID)
		assert.Equal(t, "C02XYZ", entry.Serial)
		assert.Equal(t, []string{"a4:bb:cc:dd:ee:01"}, entry.MACs)
		assert.Equal(t, "alice-mbp.local", entry.Hostname)
		assert.Equal(t, corev1.Device_Status_MAC, entry.OSType)
		assert.Equal(t, "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D", entry.MachineID)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Enrolled)
		assert.Equal(t, corev1.Device_Status_Posture_COMPANY, p.Ownership)
		assert.Equal(t, corev1.Device_Status_Posture_WORKSTATION, p.DeviceClass)
		assert.EqualValues(t, 0, p.GetFailingChecks())
		assert.NotNil(t, p.FailingChecks, "a reported zero must not be dropped")
		assert.EqualValues(t, 14, p.Os.GetMajor())
		assert.EqualValues(t, 2, p.Attrs.Fields["criticalVulnerabilities"].GetNumberValue())
		assert.Nil(t, p.Attrs.Fields["failingPoliciesCount"])
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.True(t, p.LastSeenAt.IsValid())
	}

	{
		entry := toEntry(getHost(`{"uuid": "u", "platform": "ubuntu", "status": "offline",
			"issues": {"failing_policies_count": 3}}`), now)
		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
		assert.EqualValues(t, 3, p.GetFailingChecks())
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Enrolled)
		assert.Empty(t, entry.MACs)
		assert.Equal(t, corev1.Device_Status_LINUX, entry.OSType)
		assert.Empty(t, entry.MachineID)
	}

	{
		entry := toEntry(getHost(`{"uuid": "u", "status": "new", "issues": {}}`), now)
		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AgentHealthy)
	}

	{
		entry := toEntry(getHost(`{"uuid": "u"}`), now)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, entry.Posture.Compliant)
	}

	for _, tc := range []struct {
		name string
		arg  string
		want corev1.Device_Status_Posture_SignalState
	}{
		{
			name: "NoFailuresWithResults",
			arg:  `{"issues": {"failing_policies_count": 0}, "policy_updated_at": "2026-09-28T10:00:00Z"}`,
			want: corev1.Device_Status_Posture_PASS,
		},
		{
			name: "NoFailuresNeverChecked",
			arg:  `{"issues": {"failing_policies_count": 0}}`,
			want: corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN,
		},
		{
			name: "NoFailuresZeroTime",
			arg:  `{"issues": {"failing_policies_count": 0}, "policy_updated_at": "0001-01-01T00:00:00Z"}`,
			want: corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN,
		},
		{
			name: "NoFailuresFleetDefaultTime",
			arg:  `{"issues": {"failing_policies_count": 0}, "policy_updated_at": "2000-01-01T00:00:00Z"}`,
			want: corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN,
		},
		{
			name: "Failures",
			arg:  `{"issues": {"failing_policies_count": 1}}`,
			want: corev1.Device_Status_Posture_FAIL,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, toEntry(getHost(tc.arg), now).Posture.Compliant)
		})
	}

	for _, tc := range []struct {
		status    string
		enrolled  corev1.Device_Status_Posture_SignalState
		ownership corev1.Device_Status_Posture_Ownership
	}{
		{"On (automatic)", corev1.Device_Status_Posture_PASS, corev1.Device_Status_Posture_COMPANY},
		{"On (manual)", corev1.Device_Status_Posture_PASS, corev1.Device_Status_Posture_COMPANY},
		{"On (personal)", corev1.Device_Status_Posture_PASS, corev1.Device_Status_Posture_PERSONAL},
		{"On (manual - personal)", corev1.Device_Status_Posture_PASS, corev1.Device_Status_Posture_PERSONAL},
		{"Pending", corev1.Device_Status_Posture_FAIL, corev1.Device_Status_Posture_OWNERSHIP_UNKNOWN},
		{"Off", corev1.Device_Status_Posture_FAIL, corev1.Device_Status_Posture_OWNERSHIP_UNKNOWN},
		{"", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, corev1.Device_Status_Posture_OWNERSHIP_UNKNOWN},
	} {
		assert.Equal(t, tc.enrolled, fleetEnrolled(tc.status), tc.status)
		assert.Equal(t, tc.ownership, fleetOwnership(tc.status), tc.status)
	}

	{
		p := toEntry(getHost(`{"os_version": "Microsoft Windows Server 2022 Standard 10.0.20348.2527",
			"platform": "windows"}`), now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_SERVER, p.DeviceClass)
		assert.EqualValues(t, 10, p.Os.GetMajor())
		assert.EqualValues(t, 20348, p.Os.GetPatch())
		assert.EqualValues(t, 2527, p.Os.GetRevision())
	}

	for _, tc := range []struct {
		status   string
		seenTime time.Time
		want     corev1.Device_Status_Posture_SignalState
	}{
		{"online", time.Time{}, corev1.Device_Status_Posture_PASS},
		{"Online", now.Add(-48 * time.Hour), corev1.Device_Status_Posture_PASS},
		{"offline", now.Add(-time.Hour), corev1.Device_Status_Posture_PASS},
		{"offline", now.Add(-48 * time.Hour), corev1.Device_Status_Posture_FAIL},
		{"offline", now.Add(365 * 24 * time.Hour), corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
		{"offline", time.Time{}, corev1.Device_Status_Posture_FAIL},
		{"missing", now.Add(-time.Hour), corev1.Device_Status_Posture_FAIL},
		{"new", now.Add(-time.Hour), corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
		{"", time.Time{}, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
	} {
		h := &fleetHost{Status: tc.status}
		if !tc.seenTime.IsZero() {
			h.SeenTime = tc.seenTime.Format(time.RFC3339)
		}
		assert.Equal(t, tc.want, fleetAgentHealthy(h, now), "%s %s", tc.status, h.SeenTime)
	}
}

func TestListHosts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, hostsPath, r.URL.Path)
		assert.Equal(t, "id", q.Get("order_key"))
		assert.Equal(t, "asc", q.Get("order_direction"))
		assert.Equal(t, "7", q.Get("team_id"))

		w.Header().Set("Content-Type", "application/json")
		switch q.Get("page") {
		case "0":
			var hosts []string
			for i := range fleetPageSize {
				hosts = append(hosts, fmt.Sprintf(`{"id": %d, "uuid": "uuid-%d"}`, i, i))
			}
			fmt.Fprintf(w, `{"hosts": [%s]}`, strings.Join(hosts, ","))
		default:
			fmt.Fprint(w, `{"hosts": [{"id": 1000, "uuid": "uuid-1000"}]}`)
		}
	}))
	t.Cleanup(srv.Close)

	m := &Manager{
		api:    &apiClient{rc: resty.New().SetBaseURL(srv.URL)},
		teamID: 7,
	}

	fleet, err := m.Collect(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, fleetPageSize+1, fleet.Len())
}

func TestInvalidResponse(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"text/html; charset=utf-8", `<html><body>Sign in</body></html>`},
		{"", `{"hosts": []}`},
		{"application/json", `{}`},
		{"application/json", `{"hosts": null}`},
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
		require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"fleetDM\": {\"baseURL\": \"https://api.example.com\", \"apiToken\": {\"fromSecret\": \"provider-credential\"}}}}"), ret))
		return ret
	}
	dm := configuration(t)
	manager, err := New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: dm})
	require.NoError(t, err)
	require.NotNil(t, manager)
	assert.Equal(t, enterprisev1.DeviceManager_Status_FLEETDM, manager.Type())
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
			delete(document["spec"].(map[string]any)["fleetDM"].(map[string]any), field)
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
	document["spec"].(map[string]any)["fleetDM"].(map[string]any)["apiToken"] = map[string]any{"fromSecret": "missing-credential"}
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
	document["spec"].(map[string]any)["fleetDM"].(map[string]any)["baseURL"] = "http://insecure.example.com"
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
	require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"fleetDM\": {\"baseURL\": \"https://api.example.com\", \"apiToken\": {\"fromSecret\": \"provider-credential\"}}}}"), dm))
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
			fmt.Fprint(w, `{"hosts": [{"id": 1, "uuid": "uuid-1"}]}`)
		}
	}))
	t.Cleanup(srv.Close)

	manager.api.rc.SetBaseURL(srv.URL).SetTransport(srv.Client().Transport).
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
		fmt.Fprint(w, "{\"hosts\": []}")
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

func TestErrorStatus(t *testing.T) {
	for _, tc := range []struct {
		code int
		want string
	}{
		{http.StatusUnauthorized, "check API-only user token"},
		{http.StatusForbidden, "check API-only user token"},
		{http.StatusNotFound, "FleetDM status 404"},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.code)
			fmt.Fprint(w, strings.Repeat("x", 400))
		}))

		m := &Manager{api: &apiClient{rc: resty.New().SetBaseURL(srv.URL)}}
		_, err := m.Collect(context.Background())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), tc.want)
		assert.Contains(t, err.Error(), strings.Repeat("x", 300)+"...")
		assert.NotContains(t, err.Error(), strings.Repeat("x", 301))

		srv.Close()
	}
}
