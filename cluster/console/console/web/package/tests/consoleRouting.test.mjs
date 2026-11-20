import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { build, preview } from "vite";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

const root = fileURLToPath(new URL("../", import.meta.url));

const grpcResponse = (message = new Uint8Array()) => {
  const frame = (payload, flag) => {
    const header = Buffer.alloc(5);
    header[0] = flag;
    header.writeUInt32BE(payload.length, 1);
    return Buffer.concat([header, payload]);
  };
  return Buffer.concat([
    frame(message, 0),
    frame(Buffer.from("grpc-status: 0\r\n"), 128),
  ]);
};

describe("Console browser routing", () => {
  let directory, server, browser, context, page, origin, errors, responses;

  before(async () => {
    const modules = await createConsoleModules();
    try {
      const { GetStatusResponse } = await modules.load("apis/userv1/userv1.ts");
      const { User, UserList } = await modules.load("apis/corev1/corev1.ts");
      const user = User.create({
        apiVersion: "core/v1",
        kind: "User",
        metadata: { name: "alice.default", uid: "user-1" },
        spec: { email: "alice@example.test", type: 1 },
      });
      responses = new Map([
        ["GetStatus", GetStatusResponse.toBinary(GetStatusResponse.create({
          domain: "example.test",
          user: { metadata: user.metadata, spec: { email: "alice@example.test", type: 1 } },
          session: { metadata: { uid: "session-1" } },
        }))],
        ["GetUser", User.toBinary(user)],
        ["ListUser", UserList.toBinary(UserList.create({
          apiVersion: "core/v1", kind: "UserList", items: [user],
        }))],
      ]);
    } finally {
      await modules.close();
    }
    directory = await mkdtemp(join(tmpdir(), "console-routing-"));
    await build({ root, logLevel: "silent", build: { outDir: directory } });
    server = await preview({
      root,
      logLevel: "silent",
      build: { outDir: directory },
      preview: { host: "127.0.0.1", port: 0 },
    });
    origin = server.resolvedUrls.local[0];
    browser = await chromium.launch();
  });

  after(async () => {
    await browser?.close();
    if (server) await new Promise((resolve) => server.httpServer.close(resolve));
    if (directory) await rm(directory, { recursive: true, force: true });
  });

  beforeEach(async () => {
    context = await browser.newContext({ reducedMotion: "reduce", viewport: { width: 1440, height: 900 } });
    page = await context.newPage();
    errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.addInitScript(() => {
      window.__OCTELIUM_GLOBALS__ = { cluster: { domain: "example.test" } };
    });
    await page.route(/\/octelium\.api\./, async (route) => {
      const request = route.request();
      await route.fulfill({
        headers: {
          "access-control-allow-origin": new URL(origin).origin,
          "access-control-allow-credentials": "true",
          "access-control-allow-headers": request.headers()["access-control-request-headers"] ?? "*",
          "access-control-allow-methods": "POST, OPTIONS",
          "content-type": "application/grpc-web+proto",
        },
        body: request.method() === "OPTIONS"
          ? ""
          : grpcResponse(responses.get(new URL(request.url()).pathname.split("/").at(-1))),
      });
    });
  });

  afterEach(async () => {
    await context?.close();
    assert.deepEqual(errors, []);
  });

  const usersReady = () =>
    page.locator('a[href^="/core/users/alice.default"]').first().waitFor();

  it("preserves nested routes and search parameters through browser history and reload", async () => {
    await page.goto(`${origin}core/users?q=alice`);
    await usersReady();
    await page.locator('a[href="/settings"]').first().click();
    await page.waitForURL(`${origin}settings`);
    await page.goBack();
    await page.waitForURL(`${origin}core/users?q=alice`);
    await usersReady();
    await page.goForward();
    await page.waitForURL(`${origin}settings`);
    await page.goBack();
    await page.reload();
    await usersReady();
    assert.equal(page.url(), `${origin}core/users?q=alice`);
    assert.equal(await page.getByRole("alert").count(), 0);
  });

  it("preserves redirects into lazy resource routes", async () => {
    await page.goto(`${origin}?redirect=${encodeURIComponent("/core/users?q=alice")}`);
    await page.waitForURL(`${origin}core/users?q=alice`);
    await usersReady();
    assert.equal(await page.getByRole("alert").count(), 0);
  });
});
