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

type explicitHistogramRawPoint struct {
	seriesID       string
	outputID       string
	timestamp      time.Time
	startTimestamp *time.Time
	count          uint64
	hasSum         bool
	sum            float64
	min            *float64
	max            *float64
	bucketCounts   []uint64
}

type explicitHistogramAggregate struct {
	timestamp    time.Time
	count        uint64
	hasSum       bool
	sum          float64
	min          *float64
	max          *float64
	bucketCounts []uint64
}

type histogramOutputGroup struct {
	id        string
	labels    []storedAttribute
	sourceIDs []string
	outputs   []*outputSeriesSpec
}

func (s *srvMetric) queryExplicitHistogram(ctx context.Context, query *querySpec,
	descriptor *vmetricsv1.MetricDescriptor, selection *seriesSelection) (*vmetricsv1.QueryMetricsResponse, error) {
	if len(selection.items) == 0 {
		return buildResponse(query, selection, nil,
			histogramResultDescriptor(descriptor, query.req.Operation.GetHistogram().Function)), nil
	}
	if (query.req.Operation.GetHistogram().Function == vmetricsv1.HistogramOperation_MIN ||
		query.req.Operation.GetHistogram().Function == vmetricsv1.HistogramOperation_MAX) &&
		descriptor.Temporality == vmetricsv1.MetricDescriptor_CUMULATIVE {
		return nil, status.Error(codes.InvalidArgument, "min/max cannot be derived from cumulative histogram deltas")
	}

	groups := histogramOutputGroups(selection, descriptor.Id)
	mappings := histogramGroupMappings(groups)
	aggregates := map[string]map[int64]*explicitHistogramAggregate{}
	bounds := descriptor.GetExplicitHistogram().Bounds

	err := s.withSeriesMapping(ctx, mappings, func(conn *sql.Conn) error {
		includePrevious := descriptor.Temporality == vmetricsv1.MetricDescriptor_CUMULATIVE
		if err := ensureRawRowLimit(ctx, conn, "metric_histogram_points", query, includePrevious,
			maximumRawHistogramRowsPerQuery); err != nil {
			return err
		}

		rows, err := loadExplicitHistogramQueryRows(ctx, conn, query, descriptor.Temporality)
		if err != nil {
			return err
		}
		defer rows.Close()

		previous := map[string]explicitHistogramRawPoint{}
		rowCount := 0
		for rows.Next() {
			rowCount++
			if rowCount > maximumRawHistogramRowsPerQuery {
				return status.Error(codes.ResourceExhausted, "histogram query scans too many raw points")
			}

			point, err := scanExplicitHistogramRawPoint(rows)
			if err != nil {
				return err
			}
			if len(point.bucketCounts) != len(bounds)+1 {
				return status.Error(codes.FailedPrecondition, "stored histogram bucket layout does not match descriptor")
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
					if point.startTimestamp == nil || point.startTimestamp.Before(query.from) {
						continue
					}
				} else {
					delta = deltaExplicitHistogramPoint(prev, point)
					previous[point.seriesID] = point
				}
			}

			bucketIndex := int64(point.timestamp.Sub(query.from) / query.step)
			if bucketIndex < 0 {
				continue
			}
			if aggregates[point.outputID] == nil {
				aggregates[point.outputID] = map[int64]*explicitHistogramAggregate{}
			}
			aggregate := aggregates[point.outputID][bucketIndex]
			if aggregate == nil {
				aggregate = &explicitHistogramAggregate{
					timestamp:    bucketEnd(query.from, query.step, bucketIndex),
					bucketCounts: make([]uint64, len(delta.bucketCounts)),
				}
				aggregates[point.outputID][bucketIndex] = aggregate
			}
			if err := mergeExplicitHistogramAggregate(aggregate, delta,
				descriptor.Temporality == vmetricsv1.MetricDescriptor_DELTA); err != nil {
				return err
			}
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	series, err := explicitHistogramResponseSeries(query, descriptor, groups, aggregates, bounds)
	if err != nil {
		return nil, err
	}
	return buildResponse(query, selection, series, histogramResultDescriptor(descriptor,
		query.req.Operation.GetHistogram().Function)), nil
}

func loadExplicitHistogramQueryRows(ctx context.Context, conn *sql.Conn, query *querySpec,
	temporality vmetricsv1.MetricDescriptor_Temporality) (*sql.Rows, error) {
	if temporality == vmetricsv1.MetricDescriptor_CUMULATIVE {
		return conn.QueryContext(ctx, `
WITH previous_ranked AS (
	SELECT
		p.*,
		ROW_NUMBER() OVER (
			PARTITION BY p.series_id
			ORDER BY p.timestamp DESC, p.point_id DESC, p.ingested_at ASC
		) AS row_number
	FROM metric_histogram_points p
	JOIN selected_metric_series m ON m.series_id = p.series_id
	WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
), range_deduplicated AS (
	SELECT * EXCLUDE (dedupe_row)
	FROM (
		SELECT
			p.*,
			ROW_NUMBER() OVER (PARTITION BY p.point_id ORDER BY p.ingested_at ASC) AS dedupe_row
		FROM metric_histogram_points p
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
		p.bucket_counts
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
		p.bucket_counts
	FROM range_deduplicated p
	JOIN selected_metric_series m ON m.series_id = p.series_id
)
SELECT
	series_id,
	output_id,
	timestamp,
	start_timestamp,
	count,
	has_sum,
	sum,
	min,
	max,
	CAST(bucket_counts AS VARCHAR)
FROM selected_points
ORDER BY series_id, timestamp
`, metricTimeToDB(query.baselineFrom()), metricTimeToDB(query.from), metricTimeToDB(query.snapshot),
			metricTimeToDB(query.from), metricTimeToDB(query.to), metricTimeToDB(query.snapshot))
	}

	return conn.QueryContext(ctx, `
WITH deduplicated AS (
	SELECT * EXCLUDE (dedupe_row)
	FROM (
		SELECT
			p.*,
			ROW_NUMBER() OVER (PARTITION BY p.point_id ORDER BY p.ingested_at ASC) AS dedupe_row
		FROM metric_histogram_points p
		JOIN selected_metric_series m ON m.series_id = p.series_id
		WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
	)
	WHERE dedupe_row = 1
)
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
	CAST(p.bucket_counts AS VARCHAR)
FROM deduplicated p
JOIN selected_metric_series m ON m.series_id = p.series_id
ORDER BY p.series_id, p.timestamp
`, metricTimeToDB(query.from), metricTimeToDB(query.to), metricTimeToDB(query.snapshot))
}

func scanExplicitHistogramRawPoint(scanner interface{ Scan(...any) error }) (explicitHistogramRawPoint, error) {
	var ret explicitHistogramRawPoint
	var timestamp int64
	var startTimestamp sql.NullInt64
	var sum, min, max sql.NullFloat64
	var bucketCountsJSON string

	if err := scanner.Scan(&ret.seriesID, &ret.outputID, &timestamp, &startTimestamp, &ret.count,
		&ret.hasSum, &sum, &min, &max, &bucketCountsJSON); err != nil {
		return explicitHistogramRawPoint{}, err
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
	if err := json.Unmarshal([]byte(bucketCountsJSON), &ret.bucketCounts); err != nil {
		zap.L().Error("Could not decode stored histogram bucket counts",
			zap.String("seriesID", ret.seriesID), zap.Error(err))
		return explicitHistogramRawPoint{}, status.Error(codes.Internal, "stored histogram data is invalid")
	}
	return ret, nil
}

func deltaExplicitHistogramPoint(previous, current explicitHistogramRawPoint) explicitHistogramRawPoint {
	reset := current.count < previous.count || timestampsDiffer(previous.startTimestamp, current.startTimestamp)
	if !reset {
		for index := range current.bucketCounts {
			if current.bucketCounts[index] < previous.bucketCounts[index] {
				reset = true
				break
			}
		}
	}

	ret := current
	ret.min = nil
	ret.max = nil
	if reset {
		return ret
	}

	ret.count = current.count - previous.count
	ret.bucketCounts = make([]uint64, len(current.bucketCounts))
	for index := range current.bucketCounts {
		ret.bucketCounts[index] = current.bucketCounts[index] - previous.bucketCounts[index]
	}
	if current.hasSum && previous.hasSum {
		ret.sum = current.sum - previous.sum
		ret.hasSum = true
	} else {
		ret.sum = 0
		ret.hasSum = false
	}
	return ret
}

func timestampsDiffer(previous, current *time.Time) bool {
	if previous == nil && current == nil {
		return false
	}
	if previous == nil || current == nil {
		return true
	}
	return !previous.Equal(*current)
}

func mergeExplicitHistogramAggregate(aggregate *explicitHistogramAggregate, point explicitHistogramRawPoint,
	preserveMinMax bool) error {
	if math.MaxUint64-aggregate.count < point.count {
		return status.Error(codes.FailedPrecondition, "histogram count overflow")
	}
	aggregate.count += point.count

	for index, count := range point.bucketCounts {
		if math.MaxUint64-aggregate.bucketCounts[index] < count {
			return status.Error(codes.FailedPrecondition, "histogram bucket count overflow")
		}
		aggregate.bucketCounts[index] += count
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

func mergeMin(current **float64, incoming *float64) {
	if incoming == nil {
		return
	}
	if *current == nil || *incoming < **current {
		value := *incoming
		*current = &value
	}
}

func mergeMax(current **float64, incoming *float64) {
	if incoming == nil {
		return
	}
	if *current == nil || *incoming > **current {
		value := *incoming
		*current = &value
	}
}

func explicitHistogramResponseSeries(query *querySpec, descriptor *vmetricsv1.MetricDescriptor,
	groups map[string]*histogramOutputGroup, aggregates map[string]map[int64]*explicitHistogramAggregate,
	bounds []float64) ([]*vmetricsv1.TimeSeries, error) {
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
					value, ok := explicitHistogramQuantile(*output.quantile, bounds, aggregate.bucketCounts, aggregate.count)
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
		switch function {
		case vmetricsv1.HistogramOperation_BUCKETS:
			points := &vmetricsv1.HistogramPointSeries{}
			for _, index := range indices {
				aggregate := buckets[index]
				point := &vmetricsv1.HistogramPoint{
					Timestamp: pbutils.Timestamp(aggregate.timestamp),
					Count:     aggregate.count,
					Buckets:   cumulativeExplicitBuckets(bounds, aggregate.bucketCounts),
					Min:       aggregate.min,
					Max:       aggregate.max,
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
					Points: &vmetricsv1.TimeSeries_Histogram{Histogram: points},
				})
			}

		default:
			points := &vmetricsv1.NumberPointSeries{}
			for _, index := range indices {
				aggregate := buckets[index]
				value, ok := explicitHistogramNumberValue(function, aggregate)
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
	}

	return ret, nil
}

func explicitHistogramNumberValue(function vmetricsv1.HistogramOperation_Function,
	aggregate *explicitHistogramAggregate) (float64, bool) {
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

func histogramOutputGroups(selection *seriesSelection, descriptorID string) map[string]*histogramOutputGroup {
	ret := map[string]*histogramOutputGroup{}
	for _, output := range selection.items {
		labelsKey := storedAttributesKey(output.labels)
		groupID := outputSeriesID(descriptorID, labelsKey, nil)
		group := ret[groupID]
		if group == nil {
			group = &histogramOutputGroup{
				id:        groupID,
				labels:    output.labels,
				sourceIDs: append([]string(nil), output.sourceIDs...),
			}
			ret[groupID] = group
		}
		group.outputs = append(group.outputs, output)
	}
	return ret
}

func histogramGroupMappings(groups map[string]*histogramOutputGroup) []querySeriesMapping {
	var ret []querySeriesMapping
	for groupID, group := range groups {
		for _, sourceID := range group.sourceIDs {
			ret = append(ret, querySeriesMapping{sourceID: sourceID, outputID: groupID})
		}
	}
	return ret
}

func histogramResultDescriptor(descriptor *vmetricsv1.MetricDescriptor,
	function vmetricsv1.HistogramOperation_Function) *vmetricsv1.QueryResultDescriptor {
	if function == vmetricsv1.HistogramOperation_BUCKETS {
		kind := vmetricsv1.QueryResultDescriptor_HISTOGRAM
		if descriptor.Kind == vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM {
			kind = vmetricsv1.QueryResultDescriptor_EXPONENTIAL_HISTOGRAM
		}
		return &vmetricsv1.QueryResultDescriptor{PointKind: kind, Unit: descriptor.Unit}
	}

	unit := descriptor.Unit
	if function == vmetricsv1.HistogramOperation_COUNT {
		unit = "1"
	}
	return &vmetricsv1.QueryResultDescriptor{
		PointKind:       vmetricsv1.QueryResultDescriptor_NUMBER,
		NumberValueType: vmetricsv1.MetricDescriptor_DOUBLE,
		Unit:            unit,
	}
}
