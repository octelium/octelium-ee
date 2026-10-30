import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console shared presentation and controls", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-patterns-"));
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
      locale: "en-US",
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

  const load = async (scenario, extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/consolePatterns.html?${new URLSearchParams({ scenario, ...extra })}`,
    );
    await page.waitForFunction(() => window.patterns);
    await page.locator("[data-route]").waitFor();
  };
  const calls = async (name) =>
    page.evaluate(
      (name) =>
        window.patterns.state.calls.filter((call) => call.name === name),
      name,
    );
  const routeParams = async () =>
    new URL(`http://fixture${await page.locator("[data-route]").textContent()}`)
      .searchParams;
  const settled = () =>
    page.waitForFunction(() => window.patterns.queryClient.isFetching() === 0);
  const palette = () =>
    page.evaluate(() => {
      const probe = document.createElement("span");
      document.body.append(probe);
      const colors = Object.fromEntries(
        [
          "white",
          "slate-50",
          "slate-200",
          "slate-500",
          "slate-900",
          "terminal",
          "terminal-panel",
          "terminal-text",
          "code",
          "code-border",
        ].map((name) => {
          probe.style.color = `var(--color-${name})`;
          return [name, getComputedStyle(probe).color];
        }),
      );
      probe.remove();
      return colors;
    });
  const surface = (locator, border = "borderTopColor") =>
    locator.evaluate((element, border) => {
      const style = getComputedStyle(element);
      return {
        background: style.backgroundColor,
        border: style[border],
        color: style.color,
      };
    }, border);
  const contrast = (foreground, background) =>
    page.evaluate(({ foreground, background }) => {
      const canvas = document.createElement("canvas");
      canvas.width = canvas.height = 1;
      const context = canvas.getContext("2d");
      const luminance = (color) => {
        context.fillStyle = color;
        context.fillRect(0, 0, 1, 1);
        const channels = Array.from(context.getImageData(0, 0, 1, 1).data)
          .slice(0, 3)
          .map((value) => {
            const channel = value / 255;
            return channel <= 0.04045
              ? channel / 12.92
              : ((channel + 0.055) / 1.055) ** 2.4;
          });
        return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
      };
      const values = [luminance(foreground), luminance(background)];
      return (Math.max(...values) + 0.05) / (Math.min(...values) + 0.05);
    }, { foreground, background });
  const labelShape = (locator) =>
    locator.evaluate((element) => {
      const style = getComputedStyle(element);
      let borders = 0;
      for (
        let parent = element;
        parent && !parent.hasAttribute("data-kind");
        parent = parent.parentElement
      ) {
        if (parseFloat(getComputedStyle(parent).borderTopWidth) > 0) borders++;
      }
      return {
        borders,
        height: element.getBoundingClientRect().height,
        radius: style.borderRadius,
        fontSize: style.fontSize,
        fontWeight: style.fontWeight,
        padding: style.padding,
      };
    });

  for (const [state, label, tone] of [
    [1, "Active", "emerald"],
    [2, "Pending", "amber"],
    [3, "Rejected", "red"],
  ]) {
    it(`renders ${label} consistently in Session and Device rows`, async () => {
      await load("labels", { state: String(state) });
      for (const kind of ["session", "device"]) {
        const badge = page
          .locator(`[data-kind="${kind}"]`)
          .getByText(label, { exact: true });
        const style = await badge.getAttribute("class");
        assert.ok(style.includes("rounded-md"));
        assert.ok(style.includes(`text-${tone}-700`));
        assert.equal(await badge.locator('[aria-hidden="true"]').count(), 1);
      }
    });
  }

  it("does not invent a resource status for unknown states", async () => {
    await load("labels", { state: "99", disabled: "false" });
    for (const kind of [
      "session",
      "device",
      "user",
      "policy",
      "access-policy",
    ]) {
      assert.equal(
        await page
          .locator(`[data-kind="${kind}"]`)
          .getByText(/^(Active|Pending|Rejected|Disabled)$/)
          .count(),
        0,
      );
    }
  });

  it("shares the detail status style with disabled User and Policy rows", async () => {
    await load("labels");
    for (const kind of ["user", "policy", "access-policy"]) {
      const badge = page
        .locator(`[data-kind="${kind}"]`)
        .getByText("Disabled", { exact: true });
      assert.ok((await badge.getAttribute("class")).includes("text-red-700"));
    }
    const detail = page
      .locator('[data-kind="detail"]')
      .getByText("Active", { exact: true });
    assert.ok(
      (await detail.getAttribute("class")).includes("text-emerald-700"),
    );
    assert.ok((await detail.getAttribute("class")).includes("text-xs"));
    const health = page
      .locator('[data-kind="health"]')
      .getByText("Unknown", { exact: true });
    assert.ok((await health.getAttribute("class")).includes("text-slate-500"));
    assert.equal(await health.locator('svg[aria-hidden="true"]').count(), 1);
  });

  for (const theme of ["light", "dark"]) {
    it(`matches list status labels to Connected without nested borders in ${theme} mode`, async () => {
      for (const [state, label] of [
        [1, "Active"],
        [2, "Pending"],
        [3, "Rejected"],
      ]) {
        await load("labels", { state: String(state), theme });
        const connected = await labelShape(
          page.locator('[data-kind="session"] .border').filter({
            hasText: "Connected",
          }),
        );
        assert.equal(connected.borders, 1);
        for (const kind of ["session", "device"]) {
          assert.deepEqual(
            await labelShape(
              page
                .locator(`[data-kind="${kind}"]`)
                .getByText(label, { exact: true }),
            ),
            connected,
          );
        }
        for (const kind of ["user", "policy", "access-policy"]) {
          assert.deepEqual(
            await labelShape(
              page
                .locator(`[data-kind="${kind}"]`)
                .getByText("Disabled", { exact: true }),
            ),
            connected,
          );
        }
      }
    });

    it(`keeps the SSH replay frame and header dark in ${theme} mode`, async () => {
      await load("replay", { theme, route: "/visibility/ssh/recording" });
      const heading = page.getByRole("heading", { name: "Terminal replay" });
      await heading.waitFor();
      await page
        .getByRole("button", { name: "Play replay", exact: true })
        .waitFor();
      const colors = await palette();
      const frame = heading.locator("xpath=ancestor::section[1]");
      const frameStyle = await surface(frame);
      assert.equal(frameStyle.background, colors.terminal);
      assert.equal(frameStyle.border, colors["code-border"]);
      assert.deepEqual(await surface(heading.locator("..")), {
        background: colors["terminal-panel"],
        border: colors["code-border"],
        color: colors["terminal-text"],
      });
      assert.equal(
        await frame.locator(":scope > div").evaluate(
          (element) => getComputedStyle(element).borderTopWidth,
        ),
        "0px",
      );
      for (const width of [390, 1440]) {
        await page.setViewportSize({ width, height: 900 });
        assert.equal(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= window.innerWidth,
          ),
          true,
        );
      }
    });

    for (const kind of ["access", "authentication", "audit", "component"]) {
      it(`themes the ${kind} log JSON drawer surfaces in ${theme} mode`, async () => {
        await load("json", { kind, theme });
        await page.getByRole("button", { name: "JSON", exact: true }).click();
        const dialog = page.getByRole("dialog");
        await dialog.locator(".cm-editor").waitFor();
        const colors = await palette();
        const content = await surface(dialog, "borderLeftColor");
        assert.equal(content.background, colors.white);
        assert.equal(content.border, colors["slate-200"]);
        const header = await surface(
          dialog.locator(".mantine-Drawer-header"),
          "borderBottomColor",
        );
        assert.equal(header.background, colors.white);
        assert.equal(header.border, colors["slate-200"]);
        assert.equal(
          (await surface(dialog.locator(".mantine-Drawer-body"))).background,
          colors["slate-50"],
        );
        const editorFrame = await surface(
          dialog.locator(".cm-theme").locator(".."),
        );
        assert.equal(editorFrame.background, colors.code);
        assert.equal(editorFrame.border, colors["code-border"]);
        assert.equal(
          (await surface(dialog.locator(".cm-editor"))).background,
          colors.code,
        );
        assert.equal(
          await dialog
            .getByRole("button", { name: "Copy JSON", exact: true })
            .count(),
          1,
        );
      });
    }
  }

  it("shares count formatting while preserving exact totals and generic fractional numbers", async () => {
    await load("counts");
    const result = await page.evaluate(() => {
      const {
        formatCount,
        compact,
        exact,
        formatTokens,
        formatMetricValue,
        formatNumber,
      } = window.patterns;
      const values = [0, 1500, 9999, 48000, 1000000];
      return {
        count: values.map(formatCount),
        dashboard: values.map(compact),
        tokens: values.map(formatTokens),
        metrics: values.map((value) => formatMetricValue("count", value)),
        exact: exact(48000),
        missing: [formatCount(NaN), formatCount(Infinity)],
        fractional: formatNumber(0.00001),
      };
    });
    const expected = ["0", "1,500", "9,999", "48K", "1M"];
    assert.deepEqual(result, {
      count: expected,
      dashboard: expected,
      tokens: expected,
      metrics: expected,
      exact: "48,000",
      missing: ["—", "—"],
      fractional: "0.00001",
    });
    for (const kind of ["home", "summary"]) {
      assert.equal(
        await page
          .locator(`[data-kind="${kind}"]`)
          .getByText("48K", { exact: true })
          .count(),
        1,
      );
    }
    assert.equal(
      await page
        .locator('[data-kind="llm"]')
        .getByText("48K", { exact: true })
        .count(),
      2,
    );
  });

  it("provides an accessible manual refresh without disabling busy requests", async () => {
    await load("refresh", { busy: "true" });
    const button = page.getByRole("button", { name: "Refresh", exact: true });
    assert.equal(await button.getAttribute("aria-busy"), "true");
    assert.equal(await button.isDisabled(), false);
    await button.focus();
    await page.keyboard.press("Enter");
    assert.equal((await calls("refresh")).length, 1);
    await load("refresh", { disabled: "true" });
    assert.equal(
      await page
        .getByRole("button", { name: "Refresh", exact: true })
        .isDisabled(),
      true,
    );
  });

  it("refreshes the current resource list without refetching unrelated queries", async () => {
    await load("resource");
    await page.getByRole("button", { name: "Refresh resources" }).waitFor();
    await settled();
    assert.equal((await calls("listUser")).length, 1);
    await page.evaluate(() => {
      window.patterns.state.hold = true;
    });
    const refresh = page.getByRole("button", { name: "Refresh resources" });
    await refresh.click();
    await page.waitForFunction(() => window.patterns.state.resolve);
    await refresh.and(page.locator('[aria-busy="true"]')).waitFor();
    assert.equal(await refresh.getAttribute("aria-busy"), "true");
    await page.evaluate(() => window.patterns.state.resolve());
    await settled();
    assert.equal((await calls("listUser")).length, 2);
    assert.equal((await calls("unrelated")).length, 1);
  });

  for (const theme of ["light", "dark"]) {
    for (const [resourceKind, path] of [
      ["User", "/core/users"],
      ["Service", "/core/services"],
    ]) {
      it(`dims all ${resourceKind} name parents across list layouts in ${theme} mode`, async () => {
        const names = [
          "child",
          "child.parent",
          "child.parent.grandparent",
          "child.parent.grandparent.root",
        ];
        await load("resource", {
          theme,
          resourceKind,
          route: path,
          names: names.join(","),
        });
        await page.getByRole("link", { name: names[0], exact: true }).waitFor();
        const colors = await palette();
        for (const background of [colors.white, colors["slate-50"]]) {
          assert.ok(await contrast(colors["slate-500"], background) >= 4.5);
        }
        for (const density of ["comfortable", "compact", "table"]) {
          const control = page.locator(`input[value="${density}"]`);
          await control.locator("..").click();
          assert.equal(await control.isChecked(), true);
          await page.mouse.move(0, 0);
          for (const name of names) {
            const title = page.getByRole("link", { name, exact: true });
            await title.waitFor();
            assert.equal(await title.textContent(), name);
            assert.equal(await title.getAttribute("href"), `${path}/${name}`);
            assert.equal((await surface(title)).color, colors["slate-900"]);
            assert.equal(
              await title.evaluate((element) => element.firstChild.textContent),
              "child",
            );
            const parents = title.locator("span");
            if (name === "child") {
              assert.equal(await parents.count(), 0);
            } else {
              assert.equal(await parents.textContent(), name.slice(5));
              assert.equal((await surface(parents)).color, colors["slate-500"]);
              await title.hover();
              assert.equal((await surface(parents)).color, colors["slate-500"]);
              await page.mouse.move(0, 0);
            }
          }
        }
      });
    }
  }

  for (const [kind, method, title] of [
    ["access", "listAccessLog", "No log entries found"],
    [
      "authentication",
      "listAuthenticationLog",
      "No authentication log entries found",
    ],
    ["audit", "listAuditLog", "No audit log entries found"],
    ["component", "listComponentLog", "No component log entries found"],
    ["ssh", "listSSHSession", "No SSH sessions found"],
  ]) {
    it(`uses shared empty and refresh controls for ${kind} logs without disguising load failures`, async () => {
      await load("logs", { kind });
      const empty = page.getByText(title, { exact: true });
      await empty.waitFor();
      assert.ok(
        (await empty.locator("..").getAttribute("class")).includes(
          "border-dashed",
        ),
      );
      await page.getByRole("button", { name: "Refresh", exact: true }).click();
      await settled();
      assert.equal((await calls(method)).length, 2);
      await load("logs", { kind, error: "true" });
      await page.getByRole("alert").waitFor();
      assert.equal(await page.getByText(title, { exact: true }).count(), 0);
    });
  }

  for (const [scenario, label] of [
    ["search", "Search resources"],
    ["log-search", "Search access logs"],
  ]) {
    it(`shares the 250 ms search debounce and keyboard behavior on ${scenario}`, async () => {
      await page.clock.install({ time: new Date("2026-10-13T12:00:00Z") });
      await load(scenario, {
        route: "/visibility/accesslogs?common.page=4&userRef.name=alice",
      });
      const input = page.getByRole("textbox", { name: label });
      await page.keyboard.press("/");
      assert.equal(
        await input.evaluate((element) => document.activeElement === element),
        true,
      );
      await page.clock.pauseAt(new Date("2026-10-13T13:00:00Z"));
      await input.fill(scenario === "log-search" ? "  denied  " : "denied");
      await page.clock.runFor(249);
      assert.equal((await routeParams()).has("common.query"), false);
      await page.clock.runFor(1);
      await page.waitForFunction(() =>
        document
          .querySelector("[data-route]")
          .textContent.includes("common.query=denied"),
      );
      assert.equal((await routeParams()).get("userRef.name"), "alice");
      assert.equal((await routeParams()).has("common.page"), false);
      await page.keyboard.press("Escape");
      assert.equal(
        await input.evaluate((element) => document.activeElement === element),
        false,
      );
      const other = page.getByRole("textbox", { name: "Unrelated input" });
      await other.focus();
      await page.keyboard.press("/");
      assert.equal(await other.inputValue(), "/");
      await page.getByRole("button", { name: "Clear search" }).click();
      await page.clock.runFor(250);
      await page.waitForFunction(
        () =>
          !document
            .querySelector("[data-route]")
            .textContent.includes("common.query"),
      );
    });
  }

  it("clears displayed log filters and search while preserving sort and page size", async () => {
    await load("log-search", {
      route:
        "/visibility/accesslogs?userRef.name=alice&status=DENIED&common.query=old&common.page=4&common.itemsPerPage=50&common.orderBy.type=CREATED_AT",
    });
    await page.clock.install();
    await page
      .getByRole("textbox", { name: "Search access logs" })
      .fill("pending");
    await page
      .getByRole("button", { name: "Clear filters", exact: true })
      .click();
    await page.waitForFunction(
      () =>
        !document
          .querySelector("[data-route]")
          .textContent.includes("userRef.name") &&
        document.querySelector('[aria-label="Search access logs"]').value ===
          "",
    );
    await page.clock.runFor(1000);
    assert.equal(
      (await routeParams()).toString(),
      "common.itemsPerPage=50&common.orderBy.type=CREATED_AT",
    );
    assert.equal(
      await page
        .getByRole("textbox", { name: "Search access logs" })
        .inputValue(),
      "",
    );
  });

  it("uses the same quiet empty presentation for legacy logs and LLM panels", async () => {
    await load("empty");
    for (const title of [
      "No log entries found",
      "No inference activity in this range",
    ]) {
      const text = page.getByText(title, { exact: true });
      assert.ok((await text.getAttribute("class")).includes("text-xs"));
      assert.ok(
        (await text.locator("..").getAttribute("class")).includes(
          "border-dashed",
        ),
      );
    }
  });
});
