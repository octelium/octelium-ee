// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package publicserver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestRenderIndex(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C
	clusterCfg, err := tst.C.OcteliumC.CoreV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err)

	srv, err := initServer(ctx, fakeC.OcteliumC, clusterCfg)
	assert.Nil(t, err)

	srv.genCache.Set("authserver-app-js-hash", "xxx", cache.NoExpiration)

	req := httptest.NewRequest("GET", "http://localhost/", nil)
	w := httptest.NewRecorder()
	srv.handleIndex(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	assert.Nil(t, err)

	assert.True(t, strings.Contains(resp.Header.Get("Content-Security-Policy"), "frame-src"))

	assert.Equal(t, resp.StatusCode, http.StatusOK)
}
