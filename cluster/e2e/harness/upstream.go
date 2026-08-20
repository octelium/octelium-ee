// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
)

type NodeUpstream struct {
	URL string

	bearer   atomic.Value
	requests atomic.Int64
	srv      *http.Server
	lis      net.Listener
}

func (u *NodeUpstream) Requests() int64 { return u.requests.Load() }

func (u *NodeUpstream) SetBearer(bearer string) { u.bearer.Store(bearer) }

func (u *NodeUpstream) bearerValue() string {
	val, _ := u.bearer.Load().(string)
	return val
}

func (u *NodeUpstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u.requests.Add(1)

	bearer := u.bearerValue()
	if bearer != "" {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got != bearer {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"hello":"world"}`))
}

func (h *H) StartNodeUpstream(t *testing.T, bearer string) *NodeUpstream {
	t.Helper()

	port := h.Port()

	ret := &NodeUpstream{URL: fmt.Sprintf("http://%s:%d", h.ExternalIP, port)}
	ret.SetBearer(bearer)

	lis, err := listenAllInterfaces(port)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	ret.lis = lis
	ret.srv = &http.Server{Handler: ret}

	go ret.srv.Serve(lis)

	t.Cleanup(func() {
		ret.srv.Close()
		ret.lis.Close()
	})

	return ret
}

func listenAllInterfaces(port int) (net.Listener, error) {
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	deadline := time.Now().Add(30 * time.Second)

	var lastErr error
	for time.Now().Before(deadline) {
		lis, err := net.Listen("tcp", addr)
		if err == nil {
			return lis, nil
		}

		lastErr = err
		time.Sleep(time.Second)
	}

	return nil, errors.Errorf("Could not listen on %s: %+v", addr, lastErr)
}

func (u *NodeUpstream) Reachable(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.URL, nil)
	if err != nil {
		return err
	}

	if bearer := u.bearerValue(); bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.Errorf("the node upstream answered %d", res.StatusCode)
	}

	return nil
}
