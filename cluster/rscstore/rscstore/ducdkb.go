// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"go.uber.org/zap"
)

func (s *Server) insertResource(ctx context.Context, rsc umetav1.ResourceObjectI) error {
	s.setAuditLog(ctx, rsc)

	if err := s.upsertResource(ctx, rsc); err != nil {
		return err
	}

	go s.idxDebouncer.debounce()

	return nil
}

func (s *Server) removeResource(ctx context.Context, rsc umetav1.ResourceObjectI) error {
	s.setAuditLog(ctx, rsc)

	if rsc == nil || rsc.GetMetadata() == nil || rsc.GetMetadata().GetUid() == "" {
		return nil
	}

	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM resources WHERE uid = ?`,
		rsc.GetMetadata().GetUid(),
	); err != nil {
		return err
	}

	go s.idxDebouncer.debounce()

	return nil
}

func (s *Server) setAuditLog(ctx context.Context, rsc umetav1.ResourceObjectI) {
	if rsc == nil {
		return
	}
	md := rsc.GetMetadata()
	if md == nil {
		return
	}

	if md.ActorRef == nil {
		return
	}

	select {
	case s.auditLogItem <- rsc:
	case <-ctx.Done():
		return
	case <-time.After(3 * time.Second):
		zap.L().Error("Dropping audit log",
			zap.String("uid", rsc.GetMetadata().GetUid()),
			zap.String("kind", rsc.GetKind()),
		)
	}
}

func (s *Server) getRSCStr(rscJSON []byte) (string, error) {
	rscMap := make(map[string]any)
	if err := json.Unmarshal(rscJSON, &rscMap); err != nil {
		return "", err
	}

	var parts []string

	var add func(any)
	add = func(v any) {
		switch t := v.(type) {
		case nil:
			return
		case string:
			if t != "" {
				parts = append(parts, t)
			}
		case []any:
			for _, x := range t {
				add(x)
			}
		case map[string]any:
			for _, x := range t {
				add(x)
			}
		case bool:
			if t {
				parts = append(parts, "true")
			} else {
				parts = append(parts, "false")
			}
		case float64:
			parts = append(parts, fmt.Sprintf("%v", t))
		default:
			parts = append(parts, fmt.Sprintf("%v", t))
		}
	}

	add(rscMap["metadata"])
	add(rscMap["spec"])
	add(rscMap["status"])

	return strings.ToLower(strings.Join(parts, " ")), nil
}
