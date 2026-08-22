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
	"math"
	"sort"
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxSeriesPerQuery               = 200
	defaultSeriesPerQuery           = 50
	maxPointsPerSeries              = 5000
	defaultPointsPerSeries          = 1000
	maxTotalPoints                  = 50000
	defaultTotalPoints              = 10000
	maxFilters                      = 32
	maxFilterValues                 = 128
	maxGroupByAttributes            = 8
	minimumQueryStep                = time.Second
	maximumSourceSeries             = 20000
	maximumRawHistogramRowsPerQuery = 100000
	maximumRawNumberRowsPerQuery    = 2000000
)

type querySpec struct {
	req *vmetricsv1.QueryMetricsRequest

	from     time.Time
	to       time.Time
	step     time.Duration
	snapshot time.Time

	limitSeries          int
	limitPointsPerSeries int
	limitTotalPoints     int
	limitBehavior        vmetricsv1.QueryMetricsRequest_LimitBehavior

	groupBy []string
	filters []*vmetricsv1.AttributeFilter
}

func (s *srvMetric) validateQueryRequest(ctx context.Context,
	req *vmetricsv1.QueryMetricsRequest) (*querySpec, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}
	if req.Metric == nil || req.Metric.Selector == nil {
		return nil, status.Error(codes.InvalidArgument, "metric selector must be set")
	}
	metricName := strings.TrimSpace(req.Metric.GetName())
	descriptorID := strings.TrimSpace(req.Metric.GetDescriptorID())
	if metricName == "" && descriptorID == "" {
		return nil, status.Error(codes.InvalidArgument, "metric selector must be set")
	}
	if len(metricName) > maxMetricNameLength {
		return nil, status.Error(codes.InvalidArgument, "metric name is too long")
	}
	if descriptorID != "" && !isValidContentID(descriptorID) {
		return nil, status.Error(codes.InvalidArgument, "invalid metric descriptor ID")
	}
	if req.TimeRange == nil || req.TimeRange.From == nil || req.TimeRange.To == nil {
		return nil, status.Error(codes.InvalidArgument, "timeRange.from and timeRange.to must be set")
	}
	if err := req.TimeRange.From.CheckValid(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid timeRange.from")
	}
	if err := req.TimeRange.To.CheckValid(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid timeRange.to")
	}

	from := normalizeMetricTime(req.TimeRange.From.AsTime())
	to := normalizeMetricTime(req.TimeRange.To.AsTime())
	if !to.After(from) {
		return nil, status.Error(codes.InvalidArgument, "timeRange.to must be after timeRange.from")
	}
	if to.Sub(from) > rawMetricRetention {
		return nil, status.Error(codes.InvalidArgument, "query time range exceeds available raw retention")
	}
	if to.After(time.Now().UTC().Add(maximumFutureSkew)) {
		return nil, status.Error(codes.InvalidArgument, "timeRange.to is too far in the future")
	}

	if err := normalizeComponentSelector(req.Component); err != nil {
		return nil, err
	}

	if req.Operation == nil || req.Operation.Type == nil {
		return nil, status.Error(codes.InvalidArgument, "operation must be set")
	}

	isRaw := req.Operation.GetCounter() != nil && req.Operation.GetCounter().Function == vmetricsv1.CounterOperation_RAW
	step := time.Duration(0)
	if isRaw {
		if req.Step != nil {
			return nil, status.Error(codes.InvalidArgument, "step must be unset for counter RAW queries")
		}
	} else {
		if req.Step == nil {
			return nil, status.Error(codes.InvalidArgument, "step must be set for bucketed queries")
		}
		var err error
		step, err = metav1DurationToTimeDuration(req.Step)
		if err != nil {
			return nil, err
		}
		if step < minimumQueryStep {
			return nil, status.Error(codes.InvalidArgument, "step is too small")
		}

		from = alignMetricTimeDown(from, step)
		if to.Sub(from) > rawMetricRetention {
			return nil, status.Error(codes.InvalidArgument, "query time range exceeds available raw retention")
		}

		requestedPoints := int((to.Sub(from) + step - 1) / step)
		if requestedPoints > maxPointsPerSeries {
			return nil, status.Error(codes.InvalidArgument, "query produces too many time buckets")
		}
	}

	limitSeries := int(req.LimitSeries)
	if limitSeries <= 0 {
		limitSeries = defaultSeriesPerQuery
	}
	if limitSeries > maxSeriesPerQuery {
		limitSeries = maxSeriesPerQuery
	}

	limitPoints := int(req.LimitPointsPerSeries)
	if limitPoints <= 0 {
		limitPoints = defaultPointsPerSeries
	}
	if limitPoints > maxPointsPerSeries {
		limitPoints = maxPointsPerSeries
	}

	limitTotal := int(req.LimitTotalPoints)
	if limitTotal <= 0 {
		limitTotal = defaultTotalPoints
	}
	if limitTotal > maxTotalPoints {
		limitTotal = maxTotalPoints
	}

	limitBehavior := req.LimitBehavior
	if limitBehavior == vmetricsv1.QueryMetricsRequest_LIMIT_BEHAVIOR_UNSET {
		limitBehavior = vmetricsv1.QueryMetricsRequest_ERROR
	}
	if limitBehavior != vmetricsv1.QueryMetricsRequest_ERROR && limitBehavior != vmetricsv1.QueryMetricsRequest_TRUNCATE {
		return nil, status.Error(codes.InvalidArgument, "invalid limit behavior")
	}

	groupBy, err := validateGroupBy(req.GroupBy)
	if err != nil {
		return nil, err
	}
	filters, err := validateAttributeFilters(req.Filters)
	if err != nil {
		return nil, err
	}

	snapshot := normalizeMetricTime(time.Now().UTC())
	if snapshot.After(time.Now().UTC().Add(maximumFutureSkew)) {
		return nil, status.Error(codes.InvalidArgument, "invalid query snapshot")
	}
	if from.Before(snapshot.Add(-rawMetricRetention)) {
		return nil, status.Error(codes.InvalidArgument, "timeRange.from is outside the available retention window")
	}

	return &querySpec{
		req:                  req,
		from:                 from,
		to:                   to,
		step:                 step,
		snapshot:             snapshot,
		limitSeries:          limitSeries,
		limitPointsPerSeries: limitPoints,
		limitTotalPoints:     limitTotal,
		limitBehavior:        limitBehavior,
		groupBy:              groupBy,
		filters:              filters,
	}, nil
}

func alignMetricTimeDown(value time.Time, step time.Duration) time.Time {
	if step <= 0 {
		return value
	}

	nanos := value.UnixNano()
	remainder := nanos % int64(step)
	if remainder < 0 {
		remainder += int64(step)
	}

	return normalizeMetricTime(time.Unix(0, nanos-remainder))
}

func normalizeComponentSelector(component *vmetricsv1.ComponentSelector) error {
	if component == nil {
		return nil
	}

	component.Type = strings.TrimSpace(component.Type)
	component.Namespace = strings.TrimSpace(component.Namespace)
	component.Name = strings.TrimSpace(component.Name)
	if len(component.Type) > 128 || len(component.Namespace) > 255 || len(component.Name) > 255 {
		return status.Error(codes.InvalidArgument, "component selector value is too long")
	}
	return nil
}

func validateGroupBy(values []string) ([]string, error) {
	if len(values) > maxGroupByAttributes {
		return nil, status.Error(codes.InvalidArgument, "too many groupBy attributes")
	}

	seen := map[string]struct{}{}
	ret := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, status.Error(codes.InvalidArgument, "groupBy key cannot be empty")
		}
		if len(value) > maxAttributeKeyBytes {
			return nil, status.Error(codes.InvalidArgument, "groupBy key is too long")
		}
		if _, ok := seen[value]; ok {
			return nil, status.Errorf(codes.InvalidArgument, "duplicate groupBy key: %s", value)
		}
		seen[value] = struct{}{}
		ret = append(ret, value)
	}
	sort.Strings(ret)
	return ret, nil
}

func validateAttributeFilters(values []*vmetricsv1.AttributeFilter) ([]*vmetricsv1.AttributeFilter, error) {
	if len(values) > maxFilters {
		return nil, status.Error(codes.InvalidArgument, "too many attribute filters")
	}

	seen := map[string]struct{}{}
	ret := make([]*vmetricsv1.AttributeFilter, 0, len(values))
	for _, filter := range values {
		if filter == nil {
			return nil, status.Error(codes.InvalidArgument, "nil attribute filter")
		}
		filter.Key = strings.TrimSpace(filter.Key)
		if filter.Key == "" {
			return nil, status.Error(codes.InvalidArgument, "attribute filter key cannot be empty")
		}
		if len(filter.Key) > maxAttributeKeyBytes {
			return nil, status.Error(codes.InvalidArgument, "attribute filter key is too long")
		}
		if _, ok := seen[filter.Key]; ok {
			return nil, status.Errorf(codes.InvalidArgument, "duplicate attribute filter key: %s", filter.Key)
		}
		seen[filter.Key] = struct{}{}

		switch filter.Operator {
		case vmetricsv1.AttributeFilter_EQ, vmetricsv1.AttributeFilter_NOT_EQ:
			if filter.Value == nil || filter.Value.Value == nil || len(filter.Values) != 0 {
				return nil, status.Error(codes.InvalidArgument, "invalid scalar attribute filter")
			}
			if err := validateProtoAttributeValue(filter.Value); err != nil {
				return nil, err
			}
		case vmetricsv1.AttributeFilter_IN:
			if filter.Value != nil || len(filter.Values) == 0 || len(filter.Values) > maxFilterValues {
				return nil, status.Error(codes.InvalidArgument, "invalid IN attribute filter")
			}
			seenValues := map[string]struct{}{}
			for _, value := range filter.Values {
				if value == nil || value.Value == nil {
					return nil, status.Error(codes.InvalidArgument, "invalid IN attribute filter value")
				}
				if err := validateProtoAttributeValue(value); err != nil {
					return nil, err
				}
				key := protoAttributeValueKey(value)
				if _, ok := seenValues[key]; ok {
					return nil, status.Error(codes.InvalidArgument, "duplicate IN attribute filter value")
				}
				seenValues[key] = struct{}{}
			}
		case vmetricsv1.AttributeFilter_EXISTS, vmetricsv1.AttributeFilter_NOT_EXISTS:
			if filter.Value != nil || len(filter.Values) != 0 {
				return nil, status.Error(codes.InvalidArgument, "existence filters must not contain values")
			}
		default:
			return nil, status.Error(codes.InvalidArgument, "invalid attribute filter operator")
		}

		ret = append(ret, filter)
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Key < ret[j].Key
	})
	return ret, nil
}

func validateProtoAttributeValue(value *vmetricsv1.AttributeValue) error {
	if value == nil || value.Value == nil {
		return status.Error(codes.InvalidArgument, "attribute value must be set")
	}
	switch typed := value.Value.(type) {
	case *vmetricsv1.AttributeValue_StringValue:
		if len(typed.StringValue) > maxAttributeValueBytes {
			return status.Error(codes.InvalidArgument, "attribute string value is too long")
		}
	case *vmetricsv1.AttributeValue_DoubleValue:
		if math.IsNaN(typed.DoubleValue) || math.IsInf(typed.DoubleValue, 0) {
			return status.Error(codes.InvalidArgument, "attribute double value must be finite")
		}
	case *vmetricsv1.AttributeValue_BoolValue, *vmetricsv1.AttributeValue_IntValue:
	default:
		return status.Error(codes.InvalidArgument, "unsupported attribute value kind")
	}
	return nil
}

func isValidContentID(value string) bool {
	const prefix = "v1:sha256:"
	if len(value) != len(prefix)+64 || !strings.HasPrefix(value, prefix) {
		return false
	}
	for _, char := range value[len(prefix):] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func validateQueryForDescriptor(q *querySpec, descriptor *vmetricsv1.MetricDescriptor) error {
	if q.req.Metric.Kind != vmetricsv1.MetricDescriptor_KIND_UNSET && q.req.Metric.Kind != descriptor.Kind {
		return status.Error(codes.InvalidArgument, "metric kind does not match descriptor")
	}

	attributes := map[string]*vmetricsv1.MetricAttributeDescriptor{}
	for _, attribute := range descriptor.Attributes {
		attributes[attribute.Key] = attribute
	}
	for _, key := range q.groupBy {
		attribute := attributes[key]
		if attribute != nil && !attribute.Groupable {
			reason := "attribute is not groupable"
			if attribute != nil && attribute.GroupUnsupportedReason != "" {
				reason = attribute.GroupUnsupportedReason
			}
			return status.Errorf(codes.InvalidArgument, "groupBy key %s is not allowed: %s", key, reason)
		}
	}
	for _, filter := range q.filters {
		attribute := attributes[filter.Key]
		if attribute != nil && !attribute.Filterable {
			reason := "attribute is not filterable"
			if attribute != nil && attribute.FilterUnsupportedReason != "" {
				reason = attribute.FilterUnsupportedReason
			}
			return status.Errorf(codes.InvalidArgument, "filter key %s is not allowed: %s", filter.Key, reason)
		}

		values := []*vmetricsv1.AttributeValue{}
		if filter.Value != nil {
			values = append(values, filter.Value)
		}
		values = append(values, filter.Values...)
		for _, value := range values {
			if attribute == nil {
				continue
			}
			if attributeKindToProto(protoAttributeKind(value)) != attribute.ValueKind {
				return status.Errorf(codes.InvalidArgument,
					"filter value kind does not match attribute %s", filter.Key)
			}
		}
	}

	aggregation := q.req.SeriesAggregation
	operation := q.req.Operation

	if aggregation == vmetricsv1.QueryMetricsRequest_NONE && len(q.groupBy) > 0 {
		return status.Error(codes.InvalidArgument, "groupBy must be empty for SeriesAggregation.NONE")
	}

	switch descriptor.Kind {
	case vmetricsv1.MetricDescriptor_COUNTER:
		counter := operation.GetCounter()
		if counter == nil {
			return status.Error(codes.InvalidArgument, "counter metric requires a counter operation")
		}
		switch counter.Function {
		case vmetricsv1.CounterOperation_RAW:
			if aggregation != vmetricsv1.QueryMetricsRequest_NONE {
				return status.Error(codes.InvalidArgument, "counter RAW requires SeriesAggregation.NONE")
			}
		case vmetricsv1.CounterOperation_RATE, vmetricsv1.CounterOperation_INCREASE:
			if !isNumberSeriesAggregation(aggregation) {
				return status.Error(codes.InvalidArgument, "invalid counter series aggregation")
			}
		default:
			return status.Error(codes.InvalidArgument, "invalid counter operation")
		}

	case vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER, vmetricsv1.MetricDescriptor_GAUGE:
		gauge := operation.GetGauge()
		if gauge == nil {
			return status.Error(codes.InvalidArgument, "gauge metric requires a gauge operation")
		}
		if gauge.Function < vmetricsv1.GaugeOperation_LAST || gauge.Function > vmetricsv1.GaugeOperation_SUM {
			return status.Error(codes.InvalidArgument, "invalid gauge operation")
		}
		if !isNumberSeriesAggregation(aggregation) {
			return status.Error(codes.InvalidArgument, "invalid gauge series aggregation")
		}

	case vmetricsv1.MetricDescriptor_HISTOGRAM, vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM:
		histogram := operation.GetHistogram()
		if histogram == nil {
			return status.Error(codes.InvalidArgument, "histogram metric requires a histogram operation")
		}
		if histogram.Function < vmetricsv1.HistogramOperation_BUCKETS || histogram.Function > vmetricsv1.HistogramOperation_MAX {
			return status.Error(codes.InvalidArgument, "invalid histogram operation")
		}
		if aggregation != vmetricsv1.QueryMetricsRequest_NONE && aggregation != vmetricsv1.QueryMetricsRequest_MERGE {
			return status.Error(codes.InvalidArgument, "histograms support only NONE or MERGE series aggregation")
		}
		if aggregation == vmetricsv1.QueryMetricsRequest_MERGE && !descriptorMergeSupported(descriptor) {
			return status.Error(codes.FailedPrecondition, descriptorMergeUnsupportedReason(descriptor))
		}
		if histogram.Function == vmetricsv1.HistogramOperation_QUANTILE {
			if len(histogram.Quantiles) == 0 || len(histogram.Quantiles) > 10 {
				return status.Error(codes.InvalidArgument, "invalid histogram quantiles")
			}
			seen := map[uint64]struct{}{}
			for _, quantile := range histogram.Quantiles {
				if math.IsNaN(quantile) || math.IsInf(quantile, 0) || quantile < 0 || quantile > 1 {
					return status.Error(codes.InvalidArgument, "histogram quantiles must be finite and within [0, 1]")
				}
				key := math.Float64bits(quantile)
				if _, ok := seen[key]; ok {
					return status.Error(codes.InvalidArgument, "duplicate histogram quantile")
				}
				seen[key] = struct{}{}
			}
			sort.Float64s(histogram.Quantiles)
		}

	default:
		return status.Error(codes.InvalidArgument, "unsupported metric kind")
	}

	return nil
}

func isNumberSeriesAggregation(value vmetricsv1.QueryMetricsRequest_SeriesAggregation) bool {
	switch value {
	case vmetricsv1.QueryMetricsRequest_NONE,
		vmetricsv1.QueryMetricsRequest_SUM,
		vmetricsv1.QueryMetricsRequest_AVG,
		vmetricsv1.QueryMetricsRequest_MIN,
		vmetricsv1.QueryMetricsRequest_MAX,
		vmetricsv1.QueryMetricsRequest_LAST:
		return true
	default:
		return false
	}
}

func descriptorMergeSupported(descriptor *vmetricsv1.MetricDescriptor) bool {
	if explicit := descriptor.GetExplicitHistogram(); explicit != nil {
		return explicit.MergeSupported
	}
	if exponential := descriptor.GetExponentialHistogram(); exponential != nil {
		return exponential.MergeSupported
	}
	return false
}

func descriptorMergeUnsupportedReason(descriptor *vmetricsv1.MetricDescriptor) string {
	if explicit := descriptor.GetExplicitHistogram(); explicit != nil && explicit.MergeUnsupportedReason != "" {
		return explicit.MergeUnsupportedReason
	}
	if exponential := descriptor.GetExponentialHistogram(); exponential != nil && exponential.MergeUnsupportedReason != "" {
		return exponential.MergeUnsupportedReason
	}
	return "histogram series cannot be merged"
}

func metav1DurationToTimeDuration(duration interface {
	GetMilliseconds() uint32
	GetSeconds() uint32
	GetMinutes() uint32
	GetHours() uint32
	GetDays() uint32
	GetWeeks() uint32
	GetMonths() uint32
}) (time.Duration, error) {
	switch {
	case duration.GetMilliseconds() > 0:
		return time.Duration(duration.GetMilliseconds()) * time.Millisecond, nil
	case duration.GetSeconds() > 0:
		return time.Duration(duration.GetSeconds()) * time.Second, nil
	case duration.GetMinutes() > 0:
		return time.Duration(duration.GetMinutes()) * time.Minute, nil
	case duration.GetHours() > 0:
		return time.Duration(duration.GetHours()) * time.Hour, nil
	case duration.GetDays() > 0:
		return time.Duration(duration.GetDays()) * 24 * time.Hour, nil
	case duration.GetWeeks() > 0:
		return time.Duration(duration.GetWeeks()) * 7 * 24 * time.Hour, nil
	case duration.GetMonths() > 0:
		return 0, grpcutils.InvalidArg("months are not supported for metric query durations")
	default:
		return 0, grpcutils.InvalidArg("duration must be set")
	}
}
