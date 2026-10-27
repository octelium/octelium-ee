package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"slices"
	"sync"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSAddressFamiliesAndInvalidInput(t *testing.T) {
	v4, v6, err := splitIPAddresses([]string{"192.0.2.1", "2001:db8::1", "192.0.2.2"})
	require.NoError(t, err)
	assert.Equal(t, []string{"192.0.2.1", "192.0.2.2"}, v4)
	assert.Equal(t, []string{"2001:db8::1"}, v6)
	v4, v6, err = splitIPAddresses(nil)
	assert.NoError(t, err)
	assert.Empty(t, v4)
	assert.Empty(t, v6)
	for _, invalid := range []string{"", "hostname.example.com", "192.0.2.1/24", "999.0.0.1", "[2001:db8::1]"} {
		_, _, err := splitIPAddresses([]string{"192.0.2.1", invalid})
		assert.Error(t, err, invalid)
	}
}

func TestDNSZoneBoundaries(t *testing.T) {
	for _, tc := range []struct {
		domain, zone string
		want         bool
	}{
		{"example.com", "example.com", true}, {"api.example.com", "example.com", true}, {"notexample.com", "example.com", false}, {"example.com.evil", "example.com", false}, {"other.com", "example.com", false},
	} {
		assert.Equal(t, tc.want, domainMatchesZone(tc.domain, tc.zone))
	}
	assert.Equal(t, "api.example.com", normalizeDomain(" API.Example.COM. "))
}

func TestDNSDesiredValuesDeduplicated(t *testing.T) {
	assert.Equal(t, map[string]struct{}{"one": {}, "two": {}}, uniqueValues([]string{"", "one", "two", "one"}))
}

func TestCloudflareDNSRecordReconciliation(t *testing.T) {
	var mu sync.Mutex
	records := []cloudflare.DNSRecord{{ID: "keep", Type: "A", Name: "app.sub.example.com", Content: "192.0.2.1"}, {ID: "duplicate", Type: "A", Name: "app.sub.example.com", Content: "192.0.2.1"}, {ID: "stale", Type: "A", Name: "app.sub.example.com", Content: "192.0.2.9"}, {ID: "other", Type: "A", Name: "other.example.com", Content: "192.0.2.10"}}
	created, deleted := 0, 0
	readCreated := func() int { mu.Lock(); defer mu.Unlock(); return created }
	readDeleted := func() int { mu.Lock(); defer mu.Unlock(); return deleted }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/zones":
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []map[string]string{{"id": "parent", "name": "example.com"}, {"id": "child", "name": "sub.example.com"}}, "result_info": map[string]int{"page": 1, "total_pages": 1}})
		case r.Method == http.MethodGet:
			assert.Contains(t, r.URL.Path, "/zones/child/")
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": records, "result_info": map[string]int{"page": 1, "total_pages": 1}})
		case r.Method == http.MethodDelete:
			id := path.Base(r.URL.Path)
			records = slices.DeleteFunc(records, func(record cloudflare.DNSRecord) bool { return record.ID == id })
			deleted++
			io.WriteString(w, `{"success":true,"result":{}}`)
		case r.Method == http.MethodPost:
			var record cloudflare.DNSRecord
			require.NoError(t, json.NewDecoder(r.Body).Decode(&record))
			created++
			record.ID = fmt.Sprintf("new-%d", created)
			records = append(records, record)
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": record})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	client, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL), cloudflare.HTTPClient(srv.Client()), cloudflare.UsingRateLimit(10000), cloudflare.UsingRetryPolicy(0, 0, 0))
	require.NoError(t, err)
	p := &Provider{c: client, domain: "APP.SUB.EXAMPLE.COM."}
	require.NoError(t, p.Set(t.Context(), "app.sub.example.com", []string{"192.0.2.1", "192.0.2.1", "2001:db8::1"}))
	assert.Equal(t, 2, readDeleted())
	assert.Equal(t, 1, readCreated())
	require.NoError(t, p.Set(t.Context(), "app.sub.example.com", []string{"192.0.2.1", "2001:db8::1"}))
	assert.Equal(t, 1, readCreated())
	require.NoError(t, p.SetCNAME(t.Context(), "portal.sub.example.com", "app.sub.example.com"))
	assert.Equal(t, 2, readCreated())
	require.NoError(t, p.Delete(t.Context(), "app.sub.example.com"))
	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, records, 2)
	assert.True(t, slices.ContainsFunc(records, func(r cloudflare.DNSRecord) bool { return r.ID == "other" }))
}

func TestCloudflareZoneErrorsAndCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"success":true,"result":[{"id":"unrelated","name":"other.com"}],"result_info":{"page":1,"total_pages":1}}`)
	}))
	t.Cleanup(srv.Close)
	client, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL), cloudflare.HTTPClient(srv.Client()), cloudflare.UsingRetryPolicy(0, 0, 0))
	require.NoError(t, err)
	p := &Provider{c: client, domain: "example.com"}
	_, err = p.getZoneID(t.Context())
	assert.ErrorContains(t, err, "Could not find")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.Error(t, p.SetCNAME(ctx, "app.example.com", "region.example.com"))
}
