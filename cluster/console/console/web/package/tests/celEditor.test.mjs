import assert from "node:assert/strict";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("CEL editor", () => {
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
    context = await browser.newContext({
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

  const load = async (params = {}) => {
    await page.goto(
      `${origin}tests/fixtures/celEditor.html?${new URLSearchParams(params)}`,
    );
    await page.getByRole("textbox", { name: "CEL expression" }).waitFor();
  };
  const content = () => page.getByRole("textbox", { name: "CEL expression" });
  const menu = () => page.locator(".cm-tooltip-autocomplete");
  const complete = (doc, schema) =>
    page.evaluate(
      ([doc, schema]) => window.celFixture.complete(doc, schema),
      [doc, schema],
    );

  for (const scheme of ["light", "dark"]) {
    for (const width of [1440, 390]) {
      it(`shows the full suggestions outside a clipped ${scheme} drawer at ${width}px`, async () => {
        await page.setViewportSize({ width, height: 900 });
        await load({ scheme, drawer: "true" });
        await content().fill("ctx.");
        await content().press("Control+Space");
        await menu().getByText("user", { exact: true }).waitFor();
        assert.equal(
          await menu().evaluate((element) => !!element.closest(".cm-editor")),
          false,
        );
        const box = await menu().boundingBox();
        assert.ok(box.x >= 12 && box.x + box.width <= width - 12);
        assert.ok(box.y >= 12 && box.y + box.height <= 900 - 12);
        const option = menu().getByText("user", { exact: true });
        await option.scrollIntoViewIfNeeded();
        assert.equal(
          await option.evaluate((element) => {
            const rect = element.getBoundingClientRect();
            const hit = document.elementFromPoint(
              rect.x + rect.width / 2,
              rect.y + rect.height / 2,
            );
            return element.closest("li").contains(hit);
          }),
          true,
        );
        assert.ok((await menu().locator(".cm-completionDescription").count()) > 0);
        await page.screenshot({ path: `/tmp/octelium-cel-${scheme}-${width}.png` });
        await option.click();
        assert.equal(await content().innerText(), "ctx.user");
        assert.equal(await content().getAttribute("aria-multiline"), "true");
      });
    }
  }

  it("preserves multiline input, normal Enter behavior, undo and redo", async () => {
    await load();
    const expression =
      'ctx.user.metadata.name == "alice"\n&& ctx.request.http.path.startsWith("/admin")';
    await content().fill(expression);
    assert.equal(await page.getByTestId("expression").textContent(), expression);
    await content().press("Escape");
    await content().press("End");
    await content().press("Enter");
    await page.keyboard.type("&& true");
    assert.match(await content().innerText(), /\n&& true$/);
    await content().press("Control+z");
    assert.ok(!(await content().innerText()).endsWith("&& true"));
    await content().press("Control+Shift+Z");
    assert.ok((await content().innerText()).endsWith("&& true"));
  });

  it("accepts suggestions with Tab and lets Tab leave the editor after dismissal", async () => {
    await load();
    await content().fill("ctx.us");
    await content().press("Control+Space");
    await menu().getByText("user", { exact: true }).waitFor();
    await content().press("Tab");
    assert.equal(await content().innerText(), "ctx.user");
    await content().press("Escape");
    await content().press("Tab");
    assert.equal(
      await page.getByRole("button", { name: "Show CEL suggestions" }).evaluate(
        (element) => document.activeElement === element,
      ),
      true,
    );
    await page.keyboard.press("Tab");
    assert.equal(
      await page.getByRole("button", { name: "Next field" }).evaluate(
        (element) => document.activeElement === element,
      ),
      true,
    );
  });

  it("provides an accessible suggestions button and applies nested field completions", async () => {
    await load({ invalid: "true" });
    assert.equal(await content().getAttribute("aria-invalid"), "true");
    await page.getByRole("button", { name: "Show CEL suggestions" }).click();
    await menu().getByText("ctx", { exact: true }).waitFor();
    await content().press("Escape");
    await content().fill("ctx.user.spec.em");
    await content().press("Control+Space");
    await menu().getByText("email", { exact: true }).waitFor();
    await content().press("Enter");
    assert.equal(await content().innerText(), "ctx.user.spec.email");
  });

  it("resolves references and keeps properties alongside schema unions and constraints", async () => {
    await load();
    const result = await complete("ctx.user.spec.");
    assert.ok(result.options.some(
      (option) => option.label === "email" && option.detail === "string",
    ));
    const union = await complete("ctx.", {
      type: "object",
      definitions: {
        shared: { properties: { inherited: { type: "boolean" } } },
      },
      properties: {
        ctx: {
          type: "object",
          properties: { own: { type: "string" } },
          allOf: [{ $ref: "#/definitions/shared" }],
          oneOf: [
            { properties: { first: { type: "string" } } },
            { properties: { second: { type: "number" } } },
          ],
        },
      },
    });
    assert.deepEqual(
      union.options.map((option) => option.label).sort(),
      ["first", "inherited", "own", "second"],
    );
  });

  it("understands list indexes, map keys, and scoped list macro variables", async () => {
    await load();
    for (const expression of [
      "ctx.groups[0].metadata.",
      "ctx.groups.exists(group, group.metadata.",
    ]) {
      const result = await complete(expression);
      assert.ok(result.options.some((option) => option.label === "name"));
    }
    const map = await complete('ctx.user.metadata.labels["team"].');
    assert.ok(map.options.some((option) => option.label === "startsWith"));
    assert.equal(await complete("ctx.groups.exists(group, true) && group."), null);
  });

  it("does not offer fields inside literals or comments", async () => {
    await load();
    for (const expression of [
      '"ctx.user.',
      "'ctx.user.",
      "// ctx.user.",
      '"""ctx.user.',
      'r"ctx.user.',
    ]) {
      assert.equal(await complete(expression), null, expression);
    }
    assert.ok((await complete("// comment\nctx.user.")).options.some(
      (option) => option.label === "metadata",
    ));
  });

  it("suggests schema enum values inside a comparison without duplicating quotes", async () => {
    await load();
    await content().fill('ctx.user.spec.type == "HU"');
    await content().press("ArrowLeft");
    await content().press("Control+Space");
    await menu().getByText("HUMAN", { exact: true }).waitFor();
    await menu().getByText("HUMAN", { exact: true }).click();
    assert.equal(await content().innerText(), 'ctx.user.spec.type == "HUMAN"');
    const result = await complete("ctx.user.spec.type == ");
    assert.ok(result.options.some((option) => option.label === '"WORKLOAD"'));
  });

  it("traverses recursive references without flattening the entire schema", async () => {
    await load();
    const result = await complete("ctx.next.next.next.", {
      properties: { ctx: { $ref: "#/definitions/node" } },
      definitions: {
        node: {
          type: "object",
          properties: {
            next: { $ref: "#/definitions/node" },
            name: { type: "string" },
            forbidden: false,
          },
        },
      },
    });
    assert.deepEqual(
      result.options.map((option) => option.label).sort(),
      ["name", "next"],
    );
  });

  it("keeps every option reachable in a short viewport", async () => {
    await page.setViewportSize({ width: 390, height: 240 });
    await load({ drawer: "true", scheme: "dark" });
    await content().fill("ctx.");
    await content().press("Control+Space");
    const option = menu().getByText("user", { exact: true });
    await option.waitFor();
    await option.scrollIntoViewIfNeeded();
    const box = await menu().boundingBox();
    assert.ok(box.y >= 12 && box.y + box.height <= 228);
    await option.click();
    assert.equal(await content().innerText(), "ctx.user");
  });

  it("grows with the expression and caps long documents at a scrollable height", async () => {
    await load();
    const initial = await page.locator(".cm-editor").boundingBox();
    await content().fill(Array.from({ length: 6 }, () => "true &&").join("\n"));
    assert.ok(
      (await page.locator(".cm-editor").boundingBox()).height > initial.height,
    );
    await content().fill(Array.from({ length: 30 }, () => "true &&").join("\n"));
    assert.ok((await page.locator(".cm-editor").boundingBox()).height <= 240);
    assert.equal(
      await page.locator(".cm-scroller").evaluate(
        (element) => element.scrollHeight > element.clientHeight,
      ),
      true,
    );
  });

  it("lets Enter insert a new line after a fully typed value", async () => {
    await load();
    await content().click();
    await page.keyboard.type("true", { delay: 120 });
    await menu().waitFor({ state: "hidden" });
    await content().press("Enter");
    assert.equal(await page.getByTestId("expression").textContent(), "true\n");
  });

  it("opens nested field suggestions automatically while typing", async () => {
    await load();
    await content().click();
    await page.keyboard.type("ctx.user.spec.");
    await menu().getByText("email", { exact: true }).waitFor();
    await menu().getByText("email", { exact: true }).click();
    assert.equal(await content().innerText(), "ctx.user.spec.email");
  });

  it("inserts function arguments as editable snippets", async () => {
    await load();
    await content().fill("has");
    await content().press("Control+Space");
    await menu().getByText("has", { exact: true }).waitFor();
    await content().press("Enter");
    await page.keyboard.type("ctx.user.spec.email");
    assert.equal(await content().innerText(), "has(ctx.user.spec.email)");
  });
});
