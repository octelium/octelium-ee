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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

type TimeValue struct {
	Timestamp time.Time
	Labels    map[string]any
	Value     float64
}

func (s *srvMetric) doQueryCounter(
	ctx context.Context,
	name string,
	from, to time.Time,
	filterLabels map[string]string,
) (*visibilityv1.GetCounterResponse, error) {

	where := []string{
		"name = $1",
		"metric_type = 'Sum'",
		"timestamp >= $2",
		"timestamp <= $3",
	}

	args := []any{name, from, to}
	argIndex := 4

	for k, v := range filterLabels {
		where = append(where, fmt.Sprintf("attributes->>%d = $%d", argIndex, argIndex+1))
		args = append(args, k, v)
		argIndex += 2
	}

	query := fmt.Sprintf(`
        SELECT timestamp, attributes, value
        FROM metrics
        WHERE %s
        ORDER BY timestamp ASC
    `, strings.Join(where, " AND "))

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp := &visibilityv1.GetCounterResponse{}

	for rows.Next() {
		var ts time.Time
		var labels map[string]any
		var value float64

		if err := rows.Scan(&ts, &labels, &value); err != nil {
			return nil, err
		}

		resp.Attrs = pbutils.MapToStructMust(labels)
		resp.Points = append(resp.Points, &visibilityv1.GetCounterResponse_DataPoint{
			Timestamp: pbutils.Timestamp(ts),
			Value:     value,
		})

	}

	return resp, nil
}

func (s *srvMetric) doQueryGauge(
	ctx context.Context,
	name string,
	from, to time.Time,
	filterLabels map[string]string,
) (*visibilityv1.GetGaugeResponse, error) {

	where := []string{
		"name = $1",
		"metric_type = 'Gauge'",
		"timestamp >= $2",
		"timestamp <= $3",
	}

	args := []any{name, from, to}
	argIndex := 4

	for k, v := range filterLabels {
		where = append(where, fmt.Sprintf("attributes->>%d = $%d", argIndex, argIndex+1))
		args = append(args, k, v)
		argIndex += 2
	}

	query := fmt.Sprintf(`
        SELECT timestamp, attributes, value
        FROM metrics
        WHERE %s
        ORDER BY timestamp ASC
    `, strings.Join(where, " AND "))

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp := &visibilityv1.GetGaugeResponse{}

	for rows.Next() {
		var ts time.Time
		var labels map[string]any
		var value float64

		if err := rows.Scan(&ts, &labels, &value); err != nil {
			return nil, err
		}

		resp.Attrs = pbutils.MapToStructMust(labels)
		resp.Points = append(resp.Points, &visibilityv1.GetGaugeResponse_DataPoint{
			Timestamp: pbutils.Timestamp(ts),
			Value:     value,
		})

	}

	return resp, nil
}

type Bucket struct {
	UpperBound float64
	Count      uint64
}

type Histogram struct {
	Timestamp time.Time
	Labels    map[string]any
	Sum       float64
	Count     uint64
	Buckets   []Bucket
}

func (s *srvMetric) doQueryHistogram(
	ctx context.Context,
	name string,
	from, to time.Time,
	filterLabels map[string]string,
) ([]Histogram, error) {

	where := []string{
		"name = $1",
		"metric_type = 'Histogram'",
		"timestamp >= $2",
		"timestamp <= $3",
	}
	args := []any{name, from, to}
	argIndex := 4

	for k, v := range filterLabels {
		where = append(where, fmt.Sprintf("attributes->>%d = $%d", argIndex, argIndex+1))
		args = append(args, k, v)
		argIndex += 2
	}

	query := fmt.Sprintf(`
        SELECT
            timestamp,
            attributes,
            histogram_bounds,
            histogram_bucket_counts,
            histogram_sum,
            histogram_count
        FROM metrics
        WHERE %s
        ORDER BY timestamp ASC
    `, strings.Join(where, " AND "))

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	histograms := []Histogram{}

	for rows.Next() {
		var (
			ts            time.Time
			labels        map[string]any
			boundsI       []any
			bucketCountsI []any
			sumVal        float64
			countVal      int64
		)

		if err := rows.Scan(
			&ts,
			&labels,
			&boundsI,
			&bucketCountsI,
			&sumVal,
			&countVal,
		); err != nil {
			return nil, err
		}

		bounds := make([]float64, len(boundsI))
		for i, v := range boundsI {
			bounds[i] = toFloat(v)
		}

		bucketCounts := make([]uint64, len(bucketCountsI))
		for i, v := range bucketCountsI {
			bucketCounts[i] = uint64(toFloat(v))
		}

		buckets := make([]Bucket, len(bounds))
		for i := range bounds {
			buckets[i] = Bucket{
				UpperBound: bounds[i],
				Count:      bucketCounts[i],
			}
		}

		histograms = append(histograms, Histogram{
			Timestamp: ts,
			Labels:    labels,
			Sum:       sumVal,
			Count:     uint64(countVal),
			Buckets:   buckets,
		})
	}

	return histograms, nil
}

type Quantile struct {
	Quantile float64 `json:"q"`
	Value    float64 `json:"v"`
}

type Summary struct {
	Timestamp time.Time
	Labels    map[string]any
	Sum       float64
	Count     uint64
	Quantiles []Quantile
}

func (s *srvMetric) doQuerySummary(
	ctx context.Context,
	name string,
	from, to time.Time,
	filterLabels map[string]string,
) ([]Summary, error) {

	where := []string{
		"name = $1",
		"metric_type = 'Summary'",
		"timestamp >= $2",
		"timestamp <= $3",
	}

	args := []any{name, from, to}
	argIndex := 4

	for k, v := range filterLabels {
		where = append(where, fmt.Sprintf("attributes->>%d = $%d", argIndex, argIndex+1))
		args = append(args, k, v)
		argIndex += 2
	}

	query := fmt.Sprintf(`
        SELECT
            timestamp,
            attributes,
            summary_sum,
            summary_count,
            summary_quantiles
        FROM metrics
        WHERE %s
        ORDER BY timestamp ASC
    `, strings.Join(where, " AND "))

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Summary{}

	for rows.Next() {
		var (
			ts            time.Time
			labels        map[string]any
			sumVal        float64
			countVal      int64
			quantilesJSON []byte
		)

		if err := rows.Scan(
			&ts,
			&labels,
			&sumVal,
			&countVal,
			&quantilesJSON,
		); err != nil {
			return nil, err
		}

		var quantiles []Quantile
		if len(quantilesJSON) > 0 {
			if err := json.Unmarshal(quantilesJSON, &quantiles); err != nil {
				return nil, err
			}
		}

		out = append(out, Summary{
			Timestamp: ts,
			Labels:    labels,
			Sum:       sumVal,
			Count:     uint64(countVal),
			Quantiles: quantiles,
		})
	}

	return out, nil
}

func toMapStr(source map[string]any) map[string]string {
	destination := make(map[string]string, len(source))

	for key, value := range source {
		strValue, ok := value.(string)

		if ok {
			destination[key] = strValue
		}
	}

	return destination
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}
