import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Console runtime regressions", () => {
  let cacheDir;
  let server;
  let browser;
  let context;
  let page;
  let origin;
  let errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-runtime-"));
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
      timezoneId: "UTC",
      viewport: { width: 1440, height: 900 },
    });
    page = await context.newPage();
    page.setDefaultTimeout(30_000);
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.addInitScript(() => {
      window.nativeIntersectionObserver = window.IntersectionObserver;
      const observers = new Set();
      window.IntersectionObserver = class {
        constructor(callback) {
          this.callback = callback;
          this.targets = new Set();
          observers.add(this);
        }
        observe(target) {
          this.targets.add(target);
        }
        disconnect() {
          observers.delete(this);
          this.targets.clear();
        }
      };
      window.intersections = (selector, visible) => {
        for (const observer of observers) {
          const entries = [...observer.targets]
            .filter((target) => target.matches(selector))
            .map((target) => ({ target, isIntersecting: visible }));
          if (entries.length) observer.callback(entries);
        }
      };
      window.observerCount = () => observers.size;
    });
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (scenario, extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/consoleRuntime.html?${new URLSearchParams({ scenario, ...extra })}`,
    );
    await page.waitForFunction(() => window.calls !== undefined);
  };

  const visible = async (selector, value) => {
    await page.evaluate(
      ([selector, value]) => window.intersections(selector, value),
      [selector, value],
    );
  };

  const count = () => page.evaluate(() => window.calls.length);

  it("avoids document walks for present and missing Combobox targets", async () => {
    await load("once");
    const result = await page.evaluate(async () => {
      const { findElementBySelector, findElementsBySelector } =
        await import("/src/utils/mantine/find-element-in-shadow-dom.js");
      const target = document.createElement("div");
      target.className = "runtime-target";
      document.body.append(target);
      const selectors = [];
      const querySelectorAll = document.querySelectorAll;
      document.querySelectorAll = function (selector) {
        selectors.push(selector);
        return querySelectorAll.call(this, selector);
      };
      try {
        return {
          found: findElementBySelector(".runtime-target") === target,
          missing: findElementBySelector(".runtime-missing"),
          matches: findElementsBySelector(".runtime-target").length,
          empty: findElementsBySelector(".runtime-missing").length,
          selectors,
        };
      } finally {
        document.querySelectorAll = querySelectorAll;
      }
    });
    assert.deepEqual(result, {
      found: true,
      missing: null,
      matches: 1,
      empty: 0,
      selectors: [".runtime-target", ".runtime-missing"],
    });
  });

  it("pauses deferred dashboard polling while preserving mounted state", async () => {
    await page.clock.install();
    await load("dashboard");
    await page.waitForFunction(() => window.observerCount() === 1);
    assert.equal(await count(), 0);
    await visible("[data-deferred] > div", true);
    await page.getByText("Loaded", { exact: true }).waitFor();
    await page
      .getByRole("textbox", { name: "Panel state", exact: true })
      .fill("Keep this");
    await page.clock.fastForward(1100);
    await page.waitForFunction(() => window.calls.length >= 2);
    await visible("[data-deferred] > div", false);
    await page.clock.runFor(50);
    const paused = await count();
    await page.clock.fastForward(5000);
    assert.equal(await count(), paused);
    assert.equal(
      await page
        .getByRole("textbox", { name: "Panel state", exact: true })
        .inputValue(),
      "Keep this",
    );
    await visible("[data-deferred] > div", true);
    await page.clock.fastForward(1100);
    await page.waitForFunction(
      (previous) => window.calls.length > previous,
      paused,
    );
    assert.ok(
      (await page.evaluate(() => window.calls)).every(
        (call) => call === "dashboard",
      ),
    );
    await page.getByRole("button", { name: "Toggle auto refresh" }).click();
    await page.clock.runFor(50);
    const manual = await count();
    await page.clock.fastForward(5000);
    assert.equal(await count(), manual);
    await page.getByRole("button", { name: "Unmount dashboard" }).click();
    await page.waitForFunction(() => window.observerCount() === 0);
  });

  it("pauses eagerly mounted sections without delaying their initial query", async () => {
    await page.clock.install();
    await load("dashboard", { eager: "true" });
    await page.getByText("Loaded", { exact: true }).waitFor();
    assert.equal(await count(), 1);
    await page.waitForFunction(() => window.observerCount() === 1);
    await visible("[data-deferred] > div", false);
    await page.clock.runFor(50);
    await page.clock.fastForward(5000);
    assert.equal(await count(), 1);
    await visible("[data-deferred] > div", true);
    await page.clock.fastForward(1100);
    await page.waitForFunction(() => window.calls.length > 1);
  });

  it("preserves one-shot visibility and disconnects its observer", async () => {
    await load("once");
    await page.getByText("Waiting", { exact: true }).waitFor();
    await visible("[data-once]", true);
    await page.getByText("Visible", { exact: true }).waitFor();
    await page.waitForFunction(() => window.observerCount() === 0);
    await visible("[data-once]", false);
    assert.equal(await page.getByText("Visible", { exact: true }).count(), 1);
  });

  it("loads panels when IntersectionObserver is unavailable", async () => {
    await page.addInitScript(() => delete window.IntersectionObserver);
    await load("dashboard");
    await page.getByText("Loaded", { exact: true }).waitFor();
    assert.equal(await count(), 1);
  });

  for (const scenario of ["metric", "metrics", "service"]) {
    it(`pauses ${scenario} polling off-screen and resumes it on return`, async () => {
      await page.clock.install();
      await load(scenario);
      await page.waitForFunction(() => window.observerCount() > 0);
      assert.equal(await count(), 0);
      await visible("*", true);
      await page.waitForFunction(() => window.calls.length > 0);
      await page.locator("canvas").first().waitFor();
      const initial = await count();
      await page.clock.fastForward(15_100);
      await page.waitForFunction(
        (previous) => window.calls.length > previous,
        initial,
      );
      await visible("*", false);
      await page.clock.runFor(50);
      const paused = await count();
      const canvases = await page.locator("canvas").count();
      await page.clock.fastForward(30_100);
      assert.equal(await count(), paused);
      assert.equal(await page.locator("canvas").count(), canvases);
      await visible("*", true);
      await page.clock.fastForward(15_100);
      await page.waitForFunction(
        (previous) => window.calls.length > previous,
        paused,
      );
    });
  }

  it("keeps explicitly disabled metric queries disabled in view", async () => {
    await page.clock.install();
    await load("metric", { enabled: "false" });
    await page.waitForFunction(() => window.observerCount() === 1);
    await visible("*", true);
    await page.clock.fastForward(30_100);
    assert.equal(await count(), 0);
  });

  it("pauses metric polling beyond the native observer margin", async () => {
    await page.addInitScript(() => {
      window.IntersectionObserver = window.nativeIntersectionObserver;
    });
    await page.clock.install();
    await load("metric");
    await page.waitForFunction(() => window.calls.length === 1);
    await page.evaluate(() => {
      const spacer = document.createElement("div");
      spacer.style.height = "2500px";
      document.getElementById("root").prepend(spacer);
    });
    await page.clock.runFor(200);
    await page.clock.fastForward(15_100);
    assert.equal(await count(), 1);
    await page
      .getByRole("region", { name: "Test metric" })
      .scrollIntoViewIfNeeded();
    await page.clock.runFor(200);
    await page.waitForFunction(() => window.calls.length > 1);
  });

  it("renders stat sparklines without canvas instances and handles edge values", async () => {
    await load("sparklines");
    await page.locator("[data-tile] polyline").first().waitFor();
    assert.equal(await page.locator("canvas, .echarts-for-react").count(), 0);
    assert.equal(await page.locator("[data-tile] polyline").count(), 5);
    const lines = await page
      .locator("[data-tile] polyline")
      .evaluateAll((elements) =>
        elements.map((element) => ({
          points: element
            .getAttribute("points")
            .split(" ")
            .map((point) => point.split(",").map(Number)),
          color: element.getAttribute("stroke"),
          height: element.closest("svg").getBoundingClientRect().height,
        })),
      );
    for (const line of lines) {
      assert.equal(line.color, "#3b82f6");
      assert.equal(line.height, 40);
      for (const [x, y] of line.points) {
        assert.ok(Number.isFinite(x) && x >= 1 && x <= 99);
        assert.ok(Number.isFinite(y) && y >= 4 && y <= 38);
      }
      assert.deepEqual(
        line.points.map(([x]) => x),
        [...line.points.map(([x]) => x)].sort((a, b) => a - b),
      );
    }
    assert.equal(lines[0].points.length, 3);
    assert.ok(lines[1].points.every(([, y]) => y === 38));
    assert.equal(lines[3].points.length, 1);
    const gradients = await page
      .locator("[data-tile] linearGradient")
      .evaluateAll((elements) => elements.map((element) => element.id));
    assert.equal(new Set(gradients).size, 5);
    await page.getByRole("button", { name: "Update sparklines" }).click();
    await page.waitForFunction(
      () => document.querySelectorAll("[data-tile] polyline").length === 6,
    );
    assert.ok(
      (
        await page
          .locator("[data-tile] polyline")
          .evaluateAll((elements) =>
            elements.map((element) => element.getAttribute("stroke")),
          )
      ).every((color) => color === "#ef4444"),
    );
  });

  it("does not create a WebGL context while importing console utilities", async () => {
    await page.addInitScript(() => {
      window.webglCalls = [];
      const getContext = HTMLCanvasElement.prototype.getContext;
      HTMLCanvasElement.prototype.getContext = function (type, ...args) {
        if (type === "webgl" || type === "webgl2") window.webglCalls.push(type);
        return getContext.call(this, type, ...args);
      };
    });
    await load("sparklines");
    await page.locator("[data-tile]").first().waitFor();
    assert.deepEqual(await page.evaluate(() => window.webglCalls), []);
  });

  for (const reducedMotion of ["no-preference", "reduce"]) {
    it(`respects ${reducedMotion} for repeating pulses and row chevrons`, async () => {
      await page.emulateMedia({ reducedMotion });
      await load("motion");
      await page.getByText("Connected", { exact: true }).first().waitFor();
      assert.equal(
        await page.getByText("Connected", { exact: true }).count(),
        3,
      );
      assert.equal(await page.locator('[class*="animate-ping"]').count(), 0);
      const animations = await page
        .locator('[class*="animate-pulse"]')
        .evaluateAll((elements) =>
          elements.map((element) => getComputedStyle(element).animationName),
        );
      assert.equal(animations.length, 3);
      assert.ok(
        animations.every(
          (animation) =>
            animation === (reducedMotion === "reduce" ? "none" : "pulse"),
        ),
      );
      await load("logs");
      const buttons = page.locator("button[aria-expanded]");
      await buttons.nth(3).waitFor();
      for (let index = 0; index < 4; index++) {
        const button = buttons.nth(index);
        const chevron = button
          .locator("span")
          .filter({ has: page.locator("svg.lucide-chevron-down") });
        assert.equal(
          await chevron.evaluate(
            (element) => getComputedStyle(element).transitionProperty,
          ),
          reducedMotion === "reduce" ? "none" : "color, transform",
        );
        await button.click();
        assert.equal(await button.getAttribute("aria-expanded"), "true");
        assert.equal(
          await chevron.evaluate((element) => element.style.transform),
          "rotate(180deg)",
        );
        await button.click();
        assert.equal(await button.getAttribute("aria-expanded"), "false");
        assert.equal(
          await chevron.evaluate((element) => element.style.transform),
          "rotate(0deg)",
        );
      }
    });
  }

  it("removes queue abort listeners after fulfillment and both failure paths", async () => {
    await load("once");
    const result = await page.evaluate(async () => {
      const { queued } = await import("/src/utils/visibility/queue.ts");
      const removals = [];
      const controllers = Array.from(
        { length: 3 },
        () => new AbortController(),
      );
      controllers.forEach((controller, index) => {
        const remove = controller.signal.removeEventListener;
        controller.signal.removeEventListener = function (type, ...args) {
          if (type === "abort") removals.push(index);
          return remove.call(this, type, ...args);
        };
      });
      const outcomes = await Promise.allSettled([
        queued(0, async () => "Done", controllers[0].signal),
        queued(
          0,
          async () => {
            throw new Error("Async");
          },
          controllers[1].signal,
        ),
        queued(
          0,
          () => {
            throw new Error("Sync");
          },
          controllers[2].signal,
        ),
      ]);
      await Promise.resolve();
      return {
        statuses: outcomes.map((outcome) => outcome.status),
        removals: removals.sort(),
      };
    });
    assert.deepEqual(result, {
      statuses: ["fulfilled", "rejected", "rejected"],
      removals: [0, 1, 2],
    });
  });

  it("preserves queued and running cancellation without releasing active slots early", async () => {
    await load("once");
    const result = await page.evaluate(async () => {
      const { queued } = await import("/src/utils/visibility/queue.ts");
      const releases = [];
      const running = new AbortController();
      const tasks = Array.from({ length: 4 }, (_, index) =>
        queued(
          0,
          () => new Promise((resolve) => releases.push(resolve)),
          index === 0 ? running.signal : undefined,
        ),
      );
      const runningOutcome = tasks[0].catch((error) => error.message);
      await Promise.resolve();
      const pending = new AbortController();
      let pendingStarted = false;
      const pendingOutcome = queued(
        0,
        async () => {
          pendingStarted = true;
        },
        pending.signal,
      ).catch((error) => error.message);
      pending.abort(new Error("Pending cancelled"));
      running.abort(new Error("Running cancelled"));
      let nextStarted = false;
      const next = queued(0, async () => {
        nextStarted = true;
      });
      await Promise.resolve();
      const startedBeforeRelease = nextStarted;
      releases.forEach((release) => release("Done"));
      await Promise.allSettled(tasks);
      await next;
      return {
        pending: await pendingOutcome,
        running: await runningOutcome,
        pendingStarted,
        startedBeforeRelease,
        nextStarted,
      };
    });
    assert.deepEqual(result, {
      pending: "Pending cancelled",
      running: "Running cancelled",
      pendingStarted: false,
      startedBeforeRelease: false,
      nextStarted: true,
    });
  });

  for (const outcome of ["resolve", "reject"]) {
    it(`does not schedule copy feedback after unmount and clipboard ${outcome}`, async () => {
      await page.addInitScript(() => {
        window.copyTimers = [];
        const setTimeout = window.setTimeout;
        window.setTimeout = function (callback, delay, ...args) {
          if (delay === 1200) window.copyTimers.push(delay);
          return setTimeout.call(this, callback, delay, ...args);
        };
        Object.defineProperty(navigator, "clipboard", {
          value: {
            writeText: () =>
              new Promise((resolve, reject) => {
                window.finishCopy = { resolve, reject };
              }),
          },
        });
      });
      await load("copy");
      await page.getByRole("button", { name: "Copy to clipboard" }).click();
      await page.getByRole("button", { name: "Unmount copy" }).click();
      await page.evaluate((outcome) => window.finishCopy[outcome](), outcome);
      assert.deepEqual(await page.evaluate(() => window.copyTimers), []);
    });
  }

  it("clears existing copy feedback timers on unmount", async () => {
    await page.addInitScript(() => {
      const timers = new Set();
      window.copyTimerCount = () => timers.size;
      const setTimeout = window.setTimeout;
      const clearTimeout = window.clearTimeout;
      window.setTimeout = function (callback, delay, ...args) {
        const timer = setTimeout.call(this, callback, delay, ...args);
        if (delay === 1200) timers.add(timer);
        return timer;
      };
      window.clearTimeout = function (timer) {
        timers.delete(timer);
        return clearTimeout.call(this, timer);
      };
      Object.defineProperty(navigator, "clipboard", {
        value: { writeText: async () => {} },
      });
    });
    await load("copy");
    await page.getByRole("button", { name: "Copy to clipboard" }).click();
    await page.waitForFunction(() => window.copyTimerCount() === 1);
    await page.getByRole("button", { name: "Unmount copy" }).click();
    await page.waitForFunction(() => window.copyTimerCount() === 0);
  });
});
