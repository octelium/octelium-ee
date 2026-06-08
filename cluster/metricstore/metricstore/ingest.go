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
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type srvMetric struct {
	pmetricotlp.UnimplementedGRPCServer
	vmetricsv1.UnimplementedMetricsServiceServer

	s      *Server
	itemCh chan pmetric.Metrics
}

func (s *Server) newSrvMetric() *srvMetric {
	return &srvMetric{
		s:      s,
		itemCh: make(chan pmetric.Metrics, 10000),
	}
}

func (s *srvMetric) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	md := pmetric.NewMetrics()
	req.Metrics().CopyTo(md)

	select {
	case s.itemCh <- md:
		return pmetricotlp.NewExportResponse(), nil
	case <-ctx.Done():
		return pmetricotlp.NewExportResponse(), ctx.Err()
	default:
		return pmetricotlp.NewExportResponse(), status.Error(codes.Unavailable, "metricstore ingestion queue is full")
	}
}

func (s *srvMetric) startProcessLoop(ctx context.Context) {
	defer zap.L().Debug("Exiting metricstore process loop")

	for {
		select {
		case <-ctx.Done():
			return
		case m := <-s.itemCh:
			if err := s.storeMetrics(ctx, m); err != nil {
				zap.L().Warn("Could not store metrics", zap.Error(err))
			}
		}
	}
}

func (s *srvMetric) storeMetrics(ctx context.Context, m pmetric.Metrics) error {
	tx, err := s.s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, insertMetricSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rms := m.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)

		sms := rm.ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			sm := sms.At(j)

			metrics := sm.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)

				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					if err := s.handleGauge(ctx, stmt, metric, sm, rm); err != nil {
						return err
					}

				case pmetric.MetricTypeSum:
					if err := s.handleSum(ctx, stmt, metric, sm, rm); err != nil {
						return err
					}

				case pmetric.MetricTypeHistogram:
					if err := s.handleHistogram(ctx, stmt, metric, sm, rm); err != nil {
						return err
					}

				case pmetric.MetricTypeExponentialHistogram:
					if err := s.handleExponentialHistogram(ctx, stmt, metric, sm, rm); err != nil {
						return err
					}

				case pmetric.MetricTypeSummary:
					zap.L().Debug("Dropping unsupported summary metric", zap.String("name", metric.Name()))

				default:
					zap.L().Debug("Dropping unsupported metric type",
						zap.String("name", metric.Name()),
						zap.String("type", metric.Type().String()))
				}
			}
		}
	}

	return tx.Commit()
}

const insertMetricSQL = `
INSERT INTO metrics (
    timestamp,
    name,
    unit,
    description,
    kind,
    value_type,
    temporality,
    resource,
    scope,
    attributes,
    component_type,
    component_namespace,
    component_name,
    number_int,
    number_double,
    histogram_count,
    histogram_has_sum,
    histogram_sum,
    histogram_min,
    histogram_max,
    histogram_bounds,
    histogram_bucket_counts,
    exp_count,
    exp_has_sum,
    exp_sum,
    exp_min,
    exp_max,
    exp_scale,
    exp_zero_count,
    exp_zero_threshold,
    exp_positive_offset,
    exp_positive_counts,
    exp_negative_offset,
    exp_negative_counts
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

func (s *srvMetric) handleGauge(
	ctx context.Context,
	stmt *sql.Stmt,
	metric pmetric.Metric,
	sm pmetric.ScopeMetrics,
	rm pmetric.ResourceMetrics,
) error {
	dps := metric.Gauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if err := s.insertNumberPoint(ctx, stmt, metric, sm, rm, dp, "GAUGE", ""); err != nil {
			return err
		}
	}

	return nil
}

func (s *srvMetric) handleSum(
	ctx context.Context,
	stmt *sql.Stmt,
	metric pmetric.Metric,
	sm pmetric.ScopeMetrics,
	rm pmetric.ResourceMetrics,
) error {
	sum := metric.Sum()

	kind := "UP_DOWN_COUNTER"
	if sum.IsMonotonic() {
		kind = "COUNTER"
	}

	temporality := temporalityToString(sum.AggregationTemporality())

	dps := sum.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if err := s.insertNumberPoint(ctx, stmt, metric, sm, rm, dp, kind, temporality); err != nil {
			return err
		}
	}

	return nil
}

func (s *srvMetric) insertNumberPoint(
	ctx context.Context,
	stmt *sql.Stmt,
	metric pmetric.Metric,
	sm pmetric.ScopeMetrics,
	rm pmetric.ResourceMetrics,
	dp pmetric.NumberDataPoint,
	kind string,
	temporality string,
) error {
	resourceAttrs := rm.Resource().Attributes().AsRaw()
	scopeAttrs := sm.Scope().Attributes().AsRaw()
	pointAttrs := dp.Attributes().AsRaw()
	attrs := mergeAttrs(resourceAttrs, scopeAttrs, pointAttrs)

	resourceJSON, err := json.Marshal(resourceAttrs)
	if err != nil {
		return err
	}

	scopeJSON, err := json.Marshal(scopeAttrs)
	if err != nil {
		return err
	}

	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		return err
	}

	componentType, _ := resourceAttrs["octelium.component.type"].(string)
	componentNamespace, _ := resourceAttrs["octelium.component.namespace"].(string)
	componentName, _ := resourceAttrs["octelium.component.name"].(string)

	ts := timestampOrNow(dp.Timestamp())

	var numberInt any
	var numberDouble any
	valueType := "DOUBLE"

	switch dp.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		valueType = "INT64"
		numberInt = dp.IntValue()
	case pmetric.NumberDataPointValueTypeDouble:
		numberDouble = dp.DoubleValue()
	default:
		return nil
	}

	_, err = stmt.ExecContext(ctx,
		ts,
		metric.Name(),
		metric.Unit(),
		metric.Description(),
		kind,
		valueType,
		temporality,
		string(resourceJSON),
		string(scopeJSON),
		string(attrsJSON),
		componentType,
		componentNamespace,
		componentName,

		numberInt,
		numberDouble,

		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,

		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	return err
}

func (s *srvMetric) handleHistogram(
	ctx context.Context,
	stmt *sql.Stmt,
	metric pmetric.Metric,
	sm pmetric.ScopeMetrics,
	rm pmetric.ResourceMetrics,
) error {
	h := metric.Histogram()
	temporality := temporalityToString(h.AggregationTemporality())

	resourceAttrs := rm.Resource().Attributes().AsRaw()
	scopeAttrs := sm.Scope().Attributes().AsRaw()

	resourceJSON, err := json.Marshal(resourceAttrs)
	if err != nil {
		return err
	}

	scopeJSON, err := json.Marshal(scopeAttrs)
	if err != nil {
		return err
	}

	componentType, _ := resourceAttrs["octelium.component.type"].(string)
	componentNamespace, _ := resourceAttrs["octelium.component.namespace"].(string)
	componentName, _ := resourceAttrs["octelium.component.name"].(string)

	dps := h.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)

		attrs := mergeAttrs(resourceAttrs, scopeAttrs, dp.Attributes().AsRaw())
		attrsJSON, err := json.Marshal(attrs)
		if err != nil {
			return err
		}

		boundsJSON, err := json.Marshal(dp.ExplicitBounds().AsRaw())
		if err != nil {
			return err
		}

		bucketCountsJSON, err := json.Marshal(dp.BucketCounts().AsRaw())
		if err != nil {
			return err
		}

		var sumVal any
		histogramHasSum := dp.HasSum()
		if histogramHasSum {
			sumVal = dp.Sum()
		}

		var minVal any
		if dp.HasMin() {
			minVal = dp.Min()
		}

		var maxVal any
		if dp.HasMax() {
			maxVal = dp.Max()
		}

		_, err = stmt.ExecContext(ctx,
			timestampOrNow(dp.Timestamp()),
			metric.Name(),
			metric.Unit(),
			metric.Description(),
			"HISTOGRAM",
			"DOUBLE",
			temporality,
			string(resourceJSON),
			string(scopeJSON),
			string(attrsJSON),
			componentType,
			componentNamespace,
			componentName,

			nil,
			nil,

			dp.Count(),
			histogramHasSum,
			sumVal,
			minVal,
			maxVal,
			string(boundsJSON),
			string(bucketCountsJSON),

			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *srvMetric) handleExponentialHistogram(
	ctx context.Context,
	stmt *sql.Stmt,
	metric pmetric.Metric,
	sm pmetric.ScopeMetrics,
	rm pmetric.ResourceMetrics,
) error {
	h := metric.ExponentialHistogram()
	temporality := temporalityToString(h.AggregationTemporality())

	resourceAttrs := rm.Resource().Attributes().AsRaw()
	scopeAttrs := sm.Scope().Attributes().AsRaw()

	resourceJSON, err := json.Marshal(resourceAttrs)
	if err != nil {
		return err
	}

	scopeJSON, err := json.Marshal(scopeAttrs)
	if err != nil {
		return err
	}

	componentType, _ := resourceAttrs["octelium.component.type"].(string)
	componentNamespace, _ := resourceAttrs["octelium.component.namespace"].(string)
	componentName, _ := resourceAttrs["octelium.component.name"].(string)

	dps := h.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)

		attrs := mergeAttrs(resourceAttrs, scopeAttrs, dp.Attributes().AsRaw())
		attrsJSON, err := json.Marshal(attrs)
		if err != nil {
			return err
		}

		posCountsJSON, err := json.Marshal(dp.Positive().BucketCounts().AsRaw())
		if err != nil {
			return err
		}

		negCountsJSON, err := json.Marshal(dp.Negative().BucketCounts().AsRaw())
		if err != nil {
			return err
		}

		var sumVal any
		expHasSum := dp.HasSum()
		if expHasSum {
			sumVal = dp.Sum()
		}

		var minVal any
		if dp.HasMin() {
			minVal = dp.Min()
		}

		var maxVal any
		if dp.HasMax() {
			maxVal = dp.Max()
		}

		_, err = stmt.ExecContext(ctx,
			timestampOrNow(dp.Timestamp()),
			metric.Name(),
			metric.Unit(),
			metric.Description(),
			"EXPONENTIAL_HISTOGRAM",
			"DOUBLE",
			temporality,
			string(resourceJSON),
			string(scopeJSON),
			string(attrsJSON),
			componentType,
			componentNamespace,
			componentName,

			nil,
			nil,

			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,

			dp.Count(),
			expHasSum,
			sumVal,
			minVal,
			maxVal,
			dp.Scale(),
			dp.ZeroCount(),
			dp.ZeroThreshold(),
			dp.Positive().Offset(),
			string(posCountsJSON),
			dp.Negative().Offset(),
			string(negCountsJSON),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func timestampOrNow(ts pcommon.Timestamp) time.Time {
	if ts == 0 {
		return time.Now().UTC()
	}
	return ts.AsTime().UTC()
}
