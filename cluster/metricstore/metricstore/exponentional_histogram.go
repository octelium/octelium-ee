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
	"math"
	"sort"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type exponentialHistogramRawPoint struct {
	seriesID       string
	outputID       string
	timestamp      time.Time
	startTimestamp *time.Time
	count          uint64
	hasSum         bool
	sum            float64
	min            *float64
	max            *float64
	scale          int32
	zeroCount      uint64
	zeroThreshold  float64
	positive       map[int32]uint64
	negative       map[int32]uint64
}

type exponentialHistogramAggregate struct {
	timestamp     time.Time
	count         uint64
	hasSum        bool
	sum           float64
	min           *float64
	max           *float64
	scale         int32
	zeroCount     uint64
	zeroThreshold float64
	positive      map[int32]uint64
	negative      map[int32]uint64
}

func (s *srvMetric) queryExponentialHistogram(ctx context.Context, query *querySpec,
	descriptor *vmetricsv1.MetricDescriptor, selection *seriesSelection) (*vmetricsv1.QueryMetricsResponse, error) {
	if len(selection.items) == 0 {
		return buildResponse(query, selection, nil, histogramResultDescriptor(descriptor,
			query.req.Operation.GetHistogram().Function)), nil
	}
	if (query.req.Operation.GetHistogram().Function == vmetricsv1.HistogramOperation_MIN ||
		query.req.Operation.GetHistogram().Function == vmetricsv1.HistogramOperation_MAX) &&
		descriptor.Temporality == vmetricsv1.MetricDescriptor_CUMULATIVE {
		return nil, status.Error(codes.InvalidArgument, "min/max cannot be derived from cumulative histogram deltas")
	}

	groups := histogramOutputGroups(selection, descriptor.Id)
	mappings := histogramGroupMappings(groups)
	aggregates := map[string]map[int64]*exponentialHistogramAggregate{}

	err := s.withSeriesMapping(ctx, mappings, func(conn *sql.Conn) error {
		includePrevious := descriptor.Temporality == vmetricsv1.MetricDescriptor_CUMULATIVE
		if err := ensureRawRowLimit(ctx, conn, "metric_exponential_histogram_points", query, includePrevious,
			maximumRawHistogramRowsPerQuery); err != nil {
			return err
		}

		rows, err := loadExponentialHistogramQueryRows(ctx, conn, query, descriptor.Temporality)
		if err != nil {
			return err
		}
		defer rows.Close()

		previous := map[string]exponentialHistogramRawPoint{}
		rowCount := 0
		for rows.Next() {
			rowCount++
			if rowCount > maximumRawHistogramRowsPerQuery {
				return status.Error(codes.ResourceExhausted, "exponential histogram query scans too many raw points")
			}

			point, err := scanExponentialHistogramRawPoint(rows)
			if err != nil {
				return err
			}
			if point.timestamp.Before(query.from) {
				previous[point.seriesID] = point
				continue
			}
			if !point.timestamp.Before(query.to) {
				continue
			}

			delta := point
			if descriptor.Temporality == vmetricsv1.MetricDescriptor_CUMULATIVE {
				prev, ok := previous[point.seriesID]
				if !ok {
					previous[point.seriesID] = point
					continue
				}
				delta = deltaExponentialHistogramPoint(prev, point)
				previous[point.seriesID] = point
			}

			bucketIndex := int64(point.timestamp.Sub(query.from) / query.step)
			if bucketIndex < 0 {
				continue
			}
			if aggregates[point.outputID] == nil {
				aggregates[point.outputID] = map[int64]*exponentialHistogramAggregate{}
			}
			aggregate := aggregates[point.outputID][bucketIndex]
			if aggregate == nil {
				aggregate = &exponentialHistogramAggregate{
					timestamp:     bucketEnd(query.from, query.step, bucketIndex),
					scale:         delta.scale,
					zeroThreshold: delta.zeroThreshold,
					positive:      map[int32]uint64{},
					negative:      map[int32]uint64{},
				}
				aggregates[point.outputID][bucketIndex] = aggregate
			}
			if err := mergeExponentialHistogramAggregate(aggregate, delta,
				descriptor.Temporality == vmetricsv1.MetricDescriptor_DELTA); err != nil {
				return err
			}
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	series, err := exponentialHistogramResponseSeries(query, descriptor, groups, aggregates)
	if err != nil {
		return nil, err
	}
	return buildResponse(query, selection, series, histogramResultDescriptor(descriptor,
		query.req.Operation.GetHistogram().Function)), nil
}

func loadExponentialHistogramQueryRows(ctx context.Context, conn *sql.Conn, query *querySpec,
	temporality vmetricsv1.MetricDescriptor_Temporality) (*sql.Rows, error) {
	columns := `
	series_id,
	output_id,
	timestamp,
	start_timestamp,
	count,
	has_sum,
	sum,
	min,
	max,
	scale,
	zero_count,
	zero_threshold,
	positive_offset,
	CAST(positive_counts AS VARCHAR),
	negative_offset,
	CAST(negative_counts AS VARCHAR)
`

	if temporality == vmetricsv1.MetricDescriptor_CUMULATIVE {
		return conn.QueryContext(ctx, `
WITH previous_deduplicated AS (
	SELECT * EXCLUDE (dedupe_row)
	FROM (
		SELECT
			p.*,
			ROW_NUMBER() OVER (PARTITION BY p.point_id ORDER BY p.ingested_at ASC) AS dedupe_row
		FROM metric_exponential_histogram_points p
		JOIN selected_metric_series m ON m.series_id = p.series_id
		WHERE p.timestamp < ? AND p.ingested_at <= ?
	)
	WHERE dedupe_row = 1
), previous_ranked AS (
	SELECT
		p.*,
		ROW_NUMBER() OVER (PARTITION BY p.series_id ORDER BY p.timestamp DESC, p.point_id DESC) AS row_number
	FROM previous_deduplicated p
), range_deduplicated AS (
	SELECT * EXCLUDE (dedupe_row)
	FROM (
		SELECT
			p.*,
			ROW_NUMBER() OVER (PARTITION BY p.point_id ORDER BY p.ingested_at ASC) AS dedupe_row
		FROM metric_exponential_histogram_points p
		JOIN selected_metric_series m ON m.series_id = p.series_id
		WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
	)
	WHERE dedupe_row = 1
), selected_points AS (
	SELECT
		p.series_id,
		m.output_id,
		p.timestamp,
		p.start_timestamp,
		p.count,
		p.has_sum,
		p.sum,
		p.min,
		p.max,
		p.scale,
		p.zero_count,
		p.zero_threshold,
		p.positive_offset,
		p.positive_counts,
		p.negative_offset,
		p.negative_counts
	FROM previous_ranked p
	JOIN selected_metric_series m ON m.series_id = p.series_id
	WHERE p.row_number = 1
	UNION ALL
	SELECT
		p.series_id,
		m.output_id,
		p.timestamp,
		p.start_timestamp,
		p.count,
		p.has_sum,
		p.sum,
		p.min,
		p.max,
		p.scale,
		p.zero_count,
		p.zero_threshold,
		p.positive_offset,
		p.positive_counts,
		p.negative_offset,
		p.negative_counts
	FROM range_deduplicated p
	JOIN selected_metric_series m ON m.series_id = p.series_id
)
SELECT `+columns+`
FROM selected_points
ORDER BY series_id, timestamp
`, metricTimeToDB(query.from), metricTimeToDB(query.snapshot), metricTimeToDB(query.from),
			metricTimeToDB(query.to), metricTimeToDB(query.snapshot))
	}

	return conn.QueryContext(ctx, `
WITH deduplicated AS (
	SELECT * EXCLUDE (dedupe_row)
	FROM (
		SELECT
			p.*,
			ROW_NUMBER() OVER (PARTITION BY p.point_id ORDER BY p.ingested_at ASC) AS dedupe_row
		FROM metric_exponential_histogram_points p
		JOIN selected_metric_series m ON m.series_id = p.series_id
		WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
	)
	WHERE dedupe_row = 1
), selected_points AS (
	SELECT p.*, m.output_id
	FROM deduplicated p
	JOIN selected_metric_series m ON m.series_id = p.series_id
)
SELECT `+columns+`
FROM selected_points
ORDER BY series_id, timestamp
`, metricTimeToDB(query.from), metricTimeToDB(query.to), metricTimeToDB(query.snapshot))
}

func scanExponentialHistogramRawPoint(scanner interface{ Scan(...any) error }) (exponentialHistogramRawPoint, error) {
	var ret exponentialHistogramRawPoint
	var timestamp int64
	var startTimestamp sql.NullInt64
	var sum, min, max sql.NullFloat64
	var positiveOffset, negativeOffset int32
	var positiveJSON, negativeJSON string

	if err := scanner.Scan(&ret.seriesID, &ret.outputID, &timestamp, &startTimestamp, &ret.count,
		&ret.hasSum, &sum, &min, &max, &ret.scale, &ret.zeroCount, &ret.zeroThreshold,
		&positiveOffset, &positiveJSON, &negativeOffset, &negativeJSON); err != nil {
		return exponentialHistogramRawPoint{}, err
	}
	ret.timestamp = metricTimeFromDB(timestamp)
	if startTimestamp.Valid {
		value := metricTimeFromDB(startTimestamp.Int64)
		ret.startTimestamp = &value
	}
	if ret.hasSum && sum.Valid {
		ret.sum = sum.Float64
	} else {
		ret.hasSum = false
	}
	if min.Valid {
		value := min.Float64
		ret.min = &value
	}
	if max.Valid {
		value := max.Float64
		ret.max = &value
	}

	var positiveCounts, negativeCounts []uint64
	if err := json.Unmarshal([]byte(positiveJSON), &positiveCounts); err != nil {
		zap.L().Error("Could not decode stored exponential histogram positive counts",
			zap.String("seriesID", ret.seriesID), zap.Error(err))
		return exponentialHistogramRawPoint{}, status.Error(codes.Internal, "stored histogram data is invalid")
	}
	if err := json.Unmarshal([]byte(negativeJSON), &negativeCounts); err != nil {
		zap.L().Error("Could not decode stored exponential histogram negative counts",
			zap.String("seriesID", ret.seriesID), zap.Error(err))
		return exponentialHistogramRawPoint{}, status.Error(codes.Internal, "stored histogram data is invalid")
	}
	ret.positive = denseBucketMap(positiveOffset, positiveCounts)
	ret.negative = denseBucketMap(negativeOffset, negativeCounts)
	return ret, nil
}

func denseBucketMap(offset int32, counts []uint64) map[int32]uint64 {
	ret := make(map[int32]uint64, len(counts))
	for index, count := range counts {
		if count != 0 {
			ret[offset+int32(index)] = count
		}
	}
	return ret
}

func deltaExponentialHistogramPoint(previous, current exponentialHistogramRawPoint) exponentialHistogramRawPoint {
	reset := current.scale != previous.scale || current.zeroThreshold != previous.zeroThreshold ||
		current.count < previous.count || current.zeroCount < previous.zeroCount ||
		timestampsDiffer(previous.startTimestamp, current.startTimestamp) ||
		bucketMapDecreased(previous.positive, current.positive) || bucketMapDecreased(previous.negative, current.negative)

	ret := current
	ret.min = nil
	ret.max = nil
	if reset {
		return ret
	}

	ret.count = current.count - previous.count
	ret.zeroCount = current.zeroCount - previous.zeroCount
	ret.positive = subtractBucketMaps(previous.positive, current.positive)
	ret.negative = subtractBucketMaps(previous.negative, current.negative)
	if current.hasSum && previous.hasSum {
		ret.sum = current.sum - previous.sum
		ret.hasSum = true
	} else {
		ret.sum = 0
		ret.hasSum = false
	}
	return ret
}

func bucketMapDecreased(previous, current map[int32]uint64) bool {
	for index, previousCount := range previous {
		if current[index] < previousCount {
			return true
		}
	}
	return false
}

func subtractBucketMaps(previous, current map[int32]uint64) map[int32]uint64 {
	ret := map[int32]uint64{}
	for index, currentCount := range current {
		value := currentCount - previous[index]
		if value != 0 {
			ret[index] = value
		}
	}
	return ret
}

func mergeExponentialHistogramAggregate(aggregate *exponentialHistogramAggregate,
	point exponentialHistogramRawPoint, preserveMinMax bool) error {
	if aggregate.scale != point.scale || aggregate.zeroThreshold != point.zeroThreshold {
		return status.Error(codes.FailedPrecondition, "cannot merge exponential histograms with incompatible layouts")
	}
	if math.MaxUint64-aggregate.count < point.count || math.MaxUint64-aggregate.zeroCount < point.zeroCount {
		return status.Error(codes.FailedPrecondition, "exponential histogram count overflow")
	}
	aggregate.count += point.count
	aggregate.zeroCount += point.zeroCount

	for index, count := range point.positive {
		if math.MaxUint64-aggregate.positive[index] < count {
			return status.Error(codes.FailedPrecondition, "exponential histogram bucket count overflow")
		}
		aggregate.positive[index] += count
	}
	for index, count := range point.negative {
		if math.MaxUint64-aggregate.negative[index] < count {
			return status.Error(codes.FailedPrecondition, "exponential histogram bucket count overflow")
		}
		aggregate.negative[index] += count
	}

	if point.hasSum {
		aggregate.sum += point.sum
		aggregate.hasSum = true
	}
	if preserveMinMax {
		mergeMin(&aggregate.min, point.min)
		mergeMax(&aggregate.max, point.max)
	}
	return nil
}

func exponentialHistogramResponseSeries(query *querySpec, descriptor *vmetricsv1.MetricDescriptor,
	groups map[string]*histogramOutputGroup,
	aggregates map[string]map[int64]*exponentialHistogramAggregate) ([]*vmetricsv1.TimeSeries, error) {
	function := query.req.Operation.GetHistogram().Function
	var ret []*vmetricsv1.TimeSeries

	for groupID, group := range groups {
		buckets := aggregates[groupID]
		indices := make([]int64, 0, len(buckets))
		for index := range buckets {
			indices = append(indices, index)
		}
		sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })

		if function == vmetricsv1.HistogramOperation_QUANTILE {
			for _, output := range group.outputs {
				points := &vmetricsv1.NumberPointSeries{}
				for _, index := range indices {
					aggregate := buckets[index]
					value, ok := exponentialHistogramQuantile(*output.quantile, aggregate)
					if !ok {
						continue
					}
					points.Points = append(points.Points, numberPointDouble(aggregate.timestamp, nil, value))
				}
				if len(points.Points) > 0 {
					ret = append(ret, &vmetricsv1.TimeSeries{
						Id:       output.id,
						Labels:   storedAttributesToProto(output.labels),
						Points:   &vmetricsv1.TimeSeries_Number{Number: points},
						Quantile: output.quantile,
					})
				}
			}
			continue
		}

		output := group.outputs[0]
		if function == vmetricsv1.HistogramOperation_BUCKETS {
			points := &vmetricsv1.ExponentialHistogramPointSeries{}
			for _, index := range indices {
				aggregate := buckets[index]
				positiveOffset, positiveCounts := denseBucketSlice(aggregate.positive)
				negativeOffset, negativeCounts := denseBucketSlice(aggregate.negative)
				point := &vmetricsv1.ExponentialHistogramPoint{
					Timestamp:     pbutils.Timestamp(aggregate.timestamp),
					Count:         aggregate.count,
					Min:           aggregate.min,
					Max:           aggregate.max,
					Scale:         aggregate.scale,
					ZeroCount:     aggregate.zeroCount,
					ZeroThreshold: aggregate.zeroThreshold,
					Positive: &vmetricsv1.ExponentialHistogramPoint_Buckets{
						Offset: positiveOffset,
						Counts: positiveCounts,
					},
					Negative: &vmetricsv1.ExponentialHistogramPoint_Buckets{
						Offset: negativeOffset,
						Counts: negativeCounts,
					},
				}
				if aggregate.hasSum {
					sum := aggregate.sum
					point.Sum = &sum
				}
				points.Points = append(points.Points, point)
			}
			if len(points.Points) > 0 {
				ret = append(ret, &vmetricsv1.TimeSeries{
					Id:     output.id,
					Labels: storedAttributesToProto(output.labels),
					Points: &vmetricsv1.TimeSeries_ExponentialHistogram{ExponentialHistogram: points},
				})
			}
			continue
		}

		points := &vmetricsv1.NumberPointSeries{}
		for _, index := range indices {
			aggregate := buckets[index]
			value, ok := exponentialHistogramNumberValue(function, aggregate)
			if !ok {
				return nil, status.Error(codes.FailedPrecondition, "requested histogram value is not available")
			}
			points.Points = append(points.Points, numberPointDouble(aggregate.timestamp, nil, value))
		}
		if len(points.Points) > 0 {
			ret = append(ret, &vmetricsv1.TimeSeries{
				Id:     output.id,
				Labels: storedAttributesToProto(output.labels),
				Points: &vmetricsv1.TimeSeries_Number{Number: points},
			})
		}
	}

	return ret, nil
}

func denseBucketSlice(values map[int32]uint64) (int32, []uint64) {
	if len(values) == 0 {
		return 0, nil
	}
	keys := make([]int, 0, len(values))
	for key := range values {
		keys = append(keys, int(key))
	}
	sort.Ints(keys)
	minimum := keys[0]
	maximum := keys[len(keys)-1]
	ret := make([]uint64, maximum-minimum+1)
	for key, value := range values {
		ret[int(key)-minimum] = value
	}
	return int32(minimum), ret
}

func exponentialHistogramNumberValue(function vmetricsv1.HistogramOperation_Function,
	aggregate *exponentialHistogramAggregate) (float64, bool) {
	switch function {
	case vmetricsv1.HistogramOperation_COUNT:
		return float64(aggregate.count), true
	case vmetricsv1.HistogramOperation_SUM:
		return aggregate.sum, aggregate.hasSum
	case vmetricsv1.HistogramOperation_AVG:
		if !aggregate.hasSum {
			return 0, false
		}
		if aggregate.count == 0 {
			return 0, true
		}
		return aggregate.sum / float64(aggregate.count), true
	case vmetricsv1.HistogramOperation_MIN:
		if aggregate.min == nil {
			return 0, false
		}
		return *aggregate.min, true
	case vmetricsv1.HistogramOperation_MAX:
		if aggregate.max == nil {
			return 0, false
		}
		return *aggregate.max, true
	default:
		return 0, false
	}
}

type exponentialQuantileBucket struct {
	lower float64
	upper float64
	count uint64
}

func exponentialHistogramQuantile(quantile float64, aggregate *exponentialHistogramAggregate) (float64, bool) {
	if aggregate.count == 0 {
		return 0, false
	}

	base := math.Pow(2, math.Ldexp(1, -int(aggregate.scale)))
	if math.IsNaN(base) || math.IsInf(base, 0) || base <= 1 {
		return 0, false
	}

	var buckets []exponentialQuantileBucket
	negativeKeys := sortedBucketKeys(aggregate.negative, true)
	for _, index := range negativeKeys {
		absoluteLower := math.Pow(base, float64(index))
		absoluteUpper := math.Pow(base, float64(index+1))
		buckets = append(buckets, exponentialQuantileBucket{
			lower: -absoluteUpper,
			upper: -absoluteLower,
			count: aggregate.negative[index],
		})
	}
	if aggregate.zeroCount > 0 {
		buckets = append(buckets, exponentialQuantileBucket{
			lower: -aggregate.zeroThreshold,
			upper: aggregate.zeroThreshold,
			count: aggregate.zeroCount,
		})
	}
	positiveKeys := sortedBucketKeys(aggregate.positive, false)
	for _, index := range positiveKeys {
		buckets = append(buckets, exponentialQuantileBucket{
			lower: math.Pow(base, float64(index)),
			upper: math.Pow(base, float64(index+1)),
			count: aggregate.positive[index],
		})
	}

	target := quantile * float64(aggregate.count)
	if quantile == 0 {
		target = 1
	}
	var cumulative uint64
	for _, bucket := range buckets {
		previous := cumulative
		if math.MaxUint64-cumulative < bucket.count {
			return 0, false
		}
		cumulative += bucket.count
		if float64(cumulative) < target {
			continue
		}
		if bucket.count == 0 {
			return bucket.upper, true
		}
		position := (target - float64(previous)) / float64(bucket.count)
		return bucket.lower + position*(bucket.upper-bucket.lower), true
	}
	return 0, false
}

func sortedBucketKeys(values map[int32]uint64, descending bool) []int32 {
	ret := make([]int32, 0, len(values))
	for key, count := range values {
		if count != 0 {
			ret = append(ret, key)
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		if descending {
			return ret[i] > ret[j]
		}
		return ret[i] < ret[j]
	})
	return ret
}
