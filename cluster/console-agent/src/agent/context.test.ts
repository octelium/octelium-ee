import { strict as assert } from "node:assert";
import { describe, it } from "node:test";
import { HTTPError } from "../errors.ts";
import {
  CONTEXT_LIMITS,
  contextSummary,
  formatContext,
  sanitizeContext,
} from "./context.ts";

const rejects = (raw: unknown, message: RegExp) =>
  assert.throws(
    () => sanitizeContext(raw),
    (err: unknown) =>
      err instanceof HTTPError &&
      err.status === 400 &&
      message.test(err.message),
  );

describe("console context", () => {
  it("keeps the known fields and drops the empty and unknown ones", () => {
    assert.equal(sanitizeContext(undefined), undefined);
    assert.equal(sanitizeContext({}), undefined);
    assert.equal(sanitizeContext({ resources: [], items: [] }), undefined);

    assert.deepEqual(
      sanitizeContext({
        page: {
          path: "/core/services?common.query=nginx",
          title: "  Core › Services ",
          details: { "Time range": "last 1h", Empty: "" },
          extra: true,
        },
        resources: [
          {
            apiVersion: "core/v1",
            kind: "Service",
            name: "nginx.default",
            metadata: {},
          },
          { apiVersion: "core/v1", kind: "Session", uid: "s-1" },
        ],
        items: [
          {
            kind: "AccessLog",
            id: "log-1",
            title: "alice → nginx.default",
            data: { status: "DENIED" },
            other: 1,
          },
        ],
        other: "ignored",
      }),
      {
        page: {
          path: "/core/services?common.query=nginx",
          title: "Core › Services",
          details: { "Time range": "last 1h" },
        },
        resources: [
          { apiVersion: "core/v1", kind: "Service", name: "nginx.default" },
          { apiVersion: "core/v1", kind: "Session", uid: "s-1" },
        ],
        items: [
          {
            kind: "AccessLog",
            id: "log-1",
            title: "alice → nginx.default",
            data: { status: "DENIED" },
          },
        ],
      },
    );
  });

  it("rejects invalid contexts", () => {
    rejects("page", /"context" field must be an object/);
    rejects({ page: { title: "x" } }, /"page\.path" is required/);
    rejects({ page: { path: "https://example.com" } }, /console path/);
    rejects({ page: { path: "//example.com" } }, /console path/);
    rejects(
      { page: { path: "/", title: 1 } },
      /"page\.title" must be a string/,
    );
    rejects({ page: { path: "/", details: [] } }, /"page\.details"/);
    rejects({ resources: {} }, /"resources" must be an array/);
    rejects(
      { resources: [{ apiVersion: "core/v1", kind: "User" }] },
      /requires a name or a uid/,
    );
    rejects({ resources: [{ kind: "User", name: "a" }] }, /apiVersion/);
    rejects({ items: [{ id: "x" }] }, /"items\[0\]\.kind" is required/);
    rejects(
      {
        items: Array.from({ length: CONTEXT_LIMITS.items + 1 }, () => ({
          kind: "A",
        })),
      },
      /exceeds 10 entries/,
    );
    rejects(
      { page: { path: `/${"a".repeat(CONTEXT_LIMITS.path)}` } },
      /exceeds 2048 characters/,
    );
  });

  it("truncates large item data", () => {
    const ctx = sanitizeContext({
      items: [{ kind: "AuditLog", data: { text: "x".repeat(20000) } }],
    });
    const data = ctx?.items?.[0].data;
    assert.equal(typeof data, "string");
    assert.ok((data as string).length < CONTEXT_LIMITS.itemData + 20);
    assert.match(data as string, /… \(truncated\)$/);
  });

  it("formats the context for the model", () => {
    assert.equal(
      formatContext({
        page: {
          path: "/visibility/accesslogs?status=denied",
          title: "Visibility › Access logs",
          details: { "Time range": "last 6h" },
        },
        resources: [
          { apiVersion: "core/v1", kind: "Service", name: "nginx.default" },
          { apiVersion: "core/v1", kind: "Session", uid: "s-1" },
        ],
        items: [
          { kind: "AccessLog", id: "l1", title: "alice", data: { a: 1 } },
          { kind: "AuditLog", data: "raw" },
        ],
      }),
      [
        "<console_context>",
        "Page: Visibility › Access logs (/visibility/accesslogs?status=denied)",
        "- Time range: last 6h",
        "Resources:",
        '- core/v1 Service "nginx.default" octelium://resource/core/v1/Service/nginx.default',
        "- core/v1 Session (UID: s-1)",
        "Items:",
        '- AccessLog "alice" (ID: l1)',
        '  {"a":1}',
        "- AuditLog",
        "  raw",
        "</console_context>",
      ].join("\n"),
    );
    assert.equal(
      formatContext({ page: { path: "/" } }),
      "<console_context>\nPage: /\n</console_context>",
    );
  });

  it("summarizes the context", () => {
    assert.equal(
      contextSummary({
        page: { path: "/core/users/alice", title: "Users › alice" },
        resources: [{ apiVersion: "core/v1", kind: "User", name: "alice" }],
      }),
      "User alice",
    );
    assert.equal(
      contextSummary({ page: { path: "/core/users", title: "Core › Users" } }),
      "Core › Users",
    );
    assert.equal(
      contextSummary({
        items: [
          { kind: "AccessLog", title: "a" },
          { kind: "AccessLog" },
          { kind: "AuditLog" },
          { kind: "ComponentLog" },
        ],
      }),
      "AccessLog a, AccessLog, AuditLog",
    );
  });
});
