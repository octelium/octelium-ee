import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console forms, controls and lifecycle stores", () => {
  let cacheDir;
  let server;
  let browser;
  let context;
  let page;
  let origin;
  let errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-controls-"));
    server = await createServer({ root, cacheDir, server: { host: "127.0.0.1", port: 0, watch: null } });
    await server.listen();
    origin = server.resolvedUrls.local[0];
    browser = await chromium.launch();
  });
  beforeEach(async () => {
    context = await browser.newContext({ timezoneId: "UTC", viewport: { width: 1440, height: 900 } });
    page = await context.newPage();
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
  });
  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });
  after(async () => {
    await browser?.close();
    await server?.close();
    if (cacheDir) await rm(cacheDir, { recursive: true, force: true });
  });
  const load = async (scenario = "forms", extra = {}) => {
    await page.goto(`${origin}tests/fixtures/consoleControls.html?${new URLSearchParams({ scenario, ...extra })}`);
    await page.locator("[data-ready]").waitFor();
  };
  const dirty = async (value) => page.waitForFunction((value) => document.querySelector('[aria-label="Dirty forms"]').textContent === value, value);

  it("tracks dirty forms under StrictMode and becomes clean after saving", async () => {
    await load();
    await dirty("clean");
    await page.getByRole("textbox", { name: "First form", exact: true }).fill("Edited");
    await dirty("dirty");
    await page.getByRole("button", { name: "Save First form", exact: true }).click();
    await dirty("clean");
  });
  it("stays dirty until every edited form has been saved or unmounted", async () => {
    await load();
    await page.getByRole("textbox", { name: "First form", exact: true }).fill("First edit");
    await page.getByRole("textbox", { name: "Second form", exact: true }).fill("Second edit");
    await page.getByRole("button", { name: "Save First form", exact: true }).click();
    await dirty("dirty");
    await page.getByRole("button", { name: "Toggle second form", exact: true }).click();
    await dirty("clean");
  });
  it("cleans up dirty registrations on unmount and remounts a clean form", async () => {
    await load();
    await page.getByRole("textbox", { name: "First form", exact: true }).fill("Edited");
    await page.getByRole("button", { name: "Toggle first form", exact: true }).click();
    await dirty("clean");
    await page.getByRole("button", { name: "Toggle first form", exact: true }).click();
    assert.equal(await page.getByRole("textbox", { name: "First form", exact: true }).inputValue(), "");
    await dirty("clean");
  });
  it("becomes clean when an edit is reverted to its saved value", async () => {
    await load();
    const input = page.getByRole("textbox", { name: "First form", exact: true });
    await input.fill("Edited");
    await dirty("dirty");
    await input.fill("");
    await dirty("clean");
  });

  it("consumes navigation approval once and expires unused approval", async () => {
    await page.clock.install();
    await load();
    assert.deepEqual(await page.evaluate(async () => {
      const { approveNextNavigation, consumeNavigationApproval } = await import("/src/utils/forms.ts");
      const initial = consumeNavigationApproval();
      approveNextNavigation();
      return [initial, consumeNavigationApproval(), consumeNavigationApproval()];
    }), [false, true, false]);
    await page.evaluate(async () => (await import("/src/utils/forms.ts")).approveNextNavigation());
    await page.clock.runFor(4001);
    assert.equal(await page.evaluate(async () => (await import("/src/utils/forms.ts")).consumeNavigationApproval()), false);
  });
  it("extends navigation approval when it is renewed", async () => {
    await page.clock.install();
    await load();
    const approve = () => page.evaluate(async () => (await import("/src/utils/forms.ts")).approveNextNavigation());
    await approve();
    await page.clock.runFor(3000);
    await approve();
    await page.clock.runFor(2000);
    assert.equal(await page.evaluate(async () => (await import("/src/utils/forms.ts")).consumeNavigationApproval()), true);
  });

  it("preserves row keys and edited values when deleting a preceding row", async () => {
    await load("keys");
    const key = await page.locator('[data-row="beta"]').getAttribute("data-key");
    await page.getByRole("textbox", { name: "beta", exact: true }).fill("Edited beta");
    await page.getByRole("button", { name: "Remove first row", exact: true }).click();
    assert.equal(await page.locator('[data-row="beta"]').getAttribute("data-key"), key);
    assert.equal(await page.getByRole("textbox", { name: "beta", exact: true }).inputValue(), "Edited beta");
    assert.equal(await page.locator("[data-row]").first().getAttribute("data-row"), "beta");
  });
  it("moves row identity with its edited value and allocates independent new keys", async () => {
    await load("keys");
    const key = await page.locator('[data-row="gamma"]').getAttribute("data-key");
    await page.getByRole("textbox", { name: "gamma", exact: true }).fill("Edited gamma");
    await page.getByRole("button", { name: "Move last row first", exact: true }).click();
    assert.equal(await page.locator("[data-row]").first().getAttribute("data-key"), key);
    assert.equal(await page.getByRole("textbox", { name: "gamma", exact: true }).inputValue(), "Edited gamma");
    await page.getByRole("button", { name: "Add row", exact: true }).click();
    const keys = await page.locator("[data-row]").evaluateAll((items) => items.map((item) => item.getAttribute("data-key")));
    const other = await page.locator("[data-other-key]").getAttribute("data-other-key");
    assert.equal(new Set([...keys, other]).size, 5);
  });

  it("reuses KeepAlive values and releases exactly the requested key", async () => {
    await load();
    const result = await page.evaluate(async () => {
      const { KeepAliveStore } = await import("/src/components/ResourceLayout/keepAlive.ts");
      const store = new KeepAliveStore();
      const disposed = [];
      let creations = 0;
      const make = (key) => store.ensure(key, () => { creations++; return { dispose: () => disposed.push(key) }; });
      const first = make("first");
      const reused = first === make("first");
      make("second");
      store.release("first");
      store.release("first");
      const recreated = first !== make("first");
      store.release("second");
      store.release("first");
      return { reused, recreated, creations, disposed };
    });
    assert.deepEqual(result, { reused: true, recreated: true, creations: 3, disposed: ["first", "second", "first"] });
  });
  it("cancels scheduled disposal when a KeepAlive scope is retained", async () => {
    await page.clock.install();
    await load();
    assert.equal(await page.evaluate(async () => {
      const { KeepAliveStore } = await import("/src/components/ResourceLayout/keepAlive.ts");
      const store = new KeepAliveStore();
      let disposals = 0;
      store.ensure("terminal", () => ({ dispose: () => disposals++ }));
      store.scheduleDispose();
      store.retain();
      store.release("terminal");
      return disposals;
    }), 1);
    await page.clock.runFor(10);
  });
  it("disposes every retained value once and allows fresh values afterwards", async () => {
    await load();
    assert.deepEqual(await page.evaluate(async () => {
      const { KeepAliveStore } = await import("/src/components/ResourceLayout/keepAlive.ts");
      const store = new KeepAliveStore();
      const disposed = [];
      const original = store.ensure("first", () => ({ dispose: () => disposed.push("first") }));
      store.ensure("second", () => ({ dispose: () => disposed.push("second") }));
      store.scheduleDispose();
      store.scheduleDispose();
      await new Promise((resolve) => setTimeout(resolve, 20));
      const fresh = store.ensure("first", () => ({ dispose: () => disposed.push("fresh") }));
      const recreated = fresh !== original;
      store.release("first");
      return { disposed, recreated };
    }), { disposed: ["first", "second", "fresh"], recreated: true });
  });

  it("captures visible dashboard values, prior values and unique links", async () => {
    await load();
    const snapshot = await page.evaluate(async () => {
      const { snapshotSection } = await import("/src/components/Dashboard/snapshot.ts");
      const element = document.createElement("section");
      element.innerHTML = '<a href="/core/users"><span data-agent-label="Users" data-agent-value="12" data-agent-previous="10">  Users  12 </span></a><a href="/core/users">Duplicate</a><a href="/core/services">Services</a>';
      document.body.append(element);
      return snapshotSection(element, { id: "inventory", title: "Inventory", path: "/", description: "Resources", details: { "Time range": "1h" } });
    });
    assert.equal(snapshot.kind, "DashboardSection");
    assert.equal(snapshot.id, "inventory");
    assert.equal(snapshot.title, "Inventory");
    assert.equal(snapshot.data.window, "1h");
    assert.equal(snapshot.data.description, "Resources");
    assert.deepEqual(snapshot.data.values, [{ label: "Users", value: "12", previous: "10", to: "/core/users" }]);
    assert.deepEqual(snapshot.data.links, [{ text: "Users 12", to: "/core/users" }, { text: "Services", to: "/core/services" }]);
    assert.ok(Number.isFinite(Date.parse(snapshot.data.capturedAt)));
  });
  it("caps dashboard snapshot value counts, links and visible text", async () => {
    await load();
    assert.deepEqual(await page.evaluate(async () => {
      const { snapshotSection } = await import("/src/components/Dashboard/snapshot.ts");
      const element = document.createElement("section");
      element.innerHTML = Array.from({ length: 60 }, (_, index) => `<a href="/item-${index}" data-agent-label="Value ${index}" data-agent-value="${index}">Link ${index}</a>`).join("") + "x".repeat(5000);
      document.body.append(element);
      const { data } = snapshotSection(element, { title: "Values", path: "/" });
      return { values: data.values.length, links: data.links.length, text: data.text.length <= 3000 };
    }), { values: 40, links: 20, text: true });
  });

  it("selects and clears a duration preset", async () => {
    await load("pickers");
    await page.getByRole("button", { name: "Duration", exact: true }).click();
    await page.getByRole("button", { name: "5 Minutes", exact: true }).click();
    assert.deepEqual(JSON.parse(await page.getByLabel("Duration value", { exact: true }).textContent()), { type: { oneofKind: "minutes", minutes: 5 } });
    await page.locator("[data-duration]").getByRole("button", { name: "5 Minutes", exact: true }).click();
    await page.getByRole("button", { name: "Clear", exact: true }).click();
    assert.equal(await page.getByLabel("Duration value", { exact: true }).textContent(), "null");
  });
  it("applies custom durations and follows externally changed values", async () => {
    await load("pickers");
    await page.getByRole("button", { name: "Set duration externally", exact: true }).click();
    await page.getByRole("button", { name: "2 Hours", exact: true }).click();
    await page.getByText("Custom", { exact: true }).click();
    await page.getByRole("textbox", { name: "Value", exact: true }).fill("7");
    await page.getByRole("button", { name: "Apply duration", exact: true }).click();
    assert.deepEqual(JSON.parse(await page.getByLabel("Duration value", { exact: true }).textContent()), { type: { oneofKind: "hours", hours: 7 } });
  });
  it("disables applying an empty custom duration", async () => {
    await load("pickers");
    await page.getByRole("button", { name: "Duration", exact: true }).click();
    await page.getByText("Custom", { exact: true }).click();
    await page.getByRole("textbox", { name: "Value", exact: true }).fill("");
    assert.equal(await page.getByRole("button", { name: "Apply duration", exact: true }).isDisabled(), true);
  });
  for (const future of [false, true]) {
    it(`calculates relative timestamps at selection time with future=${future}`, async () => {
      await page.clock.install({ time: new Date("2026-10-30T12:00:00Z") });
      await load("pickers", { future: String(future) });
      await page.getByRole("button", { name: "Timestamp", exact: true }).click();
      await page.clock.runFor(60_000);
      await page.clock.setFixedTime(new Date("2026-10-30T12:01:00Z"));
      await page.getByRole("button", { name: future ? "In 5 minutes" : "Before now by 5 minutes", exact: true }).click();
      assert.equal(await page.getByLabel("Timestamp value", { exact: true }).textContent(), future ? "2026-10-30T12:06:00.000Z" : "2026-10-30T11:56:00.000Z");
    });
  }
  it("clears a timestamp set by an external update", async () => {
    await load("pickers");
    await page.getByRole("button", { name: "Set timestamp externally", exact: true }).click();
    await page.locator("[data-timestamp]").getByRole("button", { name: "Oct 30, 2026 · 12:15 PM", exact: true }).click();
    await page.getByRole("button", { name: "Clear", exact: true }).click();
    assert.equal(await page.getByLabel("Timestamp value", { exact: true }).textContent(), "none");
  });

  it("CONSOLE-020: bounds snapshot metadata even when there is no text left to truncate", { todo: "CONSOLE-020" }, async () => {
    await load();
    const length = await page.evaluate(async () => {
      const { snapshotSection } = await import("/src/components/Dashboard/snapshot.ts");
      const element = document.createElement("section");
      for (let index = 0; index < 40; index++) {
        const value = document.createElement("span");
        value.setAttribute("data-agent-label", `Value ${index}`);
        value.setAttribute("data-agent-value", "x".repeat(1000));
        element.append(value);
      }
      return JSON.stringify(snapshotSection(element, { title: "Large values", path: "/" }).data).length;
    });
    assert.ok(length <= 7500, `Snapshot contains ${length} characters`);
  });
});
