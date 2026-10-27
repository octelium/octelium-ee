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
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMetricAttributeGroupCapability(t *testing.T) {
	for _, key := range []string{
		"octelium.vigil.svc.name",
		"octelium.vigil.svc.namespace.name",
		"octelium.vigil.svc.region.name",
		"octelium.vigil.svc.mode",
		"reason",
		"state",
		"req.http.method",
		"req.mcp.method",
		"req.authorized",
		"op",
	} {
		groupable, reason := metricAttributeGroupCapability(key, maxSeriesPerQuery*10)
		assert.True(t, groupable, key)
		assert.Empty(t, reason, key)
	}

	groupable, reason := metricAttributeGroupCapability("future.metric.dimension", 1)
	assert.True(t, groupable)
	assert.Empty(t, reason)

	groupable, reason = metricAttributeGroupCapability("state", maxSeriesPerQuery*10+1)
	assert.True(t, groupable)
	assert.Empty(t, reason)
}

func TestMetricCatalogMatchesStoredMetricKinds(t *testing.T) {
	s := &srvMetric{}

	response, err := s.ListMetricCatalog(context.Background(), &vmetricsv1.ListMetricCatalogRequest{
		Component: &vmetricsv1.ComponentSelector{Type: "rscserver"},
	})
	assert.NoError(t, err)
	assert.Len(t, response.Items, 7)
	assert.Equal(t, "us", response.Items[4].Unit)
	assert.Equal(t, "resource_operations_rate", response.Items[5].Id)
	assert.Equal(t, []string{"op"}, response.Items[5].DefaultGroupBy)
	assert.Equal(t, "resource_operations_error_rate", response.Items[6].Id)
	assert.Equal(t, []string{"op"}, response.Items[6].DefaultGroupBy)

	response, err = s.ListMetricCatalog(context.Background(), &vmetricsv1.ListMetricCatalogRequest{
		Component: &vmetricsv1.ComponentSelector{Type: "vigil"},
	})
	assert.NoError(t, err)
	assert.Len(t, response.Items, 11)
	assert.Equal(t, "session.active", response.Items[5].Metric.GetName())
	assert.Equal(t, "conn.rejected", response.Items[7].Metric.GetName())
	assert.Equal(t, "req.ttfb", response.Items[10].Metric.GetName())

	response, err = s.ListMetricCatalog(context.Background(), &vmetricsv1.ListMetricCatalogRequest{
		Component: &vmetricsv1.ComponentSelector{Type: "octovigil"},
	})
	assert.NoError(t, err)
	assert.Len(t, response.Items, 5)
	assert.Equal(t, "authorization.req.total", response.Items[2].Metric.GetName())
	assert.Equal(t, "us", response.Items[4].Unit)
}

func TestMetricCatalogWithoutAComponentSelector(t *testing.T) {
	s := &srvMetric{}

	response, err := s.ListMetricCatalog(context.Background(), nil)
	assert.NoError(t, err)

	empty, err := s.ListMetricCatalog(context.Background(), &vmetricsv1.ListMetricCatalogRequest{
		Component: &vmetricsv1.ComponentSelector{},
	})
	assert.NoError(t, err)
	assert.Len(t, response.Items, len(empty.Items))

	ids := map[string]struct{}{}
	for _, item := range response.Items {
		_, ok := ids[item.Id]
		assert.False(t, ok, item.Id)
		ids[item.Id] = struct{}{}
	}

	for _, id := range []string{
		"process_goroutines",
		"requests_rate",
		"resource_operations_rate",
		"active_sessions",
		"authorization_requests_rate",
	} {
		_, ok := ids[id]
		assert.True(t, ok, id)
	}
}

func TestMetricCatalogGroupByKeysAreEmitted(t *testing.T) {
	s := &srvMetric{}

	response, err := s.ListMetricCatalog(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Items)

	for _, item := range response.Items {
		for _, key := range item.DefaultGroupBy {
			assert.NotEqual(t, "octelium.component.name", key, item.Id)
		}
	}
}

func TestMetricAttributeFilterCapabilityAllowsFutureDimensions(t *testing.T) {
	filterable, reason := metricAttributeFilterCapability("future.metric.dimension")
	assert.True(t, filterable)
	assert.Empty(t, reason)
}

func TestUnknownMetricDimensionsAreValidEmptyMatches(t *testing.T) {
	query := &querySpec{
		req: &vmetricsv1.QueryMetricsRequest{
			Metric: &vmetricsv1.MetricSelector{
				Selector: &vmetricsv1.MetricSelector_Name{Name: "req.total"},
			},
			GroupBy: []string{"future.group"},
			Filters: []*vmetricsv1.AttributeFilter{{
				Key:      "future.filter",
				Operator: vmetricsv1.AttributeFilter_EQ,
				Value: &vmetricsv1.AttributeValue{Value: &vmetricsv1.AttributeValue_StringValue{
					StringValue: "value",
				}},
			}},
			Operation: &vmetricsv1.QueryOperation{Type: &vmetricsv1.QueryOperation_Counter{
				Counter: &vmetricsv1.CounterOperation{Function: vmetricsv1.CounterOperation_RATE},
			}},
			SeriesAggregation: vmetricsv1.QueryMetricsRequest_SUM,
		},
		groupBy: []string{"future.group"},
		filters: []*vmetricsv1.AttributeFilter{{
			Key:      "future.filter",
			Operator: vmetricsv1.AttributeFilter_EQ,
			Value: &vmetricsv1.AttributeValue{Value: &vmetricsv1.AttributeValue_StringValue{
				StringValue: "value",
			}},
		}},
	}
	descriptor := &vmetricsv1.MetricDescriptor{Kind: vmetricsv1.MetricDescriptor_COUNTER}
	assert.NoError(t, validateQueryForDescriptor(query, descriptor))
}

func TestMetricQueryValidationBoundaries(t *testing.T) {
	base := time.Now().UTC().Truncate(time.Minute).Add(-time.Hour)
	valid := &vmetricsv1.QueryMetricsRequest{
		Metric:    &vmetricsv1.MetricSelector{Selector: &vmetricsv1.MetricSelector_Name{Name: "metric"}},
		TimeRange: testMetricTimeRange(base, base.Add(30*time.Minute)),
		Step:      testMetricStep(),
		Operation: &vmetricsv1.QueryOperation{
			Type: &vmetricsv1.QueryOperation_Gauge{Gauge: &vmetricsv1.GaugeOperation{Function: vmetricsv1.GaugeOperation_LAST}},
		},
	}
	for _, tc := range []struct {
		name   string
		change func(*vmetricsv1.QueryMetricsRequest)
	}{
		{"Selector", func(r *vmetricsv1.QueryMetricsRequest) { r.Metric = nil }},
		{"EmptySelector", func(r *vmetricsv1.QueryMetricsRequest) { r.Metric = &vmetricsv1.MetricSelector{} }},
		{"LongName", func(r *vmetricsv1.QueryMetricsRequest) {
			r.Metric.Selector = &vmetricsv1.MetricSelector_Name{Name: strings.Repeat("m", maxMetricNameLength+1)}
		}},
		{"InvalidID", func(r *vmetricsv1.QueryMetricsRequest) {
			r.Metric.Selector = &vmetricsv1.MetricSelector_DescriptorID{DescriptorID: "invalid"}
		}},
		{"MissingRange", func(r *vmetricsv1.QueryMetricsRequest) { r.TimeRange = nil }},
		{"MissingFrom", func(r *vmetricsv1.QueryMetricsRequest) { r.TimeRange.From = nil }},
		{"InvalidTimestamp", func(r *vmetricsv1.QueryMetricsRequest) { r.TimeRange.From = &timestamppb.Timestamp{Nanos: -1} }},
		{"Reversed", func(r *vmetricsv1.QueryMetricsRequest) { r.TimeRange.To = r.TimeRange.From }},
		{"Future", func(r *vmetricsv1.QueryMetricsRequest) {
			r.TimeRange = testMetricTimeRange(time.Now(), time.Now().Add(maximumFutureSkew+time.Hour))
		}},
		{"Old", func(r *vmetricsv1.QueryMetricsRequest) {
			r.TimeRange = testMetricTimeRange(time.Now().Add(-rawMetricRetention-time.Hour), base)
		}},
		{"NoOperation", func(r *vmetricsv1.QueryMetricsRequest) { r.Operation = nil }},
		{"NoStep", func(r *vmetricsv1.QueryMetricsRequest) { r.Step = nil }},
		{"SmallStep", func(r *vmetricsv1.QueryMetricsRequest) {
			r.Step = &metav1.Duration{Type: &metav1.Duration_Milliseconds{Milliseconds: 1}}
		}},
		{"Months", func(r *vmetricsv1.QueryMetricsRequest) {
			r.Step = &metav1.Duration{Type: &metav1.Duration_Months{Months: 1}}
		}},
		{"TooManyBuckets", func(r *vmetricsv1.QueryMetricsRequest) {
			r.TimeRange = testMetricTimeRange(base.Add(-2*time.Hour), base)
			r.Step = &metav1.Duration{Type: &metav1.Duration_Seconds{Seconds: 1}}
		}},
		{"InvalidLimitBehavior", func(r *vmetricsv1.QueryMetricsRequest) { r.LimitBehavior = 999 }},
		{"ComponentTooLong", func(r *vmetricsv1.QueryMetricsRequest) {
			r.Component = &vmetricsv1.ComponentSelector{Type: strings.Repeat("a", 129)}
		}},
		{"RawWithStep", func(r *vmetricsv1.QueryMetricsRequest) {
			r.Operation = &vmetricsv1.QueryOperation{
				Type: &vmetricsv1.QueryOperation_Counter{Counter: &vmetricsv1.CounterOperation{Function: vmetricsv1.CounterOperation_RAW}},
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := proto.Clone(valid).(*vmetricsv1.QueryMetricsRequest)
			tc.change(r)
			_, err := (&srvMetric{}).validateQueryRequest(t.Context(), r)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
	_, err := (&srvMetric{}).validateQueryRequest(t.Context(), nil)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	valid.LimitSeries = math.MaxUint32
	valid.LimitPointsPerSeries = math.MaxUint32
	valid.LimitTotalPoints = math.MaxUint32
	q, err := (&srvMetric{}).validateQueryRequest(t.Context(), valid)
	require.NoError(t, err)
	assert.Equal(t, maxSeriesPerQuery, q.limitSeries)
	assert.Equal(t, maxPointsPerSeries, q.limitPointsPerSeries)
	assert.Equal(t, maxTotalPoints, q.limitTotalPoints)
	assert.EqualValues(t, 30, q.bucketCount())
}

func TestMetricAttributeFilterValidationMatrix(t *testing.T) {
	value := &vmetricsv1.AttributeValue{Value: &vmetricsv1.AttributeValue_StringValue{StringValue: "value"}}
	for _, filters := range [][]*vmetricsv1.AttributeFilter{
		{nil}, {{Key: "", Operator: vmetricsv1.AttributeFilter_EXISTS}}, {{Key: strings.Repeat("x", maxAttributeKeyBytes+1), Operator: vmetricsv1.AttributeFilter_EXISTS}},
		{{Key: "a", Operator: 999}}, {{Key: "a", Operator: vmetricsv1.AttributeFilter_EQ}},
		{{Key: "a", Operator: vmetricsv1.AttributeFilter_EXISTS, Value: value}},
		{{Key: "a", Operator: vmetricsv1.AttributeFilter_IN}},
		{{Key: "a", Operator: vmetricsv1.AttributeFilter_IN, Values: []*vmetricsv1.AttributeValue{nil}}},
		{{
			Key:      "a",
			Operator: vmetricsv1.AttributeFilter_IN,
			Values:   []*vmetricsv1.AttributeValue{value, value},
		}},
		{{Key: "a", Operator: vmetricsv1.AttributeFilter_EXISTS}, {Key: " a ", Operator: vmetricsv1.AttributeFilter_NOT_EXISTS}},
	} {
		_, err := validateAttributeFilters(filters)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
	for _, op := range []vmetricsv1.AttributeFilter_Operator{vmetricsv1.AttributeFilter_EQ, vmetricsv1.AttributeFilter_NOT_EQ, vmetricsv1.AttributeFilter_IN, vmetricsv1.AttributeFilter_EXISTS, vmetricsv1.AttributeFilter_NOT_EXISTS} {
		f := &vmetricsv1.AttributeFilter{Key: " key ", Operator: op}
		if op == vmetricsv1.AttributeFilter_IN {
			f.Values = []*vmetricsv1.AttributeValue{value}
		} else if op == vmetricsv1.AttributeFilter_EQ || op == vmetricsv1.AttributeFilter_NOT_EQ {
			f.Value = value
		}
		filters, err := validateAttributeFilters([]*vmetricsv1.AttributeFilter{f})
		require.NoError(t, err)
		assert.Equal(t, "key", filters[0].Key)
	}
	for _, values := range [][]string{{""}, {"a", " a "}, {strings.Repeat("x", maxAttributeKeyBytes+1)}, make([]string, maxGroupByAttributes+1)} {
		_, err := validateGroupBy(values)
		assert.Error(t, err)
	}
	group, err := validateGroupBy([]string{" z ", "a"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "z"}, group)
	for _, v := range []*vmetricsv1.AttributeValue{nil, {}, {
		Value: &vmetricsv1.AttributeValue_StringValue{StringValue: strings.Repeat("a", maxAttributeValueBytes+1)},
	}, {Value: &vmetricsv1.AttributeValue_DoubleValue{DoubleValue: math.NaN()}}, {Value: &vmetricsv1.AttributeValue_DoubleValue{DoubleValue: math.Inf(1)}}} {
		assert.Error(t, validateProtoAttributeValue(v))
	}
	for _, v := range []*vmetricsv1.AttributeValue{{Value: &vmetricsv1.AttributeValue_BoolValue{BoolValue: true}}, {Value: &vmetricsv1.AttributeValue_IntValue{IntValue: -1}}, {Value: &vmetricsv1.AttributeValue_DoubleValue{DoubleValue: 1}}} {
		assert.NoError(t, validateProtoAttributeValue(v))
	}
}
