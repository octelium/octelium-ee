// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package sentinelone

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/stretchr/testify/assert"
)

func TestGetAgentsQuery(t *testing.T) {
	{
		q, err := getAgentsQuery([]string{"site-1", "site-2"}, []string{"account-1"}, map[string]string{
			"isDecommissioned": "false",
		})
		assert.Nil(t, err)
		assert.Equal(t, "site-1,site-2", q.Get("siteIds"))
		assert.Equal(t, "account-1", q.Get("accountIds"))
		assert.Equal(t, "false", q.Get("isDecommissioned"))
	}

	{
		q, err := getAgentsQuery(nil, nil, nil)
		assert.Nil(t, err)
		assert.Empty(t, q)
	}

	for _, key := range []string{"siteIds", "accountIds", "cursor", "limit", "SiteIDs"} {
		_, err := getAgentsQuery([]string{"site-1"}, nil, map[string]string{key: "x"})
		assert.NotNil(t, err, key)
	}
}

func TestParseExternalID(t *testing.T) {
	m := &Manager{}

	for _, tc := range []struct {
		results []*devicemgrcommon.ProbeResult
		want    string
	}{
		{nil, ""},
		{[]*devicemgrcommon.ProbeResult{nil, {}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Text: "Agent ID: 0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D\n"}},
			"0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
		{[]*devicemgrcommon.ProbeResult{{Text: "0123456789abcdef0123"}}, "0123456789abcdef0123"},
		{[]*devicemgrcommon.ProbeResult{{Text: "sentinelctl: permission denied"}}, ""},
		{[]*devicemgrcommon.ProbeResult{{Data: []byte("0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d")}}, ""},
	} {
		ret, err := m.ParseExternalID(corev1.Device_Status_LINUX, tc.results)
		assert.Nil(t, err)
		assert.Equal(t, tc.want, ret)
	}
}

func TestToEntry(t *testing.T) {
	now := time.Now()

	getAgent := func(arg string) *s1Agent {
		ret := &s1Agent{}
		assert.Nil(t, json.Unmarshal([]byte(arg), ret))
		return ret
	}

	{
		entry := toEntry(getAgent(fmt.Sprintf(`{
			"id": "1000",
			"uuid": "0A1B2C3D4E5F",
			"serialNumber": "SERIAL-1",
			"isActive": true,
			"isUpToDate": false,
			"infected": false,
			"activeThreats": 0,
			"encryptedApplications": true,
			"firewallEnabled": false,
			"networkStatus": "connected",
			"lastActiveDate": %q,
			"networkInterfaces": [
				{"name": "en0", "physical": "a4:bb:cc:dd:ee:01"},
				{"name": "en1", "physical": "A4:BB:CC:DD:EE:01"},
				{"name": "lo", "physical": "00:00:00:00:00:00"}
			]
		}`, now.Add(-time.Hour).UTC().Format(time.RFC3339))), now)

		assert.Equal(t, "0a1b2c3d4e5f", entry.ExternalID)
		assert.Equal(t, "SERIAL-1", entry.Serial)
		assert.Equal(t, []string{"a4:bb:cc:dd:ee:01"}, entry.MACs)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.Firewall)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.Signals["x.sentinelone.agentUpToDate"])
		assert.True(t, p.LastSeenAt.IsValid())
	}

	{
		p := toEntry(getAgent(`{"uuid": "u"}`), now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.DiskEncryption)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Firewall)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Signals["x.sentinelone.agentUpToDate"])
		assert.Nil(t, p.GetAttrs().GetFields()["infected"])
	}

	{
		p := toEntry(getAgent(`{"uuid": "u", "infected": false}`), now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
	}

	{
		p := toEntry(getAgent(`{"uuid": "u", "activeThreats": 2}`), now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.ThreatFree)
	}

	{
		p := toEntry(getAgent(`{"uuid": "u", "infected": true, "activeThreats": 0, "isActive": false}`), now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
	}

	{
		p := toEntry(getAgent(fmt.Sprintf(`{"uuid": "u", "isActive": true, "lastActiveDate": %q}`,
			now.Add(-48*time.Hour).UTC().Format(time.RFC3339))), now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
	}

	for _, tc := range []struct {
		status string
		want   corev1.Device_Status_Posture_SignalState
	}{
		{"connected", corev1.Device_Status_Posture_PASS},
		{"disconnected", corev1.Device_Status_Posture_FAIL},
		{"disconnecting", corev1.Device_Status_Posture_FAIL},
		{"connecting", corev1.Device_Status_Posture_FAIL},
		{"", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
		{"other", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
	} {
		assert.Equal(t, tc.want, s1NotContained(tc.status), tc.status)
	}
}

func TestListAgents(t *testing.T) {
	var mu sync.Mutex
	var queries []url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		queries = append(queries, r.URL.Query())
		mu.Unlock()

		assert.Equal(t, agentsPath, r.URL.Path)
		assert.Equal(t, "ApiToken token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("cursor") {
		case "":
			fmt.Fprint(w, `{"data": [{"uuid": "a"}, {"uuid": "b"}], "pagination": {"nextCursor": "page-2"}}`)
		case "page-2":
			fmt.Fprint(w, `{"data": [{"uuid": "c"}], "pagination": {"nextCursor": null}}`)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	m := &Manager{
		api: &apiClient{
			rc: resty.New().SetBaseURL(srv.URL).SetHeader("Authorization", "ApiToken token"),
		},
		siteIDs:      []string{"site-1"},
		agentFilters: map[string]string{"isDecommissioned": "false"},
	}

	fleet, err := m.Collect(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 3, fleet.Len())
	assert.Equal(t, devicemgrcommon.MatchStateUnique, fleet.MatchExternalID("c").State)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, queries, 2)
	for _, q := range queries {
		assert.Equal(t, "site-1", q.Get("siteIds"))
		assert.Equal(t, "false", q.Get("isDecommissioned"))
		assert.Equal(t, "1000", q.Get("limit"))
	}
}

func TestListAgentsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"errors": [{"title": "Forbidden"}]}`)
	}))
	t.Cleanup(srv.Close)

	m := &Manager{
		api: &apiClient{
			rc: resty.New().SetBaseURL(srv.URL),
		},
	}

	_, err := m.Collect(context.Background())
	assert.NotNil(t, err)

	m.agentFilters = map[string]string{"cursor": "x"}
	_, err = m.Collect(context.Background())
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

func TestExtractAgentID(t *testing.T) {
	for _, tc := range []struct {
		arg  string
		want string
	}{
		{" 0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D\r\n", "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
		{"0123456789abcdef0123456789abcdef", "0123456789abcdef0123456789abcdef"},
		{"Agent ID: 0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d\nAgent ID: 0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d\n", "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
		{"Agent ID: 0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d\nSite ID: 1a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d\n", ""},
		{"Agent ID: 0123456789abcdef0123456789abcdef", ""},
		{"", ""},
	} {
		assert.Equal(t, tc.want, extractAgentID(tc.arg), tc.arg)
	}

	m := &Manager{}
	_, err := m.ParseExternalID(corev1.Device_Status_MAC, []*devicemgrcommon.ProbeResult{
		{Text: "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
		{Text: "1a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"},
	})
	assert.NotNil(t, err)
}
