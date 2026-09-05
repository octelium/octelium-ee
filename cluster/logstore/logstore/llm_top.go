// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"context"
	"fmt"

	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

func (s *Server) listLLMTopResource(ctx context.Context, dim vllmv1.Dimension, f *vllmv1.Filter,
	limit uint32, orderBy vllmv1.Metric, quantiles bool) ([]*vllmv1.DimensionItem, *vllmv1.Stats, uint64, error) {

	filters, err := getLLMFilters(f)
	if err != nil {
		return nil, nil, 0, err
	}

	limit, err = getLLMTopLimit(limit)
	if err != nil {
		return nil, nil, 0, err
	}

	return s.listLLMDimensionItems(ctx, dim, filters, limit, orderBy, quantiles)
}

func (s *Server) listLLMTopDimension(ctx context.Context,
	req *vllmv1.ListTopDimensionRequest) (*vllmv1.ListTopDimensionResponse, error) {

	filters, err := getLLMFilters(req.Filter)
	if err != nil {
		return nil, err
	}

	limit, err := getLLMTopLimit(req.Limit)
	if err != nil {
		return nil, err
	}

	items, other, totalCount, err := s.listLLMDimensionItems(ctx, req.Dimension, filters,
		limit, req.OrderBy, req.IncludeQuantiles)
	if err != nil {
		return nil, err
	}

	return &vllmv1.ListTopDimensionResponse{
		Items:      items,
		Other:      other,
		TotalCount: totalCount,
	}, nil
}

func (s *Server) listLLMTopModel(ctx context.Context,
	req *vllmv1.ListTopModelRequest) (*vllmv1.ListTopModelResponse, error) {

	filters, err := getLLMFilters(req.Filter)
	if err != nil {
		return nil, err
	}

	limit, err := getLLMTopLimit(req.Limit)
	if err != nil {
		return nil, err
	}

	dim, err := func() (vllmv1.Dimension, error) {
		switch req.Field {
		case vllmv1.ModelField_MODEL_FIELD_UNSET, vllmv1.ModelField_EFFECTIVE:
			return vllmv1.Dimension_MODEL, nil
		case vllmv1.ModelField_REQUESTED:
			return vllmv1.Dimension_MODEL_REQUESTED, nil
		case vllmv1.ModelField_REPORTED:
			return vllmv1.Dimension_MODEL_REPORTED, nil
		default:
			return vllmv1.Dimension_DIMENSION_UNSET, grpcutils.InvalidArg("Invalid ModelField")
		}
	}()
	if err != nil {
		return nil, err
	}

	items, other, totalCount, err := s.listLLMDimensionItems(ctx, dim, filters,
		limit, req.OrderBy, req.IncludeQuantiles)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, item := range items {
		keys = append(keys, item.Key)
	}

	requestedCounts, err := s.getLLMKeyCounts(ctx, filters, llmExprModelRequested, keys)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	effectiveCounts, err := s.getLLMKeyCounts(ctx, filters, llmExprModelEffective, keys)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	reportedCounts, err := s.getLLMKeyCounts(ctx, filters, llmExprModelReported, keys)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret := &vllmv1.ListTopModelResponse{
		Other:      other,
		TotalCount: totalCount,
	}

	for _, item := range items {
		ret.Items = append(ret.Items, &vllmv1.ListTopModelResponse_Item{
			Model:          item.Key,
			Stats:          item.Stats,
			RequestedCount: requestedCounts[item.Key],
			EffectiveCount: effectiveCounts[item.Key],
			ReportedCount:  reportedCounts[item.Key],
		})
	}

	return ret, nil
}

func (s *Server) listLLMTopTool(ctx context.Context,
	req *vllmv1.ListTopToolRequest) (*vllmv1.ListTopToolResponse, error) {

	filters, err := getLLMFilters(req.Filter)
	if err != nil {
		return nil, err
	}

	limit, err := getLLMTopLimit(req.Limit)
	if err != nil {
		return nil, err
	}

	dim, err := func() (vllmv1.Dimension, error) {
		switch req.Scope {
		case vllmv1.ToolScope_TOOL_SCOPE_UNSET, vllmv1.ToolScope_DECLARED:
			return vllmv1.Dimension_TOOL, nil
		case vllmv1.ToolScope_CALLED:
			return vllmv1.Dimension_CALLED_TOOL, nil
		default:
			return vllmv1.Dimension_DIMENSION_UNSET, grpcutils.InvalidArg("Invalid ToolScope")
		}
	}()
	if err != nil {
		return nil, err
	}

	items, other, totalCount, err := s.listLLMDimensionItems(ctx, dim, filters,
		limit, req.OrderBy, req.IncludeQuantiles)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, item := range items {
		keys = append(keys, item.Key)
	}

	declaredCounts, err := s.getLLMKeyCounts(ctx, filters,
		fmt.Sprintf(`unnest(%s)`, llmExprToolNames), keys)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	calledCounts, err := s.getLLMKeyCounts(ctx, filters,
		fmt.Sprintf(`unnest(%s)`, llmExprCalledToolsL), keys)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret := &vllmv1.ListTopToolResponse{
		Other:      other,
		TotalCount: totalCount,
	}

	for _, item := range items {
		ret.Items = append(ret.Items, &vllmv1.ListTopToolResponse_Item{
			Tool:          item.Key,
			Stats:         item.Stats,
			DeclaredCount: declaredCounts[item.Key],
			CalledCount:   calledCounts[item.Key],
		})
	}

	return ret, nil
}

func (s *Server) listLLMTopUser(ctx context.Context,
	req *vllmv1.ListTopUserRequest) (*vllmv1.ListTopUserResponse, error) {

	items, other, totalCount, err := s.listLLMTopResource(ctx, vllmv1.Dimension_USER,
		req.Filter, req.Limit, req.OrderBy, req.IncludeQuantiles)
	if err != nil {
		return nil, err
	}

	ret := &vllmv1.ListTopUserResponse{
		Other:      other,
		TotalCount: totalCount,
	}

	for _, item := range items {
		usr, err := s.octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
			Uid: item.Key,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &vllmv1.ListTopUserResponse_Item{
			User:  usr,
			Stats: item.Stats,
		})
	}

	return ret, nil
}

func (s *Server) listLLMTopSession(ctx context.Context,
	req *vllmv1.ListTopSessionRequest) (*vllmv1.ListTopSessionResponse, error) {

	items, other, totalCount, err := s.listLLMTopResource(ctx, vllmv1.Dimension_SESSION,
		req.Filter, req.Limit, req.OrderBy, req.IncludeQuantiles)
	if err != nil {
		return nil, err
	}

	ret := &vllmv1.ListTopSessionResponse{
		Other:      other,
		TotalCount: totalCount,
	}

	for _, item := range items {
		sess, err := s.octeliumC.CoreC().GetSession(ctx, &rmetav1.GetOptions{
			Uid: item.Key,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &vllmv1.ListTopSessionResponse_Item{
			Session: sess,
			Stats:   item.Stats,
		})
	}

	return ret, nil
}

func (s *Server) listLLMTopService(ctx context.Context,
	req *vllmv1.ListTopServiceRequest) (*vllmv1.ListTopServiceResponse, error) {

	items, other, totalCount, err := s.listLLMTopResource(ctx, vllmv1.Dimension_SERVICE,
		req.Filter, req.Limit, req.OrderBy, req.IncludeQuantiles)
	if err != nil {
		return nil, err
	}

	ret := &vllmv1.ListTopServiceResponse{
		Other:      other,
		TotalCount: totalCount,
	}

	for _, item := range items {
		svc, err := s.octeliumC.CoreC().GetService(ctx, &rmetav1.GetOptions{
			Uid: item.Key,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &vllmv1.ListTopServiceResponse_Item{
			Service: svc,
			Stats:   item.Stats,
		})
	}

	return ret, nil
}

func (s *Server) listLLMTopPolicy(ctx context.Context,
	req *vllmv1.ListTopPolicyRequest) (*vllmv1.ListTopPolicyResponse, error) {

	items, other, totalCount, err := s.listLLMTopResource(ctx, vllmv1.Dimension_POLICY,
		req.Filter, req.Limit, req.OrderBy, req.IncludeQuantiles)
	if err != nil {
		return nil, err
	}

	ret := &vllmv1.ListTopPolicyResponse{
		Other:      other,
		TotalCount: totalCount,
	}

	for _, item := range items {
		policy, err := s.octeliumC.CoreC().GetPolicy(ctx, &rmetav1.GetOptions{
			Uid: item.Key,
		})
		if err != nil {
			continue
		}

		ret.Items = append(ret.Items, &vllmv1.ListTopPolicyResponse_Item{
			Policy: policy,
			Stats:  item.Stats,
		})
	}

	return ret, nil
}
