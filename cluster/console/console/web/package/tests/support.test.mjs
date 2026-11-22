import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { gunzipSync } from "node:zlib";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

const WAITING = "0b1c2d3e-4f5a-4b6c-8d7e-9f0a1b2c3d4e";

describe("Support", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-support-"));
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
      viewport: { width: 1440, height: 1100 },
      acceptDownloads: true,
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

  const load = async (extra = {}) => {
    await page.goto(
      `${origin}tests/fixtures/support.html?${new URLSearchParams(extra)}`,
    );
  };
  const calls = (method) =>
    page.evaluate(
      (name) =>
        window.support.calls
          .filter((call) => call.method === name)
          .map((call) => JSON.parse(JSON.stringify(call.request))),
      method,
    );
  const location = () => page.evaluate(() => window.support.location);
  const requestsList = () => page.getByRole("list", { name: "Requests" });

  it("lists the requests of the Cluster and opens one", async () => {
    await load();
    await requestsList().getByText("#1042").waitFor();
    for (const text of [
      "Acme Inc.",
      "Standard support",
      "cluster.acme.example",
      "3f9a1c0d7e2b4a65",
      "Awaiting your reply",
    ]) {
      await page.getByText(text, { exact: true }).first().waitFor();
    }
    assert.equal(await requestsList().getByText("#1038").count(), 0);
    assert.deepEqual(
      (await calls("listSupportRequest")).map((call) => call.filter ?? 0),
      [1],
    );

    await page.locator('label:has-text("Resolved")').click();
    await requestsList().getByText("#1038").waitFor();
    assert.equal(await requestsList().getByText("#1042").count(), 0);
    assert.match(await location(), /status=closed/);

    await page.locator('label:has-text("All")').click();
    await requestsList().getByText("#1042").waitFor();
    await requestsList()
      .getByRole("link", { name: /Gateways restart every few minutes/ })
      .click();
    await page
      .getByRole("heading", { name: /Gateways restart every few minutes/ })
      .waitFor();
    assert.equal(await location(), `/support/requests/${WAITING}`);
    await page.getByRole("link", { name: "Support" }).first().waitFor();
  });

  for (const [state, title, action] of [
    ["no-license", "This Cluster needs a valid license", "Manage the license"],
    ["unreachable", "The customer portal cannot be reached", "Customer portal"],
    ["not-entitled", "Technical support is not included", "Customer portal"],
    ["unauthorized", "Octelium rejected the license of this Cluster", "support@octelium.com"],
  ]) {
    it(`explains why requests cannot be opened (${state})`, async () => {
      await load({ state });
      const notice = page.getByRole("status").filter({ hasText: title });
      await notice.waitFor();
      await notice.getByRole("link", { name: action }).waitFor();
      assert.equal(
        await page
          .getByRole("link", { name: "New request" })
          .first()
          .getAttribute("data-disabled"),
        "true",
      );
      assert.deepEqual(await calls("listSupportRequest"), []);
    });
  }

  it("opens a request with reviewed and redacted diagnostics", async () => {
    await load({ path: "/support/new" });
    const send = page.getByRole("button", { name: "Send request" });
    await send.waitFor();
    assert.equal(await send.isDisabled(), true);

    await page.getByRole("radio", { name: /P2\s*High/ }).click();
    assert.equal(
      await page.getByRole("radio", { name: /P2\s*High/ }).getAttribute("aria-checked"),
      "true",
    );
    await page.getByRole("textbox", { name: "Subject" }).fill("Gateways restart");
    assert.equal(await send.isDisabled(), true);
    const description = page.getByRole("textbox", { name: "Description" });
    await description.fill("Every gateway restarts with a panic.");
    assert.equal(await send.isDisabled(), false);

    await page.getByRole("button", { name: "Cluster details" }).click();
    await page.getByText("ENTERPRISE E2").waitFor();

    await page.getByRole("button", { name: "Collect diagnostics" }).click();
    const drawer = page.getByRole("dialog").filter({ hasText: "Collect diagnostics" });
    await drawer.getByRole("button", { name: "Collect and review" }).click();
    const preview = drawer.getByLabel("Diagnostics preview");
    await preview.waitFor();
    const text = await drawer.innerText();
    for (const secret of [
      "hunter2-secret",
      "abcdefghijklmnop0123",
      "sk_live_123",
      "alice@acme.example",
      "eyJzdWIiOiIxMjM0In0",
    ]) {
      assert.ok(!text.includes(secret), secret);
    }
    await drawer.getByText("Secret fields · 1").waitFor();
    await drawer.getByText("JWTs · 1").waitFor();

    const [listCall] = await calls("listComponentLog");
    assert.deepEqual(listCall.levels, [4, 5, 6]);

    const attach = drawer.getByRole("button", { name: "Attach diagnostics" });
    assert.equal(await attach.isDisabled(), true);
    await drawer.getByRole("checkbox", { name: /I reviewed these diagnostics/ }).check();
    await attach.click();
    await page.getByRole("list", { name: "Files to send" }).getByText(/component-logs-.*\.ndjson\.gz/).waitFor();

    await send.click();
    await page.getByRole("heading", { name: "Gateways restart" }).waitFor();

    const [create] = await calls("createSupportRequest");
    assert.equal(create.subject, "Gateways restart");
    assert.equal(create.description, "Every gateway restarts with a panic.");
    assert.equal(create.priority, 2);
    assert.match(create.idempotencyKey, /^console-[0-9a-f-]{36}$/);

    const uploads = await page.evaluate(() => window.support.uploads);
    assert.equal(uploads.length, 1);
    assert.equal(uploads[0].purpose, 2);
    const ndjson = gunzipSync(Buffer.from(uploads[0].data)).toString("utf8");
    const [manifest, ...entries] = ndjson.trim().split("\n").map((line) => JSON.parse(line));
    assert.equal(manifest.type, "manifest");
    assert.equal(manifest.included, 3);
    assert.equal(manifest.clusterDomain, "cluster.acme.example");
    assert.equal(manifest.redaction.counts.field, 1);
    assert.equal(entries[0].level, "ERROR");
    for (const secret of ["hunter2-secret", "abcdefghijklmnop0123", "sk_live_123", "alice@acme.example"]) {
      assert.ok(!ndjson.includes(secret), secret);
    }
    assert.match(await location(), /^\/support\/requests\/3e4f5a6b-/);
  });

  it("prefills a report from a component log entry", async () => {
    const time = new Date(Date.now() - 5 * 60_000).toISOString();
    await load({
      path: `/support/new?${new URLSearchParams({
        subject: "gateway: runtime error",
        message: "runtime error: invalid memory address",
        time,
        level: "PANIC",
        "component.namespace": "octelium",
        "component.type": "gateway",
        "component.uid": "gw-1",
      })}`,
    });
    assert.equal(
      await page.getByRole("textbox", { name: "Subject" }).inputValue(),
      "gateway: runtime error",
    );
    assert.match(
      await page.getByRole("textbox", { name: "Description" }).inputValue(),
      /runtime error: invalid memory address/,
    );
    await page.getByText("Attach the component logs around the reported entry of gateway").waitFor();
    await page.getByRole("button", { name: "Review and attach" }).click();
    const drawer = page.getByRole("dialog").filter({ hasText: "Collect diagnostics" });
    assert.equal(
      await drawer.locator('input[type="radio"][value="around"]').isChecked(),
      true,
    );
    await drawer.getByRole("button", { name: "Collect and review" }).click();
    await drawer.getByLabel("Diagnostics preview").waitFor();
    const [listCall] = await calls("listComponentLog");
    assert.deepEqual(listCall.component, {
      namespace: "octelium",
      type: "gateway",
      uid: "gw-1",
    });
    const from = Number(listCall.from.seconds) * 1000;
    assert.ok(Math.abs(from - (Date.parse(time) - 15 * 60_000)) < 2000);
  });

  it("follows a request: replies with files, escalates and resolves it", async () => {
    await load({ path: `/support/requests/${WAITING}` });
    await page.getByText("Octelium is waiting for your reply").waitFor();
    await page.getByText("Thanks for the logs.", { exact: false }).waitFor();
    await page.getByText("changed the priority from P1 to P2").waitFor();
    await page.getByText("One Region of two is affected and traffic fails over.").waitFor();
    assert.equal(await page.locator("b", { hasText: "Not HTML" }).count(), 0);

    await page.getByRole("button", { name: "Show earlier activity" }).click();
    await page.getByText("An earlier message").waitFor();

    const [download] = await Promise.all([
      page.waitForEvent("download"),
      page.getByRole("button", { name: "Download component-logs.ndjson.gz" }).first().click(),
    ]);
    assert.equal(download.suggestedFilename(), "component-logs.ndjson.gz");

    const reply = page.getByRole("form", { name: "Reply" });
    const [chooser] = await Promise.all([
      page.waitForEvent("filechooser"),
      reply.getByRole("button", { name: "Attach files" }).click(),
    ]);
    await chooser.setFiles([
      { name: "probe.txt", mimeType: "text/plain", buffer: Buffer.from("ready: false") },
      { name: "run.exe", mimeType: "application/octet-stream", buffer: Buffer.from("x") },
    ]);
    await page.getByText("run.exe: this file type is not accepted").waitFor();
    await reply.getByText("probe.txt").waitFor();
    await reply.getByRole("textbox", { name: "Message" }).fill("The probe reports ready: false.");
    await reply.getByRole("button", { name: "Send" }).click();
    await page.getByText("The probe reports ready: false.").waitFor();
    const [message] = await calls("addSupportRequestMessage");
    assert.equal(message.body, "The probe reports ready: false.");
    assert.equal(message.attachmentIDs.length, 1);
    assert.match(message.idempotencyKey, /^console-/);
    assert.equal(await reply.getByRole("textbox", { name: "Message" }).inputValue(), "");

    await page.getByRole("button", { name: "Escalate" }).click();
    const modal = page.getByRole("dialog", { name: /Escalate request #1042/ });
    const confirm = modal.getByRole("button", { name: "Escalate" });
    await modal.getByRole("textbox").fill("Urgent");
    assert.equal(await confirm.isDisabled(), true);
    await modal.getByRole("textbox").fill("Production traffic is failing over constantly.");
    await confirm.click();
    await modal.waitFor({ state: "hidden" });
    await page.getByText("Escalated", { exact: true }).first().waitFor();
    assert.equal(await page.getByRole("button", { name: "Escalate" }).count(), 0);

    await page.getByRole("button", { name: "Mark as resolved" }).click();
    await page
      .getByRole("dialog", { name: /Mark request #1042 as resolved/ })
      .getByRole("button", { name: "Mark as resolved" })
      .click();
    await page.getByText("This request is resolved").waitFor();
    assert.equal((await calls("resolveSupportRequest")).length, 1);
  });

  it("explains that requests of other installations are not available", async () => {
    await load({ path: "/support/requests/9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d" });
    await page.getByText("This request is not available from this Cluster").waitFor();
    errors = errors.filter((error) => !error.includes("404"));
  });

  for (const theme of ["light", "dark"]) {
    for (const width of [320, 390, 768, 1280]) {
      it(`fits ${width}px in the ${theme} theme`, async () => {
        await page.setViewportSize({ width, height: 900 });
        for (const path of ["/support", "/support/new", `/support/requests/${WAITING}`]) {
          await load({ path, theme, width: String(width) });
          await page.getByRole("link", { name: "Support" }).first().waitFor();
          await page.waitForTimeout(150);
          const overflow = await page.evaluate(
            () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
          );
          assert.ok(overflow <= 1, `${path} overflows by ${overflow}px`);
        }
      });
    }
  }
});
