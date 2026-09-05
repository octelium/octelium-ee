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
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/pkg/errors"
)

const llmTable = "access_logs"

func llmDialect() goqu.DialectWrapper {
	return goqu.Dialect("postgres")
}

func (s *Server) getLLMStats(ctx context.Context,
	filters []exp.Expression, quantiles bool) (*llmStats, error) {

	ds := llmDialect().From(llmTable).Where(filters...).Select(llmStatsSelects(quantiles)...)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	ret := &llmStats{}
	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(ret.scanDest(quantiles)...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}
		return nil, err
	}

	return ret, nil
}

func llmDimensionDataset(dim *llmDimension, filters []exp.Expression) *goqu.SelectDataset {
	selects := []any{goqu.L(dim.key).As("dim_key")}
	if dim.name != "" {
		selects = append(selects, goqu.L(dim.name).As("dim_name"))
	}
	selects = append(selects, goqu.C("rsc"))

	return llmDialect().From(llmTable).Where(filters...).Select(selects...).As("llm_dim")
}

type llmDimensionRow struct {
	key   string
	name  string
	stats *llmStats
}

func (s *Server) getLLMDimensionRows(ctx context.Context, dim *llmDimension,
	filters []exp.Expression, limit uint32, orderAlias string, quantiles bool) ([]*llmDimensionRow, uint64, error) {

	selects := []any{goqu.L("dim_key")}
	if dim.name != "" {
		selects = append(selects, goqu.L(`ANY_VALUE(dim_name)`))
	}
	selects = append(selects, goqu.L(`COUNT(*) OVER ()`))
	selects = append(selects, llmStatsSelects(quantiles)...)

	ds := llmDialect().From(llmDimensionDataset(dim, filters)).
		Select(selects...).
		Where(goqu.L(`dim_key IS NOT NULL AND dim_key <> ''`)).
		GroupBy(goqu.L("dim_key")).
		Order(goqu.L(orderAlias).Desc()).
		Limit(uint(limit))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	defer rows.Close()

	var ret []*llmDimensionRow
	var totalCount int64

	for rows.Next() {
		item := &llmDimensionRow{
			stats: &llmStats{},
		}

		dest := []any{&item.key}
		if dim.name != "" {
			dest = append(dest, &item.name)
		}
		dest = append(dest, &totalCount)
		dest = append(dest, item.stats.scanDest(quantiles)...)

		if err := rows.Scan(dest...); err != nil {
			return nil, 0, err
		}

		ret = append(ret, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return ret, uint64(totalCount), nil
}

func (s *Server) getLLMDimensionOtherStats(ctx context.Context, dim *llmDimension,
	filters []exp.Expression, keys []string, quantiles bool) (*llmStats, error) {

	ds := llmDialect().From(llmDimensionDataset(dim, filters)).
		Select(llmStatsSelects(quantiles)...).
		Where(goqu.L(`dim_key IS NOT NULL AND dim_key <> ''`), goqu.L("dim_key").NotIn(keys))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	ret := &llmStats{}
	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(ret.scanDest(quantiles)...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}
		return nil, err
	}

	return ret, nil
}

func (s *Server) getLLMKeyCounts(ctx context.Context, filters []exp.Expression,
	keyExpr string, keys []string) (map[string]uint64, error) {

	ret := make(map[string]uint64)
	if len(keys) < 1 {
		return ret, nil
	}

	inner := llmDialect().From(llmTable).Where(filters...).
		Select(goqu.L(keyExpr).As("dim_key")).As("llm_keys")

	ds := llmDialect().From(inner).
		Select(goqu.L("dim_key"), goqu.L(`COUNT(*)`)).
		Where(goqu.L("dim_key").In(keys)).
		GroupBy(goqu.L("dim_key"))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		ret[key] = uint64(count)
	}

	return ret, rows.Err()
}

func (s *Server) getLLMDistinctListCount(ctx context.Context,
	filters []exp.Expression, listExpr string) (uint64, error) {

	inner := llmDialect().From(llmTable).Where(filters...).
		Select(goqu.L(fmt.Sprintf(`unnest(%s)`, listExpr)).As("dim_key")).As("llm_list")

	ds := llmDialect().From(inner).
		Select(goqu.L(`COUNT(DISTINCT dim_key)`)).
		Where(goqu.L(`dim_key IS NOT NULL AND dim_key <> ''`))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return 0, err
	}

	var ret int64
	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&ret); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}

	return uint64(ret), nil
}

func (s *Server) getLLMCardinality(ctx context.Context, filters []exp.Expression) (*vllmv1.Cardinality, error) {

	distinct := func(expr string) exp.LiteralExpression {
		return goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, expr))
	}

	ds := llmDialect().From(llmTable).Where(filters...).Select(
		distinct(llmExprModelEffective),
		distinct(llmExprModelRequested),
		distinct(llmJSONStr("entry.common.userRef.uid")),
		distinct(llmJSONStr("entry.common.sessionRef.uid")),
		distinct(llmJSONStr("entry.common.deviceRef.uid")),
		distinct(llmJSONStr("entry.common.serviceRef.uid")),
		distinct(llmJSONStr("entry.common.namespaceRef.uid")),
		distinct(llmExprProtocol),
		distinct(llmJSONStr("entry.common.reason.details.policyMatch.policy.policyRef.uid")),
	)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	var models, requestedModels, users, sessions, devices, services, namespaces, protocols, policies int64

	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(
		&models, &requestedModels, &users, &sessions, &devices,
		&services, &namespaces, &protocols, &policies); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	ret := &vllmv1.Cardinality{
		Models:          uint64(models),
		RequestedModels: uint64(requestedModels),
		Users:           uint64(users),
		Sessions:        uint64(sessions),
		Devices:         uint64(devices),
		Services:        uint64(services),
		Namespaces:      uint64(namespaces),
		Protocols:       uint64(protocols),
		Policies:        uint64(policies),
	}

	ret.Tools, err = s.getLLMDistinctListCount(ctx, filters, llmExprToolNames)
	if err != nil {
		return nil, err
	}

	ret.CalledTools, err = s.getLLMDistinctListCount(ctx, filters, llmExprCalledToolsL)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func getLLMTopLimit(limit uint32) (uint32, error) {
	switch {
	case limit == 0:
		return llmDefaultTopLimit, nil
	case limit > llmMaxTopLimit:
		return 0, grpcutils.InvalidArg("Limit is too high")
	default:
		return limit, nil
	}
}

func (s *Server) listLLMDimensionItems(ctx context.Context, dim vllmv1.Dimension,
	filters []exp.Expression, limit uint32, orderBy vllmv1.Metric,
	quantiles bool) ([]*vllmv1.DimensionItem, *vllmv1.Stats, uint64, error) {

	dimension, err := getLLMDimension(dim)
	if err != nil {
		return nil, nil, 0, err
	}

	orderAlias, err := llmMetricAlias(orderBy)
	if err != nil {
		return nil, nil, 0, err
	}

	quantiles = quantiles || llmMetricNeedsQuantiles(orderBy)

	rows, totalCount, err := s.getLLMDimensionRows(ctx, dimension, filters, limit, orderAlias, quantiles)
	if err != nil {
		return nil, nil, 0, grpcutils.InternalWithErr(err)
	}

	var items []*vllmv1.DimensionItem
	var keys []string

	for _, row := range rows {
		items = append(items, &vllmv1.DimensionItem{
			Key:   row.key,
			Ref:   dimension.getRef(row.key, row.name),
			Stats: row.stats.toPB(),
		})
		keys = append(keys, row.key)
	}

	var other *vllmv1.Stats
	if totalCount > uint64(len(items)) {
		otherStats, err := s.getLLMDimensionOtherStats(ctx, dimension, filters, keys, quantiles)
		if err != nil {
			return nil, nil, 0, grpcutils.InternalWithErr(err)
		}
		other = otherStats.toPB()
	}

	return items, other, totalCount, nil
}
