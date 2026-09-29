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
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/stretchr/testify/assert"
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

	for _, probe := range m.IdentityProbes() {
		assert.Nil(t, probe.RunCommand)
		assert.NotNil(t, probe.PlatformIdentifier)
		assert.Equal(t, probe.ID == "hardware-uuid-linux", probe.RequireElevation)
	}
}

func TestToEntry(t *testing.T) {
	getHost := func(arg string) *fleetHost {
		ret := &fleetHost{}
		assert.Nil(t, json.Unmarshal([]byte(arg), ret))
		return ret
	}

	{
		entry := toEntry(getHost(`{
			"id": 10,
			"uuid": "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D",
			"hardware_serial": "C02XYZ",
			"primary_mac": "aa:bb:cc:dd:ee:01",
			"status": "online",
			"seen_time": "2026-09-28T10:00:00Z",
			"disk_encryption_enabled": true,
			"issues": {"failing_policies_count": 0, "total_issues_count": 0},
			"mdm": {"enrollment_status": "On (automatic)", "name": "Fleet"}
		}`))

		assert.Equal(t, "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", entry.ExternalID)
		assert.Equal(t, "C02XYZ", entry.Serial)
		assert.Equal(t, []string{"aa:bb:cc:dd:ee:01"}, entry.MACs)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Signals["x.fleetdm.mdmEnrolled"])
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.True(t, p.LastSeenAt.IsValid())
	}

	{
		entry := toEntry(getHost(`{"uuid": "u", "status": "offline", "issues": {"failing_policies_count": 3}}`))
		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
		assert.Empty(t, p.Signals)
		assert.Empty(t, entry.MACs)
	}

	{
		entry := toEntry(getHost(`{"uuid": "u", "status": "new", "issues": {}}`))
		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AgentHealthy)
	}

	{
		entry := toEntry(getHost(`{"uuid": "u"}`))
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, entry.Posture.Compliant)
	}

	assert.Equal(t, corev1.Device_Status_Posture_FAIL, fleetMDMEnrolled("Off"))
	assert.Equal(t, corev1.Device_Status_Posture_PASS, fleetMDMEnrolled("On (manual)"))
	assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, fleetMDMEnrolled("Pending"))
	assert.Equal(t, corev1.Device_Status_Posture_FAIL, fleetAgentHealthy("missing"))
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
