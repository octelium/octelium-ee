// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package intune

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

	output := `
+----------------------------------------------------------------------+
| Device State                                                         |
+----------------------------------------------------------------------+

             AzureAdJoined : YES
                  DeviceId : 0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D
                Thumbprint : 0123456789ABCDEF
`

	{
		ret, err := m.ParseExternalID(corev1.Device_Status_WINDOWS, []*devicemgrcommon.ProbeResult{{Text: output}})
		assert.Nil(t, err)
		assert.Equal(t, "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", ret)
	}

	{
		ret, err := m.ParseExternalID(corev1.Device_Status_MAC, []*devicemgrcommon.ProbeResult{{Text: output}})
		assert.Nil(t, err)
		assert.Equal(t, "", ret)
	}

	{
		ret, err := m.ParseExternalID(corev1.Device_Status_WINDOWS, []*devicemgrcommon.ProbeResult{
			nil,
			{Text: "DeviceId : 00000000-0000-0000-0000-000000000000"},
		})
		assert.Nil(t, err)
		assert.Equal(t, "", ret)
	}
}

func TestToEntry(t *testing.T) {
	getDevice := func(arg string) *managedDevice {
		ret := &managedDevice{}
		assert.Nil(t, json.Unmarshal([]byte(arg), ret))
		return ret
	}

	m := &Manager{}

	{
		entry := m.toEntry(getDevice(`{
			"id": "INTUNE-ID-1",
			"serialNumber": "SERIAL-1",
			"wiFiMacAddress": "a4bbccddee01",
			"ethernetMacAddress": "",
			"operatingSystem": "Windows",
			"complianceState": "compliant",
			"managementState": "managed",
			"lastSyncDateTime": "2026-09-28T10:00:00Z",
			"userPrincipalName": "upn@corp.example.com",
			"emailAddress": "user@example.com",
			"azureADDeviceId": "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D",
			"jailBroken": "Unknown",
			"isEncrypted": true,
			"partnerReportedThreatState": "secured"
		}`))

		assert.Equal(t, "intune-id-1", entry.ExternalID)
		assert.Equal(t, []string{"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"}, entry.Aliases)
		assert.Equal(t, []string{"user@example.com", "upn@corp.example.com"}, entry.OwnerEmails)
		assert.Equal(t, []string{"a4bbccddee01"}, entry.MACs)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Enrolled)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_NOT_APPLICABLE, p.MobileIntegrity)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.True(t, p.LastSeenAt.IsValid())
	}

	{
		entry := m.toEntry(getDevice(`{
			"id": "intune-id-2",
			"azureADDeviceId": "00000000-0000-0000-0000-000000000000",
			"userPrincipalName": "not-an-email",
			"operatingSystem": "iOS",
			"jailBroken": "True"
		}`))

		assert.Empty(t, entry.Aliases)
		assert.Empty(t, entry.OwnerEmails)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Enrolled)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.MobileIntegrity)
		assert.Nil(t, p.LastSeenAt)
	}

	for _, tc := range []struct {
		m     *Manager
		state string
		want  corev1.Device_Status_Posture_SignalState
	}{
		{m, "compliant", corev1.Device_Status_Posture_PASS},
		{m, "inGracePeriod", corev1.Device_Status_Posture_FAIL},
		{&Manager{isGracePeriodCompliant: true}, "inGracePeriod", corev1.Device_Status_Posture_PASS},
		{m, "configManager", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
		{&Manager{isConfigManagerCompliant: true}, "configManager", corev1.Device_Status_Posture_PASS},
		{m, "noncompliant", corev1.Device_Status_Posture_FAIL},
		{m, "conflict", corev1.Device_Status_Posture_FAIL},
		{m, "error", corev1.Device_Status_Posture_FAIL},
		{m, "unknown", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
		{m, "", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
	} {
		assert.Equal(t, tc.want, tc.m.complianceSignal(tc.state), tc.state)
	}

	for _, tc := range []struct {
		state string
		want  corev1.Device_Status_Posture_SignalState
	}{
		{"secured", corev1.Device_Status_Posture_PASS},
		{"lowSeverity", corev1.Device_Status_Posture_FAIL},
		{"highSeverity", corev1.Device_Status_Posture_FAIL},
		{"compromised", corev1.Device_Status_Posture_FAIL},
		{"activated", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
		{"unresponsive", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
		{"", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
	} {
		assert.Equal(t, tc.want, threatSignal(tc.state), tc.state)
	}

	assert.Equal(t, corev1.Device_Status_Posture_FAIL, enrolledSignal("retirePending"))
	assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, enrolledSignal("unknown"))
	assert.Equal(t, corev1.Device_Status_Posture_PASS, mobileIntegritySignal("Android", "False"))
	assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, mobileIntegritySignal("iPadOS", "Unknown"))
	assert.Equal(t, corev1.Device_Status_Posture_NOT_APPLICABLE, mobileIntegritySignal("macOS", "True"))
}

func TestListManagedDevices(t *testing.T) {
	var srvURL string
	crossOrigin := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, managedDevicesPath, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Query().Get("$skiptoken") == "" {
			assert.Equal(t, "500", r.URL.Query().Get("$top"))
			assert.Equal(t, "operatingSystem eq 'Windows'", r.URL.Query().Get("$filter"))

			nextLink := srvURL + managedDevicesPath + "?$skiptoken=page-2"
			if crossOrigin {
				nextLink = "https://evil.example.com" + managedDevicesPath + "?$skiptoken=page-2"
			}

			fmt.Fprintf(w, `{"value": [{"id": "a", "azureADDeviceId": "alias-a"}], "@odata.nextLink": %q}`, nextLink)
			return
		}

		fmt.Fprint(w, `{"value": [{"id": "b"}, {"id": ""}]}`)
	}))
	t.Cleanup(srv.Close)
	srvURL = srv.URL

	m := &Manager{
		graph: &graphClient{
			rc:   resty.New(),
			base: srv.URL,
		},
		filter: "operatingSystem eq 'Windows'",
	}

	fleet, err := m.Collect(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 2, fleet.Len())
	assert.Equal(t, "a", fleet.MatchProbeID("alias-a").Entry.ExternalID)

	crossOrigin = true
	_, err = m.Collect(context.Background())
	assert.NotNil(t, err)
}

func TestInvalidResponse(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"text/html; charset=utf-8", `<html><body>Sign in</body></html>`},
		{"", `{"value": []}`},
		{"application/json", `{}`},
		{"application/json", `{"value": null}`},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tc.contentType != "" {
				w.Header().Set("Content-Type", tc.contentType)
			}
			fmt.Fprint(w, tc.body)
		}))

		m := &Manager{
			graph: &graphClient{rc: resty.New(), base: srv.URL},
		}
		_, err := m.Collect(context.Background())
		assert.NotNil(t, err, tc.body)

		srv.Close()
	}
}

func TestParseExternalIDDisagreement(t *testing.T) {
	m := &Manager{}

	_, err := m.ParseExternalID(corev1.Device_Status_WINDOWS, []*devicemgrcommon.ProbeResult{
		{Text: "  DeviceId : 0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d\n"},
		{Text: "  DeviceId : 1a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d\n"},
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
		require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"microsoftIntune\": {\"tenantID\": \"tenant\", \"clientID\": \"client\", \"clientSecret\": {\"fromSecret\": \"provider-credential\"}}}}"), ret))
		return ret
	}
	dm := configuration(t)
	manager, err := New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: dm})
	require.NoError(t, err)
	require.NotNil(t, manager)
	assert.Equal(t, enterprisev1.DeviceManager_Status_MICROSOFT_INTUNE, manager.Type())
	require.NoError(t, manager.Close())
	wrong := &enterprisev1.DeviceManager{Metadata: &metav1.Metadata{Name: "wrong"}, Spec: &enterprisev1.DeviceManager_Spec{}}
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: wrong})
	require.Error(t, err)
	for _, field := range []string{"clientSecret", "tenantID", "clientID"} {
		t.Run("Missing/"+field, func(t *testing.T) {
			encoded, err := protojson.Marshal(configuration(t))
			require.NoError(t, err)
			var document map[string]any
			require.NoError(t, json.Unmarshal(encoded, &document))
			delete(document["spec"].(map[string]any)["microsoftIntune"].(map[string]any), field)
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
	document["spec"].(map[string]any)["microsoftIntune"].(map[string]any)["clientSecret"] = map[string]any{"fromSecret": "missing-credential"}
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

}

func TestProviderRetryAndRateLimitBoundaries(t *testing.T) {
	for _, tc := range []struct {
		code  int
		retry bool
	}{
		{200, false}, {400, false}, {401, false}, {403, false}, {429, true}, {500, true}, {503, true},
	} {
		response := &resty.Response{RawResponse: &http.Response{StatusCode: tc.code}}
		assert.Equal(t, tc.retry, retryCondition(response, nil), tc.code)
	}
	assert.True(t, retryCondition(nil, fmt.Errorf("transport failure")))
	for _, tc := range []struct {
		value   string
		seconds int
	}{
		{"", 0}, {"0", 0}, {" 2 ", 2}, {"300", 300}, {"301", 0}, {"-1", 0}, {"invalid", 0},
	} {
		response := &resty.Response{RawResponse: &http.Response{Header: http.Header{"Retry-After": []string{tc.value}}}}
		delay, err := retryAfter(nil, response)
		require.NoError(t, err)
		assert.Equal(t, time.Duration(tc.seconds)*time.Second, delay, tc.value)
	}
	delay, err := retryAfter(nil, nil)
	require.NoError(t, err)
	assert.Zero(t, delay)
}

func TestEmptyCollectionAndCanceledRequest(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{\"value\": []}")
	}))
	t.Cleanup(srv.Close)
	rc := resty.New().SetBaseURL(srv.URL).SetRetryCount(0)
	manager := &Manager{graph: &graphClient{rc: rc, base: srv.URL}}
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

func TestCloudEndpointsStayWithinSelectedCloud(t *testing.T) {
	for _, tc := range []struct {
		cloud        enterprisev1.DeviceManager_Spec_MicrosoftIntune_Cloud
		login, graph string
	}{
		{enterprisev1.DeviceManager_Spec_MicrosoftIntune_US_GOV, "https://login.microsoftonline.us", "https://graph.microsoft.us"},
		{enterprisev1.DeviceManager_Spec_MicrosoftIntune_CHINA, "https://login.partner.microsoftonline.cn", "https://microsoftgraph.chinacloudapi.cn"},
		{0, "https://login.microsoftonline.com", "https://graph.microsoft.com"},
	} {
		got := cloudEndpoints(tc.cloud)
		assert.Equal(t, tc.login, got.login)
		assert.Equal(t, tc.graph, got.graph)
	}
}
