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
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/stretchr/testify/assert"
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
				"lastContactTime": %q,
				"remoteManagement": {"managed": true}
			},
			"hardware": {
				"serialNumber": "C02XYZ",
				"macAddress": "aa:bb:cc:dd:ee:01",
				"altMacAddress": "AA:BB:CC:DD:EE:01"
			},
			"userAndLocation": {"email": "user@example.com"},
			"security": {
				"sipStatus": "ENABLED",
				"gatekeeperStatus": "APP_STORE_AND_IDENTIFIED_DEVELOPERS",
				"firewallEnabled": true,
				"secureBootLevel": "FULL_SECURITY"
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
		assert.Equal(t, "C02XYZ", entry.Serial)
		assert.Equal(t, []string{"aa:bb:cc:dd:ee:01"}, entry.MACs)
		assert.Equal(t, []string{"user@example.com"}, entry.OwnerEmails)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Firewall)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.SecureBoot)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Enrolled)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Signals["x.jamf.sip"])
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Signals["x.jamf.gatekeeper"])
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
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
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Signals["x.jamf.sip"])
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
