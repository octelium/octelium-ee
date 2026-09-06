// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package apiserver

import (
	"context"
	"fmt"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
)

type tstLLMService struct {
	octeliumC octeliumc.ClientInterface
	vllmv1.UnimplementedLLMServiceServer
}

var tstLLMModels = []string{
	"gpt-5",
	"claude-opus-5",
	"claude-sonnet-5",
	"gemini-3-pro",
	"gpt-5-mini",
	"claude-haiku-4-5",
	"text-embedding-3-large",
	"llama-4-70b",
}

var tstLLMTools = []string{
	"search_web",
	"read_file",
	"write_file",
	"run_query",
	"send_email",
	"list_tickets",
	"exec_shell",
	"fetch_url",
}

var tstLLMPlugins = []string{
	"pii-guardrail",
	"secrets-guardrail",
	"prompt-injection",
	"quota-per-user",
	"prompt-cache",
	"cost-router",
}

var tstLLMUserAgents = []string{
	"OpenAI/Python 1.99.0",
	"anthropic-sdk-python/0.60.0",
	"cursor/1.7.11",
	"octelium-cli/0.9.2",
	"node-fetch/3.3.2",
}

var tstLLMPaths = []string{
	"/v1/chat/completions",
	"/v1/responses",
	"/v1/messages",
	"/v1/embeddings",
	"/v1beta/models/gemini-3-pro:generateContent",
}

func tstLLMRand(min, max uint64) uint64 {
	if max <= min {
		return min
	}
	return uint64(utilrand.GetRandomRangeMath(int(min), int(max)))
}

func tstLLMDuration(base float64) *vllmv1.Stats_DurationStats {
	avg := base + float64(utilrand.GetRandomRangeMath(0, int(base)))
	return &vllmv1.Stats_DurationStats{
		Count: tstLLMRand(100, 400),
		SumMs: avg * 240,
		AvgMs: avg,
		MaxMs: avg * 7.5,
		P50Ms: avg * 0.82,
		P90Ms: avg * 1.9,
		P95Ms: avg * 2.6,
		P99Ms: avg * 4.4,
	}
}

func tstLLMStats(total uint64) *vllmv1.Stats {
	if total < 8 {
		total = 8
	}

	denied := total / 9
	failed := total / 21
	allowed := total - denied
	streamed := total / 2
	cacheExact := total / 12
	cacheSemantic := total / 16
	cacheStored := total / 7
	upstreamInvoked := total - cacheExact - cacheSemantic - denied
	withUsage := upstreamInvoked - upstreamInvoked/14
	withTools := total / 3
	withToolCalls := withTools / 2

	input := total * tstLLMRand(400, 2400)
	output := total * tstLLMRand(120, 900)
	cacheRead := input / 4
	cacheWrite := input / 11
	reasoning := output / 5

	return &vllmv1.Stats{
		Requests: &vllmv1.Stats_Requests{
			Total:                    total,
			Allowed:                  allowed,
			Denied:                   denied,
			Succeeded:                allowed - failed,
			Failed:                   failed,
			ClientError:              failed - failed/3,
			ServerError:              failed / 3,
			Streamed:                 streamed,
			UpstreamInvoked:          upstreamInvoked,
			DiscardedInference:       total / 40,
			SourceUpstream:           upstreamInvoked,
			SourceSemanticCache:      cacheExact + cacheSemantic,
			SourceOctelium:           denied,
			WithUsage:                withUsage,
			WithoutUsage:             total - withUsage,
			UsageComplete:            withUsage - withUsage/13,
			UsagePartial:             withUsage / 13,
			GuardrailInspected:       total - total/10,
			GuardrailPassed:          total - total/10 - total/13 - total/25 - total/60,
			GuardrailModified:        total / 13,
			GuardrailDenied:          total / 25,
			GuardrailError:           total / 60,
			TokenRateLimitAllowed:    total - total/30,
			TokenRateLimitDenied:     total / 30,
			CacheExactHit:            cacheExact,
			CacheSemanticHit:         cacheSemantic,
			CacheMiss:                total - cacheExact - cacheSemantic - total/18,
			CacheBypass:              total / 18,
			CacheError:               total / 90,
			CacheStored:              cacheStored,
			RouterMatch:              total / 5,
			RouterNoMatch:            total / 6,
			RouterBypass:             total / 10,
			RouterError:              total / 100,
			ModelOverridden:          total / 8,
			ModelRouted:              total / 5,
			WithTools:                withTools,
			WithToolCalls:            withToolCalls,
			WithToolsRemoved:         withTools / 6,
			WithCalledToolsTruncated: withToolCalls / 12,
			WithManagedReasoning:     total / 4,
			ReasoningDisabled:        total / 15,
			WithImageInput:           total / 9,
			WithAudioInput:           total / 26,
			FinishStop:               total / 2,
			FinishLength:             total / 14,
			FinishToolCall:           withToolCalls,
			FinishContentFilter:      total / 45,
			FinishError:              failed,
		},
		Tokens: &vllmv1.Stats_Tokens{
			Input:           input,
			Output:          output,
			Total:           input + output,
			CacheReadInput:  cacheRead,
			CacheWriteInput: cacheWrite,
			ReasoningOutput: reasoning,
			EstimatedInput:  input + input/12,
			Discarded:       output / 22,
		},
		Latency:             tstLLMDuration(620),
		TimeToFirstToken:    tstLLMDuration(180),
		StreamEvents:        streamed * tstLLMRand(40, 260),
		ToolsOffered:        withTools * tstLLMRand(3, 12),
		ToolsRemoved:        withTools / 6,
		ToolCalls:           withToolCalls * tstLLMRand(1, 4),
		DistinctToolsCalled: tstLLMRand(2, uint64(len(tstLLMTools))),
		InputItems:          total * tstLLMRand(2, 18),
		RequestBodyBytes:    input * 5,
		ResponseBodyBytes:   output * 6,
	}
}

func tstLLMDimensionKeys(dimension vllmv1.Dimension) []string {
	enumKeys := func(vals []string) []string {
		return vals
	}

	switch dimension {
	case vllmv1.Dimension_MODEL, vllmv1.Dimension_MODEL_REQUESTED,
		vllmv1.Dimension_MODEL_REPORTED, vllmv1.Dimension_SEMANTIC_ROUTER_MODEL:
		return tstLLMModels
	case vllmv1.Dimension_TOOL, vllmv1.Dimension_CALLED_TOOL, vllmv1.Dimension_REMOVED_TOOL:
		return tstLLMTools
	case vllmv1.Dimension_MODEL_PLUGIN, vllmv1.Dimension_GUARDRAIL_PLUGIN,
		vllmv1.Dimension_TOKEN_RATE_LIMIT_PLUGIN, vllmv1.Dimension_SEMANTIC_CACHE_PLUGIN,
		vllmv1.Dimension_SEMANTIC_ROUTER_PLUGIN:
		return tstLLMPlugins
	case vllmv1.Dimension_MODEL_SOURCE:
		return enumKeys([]string{"SOURCE_UNSET", "CONFIG", "PLUGIN", "SEMANTIC_ROUTER"})
	case vllmv1.Dimension_PROTOCOL:
		return enumKeys([]string{"OPENAI", "ANTHROPIC", "GEMINI", "BEDROCK"})
	case vllmv1.Dimension_OPERATION:
		return enumKeys([]string{"GENERATE", "EMBED", "MODERATE", "COUNT_TOKENS", "LIST_MODELS"})
	case vllmv1.Dimension_ROUTE:
		return enumKeys([]string{"CHAT_COMPLETIONS", "RESPONSES", "MESSAGES", "EMBEDDINGS", "GENERATE_CONTENT", "CONVERSE"})
	case vllmv1.Dimension_SOURCE:
		return enumKeys([]string{"UPSTREAM", "SEMANTIC_CACHE", "OCTELIUM"})
	case vllmv1.Dimension_USAGE_STATE:
		return enumKeys([]string{"STATE_UNSET", "COMPLETE", "PARTIAL"})
	case vllmv1.Dimension_ESTIMATE_QUALITY:
		return enumKeys([]string{"COMPLETE", "PARTIAL", "UNAVAILABLE"})
	case vllmv1.Dimension_GUARDRAIL_RESULT:
		return enumKeys([]string{"PASS", "MODIFIED", "DENIED", "ERROR"})
	case vllmv1.Dimension_GUARDRAIL_LEG:
		return enumKeys([]string{"REQUEST", "RESPONSE"})
	case vllmv1.Dimension_FINISH_REASON:
		return enumKeys([]string{"FINISH_REASON_UNSET", "STOP", "LENGTH", "TOOL_CALL", "CONTENT_FILTER", "ERROR"})
	case vllmv1.Dimension_FINISH_REASON_RAW:
		return enumKeys([]string{"stop", "length", "tool_calls", "end_turn", "max_tokens", "content_filter"})
	case vllmv1.Dimension_REASONING_EFFORT:
		return enumKeys([]string{"low", "medium", "high", "minimal"})
	case vllmv1.Dimension_STATUS:
		return enumKeys([]string{"ALLOWED", "DENIED"})
	case vllmv1.Dimension_DENY_REASON:
		return enumKeys([]string{"POLICY_MATCH", "NO_POLICY_MATCH", "SESSION_EXPIRED", "DEVICE_NOT_ACTIVE"})
	case vllmv1.Dimension_HTTP_STATUS_CODE:
		return enumKeys([]string{"200", "400", "403", "429", "500", "502"})
	case vllmv1.Dimension_HTTP_STATUS_CLASS:
		return enumKeys([]string{"SUCCESS", "CLIENT_ERROR", "SERVER_ERROR"})
	case vllmv1.Dimension_IS_STREAM, vllmv1.Dimension_IS_UPSTREAM_INVOKED,
		vllmv1.Dimension_HAS_IMAGE_INPUT, vllmv1.Dimension_HAS_AUDIO_INPUT:
		return enumKeys([]string{"true", "false"})
	case vllmv1.Dimension_USER_AGENT:
		return tstLLMUserAgents
	case vllmv1.Dimension_HTTP_PATH:
		return tstLLMPaths
	case vllmv1.Dimension_SEMANTIC_CACHE_RESULT:
		return enumKeys([]string{"EXACT_HIT", "SEMANTIC_HIT", "MISS", "BYPASS", "ERROR"})
	case vllmv1.Dimension_SEMANTIC_ROUTER_RESULT:
		return enumKeys([]string{"MATCH", "NO_MATCH", "BYPASS", "ERROR"})
	case vllmv1.Dimension_SEMANTIC_ROUTER_ROUTE:
		return enumKeys([]string{"coding", "support", "summarization", "analytics"})
	case vllmv1.Dimension_TOKEN_RATE_LIMIT_RESULT:
		return enumKeys([]string{"ALLOWED", "DENIED"})
	case vllmv1.Dimension_TOKEN_RATE_LIMIT_SCOPE:
		return enumKeys([]string{"USER", "SESSION", "DEVICE", "SERVICE"})
	default:
		return enumKeys([]string{"alpha", "beta", "gamma", "delta", "epsilon"})
	}
}

func (s *tstLLMService) dimensionItems(ctx context.Context,
	dimension vllmv1.Dimension, limit uint32) ([]*vllmv1.DimensionItem, uint64) {

	switch dimension {
	case vllmv1.Dimension_USER:
		itmList, err := s.octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{})
		if err == nil {
			return s.refItems(func() []*metav1.ObjectReference {
				var ret []*metav1.ObjectReference
				for _, itm := range itmList.Items {
					ret = append(ret, umetav1.GetObjectReference(itm))
				}
				return ret
			}(), limit)
		}
	case vllmv1.Dimension_SESSION:
		itmList, err := s.octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{})
		if err == nil {
			return s.refItems(func() []*metav1.ObjectReference {
				var ret []*metav1.ObjectReference
				for _, itm := range itmList.Items {
					ret = append(ret, umetav1.GetObjectReference(itm))
				}
				return ret
			}(), limit)
		}
	case vllmv1.Dimension_SERVICE:
		itmList, err := s.octeliumC.CoreC().ListService(ctx, &rmetav1.ListOptions{})
		if err == nil {
			return s.refItems(func() []*metav1.ObjectReference {
				var ret []*metav1.ObjectReference
				for _, itm := range itmList.Items {
					ret = append(ret, umetav1.GetObjectReference(itm))
				}
				return ret
			}(), limit)
		}
	case vllmv1.Dimension_NAMESPACE:
		itmList, err := s.octeliumC.CoreC().ListNamespace(ctx, &rmetav1.ListOptions{})
		if err == nil {
			return s.refItems(func() []*metav1.ObjectReference {
				var ret []*metav1.ObjectReference
				for _, itm := range itmList.Items {
					ret = append(ret, umetav1.GetObjectReference(itm))
				}
				return ret
			}(), limit)
		}
	case vllmv1.Dimension_DEVICE:
		itmList, err := s.octeliumC.CoreC().ListDevice(ctx, &rmetav1.ListOptions{})
		if err == nil {
			return s.refItems(func() []*metav1.ObjectReference {
				var ret []*metav1.ObjectReference
				for _, itm := range itmList.Items {
					ret = append(ret, umetav1.GetObjectReference(itm))
				}
				return ret
			}(), limit)
		}
	case vllmv1.Dimension_POLICY:
		itmList, err := s.octeliumC.CoreC().ListPolicy(ctx, &rmetav1.ListOptions{})
		if err == nil {
			return s.refItems(func() []*metav1.ObjectReference {
				var ret []*metav1.ObjectReference
				for _, itm := range itmList.Items {
					ret = append(ret, umetav1.GetObjectReference(itm))
				}
				return ret
			}(), limit)
		}
	case vllmv1.Dimension_REGION:
		itmList, err := s.octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{})
		if err == nil {
			return s.refItems(func() []*metav1.ObjectReference {
				var ret []*metav1.ObjectReference
				for _, itm := range itmList.Items {
					ret = append(ret, umetav1.GetObjectReference(itm))
				}
				return ret
			}(), limit)
		}
	}

	keys := tstLLMDimensionKeys(dimension)
	var ret []*vllmv1.DimensionItem
	base := uint64(4000)
	for i, key := range keys {
		if limit > 0 && uint32(len(ret)) >= limit {
			break
		}
		ret = append(ret, &vllmv1.DimensionItem{
			Key:   key,
			Stats: tstLLMStats(base / uint64(i+1)),
		})
	}

	return ret, uint64(len(keys))
}

func (s *tstLLMService) refItems(refs []*metav1.ObjectReference,
	limit uint32) ([]*vllmv1.DimensionItem, uint64) {
	var ret []*vllmv1.DimensionItem
	base := uint64(3200)
	for i, ref := range refs {
		if limit > 0 && uint32(len(ret)) >= limit {
			break
		}
		ret = append(ret, &vllmv1.DimensionItem{
			Key:   ref.Name,
			Ref:   ref,
			Stats: tstLLMStats(base / uint64(i+1)),
		})
	}

	return ret, uint64(len(refs))
}

func (s *tstLLMService) GetSummary(ctx context.Context,
	req *vllmv1.GetSummaryRequest) (*vllmv1.GetSummaryResponse, error) {

	ret := &vllmv1.GetSummaryResponse{
		Stats: tstLLMStats(48000),
	}

	for _, dimension := range req.Cardinalities {
		ret.Cardinalities = append(ret.Cardinalities, &vllmv1.CardinalityItem{
			Dimension: dimension,
			Count:     uint64(len(tstLLMDimensionKeys(dimension))),
		})
	}

	for _, breakdown := range req.Breakdowns {
		limit := breakdown.Limit
		if limit == 0 {
			limit = 10
		}
		items, totalCount := s.dimensionItems(ctx, breakdown.Dimension, limit)
		ret.Breakdowns = append(ret.Breakdowns, &vllmv1.Breakdown{
			Dimension:  breakdown.Dimension,
			Items:      items,
			TotalCount: totalCount,
			Other: func() *vllmv1.Stats {
				if totalCount > uint64(len(items)) {
					return tstLLMStats(1200)
				}
				return nil
			}(),
		})
	}

	return ret, nil
}

func tstLLMIntervalDuration(interval *metav1.Duration) time.Duration {
	if interval == nil {
		return time.Hour
	}
	switch tp := interval.Type.(type) {
	case *metav1.Duration_Milliseconds:
		return time.Duration(tp.Milliseconds) * time.Millisecond
	case *metav1.Duration_Seconds:
		return time.Duration(tp.Seconds) * time.Second
	case *metav1.Duration_Minutes:
		return time.Duration(tp.Minutes) * time.Minute
	case *metav1.Duration_Hours:
		return time.Duration(tp.Hours) * time.Hour
	case *metav1.Duration_Days:
		return time.Duration(tp.Days) * 24 * time.Hour
	case *metav1.Duration_Weeks:
		return time.Duration(tp.Weeks) * 7 * 24 * time.Hour
	case *metav1.Duration_Months:
		return time.Duration(tp.Months) * 30 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func tstLLMDataPoints(interval *metav1.Duration, base uint64) []*vllmv1.GetDataPointResponse_DataPoint {
	d := tstLLMIntervalDuration(interval)
	count := 48

	var ret []*vllmv1.GetDataPointResponse_DataPoint
	current := base
	for i := range count {
		current = tstLLMRand(base/2, base*2)
		ret = append(ret, &vllmv1.GetDataPointResponse_DataPoint{
			Timestamp: pbutils.Timestamp(time.Now().Add(time.Duration(i-count) * d)),
			Stats:     tstLLMStats(current),
		})
	}

	return ret
}

func (s *tstLLMService) GetDataPoint(ctx context.Context,
	req *vllmv1.GetDataPointRequest) (*vllmv1.GetDataPointResponse, error) {

	if req.GroupBy == vllmv1.Dimension_DIMENSION_UNSET {
		return &vllmv1.GetDataPointResponse{
			Series: []*vllmv1.GetDataPointResponse_Series{
				{
					Datapoints: tstLLMDataPoints(req.Interval, 900),
					Stats:      tstLLMStats(43000),
				},
			},
			TotalCount: 1,
		}, nil
	}

	limit := req.Limit
	if limit == 0 {
		limit = 5
	}

	items, totalCount := s.dimensionItems(ctx, req.GroupBy, limit)

	ret := &vllmv1.GetDataPointResponse{
		TotalCount: totalCount,
	}

	for i, item := range items {
		ret.Series = append(ret.Series, &vllmv1.GetDataPointResponse_Series{
			Key:        item.Key,
			Ref:        item.Ref,
			Datapoints: tstLLMDataPoints(req.Interval, 900/uint64(i+1)),
			Stats:      item.Stats,
		})
	}

	if totalCount > uint64(len(items)) {
		ret.Other = &vllmv1.GetDataPointResponse_Series{
			Datapoints: tstLLMDataPoints(req.Interval, 120),
			Stats:      tstLLMStats(2400),
		}
	}

	return ret, nil
}

func (s *tstLLMService) ListTopDimension(ctx context.Context,
	req *vllmv1.ListTopDimensionRequest) (*vllmv1.ListTopDimensionResponse, error) {

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	items, totalCount := s.dimensionItems(ctx, req.Dimension, limit)

	return &vllmv1.ListTopDimensionResponse{
		Items:      items,
		TotalCount: totalCount,
		Other: func() *vllmv1.Stats {
			if totalCount > uint64(len(items)) {
				return tstLLMStats(1200)
			}
			return nil
		}(),
	}, nil
}

func (s *tstLLMService) ListTopModel(ctx context.Context,
	req *vllmv1.ListTopModelRequest) (*vllmv1.ListTopModelResponse, error) {

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	ret := &vllmv1.ListTopModelResponse{
		TotalCount: uint64(len(tstLLMModels)),
	}

	base := uint64(9000)
	for i, model := range tstLLMModels {
		if uint32(len(ret.Items)) >= limit {
			break
		}
		count := base / uint64(i+1)
		ret.Items = append(ret.Items, &vllmv1.ListTopModelResponse_Item{
			Model:          model,
			Stats:          tstLLMStats(count),
			RequestedCount: count,
			EffectiveCount: count + count/9,
			ReportedCount:  count - count/12,
		})
	}

	if ret.TotalCount > uint64(len(ret.Items)) {
		ret.Other = tstLLMStats(800)
	}

	return ret, nil
}

func (s *tstLLMService) ListTopTool(ctx context.Context,
	req *vllmv1.ListTopToolRequest) (*vllmv1.ListTopToolResponse, error) {

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	ret := &vllmv1.ListTopToolResponse{
		TotalCount: uint64(len(tstLLMTools)),
	}

	base := uint64(5200)
	for i, tool := range tstLLMTools {
		if uint32(len(ret.Items)) >= limit {
			break
		}
		count := base / uint64(i+1)
		ret.Items = append(ret.Items, &vllmv1.ListTopToolResponse_Item{
			Tool:         tool,
			Stats:        tstLLMStats(count),
			OfferedCount: count,
			CalledCount:  count / 3,
			RemovedCount: count / 14,
		})
	}

	if ret.TotalCount > uint64(len(ret.Items)) {
		ret.Other = tstLLMStats(600)
	}

	return ret, nil
}

func (s *tstLLMService) ListTopUser(ctx context.Context,
	req *vllmv1.ListTopUserRequest) (*vllmv1.ListTopUserResponse, error) {

	itmList, err := s.octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := &vllmv1.ListTopUserResponse{
		TotalCount: uint64(len(itmList.Items)),
	}
	for i, itm := range itmList.Items[:min(len(itmList.Items), int(tstLLMLimit(req.Limit)))] {
		ret.Items = append(ret.Items, &vllmv1.ListTopUserResponse_Item{
			User:  itm,
			Stats: tstLLMStats(7000 / uint64(i+1)),
		})
	}

	return ret, nil
}

func (s *tstLLMService) ListTopSession(ctx context.Context,
	req *vllmv1.ListTopSessionRequest) (*vllmv1.ListTopSessionResponse, error) {

	itmList, err := s.octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := &vllmv1.ListTopSessionResponse{
		TotalCount: uint64(len(itmList.Items)),
	}
	for i, itm := range itmList.Items[:min(len(itmList.Items), int(tstLLMLimit(req.Limit)))] {
		ret.Items = append(ret.Items, &vllmv1.ListTopSessionResponse_Item{
			Session: itm,
			Stats:   tstLLMStats(6400 / uint64(i+1)),
		})
	}

	return ret, nil
}

func (s *tstLLMService) ListTopService(ctx context.Context,
	req *vllmv1.ListTopServiceRequest) (*vllmv1.ListTopServiceResponse, error) {

	itmList, err := s.octeliumC.CoreC().ListService(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := &vllmv1.ListTopServiceResponse{
		TotalCount: uint64(len(itmList.Items)),
	}
	for i, itm := range itmList.Items[:min(len(itmList.Items), int(tstLLMLimit(req.Limit)))] {
		ret.Items = append(ret.Items, &vllmv1.ListTopServiceResponse_Item{
			Service: itm,
			Stats:   tstLLMStats(8200 / uint64(i+1)),
		})
	}

	return ret, nil
}

func (s *tstLLMService) ListTopPolicy(ctx context.Context,
	req *vllmv1.ListTopPolicyRequest) (*vllmv1.ListTopPolicyResponse, error) {

	itmList, err := s.octeliumC.CoreC().ListPolicy(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ret := &vllmv1.ListTopPolicyResponse{
		TotalCount: uint64(len(itmList.Items)),
	}
	for i, itm := range itmList.Items[:min(len(itmList.Items), int(tstLLMLimit(req.Limit)))] {
		ret.Items = append(ret.Items, &vllmv1.ListTopPolicyResponse_Item{
			Policy: itm,
			Stats:  tstLLMStats(3600 / uint64(i+1)),
		})
	}

	return ret, nil
}

func tstLLMLimit(limit uint32) uint32 {
	if limit == 0 {
		return 10
	}
	return limit
}

func (s *tstLLMService) ListAccessLog(ctx context.Context,
	req *vllmv1.ListAccessLogRequest) (*vllmv1.ListAccessLogResponse, error) {

	sessList, err := s.octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	svcList, err := s.octeliumC.CoreC().ListService(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	itemsPerPage := req.ItemsPerPage
	if itemsPerPage == 0 {
		itemsPerPage = 25
	}

	ret := &vllmv1.ListAccessLogResponse{
		ListResponseMeta: &metav1.ListResponseMeta{
			Page:         req.Page,
			ItemsPerPage: itemsPerPage,
			TotalCount:   4000,
			HasMore:      req.Page < 20,
		},
	}

	for i := range int(itemsPerPage) {
		sess := sessList.Items[i%len(sessList.Items)]
		svc := svcList.Items[i%len(svcList.Items)]

		lg := vutils.GenerateLog()
		lg.Metadata.CreatedAt = pbutils.Timestamp(
			time.Now().Add(-time.Duration(i*3) * time.Minute))
		lg.Entry = &corev1.AccessLog_Entry{
			Common: &corev1.AccessLog_Entry_Common{
				StartedAt:    lg.Metadata.CreatedAt,
				EndedAt:      lg.Metadata.CreatedAt,
				Mode:         corev1.Service_Spec_LLM,
				SessionRef:   umetav1.GetObjectReference(sess),
				UserRef:      sess.Status.UserRef,
				DeviceRef:    sess.Status.DeviceRef,
				ServiceRef:   umetav1.GetObjectReference(svc),
				NamespaceRef: svc.Status.NamespaceRef,
				Status: func() corev1.AccessLog_Entry_Common_Status {
					if i%9 == 0 {
						return corev1.AccessLog_Entry_Common_DENIED
					}
					return corev1.AccessLog_Entry_Common_ALLOWED
				}(),
			},
			Info: &corev1.AccessLog_Entry_Info{
				Type: &corev1.AccessLog_Entry_Info_Llm{
					Llm: tstLLMInfo(i),
				},
			},
		}
		ret.Items = append(ret.Items, lg)
	}

	return ret, nil
}

func tstLLMInfo(i int) *corev1.AccessLog_Entry_Info_LLM {
	model := tstLLMModels[i%len(tstLLMModels)]
	inputTokens := tstLLMRand(300, 12000)
	outputTokens := tstLLMRand(40, 3200)
	isStream := i%2 == 0

	ret := &corev1.AccessLog_Entry_Info_LLM{
		Http: &corev1.AccessLog_Entry_Info_HTTP{
			Request: &corev1.AccessLog_Entry_Info_HTTP_Request{
				Method: "POST",
				Path:   tstLLMPaths[i%len(tstLLMPaths)],
			},
			Response: &corev1.AccessLog_Entry_Info_HTTP_Response{
				Code: func() uint32 {
					switch {
					case i%9 == 0:
						return 403
					case i%23 == 0:
						return 429
					case i%31 == 0:
						return 502
					default:
						return 200
					}
				}(),
			},
		},
		Type: func() corev1.AccessLog_Entry_Info_LLM_Type {
			if isStream {
				return corev1.AccessLog_Entry_Info_LLM_STREAM_END
			}
			return corev1.AccessLog_Entry_Info_LLM_COMPLETE
		}(),
		Protocol: corev1.Service_Spec_Config_LLM_Protocol(1 + i%4),
		Operation: func() corev1.Service_Spec_Config_LLM_Operation {
			if i%7 == 0 {
				return corev1.Service_Spec_Config_LLM_EMBED
			}
			return corev1.Service_Spec_Config_LLM_GENERATE
		}(),
		Route:             corev1.RequestContext_Request_LLM_Route(1 + i%13),
		Source:            corev1.AccessLog_Entry_Info_LLM_Source(1 + i%3),
		IsUpstreamInvoked: i%12 != 0,
		Model: &corev1.AccessLog_Entry_Info_LLM_Model{
			Requested: model,
			Effective: model,
			Reported:  model,
			Source: func() corev1.AccessLog_Entry_Info_LLM_Model_Source {
				if i%5 == 0 {
					return corev1.AccessLog_Entry_Info_LLM_Model_SEMANTIC_ROUTER
				}
				return corev1.AccessLog_Entry_Info_LLM_Model_SOURCE_UNSET
			}(),
		},
		Stream:               isStream,
		MaxOutputTokens:      4096,
		InputItemCount:       uint32(1 + i%14),
		HasImageInput:        i%9 == 1,
		HasAudioInput:        i%26 == 1,
		EstimatedInputTokens: inputTokens + inputTokens/10,
		EstimateQuality:      corev1.RequestContext_Request_LLM_COMPLETE,
		Usage: &corev1.AccessLog_Entry_Info_LLM_Usage{
			State:                 corev1.AccessLog_Entry_Info_LLM_Usage_COMPLETE,
			InputTokens:           inputTokens,
			OutputTokens:          outputTokens,
			TotalTokens:           inputTokens + outputTokens,
			CacheReadInputTokens:  inputTokens / 4,
			CacheWriteInputTokens: inputTokens / 11,
			ReasoningOutputTokens: outputTokens / 5,
		},
		ResponseID:   fmt.Sprintf("resp_%s", utilrand.GetRandomStringLowercase(20)),
		FinishReason: corev1.AccessLog_Entry_Info_LLM_FinishReason(1 + i%5),
		RawFinishReason: []string{
			"stop", "length", "tool_calls", "content_filter", "error",
		}[i%5],
		TimeToFirstToken: &metav1.Duration{
			Type: &metav1.Duration_Milliseconds{
				Milliseconds: uint32(tstLLMRand(80, 900)),
			},
		},
		EventCount: func() uint64 {
			if isStream {
				return tstLLMRand(30, 400)
			}
			return 0
		}(),
		Tools: &corev1.AccessLog_Entry_Info_LLM_Tools{
			Count:        uint32(i % 6),
			Names:        tstLLMTools[:i%6],
			RemovedCount: uint32(i % 2),
			RemovedNames: tstLLMTools[:i%2],
			CalledNames:  tstLLMTools[:i%3],
			CallCount:    uint32(i % 4),
		},
		Guardrails: []*corev1.AccessLog_Entry_Info_LLM_Guardrail{
			{
				Result: corev1.AccessLog_Entry_Info_LLM_Guardrail_Result(1 + i%4),
				Leg:    corev1.Service_Spec_Config_LLM_Plugin_Guardrail_Leg(1 + i%2),
				Plugin: tstLLMPlugins[i%3],
			},
		},
		TokenRateLimit: &corev1.AccessLog_Entry_Info_LLM_TokenRateLimit{
			Result: corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_Result(1 + i%2),
			Plugin: "quota-per-user",
			Scope:  corev1.Service_Spec_Config_LLM_Plugin_TokenRateLimit_Scope(1 + i%3),
		},
		SemanticCache: &corev1.AccessLog_Entry_Info_LLM_SemanticCache{
			Result:     corev1.AccessLog_Entry_Info_LLM_SemanticCache_Result(1 + i%5),
			Similarity: 0.5 + float32(i%40)/100,
			IsStored:   i%7 == 0,
			Plugin:     "prompt-cache",
		},
		SemanticRouter: &corev1.AccessLog_Entry_Info_LLM_SemanticRouter{
			Result:     corev1.AccessLog_Entry_Info_LLM_SemanticRouter_Result(1 + i%4),
			Route:      []string{"coding", "support", "summarization", "analytics"}[i%4],
			Similarity: 0.6 + float32(i%30)/100,
			Model:      model,
			Plugin:     "cost-router",
		},
		Reasoning: &corev1.AccessLog_Entry_Info_LLM_Reasoning{
			IsDisabled:  i%15 == 0,
			Effort:      []string{"low", "medium", "high"}[i%3],
			TokenBudget: 8192,
		},
	}

	return ret
}
