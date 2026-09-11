// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package publicserver

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accessintg"
	"github.com/octelium/octelium-ee/cluster/common/accessintg/ingress"
	"go.uber.org/zap"
)

const integrationPathPrefix = "/integration/v1/callback/"

const maxIntegrationBodyBytes = 1 << 20

func (s *Server) handleIntegration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	segments := strings.Split(strings.Trim(
		strings.TrimPrefix(r.URL.Path, integrationPathPrefix), "/"), "/")

	if len(segments) == 0 || segments[0] == "" {
		http.NotFound(w, r)
		return
	}

	integrationID := segments[0]
	path := []string{}
	for _, segment := range segments[1:] {
		if segment != "" {
			path = append(path, segment)
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxIntegrationBodyBytes+1))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(body) > maxIntegrationBodyBytes {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	resp, err := s.integrationSrv.Handle(r.Context(), integrationID, &accessintg.InboundRequest{
		Method: r.Method,
		Path:   path,
		Header: r.Header,
		Body:   body,
		Now:    time.Now(),
	})
	if err != nil {
		zap.L().Debug("Could not handle an inbound integration request",
			zap.String("integrationID", integrationID), zap.Error(err))

		if ingress.IsAuthError(err) {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	if resp == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	if resp.ContentType != "" {
		w.Header().Set("Content-Type", resp.ContentType)
	}

	statusCode := resp.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	w.WriteHeader(statusCode)

	if len(resp.Body) > 0 {
		if _, err := w.Write(resp.Body); err != nil {
			zap.L().Debug("Could not write the inbound integration response", zap.Error(err))
		}
	}
}
