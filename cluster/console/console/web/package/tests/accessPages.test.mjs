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
    assert.equal(value.maxDuration, undefined);
    await page.getByText("Matching requests are rejected as soon as they are created.", { exact: false }).waitFor();
    assert.equal(await page.getByText("Review steps", { exact: true }).count(), 0);

    await effect("Require review").click();
    value = await rule();
    assert.equal(value.effect, "REVIEW");
    assert.deepEqual(
      value.action.review.steps.map((item) => item.name),
      ["team-lead", "security"],
    );
    assert.deepEqual(value.maxDuration, { hours: 8 });

    await effect("Auto-approve").click();
    value = await rule();
    assert.equal(value.action, undefined);
    assert.deepEqual(value.maxDuration, { hours: 8 });
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

  it("shows the frozen scope of an approved request and links to its access logs", async () => {
    await load("/access/requests/req-approved");
    await page.getByText("Access granted", { exact: true }).waitFor();
    await page.getByText("Read only", { exact: true }).first().waitFor();
    await page.getByText("1 system Service excluded", { exact: true }).waitFor();
    await page.getByText("Active", { exact: true }).waitFor();
    const link = page.getByRole("link", { name: /Access logs under this grant/ });
    assert.equal(
      new URL(await link.getAttribute("href"), origin).searchParams.get("matchInlinePolicyRef.uid"),
      "uid-req-approved",
    );
  });

  it("names who revoked a request and why", async () => {
    await load("/access/requests/req-revoked");
    await page.getByText("Revoked by alice", { exact: true }).waitFor();
    await page.getByText("“The incident is closed.”", { exact: true }).waitFor();
  });

  it("shows the access levels, audience and issues of a Catalog", async () => {
    await load("/access/catalogs/production-support");
    await page.getByText("Read the production data without changing it.", { exact: true }).waitFor();
    await page.getByText(/^Break glass/).first().waitFor();
    await page.getByText("The access level break-glass overrides deny guardrails.", { exact: true }).waitFor();
    await page.getByText("Change ticket", { exact: false }).first().waitFor();
  });

  it("expands the Catalog creation sections and preserves their fields", async () => {
    await load("/access/catalogs/create");

    for (const [title, content] of [
      ["Access levels", page.getByRole("button", { name: "Add full access", exact: true })],
      ["Who can request", page.getByRole("textbox", { name: "Requester condition", exact: true })],
      ["Owners", page.getByText("Owner Groups", { exact: true })],
      ["Requirements", page.getByRole("switch", { name: "Require a justification", exact: true })],
      ["Options", page.getByRole("switch", { name: "Let requesters pick individual Services", exact: true })],
    ]) {
      const toggle = page.getByRole("button", { name: new RegExp(`^${title}`) });
      await toggle.waitFor();
      assert.equal(await toggle.isEnabled(), true);
      if (await toggle.getAttribute("aria-expanded") !== "true") await toggle.click();
      await content.waitFor();
      await toggle.click();
      await content.waitFor({ state: "hidden" });
      assert.equal(await toggle.getAttribute("aria-expanded"), "false");
      await toggle.press("Enter");
      await content.waitFor();
      assert.equal(await toggle.getAttribute("aria-expanded"), "true");
    }

    await page.getByRole("button", { name: "Add full access", exact: true }).click();
    const condition = page.getByRole("textbox", { name: "Requester condition", exact: true });
    await condition.fill('ctx.user.metadata.name == "alice"');
    const required = page.getByRole("switch", { name: "Require a justification", exact: true });
    await required.click();
    await page.getByRole("button", { name: /^Who can request/ }).click();
    await page.getByRole("button", { name: /^Requirements/ }).click();
    await page.getByRole("button", { name: /^Who can request/ }).click();
    await page.getByRole("button", { name: /^Requirements/ }).click();
    assert.equal(await condition.inputValue(), 'ctx.user.metadata.name == "alice"');
    assert.equal(await required.isChecked(), true);
    await page.getByRole("textbox", { name: "Name", exact: true }).last().waitFor();
    assert.equal(await page.getByRole("textbox", { name: "Name", exact: true }).last().inputValue(), "full");

    await page.getByRole("button", { name: /^Catalog resources/ }).click();
    await page.getByText("Individual Services", { exact: true }).waitFor();
  });

  it("keeps the selected requester visible when reopening a Catalog simulation", async () => {
    await load("/access/catalogs/production-support");
    await page.getByRole("button", { name: "Simulate", exact: true }).click();
    const modal = page.getByRole("dialog").filter({
      has: page.getByRole("heading", { name: "Simulate a request", exact: true }),
    });
    assert.equal(await modal.getByRole("button", { name: "Run simulation", exact: true }).isDisabled(), true);
    await modal.getByText("Select a User…", { exact: true }).click();
    const picker = page.getByRole("dialog", { name: /Select User/ });
    await picker.getByRole("option", { name: /^alice/ }).click();
    await picker.waitFor({ state: "hidden" });
    assert.equal(await modal.getByRole("button", { name: /^alice/ }).count(), 1);
    assert.equal(await modal.getByRole("button", { name: "Run simulation", exact: true }).isEnabled(), true);
    await modal.getByRole("button", { name: "Close", exact: true }).click();
    await modal.waitFor({ state: "hidden" });
    await page.getByRole("button", { name: "Simulate", exact: true }).click();
    await modal.waitFor();
    assert.equal(await modal.getByRole("button", { name: /^alice/ }).count(), 1);
    assert.equal(await modal.getByRole("button", { name: "Run simulation", exact: true }).isEnabled(), true);
  });

  it("simulates Catalog access for the selected requester and updates the subject when changed", async () => {
    await load("/access/catalogs/production-support");
    await page.getByRole("button", { name: "Simulate", exact: true }).click();
    const modal = page.getByRole("dialog").filter({
      has: page.getByRole("heading", { name: "Simulate a request", exact: true }),
    });
    await modal.getByText("Select a User…", { exact: true }).click();
    const picker = page.getByRole("dialog", { name: /Select User/ });
    await picker.getByRole("option", { name: /^alice/ }).click();
    await picker.waitFor({ state: "hidden" });
    await modal.getByRole("button", { name: "Run simulation", exact: true }).click();
    await page.waitForFunction(() => window.access.calls.some((call) => call.method === "simulateRequest"));
    const first = await page.evaluate(() =>
      window.access.calls.find((call) => call.method === "simulateRequest").request,
    );
    assert.equal(first.requesterRef.name, "alice");
    assert.deepEqual(first.spec.subject?.type, { oneofKind: "userRef", userRef: first.requesterRef });
    assert.equal(first.spec.resource.type.catalog.catalogRef.name, "production-support");
    assert.equal(first.spec.resource.type.catalog.accessLevel, "read-only");
    await modal.getByText("Goes to review", { exact: true }).waitFor();

    await modal.getByRole("button", { name: /^alice/ }).click();
    assert.equal(await picker.getByRole("option", { name: /^alice/ }).getAttribute("aria-selected"), "true");
    await picker.getByRole("option", { name: /^bob/ }).click();
    await picker.waitFor({ state: "hidden" });
    assert.equal(await modal.getByText("Goes to review", { exact: true }).count(), 0);
    await modal.getByRole("button", { name: "Run simulation", exact: true }).click();
    await page.waitForFunction(() => window.access.calls.filter((call) => call.method === "simulateRequest").length === 2);
    const second = await page.evaluate(() =>
      window.access.calls.filter((call) => call.method === "simulateRequest")[1].request,
    );
    assert.equal(second.requesterRef.name, "bob");
    assert.deepEqual(second.spec.subject?.type, { oneofKind: "userRef", userRef: second.requesterRef });
    await modal.getByText("Goes to review", { exact: true }).waitFor();
  });

  it("edits the message, limits and reason rules of a rule", async () => {
    await load("/policy-edit");
    await page.getByRole("textbox", { name: /^Message/ }).fill("Ask in #access instead.");
    await page.getByText("Always required", { exact: true }).first().click();
    const value = await rule();
    assert.equal(value.message, "Ask in #access instead.");
    assert.equal(value.action.review.steps[0].reasonRequirement, "ALWAYS");
    assert.equal(value.authorization, undefined);
  });

  it("matches requests by access level", async () => {
    await load("/policy-edit");
    await page.getByRole("button", { name: "An access level", exact: true }).click();
    await page.getByRole("textbox", { name: /^Access level/ }).fill("break-glass");
    const value = await rule();
    assert.equal(value.condition.accessLevel, "break-glass");
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
