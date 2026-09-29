// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package crowdstrike

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/crowdstrike/gofalcon/falcon"
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
			"mac_address": "a4-bb-cc-dd-ee-01",
			"connection_mac_address": "A4-BB-CC-DD-EE-01",
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
		assert.Equal(t, []string{"a4-bb-cc-dd-ee-01"}, entry.MACs)

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

func TestParseExternalIDDisagreement(t *testing.T) {
	m := &Manager{}

	_, err := m.ParseExternalID(corev1.Device_Status_LINUX, []*devicemgrcommon.ProbeResult{
		{Text: `aid="0123456789abcdef0123456789abcdef"`},
		{Text: `aid="ffffffffffffffffffffffffffffffff"`},
	})
	assert.NotNil(t, err)
}

func TestGetPayloadError(t *testing.T) {
	code := int32(404)
	message := "not found"

	assert.Nil(t, getPayloadError(nil))
	assert.Nil(t, getPayloadError([]*models.MsaAPIError{nil}))

	{
		err := getPayloadError([]*models.MsaAPIError{{Code: &code, Message: &message}})
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "404")
		assert.Contains(t, err.Error(), message)
	}

	{
		err := getPayloadError([]*models.MsaAPIError{{Code: &code, Message: &message}, {}})
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "2 errors")
	}

	{
		err := getPayloadError([]*models.MsaAPIError{{}})
		assert.NotNil(t, err)
	}
}

type tstTransport struct{}

func (tstTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	return http.DefaultTransport.RoundTrip(req)
}

func TestCollect(t *testing.T) {
	aid1 := "0123456789abcdef0123456789abcdef"
	aid2 := "fedcba9876543210fedcba9876543210"

	mode := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/devices/queries/devices-scroll/v1":
			switch mode {
			case "scrollError":
				fmt.Fprint(w, `{"meta": {}, "resources": [], "errors": [{"code": 500, "message": "internal"}]}`)
			case "empty":
				fmt.Fprint(w, `{"meta": {"pagination": {"offset": "", "total": 0}}, "resources": [], "errors": []}`)
			default:
				fmt.Fprintf(w, `{"meta": {"pagination": {"offset": "", "total": 2}}, "resources": [%q, %q], "errors": []}`,
					strings.ToUpper(aid1), aid2)
			}
		case "/devices/entities/devices/v2":
			assert.Equal(t, http.MethodPost, r.Method)

			req := &models.MsaIdsRequest{}
			assert.Nil(t, json.NewDecoder(r.Body).Decode(req))
			assert.Equal(t, []string{strings.ToUpper(aid1), aid2}, req.Ids)

			errs := `[]`
			if mode == "detailsError" {
				errs = `[{"code": 404, "message": "not found"}]`
			}
			fmt.Fprintf(w, `{"meta": {}, "resources": [
				{"device_id": %q, "status": "normal", "serial_number": "SERIAL-1"},
				{"device_id": %q, "status": "contained"}
			], "errors": %s}`, strings.ToUpper(aid1), aid2, errs)
		case "/zero-trust-assessment/entities/assessments/v1":
			if mode == "ztaError" {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"meta": {}, "errors": [{"code": 403, "message": "access denied"}]}`)
				return
			}
			fmt.Fprintf(w, `{"meta": {}, "resources": [{"aid": %q, "assessment": {"overall": 95}}],
				"errors": [{"code": 404, "message": "no assessment"}]}`, aid1)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := falcon.NewClient(&falcon.ApiConfig{
		Context:      context.Background(),
		AccessToken:  "token",
		HostOverride: strings.TrimPrefix(srv.URL, "http://"),
		TransportDecorator: func(http.RoundTripper) http.RoundTripper {
			return tstTransport{}
		},
	})
	assert.Nil(t, err)

	m := &Manager{c: c, ztaEnabled: true}

	{
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 2, fleet.Len())
		assert.Empty(t, fleet.DegradedReason())

		res := fleet.MatchExternalID(aid1)
		assert.Equal(t, devicemgrcommon.MatchStateUnique, res.State)
		assert.Equal(t, corev1.Device_Status_Posture_LOW, res.Entry.Posture.RiskLevel)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, res.Entry.Posture.NotContained)

		res = fleet.MatchExternalID(aid2)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, res.Entry.Posture.RiskLevel)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, res.Entry.Posture.NotContained)
	}

	{
		mode = "ztaError"
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 2, fleet.Len())
		assert.NotEmpty(t, fleet.DegradedReason())
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN,
			fleet.MatchExternalID(aid1).Entry.Posture.RiskLevel)
	}

	{
		mode = "detailsError"
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 2, fleet.Len())
		assert.Contains(t, fleet.DegradedReason(), "not found")
	}

	{
		mode = "scrollError"
		_, err := m.Collect(context.Background())
		assert.NotNil(t, err)
	}

	{
		mode = "empty"
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 0, fleet.Len())
	}

	{
		mode = ""
		m.ztaEnabled = false
		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Empty(t, fleet.DegradedReason())
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN,
			fleet.MatchExternalID(aid1).Entry.Posture.RiskLevel)
	}
}
