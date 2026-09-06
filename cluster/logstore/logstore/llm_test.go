// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vllmv1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/types"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type llmGuardrailOptions struct {
	Result corev1.AccessLog_Entry_Info_LLM_Guardrail_Result
	Leg    corev1.Service_Spec_Config_LLM_Plugin_Guardrail_Leg
	Plugin string
}

type llmAccessLogOptions struct {
	CreatedAt    time.Time
	Duration     time.Duration
	Status       corev1.AccessLog_Entry_Common_Status
	UserRef      *metav1.ObjectReference
	DeviceRef    *metav1.ObjectReference
	SessionRef   *metav1.ObjectReference
	ServiceRef   *metav1.ObjectReference
	NamespaceRef *metav1.ObjectReference
	RegionRef    *metav1.ObjectReference
	PolicyRef    *metav1.ObjectReference

	Type      corev1.AccessLog_Entry_Info_LLM_Type
	Protocol  corev1.Service_Spec_Config_LLM_Protocol
	Operation corev1.Service_Spec_Config_LLM_Operation
	Route     corev1.RequestContext_Request_LLM_Route
	Source    corev1.AccessLog_Entry_Info_LLM_Source

	IsUpstreamInvoked bool

	ModelRequested string
	ModelEffective string
	ModelReported  string
	ModelSource    corev1.AccessLog_Entry_Info_LLM_Model_Source
	ModelPlugin    string

	Stream         bool
	InputItemCount uint32
	HasImageInput  bool
	HasAudioInput  bool

	HasUsage              bool
	UsageState            corev1.AccessLog_Entry_Info_LLM_Usage_State
	InputTokens           uint64
	OutputTokens          uint64
	TotalTokens           uint64
	CacheReadInputTokens  uint64
	CacheWriteInputTokens uint64
	ReasoningOutputTokens uint64
	EstimatedInputTokens  uint64

	FinishReason       corev1.AccessLog_Entry_Info_LLM_FinishReason
	RawFinishReason    string
	TimeToFirstTokenMS uint32
	EventCount         uint64

	Tools                []string
	CalledTools          []string
	RemovedTools         []string
	ToolCallCount        uint32
	CalledToolsTruncated bool

	ReasoningEffort   string
	ReasoningDisabled bool

	Guardrails []*llmGuardrailOptions

	RateLimitResult corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_Result
	RateLimitPlugin string
	RateLimitScope  corev1.Service_Spec_Config_LLM_Plugin_TokenRateLimit_Scope

	CacheResult     corev1.AccessLog_Entry_Info_LLM_SemanticCache_Result
	CacheSimilarity float32
	CacheStored     bool
	CachePlugin     string

	RouterResult corev1.AccessLog_Entry_Info_LLM_SemanticRouter_Result
	RouterRoute  string
	RouterModel  string
	RouterPlugin string

	HTTPCode          uint32
	HTTPPath          string
	UserAgent         string
	RequestBodyBytes  uint64
	ResponseBodyBytes uint64

	IsPublic    bool
	IsAnonymous bool
}

func newLLMAccessLog(opts *llmAccessLogOptions) *corev1.AccessLog {
	if opts == nil {
		opts = &llmAccessLogOptions{}
	}
	if opts.CreatedAt.IsZero() {
		opts.CreatedAt = time.Now()
	}
	if opts.Duration == 0 {
		opts.Duration = time.Second
	}
	if opts.HTTPPath == "" {
		opts.HTTPPath = "/v1/chat/completions"
	}
	if opts.Type == corev1.AccessLog_Entry_Info_LLM_TYPE_UNSET {
		opts.Type = corev1.AccessLog_Entry_Info_LLM_COMPLETE
	}

	ret := newAccessLog(&accessLogOptions{
		CreatedAt:    opts.CreatedAt,
		Status:       opts.Status,
		UserRef:      opts.UserRef,
		DeviceRef:    opts.DeviceRef,
		SessionRef:   opts.SessionRef,
		ServiceRef:   opts.ServiceRef,
		NamespaceRef: opts.NamespaceRef,
		RegionRef:    opts.RegionRef,
		PolicyRef:    opts.PolicyRef,
	})

	ret.Entry.Common.StartedAt = pbutils.Timestamp(opts.CreatedAt.Add(-1 * opts.Duration))
	ret.Entry.Common.Mode = corev1.Service_Spec_LLM
	ret.Entry.Common.IsPublic = opts.IsPublic
	ret.Entry.Common.IsAnonymous = opts.IsAnonymous

	llm := &corev1.AccessLog_Entry_Info_LLM{
		Type:              opts.Type,
		Protocol:          opts.Protocol,
		Operation:         opts.Operation,
		Route:             opts.Route,
		Source:            opts.Source,
		IsUpstreamInvoked: opts.IsUpstreamInvoked,
		Stream:            opts.Stream,
		InputItemCount:    opts.InputItemCount,
		HasImageInput:     opts.HasImageInput,
		HasAudioInput:     opts.HasAudioInput,
		Model: &corev1.AccessLog_Entry_Info_LLM_Model{
			Requested: opts.ModelRequested,
			Effective: opts.ModelEffective,
			Reported:  opts.ModelReported,
			Source:    opts.ModelSource,
			Plugin:    opts.ModelPlugin,
		},
		EstimatedInputTokens: opts.EstimatedInputTokens,
		FinishReason:         opts.FinishReason,
		RawFinishReason:      opts.RawFinishReason,
		EventCount:           opts.EventCount,
		Http: &corev1.AccessLog_Entry_Info_HTTP{
			Request: &corev1.AccessLog_Entry_Info_HTTP_Request{
				Path:      opts.HTTPPath,
				Method:    "POST",
				UserAgent: opts.UserAgent,
				BodyBytes: opts.RequestBodyBytes,
			},
			Response: &corev1.AccessLog_Entry_Info_HTTP_Response{
				Code:      opts.HTTPCode,
				BodyBytes: opts.ResponseBodyBytes,
			},
		},
	}

	if opts.HasUsage {
		state := opts.UsageState
		if state == corev1.AccessLog_Entry_Info_LLM_Usage_STATE_UNSET {
			state = corev1.AccessLog_Entry_Info_LLM_Usage_COMPLETE
		}

		llm.Usage = &corev1.AccessLog_Entry_Info_LLM_Usage{
			State:                 state,
			InputTokens:           opts.InputTokens,
			OutputTokens:          opts.OutputTokens,
			TotalTokens:           opts.TotalTokens,
			CacheReadInputTokens:  opts.CacheReadInputTokens,
			CacheWriteInputTokens: opts.CacheWriteInputTokens,
			ReasoningOutputTokens: opts.ReasoningOutputTokens,
		}
	}

	if opts.TimeToFirstTokenMS > 0 {
		llm.TimeToFirstToken = &metav1.Duration{
			Type: &metav1.Duration_Milliseconds{
				Milliseconds: opts.TimeToFirstTokenMS,
			},
		}
	}

	if len(opts.Tools) > 0 || len(opts.CalledTools) > 0 || len(opts.RemovedTools) > 0 {
		llm.Tools = &corev1.AccessLog_Entry_Info_LLM_Tools{
			Count:                  uint32(len(opts.Tools)),
			Names:                  opts.Tools,
			CalledNames:            opts.CalledTools,
			CallCount:              opts.ToolCallCount,
			RemovedNames:           opts.RemovedTools,
			RemovedCount:           uint32(len(opts.RemovedTools)),
			IsCalledNamesTruncated: opts.CalledToolsTruncated,
		}
	}

	if opts.ReasoningEffort != "" || opts.ReasoningDisabled {
		llm.Reasoning = &corev1.AccessLog_Entry_Info_LLM_Reasoning{
			Effort:     opts.ReasoningEffort,
			IsDisabled: opts.ReasoningDisabled,
		}
	}

	for _, guardrail := range opts.Guardrails {
		llm.Guardrails = append(llm.Guardrails, &corev1.AccessLog_Entry_Info_LLM_Guardrail{
			Result: guardrail.Result,
			Leg:    guardrail.Leg,
			Plugin: guardrail.Plugin,
		})
	}

	if opts.RateLimitResult != corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_RESULT_UNSET {
		llm.TokenRateLimit = &corev1.AccessLog_Entry_Info_LLM_TokenRateLimit{
			Result: opts.RateLimitResult,
			Plugin: opts.RateLimitPlugin,
			Scope:  opts.RateLimitScope,
		}
	}

	if opts.CacheResult != corev1.AccessLog_Entry_Info_LLM_SemanticCache_RESULT_UNSET {
		llm.SemanticCache = &corev1.AccessLog_Entry_Info_LLM_SemanticCache{
			Result:     opts.CacheResult,
			Similarity: opts.CacheSimilarity,
			IsStored:   opts.CacheStored,
			Plugin:     opts.CachePlugin,
		}
	}

	if opts.RouterResult != corev1.AccessLog_Entry_Info_LLM_SemanticRouter_RESULT_UNSET {
		llm.SemanticRouter = &corev1.AccessLog_Entry_Info_LLM_SemanticRouter{
			Result: opts.RouterResult,
			Route:  opts.RouterRoute,
			Model:  opts.RouterModel,
			Plugin: opts.RouterPlugin,
		}
	}

	ret.Entry.Info = &corev1.AccessLog_Entry_Info{
		Type: &corev1.AccessLog_Entry_Info_Llm{
			Llm: llm,
		},
	}

	return ret
}

func insertLLMAccessLog(t *testing.T, ts *testServer, opts *llmAccessLogOptions) {
	t.Helper()

	insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newLLMAccessLog(opts)))
}

func insertLLMSummaryAccessLogs(t *testing.T, ts *testServer, base time.Time,
	userRef, serviceRef, sessionRef *metav1.ObjectReference) {
	t.Helper()

	for idx := range 40 {
		opts := &llmAccessLogOptions{
			CreatedAt:            base.Add(time.Duration(idx) * time.Second),
			Duration:             time.Duration(100+idx) * time.Millisecond,
			Status:               corev1.AccessLog_Entry_Common_ALLOWED,
			UserRef:              userRef,
			ServiceRef:           serviceRef,
			SessionRef:           sessionRef,
			Protocol:             corev1.Service_Spec_Config_LLM_OPENAI,
			Operation:            corev1.Service_Spec_Config_LLM_GENERATE,
			Route:                corev1.RequestContext_Request_LLM_CHAT_COMPLETIONS,
			Source:               corev1.AccessLog_Entry_Info_LLM_UPSTREAM,
			IsUpstreamInvoked:    true,
			ModelRequested:       "gpt-5",
			ModelEffective:       "gpt-5",
			ModelReported:        "gpt-5-2026",
			HasUsage:             true,
			InputTokens:          100,
			OutputTokens:         10,
			TotalTokens:          110,
			EstimatedInputTokens: 90,
			FinishReason:         corev1.AccessLog_Entry_Info_LLM_STOP,
			RawFinishReason:      "stop",
			InputItemCount:       3,
			HTTPCode:             200,
			EventCount:           4,
			RequestBodyBytes:     1000,
			ResponseBodyBytes:    2000,
			RateLimitResult:      corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_ALLOWED,
			CacheResult:          corev1.AccessLog_Entry_Info_LLM_SemanticCache_MISS,
			CachePlugin:          "cache",
		}

		switch idx % 10 {
		case 0:
			opts.Source = corev1.AccessLog_Entry_Info_LLM_OCTELIUM
			opts.IsUpstreamInvoked = false
			opts.HasUsage = false
			opts.HTTPCode = 403
			opts.FinishReason = corev1.AccessLog_Entry_Info_LLM_FINISH_REASON_UNSET
			opts.RawFinishReason = ""
			opts.RateLimitResult = corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_RESULT_UNSET
			opts.CacheResult = corev1.AccessLog_Entry_Info_LLM_SemanticCache_RESULT_UNSET
			opts.Guardrails = []*llmGuardrailOptions{
				{
					Result: corev1.AccessLog_Entry_Info_LLM_Guardrail_DENIED,
					Leg:    corev1.Service_Spec_Config_LLM_Plugin_Guardrail_REQUEST,
					Plugin: "pii",
				},
			}
		case 1:
			opts.Source = corev1.AccessLog_Entry_Info_LLM_OCTELIUM
			opts.HTTPCode = 403
			opts.Guardrails = []*llmGuardrailOptions{
				{
					Result: corev1.AccessLog_Entry_Info_LLM_Guardrail_MODIFIED,
					Leg:    corev1.Service_Spec_Config_LLM_Plugin_Guardrail_REQUEST,
					Plugin: "pii",
				},
				{
					Result: corev1.AccessLog_Entry_Info_LLM_Guardrail_DENIED,
					Leg:    corev1.Service_Spec_Config_LLM_Plugin_Guardrail_RESPONSE,
					Plugin: "secrets",
				},
			}
		case 2:
			opts.Source = corev1.AccessLog_Entry_Info_LLM_SEMANTIC_CACHE
			opts.IsUpstreamInvoked = false
			opts.HasUsage = false
			opts.CacheResult = corev1.AccessLog_Entry_Info_LLM_SemanticCache_EXACT_HIT
		case 3:
			opts.Source = corev1.AccessLog_Entry_Info_LLM_OCTELIUM
			opts.IsUpstreamInvoked = false
			opts.HasUsage = false
			opts.HTTPCode = 429
			opts.FinishReason = corev1.AccessLog_Entry_Info_LLM_FINISH_REASON_UNSET
			opts.RawFinishReason = ""
			opts.CacheResult = corev1.AccessLog_Entry_Info_LLM_SemanticCache_RESULT_UNSET
			opts.RateLimitResult = corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_DENIED
			opts.RateLimitPlugin = "quota"
			opts.RateLimitScope = corev1.Service_Spec_Config_LLM_Plugin_TokenRateLimit_INPUT
		case 4:
			opts.Status = corev1.AccessLog_Entry_Common_DENIED
			opts.Source = corev1.AccessLog_Entry_Info_LLM_OCTELIUM
			opts.IsUpstreamInvoked = false
			opts.HasUsage = false
			opts.HTTPCode = 403
			opts.FinishReason = corev1.AccessLog_Entry_Info_LLM_FINISH_REASON_UNSET
			opts.RawFinishReason = ""
			opts.RateLimitResult = corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_RESULT_UNSET
			opts.CacheResult = corev1.AccessLog_Entry_Info_LLM_SemanticCache_RESULT_UNSET
		case 5:
			opts.CacheStored = true
		case 6:
			opts.CacheResult = corev1.AccessLog_Entry_Info_LLM_SemanticCache_ERROR
		case 7:
			opts.ModelSource = corev1.AccessLog_Entry_Info_LLM_Model_SEMANTIC_ROUTER
			opts.ModelPlugin = "router"
			opts.RouterResult = corev1.AccessLog_Entry_Info_LLM_SemanticRouter_MATCH
			opts.RouterRoute = "coding"
			opts.RouterModel = "gpt-5"
			opts.RouterPlugin = "router"
			opts.CacheSimilarity = 0.91
		case 8:
			opts.Guardrails = []*llmGuardrailOptions{
				{
					Result: corev1.AccessLog_Entry_Info_LLM_Guardrail_ERROR,
					Leg:    corev1.Service_Spec_Config_LLM_Plugin_Guardrail_RESPONSE,
					Plugin: "secrets",
				},
			}
		case 9:
			opts.UsageState = corev1.AccessLog_Entry_Info_LLM_Usage_PARTIAL
			opts.FinishReason = corev1.AccessLog_Entry_Info_LLM_LENGTH
			opts.RawFinishReason = "max_tokens"
			opts.Guardrails = []*llmGuardrailOptions{
				{
					Result: corev1.AccessLog_Entry_Info_LLM_Guardrail_PASS,
					Leg:    corev1.Service_Spec_Config_LLM_Plugin_Guardrail_BOTH,
					Plugin: "pii",
				},
			}
		}

		if idx%2 == 0 {
			opts.Protocol = corev1.Service_Spec_Config_LLM_ANTHROPIC
			opts.ModelRequested = "claude-opus-5"
			opts.ModelEffective = "claude-opus-5"
			opts.ModelReported = "claude-opus-5-2026"
		}

		if idx%3 == 0 {
			opts.Tools = []string{"read_file", "write_file"}
			opts.CalledTools = []string{"read_file"}
			opts.ToolCallCount = 2
			opts.RemovedTools = []string{"run_command"}
		}

		if idx%4 == 0 {
			opts.Stream = true
			opts.TimeToFirstTokenMS = uint32(50 + idx)

			insertLLMAccessLog(t, ts, &llmAccessLogOptions{
				CreatedAt:      opts.CreatedAt,
				Status:         opts.Status,
				UserRef:        userRef,
				ServiceRef:     serviceRef,
				SessionRef:     sessionRef,
				Type:           corev1.AccessLog_Entry_Info_LLM_STREAM_START,
				Protocol:       opts.Protocol,
				Operation:      opts.Operation,
				ModelRequested: opts.ModelRequested,
				ModelEffective: opts.ModelEffective,
				Stream:         true,
				HTTPCode:       opts.HTTPCode,
			})

			opts.Type = corev1.AccessLog_Entry_Info_LLM_STREAM_END
		}

		insertLLMAccessLog(t, ts, opts)
	}
}

func TestLLMSummary(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	base := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)

	userRef := randomObjectReference()
	serviceRef := randomObjectReference()
	sessionRef := randomObjectReference()

	insertLLMSummaryAccessLogs(t, ts, base, userRef, serviceRef, sessionRef)

	insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{
		CreatedAt: base,
		UserRef:   userRef,
	})))

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{})
		assert.Nil(t, err, "%+v", err)

		reqs := resp.Stats.Requests

		assert.Equal(t, uint64(40), reqs.Total)
		assert.Equal(t, uint64(36), reqs.Allowed)
		assert.Equal(t, uint64(4), reqs.Denied)
		assert.Equal(t, uint64(24), reqs.Succeeded)
		assert.Equal(t, uint64(16), reqs.Failed)
		assert.Equal(t, uint64(16), reqs.ClientError)
		assert.Equal(t, uint64(0), reqs.ServerError)
		assert.Equal(t, uint64(10), reqs.Streamed)

		assert.Equal(t, uint64(24), reqs.UpstreamInvoked)
		assert.Equal(t, uint64(4), reqs.DiscardedInference)

		assert.Equal(t, uint64(20), reqs.SourceUpstream)
		assert.Equal(t, uint64(4), reqs.SourceSemanticCache)
		assert.Equal(t, uint64(16), reqs.SourceOctelium)

		assert.Equal(t, uint64(24), reqs.WithUsage)
		assert.Equal(t, uint64(16), reqs.WithoutUsage)
		assert.Equal(t, uint64(20), reqs.UsageComplete)
		assert.Equal(t, uint64(4), reqs.UsagePartial)

		assert.Equal(t, uint64(16), reqs.GuardrailInspected)
		assert.Equal(t, uint64(4), reqs.GuardrailPassed)
		assert.Equal(t, uint64(4), reqs.GuardrailModified)
		assert.Equal(t, uint64(8), reqs.GuardrailDenied)
		assert.Equal(t, uint64(4), reqs.GuardrailError)

		assert.Equal(t, uint64(28), reqs.TokenRateLimitAllowed)
		assert.Equal(t, uint64(4), reqs.TokenRateLimitDenied)

		assert.Equal(t, uint64(4), reqs.CacheExactHit)
		assert.Equal(t, uint64(0), reqs.CacheSemanticHit)
		assert.Equal(t, uint64(20), reqs.CacheMiss)
		assert.Equal(t, uint64(0), reqs.CacheBypass)
		assert.Equal(t, uint64(4), reqs.CacheError)
		assert.Equal(t, uint64(4), reqs.CacheStored)

		assert.Equal(t, uint64(4), reqs.RouterMatch)
		assert.Equal(t, uint64(0), reqs.RouterNoMatch)

		assert.Equal(t, uint64(4), reqs.ModelOverridden)
		assert.Equal(t, uint64(4), reqs.ModelRouted)

		assert.Equal(t, uint64(14), reqs.WithTools)
		assert.Equal(t, uint64(14), reqs.WithToolCalls)
		assert.Equal(t, uint64(14), reqs.WithToolsRemoved)
		assert.Equal(t, uint64(0), reqs.WithCalledToolsTruncated)

		assert.Equal(t, uint64(24), reqs.FinishStop)
		assert.Equal(t, uint64(4), reqs.FinishLength)
		assert.Equal(t, uint64(0), reqs.FinishToolCall)

		assert.Equal(t, uint64(24*100), resp.Stats.Tokens.Input)
		assert.Equal(t, uint64(24*10), resp.Stats.Tokens.Output)
		assert.Equal(t, uint64(24*110), resp.Stats.Tokens.Total)
		assert.Equal(t, uint64(4*110), resp.Stats.Tokens.Discarded)
		assert.Equal(t, uint64(40*90), resp.Stats.Tokens.EstimatedInput)

		assert.Equal(t, uint64(40), resp.Stats.Latency.Count)
		assert.True(t, resp.Stats.Latency.AvgMs > 100)
		assert.True(t, resp.Stats.Latency.MaxMs >= 139)
		assert.Equal(t, uint64(10), resp.Stats.TimeToFirstToken.Count)

		assert.Equal(t, uint64(40*4), resp.Stats.StreamEvents)
		assert.Equal(t, uint64(28), resp.Stats.ToolsOffered)
		assert.Equal(t, uint64(14), resp.Stats.ToolsRemoved)
		assert.Equal(t, uint64(28), resp.Stats.ToolCalls)
		assert.Equal(t, uint64(14), resp.Stats.DistinctToolsCalled)
		assert.Equal(t, uint64(40*3), resp.Stats.InputItems)
		assert.Equal(t, uint64(40*1000), resp.Stats.RequestBodyBytes)

		assert.Equal(t, 0, len(resp.Cardinalities))
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Cardinalities: []vllmv1.Dimension{
				vllmv1.Dimension_MODEL,
				vllmv1.Dimension_PROTOCOL,
				vllmv1.Dimension_USER,
				vllmv1.Dimension_TOOL,
				vllmv1.Dimension_CALLED_TOOL,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 5, len(resp.Cardinalities))
		assert.Equal(t, vllmv1.Dimension_MODEL, resp.Cardinalities[0].Dimension)
		assert.Equal(t, uint64(2), resp.Cardinalities[0].Count)
		assert.Equal(t, uint64(2), resp.Cardinalities[1].Count)
		assert.Equal(t, uint64(1), resp.Cardinalities[2].Count)
		assert.Equal(t, uint64(2), resp.Cardinalities[3].Count)
		assert.Equal(t, uint64(1), resp.Cardinalities[4].Count)
	}

	{
		_, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Cardinalities: []vllmv1.Dimension{
				vllmv1.Dimension_MODEL,
				vllmv1.Dimension_MODEL,
			},
		})
		assert.NotNil(t, err)
	}

	{
		_, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				EntryScope: vllmv1.EntryScope_ALL,
			},
		})
		assert.NotNil(t, err)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				EntryScope: vllmv1.EntryScope_STREAM_START,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(10), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				Models: []string{"gpt-5"},
			},
			IncludeQuantiles: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(20), resp.Stats.Requests.Total)
		assert.True(t, resp.Stats.Latency.P95Ms > 0)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				Models:     []string{"gpt-5-2026"},
				ModelField: vllmv1.ModelField_REPORTED,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(20), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				GuardrailResults: []corev1.AccessLog_Entry_Info_LLM_Guardrail_Result{
					corev1.AccessLog_Entry_Info_LLM_Guardrail_DENIED,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(8), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				GuardrailLegs: []corev1.Service_Spec_Config_LLM_Plugin_Guardrail_Leg{
					corev1.Service_Spec_Config_LLM_Plugin_Guardrail_RESPONSE,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(8), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				GuardrailPlugins: []string{"secrets"},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(8), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				SemanticCacheResults: []corev1.AccessLog_Entry_Info_LLM_SemanticCache_Result{
					corev1.AccessLog_Entry_Info_LLM_SemanticCache_ERROR,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(4), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				TokenRateLimitResults: []corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_Result{
					corev1.AccessLog_Entry_Info_LLM_TokenRateLimit_DENIED,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(4), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				SemanticRouterRoutes: []string{"coding"},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(4), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				FinishReasons: []corev1.AccessLog_Entry_Info_LLM_FinishReason{
					corev1.AccessLog_Entry_Info_LLM_LENGTH,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(4), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				RawFinishReasons: []string{"max_tokens"},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(4), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				CalledTools: []string{"read_file"},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(14), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				RemovedTools: []string{"run_command"},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(14), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				Stream: utils_types.BoolToPtr(true),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(10), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				IsUpstreamInvoked: utils_types.BoolToPtr(true),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(24), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				HasUsage: utils_types.BoolToPtr(false),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(16), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				MinTotalTokens: proto.Uint64(1),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(24), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				MaxTotalTokens: proto.Uint64(0),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(16), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Breakdowns: []*vllmv1.BreakdownRequest{
				{
					Dimension: vllmv1.Dimension_MODEL,
					OrderBy:   vllmv1.Metric_TOTAL_TOKENS,
				},
				{
					Dimension: vllmv1.Dimension_MODEL_SOURCE,
				},
				{
					Dimension: vllmv1.Dimension_FINISH_REASON,
					Limit:     1,
				},
				{
					Dimension: vllmv1.Dimension_GUARDRAIL_RESULT,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 4, len(resp.Breakdowns))

		assert.Equal(t, vllmv1.Dimension_MODEL, resp.Breakdowns[0].Dimension)
		assert.Equal(t, uint64(2), resp.Breakdowns[0].TotalCount)
		assert.Equal(t, 2, len(resp.Breakdowns[0].Items))
		assert.Nil(t, resp.Breakdowns[0].Other)

		assert.Equal(t, uint64(2), resp.Breakdowns[1].TotalCount)
		assert.Equal(t, corev1.AccessLog_Entry_Info_LLM_Model_SOURCE_UNSET.String(),
			resp.Breakdowns[1].Items[0].Key)
		assert.Equal(t, uint64(36), resp.Breakdowns[1].Items[0].Stats.Requests.Total)
		assert.Equal(t, corev1.AccessLog_Entry_Info_LLM_Model_SEMANTIC_ROUTER.String(),
			resp.Breakdowns[1].Items[1].Key)
		assert.Equal(t, uint64(4), resp.Breakdowns[1].Items[1].Stats.Requests.Total)

		assert.Equal(t, uint64(3), resp.Breakdowns[2].TotalCount)
		assert.Equal(t, 1, len(resp.Breakdowns[2].Items))
		assert.Equal(t, corev1.AccessLog_Entry_Info_LLM_STOP.String(), resp.Breakdowns[2].Items[0].Key)
		assert.NotNil(t, resp.Breakdowns[2].Other)
		assert.Equal(t, uint64(16), resp.Breakdowns[2].Other.Requests.Total)

		assert.Equal(t, uint64(4), resp.Breakdowns[3].TotalCount)
		assert.Nil(t, resp.Breakdowns[3].Other)
		assert.Equal(t, corev1.AccessLog_Entry_Info_LLM_Guardrail_DENIED.String(),
			resp.Breakdowns[3].Items[0].Key)
		assert.Equal(t, uint64(8), resp.Breakdowns[3].Items[0].Stats.Requests.Total)
	}

	{
		_, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Breakdowns: []*vllmv1.BreakdownRequest{
				{
					Dimension: vllmv1.Dimension_MODEL,
				},
				{
					Dimension: vllmv1.Dimension_MODEL,
				},
			},
		})
		assert.NotNil(t, err)
	}
}

func TestLLMTop(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	user1 := createTestUser(t, ts)
	user2 := createTestUser(t, ts)
	service1 := createTestService(t, ts)
	service2 := createTestService(t, ts)

	userRef1 := umetav1.GetObjectReference(user1)
	userRef2 := umetav1.GetObjectReference(user2)
	serviceRef1 := umetav1.GetObjectReference(service1)
	serviceRef2 := umetav1.GetObjectReference(service2)

	base := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)

	for idx := range 30 {
		opts := &llmAccessLogOptions{
			CreatedAt:         base.Add(time.Duration(idx) * time.Second),
			Status:            corev1.AccessLog_Entry_Common_ALLOWED,
			UserRef:           userRef1,
			ServiceRef:        serviceRef1,
			Protocol:          corev1.Service_Spec_Config_LLM_OPENAI,
			Operation:         corev1.Service_Spec_Config_LLM_GENERATE,
			Source:            corev1.AccessLog_Entry_Info_LLM_UPSTREAM,
			IsUpstreamInvoked: true,
			ModelRequested:    "gpt-5",
			ModelEffective:    "gpt-5",
			ModelReported:     "gpt-5",
			HasUsage:          true,
			InputTokens:       10,
			OutputTokens:      1,
			TotalTokens:       11,
			HTTPCode:          200,
			Tools:             []string{"read_file"},
			CalledTools:       []string{"read_file"},
			ToolCallCount:     1,
		}

		switch {
		case idx < 5:
			opts.ModelRequested = "claude-opus-5"
			opts.ModelEffective = "claude-sonnet-5"
			opts.ModelReported = "claude-sonnet-5"
			opts.ModelSource = corev1.AccessLog_Entry_Info_LLM_Model_PLUGIN
			opts.ModelPlugin = "downgrade"
			opts.Protocol = corev1.Service_Spec_Config_LLM_ANTHROPIC
			opts.UserRef = userRef2
			opts.ServiceRef = serviceRef2
			opts.InputTokens = 1000
			opts.OutputTokens = 100
			opts.TotalTokens = 1100
			opts.Tools = []string{"read_file", "run_command"}
			opts.CalledTools = []string{"run_command"}
			opts.ToolCallCount = 3
			opts.RemovedTools = []string{"delete_file"}
		case idx < 12:
			opts.ModelRequested = "gpt-5-mini"
			opts.ModelEffective = "gpt-5-mini"
			opts.ModelReported = "gpt-5-mini"
			opts.Tools = nil
			opts.CalledTools = nil
			opts.ToolCallCount = 0
		}

		insertLLMAccessLog(t, ts, opts)
	}

	{
		resp, err := ts.srv.listLLMTopModel(ts.ctx, &vllmv1.ListTopModelRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 3, len(resp.Items))
		assert.Equal(t, uint64(3), resp.TotalCount)
		assert.Nil(t, resp.Other)
		assert.Equal(t, "gpt-5", resp.Items[0].Model)
		assert.Equal(t, uint64(18), resp.Items[0].Stats.Requests.Total)
		assert.Equal(t, uint64(7), resp.Items[1].Stats.Requests.Total)
		assert.Equal(t, uint64(5), resp.Items[2].Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.listLLMTopModel(ts.ctx, &vllmv1.ListTopModelRequest{
			OrderBy: vllmv1.Metric_TOTAL_TOKENS,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, "claude-sonnet-5", resp.Items[0].Model)
		assert.Equal(t, uint64(5*1100), resp.Items[0].Stats.Tokens.Total)
		assert.Equal(t, uint64(0), resp.Items[0].RequestedCount)
		assert.Equal(t, uint64(5), resp.Items[0].EffectiveCount)
		assert.Equal(t, uint64(5), resp.Items[0].ReportedCount)
	}

	{
		resp, err := ts.srv.listLLMTopModel(ts.ctx, &vllmv1.ListTopModelRequest{
			Field: vllmv1.ModelField_REQUESTED,
			Limit: 1,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, uint64(3), resp.TotalCount)
		assert.Equal(t, "gpt-5", resp.Items[0].Model)
		assert.NotNil(t, resp.Other)
		assert.Equal(t, uint64(12), resp.Other.Requests.Total)
	}

	{
		resp, err := ts.srv.listLLMTopTool(ts.ctx, &vllmv1.ListTopToolRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Items))
		assert.Nil(t, resp.Other)
		assert.Equal(t, "read_file", resp.Items[0].Tool)
		assert.Equal(t, uint64(23), resp.Items[0].OfferedCount)
		assert.Equal(t, uint64(18), resp.Items[0].CalledCount)
		assert.Equal(t, uint64(0), resp.Items[0].RemovedCount)
		assert.Equal(t, "run_command", resp.Items[1].Tool)
		assert.Equal(t, uint64(5), resp.Items[1].OfferedCount)
		assert.Equal(t, uint64(5), resp.Items[1].CalledCount)
	}

	{
		resp, err := ts.srv.listLLMTopTool(ts.ctx, &vllmv1.ListTopToolRequest{
			Scope: vllmv1.ToolScope_CALLED,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Items))
		assert.Equal(t, uint64(18), resp.Items[0].Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.listLLMTopTool(ts.ctx, &vllmv1.ListTopToolRequest{
			Scope: vllmv1.ToolScope_REMOVED,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, "delete_file", resp.Items[0].Tool)
		assert.Equal(t, uint64(5), resp.Items[0].RemovedCount)
		assert.Equal(t, uint64(0), resp.Items[0].OfferedCount)
	}

	{
		resp, err := ts.srv.listLLMTopUser(ts.ctx, &vllmv1.ListTopUserRequest{
			OrderBy: vllmv1.Metric_TOTAL_TOKENS,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Items))
		assert.Equal(t, user2.Metadata.Uid, resp.Items[0].User.Metadata.Uid)
		assert.Equal(t, uint64(5*1100), resp.Items[0].Stats.Tokens.Total)
		assert.Equal(t, user1.Metadata.Uid, resp.Items[1].User.Metadata.Uid)
		assert.Equal(t, uint64(25), resp.Items[1].Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.listLLMTopService(ts.ctx, &vllmv1.ListTopServiceRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Items))
		assert.Equal(t, service1.Metadata.Uid, resp.Items[0].Service.Metadata.Uid)
		assert.Equal(t, service2.Metadata.Uid, resp.Items[1].Service.Metadata.Uid)
	}

	{
		resp, err := ts.srv.listLLMTopDimension(ts.ctx, &vllmv1.ListTopDimensionRequest{
			Dimension: vllmv1.Dimension_PROTOCOL,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Items))
		assert.Equal(t, corev1.Service_Spec_Config_LLM_OPENAI.String(), resp.Items[0].Key)
		assert.Equal(t, uint64(25), resp.Items[0].Stats.Requests.Total)
		assert.Nil(t, resp.Items[0].Ref)
	}

	{
		resp, err := ts.srv.listLLMTopDimension(ts.ctx, &vllmv1.ListTopDimensionRequest{
			Dimension: vllmv1.Dimension_MODEL_PLUGIN,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, "downgrade", resp.Items[0].Key)
		assert.Equal(t, uint64(5), resp.Items[0].Stats.Requests.ModelOverridden)
	}

	{
		resp, err := ts.srv.listLLMTopDimension(ts.ctx, &vllmv1.ListTopDimensionRequest{
			Dimension: vllmv1.Dimension_USER,
			Limit:     1,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Items))
		assert.Equal(t, uint64(2), resp.TotalCount)
		assert.Equal(t, user1.Metadata.Uid, resp.Items[0].Ref.Uid)
		assert.Equal(t, user1.Metadata.Name, resp.Items[0].Ref.Name)
		assert.NotNil(t, resp.Other)
		assert.Equal(t, uint64(5), resp.Other.Requests.Total)
	}

	{
		_, err := ts.srv.listLLMTopDimension(ts.ctx, &vllmv1.ListTopDimensionRequest{
			Dimension: vllmv1.Dimension_MODEL,
			Limit:     llmMaxTopLimit + 1,
		})
		assert.NotNil(t, err)
	}

	{
		_, err := ts.srv.listLLMTopDimension(ts.ctx, &vllmv1.ListTopDimensionRequest{
			Dimension: vllmv1.Dimension_DIMENSION_UNSET,
		})
		assert.NotNil(t, err)
	}
}

func TestLLMDataPoint(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	base := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Minute)

	for idx := range 30 {
		opts := &llmAccessLogOptions{
			CreatedAt:         base.Add(time.Duration(idx) * time.Minute),
			Status:            corev1.AccessLog_Entry_Common_ALLOWED,
			Protocol:          corev1.Service_Spec_Config_LLM_OPENAI,
			Operation:         corev1.Service_Spec_Config_LLM_GENERATE,
			Source:            corev1.AccessLog_Entry_Info_LLM_UPSTREAM,
			IsUpstreamInvoked: true,
			ModelRequested:    "gpt-5",
			ModelEffective:    "gpt-5",
			HasUsage:          true,
			InputTokens:       10,
			OutputTokens:      2,
			TotalTokens:       12,
			HTTPCode:          200,
		}

		if idx%3 == 0 {
			opts.ModelRequested = "claude-opus-5"
			opts.ModelEffective = "claude-opus-5"
			opts.InputTokens = 100
			opts.TotalTokens = 102
		}
		if idx%10 == 0 {
			opts.ModelRequested = "gpt-5-mini"
			opts.ModelEffective = "gpt-5-mini"
		}

		insertLLMAccessLog(t, ts, opts)
	}

	from := base.Add(-1 * time.Minute)
	to := base.Add(31 * time.Minute)

	{
		resp, err := ts.srv.getLLMDataPoint(ts.ctx, &vllmv1.GetDataPointRequest{
			Filter: &vllmv1.Filter{
				From: pbutils.Timestamp(from),
				To:   pbutils.Timestamp(to),
			},
			Interval: &metav1.Duration{
				Type: &metav1.Duration_Minutes{Minutes: 1},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Series))
		assert.Equal(t, 32, len(resp.Series[0].Datapoints))
		assert.Equal(t, uint64(30), resp.Series[0].Stats.Requests.Total)

		var total uint64
		for _, dp := range resp.Series[0].Datapoints {
			total = total + dp.Stats.Requests.Total
		}
		assert.Equal(t, uint64(30), total)
		assert.Equal(t, uint64(0), resp.Series[0].Datapoints[0].Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMDataPoint(ts.ctx, &vllmv1.GetDataPointRequest{
			Filter: &vllmv1.Filter{
				From: pbutils.Timestamp(from),
				To:   pbutils.Timestamp(to),
			},
			Interval: &metav1.Duration{
				Type: &metav1.Duration_Minutes{Minutes: 5},
			},
			GroupBy: vllmv1.Dimension_MODEL,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 3, len(resp.Series))
		assert.Equal(t, uint64(3), resp.TotalCount)
		assert.Nil(t, resp.Other)
		assert.Equal(t, "gpt-5", resp.Series[0].Key)

		var total uint64
		for _, series := range resp.Series {
			total = total + series.Stats.Requests.Total
			for _, dp := range series.Datapoints {
				assert.NotNil(t, dp.Stats.Tokens)
			}
		}
		assert.Equal(t, uint64(30), total)
	}

	{
		resp, err := ts.srv.getLLMDataPoint(ts.ctx, &vllmv1.GetDataPointRequest{
			Filter: &vllmv1.Filter{
				From: pbutils.Timestamp(from),
				To:   pbutils.Timestamp(to),
			},
			Interval: &metav1.Duration{
				Type: &metav1.Duration_Minutes{Minutes: 5},
			},
			GroupBy: vllmv1.Dimension_MODEL,
			OrderBy: vllmv1.Metric_TOTAL_TOKENS,
			Limit:   1,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(resp.Series))
		assert.Equal(t, "claude-opus-5", resp.Series[0].Key)
		assert.NotNil(t, resp.Other)
		assert.Equal(t, uint64(21), resp.Other.Stats.Requests.Total)
		assert.Equal(t, len(resp.Series[0].Datapoints), len(resp.Other.Datapoints))

		var total uint64
		for _, dp := range resp.Other.Datapoints {
			total = total + dp.Stats.Requests.Total
		}
		assert.Equal(t, uint64(21), total)
	}

	{
		_, err := ts.srv.getLLMDataPoint(ts.ctx, &vllmv1.GetDataPointRequest{
			GroupBy: vllmv1.Dimension_MODEL,
			Limit:   llmMaxSeriesLimit + 1,
		})
		assert.NotNil(t, err)
	}

	{
		_, err := ts.srv.getLLMDataPoint(ts.ctx, &vllmv1.GetDataPointRequest{
			Filter: &vllmv1.Filter{
				From: pbutils.Timestamp(base.Add(-1 * 8760 * time.Hour)),
				To:   pbutils.Timestamp(base),
			},
			Interval: &metav1.Duration{
				Type: &metav1.Duration_Minutes{Minutes: 1},
			},
		})
		assert.NotNil(t, err)
	}
}

func TestLLMListAccessLog(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	base := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)

	for idx := range 20 {
		opts := &llmAccessLogOptions{
			CreatedAt:         base.Add(time.Duration(idx) * time.Second),
			Duration:          time.Duration(idx+1) * time.Second,
			Status:            corev1.AccessLog_Entry_Common_ALLOWED,
			Protocol:          corev1.Service_Spec_Config_LLM_OPENAI,
			Operation:         corev1.Service_Spec_Config_LLM_GENERATE,
			Route:             corev1.RequestContext_Request_LLM_CHAT_COMPLETIONS,
			Source:            corev1.AccessLog_Entry_Info_LLM_UPSTREAM,
			IsUpstreamInvoked: true,
			ModelRequested:    "gpt-5",
			ModelEffective:    "gpt-5",
			HasUsage:          true,
			InputTokens:       uint64(10 * idx),
			OutputTokens:      1,
			TotalTokens:       uint64(10*idx) + 1,
			HTTPCode:          200,
			UserAgent:         "octelium-test",
		}

		if idx%2 == 0 {
			opts.Operation = corev1.Service_Spec_Config_LLM_EMBED
			opts.Route = corev1.RequestContext_Request_LLM_EMBEDDINGS
			opts.HTTPPath = "/v1/embeddings"
		}

		insertLLMAccessLog(t, ts, opts)
	}

	insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{
		CreatedAt: base,
	})))

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(20), resp.ListResponseMeta.TotalCount)
		assert.Equal(t, 20, len(resp.Items))
		assert.NotNil(t, resp.Items[0].Entry.Info.GetLlm())
		assert.True(t, resp.Items[0].Metadata.CreatedAt.AsTime().
			After(resp.Items[len(resp.Items)-1].Metadata.CreatedAt.AsTime()))
	}

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			ItemsPerPage: 5,
			Page:         1,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 5, len(resp.Items))
		assert.True(t, resp.ListResponseMeta.HasMore)
	}

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			OrderBy: &vllmv1.ListAccessLogRequest_OrderBy{
				Type: vllmv1.ListAccessLogRequest_OrderBy_TOTAL_TOKENS,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(191), resp.Items[0].Entry.Info.GetLlm().Usage.TotalTokens)
	}

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			OrderBy: &vllmv1.ListAccessLogRequest_OrderBy{
				Type: vllmv1.ListAccessLogRequest_OrderBy_LATENCY,
				Mode: vllmv1.ListAccessLogRequest_OrderBy_ASC,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(1), resp.Items[0].Entry.Info.GetLlm().Usage.TotalTokens)
	}

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			Filter: &vllmv1.Filter{
				EntryScope: vllmv1.EntryScope_ALL,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 20, len(resp.Items))
	}

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			Filter: &vllmv1.Filter{
				Operations: []corev1.Service_Spec_Config_LLM_Operation{
					corev1.Service_Spec_Config_LLM_EMBED,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 10, len(resp.Items))
	}

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			Filter: &vllmv1.Filter{
				Routes: []corev1.RequestContext_Request_LLM_Route{
					corev1.RequestContext_Request_LLM_EMBEDDINGS,
				},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 10, len(resp.Items))
	}

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			Filter: &vllmv1.Filter{
				HttpPaths: []string{"/v1/embeddings"},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 10, len(resp.Items))
	}

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			Filter: &vllmv1.Filter{
				MinLatencyMs: proto.Uint64(15000),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 6, len(resp.Items))
	}

	{
		_, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			Filter: &vllmv1.Filter{
				Models: []string{""},
			},
		})
		assert.NotNil(t, err)
	}
}
