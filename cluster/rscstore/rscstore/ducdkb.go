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

	_ "github.com/marcboeker/go-duckdb"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func (s *Server) insertResource(ctx context.Context, rsc umetav1.ResourceObjectI) error {
	s.setAuditLog(rsc)

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

	s.setAuditLog(rsc)

	if _, err := s.db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM resources WHERE uid = '%s'`,
			rsc.GetMetadata().Uid)); err != nil {
		return err
	}

	go s.idxDebouncer.debounce()

	return nil
}

func (s *Server) setAuditLog(rsc umetav1.ResourceObjectI) {
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
	s.auditLogItem <- rsc
}

func (s *Server) getRSCStr(rscJSON []byte) (string, error) {
	rscMap := make(map[string]any)
	if err := json.Unmarshal(rscJSON, &rscMap); err != nil {
		return "", err
	}
	metadata := rscMap["metadata"].(map[string]any)
	spec := rscMap["spec"].(map[string]any)

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	doTransform := func(arg []byte) string {

		return strings.ToLower(string(arg))
		/*
			return strings.Map(func(r rune) rune {
				if unicode.IsPrint(r) && (unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r)) {
					return r
				}

				return -1
			}, strings.ToLower(string(arg)))
		*/
	}

	return doTransform(metadataJSON) + " " + doTransform(specJSON), nil
}
