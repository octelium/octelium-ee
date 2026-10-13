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
	"testing/fstest"

	"github.com/PuerkitoBio/goquery"
	"github.com/octelium/octelium/cluster/common/tests"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestSetDocumentNonce(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<!DOCTYPE html>
<html><head>
<script>document.documentElement.dataset.mantineColorScheme = "dark"</script>
<script nonce="old">window.__OCTELIUM_GLOBALS__ = {}</script>
<link rel="modulepreload" href="/assets/vendor.js">
<link rel="stylesheet" href="/assets/index.css">
</head><body><script type="module" src="/assets/index.js"></script></body></html>`))
	assert.Nil(t, err)

	setDocumentNonce(doc, "test-nonce")

	assert.Equal(t, 3, doc.Find("script[nonce='test-nonce']").Length())
	assert.Equal(t, 1, doc.Find("link[rel='modulepreload'][nonce='test-nonce']").Length())
	assert.Equal(t, 0, doc.Find("link[rel='stylesheet'][nonce]").Length())
}

func TestStaticAssetCache(t *testing.T) {
	handler := newStaticHandler(fstest.MapFS{
		"assets/index-Abcd1234.js": {Data: []byte("export default 1;")},
		"assets/index.js":          {Data: []byte("export default 2;")},
	})

	for _, tt := range []struct {
		name         string
		method       string
		path         string
		rangeHeader  string
		status       int
		cacheControl string
	}{
		{"hashed asset", "GET", "/assets/index-Abcd1234.js", "", http.StatusOK, "public, max-age=31536000, immutable"},
		{"hashed asset head", "HEAD", "/assets/index-Abcd1234.js", "", http.StatusOK, "public, max-age=31536000, immutable"},
		{"hashed asset range", "GET", "/assets/index-Abcd1234.js", "bytes=0-5", http.StatusPartialContent, "public, max-age=31536000, immutable"},
		{"invalid range", "GET", "/assets/index-Abcd1234.js", "bytes=100-200", http.StatusRequestedRangeNotSatisfiable, "no-store"},
		{"missing asset", "GET", "/assets/missing-Abcd1234.js", "", http.StatusNotFound, "no-store"},
		{"unhashed asset", "GET", "/assets/index.js", "", http.StatusOK, "no-store"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Range", tt.rangeHeader)
			handler.ServeHTTP(w, req)
			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.cacheControl, w.Header().Get("Cache-Control"))
			if tt.method == "HEAD" {
				assert.Empty(t, w.Body.String())
			}
		})
	}
}

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
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"),
		"connect-src 'self' https://*.")

	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
}
