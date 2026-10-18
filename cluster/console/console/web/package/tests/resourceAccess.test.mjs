import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Resource access pages", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-resource-access-"));
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

  const load = async (
    path = "/core/users/john/access/requests",
    extra = {},
  ) => {
    await page.goto(
      `${origin}tests/fixtures/resourceAccess.html?${new URLSearchParams({ path, ...extra })}`,
    );
    await page.getByRole("dialog").waitFor();
  };
  const dialog = () => page.getByRole("dialog");
  const selectTab = async (name) => {
    const id = await dialog()
      .getByRole("radio", { name, exact: true })
      .getAttribute("id");
    await dialog().locator(`label[for="${id}"]`).click();
  };
  const scopedCall = async (method, field, name) => {
    await page.waitForFunction(
      ({ method, field, name }) =>
        window.resourceAccess.calls.some(
          (call) =>
            call.client === "visibilityAccess" &&
            call.method === method &&
            call.request[field]?.name === name,
        ),
      { method, field, name },
    );
    return page.evaluate(
      ({ method, field, name }) =>
        window.resourceAccess.calls
          .filter(
            (call) =>
              call.client === "visibilityAccess" &&
              call.method === method &&
              call.request[field]?.name === name,
          )
          .at(-1).request,
      { method, field, name },
    );
  };
  const choose = async (name, option) => {
    await dialog().getByRole("combobox", { name, exact: true }).click();
    await page.getByRole("option", { name: option, exact: true }).click();
    await page.waitForFunction(
      ({ name, option }) =>
        Array.from(document.querySelectorAll('[role="dialog"] input')).some(
          (input) =>
            input.getAttribute("aria-label") === name && input.value === option,
        ),
      { name, option },
    );
  };
  const route = () => page.locator("[data-route]").textContent();

  it("separates a user's requests, reviews, subjects, reviewer assignments, policies and integrations", async () => {
    await load();
    for (const [tab, method, field] of [
      ["Requests", "listRequest", "userRef"],
      ["Reviews", "listReview", "userRef"],
      ["Subjects", "listRequest", "subjectUserRef"],
      ["Assigned requests", "listRequest", "reviewerRef"],
      ["Policies", "listPolicy", "userRef"],
      ["Linked accounts", "listIntegrationIdentity", "userRef"],
      ["Deliveries", "listIntegrationBinding", "userRef"],
    ]) {
      await selectTab(tab);
      const request = await scopedCall(method, field, "john");
      assert.equal(request[field].uid, "uid-john");
      assert.equal(
        await dialog()
          .getByRole("radio", { name: "Access", exact: true })
          .isChecked(),
        true,
      );
    }
    await dialog().getByText("Delivered", { exact: true }).waitFor();
    await dialog()
      .getByRole("link", { name: "slack", exact: false })
      .first()
      .waitFor();
  });

  for (const [path, method, field, name] of [
    [
      "/core/groups/security/access/policies",
      "listPolicy",
      "groupRef",
      "security",
    ],
    [
      "/core/services/billing.production/access/requests",
      "listRequest",
      "serviceRef",
      "billing.production",
    ],
    [
      "/core/services/billing.production/access/policies",
      "listPolicy",
      "serviceRef",
      "billing.production",
    ],
    [
      "/core/services/billing.production/access/catalogs",
      "listCatalog",
      "serviceRef",
      "billing.production",
    ],
    [
      "/core/namespaces/production/access/catalogs",
      "listCatalog",
      "namespaceRef",
      "production",
    ],
    [
      "/access/policies/production-access/access/requests",
      "listRequest",
      "policyRef",
      "production-access",
    ],
    [
      "/access/catalogs/production-support/access/requests",
      "listRequest",
      "catalogRef",
      "production-support",
    ],
    [
      "/access/catalogs/production-support/access/policies",
      "listPolicy",
      "catalogRef",
      "production-support",
    ],
    [
      "/access/requests/request-01/access/reviews",
      "listReview",
      "requestRef",
      "request-01",
    ],
    [
      "/access/requests/request-01/access/deliveries",
      "listIntegrationBinding",
      "requestRef",
      "request-01",
    ],
    [
      "/access/integrations/slack/access/deliveries",
      "listIntegrationBinding",
      "integrationRef",
      "slack",
    ],
    [
      "/access/integrations/slack/access/identities",
      "listIntegrationIdentity",
      "integrationRef",
      "slack",
    ],
  ]) {
    it(`scopes ${path} to the resource reference`, async () => {
      await load(path);
      const request = await scopedCall(method, field, name);
      assert.equal(request[field].uid, `uid-${name}`);
      assert.equal(request.common.itemsPerPage, 25);
      assert.equal(
        await dialog()
          .getByRole("radio", { name: "Access", exact: true })
          .isChecked(),
        true,
      );
    });
  }

  it("adds the Access tab beside the log tabs and defaults to the first supported section", async () => {
    await load("/core/users/john");
    await selectTab("Access");
    await scopedCall("listRequest", "userRef", "john");
    assert.equal(await route(), "/core/users/john/access/requests");
    await dialog()
      .getByRole("radio", { name: "Auth Logs", exact: true })
      .waitFor({ state: "attached" });
    await dialog()
      .getByRole("radio", { name: "Access Logs", exact: true })
      .waitFor({ state: "attached" });
    await load("/core/namespaces/production/access");
    await scopedCall("listCatalog", "namespaceRef", "production");
    assert.equal(await route(), "/core/namespaces/production/access/catalogs");
    await load("/core/policies/authorization");
    await dialog()
      .getByRole("radio", { name: "Overview", exact: true })
      .waitFor({ state: "attached" });
    assert.equal(
      await dialog()
        .getByRole("radio", { name: "Access", exact: true })
        .count(),
      0,
    );
  });

  it("paginates, resets the page for filters, and keeps the resource scope when clearing filters", async () => {
    await load(
      "/core/users/john/access/requests?common.itemsPerPage=10&common.page=2",
    );
    let request = await scopedCall("listRequest", "userRef", "john");
    assert.equal(request.common.page, 1);
    await dialog()
      .getByRole("link", { name: "Open Request request-11", exact: true })
      .waitFor();
    await choose("State", "Pending");
    await choose("Urgency", "High");
    await page.waitForFunction(() =>
      window.resourceAccess.calls.some(
        (call) =>
          call.request.userRef?.name === "john" &&
          call.request.state === 1 &&
          call.request.urgency === 4,
      ),
    );
    request = await scopedCall("listRequest", "userRef", "john");
    assert.equal(request.common.page, 0);
    assert.equal(
      new URLSearchParams((await route()).split("?")[1]).has("common.page"),
      false,
    );
    await dialog()
      .getByRole("button", { name: "More filters", exact: true })
      .click();
    await page
      .getByRole("checkbox", { name: "Active access", exact: true })
      .click();
    await page.waitForFunction(() =>
      document
        .querySelector("[data-route]")
        .textContent.includes("isActive=true"),
    );
    await page.keyboard.press("Escape");
    await dialog()
      .getByRole("button", { name: "Clear filters", exact: true })
      .click();
    request = await scopedCall("listRequest", "userRef", "john");
    await page.waitForFunction(
      () =>
        !document.querySelector("[data-route]").textContent.includes("state="),
    );
    assert.equal(
      await route(),
      "/core/users/john/access/requests?common.itemsPerPage=10",
    );
    assert.equal(request.userRef.uid, "uid-john");
  });

  it("searches and restores filters through navigation without carrying request filters into reviews", async () => {
    await load();
    await dialog()
      .getByRole("textbox", { name: "Search access resources", exact: true })
      .fill("request-01");
    await page.waitForFunction(() =>
      window.resourceAccess.calls.some(
        (call) =>
          call.request.userRef?.name === "john" &&
          call.request.common?.query === "request-01",
      ),
    );
    await selectTab("Reviews");
    await scopedCall("listReview", "userRef", "john");
    await choose("Decision", "Approved");
    await page.evaluate(() => window.resourceAccess.navigate(-1));
    await page.waitForFunction(
      () =>
        document.querySelector("[data-route]").textContent ===
        "/core/users/john/access/reviews",
    );
    await page.evaluate(() => window.resourceAccess.navigate(-1));
    await dialog()
      .getByRole("link", { name: "Open Request request-01", exact: true })
      .waitFor();
    assert.match(await route(), /common.query=request-01/);
    await page.evaluate(() => window.resourceAccess.navigate(1));
    await dialog()
      .getByRole("combobox", { name: "Decision", exact: true })
      .waitFor();
  });

  it("links to an item and returns to the filtered resource page when the item is closed", async () => {
    await load("/core/users/john/access/requests?state=PENDING");
    await dialog()
      .getByRole("link", { name: "Open Request request-01", exact: true })
      .click();
    await page.waitForFunction(
      () =>
        document.querySelector("[data-route]").textContent ===
        "/access/requests/request-01",
    );
    await dialog().locator(".mantine-Drawer-close").click();
    await page.waitForFunction(
      () =>
        document.querySelector("[data-route]").textContent ===
        "/core/users/john/access/requests?state=PENDING",
    );
    await dialog()
      .getByRole("combobox", { name: "State", exact: true })
      .waitFor();
    assert.equal(
      await dialog()
        .getByRole("combobox", { name: "State", exact: true })
        .inputValue(),
      "Pending",
    );
  });

  it("keeps the original resource list as the return destination while searching access", async () => {
    await load(undefined, { returnTo: "/core/users?common.query=john" });
    await dialog()
      .getByRole("textbox", { name: "Search access resources", exact: true })
      .fill("request-01");
    await page.waitForFunction(() =>
      document
        .querySelector("[data-route]")
        .textContent.includes("common.query=request-01"),
    );
    await dialog().locator(".mantine-Drawer-close").click();
    await page.waitForFunction(
      () =>
        document.querySelector("[data-route]").textContent ===
        "/core/users?common.query=john",
    );
  });

  it("retains valid filters and the route's scope when URL parameters are invalid or name a different user", async () => {
    await load(
      "/core/users/john/access/requests?userRef.name=mallory&state=PENDING&urgency=invalid&common.page=-5&common.itemsPerPage=999",
    );
    const request = await scopedCall("listRequest", "userRef", "john");
    assert.equal(request.userRef.name, "john");
    assert.equal(request.state, 1);
    assert.equal(request.urgency, 0);
    assert.equal(request.common.page, 0);
    assert.equal(request.common.itemsPerPage, 25);
    await dialog()
      .getByText("Some URL filters were invalid and have been ignored.", {
        exact: true,
      })
      .waitFor();
    await page.reload();
    assert.equal((await scopedCall("listRequest", "userRef", "john")).state, 1);
  });

  it("serializes the advanced request, review and delivery filters supported by the API", async () => {
    const flags = [
      "isActive",
      "hasDeadline",
      "isDeadlinePassed",
      "isUnmatched",
      "isExpiringSoon",
      "isPendingStale",
      "isOnBehalf",
    ];
    await load(
      `/core/users/john/access/requests?effect=REVIEW&originType=INTEGRATION&${flags.map((key) => `${key}=true`).join("&")}`,
    );
    let request = await scopedCall("listRequest", "userRef", "john");
    assert.equal(request.effect, 2);
    assert.equal(request.originType, 2);
    for (const flag of flags) assert.equal(request[flag], true);

    await load(
      "/core/users/john/access/reviews?decision=REJECT&originType=API&isDecided=true",
    );
    request = await scopedCall("listReview", "userRef", "john");
    assert.equal(request.decision, 2);
    assert.equal(request.originType, 1);
    assert.equal(request.isDecided, true);

    await load(
      "/access/requests/request-01/access/deliveries?state=READY&purpose=NOTIFICATION&audience=REQUESTER&interactionMode=INTERACTIVE&isFailing=true&isOutOfDate=true",
    );
    request = await scopedCall(
      "listIntegrationBinding",
      "requestRef",
      "request-01",
    );
    assert.equal(request.state, 2);
    assert.equal(request.purpose, 2);
    assert.equal(request.audience, 3);
    assert.equal(request.interactionMode, 2);
    assert.equal(request.isFailing, true);
    assert.equal(request.isOutOfDate, true);
  });

  it("loads the new scope when navigating between resources of the same kind", async () => {
    await load();
    await scopedCall("listRequest", "userRef", "john");
    await page.evaluate(() =>
      window.resourceAccess.navigate("/core/users/123/access/requests"),
    );
    const request = await scopedCall("listRequest", "userRef", "123");
    assert.equal(request.userRef.name, "123");
    assert.equal(request.userRef.uid, "uid-123");
    await dialog()
      .getByRole("link", { name: "Requester 123", exact: true })
      .first()
      .waitFor();
    assert.equal(
      await dialog()
        .getByRole("link", { name: "Requester john", exact: true })
        .count(),
      0,
    );
  });

  it("shows empty results and retries a failed refresh", async () => {
    await load("/core/users/john/access/requests?state=CANCELLED");
    await dialog()
      .getByText("No requests found matching these filters.", { exact: true })
      .waitFor();
    await dialog()
      .getByRole("button", { name: "Clear filters", exact: true })
      .click();
    await dialog()
      .getByRole("link", { name: "Open Request request-01", exact: true })
      .waitFor();
    await page.evaluate(() => {
      window.resourceAccess.failAccess = true;
    });
    await dialog()
      .getByRole("button", { name: "Refresh access resources", exact: true })
      .click();
    await dialog().getByRole("alert").waitFor();
    await page.evaluate(() => {
      window.resourceAccess.failAccess = false;
    });
    await dialog()
      .getByRole("button", { name: "Try again", exact: true })
      .click();
    await dialog().getByRole("alert").waitFor({ state: "hidden" });
  });

  it("uses the same controls in the main access list and preserves reference filters", async () => {
    await page.goto(
      `${origin}tests/fixtures/resourceAccess.html?${new URLSearchParams({ path: "/access/requests?userRef.name=john&common.page=2" })}`,
    );
    await page.getByRole("combobox", { name: "State", exact: true }).click();
    await page.getByRole("option", { name: "Pending", exact: true }).click();
    await page.waitForFunction(() =>
      window.resourceAccess.calls.some(
        (call) =>
          call.client === "visibilityAccess" &&
          call.method === "listRequest" &&
          call.request.userRef?.name === "john" &&
          call.request.state === 1 &&
          call.request.common?.page === 0,
      ),
    );
    const request = await scopedCall("listRequest", "userRef", "john");
    assert.equal(request.state, 1);
    assert.equal(request.common.page, 0);
    assert.match(await route(), /userRef.name=john/);
  });

  for (const theme of ["light", "dark"]) {
    for (const width of [320, 768, 1440]) {
      it(`keeps navigation and filters usable in ${theme} mode at ${width}px`, async () => {
        await page.setViewportSize({ width, height: 1000 });
        await load(undefined, { theme });
        await dialog()
          .getByRole("link", { name: "Open Request request-01", exact: true })
          .waitFor();
        await choose("State", "Pending");
        const overflow = await dialog().evaluate(
          (element) => element.scrollWidth > element.clientWidth + 1,
        );
        assert.equal(overflow, false);
        await page.screenshot({
          path: `/tmp/console-resource-access-${theme}-${width}.png`,
        });
      });
    }
  }
});
