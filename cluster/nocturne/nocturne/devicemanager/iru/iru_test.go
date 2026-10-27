// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package iru

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

func TestToEntry(t *testing.T) {
	now := time.Now()

	getDevice := func(arg string) *iruDevice {
		ret := &iruDevice{}
		assert.Nil(t, json.Unmarshal([]byte(arg), ret))
		return ret
	}

	{
		entry := toEntry(getDevice(fmt.Sprintf(`{
			"device_id": "0A1B2C3D-4E5F-6A7B-8C9D-0E1F2A3B4C5D",
			"serial_number": "C02XYZ",
			"mac_address": "a4:bb:cc:dd:ee:01",
			"last_check_in": %q,
			"is_missing": false,
			"mdm_enabled": true,
			"agent_installed": true,
			"user": {"email": "user@example.com", "name": "User"}
		}`, now.Add(-time.Hour).UTC().Format(time.RFC3339))), now)

		assert.Equal(t, "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", entry.ExternalID)
		assert.Equal(t, "C02XYZ", entry.Serial)
		assert.Equal(t, []string{"a4:bb:cc:dd:ee:01"}, entry.MACs)
		assert.Equal(t, []string{"user@example.com"}, entry.OwnerEmails)

		p := entry.Posture
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.Enrolled)
		assert.Equal(t, corev1.Device_Status_Posture_PASS, p.AgentHealthy)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, p.Compliant)
		assert.Equal(t, corev1.Device_Status_Posture_RISK_LEVEL_UNKNOWN, p.RiskLevel)
		assert.Equal(t, "User", p.Attrs.Fields["userName"].GetStringValue())
	}

	for _, user := range []string{`""`, `null`, `[]`, `"user@example.com"`} {
		entry := toEntry(getDevice(fmt.Sprintf(`{
			"device_id": "d",
			"is_missing": true,
			"mdm_enabled": false,
			"agent_installed": true,
			"user": %s
		}`, user)), now)

		assert.Empty(t, entry.OwnerEmails, user)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, entry.Posture.Enrolled)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, entry.Posture.AgentHealthy)
	}

	{
		entry := toEntry(getDevice(`{"device_id": "d"}`), now)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, entry.Posture.Enrolled)
		assert.Equal(t, corev1.Device_Status_Posture_SIGNAL_STATE_UNKNOWN, entry.Posture.AgentHealthy)
		assert.Empty(t, entry.MACs)
	}

	{
		entry := toEntry(getDevice(fmt.Sprintf(`{
			"device_id": "d",
			"is_missing": false,
			"agent_installed": true,
			"last_check_in": %q
		}`, now.Add(-72*time.Hour).UTC().Format(time.RFC3339))), now)
		assert.Equal(t, corev1.Device_Status_Posture_FAIL, entry.Posture.AgentHealthy)
	}
}

func TestListDevices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, devicesPath, r.URL.Path)
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"device_id": "a", "user": ""},
			{"device_id": "b", "is_removed": true},
			{"device_id": "", "serial_number": "no-id"},
			{"device_id": "c", "user": {"email": "user@example.com"}}
		]`)
	}))
	t.Cleanup(srv.Close)

	m := &Manager{
		api: &apiClient{rc: resty.New().SetBaseURL(srv.URL).SetAuthToken("token")},
	}

	fleet, err := m.Collect(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 2, fleet.Len())
	assert.Equal(t, devicemgrcommon.MatchStateNone, fleet.MatchExternalID("b").State)
	assert.Equal(t, []string{"user@example.com"}, fleet.MatchExternalID("c").Entry.OwnerEmails)
}

func TestInvalidResponse(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"text/html; charset=utf-8", `<html><body>Sign in</body></html>`},
		{"", `[]`},
		{"application/json", `{}`},
		{"application/json", `null`},
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
		require.NoError(t, protojson.Unmarshal([]byte("{\"metadata\": {\"name\": \"constructor-test\"}, \"spec\": {\"iru\": {\"baseURL\": \"https://api.example.com\", \"apiToken\": {\"fromSecret\": \"provider-credential\"}}}}"), ret))
		return ret
	}
	dm := configuration(t)
	manager, err := New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: dm})
	require.NoError(t, err)
	require.NotNil(t, manager)
	assert.Equal(t, enterprisev1.DeviceManager_Status_IRU, manager.Type())
	require.NoError(t, manager.Close())
	wrong := &enterprisev1.DeviceManager{Metadata: &metav1.Metadata{Name: "wrong"}, Spec: &enterprisev1.DeviceManager_Spec{}}
	_, err = New(t.Context(), tst.C.OcteliumC, &devicemgrcommon.ManagerOpts{DeviceManager: wrong})
	require.Error(t, err)
	for _, field := range []string{"apiToken"} {
		t.Run("Missing/"+field, func(t *testing.T) {
			encoded, err := protojson.Marshal(configuration(t))
			require.NoError(t, err)
			var document map[string]any
			require.NoError(t, json.Unmarshal(encoded, &document))
			delete(document["spec"].(map[string]any)["iru"].(map[string]any), field)
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
	document["spec"].(map[string]any)["iru"].(map[string]any)["apiToken"] = map[string]any{"fromSecret": "missing-credential"}
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
	document["spec"].(map[string]any)["iru"].(map[string]any)["baseURL"] = "http://insecure.example.com"
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
		fmt.Fprint(w, "[]")
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
