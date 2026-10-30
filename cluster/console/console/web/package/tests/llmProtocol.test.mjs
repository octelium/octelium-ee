import assert from "node:assert/strict";
import { after, afterEach, before, describe, it, mock } from "node:test";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("LLM playground protocols", () => {
  let modules;
  let protocol;
  let runner;
  let types;
  let core;
  const options = {
    model: " example/model ", systemPrompt: " Be helpful ", temperature: "0.5",
    topP: "0.9", maxTokens: "512", reasoning: "", stream: false, apiShape: "chat",
  };
  const message = (extra = {}) => ({
    id: "message-1", role: "user", parts: [{ kind: "text", text: "Hello" }],
    attachments: [], createdAt: 0, ...extra,
  });
  const image = { id: "image-1", name: "example.jpg", mime: "image/jpg", dataURL: "data:image/jpg;base64,aGVsbG8=" };
  const run = async (extra = {}) => {
    const events = [];
    await runner.runCompletion({
      endpoint: "https://service.example.test/v1///", protocol: "openai", options,
      messages: [message()], signal: new AbortController().signal,
      onEvent: (event) => events.push(event), ...extra,
    });
    return events;
  };
  const streamResponse = (chunks) => new Response(new ReadableStream({
    start(controller) {
      for (const chunk of chunks) controller.enqueue(typeof chunk === "string" ? new TextEncoder().encode(chunk) : chunk);
      controller.close();
    },
  }), { headers: { "content-type": "text/event-stream; charset=utf-8" } });

  before(async () => {
    modules = await createConsoleModules();
    [protocol, runner, types, core] = await Promise.all([
      modules.load("components/LLMPlayground/protocols.ts"),
      modules.load("components/LLMPlayground/runner.ts"),
      modules.load("components/LLMPlayground/types.ts"),
      modules.load("apis/corev1/corev1.ts"),
    ]);
  });
  afterEach(() => mock.restoreAll());
  after(async () => modules?.close());

  for (const [name, endpoint, path] of [
    ["openai", "/v1", "/chat/completions"],
    ["anthropic", "/v1", "/messages"],
    ["gemini", "/v1beta", "/models/example%2Fmodel:generateContent"],
    ["bedrock", "", "/model/example%2Fmodel/converse"],
  ]) {
    it(`selects the ${name} endpoint and encoded request path`, () => {
      assert.equal(protocol.defaultEndpoint("https://service.example.test///", name), `https://service.example.test${endpoint}`);
      assert.equal(protocol.requestPath(name, options), path);
      assert.equal(protocol.supportsStreaming(name), name !== "bedrock");
    });
  }

  it("selects the Responses API and Gemini streaming request paths", () => {
    assert.equal(protocol.requestPath("openai", { ...options, apiShape: "responses" }), "/responses");
    assert.equal(protocol.requestPath("gemini", { ...options, stream: true }), "/models/example%2Fmodel:streamGenerateContent?alt=sse");
  });

  it("maps every configured protocol and defaults unset values to OpenAI", () => {
    const value = core.Service_Spec_Config_LLM_Protocol;
    for (const [key, expected] of [["ANTHROPIC", "anthropic"], ["GEMINI", "gemini"], ["BEDROCK", "bedrock"], ["OPENAI", "openai"]]) {
      assert.equal(types.protocolFromEnum(value[key]), expected);
    }
    assert.equal(types.protocolFromEnum(), "openai");
    assert.equal(types.protocolFromEnum(999), "openai");
  });

  it("extracts only user-visible text parts for request history", () => {
    assert.equal(protocol.messageText(message({ parts: [
      { kind: "thinking", text: "private" }, { kind: "text", text: "Hello" },
      { kind: "tool", name: "search", input: "{}" }, { kind: "text", text: " world" },
    ] })), "Hello world");
  });

  it("builds OpenAI chat history, images, system instructions and reasoning settings", () => {
    assert.deepEqual(protocol.buildBody("openai", [message({ attachments: [image] }), message({ role: "assistant" })], { ...options, stream: true, reasoning: "high" }), {
      model: "example/model",
      messages: [
        { role: "system", content: "Be helpful" },
        { role: "user", content: [{ type: "text", text: "Hello" }, { type: "image_url", image_url: { url: image.dataURL } }] },
        { role: "assistant", content: "Hello" },
      ],
      stream: true, stream_options: { include_usage: true }, temperature: 0.5,
      top_p: 0.9, max_completion_tokens: 512, reasoning_effort: "high",
    });
  });

  it("builds OpenAI Responses input and output-token settings", () => {
    assert.deepEqual(protocol.buildBody("openai", [message({ attachments: [image] })], { ...options, apiShape: "responses", reasoning: "low" }), {
      model: "example/model", input: [
        { role: "system", content: "Be helpful" },
        { role: "user", content: [{ type: "input_text", text: "Hello" }, { type: "input_image", image_url: image.dataURL }] },
      ], temperature: 0.5, top_p: 0.9, max_output_tokens: 512, reasoning: { effort: "low" },
    });
  });

  it("builds Anthropic image content and thinking budgets", () => {
    assert.deepEqual(protocol.buildBody("anthropic", [message({ attachments: [image] })], { ...options, reasoning: "128", stream: true }), {
      model: "example/model", max_tokens: 512, system: "Be helpful",
      messages: [{ role: "user", content: [
        { type: "image", source: { type: "base64", media_type: image.mime, data: "aGVsbG8=" } },
        { type: "text", text: "Hello" },
      ] }], temperature: 0.5, top_p: 0.9, thinking: { type: "enabled", budget_tokens: 128 }, stream: true,
    });
    assert.equal(protocol.buildBody("anthropic", [], { ...options, maxTokens: "" }).max_tokens, 4096);
  });

  it("builds Gemini model roles, inline image data and generation settings", () => {
    assert.deepEqual(protocol.buildBody("gemini", [message({ role: "assistant", attachments: [image] })], { ...options, reasoning: "0" }), {
      contents: [{ role: "model", parts: [{ text: "Hello" }, { inline_data: { mime_type: image.mime, data: "aGVsbG8=" } }] }],
      systemInstruction: { parts: [{ text: "Be helpful" }] },
      generationConfig: { temperature: 0.5, topP: 0.9, maxOutputTokens: 512, thinkingConfig: { thinkingBudget: 0, includeThoughts: true } },
    });
  });

  it("builds Bedrock Converse content and normalizes JPEG format names", () => {
    assert.deepEqual(protocol.buildBody("bedrock", [message({ attachments: [image] })], options), {
      messages: [{ role: "user", content: [{ text: "Hello" }, { image: { format: "jpeg", source: { bytes: "aGVsbG8=" } } }] }],
      system: [{ text: "Be helpful" }], inferenceConfig: { temperature: 0.5, topP: 0.9, maxTokens: 512 },
    });
  });

  for (const name of ["openai", "anthropic", "gemini", "bedrock"]) {
    it(`omits blank and non-finite optional settings for ${name}`, () => {
      const body = protocol.buildBody(name, [], { ...options, systemPrompt: " ", temperature: "NaN", topP: "Infinity", maxTokens: " ", reasoning: " " });
      const json = JSON.parse(JSON.stringify(body));
      assert.equal(json.system, undefined);
      assert.equal(json.systemInstruction, undefined);
      assert.equal(json.temperature, undefined);
      assert.equal(json.top_p, undefined);
      assert.equal(json.inferenceConfig?.temperature, undefined);
      assert.equal(json.generationConfig?.topP, undefined);
      if (name === "openai") assert.deepEqual(json.messages, []);
    });
  }

  it("uses cookie credentials and removes caller-supplied API authentication", async () => {
    const fetch = mock.method(globalThis, "fetch", async () => new Response("{}"));
    const controller = new AbortController();
    await protocol.fetchThroughOctelium("https://service.example.test", { method: "POST", credentials: "omit", signal: controller.signal, headers: {
      Authorization: "secret", "X-Api-Key": "secret", "API-Key": "secret", "X-Request-Id": "request-1",
    } });
    const init = fetch.mock.calls[0].arguments[1];
    assert.equal(init.credentials, "include");
    assert.equal(init.signal, controller.signal);
    assert.deepEqual([...init.headers], [["x-request-id", "request-1"]]);
  });

  for (const name of ["openai", "anthropic", "gemini", "bedrock"]) {
    it(`sets the JSON request content type for ${name}`, () => {
      assert.equal(protocol.requestHeaders(name)["content-type"], "application/json");
    });
    it(`ignores empty SSE and completion envelopes for ${name}`, () => {
      assert.deepEqual(protocol.parseSSEChunk(name, "chat", "[DONE]"), []);
      assert.deepEqual(protocol.parseSSEChunk(name, "chat", "not json"), []);
      assert.deepEqual(protocol.parseSSEChunk(name, "chat", "{}"), []);
      assert.deepEqual(protocol.parseCompleteBody(name, "chat", {}), []);
    });
  }

  it("parses OpenAI text, reasoning, tools, finish state and detailed usage", () => {
    const events = protocol.parseSSEChunk("openai", "chat", JSON.stringify({
      model: "model-1", choices: [{ delta: { content: "Hello", reasoning_content: "Thinking", tool_calls: [{ function: { name: "search", arguments: "{}" } }] }, finish_reason: "tool_calls" }],
      usage: { prompt_tokens: 10, completion_tokens: 3, total_tokens: 13, completion_tokens_details: { reasoning_tokens: 2 }, prompt_tokens_details: { cached_tokens: 4 } },
    }));
    assert.deepEqual(events, [
      { type: "model", model: "model-1" }, { type: "text", text: "Hello" },
      { type: "thinking", text: "Thinking" }, { type: "tool", name: "search", input: "{}" },
      { type: "usage", usage: { inputTokens: 10, outputTokens: 3, totalTokens: 13, reasoningTokens: 2, cacheReadTokens: 4 } },
      { type: "finish", reason: "tool_calls" },
    ]);
    assert.deepEqual(protocol.parseSSEChunk("openai", "chat", '{"choices":[{"delta":{"reasoning":"alternate"}}]}'), [{ type: "thinking", text: "alternate" }]);
  });

  for (const type of ["response.output_text.delta", "response.reasoning_summary_text.delta", "response.reasoning_text.delta"]) {
    it(`parses the Responses event ${type}`, () => {
      assert.deepEqual(protocol.parseSSEChunk("openai", "responses", JSON.stringify({ type, delta: "part" })), [{ type: type === "response.output_text.delta" ? "text" : "thinking", text: "part" }]);
    });
  }

  it("parses Responses completion and alternate usage field names", () => {
    assert.deepEqual(protocol.parseSSEChunk("openai", "responses", JSON.stringify({ type: "response.completed", response: {
      model: "responses-model", usage: { input_tokens: 4, output_tokens: 2, total_tokens: 6, output_tokens_details: { reasoning_tokens: 1 }, input_tokens_details: { cached_tokens: 3 } },
    } })), [
      { type: "model", model: "responses-model" },
      { type: "usage", usage: { inputTokens: 4, outputTokens: 2, totalTokens: 6, reasoningTokens: 1, cacheReadTokens: 3 } },
      { type: "finish", reason: "completed" },
    ]);
  });

  it("parses Anthropic message, content, thinking and tool delta events", () => {
    const parse = (event) => protocol.parseSSEChunk("anthropic", "chat", JSON.stringify(event));
    assert.deepEqual(parse({ type: "message_start", message: { model: "claude", usage: { input_tokens: 8, output_tokens: 0, cache_read_input_tokens: 2, cache_creation_input_tokens: 1 } } }), [
      { type: "model", model: "claude" }, { type: "usage", usage: { inputTokens: 8, outputTokens: 0, totalTokens: 8, cacheReadTokens: 2, cacheWriteTokens: 1 } },
    ]);
    assert.deepEqual(parse({ type: "content_block_start", content_block: { type: "tool_use", name: "search" } }), [{ type: "tool", name: "search", input: "" }]);
    assert.deepEqual(parse({ type: "content_block_delta", delta: { type: "text_delta", text: "Hello" } }), [{ type: "text", text: "Hello" }]);
    assert.deepEqual(parse({ type: "content_block_delta", delta: { type: "thinking_delta", thinking: "Think" } }), [{ type: "thinking", text: "Think" }]);
    assert.deepEqual(parse({ type: "content_block_delta", delta: { type: "input_json_delta", partial_json: '{"q":' } }), [{ type: "tool", name: "", input: '{"q":' }]);
    assert.deepEqual(parse({ type: "message_delta", delta: { stop_reason: "end_turn" } }), [{ type: "finish", reason: "end_turn" }]);
  });

  it("parses Gemini text, thoughts, tools, usage and completion state", () => {
    const body = {
      modelVersion: "gemini-model", candidates: [{ content: { parts: [{ text: "Think", thought: true }, { text: "Hello" }, { functionCall: { name: "search", args: { q: "test" } } }] }, finishReason: "STOP" }],
      usageMetadata: { promptTokenCount: 5, candidatesTokenCount: 3, totalTokenCount: 8, thoughtsTokenCount: 1, cachedContentTokenCount: 2 },
    };
    const expected = [
      { type: "model", model: "gemini-model" }, { type: "thinking", text: "Think" },
      { type: "text", text: "Hello" }, { type: "tool", name: "search", input: '{\n  "q": "test"\n}' },
      { type: "usage", usage: { inputTokens: 5, outputTokens: 3, totalTokens: 8, reasoningTokens: 1, cacheReadTokens: 2 } },
      { type: "finish", reason: "STOP" },
    ];
    assert.deepEqual(protocol.parseSSEChunk("gemini", "chat", JSON.stringify(body)), expected);
    assert.deepEqual(protocol.parseCompleteBody("gemini", "chat", body), expected);
  });

  it("parses complete OpenAI chat responses and function calls", () => {
    assert.deepEqual(protocol.parseCompleteBody("openai", "chat", { model: "chat-model", choices: [{ message: { content: "Answer", reasoning_content: "Think", tool_calls: [{ function: { name: "list", arguments: "{}" } }] }, finish_reason: "stop" }] }), [
      { type: "model", model: "chat-model" }, { type: "thinking", text: "Think" },
      { type: "text", text: "Answer" }, { type: "tool", name: "list", input: "{}" }, { type: "finish", reason: "stop" },
    ]);
  });

  it("parses complete Responses reasoning, output and tool items", () => {
    assert.deepEqual(protocol.parseCompleteBody("openai", "responses", { output: [
      { type: "reasoning", summary: [{ text: "Think" }] },
      { type: "message", content: [{ type: "output_text", text: "Answer" }] },
      { type: "function_call", name: "search", arguments: "{}" },
    ], status: "completed" }), [
      { type: "thinking", text: "Think" }, { type: "text", text: "Answer" },
      { type: "tool", name: "search", input: "{}" }, { type: "finish", reason: "completed" },
    ]);
  });

  it("parses complete Anthropic text, thinking and tool blocks", () => {
    assert.deepEqual(protocol.parseCompleteBody("anthropic", "chat", { model: "claude", content: [
      { type: "thinking", thinking: "Think" }, { type: "text", text: "Answer" },
      { type: "tool_use", name: "search", input: { q: 1 } },
    ], stop_reason: "tool_use" }), [
      { type: "model", model: "claude" }, { type: "thinking", text: "Think" },
      { type: "text", text: "Answer" }, { type: "tool", name: "search", input: '{\n  "q": 1\n}' }, { type: "finish", reason: "tool_use" },
    ]);
  });

  it("parses Bedrock text, reasoning, tools and cache accounting", () => {
    assert.deepEqual(protocol.parseCompleteBody("bedrock", "chat", {
      output: { message: { content: [{ text: "Answer" }, { reasoningContent: { reasoningText: { text: "Think" } } }, { toolUse: { name: "search", input: {} } }] } },
      usage: { inputTokens: 4, outputTokens: 2, totalTokens: 6, cacheReadInputTokens: 1, cacheWriteInputTokens: 3 }, stopReason: "tool_use",
    }), [
      { type: "text", text: "Answer" }, { type: "thinking", text: "Think" }, { type: "tool", name: "search", input: "{}" },
      { type: "usage", usage: { inputTokens: 4, outputTokens: 2, totalTokens: 6, cacheReadTokens: 1, cacheWriteTokens: 3 } }, { type: "finish", reason: "tool_use" },
    ]);
  });

  for (const body of [{ error: { message: "Denied" } }, { error: "Denied" }, { message: "Denied" }, { Message: "Denied" }, { detail: "Denied" }]) {
    it(`extracts the HTTP error shape ${JSON.stringify(body)}`, () => assert.equal(protocol.errorMessage("openai", 403, JSON.stringify(body)), "403: Denied"));
  }

  it("bounds raw HTTP errors and handles unknown JSON error shapes", () => {
    assert.equal(protocol.errorMessage("anthropic", 502, "x".repeat(500)), `502: ${"x".repeat(400)}`);
    assert.equal(protocol.errorMessage("gemini", 503, "{}"), "The gemini request failed with status 503.");
    assert.equal(protocol.errorMessage("bedrock", 500, ""), "The bedrock request failed with status 500.");
  });

  it("sends the completion request with its abort signal and parses JSON fallback", async () => {
    const fetch = mock.method(globalThis, "fetch", async () => new Response('{"choices":[{"message":{"content":"Answer"}}]}', { headers: { "content-type": "application/json" } }));
    const controller = new AbortController();
    assert.deepEqual(await run({ options: { ...options, stream: true }, signal: controller.signal }), [{ type: "text", text: "Answer" }]);
    const [url, init] = fetch.mock.calls[0].arguments;
    assert.equal(url, "https://service.example.test/v1/chat/completions");
    assert.equal(init.signal, controller.signal);
    assert.equal(init.credentials, "include");
    assert.equal(init.headers.has("authorization"), false);
    assert.equal(JSON.parse(init.body).stream, true);
  });

  it("disables Bedrock streaming even when requested", async () => {
    const fetch = mock.method(globalThis, "fetch", async () => new Response("{}"));
    assert.deepEqual(await run({ protocol: "bedrock", options: { ...options, stream: true } }), []);
    assert.equal(fetch.mock.calls[0].arguments[0], "https://service.example.test/v1/model/example%2Fmodel/converse");
    assert.equal(JSON.parse(fetch.mock.calls[0].arguments[1].body).stream, undefined);
  });

  it("decodes fragmented SSE frames and UTF-8 characters across byte boundaries", async () => {
    const bytes = new TextEncoder().encode('event: message\ndata: {"choices":[{"delta":{"content":"café 工具"}}]}\n\ndata: [DONE]\n\n');
    mock.method(globalThis, "fetch", async () => streamResponse([...bytes].map((byte) => new Uint8Array([byte]))));
    assert.deepEqual(await run({ options: { ...options, stream: true } }), [{ type: "text", text: "café 工具" }]);
  });

  it("joins multiline SSE payloads and accepts a final unterminated frame", async () => {
    mock.method(globalThis, "fetch", async () => streamResponse([': keep-alive\n\ndata: {"choices":\ndata: [{"delta":{"content":"one"}}]}\n\ndata: {"choices":[{"delta":{"content":"two"}}]}']));
    assert.deepEqual(await run({ options: { ...options, stream: true } }), [{ type: "text", text: "one" }, { type: "text", text: "two" }]);
  });

  it("propagates HTTP errors and network cancellation", async () => {
    mock.method(globalThis, "fetch", async () => new Response('{"error":{"message":"Policy denied"}}', { status: 403 }));
    await assert.rejects(run(), /403: Policy denied/);
    mock.restoreAll();
    mock.method(globalThis, "fetch", async () => { throw new DOMException("Aborted", "AbortError"); });
    await assert.rejects(run(), { name: "AbortError" });
  });

  it("CONSOLE-008: parses more than one CRLF-delimited SSE frame", { todo: "CONSOLE-008" }, async () => {
    mock.method(globalThis, "fetch", async () => streamResponse(['data: {"choices":[{"delta":{"content":"one"}}]}\r\n\r\ndata: {"choices":[{"delta":{"content":"two"}}]}\r\n\r\n']));
    assert.deepEqual(await run({ options: { ...options, stream: true } }), [{ type: "text", text: "one" }, { type: "text", text: "two" }]);
  });

  it("CONSOLE-009: preserves OpenAI tool argument continuation chunks", { todo: "CONSOLE-009" }, () => {
    assert.deepEqual(protocol.parseSSEChunk("openai", "chat", JSON.stringify({ choices: [{ delta: { tool_calls: [{ index: 0, function: { arguments: '"complete"}' } }] } }] })), [{ type: "tool", name: "", input: '"complete"}' }]);
  });

  it("CONSOLE-010: surfaces an upstream error delivered inside a successful SSE response", { todo: "CONSOLE-010" }, async () => {
    mock.method(globalThis, "fetch", async () => streamResponse(['data: {"error":{"message":"Quota exceeded","code":"rate_limit"}}\n\n']));
    await assert.rejects(run({ options: { ...options, stream: true } }), /Quota exceeded/);
  });

  it("CONSOLE-011: rejects malformed successful JSON instead of returning an empty answer", { todo: "CONSOLE-011" }, async () => {
    mock.method(globalThis, "fetch", async () => new Response("<html>gateway failure</html>", { headers: { "content-type": "application/json" } }));
    await assert.rejects(run(), /JSON|response|Unexpected/i);
  });

  it("CONSOLE-021: ignores a null SSE payload without crashing the stream", { todo: "CONSOLE-021" }, () => {
    assert.deepEqual(protocol.parseSSEChunk("openai", "chat", "null"), []);
  });
});
