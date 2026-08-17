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
	"sort"
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const metricQueryTimeout = 30 * time.Second

type sourceSeries struct {
	id        string
	labels    []storedAttribute
	labelsKey string
}

type outputSeriesSpec struct {
	id           string
	canonicalKey string
	labels       []storedAttribute
	sourceIDs    []string
	quantile     *float64
}

type seriesSelection struct {
	items           []*outputSeriesSpec
	total           uint32
	seriesTruncated bool
}

func (s *srvMetric) QueryMetrics(ctx context.Context,
	req *vmetricsv1.QueryMetricsRequest) (*vmetricsv1.QueryMetricsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, metricQueryTimeout)
	defer cancel()

	query, err := s.validateQueryRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	descriptor, err := s.resolveDescriptor(ctx, query)
	if err != nil {
		return nil, err
	}
	if err := validateQueryForDescriptor(query, descriptor); err != nil {
		return nil, err
	}

	selection, err := s.selectOutputSeries(ctx, query, descriptor)
	if err != nil {
		return nil, err
	}

	var response *vmetricsv1.QueryMetricsResponse
	switch descriptor.Kind {
	case vmetricsv1.MetricDescriptor_COUNTER:
		response, err = s.queryCounter(ctx, query, descriptor, selection)
	case vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_GAUGE:
		response, err = s.queryGauge(ctx, query, descriptor, selection)
	case vmetricsv1.MetricDescriptor_HISTOGRAM:
		response, err = s.queryExplicitHistogram(ctx, query, descriptor, selection)
	case vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM:
		response, err = s.queryExponentialHistogram(ctx, query, descriptor, selection)
	default:
		err = status.Error(codes.InvalidArgument, "unsupported metric kind")
	}
	if err != nil {
		return nil, err
	}

	response.SourceDescriptor = descriptor
	response.Operation = req.Operation
	if query.step > 0 {
		response.Step = durationPB(query.step)
	}
	response.SnapshotTime = pbutils.Timestamp(query.snapshot)
	if req.IncludeTotalSeries {
		total := selection.total
		response.TotalSeries = &total
	}

	if response.Truncation == nil {
		response.Truncation = &vmetricsv1.TruncationInfo{}
	}
	if selection.seriesTruncated {
		response.Truncation.SeriesTruncated = true
		response.Truncation.Reasons = append(response.Truncation.Reasons, vmetricsv1.TruncationInfo_SERVER_LIMIT)
	}

	if err := enforceResponseLimits(query, response); err != nil {
		return nil, err
	}
	response.Truncation.ReturnedSeries = uint32(len(response.Series))
	response.Truncation.ReturnedPoints = uint32(countResponsePoints(response.Series))
	response.Truncation.Reasons = uniqueTruncationReasons(response.Truncation.Reasons)

	return response, nil
}

func (s *srvMetric) selectOutputSeries(ctx context.Context, query *querySpec,
	descriptor *vmetricsv1.MetricDescriptor) (*seriesSelection, error) {
	source, err := s.loadSourceSeries(ctx, query, descriptor)
	if err != nil {
		return nil, err
	}

	grouped := map[string]*outputSeriesSpec{}
	for _, item := range source {
		labels := item.labels
		if query.req.SeriesAggregation != vmetricsv1.QueryMetricsRequest_NONE {
			labels = pickStoredAttributes(item.labels, query.groupBy)
			if len(labels) != len(query.groupBy) {
				continue
			}
		}
		labelsKey := storedAttributesKey(labels)
		key := labelsKey

		output := grouped[key]
		if output == nil {
			output = &outputSeriesSpec{
				labels:    labels,
				sourceIDs: []string{},
			}
			grouped[key] = output
		}
		output.sourceIDs = append(output.sourceIDs, item.id)
	}

	var outputs []*outputSeriesSpec
	operation := query.req.Operation.GetHistogram()
	for _, output := range grouped {
		labelsKey := storedAttributesKey(output.labels)
		if operation != nil && operation.Function == vmetricsv1.HistogramOperation_QUANTILE {
			for _, quantile := range operation.Quantiles {
				q := quantile
				item := &outputSeriesSpec{
					labels:    output.labels,
					sourceIDs: append([]string(nil), output.sourceIDs...),
					quantile:  &q,
				}
				item.id = outputSeriesID(descriptor.Id, labelsKey, item.quantile)
				item.canonicalKey = item.id
				outputs = append(outputs, item)
			}
		} else {
			output.id = outputSeriesID(descriptor.Id, labelsKey, nil)
			output.canonicalKey = output.id
			outputs = append(outputs, output)
		}
	}

	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].canonicalKey < outputs[j].canonicalKey
	})

	total := uint32(len(outputs))
	outputs, truncated, err := limitOutputSeries(
		outputs,
		query.limitSeries,
		query.limitBehavior,
	)
	if err != nil {
		return nil, err
	}

	return &seriesSelection{
		items:           outputs,
		total:           total,
		seriesTruncated: truncated,
	}, nil
}

func limitOutputSeries(
	items []*outputSeriesSpec,
	limit int,
	behavior vmetricsv1.QueryMetricsRequest_LimitBehavior,
) ([]*outputSeriesSpec, bool, error) {
	if limit < 0 {
		limit = 0
	}
	if len(items) <= limit {
		return items, false, nil
	}

	if behavior == vmetricsv1.QueryMetricsRequest_ERROR ||
		behavior == vmetricsv1.QueryMetricsRequest_LIMIT_BEHAVIOR_UNSET {
		return nil, false, status.Error(
			codes.ResourceExhausted,
			"metric query exceeds the output-series limit",
		)
	}

	if behavior != vmetricsv1.QueryMetricsRequest_TRUNCATE {
		return nil, false, status.Error(
			codes.InvalidArgument,
			"invalid limit behavior",
		)
	}

	return items[:limit], true, nil
}

func (s *srvMetric) loadSourceSeries(ctx context.Context, query *querySpec,
	descriptor *vmetricsv1.MetricDescriptor) ([]sourceSeries, error) {
	pointTable := pointTableForKind(descriptor.Kind)
	if pointTable == "" {
		return nil, status.Error(codes.InvalidArgument, "unsupported metric kind")
	}

	where := []string{"s.descriptor_id = ?"}
	args := []any{metricTimeToDB(query.from), metricTimeToDB(query.to), metricTimeToDB(query.snapshot), descriptor.Id}

	if component := query.req.Component; component != nil {
		if component.Type != "" {
			where = append(where, "s.component_type = ?")
			args = append(args, component.Type)
		}
		if component.Namespace != "" {
			where = append(where, "s.component_namespace = ?")
			args = append(args, component.Namespace)
		}
		if component.Name != "" {
			where = append(where, "s.component_name = ?")
			args = append(args, component.Name)
		}
	}

	appendSeriesFilterSQL(&where, &args, query.filters)

	querySQL := `
WITH active_series AS (
	SELECT DISTINCT series_id
	FROM ` + pointTable + `
	WHERE timestamp >= ? AND timestamp < ? AND ingested_at <= ?
)
SELECT s.id, CAST(s.labels AS VARCHAR), s.labels_key
FROM metric_series s
JOIN active_series a ON a.series_id = s.id
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY s.labels_key ASC, s.id ASC
LIMIT ?
`
	args = append(args, maximumSourceSeries+1)

	rows, err := s.s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := make([]sourceSeries, 0)
	for rows.Next() {
		var id, labelsJSON, labelsKey string
		if err := rows.Scan(&id, &labelsJSON, &labelsKey); err != nil {
			return nil, err
		}
		labels, err := decodeStoredAttributes(labelsJSON)
		if err != nil {
			return nil, err
		}
		ret = append(ret, sourceSeries{id: id, labels: labels, labelsKey: labelsKey})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ret) > maximumSourceSeries {
		return nil, status.Error(codes.ResourceExhausted,
			"metric query exceeds the source-series cardinality limit")
	}
	return ret, nil
}

func appendSeriesFilterSQL(where *[]string, args *[]any, filters []*vmetricsv1.AttributeFilter) {
	for _, filter := range filters {
		switch filter.Operator {
		case vmetricsv1.AttributeFilter_EQ:
			*where = append(*where, `EXISTS (
				SELECT 1 FROM metric_series_attributes a
				WHERE a.series_id = s.id AND a.key = ? AND a.value_key = ?
			)`)
			*args = append(*args, filter.Key, protoAttributeValueKey(filter.Value))

		case vmetricsv1.AttributeFilter_NOT_EQ:
			*where = append(*where, `EXISTS (
				SELECT 1 FROM metric_series_attributes a
				WHERE a.series_id = s.id AND a.key = ? AND a.value_key != ?
			)`)
			*args = append(*args, filter.Key, protoAttributeValueKey(filter.Value))

		case vmetricsv1.AttributeFilter_EXISTS:
			*where = append(*where, `EXISTS (
				SELECT 1 FROM metric_series_attributes a
				WHERE a.series_id = s.id AND a.key = ?
			)`)
			*args = append(*args, filter.Key)

		case vmetricsv1.AttributeFilter_NOT_EXISTS:
			*where = append(*where, `NOT EXISTS (
				SELECT 1 FROM metric_series_attributes a
				WHERE a.series_id = s.id AND a.key = ?
			)`)
			*args = append(*args, filter.Key)

		case vmetricsv1.AttributeFilter_IN:
			placeholders := make([]string, 0, len(filter.Values))
			*args = append(*args, filter.Key)
			for _, value := range filter.Values {
				placeholders = append(placeholders, "?")
				*args = append(*args, protoAttributeValueKey(value))
			}
			*where = append(*where, `EXISTS (
				SELECT 1 FROM metric_series_attributes a
				WHERE a.series_id = s.id AND a.key = ? AND a.value_key IN (`+strings.Join(placeholders, ",")+`)
			)`)
		}
	}
}

func pointTableForKind(kind vmetricsv1.MetricDescriptor_Kind) string {
	switch kind {
	case vmetricsv1.MetricDescriptor_COUNTER,
		vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		vmetricsv1.MetricDescriptor_GAUGE:
		return "metric_number_points"
	case vmetricsv1.MetricDescriptor_HISTOGRAM:
		return "metric_histogram_points"
	case vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM:
		return "metric_exponential_histogram_points"
	default:
		return ""
	}
}

func buildResponse(query *querySpec, selection *seriesSelection,
	series []*vmetricsv1.TimeSeries, result *vmetricsv1.QueryResultDescriptor) *vmetricsv1.QueryMetricsResponse {
	byID := make(map[string]*vmetricsv1.TimeSeries, len(series))
	for _, item := range series {
		byID[item.Id] = item
	}

	ordered := make([]*vmetricsv1.TimeSeries, 0, len(series))
	for _, selected := range selection.items {
		if item := byID[selected.id]; item != nil {
			ordered = append(ordered, item)
		}
	}

	return &vmetricsv1.QueryMetricsResponse{
		Series:     ordered,
		Result:     result,
		Truncation: &vmetricsv1.TruncationInfo{},
	}
}

func enforceResponseLimits(query *querySpec, response *vmetricsv1.QueryMetricsResponse) error {
	if response.Truncation == nil {
		response.Truncation = &vmetricsv1.TruncationInfo{}
	}

	for _, series := range response.Series {
		count := timeSeriesPointCount(series)
		if count <= query.limitPointsPerSeries {
			continue
		}
		if query.limitBehavior == vmetricsv1.QueryMetricsRequest_ERROR {
			return status.Error(codes.ResourceExhausted, "metric series exceeds limitPointsPerSeries")
		}
		trimTimeSeriesToNewest(series, query.limitPointsPerSeries)
		response.Truncation.PointsTruncated = true
		response.Truncation.Reasons = append(response.Truncation.Reasons,
			vmetricsv1.TruncationInfo_POINTS_PER_SERIES_LIMIT)
	}

	total := countResponsePoints(response.Series)
	if total <= query.limitTotalPoints {
		return nil
	}
	if query.limitBehavior == vmetricsv1.QueryMetricsRequest_ERROR {
		return status.Error(codes.ResourceExhausted, "metric response exceeds limitTotalPoints")
	}

	remaining := query.limitTotalPoints
	for _, series := range response.Series {
		count := timeSeriesPointCount(series)
		if remaining <= 0 {
			clearTimeSeriesPoints(series)
			continue
		}
		if count > remaining {
			trimTimeSeriesToNewest(series, remaining)
			remaining = 0
		} else {
			remaining -= count
		}
	}
	response.Truncation.PointsTruncated = true
	response.Truncation.Reasons = append(response.Truncation.Reasons, vmetricsv1.TruncationInfo_TOTAL_POINTS_LIMIT)
	return nil
}

func countResponsePoints(series []*vmetricsv1.TimeSeries) int {
	ret := 0
	for _, item := range series {
		ret += timeSeriesPointCount(item)
	}
	return ret
}

func timeSeriesPointCount(series *vmetricsv1.TimeSeries) int {
	switch points := series.Points.(type) {
	case *vmetricsv1.TimeSeries_Number:
		return len(points.Number.Points)
	case *vmetricsv1.TimeSeries_Histogram:
		return len(points.Histogram.Points)
	case *vmetricsv1.TimeSeries_ExponentialHistogram:
		return len(points.ExponentialHistogram.Points)
	default:
		return 0
	}
}

func trimTimeSeriesToNewest(series *vmetricsv1.TimeSeries, limit int) {
	if limit < 0 {
		limit = 0
	}
	switch points := series.Points.(type) {
	case *vmetricsv1.TimeSeries_Number:
		if len(points.Number.Points) > limit {
			points.Number.Points = points.Number.Points[len(points.Number.Points)-limit:]
		}
	case *vmetricsv1.TimeSeries_Histogram:
		if len(points.Histogram.Points) > limit {
			points.Histogram.Points = points.Histogram.Points[len(points.Histogram.Points)-limit:]
		}
	case *vmetricsv1.TimeSeries_ExponentialHistogram:
		if len(points.ExponentialHistogram.Points) > limit {
			points.ExponentialHistogram.Points = points.ExponentialHistogram.Points[len(points.ExponentialHistogram.Points)-limit:]
		}
	}
}

func clearTimeSeriesPoints(series *vmetricsv1.TimeSeries) {
	switch points := series.Points.(type) {
	case *vmetricsv1.TimeSeries_Number:
		points.Number.Points = nil
	case *vmetricsv1.TimeSeries_Histogram:
		points.Histogram.Points = nil
	case *vmetricsv1.TimeSeries_ExponentialHistogram:
		points.ExponentialHistogram.Points = nil
	}
}

func selectedSourceSeries(selection *seriesSelection) ([]string, map[string]*outputSeriesSpec) {
	ids := map[string]struct{}{}
	bySource := map[string]*outputSeriesSpec{}
	for _, output := range selection.items {
		for _, sourceID := range output.sourceIDs {
			ids[sourceID] = struct{}{}
			if output.quantile == nil {
				bySource[sourceID] = output
			}
		}
	}

	ret := make([]string, 0, len(ids))
	for id := range ids {
		ret = append(ret, id)
	}
	sort.Strings(ret)
	return ret, bySource
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func bucketIndexExpression() string {
	return `CAST(FLOOR((timestamp - ?) / CAST(? AS DOUBLE)) AS BIGINT)`
}

func bucketEnd(from time.Time, step time.Duration, index int64) time.Time {
	return from.Add(time.Duration(index+1) * step)
}
