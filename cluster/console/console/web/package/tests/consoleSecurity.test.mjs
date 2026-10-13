import assert from "node:assert/strict";
import { mkdtemp, readFile, readdir, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import ts from "typescript";
import { build, preview } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console security regressions", () => {
  let directory;
  let browser;
  let context;
  let page;
  let errors;
  const servers = new Map();

  before(async () => {
    directory = await mkdtemp(join(tmpdir(), "console-security-"));
    for (const mode of ["production", "development"]) {
      const outDir = join(directory, mode);
      await build({
        root,
        mode,
        logLevel: "silent",
        build: {
          outDir,
          rolldownOptions: {
            input: join(root, "tests/fixtures/consoleSecurity.html"),
          },
        },
      });
      servers.set(
        mode,
        await preview({
          root,
          logLevel: "silent",
          build: { outDir },
          preview: { host: "127.0.0.1", port: 0 },
        }),
      );
    }
    browser = await chromium.launch();
  });

  after(async () => {
    await browser?.close();
    for (const server of servers.values())
      await new Promise((resolve) => server.httpServer.close(resolve));
    if (directory) await rm(directory, { recursive: true, force: true });
  });

  beforeEach(async () => {
    context = await browser.newContext({
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

  const load = async (
    params = {},
    mode = "production",
    globals = { cluster: { domain: "example.test" } },
  ) => {
    await page.addInitScript((globals) => {
      window.__OCTELIUM_GLOBALS__ = globals;
    }, globals);
    const origin = servers.get(mode).resolvedUrls.local[0];
    await page.goto(
      `${origin}tests/fixtures/consoleSecurity.html?${new URLSearchParams(params)}`,
    );
    await page.waitForFunction(() => window.security);
  };

  it("uses the server domain for credentialed RPCs despite an injected domain cookie", async () => {
    await page.addInitScript(() => {
      document.cookie = "octelium_domain=namespace.example.test; path=/";
    });
    await load();
    const result = await page.evaluate(async () => {
      const requests = [];
      const originalFetch = window.fetch;
      window.fetch = async (url, init) => {
        requests.push({ url, credentials: init.credentials });
        const trailer = new TextEncoder().encode("grpc-status: 0\r\n");
        return new Response(
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
        );
      };
      try {
        await window.security.requestStatus();
      } finally {
        window.fetch = originalFetch;
      }
      document.cookie = "octelium_domain=other.example.test; path=/";
      return {
        requests,
        domain: window.security.getDomain(),
        isDev: window.security.isDev(),
        cookie: document.cookie,
      };
    });
    assert.equal(result.isDev, false);
    assert.equal(result.domain, "example.test");
    assert.ok(result.cookie.includes("octelium_domain=other.example.test"));
    assert.equal(result.requests.length, 1);
    assert.ok(
      result.requests[0].url.startsWith("http://octelium-api.example.test/"),
    );
    assert.ok(result.requests[0].url.endsWith("/GetStatus"));
    assert.equal(result.requests[0].credentials, "include");
  });

  for (const globals of [
    null,
    {},
    { cluster: null },
    { cluster: {} },
    { cluster: { domain: "" } },
    { cluster: { domain: 123 } },
  ]) {
    it(`rejects invalid server globals ${JSON.stringify(globals)} without trusting the cookie`, async () => {
      await page.addInitScript(() => {
        document.cookie = "octelium_domain=namespace.example.test; path=/";
      });
      await load({}, "production", globals);
      const result = await page.evaluate(() => {
        try {
          window.security.transportOptions();
          return "unexpected transport";
        } catch (error) {
          return error.message;
        }
      });
      assert.equal(result, "The cluster domain is unavailable");
    });
  }

  it("keeps development RPCs on the current host without bootstrap globals", async () => {
    await page.addInitScript(() => {
      delete window.__OCTELIUM_GLOBALS__;
      document.cookie = "octelium_domain=namespace.example.test; path=/";
    });
    await load({}, "development", null);
    const result = await page.evaluate(() => ({
      domain: window.security.getDomain(),
      isDev: window.security.isDev(),
      options: window.security.transportOptions(),
    }));
    assert.equal(result.isDev, true);
    assert.equal(result.domain, new URL(page.url()).host);
    assert.equal(result.options.baseUrl, new URL(page.url()).origin);
    assert.equal(result.options.fetchInit.credentials, "include");
  });

  for (const [mode, expected] of [
    [
      "avatars",
      [
        "topbar",
        "settings",
        "overview",
        "info",
        "top-list",
        "leaderboard",
        "selected",
        "row",
      ],
    ],
    ["list", ["list"]],
  ]) {
    it(`loads ${mode === "list" ? "resource list" : "detail and picker"} avatars without a referrer or requiring provider CORS headers`, async () => {
      const requests = [];
      if (mode === "list") {
        await page.addInitScript(() => {
          window.localStorage.setItem(
            "octelium.console.listDensity.core.listUser",
            "comfortable",
          );
        });
      }
      await context.route("https://avatars.example.test/**", async (route) => {
        requests.push({
          name: new URL(route.request().url()).pathname.slice(1, -4),
          headers: await route.request().allHeaders(),
        });
        await route.fulfill({
          contentType: "image/svg+xml",
          body: '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24"><rect width="24" height="24" fill="blue"/></svg>',
        });
      });
      await load({ mode });
      const images = page.locator('img[src^="https://avatars.example.test/"]');
      await images.nth(expected.length - 1).waitFor({ state: "attached" });
      assert.equal(await images.count(), expected.length);
      for (const name of expected) {
        const image = page.locator(
          `img[src="https://avatars.example.test/${name}.svg"]`,
        );
        await image.scrollIntoViewIfNeeded();
        await page.waitForFunction(
          (name) =>
            document.querySelector(
              `img[src="https://avatars.example.test/${name}.svg"]`,
            )?.naturalWidth > 0,
          name,
        );
      }
      assert.deepEqual(
        requests.map((request) => request.name).sort(),
        [...expected].sort(),
      );
      for (const request of requests)
        assert.equal(request.headers.referer, undefined);
    });
  }

  it("redacts every current-route query value in error reports", async () => {
    await load({ email: "alice@example.test", uid: "private-uid", flag: "" });
    const result = await page.evaluate(() => {
      const report = window.security.buildErrorReport("Failure", {
        source: "test",
      });
      return { report, text: window.security.formatErrorReport(report) };
    });
    const route = new URL(result.report.route, "https://console.example.test");
    assert.equal(route.pathname, "/tests/fixtures/consoleSecurity.html");
    assert.deepEqual(
      [...route.searchParams],
      [
        ["email", "[redacted]"],
        ["uid", "[redacted]"],
        ["flag", "[redacted]"],
      ],
    );
    assert.ok(!result.text.includes("alice"));
    assert.ok(!result.text.includes("private-uid"));
    assert.equal(result.report.message, "Failure");
  });

  it("redacts encoded and repeated values and drops fragments from explicit report routes", async () => {
    await load();
    const result = await page.evaluate(() =>
      window.security.buildErrorReport("Failure", {
        source: "test",
        route:
          "/core/users?email=alice%40example.test&uid=one&uid=two&flag&name=%E6%97%A5%E6%9C%AC&redirect=%2Fcore%3Ftoken%3Dsecret#private-fragment",
      }),
    );
    const route = new URL(result.route, "https://console.example.test");
    assert.equal(route.pathname, "/core/users");
    assert.equal(route.hash, "");
    assert.deepEqual(
      [...route.searchParams],
      [
        ["email", "[redacted]"],
        ["uid", "[redacted]"],
        ["uid", "[redacted]"],
        ["flag", "[redacted]"],
        ["name", "[redacted]"],
        ["redirect", "[redacted]"],
      ],
    );
  });

  it("preserves routes without query values", async () => {
    await load();
    const routes = await page.evaluate(() =>
      [
        "/core/users/alice",
        "",
        "/core/users?",
        "/settings#private-fragment",
      ].map(
        (route) =>
          window.security.buildErrorReport("Failure", { source: "test", route })
            .route,
      ),
    );
    assert.deepEqual(routes, [
      "/core/users/alice",
      "",
      "/core/users?",
      "/settings",
    ]);
  });

  it("keeps query values out of copied reports and console route diagnostics", async () => {
    await page.addInitScript(() => {
      window.copiedReports = [];
      Object.defineProperty(navigator, "clipboard", {
        value: { writeText: async (text) => window.copiedReports.push(text) },
      });
    });
    const logs = [];
    page.on("console", (message) => {
      if (message.type() === "error") logs.push(message.text());
    });
    await load({
      mode: "error",
      email: "alice@example.test",
      token: "private-token",
    });
    await page.getByRole("button", { name: "Copy report" }).click();
    await page.getByRole("button", { name: "Copied", exact: true }).waitFor();
    await page.evaluate(() =>
      window.security.logAppError("Failure", { source: "test" }),
    );
    const copied = await page.evaluate(() => window.copiedReports);
    assert.equal(copied.length, 1);
    assert.equal(logs.length, 1);
    for (const text of [...copied, ...logs]) {
      assert.ok(text.includes("redacted"));
      assert.ok(!text.includes("alice"));
      assert.ok(!text.includes("private-token"));
    }
  });

  it("opens public Services without an opener or referrer", async () => {
    let headers;
    await context.route("https://web.example.test/**", async (route) => {
      headers = await route.request().allHeaders();
      await route.fulfill({
        contentType: "text/html",
        body: "<html><body>Service</body></html>",
      });
    });
    await load({ mode: "service" });
    const popupPromise = context.waitForEvent("page");
    await page.getByRole("link", { name: "Visit" }).click();
    const popup = await popupPromise;
    await popup.waitForLoadState();
    assert.equal(await popup.evaluate(() => window.opener), null);
    assert.equal(headers.referer, undefined);
    await popup.close();
  });

  it("protects all metadata images and new-tab links in console JSX", async () => {
    const files = await readdir(join(root, "src"), { recursive: true });
    let images = 0;
    let links = 0;
    for (const file of files.filter((file) => file.endsWith(".tsx"))) {
      const source = ts.createSourceFile(
        file,
        await readFile(join(root, "src", file), "utf8"),
        ts.ScriptTarget.Latest,
        true,
        ts.ScriptKind.TSX,
      );
      const visit = (node) => {
        if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
          const attrs = new Map(
            node.attributes.properties
              .filter(ts.isJsxAttribute)
              .map((attr) => [attr.name.getText(source), attr.initializer]),
          );
          if (
            node.tagName.getText(source) === "img" &&
            attrs.get("src")?.getText(source).includes("picURL")
          ) {
            images++;
            assert.equal(
              attrs.get("referrerPolicy")?.getText(source),
              '"no-referrer"',
              file,
            );
          }
          if (attrs.get("target")?.getText(source) === '"_blank"') {
            links++;
            const rel = attrs.get("rel");
            assert.ok(rel && ts.isStringLiteral(rel), file);
            assert.ok(rel.text.split(/\s+/).includes("noopener"), file);
            assert.ok(rel.text.split(/\s+/).includes("noreferrer"), file);
          }
        }
        ts.forEachChild(node, visit);
      };
      visit(source);
    }
    assert.ok(images >= 9);
    assert.ok(links > 0);
  });
});
