import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { ESLint } from "eslint";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console lint configuration", () => {
  const eslint = new ESLint({ cwd: root });
  const lint = async (code, filePath = "src/LintFixture.tsx") =>
    (await eslint.lintText(code, { filePath }))[0].messages;

  it("accepts TypeScript, hooks and browser globals", async () => {
    assert.deepEqual(
      await lint(`
        import { useState } from "react";
        export default function Counter() {
          const [count, setCount] = useState(0);
          return <button onClick={() => setCount(count + 1)}>{window.location.pathname}:{count}</button>;
        }
      `),
      [],
    );
  });

  it("accepts Node globals in tooling and browser callbacks in Playwright tests", async () => {
    assert.deepEqual(
      await lint(
        "export const environment = process.env.NODE_ENV;",
        "fixture.config.js",
      ),
      [],
    );
    assert.deepEqual(
      await lint(
        "export const evaluate = (page) => page.evaluate(() => window.location.pathname);",
        "tests/lintFixture.test.mjs",
      ),
      [],
    );
  });

  it("reports conditional hooks", async () => {
    const messages = await lint(`
      import { useState } from "react";
      export default function Counter({ enabled }: { enabled: boolean }) {
        if (enabled) useState(0);
        return null;
      }
    `);
    assert.ok(
      messages.some(
        (message) => message.ruleId === "react-hooks/rules-of-hooks",
      ),
    );
  });

  it("reports query dependencies missing from the cache key", async () => {
    const messages = await lint(`
      import { useQuery } from "@tanstack/react-query";
      export default function User({ id }: { id: string }) {
        useQuery({ queryKey: ["user"], queryFn: () => fetch("/users/" + id) });
        return null;
      }
    `);
    assert.ok(
      messages.some(
        (message) => message.ruleId === "@tanstack/query/exhaustive-deps",
      ),
    );
    assert.deepEqual(
      await lint(`
        import { useQuery } from "@tanstack/react-query";
        export default function User({ id }: { id: string }) {
          useQuery({ queryKey: ["user", id], queryFn: () => fetch("/users/" + id) });
          return null;
        }
      `),
      [],
    );
  });

  it("ignores generated protobuf code and build outputs", async () => {
    for (const file of [
      "src/apis/corev1/corev1.ts",
      "dist/index.js",
      "tmp/generated.ts",
    ])
      assert.equal(await eslint.isPathIgnored(join(root, file)), true);
    assert.equal(
      await eslint.isPathIgnored(join(root, "src/pages/index.tsx")),
      false,
    );
  });
});

describe("Console code health regressions", () => {
  let cacheDir;
  let server;
  let browser;
  let context;
  let page;
  let origin;
  let errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-code-health-"));
    server = await createServer({
      root,
      cacheDir,
      server: { host: "127.0.0.1", port: 0, watch: null },
    });
    await server.listen();
    origin = server.resolvedUrls.local[0];
    browser = await chromium.launch();
  });

  after(async () => {
    await browser?.close();
    await server?.close();
    if (cacheDir) await rm(cacheDir, { recursive: true, force: true });
  });

  beforeEach(async () => {
    context = await browser.newContext();
    page = await context.newPage();
    page.setDefaultTimeout(30_000);
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (kind = "access", resource = "user") => {
    await page.goto(
      `${origin}tests/fixtures/consoleCodeHealth.html?${new URLSearchParams({ kind, resource, periodMinutes: "180" })}`,
    );
    await page.locator("output").filter({ hasText: /^\{/ }).waitFor();
  };

  for (const [kind, resource, method] of [
    ["access", "user", "listAccessLogTopUser"],
    ["access", "service", "listAccessLogTopService"],
    ["access", "policy", "listAccessLogTopPolicy"],
    ["access", "session", "listAccessLogTopSession"],
    ["auth", "user", "listAuthenticationLogTopUser"],
    ["auth", "identityProvider", "listAuthenticationLogTopIdentityProvider"],
    ["auth", "credential", "listAuthenticationLogTopCredential"],
    ["audit", "user", "listAuditLogTopUser"],
    ["audit", "session", "listAuditLogTopSession"],
  ]) {
    it(`sends a valid protobuf request for ${method} and preserves the response`, async () => {
      await load(kind, resource);
      const calls = await page.evaluate(() => window.codeHealth.calls);
      assert.equal(calls.length, 1);
      const call = calls[0];
      assert.equal(call.name, method);
      assert.equal(
        call.type.split(".").at(-1),
        `${method[0].toUpperCase()}${method.slice(1)}Request`,
      );
      assert.equal(call.valid, true);
      assert.equal(call.abort, true);
      assert.equal(call.request.limit, 10);
      assert.equal(
        Number(call.request.to.seconds) - Number(call.request.from.seconds),
        180 * 60,
      );
      assert.equal(call.request.to.nanos, call.request.from.nanos);
      if (kind === "access") {
        assert.equal(call.request.status, 0);
        assert.equal(call.request.isPublic, false);
        assert.equal(call.request.isAnonymous, false);
      }
      assert.deepEqual(JSON.parse(await page.locator("output").textContent()), {
        items: [{ count: 3 }],
        totalCount: 4,
        totalOther: 2,
      });
    });
  }

  it("preserves Policy Tester reason labels and fallback behavior", async () => {
    await load();
    const labels = {
      POLICY_MATCH: "Policy Match",
      NO_POLICY_MATCH: "No Policy Match",
      USER_DEACTIVATED: "User Deactivated",
      SESSION_NOT_ACTIVE: "Session Not Active",
      SESSION_EXPIRED: "Session Expired",
      ACCESS_TOKEN_EXPIRED: "Access Token Expired",
      AUTHENTICATOR_AUTHENTICATION_REQUIRED:
        "Authenticator Authentication Required",
      AUTHENTICATOR_REGISTRATION_REQUIRED:
        "Authenticator Registration Required",
      SCOPE_UNAUTHORIZED: "Unauthorized Scope",
      DEVICE_NOT_ACTIVE: "Device not Active",
      SESSION_CLIENT_TYPE_INVALID: "Invalid Session type",
      DEVICE_LOCKED: "Locked Device",
      SESSION_LOCKED: "Locked Session",
      USER_LOCKED: "Locked User",
    };
    assert.deepEqual(
      await page.evaluate((labels) => {
        const { getPolicyReason, reasons } = window.codeHealth;
        return Object.fromEntries(
          Object.keys(labels).map((name) => [
            name,
            getPolicyReason(reasons[name]),
          ]),
        );
      }, labels),
      labels,
    );
    assert.deepEqual(
      await page.evaluate(() => {
        const { getPolicyReason, reasons } = window.codeHealth;
        return [
          getPolicyReason(),
          getPolicyReason(reasons.TYPE_UNKNOWN_REASON),
          getPolicyReason(9999),
        ];
      }),
      ["", "", undefined],
    );
  });
});
