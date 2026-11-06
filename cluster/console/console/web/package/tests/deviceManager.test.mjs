import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("DeviceManager pages", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-manager-"));
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

  const open = (path, extra = {}) =>
    page.goto(
      `${origin}tests/fixtures/deviceManager.html?${new URLSearchParams({ path, ...extra })}`,
    );
  const load = async (path, extra = {}) => {
    await open(path, extra);
    await page.getByText("Provider", { exact: true }).first().waitFor();
  };
  const value = () => page.evaluate(() => window.manager.value);
  const choose = (group, name) =>
    page
      .getByRole("radiogroup", { name: group })
      .getByRole("radio", { name: new RegExp(`^${name}`) })
      .click();
  const pickProvider = async (query) => {
    await page.getByRole("button", { name: /Change the provider/ }).click();
    await page.getByRole("textbox", { name: "Search providers" }).fill(query);
    await page.keyboard.press("Enter");
  };
  const overflow = () =>
    page.evaluate(() => {
      const root =
        document.querySelector("#resource-item-drawer .mantine-Drawer-body") ??
        document.body;
      const right = root.getBoundingClientRect().right;
      return [...root.querySelectorAll("*")]
        .filter((el) => {
          const rect = el.getBoundingClientRect();
          return rect.width > 0 && rect.right > right + 1;
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
      await load("/enterprise/devicemanagers/crowdstrike/edit", { theme });
      await page.getByRole("button", { name: /Advanced options/ }).click();
      await page.getByLabel("Host filter").waitFor();
      assert.deepEqual(await overflow(), []);
    });
  }

  it("creates a CrowdStrike manager by default", async () => {
    await load("/enterprise/devicemanagers/create");
    assert.ok(
      await page.getByRole("button", { name: /CrowdStrike Falcon/ }).count(),
    );
    await page.getByLabel("Client ID").waitFor();
  });

  it("searches the providers by vendor keywords and keeps their settings", async () => {
    await load("/edit-only", { name: "crowdstrike" });
    await page.getByRole("button", { name: /Advanced options/ }).click();
    await page.getByLabel("Host filter").fill("platform_name:'Mac'");

    await pickProvider("kolide");
    await page
      .getByRole("button", { name: /1Password Device Trust/ })
      .waitFor();
    assert.ok((await value()).spec.onePassword);
    assert.equal((await value()).spec.crowdStrike, undefined);

    await pickProvider("falcon");
    assert.equal(
      (await value()).spec.crowdStrike.hostFilter,
      "platform_name:'Mac'",
    );
    assert.equal(
      (await value()).spec.crowdStrike.clientSecret.fromSecret,
      "falcon-client-secret",
    );
  });

  it("drops the linking options that the next provider does not support", async () => {
    await load("/edit-only", { name: "intune" });
    await page.getByText("Verification interval").first().waitFor();

    await pickProvider("crowdstrike");
    await page.getByText("CrowdStrike Falcon reports no owners").waitFor();
    assert.deepEqual((await value()).spec.linking.requireOwnerMatch, undefined);
    assert.ok((await value()).spec.linking.verificationInterval);

    await pickProvider("huntress");
    await page.getByText("Huntress publishes no Probes").waitFor();
    assert.equal((await value()).spec.linking, undefined);
  });

  it("offers the owner match only to the providers that report owners", async () => {
    await load("/edit-only", { name: "fleet" });
    await page.getByText("Fleet reports no owners").waitFor();
    await pickProvider("jamf");
    await page.getByLabel("Require an owner match").click({ force: true });
    assert.equal((await value()).spec.linking.requireOwnerMatch, true);
    await page.getByLabel("Require an owner match").click({ force: true });
    assert.equal((await value()).spec.linking, undefined);
  });

  it("requires plain HTTPS provider URLs", async () => {
    await load("/edit-only", { name: "fleet" });
    const url = page.getByLabel("Server URL");
    await url.fill("http://fleet.example.com");
    await page.getByText("Use an https:// URL").waitFor();
    await url.fill("https://fleet.example.com?a=b");
    await page.getByText("Remove the query and the fragment").waitFor();
    await url.fill("https://user@fleet.example.com");
    await page.getByText("Enter a host without credentials").waitFor();
    await url.fill("https://fleet.example.com");
    assert.equal(await page.getByText("Use an https:// URL").count(), 0);
  });

  it("disables polling without keeping an empty polling configuration", async () => {
    await load("/edit-only", { name: "fleet" });
    await page
      .getByLabel("Collect the inventory periodically")
      .click({ force: true });
    assert.equal((await value()).spec.polling.isDisabled, true);
    await page
      .getByLabel("Collect the inventory periodically")
      .click({ force: true });
    assert.equal((await value()).spec.polling, undefined);
  });

  it("warns about an inventory that goes stale before the next collection", async () => {
    await load("/edit-only", { name: "jamf" });
    await page
      .getByText("Stale after must be longer than the interval")
      .waitFor();
    await load("/edit-only", { name: "fleet" });
    assert.equal(
      await page
        .getByText("Stale after must be longer than the interval")
        .count(),
      0,
    );
  });

  it("keeps the SentinelOne agent filters as a map without reserved keys", async () => {
    await load("/edit-only", { name: "sentinelone" });
    await page.getByRole("button", { name: /Advanced options/ }).click();
    await page.getByRole("button", { name: "Add filter" }).click();
    await page.getByLabel("Agent filters 2 key").fill("cursor");
    await page.getByText("Reserved").waitFor();
    await page.getByLabel("Agent filters 2 key").fill("isActive");
    await page.getByLabel("Agent filters 2 value").fill("true");
    assert.deepEqual((await value()).spec.sentinelOne.agentFilters, {
      isDecommissioned: "false",
      isActive: "true",
    });
  });

  it("maps the Fleet team scope to the optional team ID", async () => {
    await load("/edit-only", { name: "fleet" });
    await choose("Hosts", "No team");
    assert.equal((await value()).spec.fleetDM.teamID, 0);
    await choose("Hosts", "A single team");
    await page.getByLabel("Team ID").fill("7");
    assert.equal((await value()).spec.fleetDM.teamID, 7);
    await choose("Hosts", "Every team");
    assert.equal((await value()).spec.fleetDM.teamID, undefined);
  });

  it("keeps only valid Huntress organization IDs", async () => {
    await load("/edit-only", { name: "huntress" });
    await page.getByRole("button", { name: /Advanced options/ }).click();
    const field = page.getByRole("combobox", { name: "Organization IDs" });
    await field.fill("2048");
    await page.keyboard.press("Enter");
    await field.fill("acme");
    await page.keyboard.press("Enter");
    assert.deepEqual((await value()).spec.huntress.organizationIDs, [
      "1042",
      "2048",
    ]);
  });

  it("restricts the manager to a condition only on request", async () => {
    await load("/edit-only", { name: "fleet" });
    await choose("Devices", "Devices that match a condition");
    assert.deepEqual((await value()).spec.condition, { match: "" });
    await choose("Devices", "Every Device");
    assert.equal((await value()).spec.condition, undefined);
  });

  it("saves the edited manager from the item drawer", async () => {
    await load("/enterprise/devicemanagers/fleet/edit");
    await page.getByLabel("Server URL").fill("https://fleet2.example.com");
    await page.getByRole("button", { name: "Save changes" }).click();
    await page.waitForFunction(() =>
      window.manager.calls.some(
        (call) => call.method === "updateDeviceManager",
      ),
    );
    const call = await page.evaluate(() =>
      window.manager.calls.find(
        (item) => item.method === "updateDeviceManager",
      ),
    );
    assert.equal(
      call.request.spec.type.fleetDM.baseURL,
      "https://fleet2.example.com",
    );
    assert.equal(
      call.request.spec.type.fleetDM.apiToken.type.fromSecret,
      "fleet-token",
    );
  });

  it("summarizes the managers and links their issues to the Devices", async () => {
    await open("/list-only");
    const link = (name) =>
      page.getByRole("link", { name: new RegExp(`${name}$`) }).first();
    await link("CrowdStrike Falcon").waitFor();
    assert.equal(
      await link("CrowdStrike Falcon").getAttribute("href"),
      "/enterprise/devicemanagers?type=CROWDSTRIKE",
    );
    assert.equal(
      await link("Conflicts").getAttribute("href"),
      "/core/devices?bindingState=CONFLICT",
    );
    assert.equal(
      await link("Suspended").getAttribute("href"),
      "/core/devices?hasInvalidBinding=true",
    );
    assert.equal(
      await link("Polling disabled").getAttribute("href"),
      "/enterprise/devicemanagers?isPollingDisabled=true",
    );
    assert.equal(
      await page.getByText("1Password Device Trust", { exact: true }).count(),
      0,
    );

    const label = page.getByTestId("crowdstrike");
    await label.getByText("Healthy").waitFor();
    await label.getByText("1,240").waitFor();
    await label.getByText("1,180").waitFor();
    await label.getByText("3", { exact: true }).waitFor();
  });

  it("draws a mark for every provider", async () => {
    await open("/list-only");
    await page.getByTestId("tiles").locator("svg path").first().waitFor();
    assert.equal(await page.getByTestId("tiles").locator("svg").count(), 8);
  });

  it("orders the managers of the ClusterConfig", async () => {
    await open("/order-only");
    await page.getByRole("button", { name: "Move crowdstrike up" }).click();
    assert.deepEqual(await page.evaluate(() => window.manager.order), [
      "crowdstrike",
      "intune",
      "fleet",
    ]);
    assert.equal(
      await page
        .getByRole("button", { name: "Move crowdstrike up" })
        .isDisabled(),
      true,
    );
    await page.getByRole("button", { name: "Remove intune" }).click();
    assert.deepEqual(await page.evaluate(() => window.manager.order), [
      "crowdstrike",
      "fleet",
    ]);
  });
});
