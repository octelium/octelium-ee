// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package gcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gcpdns "google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
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

func TestGCPRecordValueNormalization(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, normalizeValues([]string{"b", "a", "b", ""}))
	assert.True(t, equalValues([]string{"a"}, []string{"a"}))
	assert.False(t, equalValues([]string{"a"}, nil))
	assert.False(t, equalValues([]string{"a"}, []string{"b"}))
	assert.Equal(t, "api.example.com.", normalizeFQDN(" API.Example.COM. "))
}

func TestGCPDNSRecordReconciliation(t *testing.T) {
	var mu sync.Mutex
	records := map[string]*gcpdns.ResourceRecordSet{}
	changes := 0
	readChanges := func() int { mu.Lock(); defer mu.Unlock(); return changes }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/managedZones"):
			io.WriteString(w, `{"managedZones":[{"name":"parent","dnsName":"example.com."},{"name":"child","dnsName":"sub.example.com."}]}`)
		case strings.HasSuffix(r.URL.Path, "/rrsets"):
			assert.Contains(t, r.URL.Path, "/child/")
			var result []*gcpdns.ResourceRecordSet
			for _, record := range records {
				if record.Name == r.URL.Query().Get("name") && record.Type == r.URL.Query().Get("type") {
					result = append(result, record)
				}
			}
			json.NewEncoder(w).Encode(&gcpdns.ResourceRecordSetsListResponse{Rrsets: result})
		case strings.HasSuffix(r.URL.Path, "/changes"):
			var change gcpdns.Change
			require.NoError(t, json.NewDecoder(r.Body).Decode(&change))
			changes++
			for _, record := range change.Deletions {
				delete(records, record.Name+record.Type)
			}
			for _, record := range change.Additions {
				records[record.Name+record.Type] = record
			}
			io.WriteString(w, `{"id":"change","status":"done"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	client, err := gcpdns.NewService(t.Context(), option.WithEndpoint(srv.URL+"/"), option.WithHTTPClient(srv.Client()), option.WithoutAuthentication())
	require.NoError(t, err)
	p := &Provider{
		c:      client,
		domain: "app.sub.example.com",
		p: &enterprisev1.DNSProvider{
			Spec: &enterprisev1.DNSProvider_Spec{
				Type: &enterprisev1.DNSProvider_Spec_Google_{Google: &enterprisev1.DNSProvider_Spec_Google{Project: "project"}},
			},
		},
	}
	require.NoError(t, p.Set(t.Context(), "app.sub.example.com", []string{"192.0.2.1", "192.0.2.1", "2001:db8::1"}))
	assert.Equal(t, 2, readChanges())
	require.NoError(t, p.Set(t.Context(), "app.sub.example.com", []string{"192.0.2.1", "2001:db8::1"}))
	assert.Equal(t, 2, readChanges())
	require.NoError(t, p.SetCNAME(t.Context(), "portal.sub.example.com", "app.sub.example.com"))
	assert.Equal(t, 3, readChanges())
	require.NoError(t, p.Set(t.Context(), "app.sub.example.com", []string{"192.0.2.2"}))
	assert.Equal(t, 5, readChanges())
	mu.Lock()
	defer mu.Unlock()
	assert.NotContains(t, records, "app.sub.example.com.AAAA")
	assert.Equal(t, []string{"app.sub.example.com."}, records["portal.sub.example.com.CNAME"].Rrdatas)
}
