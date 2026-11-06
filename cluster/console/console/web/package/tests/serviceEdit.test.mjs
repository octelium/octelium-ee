import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Service edit", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-service-edit-"));
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
      reducedMotion: "reduce",
      viewport: { width: 1440, height: 1000 },
    });
    page = await context.newPage();
    page.setDefaultTimeout(30_000);
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => {
      if (message.type() === "error") errors.push(message.text());
    });
  });
  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (path, extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/serviceEdit.html?${new URLSearchParams({ path, ...extra })}`,
    );
    await page.getByText("Mode", { exact: true }).first().waitFor();
  };
  const tabs = () =>
    page.getByRole("tab").evaluateAll((items) =>
      items.map((item) => item.textContent.replace(/\d+$/, "")),
    );
  const overflow = () =>
    page.evaluate(() => {
      const root =
        document.querySelector("#resource-item-drawer .mantine-Drawer-body") ??
        document.body;
      const right = root.getBoundingClientRect().right;
      return [...root.querySelectorAll("*")]
        .filter((el) => {
          const rect = el.getBoundingClientRect();
          if (rect.width === 0 || rect.right <= right + 1) return false;
          return !el.closest('[role="tablist"]');
        })
        .map((el) => el.tagName);
    });

  for (const [theme, width] of [
    ["light", 1440],
    ["dark", 1440],
    ["light", 390],
  ]) {
    it(`fits the item drawer in the ${theme} theme at ${width}px`, async () => {
      await page.setViewportSize({ width, height: 1000 });
      await load("/core/services/web-app/edit", { theme });
      await page
        .getByRole("button", { name: "Environment variables", exact: true })
        .click();
      await page.getByRole("textbox", { name: "Variable 1 name" }).waitFor();
      assert.deepEqual(await overflow(), []);
      assert.equal(
        await page.evaluate(() => document.documentElement.scrollWidth),
        width,
      );
    });
  }

  it("offers upstream TLS only in the modes that use it", async () => {
    await load("/edit-only", { name: "prod-ssh" });
    assert.equal(await page.getByText("Upstream TLS").count(), 0);
    await load("/edit-only", { name: "k8s-cluster" });
    assert.equal(await page.getByText("Upstream TLS").count(), 0);
    await load("/edit-only", { name: "mysql-main" });
    assert.equal(await page.getByText("Upstream TLS").count(), 0);
    await load("/edit-only", { name: "tcp-raw" });
    await page.getByText("Upstream TLS").waitFor();
  });

  it("shows a protocol section only for modes with their own settings", async () => {
    await load("/edit-only", { name: "tcp-raw" });
    assert.deepEqual(await tabs(), [
      "Upstream",
      "Access",
      "Dynamic",
      "Settings",
    ]);
    await load("/edit-only", { name: "prod-ssh" });
    assert.ok((await tabs()).includes("SSH"));
  });

  it("offers the Kubernetes client certificate stored in the Service", async () => {
    await load("/edit-only", { name: "k8s-cluster" });
    await page.getByRole("tab", { name: "Kubernetes" }).click();
    assert.equal(
      await page
        .getByRole("radio", { name: /^Client certificate/ })
        .getAttribute("aria-checked"),
      "true",
    );
  });

  it("resets the exposure when the new mode cannot be public", async () => {
    await load("/edit-only", { name: "api-gateway" });
    await page.getByRole("button", { name: /Change the mode/ }).click();
    await page.getByRole("textbox", { name: "Search modes" }).fill("ssh");
    await page.keyboard.press("Enter");
    await page.getByRole("tab", { name: "SSH" }).waitFor();
    const value = await page.evaluate(() => window.serviceEdit.value);
    assert.equal(value.spec.mode, "SSH");
    assert.equal(value.spec.isPublic, undefined);
    assert.equal(value.spec.isTLS, undefined);
    assert.equal(value.spec.config.tls, undefined);
  });

  it("only offers public access without anonymous for Kubernetes", async () => {
    await load("/edit-only", { name: "k8s-cluster" });
    await page.getByRole("tab", { name: "Access" }).click();
    await page.getByRole("radio", { name: /^Public/ }).waitFor();
    assert.equal(await page.getByRole("radio", { name: /^Anonymous/ }).count(), 0);
  });
});
