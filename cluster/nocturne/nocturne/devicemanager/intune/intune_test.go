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
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/stretchr/testify/assert"
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
			"wiFiMacAddress": "aabbccddee01",
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
		assert.Equal(t, []string{"aabbccddee01"}, entry.MACs)

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
