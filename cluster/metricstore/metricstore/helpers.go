// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func normalizeMetricTime(value time.Time) time.Time {
	return value.UTC()
}

func metricTimeToDB(value time.Time) int64 {
	return normalizeMetricTime(value).UnixNano()
}

func metricTimeFromDB(value int64) time.Time {
	return time.Unix(0, value).UTC()
}

func nullableMetricTimeToDB(value *time.Time) any {
	if value == nil {
		return nil
	}
	return metricTimeToDB(*value)
}

func kindToString(kind vmetricsv1.MetricDescriptor_Kind) string {
	switch kind {
	case vmetricsv1.MetricDescriptor_COUNTER:
		return "COUNTER"
	case vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER:
		return "UP_DOWN_COUNTER"
	case vmetricsv1.MetricDescriptor_GAUGE:
		return "GAUGE"
	case vmetricsv1.MetricDescriptor_HISTOGRAM:
		return "HISTOGRAM"
	case vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM:
		return "EXPONENTIAL_HISTOGRAM"
	default:
		return ""
	}
}

func kindFromString(kind string) vmetricsv1.MetricDescriptor_Kind {
	switch kind {
	case "COUNTER":
		return vmetricsv1.MetricDescriptor_COUNTER
	case "UP_DOWN_COUNTER":
		return vmetricsv1.MetricDescriptor_UP_DOWN_COUNTER
	case "GAUGE":
		return vmetricsv1.MetricDescriptor_GAUGE
	case "HISTOGRAM":
		return vmetricsv1.MetricDescriptor_HISTOGRAM
	case "EXPONENTIAL_HISTOGRAM":
		return vmetricsv1.MetricDescriptor_EXPONENTIAL_HISTOGRAM
	default:
		return vmetricsv1.MetricDescriptor_KIND_UNSET
	}
}

func numberValueTypeToString(valueType vmetricsv1.MetricDescriptor_NumberValueType) string {
	switch valueType {
	case vmetricsv1.MetricDescriptor_INT64:
		return "INT64"
	case vmetricsv1.MetricDescriptor_DOUBLE:
		return "DOUBLE"
	default:
		return ""
	}
}

func numberValueTypeFromString(valueType string) vmetricsv1.MetricDescriptor_NumberValueType {
	switch valueType {
	case "INT64":
		return vmetricsv1.MetricDescriptor_INT64
	case "DOUBLE":
		return vmetricsv1.MetricDescriptor_DOUBLE
	default:
		return vmetricsv1.MetricDescriptor_NUMBER_VALUE_TYPE_UNSET
	}
}

func temporalityToString(temporality vmetricsv1.MetricDescriptor_Temporality) string {
	switch temporality {
	case vmetricsv1.MetricDescriptor_DELTA:
		return "DELTA"
	case vmetricsv1.MetricDescriptor_CUMULATIVE:
		return "CUMULATIVE"
	default:
		return ""
	}
}

func temporalityFromString(temporality string) vmetricsv1.MetricDescriptor_Temporality {
	switch temporality {
	case "DELTA":
		return vmetricsv1.MetricDescriptor_DELTA
	case "CUMULATIVE":
		return vmetricsv1.MetricDescriptor_CUMULATIVE
	default:
		return vmetricsv1.MetricDescriptor_TEMPORALITY_UNSET
	}
}

func attributeKindToProto(kind string) vmetricsv1.AttributeValue_Kind {
	switch kind {
	case "STRING":
		return vmetricsv1.AttributeValue_STRING
	case "BOOL":
		return vmetricsv1.AttributeValue_BOOL
	case "INT64":
		return vmetricsv1.AttributeValue_INT64
	case "DOUBLE":
		return vmetricsv1.AttributeValue_DOUBLE
	default:
		return vmetricsv1.AttributeValue_KIND_UNSET
	}
}

func storedAttributeToProto(attr storedAttribute) *vmetricsv1.Attribute {
	value := &vmetricsv1.AttributeValue{}

	switch attr.Kind {
	case "STRING":
		value.Value = &vmetricsv1.AttributeValue_StringValue{StringValue: attr.StringValue}
	case "BOOL":
		if attr.BoolValue != nil {
			value.Value = &vmetricsv1.AttributeValue_BoolValue{BoolValue: *attr.BoolValue}
		}
	case "INT64":
		if attr.IntValue != nil {
			value.Value = &vmetricsv1.AttributeValue_IntValue{IntValue: *attr.IntValue}
		}
	case "DOUBLE":
		if attr.DoubleValue != nil {
			value.Value = &vmetricsv1.AttributeValue_DoubleValue{DoubleValue: *attr.DoubleValue}
		}
	}

	return &vmetricsv1.Attribute{
		Key:   attr.Key,
		Value: value,
	}
}

func storedAttributesToProto(attrs []storedAttribute) []*vmetricsv1.Attribute {
	ret := make([]*vmetricsv1.Attribute, 0, len(attrs))
	for _, attr := range attrs {
		ret = append(ret, storedAttributeToProto(attr))
	}
	return ret
}

func protoAttributeCanonicalValue(value *vmetricsv1.AttributeValue) string {
	if value == nil {
		return ""
	}

	switch val := value.Value.(type) {
	case *vmetricsv1.AttributeValue_StringValue:
		return val.StringValue
	case *vmetricsv1.AttributeValue_BoolValue:
		return strconv.FormatBool(val.BoolValue)
	case *vmetricsv1.AttributeValue_IntValue:
		return strconv.FormatInt(val.IntValue, 10)
	case *vmetricsv1.AttributeValue_DoubleValue:
		return strconv.FormatUint(math.Float64bits(val.DoubleValue), 16)
	default:
		return ""
	}
}

func protoAttributeKind(value *vmetricsv1.AttributeValue) string {
	if value == nil {
		return ""
	}

	switch value.Value.(type) {
	case *vmetricsv1.AttributeValue_StringValue:
		return "STRING"
	case *vmetricsv1.AttributeValue_BoolValue:
		return "BOOL"
	case *vmetricsv1.AttributeValue_IntValue:
		return "INT64"
	case *vmetricsv1.AttributeValue_DoubleValue:
		return "DOUBLE"
	default:
		return ""
	}
}

func protoAttributeValueKey(value *vmetricsv1.AttributeValue) string {
	return protoAttributeKind(value) + ":" + protoAttributeCanonicalValue(value)
}

func durationPB(duration time.Duration) *metav1.Duration {
	if duration <= 0 {
		return nil
	}
	if duration%time.Hour == 0 {
		return &metav1.Duration{Type: &metav1.Duration_Hours{Hours: uint32(duration / time.Hour)}}
	}
	if duration%time.Minute == 0 {
		return &metav1.Duration{Type: &metav1.Duration_Minutes{Minutes: uint32(duration / time.Minute)}}
	}
	if duration%time.Second == 0 {
		return &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: uint32(duration / time.Second)}}
	}
	return &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: uint32(duration / time.Millisecond)}}
}

func numberPointDouble(timestamp time.Time, startTimestamp *time.Time, value float64) *vmetricsv1.NumberPoint {
	ret := &vmetricsv1.NumberPoint{
		Timestamp: pbutils.Timestamp(timestamp),
		Value: &vmetricsv1.NumberPoint_AsDouble{
			AsDouble: value,
		},
	}
	if startTimestamp != nil {
		ret.StartTimestamp = pbutils.Timestamp(*startTimestamp)
	}
	return ret
}

func numberPointInt(timestamp time.Time, startTimestamp *time.Time, value int64) *vmetricsv1.NumberPoint {
	ret := &vmetricsv1.NumberPoint{
		Timestamp: pbutils.Timestamp(timestamp),
		Value: &vmetricsv1.NumberPoint_AsInt{
			AsInt: value,
		},
	}
	if startTimestamp != nil {
		ret.StartTimestamp = pbutils.Timestamp(*startTimestamp)
	}
	return ret
}

func cumulativeExplicitBuckets(bounds []float64, counts []uint64) []*vmetricsv1.HistogramBucket {
	ret := make([]*vmetricsv1.HistogramBucket, 0, len(counts))
	var cumulative uint64

	for i, count := range counts {
		if math.MaxUint64-cumulative < count {
			return nil
		}
		cumulative += count

		if i < len(bounds) {
			ret = append(ret, &vmetricsv1.HistogramBucket{
				Le:    bounds[i],
				Count: cumulative,
			})
		} else {
			ret = append(ret, &vmetricsv1.HistogramBucket{
				Count: cumulative,
				IsInf: true,
			})
		}
	}

	return ret
}

func explicitHistogramQuantile(quantile float64, bounds []float64, counts []uint64, count uint64) (float64, bool) {
	if count == 0 || len(counts) == 0 || len(counts) != len(bounds)+1 {
		return 0, false
	}

	target := quantile * float64(count)
	if quantile == 0 {
		target = 1
	}

	var cumulative uint64
	var previousCumulative uint64
	lower := math.Inf(-1)
	if len(bounds) > 0 && bounds[0] > 0 {
		lower = 0
	}

	for i, bucketCount := range counts {
		if math.MaxUint64-cumulative < bucketCount {
			return 0, false
		}
		cumulative += bucketCount
		if float64(cumulative) < target {
			previousCumulative = cumulative
			if i < len(bounds) {
				lower = bounds[i]
			}
			continue
		}

		if i >= len(bounds) {
			if math.IsInf(lower, -1) {
				return 0, false
			}
			return lower, true
		}

		upper := bounds[i]
		if math.IsInf(lower, -1) {
			return upper, true
		}

		inside := cumulative - previousCumulative
		if inside == 0 {
			return upper, true
		}

		position := (target - float64(previousCumulative)) / float64(inside)
		return lower + position*(upper-lower), true
	}

	return 0, false
}

func sortTimeSeries(series []*vmetricsv1.TimeSeries) {
	sort.Slice(series, func(i, j int) bool {
		return series[i].Id < series[j].Id
	})
}

func uniqueTruncationReasons(reasons []vmetricsv1.TruncationInfo_Reason) []vmetricsv1.TruncationInfo_Reason {
	seen := map[vmetricsv1.TruncationInfo_Reason]struct{}{}
	ret := make([]vmetricsv1.TruncationInfo_Reason, 0, len(reasons))
	for _, reason := range reasons {
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		ret = append(ret, reason)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i] < ret[j]
	})
	return ret
}

func formatQuantile(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
