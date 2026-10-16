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

    const load = async (path = "/user/new", options = {}) => {
      await page.goto(`${origin}?${new URLSearchParams({ path, ...options })}`);
      await page.getByRole("heading", { level: 1 }).waitFor();
    };

    const scheme = () => page.locator("html").getAttribute("data-mantine-color-scheme");

    const discardDialog = () => page.getByRole("dialog", { name: "Discard unsaved changes?", exact: true });

    const warnsBeforeUnload = () => page.evaluate(() => {
      const event = new Event("beforeunload", { cancelable: true });
      window.dispatchEvent(event);
      return event.defaultPrevented;
    });

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
          await action.evaluate(async (element) => {
            await Promise.all(element.getAnimations().map((animation) => animation.finished.catch(() => null)));
          });
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
      await page.getByRole("button", { name: "Custom duration", exact: true }).click();
      const amount = page.getByRole("textbox", { name: "Duration amount", exact: true });
      await amount.fill("");
      await amount.pressSequentially("1.5");
      assert.equal((await amount.inputValue()).includes("."), false);
    });

    it("offers the access duration presets in both themes", async () => {
      await load();
      const duration = page.getByRole("group", { name: "Duration", exact: true });
      const summary = page.locator("section", { has: page.getByRole("heading", { name: "Access details", exact: true }) })
        .locator("p", { hasText: "Alice Morgan" });
      const presets = ["30 minutes", "1 hour", "4 hours", "8 hours", "1 day"];
      for (const colorScheme of ["light", "dark"]) {
        await page.emulateMedia({ colorScheme });
        await page.waitForFunction((value) => document.documentElement.dataset.mantineColorScheme === value, colorScheme);
        await duration.getByRole("button", { name: "1 hour", exact: true }).click();
        assert.equal(await duration.getByRole("button", { name: "1 hour", exact: true }).getAttribute("aria-pressed"), "true");
        assert.equal(await page.getByRole("textbox", { name: "Duration amount", exact: true }).count(), 0);
        for (const label of presets) {
          const preset = duration.getByRole("button", { name: label, exact: true });
          await preset.click();
          assert.equal(await preset.getAttribute("aria-pressed"), "true");
          assert.equal(await duration.locator('[aria-pressed="true"]').count(), 1);
          assert.ok((await summary.textContent()).includes(label));
        }
      }
      assert.equal((await page.evaluate(() => window.calls)).some((call) => call.method === "createRequest"), false);
    });

    it("applies custom durations explicitly and discards canceled changes", async () => {
      await load();
      const custom = page.getByRole("button", { name: "Custom duration", exact: true });
      const dialog = page.getByRole("dialog", { name: "Custom duration", exact: true });
      const amount = dialog.getByRole("textbox", { name: "Duration amount", exact: true });
      const unit = dialog.getByRole("combobox", { name: "Duration unit", exact: true });
      await custom.click();
      assert.equal(await amount.inputValue(), "1");
      assert.equal(await unit.inputValue(), "Hours");
      await amount.fill("3");
      assert.equal(await warnsBeforeUnload(), false);
      await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
      await dialog.waitFor({ state: "hidden" });
      assert.equal(await warnsBeforeUnload(), false);
      await custom.click();
      assert.equal(await amount.inputValue(), "1");
      await amount.fill("90");
      await unit.click();
      await page.getByRole("option", { name: "Minutes", exact: true }).click();
      await dialog.getByText("Access for 90 minutes", { exact: true }).waitFor();
      await dialog.getByRole("button", { name: "Apply duration", exact: true }).click();
      await dialog.waitFor({ state: "hidden" });
      await page.getByText("Custom duration: 90 minutes", { exact: true }).waitFor();
      assert.equal(await custom.getAttribute("aria-pressed"), "true");
      assert.equal(await warnsBeforeUnload(), true);
      await custom.click();
      assert.equal(await amount.inputValue(), "90");
      assert.equal(await unit.inputValue(), "Minutes");
      await amount.fill("120");
      await page.keyboard.press("Escape");
      await dialog.waitFor({ state: "hidden" });
      await page.getByText("Custom duration: 90 minutes", { exact: true }).waitFor();
      await custom.click();
      assert.equal(await amount.inputValue(), "90");
    });

    it("requires a positive custom duration and supports keyboard apply", async () => {
      await load();
      const custom = page.getByRole("button", { name: "Custom duration", exact: true });
      await custom.focus();
      await page.keyboard.press("Enter");
      const dialog = page.getByRole("dialog", { name: "Custom duration", exact: true });
      const amount = dialog.getByRole("textbox", { name: "Duration amount", exact: true });
      const apply = dialog.getByRole("button", { name: "Apply duration", exact: true });
      await amount.fill("");
      assert.equal(await apply.isDisabled(), true);
      await dialog.getByText("Enter a whole number from 1 to 100,000", { exact: true }).waitFor();
      await amount.press("Enter");
      assert.equal(await dialog.isVisible(), true);
      assert.equal(await warnsBeforeUnload(), false);
      await amount.fill("6");
      await amount.press("Enter");
      await dialog.waitFor({ state: "hidden" });
      await page.getByText("Custom duration: 6 hours", { exact: true }).waitFor();
      assert.equal(await custom.evaluate((element) => element === document.activeElement), true);
    });

    for (const [label, kind] of [["Days", "days"], ["Weeks", "weeks"]]) {
      it(`submits custom access durations in ${kind}`, async () => {
        await load();
        await page.getByRole("button", { name: /Database/ }).click();
        const trigger = page.getByRole("button", { name: "Custom duration", exact: true });
        const dialog = page.getByRole("dialog", { name: "Custom duration", exact: true });
        const amount = dialog.getByRole("textbox", { name: "Duration amount", exact: true });
        const unit = dialog.getByRole("combobox", { name: "Duration unit", exact: true });
        await trigger.click();
        await amount.fill("2");
        await unit.click();
        await page.getByRole("option", { name: label, exact: true }).click();
        await dialog.getByText(`Access for 2 ${kind}`, { exact: true }).waitFor();
        await dialog.getByRole("button", { name: "Apply duration", exact: true }).click();
        await dialog.waitFor({ state: "hidden" });
        await page.getByText(`Custom duration: 2 ${kind}`, { exact: true }).waitFor();
        await trigger.click();
        assert.equal(await amount.inputValue(), "2");
        assert.equal(await unit.inputValue(), label);
        await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
        await dialog.waitFor({ state: "hidden" });
        await page.getByRole("button", { name: "Submit Request", exact: true }).click();
        await page.getByRole("dialog", { name: "Submit this access request?", exact: true })
          .getByRole("button", { name: "Submit request", exact: true }).click();
        await page.getByText("Access request submitted", { exact: true }).waitFor();
        const create = (await page.evaluate(() => window.calls)).find((call) => call.method === "createRequest");
        assert.equal(create.request.spec.duration.type.oneofKind, kind);
        assert.equal(create.request.spec.duration.type[kind], 2);
      });
    }

    it("edits existing custom durations without submitting the request form", async () => {
      await load("/user/requests/own-pending");
      await page.getByRole("button", { name: "Edit", exact: true }).click();
      const edit = page.getByRole("dialog", { name: "Edit pending request", exact: true });
      await edit.getByText("Custom duration: 2 hours", { exact: true }).waitFor();
      await edit.getByRole("button", { name: "Custom duration", exact: true }).click();
      const custom = page.getByRole("dialog", { name: "Custom duration", exact: true });
      const amount = custom.getByRole("textbox", { name: "Duration amount", exact: true });
      assert.equal(await amount.inputValue(), "2");
      await amount.fill("3");
      await custom.getByRole("combobox", { name: "Duration unit", exact: true }).click();
      await page.keyboard.press("Escape");
      await page.getByRole("listbox").waitFor({ state: "hidden" });
      assert.equal(await custom.isVisible(), true);
      await amount.focus();
      await page.keyboard.press("Escape");
      await custom.waitFor({ state: "hidden" });
      assert.equal(await edit.isVisible(), true);
      assert.equal(await warnsBeforeUnload(), false);
      await edit.getByRole("button", { name: "Custom duration", exact: true }).click();
      assert.equal(await amount.inputValue(), "2");
      await amount.fill("24");
      await amount.press("Enter");
      await custom.waitFor({ state: "hidden" });
      assert.equal(await edit.getByRole("button", { name: "1 day", exact: true }).getAttribute("aria-pressed"), "true");
      assert.equal((await page.evaluate(() => window.calls)).some((call) => call.method === "updateRequest"), false);
      await edit.getByRole("button", { name: "Save changes", exact: true }).click();
      await edit.waitFor({ state: "hidden" });
      const update = (await page.evaluate(() => window.calls)).find((call) => call.method === "updateRequest");
      assert.equal(update.request.spec.duration.type.oneofKind, "hours");
      assert.equal(update.request.spec.duration.type.hours, 24);
    });

    it("keeps duration presets and the custom popover within narrow screens", async () => {
      await page.setViewportSize({ width: 320, height: 844 });
      await load();
      const duration = page.getByRole("group", { name: "Duration", exact: true });
      const custom = duration.getByRole("button", { name: "Custom duration", exact: true });
      for (const colorScheme of ["light", "dark"]) {
        await page.emulateMedia({ colorScheme });
        await custom.click();
        const dialog = page.getByRole("dialog", { name: "Custom duration", exact: true });
        await dialog.getByRole("textbox", { name: "Duration amount", exact: true }).waitFor();
        const bounds = await dialog.boundingBox();
        assert.ok(bounds.x >= 0);
        assert.ok(bounds.x + bounds.width <= 320);
        assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
        await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
        await dialog.waitFor({ state: "hidden" });
      }
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

    it("keeps the topbar account icon without displaying the user's name", async () => {
      await load("/settings");
      const account = page.getByRole("navigation", { name: "Account", exact: true });
      await account.getByRole("link", { name: "Account settings for Alice Morgan", exact: true }).waitFor();
      assert.equal(await account.getByText("Alice Morgan", { exact: true }).count(), 0);
      assert.equal(await account.getByText("alice", { exact: true }).count(), 0);
    });

    it("keeps a new request draft until navigation is confirmed", async () => {
      await load();
      assert.equal(await warnsBeforeUnload(), false);
      const reason = page.getByRole("textbox", { name: /Justification/ });
      await reason.fill("Investigating a database incident");
      assert.equal(await warnsBeforeUnload(), true);
      await page.getByRole("link", { name: "Settings", exact: true }).click();
      await discardDialog().getByRole("button", { name: "Keep editing", exact: true }).click();
      assert.equal(await reason.inputValue(), "Investigating a database incident");
      await page.getByRole("link", { name: "Settings", exact: true }).click();
      await discardDialog().getByRole("button", { name: "Discard changes", exact: true }).click();
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
      assert.equal(await warnsBeforeUnload(), false);
    });

    it("does not warn after new request fields are restored", async () => {
      await load();
      const reason = page.getByRole("textbox", { name: /Justification/ });
      await reason.fill("Temporary draft");
      await page.getByRole("button", { name: "4 hours", exact: true }).click();
      assert.equal(await warnsBeforeUnload(), true);
      await reason.fill("");
      assert.equal(await warnsBeforeUnload(), true);
      await page.getByRole("button", { name: "1 hour", exact: true }).click();
      assert.equal(await warnsBeforeUnload(), false);
      await page.getByRole("link", { name: "Settings", exact: true }).click();
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
      assert.equal(await discardDialog().count(), 0);
    });

    it("does not treat a prefilled resource as an unsaved change", async () => {
      await load("/user/new?serviceRef.name=default.database");
      await page.getByRole("button", { name: "Submit Request", exact: true }).waitFor();
      assert.equal(await warnsBeforeUnload(), false);
      await page.getByRole("link", { name: "Settings", exact: true }).click();
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
      assert.equal(await discardDialog().count(), 0);
    });

    it("protects new request drafts when going back", async () => {
      await load("/settings", { browserHistory: "1" });
      await page.getByRole("link", { name: "New Request", exact: true }).click();
      await page.getByRole("textbox", { name: /Justification/ }).fill("Unsaved reason");
      await page.goBack();
      await discardDialog().getByRole("button", { name: "Keep editing", exact: true }).click();
      assert.equal(await page.getByRole("textbox", { name: /Justification/ }).inputValue(), "Unsaved reason");
      await page.goBack();
      await discardDialog().getByRole("button", { name: "Discard changes", exact: true }).click();
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
    });

    it("warns before reloading a new request draft", async () => {
      await load();
      const reason = page.getByRole("textbox", { name: /Justification/ });
      await reason.fill("Keep this draft when reload is canceled");
      const dialogPromise = page.waitForEvent("dialog");
      const reload = page.evaluate(() => window.location.reload());
      const dialog = await dialogPromise;
      assert.equal(dialog.type(), "beforeunload");
      await dialog.dismiss();
      await reload;
      assert.equal(await reason.inputValue(), "Keep this draft when reload is canceled");
    });

    it("submits a new request without a discard prompt", async () => {
      await load();
      await page.getByRole("button", { name: /Database/ }).click();
      await page.getByRole("button", { name: "30 minutes", exact: true }).click();
      await page.getByRole("textbox", { name: /Justification/ }).fill("Investigating an incident");
      await page.getByRole("button", { name: "Submit Request", exact: true }).click();
      await page.getByRole("dialog", { name: "Submit this access request?", exact: true })
        .getByRole("button", { name: "Submit request", exact: true }).click();
      await page.getByText("Access request submitted", { exact: true }).waitFor();
      await page.getByRole("heading", { name: "default.database", exact: true }).waitFor();
      assert.equal(await discardDialog().count(), 0);
      assert.equal(await warnsBeforeUnload(), false);
      const create = (await page.evaluate(() => window.calls)).find((call) => call.method === "createRequest");
      assert.equal(create.request.spec.duration.type.oneofKind, "minutes");
      assert.equal(create.request.spec.duration.type.minutes, 30);
    });

    it("keeps a failed submission protected", async () => {
      await load("/user/new", { failCreate: "1" });
      await page.getByRole("button", { name: /Database/ }).click();
      await page.getByRole("textbox", { name: /Justification/ }).fill("Retry this request");
      await page.getByRole("button", { name: "Submit Request", exact: true }).click();
      const submit = page.getByRole("dialog", { name: "Submit this access request?", exact: true });
      await submit.getByRole("button", { name: "Submit request", exact: true }).click();
      await page.getByText("Submission failed", { exact: true }).waitFor();
      await submit.getByRole("button", { name: "Cancel", exact: true }).click();
      await page.getByRole("link", { name: "Settings", exact: true }).click();
      await discardDialog().getByRole("button", { name: "Keep editing", exact: true }).click();
      assert.equal(await page.getByRole("textbox", { name: /Justification/ }).inputValue(), "Retry this request");
      assert.equal(await warnsBeforeUnload(), true);
    });

    it("confirms closing a changed request edit and resets discarded fields", async () => {
      await load("/user/requests/own-pending");
      await page.getByRole("button", { name: "Edit", exact: true }).click();
      const edit = page.getByRole("dialog", { name: "Edit pending request", exact: true });
      assert.equal(await warnsBeforeUnload(), false);
      await edit.getByRole("textbox", { name: /Justification/ }).fill("Changed reason");
      await page.keyboard.press("Escape");
      await discardDialog().getByRole("button", { name: "Keep editing", exact: true }).click();
      assert.equal(await edit.getByRole("textbox", { name: /Justification/ }).inputValue(), "Changed reason");
      await edit.getByRole("button", { name: "Discard", exact: true }).click();
      await discardDialog().getByRole("button", { name: "Discard changes", exact: true }).click();
      await edit.waitFor({ state: "hidden" });
      assert.equal(await warnsBeforeUnload(), false);
      await page.getByRole("button", { name: "Edit", exact: true }).click();
      assert.equal(await edit.getByRole("textbox", { name: /Justification/ }).inputValue(), "");
      await edit.getByRole("button", { name: "Discard", exact: true }).click();
      await edit.waitFor({ state: "hidden" });
      assert.equal(await discardDialog().count(), 0);
    });

    it("protects request edits on navigation and clears the guard after saving", async () => {
      await load("/user/requests/own-pending");
      await page.getByRole("button", { name: "Edit", exact: true }).click();
      const edit = page.getByRole("dialog", { name: "Edit pending request", exact: true });
      await edit.getByRole("textbox", { name: /Justification/ }).fill("Save this reason");
      await page.evaluate(() => window.portalRouter.navigate("/settings"));
      await discardDialog().getByRole("button", { name: "Keep editing", exact: true }).click();
      await edit.getByRole("button", { name: "Save changes", exact: true }).click();
      await edit.waitFor({ state: "hidden" });
      assert.equal(await warnsBeforeUnload(), false);
      await page.getByRole("link", { name: "Settings", exact: true }).click();
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
      assert.equal(await discardDialog().count(), 0);
    });

    it("keeps review reasons on canceled navigation and submits without a discard prompt", async () => {
      await load("/reviewer/requests/queue-first");
      const reason = page.getByRole("textbox", { name: /Reason/ }).filter({ visible: true });
      await reason.fill("Approved for the incident response");
      await page.locator("#portal-main").getByRole("link", { name: "Review Queue", exact: true }).click();
      await discardDialog().getByRole("button", { name: "Keep editing", exact: true }).click();
      assert.equal(await reason.inputValue(), "Approved for the incident response");
      assert.equal(await warnsBeforeUnload(), true);
      await page.getByRole("button", { name: "Approve", exact: true }).filter({ visible: true }).click();
      await page.getByRole("dialog", { name: "Approve this request?", exact: true })
        .getByRole("button", { name: "Approve request", exact: true }).click();
      await page.getByRole("heading", { name: "Review Queue", exact: true }).waitFor();
      assert.equal(await discardDialog().count(), 0);
      assert.equal(await warnsBeforeUnload(), false);
    });

    it("protects changed review decisions and clears the guard after saving", async () => {
      await load("/reviewer/reviews/rv1.queue-second");
      await page.getByRole("button", { name: "Set decision", exact: true }).click();
      assert.equal(await warnsBeforeUnload(), false);
      await page.getByText("Approve", { exact: true }).click();
      assert.equal(await warnsBeforeUnload(), true);
      await page.getByRole("link", { name: "Settings", exact: true }).click();
      await discardDialog().getByRole("button", { name: "Keep editing", exact: true }).click();
      await page.getByRole("button", { name: "Discard", exact: true }).click();
      await discardDialog().getByRole("button", { name: "Keep editing", exact: true }).click();
      await page.getByRole("button", { name: "Save", exact: true }).click();
      await page.getByText("Review updated", { exact: true }).waitFor();
      assert.equal(await warnsBeforeUnload(), false);
      await page.getByRole("link", { name: "Settings", exact: true }).click();
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
      assert.equal(await discardDialog().count(), 0);
    });

    it("uses the navigation drawer below 1024px and restores the desktop sidebar", async () => {
      await page.setViewportSize({ width: 820, height: 1000 });
      await load("/user/requests");
      const open = page.getByRole("button", { name: "Open navigation", exact: true });
      const sidebar = page.getByRole("navigation", { name: "Portal sidebar", exact: true });
      await open.waitFor();
      const main = await page.locator("#portal-main").boundingBox();
      assert.ok(main.x < 40);
      await open.click();
      const drawer = page.getByRole("dialog", { name: "Portal navigation", exact: true });
      await drawer.getByRole("link", { name: "Settings", exact: true }).click();
      await drawer.waitFor({ state: "hidden" });
      await page.getByRole("heading", { name: "Settings", exact: true }).waitFor();
      await page.setViewportSize({ width: 1023, height: 1000 });
      await open.waitFor();
      await open.click();
      await drawer.waitFor();
      await page.setViewportSize({ width: 1024, height: 1000 });
      await open.waitFor({ state: "hidden" });
      await drawer.waitFor({ state: "hidden" });
      await sidebar.waitFor();
      await page.setViewportSize({ width: 820, height: 1000 });
      await open.waitFor();
      assert.equal(await open.getAttribute("aria-expanded"), "false");
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
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
