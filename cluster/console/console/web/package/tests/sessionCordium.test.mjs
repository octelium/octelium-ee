import assert from "node:assert/strict";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Session Cordium information", () => {
  let server;
  let browser;
  let context;
  let page;
  let origin;

  const loadScenario = async (scenario = "workspace") => {
    await page.goto(
      `${origin}tests/fixtures/sessionCordium.html?scenario=${scenario}`,
    );
    await page.getByText("Security signals", { exact: true }).waitFor();
  };

  const resourceLink = (kind, name) =>
    page.getByRole("link", { name: `${kind}${name}`, exact: true });

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
  });

  afterEach(async () => {
    await context?.close();
  });

  it("shows Cordium references, Space type, User and Device", async () => {
    await loadScenario();
    for (const [kind, name, path] of [
      ["Workspace", "aoo", "/cordium/workspaces/aoo"],
      ["Space", "default.usr1", "/cordium/spaces/default.usr1"],
      ["Template", "default.default.usr1", "/cordium/templates/default.default.usr1"],
      ["User", "usr1", "/core/users/usr1"],
      ["Device", "dev1", "/core/devices/dev1"],
    ]) {
      assert.equal(
        await resourceLink(kind, name).getAttribute("href"),
        path,
      );
    }
    assert.equal(await page.getByText("Cordium", { exact: true }).count(), 1);
    assert.equal(await page.getByText("Space type", { exact: true }).count(), 1);
    assert.equal(await page.getByText("Personal", { exact: true }).count(), 1);
  });

  it("navigates to a Cordium resource with a return link to the Session", async () => {
    await loadScenario();
    await resourceLink("Workspace", "aoo").click();
    await page.getByText("Reference", { exact: true }).waitFor();
    assert.equal(new URL(page.url()).pathname, "/cordium/workspaces/aoo");
    assert.equal(
      await page.evaluate(() => window.history.state.usr.returnTo),
      "/core/sessions/usr1-fvhdnl",
    );
  });

  it("handles numeric Space types and future extension fields", async () => {
    await loadScenario("organization");
    assert.equal(await page.getByText("Organization", { exact: true }).count(), 1);
    assert.equal(
      await resourceLink("Workspace", "aoo").getAttribute("href"),
      "/cordium/workspaces/aoo",
    );
  });

  it("links partial references and omits empty references and Space types", async () => {
    await loadScenario("partial");
    assert.equal(
      await resourceLink("Workspace", "aoo").getAttribute("href"),
      "/cordium/workspaces/aoo",
    );
    for (const label of ["Space", "Template", "Space type"]) {
      assert.equal(await page.getByText(label, { exact: true }).count(), 0);
    }
  });

  it("shows UID-only references without a broken link", async () => {
    await loadScenario("uid");
    assert.equal(
      await page
        .getByText("69dd3308-1a9c-4cee-ac37-0e6743c0c7bc", { exact: true })
        .count(),
      1,
    );
    assert.equal(await page.locator('a[href^="/cordium/"]').count(), 0);
  });

  for (const scenario of [
    "normal",
    "enterprise",
    "empty",
    "malformed",
    "no-status",
  ]) {
    it(`omits Cordium information for ${scenario} Session metadata`, async () => {
      await loadScenario(scenario);
      assert.equal(await page.getByText("Cordium", { exact: true }).count(), 0);
      assert.equal(await page.locator('a[href^="/cordium/"]').count(), 0);
      assert.equal(await page.getByText("Space type", { exact: true }).count(), 0);
    });
  }
});
