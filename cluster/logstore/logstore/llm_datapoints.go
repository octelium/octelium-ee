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
	"slices"
	"sort"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
)

const llmExprCreatedAtTimestamp = `CAST(json_extract(rsc, '$.metadata.createdAt') AS TIMESTAMP)`

func getLLMDataPointRange(f *vllmv1.Filter) (time.Time, time.Time) {
	var from, to time.Time

	if f.GetFrom() == nil && f.GetTo() == nil {
		from = time.Now().Add(-1 * time.Hour)
		to = from.Add(1 * time.Hour)
	}

	if f.GetFrom() != nil {
		from = f.GetFrom().AsTime()
		if f.GetTo() == nil {
			to = time.Now()
		}
	}
	if f.GetTo() != nil {
		to = f.GetTo().AsTime()
		if f.GetFrom() == nil {
			from = to.Add(-1 * time.Hour)
		}
	}

	return from.UTC(), to.UTC()
}

func llmDataPointBuckets(from, to time.Time, interval *intervalDataPoint) []time.Time {
	var ret []time.Time

	d := llmIntervalDuration(interval)
	if d <= 0 {
		return ret
	}

	current := from.Truncate(d)
	for current.Before(to) {
		ret = append(ret, current)
		current = current.Add(d)

		if len(ret) > 10000 {
			break
		}
	}

	return ret
}

func llmIntervalDuration(interval *intervalDataPoint) time.Duration {
	switch interval.Unit {
	case "second":
		return time.Duration(interval.Value) * time.Second
	case "minute":
		return time.Duration(interval.Value) * time.Minute
	case "hour":
		return time.Duration(interval.Value) * time.Hour
	case "day":
		return time.Duration(interval.Value) * 24 * time.Hour
	default:
		return 0
	}
}

type llmDataPointRow struct {
	timestamp time.Time
	key       string
	stats     *llmStats
}

type llmDataPointOpts struct {
	dim        *llmDimension
	filters    []exp.Expression
	interval   *intervalDataPoint
	quantiles  bool
	keys       []string
	exclude    bool
	groupByKey bool
}

func (s *Server) getLLMDataPointRows(ctx context.Context, o *llmDataPointOpts) ([]*llmDataPointRow, error) {

	bucket := goqu.L(fmt.Sprintf(`time_bucket(INTERVAL '%d %s', %s)`,
		o.interval.Value, o.interval.Unit, llmExprCreatedAtTimestamp)).As("bucket")

	var ds *goqu.SelectDataset

	switch {
	case o.dim == nil:
		ds = llmDialect().From(llmTable).Where(o.filters...).
			Select(append([]any{bucket}, llmStatsSelects(o.quantiles)...)...).
			GroupBy(goqu.L("bucket"))
	default:
		selects := []any{bucket}
		groupBy := []any{goqu.L("bucket")}

		if o.groupByKey {
			selects = append(selects, goqu.L("dim_key"))
			groupBy = append(groupBy, goqu.L("dim_key"))
		}

		ds = llmDialect().From(llmDimensionDataset(o.dim, o.filters)).
			Select(append(selects, llmStatsSelects(o.quantiles)...)...).
			Where(goqu.L(`dim_key IS NOT NULL AND dim_key <> ''`)).
			GroupBy(groupBy...)

		if len(o.keys) > 0 {
			if o.exclude {
				ds = ds.Where(goqu.L("dim_key").NotIn(o.keys))
			} else {
				ds = ds.Where(goqu.L("dim_key").In(o.keys))
			}
		}
	}

	ds = ds.Order(goqu.L("bucket").Asc())

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var ret []*llmDataPointRow

	for rows.Next() {
		item := &llmDataPointRow{
			stats: &llmStats{},
		}

		dest := []any{&item.timestamp}
		if o.dim != nil && o.groupByKey {
			dest = append(dest, &item.key)
		}
		dest = append(dest, item.stats.scanDest(o.quantiles)...)

		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		ret = append(ret, item)
	}

	return ret, rows.Err()
}

func llmDataPoints(buckets []time.Time, rows []*llmDataPointRow, key string) []*vllmv1.GetDataPointResponse_DataPoint {

	statsMap := make(map[time.Time]*llmStats)
	for _, row := range rows {
		if row.key != key {
			continue
		}
		statsMap[row.timestamp.UTC()] = row.stats
	}

	all := append([]time.Time{}, buckets...)
	for ts := range statsMap {
		if !slices.ContainsFunc(all, func(arg time.Time) bool {
			return arg.Equal(ts)
		}) {
			all = append(all, ts)
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Before(all[j])
	})

	var ret []*vllmv1.GetDataPointResponse_DataPoint
	for _, ts := range all {
		stats := newLLMEmptyStats()
		if cur, ok := statsMap[ts]; ok {
			stats = cur.toPB()
		}

		ret = append(ret, &vllmv1.GetDataPointResponse_DataPoint{
			Timestamp: pbutils.Timestamp(ts),
			Stats:     stats,
		})
	}

	return ret
}

func (s *Server) getLLMDataPoint(ctx context.Context,
	req *vllmv1.GetDataPointRequest) (*vllmv1.GetDataPointResponse, error) {

	filters, err := getLLMFilters(req.Filter)
	if err != nil {
		return nil, err
	}

	from, to := getLLMDataPointRange(req.Filter)
	if from.After(to) {
		return nil, grpcutils.InvalidArg("The `from` timestamp must be before the `to` timestamp")
	}

	filters = append(filters,
		goqu.L(fmt.Sprintf(`%s >= ?`, llmExprCreatedAtTimestamp), from),
		goqu.L(fmt.Sprintf(`%s < ?`, llmExprCreatedAtTimestamp), to))

	interval := s.getDataPointInterval(req.Interval)
	buckets := llmDataPointBuckets(from, to, interval)

	quantiles := req.IncludeQuantiles || llmMetricNeedsQuantiles(req.OrderBy)

	ret := &vllmv1.GetDataPointResponse{}

	if req.GroupBy == vllmv1.Dimension_DIMENSION_UNSET {
		rows, err := s.getLLMDataPointRows(ctx, &llmDataPointOpts{
			filters:   filters,
			interval:  interval,
			quantiles: quantiles,
		})
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		stats, err := s.getLLMStats(ctx, filters, quantiles)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		ret.Series = append(ret.Series, &vllmv1.GetDataPointResponse_Series{
			Datapoints: llmDataPoints(buckets, rows, ""),
			Stats:      stats.toPB(),
		})

		return ret, nil
	}

	dim, err := getLLMDimension(req.GroupBy)
	if err != nil {
		return nil, err
	}

	orderAlias, err := llmMetricAlias(req.OrderBy)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit == 0 {
		limit = llmDefaultSeriesLimit
	} else if limit > llmMaxSeriesLimit {
		return nil, grpcutils.InvalidArg("Limit is too high")
	}

	topRows, totalCount, err := s.getLLMDimensionRows(ctx, dim, filters, limit, orderAlias, quantiles)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret.TotalCount = totalCount

	var keys []string
	for _, row := range topRows {
		keys = append(keys, row.key)
	}

	if len(keys) < 1 {
		return ret, nil
	}

	rows, err := s.getLLMDataPointRows(ctx, &llmDataPointOpts{
		dim:        dim,
		filters:    filters,
		interval:   interval,
		quantiles:  quantiles,
		keys:       keys,
		groupByKey: true,
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	for _, topRow := range topRows {
		ret.Series = append(ret.Series, &vllmv1.GetDataPointResponse_Series{
			Key:        topRow.key,
			Ref:        dim.getRef(topRow.key, topRow.name),
			Datapoints: llmDataPoints(buckets, rows, topRow.key),
			Stats:      topRow.stats.toPB(),
		})
	}

	if totalCount <= uint64(len(keys)) {
		return ret, nil
	}

	otherRows, err := s.getLLMDataPointRows(ctx, &llmDataPointOpts{
		dim:       dim,
		filters:   filters,
		interval:  interval,
		quantiles: quantiles,
		keys:      keys,
		exclude:   true,
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	otherStats, err := s.getLLMDimensionOtherStats(ctx, dim, filters, keys, quantiles)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret.Other = &vllmv1.GetDataPointResponse_Series{
		Datapoints: llmDataPoints(buckets, otherRows, ""),
		Stats:      otherStats.toPB(),
	}

	return ret, nil
}
