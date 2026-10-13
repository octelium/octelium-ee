import assert from "node:assert/strict";
import { mkdtemp, readFile, readdir, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import ts from "typescript";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console network regressions", () => {
  let cacheDir;
  let server;
  let browser;
  let context;
  let page;
  let origin;
  let errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-network-"));
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
    context = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    page = await context.newPage();
    page.setDefaultTimeout(30_000);
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (scenario = "utilities", extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/consoleNetwork.html?${new URLSearchParams({ scenario, ...extra })}`,
    );
    await page.waitForFunction(() => window.network);
  };
  const calls = () => page.evaluate(() => window.network.calls);
  const settled = () =>
    page.waitForFunction(() => window.network.queryClient.isFetching() === 0);

  for (const code of [
    "PERMISSION_DENIED",
    "INVALID_ARGUMENT",
    "NOT_FOUND",
    "UNIMPLEMENTED",
    "CANCELLED",
  ]) {
    it(`does not retry or reload queries failing with ${code}`, async () => {
      await load("query", { code });
      await page
        .locator("output")
        .filter({ hasText: /^error$/ })
        .waitFor();
      await page.clock.install();
      await page.clock.runFor(15_000);
      assert.deepEqual(await calls(), [{ name: "query" }]);
    });
  }

  it("bounds transient query retries to two", async () => {
    await load("query", { code: "UNAVAILABLE" });
    await page
      .locator("output")
      .filter({ hasText: /^error$/ })
      .waitFor();
    assert.deepEqual(
      await calls(),
      Array.from({ length: 3 }, () => ({ name: "query" })),
    );
  });

  for (const [code, count] of [
    ["PERMISSION_DENIED", 1],
    ["UNAVAILABLE", 2],
  ]) {
    it(`applies the permanent-error policy to dashboard retries for ${code}`, async () => {
      await load("dashboard-retry", { code });
      await page
        .locator("output")
        .filter({ hasText: /^error:/ })
        .waitFor();
      assert.equal((await calls()).length, count);
    });
  }

  it("normalizes RPC codes, skips aborts and caps jittered back-off", async () => {
    await load();
    const result = await page.evaluate(() => {
      const { retryQuery, queryRetryDelay, rpcErrorCode } = window.network;
      const random = Math.random;
      const delays = [];
      try {
        for (const value of [0, 0.5, 1]) {
          Math.random = () => value;
          delays.push([
            queryRetryDelay(0),
            queryRetryDelay(1),
            queryRetryDelay(20),
          ]);
        }
      } finally {
        Math.random = random;
      }
      return {
        normalized: rpcErrorCode({
          message: "rpc error: code = PermissionDenied desc = blocked",
        }),
        wrapped: retryQuery(0, {
          message: "rpc error: code = Unimplemented desc = unavailable",
        }),
        abort: retryQuery(0, new DOMException("Cancelled", "AbortError")),
        exhausted: retryQuery(0, { code: "RESOURCE_EXHAUSTED" }),
        transient: retryQuery(1, { code: "DEADLINE_EXCEEDED" }),
        capped: retryQuery(2, new Error("Connection failed")),
        delays,
      };
    });
    assert.deepEqual(result, {
      normalized: "PERMISSION_DENIED",
      wrapped: false,
      abort: false,
      exhausted: false,
      transient: true,
      capped: false,
      delays: [
        [500, 1000, 15_000],
        [750, 1500, 22_500],
        [1000, 2000, 30_000],
      ],
    });
  });

  for (const source of ["query", "mutation"]) {
    it(`reloads the same URL after an UNAUTHENTICATED ${source} without retrying`, async () => {
      await load("utilities", { returnTo: "keep-this-route" });
      const originalURL = page.url();
      let requests = 0;
      let reloads = 0;
      await context.route("**/GetStatus", async (route) => {
        requests++;
        await route.fulfill({
          contentType: "application/grpc-web-text",
          headers: { "grpc-status": "16", "grpc-message": "Expired" },
          body: "",
        });
      });
      await context.route(originalURL, async (route) => {
        reloads++;
        await route.fulfill({
          contentType: "text/html",
          body: "<html><body>Reloaded for authentication</body></html>",
        });
      });
      await page.evaluate((source) => {
        const { queryClient, clients } = window.network;
        const run = async () =>
          (await clients.getClientUser().getStatus({})).response;
        if (source === "query") {
          void queryClient
            .fetchQuery({ queryKey: ["expired"], queryFn: run })
            .catch(() => {});
        } else {
          const mutation = queryClient
            .getMutationCache()
            .build(queryClient, { mutationFn: run, onError: () => {} });
          void mutation.execute({}).catch(() => {});
        }
      }, source);
      await page
        .getByText("Reloaded for authentication", { exact: true })
        .waitFor();
      assert.equal(page.url(), originalURL);
      assert.equal(requests, 1);
      assert.equal(reloads, 1);
    });
  }

  it("requests one reload for concurrent authentication failures", async () => {
    await load();
    let reloads = 0;
    await context.route(page.url(), async (route) => {
      reloads++;
      await route.fulfill({
        contentType: "text/html",
        body: "<html><body>Reloaded once</body></html>",
      });
    });
    await page.evaluate(() => {
      for (let index = 0; index < 10; index++)
        window.network.onQueryError({ code: "UNAUTHENTICATED" });
    });
    await page.getByText("Reloaded once", { exact: true }).waitFor();
    assert.equal(reloads, 1);
  });

  it("refreshes stale non-polled reads on focus without bursting dashboard polling", async () => {
    await load("focus");
    await settled();
    await page.clock.install();
    await page.clock.runFor(31_000);
    await page.evaluate(() => window.network.setFocused(false));
    await page.evaluate(() => window.network.setFocused(true));
    await page.waitForFunction(
      () =>
        window.network.calls.filter((call) => call.name === "static").length ===
        2,
    );
    assert.equal(
      (await calls()).filter((call) => call.name === "dashboard").length,
      1,
    );
  });

  it("refreshes the rendered dashboard and leaves unrelated caches intact", async () => {
    await load("dashboard");
    await page.waitForFunction(() => window.network.calls.length >= 4);
    await settled();
    await page.getByRole("button", { name: "Refresh the page" }).click();
    await page.waitForFunction(
      () =>
        window.network.calls.filter((call) => call.name === "dashboard")
          .length === 2,
    );
    await settled();
    const result = await page.evaluate(() => ({
      calls: window.network.calls,
      old: window.network.queryClient.getQueryState(["old-dashboard"])
        .isInvalidated,
      hidden: window.network.queryClient.getQueryState(["hidden-dashboard"])
        .isInvalidated,
      status: window.network.queryClient.getQueryState(["user", "status"])
        .isInvalidated,
      other: window.network.queryClient.getQueryState(["other"]).isInvalidated,
    }));
    for (const name of ["dashboard", "clusterInfo"])
      assert.equal(result.calls.filter((call) => call.name === name).length, 2);
    for (const name of ["status", "other"])
      assert.equal(result.calls.filter((call) => call.name === name).length, 1);
    assert.equal(result.old, false);
    assert.equal(result.hidden, true);
    assert.equal(result.status, false);
    assert.equal(result.other, false);
    await page.getByRole("button", { name: "Show panel" }).click();
    await page.waitForFunction(() =>
      window.network.calls.some((call) => call.name === "hidden-dashboard"),
    );
  });

  it("refreshes only Cluster Management queries", async () => {
    await load("cluster");
    await page.waitForFunction(() => window.network.calls.length >= 5);
    await settled();
    await page.getByRole("button", { name: "Refresh the page" }).click();
    await page.waitForFunction(
      () =>
        window.network.calls.filter((call) => call.name === "license")
          .length === 2,
    );
    await settled();
    const result = await calls();
    for (const name of ["clusterInfo", "clusterConfig", "license"])
      assert.equal(result.filter((call) => call.name === name).length, 2);
    for (const name of ["status", "other"])
      assert.equal(result.filter((call) => call.name === name).length, 1);
  });

  it("debounces period changes and preserves unrelated URL parameters", async () => {
    await load("range");
    await page.locator("[data-period]").filter({ hasText: "60" }).waitFor();
    await page.clock.install();
    await page.evaluate(() => window.network.setPeriod(360));
    await page.clock.runFor(100);
    await page.evaluate(() => window.network.setPeriod(1440));
    await page.clock.runFor(249);
    assert.deepEqual(await calls(), [{ name: "range:60" }]);
    await page.clock.runFor(1);
    await page.locator("[data-period]").filter({ hasText: "1440" }).waitFor();
    assert.equal(
      await page.locator("[data-route]").textContent(),
      "/dashboard?keep=yes&range=1440",
    );
    assert.deepEqual(await calls(), [
      { name: "range:60" },
      { name: "range:1440" },
    ]);
    await page.evaluate(() => window.network.setPeriod(60));
    await page.clock.runFor(250);
    await page
      .locator("[data-route]")
      .filter({ hasText: /^\/dashboard\?keep=yes$/ })
      .waitFor();
  });

  it("cancels pending period changes when navigation supersedes them", async () => {
    await load("range");
    await page.locator("[data-period]").filter({ hasText: "60" }).waitFor();
    await page.clock.install();
    await page.evaluate(() => window.network.setPeriod(1440));
    await page.evaluate(() =>
      window.network.navigate("/other?range=360&keep=new"),
    );
    await page.locator("[data-period]").filter({ hasText: "360" }).waitFor();
    await page.clock.runFor(1000);
    assert.equal(
      await page.locator("[data-route]").textContent(),
      "/other?range=360&keep=new",
    );
    assert.ok(!(await calls()).some((call) => call.name === "range:1440"));
  });

  it("cancels pending period changes on unmount", async () => {
    await load("range");
    await page.locator("[data-period]").filter({ hasText: "60" }).waitFor();
    await page.clock.install();
    await page.evaluate(() => window.network.setPeriod(1440));
    await page.getByRole("button", { name: "Unmount range" }).click();
    await page.clock.runFor(1000);
    assert.deepEqual(await calls(), [{ name: "range:60" }]);
  });

  it("aborts superseded picker searches while retaining the last loaded options", async () => {
    await load("picker");
    await page
      .locator("output")
      .filter({ hasText: /^alice$/ })
      .waitFor();
    await page.clock.install();
    await page.getByRole("textbox", { name: "Search picker" }).fill("first");
    await page.clock.runFor(220);
    await page.waitForFunction(() =>
      window.network.calls.some((call) => call.name === "picker:first"),
    );
    assert.equal(await page.locator("output").textContent(), "alice");
    await page.getByRole("textbox", { name: "Search picker" }).fill("second");
    await page.clock.runFor(220);
    await page.waitForFunction(() =>
      window.network.calls.some((call) => call.name === "picker:second"),
    );
    assert.equal(await page.locator("output").textContent(), "alice");
    assert.equal(
      (await calls()).find((call) => call.name === "picker:first").aborted,
      true,
    );
    await page.evaluate(() => window.network.resolvePicker("second"));
    await page
      .locator("output")
      .filter({ hasText: /^second$/ })
      .waitFor();
    assert.equal(
      (await calls()).find((call) => call.name === "picker:second").aborted,
      false,
    );
  });

  it("reuses every client and the transport without sharing call cancellation or overrides", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { clients } = window.network;
      const factories = Object.entries(clients).filter(([name]) =>
        name.startsWith("getClient"),
      );
      const identities = factories.map(([name, factory]) => [
        name,
        factory() === factory(),
      ]);
      const transport = clients.getTransport();
      const controller = new AbortController();
      const pending = new Map();
      const requests = [];
      const fetch = (url, init) =>
        new Promise((resolve, reject) => {
          const name = init.headers.get("x-test-call");
          requests.push({ name, credentials: init.credentials });
          init.signal?.addEventListener(
            "abort",
            () => reject(new DOMException("Cancelled", "AbortError")),
            { once: true },
          );
          pending.set(name, resolve);
        });
      const client = clients.getClientUser();
      const first = client
        .getStatus(
          {},
          {
            fetch,
            abort: controller.signal,
            meta: { "x-test-call": "first" },
            timeout: 123,
          },
        )
        .then(
          () => "OK",
          (error) => error.code,
        );
      const second = client
        .getStatus({}, { fetch, meta: { "x-test-call": "second" } })
        .then(
          () => "OK",
          (error) => error.code,
        );
      controller.abort();
      const trailer = new TextEncoder().encode("grpc-status: 0\r\n");
      pending.get("second")(
        new Response(
          new Uint8Array([
            0,
            0,
            0,
            0,
            0,
            128,
            0,
            0,
            0,
            trailer.length,
            ...trailer,
          ]),
          { headers: { "content-type": "application/grpc-web+proto" } },
        ),
      );
      const outcomes = await Promise.all([first, second]);
      const defaults = transport.mergeOptions({});
      return {
        identities,
        sameTransport: transport === clients.getTransport(),
        outcomes,
        requests,
        timeout: defaults.timeout ?? null,
        abort: defaults.abort ?? null,
        meta: defaults.meta ?? null,
      };
    });
    assert.equal(result.identities.length, 20);
    assert.ok(result.identities.every(([, same]) => same));
    assert.equal(result.sameTransport, true);
    assert.deepEqual(result.outcomes, ["CANCELLED", "OK"]);
    assert.deepEqual(result.requests, [
      { name: "first", credentials: "include" },
      { name: "second", credentials: "include" },
    ]);
    assert.equal(result.timeout, null);
    assert.equal(result.abort, null);
    assert.equal(result.meta, null);
  });

  it("disables focus refetches for every polled query", async () => {
    const files = await readdir(join(root, "src"), { recursive: true });
    let polled = 0;
    for (const file of files.filter(
      (file) => /\.tsx?$/.test(file) && !file.startsWith("apis/"),
    )) {
      const source = ts.createSourceFile(
        file,
        await readFile(join(root, "src", file), "utf8"),
        ts.ScriptTarget.Latest,
        true,
        file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS,
      );
      const visit = (node) => {
        if (ts.isObjectLiteralExpression(node)) {
          const props = new Map(
            node.properties
              .filter(ts.isPropertyAssignment)
              .map((prop) => [prop.name.getText(source), prop.initializer]),
          );
          if (props.has("refetchInterval")) {
            polled++;
            assert.equal(
              props.get("refetchOnWindowFocus")?.kind,
              ts.SyntaxKind.FalseKeyword,
              file,
            );
          }
        }
        ts.forEachChild(node, visit);
      };
      visit(source);
    }
    assert.ok(polled > 40);
  });
});
