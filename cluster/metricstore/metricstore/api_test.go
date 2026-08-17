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
	"testing"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/stretchr/testify/assert"
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
	response, err := s.ListMetricCatalog(context.Background(), nil)
	assert.NoError(t, err)
	assert.Len(t, response.Items, 2)
	for _, item := range response.Items {
		assert.Equal(t, vmetricsv1.MetricDescriptor_GAUGE, item.Metric.Kind)
	}

	response, err = s.ListMetricCatalog(context.Background(), &vmetricsv1.ListMetricCatalogRequest{
		Component: &vmetricsv1.ComponentSelector{Type: "rscserver"},
	})
	assert.NoError(t, err)
	assert.Len(t, response.Items, 5)
	assert.Equal(t, "us", response.Items[4].Unit)

	response, err = s.ListMetricCatalog(context.Background(), &vmetricsv1.ListMetricCatalogRequest{
		Component: &vmetricsv1.ComponentSelector{Type: "octovigil"},
	})
	assert.NoError(t, err)
	assert.Len(t, response.Items, 5)
	assert.Equal(t, "authorization.req.total", response.Items[2].Metric.GetName())
	assert.Equal(t, "us", response.Items[4].Unit)
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
