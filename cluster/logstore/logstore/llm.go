// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
)

const (
	llmDefaultTopLimit       = 10
	llmMaxTopLimit           = 100
	llmDefaultSeriesLimit    = 5
	llmMaxSeriesLimit        = 20
	llmDefaultBreakdownLimit = 5
	llmMaxBreakdowns         = 8
	llmMaxCardinalities      = 12
	llmMaxFilterValues       = 64
	llmMaxDataPointBuckets   = 10000
)

func llmJSONStr(pth string) string {
	return fmt.Sprintf(`json_extract_string(rsc, '$.%s')`, pth)
}

func llmJSONEnum(pth string, unset fmt.Stringer) string {
	return fmt.Sprintf(`COALESCE(json_extract_string(rsc, '$.%s'), '%s')`, pth, unset.String())
}

func llmJSONInt(pth string) string {
	return fmt.Sprintf(`COALESCE(TRY_CAST(json_extract_string(rsc, '$.%s') AS BIGINT), 0)`, pth)
}

func llmJSONFloat(pth string) string {
	return fmt.Sprintf(`COALESCE(TRY_CAST(json_extract_string(rsc, '$.%s') AS DOUBLE), 0)`, pth)
}

func llmJSONBool(pth string) string {
	return fmt.Sprintf(`COALESCE(TRY_CAST(json_extract_string(rsc, '$.%s') AS BOOLEAN), false)`, pth)
}

func llmJSONList(pth string) string {
	return fmt.Sprintf(`json_extract_string(rsc, '$.%s[*]')`, pth)
}

func llmJSONObjList(pth, field string) string {
	return fmt.Sprintf(`json_extract_string(rsc, '$.%s[*].%s')`, pth, field)
}

func llmJSONExists(pth string) string {
	return fmt.Sprintf(`json_extract(rsc, '$.%s') IS NOT NULL`, pth)
}

func llmListLen(expr string) string {
	return fmt.Sprintf(`COALESCE(len(%s), 0)`, expr)
}

func llmListHas(expr, val string) string {
	return fmt.Sprintf(`COALESCE(list_contains(%s, '%s'), false)`, expr, val)
}

var (
	llmExprCreatedAt   = llmJSONStr("metadata.createdAt")
	llmExprStatus      = llmJSONEnum("entry.common.status", corev1.AccessLog_Entry_Common_STATUS_UNSET)
	llmExprReasonType  = llmJSONEnum("entry.common.reason.type", corev1.AccessLog_Entry_Common_Reason_TYPE_UNKNOWN_REASON)
	llmExprIsPublic    = llmJSONBool("entry.common.isPublic")
	llmExprIsAnonymous = llmJSONBool("entry.common.isAnonymous")

	llmExprType            = llmJSONStr("entry.info.llm.type")
	llmExprProtocol        = llmJSONEnum("entry.info.llm.protocol", corev1.Service_Spec_Config_LLM_PROTOCOL_UNSET)
	llmExprOperation       = llmJSONEnum("entry.info.llm.operation", corev1.Service_Spec_Config_LLM_OPERATION_UNSET)
	llmExprRoute           = llmJSONEnum("entry.info.llm.route", corev1.RequestContext_Request_LLM_ROUTE_UNSET)
	llmExprSource          = llmJSONEnum("entry.info.llm.source", corev1.AccessLog_Entry_Info_LLM_SOURCE_UNSET)
	llmExprEstimateQuality = llmJSONEnum("entry.info.llm.estimateQuality", corev1.RequestContext_Request_LLM_ESTIMATE_QUALITY_UNSET)
	llmExprFinishReason    = llmJSONEnum("entry.info.llm.finishReason", corev1.AccessLog_Entry_Info_LLM_FINISH_REASON_UNSET)
	llmExprFinishReasonRaw = llmJSONStr("entry.info.llm.rawFinishReason")
	llmExprStream          = llmJSONBool("entry.info.llm.stream")
	llmExprUpstreamInvoked = llmJSONBool("entry.info.llm.isUpstreamInvoked")
	llmExprImageInput      = llmJSONBool("entry.info.llm.hasImageInput")
	llmExprAudioInput      = llmJSONBool("entry.info.llm.hasAudioInput")
	llmExprInputItemCount  = llmJSONInt("entry.info.llm.inputItemCount")

	llmExprModelRequested = llmJSONStr("entry.info.llm.model.requested")
	llmExprModelEffective = llmJSONStr("entry.info.llm.model.effective")
	llmExprModelReported  = llmJSONStr("entry.info.llm.model.reported")
	llmExprModelSource    = llmJSONEnum("entry.info.llm.model.source", corev1.AccessLog_Entry_Info_LLM_Model_SOURCE_UNSET)
	llmExprModelPlugin    = llmJSONStr("entry.info.llm.model.plugin")

	llmExprGuardrailResults = llmJSONObjList("entry.info.llm.guardrails", "result")
	llmExprGuardrailLegs    = llmJSONObjList("entry.info.llm.guardrails", "leg")
	llmExprGuardrailPlugins = llmJSONObjList("entry.info.llm.guardrails", "plugin")
	llmExprGuardrailCount   = llmListLen(llmExprGuardrailResults)

	llmExprRateLimitResult = llmJSONEnum("entry.info.llm.tokenRateLimit.result",
		corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_RESULT_UNSET)
	llmExprRateLimitPlugin = llmJSONStr("entry.info.llm.tokenRateLimit.plugin")
	llmExprRateLimitScope  = llmJSONEnum("entry.info.llm.tokenRateLimit.scope",
		corev1.Service_Spec_Config_LLM_Plugin_TokenRateLimit_SCOPE_UNSET)

	llmExprCacheResult = llmJSONEnum("entry.info.llm.semanticCache.result",
		corev1.AccessLog_Entry_Info_LLM_SemanticCache_RESULT_UNSET)
	llmExprCachePlugin     = llmJSONStr("entry.info.llm.semanticCache.plugin")
	llmExprCacheStored     = llmJSONBool("entry.info.llm.semanticCache.isStored")
	llmExprCacheSimilarity = llmJSONFloat("entry.info.llm.semanticCache.similarity")

	llmExprRouterResult = llmJSONEnum("entry.info.llm.semanticRouter.result",
		corev1.AccessLog_Entry_Info_LLM_SemanticRouter_RESULT_UNSET)
	llmExprRouterRoute  = llmJSONStr("entry.info.llm.semanticRouter.route")
	llmExprRouterPlugin = llmJSONStr("entry.info.llm.semanticRouter.plugin")
	llmExprRouterModel  = llmJSONStr("entry.info.llm.semanticRouter.model")

	llmExprReasoningEffort   = llmJSONStr("entry.info.llm.reasoning.effort")
	llmExprReasoningDisabled = llmJSONBool("entry.info.llm.reasoning.isDisabled")
	llmExprReasoningExists   = llmJSONExists("entry.info.llm.reasoning")
	llmExprReasoningManaged  = fmt.Sprintf(`%s AND NOT %s`, llmExprReasoningExists, llmExprReasoningDisabled)

	llmExprToolsCount       = llmJSONInt("entry.info.llm.tools.count")
	llmExprToolsRemoved     = llmJSONInt("entry.info.llm.tools.removedCount")
	llmExprToolNames        = llmJSONList("entry.info.llm.tools.names")
	llmExprRemovedToolNames = llmJSONList("entry.info.llm.tools.removedNames")
	llmExprCalledToolNames  = llmJSONList("entry.info.llm.tools.calledNames")
	llmExprCalledToolsCount = llmListLen(llmExprCalledToolNames)
	llmExprToolCallCount    = llmJSONInt("entry.info.llm.tools.callCount")
	llmExprToolsTruncated   = llmJSONBool("entry.info.llm.tools.isCalledNamesTruncated")
	llmExprHasToolCalls     = fmt.Sprintf(`(%s > 0 OR %s > 0)`, llmExprToolCallCount, llmExprCalledToolsCount)

	llmExprUsageExists     = llmJSONExists("entry.info.llm.usage")
	llmExprUsageState      = llmJSONEnum("entry.info.llm.usage.state", corev1.AccessLog_Entry_Info_LLM_Usage_STATE_UNSET)
	llmExprTokensInput     = llmJSONInt("entry.info.llm.usage.inputTokens")
	llmExprTokensOutput    = llmJSONInt("entry.info.llm.usage.outputTokens")
	llmExprTokensTotal     = llmJSONInt("entry.info.llm.usage.totalTokens")
	llmExprTokensCacheRead = llmJSONInt("entry.info.llm.usage.cacheReadInputTokens")
	llmExprTokensCacheWrit = llmJSONInt("entry.info.llm.usage.cacheWriteInputTokens")
	llmExprTokensReasoning = llmJSONInt("entry.info.llm.usage.reasoningOutputTokens")
	llmExprTokensEstimated = llmJSONInt("entry.info.llm.estimatedInputTokens")

	llmExprDiscarded = fmt.Sprintf(`%s AND %s <> '%s'`, llmExprUpstreamInvoked, llmExprSource,
		corev1.AccessLog_Entry_Info_LLM_UPSTREAM.String())

	llmExprEventCount       = llmJSONInt("entry.info.llm.eventCount")
	llmExprHTTPCode         = llmJSONInt("entry.info.llm.http.response.code")
	llmExprHTTPPath         = llmJSONStr("entry.info.llm.http.request.path")
	llmExprUserAgent        = llmJSONStr("entry.info.llm.http.request.userAgent")
	llmExprReqBodyBytes     = llmJSONInt("entry.info.llm.http.request.bodyBytes")
	llmExprRespBodyBytes    = llmJSONInt("entry.info.llm.http.response.bodyBytes")
	llmExprHTTPStatusClass  = fmt.Sprintf(`CAST(%s / 100 AS BIGINT)`, llmExprHTTPCode)
	llmExprEntryLLM         = llmJSONExists("entry.info.llm")
	llmExprLatencyMs        = `CAST(date_diff('millisecond', CAST(json_extract(rsc, '$.entry.common.startedAt') AS TIMESTAMP), CAST(json_extract(rsc, '$.entry.common.endedAt') AS TIMESTAMP)) AS DOUBLE)`
	llmExprTimeToFirstToken = fmt.Sprintf(`COALESCE(TRY_CAST(%s AS DOUBLE), TRY_CAST(%s AS DOUBLE) * 1000)`,
		llmJSONStr("entry.info.llm.timeToFirstToken.milliseconds"),
		llmJSONStr("entry.info.llm.timeToFirstToken.seconds"))
)

type llmStatColumn struct {
	alias string
	expr  string
}

func llmCountFilter(cond string) string {
	return fmt.Sprintf(`COUNT(*) FILTER (WHERE %s)`, cond)
}

func llmCountEq(expr, val string) string {
	return llmCountFilter(fmt.Sprintf(`%s = '%s'`, expr, val))
}

func llmSumInt(expr string) string {
	return fmt.Sprintf(`COALESCE(CAST(SUM(%s) AS BIGINT), 0)`, expr)
}

func llmSumIntFilter(expr, cond string) string {
	return fmt.Sprintf(`COALESCE(CAST(SUM(%s) FILTER (WHERE %s) AS BIGINT), 0)`, expr, cond)
}

var llmStatColumns = []llmStatColumn{
	{"count_total", `COUNT(*)`},
	{"count_allowed", llmCountEq(llmExprStatus, corev1.AccessLog_Entry_Common_ALLOWED.String())},
	{"count_denied", llmCountEq(llmExprStatus, corev1.AccessLog_Entry_Common_DENIED.String())},
	{"count_succeeded", llmCountFilter(fmt.Sprintf(`%s > 0 AND %s < 400`, llmExprHTTPCode, llmExprHTTPCode))},
	{"count_failed", llmCountFilter(fmt.Sprintf(`%s >= 400`, llmExprHTTPCode))},
	{"count_client_error", llmCountFilter(fmt.Sprintf(`%s >= 400 AND %s < 500`, llmExprHTTPCode, llmExprHTTPCode))},
	{"count_server_error", llmCountFilter(fmt.Sprintf(`%s >= 500`, llmExprHTTPCode))},
	{"count_streamed", llmCountFilter(llmExprStream)},

	{"count_upstream_invoked", llmCountFilter(llmExprUpstreamInvoked)},
	{"count_discarded_inference", llmCountFilter(llmExprDiscarded)},

	{"count_source_upstream", llmCountEq(llmExprSource, corev1.AccessLog_Entry_Info_LLM_UPSTREAM.String())},
	{"count_source_semantic_cache", llmCountEq(llmExprSource, corev1.AccessLog_Entry_Info_LLM_SEMANTIC_CACHE.String())},
	{"count_source_octelium", llmCountEq(llmExprSource, corev1.AccessLog_Entry_Info_LLM_OCTELIUM.String())},

	{"count_with_usage", llmCountFilter(llmExprUsageExists)},
	{"count_without_usage", llmCountFilter(fmt.Sprintf(`NOT (%s)`, llmExprUsageExists))},
	{"count_usage_complete", llmCountEq(llmExprUsageState, corev1.AccessLog_Entry_Info_LLM_Usage_COMPLETE.String())},
	{"count_usage_partial", llmCountEq(llmExprUsageState, corev1.AccessLog_Entry_Info_LLM_Usage_PARTIAL.String())},

	{"count_guardrail_inspected", llmCountFilter(fmt.Sprintf(`%s > 0`, llmExprGuardrailCount))},
	{"count_guardrail_passed", llmCountFilter(fmt.Sprintf(`%s > 0 AND NOT %s AND NOT %s AND NOT %s`,
		llmExprGuardrailCount,
		llmListHas(llmExprGuardrailResults, corev1.AccessLog_Entry_Info_LLM_Guardrail_MODIFIED.String()),
		llmListHas(llmExprGuardrailResults, corev1.AccessLog_Entry_Info_LLM_Guardrail_DENIED.String()),
		llmListHas(llmExprGuardrailResults, corev1.AccessLog_Entry_Info_LLM_Guardrail_ERROR.String())))},
	{"count_guardrail_modified", llmCountFilter(
		llmListHas(llmExprGuardrailResults, corev1.AccessLog_Entry_Info_LLM_Guardrail_MODIFIED.String()))},
	{"count_guardrail_denied", llmCountFilter(
		llmListHas(llmExprGuardrailResults, corev1.AccessLog_Entry_Info_LLM_Guardrail_DENIED.String()))},
	{"count_guardrail_error", llmCountFilter(
		llmListHas(llmExprGuardrailResults, corev1.AccessLog_Entry_Info_LLM_Guardrail_ERROR.String()))},

	{"count_rate_limit_allowed", llmCountEq(llmExprRateLimitResult,
		corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_ALLOWED.String())},
	{"count_rate_limit_denied", llmCountEq(llmExprRateLimitResult,
		corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_DENIED.String())},

	{"count_cache_exact_hit", llmCountEq(llmExprCacheResult,
		corev1.AccessLog_Entry_Info_LLM_SemanticCache_EXACT_HIT.String())},
	{"count_cache_semantic_hit", llmCountEq(llmExprCacheResult,
		corev1.AccessLog_Entry_Info_LLM_SemanticCache_SEMANTIC_HIT.String())},
	{"count_cache_miss", llmCountEq(llmExprCacheResult,
		corev1.AccessLog_Entry_Info_LLM_SemanticCache_MISS.String())},
	{"count_cache_bypass", llmCountEq(llmExprCacheResult,
		corev1.AccessLog_Entry_Info_LLM_SemanticCache_BYPASS.String())},
	{"count_cache_error", llmCountEq(llmExprCacheResult,
		corev1.AccessLog_Entry_Info_LLM_SemanticCache_ERROR.String())},
	{"count_cache_stored", llmCountFilter(llmExprCacheStored)},

	{"count_router_match", llmCountEq(llmExprRouterResult,
		corev1.AccessLog_Entry_Info_LLM_SemanticRouter_MATCH.String())},
	{"count_router_no_match", llmCountEq(llmExprRouterResult,
		corev1.AccessLog_Entry_Info_LLM_SemanticRouter_NO_MATCH.String())},
	{"count_router_bypass", llmCountEq(llmExprRouterResult,
		corev1.AccessLog_Entry_Info_LLM_SemanticRouter_BYPASS.String())},
	{"count_router_error", llmCountEq(llmExprRouterResult,
		corev1.AccessLog_Entry_Info_LLM_SemanticRouter_ERROR.String())},

	{"count_model_overridden", llmCountFilter(fmt.Sprintf(`%s <> '%s'`, llmExprModelSource,
		corev1.AccessLog_Entry_Info_LLM_Model_SOURCE_UNSET.String()))},
	{"count_model_routed", llmCountEq(llmExprModelSource,
		corev1.AccessLog_Entry_Info_LLM_Model_SEMANTIC_ROUTER.String())},

	{"count_with_tools", llmCountFilter(fmt.Sprintf(`%s > 0`, llmExprToolsCount))},
	{"count_with_tool_calls", llmCountFilter(llmExprHasToolCalls)},
	{"count_with_tools_removed", llmCountFilter(fmt.Sprintf(`%s > 0`, llmExprToolsRemoved))},
	{"count_with_called_tools_truncated", llmCountFilter(llmExprToolsTruncated)},

	{"count_with_managed_reasoning", llmCountFilter(llmExprReasoningManaged)},
	{"count_reasoning_disabled", llmCountFilter(llmExprReasoningDisabled)},

	{"count_with_image_input", llmCountFilter(llmExprImageInput)},
	{"count_with_audio_input", llmCountFilter(llmExprAudioInput)},

	{"count_finish_stop", llmCountEq(llmExprFinishReason, corev1.AccessLog_Entry_Info_LLM_STOP.String())},
	{"count_finish_length", llmCountEq(llmExprFinishReason, corev1.AccessLog_Entry_Info_LLM_LENGTH.String())},
	{"count_finish_tool_call", llmCountEq(llmExprFinishReason, corev1.AccessLog_Entry_Info_LLM_TOOL_CALL.String())},
	{"count_finish_content_filter", llmCountEq(llmExprFinishReason,
		corev1.AccessLog_Entry_Info_LLM_CONTENT_FILTER.String())},
	{"count_finish_error", llmCountEq(llmExprFinishReason, corev1.AccessLog_Entry_Info_LLM_ERROR.String())},

	{"tokens_input", llmSumInt(llmExprTokensInput)},
	{"tokens_output", llmSumInt(llmExprTokensOutput)},
	{"tokens_total", llmSumInt(llmExprTokensTotal)},
	{"tokens_cache_read_input", llmSumInt(llmExprTokensCacheRead)},
	{"tokens_cache_write_input", llmSumInt(llmExprTokensCacheWrit)},
	{"tokens_reasoning_output", llmSumInt(llmExprTokensReasoning)},
	{"tokens_estimated_input", llmSumInt(llmExprTokensEstimated)},
	{"tokens_discarded", llmSumIntFilter(llmExprTokensTotal, llmExprDiscarded)},

	{"latency_count", fmt.Sprintf(`COUNT(%s)`, llmExprLatencyMs)},
	{"latency_sum", fmt.Sprintf(`COALESCE(SUM(%s), 0)`, llmExprLatencyMs)},
	{"latency_avg", fmt.Sprintf(`COALESCE(AVG(%s), 0)`, llmExprLatencyMs)},
	{"latency_max", fmt.Sprintf(`COALESCE(MAX(%s), 0)`, llmExprLatencyMs)},

	{"ttft_count", fmt.Sprintf(`COUNT(%s)`, llmExprTimeToFirstToken)},
	{"ttft_sum", fmt.Sprintf(`COALESCE(SUM(%s), 0)`, llmExprTimeToFirstToken)},
	{"ttft_avg", fmt.Sprintf(`COALESCE(AVG(%s), 0)`, llmExprTimeToFirstToken)},
	{"ttft_max", fmt.Sprintf(`COALESCE(MAX(%s), 0)`, llmExprTimeToFirstToken)},

	{"stream_events", llmSumInt(llmExprEventCount)},
	{"tools_offered", llmSumInt(llmExprToolsCount)},
	{"tools_removed", llmSumInt(llmExprToolsRemoved)},
	{"tool_calls", llmSumInt(llmExprToolCallCount)},
	{"distinct_tools_called", llmSumInt(llmExprCalledToolsCount)},
	{"input_items", llmSumInt(llmExprInputItemCount)},
	{"request_body_bytes", llmSumInt(llmExprReqBodyBytes)},
	{"response_body_bytes", llmSumInt(llmExprRespBodyBytes)},
}

var llmStatQuantileColumns = []llmStatColumn{
	{"latency_p50", fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.5), 0)`, llmExprLatencyMs)},
	{"latency_p90", fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.9), 0)`, llmExprLatencyMs)},
	{"latency_p95", fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.95), 0)`, llmExprLatencyMs)},
	{"latency_p99", fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.99), 0)`, llmExprLatencyMs)},
	{"ttft_p50", fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.5), 0)`, llmExprTimeToFirstToken)},
	{"ttft_p90", fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.9), 0)`, llmExprTimeToFirstToken)},
	{"ttft_p95", fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.95), 0)`, llmExprTimeToFirstToken)},
	{"ttft_p99", fmt.Sprintf(`COALESCE(quantile_cont(%s, 0.99), 0)`, llmExprTimeToFirstToken)},
}

func llmStatsSelects(quantiles bool) []any {
	var ret []any
	for _, c := range llmStatColumns {
		ret = append(ret, goqu.L(c.expr).As(c.alias))
	}
	if quantiles {
		for _, c := range llmStatQuantileColumns {
			ret = append(ret, goqu.L(c.expr).As(c.alias))
		}
	}

	return ret
}

type llmDurationStats struct {
	count int64
	sum   float64
	avg   float64
	max   float64
	p50   float64
	p90   float64
	p95   float64
	p99   float64
}

type llmStats struct {
	total       int64
	allowed     int64
	denied      int64
	succeeded   int64
	failed      int64
	clientError int64
	serverError int64
	streamed    int64

	upstreamInvoked    int64
	discardedInference int64

	sourceUpstream int64
	sourceCache    int64
	sourceOctelium int64

	withUsage     int64
	withoutUsage  int64
	usageComplete int64
	usagePartial  int64

	guardrailInspected int64
	guardrailPassed    int64
	guardrailModified  int64
	guardrailDenied    int64
	guardrailError     int64

	rateLimitAllowed int64
	rateLimitDenied  int64

	cacheExactHit    int64
	cacheSemanticHit int64
	cacheMiss        int64
	cacheBypass      int64
	cacheError       int64
	cacheStored      int64

	routerMatch   int64
	routerNoMatch int64
	routerBypass  int64
	routerError   int64

	modelOverridden int64
	modelRouted     int64

	withTools             int64
	withToolCalls         int64
	withToolsRemoved      int64
	withCalledToolsTrunc  int64
	withManagedReasoning  int64
	reasoningDisabled     int64
	withImageInput        int64
	withAudioInput        int64
	finishStop            int64
	finishLength          int64
	finishToolCall        int64
	finishContentFilter   int64
	finishError           int64
	tokensInput           int64
	tokensOutput          int64
	tokensTotal           int64
	tokensCacheReadInput  int64
	tokensCacheWriteInput int64
	tokensReasoningOutput int64
	tokensEstimatedInput  int64
	tokensDiscarded       int64

	latency llmDurationStats
	ttft    llmDurationStats

	streamEvents        int64
	toolsOffered        int64
	toolsRemoved        int64
	toolCalls           int64
	distinctToolsCalled int64
	inputItems          int64
	requestBodyBytes    int64
	responseBodyBytes   int64
}

func (v *llmStats) scanDest(quantiles bool) []any {
	ret := []any{
		&v.total, &v.allowed, &v.denied, &v.succeeded, &v.failed,
		&v.clientError, &v.serverError, &v.streamed,

		&v.upstreamInvoked, &v.discardedInference,
		&v.sourceUpstream, &v.sourceCache, &v.sourceOctelium,
		&v.withUsage, &v.withoutUsage, &v.usageComplete, &v.usagePartial,

		&v.guardrailInspected, &v.guardrailPassed, &v.guardrailModified,
		&v.guardrailDenied, &v.guardrailError,

		&v.rateLimitAllowed, &v.rateLimitDenied,

		&v.cacheExactHit, &v.cacheSemanticHit, &v.cacheMiss, &v.cacheBypass,
		&v.cacheError, &v.cacheStored,

		&v.routerMatch, &v.routerNoMatch, &v.routerBypass, &v.routerError,

		&v.modelOverridden, &v.modelRouted,

		&v.withTools, &v.withToolCalls, &v.withToolsRemoved, &v.withCalledToolsTrunc,
		&v.withManagedReasoning, &v.reasoningDisabled,
		&v.withImageInput, &v.withAudioInput,

		&v.finishStop, &v.finishLength, &v.finishToolCall,
		&v.finishContentFilter, &v.finishError,

		&v.tokensInput, &v.tokensOutput, &v.tokensTotal,
		&v.tokensCacheReadInput, &v.tokensCacheWriteInput,
		&v.tokensReasoningOutput, &v.tokensEstimatedInput, &v.tokensDiscarded,

		&v.latency.count, &v.latency.sum, &v.latency.avg, &v.latency.max,
		&v.ttft.count, &v.ttft.sum, &v.ttft.avg, &v.ttft.max,

		&v.streamEvents, &v.toolsOffered, &v.toolsRemoved, &v.toolCalls,
		&v.distinctToolsCalled, &v.inputItems,
		&v.requestBodyBytes, &v.responseBodyBytes,
	}

	if quantiles {
		ret = append(ret,
			&v.latency.p50, &v.latency.p90, &v.latency.p95, &v.latency.p99,
			&v.ttft.p50, &v.ttft.p90, &v.ttft.p95, &v.ttft.p99)
	}

	return ret
}

func (v *llmDurationStats) toPB() *vllmv1.Stats_DurationStats {
	return &vllmv1.Stats_DurationStats{
		Count: uint64(v.count),
		SumMs: v.sum,
		AvgMs: v.avg,
		MaxMs: v.max,
		P50Ms: v.p50,
		P90Ms: v.p90,
		P95Ms: v.p95,
		P99Ms: v.p99,
	}
}

func (v *llmStats) toPB() *vllmv1.Stats {
	return &vllmv1.Stats{
		Requests: &vllmv1.Stats_Requests{
			Total:       uint64(v.total),
			Allowed:     uint64(v.allowed),
			Denied:      uint64(v.denied),
			Succeeded:   uint64(v.succeeded),
			Failed:      uint64(v.failed),
			ClientError: uint64(v.clientError),
			ServerError: uint64(v.serverError),
			Streamed:    uint64(v.streamed),

			UpstreamInvoked:    uint64(v.upstreamInvoked),
			DiscardedInference: uint64(v.discardedInference),

			SourceUpstream:      uint64(v.sourceUpstream),
			SourceSemanticCache: uint64(v.sourceCache),
			SourceOctelium:      uint64(v.sourceOctelium),

			WithUsage:     uint64(v.withUsage),
			WithoutUsage:  uint64(v.withoutUsage),
			UsageComplete: uint64(v.usageComplete),
			UsagePartial:  uint64(v.usagePartial),

			GuardrailInspected: uint64(v.guardrailInspected),
			GuardrailPassed:    uint64(v.guardrailPassed),
			GuardrailModified:  uint64(v.guardrailModified),
			GuardrailDenied:    uint64(v.guardrailDenied),
			GuardrailError:     uint64(v.guardrailError),

			TokenRateLimitAllowed: uint64(v.rateLimitAllowed),
			TokenRateLimitDenied:  uint64(v.rateLimitDenied),

			CacheExactHit:    uint64(v.cacheExactHit),
			CacheSemanticHit: uint64(v.cacheSemanticHit),
			CacheMiss:        uint64(v.cacheMiss),
			CacheBypass:      uint64(v.cacheBypass),
			CacheError:       uint64(v.cacheError),
			CacheStored:      uint64(v.cacheStored),

			RouterMatch:   uint64(v.routerMatch),
			RouterNoMatch: uint64(v.routerNoMatch),
			RouterBypass:  uint64(v.routerBypass),
			RouterError:   uint64(v.routerError),

			ModelOverridden: uint64(v.modelOverridden),
			ModelRouted:     uint64(v.modelRouted),

			WithTools:                uint64(v.withTools),
			WithToolCalls:            uint64(v.withToolCalls),
			WithToolsRemoved:         uint64(v.withToolsRemoved),
			WithCalledToolsTruncated: uint64(v.withCalledToolsTrunc),

			WithManagedReasoning: uint64(v.withManagedReasoning),
			ReasoningDisabled:    uint64(v.reasoningDisabled),

			WithImageInput: uint64(v.withImageInput),
			WithAudioInput: uint64(v.withAudioInput),

			FinishStop:          uint64(v.finishStop),
			FinishLength:        uint64(v.finishLength),
			FinishToolCall:      uint64(v.finishToolCall),
			FinishContentFilter: uint64(v.finishContentFilter),
			FinishError:         uint64(v.finishError),
		},
		Tokens: &vllmv1.Stats_Tokens{
			Input:           uint64(v.tokensInput),
			Output:          uint64(v.tokensOutput),
			Total:           uint64(v.tokensTotal),
			CacheReadInput:  uint64(v.tokensCacheReadInput),
			CacheWriteInput: uint64(v.tokensCacheWriteInput),
			ReasoningOutput: uint64(v.tokensReasoningOutput),
			EstimatedInput:  uint64(v.tokensEstimatedInput),
			Discarded:       uint64(v.tokensDiscarded),
		},
		Latency:             v.latency.toPB(),
		TimeToFirstToken:    v.ttft.toPB(),
		StreamEvents:        uint64(v.streamEvents),
		ToolsOffered:        uint64(v.toolsOffered),
		ToolsRemoved:        uint64(v.toolsRemoved),
		ToolCalls:           uint64(v.toolCalls),
		DistinctToolsCalled: uint64(v.distinctToolsCalled),
		InputItems:          uint64(v.inputItems),
		RequestBodyBytes:    uint64(v.requestBodyBytes),
		ResponseBodyBytes:   uint64(v.responseBodyBytes),
	}
}

func newLLMEmptyStats() *vllmv1.Stats {
	return (&llmStats{}).toPB()
}

func llmMetricAlias(metric vllmv1.Metric) (string, error) {
	switch metric {
	case vllmv1.Metric_METRIC_UNSET, vllmv1.Metric_REQUESTS:
		return "count_total", nil
	case vllmv1.Metric_ALLOWED_REQUESTS:
		return "count_allowed", nil
	case vllmv1.Metric_DENIED_REQUESTS:
		return "count_denied", nil
	case vllmv1.Metric_FAILED_REQUESTS:
		return "count_failed", nil
	case vllmv1.Metric_STREAMED_REQUESTS:
		return "count_streamed", nil
	case vllmv1.Metric_CACHED_REQUESTS:
		return "count_source_semantic_cache", nil
	case vllmv1.Metric_GUARDRAIL_DENIED_REQUESTS:
		return "count_guardrail_denied", nil
	case vllmv1.Metric_TOOL_CALL_REQUESTS:
		return "count_with_tool_calls", nil
	case vllmv1.Metric_UPSTREAM_INVOKED_REQUESTS:
		return "count_upstream_invoked", nil
	case vllmv1.Metric_TOKEN_RATE_LIMIT_DENIED_REQUESTS:
		return "count_rate_limit_denied", nil
	case vllmv1.Metric_INPUT_TOKENS:
		return "tokens_input", nil
	case vllmv1.Metric_OUTPUT_TOKENS:
		return "tokens_output", nil
	case vllmv1.Metric_TOTAL_TOKENS:
		return "tokens_total", nil
	case vllmv1.Metric_CACHE_READ_INPUT_TOKENS:
		return "tokens_cache_read_input", nil
	case vllmv1.Metric_CACHE_WRITE_INPUT_TOKENS:
		return "tokens_cache_write_input", nil
	case vllmv1.Metric_REASONING_OUTPUT_TOKENS:
		return "tokens_reasoning_output", nil
	case vllmv1.Metric_ESTIMATED_INPUT_TOKENS:
		return "tokens_estimated_input", nil
	case vllmv1.Metric_DISCARDED_TOKENS:
		return "tokens_discarded", nil
	case vllmv1.Metric_LATENCY_AVG:
		return "latency_avg", nil
	case vllmv1.Metric_LATENCY_MAX:
		return "latency_max", nil
	case vllmv1.Metric_LATENCY_P95:
		return "latency_p95", nil
	case vllmv1.Metric_TIME_TO_FIRST_TOKEN_AVG:
		return "ttft_avg", nil
	case vllmv1.Metric_TIME_TO_FIRST_TOKEN_P95:
		return "ttft_p95", nil
	case vllmv1.Metric_TOOL_CALLS:
		return "tool_calls", nil
	case vllmv1.Metric_TOOLS_OFFERED:
		return "tools_offered", nil
	case vllmv1.Metric_STREAM_EVENTS:
		return "stream_events", nil
	case vllmv1.Metric_REQUEST_BODY_BYTES:
		return "request_body_bytes", nil
	case vllmv1.Metric_RESPONSE_BODY_BYTES:
		return "response_body_bytes", nil
	default:
		return "", grpcutils.InvalidArg("Invalid Metric: %s", metric.String())
	}
}

func llmMetricNeedsQuantiles(metric vllmv1.Metric) bool {
	switch metric {
	case vllmv1.Metric_LATENCY_P95, vllmv1.Metric_TIME_TO_FIRST_TOKEN_P95:
		return true
	default:
		return false
	}
}

func llmEnumStrings[T fmt.Stringer](vals []T) []string {
	ret := make([]string, 0, len(vals))
	for _, v := range vals {
		ret = append(ret, v.String())
	}

	return ret
}

func llmAppendBoolFilter(filters []exp.Expression, cond string, val *bool) []exp.Expression {
	if val == nil {
		return filters
	}
	if *val {
		return append(filters, goqu.L(cond))
	}

	return append(filters, goqu.L(fmt.Sprintf(`NOT (%s)`, cond)))
}

func llmAppendStringFilter(filters []exp.Expression, expr string, vals []string, name string) ([]exp.Expression, error) {
	if len(vals) < 1 {
		return filters, nil
	}
	if len(vals) > llmMaxFilterValues {
		return nil, grpcutils.InvalidArg("Too many %s filter values", name)
	}
	for _, val := range vals {
		if val == "" {
			return nil, grpcutils.InvalidArg("Empty %s filter value", name)
		}
		if len(val) > 256 {
			return nil, grpcutils.InvalidArg("%s filter value is too long", name)
		}
	}

	return append(filters, goqu.L(expr).In(vals)), nil
}

func llmAppendListFilter(filters []exp.Expression, expr string, vals []string, name string) ([]exp.Expression, error) {
	if len(vals) < 1 {
		return filters, nil
	}
	if len(vals) > llmMaxFilterValues {
		return nil, grpcutils.InvalidArg("Too many %s filter values", name)
	}

	var conds []exp.Expression
	for _, val := range vals {
		if val == "" {
			return nil, grpcutils.InvalidArg("Empty %s filter value", name)
		}
		if len(val) > 256 {
			return nil, grpcutils.InvalidArg("%s filter value is too long", name)
		}
		conds = append(conds, goqu.L(fmt.Sprintf(`list_contains(%s, ?)`, expr), val))
	}

	return append(filters, goqu.Or(conds...)), nil
}

func llmAppendEnumListFilter[T fmt.Stringer](filters []exp.Expression, expr string, vals []T) []exp.Expression {
	if len(vals) < 1 {
		return filters
	}

	var conds []exp.Expression
	for _, val := range vals {
		conds = append(conds, goqu.L(llmListHas(expr, val.String())))
	}

	return append(filters, goqu.Or(conds...))
}

func llmModelExpr(field vllmv1.ModelField) (string, error) {
	switch field {
	case vllmv1.ModelField_MODEL_FIELD_UNSET, vllmv1.ModelField_EFFECTIVE:
		return llmExprModelEffective, nil
	case vllmv1.ModelField_REQUESTED:
		return llmExprModelRequested, nil
	case vllmv1.ModelField_REPORTED:
		return llmExprModelReported, nil
	default:
		return "", grpcutils.InvalidArg("Invalid ModelField")
	}
}

func llmToolExpr(scope vllmv1.ToolScope) (string, error) {
	switch scope {
	case vllmv1.ToolScope_TOOL_SCOPE_UNSET, vllmv1.ToolScope_OFFERED:
		return llmExprToolNames, nil
	case vllmv1.ToolScope_CALLED:
		return llmExprCalledToolNames, nil
	case vllmv1.ToolScope_REMOVED:
		return llmExprRemovedToolNames, nil
	default:
		return "", grpcutils.InvalidArg("Invalid ToolScope")
	}
}

func getLLMAggregateFilters(f *vllmv1.Filter) ([]exp.Expression, error) {
	if f.GetEntryScope() == vllmv1.EntryScope_ALL {
		return nil, grpcutils.InvalidArg(
			"The ALL EntryScope counts a streamed request twice. It is only available for the ListAccessLog method")
	}

	return getLLMFilters(f)
}

func getLLMFilters(f *vllmv1.Filter) ([]exp.Expression, error) {
	if f == nil {
		f = &vllmv1.Filter{}
	}

	filters := []exp.Expression{goqu.L(llmExprEntryLLM)}

	switch f.EntryScope {
	case vllmv1.EntryScope_ENTRY_SCOPE_UNSET, vllmv1.EntryScope_TERMINAL:
		filters = append(filters, goqu.L(fmt.Sprintf(`COALESCE(%s, '') <> '%s'`, llmExprType,
			corev1.AccessLog_Entry_Info_LLM_STREAM_START.String())))
	case vllmv1.EntryScope_STREAM_START:
		filters = append(filters, goqu.L(llmExprType).Eq(corev1.AccessLog_Entry_Info_LLM_STREAM_START.String()))
	case vllmv1.EntryScope_ALL:
	default:
		return nil, grpcutils.InvalidArg("Invalid EntryScope")
	}

	var err error

	filters, err = appendRefFilter(filters, f.UserRef, nil, "entry.common.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, f.DeviceRef, nil, "entry.common.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, f.SessionRef, nil, "entry.common.sessionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, f.ServiceRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}, "entry.common.serviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, f.NamespaceRef, nil, "entry.common.namespaceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, f.RegionRef, nil, "entry.common.regionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, f.PolicyRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 8,
	}, "entry.common.reason.details.policyMatch.policy.policyRef")
	if err != nil {
		return nil, err
	}

	switch f.Status {
	case corev1.AccessLog_Entry_Common_ALLOWED, corev1.AccessLog_Entry_Common_DENIED:
		filters = append(filters, goqu.L(llmExprStatus).Eq(f.Status.String()))
	}

	if f.From != nil {
		filters = append(filters, goqu.L(llmExprCreatedAt).Gte(f.From.AsTime().UTC().Format(time.RFC3339Nano)))
	}
	if f.To != nil {
		filters = append(filters, goqu.L(llmExprCreatedAt).Lt(f.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if len(f.Protocols) > 0 {
		filters = append(filters, goqu.L(llmExprProtocol).In(llmEnumStrings(f.Protocols)))
	}
	if len(f.Operations) > 0 {
		filters = append(filters, goqu.L(llmExprOperation).In(llmEnumStrings(f.Operations)))
	}
	if len(f.Routes) > 0 {
		filters = append(filters, goqu.L(llmExprRoute).In(llmEnumStrings(f.Routes)))
	}
	if len(f.Sources) > 0 {
		filters = append(filters, goqu.L(llmExprSource).In(llmEnumStrings(f.Sources)))
	}
	if len(f.UsageStates) > 0 {
		filters = append(filters, goqu.L(llmExprUsageState).In(llmEnumStrings(f.UsageStates)))
	}
	if len(f.ModelSources) > 0 {
		filters = append(filters, goqu.L(llmExprModelSource).In(llmEnumStrings(f.ModelSources)))
	}
	if len(f.EstimateQualities) > 0 {
		filters = append(filters, goqu.L(llmExprEstimateQuality).In(llmEnumStrings(f.EstimateQualities)))
	}
	if len(f.FinishReasons) > 0 {
		filters = append(filters, goqu.L(llmExprFinishReason).In(llmEnumStrings(f.FinishReasons)))
	}
	if len(f.TokenRateLimitResults) > 0 {
		filters = append(filters, goqu.L(llmExprRateLimitResult).In(llmEnumStrings(f.TokenRateLimitResults)))
	}
	if len(f.TokenRateLimitScopes) > 0 {
		filters = append(filters, goqu.L(llmExprRateLimitScope).In(llmEnumStrings(f.TokenRateLimitScopes)))
	}
	if len(f.SemanticCacheResults) > 0 {
		filters = append(filters, goqu.L(llmExprCacheResult).In(llmEnumStrings(f.SemanticCacheResults)))
	}
	if len(f.SemanticRouterResults) > 0 {
		filters = append(filters, goqu.L(llmExprRouterResult).In(llmEnumStrings(f.SemanticRouterResults)))
	}

	filters = llmAppendEnumListFilter(filters, llmExprGuardrailResults, f.GuardrailResults)
	filters = llmAppendEnumListFilter(filters, llmExprGuardrailLegs, f.GuardrailLegs)

	if len(f.Models) > 0 {
		modelExpr, err := llmModelExpr(f.ModelField)
		if err != nil {
			return nil, err
		}
		filters, err = llmAppendStringFilter(filters, modelExpr, f.Models, "model")
		if err != nil {
			return nil, err
		}
	}

	filters, err = llmAppendStringFilter(filters, llmExprModelPlugin, f.ModelPlugins, "model plugin")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendStringFilter(filters, llmExprRateLimitPlugin, f.TokenRateLimitPlugins, "TokenRateLimit plugin")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendStringFilter(filters, llmExprCachePlugin, f.SemanticCachePlugins, "SemanticCache plugin")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendStringFilter(filters, llmExprRouterRoute, f.SemanticRouterRoutes, "SemanticRouter route")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendStringFilter(filters, llmExprRouterPlugin, f.SemanticRouterPlugins, "SemanticRouter plugin")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendStringFilter(filters, llmExprFinishReasonRaw, f.RawFinishReasons, "raw finish reason")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendStringFilter(filters, llmExprUserAgent, f.UserAgents, "user agent")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendStringFilter(filters, llmExprHTTPPath, f.HttpPaths, "HTTP path")
	if err != nil {
		return nil, err
	}

	filters, err = llmAppendListFilter(filters, llmExprGuardrailPlugins, f.GuardrailPlugins, "guardrail plugin")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendListFilter(filters, llmExprToolNames, f.Tools, "tool")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendListFilter(filters, llmExprCalledToolNames, f.CalledTools, "called tool")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendListFilter(filters, llmExprRemovedToolNames, f.RemovedTools, "removed tool")
	if err != nil {
		return nil, err
	}

	if len(f.HttpStatusCodes) > 0 {
		if len(f.HttpStatusCodes) > llmMaxFilterValues {
			return nil, grpcutils.InvalidArg("Too many HTTP status code filter values")
		}
		var codes []int64
		for _, code := range f.HttpStatusCodes {
			if code < 100 || code > 599 {
				return nil, grpcutils.InvalidArg("Invalid HTTP status code: %d", code)
			}
			codes = append(codes, int64(code))
		}
		filters = append(filters, goqu.L(llmExprHTTPCode).In(codes))
	}

	if len(f.HttpStatusClasses) > 0 {
		var classes []int64
		for _, class := range f.HttpStatusClasses {
			switch class {
			case vllmv1.HTTPStatusClass_SUCCESS:
				classes = append(classes, 2)
			case vllmv1.HTTPStatusClass_REDIRECT:
				classes = append(classes, 3)
			case vllmv1.HTTPStatusClass_CLIENT_ERROR:
				classes = append(classes, 4)
			case vllmv1.HTTPStatusClass_SERVER_ERROR:
				classes = append(classes, 5)
			default:
				return nil, grpcutils.InvalidArg("Invalid HTTPStatusClass")
			}
		}
		filters = append(filters, goqu.L(llmExprHTTPStatusClass).In(classes))
	}

	filters = llmAppendBoolFilter(filters, llmExprStream, f.Stream)
	filters = llmAppendBoolFilter(filters, llmExprUpstreamInvoked, f.IsUpstreamInvoked)
	filters = llmAppendBoolFilter(filters, llmExprUsageExists, f.HasUsage)
	filters = llmAppendBoolFilter(filters, fmt.Sprintf(`%s > 0`, llmExprToolsCount), f.HasTools)
	filters = llmAppendBoolFilter(filters, llmExprHasToolCalls, f.HasToolCalls)
	filters = llmAppendBoolFilter(filters, fmt.Sprintf(`%s > 0`, llmExprToolsRemoved), f.HasToolsRemoved)
	filters = llmAppendBoolFilter(filters, llmExprReasoningManaged, f.HasManagedReasoning)
	filters = llmAppendBoolFilter(filters, llmExprReasoningDisabled, f.IsReasoningDisabled)
	filters = llmAppendBoolFilter(filters, llmExprImageInput, f.HasImageInput)
	filters = llmAppendBoolFilter(filters, llmExprAudioInput, f.HasAudioInput)
	filters = llmAppendBoolFilter(filters, llmExprCacheStored, f.IsCacheStored)
	filters = llmAppendBoolFilter(filters, llmExprIsPublic, f.IsPublic)
	filters = llmAppendBoolFilter(filters, llmExprIsAnonymous, f.IsAnonymous)

	if f.MinTotalTokens != nil {
		filters = append(filters, goqu.L(llmExprTokensTotal).Gte(int64(f.GetMinTotalTokens())))
	}
	if f.MaxTotalTokens != nil {
		filters = append(filters, goqu.L(llmExprTokensTotal).Lte(int64(f.GetMaxTotalTokens())))
	}
	if f.MinLatencyMs != nil {
		filters = append(filters, goqu.L(llmExprLatencyMs).Gte(float64(f.GetMinLatencyMs())))
	}
	if f.MaxLatencyMs != nil {
		filters = append(filters, goqu.L(llmExprLatencyMs).Lte(float64(f.GetMaxLatencyMs())))
	}

	return filters, nil
}

type llmDimension struct {
	key   string
	name  string
	kind  string
	multi bool
}

func llmUnnestDimension(expr string) *llmDimension {
	return &llmDimension{key: fmt.Sprintf(`unnest(%s)`, expr), multi: true}
}

func getLLMDimension(dim vllmv1.Dimension) (*llmDimension, error) {
	switch dim {
	case vllmv1.Dimension_MODEL:
		return &llmDimension{key: llmExprModelEffective}, nil
	case vllmv1.Dimension_MODEL_REQUESTED:
		return &llmDimension{key: llmExprModelRequested}, nil
	case vllmv1.Dimension_MODEL_REPORTED:
		return &llmDimension{key: llmExprModelReported}, nil
	case vllmv1.Dimension_MODEL_SOURCE:
		return &llmDimension{key: llmExprModelSource}, nil
	case vllmv1.Dimension_MODEL_PLUGIN:
		return &llmDimension{key: llmExprModelPlugin}, nil
	case vllmv1.Dimension_PROTOCOL:
		return &llmDimension{key: llmExprProtocol}, nil
	case vllmv1.Dimension_OPERATION:
		return &llmDimension{key: llmExprOperation}, nil
	case vllmv1.Dimension_ROUTE:
		return &llmDimension{key: llmExprRoute}, nil
	case vllmv1.Dimension_SOURCE:
		return &llmDimension{key: llmExprSource}, nil
	case vllmv1.Dimension_USAGE_STATE:
		return &llmDimension{key: llmExprUsageState}, nil
	case vllmv1.Dimension_ESTIMATE_QUALITY:
		return &llmDimension{key: llmExprEstimateQuality}, nil
	case vllmv1.Dimension_GUARDRAIL_RESULT:
		return llmUnnestDimension(llmExprGuardrailResults), nil
	case vllmv1.Dimension_GUARDRAIL_PLUGIN:
		return llmUnnestDimension(llmExprGuardrailPlugins), nil
	case vllmv1.Dimension_GUARDRAIL_LEG:
		return llmUnnestDimension(llmExprGuardrailLegs), nil
	case vllmv1.Dimension_TOKEN_RATE_LIMIT_RESULT:
		return &llmDimension{key: llmExprRateLimitResult}, nil
	case vllmv1.Dimension_TOKEN_RATE_LIMIT_PLUGIN:
		return &llmDimension{key: llmExprRateLimitPlugin}, nil
	case vllmv1.Dimension_TOKEN_RATE_LIMIT_SCOPE:
		return &llmDimension{key: llmExprRateLimitScope}, nil
	case vllmv1.Dimension_SEMANTIC_CACHE_RESULT:
		return &llmDimension{key: llmExprCacheResult}, nil
	case vllmv1.Dimension_SEMANTIC_CACHE_PLUGIN:
		return &llmDimension{key: llmExprCachePlugin}, nil
	case vllmv1.Dimension_SEMANTIC_ROUTER_RESULT:
		return &llmDimension{key: llmExprRouterResult}, nil
	case vllmv1.Dimension_SEMANTIC_ROUTER_ROUTE:
		return &llmDimension{key: llmExprRouterRoute}, nil
	case vllmv1.Dimension_SEMANTIC_ROUTER_PLUGIN:
		return &llmDimension{key: llmExprRouterPlugin}, nil
	case vllmv1.Dimension_SEMANTIC_ROUTER_MODEL:
		return &llmDimension{key: llmExprRouterModel}, nil
	case vllmv1.Dimension_FINISH_REASON:
		return &llmDimension{key: llmExprFinishReason}, nil
	case vllmv1.Dimension_FINISH_REASON_RAW:
		return &llmDimension{key: llmExprFinishReasonRaw}, nil
	case vllmv1.Dimension_REASONING_EFFORT:
		return &llmDimension{key: llmExprReasoningEffort}, nil
	case vllmv1.Dimension_TOOL:
		return llmUnnestDimension(llmExprToolNames), nil
	case vllmv1.Dimension_CALLED_TOOL:
		return llmUnnestDimension(llmExprCalledToolNames), nil
	case vllmv1.Dimension_REMOVED_TOOL:
		return llmUnnestDimension(llmExprRemovedToolNames), nil
	case vllmv1.Dimension_STATUS:
		return &llmDimension{key: llmExprStatus}, nil
	case vllmv1.Dimension_DENY_REASON:
		return &llmDimension{key: llmExprReasonType}, nil
	case vllmv1.Dimension_HTTP_STATUS_CODE:
		return &llmDimension{key: fmt.Sprintf(`CAST(%s AS VARCHAR)`, llmExprHTTPCode)}, nil
	case vllmv1.Dimension_HTTP_STATUS_CLASS:
		return &llmDimension{key: fmt.Sprintf(
			`CASE WHEN %s >= 500 THEN 'SERVER_ERROR' WHEN %s >= 400 THEN 'CLIENT_ERROR' WHEN %s >= 300 THEN 'REDIRECT' WHEN %s >= 200 THEN 'SUCCESS' END`,
			llmExprHTTPCode, llmExprHTTPCode, llmExprHTTPCode, llmExprHTTPCode)}, nil
	case vllmv1.Dimension_HTTP_PATH:
		return &llmDimension{key: llmExprHTTPPath}, nil
	case vllmv1.Dimension_IS_STREAM:
		return &llmDimension{key: fmt.Sprintf(`CAST(%s AS VARCHAR)`, llmExprStream)}, nil
	case vllmv1.Dimension_IS_UPSTREAM_INVOKED:
		return &llmDimension{key: fmt.Sprintf(`CAST(%s AS VARCHAR)`, llmExprUpstreamInvoked)}, nil
	case vllmv1.Dimension_HAS_IMAGE_INPUT:
		return &llmDimension{key: fmt.Sprintf(`CAST(%s AS VARCHAR)`, llmExprImageInput)}, nil
	case vllmv1.Dimension_HAS_AUDIO_INPUT:
		return &llmDimension{key: fmt.Sprintf(`CAST(%s AS VARCHAR)`, llmExprAudioInput)}, nil
	case vllmv1.Dimension_USER_AGENT:
		return &llmDimension{key: llmExprUserAgent}, nil
	case vllmv1.Dimension_USER:
		return &llmDimension{
			key:  llmJSONStr("entry.common.userRef.uid"),
			name: llmJSONStr("entry.common.userRef.name"),
			kind: ucorev1.KindUser,
		}, nil
	case vllmv1.Dimension_SESSION:
		return &llmDimension{
			key:  llmJSONStr("entry.common.sessionRef.uid"),
			name: llmJSONStr("entry.common.sessionRef.name"),
			kind: ucorev1.KindSession,
		}, nil
	case vllmv1.Dimension_DEVICE:
		return &llmDimension{
			key:  llmJSONStr("entry.common.deviceRef.uid"),
			name: llmJSONStr("entry.common.deviceRef.name"),
			kind: ucorev1.KindDevice,
		}, nil
	case vllmv1.Dimension_SERVICE:
		return &llmDimension{
			key:  llmJSONStr("entry.common.serviceRef.uid"),
			name: llmJSONStr("entry.common.serviceRef.name"),
			kind: ucorev1.KindService,
		}, nil
	case vllmv1.Dimension_NAMESPACE:
		return &llmDimension{
			key:  llmJSONStr("entry.common.namespaceRef.uid"),
			name: llmJSONStr("entry.common.namespaceRef.name"),
			kind: ucorev1.KindNamespace,
		}, nil
	case vllmv1.Dimension_REGION:
		return &llmDimension{
			key:  llmJSONStr("entry.common.regionRef.uid"),
			name: llmJSONStr("entry.common.regionRef.name"),
			kind: ucorev1.KindRegion,
		}, nil
	case vllmv1.Dimension_POLICY:
		return &llmDimension{
			key:  llmJSONStr("entry.common.reason.details.policyMatch.policy.policyRef.uid"),
			name: llmJSONStr("entry.common.reason.details.policyMatch.policy.policyRef.name"),
			kind: ucorev1.KindPolicy,
		}, nil
	default:
		return nil, grpcutils.InvalidArg("Invalid Dimension")
	}
}

func (d *llmDimension) getRef(uid, name string) *metav1.ObjectReference {
	if d.kind == "" {
		return nil
	}

	return &metav1.ObjectReference{
		ApiVersion: ucorev1.APIVersion,
		Kind:       d.kind,
		Uid:        uid,
		Name:       name,
	}
}
