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
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.uber.org/zap"
)

type srvMetric struct {
	pmetricotlp.UnimplementedGRPCServer
	visibilityv1.UnimplementedMetricsServiceServer
	s      *Server
	itemCh chan pmetric.Metrics
}

func (s *srvMetric) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {

	s.itemCh <- req.Metrics()

	return pmetricotlp.NewExportResponse(), nil
}

func (s *srvMetric) storeMetrics(m pmetric.Metrics) error {
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
					s.handleGauge(metric, sm, rm)

				case pmetric.MetricTypeSum:
					s.handleSum(metric, sm, rm)

				case pmetric.MetricTypeHistogram:
					s.handleHistogram(metric, sm, rm)

				case pmetric.MetricTypeSummary:
					s.handleSummary(metric, sm, rm)
				}
			}
		}
	}
	return nil
}

func (s *srvMetric) insertNumberPoint(ts time.Time, name, unit, mtype string,
	sm pmetric.ScopeMetrics, rm pmetric.ResourceMetrics, dp pmetric.NumberDataPoint, value float64) error {

	scopeAttrs := sm.Scope().Attributes().AsRaw()
	resourceAttrs := rm.Resource().Attributes().AsRaw()

	scopeJSON, _ := json.Marshal(scopeAttrs)
	resourceJSON, _ := json.Marshal(resourceAttrs)

	attrs, _ := json.Marshal(mergeMaps(mergeMaps(resourceAttrs, scopeAttrs), dp.Attributes().AsRaw()))

	componentType, _ := resourceAttrs["octelium.component.type"].(string)
	componentNamespace, _ := resourceAttrs["octelium.component.namespace"].(string)

	_, err := s.s.db.Exec(`
        INSERT INTO metrics (
            timestamp, name, unit, metric_type,
            resource, scope, attributes, value,
			component_type, component_namespace
        )
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
		ts, name, unit, mtype,
		string(resourceJSON), string(scopeJSON), string(attrs), value,
		componentType, componentNamespace)

	return err
}

func (s *srvMetric) handleGauge(metric pmetric.Metric, sm pmetric.ScopeMetrics, rm pmetric.ResourceMetrics) {
	dp := metric.Gauge().DataPoints()
	for i := 0; i < dp.Len(); i++ {
		s.writeNumberDP(dp.At(i), metric, sm, rm)
	}
}

func (s *srvMetric) handleSum(metric pmetric.Metric, sm pmetric.ScopeMetrics, rm pmetric.ResourceMetrics) {
	dp := metric.Sum().DataPoints()
	for i := 0; i < dp.Len(); i++ {
		s.writeNumberDP(dp.At(i), metric, sm, rm)
	}
}

func (s *srvMetric) handleHistogram(metric pmetric.Metric, sm pmetric.ScopeMetrics, rm pmetric.ResourceMetrics) {
	dps := metric.Histogram().DataPoints()

	scopeAttrs := sm.Scope().Attributes().AsRaw()
	resourceAttrs := rm.Resource().Attributes().AsRaw()

	scopeJSON, _ := json.Marshal(scopeAttrs)
	resourceJSON, _ := json.Marshal(resourceAttrs)

	componentType, _ := resourceAttrs["octelium.component.type"].(string)
	componentNamespace, _ := resourceAttrs["octelium.component.namespace"].(string)

	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)

		ts := dp.Timestamp().AsTime()
		attrs, _ := json.Marshal(mergeMaps(mergeMaps(resourceAttrs, scopeAttrs), dp.Attributes().AsRaw()))
		bounds, _ := json.Marshal(dp.ExplicitBounds().AsRaw())
		buckets, _ := json.Marshal(dp.BucketCounts().AsRaw())

		_, _ = s.s.db.Exec(`
            INSERT INTO metrics (
                timestamp, name, unit, metric_type,
                resource, scope, attributes,
                histogram_count, histogram_sum,
                histogram_bounds, histogram_bucket_counts,
				component_type, component_namespace
            )
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `,
			ts,
			metric.Name(),
			metric.Unit(),
			metric.Type().String(),
			string(resourceJSON),
			string(scopeJSON),
			string(attrs),

			dp.Count(),
			dp.Sum(),

			string(bounds),
			string(buckets),
			componentType, componentNamespace,
		)
	}
}

func (s *srvMetric) handleSummary(metric pmetric.Metric, sm pmetric.ScopeMetrics, rm pmetric.ResourceMetrics) {
	dps := metric.Summary().DataPoints()
	scopeJSON, _ := json.Marshal(sm.Scope().Attributes().AsRaw())
	resourceJSON, _ := json.Marshal(rm.Resource().Attributes().AsRaw())

	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)

		ts := dp.Timestamp().AsTime()
		attrs, _ := json.Marshal(dp.Attributes().AsRaw())

		quantiles := make([]map[string]float64, dp.QuantileValues().Len())
		for j := 0; j < dp.QuantileValues().Len(); j++ {
			qv := dp.QuantileValues().At(j)
			quantiles[j] = map[string]float64{
				"quantile": qv.Quantile(),
				"value":    qv.Value(),
			}
		}
		quantilesJSON, _ := json.Marshal(quantiles)

		_, _ = s.s.db.Exec(`
            INSERT INTO metrics (
                timestamp, name, unit, metric_type,
                resource, scope, attributes,
                summary_count, summary_sum, summary_quantiles
            )
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `,
			ts,
			metric.Name(),
			metric.Unit(),
			metric.Type().String(),
			string(resourceJSON),
			string(scopeJSON),
			string(attrs),

			dp.Count(),
			dp.Sum(),
			string(quantilesJSON),
		)
	}
}

func (s *srvMetric) writeNumberDP(dp pmetric.NumberDataPoint, metric pmetric.Metric,
	sm pmetric.ScopeMetrics, rm pmetric.ResourceMetrics) {

	ts := dp.Timestamp().AsTime()

	// zap.L().Debug("Doing writeNumberDP", zap.Any("dp", dp), zap.Any("sm", sm), zap.Any("rm", rm))

	var value float64
	if dp.ValueType() == pmetric.NumberDataPointValueTypeInt {
		value = float64(dp.IntValue())
	} else {
		value = dp.DoubleValue()
	}

	if err := s.insertNumberPoint(ts, metric.Name(), metric.Unit(), metric.Type().String(),
		sm, rm, dp, value); err != nil {
		zap.L().Warn("Could not insertNumberPoint", zap.Error(err))
	}
}

func (s *srvMetric) startProcessLoop(ctx context.Context) {
	defer zap.L().Debug("Exiting process loop")

	for {
		select {
		case <-ctx.Done():
			return
		case m := <-s.itemCh:
			s.storeMetrics(m)
		}
	}
}

func (s *Server) newSrvMetric() *srvMetric {
	return &srvMetric{
		s:      s,
		itemCh: make(chan pmetric.Metrics, 10000),
	}
}

func mergeMaps(dst, src map[string]any) map[string]any {
	for key, srcValue := range src {
		if dstValue, ok := dst[key]; ok {
			if nestedDst, ok := dstValue.(map[string]any); ok {
				if nestedSrc, ok := srcValue.(map[string]any); ok {
					dst[key] = mergeMaps(nestedDst, nestedSrc)
					continue
				}
			}
		}
		dst[key] = srcValue
	}
	return dst
}
