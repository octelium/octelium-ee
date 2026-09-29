// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package huntress

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/stretchr/testify/assert"
)

func TestToEntry(t *testing.T) {
	now := time.Now()

	getAgent := func(arg string) *huntressAgent {
		ret := &huntressAgent{}
		assert.Nil(t, json.Unmarshal([]byte(arg), ret))
		return ret
	}

	{
		entry := toEntry(getAgent(fmt.Sprintf(`{
			"id": 42,
			"platform": "windows",
			"serial_number": "SERIAL-1",
			"mac_addresses": ["aa:bb:cc:dd:ee:01"],
			"last_callback_at": %q,
			"defender_status": "Protected",
			"firewall_status": "Enabled",
			"tamper_protection_actual": true,
			"organization_id": 7
		}`, now.Add(-time.Hour).UTC().Format(time.RFC3339))), now)

		assert.Equal(t, "42", entry.ExternalID)
		assert.Equal(t, "SERIAL-1", entry.Serial)
		assert.Equal(t, []string{"aa:bb:cc:dd:ee:01"}, entry.MACs)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Signals["x.huntress.defender"])
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Signals["x.huntress.tamperProtection"])
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.ThreatFree)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.True(t, p.LastSeenAt.IsValid())
		assert.Equal(t, "Enabled", p.Attrs.Fields["firewallStatus"].GetStringValue())
	}

	{
		entry := toEntry(getAgent(fmt.Sprintf(`{
			"id": 43,
			"platform": "darwin",
			"last_callback_at": %q,
			"firewall_status": "Isolated",
			"tamper_protection_actual": null
		}`, now.Add(-48*time.Hour).UTC().Format(time.RFC3339))), now)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_NOT_APPLICABLE, p.Signals["x.huntress.defender"])
		_, ok := p.Signals["x.huntress.tamperProtection"]
		assert.False(t, ok)
	}

	{
		p := toEntry(getAgent(`{"id": 44, "platform": "windows", "last_seen": "2026-09-28T10:00:00Z"}`), now).Posture
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.NotContained)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Signals["x.huntress.defender"])
		assert.Nil(t, p.LastSeenAt)
	}

	for _, tc := range []struct {
		status string
		want   corev1.Device_Status_Posture_SignalState
	}{
		{"Enabled", corev1.Device_Status_Posture_PASS},
		{"Disabled", corev1.Device_Status_Posture_PASS},
		{"Pending Isolation", corev1.Device_Status_Posture_FAIL},
		{"Isolated", corev1.Device_Status_Posture_FAIL},
		{"Pending Release", corev1.Device_Status_Posture_FAIL},
		{"", corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN},
	} {
		assert.Equal(t, tc.want, notContainedSignal(tc.status), tc.status)
	}

	assert.Equal(t, corev1.Device_Status_Posture_FAIL, defenderSignal("windows", "Unprotected"))
}

func TestListAgents(t *testing.T) {
	var mu sync.Mutex
	var orgs []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, agentsPath, r.URL.Path)
		assert.Equal(t, "500", q.Get("limit"))

		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "key", user)
		assert.Equal(t, "secret", pass)

		mu.Lock()
		orgs = append(orgs, q.Get("organization_id"))
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch q.Get("organization_id") + "|" + q.Get("page_token") {
		case "1|":
			fmt.Fprint(w, `{"agents": [{"id": 1}, {"id": 2}], "pagination": {"next_page_token": "t2", "next_page_url": "x"}}`)
		case "1|t2":
			fmt.Fprint(w, `{"agents": [{"id": 3}, {"id": 0}], "pagination": {"next_page_token": null}}`)
		case "2|":
			fmt.Fprint(w, `{"agents": [{"id": 4}], "pagination": {"next_page_token": ""}}`)
		case "|":
			fmt.Fprint(w, `{"agents": [{"id": 5}], "pagination": {"next_page_token": "t2"}}`)
		case "|t2":
			fmt.Fprint(w, `{"agents": [{"id": 6}], "pagination": {"next_page_token": "t2"}}`)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	rc := resty.New().SetBaseURL(srv.URL).SetBasicAuth("key", "secret")

	{
		m := &Manager{
			api:             &apiClient{rc: rc},
			organizationIDs: []int64{1, 2},
		}

		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 4, fleet.Len())
		assert.Equal(t, devicemgrcommon.MatchStateUnique, fleet.MatchExternalID("4").State)
		assert.Equal(t, devicemgrcommon.MatchStateNone, fleet.MatchExternalID("0").State)
	}

	{
		m := &Manager{
			api: &apiClient{rc: rc},
		}

		fleet, err := m.Collect(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 2, fleet.Len())
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"1", "1", "2", "", ""}, orgs)
}
