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
)

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
	Source    corev1.AccessLog_Entry_Info_LLM_Source

	ModelRequested string
	ModelEffective string
	ModelReported  string
	ModelSource    corev1.AccessLog_Entry_Info_LLM_Model_Source
	ModelPlugin    string

	Stream bool

	UsageSource              corev1.AccessLog_Entry_Info_LLM_Usage_Source
	InputTokens              uint64
	OutputTokens             uint64
	TotalTokens              uint64
	CacheReadInputTokens     uint64
	CacheCreationInputTokens uint64
	ReasoningTokens          uint64
	EstimatedInputTokens     uint64

	FinishReason       string
	TimeToFirstTokenMS uint32
	EventCount         uint64

	Tools        []string
	CalledTools  []string
	ToolsRemoved uint32

	ReasoningEffort   string
	ReasoningDisabled bool

	GuardrailResult corev1.AccessLog_Entry_Info_LLM_Guardrail_Result
	GuardrailLeg    corev1.Service_Spec_Config_LLM_Plugin_Guardrail_Leg
	GuardrailPlugin string

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
		Type:      opts.Type,
		Protocol:  opts.Protocol,
		Operation: opts.Operation,
		Source:    opts.Source,
		Stream:    opts.Stream,
		Model: &corev1.AccessLog_Entry_Info_LLM_Model{
			Requested: opts.ModelRequested,
			Effective: opts.ModelEffective,
			Reported:  opts.ModelReported,
			Source:    opts.ModelSource,
			Plugin:    opts.ModelPlugin,
		},
		Usage: &corev1.AccessLog_Entry_Info_LLM_Usage{
			Source:                   opts.UsageSource,
			InputTokens:              opts.InputTokens,
			OutputTokens:             opts.OutputTokens,
			TotalTokens:              opts.TotalTokens,
			CacheReadInputTokens:     opts.CacheReadInputTokens,
			CacheCreationInputTokens: opts.CacheCreationInputTokens,
			ReasoningTokens:          opts.ReasoningTokens,
		},
		EstimatedInputTokens: opts.EstimatedInputTokens,
		FinishReason:         opts.FinishReason,
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

	if opts.TimeToFirstTokenMS > 0 {
		llm.TimeToFirstToken = &metav1.Duration{
			Type: &metav1.Duration_Milliseconds{
				Milliseconds: opts.TimeToFirstTokenMS,
			},
		}
	}

	if len(opts.Tools) > 0 || len(opts.CalledTools) > 0 || opts.ToolsRemoved > 0 {
		llm.Tools = &corev1.AccessLog_Entry_Info_LLM_Tools{
			Count:        uint32(len(opts.Tools)),
			Names:        opts.Tools,
			CalledNames:  opts.CalledTools,
			RemovedCount: opts.ToolsRemoved,
		}
	}

	if opts.ReasoningEffort != "" || opts.ReasoningDisabled {
		llm.Reasoning = &corev1.AccessLog_Entry_Info_LLM_Reasoning{
			Effort:     opts.ReasoningEffort,
			IsDisabled: opts.ReasoningDisabled,
		}
	}

	if opts.GuardrailResult != corev1.AccessLog_Entry_Info_LLM_Guardrail_RESULT_UNSET {
		llm.Guardrail = &corev1.AccessLog_Entry_Info_LLM_Guardrail{
			Result: opts.GuardrailResult,
			Leg:    opts.GuardrailLeg,
			Plugin: opts.GuardrailPlugin,
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

func TestLLMSummary(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	base := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)

	userRef := randomObjectReference()
	serviceRef := randomObjectReference()
	sessionRef := randomObjectReference()

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
			Source:               corev1.AccessLog_Entry_Info_LLM_UPSTREAM,
			UsageSource:          corev1.AccessLog_Entry_Info_LLM_Usage_PROVIDER,
			ModelRequested:       "gpt-5",
			ModelEffective:       "gpt-5",
			ModelReported:        "gpt-5-2026",
			InputTokens:          100,
			OutputTokens:         10,
			TotalTokens:          110,
			EstimatedInputTokens: 90,
			HTTPCode:             200,
			EventCount:           4,
			RequestBodyBytes:     1000,
			ResponseBodyBytes:    2000,
		}

		if idx%2 == 0 {
			opts.ModelRequested = "claude-opus-5"
			opts.ModelEffective = "claude-opus-5"
			opts.ModelReported = "claude-opus-5-2026"
			opts.Protocol = corev1.Service_Spec_Config_LLM_ANTHROPIC
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
				HTTPCode:       200,
			})
			opts.Type = corev1.AccessLog_Entry_Info_LLM_STREAM_END
		}

		if idx%5 == 0 {
			opts.Status = corev1.AccessLog_Entry_Common_DENIED
			opts.HTTPCode = 403
			opts.Source = corev1.AccessLog_Entry_Info_LLM_OCTELIUM
			opts.GuardrailResult = corev1.AccessLog_Entry_Info_LLM_Guardrail_DENIED
			opts.GuardrailLeg = corev1.Service_Spec_Config_LLM_Plugin_Guardrail_REQUEST
			opts.GuardrailPlugin = "pii"
		}

		if idx%8 == 0 {
			opts.Source = corev1.AccessLog_Entry_Info_LLM_SEMANTIC_CACHE
			opts.UsageSource = corev1.AccessLog_Entry_Info_LLM_Usage_CACHED
			opts.InputTokens = 0
			opts.OutputTokens = 0
			opts.TotalTokens = 0
		}

		if idx%3 == 0 {
			opts.Tools = []string{"read_file", "write_file"}
			opts.CalledTools = []string{"read_file"}
		}

		insertLLMAccessLog(t, ts, opts)
	}

	insertLogJSON(t, ts.srv, "access_logs", marshalLog(t, newAccessLog(&accessLogOptions{
		CreatedAt: base,
		UserRef:   userRef,
	})))

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{})
		assert.Nil(t, err, "%+v", err)

		assert.Equal(t, uint64(40), resp.Stats.Requests.Total)
		assert.Equal(t, uint64(32), resp.Stats.Requests.Allowed)
		assert.Equal(t, uint64(8), resp.Stats.Requests.Denied)
		assert.Equal(t, uint64(10), resp.Stats.Requests.Streamed)
		assert.Equal(t, uint64(8), resp.Stats.Requests.Failed)
		assert.Equal(t, uint64(8), resp.Stats.Requests.ClientError)
		assert.Equal(t, uint64(0), resp.Stats.Requests.ServerError)
		assert.Equal(t, uint64(8), resp.Stats.Requests.GuardrailDenied)
		assert.Equal(t, uint64(5), resp.Stats.Requests.SourceSemanticCache)
		assert.Equal(t, uint64(5), resp.Stats.Requests.UsageCached)
		assert.Equal(t, uint64(14), resp.Stats.Requests.WithTools)
		assert.Equal(t, uint64(14), resp.Stats.Requests.WithToolCalls)

		assert.Equal(t, uint64(35*100), resp.Stats.Tokens.Input)
		assert.Equal(t, uint64(35*10), resp.Stats.Tokens.Output)
		assert.Equal(t, uint64(35*110), resp.Stats.Tokens.Total)
		assert.Equal(t, uint64(40*90), resp.Stats.Tokens.EstimatedInput)

		assert.Equal(t, uint64(40), resp.Stats.Latency.Count)
		assert.True(t, resp.Stats.Latency.AvgMs > 100)
		assert.True(t, resp.Stats.Latency.MaxMs >= 139)
		assert.Equal(t, uint64(10), resp.Stats.TimeToFirstToken.Count)

		assert.Equal(t, uint64(40*4), resp.Stats.StreamEvents)
		assert.Equal(t, uint64(28), resp.Stats.ToolsDeclared)
		assert.Equal(t, uint64(14), resp.Stats.ToolCalls)
		assert.Equal(t, uint64(40*1000), resp.Stats.RequestBodyBytes)

		assert.Equal(t, uint64(2), resp.Cardinality.Models)
		assert.Equal(t, uint64(1), resp.Cardinality.Users)
		assert.Equal(t, uint64(1), resp.Cardinality.Services)
		assert.Equal(t, uint64(2), resp.Cardinality.Protocols)
		assert.Equal(t, uint64(2), resp.Cardinality.Tools)
		assert.Equal(t, uint64(1), resp.Cardinality.CalledTools)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Filter: &vllmv1.Filter{
				EntryScope: vllmv1.EntryScope_ALL,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(50), resp.Stats.Requests.Total)
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
				CalledTools: []string{"read_file"},
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
				MinTotalTokens: 1,
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(35), resp.Stats.Requests.Total)
	}

	{
		resp, err := ts.srv.getLLMSummary(ts.ctx, &vllmv1.GetSummaryRequest{
			Breakdowns: []vllmv1.Dimension{
				vllmv1.Dimension_MODEL,
				vllmv1.Dimension_PROTOCOL,
			},
			BreakdownOrderBy: vllmv1.Metric_TOTAL_TOKENS,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 2, len(resp.Breakdowns))
		assert.Equal(t, vllmv1.Dimension_MODEL, resp.Breakdowns[0].Dimension)
		assert.Equal(t, uint64(2), resp.Breakdowns[0].TotalCount)
		assert.Equal(t, 2, len(resp.Breakdowns[0].Items))
		assert.Nil(t, resp.Breakdowns[0].Other)
		assert.Equal(t, "gpt-5", resp.Breakdowns[0].Items[0].Key)
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
			CreatedAt:      base.Add(time.Duration(idx) * time.Second),
			Status:         corev1.AccessLog_Entry_Common_ALLOWED,
			UserRef:        userRef1,
			ServiceRef:     serviceRef1,
			Protocol:       corev1.Service_Spec_Config_LLM_OPENAI,
			Operation:      corev1.Service_Spec_Config_LLM_GENERATE,
			Source:         corev1.AccessLog_Entry_Info_LLM_UPSTREAM,
			UsageSource:    corev1.AccessLog_Entry_Info_LLM_Usage_PROVIDER,
			ModelRequested: "gpt-5",
			ModelEffective: "gpt-5",
			ModelReported:  "gpt-5",
			InputTokens:    10,
			OutputTokens:   1,
			TotalTokens:    11,
			HTTPCode:       200,
			Tools:          []string{"read_file"},
			CalledTools:    []string{"read_file"},
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
		case idx < 12:
			opts.ModelRequested = "gpt-5-mini"
			opts.ModelEffective = "gpt-5-mini"
			opts.ModelReported = "gpt-5-mini"
			opts.Tools = nil
			opts.CalledTools = nil
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
		assert.Equal(t, "read_file", resp.Items[0].Tool)
		assert.Equal(t, uint64(23), resp.Items[0].DeclaredCount)
		assert.Equal(t, uint64(18), resp.Items[0].CalledCount)
		assert.Equal(t, "run_command", resp.Items[1].Tool)
		assert.Equal(t, uint64(5), resp.Items[1].DeclaredCount)
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
			CreatedAt:      base.Add(time.Duration(idx) * time.Minute),
			Status:         corev1.AccessLog_Entry_Common_ALLOWED,
			Protocol:       corev1.Service_Spec_Config_LLM_OPENAI,
			Operation:      corev1.Service_Spec_Config_LLM_GENERATE,
			Source:         corev1.AccessLog_Entry_Info_LLM_UPSTREAM,
			UsageSource:    corev1.AccessLog_Entry_Info_LLM_Usage_PROVIDER,
			ModelRequested: "gpt-5",
			ModelEffective: "gpt-5",
			InputTokens:    10,
			OutputTokens:   2,
			TotalTokens:    12,
			HTTPCode:       200,
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
}

func TestLLMListAccessLog(t *testing.T) {
	ts := newTestServer(t)
	if ts == nil {
		return
	}

	base := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)

	for idx := range 20 {
		opts := &llmAccessLogOptions{
			CreatedAt:      base.Add(time.Duration(idx) * time.Second),
			Duration:       time.Duration(idx+1) * time.Second,
			Status:         corev1.AccessLog_Entry_Common_ALLOWED,
			Protocol:       corev1.Service_Spec_Config_LLM_OPENAI,
			Operation:      corev1.Service_Spec_Config_LLM_GENERATE,
			Source:         corev1.AccessLog_Entry_Info_LLM_UPSTREAM,
			UsageSource:    corev1.AccessLog_Entry_Info_LLM_Usage_PROVIDER,
			ModelRequested: "gpt-5",
			ModelEffective: "gpt-5",
			InputTokens:    uint64(10 * idx),
			OutputTokens:   1,
			TotalTokens:    uint64(10*idx) + 1,
			HTTPCode:       200,
			UserAgent:      "octelium-test",
		}

		if idx%2 == 0 {
			opts.Operation = corev1.Service_Spec_Config_LLM_EMBED
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
				HttpPaths: []string{"/v1/embeddings"},
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 10, len(resp.Items))
	}

	{
		resp, err := ts.srv.listLLMAccessLog(ts.ctx, &vllmv1.ListAccessLogRequest{
			Filter: &vllmv1.Filter{
				MinLatencyMs: 15000,
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
