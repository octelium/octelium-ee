import assert from "node:assert/strict";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Agent chat", () => {
  let server;
  let browser;
  let context;
  let page;
  let origin;

  const composer = () => page.getByRole("textbox", { name: "Message" });

  const calls = () => page.evaluate(() => window.agentCalls);

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
    context = await browser.newContext({
      timezoneId: "UTC",
      viewport: { width: 1440, height: 900 },
    });
    page = await context.newPage();
    await page.goto(`${origin}tests/fixtures/agentChat.html`);
    await page.getByText("Cluster overview", { exact: true }).waitFor();
  });

  afterEach(async () => {
    await context?.close();
  });

  it("shows the empty state and groups the conversations by date", async () => {
    await page.getByRole("heading", { name: /, George$/ }).waitFor();
    const nav = page.getByRole("navigation", { name: "Conversations" });
    for (const label of ["Today", "Yesterday"]) {
      assert.equal(await nav.getByText(label, { exact: true }).count(), 1);
    }
    assert.deepEqual(
      await nav
        .locator("li button[title]")
        .evaluateAll((items) =>
          items.map((item) => item.getAttribute("title")),
        ),
      [
        "Audit admins group permissions",
        "Rotate the Postgres credential",
        "Certificate expirations",
      ],
    );

    await page.getByText("Cluster overview", { exact: true }).click();
    assert.equal(
      await composer().inputValue(),
      "Give me an overview of the Cluster's Services and their health",
    );
    await assert.doesNotReject(
      page.waitForFunction(
        () => document.activeElement?.getAttribute("aria-label") === "Message",
      ),
    );
  });

  it("auto-grows the composer without a resize handle", async () => {
    assert.equal(
      await composer().evaluate((el) => getComputedStyle(el).resize),
      "none",
    );
    const initial = (await composer().boundingBox()).height;
    await composer().fill("one\ntwo\nthree\nfour");
    const grown = (await composer().boundingBox()).height;
    assert.ok(grown > initial);
    await composer().fill(Array.from({ length: 60 }, (_, i) => i).join("\n"));
    const capped = (await composer().boundingBox()).height;
    assert.ok(capped <= 320);
    await composer().fill("");
    assert.equal((await composer().boundingBox()).height, initial);
    assert.equal(
      await page.getByRole("button", { name: "Send" }).isDisabled(),
      true,
    );
  });

  it("searches the titles and the messages", async () => {
    await page.keyboard.press("Control+k");
    const search = page.getByRole("searchbox", {
      name: "Search conversations",
    });
    await assert.doesNotReject(
      page.waitForFunction(
        () => document.activeElement?.getAttribute("type") === "search",
      ),
    );
    await search.fill("admins");
    await page.getByText("Messages", { exact: true }).waitFor();
    assert.equal(await page.getByText("Titles", { exact: true }).count(), 1);
    assert.ok((await page.locator("nav mark").count()) >= 2);
    assert.ok(
      (await calls()).some((c) => c.path === "/conversations/search?q=admins"),
    );

    await page
      .locator("nav")
      .getByText("The Group admins has 3 Policies attached.")
      .click();
    await page
      .locator('[data-message-id="c1-a"]')
      .getByText("The Group admins has 3 Policies attached.")
      .waitFor();
    assert.equal(new URL(page.url()).searchParams.get("c"), "c1");

    await search.fill("nothing matches this");
    await page.getByText("No results", { exact: true }).waitFor();
  });

  it("sends a message and streams the response", async () => {
    await composer().fill("List my Services");
    await page.keyboard.press("Enter");
    await page.getByRole("button", { name: "Stop" }).waitFor();
    assert.equal(await composer().inputValue(), "");
    await page.getByText("You have 2 Services.").waitFor();
    await page.getByRole("button", { name: "Send" }).waitFor();
    assert.equal(
      await page
        .locator("[data-message-id]")
        .getByText("List my Services", { exact: true })
        .count(),
      1,
    );
    assert.equal(
      await page
        .getByRole("navigation", { name: "Conversations" })
        .getByText("List my Services", { exact: true })
        .count(),
      1,
    );

    const created = (await calls()).find(
      (c) => c.method === "POST" && c.path === "/conversations",
    );
    assert.deepEqual(created.body, { input: { text: "List my Services" } });
    assert.ok(new URL(page.url()).searchParams.get("c")?.startsWith("c-"));
  });

  it("asks for an approval and sends the decision", async () => {
    await composer().fill("Delete the nginx Service");
    await page.keyboard.press("Enter");
    await page.getByText("Approval required", { exact: true }).waitFor();
    assert.equal((await page.getByText("Destructive").count()) > 0, true);

    await page.getByRole("button", { name: "Approve" }).click();
    await page.getByText("Approved", { exact: true }).waitFor();
    await page.getByText("You have 2 Services.").waitFor();

    const decision = (await calls()).find((c) =>
      c.path.endsWith("/approvals/ap-1"),
    );
    assert.equal(decision.method, "POST");
    assert.equal(decision.body.decision, "approve");
  });

  it("renames and deletes a conversation", async () => {
    await page
      .getByRole("navigation", { name: "Conversations" })
      .getByText("Rotate the Postgres credential")
      .click();
    await page
      .getByText("Create a new Secret and update the upstream.")
      .waitFor();

    const header = page.locator("header");
    await header
      .getByRole("button", { name: "Rotate the Postgres credential" })
      .click();
    const title = header.getByRole("textbox", { name: "Conversation title" });
    await title.fill("Rotate pg.prod");
    await title.press("Enter");
    await header.getByRole("button", { name: "Rotate pg.prod" }).waitFor();
    const renamed = (await calls()).find((c) => c.method === "PATCH");
    assert.deepEqual(renamed, {
      method: "PATCH",
      path: "/conversations/c2",
      body: { title: "Rotate pg.prod" },
    });

    await header.getByRole("button", { name: "Conversation actions" }).click();
    await page.getByRole("menuitem", { name: "Delete" }).click();
    await page
      .getByRole("dialog")
      .getByRole("button", { name: "Delete" })
      .click();
    await page.getByRole("dialog").waitFor({ state: "detached" });
    assert.equal(new URL(page.url()).searchParams.get("c"), null);
    assert.equal(
      await page
        .getByRole("navigation", { name: "Conversations" })
        .getByText("Rotate pg.prod")
        .count(),
      0,
    );
    assert.ok(
      (await calls()).some(
        (c) => c.method === "DELETE" && c.path === "/conversations/c2",
      ),
    );
  });
});
