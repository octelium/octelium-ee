package suite

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/go-resty/resty/v2"
	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrafficDriverContext(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		requests int32
	}{
		{name: "Allowed", status: http.StatusOK, requests: driveBatch},
		{name: "Denied", status: http.StatusForbidden, requests: 1},
		{name: "Unavailable", status: http.StatusServiceUnavailable, requests: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			d := &trafficDriver{
				h: eeharness.Wrap(&harness.H{}),
				c: resty.New().SetBaseURL(srv.URL),
			}
			err := d.driveContext(t.Context())
			if tc.status == http.StatusOK {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
			assert.Equal(t, tc.requests, requests.Load())
		})
	}
}

func TestTrafficDriverContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
	}))
	defer srv.Close()

	d := &trafficDriver{
		h: eeharness.Wrap(&harness.H{}),
		c: resty.New().SetBaseURL(srv.URL),
	}
	require.ErrorIs(t, d.driveContext(ctx), context.Canceled)
}
