import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

const png = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
  "base64",
);

describe("Service playground", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-playground-"));
    server = await createServer({
      root,
      cacheDir,
      server: { host: "localhost", port: 0, watch: null },
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
      viewport: { width: 1440, height: 900 },
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

  const load = async (path, extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/servicePlayground.html?${new URLSearchParams({ path, ...extra })}`,
    );
    await page.waitForFunction(() => window.playground);
  };
  const calls = () => page.evaluate(() => window.playground.calls);
  const rpc = async (method) =>
    (await calls()).filter((call) => call.body?.method === method);
  const location = () => page.getByTestId("location").textContent();
  const composer = () => page.getByRole("textbox", { name: "Message" });
  const tab = (name) =>
    page.locator(".mantine-SegmentedControl-label").filter({ hasText: name });
  const connect = async () => {
    await page.getByRole("button", { name: "Connect", exact: true }).click();
    await page.getByText("Connected", { exact: true }).waitFor();
    await page.getByRole("button", { name: /^search_repositories/ }).waitFor();
  };

  it("shows the Playground tab only for public LLM and MCP Services", async () => {
    for (const name of ["openai", "mcp-github"]) {
      await load(`/core/services/${name}`);
      await tab("Playground").waitFor();
    }
    await load("/core/services/web");
    await tab("Overview").waitFor();
    assert.equal(await tab("Playground").count(), 0);

    await load("/core/services/web/playground");
    await page.waitForFunction(
      () =>
        document.querySelector("[data-testid=location]")?.textContent ===
        "/core/services/web",
    );
  });

  it("streams a chat completion through the Service with the session cookie", async () => {
    await load("/core/services/openai/playground");
    await page.getByRole("heading", { name: "Chat with openai" }).waitFor();
    await composer().fill("Compare TCP and UDP in a short table");
    await composer().press("Enter");
    await page.getByRole("cell", { name: "Connectionless" }).waitFor();
    await page.getByText("138 tokens").waitFor();

    const [call] = await calls();
    assert.equal(
      call.url,
      `${origin.replace("http://", "https://openai.")}v1/chat/completions`,
    );
    assert.equal(call.method, "POST");
    assert.equal(call.headers.authorization, undefined);
    assert.equal(call.body.model, "gpt-5-mini");
    assert.equal(call.body.stream, true);
    assert.deepEqual(call.body.stream_options, { include_usage: true });
    assert.deepEqual(call.body.messages, [
      { role: "user", content: "Compare TCP and UDP in a short table" },
    ]);
    assert.equal(await page.getByText("gpt-5-mini-2026-08-07").count(), 1);
    assert.equal(
      await page.getByRole("button", { name: /Thought/ }).count(),
      1,
    );
    assert.equal(await composer().inputValue(), "");
  });

  it("applies the request settings to the request body", async () => {
    await load("/core/services/openai/playground");
    const settings = page.getByRole("complementary", {
      name: "Request settings",
    });
    await settings
      .getByRole("textbox", { name: "System prompt" })
      .fill("Be brief");
    await settings.getByRole("textbox", { name: "Temperature" }).fill("0.2");
    await settings
      .getByRole("textbox", { name: "Max output tokens" })
      .fill("256");
    await settings.getByRole("combobox", { name: "Reasoning effort" }).click();
    await page.getByRole("option", { name: "low" }).click();
    await settings
      .locator(".mantine-SegmentedControl-label")
      .filter({ hasText: /^Responses$/ })
      .click();
    await page.getByText("OpenAI · Responses").waitFor();

    await composer().fill("Say hello");
    await composer().press("Enter");
    await page.getByText("100 tokens").waitFor();

    const [call] = await calls();
    assert.ok(call.url.endsWith("/v1/responses"));
    assert.deepEqual(call.body.input[0], {
      role: "system",
      content: "Be brief",
    });
    assert.equal(call.body.temperature, 0.2);
    assert.equal(call.body.max_output_tokens, 256);
    assert.deepEqual(call.body.reasoning, { effort: "low" });
    await page.getByText(/^Hello! I am a helpful assistant/).waitFor();

    await settings
      .getByRole("button", { name: "Reset to the defaults" })
      .click();
    assert.equal(
      await settings
        .getByRole("textbox", { name: "System prompt" })
        .inputValue(),
      "",
    );
    await page.getByText("OpenAI · Chat Completions").waitFor();
  });

  it("stops a generation and keeps the partial answer", async () => {
    await load("/core/services/openai/playground");
    await composer().fill("hold please");
    await composer().press("Enter");
    await page.getByText("Generating", { exact: true }).waitFor();
    await page.getByRole("button", { name: "Stop", exact: true }).click();
    await page.getByText("Stopped", { exact: true }).waitFor();
    assert.equal(
      await page.getByText("Generating", { exact: true }).count(),
      0,
    );
    await page.getByRole("button", { name: /Thought/ }).waitFor();

    await composer().fill("deny this");
    await composer().press("Enter");
    await page
      .getByRole("alert")
      .filter({ hasText: "403: The Policy deny-llm denied the request" })
      .waitFor();
    await page.getByRole("button", { name: "Regenerate" }).click();
    await page.waitForFunction(() => window.playground.calls.length === 3);
    assert.equal((await calls())[2].body.messages.at(-1).content, "deny this");
  });

  it("asks for the model when the Service does not set one", async () => {
    await load("/core/services/claude/playground");
    await page
      .getByRole("button", { name: "Change the model" })
      .filter({ hasText: "Set the model" })
      .waitFor();
    await composer().fill("A bat and a ball");
    await composer().press("Enter");
    await page.getByText("Enter the model to send the request to").waitFor();
    assert.equal((await calls()).length, 0);
    assert.equal(await composer().inputValue(), "A bat and a ball");

    const settings = page.getByRole("complementary", {
      name: "Request settings",
    });
    await settings
      .getByRole("textbox", { name: "Model" })
      .fill("claude-sonnet-5");
    await settings
      .getByRole("textbox", { name: "Thinking budget" })
      .fill("2048");
    assert.equal(
      await page.getByText("Enter the model to send the request to").count(),
      0,
    );
    await composer().press("Enter");
    await page.getByText("118 tokens").waitFor();

    const [call] = await calls();
    assert.ok(call.url.endsWith("/v1/messages"));
    assert.equal(call.headers["x-api-key"], undefined);
    assert.equal(call.headers["anthropic-version"], "2023-06-01");
    assert.equal(call.body.model, "claude-sonnet-5");
    assert.equal(call.body.max_tokens, 4096);
    assert.deepEqual(call.body.thinking, {
      type: "enabled",
      budget_tokens: 2048,
    });
    await page.getByRole("button", { name: /Thought/ }).click();
    await page
      .getByText("Let me think about the cost of the ball step by step.")
      .waitFor();
  });

  it("attaches images and keeps the conversation across the drawer tabs", async () => {
    await load("/core/services/openai/playground");
    await page.locator('input[type="file"][accept="image/*"]').setInputFiles({
      name: "diagram.png",
      mimeType: "image/png",
      buffer: png,
    });
    await page.getByRole("img", { name: "diagram.png" }).waitFor();
    await composer().fill("Describe the image");
    await composer().press("Enter");
    await page.getByText(/^Hello! I am a helpful assistant/).waitFor();

    const [call] = await calls();
    assert.deepEqual(call.body.messages[0].content[0], {
      type: "text",
      text: "Describe the image",
    });
    assert.equal(call.body.messages[0].content[1].type, "image_url");
    assert.match(
      call.body.messages[0].content[1].image_url.url,
      /^data:image\/png;base64,/,
    );

    await tab("Overview").click();
    await page.waitForFunction(
      () =>
        document.querySelector("[data-testid=location]")?.textContent ===
        "/core/services/openai",
    );
    await tab("Playground").click();
    await page.getByText("Describe the image").waitFor();
    await page.getByText(/^Hello! I am a helpful assistant/).waitFor();
    assert.equal(await location(), "/core/services/openai/playground");

    await page.getByRole("button", { name: "New conversation" }).click();
    await page.getByRole("heading", { name: "Chat with openai" }).waitFor();
  });

  it("opens the request settings over the chat on phones", async () => {
    await page.setViewportSize({ width: 390, height: 844 });
    await load("/core/services/openai/playground");
    const settings = page.getByRole("complementary", {
      name: "Request settings",
    });
    assert.equal(await settings.count(), 0);
    await page.getByRole("button", { name: "Request settings" }).click();
    await settings.getByRole("textbox", { name: "Model" }).waitFor();
    await page.keyboard.press("Escape");
    await settings.waitFor({ state: "detached" });
    assert.equal(await location(), "/core/services/openai/playground");

    await page.getByRole("button", { name: "Change the model" }).click();
    await page.waitForFunction(
      () => document.activeElement?.value === "gpt-5-mini",
    );
    await page.keyboard.press("Escape");
    await settings.waitFor({ state: "detached" });
    await page.keyboard.press("Escape");
    await page.waitForFunction(
      () =>
        document.querySelector("[data-testid=location]")?.textContent ===
        "/core/services",
    );
  });

  it("connects to the MCP server and lists what it declares", async () => {
    await load("/core/services/mcp-github/playground");
    await page.getByText("Not connected", { exact: true }).waitFor();
    await page.getByRole("button", { name: "Connection settings" }).click();
    const capabilities = page.getByRole("textbox", {
      name: "Client capabilities",
    });
    await capabilities.fill("{");
    await page.getByRole("button", { name: "Connect", exact: true }).click();
    await page
      .getByRole("alert")
      .filter({ hasText: "The client capabilities must be valid JSON." })
      .waitFor();
    assert.equal((await calls()).length, 0);
    await page.getByRole("button", { name: "Connection settings" }).click();
    await capabilities.fill("{}");
    await page.keyboard.press("Escape");
    await capabilities.waitFor({ state: "detached" });
    assert.equal(await location(), "/core/services/mcp-github/playground");
    await connect();
    assert.equal(await page.getByRole("alert").count(), 0);

    const [initialize] = await rpc("initialize");
    assert.ok(initialize.url.endsWith("/mcp"));
    assert.equal(initialize.headers["mcp-session-id"], undefined);
    assert.equal(initialize.headers["mcp-protocol-version"], "2025-06-18");
    assert.equal(initialize.headers["mcp-method"], "initialize");
    assert.equal(initialize.body.params.protocolVersion, "2025-06-18");
    assert.deepEqual(initialize.body.params.clientInfo, {
      name: "Octelium Console",
      version: "1.0.0",
    });

    const [initialized] = await rpc("notifications/initialized");
    assert.equal(initialized.body.id, undefined);
    assert.equal(initialized.headers["mcp-session-id"], "mcp-session-7f3a9c2e");
    for (const method of ["tools/list", "resources/list", "prompts/list"]) {
      const [call] = await rpc(method);
      assert.equal(call.headers["mcp-session-id"], "mcp-session-7f3a9c2e");
    }

    await page.getByRole("heading", { name: "github-mcp" }).waitFor();
    await page.getByRole("button", { name: "Instructions" }).click();
    await page
      .getByText(/^Use search_repositories before reading files/)
      .waitFor();
    await page.getByRole("button", { name: "Ping" }).click();
    await page.getByText(/^Ping/).filter({ hasText: "ms" }).waitFor();
  });

  it("runs a tool with typed arguments from its input schema", async () => {
    await load("/core/services/mcp-github/playground");
    await connect();
    await page.getByRole("button", { name: /^search_repositories/ }).click();
    await page.getByRole("heading", { name: "search_repositories" }).waitFor();

    const run = page.getByRole("button", { name: "Run tool" });
    await run.click();
    await page
      .getByRole("alert")
      .filter({ hasText: "“query” is required." })
      .waitFor();
    assert.equal((await rpc("tools/call")).length, 0);

    await page.getByRole("textbox", { name: /^query/ }).fill("zero trust");
    await page.getByRole("textbox", { name: /^page/ }).fill("2");
    await page.getByRole("combobox", { name: /^sort/ }).click();
    await page.getByRole("option", { name: "forks" }).click();
    await page.getByRole("combobox", { name: /^archived/ }).click();
    await page.getByRole("option", { name: "true" }).click();
    await run.click();
    await page.getByText(/"octelium\/cordium"/).waitFor();

    const [call] = await rpc("tools/call");
    assert.equal(call.headers["mcp-name"], "search_repositories");
    assert.deepEqual(call.body.params, {
      name: "search_repositories",
      arguments: {
        query: "zero trust",
        page: 2,
        sort: "forks",
        archived: true,
      },
    });

    await page
      .locator("label")
      .filter({ hasText: /^JSON$/ })
      .first()
      .click();
    const json = page.getByRole("textbox", { name: "Arguments as JSON" });
    assert.deepEqual(JSON.parse(await json.inputValue()), {
      query: "zero trust",
      page: 2,
      sort: "forks",
      archived: true,
    });
    await json.fill('{"query": "cordium"}');
    await json.press("Control+Enter");
    await page.waitForFunction(
      () =>
        window.playground.calls.filter((c) => c.body?.method === "tools/call")
          .length === 2,
    );
    assert.deepEqual((await rpc("tools/call"))[1].body.params.arguments, {
      query: "cordium",
    });
  });

  it("pages the tools, calls one by name and shows tool errors", async () => {
    await load("/core/services/mcp-github/playground");
    await connect();
    await page.getByRole("button", { name: "Load more" }).click();
    await page.getByRole("button", { name: /^list_commits/ }).waitFor();
    assert.equal((await rpc("tools/list"))[1].body.params.cursor, "page-2");

    await page.getByRole("button", { name: /^create_issue/ }).click();
    await page.getByRole("textbox", { name: /^owner/ }).fill("octelium");
    await page.getByRole("textbox", { name: /^repo/ }).fill("octelium");
    await page.getByRole("textbox", { name: /^title/ }).fill("Broken link");
    await page.getByRole("button", { name: "Run tool" }).click();
    await page.getByText("The tool reported an error").waitFor();
    await page.getByText("Issues are disabled for this repository.").waitFor();

    await page.getByRole("button", { name: "Call a tool by name" }).click();
    await page.getByRole("textbox", { name: "Tool name" }).fill("unknown_tool");
    await page.getByRole("button", { name: "Run tool" }).click();
    await page.getByText("-32602: Unknown tool: unknown_tool").waitFor();
    assert.deepEqual((await rpc("tools/call")).at(-1).body.params, {
      name: "unknown_tool",
      arguments: {},
    });
  });

  it("reads resources, gets prompts and records the exchanges", async () => {
    await load("/core/services/mcp-github/playground");
    await connect();

    await tab("Resources").click();
    await page.getByRole("button", { name: /^logo\.png/ }).click();
    await page.getByRole("button", { name: "Read resource" }).click();
    await page.getByRole("img", { name: "logo.png" }).waitFor();
    assert.equal(
      (await rpc("resources/read"))[0].headers["mcp-name"],
      "repo://octelium/octelium/logo.png",
    );

    await tab("Prompts").click();
    await page.getByRole("button", { name: /^review_pull_request/ }).click();
    await page.getByRole("textbox", { name: /^owner/ }).fill("octelium");
    await page.getByRole("textbox", { name: /^repo/ }).fill("octelium");
    await page.getByRole("textbox", { name: /^number/ }).fill("42");
    await page.getByRole("button", { name: "Get prompt" }).click();
    await page
      .getByText("Review pull request #42 of octelium/octelium.")
      .waitFor();
    assert.deepEqual((await rpc("prompts/get"))[0].body.params.arguments, {
      owner: "octelium",
      repo: "octelium",
      number: "42",
    });

    await tab("History").click();
    await page.getByText("7 requests").waitFor();
    await page.getByRole("button", { name: /^prompts\/get/ }).click();
    await page.getByText("Response · HTTP 200").waitFor();

    await page.getByRole("button", { name: "Disconnect" }).click();
    await page.getByText("Not connected", { exact: true }).waitFor();
    const remove = (await calls()).find((call) => call.method === "DELETE");
    assert.equal(remove.headers["mcp-session-id"], "mcp-session-7f3a9c2e");
  });

  for (const theme of ["light", "dark"]) {
    it(`fits phone, tablet and desktop widths in ${theme} mode`, async () => {
      for (const name of ["openai", "mcp-github"]) {
        for (const [width, height] of [
          [320, 640],
          [390, 844],
          [768, 1024],
          [1024, 768],
          [1440, 900],
        ]) {
          await page.setViewportSize({ width, height });
          await load(`/core/services/${name}/playground`, { theme });
          await page.locator(".mantine-Drawer-content").waitFor();
          await page.waitForTimeout(350);
          const layout = await page.evaluate(() => {
            const content = document.querySelector(".mantine-Drawer-content");
            return {
              page: document.documentElement.scrollWidth - window.innerWidth,
              drawer: content.scrollWidth - content.clientWidth,
              height: content.scrollHeight - content.clientHeight,
              scheme: document.documentElement.getAttribute(
                "data-mantine-color-scheme",
              ),
            };
          });
          assert.equal(layout.scheme, theme);
          assert.ok(layout.page <= 0, `${name} ${width}: page overflow`);
          assert.ok(layout.drawer <= 0, `${name} ${width}: drawer overflow`);
          if (width >= 1024) {
            assert.ok(layout.height <= 1, `${name} ${width}: drawer scrolls`);
          }
        }
      }
    });
  }
});
