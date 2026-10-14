import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console bug regressions", () => {
  let server;
  let browser;
  let context;
  let page;
  let origin;

  before(async () => {
    server = await createServer({
      root,
      server: { host: "127.0.0.1", port: 0, watch: null },
    });
    await server.listen();
    origin = server.resolvedUrls.local[0];
    browser = await chromium.launch();
  });

  after(async () => {
    await browser?.close();
    await server?.close();
  });

  beforeEach(async () => {
    context = await browser.newContext({
      timezoneId: "UTC",
      viewport: { width: 1440, height: 900 },
    });
    page = await context.newPage();
  });

  afterEach(async () => {
    await context?.close();
  });

  const load = async (scenario = "shared", path = "/", extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/consoleBugs.html?${new URLSearchParams({ scenario, path, ...extra })}`,
    );
  };

  it("keeps Select usable when a document element throws on shadowRoot", async () => {
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await load("shadow");
    const input = page.locator("css:light=.mantine-Select-input");
    await input.fill("Tw");
    await page.locator('css:light=[role="option"]').click();
    assert.equal(await input.inputValue(), "Two");
    assert.deepEqual(errors, []);
  });

  it("preserves searches in shadow roots and skips inaccessible hosts", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { findElementBySelector, findElementsBySelector, getRootElement } =
        await import("/src/utils/mantine/find-element-in-shadow-dom.js");
      const host = document.createElement("div");
      const root = host.attachShadow({ mode: "open" });
      const forbidden = document.createElement("div");
      Object.defineProperty(forbidden, "shadowRoot", {
        get() {
          throw new Error("Denied");
        },
      });
      root.append(forbidden);
      const nested = document.createElement("div");
      const childRoot = nested.attachShadow({ mode: "open" });
      childRoot.innerHTML = '<span class="target">Found</span>';
      root.append(nested);
      document.body.append(host);
      return [
        findElementBySelector(".target", root)?.textContent,
        findElementsBySelector(".target", root).length,
        getRootElement(childRoot.querySelector("span")) === childRoot,
        findElementBySelector(".missing"),
      ];
    });
    assert.deepEqual(result, ["Found", 1, true, null]);
  });

  it("keeps custom text sizes alongside colors in either order", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { cn, twMerge } = await import("/src/lib/utils.ts");
      return [
        cn("text-micro", "text-red-700"),
        twMerge("text-red-700", "text-body"),
        twMerge("text-micro text-body text-red-700"),
        twMerge("text-body text-xs text-red-700"),
      ];
    });
    assert.deepEqual(result, [
      "text-micro text-red-700",
      "text-red-700 text-body",
      "text-body text-red-700",
      "text-xs text-red-700",
    ]);
    assert.equal(
      await page
        .getByText("Critical", { exact: true })
        .evaluate((element) => getComputedStyle(element).fontSize),
      "11px",
    );
  });

  it("allows Tailwind to style native buttons and inputs", async () => {
    await load();
    await page.locator("[data-control]").first().waitFor();
    assert.deepEqual(
      await page.locator("[data-control]").evaluateAll((elements) =>
        elements.map((element) => ({
          size: getComputedStyle(element).fontSize,
          weight: getComputedStyle(element).fontWeight,
        })),
      ),
      [
        { size: "12px", weight: "500" },
        { size: "12px", weight: "500" },
      ],
    );
  });

  it("distinguishes new growth, zero activity and percentage changes", async () => {
    await load();
    await page.getByText("New · +81", { exact: true }).waitFor();
    await page.getByText("No activity", { exact: true }).waitFor();
    const result = await page.evaluate(async () => {
      const { deltaPct } = await import("/src/utils/visibility/index.ts");
      return [
        deltaPct(81, 0),
        deltaPct(0, 0),
        deltaPct(10, 10),
        deltaPct(20, 10),
        deltaPct(0, 10),
      ];
    });
    assert.deepEqual(result, [null, null, 0, 100, -100]);
    assert.equal(await page.getByText("Flat", { exact: true }).count(), 0);
  });

  it("formats byte edge cases and both unit systems", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { formatBytes } = await import("/src/utils/index.ts");
      return [
        formatBytes(0),
        formatBytes(1),
        formatBytes(999),
        formatBytes(1000),
        formatBytes(1024, { useBinaryUnits: true }),
        formatBytes(-1024, { useBinaryUnits: true }),
        formatBytes(0.1),
        formatBytes(NaN),
        formatBytes(Infinity),
        formatBytes(Number.MAX_VALUE),
      ];
    });
    assert.deepEqual(result.slice(0, 9), [
      "0 Bytes",
      "1 Bytes",
      "999 Bytes",
      "1.00 KB",
      "1.00 KiB",
      "-1.00 KiB",
      "0 Bytes",
      "—",
      "—",
    ]);
    assert.ok(result[9].endsWith(" YB"));
  });

  it("renders numeric hours on all three time-series axes", async () => {
    await load("charts");
    await page.waitForFunction(
      () => window.chartLabels?.().filter(Boolean).length === 3,
    );
    assert.deepEqual(await page.evaluate(() => window.chartLabels()), [
      "10/12\n13:14",
      "10/12\n13:14",
      "10/12\n13:14",
    ]);
  });

  for (const scenario of [
    "source-error",
    "health-error",
    "partial",
    "unknown",
  ]) {
    it(`does not claim all-clear with ${scenario} signals`, async () => {
      await load(scenario);
      await page.getByText("Incomplete assessment", { exact: true }).waitFor();
      assert.equal(
        await page
          .getByText("Nothing needs your attention", { exact: true })
          .count(),
        0,
      );
      assert.equal(
        await page
          .getByText(
            "Everything across Core, Access and Enterprise looks healthy",
            { exact: true },
          )
          .count(),
        0,
      );
    });
  }

  it("includes critical health subsystems in attention", async () => {
    await load("critical");
    await page.getByText("Critical", { exact: true }).waitFor();
    assert.equal(
      await page
        .getByText("Nothing needs your attention", { exact: true })
        .count(),
      0,
    );
  });

  it("shows all-clear when every source and health subsystem is healthy", async () => {
    await load("healthy");
    await page
      .getByText("Nothing needs your attention", { exact: true })
      .waitFor();
  });

  it("keeps raw RPC details behind a working disclosure", async () => {
    await load();
    await page
      .getByText("This feature is not available on this cluster.", {
        exact: true,
      })
      .waitFor();
    const detail = page.getByText(
      "Could not be evaluated: rpc error: code = Unimplemented desc = unknown service",
      { exact: true },
    );
    assert.equal(await detail.isVisible(), false);
    await page.getByText("Details", { exact: true }).click();
    assert.equal(await detail.isVisible(), true);
    assert.ok(page.url().includes("consoleBugs.html"));
  });

  it("maps direct RPC codes and embedded server errors", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { describeRpcError } = await import("/src/utils/rpcError.ts");
      return [
        describeRpcError({
          code: "PERMISSION_DENIED",
          message: "Denied by policy",
        }).message,
        describeRpcError("rpc error: code = Unauthenticated desc = expired")
          .message,
        describeRpcError("rpc error: code = Internal desc = backend exploded")
          .message,
      ];
    });
    assert.deepEqual(result, [
      "You do not have permission to view this information.",
      "Your session has expired. Sign in again.",
      "This information could not be loaded. Try again.",
    ]);
  });

  it("links the overdue request count to the deadline filter", async () => {
    await load("deadline");
    await page
      .getByRole("link", { name: /Access requests past their deadline/ })
      .waitFor();
    assert.equal(
      await page
        .getByRole("link", { name: /Access requests past their deadline/ })
        .getAttribute("href"),
      "/access/requests?isDeadlinePassed=true",
    );
  });

  for (const path of ["/", "/agent", "/settings"]) {
    it(`avoids selecting Core for ${path}`, async () => {
      await load("sidebar", path);
      await page.getByRole("button", { name: /Choose workspace/ }).waitFor();
      assert.equal(
        await page.getByRole("link", { name: "Services", exact: true }).count(),
        0,
      );
    });
  }

  it("navigates to the current section overview and exposes Cluster management", async () => {
    await load("sidebar", "/core/services");
    await page.getByRole("button", { name: /API workspace Core/ }).click();
    await page
      .getByRole("menuitemradio", { name: "Core", exact: true })
      .click();
    await page.waitForFunction(
      () => document.querySelector("[data-location]")?.textContent === "/core",
    );
    assert.equal(await page.locator("[data-location]").textContent(), "/core");
    await page.getByRole("button", { name: /API workspace Core/ }).click();
    await page
      .getByRole("menuitemradio", { name: "Cluster Management", exact: true })
      .click();
    await page.waitForFunction(
      () =>
        document.querySelector("[data-location]")?.textContent ===
        "/clusterman",
    );
    assert.equal(
      await page.locator("[data-location]").textContent(),
      "/clusterman",
    );
    assert.equal(
      await page
        .getByRole("link", { name: "Cluster overview", exact: true })
        .getAttribute("href"),
      "/clusterman",
    );
  });

  it("validates redirect paths and removes recursive redirect parameters", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { getRedirectPath } = await import("/src/utils/redirect.ts");
      return [
        "/core/users?state=ACTIVE#row",
        "//host",
        "/\\host",
        "https://host",
        "/\n/host",
        "/core?redirect=%2Fcore",
        "/%2Fhost",
      ].map((value) => getRedirectPath(value, "https://console.example.com"));
    });
    assert.deepEqual(result, [
      "/core/users?state=ACTIVE#row",
      null,
      null,
      null,
      null,
      "/core",
      "/%2Fhost",
    ]);
  });

  it("uses API responses for development log lists", async () => {
    await load("logs");
    await page.waitForFunction(
      () =>
          window.calls?.filter((call) => call.name.startsWith("list")).length >=
          5,
    );
    assert.deepEqual(
      await page.evaluate(() =>
        window.calls
          .filter((call) => call.name.startsWith("list"))
          .map((call) => call.name)
          .sort(),
      ),
      [
        "listAccessLog",
        "listAuditLog",
        "listAuthenticationLog",
        "listComponentLog",
        "listSSHSession",
      ],
    );
    assert.equal(await page.getByText("alice", { exact: true }).count(), 0);
  });

  it("recomputes dashboard windows on refetch without a period change", async () => {
    await page.clock.install({ time: new Date("2026-10-12T13:00:00Z") });
    await load("query");
    await page.waitForFunction(() => window.calls?.length === 1);
    await page.clock.fastForward(10_000);
    await page.getByRole("button", { name: "Refetch window" }).click();
    await page.waitForFunction(() => window.calls.length === 2);
    const result = await page.evaluate(() =>
      window.calls.map((call) => call.request.common),
    );
    assert.equal(
      Number(result[1].to.seconds) - Number(result[0].to.seconds),
      10,
    );
    assert.equal(
      Number(result[1].from.seconds) - Number(result[0].from.seconds),
      10,
    );
    assert.equal(
      Number(result[1].compareTo.seconds),
      Number(result[1].from.seconds),
    );
  });

  for (const scenario of ["avatar", "avatar-empty"]) {
    it(`shows an account identity for ${scenario}`, async () => {
      await load(scenario);
      const account = page.getByRole("button", { name: "Account menu" });
      await account.waitFor();
      if (scenario === "avatar")
        assert.equal(await account.textContent(), "a");
      else
        assert.equal(await account.locator("svg.lucide-user-round").count(), 1);
      await account.click();
      const menu = page.getByRole("menu");
      await menu.waitFor();
      assert.equal(
        await menu.getByText("alice", { exact: true }).count(),
        scenario === "avatar" ? 1 : 0,
      );
      assert.equal(await menu.getByText(/@/).count(), 0);
    });
  }

  for (const displayName of ["", "Alice Example"]) {
    it(`shows the email beneath the ${displayName ? "display name" : "user name"} in the account menu`, async () => {
      await load("avatar", "/", {
        displayName,
        email: "  alice@example.test  ",
      });
      const account = page.getByRole("button", { name: "Account menu" });
      await account.waitFor();
      assert.equal(await account.textContent(), displayName ? "AE" : "a");
      await account.click();
      const menu = page.getByRole("menu");
      const name = menu.getByText(displayName || "alice", { exact: true });
      const email = menu.getByText("alice@example.test", { exact: true });
      await email.waitFor();
      const nameBox = await name.boundingBox();
      const emailBox = await email.boundingBox();
      assert.ok(emailBox.y >= nameBox.y + nameBox.height);
    });
  }

  for (const email of ["", "   "]) {
    it(`omits an ${email ? "all-whitespace" : "empty"} email from the account menu`, async () => {
      await load("avatar", "/", { email });
      await page.getByRole("button", { name: "Account menu" }).click();
      const menu = page.getByRole("menu");
      await menu.waitFor();
      assert.equal(
        await menu.locator(".mantine-Menu-label").textContent(),
        "alice",
      );
    });
  }

  it("shows the email when the account name is missing", async () => {
    await load("avatar-empty", "/", { email: "alice@example.test" });
    await page.getByRole("button", { name: "Account menu" }).click();
    await page
      .getByRole("menu")
      .getByText("alice@example.test", { exact: true })
      .waitFor();
  });

  for (const width of [1440, 390]) {
    it(`nests the resource schema inside YAML at ${width}px`, async () => {
      await page.setViewportSize({ width, height: 900 });
      await load("overview", "/core/users/alice");
      const overview = page.getByRole("radiogroup", {
        name: "Overview content",
      });
      const yaml = page.getByRole("radiogroup", { name: "YAML content" });
      await overview.waitFor();
      assert.equal(await overview.getByRole("radio").count(), 3);
      assert.equal(
        await overview.getByRole("radio", { name: "Schema" }).count(),
        0,
      );
      assert.equal(await yaml.count(), 0);

      await overview.getByText("YAML", { exact: true }).click();
      await page.locator(".cm-content").waitFor();
      assert.equal(await yaml.getByRole("radio").count(), 2);
      assert.equal(
        await yaml
          .getByRole("radio", { name: "Content", exact: true })
          .isChecked(),
        true,
      );
      const content = page.locator(".cm-content");
      const value = await content.innerText();
      assert.match(value, /name: alice/);
      await content.click();
      await page.keyboard.type("changed");
      assert.equal(await content.innerText(), value);

      const style = (element) => {
        const computed = getComputedStyle(element);
        return [
          computed.backgroundColor,
          computed.borderRadius,
          computed.padding,
        ];
      };
      assert.deepEqual(
        await yaml.evaluate(style),
        await overview.evaluate(style),
      );
      const box = await yaml.boundingBox();
      assert.ok(box.x >= 0 && box.x + box.width <= width);

      await yaml.getByText("Schema", { exact: true }).click();
      await page.getByPlaceholder("Search fields or descriptions").waitFor();
      assert.equal(
        await page.getByText("User fields", { exact: true }).count(),
        1,
      );
      assert.equal(await page.locator(".cm-content").count(), 0);
      assert.equal(
        await page.getByRole("button", { name: "Copy YAML" }).count(),
        0,
      );

      await overview.getByText("Metadata", { exact: true }).click();
      assert.equal(await yaml.count(), 0);
      await overview.getByText("YAML", { exact: true }).click();
      assert.equal(
        await yaml
          .getByRole("radio", { name: "Schema", exact: true })
          .isChecked(),
        true,
      );
      await yaml.getByText("Content", { exact: true }).click();
      await page.locator(".cm-content").waitFor();
      await page.getByRole("button", { name: "Copy YAML" }).waitFor();
    });
  }

  it("wraps stat tile labels and footers without ellipses", async () => {
    await load();
    const footer = page.getByText(
      "81 new sessions in the last 24 hours, including all available client types",
      { exact: true },
    );
    await footer.waitFor();
    assert.equal(
      await footer.evaluate((element) => getComputedStyle(element).whiteSpace),
      "normal",
    );
    assert.equal(
      await page
        .getByText("Approved · 24h", { exact: true })
        .evaluate((element) => getComputedStyle(element).textOverflow),
      "clip",
    );
  });

  it("releases every queue slot after synchronous throws", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { queued } = await import("/src/utils/visibility/queue.ts");
      const outcomes = await Promise.allSettled(
        Array.from({ length: 8 }, () =>
          queued(0, () => {
            throw new Error("Synchronous failure");
          }),
        ),
      );
      return [
        outcomes.filter((outcome) => outcome.status === "rejected").length,
        await queued(0, async () => "Recovered"),
      ];
    });
    assert.deepEqual(result, [8, "Recovered"]);
  });

  for (const [stored, system, expected] of [
    [null, "dark", "dark"],
    ["auto", "light", "light"],
    ["dark", "light", "dark"],
    ["light", "dark", "light"],
    ["invalid", "dark", "dark"],
  ]) {
    it(`initializes the theme under CSP for ${stored} / ${system}`, async () => {
      const html = (await readFile(`${root}index.html`, "utf8"))
        .replace(/<script type="module"[^>]*><\/script>/g, "")
        .replace(/<script>/g, '<script nonce="test-theme">');
      await page.emulateMedia({ colorScheme: system });
      await page.addInitScript((value) => {
        if (value === null) localStorage.removeItem("octelium-color-scheme");
        else localStorage.setItem("octelium-color-scheme", value);
      }, stored);
      await page.route("**/theme.html", (route) =>
        route.fulfill({
          contentType: "text/html",
          headers: {
            "Content-Security-Policy":
              "script-src 'self' 'nonce-test-theme'; style-src 'self' 'unsafe-inline'",
          },
          body: html,
        }),
      );
      await page.goto(`${origin}theme.html`);
      assert.equal(
        await page.locator("html").getAttribute("data-mantine-color-scheme"),
        expected,
      );
      assert.equal(
        await page.locator('meta[name="color-scheme"]').getAttribute("content"),
        "light dark",
      );
      assert.equal(
        await page
          .locator("html")
          .evaluate((element) => getComputedStyle(element).colorScheme),
        expected,
      );
    });
  }
});

it("uses the configured Tailwind merge for every console import", async () => {
  const files = await readdir(`${root}src`, { recursive: true });
  for (const file of files.filter(
    (file) => /\.(ts|tsx)$/.test(file) && file !== "lib/utils.ts",
  )) {
    assert.doesNotMatch(
      await readFile(`${root}src/${file}`, "utf8"),
      /from ["']tailwind-merge["']/,
      file,
    );
  }
});
