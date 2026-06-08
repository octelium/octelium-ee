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
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxSeriesLimit       = 200
	maxPointsPerSeries  = 5000
	maxDescriptorLimit   = 1000
	defaultSeriesLimit   = 50
	defaultPointsLimit   = 5000
	maxQueryTimeRange    = 30 * 24 * time.Hour
	minQueryStep         = time.Second
	defaultQueryStep     = 30 * time.Second
	maxFilterValues      = 128
	maxGroupByAttributes = 8
)

type querySpec struct {
	req *vmetricsv1.QueryMetricsRequest

	name string

	from time.Time
	to   time.Time
	step time.Duration

	limitSeries          int
	limitPointsPerSeries int

	filters map[string]attributeFilter
	groupBy []string
}

type attributeFilter struct {
	op     vmetricsv1.AttributeFilter_Operator
	value  string
	values []string
}

func (s *srvMetric) validateQueryRequest(ctx context.Context, req *vmetricsv1.QueryMetricsRequest) (*querySpec, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	if req.Metric == nil || strings.TrimSpace(req.Metric.Name) == "" {
		return nil, status.Error(codes.InvalidArgument, "metric.name must be set")
	}

	if req.TimeRange == nil || req.TimeRange.From == nil || req.TimeRange.To == nil {
		return nil, status.Error(codes.InvalidArgument, "timeRange.from and timeRange.to must be set")
	}

	from := req.TimeRange.From.AsTime().UTC()
	to := req.TimeRange.To.AsTime().UTC()

	if !to.After(from) {
		return nil, status.Error(codes.InvalidArgument, "timeRange.to must be after timeRange.from")
	}

	if to.Sub(from) > maxQueryTimeRange {
		return nil, status.Error(codes.InvalidArgument, "query time range is too large")
	}

	step := defaultQueryStep
	if req.Step != nil {
		d, err := metav1DurationToTimeDuration(req.Step)
		if err != nil {
			return nil, err
		}
		step = d
	}

	if step < minQueryStep {
		return nil, status.Error(codes.InvalidArgument, "step is too small")
	}

	requestedPoints := int(to.Sub(from) / step)
	if requestedPoints <= 0 {
		return nil, status.Error(codes.InvalidArgument, "query time range and step produce no points")
	}
	if requestedPoints > maxPointsPerSeries {
		return nil, status.Error(codes.InvalidArgument, "too many points for requested time range and step")
	}

	limitSeries := int(req.LimitSeries)
	if limitSeries <= 0 {
		limitSeries = defaultSeriesLimit
	}
	if limitSeries > maxSeriesLimit {
		limitSeries = maxSeriesLimit
	}

	limitPoints := int(req.LimitPointsPerSeries)
	if limitPoints <= 0 {
		limitPoints = defaultPointsLimit
	}
	if limitPoints > maxPointsPerSeries {
		limitPoints = maxPointsPerSeries
	}
	if requestedPoints > limitPoints {
		return nil, status.Error(codes.InvalidArgument, "requested time range exceeds limitPointsPerSeries")
	}

	if len(req.GroupBy) > maxGroupByAttributes {
		return nil, status.Error(codes.InvalidArgument, "too many groupBy attributes")
	}

	groupBy := make([]string, 0, len(req.GroupBy))
	seenGroupBy := map[string]struct{}{}

	for _, key := range req.GroupBy {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, status.Error(codes.InvalidArgument, "groupBy key cannot be empty")
		}
		if !isAllowedMetricAttributeKey(key) {
			return nil, status.Errorf(codes.InvalidArgument, "groupBy key is not allowed: %s", key)
		}
		if _, ok := seenGroupBy[key]; ok {
			return nil, status.Errorf(codes.InvalidArgument, "duplicate groupBy key: %s", key)
		}

		seenGroupBy[key] = struct{}{}
		groupBy = append(groupBy, key)
	}

	filters := map[string]attributeFilter{}

	if req.Component != nil {
		if req.Component.Type != "" {
			filters["octelium.component.type"] = attributeFilter{
				op:    vmetricsv1.AttributeFilter_EQ,
				value: req.Component.Type,
			}
		}
		if req.Component.Namespace != "" {
			filters["octelium.component.namespace"] = attributeFilter{
				op:    vmetricsv1.AttributeFilter_EQ,
				value: req.Component.Namespace,
			}
		}
		if req.Component.Name != "" {
			filters["octelium.component.name"] = attributeFilter{
				op:    vmetricsv1.AttributeFilter_EQ,
				value: req.Component.Name,
			}
		}
	}

	for _, f := range req.Filters {
		if f == nil {
			return nil, status.Error(codes.InvalidArgument, "nil attribute filter")
		}

		key := strings.TrimSpace(f.Key)
		if key == "" {
			return nil, status.Error(codes.InvalidArgument, "attribute filter key cannot be empty")
		}
		if !isAllowedMetricAttributeKey(key) {
			return nil, status.Errorf(codes.InvalidArgument, "attribute filter key is not allowed: %s", key)
		}
		if _, ok := filters[key]; ok {
			return nil, status.Errorf(codes.InvalidArgument, "duplicate attribute filter key: %s", key)
		}

		switch f.Operator {
		case vmetricsv1.AttributeFilter_EQ, vmetricsv1.AttributeFilter_NOT_EQ:
			if f.Value == "" {
				return nil, status.Error(codes.InvalidArgument, "attribute filter value must be set")
			}
			if len(f.Values) != 0 {
				return nil, status.Error(codes.InvalidArgument, "attribute filter values must not be set")
			}
			filters[key] = attributeFilter{
				op:    f.Operator,
				value: f.Value,
			}

		case vmetricsv1.AttributeFilter_IN:
			if f.Value != "" {
				return nil, status.Error(codes.InvalidArgument, "attribute filter value must not be set")
			}
			if len(f.Values) == 0 {
				return nil, status.Error(codes.InvalidArgument, "attribute filter values must be set")
			}
			if len(f.Values) > maxFilterValues {
				return nil, status.Error(codes.InvalidArgument, "attribute filter has too many values")
			}
			filters[key] = attributeFilter{
				op:     f.Operator,
				values: f.Values,
			}

		case vmetricsv1.AttributeFilter_EXISTS, vmetricsv1.AttributeFilter_NOT_EXISTS:
			if f.Value != "" || len(f.Values) != 0 {
				return nil, status.Error(codes.InvalidArgument, "attribute filter value fields must not be set")
			}
			filters[key] = attributeFilter{
				op: f.Operator,
			}

		default:
			return nil, status.Error(codes.InvalidArgument, "invalid attribute filter operator")
		}
	}

	if req.Operation == nil || req.Operation.Type == nil {
		return nil, status.Error(codes.InvalidArgument, "operation must be set")
	}

	if c := req.Operation.GetCounter(); c != nil &&
		c.Function == vmetricsv1.CounterOperation_RAW &&
		len(groupBy) > 0 {
		return nil, status.Error(codes.InvalidArgument, "counter RAW does not support groupBy")
	}

	if h := req.Operation.GetHistogram(); h != nil {
		if h.Function == vmetricsv1.HistogramOperation_QUANTILE {
			if len(h.Quantiles) == 0 {
				return nil, status.Error(codes.InvalidArgument, "histogram quantiles must be set")
			}
			if len(h.Quantiles) > 10 {
				return nil, status.Error(codes.InvalidArgument, "too many histogram quantiles")
			}
			for _, q := range h.Quantiles {
				if q < 0 || q > 1 {
					return nil, status.Error(codes.InvalidArgument, "histogram quantiles must be within [0, 1]")
				}
			}
		}
	}

	return &querySpec{
		req:                  req,
		name:                 strings.TrimSpace(req.Metric.Name),
		from:                 from,
		to:                   to,
		step:                 step,
		limitSeries:          limitSeries,
		limitPointsPerSeries: limitPoints,
		filters:              filters,
		groupBy:              groupBy,
	}, nil
}

func validateOperationForKind(kind vmetricsv1.MetricDescriptor_Kind, op *vmetricsv1.QueryOperation) error {
	if op == nil || op.Type == nil {
		return status.Error(codes.InvalidArgument, "operation must be set")
	}

	switch kind {
	case vmetricsv1.MetricDescriptor_COUNTER:
		if op.GetCounter() == nil {
			return status.Error(codes.InvalidArgument, "counter metric requires counter operation")
		}
		switch op.GetCounter().Function {
		case vmetricsv1.CounterOperation_RAW,
			vmetricsv1.CounterOperation_RATE,
			vmetricsv1.CounterOperation_INCREASE:
			return nil
		default:
			return status.Error(codes.InvalidArgument, "invalid counter operation")
		}

	case vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER,
		vmetricsv1.MetricDescriptor_GAUGE:
		if op.GetGauge() == nil {
			return status.Error(codes.InvalidArgument, "gauge/updowncounter metric requires gauge operation")
		}
		switch op.GetGauge().Function {
		case vmetricsv1.GaugeOperation_LAST,
			vmetricsv1.GaugeOperation_AVG,
			vmetricsv1.GaugeOperation_MIN,
			vmetricsv1.GaugeOperation_MAX,
			vmetricsv1.GaugeOperation_SUM:
			return nil
		default:
			return status.Error(codes.InvalidArgument, "invalid gauge operation")
		}

	case vmetricsv1.MetricDescriptor_HISTOGRAM:
		if op.GetHistogram() == nil {
			return status.Error(codes.InvalidArgument, "histogram metric requires histogram operation")
		}
		switch op.GetHistogram().Function {
		case vmetricsv1.HistogramOperation_BUCKETS,
			vmetricsv1.HistogramOperation_QUANTILE,
			vmetricsv1.HistogramOperation_AVG,
			vmetricsv1.HistogramOperation_COUNT,
			vmetricsv1.HistogramOperation_SUM,
			vmetricsv1.HistogramOperation_MIN,
			vmetricsv1.HistogramOperation_MAX:
			return nil
		default:
			return status.Error(codes.InvalidArgument, "invalid histogram operation")
		}

	case vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM:
		return status.Error(codes.Unimplemented, "exponential histogram queries are not implemented")

	default:
		return status.Error(codes.InvalidArgument, "unsupported metric kind")
	}
}

func isAllowedMetricAttributeKey(key string) bool {
	switch key {
	case "octelium.component.type",
		"octelium.component.namespace",
		"octelium.component.name":
		return true
	}

	allowedPrefixes := []string{
		"octelium.vigil.svc.",
		"service.",
		"deployment.",
		"k8s.",
		"http.",
		"rpc.",
		"net.",
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

func metav1DurationToTimeDuration(d interface {
	GetMilliseconds() uint32
	GetSeconds() uint32
	GetMinutes() uint32
	GetHours() uint32
	GetDays() uint32
	GetWeeks() uint32
	GetMonths() uint32
}) (time.Duration, error) {
	switch {
	case d.GetMilliseconds() > 0:
		return time.Duration(d.GetMilliseconds()) * time.Millisecond, nil
	case d.GetSeconds() > 0:
		return time.Duration(d.GetSeconds()) * time.Second, nil
	case d.GetMinutes() > 0:
		return time.Duration(d.GetMinutes()) * time.Minute, nil
	case d.GetHours() > 0:
		return time.Duration(d.GetHours()) * time.Hour, nil
	case d.GetDays() > 0:
		return time.Duration(d.GetDays()) * 24 * time.Hour, nil
	case d.GetWeeks() > 0:
		return time.Duration(d.GetWeeks()) * 7 * 24 * time.Hour, nil
	case d.GetMonths() > 0:
		return 0, grpcutils.InvalidArg("months are not supported for metric query durations")
	default:
		return 0, grpcutils.InvalidArg("duration must be set")
	}
}