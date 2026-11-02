// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFleetServeHTTP(t *testing.T) {
	f := &Fleet{}
	f.SetBearer("token")
	seenTime := "2026-01-02T03:04:05Z"
	f.SetHosts([]FleetHost{
		{ID: 1, Status: "online"},
		{ID: 2, Status: "offline", SeenTime: seenTime},
	})

	do := func(path, bearer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		f.ServeHTTP(w, req)
		return w
	}

	getHosts := func(w *httptest.ResponseRecorder) []FleetHost {
		require.Equal(t, http.StatusOK, w.Code)
		var ret struct {
			Hosts []FleetHost `json:"hosts"`
		}
		require.Nil(t, json.NewDecoder(w.Body).Decode(&ret))
		return ret.Hosts
	}

	const hostsPath = "/api/v1/fleet/hosts?page=0&per_page=100"

	{
		before := time.Now().UTC().Truncate(time.Second)
		hosts := getHosts(do(hostsPath, "token"))
		require.Len(t, hosts, 2)

		seenAt, err := time.Parse(time.RFC3339, hosts[0].SeenTime)
		require.Nil(t, err, "a host without a seen_time must be reported as seen now")
		assert.False(t, seenAt.Before(before))
		assert.Equal(t, seenTime, hosts[1].SeenTime)

		f.mu.Lock()
		assert.Empty(t, f.hosts[0].SeenTime)
		f.mu.Unlock()
	}

	assert.Len(t, getHosts(do("/api/v1/fleet/hosts?page=1&per_page=1", "token")), 1)
	assert.Empty(t, getHosts(do("/api/v1/fleet/hosts?page=2&per_page=1", "token")))

	assert.Equal(t, http.StatusUnauthorized, do(hostsPath, "").Code)
	assert.Equal(t, http.StatusUnauthorized, do(hostsPath, "other").Code)
	assert.Equal(t, http.StatusNotFound, do("/api/v1/fleet/other", "token").Code)
	assert.Equal(t, http.StatusBadRequest, do("/api/v1/fleet/hosts?page=0&per_page=0", "token").Code)

	{
		f.SetFailures(http.StatusServiceUnavailable, http.StatusTooManyRequests)
		assert.Equal(t, 2, f.PendingFailures())

		assert.Equal(t, http.StatusServiceUnavailable, do(hostsPath, "token").Code)
		w := do(hostsPath, "token")
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "1", w.Header().Get("Retry-After"))
		assert.Zero(t, f.PendingFailures())
		assert.Len(t, getHosts(do(hostsPath, "token")), 2)
	}

	{
		f.SetRedirect(true)
		w := do(hostsPath, "token")
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "http://")

		f.SetRedirect(false)
		assert.Len(t, getHosts(do(hostsPath, "token")), 2)
	}
}
