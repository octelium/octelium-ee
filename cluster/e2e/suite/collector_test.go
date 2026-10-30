// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/go-resty/resty/v2"
	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type collectorExporterPaginationClient struct {
	enterprisev1.MainServiceClient
	t     *testing.T
	ctx   context.Context
	items []*enterprisev1.CollectorExporter
	calls uint32
}

func (c *collectorExporterPaginationClient) ListCollectorExporter(ctx context.Context,
	req *enterprisev1.ListCollectorExporterOptions, opts ...grpc.CallOption) (*enterprisev1.CollectorExporterList, error) {
	require.Equal(c.t, c.ctx, ctx)
	require.Nil(c.t, ctx.Err())
	require.NotNil(c.t, req.Common)
	assert.Equal(c.t, c.calls, req.Common.Page)
	assert.Equal(c.t, uint32(2), req.Common.ItemsPerPage)
	assert.Equal(c.t, &metav1.CommonListOptions_OrderBy{
		Type: metav1.CommonListOptions_OrderBy_NAME,
		Mode: metav1.CommonListOptions_OrderBy_ASC,
	}, req.Common.OrderBy)
	c.calls++

	start := int(req.Common.Page * req.Common.ItemsPerPage)
	if start >= len(c.items) && req.Common.Page > 0 {
		return nil, status.Error(codes.NotFound, "Not Items found for that page")
	}
	end := min(start+int(req.Common.ItemsPerPage), len(c.items))
	return &enterprisev1.CollectorExporterList{
		Items: c.items[start:end],
		ListResponseMeta: &metav1.ListResponseMeta{
			Page: req.Common.Page, ItemsPerPage: req.Common.ItemsPerPage,
			TotalCount: uint32(len(c.items)), HasMore: end < len(c.items),
		},
	}, nil
}

func TestCollectorExporterPagination(t *testing.T) {
	for _, tc := range []struct {
		name  string
		count int
	}{
		{name: "Empty"},
		{name: "ShortFirstPage", count: 1},
		{name: "FullFirstPage", count: 2},
		{name: "ShortFinalPage", count: 3},
		{name: "FullFinalPage", count: 4},
		{name: "ExistingExporter", count: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &collectorExporterPaginationClient{t: t, ctx: t.Context()}
			wanted := make(map[string]bool)
			for i := range tc.count {
				uid := fmt.Sprintf("exporter-%d", i)
				c.items = append(c.items, &enterprisev1.CollectorExporter{
					Metadata: &metav1.Metadata{Name: uid, Uid: uid},
				})
				if i < 4 {
					wanted[uid] = true
				}
			}

			checkCollectorExporterPagination(t, c.ctx, c, wanted)
			assert.Equal(t, uint32(max(1, (tc.count+1)/2)), c.calls)
		})
	}
}

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
