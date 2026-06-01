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
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/common/rgx"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type doListReq struct {
	filters []exp.Expression
	api     string
	version string
	kind    string
	common  *vmetav1.CommonListOptions
}

const defaultItemsPerPage = 10
const maxItemsPerPage = 1000

func (s *Server) doList(ctx context.Context, req *doListReq) (proto.Message, error) {
	var filters []exp.Expression

	var items []umetav1.ResourceObjectI
	listMeta := &metav1.ListResponseMeta{}
	if req.common == nil {
		req.common = &vmetav1.CommonListOptions{}
	}

	query := strings.TrimSpace(req.common.Query)
	hasQuery := query != ""
	if len(query) > 100 {
		return nil, grpcutils.InvalidArg("Query is too long")
	}

	useFTS := hasQuery && isSimpleFTSQuery(query)

	zap.L().Debug("New list req",
		zap.String("api", req.api), zap.String("kind", req.kind), zap.Any("req", req.common))

	filters = append(filters, goqu.L(`api`).Eq(req.api))
	filters = append(filters, goqu.L(`version`).Eq(req.version))
	filters = append(filters, goqu.L(`kind`).Eq(req.kind))
	filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())

	if req.common.From != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).
			Gte(req.common.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if req.common.To != nil {
		filters = append(filters, goqu.L(`rsc->>'$.metadata.createdAt'`).
			Lte(req.common.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	selects := []any{
		goqu.L(`COUNT(*) OVER() as count`),
		goqu.L(`rsc`),
	}

	if hasQuery {
		queryLike := "%" + query + "%"
		queryPrefixStart := query + "%"
		queryPrefixWord := "% " + query + "%"

		if useFTS {
			filters = append(filters,
				goqu.Or(
					goqu.L(`fts_main_resources.match_bm25(uid, ?) IS NOT NULL`, query),
					goqu.L(`rsc_str ILIKE ?`, queryLike),
				),
			)

			selects = append(selects,
				goqu.L(`fts_main_resources.match_bm25(uid, ?)`, query).As("score"),
				goqu.L(`
					CASE
						WHEN rsc_str ILIKE ? THEN 4
						WHEN rsc_str ILIKE ? OR rsc_str ILIKE ? THEN 3
						WHEN rsc_str ILIKE ? THEN 2
						WHEN fts_main_resources.match_bm25(uid, ?) IS NOT NULL THEN 1
						ELSE 0
					END
				`, query, queryPrefixStart, queryPrefixWord, queryLike, query).As("text_rank"),
			)
		} else {
			filters = append(filters,
				goqu.L(`rsc_str ILIKE ?`, queryLike),
			)

			selects = append(selects,
				goqu.L(`CAST(NULL AS DOUBLE)`).As("score"),
				goqu.L(`
					CASE
						WHEN rsc_str ILIKE ? THEN 4
						WHEN rsc_str ILIKE ? OR rsc_str ILIKE ? THEN 3
						WHEN rsc_str ILIKE ? THEN 2
						ELSE 0
					END
				`, query, queryPrefixStart, queryPrefixWord, queryLike).As("text_rank"),
			)
		}
	}

	filters = append(filters, req.filters...)

	if req.common.Tag != "" {
		if !rgx.NameMain.MatchString(req.common.Tag) {
			return nil, grpcutils.InvalidArg("Invalid tag: %s", req.common.Tag)
		}

		filters = append(filters,
			goqu.L(
				`list_contains(CAST(json_extract(rsc, '$.metadata.tags') AS VARCHAR[]), ?)`,
				req.common.Tag,
			),
		)
	}

	ds := goqu.From("resources").
		Prepared(true).
		Where(filters...).
		Select(selects...)

	limit := req.common.ItemsPerPage
	if req.common.Page > 10000 {
		return nil, grpcutils.InvalidArgWithErr(errors.Errorf("Page number is too high"))
	}

	if limit == 0 {
		limit = defaultItemsPerPage
	} else if limit > maxItemsPerPage {
		limit = maxItemsPerPage
	}

	offset := req.common.Page * limit

	ds = ds.Offset(uint(offset)).Limit(uint(limit))

	listMeta.ItemsPerPage = limit
	listMeta.Page = req.common.Page

	if hasQuery {
		ds = ds.OrderAppend(
			goqu.L(`text_rank`).Desc(),
			goqu.L(`score`).Desc().NullsLast(),
		)
	}

	if req.common.OrderBy != nil {
		switch req.common.OrderBy.Type {
		case vmetav1.CommonListOptions_OrderBy_CREATED_AT:
			if req.common.OrderBy.Mode == vmetav1.CommonListOptions_OrderBy_DESC {
				ds = ds.OrderAppend(goqu.L(`rsc->'metadata'->>'createdAt'`).Desc())
			} else {
				ds = ds.OrderAppend(goqu.L(`rsc->'metadata'->>'createdAt'`).Asc())
			}
		case vmetav1.CommonListOptions_OrderBy_NAME:
			if req.common.OrderBy.Mode == vmetav1.CommonListOptions_OrderBy_DESC {
				ds = ds.OrderAppend(goqu.L(`rsc->'metadata'->>'name'`).Desc())
			} else {
				ds = ds.OrderAppend(goqu.L(`rsc->'metadata'->>'name'`).Asc())
			}
		default:
			ds = ds.OrderAppend(goqu.L(`rsc->'metadata'->>'createdAt'`).Desc())
		}
	} else {
		ds = ds.OrderAppend(goqu.L(`rsc->'metadata'->>'createdAt'`).Desc())
	}

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		rscMap := make(map[string]any)
		var count int

		if hasQuery {
			var score sql.NullFloat64
			var textRank int

			if err := rows.Scan(&count, &rscMap, &score, &textRank); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&count, &rscMap); err != nil {
				return nil, err
			}
		}

		listMeta.TotalCount = uint32(count)

		rsc, err := ovutils.NewResourceObject(req.api, req.version, req.kind)
		if err != nil {
			return nil, err
		}

		if err := pbutils.UnmarshalFromMap(rscMap, rsc); err != nil {
			return nil, err
		}

		items = append(items, rsc)
	}

	if len(items) == 0 && listMeta.Page > 0 {
		return nil, grpcutils.NotFound("Not Items found for that page")
	}

	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if listMeta.TotalCount > (listMeta.Page+1)*listMeta.ItemsPerPage {
		listMeta.HasMore = true
	}

	return s.toResourceList(items, listMeta, req.api, req.version, req.kind)
}

func isSimpleFTSQuery(q string) bool {
	if q == "" || len(q) > 100 {
		return false
	}

	for _, r := range q {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == ' ' || r == '-' || r == '_' || r == '.' || r == '@' || r == ':':
		default:
			return false
		}
	}

	return true
}

func (s *Server) toResourceList(lst []umetav1.ResourceObjectI, listMeta *metav1.ListResponseMeta, api, version, kind string) (proto.Message, error) {
	retMap := map[string]any{
		"apiVersion": vutils.GetApiVersion(api, version),
		"kind":       fmt.Sprintf("%sList", kind),
		"items":      []map[string]any{},
	}

	if listMeta != nil {
		retMap["listResponseMeta"] = map[string]any{
			"page":         listMeta.Page,
			"hasMore":      listMeta.HasMore,
			"totalCount":   listMeta.TotalCount,
			"itemsPerPage": listMeta.ItemsPerPage,
		}
	}

	itemsMap := []map[string]any{}
	for _, itm := range lst {
		objMap, err := pbutils.ConvertToMap(itm)
		if err != nil {
			return nil, err
		}
		itemsMap = append(itemsMap, objMap)
	}

	retMap["items"] = itemsMap
	jsonBytes, err := json.Marshal(retMap)
	if err != nil {
		return nil, err
	}

	objList, err := ovutils.NewResourceObjectList(api, version, kind)
	if err != nil {
		return nil, err
	}

	if err := pbutils.UnmarshalJSON(jsonBytes, objList); err != nil {
		return nil, err
	}

	return objList, nil
}
