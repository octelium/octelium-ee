import assert from "node:assert/strict";
import { after, afterEach, before, beforeEach, describe, it, mock } from "node:test";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("Agent protocol, reducer and client", () => {
  let modules;
  let protocol;
  let reducer;
  let clientModule;
  let client;
  let sources;
  const previousEventSource = globalThis.EventSource;
  const message = (extra = {}) => ({ id: "message-1", conversationId: "conversation-1", role: "assistant", status: "streaming", createdAt: "2026-10-30T00:00:00Z", blocks: [], ...extra });
  const run = (extra = {}) => ({ id: "run-1", conversationId: "conversation-1", status: "running", createdAt: "2026-10-30T00:00:00Z", userMessageId: "user-1", assistantMessageId: "message-1", lastSeq: 0, ...extra });
  const event = (type, extra = {}) => ({ type, runId: "run-1", conversationId: "conversation-1", seq: 2, ts: "2026-10-30T00:00:00Z", ...extra });
  const state = (extra = {}) => ({ messages: [message()], run: run(), lastSeq: 1, ...extra });
  const respond = (body = {}, init = {}) => mock.method(globalThis, "fetch", async () => new Response(init.status === 204 ? null : JSON.stringify(body), init));

  before(async () => {
    modules = await createConsoleModules();
    [protocol, reducer, clientModule] = await Promise.all([
      modules.load("apis/consoleagent/protocol.ts"),
      modules.load("pages/Agent/reducer.ts"),
      modules.load("pages/Agent/client.ts"),
    ]);
    globalThis.EventSource = class {
      static CLOSED = 2;
      readyState = 1;
      listeners = new Map();
      closeCount = 0;
      constructor(url, options) {
        this.url = url;
        this.options = options;
        sources.push(this);
      }
      addEventListener(type, listener) { this.listeners.set(type, listener); }
      close() { this.closeCount++; this.readyState = 2; }
      emit(type, value) { this.listeners.get(type)?.({ data: typeof value === "string" ? value : JSON.stringify(value) }); }
    };
  });
  beforeEach(() => {
    client = new clientModule.AgentClient("https://agent.example.test///");
    sources = [];
  });
  afterEach(() => mock.restoreAll());
  after(async () => {
    await modules?.close();
    if (previousEventSource === undefined) delete globalThis.EventSource;
    else globalThis.EventSource = previousEventSource;
  });

  for (const name of ["alice", "my namespace/service", "café 工具", "literal%2Fname", "name?#&"]) {
    it(`round-trips the encoded resource name ${JSON.stringify(name)}`, () => {
      const ref = { apiVersion: "core/v1", kind: "Service", name };
      assert.deepEqual(protocol.parseResourceURI(protocol.formatResourceURI(ref)), ref);
    });
  }
  for (const uri of ["https://example.test", "octelium://resource/", "octelium://resource/core/v1/User/", "octelium://resource/core//User/alice", "octelium://resource/core/v1/User/a/extra", "octelium://resource/core/v1/User/%broken"]) {
    it(`rejects malformed resource URI ${JSON.stringify(uri)}`, () => assert.equal(protocol.parseResourceURI(uri), undefined));
  }
  it("distinguishes resource API versions, kinds, names and UID-only references", () => {
    assert.notEqual(protocol.resourceKey({ apiVersion: "core/v1", kind: "Secret", name: "a" }), protocol.resourceKey({ apiVersion: "access/v1", kind: "Secret", name: "a" }));
    assert.equal(protocol.resourceKey({ apiVersion: "core/v1", kind: "User", uid: "uid-1" }), "core/v1/User/uid-1");
  });
  for (const status of ["running", "awaiting_approval", "completed", "failed", "cancelled"]) {
    it(`identifies terminal run state ${status}`, () => assert.equal(protocol.isTerminalRunStatus(status), ["completed", "failed", "cancelled"].includes(status)));
  }
  for (const status of ["pending", "completed", "failed", "cancelled"]) {
    it(`identifies terminal login state ${status}`, () => assert.equal(protocol.isTerminalLoginStatus(status), status !== "pending"));
  }

  for (const type of ["run.started", "run.updated", "run.completed", "run.failed", "run.cancelled"]) {
    it(`applies ${type} and preserves the transcript`, () => {
      const before = state();
      const current = run({ status: type.split(".")[1] });
      const next = reducer.applyEvent(before, event(type, { run: current }));
      assert.equal(next.run, current);
      assert.equal(next.messages, before.messages);
      assert.equal(next.lastSeq, 2);
      assert.equal(before.lastSeq, 1);
    });
  }
  it("ignores duplicate and out-of-order events in the same run", () => {
    const current = state({ lastSeq: 5 });
    assert.equal(reducer.applyEvent(current, event("run.updated", { seq: 5, run: run() })), current);
    assert.equal(reducer.applyEvent(current, event("run.updated", { seq: 4, run: run() })), current);
  });
  it("accepts a new run with a reset event sequence", () => {
    const nextRun = run({ id: "run-2" });
    const next = reducer.applyEvent(state({ lastSeq: 100 }), event("run.started", { seq: 1, runId: "run-2", run: nextRun }));
    assert.equal(next.run, nextRun);
    assert.equal(next.lastSeq, 1);
  });
  for (const type of ["message.created", "message.completed"]) {
    it(`upserts ${type} by message ID without mutating previous messages`, () => {
      const previous = state();
      const replacement = message({ status: "completed" });
      const next = reducer.applyEvent(previous, event(type, { message: replacement }));
      assert.deepEqual(next.messages, [replacement]);
      assert.equal(previous.messages[0].status, "streaming");
      const appended = reducer.applyEvent(previous, event(type, { message: message({ id: "message-2" }) }));
      assert.deepEqual(appended.messages.map((item) => item.id), ["message-1", "message-2"]);
    });
  }
  it("creates and replaces blocks without retaining duplicate IDs", () => {
    const original = { id: "block-1", type: "markdown", text: "Old", createdAt: "" };
    const keep = { ...original, id: "block-2" };
    const updated = { ...original, text: "New" };
    const previous = state({ messages: [message({ blocks: [original, keep] })] });
    const next = reducer.applyEvent(previous, event("block.created", { messageId: "message-1", block: updated }));
    assert.deepEqual(next.messages[0].blocks, [keep, updated]);
    assert.deepEqual(previous.messages[0].blocks, [original, keep]);
  });
  it("updates the matching block while preserving other blocks and messages", () => {
    const block = { id: "block-1", type: "markdown", text: "Old", createdAt: "" };
    const otherMessage = message({ id: "message-2" });
    const previous = state({ messages: [message({ blocks: [block] }), otherMessage] });
    const updated = { ...block, text: "New" };
    const next = reducer.applyEvent(previous, event("block.updated", { messageId: "message-1", block: updated }));
    assert.deepEqual(next.messages[0].blocks, [updated]);
    assert.equal(next.messages[1], otherMessage);
    assert.equal(previous.messages[0].blocks[0].text, "Old");
  });
  for (const type of ["markdown", "thinking", "notice"]) {
    it(`handles text deltas for a ${type} block`, () => {
      const block = { id: "block-1", type, text: "Start", createdAt: "" };
      const previous = state({ messages: [message({ blocks: [block] })] });
      const next = reducer.applyEvent(previous, event("block.delta", { messageId: "message-1", blockId: "block-1", text: " end" }));
      assert.equal(next.messages[0].blocks[0].text, type === "notice" ? "Start" : "Start end");
      assert.equal(previous.messages[0].blocks[0].text, "Start");
    });
  }
  it("ignores missing message and block targets without inventing transcript entries", () => {
    const previous = state();
    for (const update of [
      event("block.created", { messageId: "missing", block: { id: "new" } }),
      event("block.updated", { messageId: "message-1", block: { id: "missing" } }),
      event("block.delta", { messageId: "message-1", blockId: "missing", text: "ignored" }),
      event("conversation.updated", { conversation: { title: "Renamed" } }),
    ]) assert.deepEqual(reducer.applyEvent(previous, update).messages, previous.messages);
  });

  for (const [name, args, method, path, body] of [
    ["info", [], "GET", "/info"], ["listConversations", [], "GET", "/conversations"],
    ["getConversation", ["c-1"], "GET", "/conversations/c-1"],
    ["listModels", [], "GET", "/models"], ["listAuthProviders", [], "GET", "/auth/providers"],
    ["getLogin", ["login-1"], "GET", "/auth/logins/login-1"],
    ["createConversation", [{ text: "Hello", attachments: ["file-1"] }], "POST", "/conversations", { input: { text: "Hello", attachments: ["file-1"] } }],
    ["renameConversation", ["c-1", "New name"], "PATCH", "/conversations/c-1", { title: "New name" }],
    ["deleteConversation", ["c-1"], "DELETE", "/conversations/c-1"],
    ["startRun", ["c-1", { text: "Next" }], "POST", "/conversations/c-1/runs", { input: { text: "Next" } }],
    ["cancelRun", ["r-1"], "POST", "/runs/r-1/cancel", {}],
    ["decideApproval", ["r-1", "a-1", "reject", "Unsafe"], "POST", "/runs/r-1/approvals/a-1", { decision: "reject", reason: "Unsafe" }],
    ["setModel", [{ provider: "example", id: "model", thinkingLevel: "high" }], "PUT", "/model", { provider: "example", id: "model", thinkingLevel: "high" }],
    ["logout", ["provider-1"], "DELETE", "/auth/providers/provider-1"],
    ["startLogin", ["provider-1"], "POST", "/auth/logins", { provider: "provider-1", selectModel: true }],
    ["answerLogin", ["login-1", "prompt-1", "answer"], "POST", "/auth/logins/login-1/prompts/prompt-1", { value: "answer" }],
    ["cancelLogin", ["login-1"], "DELETE", "/auth/logins/login-1"],
  ]) {
    it(`sends ${name} to its API endpoint with credentials and the expected body`, async () => {
      const fetch = respond({ ok: true });
      assert.deepEqual(await client[name](...args), { ok: true });
      const [url, init] = fetch.mock.calls[0].arguments;
      assert.equal(url, `https://agent.example.test/v1${path}`);
      assert.equal(init.method, method);
      assert.equal(init.credentials, "include");
      assert.equal(init.body, body === undefined ? undefined : JSON.stringify(body));
      assert.deepEqual(init.headers, body === undefined ? undefined : { "content-type": "application/json" });
    });
  }
  it("encodes conversation search text and forwards cancellation", async () => {
    const fetch = respond({ items: [] });
    const controller = new AbortController();
    await client.searchConversations("a & café?", controller.signal);
    assert.equal(fetch.mock.calls[0].arguments[0], "https://agent.example.test/v1/conversations/search?q=a%20%26%20caf%C3%A9%3F");
    assert.equal(fetch.mock.calls[0].arguments[1].signal, controller.signal);
    await client.info(controller.signal);
    assert.equal(fetch.mock.calls[1].arguments[1].signal, controller.signal);
  });
  it("accepts 204 responses without trying to parse JSON", async () => {
    respond(undefined, { status: 204 });
    assert.equal(await client.deleteConversation("c-1"), undefined);
  });
  it("preserves the status and structured error code on failed requests", async () => {
    respond({ error: { code: "PERMISSION_DENIED", message: "Cannot view conversations" } }, { status: 403 });
    await assert.rejects(client.listConversations(), (error) => {
      assert.ok(error instanceof clientModule.AgentRequestError);
      assert.equal(error.status, 403);
      assert.equal(error.code, "PERMISSION_DENIED");
      assert.equal(error.message, "Cannot view conversations");
      return true;
    });
  });
  it("uses the HTTP status when the server error is not JSON", async () => {
    mock.method(globalThis, "fetch", async () => new Response("Bad gateway", { status: 502, statusText: "Bad Gateway" }));
    await assert.rejects(client.info(), { status: 502, message: "502 Bad Gateway", code: undefined });
  });
  for (const mime of ["text/plain", ""]) {
    it(`uploads raw file bytes with MIME type ${JSON.stringify(mime)}`, async () => {
      const fetch = respond({ id: "file-1" });
      const file = new File(["content"], "a & b.txt", { type: mime });
      await client.uploadFile(file);
      const [url, init] = fetch.mock.calls[0].arguments;
      assert.equal(url, "https://agent.example.test/v1/files?name=a%20%26%20b.txt");
      assert.equal(init.body, file);
      assert.deepEqual(init.headers, { "content-type": mime || "application/octet-stream" });
    });
  }
  it("downloads artifacts with credentials and preserves their bytes", async () => {
    const fetch = mock.method(globalThis, "fetch", async () => new Response("artifact content"));
    assert.equal(await (await client.downloadArtifact({ url: "/v1/artifacts/a-1", name: "report" })).text(), "artifact content");
    assert.deepEqual(fetch.mock.calls[0].arguments, ["https://agent.example.test/v1/artifacts/a-1", { credentials: "include" }]);
  });
  it("reports failed artifact downloads with the artifact name", async () => {
    respond({}, { status: 404 });
    await assert.rejects(client.downloadArtifact({ url: "/missing", name: "report.csv" }), { status: 404, message: "Could not download report.csv" });
  });
  it("subscribes to every event type and ignores malformed event JSON", () => {
    const received = [];
    const close = mock.fn();
    const dispose = client.subscribe("run-1", 12, (value) => received.push(value), close);
    assert.equal(sources[0].url, "https://agent.example.test/v1/runs/run-1/events?after=12");
    assert.deepEqual(sources[0].options, { withCredentials: true });
    assert.deepEqual([...sources[0].listeners.keys()], clientModule.AGENT_EVENT_TYPES);
    sources[0].emit("block.delta", "not JSON");
    const update = event("block.delta", { text: "Hello" });
    sources[0].emit("block.delta", update);
    assert.deepEqual(received, [update]);
    dispose();
    assert.equal(close.mock.callCount(), 0);
    assert.equal(sources[0].closeCount, 1);
  });
  for (const type of ["run.completed", "run.failed", "run.cancelled"]) {
    it(`closes a subscription exactly once after ${type}`, () => {
      const received = mock.fn();
      const close = mock.fn();
      client.subscribe("run-1", 0, received, close);
      sources[0].emit(type, event(type, { run: run() }));
      sources[0].onerror();
      assert.equal(received.mock.callCount(), 1);
      assert.equal(close.mock.callCount(), 1);
      assert.equal(sources[0].closeCount, 1);
    });
  }
  it("keeps transient EventSource errors reconnectable and handles permanent closure", () => {
    const close = mock.fn();
    client.subscribe("run-1", 0, () => {}, close);
    sources[0].readyState = 0;
    sources[0].onerror();
    assert.equal(close.mock.callCount(), 0);
    sources[0].readyState = 2;
    sources[0].onerror();
    sources[0].onerror();
    assert.equal(close.mock.callCount(), 1);
  });

  it("CONSOLE-013: ignores a late event from the previous run", { todo: "CONSOLE-013" }, () => {
    const current = state({ run: run({ id: "new-run" }), lastSeq: 3 });
    assert.equal(reducer.applyEvent(current, event("run.completed", { seq: 99, run: run({ status: "completed" }) })), current);
  });
});
