// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package publicserver

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"

	"go.uber.org/zap"
)

//go:embed web
var fsWeb embed.FS

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {

	blob, err := fs.ReadFile(fsWeb, "web/index.html")
	if err != nil {
		zap.L().Debug("Could not read index.html file from web fs", zap.Error(err))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "octelium_domain",
		Value:    s.clusterDomain,
		Secure:   true,
		Domain:   s.clusterDomain,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.Write(blob)
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {

	blob, err := fs.ReadFile(fsWeb, filepath.Join("web", r.URL.Path))
	if err != nil {
		zap.L().Debug("Could not read blob from web fs", zap.Error(err))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(r.URL.Path)))

	w.Write(blob)
}
