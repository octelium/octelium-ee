import assert from "node:assert/strict";
import { after, before, describe, it } from "node:test";
import { randomBytes } from "node:crypto";
import { gunzipSync } from "node:zlib";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("Support diagnostics and navigation helpers", () => {
  let modules;
  let diagnostics;
  let context;
  let catalog;

  const options = { keepEmails: false };
  const selection = {
    from: "2026-11-22T09:00:00.000Z",
    to: "2026-11-22T10:00:00.000Z",
    levels: ["ERROR", "PANIC", "FATAL"],
    maxEntries: 1000,
    keepEmails: false,
  };
  const entry = (minute, message, fields) => ({
    time: `2026-11-22T09:${String(minute).padStart(2, "0")}:00.000Z`,
    level: "ERROR",
    component: { namespace: "octelium", type: "gateway", uid: "gw-1" },
    message,
    ...(fields ? { fields } : {}),
  });

  before(async () => {
    modules = await createConsoleModules();
    [diagnostics, context, catalog] = await Promise.all([
      modules.load("pages/support/diagnostics.ts"),
      modules.load("components/Support/context.ts"),
      modules.load("components/Navigation/catalog.ts"),
    ]);
  });
  after(async () => {
    await modules?.close();
  });

  it("recognizes secret field names without hiding counters and identifiers", () => {
    for (const key of [
      "password",
      "Password",
      "db_passwd",
      "clientSecret",
      "client-secret",
      "accessToken",
      "refresh_token",
      "token",
      "Authorization",
      "cookie",
      "apiKey",
      "api-key",
      "private_key",
      "key",
      "jwt",
      "credentials",
      "signature",
    ]) {
      assert.equal(diagnostics.isSecretKey(key), true, key);
    }
    for (const key of [
      "promptTokens",
      "tokenID",
      "keyID",
      "user",
      "service",
      "status",
      "monkey",
      "durationMs",
    ]) {
      assert.equal(diagnostics.isSecretKey(key), false, key);
    }
  });

  it("removes secrets from free text and counts them per rule", () => {
    const stats = diagnostics.newRedactionStats();
    const text = [
      "Authorization: Bearer abcdefghijklmnop0123",
      "dial postgres://admin:s3cr3t@db.internal:5432/octelium",
      "GET https://api.acme.example/v1/items?api_key=sk_live_123&debug=1 failed",
      "token=abc123def password: 'hunter2' secret=\"x y\"",
      "license eyJhbGciOiJFZERTQSJ9.eyJzdWIiOiIxMjM0In0.c2lnbmF0dXJl rejected",
      "owner alice@acme.example",
      "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEI\n-----END EC PRIVATE KEY-----",
    ].join("\n");
    const ret = diagnostics.redactText(text, options, stats);
    for (const secret of [
      "abcdefghijklmnop0123",
      "s3cr3t",
      "sk_live_123",
      "abc123def",
      "hunter2",
      "x y",
      "eyJzdWIiOiIxMjM0In0",
      "alice@acme.example",
      "MHcCAQEEI",
    ]) {
      assert.ok(!ret.includes(secret), secret);
    }
    assert.match(ret, /Authorization: Bearer \[REDACTED\]/);
    assert.match(ret, /postgres:\/\/\[REDACTED\]@db\.internal:5432\/octelium/);
    assert.match(ret, /https:\/\/api\.acme\.example\/v1\/items\?\[REDACTED\] failed/);
    assert.match(ret, /token=\[REDACTED\] password: \[REDACTED\] secret=\[REDACTED\]/);
    assert.deepEqual(stats, {
      field: 0,
      privateKey: 1,
      jwt: 1,
      authorization: 1,
      urlCredentials: 1,
      urlQuery: 1,
      assignment: 3,
      email: 1,
    });

    const kept = diagnostics.newRedactionStats();
    assert.equal(
      diagnostics.redactText("owner alice@acme.example", { keepEmails: true }, kept),
      "owner alice@acme.example",
    );
    assert.equal(diagnostics.countRedactions(kept), 0);
  });

  it("redacts nested fields recursively", () => {
    const stats = diagnostics.newRedactionStats();
    const ret = diagnostics.redactValue(
      {
        user: "alice",
        request: {
          headers: { Authorization: "Bearer abc", Accept: "application/json" },
          url: "https://acme.example/callback?code=xyz",
        },
        attempts: 3,
        secrets: ["a", "b"],
        empty: { token: "" },
        list: [{ password: "p" }, "contact bob@acme.example"],
      },
      options,
      stats,
    );
    assert.deepEqual(ret, {
      user: "alice",
      request: {
        headers: { Authorization: "[REDACTED]", Accept: "application/json" },
        url: "https://acme.example/callback?[REDACTED]",
      },
      attempts: 3,
      secrets: "[REDACTED]",
      empty: { token: "" },
      list: [{ password: "[REDACTED]" }, "contact [REDACTED EMAIL]"],
    });
    assert.equal(stats.field, 3);
  });

  it("builds a compressed NDJSON bundle with a manifest, oldest entries first", async () => {
    const bundle = await diagnostics.buildDiagnostics({
      entries: [
        entry(30, "second", { password: "hunter2" }),
        entry(10, "first"),
        entry(50, "third alice@acme.example"),
      ],
      selection: { ...selection, component: { type: "gateway" } },
      matched: 3,
      maxBytes: 1 << 20,
      clusterDomain: "cluster.acme.example",
      now: new Date("2026-11-22T10:01:00.000Z"),
    });
    assert.equal(bundle.compressed, true);
    assert.equal(bundle.name, "component-logs-20261122T0900Z-20261122T1000Z.ndjson.gz");
    const lines = gunzipSync(Buffer.from(bundle.data))
      .toString("utf8")
      .trim()
      .split("\n")
      .map((line) => JSON.parse(line));
    assert.deepEqual(lines[0], {
      type: "manifest",
      version: 1,
      generatedAt: "2026-11-22T10:01:00.000Z",
      clusterDomain: "cluster.acme.example",
      window: { from: selection.from, to: selection.to },
      levels: selection.levels,
      component: { type: "gateway" },
      requested: 1000,
      matched: 3,
      included: 3,
      truncated: false,
      redaction: {
        version: 1,
        keepEmails: false,
        counts: {
          field: 1,
          privateKey: 0,
          jwt: 0,
          authorization: 0,
          urlCredentials: 0,
          urlQuery: 0,
          assignment: 0,
          email: 1,
        },
      },
    });
    assert.deepEqual(
      lines.slice(1).map((line) => [line.type, line.message]),
      [
        ["log", "first"],
        ["log", "second"],
        ["log", "third [REDACTED EMAIL]"],
      ],
    );
    assert.equal(lines[2].fields.password, "[REDACTED]");
  });

  it("keeps the most recent entries within the entry and size limits", async () => {
    const entries = Array.from({ length: 60 }, (_, index) =>
      entry(index, `${index} ${randomBytes(200).toString("hex")}`),
    );
    const limited = await diagnostics.buildDiagnostics({
      entries,
      selection: { ...selection, maxEntries: 10 },
      matched: 600,
      maxBytes: 1 << 20,
    });
    assert.equal(limited.manifest.included, 10);
    assert.equal(limited.manifest.matched, 600);
    assert.equal(limited.manifest.truncated, true);
    assert.deepEqual(
      limited.entries.map((item) => item.time),
      entries.slice(50).map((item) => item.time),
    );

    const small = await diagnostics.buildDiagnostics({
      entries,
      selection,
      matched: 60,
      maxBytes: 4096,
    });
    assert.ok(small.data.byteLength <= 4096);
    assert.ok(small.manifest.included > 0 && small.manifest.included < 60);
    assert.equal(small.manifest.truncated, true);
    assert.equal(small.entries.at(-1).time, entries.at(-1).time);
  });

  it("computes the collection windows", () => {
    const now = new Date("2026-11-22T10:00:00.000Z");
    assert.deepEqual(diagnostics.getWindow("1h", undefined, now), {
      from: "2026-11-22T09:00:00.000Z",
      to: "2026-11-22T10:00:00.000Z",
    });
    assert.deepEqual(diagnostics.getWindow("15m", undefined, now).from, "2026-11-22T09:45:00.000Z");
    assert.deepEqual(diagnostics.getWindow("around", "2026-11-22T09:00:00.000Z", now), {
      from: "2026-11-22T08:45:00.000Z",
      to: "2026-11-22T09:15:00.000Z",
    });
    assert.deepEqual(diagnostics.getWindow("around", "2026-11-22T09:55:00.000Z", now), {
      from: "2026-11-22T09:40:00.000Z",
      to: "2026-11-22T10:00:00.000Z",
    });
    assert.deepEqual(diagnostics.getWindow("around", "not-a-date", now).from, "2026-11-22T09:00:00.000Z");
  });

  it("links component log entries to a prefilled request and names the Support pages", () => {
    const path = context.getSupportReportPath({
      subject: "gateway: panic",
      message: "x".repeat(3000),
      time: "2026-11-22T09:00:00.000Z",
      level: "PANIC",
      component: { namespace: "octelium", type: "gateway", uid: "gw-1" },
    });
    const url = new URL(path, "https://console.example");
    assert.equal(url.pathname, "/support/new");
    assert.equal(url.searchParams.get("subject"), "gateway: panic");
    assert.equal(url.searchParams.get("message").length, 2000);
    assert.equal(url.searchParams.get("component.uid"), "gw-1");
    assert.equal(context.getSupportReportPath({}), "/support/new");

    const labels = (pathname) => catalog.resolveTrail(pathname).map((crumb) => crumb.label);
    assert.deepEqual(labels("/support"), ["Overview", "Support"]);
    assert.deepEqual(labels("/support/new"), ["Overview", "Support", "New request"]);
    assert.deepEqual(labels("/support/requests/0b1c2d3e"), ["Overview", "Support", "Request"]);
  });
});
