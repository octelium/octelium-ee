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

	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const descriptorAttributeKeyLookback = 24 * time.Hour

func (s *srvMetric) QueryMetrics(
	ctx context.Context,
	req *vmetricsv1.QueryMetricsRequest,
) (*vmetricsv1.QueryMetricsResponse, error) {
	q, err := s.validateQueryRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	desc, err := s.resolveDescriptor(ctx, q)
	if err != nil {
		return nil, err
	}

	if err := validateOperationForKind(desc.Kind, req.Operation); err != nil {
		return nil, err
	}

	if req.Metric.Kind != vmetricsv1.MetricDescriptor_KIND_UNSET && req.Metric.Kind != desc.Kind {
		return nil, status.Error(codes.InvalidArgument, "metric kind does not match descriptor")
	}

	switch desc.Kind {
	case vmetricsv1.MetricDescriptor_COUNTER:
		return s.queryCounter(ctx, q, desc)

	case vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		vmetricsv1.MetricDescriptor_GAUGE:
		return s.queryGauge(ctx, q, desc)

	case vmetricsv1.MetricDescriptor_HISTOGRAM:
		return s.queryExplicitHistogram(ctx, q, desc)

	case vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM:
		return nil, status.Error(codes.Unimplemented, "exponential histogram queries are not implemented")

	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported metric kind")
	}
}

func (s *srvMetric) ListMetricDescriptors(
	ctx context.Context,
	req *vmetricsv1.ListMetricDescriptorsRequest,
) (*vmetricsv1.ListMetricDescriptorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	limit := int(req.Limit)
	if limit <= 0 || limit > maxDescriptorLimit {
		limit = maxDescriptorLimit
	}

	where := []string{"1 = 1"}
	args := []any{}

	if req.NamePrefix != "" {
		where = append(where, "name LIKE ?")
		args = append(args, req.NamePrefix+"%")
	}

	if req.Component != nil {
		if req.Component.Type != "" {
			where = append(where, "component_type = ?")
			args = append(args, req.Component.Type)
		}
		if req.Component.Namespace != "" {
			where = append(where, "component_namespace = ?")
			args = append(args, req.Component.Namespace)
		}
		if req.Component.Name != "" {
			where = append(where, "component_name = ?")
			args = append(args, req.Component.Name)
		}
	}

	if req.PageToken != "" {
		where = append(where, "name > ?")
		args = append(args, req.PageToken)
	}

	query := `
SELECT
    name,
    any_value(kind) AS kind,
    any_value(value_type) AS value_type,
    any_value(unit) AS unit,
    any_value(description) AS description,
    any_value(temporality) AS temporality
FROM metrics
WHERE ` + strings.Join(where, " AND ") + `
GROUP BY name
ORDER BY name ASC
LIMIT ?
`
	args = append(args, limit+1)

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descs []*vmetricsv1.MetricDescriptor

	for rows.Next() {
		var name, kind, valueType, unit, description, temporality string

		if err := rows.Scan(&name, &kind, &valueType, &unit, &description, &temporality); err != nil {
			return nil, err
		}

		descs = append(descs, &vmetricsv1.MetricDescriptor{
			Name:        name,
			Kind:        kindFromString(kind),
			ValueType:   valueTypeFromString(valueType),
			Unit:        unit,
			Description: description,
			Temporality: temporalityFromString(temporality),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	truncated := false
	if len(descs) > limit {
		truncated = true
		descs = descs[:limit]
	}

	if len(descs) > 0 {
		if err := s.populateDescriptorAttributeKeys(ctx, descs); err != nil {
			return nil, err
		}
	}

	resp := &vmetricsv1.ListMetricDescriptorsResponse{
		Items: descs,
	}

	if truncated {
		resp.NextPageToken = descs[len(descs)-1].Name
	}

	return resp, nil
}

func (s *srvMetric) populateDescriptorAttributeKeys(
	ctx context.Context,
	descs []*vmetricsv1.MetricDescriptor,
) error {
	if len(descs) == 0 {
		return nil
	}

	names := make([]string, 0, len(descs))
	byName := make(map[string]*vmetricsv1.MetricDescriptor, len(descs))

	for _, d := range descs {
		names = append(names, d.Name)
		byName[d.Name] = d
	}

	placeholders := make([]string, 0, len(names))
	args := make([]any, 0, len(names)+1)
	for _, name := range names {
		placeholders = append(placeholders, "?")
		args = append(args, name)
	}

	args = append(args, time.Now().UTC().Add(-descriptorAttributeKeyLookback))

	query := `
SELECT DISTINCT name, unnest(json_keys(attributes)) AS attr_key
FROM metrics
WHERE name IN (` + strings.Join(placeholders, ",") + `)
  AND timestamp >= ?
`

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	keysByName := map[string]map[string]struct{}{}

	for rows.Next() {
		var name string
		var key string

		if err := rows.Scan(&name, &key); err != nil {
			return err
		}

		if key == "" {
			continue
		}

		if keysByName[name] == nil {
			keysByName[name] = map[string]struct{}{}
		}

		keysByName[name][key] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	for name, keySet := range keysByName {
		desc := byName[name]
		if desc == nil {
			continue
		}

		for k := range keySet {
			desc.AttributeKeys = append(desc.AttributeKeys, k)
		}
		sort.Strings(desc.AttributeKeys)
	}

	return nil
}

func (s *srvMetric) ListMetricCatalog(
	ctx context.Context,
	req *vmetricsv1.ListMetricCatalogRequest,
) (*vmetricsv1.ListMetricCatalogResponse, error) {
	var items []*vmetricsv1.MetricCatalogItem

	items = append(items,
		&vmetricsv1.MetricCatalogItem{
			Id:          "process_goroutines",
			DisplayName: "Goroutines",
			Description: "Number of goroutines",
			Metric: &vmetricsv1.MetricSelector{
				Name: "process.goroutines",
				Kind: vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
			},
			DefaultOperation: &vmetricsv1.QueryOperation{
				Type: &vmetricsv1.QueryOperation_Gauge{
					Gauge: &vmetricsv1.GaugeOperation{
						Function: vmetricsv1.GaugeOperation_LAST,
					},
				},
			},
			DefaultGroupBy: []string{"octelium.component.type", "octelium.component.namespace"},
			Unit:           "goroutines",
		},
		&vmetricsv1.MetricCatalogItem{
			Id:          "process_heap_alloc",
			DisplayName: "Heap allocation",
			Description: "Bytes allocated by heap",
			Metric: &vmetricsv1.MetricSelector{
				Name: "process.mem.heap_alloc",
				Kind: vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
			},
			DefaultOperation: &vmetricsv1.QueryOperation{
				Type: &vmetricsv1.QueryOperation_Gauge{
					Gauge: &vmetricsv1.GaugeOperation{
						Function: vmetricsv1.GaugeOperation_LAST,
					},
				},
			},
			DefaultGroupBy: []string{"octelium.component.type", "octelium.component.namespace"},
			Unit:           "bytes",
		},
	)

	if catalogMatchesComponent(req, "octovigil", "rscserver", "portal", "authserver", "vigil") {
		items = append(items,
			&vmetricsv1.MetricCatalogItem{
				Id:          "requests_rate",
				DisplayName: "Requests rate",
				Description: "Per-second request rate",
				Metric: &vmetricsv1.MetricSelector{
					Name: "req.total",
					Kind: vmetricsv1.MetricDescriptor_COUNTER,
				},
				DefaultOperation: &vmetricsv1.QueryOperation{
					Type: &vmetricsv1.QueryOperation_Counter{
						Counter: &vmetricsv1.CounterOperation{
							Function: vmetricsv1.CounterOperation_RATE,
						},
					},
				},
				DefaultGroupBy: []string{"octelium.component.type", "octelium.component.namespace"},
				Unit:           "requests/s",
			},
			&vmetricsv1.MetricCatalogItem{
				Id:          "active_requests",
				DisplayName: "Active requests",
				Description: "Current active requests",
				Metric: &vmetricsv1.MetricSelector{
					Name: "req.active",
					Kind: vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
				},
				DefaultOperation: &vmetricsv1.QueryOperation{
					Type: &vmetricsv1.QueryOperation_Gauge{
						Gauge: &vmetricsv1.GaugeOperation{
							Function: vmetricsv1.GaugeOperation_LAST,
						},
					},
				},
				DefaultGroupBy: []string{"octelium.component.type", "octelium.component.namespace"},
				Unit:           "requests",
			},
			&vmetricsv1.MetricCatalogItem{
				Id:          "request_p95_latency",
				DisplayName: "Request p95 latency",
				Description: "p95 request duration",
				Metric: &vmetricsv1.MetricSelector{
					Name: "req.duration",
					Kind: vmetricsv1.MetricDescriptor_HISTOGRAM,
				},
				DefaultOperation: &vmetricsv1.QueryOperation{
					Type: &vmetricsv1.QueryOperation_Histogram{
						Histogram: &vmetricsv1.HistogramOperation{
							Function:  vmetricsv1.HistogramOperation_QUANTILE,
							Quantiles: []float64{0.95},
						},
					},
				},
				DefaultGroupBy: []string{"octelium.component.type", "octelium.component.namespace"},
				Unit:           "ms",
			},
		)
	}

	return &vmetricsv1.ListMetricCatalogResponse{
		Items: items,
	}, nil
}

func catalogMatchesComponent(req *vmetricsv1.ListMetricCatalogRequest, componentTypes ...string) bool {
	if req == nil || req.Component == nil || req.Component.Type == "" {
		return true
	}

	for _, typ := range componentTypes {
		if req.Component.Type == typ {
			return true
		}
	}

	return false
}

func buildResponse(
	q *querySpec,
	desc *vmetricsv1.MetricDescriptor,
	series []*vmetricsv1.TimeSeries,
	totalSeries uint32,
	truncated bool,
) *vmetricsv1.QueryMetricsResponse {
	return &vmetricsv1.QueryMetricsResponse{
		Descriptor_: desc,
		Operation:   q.req.Operation,
		Step:        durationPB(q.step),
		Series:      series,
		Truncated:   truncated,
		TotalSeries: totalSeries,
	}
}

func durationPB(d time.Duration) *metav1.Duration {
	if d%time.Second == 0 {
		return &metav1.Duration{
			Type: &metav1.Duration_Seconds{Seconds: uint32(d / time.Second)},
		}
	}
	return &metav1.Duration{
		Type: &metav1.Duration_Milliseconds{Milliseconds: uint32(d / time.Millisecond)},
	}
}
