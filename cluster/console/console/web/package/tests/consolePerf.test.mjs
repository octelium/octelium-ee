import assert from "node:assert/strict";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { build, preview } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console performance regressions", () => {
  let directory;
  let server;
  let browser;
  let context;
  let page;
  let origin;
  let manifest;
  let sources;
  let errors;

  before(async () => {
    directory = await mkdtemp(join(tmpdir(), "console-perf-"));
    const appDirectory = join(directory, "app");
    await build({
      root,
      logLevel: "silent",
      build: { outDir: appDirectory, sourcemap: true },
    });
    manifest = JSON.parse(
      await readFile(join(appDirectory, ".vite/manifest.json"), "utf8"),
    );
    sources = new Map();
    for (const [key, entry] of Object.entries(manifest)) {
      if (!entry.file.endsWith(".js")) continue;
      const map = JSON.parse(
        await readFile(join(appDirectory, `${entry.file}.map`), "utf8"),
      );
      sources.set(key, map.sources);
    }
    const fixtureDirectory = join(directory, "fixture");
    await build({
      root,
      logLevel: "silent",
      build: {
        outDir: fixtureDirectory,
        rolldownOptions: {
          input: join(root, "tests/fixtures/consolePerf.html"),
        },
      },
    });
    server = await preview({
      root,
      logLevel: "silent",
      build: { outDir: fixtureDirectory },
      preview: { host: "127.0.0.1", port: 0 },
    });
    origin = server.resolvedUrls.local[0];
    browser = await chromium.launch();
  });

  after(async () => {
    await browser?.close();
    if (server)
      await new Promise((resolve) => server.httpServer.close(resolve));
    if (directory) await rm(directory, { recursive: true, force: true });
  });

  beforeEach(async () => {
    context = await browser.newContext({
      timezoneId: "UTC",
      viewport: { width: 1440, height: 900 },
    });
    page = await context.newPage();
    page.setDefaultTimeout(10_000);
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (mode, options = {}) =>
    page.goto(
      `${origin}tests/fixtures/consolePerf.html?${new URLSearchParams({ mode, ...options })}`,
    );

  const closure = (key, seen = new Set()) => {
    if (seen.has(key)) return seen;
    seen.add(key);
    for (const dependency of manifest[key].imports ?? [])
      closure(dependency, seen);
    return seen;
  };

  it("keeps editors and policy schemas out of the section routers' static imports", () => {
    for (const section of ["core", "enterprise", "access", "cordium"]) {
      const key = [...sources].find(([, files]) =>
        files.some((file) => file.endsWith(`/src/pages/${section}/router.tsx`)),
      )?.[0];
      assert.ok(key, section);
      const files = [...closure(key)].flatMap(
        (dependency) => sources.get(dependency) ?? [],
      );
      for (const file of files) {
        assert.ok(!file.includes("/@codemirror/"), file);
        assert.ok(!file.endsWith("RequestContext.json"), file);
      }
    }
  });

  it("omits the full ECharts wrapper, the language collection, and unused global shims", () => {
    const files = [...sources.values()].flat();
    for (const suffix of [
      "echarts-for-react/esm/index.js",
      "@uiw/codemirror-extensions-langs/esm/index.js",
      "shims/process/dist/index.js",
      "shims/global/dist/index.js",
    ]) {
      assert.ok(!files.some((file) => file.endsWith(suffix)), suffix);
    }
    assert.ok(
      files.some((file) => file.endsWith("shims/buffer/dist/index.js")),
    );
  });

  it("keeps dates, terminal, and flag styles out of the entry CSS", async () => {
    const css = (
      await Promise.all(
        (manifest["index.html"].css ?? []).map((file) =>
          readFile(join(directory, "app", file), "utf8"),
        ),
      )
    ).join("\n");
    for (const suffix of [
      "/src/components/TimestampPicker/index.tsx",
      "/src/components/SSHRecordingPlayer/Player.tsx",
    ]) {
      const key = [...sources].find(([, files]) =>
        files.some((file) => file.endsWith(suffix)),
      )?.[0];
      assert.ok(key, suffix);
      assert.ok(manifest[key].css?.length > 0, suffix);
      assert.ok(
        manifest[key].css.every(
          (file) => !manifest["index.html"].css.includes(file),
        ),
        suffix,
      );
    }
    assert.ok(!css.includes(".xterm-screen"));
    assert.ok(!css.includes(".fi-us"));
  });

  for (const [mode, type, variant] of [
    ["LineChart", "line"],
    ["LineChart", "bar", "bar"],
    ["SeriesChart", "line"],
    ["MetricChart", "line"],
    ["PieChart", "pie"],
    ["PolarChart", "bar"],
    ["GaugeChart", "gauge"],
    ["CounterChart", "gauge"],
    ["Agent", "bar", "bar"],
    ["Agent", "scatter", "scatter"],
    ["Agent", "pie", "pie"],
  ]) {
    it(`renders ${mode} ${type} with the modular ECharts wrapper`, async () => {
      await load(mode, variant ? { variant } : {});
      await page.waitForFunction(
        () => window.chartOption()?.series?.length > 0,
      );
      const option = await page.evaluate(() => window.chartOption());
      assert.equal(option.series[0].type, type);
      assert.ok(option.series[0].data.length > 0);
      assert.ok((await page.locator(".echarts-for-react canvas").count()) > 0);
    });
  }

  it("loads only the Lua mode and highlights Lua tokens", async () => {
    await load("lua");
    await page.waitForFunction(() =>
      window.editorTokens?.().includes("keyword"),
    );
    const tokens = await page.evaluate(() => window.editorTokens());
    assert.ok(tokens.includes("number"));
    assert.ok(tokens.includes("string"));
    assert.equal(
      await page.locator(".cm-content").innerText(),
      'local answer = 42\nreturn "hello"',
    );
  });

  it("dereferences resource schemas and remote refs with Buffer and no process shim", async () => {
    await page.route("**/remote-schema.json", (route) =>
      route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ type: "string" }),
      }),
    );
    await load("schema");
    await page.getByPlaceholder("Search fields or descriptions").waitFor();
    const schema = await page.evaluate(() => window.dereference());
    assert.equal(schema.properties.remote.type, "string");
    assert.equal(await page.evaluate(() => typeof window.process), "undefined");
  });

  it("keeps CEL context completion working without the process shim", async () => {
    await load("cel");
    await page.locator(".cm-content").fill("ctx.");
    await page.locator(".cm-content").press("Control+Space");
    await page.locator(".cm-tooltip-autocomplete li").first().waitFor();
    assert.ok((await page.locator(".cm-tooltip-autocomplete li").count()) > 0);
    assert.equal(
      await page
        .getByText("Context completion is unavailable", { exact: false })
        .count(),
      0,
    );
  });

  it("keeps CEL text and the cursor readable when switching light and dark modes", async () => {
    await load("cel");
    const content = page.locator(".cm-content");
    const expression = '(ctx.user.metadata.name == "alice") && true';
    await content.fill(expression);
    await page.locator(".cm-cursor").waitFor();

    for (const scheme of ["dark", "light", "dark"]) {
      const colors = await page.evaluate((scheme) => {
        document.documentElement.setAttribute(
          "data-mantine-color-scheme",
          scheme,
        );
        const probe = document.createElement("div");
        probe.style.backgroundColor = "var(--color-white)";
        probe.style.color = "var(--color-slate-800)";
        probe.style.caretColor = "var(--color-slate-900)";
        document.body.append(probe);
        const expected = getComputedStyle(probe);
        const editor = getComputedStyle(document.querySelector(".cm-editor"));
        const cursor = getComputedStyle(document.querySelector(".cm-cursor"));
        const result = {
          background: editor.backgroundColor,
          expectedBackground: expected.backgroundColor,
          foreground: editor.color,
          expectedForeground: expected.color,
          cursor: cursor.borderLeftColor,
          expectedCursor: expected.caretColor,
          tokens: Array.from(document.querySelectorAll(".cm-line span")).map(
            (element) => getComputedStyle(element).color,
          ),
        };
        probe.remove();
        return result;
      }, scheme);
      assert.equal(colors.background, colors.expectedBackground, scheme);
      assert.equal(colors.foreground, colors.expectedForeground, scheme);
      assert.equal(colors.cursor, colors.expectedCursor, scheme);
      assert.notEqual(colors.foreground, colors.background, scheme);
      assert.ok(colors.tokens.length > 0);
      assert.ok(colors.tokens.every((color) => color !== colors.background));
      assert.equal(await content.innerText(), expression);
    }
  });

  it("fetches just the displayed flags and handles invalid country codes", async () => {
    const requests = [];
    page.on("request", (request) => {
      if (request.url().endsWith(".svg")) requests.push(request.url());
    });
    await load("flags");
    await page.waitForFunction(
      () => document.querySelector("img")?.naturalWidth > 0,
    );
    assert.equal(await page.locator("img").count(), 1);
    assert.equal(requests.length, 1);
    assert.ok(new URL(requests[0]).pathname.startsWith("/assets/us-"));
    await page.getByRole("button", { name: "Change flag" }).click();
    await page.waitForFunction(
      () =>
        document.querySelector("img")?.src.includes("/fr-") &&
        document.querySelector("img")?.naturalWidth > 0,
    );
    assert.equal(requests.length, 2);
  });

  it("loads date styles with the timestamp picker", async () => {
    await load("dates");
    await page.getByRole("button", { name: "Select date and time" }).click();
    await page
      .locator(".mantine-SegmentedControl-label")
      .filter({ hasText: "Date and time" })
      .click();
    const day = page.locator(".mantine-DatePicker-day").first();
    await day.waitFor();
    assert.equal(
      await day.evaluate((element) => getComputedStyle(element).display),
      "inline-flex",
    );
  });

  it("loads terminal styles with the SSH player", async () => {
    await load("terminal");
    const screen = page.locator(".xterm-screen");
    await screen.waitFor();
    assert.equal(
      await screen.evaluate((element) => getComputedStyle(element).position),
      "relative",
    );
  });

  it("opens the YAML editor on demand", async () => {
    await load("yaml");
    await page.getByRole("button", { name: /YAML/ }).click();
    await page.locator(".cm-content").waitFor();
    assert.ok(
      (await page.locator(".cm-content").innerText()).includes("perf-secret"),
    );
  });

  it("loads the inline policy form only when a policy is present", async () => {
    await load("inline");
    await page.getByRole("button", { name: "Open policy" }).click();
    await page.getByRole("textbox", { name: "Name", exact: true }).waitFor();
    await page.getByText("Rules", { exact: true }).waitFor();
  });

  for (const path of [
    "/core/secrets/create",
    "/core/secrets/perf-secret/edit",
  ]) {
    it(`loads the resource form at ${path}`, async () => {
      await load("routes", { path });
      await page
        .locator(".mantine-SegmentedControl-label")
        .filter({ hasText: "YAML" })
        .last()
        .click();
      await page.locator(".cm-content").waitFor();
      assert.ok(
        (await page.locator(".cm-content").innerText()).includes(
          "kind: Secret",
        ),
      );
    });
  }
});
