// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package metricstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/visibilityv1/vmetricsv1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPageTokenRoundTrip(t *testing.T) {
	server := &Server{pageTokenKey: []byte("01234567890123456789012345678901")}
	snapshot := normalizeMetricTime(time.Now())
	payload := newPageTokenPayload("series", "fingerprint", "cursor", snapshot)

	token, err := server.encodePageToken(payload)
	assert.Nil(t, err, "%+v", err)

	decoded, err := server.decodePageToken(token, "series", "fingerprint")
	if !assert.Nil(t, err, "%+v", err) {
		return
	}
	if !assert.NotNil(t, decoded) {
		return
	}

	assert.Equal(t, payload.Cursor, decoded.Cursor)
	assert.Equal(t, payload.SnapshotNS, decoded.SnapshotNS)
	assert.Equal(t, payload.ExpiresNS, decoded.ExpiresNS)
}

func TestPageTokenRejectsMismatchAndExpiry(t *testing.T) {
	server := &Server{pageTokenKey: []byte("01234567890123456789012345678901")}
	snapshot := normalizeMetricTime(time.Now().Add(-seriesPageTokenTTL - time.Second))
	payload := newPageTokenPayload("series", "fingerprint", "cursor", snapshot)

	token, err := server.encodePageToken(payload)
	assert.Nil(t, err, "%+v", err)

	_, err = server.decodePageToken(token, "series", "fingerprint")
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	_, err = server.decodePageToken(token, "descriptor", "fingerprint")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = server.decodePageToken(token, "series", "another-fingerprint")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestQueryFingerprintNormalizesOrdering(t *testing.T) {
	request1 := &vmetricsv1.QueryMetricsRequest{
		Metric:  &vmetricsv1.MetricSelector{Selector: &vmetricsv1.MetricSelector_Name{Name: " metric.name "}},
		GroupBy: []string{"rpc.method", "service.name"},
		Filters: []*vmetricsv1.AttributeFilter{
			{Key: "rpc.method", Operator: vmetricsv1.AttributeFilter_IN, Values: []*vmetricsv1.AttributeValue{
				{Value: &vmetricsv1.AttributeValue_StringValue{StringValue: "List"}},
				{Value: &vmetricsv1.AttributeValue_StringValue{StringValue: "Get"}},
			}},
		},
		Operation: &vmetricsv1.QueryOperation{Type: &vmetricsv1.QueryOperation_Histogram{
			Histogram: &vmetricsv1.HistogramOperation{
				Function:  vmetricsv1.HistogramOperation_QUANTILE,
				Quantiles: []float64{0.99, 0.5},
			},
		}},
	}
	request2 := &vmetricsv1.QueryMetricsRequest{
		Metric:  &vmetricsv1.MetricSelector{Selector: &vmetricsv1.MetricSelector_Name{Name: "metric.name"}},
		GroupBy: []string{"service.name", "rpc.method"},
		Filters: []*vmetricsv1.AttributeFilter{
			{Key: "rpc.method", Operator: vmetricsv1.AttributeFilter_IN, Values: []*vmetricsv1.AttributeValue{
				{Value: &vmetricsv1.AttributeValue_StringValue{StringValue: "Get"}},
				{Value: &vmetricsv1.AttributeValue_StringValue{StringValue: "List"}},
			}},
		},
		Operation: &vmetricsv1.QueryOperation{Type: &vmetricsv1.QueryOperation_Histogram{
			Histogram: &vmetricsv1.HistogramOperation{
				Function:  vmetricsv1.HistogramOperation_QUANTILE,
				Quantiles: []float64{0.5, 0.99},
			},
		}},
	}

	fingerprint1, err := queryRequestFingerprint(request1)
	assert.Nil(t, err, "%+v", err)
	fingerprint2, err := queryRequestFingerprint(request2)
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, fingerprint1, fingerprint2)
}
