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
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
)

func (s *srvMetric) queryGauge(ctx context.Context, query *querySpec,
	descriptor *vmetricsv1.MetricDescriptor, selection *seriesSelection) (*vmetricsv1.QueryMetricsResponse, error) {
	if len(selection.items) == 0 {
		return buildResponse(query, selection, nil, numberResultDescriptor(descriptor, descriptor.Unit)), nil
	}

	mappings, byOutput := numberSeriesMappings(selection)
	series := make(map[string]*vmetricsv1.TimeSeries, len(byOutput))
	for outputID, output := range byOutput {
		series[outputID] = &vmetricsv1.TimeSeries{
			Id:     output.id,
			Labels: storedAttributesToProto(output.labels),
			Points: &vmetricsv1.TimeSeries_Number{Number: &vmetricsv1.NumberPointSeries{}},
		}
	}

	err := s.withSeriesMapping(ctx, mappings, func(conn *sql.Conn) error {
		if err := ensureRawRowLimit(ctx, conn, "metric_number_points", query, false,
			maximumRawNumberRowsPerQuery); err != nil {
			return err
		}

		perSourceValue := gaugePerSourceExpression(query.req.Operation.GetGauge().Function)
		seriesAggregation := numberSeriesAggregationExpression(query.req.SeriesAggregation)

		querySQL := `
WITH deduplicated AS (
	SELECT * EXCLUDE (dedupe_row)
	FROM (
		SELECT
			p.*,
			ROW_NUMBER() OVER (PARTITION BY p.point_id ORDER BY p.ingested_at ASC) AS dedupe_row
		FROM metric_number_points p
		JOIN selected_metric_series m ON m.series_id = p.series_id
		WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
	)
	WHERE dedupe_row = 1
), per_source AS (
	SELECT
		m.output_id,
		p.series_id,
		` + bucketIndexExpression() + ` AS bucket_idx,
		COUNT(*) AS value_count,
		SUM(COALESCE(p.number_double, CAST(p.number_int AS DOUBLE))) AS value_sum,
		MIN(COALESCE(p.number_double, CAST(p.number_int AS DOUBLE))) AS value_min,
		MAX(COALESCE(p.number_double, CAST(p.number_int AS DOUBLE))) AS value_max,
		arg_max(COALESCE(p.number_double, CAST(p.number_int AS DOUBLE)),
			printf('%020d:%s', p.timestamp, p.point_id)) AS value_last,
		MAX(p.timestamp) AS last_timestamp
	FROM deduplicated p
	JOIN selected_metric_series m ON m.series_id = p.series_id
	GROUP BY m.output_id, p.series_id, bucket_idx
), source_values AS (
	SELECT
		output_id,
		series_id,
		bucket_idx,
		` + perSourceValue + ` AS source_value,
		last_timestamp
	FROM per_source
)
SELECT
	output_id,
	bucket_idx,
	` + seriesAggregation + ` AS output_value
FROM source_values
GROUP BY output_id, bucket_idx
ORDER BY output_id, bucket_idx
`
		rows, err := conn.QueryContext(ctx, querySQL,
			metricTimeToDB(query.from), metricTimeToDB(query.to), metricTimeToDB(query.snapshot),
			metricTimeToDB(query.from), query.step.Nanoseconds())
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var outputID string
			var bucketIndex int64
			var value float64
			if err := rows.Scan(&outputID, &bucketIndex, &value); err != nil {
				return err
			}
			item := series[outputID]
			if item == nil {
				continue
			}
			item.GetNumber().Points = append(item.GetNumber().Points,
				numberPointDouble(bucketEnd(query.from, query.step, bucketIndex), nil, value))
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return buildResponse(query, selection, numberSeriesMapValues(series),
		numberResultDescriptor(descriptor, descriptor.Unit)), nil
}

func (s *srvMetric) queryCounter(ctx context.Context, query *querySpec,
	descriptor *vmetricsv1.MetricDescriptor, selection *seriesSelection) (*vmetricsv1.QueryMetricsResponse, error) {
	if query.req.Operation.GetCounter().Function == vmetricsv1.CounterOperation_RAW {
		return s.queryCounterRaw(ctx, query, descriptor, selection)
	}
	if len(selection.items) == 0 {
		return buildResponse(query, selection, nil, numberResultDescriptor(descriptor, counterResultUnit(descriptor.Unit,
			query.req.Operation.GetCounter().Function))), nil
	}

	mappings, byOutput := numberSeriesMappings(selection)
	series := make(map[string]*vmetricsv1.TimeSeries, len(byOutput))
	for outputID, output := range byOutput {
		series[outputID] = &vmetricsv1.TimeSeries{
			Id:     output.id,
			Labels: storedAttributesToProto(output.labels),
			Points: &vmetricsv1.TimeSeries_Number{Number: &vmetricsv1.NumberPointSeries{}},
		}
	}

	values := map[string]map[int64]float64{}

	err := s.withSeriesMapping(ctx, mappings, func(conn *sql.Conn) error {
		includePrevious := descriptor.Temporality == vmetricsv1.MetricDescriptor_CUMULATIVE
		if err := ensureRawRowLimit(ctx, conn, "metric_number_points", query, includePrevious,
			maximumRawNumberRowsPerQuery); err != nil {
			return err
		}

		isDelta := descriptor.Temporality == vmetricsv1.MetricDescriptor_DELTA
		var querySQL string
		args := []any{}
		if isDelta {
			querySQL = counterDeltaSQL(query)
			args = append(args, metricTimeToDB(query.from), metricTimeToDB(query.to), metricTimeToDB(query.snapshot),
				metricTimeToDB(query.from), query.step.Nanoseconds())
		} else {
			querySQL = counterCumulativeSQL(query)
			args = append(args,
				metricTimeToDB(query.baselineFrom()), metricTimeToDB(query.from), metricTimeToDB(query.snapshot),
				metricTimeToDB(query.from), metricTimeToDB(query.to), metricTimeToDB(query.snapshot),
				metricTimeToDB(query.from), query.step.Nanoseconds(), metricTimeToDB(query.from),
				metricTimeToDB(query.from), metricTimeToDB(query.from),
			)
		}

		rows, err := conn.QueryContext(ctx, querySQL, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var outputID string
			var bucketIndex int64
			var value float64
			if err := rows.Scan(&outputID, &bucketIndex, &value); err != nil {
				return err
			}
			if isDelta && query.req.Operation.GetCounter().Function == vmetricsv1.CounterOperation_RATE {
				value /= query.step.Seconds()
			}
			if series[outputID] == nil {
				continue
			}
			if values[outputID] == nil {
				values[outputID] = map[int64]float64{}
			}
			values[outputID][bucketIndex] = value
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	for outputID, item := range series {
		fillCounterBuckets(query, item, values[outputID])
	}

	unit := counterResultUnit(descriptor.Unit, query.req.Operation.GetCounter().Function)
	return buildResponse(query, selection, numberSeriesMapValues(series), numberResultDescriptor(descriptor, unit)), nil
}

func (s *srvMetric) queryCounterRaw(ctx context.Context, query *querySpec,
	descriptor *vmetricsv1.MetricDescriptor, selection *seriesSelection) (*vmetricsv1.QueryMetricsResponse, error) {
	if len(selection.items) == 0 {
		return buildResponse(query, selection, nil, rawNumberResultDescriptor(descriptor)), nil
	}

	mappings, byOutput := numberSeriesMappings(selection)
	series := make(map[string]*vmetricsv1.TimeSeries, len(byOutput))
	for outputID, output := range byOutput {
		series[outputID] = &vmetricsv1.TimeSeries{
			Id:     output.id,
			Labels: storedAttributesToProto(output.labels),
			Points: &vmetricsv1.TimeSeries_Number{Number: &vmetricsv1.NumberPointSeries{}},
		}
	}

	err := s.withSeriesMapping(ctx, mappings, func(conn *sql.Conn) error {
		if err := ensureRawRowLimit(ctx, conn, "metric_number_points", query, false,
			maximumRawNumberRowsPerQuery); err != nil {
			return err
		}

		rows, err := conn.QueryContext(ctx, `
WITH deduplicated AS (
	SELECT * EXCLUDE (dedupe_row)
	FROM (
		SELECT
			p.*,
			ROW_NUMBER() OVER (PARTITION BY p.point_id ORDER BY p.ingested_at ASC) AS dedupe_row
		FROM metric_number_points p
		JOIN selected_metric_series m ON m.series_id = p.series_id
		WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
	)
	WHERE dedupe_row = 1
), ranked AS (
	SELECT
		m.output_id,
		p.timestamp,
		p.start_timestamp,
		p.number_int,
		p.number_double,
		ROW_NUMBER() OVER (PARTITION BY m.output_id ORDER BY p.timestamp DESC, p.point_id DESC) AS row_number
	FROM deduplicated p
	JOIN selected_metric_series m ON m.series_id = p.series_id
)
SELECT output_id, timestamp, start_timestamp, number_int, number_double
FROM ranked
WHERE row_number <= ?
ORDER BY output_id, timestamp
`, metricTimeToDB(query.from), metricTimeToDB(query.to), metricTimeToDB(query.snapshot),
			query.limitPointsPerSeries+1)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var outputID string
			var timestamp int64
			var startTimestamp sql.NullInt64
			var intValue sql.NullInt64
			var doubleValue sql.NullFloat64
			if err := rows.Scan(&outputID, &timestamp, &startTimestamp, &intValue, &doubleValue); err != nil {
				return err
			}

			item := series[outputID]
			if item == nil {
				continue
			}
			var start *time.Time
			if startTimestamp.Valid {
				value := metricTimeFromDB(startTimestamp.Int64)
				start = &value
			}
			pointTime := metricTimeFromDB(timestamp)
			if intValue.Valid {
				item.GetNumber().Points = append(item.GetNumber().Points,
					numberPointInt(pointTime, start, intValue.Int64))
			} else if doubleValue.Valid {
				item.GetNumber().Points = append(item.GetNumber().Points,
					numberPointDouble(pointTime, start, doubleValue.Float64))
			}
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return buildResponse(query, selection, numberSeriesMapValues(series), rawNumberResultDescriptor(descriptor)), nil
}

func fillCounterBuckets(query *querySpec, series *vmetricsv1.TimeSeries, values map[int64]float64) {
	points := series.GetNumber()
	if points == nil {
		return
	}

	count := query.bucketCount()
	points.Points = make([]*vmetricsv1.NumberPoint, 0, count)
	for index := int64(0); index < count; index++ {
		points.Points = append(points.Points,
			numberPointDouble(bucketEnd(query.from, query.step, index), nil, values[index]))
	}
}

func gaugePerSourceExpression(function vmetricsv1.GaugeOperation_Function) string {
	switch function {
	case vmetricsv1.GaugeOperation_LAST:
		return "value_last"
	case vmetricsv1.GaugeOperation_AVG:
		return "value_sum / value_count"
	case vmetricsv1.GaugeOperation_MIN:
		return "value_min"
	case vmetricsv1.GaugeOperation_MAX:
		return "value_max"
	case vmetricsv1.GaugeOperation_SUM:
		return "value_sum"
	default:
		return "value_last"
	}
}

func numberSeriesAggregationExpression(aggregation vmetricsv1.QueryMetricsRequest_SeriesAggregation) string {
	switch aggregation {
	case vmetricsv1.QueryMetricsRequest_NONE:
		return "MAX(source_value)"
	case vmetricsv1.QueryMetricsRequest_SUM:
		return "SUM(source_value)"
	case vmetricsv1.QueryMetricsRequest_AVG:
		return "AVG(source_value)"
	case vmetricsv1.QueryMetricsRequest_MIN:
		return "MIN(source_value)"
	case vmetricsv1.QueryMetricsRequest_MAX:
		return "MAX(source_value)"
	case vmetricsv1.QueryMetricsRequest_LAST:
		return "arg_max(source_value, printf('%020d:%s', last_timestamp, series_id))"
	default:
		return "MAX(source_value)"
	}
}

func counterDeltaSQL(query *querySpec) string {
	aggregation := numberSeriesAggregationExpression(query.req.SeriesAggregation)
	return `
WITH deduplicated AS (
	SELECT * EXCLUDE (dedupe_row)
	FROM (
		SELECT
			p.*,
			ROW_NUMBER() OVER (PARTITION BY p.point_id ORDER BY p.ingested_at ASC) AS dedupe_row
		FROM metric_number_points p
		JOIN selected_metric_series m ON m.series_id = p.series_id
		WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
	)
	WHERE dedupe_row = 1
), per_source AS (
	SELECT
		m.output_id,
		p.series_id,
		` + bucketIndexExpression() + ` AS bucket_idx,
		SUM(COALESCE(p.number_double, CAST(p.number_int AS DOUBLE))) AS source_value,
		MAX(p.timestamp) AS last_timestamp
	FROM deduplicated p
	JOIN selected_metric_series m ON m.series_id = p.series_id
	GROUP BY m.output_id, p.series_id, bucket_idx
)
SELECT output_id, bucket_idx, ` + aggregation + ` AS output_value
FROM per_source
GROUP BY output_id, bucket_idx
ORDER BY output_id, bucket_idx
`
}

func counterCumulativeSQL(query *querySpec) string {
	aggregation := numberSeriesAggregationExpression(query.req.SeriesAggregation)
	sourceValue := "increment_sum"
	if query.req.Operation.GetCounter().Function == vmetricsv1.CounterOperation_RATE {
		sourceValue = "increment_sum / (covered_nanos / 1000000000.0)"
	}
	return `
WITH previous_ranked AS (
	SELECT
		p.series_id,
		p.timestamp,
		p.start_timestamp,
		COALESCE(p.number_double, CAST(p.number_int AS DOUBLE)) AS value,
		ROW_NUMBER() OVER (
			PARTITION BY p.series_id
			ORDER BY p.timestamp DESC, p.point_id DESC, p.ingested_at ASC
		) AS row_number
	FROM metric_number_points p
	JOIN selected_metric_series m ON m.series_id = p.series_id
	WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
), range_deduplicated AS (
	SELECT * EXCLUDE (dedupe_row)
	FROM (
		SELECT
			p.*,
			ROW_NUMBER() OVER (PARTITION BY p.point_id ORDER BY p.ingested_at ASC) AS dedupe_row
		FROM metric_number_points p
		JOIN selected_metric_series m ON m.series_id = p.series_id
		WHERE p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
	)
	WHERE dedupe_row = 1
), selected_points AS (
	SELECT series_id, timestamp, start_timestamp, value
	FROM previous_ranked
	WHERE row_number = 1
	UNION ALL
	SELECT
		p.series_id,
		p.timestamp,
		p.start_timestamp,
		COALESCE(p.number_double, CAST(p.number_int AS DOUBLE)) AS value
	FROM range_deduplicated p
), with_previous AS (
	SELECT
		*,
		LAG(value) OVER (PARTITION BY series_id ORDER BY timestamp, start_timestamp NULLS FIRST) AS previous_value,
		LAG(timestamp) OVER (PARTITION BY series_id ORDER BY timestamp, start_timestamp NULLS FIRST) AS previous_timestamp,
		LAG(start_timestamp) OVER (
			PARTITION BY series_id
			ORDER BY timestamp, start_timestamp NULLS FIRST
		) AS previous_start_timestamp
	FROM selected_points
), increments AS (
	SELECT
		m.output_id,
		p.series_id,
		` + bucketIndexExpression() + ` AS bucket_idx,
		CASE
			WHEN p.previous_value IS NULL AND p.start_timestamp IS NOT NULL AND p.start_timestamp >= ? THEN p.value
			WHEN p.previous_value IS NULL THEN NULL
			WHEN p.start_timestamp IS DISTINCT FROM p.previous_start_timestamp THEN p.value
			WHEN p.value >= p.previous_value THEN p.value - p.previous_value
			ELSE p.value
		END AS increment,
		CASE
			WHEN p.previous_value IS NULL AND p.start_timestamp IS NOT NULL AND p.start_timestamp >= ?
				THEN p.timestamp - p.start_timestamp
			WHEN p.previous_value IS NULL THEN NULL
			WHEN p.start_timestamp IS DISTINCT FROM p.previous_start_timestamp
				THEN p.timestamp - COALESCE(p.start_timestamp, p.previous_timestamp)
			ELSE p.timestamp - p.previous_timestamp
		END AS covered,
		p.timestamp
	FROM with_previous p
	JOIN selected_metric_series m ON m.series_id = p.series_id
	WHERE p.timestamp >= ?
), per_source AS (
	SELECT
		output_id,
		series_id,
		bucket_idx,
		SUM(increment) AS increment_sum,
		SUM(covered) AS covered_nanos,
		MAX(timestamp) AS last_timestamp
	FROM increments
	WHERE increment IS NOT NULL AND increment >= 0 AND covered > 0
	GROUP BY output_id, series_id, bucket_idx
), source_values AS (
	SELECT
		output_id,
		series_id,
		bucket_idx,
		last_timestamp,
		` + sourceValue + ` AS source_value
	FROM per_source
)
SELECT output_id, bucket_idx, ` + aggregation + ` AS output_value
FROM source_values
GROUP BY output_id, bucket_idx
ORDER BY output_id, bucket_idx
`
}

func numberSeriesMappings(selection *seriesSelection) ([]querySeriesMapping, map[string]*outputSeriesSpec) {
	mappings := make([]querySeriesMapping, 0)
	byOutput := make(map[string]*outputSeriesSpec, len(selection.items))
	for _, output := range selection.items {
		byOutput[output.id] = output
		for _, sourceID := range output.sourceIDs {
			mappings = append(mappings, querySeriesMapping{sourceID: sourceID, outputID: output.id})
		}
	}
	return mappings, byOutput
}

func numberSeriesMapValues(values map[string]*vmetricsv1.TimeSeries) []*vmetricsv1.TimeSeries {
	ret := make([]*vmetricsv1.TimeSeries, 0, len(values))
	for _, value := range values {
		if len(value.GetNumber().Points) > 0 {
			ret = append(ret, value)
		}
	}
	return ret
}

func numberResultDescriptor(descriptor *vmetricsv1.MetricDescriptor, unit string) *vmetricsv1.QueryResultDescriptor {
	return &vmetricsv1.QueryResultDescriptor{
		PointKind:       vmetricsv1.QueryResultDescriptor_NUMBER,
		NumberValueType: vmetricsv1.MetricDescriptor_DOUBLE,
		Unit:            unit,
	}
}

func rawNumberResultDescriptor(descriptor *vmetricsv1.MetricDescriptor) *vmetricsv1.QueryResultDescriptor {
	return &vmetricsv1.QueryResultDescriptor{
		PointKind:       vmetricsv1.QueryResultDescriptor_NUMBER,
		NumberValueType: descriptor.NumberValueType,
		Unit:            descriptor.Unit,
	}
}

func counterResultUnit(unit string, function vmetricsv1.CounterOperation_Function) string {
	if function != vmetricsv1.CounterOperation_RATE {
		return unit
	}
	if strings.TrimSpace(unit) == "" || unit == "1" {
		return "1/s"
	}
	return unit + "/s"
}
