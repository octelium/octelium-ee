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
	llmMaxFilterValues       = 64
)

func llmJSONStr(pth string) string {
	return fmt.Sprintf(`json_extract_string(rsc, '$.%s')`, pth)
}

func llmJSONInt(pth string) string {
	return fmt.Sprintf(`COALESCE(TRY_CAST(json_extract_string(rsc, '$.%s') AS BIGINT), 0)`, pth)
}

func llmJSONBool(pth string) string {
	return fmt.Sprintf(`COALESCE(TRY_CAST(json_extract_string(rsc, '$.%s') AS BOOLEAN), false)`, pth)
}

func llmJSONList(pth string) string {
	return fmt.Sprintf(`json_extract_string(rsc, '$.%s[*]')`, pth)
}

func llmJSONExists(pth string) string {
	return fmt.Sprintf(`json_extract(rsc, '$.%s') IS NOT NULL`, pth)
}

var (
	llmExprCreatedAt   = llmJSONStr("metadata.createdAt")
	llmExprStatus      = llmJSONStr("entry.common.status")
	llmExprReasonType  = llmJSONStr("entry.common.reason.type")
	llmExprIsPublic    = llmJSONBool("entry.common.isPublic")
	llmExprIsAnonymous = llmJSONBool("entry.common.isAnonymous")

	llmExprType            = llmJSONStr("entry.info.llm.type")
	llmExprProtocol        = llmJSONStr("entry.info.llm.protocol")
	llmExprOperation       = llmJSONStr("entry.info.llm.operation")
	llmExprSource          = llmJSONStr("entry.info.llm.source")
	llmExprUsageSource     = llmJSONStr("entry.info.llm.usage.source")
	llmExprEstimateQuality = llmJSONStr("entry.info.llm.estimateQuality")
	llmExprFinishReason    = llmJSONStr("entry.info.llm.finishReason")
	llmExprStream          = llmJSONBool("entry.info.llm.stream")

	llmExprModelRequested = llmJSONStr("entry.info.llm.model.requested")
	llmExprModelEffective = llmJSONStr("entry.info.llm.model.effective")
	llmExprModelReported  = llmJSONStr("entry.info.llm.model.reported")
	llmExprModelSource    = llmJSONStr("entry.info.llm.model.source")
	llmExprModelPlugin    = llmJSONStr("entry.info.llm.model.plugin")

	llmExprGuardrailResult = llmJSONStr("entry.info.llm.guardrail.result")
	llmExprGuardrailLeg    = llmJSONStr("entry.info.llm.guardrail.leg")
	llmExprGuardrailPlugin = llmJSONStr("entry.info.llm.guardrail.plugin")

	llmExprReasoningEffort   = llmJSONStr("entry.info.llm.reasoning.effort")
	llmExprReasoningDisabled = llmJSONBool("entry.info.llm.reasoning.isDisabled")
	llmExprReasoningExists   = llmJSONExists("entry.info.llm.reasoning")

	llmExprToolsCount    = llmJSONInt("entry.info.llm.tools.count")
	llmExprToolsRemoved  = llmJSONInt("entry.info.llm.tools.removedCount")
	llmExprToolNames     = llmJSONList("entry.info.llm.tools.names")
	llmExprCalledToolsL  = llmJSONList("entry.info.llm.tools.calledNames")
	llmExprCalledToolCnt = fmt.Sprintf(`COALESCE(len(%s), 0)`, llmExprCalledToolsL)

	llmExprTokensInput         = llmJSONInt("entry.info.llm.usage.inputTokens")
	llmExprTokensOutput        = llmJSONInt("entry.info.llm.usage.outputTokens")
	llmExprTokensTotal         = llmJSONInt("entry.info.llm.usage.totalTokens")
	llmExprTokensCacheRead     = llmJSONInt("entry.info.llm.usage.cacheReadInputTokens")
	llmExprTokensCacheCreation = llmJSONInt("entry.info.llm.usage.cacheCreationInputTokens")
	llmExprTokensReasoning     = llmJSONInt("entry.info.llm.usage.reasoningTokens")
	llmExprTokensEstimated     = llmJSONInt("entry.info.llm.estimatedInputTokens")

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

func llmSumInt(expr string) string {
	return fmt.Sprintf(`COALESCE(CAST(SUM(%s) AS BIGINT), 0)`, expr)
}

var llmStatColumns = []llmStatColumn{
	{"count_total", `COUNT(*)`},
	{"count_allowed", llmCountFilter(fmt.Sprintf(`%s = 'ALLOWED'`, llmExprStatus))},
	{"count_denied", llmCountFilter(fmt.Sprintf(`%s = 'DENIED'`, llmExprStatus))},
	{"count_succeeded", llmCountFilter(fmt.Sprintf(`%s > 0 AND %s < 400`, llmExprHTTPCode, llmExprHTTPCode))},
	{"count_failed", llmCountFilter(fmt.Sprintf(`%s >= 400`, llmExprHTTPCode))},
	{"count_client_error", llmCountFilter(fmt.Sprintf(`%s >= 400 AND %s < 500`, llmExprHTTPCode, llmExprHTTPCode))},
	{"count_server_error", llmCountFilter(fmt.Sprintf(`%s >= 500`, llmExprHTTPCode))},
	{"count_streamed", llmCountFilter(llmExprStream)},
	{"count_source_upstream", llmCountFilter(fmt.Sprintf(`%s = 'UPSTREAM'`, llmExprSource))},
	{"count_source_semantic_cache", llmCountFilter(fmt.Sprintf(`%s = 'SEMANTIC_CACHE'`, llmExprSource))},
	{"count_source_octelium", llmCountFilter(fmt.Sprintf(`%s = 'OCTELIUM'`, llmExprSource))},
	{"count_usage_provider", llmCountFilter(fmt.Sprintf(`%s = 'PROVIDER'`, llmExprUsageSource))},
	{"count_usage_estimated", llmCountFilter(fmt.Sprintf(`%s = 'ESTIMATED'`, llmExprUsageSource))},
	{"count_usage_partial", llmCountFilter(fmt.Sprintf(`%s = 'PARTIAL'`, llmExprUsageSource))},
	{"count_usage_cached", llmCountFilter(fmt.Sprintf(`%s = 'CACHED'`, llmExprUsageSource))},
	{"count_usage_unset", llmCountFilter(fmt.Sprintf(`%s IS NULL OR %s = 'SOURCE_UNSET'`, llmExprUsageSource, llmExprUsageSource))},
	{"count_guardrail_pass", llmCountFilter(fmt.Sprintf(`%s = 'PASS'`, llmExprGuardrailResult))},
	{"count_guardrail_modified", llmCountFilter(fmt.Sprintf(`%s = 'MODIFIED'`, llmExprGuardrailResult))},
	{"count_guardrail_denied", llmCountFilter(fmt.Sprintf(`%s = 'DENIED'`, llmExprGuardrailResult))},
	{"count_model_overridden", llmCountFilter(fmt.Sprintf(`%s IS NOT NULL AND %s <> 'SOURCE_UNSET'`, llmExprModelSource, llmExprModelSource))},
	{"count_model_routed", llmCountFilter(fmt.Sprintf(`%s = 'SEMANTIC_ROUTER'`, llmExprModelSource))},
	{"count_with_tools", llmCountFilter(fmt.Sprintf(`%s > 0`, llmExprToolsCount))},
	{"count_with_tool_calls", llmCountFilter(fmt.Sprintf(`%s > 0`, llmExprCalledToolCnt))},
	{"count_with_tools_removed", llmCountFilter(fmt.Sprintf(`%s > 0`, llmExprToolsRemoved))},
	{"count_with_reasoning", llmCountFilter(fmt.Sprintf(`%s AND NOT %s`, llmExprReasoningExists, llmExprReasoningDisabled))},
	{"count_reasoning_disabled", llmCountFilter(llmExprReasoningDisabled)},
	{"count_finished_by_length", llmCountFilter(fmt.Sprintf(`lower(%s) IN ('length', 'max_tokens')`, llmExprFinishReason))},

	{"tokens_input", llmSumInt(llmExprTokensInput)},
	{"tokens_output", llmSumInt(llmExprTokensOutput)},
	{"tokens_total", llmSumInt(llmExprTokensTotal)},
	{"tokens_cache_read_input", llmSumInt(llmExprTokensCacheRead)},
	{"tokens_cache_creation_input", llmSumInt(llmExprTokensCacheCreation)},
	{"tokens_reasoning", llmSumInt(llmExprTokensReasoning)},
	{"tokens_estimated_input", llmSumInt(llmExprTokensEstimated)},

	{"latency_count", fmt.Sprintf(`COUNT(%s)`, llmExprLatencyMs)},
	{"latency_sum", fmt.Sprintf(`COALESCE(SUM(%s), 0)`, llmExprLatencyMs)},
	{"latency_avg", fmt.Sprintf(`COALESCE(AVG(%s), 0)`, llmExprLatencyMs)},
	{"latency_max", fmt.Sprintf(`COALESCE(MAX(%s), 0)`, llmExprLatencyMs)},

	{"ttft_count", fmt.Sprintf(`COUNT(%s)`, llmExprTimeToFirstToken)},
	{"ttft_sum", fmt.Sprintf(`COALESCE(SUM(%s), 0)`, llmExprTimeToFirstToken)},
	{"ttft_avg", fmt.Sprintf(`COALESCE(AVG(%s), 0)`, llmExprTimeToFirstToken)},
	{"ttft_max", fmt.Sprintf(`COALESCE(MAX(%s), 0)`, llmExprTimeToFirstToken)},

	{"stream_events", llmSumInt(llmExprEventCount)},
	{"tools_declared", llmSumInt(llmExprToolsCount)},
	{"tools_removed", llmSumInt(llmExprToolsRemoved)},
	{"tool_calls", llmSumInt(llmExprCalledToolCnt)},
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
	total          int64
	allowed        int64
	denied         int64
	succeeded      int64
	failed         int64
	clientError    int64
	serverError    int64
	streamed       int64
	sourceUpstream int64
	sourceCache    int64
	sourceOctelium int64

	usageProvider  int64
	usageEstimated int64
	usagePartial   int64
	usageCached    int64
	usageUnset     int64

	guardrailPass     int64
	guardrailModified int64
	guardrailDenied   int64

	modelOverridden int64
	modelRouted     int64

	withTools        int64
	withToolCalls    int64
	withToolsRemoved int64

	withReasoning     int64
	reasoningDisabled int64
	finishedByLength  int64

	tokensInput         int64
	tokensOutput        int64
	tokensTotal         int64
	tokensCacheRead     int64
	tokensCacheCreation int64
	tokensReasoning     int64
	tokensEstimated     int64

	latency llmDurationStats
	ttft    llmDurationStats

	streamEvents      int64
	toolsDeclared     int64
	toolsRemoved      int64
	toolCalls         int64
	requestBodyBytes  int64
	responseBodyBytes int64
}

func (v *llmStats) scanDest(quantiles bool) []any {
	ret := []any{
		&v.total, &v.allowed, &v.denied, &v.succeeded, &v.failed,
		&v.clientError, &v.serverError, &v.streamed,
		&v.sourceUpstream, &v.sourceCache, &v.sourceOctelium,
		&v.usageProvider, &v.usageEstimated, &v.usagePartial, &v.usageCached, &v.usageUnset,
		&v.guardrailPass, &v.guardrailModified, &v.guardrailDenied,
		&v.modelOverridden, &v.modelRouted,
		&v.withTools, &v.withToolCalls, &v.withToolsRemoved,
		&v.withReasoning, &v.reasoningDisabled, &v.finishedByLength,

		&v.tokensInput, &v.tokensOutput, &v.tokensTotal,
		&v.tokensCacheRead, &v.tokensCacheCreation, &v.tokensReasoning, &v.tokensEstimated,

		&v.latency.count, &v.latency.sum, &v.latency.avg, &v.latency.max,
		&v.ttft.count, &v.ttft.sum, &v.ttft.avg, &v.ttft.max,

		&v.streamEvents, &v.toolsDeclared, &v.toolsRemoved, &v.toolCalls,
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
			Total:               uint64(v.total),
			Allowed:             uint64(v.allowed),
			Denied:              uint64(v.denied),
			Succeeded:           uint64(v.succeeded),
			Failed:              uint64(v.failed),
			ClientError:         uint64(v.clientError),
			ServerError:         uint64(v.serverError),
			Streamed:            uint64(v.streamed),
			SourceUpstream:      uint64(v.sourceUpstream),
			SourceSemanticCache: uint64(v.sourceCache),
			SourceOctelium:      uint64(v.sourceOctelium),
			UsageProvider:       uint64(v.usageProvider),
			UsageEstimated:      uint64(v.usageEstimated),
			UsagePartial:        uint64(v.usagePartial),
			UsageCached:         uint64(v.usageCached),
			UsageUnset:          uint64(v.usageUnset),
			GuardrailPass:       uint64(v.guardrailPass),
			GuardrailModified:   uint64(v.guardrailModified),
			GuardrailDenied:     uint64(v.guardrailDenied),
			ModelOverridden:     uint64(v.modelOverridden),
			ModelRouted:         uint64(v.modelRouted),
			WithTools:           uint64(v.withTools),
			WithToolCalls:       uint64(v.withToolCalls),
			WithToolsRemoved:    uint64(v.withToolsRemoved),
			WithReasoning:       uint64(v.withReasoning),
			ReasoningDisabled:   uint64(v.reasoningDisabled),
			FinishedByLength:    uint64(v.finishedByLength),
		},
		Tokens: &vllmv1.Stats_Tokens{
			Input:              uint64(v.tokensInput),
			Output:             uint64(v.tokensOutput),
			Total:              uint64(v.tokensTotal),
			CacheReadInput:     uint64(v.tokensCacheRead),
			CacheCreationInput: uint64(v.tokensCacheCreation),
			Reasoning:          uint64(v.tokensReasoning),
			EstimatedInput:     uint64(v.tokensEstimated),
		},
		Latency:           v.latency.toPB(),
		TimeToFirstToken:  v.ttft.toPB(),
		StreamEvents:      uint64(v.streamEvents),
		ToolsDeclared:     uint64(v.toolsDeclared),
		ToolsRemoved:      uint64(v.toolsRemoved),
		ToolCalls:         uint64(v.toolCalls),
		RequestBodyBytes:  uint64(v.requestBodyBytes),
		ResponseBodyBytes: uint64(v.responseBodyBytes),
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
	case vllmv1.Metric_INPUT_TOKENS:
		return "tokens_input", nil
	case vllmv1.Metric_OUTPUT_TOKENS:
		return "tokens_output", nil
	case vllmv1.Metric_TOTAL_TOKENS:
		return "tokens_total", nil
	case vllmv1.Metric_CACHE_READ_INPUT_TOKENS:
		return "tokens_cache_read_input", nil
	case vllmv1.Metric_CACHE_CREATION_INPUT_TOKENS:
		return "tokens_cache_creation_input", nil
	case vllmv1.Metric_REASONING_TOKENS:
		return "tokens_reasoning", nil
	case vllmv1.Metric_ESTIMATED_INPUT_TOKENS:
		return "tokens_estimated_input", nil
	case vllmv1.Metric_LATENCY_AVG:
		return "latency_avg", nil
	case vllmv1.Metric_LATENCY_P95:
		return "latency_p95", nil
	case vllmv1.Metric_TIME_TO_FIRST_TOKEN_AVG:
		return "ttft_avg", nil
	case vllmv1.Metric_TIME_TO_FIRST_TOKEN_P95:
		return "ttft_p95", nil
	case vllmv1.Metric_TOOL_CALLS:
		return "tool_calls", nil
	case vllmv1.Metric_STREAM_EVENTS:
		return "stream_events", nil
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

func getLLMFilters(f *vllmv1.Filter) ([]exp.Expression, error) {
	if f == nil {
		f = &vllmv1.Filter{}
	}

	filters := []exp.Expression{goqu.L(llmExprEntryLLM)}

	switch f.EntryScope {
	case vllmv1.EntryScope_ENTRY_SCOPE_UNSET, vllmv1.EntryScope_TERMINAL:
		filters = append(filters, goqu.L(fmt.Sprintf(`COALESCE(%s, '') <> 'STREAM_START'`, llmExprType)))
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
		filters = append(filters, goqu.L(llmExprCreatedAt).Lte(f.To.AsTime().UTC().Format(time.RFC3339Nano)))
	}

	if len(f.Protocols) > 0 {
		filters = append(filters, goqu.L(llmExprProtocol).In(llmEnumStrings(f.Protocols)))
	}
	if len(f.Operations) > 0 {
		filters = append(filters, goqu.L(llmExprOperation).In(llmEnumStrings(f.Operations)))
	}
	if len(f.Sources) > 0 {
		filters = append(filters, goqu.L(llmExprSource).In(llmEnumStrings(f.Sources)))
	}
	if len(f.UsageSources) > 0 {
		filters = append(filters, goqu.L(llmExprUsageSource).In(llmEnumStrings(f.UsageSources)))
	}
	if len(f.ModelSources) > 0 {
		filters = append(filters, goqu.L(llmExprModelSource).In(llmEnumStrings(f.ModelSources)))
	}
	if len(f.EstimateQualities) > 0 {
		filters = append(filters, goqu.L(llmExprEstimateQuality).In(llmEnumStrings(f.EstimateQualities)))
	}
	if len(f.GuardrailResults) > 0 {
		filters = append(filters, goqu.L(llmExprGuardrailResult).In(llmEnumStrings(f.GuardrailResults)))
	}
	if len(f.GuardrailLegs) > 0 {
		filters = append(filters, goqu.L(llmExprGuardrailLeg).In(llmEnumStrings(f.GuardrailLegs)))
	}

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
	filters, err = llmAppendStringFilter(filters, llmExprGuardrailPlugin, f.GuardrailPlugins, "guardrail plugin")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendStringFilter(filters, llmExprFinishReason, f.FinishReasons, "finish reason")
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

	filters, err = llmAppendListFilter(filters, llmExprToolNames, f.Tools, "tool")
	if err != nil {
		return nil, err
	}
	filters, err = llmAppendListFilter(filters, llmExprCalledToolsL, f.CalledTools, "called tool")
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
	filters = llmAppendBoolFilter(filters, fmt.Sprintf(`%s > 0`, llmExprToolsCount), f.HasTools)
	filters = llmAppendBoolFilter(filters, fmt.Sprintf(`%s > 0`, llmExprCalledToolCnt), f.HasToolCalls)
	filters = llmAppendBoolFilter(filters,
		fmt.Sprintf(`%s AND NOT %s`, llmExprReasoningExists, llmExprReasoningDisabled), f.HasReasoning)
	filters = llmAppendBoolFilter(filters, llmExprIsPublic, f.IsPublic)
	filters = llmAppendBoolFilter(filters, llmExprIsAnonymous, f.IsAnonymous)

	if f.MinTotalTokens > 0 {
		filters = append(filters, goqu.L(llmExprTokensTotal).Gte(int64(f.MinTotalTokens)))
	}
	if f.MaxTotalTokens > 0 {
		filters = append(filters, goqu.L(llmExprTokensTotal).Lte(int64(f.MaxTotalTokens)))
	}
	if f.MinLatencyMs > 0 {
		filters = append(filters, goqu.L(llmExprLatencyMs).Gte(float64(f.MinLatencyMs)))
	}
	if f.MaxLatencyMs > 0 {
		filters = append(filters, goqu.L(llmExprLatencyMs).Lte(float64(f.MaxLatencyMs)))
	}

	return filters, nil
}

type llmDimension struct {
	key   string
	name  string
	kind  string
	multi bool
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
	case vllmv1.Dimension_SOURCE:
		return &llmDimension{key: llmExprSource}, nil
	case vllmv1.Dimension_USAGE_SOURCE:
		return &llmDimension{key: llmExprUsageSource}, nil
	case vllmv1.Dimension_ESTIMATE_QUALITY:
		return &llmDimension{key: llmExprEstimateQuality}, nil
	case vllmv1.Dimension_GUARDRAIL_RESULT:
		return &llmDimension{key: llmExprGuardrailResult}, nil
	case vllmv1.Dimension_GUARDRAIL_PLUGIN:
		return &llmDimension{key: llmExprGuardrailPlugin}, nil
	case vllmv1.Dimension_GUARDRAIL_LEG:
		return &llmDimension{key: llmExprGuardrailLeg}, nil
	case vllmv1.Dimension_FINISH_REASON:
		return &llmDimension{key: llmExprFinishReason}, nil
	case vllmv1.Dimension_REASONING_EFFORT:
		return &llmDimension{key: llmExprReasoningEffort}, nil
	case vllmv1.Dimension_TOOL:
		return &llmDimension{key: fmt.Sprintf(`unnest(%s)`, llmExprToolNames), multi: true}, nil
	case vllmv1.Dimension_CALLED_TOOL:
		return &llmDimension{key: fmt.Sprintf(`unnest(%s)`, llmExprCalledToolsL), multi: true}, nil
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
