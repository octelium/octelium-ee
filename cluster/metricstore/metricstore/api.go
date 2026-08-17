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
	"fmt"
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxDescriptorResults      = 5000
	attributeEstimateLookback = rawMetricRetention
)

func (s *srvMetric) ListMetricDescriptors(ctx context.Context,
	req *vmetricsv1.ListMetricDescriptorsRequest) (*vmetricsv1.ListMetricDescriptorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	req.NamePrefix = strings.TrimSpace(req.NamePrefix)
	if len(req.NamePrefix) > maxMetricNameLength {
		return nil, status.Error(codes.InvalidArgument, "descriptor name prefix is too long")
	}
	if err := normalizeComponentSelector(req.Component); err != nil {
		return nil, err
	}

	snapshot := normalizeMetricTime(time.Now().UTC())
	where := []string{"d.created_at <= ?"}
	args := []any{metricTimeToDB(snapshot)}

	if req.NamePrefix != "" {
		where = append(where, "starts_with(d.name, ?)")
		args = append(args, req.NamePrefix)
	}
	if len(req.Kinds) > 0 {
		placeholders := make([]string, 0, len(req.Kinds))
		seenKinds := make(map[vmetricsv1.MetricDescriptor_Kind]struct{}, len(req.Kinds))
		for _, kind := range req.Kinds {
			if kind == vmetricsv1.MetricDescriptor_KIND_UNSET {
				return nil, status.Error(
					codes.InvalidArgument,
					"descriptor kind filter cannot contain KIND_UNSET",
				)
			}
			if _, ok := seenKinds[kind]; ok {
				return nil, status.Errorf(
					codes.InvalidArgument,
					"duplicate descriptor kind filter: %s",
					kind.String(),
				)
			}
			seenKinds[kind] = struct{}{}
			placeholders = append(placeholders, "?")
			args = append(args, kindToString(kind))
		}
		where = append(
			where,
			"d.kind IN ("+strings.Join(placeholders, ",")+")",
		)
	}

	componentWhere, componentArgs := descriptorComponentSQL(
		req.Component,
		snapshot,
	)
	where = append(where, componentWhere...)
	args = append(args, componentArgs...)

	query := `
SELECT
	d.id,
	d.name,
	d.kind,
	d.number_value_type,
	d.unit,
	d.description,
	d.temporality,
	d.scope_name,
	d.scope_version,
	d.scope_schema_url,
	CAST(d.explicit_bounds AS VARCHAR),
	d.exp_min_scale,
	d.exp_max_scale,
	d.exp_zero_threshold_min,
	d.exp_zero_threshold_max
FROM metric_descriptors d
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY d.name ASC, d.id ASC
LIMIT ?
`
	args = append(args, maxDescriptorResults+1)

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*vmetricsv1.MetricDescriptor, 0)
	for rows.Next() {
		descriptor, err := scanMetricDescriptor(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, descriptor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(items) > maxDescriptorResults {
		return nil, status.Error(
			codes.ResourceExhausted,
			"too many metric descriptors; narrow the request",
		)
	}

	if err := s.populateDescriptorAttributes(
		ctx,
		items,
		req.Component,
	); err != nil {
		return nil, err
	}

	return &vmetricsv1.ListMetricDescriptorsResponse{
		Items: items,
	}, nil
}

func descriptorComponentSQL(component *vmetricsv1.ComponentSelector, snapshot time.Time) ([]string, []any) {
	if component == nil || (component.Type == "" && component.Namespace == "" && component.Name == "") {
		return nil, nil
	}

	where := []string{"EXISTS (SELECT 1 FROM metric_series s WHERE s.descriptor_id = d.id AND s.created_at <= ?"}
	args := []any{metricTimeToDB(snapshot)}
	if component.Type != "" {
		where[0] += " AND s.component_type = ?"
		args = append(args, component.Type)
	}
	if component.Namespace != "" {
		where[0] += " AND s.component_namespace = ?"
		args = append(args, component.Namespace)
	}
	if component.Name != "" {
		where[0] += " AND s.component_name = ?"
		args = append(args, component.Name)
	}
	where[0] += ")"
	return where, args
}

func scanMetricDescriptor(scanner interface{ Scan(...any) error }) (*vmetricsv1.MetricDescriptor, error) {
	var id, name, kind, numberValueType, unit, description, temporality string
	var scopeName, scopeVersion, scopeSchemaURL string
	var boundsJSON sql.NullString
	var expMinScale, expMaxScale sql.NullInt64
	var expZeroMin, expZeroMax sql.NullFloat64

	if err := scanner.Scan(&id, &name, &kind, &numberValueType, &unit, &description, &temporality,
		&scopeName, &scopeVersion, &scopeSchemaURL, &boundsJSON, &expMinScale, &expMaxScale,
		&expZeroMin, &expZeroMax); err != nil {
		return nil, err
	}

	ret := &vmetricsv1.MetricDescriptor{
		Id:              id,
		Name:            name,
		Kind:            kindFromString(kind),
		NumberValueType: numberValueTypeFromString(numberValueType),
		Unit:            unit,
		Description:     description,
		Temporality:     temporalityFromString(temporality),
		InstrumentationScope: &vmetricsv1.InstrumentationScope{
			Name:      scopeName,
			Version:   scopeVersion,
			SchemaURL: scopeSchemaURL,
		},
	}

	switch ret.Kind {
	case vmetricsv1.MetricDescriptor_HISTOGRAM:
		var bounds []float64
		if boundsJSON.Valid && boundsJSON.String != "" {
			if err := json.Unmarshal([]byte(boundsJSON.String), &bounds); err != nil {
				return nil, err
			}
		}
		ret.Histogram = &vmetricsv1.MetricDescriptor_ExplicitHistogram{
			ExplicitHistogram: &vmetricsv1.ExplicitHistogramDescriptor{
				Bounds:         bounds,
				MergeSupported: true,
			},
		}

	case vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM:
		mergeSupported := expMinScale.Valid && expMaxScale.Valid && expMinScale.Int64 == expMaxScale.Int64 &&
			expZeroMin.Valid && expZeroMax.Valid && expZeroMin.Float64 == expZeroMax.Float64
		reason := ""
		if !mergeSupported {
			reason = "exponential histogram series use incompatible scales or zero thresholds"
		}
		explicit := &vmetricsv1.ExponentialHistogramDescriptor{
			MergeSupported:         mergeSupported,
			MergeUnsupportedReason: reason,
		}
		if expMinScale.Valid {
			explicit.MinimumObservedScale = int32(expMinScale.Int64)
		}
		if expMaxScale.Valid {
			explicit.MaximumObservedScale = int32(expMaxScale.Int64)
		}
		ret.Histogram = &vmetricsv1.MetricDescriptor_ExponentialHistogram{
			ExponentialHistogram: explicit,
		}
	}

	return ret, nil
}

func (s *srvMetric) populateDescriptorAttributes(ctx context.Context,
	descriptors []*vmetricsv1.MetricDescriptor, component *vmetricsv1.ComponentSelector) error {
	if len(descriptors) == 0 {
		return nil
	}

	byID := make(map[string]*vmetricsv1.MetricDescriptor, len(descriptors))
	placeholders := make([]string, 0, len(descriptors))
	args := []any{}
	for _, descriptor := range descriptors {
		byID[descriptor.Id] = descriptor
		placeholders = append(placeholders, "?")
		args = append(args, descriptor.Id)
	}

	where := []string{
		"a.descriptor_id IN (" + strings.Join(placeholders, ",") + ")",
	}
	if component != nil {
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

	query := `
SELECT
	a.descriptor_id,
	a.key,
	any_value(a.value_kind) AS value_kind,
	bit_or(a.source_mask) AS source_mask,
	approx_count_distinct(a.value_key) AS distinct_values
FROM metric_series_attributes a
JOIN metric_series s ON s.id = a.series_id
WHERE ` + strings.Join(where, " AND ") + `
GROUP BY a.descriptor_id, a.key
ORDER BY a.descriptor_id, a.key
`

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	now := time.Now().UTC()
	for rows.Next() {
		var descriptorID, key, valueKind string
		var sourceMask int
		var distinct uint64
		if err := rows.Scan(&descriptorID, &key, &valueKind, &sourceMask, &distinct); err != nil {
			return err
		}

		descriptor := byID[descriptorID]
		if descriptor == nil {
			continue
		}

		filterable, filterReason := metricAttributeFilterCapability(key)
		groupable, groupReason := metricAttributeGroupCapability(key, distinct)
		distinctCopy := distinct
		descriptor.Attributes = append(descriptor.Attributes, &vmetricsv1.MetricAttributeDescriptor{
			Key:                     key,
			ValueKind:               attributeKindToProto(valueKind),
			Sources:                 attributeSources(sourceMask),
			Filterable:              filterable,
			FilterUnsupportedReason: filterReason,
			Groupable:               groupable,
			GroupUnsupportedReason:  groupReason,
			EstimatedDistinctValues: &distinctCopy,
			EstimateAsOf:            pbutils.Timestamp(now),
			EstimateLookback:        durationPB(attributeEstimateLookback),
		})
	}
	return rows.Err()
}

func attributeSources(mask int) []vmetricsv1.MetricAttributeDescriptor_Source {
	var ret []vmetricsv1.MetricAttributeDescriptor_Source
	if mask&attributeSourceResource != 0 {
		ret = append(ret, vmetricsv1.MetricAttributeDescriptor_RESOURCE)
	}
	if mask&attributeSourceScope != 0 {
		ret = append(ret, vmetricsv1.MetricAttributeDescriptor_SCOPE)
	}
	if mask&attributeSourceDataPoint != 0 {
		ret = append(ret, vmetricsv1.MetricAttributeDescriptor_DATA_POINT)
	}
	return ret
}

func metricAttributeFilterCapability(_ string) (bool, string) {
	return true, ""
}

func metricAttributeGroupCapability(_ string, _ uint64) (bool, string) {
	return true, ""
}

func (s *srvMetric) resolveDescriptor(ctx context.Context, q *querySpec) (*vmetricsv1.MetricDescriptor, error) {
	where := []string{"1 = 1"}
	args := []any{}

	if descriptorID := strings.TrimSpace(q.req.Metric.GetDescriptorID()); descriptorID != "" {
		where = append(where, "d.id = ?")
		args = append(args, descriptorID)
	} else {
		where = append(where, "d.name = ?")
		args = append(args, strings.TrimSpace(q.req.Metric.GetName()))
	}

	activeSQL, activeArgs := descriptorActiveSeriesSQL("d", q.req.Component, q.from, q.to, q.snapshot)
	where = append(where, activeSQL)
	args = append(args, activeArgs...)

	query := `
SELECT
	d.id, d.name, d.kind, d.number_value_type, d.unit, d.description, d.temporality,
	d.scope_name, d.scope_version, d.scope_schema_url, CAST(d.explicit_bounds AS VARCHAR),
	d.exp_min_scale, d.exp_max_scale, d.exp_zero_threshold_min, d.exp_zero_threshold_max
FROM metric_descriptors d
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY d.id
LIMIT 2
`

	rows, err := s.s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descriptors []*vmetricsv1.MetricDescriptor
	for rows.Next() {
		descriptor, err := scanMetricDescriptor(rows)
		if err != nil {
			return nil, err
		}
		descriptors = append(descriptors, descriptor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(descriptors) == 0 {
		return nil, status.Error(codes.NotFound, "metric descriptor not found")
	}
	if len(descriptors) > 1 {
		return nil, status.Error(codes.FailedPrecondition,
			"metric name resolves to multiple incompatible descriptors; query by descriptorID")
	}
	if err := s.populateDescriptorAttributes(ctx, descriptors, q.req.Component); err != nil {
		return nil, err
	}
	return descriptors[0], nil
}

func descriptorActiveSeriesSQL(alias string, component *vmetricsv1.ComponentSelector,
	from, to, snapshot time.Time) (string, []any) {
	componentSQL := ""
	args := []any{}
	if component != nil {
		if component.Type != "" {
			componentSQL += " AND s.component_type = ?"
			args = append(args, component.Type)
		}
		if component.Namespace != "" {
			componentSQL += " AND s.component_namespace = ?"
			args = append(args, component.Namespace)
		}
		if component.Name != "" {
			componentSQL += " AND s.component_name = ?"
			args = append(args, component.Name)
		}
	}

	query := fmt.Sprintf(`EXISTS (
	SELECT 1
	FROM metric_series s
	WHERE s.descriptor_id = %[1]s.id%[2]s
	  AND (
		(%[1]s.kind IN ('COUNTER', 'UP_DOWN_COUNTER', 'GAUGE') AND EXISTS (
			SELECT 1 FROM metric_number_points p
			WHERE p.series_id = s.id AND p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
		)) OR
		(%[1]s.kind = 'HISTOGRAM' AND EXISTS (
			SELECT 1 FROM metric_histogram_points p
			WHERE p.series_id = s.id AND p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
		)) OR
		(%[1]s.kind = 'EXPONENTIAL_HISTOGRAM' AND EXISTS (
			SELECT 1 FROM metric_exponential_histogram_points p
			WHERE p.series_id = s.id AND p.timestamp >= ? AND p.timestamp < ? AND p.ingested_at <= ?
		))
	  )
)`, alias, componentSQL)
	args = append(args, metricTimeToDB(from), metricTimeToDB(to), metricTimeToDB(snapshot),
		metricTimeToDB(from), metricTimeToDB(to), metricTimeToDB(snapshot),
		metricTimeToDB(from), metricTimeToDB(to), metricTimeToDB(snapshot))
	return query, args
}

func (s *srvMetric) ListMetricCatalog(ctx context.Context,
	req *vmetricsv1.ListMetricCatalogRequest) (*vmetricsv1.ListMetricCatalogResponse, error) {
	_ = ctx
	var items []*vmetricsv1.MetricCatalogItem

	items = append(items,
		&vmetricsv1.MetricCatalogItem{
			Id:          "process_goroutines",
			DisplayName: "Goroutines",
			Description: "Current number of goroutines across selected components",
			Metric: &vmetricsv1.MetricSelector{
				Selector: &vmetricsv1.MetricSelector_Name{Name: "process.goroutines"},
				Kind:     vmetricsv1.MetricDescriptor_GAUGE,
			},
			DefaultOperation: &vmetricsv1.QueryOperation{
				Type: &vmetricsv1.QueryOperation_Gauge{Gauge: &vmetricsv1.GaugeOperation{
					Function: vmetricsv1.GaugeOperation_LAST,
				}},
			},
			DefaultGroupBy:           []string{"octelium.component.type", "octelium.component.namespace"},
			Unit:                     "goroutines",
			DefaultSeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
			DefaultStep:              durationPB(time.Minute),
		},
		&vmetricsv1.MetricCatalogItem{
			Id:          "process_heap_alloc",
			DisplayName: "Heap allocation",
			Description: "Current heap allocation across selected components",
			Metric: &vmetricsv1.MetricSelector{
				Selector: &vmetricsv1.MetricSelector_Name{Name: "process.mem.heap_alloc"},
				Kind:     vmetricsv1.MetricDescriptor_GAUGE,
			},
			DefaultOperation: &vmetricsv1.QueryOperation{
				Type: &vmetricsv1.QueryOperation_Gauge{Gauge: &vmetricsv1.GaugeOperation{
					Function: vmetricsv1.GaugeOperation_LAST,
				}},
			},
			DefaultGroupBy:           []string{"octelium.component.type", "octelium.component.namespace"},
			Unit:                     "bytes",
			DefaultSeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
			DefaultStep:              durationPB(time.Minute),
		},
	)

	if req != nil && req.Component != nil &&
		catalogMatchesComponent(req, "rscserver", "vigil") {
		items = append(items,
			&vmetricsv1.MetricCatalogItem{
				Id:          "requests_rate",
				DisplayName: "Requests rate",
				Description: "Per-second request rate",
				Metric: &vmetricsv1.MetricSelector{
					Selector: &vmetricsv1.MetricSelector_Name{Name: "req.total"},
					Kind:     vmetricsv1.MetricDescriptor_COUNTER,
				},
				DefaultOperation: &vmetricsv1.QueryOperation{
					Type: &vmetricsv1.QueryOperation_Counter{Counter: &vmetricsv1.CounterOperation{
						Function: vmetricsv1.CounterOperation_RATE,
					}},
				},
				DefaultGroupBy:           []string{"octelium.component.type", "octelium.component.namespace"},
				Unit:                     "requests/s",
				DefaultSeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
				DefaultStep:              durationPB(time.Minute),
			},
			&vmetricsv1.MetricCatalogItem{
				Id:          "active_requests",
				DisplayName: "Active requests",
				Description: "Current active requests",
				Metric: &vmetricsv1.MetricSelector{
					Selector: &vmetricsv1.MetricSelector_Name{Name: "req.active"},
					Kind:     vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
				},
				DefaultOperation: &vmetricsv1.QueryOperation{
					Type: &vmetricsv1.QueryOperation_Gauge{Gauge: &vmetricsv1.GaugeOperation{
						Function: vmetricsv1.GaugeOperation_LAST,
					}},
				},
				DefaultGroupBy:           []string{"octelium.component.type", "octelium.component.namespace"},
				Unit:                     "requests",
				DefaultSeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
				DefaultStep:              durationPB(time.Minute),
			},
			&vmetricsv1.MetricCatalogItem{
				Id:          "request_p95_latency",
				DisplayName: "Request p95 latency",
				Description: "p95 request duration",
				Metric: &vmetricsv1.MetricSelector{
					Selector: &vmetricsv1.MetricSelector_Name{Name: "req.duration"},
					Kind:     vmetricsv1.MetricDescriptor_HISTOGRAM,
				},
				DefaultOperation: &vmetricsv1.QueryOperation{
					Type: &vmetricsv1.QueryOperation_Histogram{Histogram: &vmetricsv1.HistogramOperation{
						Function:  vmetricsv1.HistogramOperation_QUANTILE,
						Quantiles: []float64{0.95},
					}},
				},
				DefaultGroupBy:           []string{"octelium.component.type", "octelium.component.namespace"},
				Unit:                     requestDurationCatalogUnit(req),
				DefaultSeriesAggregation: vmetricsv1.QueryMetricsRequest_MERGE,
				DefaultStep:              durationPB(time.Minute),
			},
		)
	}

	if req != nil && req.Component != nil &&
		catalogMatchesComponent(req, "octovigil") {
		items = append(items,
			&vmetricsv1.MetricCatalogItem{
				Id:          "authorization_requests_rate",
				DisplayName: "Authorization rate",
				Description: "Per-second authorization request rate",
				Metric: &vmetricsv1.MetricSelector{
					Selector: &vmetricsv1.MetricSelector_Name{Name: "authorization.req.total"},
					Kind:     vmetricsv1.MetricDescriptor_COUNTER,
				},
				DefaultOperation: &vmetricsv1.QueryOperation{
					Type: &vmetricsv1.QueryOperation_Counter{Counter: &vmetricsv1.CounterOperation{
						Function: vmetricsv1.CounterOperation_RATE,
					}},
				},
				DefaultGroupBy:           []string{"octelium.component.name"},
				Unit:                     "requests/s",
				DefaultSeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
				DefaultStep:              durationPB(time.Minute),
			},
			&vmetricsv1.MetricCatalogItem{
				Id:          "active_authorization_requests",
				DisplayName: "Active authorizations",
				Description: "Current authorization requests",
				Metric: &vmetricsv1.MetricSelector{
					Selector: &vmetricsv1.MetricSelector_Name{Name: "authorization.req.active"},
					Kind:     vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
				},
				DefaultOperation: &vmetricsv1.QueryOperation{
					Type: &vmetricsv1.QueryOperation_Gauge{Gauge: &vmetricsv1.GaugeOperation{
						Function: vmetricsv1.GaugeOperation_LAST,
					}},
				},
				DefaultGroupBy:           []string{"octelium.component.name"},
				Unit:                     "requests",
				DefaultSeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
				DefaultStep:              durationPB(time.Minute),
			},
			&vmetricsv1.MetricCatalogItem{
				Id:          "authorization_p95_latency",
				DisplayName: "Authorization p95 latency",
				Description: "p95 authorization request duration",
				Metric: &vmetricsv1.MetricSelector{
					Selector: &vmetricsv1.MetricSelector_Name{Name: "authorization.req.duration"},
					Kind:     vmetricsv1.MetricDescriptor_HISTOGRAM,
				},
				DefaultOperation: &vmetricsv1.QueryOperation{
					Type: &vmetricsv1.QueryOperation_Histogram{Histogram: &vmetricsv1.HistogramOperation{
						Function:  vmetricsv1.HistogramOperation_QUANTILE,
						Quantiles: []float64{0.95},
					}},
				},
				DefaultGroupBy:           []string{"octelium.component.name"},
				Unit:                     "us",
				DefaultSeriesAggregation: vmetricsv1.QueryMetricsRequest_MERGE,
				DefaultStep:              durationPB(time.Minute),
			},
		)
	}

	return &vmetricsv1.ListMetricCatalogResponse{Items: items}, nil
}

func requestDurationCatalogUnit(req *vmetricsv1.ListMetricCatalogRequest) string {
	if req != nil && req.Component != nil && req.Component.Type == "rscserver" {
		return "us"
	}
	return "ms"
}

func catalogMatchesComponent(req *vmetricsv1.ListMetricCatalogRequest, componentTypes ...string) bool {
	if req == nil || req.Component == nil || req.Component.Type == "" {
		return true
	}
	for _, componentType := range componentTypes {
		if req.Component.Type == componentType {
			return true
		}
	}
	return false
}

func (s *srvMetric) GetMetricsCapabilities(ctx context.Context,
	req *vmetricsv1.GetMetricsCapabilitiesRequest) (*vmetricsv1.GetMetricsCapabilitiesResponse, error) {
	_ = ctx
	_ = req

	return &vmetricsv1.GetMetricsCapabilitiesResponse{
		QueryLimits: &vmetricsv1.QueryLimits{
			MaximumTimeRange:         durationPB(rawMetricRetention),
			MinimumStep:              durationPB(minimumQueryStep),
			MaximumSeries:            maxSeriesPerQuery,
			MaximumPointsPerSeries:   maxPointsPerSeries,
			MaximumTotalPoints:       maxTotalPoints,
			MaximumGroupByAttributes: maxGroupByAttributes,
			MaximumFilters:           maxFilters,
			MaximumFilterValues:      maxFilterValues,
			MaximumSourceSeries:      maximumSourceSeries,
			MaximumRawHistogramRows:  maximumRawHistogramRowsPerQuery,
			MaximumRawNumberRows:     maximumRawNumberRowsPerQuery,
		},
		RetentionTiers: []*vmetricsv1.MetricRetentionTier{
			{
				Name:      "raw",
				Retention: durationPB(rawMetricRetention),
				Raw:       true,
			},
		},
		ServerTime: pbutils.Now(),
		IngestionLimits: &vmetricsv1.MetricIngestionLimits{
			MaximumDataPointsPerExport:          maxDataPointsPerExport,
			MaximumEffectiveAttributesPerSeries: maxAttributesPerPoint,
			MaximumAttributeKeyBytes:            maxAttributeKeyBytes,
			MaximumAttributeValueBytes:          maxAttributeValueBytes,
			MaximumSeriesLabelsBytes:            maxSeriesLabelsBytes,
			MaximumHistogramBuckets:             maxHistogramBuckets,
			MaximumQueuedExports:                maxQueuedExports,
			MaximumQueuedDataPoints:             uint64(maxQueuedDataPoints),
			MaximumQueuedEstimatedBytes:         uint64(maxQueuedEstimatedBytes),
			MaximumGRPCMessageBytes:             8 << 20,
			MaximumFutureSkew:                   durationPB(maximumFutureSkew),
			AcceptedPastWindow:                  durationPB(maximumPastSkew),
		},
		MetricKinds: []*vmetricsv1.MetricKindCapability{
			{
				Kind: vmetricsv1.MetricDescriptor_COUNTER,
				CounterFunctions: []vmetricsv1.CounterOperation_Function{
					vmetricsv1.CounterOperation_RAW,
					vmetricsv1.CounterOperation_RATE,
					vmetricsv1.CounterOperation_INCREASE,
				},
				SeriesAggregations: numberSeriesAggregations(),
			},
			{
				Kind:               vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
				GaugeFunctions:     gaugeFunctions(),
				SeriesAggregations: numberSeriesAggregations(),
			},
			{
				Kind:               vmetricsv1.MetricDescriptor_GAUGE,
				GaugeFunctions:     gaugeFunctions(),
				SeriesAggregations: numberSeriesAggregations(),
			},
			{
				Kind:               vmetricsv1.MetricDescriptor_HISTOGRAM,
				HistogramFunctions: histogramFunctions(),
				SeriesAggregations: []vmetricsv1.QueryMetricsRequest_SeriesAggregation{
					vmetricsv1.QueryMetricsRequest_NONE,
					vmetricsv1.QueryMetricsRequest_MERGE,
				},
			},
			{
				Kind:               vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM,
				HistogramFunctions: histogramFunctions(),
				SeriesAggregations: []vmetricsv1.QueryMetricsRequest_SeriesAggregation{
					vmetricsv1.QueryMetricsRequest_NONE,
					vmetricsv1.QueryMetricsRequest_MERGE,
				},
			},
		},
	}, nil
}

func numberSeriesAggregations() []vmetricsv1.QueryMetricsRequest_SeriesAggregation {
	return []vmetricsv1.QueryMetricsRequest_SeriesAggregation{
		vmetricsv1.QueryMetricsRequest_NONE,
		vmetricsv1.QueryMetricsRequest_SUM,
		vmetricsv1.QueryMetricsRequest_AVG,
		vmetricsv1.QueryMetricsRequest_MIN,
		vmetricsv1.QueryMetricsRequest_MAX,
		vmetricsv1.QueryMetricsRequest_LAST,
	}
}

func gaugeFunctions() []vmetricsv1.GaugeOperation_Function {
	return []vmetricsv1.GaugeOperation_Function{
		vmetricsv1.GaugeOperation_LAST,
		vmetricsv1.GaugeOperation_AVG,
		vmetricsv1.GaugeOperation_MIN,
		vmetricsv1.GaugeOperation_MAX,
		vmetricsv1.GaugeOperation_SUM,
	}
}

func histogramFunctions() []vmetricsv1.HistogramOperation_Function {
	return []vmetricsv1.HistogramOperation_Function{
		vmetricsv1.HistogramOperation_BUCKETS,
		vmetricsv1.HistogramOperation_QUANTILE,
		vmetricsv1.HistogramOperation_AVG,
		vmetricsv1.HistogramOperation_COUNT,
		vmetricsv1.HistogramOperation_SUM,
		vmetricsv1.HistogramOperation_MIN,
		vmetricsv1.HistogramOperation_MAX,
	}
}
