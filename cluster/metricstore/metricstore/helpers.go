// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func mergeAttrs(maps ...map[string]any) map[string]any {
	ret := map[string]any{}

	for _, m := range maps {
		for k, v := range m {
			ret[k] = v
		}
	}

	return ret
}

func decodeStringMap(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}

	ret := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func anyMapToStringMap(in map[string]any) map[string]string {
	ret := map[string]string{}

	for k, v := range in {
		switch t := v.(type) {
		case string:
			ret[k] = t
		case float64:
			ret[k] = strconv.FormatFloat(t, 'f', -1, 64)
		case bool:
			ret[k] = strconv.FormatBool(t)
		case nil:
		default:
			ret[k] = fmt.Sprintf("%v", t)
		}
	}

	return ret
}

func pickLabels(labels map[string]string, keys []string) map[string]string {
	if len(keys) == 0 {
		return map[string]string{}
	}

	ret := map[string]string{}
	for _, key := range keys {
		if val, ok := labels[key]; ok {
			ret[key] = val
		}
	}

	return ret
}

func labelsToProto(labels map[string]string) []*vmetricsv1.Attribute {
	keys := sortedKeys(labels)
	ret := make([]*vmetricsv1.Attribute, 0, len(keys))

	for _, key := range keys {
		ret = append(ret, &vmetricsv1.Attribute{
			Key:   key,
			Value: labels[key],
		})
	}

	return ret
}

func sortLabels(labels []*vmetricsv1.Attribute) {
	sort.Slice(labels, func(i, j int) bool {
		if labels[i].Key == labels[j].Key {
			return labels[i].Value < labels[j].Value
		}
		return labels[i].Key < labels[j].Key
	})
}

func labelsKey(labels map[string]string, keys []string) string {
	if len(keys) == 0 {
		return ""
	}

	var parts []string
	for _, key := range keys {
		parts = append(parts, key+"="+labels[key])
	}

	return strings.Join(parts, "\xff")
}

func labelsKeyFromProto(labels []*vmetricsv1.Attribute) string {
	if len(labels) == 0 {
		return ""
	}

	cp := append([]*vmetricsv1.Attribute{}, labels...)
	sortLabels(cp)

	var parts []string
	for _, l := range cp {
		parts = append(parts, l.Key+"="+l.Value)
	}

	return strings.Join(parts, "\xff")
}

func sortedKeys(m map[string]string) []string {
	ret := make([]string, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}

func sortedIntKeys[T any](m map[int]T) []int {
	ret := make([]int, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	sort.Ints(ret)
	return ret
}

func sameFloatSlice(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
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

func valueTypeFromString(valueType string) vmetricsv1.MetricDescriptor_ValueType {
	switch valueType {
	case "INT64":
		return vmetricsv1.MetricDescriptor_INT64
	case "DOUBLE":
		return vmetricsv1.MetricDescriptor_DOUBLE
	default:
		return vmetricsv1.MetricDescriptor_VALUE_TYPE_UNSET
	}
}

func temporalityToString(t pmetric.AggregationTemporality) string {
	switch t {
	case pmetric.AggregationTemporalityDelta:
		return "DELTA"
	case pmetric.AggregationTemporalityCumulative:
		return "CUMULATIVE"
	default:
		return ""
	}
}

func temporalityFromString(t string) vmetricsv1.MetricDescriptor_Temporality {
	switch t {
	case "DELTA":
		return vmetricsv1.MetricDescriptor_DELTA
	case "CUMULATIVE":
		return vmetricsv1.MetricDescriptor_CUMULATIVE
	default:
		return vmetricsv1.MetricDescriptor_TEMPORALITY_UNSET
	}
}

func temporalityEnumToString(t vmetricsv1.MetricDescriptor_Temporality) string {
	switch t {
	case vmetricsv1.MetricDescriptor_DELTA:
		return "DELTA"
	case vmetricsv1.MetricDescriptor_CUMULATIVE:
		return "CUMULATIVE"
	default:
		return ""
	}
}

func formatQuantile(q float64) string {
	return strconv.FormatFloat(q, 'f', -1, 64)
}