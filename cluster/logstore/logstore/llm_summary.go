// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"context"
	"database/sql"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
)

func (s *Server) getLLMSummary(ctx context.Context, req *vllmv1.GetSummaryRequest) (*vllmv1.GetSummaryResponse, error) {

	filters, err := getLLMAggregateFilters(req.Filter)
	if err != nil {
		return nil, err
	}

	if len(req.Breakdowns) > llmMaxBreakdowns {
		return nil, grpcutils.InvalidArg("Too many breakdowns")
	}

	stats, err := s.getLLMStats(ctx, filters, req.IncludeQuantiles)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	cardinalities, err := s.getLLMCardinalities(ctx, filters, req.Cardinalities)
	if err != nil {
		return nil, err
	}

	ret := &vllmv1.GetSummaryResponse{
		Stats:         stats.toPB(),
		Cardinalities: cardinalities,
	}

	seen := make(map[vllmv1.Dimension]struct{})

	for _, breakdown := range req.Breakdowns {
		if _, ok := seen[breakdown.GetDimension()]; ok {
			return nil, grpcutils.InvalidArg("Duplicate breakdown Dimension: %s", breakdown.GetDimension().String())
		}
		seen[breakdown.GetDimension()] = struct{}{}

		limit := breakdown.GetLimit()
		switch {
		case limit == 0:
			limit = llmDefaultBreakdownLimit
		case limit > llmMaxTopLimit:
			return nil, grpcutils.InvalidArg("Breakdown limit is too high")
		}

		items, other, totalCount, err := s.listLLMDimensionItems(ctx, breakdown.GetDimension(), filters,
			limit, breakdown.GetOrderBy(), req.IncludeQuantiles)
		if err != nil {
			return nil, err
		}

		ret.Breakdowns = append(ret.Breakdowns, &vllmv1.Breakdown{
			Dimension:  breakdown.GetDimension(),
			Items:      items,
			Other:      other,
			TotalCount: totalCount,
		})
	}

	return ret, nil
}

func llmListOrderExpr(orderBy *vllmv1.ListAccessLogRequest_OrderBy) (string, error) {
	switch orderBy.GetType() {
	case vllmv1.ListAccessLogRequest_OrderBy_TYPE_UNSET, vllmv1.ListAccessLogRequest_OrderBy_CREATED_AT:
		return llmExprCreatedAt, nil
	case vllmv1.ListAccessLogRequest_OrderBy_TOTAL_TOKENS:
		return llmExprTokensTotal, nil
	case vllmv1.ListAccessLogRequest_OrderBy_INPUT_TOKENS:
		return llmExprTokensInput, nil
	case vllmv1.ListAccessLogRequest_OrderBy_OUTPUT_TOKENS:
		return llmExprTokensOutput, nil
	case vllmv1.ListAccessLogRequest_OrderBy_LATENCY:
		return llmExprLatencyMs, nil
	case vllmv1.ListAccessLogRequest_OrderBy_TIME_TO_FIRST_TOKEN:
		return llmExprTimeToFirstToken, nil
	case vllmv1.ListAccessLogRequest_OrderBy_ESTIMATED_INPUT_TOKENS:
		return llmExprTokensEstimated, nil
	case vllmv1.ListAccessLogRequest_OrderBy_TOOL_CALLS:
		return llmExprToolCallCount, nil
	default:
		return "", grpcutils.InvalidArg("Invalid OrderBy type")
	}
}

func (s *Server) listLLMAccessLog(ctx context.Context, req *vllmv1.ListAccessLogRequest) (*vllmv1.ListAccessLogResponse, error) {

	ret := &vllmv1.ListAccessLogResponse{
		ListResponseMeta: &metav1.ListResponseMeta{},
	}

	filters, err := getLLMFilters(req.Filter)
	if err != nil {
		return nil, err
	}

	orderExpr, err := llmListOrderExpr(req.OrderBy)
	if err != nil {
		return nil, err
	}

	totalCount, err := s.countLogs(ctx, llmTable, filters)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ds := llmDialect().From(llmTable).Where(filters...).Select("rsc")

	listMeta := ret.ListResponseMeta
	listMeta.TotalCount = totalCount

	{
		limit := req.ItemsPerPage
		if req.Page > 100000 {
			return nil, grpcutils.InvalidArg("Page number is too high")
		}

		if limit == 0 {
			limit = defaultItemsPerPage
		} else if limit > maxItemsPerPage {
			limit = maxItemsPerPage
		}

		offset := req.Page * limit

		ds = ds.Offset(uint(offset)).Limit(uint(limit))

		listMeta.ItemsPerPage = limit
		listMeta.Page = req.Page
	}

	{
		isAsc := req.OrderBy.GetMode() == vllmv1.ListAccessLogRequest_OrderBy_ASC

		order := []exp.OrderedExpression{}
		if isAsc {
			order = append(order, goqu.L(orderExpr).Asc())
		} else {
			order = append(order, goqu.L(orderExpr).Desc())
		}

		if orderExpr != llmExprCreatedAt {
			if isAsc {
				order = append(order, goqu.L(llmExprCreatedAt).Asc())
			} else {
				order = append(order, goqu.L(llmExprCreatedAt).Desc())
			}
		}

		ds = ds.Order(order...)
	}

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	for rows.Next() {
		rsc := make(map[string]any)

		if err := rows.Scan(&rsc); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		accessLog := &corev1.AccessLog{}
		if err := pbutils.UnmarshalFromMap(rsc, accessLog); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.Items = append(ret.Items, accessLog)
	}

	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if listMeta.TotalCount > (listMeta.Page+1)*listMeta.ItemsPerPage {
		listMeta.HasMore = true
	}

	return ret, nil
}
