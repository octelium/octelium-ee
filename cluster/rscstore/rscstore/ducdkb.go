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
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
)

func (s *Server) insertResource(ctx context.Context, rsc umetav1.ResourceObjectI) error {
	s.setAuditLog(ctx, rsc)

	rscJSON, err := pbutils.MarshalJSON(rsc, false)
	if err != nil {
		return err
	}

	api, version := vutils.SplitApiVersion(rsc.GetApiVersion())
	kind := rsc.GetKind()
	uid := rsc.GetMetadata().Uid
	resourceVersion := rsc.GetMetadata().ResourceVersion

	rscStr, err := s.getRSCStr(rscJSON)
	if err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO resources VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (uid) DO UPDATE SET resource_version = EXCLUDED.resource_version, rsc = EXCLUDED.rsc, rsc_str = EXCLUDED.rsc_str`,
		api, version, kind, uid, resourceVersion, string(rscJSON), rscStr); err != nil {
		return err
	}

	go s.idxDebouncer.debounce()

	return nil
}

func (s *Server) removeResource(ctx context.Context, rsc umetav1.ResourceObjectI) error {

	s.setAuditLog(ctx, rsc)

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
