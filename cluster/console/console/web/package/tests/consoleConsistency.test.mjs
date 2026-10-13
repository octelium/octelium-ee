import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console time-range consistency", () => {
  let cacheDir;
  let server;
  let browser;
  let context;
  let page;
  let origin;
  let errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-consistency-"));
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
      timezoneId: "UTC",
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

  const load = async (scenario = "period", extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/consoleConsistency.html?${new URLSearchParams({ scenario, ...extra })}`,
    );
    await page.waitForFunction(() => window.consistency);
  };
  const selected = async (label) => {
    await page.waitForFunction(
      (element) => element.checked,
      await page
        .getByRole("radio", { name: label, exact: true })
        .elementHandle(),
    );
    assert.equal(
      await page.getByRole("radio", { name: label, exact: true }).isChecked(),
      true,
    );
  };
  const choose = async (label) =>
    page.getByRole("radio", { name: label, exact: true }).locator("..").click();
  const more = async (label) => {
    await page.getByRole("button", { name: "More time ranges" }).click();
    await page
      .getByRole("button", { name: `Last ${label}`, exact: true })
      .click();
    await page.keyboard.press("Escape");
  };
  const params = () => new URL(page.url()).searchParams;
  const requests = (name) =>
    page.evaluate(
      (name) => window.consistency.calls.filter((call) => call.name === name),
      name,
    );
  const settled = () =>
    page.waitForFunction(
      () => window.consistency.queryClient.isFetching() === 0,
    );
  const metricWindow = (call) =>
    (call.request.timeRange.to.seconds - call.request.timeRange.from.seconds) /
    60;

  it("preserves dashboard presets and displays the selected menu range", async () => {
    await load();
    await selected("1h");
    assert.equal(await page.getByRole("radio").count(), 6);
    assert.equal(await page.getByText("Custom", { exact: true }).count(), 0);
    await more("15m");
    await selected("15m");
    assert.equal(await page.locator("[data-period]").textContent(), "15");
    assert.equal(await page.getByRole("radio").count(), 7);
  });

  for (const [range, minutes, step] of [
    ["15m", 15, 1],
    ["1h", 60, 1],
    ["6h", 360, 5],
    ["24h", 1440, 15],
  ]) {
    it(`preserves legacy metrics links and resolution for ${range}`, async () => {
      await load("metrics", { "metrics.range": range, keep: "value" });
      await selected(range);
      await settled();
      const calls = await requests("queryMetrics");
      assert.ok(calls.length > 0);
      assert.ok(calls.every((call) => metricWindow(call) === minutes));
      assert.ok(calls.every((call) => call.request.step.type.minutes === step));
      await choose("3h");
      await page.waitForURL((url) => url.searchParams.get("range") === "180");
      assert.equal(params().has("metrics.range"), false);
      assert.equal(params().get("keep"), "value");
    });
  }

  it("prefers the shared URL range over the legacy metrics parameter", async () => {
    await load("metrics", { range: "1440", "metrics.range": "1h" });
    await selected("24h");
    await settled();
    assert.ok(
      (await requests("queryMetrics")).every(
        (call) => metricWindow(call) === 1440,
      ),
    );
  });

  it("keeps the six-hour metrics default for missing or invalid ranges", async () => {
    for (const range of [undefined, "invalid", "-1", "20160", "43200"]) {
      await load("metrics", range ? { range, "metrics.range": "1h" } : {});
      await selected("6h");
      await settled();
      const calls = await requests("queryMetrics");
      assert.ok(calls.length > 0);
      assert.ok(calls.every((call) => metricWindow(call) === 360));
    }
  });

  it("supports shared metric presets within the existing retention-safe range", async () => {
    await load("metrics", { "metrics.resolution": "15m", keep: "value" });
    await more("5m");
    await page.waitForURL(
      (url) =>
        url.searchParams.get("range") === "5" &&
        !url.searchParams.has("metrics.resolution"),
    );
    await selected("5m");
    await settled();
    const short = (await requests("queryMetrics")).filter(
      (call) => metricWindow(call) === 5,
    );
    assert.ok(short.length > 0);
    assert.ok(short.every((call) => call.request.step.type.minutes === 1));
    await page.getByRole("button", { name: "More time ranges" }).click();
    for (const label of ["2d", "3d", "7d", "14d"]) {
      assert.equal(
        await page
          .getByRole("button", { name: `Last ${label}`, exact: true })
          .count(),
        0,
      );
    }
    await page.keyboard.press("Escape");
    await choose("24h");
    await page.waitForURL((url) => url.searchParams.get("range") === "1440");
    await selected("24h");
    await page.waitForFunction(() =>
      window.consistency.calls.some(
        (call) =>
          call.name === "queryMetrics" &&
          call.request.timeRange.to.seconds -
            call.request.timeRange.from.seconds ===
            1440 * 60,
      ),
    );
    await settled();
    const long = (await requests("queryMetrics")).filter(
      (call) => metricWindow(call) === 1440,
    );
    assert.ok(long.length > 0);
    assert.ok(
      long.every(
        (call) =>
          call.request.step.type.minutes === 15 &&
          metricWindow(call) / 15 <= call.request.limitPointsPerSeries,
      ),
    );
    assert.equal(params().get("keep"), "value");
  });

  for (const scenario of ["service", "namespace"]) {
    it(`shares and restores ${scenario} metrics without losing resource scope`, async () => {
      await load(scenario, { range: "720", keep: "value" });
      await selected("12h");
      await settled();
      let calls = await requests("queryMetrics");
      assert.ok(calls.length > 0);
      assert.ok(calls.every((call) => metricWindow(call) === 720));
      assert.ok(
        calls.every((call) =>
          call.request.filters.some(
            (filter) =>
              filter.key === "octelium.vigil.svc.namespace.name" &&
              filter.value.value.stringValue === "production",
          ),
        ),
      );
      await choose("30m");
      await page.waitForURL((url) => url.searchParams.get("range") === "30");
      await page.reload();
      await selected("30m");
      await settled();
      calls = await requests("queryMetrics");
      assert.ok(calls.length > 0);
      assert.ok(calls.every((call) => metricWindow(call) === 30));
      await choose("6h");
      await page.waitForURL((url) => !url.searchParams.has("range"));
      assert.equal(params().get("keep"), "value");
      await load(scenario, { range: "20160" });
      await selected("6h");
      await settled();
      calls = await requests("queryMetrics");
      assert.ok(calls.length > 0);
      assert.ok(calls.every((call) => metricWindow(call) === 360));
    });
  }

  for (const [scenario, summary, list] of [
    ["access", "getAccessLogSummary", "listAccessLog"],
    ["authentication", "getAuthenticationLogSummary", "listAuthenticationLog"],
    ["audit", "getAuditLogSummary", "listAuditLog"],
  ]) {
    it(`uses the shared URL range for both ${scenario} resource summaries and event lists`, async () => {
      await load(scenario, { range: "180", keep: "value" });
      await selected("3h");
      await settled();
      const currentSummary = (await requests(summary)).find(
        (call) =>
          Math.abs(call.now - call.request.from.seconds * 1000 - 180 * 60_000) <
          1000,
      );
      assert.ok(currentSummary);
      let calls = await requests(list);
      assert.ok(calls.length > 0);
      assert.ok(
        calls.every(
          (call) =>
            Math.abs(
              call.now - call.request.from.seconds * 1000 - 180 * 60_000,
            ) < 1000,
        ),
      );
      assert.equal(
        calls[0].request[scenario === "audit" ? "resourceRef" : "userRef"].uid,
        "user-uid",
      );
      await choose("30m");
      await page.waitForURL((url) => url.searchParams.get("range") === "30");
      await page.reload();
      await selected("30m");
      await settled();
      calls = await requests(list);
      assert.ok(calls.length > 0);
      assert.ok(
        calls.every(
          (call) =>
            Math.abs(
              call.now - call.request.from.seconds * 1000 - 30 * 60_000,
            ) < 1000,
        ),
      );
      assert.equal(params().get("keep"), "value");
    });
  }

  it("restores a resource metrics range through browser back navigation", async () => {
    await load("service", { range: "60" });
    await selected("1h");
    await page.evaluate(() =>
      window.consistency.navigate({ search: "?scenario=service&range=720" }),
    );
    await selected("12h");
    await page.goBack();
    await selected("1h");
  });

  it("preserves the LLM default, thirty-day preset and custom date editing", async () => {
    await load("llm");
    await selected("24h");
    await more("30d");
    await selected("30d");
    assert.equal(
      JSON.parse(await page.locator("[data-llm]").textContent()).minutes,
      43200,
    );
    await choose("Custom");
    await page
      .getByText("Start of range (inclusive)", { exact: true })
      .waitFor();
    await page.locator('button[aria-haspopup="dialog"]').first().click();
    await choose("Date and time");
    await page.getByRole("textbox", { name: "Time" }).fill("09:30");
    await page.getByRole("button", { name: "Apply timestamp" }).click();
    let state = JSON.parse(await page.locator("[data-llm]").textContent());
    assert.equal(state.minutes, undefined);
    assert.ok(state.from.includes("T09:30:00"));
    await choose("1h");
    state = JSON.parse(await page.locator("[data-llm]").textContent());
    assert.equal(state.minutes, 60);
    assert.equal(state.to, undefined);
    await page
      .getByText("Start of range (inclusive)", { exact: true })
      .waitFor({ state: "hidden" });
  });

  it("keeps the shared selector usable on a narrow metrics screen", async () => {
    await page.setViewportSize({ width: 375, height: 900 });
    await load("metrics");
    await choose("12h");
    await page.waitForURL((url) => url.searchParams.get("range") === "720");
    const fits = await page.evaluate(() => {
      const selector = document.querySelector(
        '[aria-label="More time ranges"]',
      );
      return selector.getBoundingClientRect().right <= window.innerWidth;
    });
    assert.equal(fits, true);
  });

  it("keeps LLM presets and custom dates reachable on a narrow screen", async () => {
    await page.setViewportSize({ width: 375, height: 900 });
    await load("llm");
    await more("30d");
    await selected("30d");
    await page.screenshot({ path: "/tmp/octelium-console-cons-01-llm.png" });
    const fits = await page.evaluate(() => {
      const selector = document.querySelector(
        '[aria-label="More time ranges"]',
      );
      return selector.getBoundingClientRect().right <= window.innerWidth;
    });
    assert.equal(fits, true);
    await choose("Custom");
    await page
      .getByText("Start of range (inclusive)", { exact: true })
      .waitFor();
  });
});
