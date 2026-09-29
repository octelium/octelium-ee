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

	for _, probe := range m.IdentityProbes() {
		assert.NotEmpty(t, probe.ID)
		assert.NotNil(t, probe.PlatformIdentifier)
		assert.Equal(t, corev1.ClusterConfig_Status_Device_Probe_PlatformIdentifier_HARDWARE_UUID,
			probe.PlatformIdentifier.Kind)
	}
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
			"serial": "C02XYZ",
			"hardware_uuid": "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D",
			"auth_state": "Good",
			"last_seen_at": %q,
			"device_type": "Mac",
			"registered_owner_info": {"identifier": "person-1", "link": "https://api.kolide.com/people/person-1"}
		}`, now.Add(-time.Hour).UTC().Format(time.RFC3339))), emails, now)

		assert.Equal(t, "device-1", entry.ExternalID)
		assert.Equal(t, []string{"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"}, entry.Aliases)
		assert.Equal(t, "C02XYZ", entry.Serial)
		assert.Equal(t, []string{"user@example.com"}, entry.OwnerEmails)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Signals["x.onepassword.noIssues"])
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.Equal(t, "Mac", p.Attrs.Fields["deviceType"].GetStringValue())
		assert.Equal(t, "person-1", p.Attrs.Fields["registeredOwnerId"].GetStringValue())
	}

	{
		entry := toEntry(getDevice(`{
			"id": "device-2",
			"auth_state": "Blocked",
			"registered_owner_info": {"identifier": "person-2"}
		}`), emails, now)

		assert.Empty(t, entry.Aliases)
		assert.Empty(t, entry.OwnerEmails)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, entry.Posture.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, entry.Posture.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, entry.Posture.Signals["x.onepassword.noIssues"])
	}

	for _, tc := range []struct {
		state     string
		compliant corev1.Device_Status_Posture_SignalState
		noIssues  corev1.Device_Status_Posture_SignalState
	}{
		{"Good", corev1.Device_Status_Posture_PASS, corev1.Device_Status_Posture_PASS},
		{"Notified", corev1.Device_Status_Posture_PASS, corev1.Device_Status_Posture_FAIL},
		{"Will Block", corev1.Device_Status_Posture_PASS, corev1.Device_Status_Posture_FAIL},
		{"Blocked", corev1.Device_Status_Posture_FAIL, corev1.Device_Status_Posture_FAIL},
		{"", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
	} {
		assert.Equal(t, tc.compliant, authStateCompliant(tc.state), tc.state)
		assert.Equal(t, tc.noIssues, authStateNoIssues(tc.state), tc.state)
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
				"pagination": {"next_cursor": "Miwy"}
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
