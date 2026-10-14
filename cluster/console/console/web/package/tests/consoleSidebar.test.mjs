import assert from "node:assert/strict";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console sidebar", () => {
  let server;
  let browser;
  let context;
  let page;
  let origin;
  let errors;

  const load = async (path = "/core/services", theme = "light") => {
    await page.goto(
      `${origin}tests/fixtures/consoleSidebar.html?${new URLSearchParams({ path, theme })}`,
    );
    await page.getByTestId("location").waitFor();
  };

  const sidebar = () =>
    page.getByRole("navigation", { name: "Console sidebar", exact: true });
  const drawer = () => page.getByRole("dialog", { name: "Console navigation" });
  const functions = () =>
    sidebar().getByText("Functions", { exact: true }).click();
  const waitLocation = (path) =>
    page.waitForFunction(
      (value) =>
        document.querySelector('[data-testid="location"]')?.textContent ===
        value,
      path,
    );
  const waitWidth = (width) =>
    page.waitForFunction(
      (value) =>
        Math.abs(
          document
            .querySelector('[aria-label="Console sidebar"]')
            .getBoundingClientRect().width - value,
        ) < 1,
      width,
    );

  before(async () => {
    server = await createServer({
      root,
      configFile: false,
      cacheDir: "node_modules/.vite/console-sidebar-tests",
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
      viewport: { width: 1440, height: 900 },
    });
    page = await context.newPage();
    page.setDefaultTimeout(10000);
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  it("defaults to API navigation and selects only the resource for deep routes", async () => {
    await load("/core/services/nginx.default/edit");
    assert.equal(
      await sidebar()
        .getByRole("radio", { name: "API", exact: true })
        .isChecked(),
      true,
    );
    assert.equal(
      await sidebar().locator('[aria-current="page"]').getAttribute("href"),
      "/core/services",
    );
    assert.equal(
      await sidebar()
        .getByRole("link", { name: "Core overview" })
        .getAttribute("aria-current"),
      null,
    );
    assert.equal(
      await sidebar()
        .getByRole("link", { name: "Certificates", exact: true })
        .count(),
      0,
    );
    await sidebar()
      .getByRole("button", { name: "API workspace Core", exact: true })
      .click();
    await page
      .getByRole("menuitemradio", { name: "Enterprise", exact: true })
      .click();
    await waitLocation("/enterprise");
    assert.equal(
      await sidebar()
        .getByRole("link", { name: "Enterprise overview" })
        .getAttribute("aria-current"),
      "page",
    );
    assert.equal(
      await sidebar()
        .getByRole("link", { name: "Certificates", exact: true })
        .count(),
      1,
    );
  });

  it("shows the workspace directory without choosing a default API on global pages", async () => {
    for (const path of ["/", "/agent", "/settings"]) {
      await load(path);
      await sidebar()
        .getByRole("button", { name: /Choose workspace/ })
        .waitFor();
      assert.equal(
        await sidebar()
          .getByRole("link", { name: "Services", exact: true })
          .count(),
        0,
      );
      assert.equal(
        await sidebar()
          .getByRole("link", { name: "Core", exact: true })
          .getAttribute("href"),
        "/core",
      );
      assert.equal(
        await sidebar().locator('[aria-current="page"]').getAttribute("href"),
        path,
      );
    }
  });

  it("supports workspace menu keyboard navigation and restores focus on Escape", async () => {
    await load();
    const trigger = sidebar().getByRole("button", {
      name: "API workspace Core",
      exact: true,
    });
    await trigger.focus();
    await trigger.press("ArrowDown");
    await page.getByRole("menu").waitFor();
    await page.keyboard.press("End");
    await page.keyboard.press("Enter");
    await waitLocation("/visibility");
    const next = sidebar().getByRole("button", {
      name: "API workspace Visibility",
      exact: true,
    });
    await next.click();
    await page.keyboard.press("Escape");
    await page.getByRole("menu").waitFor({ state: "hidden" });
    await page.waitForFunction(
      () =>
        document.activeElement?.getAttribute("aria-label") ===
        "API workspace Visibility",
    );
    assert.equal(
      await next.evaluate((element) => element === document.activeElement),
      true,
    );
  });

  it("includes every existing resource once in the functional navigation", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { navSections } =
        await import("/src/components/Navigation/catalog.ts");
      const { functionGroups } =
        await import("/src/components/SideBar/functions.ts");
      const entries = functionGroups.flatMap((group) => group.items);
      return {
        api: navSections
          .flatMap((section) => section.items.map((item) => item.url))
          .sort(),
        functions: entries.map((item) => item.url).sort(),
        titles: entries.map((item) => item.title),
      };
    });
    assert.deepEqual(result.functions, result.api);
    assert.equal(new Set(result.functions).size, result.functions.length);
    for (const title of [
      "Core policies",
      "Access policies",
      "Workspace regions",
      "Core secrets",
      "Enterprise secrets",
      "Access secrets",
      "Cordium secrets",
      "Core configuration",
      "Enterprise configuration",
    ]) {
      assert.ok(result.titles.includes(title));
    }
  });

  it("switches organization without navigating and reveals the active group", async () => {
    await load("/enterprise/certificates/tls/edit");
    await functions();
    await sidebar()
      .getByRole("button", { name: "Network", exact: true })
      .waitFor();
    assert.equal(
      await page.getByTestId("location").textContent(),
      "/enterprise/certificates/tls/edit",
    );
    assert.equal(
      await sidebar()
        .getByRole("button", { name: "Network", exact: true })
        .getAttribute("aria-expanded"),
      "true",
    );
    assert.equal(
      await sidebar()
        .getByRole("link", { name: "Certificates", exact: true })
        .getAttribute("aria-current"),
      "page",
    );
    await sidebar()
      .getByRole("button", { name: "Identity", exact: true })
      .click();
    await sidebar().getByRole("link", { name: "Users", exact: true }).click();
    await waitLocation("/core/users");
    await sidebar()
      .getByRole("button", { name: "Identity", exact: true })
      .click();
    assert.equal(
      await sidebar()
        .getByRole("link", { name: "Users", exact: true })
        .isVisible(),
      false,
    );
    await page.evaluate(() => window.sidebarNavigate("/core/users/alice/edit"));
    await sidebar().getByRole("link", { name: "Users", exact: true }).waitFor();
    assert.equal(
      await sidebar()
        .getByRole("button", { name: "Identity", exact: true })
        .getAttribute("aria-expanded"),
      "true",
    );
    await sidebar().getByText("API", { exact: true }).click();
    assert.equal(
      await sidebar()
        .getByRole("button", { name: "API workspace Core", exact: true })
        .count(),
      1,
    );
  });

  it("persists the mode and collapsed width and keeps the icon rail usable", async () => {
    await load();
    await sidebar().getByRole("button", { name: "Collapse sidebar" }).click();
    await waitWidth(64);
    const services = sidebar().getByRole("link", {
      name: "Services",
      exact: true,
    });
    await page.keyboard.press("Tab");
    await services.focus();
    await page.getByRole("tooltip").filter({ hasText: "Services" }).waitFor();
    await services.press("Enter");
    await waitLocation("/core/services");
    await sidebar()
      .getByRole("button", { name: "Navigation view: API", exact: true })
      .click();
    await page
      .getByRole("menuitemradio", { name: "Functions", exact: true })
      .click();
    await sidebar()
      .getByRole("button", { name: "Network", exact: true })
      .click();
    await page
      .getByRole("menuitem", { name: "Certificates", exact: true })
      .click();
    await waitLocation("/enterprise/certificates");
    await page.reload();
    await sidebar().getByRole("button", { name: "Expand sidebar" }).waitFor();
    await waitWidth(64);
    assert.equal(
      await sidebar()
        .getByRole("button", {
          name: "Navigation view: Functions",
          exact: true,
        })
        .count(),
      1,
    );
    await sidebar().getByRole("button", { name: "Expand sidebar" }).click();
    await waitWidth(264);
    assert.equal(
      await sidebar()
        .getByRole("radio", { name: "Functions", exact: true })
        .isChecked(),
      true,
    );
  });

  it("keeps the selected resource visible within a short sidebar", async () => {
    await page.setViewportSize({ width: 1024, height: 640 });
    await load("/core/clusterconfig");
    const selected = sidebar().getByRole("link", {
      name: "ClusterConfig",
      exact: true,
    });
    await page.waitForFunction(() => {
      const nav = document.querySelector('nav[aria-label="Console"]');
      const selected = nav.querySelector('[aria-current="page"]');
      const bounds = nav.getBoundingClientRect();
      const item = selected.getBoundingClientRect();
      return item.top >= bounds.top && item.bottom <= bounds.bottom;
    });
    const bounds = await selected.boundingBox();
    const nav = await sidebar()
      .getByRole("navigation", { name: "Console", exact: true })
      .boundingBox();
    assert.ok(
      bounds.y >= nav.y && bounds.y + bounds.height <= nav.y + nav.height,
    );
    await sidebar().getByRole("button", { name: "Collapse sidebar" }).click();
    await waitWidth(64);
    const collapsed = await selected.boundingBox();
    const rail = await sidebar()
      .getByRole("navigation", { name: "Console", exact: true })
      .boundingBox();
    assert.ok(
      collapsed.y >= rail.y &&
        collapsed.y + collapsed.height <= rail.y + rail.height,
    );
  });

  it("falls back to API and expanded navigation for invalid stored preferences", async () => {
    await page.addInitScript(() => {
      localStorage.setItem(
        "octelium-console-navigation-mode",
        '{"invalid":true}',
      );
      localStorage.setItem("octelium-console-sidebar-collapsed", '"false"');
    });
    await load();
    assert.equal(
      await sidebar()
        .getByRole("radio", { name: "API", exact: true })
        .isChecked(),
      true,
    );
    await waitWidth(264);
  });

  it("remains usable when local storage is unavailable", async () => {
    await page.addInitScript(() => {
      Storage.prototype.getItem = () => {
        throw new Error("Storage blocked");
      };
      Storage.prototype.setItem = () => {
        throw new Error("Storage blocked");
      };
    });
    await load();
    await functions();
    await sidebar()
      .getByRole("button", { name: "Access", exact: true })
      .waitFor();
    await sidebar().getByRole("button", { name: "Collapse sidebar" }).click();
    await waitWidth(64);
  });

  it("uses a modal mobile drawer with focus trapping, route dismissal, and focus return", async () => {
    await page.setViewportSize({ width: 390, height: 844 });
    await load();
    const trigger = page.getByRole("button", { name: "Open navigation" });
    await trigger.click();
    await drawer().getByRole("button", { name: "Close navigation" }).waitFor();
    assert.equal(await drawer().getAttribute("aria-modal"), "true");
    for (let index = 0; index < 25; index++) await page.keyboard.press("Tab");
    assert.equal(
      await drawer().evaluate((element) =>
        element.contains(document.activeElement),
      ),
      true,
    );
    await page.keyboard.press("Escape");
    await drawer().waitFor({ state: "hidden" });
    assert.equal(
      await trigger.evaluate((element) => element === document.activeElement),
      true,
    );
    await trigger.click();
    await drawer().getByRole("link", { name: "Users", exact: true }).click();
    await waitLocation("/core/users");
    await drawer().waitFor({ state: "hidden" });
    await trigger.click();
    await page.mouse.click(380, 400);
    await drawer().waitFor({ state: "hidden" });
  });

  it("keeps mobile navigation expanded without changing the saved desktop preference", async () => {
    await load();
    await sidebar().getByRole("button", { name: "Collapse sidebar" }).click();
    await waitWidth(64);
    await page.setViewportSize({ width: 640, height: 700 });
    await page.getByRole("button", { name: "Open navigation" }).click();
    await drawer().getByText("API", { exact: true }).waitFor();
    assert.equal(
      await drawer().getByRole("button", { name: "Expand sidebar" }).count(),
      0,
    );
    await drawer().getByText("Functions", { exact: true }).click();
    await page.setViewportSize({ width: 1024, height: 768 });
    await drawer().waitFor({ state: "hidden" });
    await waitWidth(64);
    assert.equal(
      await sidebar()
        .getByRole("button", {
          name: "Navigation view: Functions",
          exact: true,
        })
        .count(),
      1,
    );
    await page.setViewportSize({ width: 640, height: 700 });
    assert.equal(await drawer().count(), 0);
  });

  for (const theme of ["light", "dark"]) {
    it(`fits narrow, tablet, and desktop viewports in ${theme} mode`, async () => {
      await load("/core/services", theme);
      for (const width of [320, 390, 640, 767, 768, 1024, 1440]) {
        await page.setViewportSize({ width, height: 640 });
        if (width < 768) {
          await page.getByRole("button", { name: "Open navigation" }).click();
          await drawer()
            .getByRole("link", { name: "Services", exact: true })
            .waitFor();
          await drawer()
            .getByRole("button", { name: "API workspace Core", exact: true })
            .click();
          await page
            .getByRole("menuitemradio", { name: "Visibility", exact: true })
            .waitFor();
          const menu = await page.getByRole("menu").boundingBox();
          assert.ok(menu.x >= 0 && menu.x + menu.width <= width);
          await page.keyboard.press("Escape");
          await drawer()
            .getByRole("button", { name: "Close navigation" })
            .click();
          await drawer().waitFor({ state: "hidden" });
        } else {
          await sidebar()
            .getByRole("button", { name: "Collapse sidebar" })
            .waitFor();
          await waitWidth(264);
        }
        assert.equal(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= window.innerWidth,
          ),
          true,
        );
      }
      const services = sidebar().getByRole("link", {
        name: "Services",
        exact: true,
      });
      assert.equal(await services.getAttribute("aria-current"), "page");
      const style = await services.evaluate((element) => ({
        color: getComputedStyle(element).color,
        background: getComputedStyle(element).backgroundColor,
      }));
      assert.notEqual(style.color, style.background);
    });
  }

  it("disables sidebar layout transitions when reduced motion is requested", async () => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await load();
    await sidebar().getByRole("button", { name: "Collapse sidebar" }).click();
    await waitWidth(64);
    assert.equal(
      await sidebar().evaluate(
        (element) => getComputedStyle(element).transitionDuration,
      ),
      "0s",
    );
    assert.equal(
      await page
        .locator(".mantine-AppShell-main")
        .evaluate((element) => getComputedStyle(element).transitionDuration),
      "0s",
    );
  });
});
