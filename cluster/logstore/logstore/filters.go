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
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultTopItems = 10
const maxTopItems = 100

const defaultDataPointSeries = 8
const maxDataPointSeries = 25

const jsonIsPublic = `json_extract_string(rsc, '$.entry.common.isPublic')`
const jsonIsAnonymous = `json_extract_string(rsc, '$.entry.common.isAnonymous')`
const jsonMode = `json_extract_string(rsc, '$.entry.common.mode')`
const jsonDenyReason = `json_extract_string(rsc, '$.entry.common.reason.type')`
const jsonComponentType = `json_extract_string(rsc, '$.entry.component.type')`
const jsonComponentNamespace = `json_extract_string(rsc, '$.entry.component.namespace')`
const jsonComponentUID = `json_extract_string(rsc, '$.entry.component.uid')`
const jsonAuditMethod = `json_extract_string(rsc, '$.entry.method')`
const jsonAuditResourceKind = `json_extract_string(rsc, '$.entry.resourceRef.kind')`

func getTopLimit(limit uint32) int {
	switch {
	case limit == 0:
		return defaultTopItems
	case limit > maxTopItems:
		return maxTopItems
	default:
		return int(limit)
	}
}

func getSeriesLimit(limit uint32) int {
	switch {
	case limit == 0:
		return defaultDataPointSeries
	case limit > maxDataPointSeries:
		return maxDataPointSeries
	default:
		return int(limit)
	}
}

func appendAccessFlagFilters(filters []exp.Expression, isPublic, isAnonymous bool) []exp.Expression {
	if isPublic {
		filters = append(filters, goqu.L(jsonIsPublic).Eq("true"))
	}
	if isAnonymous {
		filters = append(filters, goqu.L(jsonIsAnonymous).Eq("true"))
	}

	return filters
}

func appendComponentFilter(filters []exp.Expression,
	component *visibilityv1.ComponentSelector) ([]exp.Expression, error) {

	if component == nil {
		return filters, nil
	}

	for _, arg := range []struct {
		val  string
		expr string
	}{
		{val: component.Type, expr: jsonComponentType},
		{val: component.Namespace, expr: jsonComponentNamespace},
		{val: component.Uid, expr: jsonComponentUID},
	} {
		if arg.val == "" {
			continue
		}
		if len(arg.val) > 120 {
			return nil, grpcutils.InvalidArg("Invalid component selector: %s", arg.val)
		}

		filters = append(filters, goqu.L(arg.expr).Eq(arg.val))
	}

	return filters, nil
}

func setLogSummaryPrevious(cur, prev proto.Message) {
	msg := cur.ProtoReflect()
	fd := msg.Descriptor().Fields().ByName("previous")
	if fd == nil {
		return
	}

	msg.Set(fd, protoreflect.ValueOfMessage(prev.ProtoReflect()))
}

func withLogSummaryComparison[T proto.Message](ctx context.Context,
	from, to, compareFrom, compareTo *timestamppb.Timestamp,
	fn func(context.Context, *timestamppb.Timestamp, *timestamppb.Timestamp) (T, error)) (T, error) {

	ret, err := fn(ctx, from, to)
	if err != nil {
		return ret, err
	}

	if compareFrom == nil && compareTo == nil {
		return ret, nil
	}

	prev, err := fn(ctx, compareFrom, compareTo)
	if err != nil {
		return ret, err
	}

	setLogSummaryPrevious(ret, prev)

	return ret, nil
}

func (s *Server) getGroupedCounts(ctx context.Context,
	table, expr string, filters []exp.Expression) (map[string]uint64, error) {

	ds := goqu.From(table).Where(filters...).
		Where(goqu.L(fmt.Sprintf(`%s IS NOT NULL`, expr))).
		Select(
			goqu.L(expr).As("key"),
			goqu.L(`COUNT(*)`).As("count"),
		).GroupBy(goqu.L(expr))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := make(map[string]uint64)
	for rows.Next() {
		var key string
		var count uint64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		if key == "" {
			continue
		}

		ret[key] = count
	}

	return ret, rows.Err()
}
