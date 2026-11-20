import assert from "node:assert/strict";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

const EXISTING_SPEC = {
  rules: [
    {
      name: "sre-business-hours",
      condition: {
        all: {
          of: [
            { match: '"sre" in ctx.user.spec.groups' },
            {
              match:
                'now().getDayOfWeek("Europe/Berlin") in [1, 2, 3, 4, 5] && now().getHours("Europe/Berlin") >= 9 && now().getHours("Europe/Berlin") < 17',
            },
            { not: 'ctx.request.http.method == "DELETE"' },
          ],
        },
      },
      effect: "ALLOW",
    },
    {
      name: "geofence",
      condition: { not: 'ctx.session.status.authentication.info.geoip.country.code in ["DE", "FR"]' },
      effect: "DENY",
      priority: -2,
    },
    {
      name: "gold-tier",
      condition: { match: 'json.parse(ctx.request.http.headers["x-meta"]).tier == "gold"' },
      effect: "ALLOW",
      priority: 1,
    },
  ],
  enforcementRules: [
    { condition: { not: 'ctx.service.spec.mode in ["HTTP", "WEB"]' }, effect: "IGNORE" },
  ],
};

describe("Core Policy editor", () => {
  let server;
  let browser;
  let context;
  let page;
  let origin;
  let errors;

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
    context = await browser.newContext({ viewport: { width: 1280, height: 1000 } });
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);
    page = await context.newPage();
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (params = {}) => {
    await page.goto(`${origin}tests/fixtures/policyEdit.html?${new URLSearchParams(params)}`);
    await page.getByRole("button", { name: "Save" }).waitFor();
  };
  const save = async () => {
    await page.getByRole("button", { name: "Save" }).click();
    return page.evaluate(() => window.policyEdit.submitted);
  };
  const segment = async (scope, name) => {
    const id = await scope.getByRole("radio", { name, exact: true }).first().getAttribute("id");
    await page.locator(`label[for="${id}"]`).click();
  };
  const card = (index) => page.locator('[data-rule-card="rule"]').nth(index);
  const footer = () => page.locator(".sticky.bottom-0");
  const asks = () => page.evaluate(() => window.policyEdit.asks);
  const choose = async (name) => {
    await page.getByRole("menu").last().getByRole("menuitem", { name, exact: true }).click();
    await page.getByRole("menu").waitFor({ state: "detached" });
  };

  it("shows an existing Policy as blocks and saves it unchanged", async () => {
    await load();
    const first = card(0);
    await first.getByRole("button", { name: "Expand rule" }).click();
    for (const title of ["Group membership", "Time window", "HTTP method"]) {
      await first.getByText(title, { exact: true }).waitFor();
    }
    await first.getByText("Not", { exact: true }).waitFor();
    await first.getByText("Mon–Fri, 09:00–17:00 Europe/Berlin").waitFor();
    assert.match(await card(1).innerText(), /not \(Country is DE or FR\)/);
    assert.match(await card(2).innerText(), /json\.parse/);
    const submitted = await save();
    assert.deepEqual(submitted.spec, EXISTING_SPEC);
  });

  it("blocks saving until every condition is complete", async () => {
    await load({ policy: "new" });
    await page.getByRole("textbox", { name: /^Name/ }).fill("team-access");
    await page.getByRole("button", { name: "Blank rule" }).click();
    await footer().getByText("Rule 1: Add a condition").waitFor();
    assert.equal(await page.getByRole("button", { name: "Save" }).isDisabled(), true);

    await card(0).getByRole("button", { name: /^Members of a Group/ }).click();
    await footer().getByText("Rule “group-members”: Choose at least one Group").waitFor();
    assert.equal(await page.getByRole("button", { name: "Save" }).isDisabled(), true);

    await card(0).getByText("Select one or more Group resources…").click();
    await page.getByRole("option", { name: /sre/ }).click();
    await page.getByRole("button", { name: "Done" }).click();
    await footer().getByText("Unsaved changes").waitFor();
    const submitted = await save();
    assert.deepEqual(submitted.spec.rules, [
      { name: "group-members", condition: { match: '"sre" in ctx.user.spec.groups' }, effect: "ALLOW" },
    ]);
  });

  it("adds the rules of a template", async () => {
    await load({ policy: "new" });
    await page.getByRole("textbox", { name: /^Name/ }).fill("http-read-only");
    await page.getByRole("button", { name: /Read-only HTTP/ }).click();
    const submitted = await save();
    assert.deepEqual(submitted.spec, {
      rules: [
        {
          name: "allowed-methods",
          condition: { match: 'ctx.request.http.method in ["GET", "HEAD", "OPTIONS"]' },
          effect: "ALLOW",
          priority: -1,
        },
        { name: "deny-otherwise", condition: { matchAny: true }, effect: "DENY" },
      ],
      enforcementRules: [
        { condition: { not: 'ctx.service.spec.mode in ["HTTP", "WEB"]' }, effect: "IGNORE" },
      ],
    });
  });

  it("switches a condition between the visual and the CEL editors", async () => {
    await load();
    const first = card(0);
    await first.getByRole("button", { name: "Expand rule" }).click();
    await segment(first, "CEL");
    const editor = first.getByRole("textbox", { name: "Condition as CEL" });
    assert.match(await editor.innerText(), /^"sre" in ctx\.user\.spec\.groups\n&& now\(\)/);
    await first.getByText("Its NOT conditions and None groups").waitFor();
    await editor.fill('"ops" in ctx.user.spec.groups && ctx.request.http.method == "GET"');
    let submitted = await save();
    assert.deepEqual(submitted.spec.rules[0].condition, {
      match: '"ops" in ctx.user.spec.groups && ctx.request.http.method == "GET"',
    });
    await segment(first, "Visual");
    await first.getByRole("button", { name: "Expand" }).click();
    await first.getByRole("button", { name: "Split into separate conditions" }).click();
    await first.getByText("User is in ops").waitFor();
    submitted = await save();
    assert.deepEqual(submitted.spec.rules[0].condition, {
      all: {
        of: [
          { match: '"ops" in ctx.user.spec.groups' },
          { match: 'ctx.request.http.method == "GET"' },
        ],
      },
    });
  });

  it("finds conditions by searching and edits their values", async () => {
    await load();
    const first = card(0);
    await first.getByRole("button", { name: "Expand rule" }).click();
    await first.getByRole("button", { name: "Add condition" }).click();
    const dialog = page.getByRole("dialog", { name: "Add a condition" });
    await dialog.getByRole("textbox", { name: "Search conditions" }).fill("ip range office");
    await dialog.getByText("Source IP address").waitFor();
    await dialog.getByRole("textbox", { name: "Search conditions" }).press("Enter");
    await dialog.waitFor({ state: "hidden" });
    const ranges = first.getByRole("combobox", { name: "IP ranges" });
    await ranges.fill("10.0.0.1");
    await ranges.press("Enter");
    await first.getByText('net.isIPInRange(ctx.request.ip, "10.0.0.1/32")').waitFor();
    const submitted = await save();
    assert.deepEqual(submitted.spec.rules[0].condition.all.of.at(-1), {
      match: 'net.isIPInRange(ctx.request.ip, "10.0.0.1/32")',
    });
    assert.deepEqual(submitted.spec.rules[0].condition.all.of.slice(0, 3), EXISTING_SPEC.rules[0].condition.all.of);
  });

  it("negates, removes and groups conditions with an alternative", async () => {
    await load();
    const first = card(0);
    await first.getByRole("button", { name: "Expand rule" }).click();
    const actions = first.getByRole("button", { name: "Condition actions" });
    await actions.nth(2).click();
    await choose("Remove the negation");
    await actions.nth(1).click();
    await choose("Remove");
    await actions.nth(0).click();
    await choose("Add an alternative (OR)");
    const dialog = page.getByRole("dialog", { name: "Add a condition" });
    await dialog.getByRole("textbox", { name: "Search conditions" }).fill("human");
    await dialog.getByRole("button", { name: /^Human Users/ }).click();
    const submitted = await save();
    assert.deepEqual(submitted.spec.rules[0].condition, {
      all: {
        of: [
          {
            any: {
              of: [
                { match: '"sre" in ctx.user.spec.groups' },
                { match: 'ctx.user.spec.type == "HUMAN"' },
              ],
            },
          },
          { match: 'ctx.request.http.method == "DELETE"' },
        ],
      },
    });
  });

  it("asks the AI about a rule and applies its CEL proposal", async () => {
    await load();
    await card(0).getByRole("button", { name: "Ask AI about this rule" }).click();
    const dialog = page.getByRole("dialog", { name: "Ask AI about rule “sre-business-hours”" });
    await dialog.getByRole("radio", { name: /Write the condition/ }).click();
    assert.equal(await dialog.getByRole("button", { name: "Ask AI" }).isDisabled(), true);
    await dialog.getByRole("textbox", { name: "Describe it" }).fill("sre members with MFA");
    await dialog.getByRole("button", { name: "Ask AI" }).click();
    const [request] = await asks();
    assert.equal(request.send, true);
    assert.match(request.text, /It must match: sre members with MFA/);
    assert.match(request.text, /```cel/);
    assert.equal(request.pinned.items[0].kind, "PolicyDraft");
    assert.equal(request.pinned.items[0].data.spec.rules.length, 3);

    const answer = page.getByRole("complementary", { name: "AI answer" });
    await answer.getByRole("button", { name: "Apply to rule “sre-business-hours”" }).click();
    await answer.getByRole("button", { name: "Applied" }).waitFor();
    await card(0).getByText("Authentication assurance (AAL)").waitFor();
    let submitted = await save();
    assert.deepEqual(submitted.spec.rules[0].condition, {
      all: {
        of: [
          { match: '"sre" in ctx.user.spec.groups' },
          { match: 'ctx.session.status.authentication.info.aal in ["AAL2", "AAL3"]' },
        ],
      },
    });
    await page.getByRole("button", { name: "Undo" }).click();
    submitted = await save();
    assert.deepEqual(submitted.spec, EXISTING_SPEC);
  });

  it("drafts a whole Policy with the AI", async () => {
    await load({ policy: "new" });
    await page.getByRole("textbox", { name: /^Name/ }).fill("auditors");
    await page.getByRole("button", { name: "Draft with AI" }).click();
    const dialog = page.getByRole("dialog", { name: "Draft this Policy with AI" });
    await dialog.getByRole("button", { name: /Auditors can read every Octelium API/ }).click();
    await dialog.getByRole("button", { name: "Ask AI" }).click();
    const [request] = await asks();
    assert.match(request.text, /^Draft the new core Policy\. It must: Auditors can read/);
    assert.match(request.text, /```yaml code block/);

    const answer = page.getByRole("complementary", { name: "AI answer" });
    assert.equal(await answer.getByRole("button", { name: /Apply to rule/ }).count(), 0);
    await answer.getByRole("button", { name: "Apply to this Policy" }).click();
    await page.getByText("auditors-read-only", { exact: true }).waitFor();
    const submitted = await save();
    assert.deepEqual(submitted.spec.rules, [
      {
        name: "auditors-read-only",
        condition: {
          all: {
            of: [
              { match: '"auditors" in ctx.user.spec.groups' },
              { match: 'ctx.request.http.method in ["GET", "HEAD", "OPTIONS"]' },
            ],
          },
        },
        effect: "ALLOW",
        priority: -1,
      },
      { name: "deny-otherwise", condition: { matchAny: true }, effect: "DENY" },
    ]);
  });

  it("evaluates the draft with the Policy tester", async () => {
    await load();
    await page.getByRole("button", { name: /Try this draft/ }).click();
    await page.getByText("Select a User…").click();
    await page.getByRole("option", { name: /alice/ }).click();
    await page.getByText("Select a Service…").click();
    await page.getByRole("option", { name: /billing\.production/ }).click();
    await page.getByRole("button", { name: "Evaluate the draft" }).click();
    const status = page.getByRole("status").filter({ hasText: "Allowed" });
    await status.getByText("Decided by this draft").waitFor();
    const call = await page.evaluate(() =>
      window.policyEdit.calls.find((item) => item.method === "isAuthorized"),
    );
    assert.equal(call.request.downstream.userRef.name, "alice");
    assert.equal(call.request.upstream.serviceRef.name, "billing.production");
    assert.equal(call.request.additional.inlinePolicies[0].name, "sre-production");
    assert.equal(call.request.additional.inlinePolicies[0].spec.rules.length, 3);
  });

  for (const theme of ["light", "dark"]) {
    it(`fits a ${theme} phone screen inside a drawer`, async () => {
      await page.setViewportSize({ width: 390, height: 844 });
      await load({ theme, drawer: "true" });
      await card(0).getByRole("button", { name: "Expand rule" }).click();
      await card(0).getByRole("button", { name: "Expand" }).first().click();
      const overflow = await page.evaluate(() =>
        [...document.querySelectorAll("[data-rule-card], [data-condition-builder]")].some(
          (element) => element.scrollWidth > element.clientWidth + 1,
        ),
      );
      assert.equal(overflow, false);
      const box = await card(0).boundingBox();
      assert.ok(box.x >= 0 && box.x + box.width <= 390);
      await card(0).getByRole("button", { name: "Add condition" }).click();
      const dialog = page.getByRole("dialog", { name: "Add a condition" });
      await dialog.getByRole("navigation", { name: "Condition categories" }).getByRole("button", { name: /Time/ }).click();
      await dialog.getByRole("button", { name: /Time window/ }).click();
      await dialog.waitFor({ state: "hidden" });
      await card(0).getByRole("group", { name: "Days of the week" }).waitFor();
    });
  }
});
