import { strict as assert } from "node:assert";
import { after, before, describe, it } from "node:test";
import { Code, ConnectError } from "@connectrpc/connect";
import { getCatalog } from "../testutil/app.ts";
import { startMockAPI, type MockAPI } from "../testutil/mockapi.ts";
import {
  errorCodeName,
  InvalidRequestError,
  OcteliumAPIError,
  OcteliumClient,
  resolveOcteliumMode,
} from "./client.ts";

describe("octelium client", () => {
  const catalog = getCatalog();
  let mock: MockAPI;
  let client: OcteliumClient;

  before(async () => {
    mock = await startMockAPI(catalog, {
      "user.v1.MainService/GetStatus": () => ({
        user: { metadata: { name: "alice" } },
      }),
      "core.v1.MainService/GetService": (req) => {
        const name = (req as { name?: string }).name;
        if (name !== "nginx.default") {
          throw new ConnectError(`Service ${name} not found`, Code.NotFound);
        }
        return {
          apiVersion: "core/v1",
          kind: "Service",
          metadata: { name, createdAt: "2026-01-02T15:04:05Z" },
          spec: { port: 8080, mode: "HTTP", attrs: { team: "web" } },
        };
      },
      "cordium.v1.MainService/WatchWorkspace": () => [
        { create: { item: { metadata: { name: "a" } } } },
        { create: { item: { metadata: { name: "b" } } } },
        { create: { item: { metadata: { name: "c" } } } },
      ],
    });
    client = new OcteliumClient(catalog, {
      mode: "proxy",
      authProxySocket: mock.socketPath,
      userAgent: "test-agent",
    });
  });

  after(async () => {
    await mock.close();
  });

  it("invokes the unary methods over the auth proxy socket", async () => {
    const status = await client.call<{ user: { metadata: { name: string } } }>(
      "user.v1.MainService/GetStatus",
    );
    assert.equal(status.user.metadata.name, "alice");
    assert.equal(mock.calls.at(-1)?.headers.get("user-agent"), "test-agent");

    const svc = await client.call<Record<string, unknown>>(
      "/octelium.api.main.core.v1.MainService/GetService",
      { name: "nginx.default" },
    );
    assert.deepEqual(svc, {
      apiVersion: "core/v1",
      kind: "Service",
      metadata: { name: "nginx.default", createdAt: "2026-01-02T15:04:05Z" },
      spec: { port: 8080, mode: "HTTP", attrs: { team: "web" } },
    });
  });

  it("maps the gRPC errors", async () => {
    await assert.rejects(
      client.call("core.v1.MainService/GetService", { name: "nope.default" }),
      (err: Error) =>
        err instanceof OcteliumAPIError &&
        err.code === "not_found" &&
        err.message === "Service nope.default not found",
    );
    await assert.rejects(
      client.call("core.v1.MainService/ListService"),
      (err: Error) =>
        err instanceof OcteliumAPIError && err.code === "unimplemented",
    );
    assert.equal(errorCodeName(Code.PermissionDenied), "permission_denied");
    assert.equal(errorCodeName(Code.DeadlineExceeded), "deadline_exceeded");
  });

  it("validates the requests before sending them", async () => {
    const before = mock.calls.length;
    await assert.rejects(
      client.call("core.v1.MainService/GetService", { nam: "x" }),
      (err: Error) =>
        err instanceof InvalidRequestError && /nam/.test(err.message),
    );
    await assert.rejects(
      client.call("core.v1.MainService/GetService", ["x"]),
      InvalidRequestError,
    );
    await assert.rejects(
      client.call("core.v1.MainService/Nope"),
      InvalidRequestError,
    );
    assert.equal(mock.calls.length, before);
  });

  it("collects the server-streaming messages", async () => {
    const method = catalog.resolveMethod(
      "cordium.v1.MainService/WatchWorkspace",
    )!;
    assert.equal(method.kind, "server_streaming");
    const res = await client.invoke(method, {}, { maxMessages: 2 });
    assert.equal(res.messageCount, 2);
    assert.deepEqual(res.response, [
      { create: { item: { metadata: { name: "a" } } } },
      { create: { item: { metadata: { name: "b" } } } },
    ]);
  });

  it("refuses the client-streaming methods and the disabled mode", async () => {
    const exec = catalog.methods.find((m) => m.kind === "bidi_streaming");
    if (exec) {
      await assert.rejects(client.invoke(exec, {}), InvalidRequestError);
    }
    const disabled = new OcteliumClient(catalog, { mode: "disabled" });
    assert.equal(disabled.enabled, false);
    await assert.rejects(
      disabled.call("user.v1.MainService/GetStatus"),
      (err: Error) =>
        err instanceof OcteliumAPIError && err.code === "unavailable",
    );
  });

  it("resolves the connection mode", () => {
    const base = { insecureTLS: false, timeoutSeconds: 30 };
    assert.deepEqual(
      resolveOcteliumMode({
        ...base,
        mode: "auto",
        authProxySocket: mock.socketPath,
      }),
      { mode: "proxy" },
    );
    assert.deepEqual(
      resolveOcteliumMode({
        ...base,
        mode: "auto",
        authProxySocket: "/nonexistent.sock",
        domain: "example.com",
        accessToken: "tok",
      }),
      { mode: "direct" },
    );
    assert.equal(
      resolveOcteliumMode({ ...base, mode: "auto" }).mode,
      "disabled",
    );
    assert.equal(
      resolveOcteliumMode({
        ...base,
        mode: "proxy",
        authProxySocket: "/nonexistent.sock",
      }).mode,
      "disabled",
    );
    assert.equal(
      resolveOcteliumMode({ ...base, mode: "direct", domain: "x" }).mode,
      "disabled",
    );
    assert.equal(
      resolveOcteliumMode({
        ...base,
        mode: "disabled",
        authProxySocket: mock.socketPath,
      }).mode,
      "disabled",
    );
  });
});
