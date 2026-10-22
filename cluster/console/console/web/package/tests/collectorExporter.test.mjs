import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("CollectorExporter edit", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-exporter-"));
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
    page.on("console", (message) => {
      if (message.type() === "error") errors.push(message.text());
    });
  });
  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const load = async (path, extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/collectorExporter.html?${new URLSearchParams({ path, ...extra })}`,
    );
    await page.getByText("Destination", { exact: true }).first().waitFor();
  };
  const value = () => page.evaluate(() => window.exporter.value);
  const choose = (group, name) =>
    page
      .getByRole("radiogroup", { name: group })
      .getByRole("radio", { name: new RegExp(`^${name}`) })
      .click();
  const pickDestination = async (query) => {
    await page.getByRole("button", { name: /Change the destination/ }).click();
    await page
      .getByRole("textbox", { name: "Search destinations" })
      .fill(query);
    await page.keyboard.press("Enter");
  };
  const overflow = () =>
    page.evaluate(() => {
      const root =
        document.querySelector("#resource-item-drawer .mantine-Drawer-body") ??
        document.body;
      const right = root.getBoundingClientRect().right;
      return [...root.querySelectorAll("*")]
        .filter((el) => {
          const rect = el.getBoundingClientRect();
          return rect.width > 0 && rect.right > right + 1;
        })
        .map((el) => el.tagName);
    });

  for (const [theme, width] of [
    ["light", 1440],
    ["dark", 1440],
    ["light", 390],
  ]) {
    it(`fits the item drawer in the ${theme} theme at ${width}px`, async () => {
      await page.setViewportSize({ width, height: 1000 });
      await load("/enterprise/collectorexporters/otlp-tempo/edit", { theme });
      await page.getByRole("button", { name: /Advanced options/ }).click();
      await page.getByText("Request headers", { exact: true }).waitFor();
      assert.deepEqual(await overflow(), []);
      assert.equal(
        await page.evaluate(() => document.documentElement.scrollWidth),
        width,
      );
    });
  }

  it("starts a new exporter without credentials and keeps the chosen method while switching", async () => {
    await load("/edit-only");
    await page.getByRole("radio", { name: /^None/ }).waitFor();
    assert.equal(
      await page
        .getByRole("radio", { name: /^None/ })
        .getAttribute("aria-checked"),
      "true",
    );

    await page.getByLabel("Endpoint").fill("collector.monitoring.svc:4317");
    await choose("Method", "Basic");
    await page.getByLabel("Username").fill("otel");
    assert.deepEqual((await value()).spec.otlp.auth.basic, {
      username: "otel",
      password: { fromSecret: "" },
    });

    await choose("Method", "None");
    assert.equal((await value()).spec.otlp.auth, undefined);

    await choose("Method", "Basic");
    assert.equal((await value()).spec.otlp.auth.basic.username, "otel");
  });

  it("finds destinations by keyword and restores the previous one on switching back", async () => {
    await load("/edit-only", { name: "otlp-tempo" });
    await page.getByRole("button", { name: /Change the destination/ }).click();
    await page
      .getByRole("textbox", { name: "Search destinations" })
      .fill("kusto");
    const options = page.getByRole("option");
    assert.deepEqual(
      await options
        .allTextContents()
        .then((items) => items.map((item) => item.split("Logs")[0])),
      ["Azure Data Explorer"],
    );
    await page.keyboard.press("Enter");

    let spec = (await value()).spec;
    assert.equal(spec.otlp, undefined);
    assert.deepEqual(Object.keys(spec.azureDataExplorer.auth), [
      "servicePrincipal",
    ]);
    await page
      .getByLabel("Cluster URI")
      .fill("http://octelium.kusto.windows.net");
    await page.getByText("Use an https:// URL", { exact: true }).waitFor();

    await pickDestination("grpc");
    spec = (await value()).spec;
    assert.equal(spec.azureDataExplorer, undefined);
    assert.equal(spec.otlp.endpoint, "tempo.observability.svc:4317");
    assert.equal(spec.otlp.auth.bearer.fromSecret, "tempo-token");
    assert.deepEqual(spec.otlp.headers, [
      { key: "x-scope-orgid", value: "platform" },
    ]);

    await pickDestination("adx");
    assert.equal(
      (await value()).spec.azureDataExplorer.clusterURI,
      "http://octelium.kusto.windows.net",
    );
  });

  it("hides advanced options until asked and counts the customized ones", async () => {
    await load("/edit-only", { name: "otlp-tempo" });
    const advanced = page.getByRole("button", { name: /Advanced options/ });
    assert.equal(await advanced.getAttribute("aria-expanded"), "false");
    await advanced.getByText("3 customized", { exact: true }).waitFor();
    assert.equal(
      await page.getByText("Request headers", { exact: true }).isVisible(),
      false,
    );

    await advanced.click();
    await page.getByText("Request headers", { exact: true }).waitFor();
    await page
      .getByRole("button", { name: "Remove request headers 1" })
      .click();
    await advanced.getByText("2 customized", { exact: true }).waitFor();
    assert.equal((await value()).spec.otlp.headers, undefined);

    await page.getByRole("button", { name: "Add header" }).click();
    await page.getByLabel("Request headers 1 key").fill("Authorization");
    await page
      .getByText("Use Authentication instead", { exact: true })
      .waitFor();
  });

  it("maps transport security to the TLS settings", async () => {
    await load("/edit-only", { name: "otlp-plain" });
    const transport = page.getByRole("combobox", {
      name: "Transport security",
    });
    await transport.click();
    await page.getByRole("option", { name: "Plaintext, no TLS" }).click();
    assert.deepEqual((await value()).spec.otlp.tls, { insecure: true });

    await transport.click();
    await page.getByRole("option", { name: "TLS, skip verification" }).click();
    assert.deepEqual((await value()).spec.otlp.tls, {
      insecureSkipVerify: true,
    });
    await page
      .getByText("The server certificate is not verified", { exact: true })
      .waitFor();

    await transport.click();
    await page
      .getByRole("option", { name: /^TLS, verify the certificate/ })
      .click();
    assert.equal((await value()).spec.otlp.tls, undefined);
  });

  it("keeps exactly one Elasticsearch destination while switching how to connect", async () => {
    await load("/edit-only", { name: "elastic" });
    await choose("Connect with", "Elastic Cloud");
    await page.getByLabel("Cloud ID").fill("deployment:abc");
    let spec = (await value()).spec.elasticsearch;
    assert.equal(spec.endpoints, undefined);
    assert.equal(spec.endpoint, undefined);
    assert.equal(spec.cloudID, "deployment:abc");

    await choose("Connect with", "Node URLs");
    spec = (await value()).spec.elasticsearch;
    assert.equal(spec.cloudID, undefined);
    assert.deepEqual(spec.endpoints, [
      "https://es-1.example.com:9200",
      "https://es-2.example.com:9200",
    ]);
  });

  it("creates ClickHouse exporters with the schema and without a password", async () => {
    await load("/edit-only");
    await pickDestination("clickhouse");
    const spec = (await value()).spec.clickhouse;
    assert.equal(spec.createSchema, true);
    assert.equal(spec.password, undefined);
  });

  it("keeps the Kafka leader acknowledgement when only the compression is set", async () => {
    await load("/edit-only", { name: "kafka" });
    await page.getByRole("button", { name: /Advanced options/ }).click();
    await page.getByRole("combobox", { name: "Compression" }).click();
    await page.getByRole("option", { name: "Zstandard" }).click();
    assert.deepEqual((await value()).spec.kafka.producer, {
      requiredAcks: 1,
      compression: "ZSTD",
    });

    await choose("Method", "SASL");
    const sasl = (await value()).spec.kafka.auth.sasl;
    assert.equal(sasl.mechanism, "SCRAM_SHA_512");
    await choose("Method", "None");
    assert.equal((await value()).spec.kafka.auth, undefined);
  });

  it("saves the edited exporter from the item drawer", async () => {
    await load("/enterprise/collectorexporters/splunk/edit");
    await page.getByLabel("Index").fill("security");
    await page.getByRole("button", { name: "Save changes" }).click();
    await page.waitForFunction(() =>
      window.exporter.calls.some(
        (call) => call.method === "updateCollectorExporter",
      ),
    );
    const call = await page.evaluate(() =>
      window.exporter.calls.find(
        (item) => item.method === "updateCollectorExporter",
      ),
    );
    assert.equal(call.request.spec.type.splunk.index, "security");
    assert.equal(
      call.request.spec.type.splunk.token.type.fromSecret,
      "splunk-hec-token",
    );
  });
});
