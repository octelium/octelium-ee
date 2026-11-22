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

  it("prefills a report from a component log entry without secrets in the URL or the request", async () => {
    const time = new Date(Date.now() - 5 * 60_000).toISOString();
    const secret = "password=hunter2 Authorization: Bearer abcdefghijklmnop";
    await load({
      report: JSON.stringify({
        subject: `gateway: ${secret}`,
        message: `runtime error: invalid memory address ${secret}`,
        time,
        level: "PANIC",
        component: { namespace: "octelium", type: "gateway", uid: "gw-1" },
      }),
    });
    const subject = await page.getByRole("textbox", { name: "Subject" }).inputValue();
    const description = await page.getByRole("textbox", { name: "Description" }).inputValue();
    assert.equal(subject, "gateway: password=[REDACTED] Authorization: [REDACTED]");
    assert.match(description, /runtime error: invalid memory address password=\[REDACTED\] Authorization: \[REDACTED\]/);
    await page.getByText("Secrets found in the reported entry were replaced with [REDACTED].", { exact: false }).waitFor();
    const url = await location();
    assert.match(url, /^\/support\/new\?report=[0-9a-f-]{36}&time=/);
    for (const value of [url, subject, description]) {
      assert.ok(!value.includes("hunter2"), value);
      assert.ok(!value.includes("abcdefghijklmnop"), value);
    }
    await page.getByText("Attach the component logs around the reported entry of gateway").waitFor();

    await page.getByRole("button", { name: "Send request" }).click();
    await page.getByRole("heading", { name: subject }).waitFor();
    const [create] = await calls("createSupportRequest");
    assert.equal(create.subject, subject);
    assert.ok(!JSON.stringify(create).includes("hunter2"));
    assert.ok(!JSON.stringify(create).includes("abcdefghijklmnop"));
  });

  it("applies the reported instance only to the component that logged the entry", async () => {
    const time = new Date(Date.now() - 5 * 60_000).toISOString();
    await load({
      report: JSON.stringify({
        message: "runtime error: invalid memory address",
        time,
        level: "PANIC",
        component: { namespace: "octelium", type: "gateway", uid: "gw-1" },
      }),
    });
    await page.getByRole("button", { name: "Review and attach" }).click();
    const drawer = page.getByRole("dialog").filter({ hasText: "Collect diagnostics" });
    assert.equal(
      await drawer.locator('input[type="radio"][value="around"]').isChecked(),
      true,
    );
    const instance = drawer.getByRole("switch", { name: "Only the instance that logged the entry" });
    assert.equal(await instance.isChecked(), true);
    await drawer.getByRole("button", { name: "Collect and review" }).click();
    await drawer.getByLabel("Diagnostics preview").waitFor();
    let [listCall] = await calls("listComponentLog");
    assert.deepEqual(listCall.component, {
      namespace: "octelium",
      type: "gateway",
      uid: "gw-1",
    });
    const from = Number(listCall.from.seconds) * 1000;
    assert.ok(Math.abs(from - (Date.parse(time) - 15 * 60_000)) < 2000);

    await drawer.getByRole("button", { name: "Back" }).click();
    await drawer.getByLabel("Component", { exact: true }).fill("apiserver");
    await page.getByRole("option", { name: "apiserver" }).click();
    assert.equal(await instance.count(), 0);
    await drawer.getByRole("button", { name: "Collect and review" }).click();
    await drawer.getByLabel("Diagnostics preview").waitFor();
    listCall = (await calls("listComponentLog")).at(-1);
    assert.deepEqual(listCall.component, { namespace: "", type: "apiserver", uid: "" });
  });

  it("collects the diagnostics of the window that ends when the collection starts", async () => {
    await page.clock.setFixedTime(new Date("2026-11-22T10:00:00.000Z"));
    await load({ path: "/support/new" });
    await page.getByRole("textbox", { name: "Subject" }).waitFor();

    await page.clock.setFixedTime(new Date("2026-11-22T11:00:00.000Z"));
    await page.getByRole("button", { name: "Collect diagnostics" }).click();
    const drawer = page.getByRole("dialog").filter({ hasText: "Collect diagnostics" });
    await page.clock.setFixedTime(new Date("2026-11-22T11:20:00.000Z"));
    await drawer.getByRole("button", { name: "Collect and review" }).click();
    await drawer.getByLabel("Diagnostics preview").waitFor();

    await drawer.getByRole("button", { name: "Back" }).click();
    await page.clock.setFixedTime(new Date("2026-11-22T11:30:00.000Z"));
    await drawer.getByRole("button", { name: "Collect and review" }).click();
    await drawer.getByLabel("Diagnostics preview").waitFor();

    const windows = (await calls("listComponentLog")).map((call) => [
      new Date(Number(call.from.seconds) * 1000).toISOString(),
      new Date(Number(call.to.seconds) * 1000).toISOString(),
    ]);
    assert.deepEqual(windows, [
      ["2026-11-22T10:20:00.000Z", "2026-11-22T11:20:00.000Z"],
      ["2026-11-22T10:30:00.000Z", "2026-11-22T11:30:00.000Z"],
    ]);
  });

  it("searches older requests when the latest page has none of this Cluster", async () => {
    await load({ list: "paged" });
    await page.getByText("No requests of this Cluster in the latest activity").waitFor();
    assert.equal(await page.getByText("No open requests").count(), 0);
    await page.getByRole("button", { name: "Search older requests" }).click();
    await requestsList().getByText("#1042").waitFor();
    assert.deepEqual(
      (await calls("listSupportRequest")).map((call) => call.cursor),
      ["", "page-2"],
    );
  });

  it("keeps the files that could not be attached to a new request until they are retried", async () => {
    await load({ path: "/support/new", failUpload: "probe.txt" });
    await page.getByRole("textbox", { name: "Subject" }).fill("Gateways restart");
    await page.getByRole("textbox", { name: "Description" }).fill("Every gateway restarts with a panic.");
    const [chooser] = await Promise.all([
      page.waitForEvent("filechooser"),
      page.getByRole("button", { name: "Attach files" }).click(),
    ]);
    await chooser.setFiles([
      { name: "probe.txt", mimeType: "text/plain", buffer: Buffer.from("ready: false") },
      { name: "notes.txt", mimeType: "text/plain", buffer: Buffer.from("notes") },
    ]);
    await page.getByRole("button", { name: "Send request" }).click();
    await page.getByText("Request #1043 was opened", { exact: true }).waitFor();
    assert.equal(await location(), "/support/new");
    const files = page.getByRole("list", { name: "Files to send" });
    await files.getByText("Upload failed").waitFor();
    assert.equal(await files.getByText("probe.txt").count(), 1);
    assert.equal(await files.getByText("notes.txt").count(), 0);
    assert.equal(await page.getByRole("textbox", { name: "Subject" }).isDisabled(), true);

    await page.getByRole("button", { name: "Retry the files" }).click();
    await page.getByRole("heading", { name: "Gateways restart" }).waitFor();
    assert.equal((await calls("createSupportRequest")).length, 1);
    const attempts = (await calls("addSupportRequestAttachment")).map((call) => [call.name, call.idempotencyKey]);
    assert.deepEqual(attempts.map(([name]) => name), ["probe.txt", "notes.txt", "probe.txt"]);
    assert.equal(attempts[0][1], attempts[2][1]);
    assert.notEqual(attempts[0][1], attempts[1][1]);
    assert.match(attempts[0][1], /^console-[0-9a-f-]{36}$/);
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

  it("counts the collected diagnostics against the file limit of a reply", async () => {
    await load({ path: `/support/requests/${WAITING}` });
    const reply = page.getByRole("form", { name: "Reply" });
    const [chooser] = await Promise.all([
      page.waitForEvent("filechooser"),
      reply.getByRole("button", { name: "Attach files" }).click(),
    ]);
    await chooser.setFiles(
      Array.from({ length: 8 }, (_, index) => ({
        name: `probe-${index}.txt`,
        mimeType: "text/plain",
        buffer: Buffer.from(`probe ${index}`),
      })),
    );
    const diagnosticsButton = reply.getByRole("button", { name: "Collect diagnostics" });
    const drawer = page.getByRole("dialog").filter({ hasText: "Collect diagnostics" });
    for (let i = 0; i < 2; i++) {
      await diagnosticsButton.click();
      await drawer.getByRole("button", { name: "Collect and review" }).click();
      await drawer.getByRole("checkbox", { name: /I reviewed these diagnostics/ }).check();
      await drawer.getByRole("button", { name: "Attach diagnostics" }).click();
      await drawer.waitFor({ state: "hidden" });
    }
    assert.equal(await reply.getByRole("list", { name: "Files to send" }).getByRole("listitem").count(), 10);
    assert.equal(await diagnosticsButton.isDisabled(), true);

    await reply.getByRole("textbox", { name: "Message" }).fill("The probes and the component logs.");
    await reply.getByRole("button", { name: "Send" }).click();
    await page.getByText("The probes and the component logs.").waitFor();
    const [message] = await calls("addSupportRequestMessage");
    assert.equal(message.attachmentIDs.length, 10);
  });

  it("refuses to save an attachment that does not match its size", async () => {
    await load({ path: `/support/requests/${WAITING}`, download: "truncated" });
    let downloaded = false;
    page.on("download", () => {
      downloaded = true;
    });
    await page.getByRole("button", { name: "Download component-logs.ndjson.gz" }).first().click();
    await page.getByText("The file download was incomplete. Please try again").waitFor();
    assert.equal(downloaded, false);
  });

  it("refreshes a request when an update returns only its summary", async () => {
    await load({ path: `/support/requests/${WAITING}`, detail: "unavailable" });
    await page.getByText("Thanks for the logs.", { exact: false }).waitFor();
    const loaded = (await calls("getSupportRequest")).length;
    await page.getByRole("button", { name: "Escalate" }).click();
    const modal = page.getByRole("dialog", { name: /Escalate request #1042/ });
    await modal.getByRole("textbox").fill("Production traffic is failing over constantly.");
    await modal.getByRole("button", { name: "Escalate" }).click();
    await modal.waitFor({ state: "hidden" });
    await page.getByText("Escalated", { exact: true }).first().waitFor();
    await page.getByText("Thanks for the logs.", { exact: false }).waitFor();
    await page.getByRole("button", { name: "Mark as resolved" }).waitFor();
    assert.ok((await calls("getSupportRequest")).length > loaded);
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
