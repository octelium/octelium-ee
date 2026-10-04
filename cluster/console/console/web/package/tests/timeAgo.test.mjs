import assert from "node:assert/strict";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("TimeAgo", () => {
  let server;
  let browser;
  let context;
  let page;
  let origin;

  before(async () => {
    server = await createServer({
      root,
      configFile: false,
      resolve: { alias: { "@": `${root}src` } },
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
    context = await browser.newContext({ timezoneId: "UTC" });
    page = await context.newPage();
    await page.goto(`${origin}tests/fixtures/timeAgo.html`);
    await page.locator('[data-row="0"] > span').last().waitFor();
  });

  afterEach(async () => {
    await context?.close();
  });

  it("shows the absolute timestamp and dismisses after a normal hover", async () => {
    await page.locator('[data-row="0"] > span').last().hover();
    await page.getByRole("tooltip").waitFor();
    assert.equal(
      await page.getByRole("tooltip").textContent(),
      "10:00:00 AM, Sun Oct 4, 2026",
    );
    await page.waitForTimeout(250);
    await page.mouse.move(700, 650);
    await page.waitForTimeout(400);
    assert.equal(await page.getByRole("tooltip").count(), 0);
  });

  it("dismisses when the pointer leaves during the enter animation", async () => {
    for (let index = 0; index < 8; index++) {
      await page.locator(`[data-row="${index}"] > span`).last().hover();
      await page.mouse.move(700, 650);
      await page.waitForTimeout(400);
      assert.equal(await page.getByRole("tooltip").count(), 0);
    }
  });

  it("dismisses after rapid movement between timestamps and list refreshes", async () => {
    for (let index = 0; index < 32; index++) {
      const timestamp = page.locator(`[data-row="${index % 8}"] > span`).last();
      const box = await timestamp.boundingBox();
      await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
      if (index % 4 === 0) {
        await page.getByRole("button", { name: "Refresh" }).evaluate((button) => {
          button.click();
        });
      }
    }
    await page.mouse.move(700, 650);
    await page.waitForTimeout(400);
    assert.equal(await page.getByRole("tooltip").count(), 0);
  });

  it("dismisses after the shared clock refreshes while hovering", async () => {
    await page.clock.install();
    await page.locator('[data-row="0"] > span').last().hover();
    await page.clock.fastForward(30000);
    await page.mouse.move(700, 650);
    await page.clock.runFor(500);
    assert.equal(await page.getByRole("tooltip").count(), 0);
  });
});
