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
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type explicitHistogramRow struct {
	ts           time.Time
	labels       map[string]string
	fullKey      string
	groupKey     string
	sum          float64
	hasSum       bool
	count        uint64
	min          *float64
	max          *float64
	bounds       []float64
	bucketCounts []uint64
}

type explicitHistogramAgg struct {
	ts           time.Time
	sum          float64
	hasSum       bool
	count        uint64
	bounds       []float64
	bucketCounts []uint64
}

func (s *srvMetric) queryExplicitHistogram(
	ctx context.Context,
	q *querySpec,
	desc *vmetricsv1.MetricDescriptor,
) (*vmetricsv1.QueryMetricsResponse, error) {
	rows, err := s.loadExplicitHistogramRows(ctx, q, desc, desc.Temporality == vmetricsv1.MetricDescriptor_CUMULATIVE)
	if err != nil {
		return nil, err
	}

	byFullSeries := map[string][]explicitHistogramRow{}
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

	aggs := map[string]map[int]*explicitHistogramAgg{}

	for _, seriesRows := range byFullSeries {
		for i := 0; i < len(seriesRows); i++ {
			cur := seriesRows[i]
			if cur.ts.Before(q.from) || !cur.ts.Before(q.to) {
				continue
			}

			idx := int(cur.ts.Sub(q.from) / q.step)
			if idx < 0 {
				continue
			}

			delta := cur
			if desc.Temporality == vmetricsv1.MetricDescriptor_CUMULATIVE {
				if i == 0 {
					continue
				}
				var err error
				delta, err = deltaExplicitHistogram(seriesRows[i-1], cur)
				if err != nil {
					return nil, err
				}
			}

			if _, ok := aggs[cur.groupKey]; !ok {
				aggs[cur.groupKey] = map[int]*explicitHistogramAgg{}
			}

			a := aggs[cur.groupKey][idx]
			if a == nil {
				a = &explicitHistogramAgg{
					ts:           q.from.Add(time.Duration(idx+1) * q.step),
					bounds:       append([]float64(nil), delta.bounds...),
					bucketCounts: make([]uint64, len(delta.bucketCounts)),
				}
				aggs[cur.groupKey][idx] = a
			}

			if !sameFloatSlice(a.bounds, delta.bounds) {
				return nil, status.Error(codes.InvalidArgument, "cannot merge histograms with incompatible explicit bounds")
			}

			if delta.hasSum {
				a.sum += delta.sum
				a.hasSum = true
			}

			a.count += delta.count
			for j := range delta.bucketCounts {
				a.bucketCounts[j] += delta.bucketCounts[j]
			}
		}
	}

	op := q.req.Operation.GetHistogram()
	switch op.Function {
	case vmetricsv1.HistogramOperation_BUCKETS:
		return s.explicitHistogramBucketsResponse(q, desc, aggs, groupLabels)

	case vmetricsv1.HistogramOperation_AVG,
		vmetricsv1.HistogramOperation_COUNT,
		vmetricsv1.HistogramOperation_SUM,
		vmetricsv1.HistogramOperation_QUANTILE:
		return s.explicitHistogramNumberResponse(q, desc, aggs, groupLabels)

	case vmetricsv1.HistogramOperation_MIN,
		vmetricsv1.HistogramOperation_MAX:
		return nil, status.Error(codes.InvalidArgument, "min/max are not supported for windowed histogram queries")

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid histogram function")
	}
}

func (s *srvMetric) explicitHistogramBucketsResponse(
	q *querySpec,
	desc *vmetricsv1.MetricDescriptor,
	aggs map[string]map[int]*explicitHistogramAgg,
	groupLabels map[string][]*vmetricsv1.Attribute,
) (*vmetricsv1.QueryMetricsResponse, error) {
	var out []*vmetricsv1.TimeSeries

	for groupKey, buckets := range aggs {
		idxs := sortedIntKeys(buckets)

		pts := &vmetricsv1.HistogramPointSeries{}
		for _, idx := range idxs {
			a := buckets[idx]

			pts.Points = append(pts.Points, &vmetricsv1.HistogramPoint{
				Timestamp: pbutils.Timestamp(a.ts),
				Sum:       a.sum,
				Count:     a.count,
				Buckets:   cumulativeBuckets(a.bounds, a.bucketCounts),
			})
		}

		out = append(out, &vmetricsv1.TimeSeries{
			Labels: groupLabels[groupKey],
			Points: &vmetricsv1.TimeSeries_Histogram{
				Histogram: pts,
			},
		})
	}

	out, total, truncated := limitAndSortSeries(out, q.limitSeries)
	return buildResponse(q, desc, out, total, truncated), nil
}

func (s *srvMetric) explicitHistogramNumberResponse(
	q *querySpec,
	desc *vmetricsv1.MetricDescriptor,
	aggs map[string]map[int]*explicitHistogramAgg,
	groupLabels map[string][]*vmetricsv1.Attribute,
) (*vmetricsv1.QueryMetricsResponse, error) {
	op := q.req.Operation.GetHistogram()
	var out []*vmetricsv1.TimeSeries

	if op.Function == vmetricsv1.HistogramOperation_QUANTILE {
		for groupKey, buckets := range aggs {
			for _, quantile := range op.Quantiles {
				idxs := sortedIntKeys(buckets)
				pts := &vmetricsv1.NumberPointSeries{}

				for _, idx := range idxs {
					a := buckets[idx]
					val := histogramQuantile(quantile, cumulativeBuckets(a.bounds, a.bucketCounts), a.count)
					pts.Points = append(pts.Points, numberPointDouble(a.ts, val))
				}

				labels := append([]*vmetricsv1.Attribute{}, groupLabels[groupKey]...)
				labels = append(labels, &vmetricsv1.Attribute{
					Key:   "quantile",
					Value: formatQuantile(quantile),
				})
				sortLabels(labels)

				out = append(out, &vmetricsv1.TimeSeries{
					Labels: labels,
					Points: &vmetricsv1.TimeSeries_Number{
						Number: pts,
					},
				})
			}
		}

		out, total, truncated := limitAndSortSeries(out, q.limitSeries)
		return buildResponse(q, desc, out, total, truncated), nil
	}

	for groupKey, buckets := range aggs {
		idxs := sortedIntKeys(buckets)
		pts := &vmetricsv1.NumberPointSeries{}

		for _, idx := range idxs {
			a := buckets[idx]
			var val float64

			switch op.Function {
			case vmetricsv1.HistogramOperation_AVG:
				if !a.hasSum {
					return nil, status.Error(codes.InvalidArgument, "histogram sum is not available")
				}
				if a.count > 0 {
					val = a.sum / float64(a.count)
				}

			case vmetricsv1.HistogramOperation_COUNT:
				val = float64(a.count)

			case vmetricsv1.HistogramOperation_SUM:
				if !a.hasSum {
					return nil, status.Error(codes.InvalidArgument, "histogram sum is not available")
				}
				val = a.sum

			default:
				return nil, status.Error(codes.InvalidArgument, "invalid histogram number function")
			}

			pts.Points = append(pts.Points, numberPointDouble(a.ts, val))
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

func (s *srvMetric) loadExplicitHistogramRows(
	ctx context.Context,
	q *querySpec,
	desc *vmetricsv1.MetricDescriptor,
	includeLookback bool,
) ([]explicitHistogramRow, error) {
	from := q.from
	if includeLookback {
		from = from.Add(-q.step)
	}

	where := []string{
		"name = ?",
		"timestamp >= ?",
		"timestamp < ?",
		"kind = 'HISTOGRAM'",
	}
	args := []any{q.name, from, q.to}

	if desc.Temporality != vmetricsv1.MetricDescriptor_TEMPORALITY_UNSET {
		where = append(where, "temporality = ?")
		args = append(args, temporalityEnumToString(desc.Temporality))
	}

	appendFilterSQL(&where, &args, q.filters)

	query := `
SELECT
    timestamp,
    CAST(attributes AS VARCHAR),
    histogram_has_sum,
    histogram_sum,
    histogram_count,
    histogram_min,
    histogram_max,
    CAST(histogram_bounds AS VARCHAR),
    CAST(histogram_bucket_counts AS VARCHAR)
FROM metrics
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY timestamp ASC
`

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []explicitHistogramRow

	for rows.Next() {
		var ts time.Time
		var attrsJSON string
		var hasSum bool
		var sumNull sql.NullFloat64
		var count uint64
		var minNull sql.NullFloat64
		var maxNull sql.NullFloat64
		var boundsJSON string
		var countsJSON string

		if err := rows.Scan(
			&ts,
			&attrsJSON,
			&hasSum,
			&sumNull,
			&count,
			&minNull,
			&maxNull,
			&boundsJSON,
			&countsJSON,
		); err != nil {
			return nil, err
		}

		attrs, err := decodeStringMap(attrsJSON)
		if err != nil {
			return nil, err
		}
		labels := anyMapToStringMap(attrs)

		var bounds []float64
		if err := json.Unmarshal([]byte(boundsJSON), &bounds); err != nil {
			return nil, err
		}

		var counts []uint64
		if err := json.Unmarshal([]byte(countsJSON), &counts); err != nil {
			return nil, err
		}

		if len(counts) != len(bounds)+1 {
			return nil, status.Error(codes.InvalidArgument, "invalid histogram bucket layout")
		}

		sum := 0.0
		if hasSum && sumNull.Valid {
			sum = sumNull.Float64
		}

		r := explicitHistogramRow{
			ts:           ts.UTC(),
			labels:       labels,
			fullKey:      labelsKey(labels, sortedKeys(labels)),
			groupKey:     labelsKey(labels, q.groupBy),
			sum:          sum,
			hasSum:       hasSum && sumNull.Valid,
			count:        count,
			bounds:       bounds,
			bucketCounts: counts,
		}

		if minNull.Valid {
			v := minNull.Float64
			r.min = &v
		}
		if maxNull.Valid {
			v := maxNull.Float64
			r.max = &v
		}

		out = append(out, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func deltaExplicitHistogram(prev, cur explicitHistogramRow) (explicitHistogramRow, error) {
	if !sameFloatSlice(prev.bounds, cur.bounds) {
		return explicitHistogramRow{}, status.Error(codes.InvalidArgument, "histogram bounds changed within query range")
	}

	out := cur
	out.min = nil
	out.max = nil

	if cur.count >= prev.count {
		out.count = cur.count - prev.count
	} else {
		out.count = cur.count
	}

	if cur.hasSum && prev.hasSum {
		if cur.sum >= prev.sum {
			out.sum = cur.sum - prev.sum
		} else {
			out.sum = cur.sum
		}
		out.hasSum = true
	} else {
		out.sum = 0
		out.hasSum = false
	}

	out.bucketCounts = make([]uint64, len(cur.bucketCounts))
	for i := range cur.bucketCounts {
		if cur.bucketCounts[i] >= prev.bucketCounts[i] {
			out.bucketCounts[i] = cur.bucketCounts[i] - prev.bucketCounts[i]
		} else {
			out.bucketCounts[i] = cur.bucketCounts[i]
		}
	}

	return out, nil
}

func cumulativeBuckets(bounds []float64, counts []uint64) []*vmetricsv1.HistogramBucket {
	ret := make([]*vmetricsv1.HistogramBucket, 0, len(counts))
	var cumulative uint64

	for i, c := range counts {
		cumulative += c
		if i < len(bounds) {
			ret = append(ret, &vmetricsv1.HistogramBucket{
				Le:    bounds[i],
				Count: cumulative,
			})
		} else {
			ret = append(ret, &vmetricsv1.HistogramBucket{
				Count: cumulative,
				IsInf: true,
			})
		}
	}

	return ret
}
