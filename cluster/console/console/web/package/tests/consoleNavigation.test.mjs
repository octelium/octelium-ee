import assert from "node:assert/strict";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console navigation and agent panel", () => {
  let server;
  let browser;
  let context;
  let page;
  let origin;
  let errors;

  const load = async (path = "/core/services", scenario = "ready") => {
    await page.goto(
      `${origin}tests/fixtures/consoleNavigation.html?${new URLSearchParams({ path, scenario })}`,
    );
    await page.getByRole("navigation", { name: "Breadcrumb" }).waitFor();
  };

  const location = () => page.getByTestId("location").textContent();

  const waitLocation = (expected) =>
    page.waitForFunction(
      (value) =>
        typeof value === "string"
          ? document.querySelector('[data-testid="location"]')?.textContent ===
            value
          : new RegExp(value.source).test(
              document.querySelector('[data-testid="location"]')?.textContent ??
                "",
            ),
      expected instanceof RegExp ? { source: expected.source } : expected,
    );

  const calls = () => page.evaluate(() => window.agentCalls);

  const dock = () => page.getByRole("complementary", { name: "Agent" });

  before(async () => {
    process.env.VITE_CONSOLE_AGENT_URL = "/agent-api";
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
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  it("resolves the breadcrumb trail of every page kind", async () => {
    await load();
    const trails = await page.evaluate(async () => {
      const { resolveTrail } =
        await import("/src/components/Navigation/catalog.ts");
      const { pinnedResources, trailTitle } =
        await import("/src/components/AgentDock/pageContext.ts");
      const pair = (kind) =>
        pinnedResources(
          ["a", "b"].map((name) => ({
            apiVersion: "core/v1",
            kind,
            metadata: { name, uid: name },
          })),
        ).label;
      const labels = (path) => resolveTrail(path).map((crumb) => crumb.label);
      const resource = resolveTrail("/core/services/nginx.default/accesslogs");
      return {
        home: labels("/"),
        section: labels("/core"),
        list: labels("/core/services"),
        create: labels("/core/services/create"),
        tab: labels("/core/services/nginx.default/accesslogs"),
        encoded: labels("/core/users/a%20b"),
        log: labels("/visibility/ssh/abc"),
        agent: labels("/agent"),
        unknown: labels("/other/thing-one"),
        resource: resource[3].resource,
        title: trailTitle(resource),
        homeChildren: resolveTrail("/")[0].children.map((itm) => itm.url),
        accessViews: resolveTrail("/access")[1].views.map((itm) => itm.url),
        clusterman: resolveTrail("/clusterman")[1].children,
        plurals: ["User", "Policy", "Gateway"].map(pair),
      };
    });
    assert.deepEqual(trails.home, ["Overview"]);
    assert.deepEqual(trails.section, ["Overview", "Core"]);
    assert.deepEqual(trails.list, ["Overview", "Core", "Services"]);
    assert.deepEqual(trails.create, ["Overview", "Core", "Services", "Create"]);
    assert.deepEqual(trails.tab, [
      "Overview",
      "Core",
      "Services",
      "nginx.default",
      "Access Logs",
    ]);
    assert.deepEqual(trails.encoded.at(-1), "a b");
    assert.deepEqual(trails.log, [
      "Overview",
      "Visibility",
      "SSH Sessions",
      "abc",
    ]);
    assert.deepEqual(trails.agent, ["Overview", "Agent"]);
    assert.deepEqual(trails.unknown, ["Overview", "Other", "Thing one"]);
    assert.deepEqual(trails.resource, {
      api: "core",
      kind: "Service",
      name: "nginx.default",
    });
    assert.equal(trails.title, "Core › Services › nginx.default › Access Logs");
    assert.ok(trails.homeChildren.includes("/visibility"));
    assert.ok(trails.homeChildren.includes("/agent"));
    assert.ok(trails.accessViews.includes("/access/requests?state=PENDING"));
    assert.equal(trails.clusterman, undefined);
    assert.deepEqual(trails.plurals, ["2 Users", "2 Policies", "2 Gateways"]);
  });

  it("renders the trail, the page slots and navigates to sibling pages", async () => {
    await load("/core/services/nginx.default/accesslogs");
    const nav = page.getByRole("navigation", { name: "Breadcrumb" });
    assert.equal(
      await page.getByRole("heading", { level: 1 }).textContent(),
      "Access Logs",
    );
    for (const name of ["Overview", "Core", "Services", "nginx.default"]) {
      assert.equal(await nav.getByRole("link", { name }).count(), 1);
    }
    assert.equal(await nav.getByTestId("addon").textContent(), "42");
    assert.equal(
      await page
        .getByRole("button", { name: "Page action" })
        .evaluate((el) => !!el.closest("[class*='sticky']")),
      true,
    );
    const height = await page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue(
        "--page-header-height",
      ),
    );
    assert.match(height, /^\d+px$/);
    assert.ok(Number.parseInt(height, 10) >= 48);

    await nav.getByRole("button", { name: "Pages in Core" }).click();
    const menu = page.getByRole("menu");
    await menu.getByRole("menuitem", { name: "Users" }).click();
    await waitLocation("/core/users");
    assert.equal(
      await page.getByRole("heading", { level: 1 }).textContent(),
      "Users",
    );

    await page.getByRole("link", { name: "Overview" }).click();
    await waitLocation("/");
    await page.getByRole("button", { name: "Pages in Overview" }).click();
    await page.getByRole("menuitem", { name: "Visibility" }).click();
    await waitLocation("/visibility");
    await page.getByRole("button", { name: "Pages in Visibility" }).click();
    await page.getByRole("menuitem", { name: "LLM" }).click();
    await waitLocation("/visibility/llm");
  });

  it("collapses the parent pages on narrow screens", async () => {
    await page.setViewportSize({ width: 390, height: 800 });
    await load("/core/services/nginx.default/accesslogs");
    const nav = page.getByRole("navigation", { name: "Breadcrumb" });
    assert.equal(
      await nav.getByRole("link", { name: "Core" }).isVisible(),
      false,
    );
    assert.equal(
      await nav.getByRole("link", { name: "nginx.default" }).isVisible(),
      true,
    );
    await nav.getByRole("button", { name: "Show the parent pages" }).click();
    await page.getByRole("menuitem", { name: "Services" }).click();
    await waitLocation("/core/services");
  });

  it("searches the pages, remembers the visits and asks the agent", async () => {
    await load("/");
    await page.keyboard.press("Control+k");
    const input = page.getByRole("combobox", {
      name: "Search pages and resources, or ask the agent",
    });
    await input.waitFor();
    await input.fill("ssh");
    const options = page.getByRole("option");
    await assert.doesNotReject(
      page.waitForFunction(() =>
        document
          .querySelector('[role="option"]')
          ?.textContent?.includes("SSH Sessions"),
      ),
    );
    assert.equal(
      await options.filter({ hasText: "Requests past their deadline" }).count(),
      0,
    );
    await page.keyboard.press("Enter");
    await waitLocation("/visibility/ssh");
    await assert.doesNotReject(
      page.waitForFunction(() =>
        localStorage
          .getItem("octelium-console-recents")
          ?.includes("/visibility/ssh"),
      ),
    );

    await page.getByRole("button", { name: "Pages in Visibility" }).click();
    await page.getByRole("menuitem", { name: "Metrics" }).click();
    await page.waitForFunction(() =>
      localStorage
        .getItem("octelium-console-recents")
        ?.includes("/visibility/metrics"),
    );

    await page
      .getByRole("button", { name: /Search or jump to/ })
      .first()
      .click();
    await input.waitFor();
    const recent = page.getByRole("option").first();
    assert.match(await recent.textContent(), /SSH Sessions.*Last page/);
    await page.keyboard.press("ArrowDown");
    await page.keyboard.press("ArrowUp");
    await page.keyboard.press("Enter");
    await waitLocation("/visibility/ssh");

    await page.keyboard.press("Control+k");
    await input.fill("why are requests to nginx denied?");
    assert.match(
      await page.getByRole("option").first().textContent(),
      /Ask the agent: “why are requests to nginx denied\?”/,
    );
    await page.keyboard.press("Enter");
    await dock().getByText("The agent answered.").waitFor();
    const created = (await calls()).find(
      (call) => call.method === "POST" && call.path === "/conversations",
    );
    assert.equal(created.body.input.text, "why are requests to nginx denied?");
    assert.deepEqual(created.body.input.context.page, {
      path: "/visibility/ssh",
      title: "Visibility › SSH Sessions",
      details: { "Time range": "last 1h" },
    });
  });

  it("asks about the page from the header and keeps the context per conversation", async () => {
    await load("/core/services/nginx.default");
    const button = page.getByRole("button", {
      name: "Ask the agent about this page",
    });
    await button.click();
    await dock()
      .getByRole("heading", { name: "Ask about nginx.default" })
      .waitFor();
    assert.equal(await button.getAttribute("aria-pressed"), "true");
    assert.equal(
      await page.evaluate(() =>
        document.documentElement.hasAttribute("data-agent-dock"),
      ),
      true,
    );

    await dock()
      .getByRole("button", { name: /Who accessed it\?/ })
      .click();
    await dock().getByText("The agent answered.").waitFor();
    let posted = (await calls()).filter((call) => call.method === "POST");
    assert.equal(posted.length, 1);
    assert.deepEqual(posted[0].body.input.context, {
      page: {
        path: "/core/services/nginx.default",
        title: "Core › Services › nginx.default",
        details: { "Time range": "last 1h" },
      },
      resources: [
        { apiVersion: "core/v1", kind: "Service", name: "nginx.default" },
      ],
    });
    assert.equal(
      await dock().locator('[title="Service nginx.default"]').count(),
      1,
    );

    const composer = dock().getByRole("textbox", { name: "Message" });
    await composer.fill("And who owns it?");
    await composer.press("Enter");
    await page.waitForFunction(
      () =>
        window.agentCalls.filter((call) => call.method === "POST").length === 2,
    );
    posted = (await calls()).filter((call) => call.method === "POST");
    assert.match(posted[1].path, /^\/conversations\/c-\d+\/runs$/);
    assert.equal(posted[1].body.input.context, undefined);

    await page.keyboard.press("Control+j");
    assert.equal(await button.getAttribute("aria-pressed"), "false");
    await page.keyboard.press("Control+j");
    assert.equal(await button.getAttribute("aria-pressed"), "true");

    await dock()
      .getByRole("button", { name: "Open in the Agent page" })
      .click();
    await waitLocation(/^\/agent\?c=c-\d+$/);
    assert.equal(
      await page
        .getByRole("button", { name: "Ask the agent about this page" })
        .count(),
      0,
    );
  });

  it("starts a fresh conversation for a pinned resource", async () => {
    await load("/core/services");
    await page.getByRole("button", { name: "Ask about nginx" }).click();
    await dock()
      .getByRole("heading", { name: /Ask about Service nginx\.default/ })
      .waitFor();
    const remove = dock().getByRole("button", {
      name: "Remove Core › Services from the context",
    });
    await remove.click();
    const composer = dock().getByRole("textbox", { name: "Message" });
    await composer.fill("Clone it");
    await composer.press("Enter");
    await dock().getByText("The agent answered.").waitFor();
    const created = (await calls()).find((call) => call.method === "POST");
    assert.deepEqual(created.body.input.context, {
      resources: [
        {
          apiVersion: "core/v1",
          kind: "Service",
          name: "nginx.default",
          uid: "s-1",
        },
      ],
    });
  });

  it("hides the agent entry points when the agent is unavailable", async () => {
    await load("/core/services", "unavailable");
    await page.waitForFunction(() =>
      window.agentCalls.some((call) => call.path === "/info"),
    );
    await page.waitForTimeout(200);
    assert.equal(
      await page.getByRole("button", { name: /Ask the agent/ }).count(),
      0,
    );
    assert.equal(
      await page.getByRole("button", { name: "Ask about nginx" }).count(),
      0,
    );
    await page.keyboard.press("Control+j");
    assert.equal(await dock().count(), 0);
  });

  it("scores the palette entries", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { scoreText, scoreEntry, isQuestion } =
        await import("/src/components/CommandPalette/match.ts");
      return {
        exact: scoreText("users", "Users"),
        prefix: scoreText("ser", "Services"),
        word: scoreText("logs", "Access Logs"),
        initials: scoreText("al", "Access Logs"),
        tokens: scoreText("logs access", "Access Logs"),
        loose: scoreText("ssh", "Requests past their deadline"),
        missing: scoreText("zzz", "Access Logs"),
        keyword: scoreEntry("idp", {
          title: "Identity Providers",
          keywords: ["idp", "sso"],
        }),
        questions: [
          isQuestion("why is nginx down?"),
          isQuestion("which users are inactive"),
          isQuestion("create service"),
          isQuestion("users"),
        ],
      };
    });
    assert.equal(result.exact, 100);
    assert.equal(result.prefix, 80);
    assert.equal(result.word, 60);
    assert.equal(result.initials, 50);
    assert.ok(result.tokens > 0);
    assert.equal(result.loose, 0);
    assert.equal(result.missing, 0);
    assert.equal(result.keyword, 80);
    assert.deepEqual(result.questions, [true, true, false, false]);
  });
});
