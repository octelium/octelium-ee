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

	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func attributesToMap(attr []*visibilityv1.CommonQueryMetricsOptions_Attribute) map[string]string {
	if len(attr) == 0 {
		return nil
	}

	ret := make(map[string]string)
	for _, itm := range attr {
		ret[itm.Key] = itm.Value
	}

	return ret
}

func (s *srvMetric) GetCounter(ctx context.Context, req *visibilityv1.GetCounterRequest) (*visibilityv1.GetCounterResponse, error) {

	c := req.Common

	res, err := s.doQueryCounter(ctx, c.Name, c.From.AsTime(), c.To.AsTime(), attributesToMap(c.Attributes))
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *srvMetric) GetGauge(ctx context.Context, req *visibilityv1.GetGaugeRequest) (*visibilityv1.GetGaugeResponse, error) {

	c := req.Common

	res, err := s.doQueryGauge(ctx, c.Name, c.From.AsTime(), c.To.AsTime(), attributesToMap(c.Attributes))
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *srvMetric) GetHistogram(ctx context.Context, req *visibilityv1.GetHistogramRequest) (*visibilityv1.GetHistogramResponse, error) {
	resp := &visibilityv1.GetHistogramResponse{}

	c := req.Common
	res, err := s.doQueryHistogram(ctx, c.Name, c.From.AsTime(), c.To.AsTime(), attributesToMap(c.Attributes))
	if err != nil {
		return nil, err
	}

	for _, itm := range res {
		resp.Points = append(resp.Points, &visibilityv1.GetHistogramResponse_DataPoint{
			Timestamp: pbutils.Timestamp(itm.Timestamp),
			Attrs:     pbutils.MapToStructMust(itm.Labels),
			Sum:       itm.Sum,
			Count:     float64(itm.Count),
			Buckets: func() []*visibilityv1.GetHistogramResponse_DataPoint_Bucket {
				var ret []*visibilityv1.GetHistogramResponse_DataPoint_Bucket
				for _, b := range itm.Buckets {
					ret = append(ret, &visibilityv1.GetHistogramResponse_DataPoint_Bucket{
						Le:    b.UpperBound,
						Count: float64(b.Count),
					})
				}
				return ret
			}(),
		})
	}

	return resp, nil
}
