// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"context"
	"database/sql"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type numberRow struct {
	ts        time.Time
	labels    map[string]string
	fullKey   string
	groupKey  string
	intVal    *int64
	doubleVal *float64
}

type gaugeBucket struct {
	ts       time.Time
	count    int
	sum      float64
	min      float64
	max      float64
	last     float64
	lastTS   time.Time
	hasValue bool
}

func (s *srvMetric) resolveDescriptor(ctx context.Context, q *querySpec) (*vmetricsv1.MetricDescriptor, error) {
	row := s.s.db.QueryRowContext(ctx, `
SELECT name, kind, value_type, unit, description, temporality
FROM metrics
WHERE name = ?
ORDER BY timestamp DESC
LIMIT 1
`, q.name)

	var name, kind, valueType, unit, description, temporality string
	if err := row.Scan(&name, &kind, &valueType, &unit, &description, &temporality); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "metric not found: %s", q.name)
		}
		return nil, err
	}

	return &vmetricsv1.MetricDescriptor{
		Name:        name,
		Kind:        kindFromString(kind),
		ValueType:   valueTypeFromString(valueType),
		Unit:        unit,
		Description: description,
		Temporality: temporalityFromString(temporality),
	}, nil
}

func (s *srvMetric) queryGauge(
	ctx context.Context,
	q *querySpec,
	desc *vmetricsv1.MetricDescriptor,
) (*vmetricsv1.QueryMetricsResponse, error) {
	rows, err := s.loadNumberRows(ctx, q, desc, false)
	if err != nil {
		return nil, err
	}

	type fullSeries struct {
		groupKey    string
		groupLabels []*vmetricsv1.Attribute
		fullLabels  map[string]string
		buckets     map[int]*gaugeBucket
	}

	type groupedBucket struct {
		ts       time.Time
		count    int
		sum      float64
		min      float64
		max      float64
		hasValue bool
	}

	funcKind := q.req.Operation.GetGauge().Function

	byFullSeries := map[string]*fullSeries{}

	for _, r := range rows {
		if r.ts.Before(q.from) || !r.ts.Before(q.to) {
			continue
		}

		idx := int(r.ts.Sub(q.from) / q.step)
		if idx < 0 {
			continue
		}

		fs := byFullSeries[r.fullKey]
		if fs == nil {
			fs = &fullSeries{
				groupKey:    r.groupKey,
				groupLabels: labelsToProto(pickLabels(r.labels, q.groupBy)),
				fullLabels:  r.labels,
				buckets:     map[int]*gaugeBucket{},
			}
			byFullSeries[r.fullKey] = fs
		}

		val := r.numberValue()

		b := fs.buckets[idx]
		if b == nil {
			b = &gaugeBucket{
				ts:  q.from.Add(time.Duration(idx+1) * q.step),
				min: val,
				max: val,
			}
			fs.buckets[idx] = b
		}

		b.count++
		b.sum += val

		if val < b.min {
			b.min = val
		}
		if val > b.max {
			b.max = val
		}
		if !b.hasValue || r.ts.After(b.lastTS) {
			b.last = val
			b.lastTS = r.ts
			b.hasValue = true
		}
	}

	grouped := map[string]map[int]*groupedBucket{}
	labelsByGroup := map[string][]*vmetricsv1.Attribute{}

	for _, fs := range byFullSeries {
		if grouped[fs.groupKey] == nil {
			grouped[fs.groupKey] = map[int]*groupedBucket{}
			labelsByGroup[fs.groupKey] = fs.groupLabels
		}

		for idx, b := range fs.buckets {
			var val float64

			switch funcKind {
			case vmetricsv1.GaugeOperation_LAST:
				val = b.last
			case vmetricsv1.GaugeOperation_AVG:
				val = b.sum / float64(b.count)
			case vmetricsv1.GaugeOperation_MIN:
				val = b.min
			case vmetricsv1.GaugeOperation_MAX:
				val = b.max
			case vmetricsv1.GaugeOperation_SUM:
				val = b.sum
			default:
				return nil, status.Error(codes.InvalidArgument, "invalid gauge function")
			}

			gb := grouped[fs.groupKey][idx]
			if gb == nil {
				gb = &groupedBucket{
					ts:  b.ts,
					min: val,
					max: val,
				}
				grouped[fs.groupKey][idx] = gb
			}

			gb.count++
			gb.sum += val

			if val < gb.min {
				gb.min = val
			}
			if val > gb.max {
				gb.max = val
			}

			gb.hasValue = true
		}
	}

	out := make([]*vmetricsv1.TimeSeries, 0, len(grouped))

	for groupKey, buckets := range grouped {
		idxs := make([]int, 0, len(buckets))
		for idx := range buckets {
			idxs = append(idxs, idx)
		}
		sort.Ints(idxs)

		pts := &vmetricsv1.NumberPointSeries{}

		for _, idx := range idxs {
			b := buckets[idx]
			if !b.hasValue {
				continue
			}

			var val float64

			switch funcKind {
			case vmetricsv1.GaugeOperation_LAST:
				val = b.sum

			case vmetricsv1.GaugeOperation_AVG:
				val = b.sum / float64(b.count)

			case vmetricsv1.GaugeOperation_MIN:
				val = b.min

			case vmetricsv1.GaugeOperation_MAX:
				val = b.max

			case vmetricsv1.GaugeOperation_SUM:
				val = b.sum

			default:
				return nil, status.Error(codes.InvalidArgument, "invalid gauge function")
			}

			pts.Points = append(pts.Points, numberPointDouble(b.ts, val))
		}

		if len(pts.Points) == 0 {
			continue
		}

		out = append(out, &vmetricsv1.TimeSeries{
			Labels: labelsByGroup[groupKey],
			Points: &vmetricsv1.TimeSeries_Number{
				Number: pts,
			},
		})
	}

	out, total, truncated := limitAndSortSeries(out, q.limitSeries)
	return buildResponse(q, desc, out, total, truncated), nil
}

func (s *srvMetric) queryCounter(
	ctx context.Context,
	q *querySpec,
	desc *vmetricsv1.MetricDescriptor,
) (*vmetricsv1.QueryMetricsResponse, error) {
	includeLookback := q.req.Operation.GetCounter().Function != vmetricsv1.CounterOperation_RAW

	rows, err := s.loadNumberRows(ctx, q, desc, includeLookback)
	if err != nil {
		return nil, err
	}

	byFullSeries := map[string][]numberRow{}
	groupLabels := map[string][]*vmetricsv1.Attribute{}

	for _, r := range rows {
		byFullSeries[r.fullKey] = append(byFullSeries[r.fullKey], r)
		groupLabels[r.groupKey] = labelsToProto(pickLabels(r.labels, q.groupBy))
	}

	for k := range byFullSeries {
		sort.Slice(byFullSeries[k], func(i, j int) bool {
			return byFullSeries[k][i].ts.Before(byFullSeries[k][j].ts)
		})
	}

	fn := q.req.Operation.GetCounter().Function
	if fn == vmetricsv1.CounterOperation_RAW {
		return s.queryCounterRaw(q, desc, byFullSeries)
	}

	bucketsByGroup := map[string]map[int]float64{}

	for _, seriesRows := range byFullSeries {
		if len(seriesRows) == 0 {
			continue
		}

		for i := 0; i < len(seriesRows); i++ {
			cur := seriesRows[i]
			if cur.ts.Before(q.from) || !cur.ts.Before(q.to) {
				continue
			}

			idx := int(cur.ts.Sub(q.from) / q.step)
			if idx < 0 {
				continue
			}

			var inc float64

			if desc.Temporality == vmetricsv1.MetricDescriptor_DELTA {
				inc = cur.numberValue()
			} else {
				if i == 0 {
					continue
				}
				prev := seriesRows[i-1]
				inc = cur.numberValue() - prev.numberValue()
				if inc < 0 {
					inc = cur.numberValue()
				}
			}

			if inc < 0 {
				continue
			}

			if fn == vmetricsv1.CounterOperation_RATE {
				inc = inc / q.step.Seconds()
			}

			if _, ok := bucketsByGroup[cur.groupKey]; !ok {
				bucketsByGroup[cur.groupKey] = map[int]float64{}
			}
			bucketsByGroup[cur.groupKey][idx] += inc
		}
	}

	out := make([]*vmetricsv1.TimeSeries, 0, len(bucketsByGroup))
	for groupKey, buckets := range bucketsByGroup {
		idxs := make([]int, 0, len(buckets))
		for idx := range buckets {
			idxs = append(idxs, idx)
		}
		sort.Ints(idxs)

		pts := &vmetricsv1.NumberPointSeries{}
		for _, idx := range idxs {
			pts.Points = append(pts.Points, numberPointDouble(q.from.Add(time.Duration(idx+1)*q.step), buckets[idx]))
		}

		out = append(out, &vmetricsv1.TimeSeries{
			Labels: groupLabels[groupKey],
			Points: &vmetricsv1.TimeSeries_Number{
				Number: pts,
			},
		})
	}

	out, total, truncated := limitAndSortSeries(out, q.limitSeries)
	return buildResponse(q, desc, out, total, truncated), nil
}

func (s *srvMetric) queryCounterRaw(
	q *querySpec,
	desc *vmetricsv1.MetricDescriptor,
	byFullSeries map[string][]numberRow,
) (*vmetricsv1.QueryMetricsResponse, error) {
	out := make([]*vmetricsv1.TimeSeries, 0, len(byFullSeries))
	pointsTruncated := false

	for _, rows := range byFullSeries {
		if len(rows) == 0 {
			continue
		}

		filtered := make([]numberRow, 0, len(rows))
		for _, r := range rows {
			if r.ts.Before(q.from) || !r.ts.Before(q.to) {
				continue
			}
			filtered = append(filtered, r)
		}

		if len(filtered) == 0 {
			continue
		}

		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].ts.Before(filtered[j].ts)
		})

		if len(filtered) > q.limitPointsPerSeries {
			pointsTruncated = true
			filtered = filtered[len(filtered)-q.limitPointsPerSeries:]
		}

		pts := &vmetricsv1.NumberPointSeries{}

		for _, r := range filtered {
			if r.intVal != nil {
				pts.Points = append(pts.Points, numberPointInt(r.ts, *r.intVal))
			} else {
				pts.Points = append(pts.Points, numberPointDouble(r.ts, r.numberValue()))
			}
		}

		out = append(out, &vmetricsv1.TimeSeries{
			Labels: labelsToProto(filtered[0].labels),
			Points: &vmetricsv1.TimeSeries_Number{
				Number: pts,
			},
		})
	}

	out, total, seriesTruncated := limitAndSortSeries(out, q.limitSeries)
	return buildResponse(q, desc, out, total, seriesTruncated || pointsTruncated), nil
}

func (s *srvMetric) loadNumberRows(
	ctx context.Context,
	q *querySpec,
	desc *vmetricsv1.MetricDescriptor,
	includeLookback bool,
) ([]numberRow, error) {
	from := q.from
	if includeLookback {
		from = from.Add(-q.step)
	}

	where := []string{
		"name = ?",
		"timestamp >= ?",
		"timestamp < ?",
		"kind = ?",
	}
	args := []any{q.name, from, q.to, kindToString(desc.Kind)}

	if desc.Temporality != vmetricsv1.MetricDescriptor_TEMPORALITY_UNSET {
		where = append(where, "temporality = ?")
		args = append(args, temporalityEnumToString(desc.Temporality))
	}

	appendFilterSQL(&where, &args, q.filters)

	query := `
SELECT timestamp, CAST(attributes AS VARCHAR), number_int, number_double
FROM metrics
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY timestamp ASC
`

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ret []numberRow

	for rows.Next() {
		var ts time.Time
		var attrsJSON string
		var intNull sql.NullInt64
		var doubleNull sql.NullFloat64

		if err := rows.Scan(&ts, &attrsJSON, &intNull, &doubleNull); err != nil {
			return nil, err
		}

		attrs, err := decodeStringMap(attrsJSON)
		if err != nil {
			return nil, err
		}

		labels := anyMapToStringMap(attrs)
		fullKey := labelsKey(labels, sortedKeys(labels))
		groupKey := labelsKey(labels, q.groupBy)

		r := numberRow{
			ts:       ts.UTC(),
			labels:   labels,
			fullKey:  fullKey,
			groupKey: groupKey,
		}

		if intNull.Valid {
			v := intNull.Int64
			r.intVal = &v
		}
		if doubleNull.Valid {
			v := doubleNull.Float64
			r.doubleVal = &v
		}

		ret = append(ret, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r numberRow) numberValue() float64 {
	if r.doubleVal != nil {
		return *r.doubleVal
	}
	if r.intVal != nil {
		return float64(*r.intVal)
	}
	return 0
}

func numberPointDouble(ts time.Time, val float64) *vmetricsv1.NumberPoint {
	return &vmetricsv1.NumberPoint{
		Timestamp: pbutils.Timestamp(ts),
		Value: &vmetricsv1.NumberPoint_AsDouble{
			AsDouble: val,
		},
	}
}

func numberPointInt(ts time.Time, val int64) *vmetricsv1.NumberPoint {
	return &vmetricsv1.NumberPoint{
		Timestamp: pbutils.Timestamp(ts),
		Value: &vmetricsv1.NumberPoint_AsInt{
			AsInt: val,
		},
	}
}

func limitAndSortSeries(in []*vmetricsv1.TimeSeries, limit int) ([]*vmetricsv1.TimeSeries, uint32, bool) {
	sort.Slice(in, func(i, j int) bool {
		return labelsKeyFromProto(in[i].Labels) < labelsKeyFromProto(in[j].Labels)
	})

	total := uint32(len(in))
	if len(in) > limit {
		return in[:limit], total, true
	}

	return in, total, false
}

func appendFilterSQL(where *[]string, args *[]any, filters map[string]attributeFilter) {
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		f := filters[key]
		path := jsonPathForKey(key)

		switch f.op {
		case vmetricsv1.AttributeFilter_EQ:
			*where = append(*where, "json_extract_string(attributes, ?) = ?")
			*args = append(*args, path, f.value)

		case vmetricsv1.AttributeFilter_NOT_EQ:
			*where = append(*where, "(json_extract_string(attributes, ?) IS NULL OR json_extract_string(attributes, ?) != ?)")
			*args = append(*args, path, path, f.value)

		case vmetricsv1.AttributeFilter_EXISTS:
			*where = append(*where, "json_extract_string(attributes, ?) IS NOT NULL")
			*args = append(*args, path)

		case vmetricsv1.AttributeFilter_NOT_EXISTS:
			*where = append(*where, "json_extract_string(attributes, ?) IS NULL")
			*args = append(*args, path)

		case vmetricsv1.AttributeFilter_IN:
			var placeholders []string
			*args = append(*args, path)
			for _, v := range f.values {
				placeholders = append(placeholders, "?")
				*args = append(*args, v)
			}
			*where = append(*where, "json_extract_string(attributes, ?) IN ("+strings.Join(placeholders, ",")+")")
		}
	}
}

func jsonPathForKey(key string) string {
	return `$."` + strings.ReplaceAll(key, `"`, `\"`) + `"`
}

func histogramQuantile(q float64, buckets []*vmetricsv1.HistogramBucket, count uint64) float64 {
	if count == 0 || len(buckets) == 0 {
		return 0
	}

	target := q * float64(count)

	var prevCount uint64
	var prevLe float64

	for _, b := range buckets {
		c := b.Count
		if float64(c) >= target {
			if b.IsInf {
				return prevLe
			}

			bucketCount := c - prevCount
			if bucketCount == 0 {
				return b.Le
			}

			pos := (target - float64(prevCount)) / float64(bucketCount)
			return prevLe + pos*(b.Le-prevLe)
		}

		prevCount = c
		if !b.IsInf {
			prevLe = b.Le
		}
	}

	return math.NaN()
}
