// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package crowdstrike

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/stretchr/testify/assert"
)

func TestParseExternalID(t *testing.T) {
	m := &Manager{}
	aid := "0123456789abcdef0123456789abcdef"

	raw, err := hex.DecodeString(aid)
	assert.Nil(t, err)

	for _, tc := range []struct {
		results []*devicemgrcommon.ProbeResult
		want    string
	}{
		{nil, ""},
		{[]*devicemgrcommon.ProbeResult{nil, {}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Data: raw}}, aid},
		{[]*devicemgrcommon.ProbeResult{{Data: raw[:8]}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Text: fmt.Sprintf(`aid="%s".`, aid)}}, aid},
		{[]*devicemgrcommon.ProbeResult{{Text: "agentID: 01234567-89AB-CDEF-0123-456789ABCDEF\n"}}, aid},
		{[]*devicemgrcommon.ProbeResult{{Text: " 0123456789ABCDEF0123456789ABCDEF\n"}}, aid},
		{[]*devicemgrcommon.ProbeResult{{Text: "cid=ffffffffffffffffffffffffffffffff " + aid}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Text: "ERROR: aid is not available"}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Text: "invalid"}, {Text: `aid="` + aid + `"`}}, aid},
	} {
		ret, err := m.ParseExternalID(corev1.Device_Status_LINUX, tc.results)
		assert.Nil(t, err)
		assert.Equal(t, tc.want, ret)
	}
}

func TestIdentityProbes(t *testing.T) {
	m := &Manager{}
	seen := map[string]struct{}{}

	for _, probe := range m.IdentityProbes() {
		assert.NotEmpty(t, probe.ID)
		assert.Len(t, probe.OSTypes, 1)
		assert.True(t, probe.RequireElevation)

		_, ok := seen[probe.ID]
		assert.False(t, ok)
		seen[probe.ID] = struct{}{}
	}

	assert.Len(t, seen, 3)
}

func TestToEntry(t *testing.T) {
	now := time.Now()

	getDevice := func(status, rfm string, lastSeen time.Time) *models.DeviceapiDeviceSwagger {
		ret := &models.DeviceapiDeviceSwagger{}
		err := json.Unmarshal([]byte(fmt.Sprintf(`{
			"device_id": "0123456789ABCDEF0123456789ABCDEF",
			"serial_number": "C02XYZ",
			"mac_address": "aa-bb-cc-dd-ee-01",
			"connection_mac_address": "AA-BB-CC-DD-EE-01",
			"status": %q,
			"reduced_functionality_mode": %q,
			"last_seen": %q,
			"platform_name": "Windows",
			"agent_version": "7.10.0"
		}`, status, rfm, lastSeen.UTC().Format(time.RFC3339))), ret)
		assert.Nil(t, err)
		return ret
	}

	{
		entry := toEntry(getDevice("normal", "no", now.Add(-time.Hour)), nil, now)
		assert.Equal(t, "0123456789abcdef0123456789abcdef", entry.ExternalID)
		assert.Equal(t, "C02XYZ", entry.Serial)
		assert.Equal(t, []string{"aa-bb-cc-dd-ee-01"}, entry.MACs)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.Empty(t, p.Signals)
		assert.True(t, p.LastSeenAt.IsValid())
		assert.Equal(t, "7.10.0", p.Attrs.Fields["agentVersion"].GetStringValue())
	}

	{
		p := toEntry(getDevice("contained", "no", now), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
	}

	{
		p := toEntry(getDevice("lift_containment_pending", "yes", now), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
	}

	{
		p := toEntry(getDevice("", "", now), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AgentHealthy)
	}

	{
		p := toEntry(getDevice("normal", "no", now.Add(-72*time.Hour)), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
	}

	{
		p := toEntry(getDevice("normal", "", now.Add(-72*time.Hour)), nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
	}

	{
		d := getDevice("normal", "no", now)
		d.LastSeen = ""
		p := toEntry(d, nil, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AgentHealthy)
		assert.Nil(t, p.LastSeenAt)
	}

	{
		zta := &models.DomainSignalProperties{}
		err := json.Unmarshal([]byte(`{
			"aid": "0123456789abcdef0123456789abcdef",
			"cid": "cid-1",
			"assessment": {"overall": 95, "os": 80, "sensor_config": 70}
		}`), zta)
		assert.Nil(t, err)

		p := toEntry(getDevice("normal", "no", now), zta, now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_LOW, p.RiskLevel)
		assert.EqualValues(t, 95, p.Attrs.Fields["ztaScoreOverall"].GetNumberValue())
		assert.EqualValues(t, 80, p.Attrs.Fields["ztaScoreOs"].GetNumberValue())
		assert.EqualValues(t, 70, p.Attrs.Fields["ztaScoreSensor"].GetNumberValue())
		assert.Equal(t, "cid-1", p.Attrs.Fields["cid"].GetStringValue())
	}

	assert.Equal(t, corev1.Device_Status_Posture_LOW, csRisk(90))
	assert.Equal(t, corev1.Device_Status_Posture_MEDIUM, csRisk(70))
	assert.Equal(t, corev1.Device_Status_Posture_HIGH, csRisk(40))
	assert.Equal(t, corev1.Device_Status_Posture_CRITICAL, csRisk(39))
}

func TestChunk(t *testing.T) {
	assert.Nil(t, chunk(nil, 2))
	assert.Equal(t, [][]string{{"a", "b"}, {"c"}}, chunk([]string{"a", "b", "c"}, 2))
	assert.Equal(t, [][]string{{"a"}}, chunk([]string{"a"}, 0))
}
