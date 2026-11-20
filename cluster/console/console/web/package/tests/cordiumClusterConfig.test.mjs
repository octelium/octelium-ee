import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import yaml from "js-yaml";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));
const management = "octelium.api.main.cordium.v1.ManagementService";

describe("Cordium ClusterConfig", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-cordium-config-"));
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
      reducedMotion: "reduce",
      viewport: { width: 1440, height: 1000 },
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

  const navigate = (extra = {}) => page.goto(
    `${origin}tests/fixtures/cordiumClusterConfig.html?${new URLSearchParams(extra)}`,
  );
  const load = async (extra) => {
    await navigate(extra);
    await page.getByRole("button", { name: "Save changes", exact: true }).waitFor();
  };
  const section = (title) => page.locator("section").filter({ has: page.getByText(title, { exact: true }) }).last();
  const configure = (title) => section(title).getByRole("button").first().click();
  const save = async () => {
    await page.getByRole("button", { name: "Save changes", exact: true }).click();
    await page.getByText("Cordium ClusterConfig successfully updated", { exact: true }).last().waitFor();
    await page.getByText("No changes", { exact: true }).waitFor();
    return page.evaluate(() => window.cordiumConfig.saved);
  };

  it("loads the singleton through the management service and leaves the form clean", async () => {
    await load();
    assert.equal(await page.getByRole("button", { name: "Save changes", exact: true }).isDisabled(), true);
    assert.equal(await page.getByText("Metadata", { exact: true }).count(), 0);
    const config = await page.evaluate(() => window.cordiumConfig.get());
    assert.equal(config.kind, "ClusterConfig");
    assert.equal(config.spec.agent.llm.service, "llm.default");
    const calls = await page.evaluate(() => window.cordiumConfig.calls);
    assert.ok(calls.length >= 2);
    assert.ok(calls.every((call) => call.method === "getClusterConfig" && call.service === management));
    assert.ok(calls.every((call) => Object.keys(call.request).length === 0));
  });

  it("saves all configuration areas while preserving nested and unset fields", async () => {
    await load();
    await page.getByLabel("Maximum Workspaces per User", { exact: true }).fill("12");
    await page.getByLabel("Maximum Volumes per Space", { exact: true }).fill("9");
    await section("Personal Space defaults").getByLabel("Memory (MB)", { exact: true }).fill("2048");
    await section("Maximum Workspace resources").getByLabel("Memory (MB)", { exact: true }).fill("");
    await page.getByRole("switch", { name: /^Allow no inactivity timeout/ }).check();
    await page.getByLabel("Model", { exact: true }).fill("custom-model");
    await page.getByLabel("Additional configuration (JSON)", { exact: true }).fill('{"theme":"dark","tools":{"terminal":true},"retries":3}');
    const saved = await save();
    assert.equal(saved.spec.workspace.limit.maxPerUser, 12);
    assert.equal(saved.spec.volume.limit.maxPerSpace, 9);
    assert.deepEqual(saved.spec.workspace.limit.defaultUserSpaceLimit, { cpu: { millicores: 1000 }, memory: { megabytes: 2048 } });
    assert.deepEqual(saved.spec.workspace.limit.maxLimit, { cpu: { millicores: 8000 } });
    assert.equal(saved.spec.workspace.timeout.allowNoTimeout, true);
    assert.deepEqual(saved.spec.workspace.timeout.defaultDuration, { minutes: 30 });
    assert.deepEqual(saved.spec.workspace.runtime.capabilities, { add: ["NET_ADMIN"], drop: ["ALL"] });
    assert.deepEqual(saved.spec.agent.config, { theme: "dark", tools: { terminal: true }, retries: 3 });
    assert.equal(saved.spec.agent.llm.model, "custom-model");
    assert.deepEqual(saved.spec.space.ownership.rules[0], { effect: "ALLOW", condition: { matchAny: true } });
    assert.equal(await page.getByTestId("location").textContent(), "/cordium/clusterconfig");
    const calls = await page.evaluate(() => window.cordiumConfig.calls);
    assert.ok(calls.every((call) => call.service === management));
    assert.ok(calls.filter((call) => call.method === "getClusterConfig").length >= 2);
    assert.equal(calls.filter((call) => call.method === "updateClusterConfig").length, 1);
  });

  it("adds, reorders and removes rules without mixing their conditions", async () => {
    await load();
    const storage = section("Workspace storage class rules");
    await storage.getByRole("button", { name: "Move rule 2 up", exact: true }).click();
    assert.equal(await storage.getByLabel(/^Storage class/).first().inputValue(), "standard");
    await storage.getByRole("button", { name: "Remove rule 2", exact: true }).click();
    await storage.getByRole("button", { name: "Add", exact: true }).click();
    await storage.getByLabel(/^Storage class/).last().fill("archive");
    await storage.getByRole("button", { name: "Match every request", exact: true }).last().click();
    const ownership = section("Space ownership rules");
    await ownership.getByRole("button", { name: "Add", exact: true }).click();
    await ownership.getByLabel(/^Effect/).last().click();
    await page.getByRole("option", { name: "Deny", exact: true }).click();
    await ownership.getByRole("button", { name: "Match every request", exact: true }).last().click();
    const saved = await save();
    assert.deepEqual(saved.spec.workspace.storage.storageClass.rules, [
      { storageClass: "standard", condition: { matchAny: true } },
      { storageClass: "archive", condition: { matchAny: true } },
    ]);
    assert.deepEqual(saved.spec.space.ownership.rules[1], { effect: "DENY", condition: { matchAny: true } });
  });

  it("configures and removes optional sections from an empty singleton", async () => {
    await load({ scenario: "empty" });
    for (const title of ["Spaces", "Workspaces", "Volumes", "AI agent"]) await configure(title);
    await configure("Workspace limits");
    await page.getByLabel("Maximum Workspaces per User", { exact: true }).fill("7");
    await configure("Default LLM");
    await page.getByLabel("LLM Service", { exact: true }).fill("llm.custom");
    const saved = await save();
    assert.equal(saved.spec.workspace.limit.maxPerUser, 7);
    assert.equal(saved.spec.agent.llm.service, "llm.custom");
    for (const title of ["Spaces", "Workspaces", "Volumes", "AI agent"]) {
      await page.getByRole("button", { name: `Remove ${title}`, exact: true }).click();
    }
    assert.deepEqual((await save()).spec, {});
  });

  it("keeps form and YAML edits in sync and saves YAML through management", async () => {
    await load();
    await page.getByLabel("Maximum Workspaces per User", { exact: true }).fill("14");
    await page.getByText("YAML", { exact: true }).locator("xpath=ancestor::label").click();
    const editor = page.locator(".cm-content");
    await editor.waitFor();
    assert.match(await editor.innerText(), /apiVersion: cordium\/v1/);
    await page.getByText("Configuration", { exact: true }).locator("xpath=ancestor::label").click();
    assert.equal(await page.getByLabel("Maximum Workspaces per User", { exact: true }).inputValue(), "14");
    await page.getByText("YAML", { exact: true }).locator("xpath=ancestor::label").click();
    const config = await page.evaluate(() => window.cordiumConfig.get());
    config.spec.workspace.limit.maxPerUser = 16;
    config.spec.agent.config.tools = { terminal: false, nested: [1, 2, 3] };
    await editor.fill(yaml.dump(config));
    await page.getByText("Configuration", { exact: true }).locator("xpath=ancestor::label").click();
    assert.equal(await page.getByLabel("Maximum Workspaces per User", { exact: true }).inputValue(), "16");
    await page.getByText("YAML", { exact: true }).locator("xpath=ancestor::label").click();
    const saved = await save();
    assert.equal(saved.spec.workspace.limit.maxPerUser, 16);
    assert.deepEqual(saved.spec.agent.config.tools, { terminal: false, nested: [1, 2, 3] });
  });

  it("edits supported agent images and restores settings when switching sources", async () => {
    await load();
    const choose = async (label, option) => {
      await page.getByRole("combobox", { name: label, exact: true }).click();
      await page.getByRole("option", { name: option, exact: true }).click();
    };
    await choose("Image source", "Git repository");
    await page.getByLabel(/^Git repository URL/).fill("https://github.com/example/agent.git");
    await page.getByLabel("Git checkout", { exact: true }).fill("stable");
    await choose("Image source", "Container registry");
    assert.equal(await page.getByLabel(/^Container image/).inputValue(), "ghcr.io/example/agent:latest");
    await choose("Image source", "Dockerfile");
    await page.getByLabel(/^Dockerfile\s*\*?$/).fill("FROM node:24");
    await choose("Dockerfile source", "HTTPS URL");
    await page.getByLabel(/^Dockerfile URL/).fill("https://example.com/Dockerfile");
    await choose("Dockerfile source", "Inline");
    assert.equal(await page.getByLabel(/^Dockerfile\s*\*?$/).inputValue(), "FROM node:24");
    await choose("Image source", "Git repository");
    assert.equal(await page.getByLabel("Git checkout", { exact: true }).inputValue(), "stable");
    const saved = await save();
    assert.deepEqual(saved.spec.agent.image, { git: { url: "https://github.com/example/agent.git", checkout: "stable" } });
  });

  it("protects unsaved configuration when navigating away", async () => {
    await load();
    await page.getByLabel("Agent version", { exact: true }).fill("next");
    await page.getByRole("link", { name: "Leave configuration", exact: true }).click();
    await page.getByText("Discard unsaved changes?", { exact: true }).waitFor();
    await page.getByRole("button", { name: "Keep editing", exact: true }).click();
    assert.equal(await page.getByLabel("Agent version", { exact: true }).inputValue(), "next");
    await page.getByRole("link", { name: "Leave configuration", exact: true }).click();
    await page.getByRole("button", { name: "Discard changes", exact: true }).click();
    await page.getByText("Another page", { exact: true }).waitFor();
  });

  it("shows loading before the configuration arrives", async () => {
    await navigate({ scenario: "loading" });
    await page.locator(".mantine-Loader-root").waitFor();
    await page.waitForFunction(() => !!window.cordiumConfig?.resolve);
    assert.equal(await page.getByRole("button", { name: "Save changes", exact: true }).count(), 0);
    await page.evaluate(() => window.cordiumConfig.resolve());
    await page.getByRole("button", { name: "Save changes", exact: true }).waitFor();
  });

  it("shows read failures and retries the management request", async () => {
    await navigate({ scenario: "load-error" });
    await page.getByText("Could not load the Cordium cluster configuration", { exact: true }).waitFor();
    await page.getByText("Details", { exact: true }).click();
    await page.getByText("Configuration access denied", { exact: true }).waitFor();
    await page.getByRole("button", { name: "Retry", exact: true }).click();
    await page.getByRole("button", { name: "Save changes", exact: true }).waitFor();
  });

  it("handles an unavailable singleton", async () => {
    await navigate({ scenario: "unavailable" });
    await page.getByText("The Cordium cluster configuration is unavailable.", { exact: true }).waitFor();
  });

  it("keeps unsaved changes after an update is rejected", async () => {
    await load({ scenario: "save-error" });
    await page.getByLabel("Agent version", { exact: true }).fill("next");
    await page.getByRole("button", { name: "Save changes", exact: true }).click();
    await page.getByText("Update failed — check the form and try again.", { exact: true }).waitFor();
    assert.equal(await page.getByLabel("Agent version", { exact: true }).inputValue(), "next");
    assert.equal(await page.getByRole("button", { name: "Save changes", exact: true }).isEnabled(), true);
    assert.equal(await page.evaluate(() => window.cordiumConfig.saved), undefined);
  });

  for (const theme of ["light", "dark"]) {
    for (const width of [1440, 390, 320]) {
      it(`fits the ${theme} theme at ${width}px with readable text`, async () => {
        await page.setViewportSize({ width, height: 1000 });
        await load({ theme });
        assert.equal(await page.evaluate(() => document.documentElement.scrollWidth), width);
        const overflow = await page.locator("main").evaluate((main) => [...main.querySelectorAll("input, textarea, section, button")]
          .filter((element) => {
            const rect = element.getBoundingClientRect();
            return rect.width > 0 && (rect.left < -1 || rect.right > window.innerWidth + 1);
          })
          .map((element) => element.getAttribute("aria-label") || element.textContent));
        assert.deepEqual(overflow, []);
        const colors = await page.getByText("Spaces", { exact: true }).evaluate((element) => {
          const canvas = document.createElement("canvas");
          const ctx = canvas.getContext("2d");
          const rgb = (color) => {
            ctx.fillStyle = color;
            ctx.fillRect(0, 0, 1, 1);
            return [...ctx.getImageData(0, 0, 1, 1).data].slice(0, 3);
          };
          return { text: rgb(getComputedStyle(element).color), background: rgb(getComputedStyle(element.closest("section")).backgroundColor) };
        });
        const luminance = (rgb) => rgb.map((value) => {
          const s = value / 255;
          return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
        }).reduce((sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index], 0);
        const text = luminance(colors.text);
        const background = luminance(colors.background);
        assert.ok((Math.max(text, background) + 0.05) / (Math.min(text, background) + 0.05) >= 4.5);
        assert.equal(background < text, theme === "dark");
      });
    }
  }
});
