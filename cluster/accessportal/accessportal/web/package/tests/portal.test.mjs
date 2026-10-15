import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium, firefox } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));
const storageKey = "octelium-color-scheme";

for (const [name, browserType] of Object.entries({ chromium, firefox })) {
  describe(`Access portal in ${name}`, () => {
    let cacheDir;
    let server;
    let browser;
    let context;
    let page;
    let origin;
    let errors;

    before(async () => {
      cacheDir = await mkdtemp(join(root, "node_modules/.vite-portal-"));
      server = await createServer({
        root,
        cacheDir,
        optimizeDeps: {
          entries: ["tests/fixtures/portal.tsx"],
          include: ["react/jsx-runtime"],
        },
        plugins: [{
          name: "portal-test-entry",
          transformIndexHtml: {
            order: "pre",
            handler: (html) => html.replace("/src/main.tsx", "/tests/fixtures/portal.tsx"),
          },
        }],
        server: { host: "127.0.0.1", port: 0, watch: null, hmr: false },
      });
      await server.listen();
      origin = server.resolvedUrls.local[0];
      browser = await browserType.launch();
    });

    after(async () => {
      await browser?.close();
      await server?.close();
      if (cacheDir) await rm(cacheDir, { recursive: true, force: true });
    });

    beforeEach(async () => {
      context = await browser.newContext({
        colorScheme: "dark",
        reducedMotion: "reduce",
        viewport: { width: 1440, height: 1000 },
      });
      page = await context.newPage();
      page.setDefaultTimeout(15_000);
      errors = [];
      page.on("pageerror", (error) => errors.push(error.message));
    });

    afterEach(async () => {
      await context?.close();
      assert.deepEqual(errors, []);
    });

    const load = async (path = "/user/new") => {
      await page.goto(`${origin}?${new URLSearchParams({ path })}`);
      await page.getByRole("heading", { level: 1 }).waitFor();
    };

    const scheme = () => page.locator("html").getAttribute("data-mantine-color-scheme");

    it("uses the system theme before the application starts", async () => {
      await page.route("**/tests/fixtures/portal.tsx", (route) => route.abort());
      for (const colorScheme of ["dark", "light"]) {
        await page.emulateMedia({ colorScheme });
        await page.goto(origin);
        assert.equal(await scheme(), colorScheme);
        assert.equal(await page.evaluate((key) => localStorage.getItem(key), storageKey), null);
      }
    });

    it("follows system changes until an explicit choice is saved", async () => {
      await load("/settings");
      assert.equal(await scheme(), "dark");
      assert.equal(await page.getByRole("radio", { name: "System", exact: true }).isChecked(), true);
      await page.emulateMedia({ colorScheme: "light" });
      await page.waitForFunction(() => document.documentElement.dataset.mantineColorScheme === "light");
      assert.equal(await page.evaluate((key) => localStorage.getItem(key), storageKey), null);
      await page.getByText("Dark", { exact: true }).click();
      assert.equal(await scheme(), "dark");
      await page.reload();
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
      assert.equal(await scheme(), "dark");
      assert.equal(await page.evaluate((key) => localStorage.getItem(key), storageKey), "dark");
      await page.getByText("System", { exact: true }).click();
      await page.waitForFunction(() => document.documentElement.dataset.mantineColorScheme === "light");
      await page.emulateMedia({ colorScheme: "dark" });
      await page.waitForFunction(() => document.documentElement.dataset.mantineColorScheme === "dark");
    });

    it("respects a saved light choice during startup and navigation", async () => {
      await page.addInitScript((key) => localStorage.setItem(key, "light"), storageKey);
      await load();
      assert.equal(await scheme(), "light");
      await page.getByRole("link", { name: "Settings", exact: true }).click();
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
      assert.equal(await scheme(), "light");
    });

    it("falls back to the system for invalid or unavailable storage", async () => {
      await page.addInitScript((key) => localStorage.setItem(key, "invalid"), storageKey);
      await load();
      assert.equal(await scheme(), "dark");
      await page.addInitScript(() => {
        Storage.prototype.getItem = () => { throw new Error("Storage unavailable"); };
      });
      await page.reload();
      await page.getByRole("heading", { level: 1 }).waitFor();
      assert.equal(await scheme(), "dark");
    });

    it("offers all six urgency levels in a keyboard accessible selector", async () => {
      await load();
      const urgency = page.getByRole("combobox", { name: "Urgency", exact: true });
      assert.equal(await urgency.inputValue(), "Normal");
      for (const label of ["Very Low", "Low", "Normal", "High", "Very High", "Highest"]) {
        await urgency.click();
        await page.getByRole("option", { name: label, exact: true }).click();
        assert.equal(await urgency.inputValue(), label);
      }
      await urgency.focus();
      await page.keyboard.press("ArrowDown");
      await page.getByRole("listbox").waitFor();
      await page.keyboard.press("ArrowUp");
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelector('input[aria-label="Urgency"]').value === "Very High");
    });

    it("filters services by namespace and combines it with search", async () => {
      await load();
      const namespace = page.getByRole("combobox", { name: "Namespace", exact: true });
      const search = page.getByRole("textbox", { name: "Search services", exact: true });
      await page.getByRole("button", { name: /Database/ }).waitFor();
      assert.equal(await namespace.inputValue(), "All namespaces");
      await namespace.fill("this");
      await page.getByRole("option", { name: "this-namespace", exact: true }).click();
      await page.getByRole("button", { name: /Dashboard/ }).waitFor();
      assert.equal(await page.getByRole("button", { name: /Database/ }).count(), 0);
      assert.ok((await page.evaluate(() => window.calls)).some(
        (call) => call.method === "listService" && call.request.namespace === "this-namespace",
      ));
      await search.fill("database");
      await page.getByText("No services found", { exact: true }).waitFor();
      await namespace.click();
      await page.getByRole("option", { name: "All namespaces", exact: true }).click();
      await page.getByRole("button", { name: /Database/ }).waitFor();
      assert.equal(await search.inputValue(), "database");
    });

    it("keeps route changes and Select usable with an inaccessible shadow root", async () => {
      await load();
      await page.evaluate(() => {
        const forbidden = document.createElement("div");
        Object.defineProperty(forbidden, "shadowRoot", {
          get() { throw new Error('Permission denied to access property "shadowRoot"'); },
        });
        document.body.append(forbidden);
      });
      for (const path of ["/settings", "/reviewer/requests", "/user/requests", "/user/new"]) {
        await page.locator(`css:light=a[href="${path}"]`).first().click();
        await page.locator(`css:light=a[href="${path}"][aria-current="page"]`).waitFor();
        assert.equal(await page.locator('css:light=a[aria-current="page"]').getAttribute("href"), path);
      }
      await page.locator('css:light=input[aria-label="Urgency"]').click();
      await page.locator('css:light=[role="option"]').filter({ hasText: /^High$/ }).click();
      assert.equal(await page.locator('css:light=input[aria-label="Urgency"]').inputValue(), "High");
      assert.equal(await page.locator("css:light=h1").textContent(), "New Request");
    });

    it("preserves nested shadow searches while skipping inaccessible hosts", async () => {
      await load();
      const result = await page.evaluate(async () => {
        const { findElementBySelector, findElementsBySelector, getRootElement } =
          await import("/src/utils/mantine/find-element-in-shadow-dom.js");
        const host = document.createElement("div");
        const shadow = host.attachShadow({ mode: "open" });
        const forbidden = document.createElement("div");
        Object.defineProperty(forbidden, "shadowRoot", { get() { throw new Error("Denied"); } });
        shadow.append(forbidden);
        const nested = document.createElement("div");
        const child = nested.attachShadow({ mode: "open" });
        child.innerHTML = '<span class="target">Found</span>';
        shadow.append(nested);
        document.body.append(host);
        return [
          findElementBySelector(".target", shadow)?.textContent,
          findElementsBySelector(".target", shadow).length,
          getRootElement(child.querySelector("span")) === child,
          findElementBySelector(".missing"),
          findElementsBySelector(".missing").length,
        ];
      });
      assert.deepEqual(result, ["Found", 1, true, null, 0]);
    });

    it("keeps account actions visible in both themes and allows canceling sign out", async () => {
      await load();
      for (const colorScheme of ["dark", "light"]) {
        await page.emulateMedia({ colorScheme });
        await page.waitForFunction((value) => document.documentElement.dataset.mantineColorScheme === value, colorScheme);
        const authenticator = page.getByRole("link", { name: "Authenticators", exact: true });
        const signOut = page.getByRole("button", { name: "Sign out", exact: true });
        for (const action of [authenticator, signOut]) {
          assert.equal(await action.isVisible(), true);
          const contrast = await action.evaluate((element) => {
            const canvas = document.createElement("canvas");
            canvas.width = canvas.height = 1;
            const context = canvas.getContext("2d");
            const luminance = (color) => {
              context.fillStyle = color;
              context.fillRect(0, 0, 1, 1);
              const [r, g, b] = [...context.getImageData(0, 0, 1, 1).data]
                .slice(0, 3)
                .map((value) => value / 255)
                .map((value) => value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4);
              return 0.2126 * r + 0.7152 * g + 0.0722 * b;
            };
            let parent = element;
            while (parent && getComputedStyle(parent).backgroundColor === "rgba(0, 0, 0, 0)") {
              parent = parent.parentElement;
            }
            const foreground = luminance(getComputedStyle(element).color);
            const background = luminance(getComputedStyle(parent).backgroundColor);
            return (Math.max(foreground, background) + 0.05) / (Math.min(foreground, background) + 0.05);
          });
          assert.ok(contrast >= 4.5, `${colorScheme} account action contrast: ${contrast}`);
        }
        await signOut.click();
        const dialog = page.getByRole("dialog", { name: "Sign out", exact: true });
        await dialog.waitFor();
        await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
        await dialog.waitFor({ state: "hidden" });
      }
      assert.equal((await page.evaluate(() => window.calls)).filter((call) => call.method === "logout").length, 0);
    });

    it("lists only the services of a namespace scoped catalog", async () => {
      await load();
      await page.getByText("Catalogs", { exact: true }).click();
      await page.getByRole("button", { name: "View the contents of Team namespace", exact: true }).click();
      const drawer = page.getByRole("dialog", { name: "Team namespace", exact: true });
      await drawer.getByText("Dashboard", { exact: true }).waitFor();
      assert.equal(await drawer.getByText("Database", { exact: true }).count(), 0);
      assert.equal(await drawer.getByText("0", { exact: true }).count(), 0);
    });

    it("rejects fractional durations before they reach the API", async () => {
      await load();
      const amount = page.getByRole("textbox", { name: "Duration amount", exact: true });
      await amount.fill("");
      await amount.pressSequentially("1.5");
      assert.equal((await amount.inputValue()).includes("."), false);
    });

    it("shows the recipient only for requests made on behalf of another user", async () => {
      await load("/user/requests");
      await page.getByText("for", { exact: false }).filter({ hasText: "bob" }).first().waitFor();
      assert.equal(await page.getByText(/for\s+alice/).count(), 0);
    });

    it("shows the effective duration of a capped grant", async () => {
      await load("/user/requests/capped");
      await page.getByText("Granted for", { exact: true }).waitFor();
      assert.equal(await page.getByText("48 hours", { exact: true }).count() > 0, true);
      assert.equal(await page.getByText("2 hours", { exact: true }).count() > 0, true);
      assert.equal(
        await page.locator("section", { hasText: "Access window" }).getByText("48 hours", { exact: true }).count(),
        0,
      );
    });

    it("keeps a workflow cleared after its approved access expired", async () => {
      await load("/user/requests/expired-after-approval");
      await page.getByText("All 1 step cleared", { exact: true }).waitFor();
      assert.equal(await page.getByText("Stopped here", { exact: true }).count(), 0);
    });

    it("shows unknown pages and missing requests without retrying", async () => {
      await load("/nope");
      assert.equal(await page.getByRole("heading", { level: 1 }).textContent(), "Page not found");
      await load("/user/requests/missing");
      assert.equal(await page.getByRole("heading", { level: 1 }).textContent(), "Request not found");
      assert.ok(
        (await page.evaluate(() => window.calls)).filter(
          (call) => call.method === "getRequest" && call.request.name === "missing",
        ).length <= 2,
      );
    });

    it("loads every page of the review queue", async () => {
      await load("/reviewer/requests");
      await page.getByText("2 requests awaiting you", { exact: true }).waitFor();
      assert.deepEqual(
        [...new Set((await page.evaluate(() => window.calls))
          .filter((call) => call.method === "listRequest" && call.request.common?.itemsPerPage === 500)
          .map((call) => call.request.common.page))],
        [0, 1],
      );
    });

    it("binds review decisions to the displayed step", async () => {
      await load("/reviewer/requests/queue-first");
      await page.getByRole("button", { name: "Reject", exact: true }).filter({ visible: true }).click();
      await page.getByText("Add a reason before rejecting this request", { exact: true }).filter({ visible: true }).waitFor();
      assert.equal(await page.getByRole("dialog").count(), 0);
      await page.getByRole("button", { name: "Approve", exact: true }).filter({ visible: true }).click();
      const dialog = page.getByRole("dialog", { name: "Approve this request?", exact: true });
      await dialog.getByRole("button", { name: "Approve request", exact: true }).click();
      await page.getByRole("heading", { name: "Review Queue", exact: true }).waitFor();
      const calls = await page.evaluate(() => window.calls);
      const decision = calls.find((call) => call.method === "setReviewDecision");
      assert.equal(decision.request.requestRef.name, "queue-first");
      assert.equal(decision.request.expectedStepName, "security");
      assert.equal(decision.request.decision, 1);
      assert.equal(calls.filter((call) => call.method === "createReview").length, 0);
    });

    it("requires an explicit decision when editing a reset review", async () => {
      await load("/reviewer/reviews/rv1.queue-second");
      await page.getByRole("button", { name: "Set decision", exact: true }).click();
      const save = page.getByRole("button", { name: "Save", exact: true });
      assert.equal(await save.isDisabled(), true);
      await page.getByText("Approve", { exact: true }).click();
      await save.click();
      await page.waitForFunction(() => window.calls.some((call) => call.method === "updateReview"));
      const update = (await page.evaluate(() => window.calls)).find((call) => call.method === "updateReview");
      assert.equal(update.request.spec.decision, 1);
    });

    it("offers no changes for reviews applied to an earlier step", async () => {
      await load("/reviewer/reviews/rv2.queue-first");
      await page.getByText(/applied to an earlier step/).waitFor();
      assert.equal(await page.getByRole("button", { name: "Change decision", exact: true }).count(), 0);
      assert.equal(await page.getByRole("button", { name: "Reset decision", exact: true }).count(), 0);
    });

    it("closes mobile navigation after routing and avoids horizontal overflow", async () => {
      await page.setViewportSize({ width: 390, height: 844 });
      await load();
      const open = page.getByRole("button", { name: "Open navigation", exact: true });
      await open.click();
      const drawer = page.getByRole("dialog", { name: "Portal navigation", exact: true });
      await drawer.waitFor();
      await drawer.getByRole("link", { name: "Settings", exact: true }).click();
      await drawer.waitFor({ state: "hidden" });
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
      await open.click();
      await drawer.getByRole("link", { name: "New Request", exact: true }).click();
      await drawer.waitFor({ state: "hidden" });
      await page.getByRole("combobox", { name: "Namespace", exact: true }).waitFor();
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      await open.click();
      await page.keyboard.press("Escape");
      await drawer.waitFor({ state: "hidden" });
      assert.equal(await open.getAttribute("aria-expanded"), "false");
    });
  });
}
