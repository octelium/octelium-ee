import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Access pages", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-access-"));
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
  });
  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (path, extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/accessPages.html?${new URLSearchParams({ path, ...extra })}`,
    );
  };
  const policy = () => page.evaluate(() => window.access.policy);
  const rule = (index = 0) => policy().then((value) => value.spec.rules[index]);
  const effect = (name) =>
    page.getByRole("radiogroup", { name: "Effect" }).getByRole("radio", { name: new RegExp(`^${name}`) });

  it("keeps the review workflow and grants while switching a rule to Deny and back", async () => {
    await load("/policy-edit");
    await effect("Deny").click();
    let value = await rule();
    assert.equal(value.effect, "DENY");
    assert.equal(value.action, undefined);
    assert.equal(value.authorization, undefined);
    await page.getByText("Matching requests are rejected as soon as they are created.", { exact: false }).waitFor();
    assert.equal(await page.getByText("Review steps", { exact: true }).count(), 0);

    await effect("Require review").click();
    value = await rule();
    assert.equal(value.effect, "REVIEW");
    assert.deepEqual(
      value.action.review.steps.map((item) => item.name),
      ["team-lead", "security"],
    );
    assert.deepEqual(value.authorization.policies, ["prod-read-only"]);

    await effect("Auto-approve").click();
    value = await rule();
    assert.equal(value.action, undefined);
    assert.deepEqual(value.authorization.policies, ["prod-read-only"]);
  });

  it("adds rules from templates with unique names and the expected priority", async () => {
    await load("/policy-edit");
    await page.getByRole("button", { name: /^Deny guardrail/ }).click();
    await page.getByRole("button", { name: /^Deny guardrail/ }).click();
    const value = await policy();
    assert.deepEqual(
      value.spec.rules.map((item) => [item.name, item.effect, item.priority ?? 0]),
      [
        ["production-review", "REVIEW", 0],
        ["deny", "DENY", -4],
        ["deny-2", "DENY", -4],
      ],
    );
  });

  it("lists what must be fixed before a new rule can be saved", async () => {
    await load("/policy-edit");
    await page.getByRole("button", { name: /^Two-step review/ }).click();
    const card = page.locator("article").filter({ hasText: "two-step-review" });
    await card.getByText("Before saving", { exact: true }).waitFor();
    for (const issue of [
      "Choose the Catalog",
      "team-lead: add at least one reviewer",
      "security: add at least one reviewer",
    ]) {
      await card.getByText(issue, { exact: true }).first().waitFor();
    }
    await card.getByText(/\d+ to fix/).waitFor();
  });

  it("does not offer skipping past the last review step on timeout", async () => {
    await load("/policy-edit");
    const card = page.locator("article").first();
    const lastStep = card.locator("li").filter({ has: page.getByRole("textbox", { name: "Step name" }) }).last();
    await lastStep.getByText("Reject", { exact: true }).waitFor();
    assert.equal(await lastStep.getByText("Skip to the next step", { exact: true }).count(), 0);
    const firstStep = card.locator("li").filter({ has: page.getByRole("textbox", { name: "Step name" }) }).first();
    assert.equal(await firstStep.getByText("Skip to the next step", { exact: true }).count(), 1);
  });

  it("explains what a pending request waits on and who decided the earlier step", async () => {
    await load("/access/requests/req-pending");
    await page.getByText("Waiting on security", { exact: false }).first().waitFor();
    await page.getByText("step 2 of 2", { exact: false }).first().waitFor();
    const calls = await page.evaluate(() => window.access.calls);
    assert.ok(
      calls.some((call) => call.method === "listReview" && call.request.requestRef?.name === "req-pending"),
    );
    await page.getByText("“Matches the incident.”", { exact: true }).waitFor();
    await page.getByText("carol", { exact: true }).first().waitFor();
  });

  it("explains a request rejected because no rule matched", async () => {
    await load("/access/requests/req-unmatched");
    await page.getByText("Rejected · no matching rule", { exact: true }).waitFor();
    await page.getByText("None: no rule of an enabled Policy matched", { exact: true }).waitFor();
  });

  it("shows the rules that name a Catalog, including nested conditions", async () => {
    await load("/access/catalogs/production-support");
    await page.getByText("oncall-auto-approve", { exact: true }).waitFor();
    await page.getByText("production-review", { exact: true }).waitFor();
    assert.equal(await page.getByText("grafana-review", { exact: true }).count(), 0);
    const calls = await page.evaluate(() => window.access.calls);
    assert.ok(
      calls.some((call) => call.method === "listPolicy" && call.request.catalogRef?.name === "production-support"),
    );
  });

  it("reloads once when a chunk of an older build is missing and then offers a reload", async () => {
    let loads = 0;
    page.on("load", () => loads++);
    await load("/chunk-error");
    await page.getByText("A newer version of the Console is available", { exact: true }).waitFor();
    assert.equal(loads, 2);
    await page.getByRole("button", { name: "Reload", exact: true }).waitFor();
    errors = errors.filter((message) => !/error loading dynamically imported module/.test(message));
  });
});
