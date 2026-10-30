import assert from "node:assert/strict";
import { after, afterEach, before, describe, it, mock } from "node:test";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("MCP schema arguments and transport", () => {
  let modules;
  let schema;
  let client;
  const previousWindow = globalThis.window;
  const config = {
    url: "https://mcp.example.test/",
    protocolVersion: "2025-03-26",
    clientName: "Console",
    clientVersion: "1.0",
    capabilities: "{}",
  };
  const field = (kind, extra = {}) => ({ name: "value", kind, required: false, ...extra });
  const respond = (body, options = {}) =>
    mock.method(globalThis, "fetch", async (url, init) =>
      new Response(
        body === undefined ? null : typeof body === "string" ? body : JSON.stringify({ ...body, jsonrpc: "2.0", id: JSON.parse(init.body).id }),
        { headers: { "content-type": "application/json" }, ...options },
      ),
    );
  const send = (extra = {}) => client.sendMCP({ config, method: "tools/list", ...extra });

  before(async () => {
    globalThis.window = { setTimeout, clearTimeout };
    modules = await createConsoleModules();
    [schema, client] = await Promise.all([
      modules.load("components/MCPPlayground/schema.ts"),
      modules.load("components/MCPPlayground/client.ts"),
    ]);
  });

  afterEach(() => mock.restoreAll());

  after(async () => {
    await modules?.close();
    if (previousWindow === undefined) delete globalThis.window;
    else globalThis.window = previousWindow;
  });

  for (const input of [undefined, null, [], true, "schema", {}, { properties: [] }]) {
    it(`ignores non-object schema properties: ${JSON.stringify(input)}`, () => {
      assert.deepEqual(schema.schemaFields(input), []);
    });
  }

  it("preserves field metadata, nullable primitive types and JSON defaults", () => {
    assert.deepEqual(schema.schemaFields({
      required: ["count", 7],
      properties: {
        count: { type: ["null", "integer"], title: "Count", description: "Items", minimum: 1, maximum: 9, default: 3 },
        options: { type: "object", default: { active: true } },
      },
    }), [
      { name: "count", title: "Count", description: "Items", kind: "integer", required: true, options: undefined, placeholder: "3", minimum: 1, maximum: 9 },
      { name: "options", title: undefined, description: undefined, kind: "json", required: false, options: undefined, placeholder: '{"active":true}', minimum: undefined, maximum: undefined },
    ]);
  });

  it("uses enum controls only for primitive enum members", () => {
    const fields = schema.schemaFields({ properties: {
      primitive: { enum: ["one", 2, false, null] },
      object: { enum: [{ id: 1 }] },
      empty: { type: "string", enum: [] },
      missing: null,
    } });
    assert.deepEqual(fields.map(({ name, kind, options }) => ({ name, kind, options })), [
      { name: "primitive", kind: "enum", options: ['"one"', "2", "false", "null"] },
      { name: "object", kind: "json", options: ['{"id":1}'] },
      { name: "empty", kind: "string", options: [] },
      { name: "missing", kind: "json", options: undefined },
    ]);
  });

  it("filters malformed prompt arguments and requires an explicit required flag", () => {
    assert.deepEqual(schema.promptFields([null, {}, { name: 3 }, {
      name: "topic", title: "Topic", description: "Explain", required: true,
    }, { name: "optional", title: 3, description: false, required: "true" }]), [
      { name: "topic", title: "Topic", description: "Explain", kind: "string", required: true },
      { name: "optional", title: undefined, description: undefined, kind: "string", required: false },
    ]);
    assert.deepEqual(schema.promptFields({}), []);
  });

  for (const [option, label] of [['"hello"', "hello"], ["false", "false"], ["null", "null"], ["4", "4"]]) {
    it(`labels the enum member ${option}`, () => assert.equal(schema.optionLabel(option), label));
  }

  for (const [kind, raw, expected] of [
    ["string", "  keep whitespace  ", "  keep whitespace  "],
    ["number", " 2.5 ", 2.5], ["number", "0", 0], ["integer", "-2", -2],
    ["boolean", "true", true], ["boolean", "false", false],
    ["enum", '"two"', "two"], ["enum", "null", null],
    ["json", '{"nested":[1,false]}', { nested: [1, false] }],
  ]) {
    it(`converts a ${kind} argument from ${JSON.stringify(raw)}`, () => {
      assert.deepEqual(schema.buildArguments([field(kind)], { value: raw }), { value: expected });
    });
  }

  for (const [kind, raw, error] of [
    ["number", "Infinity", /must be a number/], ["number", "NaN", /must be a number/],
    ["integer", "2.2", /must be an integer/], ["integer", "invalid", /must be an integer/],
    ["json", "{", /must be valid JSON/],
  ]) {
    it(`rejects the invalid ${kind} argument ${raw}`, () => {
      assert.throws(() => schema.buildArguments([field(kind)], { value: raw }), error);
    });
  }

  it("omits blank optional inputs and rejects missing required values", () => {
    assert.deepEqual(schema.buildArguments([field("number")], { value: " \n " }), {});
    assert.deepEqual(schema.buildArguments([field("string")], {}), {});
    assert.throws(() => schema.buildArguments([field("string", { required: true })], {}), /“value” is required/);
    assert.throws(() => schema.buildArguments([field("boolean", { required: true })], { value: " " }), /required/);
  });

  it("round-trips populated fields without adding unrelated arguments", () => {
    const fields = [field("string"), field("boolean", { name: "active" }), field("json", { name: "data" }), field("enum", { name: "mode" })];
    const args = { value: "", active: false, data: { tags: [1] }, mode: 3 };
    const values = schema.toFieldValues(fields, { ...args, ignored: "secret" });
    assert.deepEqual(values, { value: "", active: "false", data: '{\n  "tags": [\n    1\n  ]\n}', mode: "3" });
    assert.deepEqual(schema.buildArguments(fields, values), { active: false, data: args.data, mode: 3 });
    assert.deepEqual(schema.toFieldValues([field("number")], { value: 0 }), { value: "0" });
    assert.deepEqual(schema.toFieldValues([field("string")], {}), {});
  });

  for (const [name, parse, prefix] of [
    ["arguments", () => schema.parseArguments, "The arguments"],
    ["capabilities", () => client.parseCapabilities, "The client capabilities"],
  ]) {
    it(`parses empty and nested ${name} as JSON objects`, () => {
      assert.deepEqual(parse()("  "), {});
      assert.deepEqual(parse()('{"nested":{"enabled":true}}'), { nested: { enabled: true } });
    });
    it(`rejects invalid JSON in ${name}`, () => assert.throws(() => parse()("{"), { message: `${prefix} must be valid JSON.` }));
    for (const value of ["null", "[]", "true", "2", '"text"']) {
      it(`rejects non-object ${name}: ${value}`, () => assert.throws(() => parse()(value), { message: `${prefix} must be a JSON object.` }));
    }
  }

  it("uses server discovery only for protocol dates at or after its introduction", () => {
    for (const version of ["2025-03-26", "2026-07-27", "invalid", ""]) assert.equal(client.usesServerDiscover(version), false);
    for (const version of ["2026-07-28", "2026-10-30"]) assert.equal(client.usesServerDiscover(version), true);
  });

  it("sends credentialed JSON-RPC requests with session and target headers", async () => {
    const fetch = respond({ jsonrpc: "2.0", id: 1, result: { tools: [] } }, { headers: { "Mcp-Session-Id": "new-session" } });
    const reply = await send({ config: { ...config, sessionID: "old-session" }, target: "tool-name", params: { cursor: "next" } });
    const [url, init] = fetch.mock.calls[0].arguments;
    const request = JSON.parse(init.body);
    assert.equal(url, config.url);
    assert.equal(init.method, "POST");
    assert.equal(init.credentials, "include");
    assert.equal(init.headers["Mcp-Session-Id"], "old-session");
    assert.equal(init.headers["Mcp-Name"], "tool-name");
    assert.equal(init.headers["MCP-Protocol-Version"], config.protocolVersion);
    assert.equal(init.headers.Accept, "application/json, text/event-stream");
    assert.deepEqual(request, { jsonrpc: "2.0", id: reply.exchange.id, method: "tools/list", params: { cursor: "next" } });
    assert.deepEqual(reply.result, { tools: [] });
    assert.equal(reply.sessionID, "new-session");
    assert.equal(reply.exchange.httpStatus, 200);
    assert.ok(reply.exchange.durationMs >= 0);
  });

  it("allocates unique request IDs and leaves notification IDs out of the body", async () => {
    const fetch = respond(undefined, { status: 202 });
    const first = await send();
    const second = await send({ notification: true, method: "notifications/initialized" });
    const third = await send();
    assert.equal(new Set([first.exchange.id, second.exchange.id, third.exchange.id]).size, 3);
    assert.deepEqual(JSON.parse(fetch.mock.calls[1].arguments[1].body), { jsonrpc: "2.0", method: "notifications/initialized" });
    assert.equal(second.result, undefined);
  });

  for (const target of ["café 工具", " surrounding ", "=?base64?literal"] ) {
    it(`encodes the target header safely for ${JSON.stringify(target)}`, async () => {
      const fetch = respond({ result: {} });
      await send({ target });
      const header = fetch.mock.calls[0].arguments[1].headers["Mcp-Name"];
      assert.match(header, /^=\?base64\?.+\?=$/);
      assert.equal(Buffer.from(header.slice(9, -2), "base64").toString("utf8"), target);
    });
  }

  it("supplies discovery metadata and default client identity without mutating parameters", async () => {
    const fetch = respond({ result: {} });
    const params = { name: "search" };
    await send({ config: { ...config, protocolVersion: "2026-07-28", clientName: " ", clientVersion: " ", capabilities: '{"roots":{}}' }, params });
    assert.deepEqual(JSON.parse(fetch.mock.calls[0].arguments[1].body).params._meta, {
      "io.modelcontextprotocol/protocolVersion": "2026-07-28",
      "io.modelcontextprotocol/clientInfo": { name: client.DEFAULT_CLIENT_NAME, version: client.DEFAULT_CLIENT_VERSION },
      "io.modelcontextprotocol/clientCapabilities": { roots: {} },
    });
    assert.deepEqual(params, { name: "search" });
  });

  it("reports invalid discovery capabilities before making a request", async () => {
    const fetch = respond({ result: {} });
    const reply = await send({ config: { ...config, protocolVersion: "2026-07-28", capabilities: "[" } });
    assert.equal(fetch.mock.callCount(), 0);
    assert.match(reply.error, /capabilities must be valid JSON/);
    assert.equal(reply.exchange.error, reply.error);
  });

  it("does not require discovery capabilities for notifications", async () => {
    const fetch = respond(undefined, { status: 202 });
    const reply = await send({ config: { ...config, protocolVersion: "2026-07-28", capabilities: "[" }, notification: true });
    assert.equal(reply.error, undefined);
    assert.equal(fetch.mock.callCount(), 1);
  });

  it("reads a JSON-RPC result from a CRLF event stream", async () => {
    mock.method(globalThis, "fetch", async (url, init) => new Response(`event: message\r\ndata: ${JSON.stringify({ jsonrpc: "2.0", id: JSON.parse(init.body).id, result: { ok: true } })}\r\n\r\ndata: [DONE]\r\n\r\n`, { headers: { "content-type": "text/event-stream" } }));
    assert.deepEqual((await send()).result, { ok: true });
  });

  it("preserves plain response text and empty notification responses", async () => {
    respond("upstream status", { headers: { "content-type": "text/plain" } });
    assert.equal((await send()).result, "upstream status");
  });

  it("reports JSON-RPC errors even when HTTP succeeded", async () => {
    respond({ error: { code: -32602, message: "Bad arguments" } });
    const reply = await send();
    assert.equal(reply.error, "-32602: Bad arguments");
    assert.equal(reply.exchange.error, reply.error);
  });

  for (const sessionID of [undefined, "existing"]) {
    it(`marks 404 as session expiry only with an existing session: ${sessionID}`, async () => {
      respond({ error: { message: "Unknown session" } }, { status: 404, statusText: "Not Found" });
      const reply = await send({ config: { ...config, sessionID } });
      assert.equal(reply.expired, !!sessionID);
      assert.equal(reply.error, "404 Not Found: Unknown session");
    });
  }

  it("bounds raw HTTP error text and reports network failures", async () => {
    respond("x".repeat(500), { status: 502, statusText: "Bad Gateway" });
    assert.equal((await send()).error, `502 Bad Gateway: ${"x".repeat(400)}`);
    mock.restoreAll();
    mock.method(globalThis, "fetch", async () => { throw new Error("Offline"); });
    assert.equal((await send()).error, "Offline");
  });

  it("forwards cancellation and removes the signal listener after settling", async () => {
    const controller = new AbortController();
    const remove = mock.method(controller.signal, "removeEventListener");
    mock.method(globalThis, "fetch", async (url, init) => {
      controller.abort();
      assert.equal(init.signal.aborted, true);
      throw new DOMException("Aborted", "AbortError");
    });
    assert.equal((await send({ signal: controller.signal })).error, "The request was cancelled.");
    assert.equal(remove.mock.callCount(), 1);
  });

  it("reports the request timeout and clears its timer", async () => {
    let expire;
    mock.method(window, "setTimeout", (callback, delay) => { assert.equal(delay, 30_000); expire = callback; return 42; });
    const clear = mock.method(window, "clearTimeout", () => {});
    mock.method(globalThis, "fetch", async (url, init) => {
      expire();
      assert.equal(init.signal.aborted, true);
      throw new DOMException("Aborted", "AbortError");
    });
    assert.equal((await send()).error, "The request timed out after 30 seconds.");
    assert.deepEqual(clear.mock.calls[0].arguments, [42]);
  });

  it("CONSOLE-001: enforces numeric schema bounds", { todo: "CONSOLE-001" }, () => {
    assert.throws(() => schema.buildArguments([field("number", { minimum: 1, maximum: 5 })], { value: "9" }), /maximum|between|range/i);
  });

  it("CONSOLE-002: preserves a required empty string when its schema permits it", { todo: "CONSOLE-002" }, () => {
    assert.deepEqual(schema.buildArguments([field("string", { required: true })], { value: "" }), { value: "" });
  });

  it("CONSOLE-003: respects an already aborted request signal", { todo: "CONSOLE-003" }, async () => {
    const controller = new AbortController();
    controller.abort();
    const fetch = respond({ result: {} });
    const reply = await send({ signal: controller.signal });
    assert.ok(fetch.mock.callCount() === 0 || fetch.mock.calls[0].arguments[1].signal.aborted);
    assert.equal(reply.error, "The request was cancelled.");
  });

  it("CONSOLE-004: preserves application metadata when adding discovery fields", { todo: "CONSOLE-004" }, async () => {
    const fetch = respond({ result: {} });
    await send({ config: { ...config, protocolVersion: "2026-07-28" }, params: { _meta: { progressToken: "progress-1" } } });
    assert.equal(JSON.parse(fetch.mock.calls[0].arguments[1].body).params._meta.progressToken, "progress-1");
  });

  it("CONSOLE-005: matches an SSE result to the outstanding JSON-RPC request", { todo: "CONSOLE-005" }, async () => {
    mock.method(globalThis, "fetch", async (url, init) => {
      const request = JSON.parse(init.body);
      return new Response(`data: ${JSON.stringify({ jsonrpc: "2.0", id: request.id, result: { tools: [] } })}\n\ndata: ${JSON.stringify({ jsonrpc: "2.0", method: "notifications/tools/list_changed" })}\n\n`, { headers: { "content-type": "text/event-stream" } });
    });
    assert.deepEqual((await send()).result, { tools: [] });
  });

  it("CONSOLE-006: does not parse JSON text containing data: as an event stream", { todo: "CONSOLE-006" }, async () => {
    respond({ result: { content: [{ type: "text", text: "data: hello" }] } });
    assert.deepEqual((await send()).result, { content: [{ type: "text", text: "data: hello" }] });
  });

  it("CONSOLE-007: joins multiline data fields within one SSE event", { todo: "CONSOLE-007" }, async () => {
    mock.method(globalThis, "fetch", async (url, init) => new Response(`data: {"jsonrpc":"2.0",\ndata: "id":${JSON.parse(init.body).id},"result":{"ok":true}}\n\n`, { headers: { "content-type": "text/event-stream" } }));
    assert.deepEqual((await send()).result, { ok: true });
  });
});
