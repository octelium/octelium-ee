import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console notifications", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-notifications-"));
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
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (scheme = "light") => {
    await page.addInitScript(
      (value) => localStorage.setItem("octelium-color-scheme", value),
      scheme,
    );
    await page.goto(`${origin}tests/fixtures/notifications.html`);
    await page.getByRole("button", { name: "Show success" }).waitFor();
  };

  const appearance = (notification) =>
    notification.evaluate((element) => {
      const style = getComputedStyle(element);
      const description = getComputedStyle(
        element.querySelector(".mantine-Notification-description"),
      );
      const probe = document.createElement("span");
      document.body.append(probe);
      const colors = Object.fromEntries(
        ["white", "slate-200", "slate-900"].map((name) => {
          probe.style.color = `var(--color-${name})`;
          return [name, getComputedStyle(probe).color];
        }),
      );
      probe.remove();
      return {
        background: style.backgroundColor,
        border: style.borderTopColor,
        text: description.color,
        accent: getComputedStyle(element, "::before").backgroundColor,
        colors,
      };
    });

  for (const scheme of ["light", "dark"]) {
    it(`uses theme colors and stays at the bottom center in ${scheme} mode`, async () => {
      await load(scheme);
      await page.getByRole("button", { name: "Show success" }).click();
      await page.getByRole("button", { name: "Show error" }).click();
      const success = page.getByRole("alert").filter({
        hasText: "Resource updated successfully",
      });
      const error = page.getByRole("alert").filter({
        hasText: "The resource could not be updated",
      });
      await error.waitFor();
      const successStyle = await appearance(success);
      const errorStyle = await appearance(error);
      for (const style of [successStyle, errorStyle]) {
        assert.equal(style.background, style.colors.white);
        assert.equal(style.border, style.colors["slate-200"]);
        assert.equal(style.text, style.colors["slate-900"]);
      }
      assert.notEqual(successStyle.accent, errorStyle.accent);
      const container = page.locator('[data-position="bottom-center"]');
      const bounds = await container.boundingBox();
      assert.ok(Math.abs(bounds.x + bounds.width / 2 - 720) < 1);
      assert.ok(Math.abs(bounds.y + bounds.height - 884) < 1);
      await success
        .getByRole("button", { name: "Dismiss notification" })
        .click();
      await success.waitFor({ state: "hidden" });
      assert.equal(await error.isVisible(), true);
    });
  }

  it("updates visible notifications when the theme changes", async () => {
    await load();
    await page.getByRole("button", { name: "Show success" }).click();
    const notification = page.getByRole("alert");
    await notification.waitFor();
    const light = await appearance(notification);
    await page.getByRole("button", { name: "Toggle theme" }).click();
    const dark = await appearance(notification);
    assert.notEqual(dark.background, light.background);
    assert.notEqual(dark.text, light.text);
    assert.equal(dark.background, dark.colors.white);
    assert.equal(dark.text, dark.colors["slate-900"]);
  });

  it("runs actions once and dismisses their notification", async () => {
    await load();
    await page.getByRole("button", { name: "Show action" }).click();
    const notification = page.getByRole("alert");
    await notification.getByRole("button", { name: "Undo" }).click();
    assert.equal(await page.getByLabel("Action count").textContent(), "1");
    await notification.waitFor({ state: "hidden" });
  });

  it("wraps long messages within a narrow viewport", async () => {
    await page.setViewportSize({ width: 375, height: 812 });
    await load("dark");
    await page.getByRole("button", { name: "Show long message" }).click();
    const notification = page.getByRole("alert");
    await notification.waitFor();
    await page.waitForFunction(
      () =>
        getComputedStyle(document.querySelector('[role="alert"]')).opacity ===
        "1",
    );
    await notification.evaluate((element) =>
      Promise.all(element.getAnimations().map((animation) => animation.finished)),
    );
    const bounds = await notification.boundingBox();
    assert.ok(bounds.x >= 0);
    assert.ok(bounds.x + bounds.width <= 375);
    assert.ok(bounds.height > 80);
    assert.equal(
      await notification.evaluate((element) => {
        const description = element.querySelector(
          ".mantine-Notification-description",
        );
        return (
          description.scrollWidth <= description.clientWidth &&
          element.scrollHeight <= element.clientHeight
        );
      }),
      true,
    );
  });

  it("automatically dismisses notifications after four seconds", async () => {
    await load();
    await page.clock.install();
    await page.getByRole("button", { name: "Show temporary" }).click();
    const notification = page.getByRole("alert");
    await notification.waitFor();
    await page.clock.runFor(3900);
    assert.equal(await notification.isVisible(), true);
    await page.clock.runFor(600);
    await notification.waitFor({ state: "hidden" });
  });
});
