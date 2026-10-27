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
			"mac_addresses": ["a4:bb:cc:dd:ee:01"],
			"last_callback_at": %q,
			"defender_status": "Protected",
			"firewall_status": "Enabled",
			"tamper_protection_actual": true,
			"organization_id": 7
		}`, now.Add(-time.Hour).UTC().Format(time.RFC3339))), now)

		assert.Equal(t, "42", entry.ExternalID)
		assert.Equal(t, "SERIAL-1", entry.Serial)
		assert.Equal(t, []string{"a4:bb:cc:dd:ee:01"}, entry.MACs)

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

func TestInvalidResponse(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"text/html; charset=utf-8", `<html><body>Sign in</body></html>`},
		{"", `{"agents": [], "pagination": {}}`},
		{"application/json", `{"pagination": {}}`},
		{"application/json", `{"agents": null}`},
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
		require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"huntress\": {\"baseURL\": \"https://api.example.com\", \"apiKey\": \"key\", \"apiSecret\": {\"fromSecret\": \"provider-credential\"}}}}"), ret))
		return ret
	}
	dm := configuration(t)
	manager, err := New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: dm})
	require.NoError(t, err)
	require.NotNil(t, manager)
	assert.Equal(t, enterprisev1.DeviceManager_Status_HUNTRESS, manager.Type())
	require.NoError(t, manager.Close())
	wrong := &enterprisev1.DeviceManager{Metadata: &metav1.Metadata{Name: "wrong"}, Spec: &enterprisev1.DeviceManager_Spec{}}
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: wrong})
	require.Error(t, err)
	for _, field := range []string{"apiSecret", "apiKey"} {
		t.Run("Missing/"+field, func(t *testing.T) {
			encoded, err := protojson.Marshal(configuration(t))
			require.NoError(t, err)
			var document map[string]any
			require.NoError(t, json.Unmarshal(encoded, &document))
			delete(document["spec"].(map[string]any)["huntress"].(map[string]any), field)
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
	document["spec"].(map[string]any)["huntress"].(map[string]any)["apiSecret"] = map[string]any{"fromSecret": "missing-credential"}
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
	document["spec"].(map[string]any)["huntress"].(map[string]any)["baseURL"] = "http://insecure.example.com"
	encoded, err = json.Marshal(document)
	require.NoError(t, err)
	badURL := &enterprisev1.DeviceManager{}
	require.NoError(t, protojson.Unmarshal(encoded, badURL))
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: badURL})
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
		fmt.Fprint(w, "{\"agents\": []}")
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
