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
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxQueuedExports        = 32
	maxQueuedDataPoints     = int64(200000)
	maxQueuedEstimatedBytes = int64(128 << 20)
	maxDataPointsPerExport  = 50000
	maxAttributesPerPoint   = 64
	maxAttributeKeyBytes    = 255
	maxAttributeValueBytes  = 4096
	maxSeriesLabelsBytes    = 64 << 10
	maxHistogramBuckets     = 512
	maxMetricNameLength     = 255
	maxMetricUnitLength     = 64
	maxMetricDescriptionLen = 1024
	maximumFutureSkew       = 10 * time.Minute
	maximumPastSkew         = rawMetricRetention + 10*time.Minute

	maxIngestBatchExports = 8
	maxIngestBatchPoints  = 100000
	ingestBatchDelay      = 20 * time.Millisecond
	ingestCommitTimeout   = 2 * time.Minute
)

type srvMetric struct {
	pmetricotlp.UnimplementedGRPCServer
	vmetricsv1.UnimplementedMetricsServiceServer

	s *Server

	itemCh chan *pendingMetricExport
	budget queueBudget

	accepting atomic.Bool
	closeOnce sync.Once

	acceptedExports atomic.Uint64
	rejectedExports atomic.Uint64
	storedPoints    atomic.Uint64
}

type pendingMetricExport struct {
	ctx            context.Context
	metrics        pmetric.Metrics
	dataPoints     int64
	estimatedBytes int64
	done           chan error
}

type queueBudget struct {
	mu sync.Mutex

	requests int64
	points   int64
	bytes    int64
}

func (b *queueBudget) reserve(points, bytes int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.requests+1 > maxQueuedExports || b.points+points > maxQueuedDataPoints || b.bytes+bytes > maxQueuedEstimatedBytes {
		return false
	}

	b.requests++
	b.points += points
	b.bytes += bytes
	return true
}

func (b *queueBudget) release(points, bytes int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.requests--
	b.points -= points
	b.bytes -= bytes
	if b.requests < 0 {
		b.requests = 0
	}
	if b.points < 0 {
		b.points = 0
	}
	if b.bytes < 0 {
		b.bytes = 0
	}
}

func (s *Server) newSrvMetric() *srvMetric {
	ret := &srvMetric{
		s:      s,
		itemCh: make(chan *pendingMetricExport, maxQueuedExports),
	}
	ret.accepting.Store(true)
	return ret
}

func (s *srvMetric) stopAccepting() {
	s.accepting.Store(false)
}

func (s *srvMetric) closeQueue() {
	s.closeOnce.Do(func() {
		close(s.itemCh)
	})
}

func (s *srvMetric) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	if !s.accepting.Load() {
		return pmetricotlp.NewExportResponse(), status.Error(codes.Unavailable, "metricstore is shutting down")
	}

	points, estimatedBytes, err := inspectMetrics(req.Metrics())
	if err != nil {
		s.rejectedExports.Add(1)
		return pmetricotlp.NewExportResponse(), err
	}

	if !s.budget.reserve(points, estimatedBytes) {
		s.rejectedExports.Add(1)
		return pmetricotlp.NewExportResponse(), status.Error(codes.Unavailable, "metricstore ingestion queue is full")
	}

	metrics := pmetric.NewMetrics()
	req.Metrics().CopyTo(metrics)

	item := &pendingMetricExport{
		ctx:            ctx,
		metrics:        metrics,
		dataPoints:     points,
		estimatedBytes: estimatedBytes,
		done:           make(chan error, 1),
	}

	select {
	case s.itemCh <- item:
	case <-ctx.Done():
		s.budget.release(points, estimatedBytes)
		return pmetricotlp.NewExportResponse(), ctx.Err()
	}

	select {
	case err := <-item.done:
		if err != nil {
			s.rejectedExports.Add(1)
			return pmetricotlp.NewExportResponse(), err
		}
		s.acceptedExports.Add(1)
		return pmetricotlp.NewExportResponse(), nil

	case <-ctx.Done():
		return pmetricotlp.NewExportResponse(), ctx.Err()
	}
}

func inspectMetrics(metrics pmetric.Metrics) (int64, int64, error) {
	var points int64
	var estimatedBytes int64
	now := time.Now().UTC()

	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		resourceMetric := resourceMetrics.At(i)
		resourceSize, err := inspectAttributeMap(resourceMetric.Resource().Attributes())
		if err != nil {
			return 0, 0, err
		}

		scopeMetrics := resourceMetric.ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			scopeMetric := scopeMetrics.At(j)
			scopeSize, err := inspectAttributeMap(scopeMetric.Scope().Attributes())
			if err != nil {
				return 0, 0, err
			}

			metricsSlice := scopeMetric.Metrics()
			for k := 0; k < metricsSlice.Len(); k++ {
				metric := metricsSlice.At(k)
				if len(metric.Name()) == 0 || len(metric.Name()) > maxMetricNameLength {
					return 0, 0, status.Error(codes.InvalidArgument, "invalid metric name length")
				}
				if len(metric.Unit()) > maxMetricUnitLength || len(metric.Description()) > maxMetricDescriptionLen {
					return 0, 0, status.Error(codes.InvalidArgument, "metric metadata is too large")
				}

				baseSize := int64(len(metric.Name())+len(metric.Unit())+len(metric.Description())) + resourceSize + scopeSize

				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					dps := metric.Gauge().DataPoints()
					for idx := 0; idx < dps.Len(); idx++ {
						dp := dps.At(idx)
						if err := validateNumberPointValue(dp, false); err != nil {
							return 0, 0, err
						}
						if err := validateMetricPointTimes(dp.Timestamp(), dp.StartTimestamp(), now); err != nil {
							return 0, 0, err
						}
						attrSize, err := inspectAttributeMap(dp.Attributes())
						if err != nil {
							return 0, 0, err
						}
						points++
						estimatedBytes += baseSize + attrSize + 192
					}

				case pmetric.MetricTypeSum:
					sum := metric.Sum()
					if !isSupportedAggregationTemporality(sum.AggregationTemporality()) {
						return 0, 0, status.Error(codes.InvalidArgument, "sum aggregation temporality must be DELTA or CUMULATIVE")
					}
					if !sum.IsMonotonic() && sum.AggregationTemporality() == pmetric.AggregationTemporalityDelta {
						return 0, 0, status.Error(codes.InvalidArgument,
							"delta non-monotonic sums are not accepted; normalize them to cumulative state before export")
					}
					dps := sum.DataPoints()
					for idx := 0; idx < dps.Len(); idx++ {
						dp := dps.At(idx)
						if err := validateNumberPointValue(dp, sum.IsMonotonic()); err != nil {
							return 0, 0, err
						}
						if err := validateMetricPointTimes(dp.Timestamp(), dp.StartTimestamp(), now); err != nil {
							return 0, 0, err
						}
						attrSize, err := inspectAttributeMap(dp.Attributes())
						if err != nil {
							return 0, 0, err
						}
						points++
						estimatedBytes += baseSize + attrSize + 208
					}

				case pmetric.MetricTypeHistogram:
					histogram := metric.Histogram()
					if !isSupportedAggregationTemporality(histogram.AggregationTemporality()) {
						return 0, 0, status.Error(codes.InvalidArgument,
							"histogram aggregation temporality must be DELTA or CUMULATIVE")
					}
					dps := histogram.DataPoints()
					for idx := 0; idx < dps.Len(); idx++ {
						dp := dps.At(idx)
						if err := validateMetricPointTimes(dp.Timestamp(), dp.StartTimestamp(), now); err != nil {
							return 0, 0, err
						}
						if dp.ExplicitBounds().Len() > maxHistogramBuckets || dp.BucketCounts().Len() > maxHistogramBuckets+1 {
							return 0, 0, status.Error(codes.InvalidArgument, "histogram has too many buckets")
						}
						attrSize, err := inspectAttributeMap(dp.Attributes())
						if err != nil {
							return 0, 0, err
						}
						points++
						estimatedBytes += baseSize + attrSize + 256 + int64(dp.BucketCounts().Len()+dp.ExplicitBounds().Len())*8
					}

				case pmetric.MetricTypeExponentialHistogram:
					histogram := metric.ExponentialHistogram()
					if !isSupportedAggregationTemporality(histogram.AggregationTemporality()) {
						return 0, 0, status.Error(codes.InvalidArgument,
							"exponential histogram aggregation temporality must be DELTA or CUMULATIVE")
					}
					dps := histogram.DataPoints()
					for idx := 0; idx < dps.Len(); idx++ {
						dp := dps.At(idx)
						if err := validateMetricPointTimes(dp.Timestamp(), dp.StartTimestamp(), now); err != nil {
							return 0, 0, err
						}
						bucketCount := dp.Positive().BucketCounts().Len() + dp.Negative().BucketCounts().Len()
						if bucketCount > maxHistogramBuckets {
							return 0, 0, status.Error(codes.InvalidArgument, "exponential histogram has too many buckets")
						}
						attrSize, err := inspectAttributeMap(dp.Attributes())
						if err != nil {
							return 0, 0, err
						}
						points++
						estimatedBytes += baseSize + attrSize + 288 + int64(bucketCount)*8
					}

				case pmetric.MetricTypeSummary:
					return 0, 0, status.Error(codes.InvalidArgument, "OTLP summary metrics are not supported")
				default:
					return 0, 0, status.Error(codes.InvalidArgument, "unsupported OTLP metric type")
				}

				if points > maxDataPointsPerExport {
					return 0, 0, status.Error(codes.ResourceExhausted, "metric export contains too many data points")
				}
			}
		}
	}

	return points, estimatedBytes, nil
}

func validateNumberPointValue(point pmetric.NumberDataPoint, monotonic bool) error {
	switch point.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		if monotonic && point.IntValue() < 0 {
			return status.Error(codes.InvalidArgument, "monotonic sum data points must not be negative")
		}
	case pmetric.NumberDataPointValueTypeDouble:
		value := point.DoubleValue()
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return status.Error(codes.InvalidArgument, "number metric data points must be finite")
		}
		if monotonic && value < 0 {
			return status.Error(codes.InvalidArgument, "monotonic sum data points must not be negative")
		}
	default:
		return status.Error(codes.InvalidArgument, "number metric data point has no value")
	}
	return nil
}

func isSupportedAggregationTemporality(value pmetric.AggregationTemporality) bool {
	return value == pmetric.AggregationTemporalityDelta || value == pmetric.AggregationTemporalityCumulative
}

func inspectAttributeMap(attrs pcommon.Map) (int64, error) {
	if attrs.Len() > maxAttributesPerPoint {
		return 0, status.Error(codes.InvalidArgument, "metric point has too many attributes")
	}

	var size int64
	var retErr error
	attrs.Range(func(key string, value pcommon.Value) bool {
		if len(key) == 0 || len(key) > maxAttributeKeyBytes {
			retErr = status.Error(codes.InvalidArgument, "metric attribute key is too large")
			return false
		}
		size += int64(len(key) + 16)

		switch value.Type() {
		case pcommon.ValueTypeStr:
			if len(value.Str()) > maxAttributeValueBytes {
				retErr = status.Error(codes.InvalidArgument, "metric attribute value is too large")
				return false
			}
			size += int64(len(value.Str()))
		case pcommon.ValueTypeBool, pcommon.ValueTypeInt:
			size += 8
		case pcommon.ValueTypeDouble:
			if math.IsNaN(value.Double()) || math.IsInf(value.Double(), 0) {
				retErr = status.Error(codes.InvalidArgument, "metric attribute double value must be finite")
				return false
			}
			size += 8
		default:
			retErr = status.Error(codes.InvalidArgument, "metric attributes must use scalar values")
			return false
		}
		return true
	})

	return size, retErr
}

func validateMetricPointTimes(timestamp, startTimestamp pcommon.Timestamp, now time.Time) error {
	if timestamp == 0 {
		return status.Error(codes.InvalidArgument, "metric point timestamp must be set")
	}

	value := timestamp.AsTime().UTC()
	if value.After(now.Add(maximumFutureSkew)) {
		return status.Error(codes.InvalidArgument, "metric point timestamp is too far in the future")
	}
	if value.Before(now.Add(-maximumPastSkew)) {
		return status.Error(codes.InvalidArgument, "metric point timestamp is older than the accepted retention window")
	}

	if startTimestamp != 0 {
		start := startTimestamp.AsTime().UTC()
		if start.After(value) {
			return status.Error(codes.InvalidArgument, "metric point start timestamp is after its timestamp")
		}
		if start.After(now.Add(maximumFutureSkew)) {
			return status.Error(codes.InvalidArgument, "metric point start timestamp is too far in the future")
		}
	}

	return nil
}

func (s *srvMetric) startProcessLoop(ctx context.Context) {
	defer zap.L().Debug("Exiting metricstore process loop")

	for first := range s.itemCh {
		items := []*pendingMetricExport{first}
		pointCount := first.dataPoints

		timer := time.NewTimer(ingestBatchDelay)
	collect:
		for len(items) < maxIngestBatchExports && pointCount < maxIngestBatchPoints {
			select {
			case item, ok := <-s.itemCh:
				if !ok {
					break collect
				}
				items = append(items, item)
				pointCount += item.dataPoints
			case <-timer.C:
				break collect
			}
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}

		s.processPendingExports(items)
	}
}

func (s *srvMetric) processPendingExports(items []*pendingMetricExport) {
	batch := newMetricWriteBatch()
	valid := make([]*pendingMetricExport, 0, len(items))

	for _, item := range items {
		if err := item.ctx.Err(); err != nil {
			s.finishPendingExport(item, err)
			continue
		}
		itemBatch, err := buildMetricWriteBatch(item.metrics)
		if err != nil {
			s.finishPendingExport(item, err)
			continue
		}
		if err := batch.merge(itemBatch); err != nil {
			s.finishPendingExport(item, status.Error(codes.InvalidArgument, err.Error()))
			continue
		}
		valid = append(valid, item)
	}

	if len(valid) == 0 {
		return
	}

	batch.normalize()

	ctx, cancel := context.WithTimeout(context.Background(), ingestCommitTimeout)
	err := s.storeMetricWriteBatch(ctx, batch)
	cancel()

	if err == nil {
		points := len(batch.numberPoints) + len(batch.histogramPoints) + len(batch.exponentialPoints)
		s.storedPoints.Add(uint64(points))
	}

	for _, item := range valid {
		s.finishPendingExport(item, err)
	}
}

func (s *srvMetric) finishPendingExport(item *pendingMetricExport, err error) {
	s.budget.release(item.dataPoints, item.estimatedBytes)
	item.done <- err
}

func buildMetricWriteBatch(metrics pmetric.Metrics) (*metricWriteBatch, error) {
	ret := newMetricWriteBatch()
	ingestedAt := normalizeMetricTime(time.Now())

	resourceMetrics := metrics.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		scopeMetrics := rm.ScopeMetrics()

		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			metricsSlice := sm.Metrics()

			for k := 0; k < metricsSlice.Len(); k++ {
				metric := metricsSlice.At(k)

				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					if err := appendGaugeMetric(ret, metric, sm, rm, ingestedAt); err != nil {
						return nil, err
					}
				case pmetric.MetricTypeSum:
					if err := appendSumMetric(ret, metric, sm, rm, ingestedAt); err != nil {
						return nil, err
					}
				case pmetric.MetricTypeHistogram:
					if err := appendHistogramMetric(ret, metric, sm, rm, ingestedAt); err != nil {
						return nil, err
					}
				case pmetric.MetricTypeExponentialHistogram:
					if err := appendExponentialHistogramMetric(ret, metric, sm, rm, ingestedAt); err != nil {
						return nil, err
					}
				case pmetric.MetricTypeSummary:
					return nil, status.Error(codes.InvalidArgument, "OTLP summary metrics are not supported")
				}
			}
		}
	}

	sort.Slice(ret.numberPoints, func(i, j int) bool {
		if ret.numberPoints[i].seriesID != ret.numberPoints[j].seriesID {
			return ret.numberPoints[i].seriesID < ret.numberPoints[j].seriesID
		}
		return ret.numberPoints[i].timestamp.Before(ret.numberPoints[j].timestamp)
	})
	sort.Slice(ret.histogramPoints, func(i, j int) bool {
		if ret.histogramPoints[i].seriesID != ret.histogramPoints[j].seriesID {
			return ret.histogramPoints[i].seriesID < ret.histogramPoints[j].seriesID
		}
		return ret.histogramPoints[i].timestamp.Before(ret.histogramPoints[j].timestamp)
	})
	sort.Slice(ret.exponentialPoints, func(i, j int) bool {
		if ret.exponentialPoints[i].seriesID != ret.exponentialPoints[j].seriesID {
			return ret.exponentialPoints[i].seriesID < ret.exponentialPoints[j].seriesID
		}
		return ret.exponentialPoints[i].timestamp.Before(ret.exponentialPoints[j].timestamp)
	})

	return ret, nil
}

func appendGaugeMetric(batch *metricWriteBatch, metric pmetric.Metric, sm pmetric.ScopeMetrics,
	rm pmetric.ResourceMetrics, ingestedAt time.Time) error {
	dps := metric.Gauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		valueType, intValue, doubleValue, err := numberDataPointValue(dp)
		if err != nil {
			return err
		}

		descriptor := newNumberDescriptor(metric, sm, vmetricsv1.MetricDescriptor_GAUGE,
			valueType, vmetricsv1.MetricDescriptor_TEMPORALITY_UNSET)
		series, err := ensureSeries(batch, descriptor, rm.Resource().Attributes(), sm.Scope().Attributes(),
			dp.Attributes(), ingestedAt)
		if err != nil {
			return err
		}

		timestamp := timestampOrNow(dp.Timestamp(), ingestedAt)
		startTimestamp := optionalTimestamp(dp.StartTimestamp())
		batch.numberPoints = append(batch.numberPoints, numberPointRecord{
			pointID:        pointID("NUMBER", series.id, timestamp, startTimestamp),
			timestamp:      timestamp,
			ingestedAt:     ingestedAt,
			startTimestamp: startTimestamp,
			seriesID:       series.id,
			intValue:       intValue,
			doubleValue:    doubleValue,
		})
	}
	return nil
}

func appendSumMetric(batch *metricWriteBatch, metric pmetric.Metric, sm pmetric.ScopeMetrics,
	rm pmetric.ResourceMetrics, ingestedAt time.Time) error {
	sum := metric.Sum()
	if !isSupportedAggregationTemporality(sum.AggregationTemporality()) {
		return status.Error(codes.InvalidArgument, "sum aggregation temporality must be DELTA or CUMULATIVE")
	}
	if !sum.IsMonotonic() && sum.AggregationTemporality() == pmetric.AggregationTemporalityDelta {
		return status.Error(codes.InvalidArgument,
			"delta non-monotonic sums are not accepted; normalize them to cumulative state before export")
	}

	kind := vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER
	if sum.IsMonotonic() {
		kind = vmetricsv1.MetricDescriptor_COUNTER
	}
	temporality := aggregationTemporalityToProto(sum.AggregationTemporality())

	dps := sum.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		valueType, intValue, doubleValue, err := numberDataPointValue(dp)
		if err != nil {
			return err
		}
		if sum.IsMonotonic() && ((intValue != nil && *intValue < 0) || (doubleValue != nil && *doubleValue < 0)) {
			return status.Error(codes.InvalidArgument, "monotonic sum data points must not be negative")
		}

		descriptor := newNumberDescriptor(metric, sm, kind, valueType, temporality)
		series, err := ensureSeries(batch, descriptor, rm.Resource().Attributes(), sm.Scope().Attributes(),
			dp.Attributes(), ingestedAt)
		if err != nil {
			return err
		}

		timestamp := timestampOrNow(dp.Timestamp(), ingestedAt)
		startTimestamp := optionalTimestamp(dp.StartTimestamp())
		batch.numberPoints = append(batch.numberPoints, numberPointRecord{
			pointID:        pointID("NUMBER", series.id, timestamp, startTimestamp),
			timestamp:      timestamp,
			ingestedAt:     ingestedAt,
			startTimestamp: startTimestamp,
			seriesID:       series.id,
			intValue:       intValue,
			doubleValue:    doubleValue,
		})
	}
	return nil
}

func appendHistogramMetric(batch *metricWriteBatch, metric pmetric.Metric, sm pmetric.ScopeMetrics,
	rm pmetric.ResourceMetrics, ingestedAt time.Time) error {
	histogram := metric.Histogram()
	if !isSupportedAggregationTemporality(histogram.AggregationTemporality()) {
		return status.Error(codes.InvalidArgument, "histogram aggregation temporality must be DELTA or CUMULATIVE")
	}
	temporality := aggregationTemporalityToProto(histogram.AggregationTemporality())

	dps := histogram.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		bounds := append([]float64(nil), dp.ExplicitBounds().AsRaw()...)
		counts := append([]uint64(nil), dp.BucketCounts().AsRaw()...)
		if err := validateExplicitHistogram(counts, bounds, dp.Count()); err != nil {
			return err
		}
		if err := validateHistogramValues(dp.HasSum(), dp.Sum(), dp.HasMin(), dp.Min(), dp.HasMax(), dp.Max()); err != nil {
			return err
		}

		descriptor := &descriptorRecord{
			name:            metric.Name(),
			kind:            vmetricsv1.MetricDescriptor_HISTOGRAM,
			numberValueType: vmetricsv1.MetricDescriptor_DOUBLE,
			unit:            metric.Unit(),
			description:     metric.Description(),
			temporality:     temporality,
			scopeName:       sm.Scope().Name(),
			scopeVersion:    sm.Scope().Version(),
			scopeSchemaURL:  sm.SchemaUrl(),
			explicitBounds:  bounds,
		}
		descriptor.id = descriptorID(descriptor)

		series, err := ensureSeries(batch, descriptor, rm.Resource().Attributes(), sm.Scope().Attributes(),
			dp.Attributes(), ingestedAt)
		if err != nil {
			return err
		}

		timestamp := timestampOrNow(dp.Timestamp(), ingestedAt)
		startTimestamp := optionalTimestamp(dp.StartTimestamp())
		batch.histogramPoints = append(batch.histogramPoints, histogramPointRecord{
			pointID:        pointID("HISTOGRAM", series.id, timestamp, startTimestamp),
			timestamp:      timestamp,
			ingestedAt:     ingestedAt,
			startTimestamp: startTimestamp,
			seriesID:       series.id,
			count:          dp.Count(),
			hasSum:         dp.HasSum(),
			sum:            optionalFloat(dp.HasSum(), dp.Sum()),
			min:            optionalFloat(dp.HasMin(), dp.Min()),
			max:            optionalFloat(dp.HasMax(), dp.Max()),
			bucketCounts:   counts,
		})
	}
	return nil
}

func appendExponentialHistogramMetric(batch *metricWriteBatch, metric pmetric.Metric, sm pmetric.ScopeMetrics,
	rm pmetric.ResourceMetrics, ingestedAt time.Time) error {
	histogram := metric.ExponentialHistogram()
	if !isSupportedAggregationTemporality(histogram.AggregationTemporality()) {
		return status.Error(codes.InvalidArgument,
			"exponential histogram aggregation temporality must be DELTA or CUMULATIVE")
	}
	temporality := aggregationTemporalityToProto(histogram.AggregationTemporality())

	dps := histogram.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		positiveCounts := append([]uint64(nil), dp.Positive().BucketCounts().AsRaw()...)
		negativeCounts := append([]uint64(nil), dp.Negative().BucketCounts().AsRaw()...)
		if err := validateExponentialHistogram(positiveCounts, negativeCounts, dp.ZeroCount(), dp.Count()); err != nil {
			return err
		}
		if err := validateHistogramValues(dp.HasSum(), dp.Sum(), dp.HasMin(), dp.Min(), dp.HasMax(), dp.Max()); err != nil {
			return err
		}

		scale := dp.Scale()
		if scale < -10 || scale > 20 {
			return status.Error(codes.InvalidArgument, "exponential histogram scale must be within [-10, 20]")
		}
		zeroThreshold := dp.ZeroThreshold()
		if math.IsNaN(zeroThreshold) || math.IsInf(zeroThreshold, 0) || zeroThreshold < 0 {
			return status.Error(codes.InvalidArgument, "invalid exponential histogram zero threshold")
		}
		if err := validateBucketOffset(dp.Positive().Offset(), len(positiveCounts)); err != nil {
			return err
		}
		if err := validateBucketOffset(dp.Negative().Offset(), len(negativeCounts)); err != nil {
			return err
		}
		descriptor := &descriptorRecord{
			name:                metric.Name(),
			kind:                vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM,
			numberValueType:     vmetricsv1.MetricDescriptor_DOUBLE,
			unit:                metric.Unit(),
			description:         metric.Description(),
			temporality:         temporality,
			scopeName:           sm.Scope().Name(),
			scopeVersion:        sm.Scope().Version(),
			scopeSchemaURL:      sm.SchemaUrl(),
			expMinimumScale:     &scale,
			expMaximumScale:     &scale,
			expMinimumThreshold: &zeroThreshold,
			expMaximumThreshold: &zeroThreshold,
		}
		descriptor.id = descriptorID(descriptor)

		series, err := ensureSeries(batch, descriptor, rm.Resource().Attributes(), sm.Scope().Attributes(),
			dp.Attributes(), ingestedAt)
		if err != nil {
			return err
		}

		timestamp := timestampOrNow(dp.Timestamp(), ingestedAt)
		startTimestamp := optionalTimestamp(dp.StartTimestamp())
		batch.exponentialPoints = append(batch.exponentialPoints, exponentialHistogramPointRecord{
			pointID:        pointID("EXPONENTIAL_HISTOGRAM", series.id, timestamp, startTimestamp),
			timestamp:      timestamp,
			ingestedAt:     ingestedAt,
			startTimestamp: startTimestamp,
			seriesID:       series.id,
			count:          dp.Count(),
			hasSum:         dp.HasSum(),
			sum:            optionalFloat(dp.HasSum(), dp.Sum()),
			min:            optionalFloat(dp.HasMin(), dp.Min()),
			max:            optionalFloat(dp.HasMax(), dp.Max()),
			scale:          scale,
			zeroCount:      dp.ZeroCount(),
			zeroThreshold:  zeroThreshold,
			positiveOffset: dp.Positive().Offset(),
			positiveCounts: positiveCounts,
			negativeOffset: dp.Negative().Offset(),
			negativeCounts: negativeCounts,
		})
	}
	return nil
}

func newNumberDescriptor(metric pmetric.Metric, sm pmetric.ScopeMetrics, kind vmetricsv1.MetricDescriptor_Kind,
	valueType vmetricsv1.MetricDescriptor_NumberValueType,
	temporality vmetricsv1.MetricDescriptor_Temporality) *descriptorRecord {
	ret := &descriptorRecord{
		name:            metric.Name(),
		kind:            kind,
		numberValueType: valueType,
		unit:            metric.Unit(),
		description:     metric.Description(),
		temporality:     temporality,
		scopeName:       sm.Scope().Name(),
		scopeVersion:    sm.Scope().Version(),
		scopeSchemaURL:  sm.SchemaUrl(),
	}
	ret.id = descriptorID(ret)
	return ret
}

func ensureSeries(batch *metricWriteBatch, descriptor *descriptorRecord, resourceAttrs, scopeAttrs,
	pointAttrs pcommon.Map, observedAt time.Time) (*seriesRecord, error) {
	if current := batch.descriptors[descriptor.id]; current != nil {
		mergeDescriptorRecord(current, descriptor)
	} else {
		batch.descriptors[descriptor.id] = descriptor
	}

	labels, err := attributesFromMaps(resourceAttrs, scopeAttrs, pointAttrs)
	if err != nil {
		return nil, err
	}
	if len(labels) > maxAttributesPerPoint {
		return nil, status.Error(codes.InvalidArgument, "metric series has too many effective attributes")
	}
	labelsJSON, err := storedAttributesJSON(labels)
	if err != nil {
		return nil, err
	}
	if len(labelsJSON) > maxSeriesLabelsBytes {
		return nil, status.Error(codes.InvalidArgument, "metric series labels are too large")
	}
	labelsKey := storedAttributesKey(labels)
	id := seriesID(descriptor.id, labelsKey)

	series := batch.series[id]
	if series == nil {
		componentType, componentNamespace, componentName := componentValues(labels)
		series = &seriesRecord{
			id:                 id,
			descriptorID:       descriptor.id,
			labels:             labels,
			labelsJSON:         labelsJSON,
			labelsKey:          labelsKey,
			componentType:      componentType,
			componentNamespace: componentNamespace,
			componentName:      componentName,
		}
		batch.series[id] = series
	}

	for _, attr := range labels {
		key := attributeRecordKey(descriptor.id, attr.Key)
		if current := batch.attributes[key]; current != nil {
			if current.valueKind != attr.Kind {
				return nil, status.Errorf(codes.InvalidArgument,
					"metric attribute %s changed value kind from %s to %s", attr.Key, current.valueKind, attr.Kind)
			}
			current.sourceMask |= attr.SourceMask
		} else {
			batch.attributes[key] = &attributeKeyRecord{
				descriptorID: descriptor.id,
				key:          attr.Key,
				valueKind:    attr.Kind,
				sourceMask:   attr.SourceMask,
			}
		}
	}

	_ = observedAt
	return series, nil
}

func numberDataPointValue(
	dp pmetric.NumberDataPoint,
) (vmetricsv1.MetricDescriptor_NumberValueType, *int64, *float64, error) {
	switch dp.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		value := dp.IntValue()
		return vmetricsv1.MetricDescriptor_INT64, &value, nil, nil
	case pmetric.NumberDataPointValueTypeDouble:
		value := dp.DoubleValue()
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return vmetricsv1.MetricDescriptor_NUMBER_VALUE_TYPE_UNSET, nil, nil,
				status.Error(codes.InvalidArgument, "non-finite number metric value")
		}
		return vmetricsv1.MetricDescriptor_DOUBLE, nil, &value, nil
	default:
		return vmetricsv1.MetricDescriptor_NUMBER_VALUE_TYPE_UNSET, nil, nil,
			status.Error(codes.InvalidArgument, "number metric point has no value")
	}
}

func aggregationTemporalityToProto(value pmetric.AggregationTemporality) vmetricsv1.MetricDescriptor_Temporality {
	switch value {
	case pmetric.AggregationTemporalityDelta:
		return vmetricsv1.MetricDescriptor_DELTA
	case pmetric.AggregationTemporalityCumulative:
		return vmetricsv1.MetricDescriptor_CUMULATIVE
	default:
		return vmetricsv1.MetricDescriptor_TEMPORALITY_UNSET
	}
}

func validateExplicitHistogram(counts []uint64, bounds []float64, count uint64) error {
	if len(counts) != len(bounds)+1 {
		return status.Error(codes.InvalidArgument, "invalid explicit histogram bucket layout")
	}
	for i, bound := range bounds {
		if math.IsNaN(bound) || math.IsInf(bound, 0) {
			return status.Error(codes.InvalidArgument, "explicit histogram bounds must be finite")
		}
		if i > 0 && !(bound > bounds[i-1]) {
			return status.Error(codes.InvalidArgument, "explicit histogram bounds must be strictly increasing")
		}
	}
	return validateHistogramCount(counts, 0, count)
}

func validateExponentialHistogram(positive, negative []uint64, zeroCount, count uint64) error {
	return validateHistogramCount(append(append([]uint64{}, positive...), negative...), zeroCount, count)
}

func validateHistogramValues(hasSum bool, sum float64, hasMin bool, min float64, hasMax bool, max float64) error {
	if hasSum && (math.IsNaN(sum) || math.IsInf(sum, 0)) {
		return status.Error(codes.InvalidArgument, "histogram sum must be finite")
	}
	if hasMin && (math.IsNaN(min) || math.IsInf(min, 0)) {
		return status.Error(codes.InvalidArgument, "histogram min must be finite")
	}
	if hasMax && (math.IsNaN(max) || math.IsInf(max, 0)) {
		return status.Error(codes.InvalidArgument, "histogram max must be finite")
	}
	if hasMin && hasMax && min > max {
		return status.Error(codes.InvalidArgument, "histogram min is greater than max")
	}
	return nil
}

func validateBucketOffset(offset int32, count int) error {
	if count == 0 {
		return nil
	}
	last := int64(offset) + int64(count) - 1
	if last < math.MinInt32 || last > math.MaxInt32 {
		return status.Error(codes.InvalidArgument, "exponential histogram bucket index overflows int32")
	}
	return nil
}

func validateHistogramCount(counts []uint64, initial, expected uint64) error {
	total := initial
	for _, count := range counts {
		if math.MaxUint64-total < count {
			return status.Error(codes.InvalidArgument, "histogram bucket count overflow")
		}
		total += count
	}
	if total != expected {
		return status.Error(codes.InvalidArgument, "histogram bucket counts do not match point count")
	}
	return nil
}

func timestampOrNow(timestamp pcommon.Timestamp, now time.Time) time.Time {
	if timestamp == 0 {
		return normalizeMetricTime(now)
	}
	return normalizeMetricTime(timestamp.AsTime())
}

func optionalTimestamp(timestamp pcommon.Timestamp) *time.Time {
	if timestamp == 0 {
		return nil
	}
	value := normalizeMetricTime(timestamp.AsTime())
	return &value
}

func optionalFloat(ok bool, value float64) *float64 {
	if !ok {
		return nil
	}
	ret := value
	return &ret
}

func (s *srvMetric) storeMetricWriteBatch(ctx context.Context, batch *metricWriteBatch) error {
	conn, err := s.s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `BEGIN TRANSACTION`); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = conn.ExecContext(rollbackCtx, `ROLLBACK`)
		}
	}()

	if err := upsertMetricMetadata(ctx, conn, batch); err != nil {
		return err
	}
	if err := replaceNumberPoints(ctx, conn, batch.numberPoints); err != nil {
		return err
	}
	if err := replaceHistogramPoints(ctx, conn, batch.histogramPoints); err != nil {
		return err
	}
	if err := replaceExponentialHistogramPoints(ctx, conn, batch.exponentialPoints); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return err
	}
	committed = true
	return nil
}

func upsertMetricMetadata(ctx context.Context, conn *sql.Conn, batch *metricWriteBatch) error {
	now := time.Now().UTC()

	for _, descriptor := range batch.descriptors {
		var explicitBounds any
		if descriptor.kind == vmetricsv1.MetricDescriptor_HISTOGRAM {
			explicitBounds = marshalFloat64Slice(descriptor.explicitBounds)
		}
		var expMinimumScale any
		if descriptor.expMinimumScale != nil {
			expMinimumScale = *descriptor.expMinimumScale
		}
		var expMaximumScale any
		if descriptor.expMaximumScale != nil {
			expMaximumScale = *descriptor.expMaximumScale
		}
		var expMinimumThreshold any
		if descriptor.expMinimumThreshold != nil {
			expMinimumThreshold = *descriptor.expMinimumThreshold
		}
		var expMaximumThreshold any
		if descriptor.expMaximumThreshold != nil {
			expMaximumThreshold = *descriptor.expMaximumThreshold
		}

		_, err := conn.ExecContext(ctx, `
INSERT INTO metric_descriptors (
	id, name, kind, number_value_type, unit, description, temporality,
	scope_name, scope_version, scope_schema_url, explicit_bounds,
	exp_min_scale, exp_max_scale, exp_zero_threshold_min, exp_zero_threshold_max,
	created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
	description = CASE
		WHEN EXCLUDED.description = '' THEN metric_descriptors.description
		ELSE EXCLUDED.description
	END,
	exp_min_scale = CASE
		WHEN metric_descriptors.exp_min_scale IS NULL THEN EXCLUDED.exp_min_scale
		WHEN EXCLUDED.exp_min_scale IS NULL THEN metric_descriptors.exp_min_scale
		ELSE LEAST(metric_descriptors.exp_min_scale, EXCLUDED.exp_min_scale)
	END,
	exp_max_scale = CASE
		WHEN metric_descriptors.exp_max_scale IS NULL THEN EXCLUDED.exp_max_scale
		WHEN EXCLUDED.exp_max_scale IS NULL THEN metric_descriptors.exp_max_scale
		ELSE GREATEST(metric_descriptors.exp_max_scale, EXCLUDED.exp_max_scale)
	END,
	exp_zero_threshold_min = CASE
		WHEN metric_descriptors.exp_zero_threshold_min IS NULL THEN EXCLUDED.exp_zero_threshold_min
		WHEN EXCLUDED.exp_zero_threshold_min IS NULL THEN metric_descriptors.exp_zero_threshold_min
		ELSE LEAST(metric_descriptors.exp_zero_threshold_min, EXCLUDED.exp_zero_threshold_min)
	END,
	exp_zero_threshold_max = CASE
		WHEN metric_descriptors.exp_zero_threshold_max IS NULL THEN EXCLUDED.exp_zero_threshold_max
		WHEN EXCLUDED.exp_zero_threshold_max IS NULL THEN metric_descriptors.exp_zero_threshold_max
		ELSE GREATEST(metric_descriptors.exp_zero_threshold_max, EXCLUDED.exp_zero_threshold_max)
	END,
	updated_at = EXCLUDED.updated_at
`, descriptor.id, descriptor.name, kindToString(descriptor.kind), numberValueTypeToString(descriptor.numberValueType),
			descriptor.unit, descriptor.description, temporalityToString(descriptor.temporality), descriptor.scopeName,
			descriptor.scopeVersion, descriptor.scopeSchemaURL, explicitBounds, expMinimumScale, expMaximumScale,
			expMinimumThreshold, expMaximumThreshold, metricTimeToDB(now), metricTimeToDB(now))
		if err != nil {
			return err
		}
	}

	for _, series := range batch.series {
		_, err := conn.ExecContext(ctx, `
INSERT INTO metric_series (
	id, descriptor_id, labels, labels_key,
	component_type, component_namespace, component_name,
	created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET updated_at = EXCLUDED.updated_at
`, series.id, series.descriptorID, series.labelsJSON, series.labelsKey, series.componentType, series.componentNamespace,
			series.componentName, metricTimeToDB(now), metricTimeToDB(now))
		if err != nil {
			return err
		}

		for _, attr := range series.labels {
			var stringValue any
			var boolValue any
			var intValue any
			var doubleValue any
			switch attr.Kind {
			case "STRING":
				stringValue = attr.StringValue
			case "BOOL":
				if attr.BoolValue != nil {
					boolValue = *attr.BoolValue
				}
			case "INT64":
				if attr.IntValue != nil {
					intValue = *attr.IntValue
				}
			case "DOUBLE":
				if attr.DoubleValue != nil {
					doubleValue = *attr.DoubleValue
				}
			}

			_, err := conn.ExecContext(ctx, `
INSERT INTO metric_series_attributes (
	series_id, descriptor_id, key, value_kind, value_key,
	value_string, value_bool, value_int, value_double,
	source_mask, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (series_id, key) DO UPDATE SET
	source_mask = metric_series_attributes.source_mask | EXCLUDED.source_mask,
	updated_at = EXCLUDED.updated_at
`, series.id, series.descriptorID, attr.Key, attr.Kind, attr.valueKey(), stringValue, boolValue, intValue,
				doubleValue, attr.SourceMask, metricTimeToDB(now))
			if err != nil {
				return err
			}
		}
	}

	for _, attr := range batch.attributes {
		var existingValueKind string
		err := conn.QueryRowContext(ctx, `
SELECT value_kind
FROM metric_attribute_keys
WHERE descriptor_id = ? AND key = ?
`, attr.descriptorID, attr.key).Scan(&existingValueKind)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil && existingValueKind != attr.valueKind {
			return status.Errorf(codes.InvalidArgument,
				"metric attribute %s changed value kind from %s to %s", attr.key, existingValueKind, attr.valueKind)
		}

		_, err = conn.ExecContext(ctx, `
INSERT INTO metric_attribute_keys (
	descriptor_id, key, value_kind, source_mask, first_seen_at, last_seen_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (descriptor_id, key) DO UPDATE SET
	source_mask = metric_attribute_keys.source_mask | EXCLUDED.source_mask,
	last_seen_at = EXCLUDED.last_seen_at
`, attr.descriptorID, attr.key, attr.valueKind, attr.sourceMask, metricTimeToDB(now), metricTimeToDB(now))
		if err != nil {
			return err
		}
	}

	return nil
}

func replaceNumberPoints(ctx context.Context, conn *sql.Conn, points []numberPointRecord) error {
	if len(points) == 0 {
		return nil
	}
	if _, err := conn.ExecContext(ctx, `DELETE FROM metric_number_staging`); err != nil {
		return err
	}
	if err := appendRows(conn, "metric_number_staging", func(appender *duckdb.Appender) error {
		for _, point := range points {
			if err := appender.AppendRow(point.pointID, metricTimeToDB(point.timestamp), metricTimeToDB(point.ingestedAt),
				nullableMetricTimeToDB(point.startTimestamp),
				point.seriesID, nullableInt64(point.intValue), nullableFloat64(point.doubleValue)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO metric_number_points
SELECT s.* FROM metric_number_staging s
WHERE NOT EXISTS (
	SELECT 1 FROM metric_number_points p WHERE p.point_id = s.point_id
)`); err != nil {
		return err
	}
	_, err := conn.ExecContext(ctx, `DELETE FROM metric_number_staging`)
	return err
}

func replaceHistogramPoints(ctx context.Context, conn *sql.Conn, points []histogramPointRecord) error {
	if len(points) == 0 {
		return nil
	}
	if _, err := conn.ExecContext(ctx, `DELETE FROM metric_histogram_staging`); err != nil {
		return err
	}
	if err := appendRows(conn, "metric_histogram_staging", func(appender *duckdb.Appender) error {
		for _, point := range points {
			if err := appender.AppendRow(point.pointID, metricTimeToDB(point.timestamp), metricTimeToDB(point.ingestedAt),
				nullableMetricTimeToDB(point.startTimestamp), point.seriesID, point.count, point.hasSum,
				nullableFloat64(point.sum), nullableFloat64(point.min), nullableFloat64(point.max),
				marshalUint64Slice(point.bucketCounts)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `
INSERT INTO metric_histogram_points (
	point_id, timestamp, ingested_at, start_timestamp, series_id,
	count, has_sum, sum, min, max, bucket_counts
)
SELECT
	s.point_id, s.timestamp, s.ingested_at, s.start_timestamp, s.series_id,
	s.count, s.has_sum, s.sum, s.min, s.max, CAST(s.bucket_counts AS JSON)
FROM metric_histogram_staging s
WHERE NOT EXISTS (
	SELECT 1 FROM metric_histogram_points p WHERE p.point_id = s.point_id
)
`); err != nil {
		return err
	}
	_, err := conn.ExecContext(ctx, `DELETE FROM metric_histogram_staging`)
	return err
}

func replaceExponentialHistogramPoints(ctx context.Context, conn *sql.Conn,
	points []exponentialHistogramPointRecord) error {
	if len(points) == 0 {
		return nil
	}
	if _, err := conn.ExecContext(ctx, `DELETE FROM metric_exponential_histogram_staging`); err != nil {
		return err
	}
	if err := appendRows(conn, "metric_exponential_histogram_staging", func(appender *duckdb.Appender) error {
		for _, point := range points {
			if err := appender.AppendRow(point.pointID, metricTimeToDB(point.timestamp), metricTimeToDB(point.ingestedAt),
				nullableMetricTimeToDB(point.startTimestamp), point.seriesID, point.count, point.hasSum,
				nullableFloat64(point.sum), nullableFloat64(point.min), nullableFloat64(point.max), point.scale,
				point.zeroCount, point.zeroThreshold, point.positiveOffset, marshalUint64Slice(point.positiveCounts),
				point.negativeOffset, marshalUint64Slice(point.negativeCounts)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `
INSERT INTO metric_exponential_histogram_points (
	point_id, timestamp, ingested_at, start_timestamp, series_id,
	count, has_sum, sum, min, max, scale, zero_count, zero_threshold,
	positive_offset, positive_counts, negative_offset, negative_counts
)
SELECT
	s.point_id, s.timestamp, s.ingested_at, s.start_timestamp, s.series_id,
	s.count, s.has_sum, s.sum, s.min, s.max, s.scale, s.zero_count, s.zero_threshold,
	s.positive_offset, CAST(s.positive_counts AS JSON), s.negative_offset, CAST(s.negative_counts AS JSON)
FROM metric_exponential_histogram_staging s
WHERE NOT EXISTS (
	SELECT 1 FROM metric_exponential_histogram_points p WHERE p.point_id = s.point_id
)
`); err != nil {
		return err
	}
	_, err := conn.ExecContext(ctx, `DELETE FROM metric_exponential_histogram_staging`)
	return err
}

func appendRows(conn *sql.Conn, table string, fn func(*duckdb.Appender) error) error {
	return conn.Raw(func(raw any) error {
		driverConn, ok := raw.(driver.Conn)
		if !ok {
			return fmt.Errorf("unexpected DuckDB driver connection type: %T", raw)
		}

		appender, err := duckdb.NewAppenderFromConn(driverConn, "main", table)
		if err != nil {
			return err
		}

		if err := fn(appender); err != nil {
			_ = appender.Close()
			return err
		}
		return appender.Close()
	})
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableFloat64(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
