import assert from "node:assert/strict";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, before, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium, firefox } from "playwright";
import { build, preview } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Production access portal", () => {
  let cacheDir;
  let server;
  let origin;
  let manifest;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-portal-build-"));
    const outDir = join(cacheDir, "dist");
    await build({
      root,
      cacheDir,
      logLevel: "error",
      plugins: [{
        name: "portal-test-entry",
        transformIndexHtml: {
          order: "pre",
          handler: (html) => html.replace("/src/main.tsx", "/tests/fixtures/portal.tsx"),
        },
      }],
      build: { outDir },
    });
    manifest = JSON.parse(await readFile(join(outDir, ".vite/manifest.json"), "utf8"));
    server = await preview({
      root,
      build: { outDir },
      preview: { host: "127.0.0.1", port: 0 },
    });
    origin = server.resolvedUrls.local[0];
  });

  after(async () => {
    if (server) await new Promise((resolve) => server.httpServer.close(resolve));
    if (cacheDir) await rm(cacheDir, { recursive: true, force: true });
  });

  for (const [name, browserType] of Object.entries({ chromium, firefox })) {
    it(`loads secondary routes on demand in ${name}`, async () => {
      const browser = await browserType.launch();
      try {
        const page = await browser.newPage({ reducedMotion: "reduce" });
        page.setDefaultTimeout(15_000);
        const errors = [];
        const requested = new Set();
        page.on("pageerror", (error) => errors.push(error.message));
        page.on("request", (request) => requested.add(new URL(request.url()).pathname));
        await page.addInitScript(() => {
          window.__OCTELIUM_GLOBALS__ = { cluster: { domain: "example.test" } };
          document.cookie = "octelium_domain=namespace.example.test; path=/";
        });
        await page.goto(`${origin}?path=/user/requests`);
        await page.getByRole("heading", { name: "My Requests", exact: true }).waitFor()
          .catch((error) => assert.fail(`${error.message}\nBrowser errors: ${errors.join("\n")}`));

        assert.equal(
          await page.getByRole("link", { name: "Authenticators", exact: true }).getAttribute("href"),
          "https://example.test/authenticators",
        );

        const routes = [
          ["/user/new", "New Request", "src/pages/user/Request/new.tsx"],
          ["/reviewer/requests", "Review Queue (2)", "src/pages/reviewer/Review/queue.tsx"],
          ["/reviewer/requests/queue-first", "Review default.database", "src/pages/reviewer/Review/request.tsx"],
          ["/reviewer/reviews", "My Reviews", "src/pages/reviewer/Review/index.tsx"],
          ["/reviewer/reviews/rv1.queue-second", "Review of queue-second", "src/pages/reviewer/Review/detail.tsx"],
          ["/user/requests/own-pending", "default.database", "src/pages/user/Request/detail.tsx"],
        ];

        for (const [, , source] of routes) {
          assert.equal(manifest[source].isDynamicEntry, true);
          assert.equal(requested.has(`/${manifest[source].file}`), false);
        }

        for (const [path, title, source] of routes) {
          await page.evaluate((target) => window.portalRouter.navigate(target), path);
          await page.waitForFunction(
            (expected) => document.title === `${expected} · Octelium Access Portal`,
            title,
          );
          await page.getByRole("heading", { level: 1 }).waitFor();
          assert.equal(requested.has(`/${manifest[source].file}`), true);
        }

        await page.getByRole("button", { name: "Edit", exact: true }).click();
        const edit = page.getByRole("dialog", { name: "Edit pending request", exact: true });
        await edit.getByRole("button", { name: "Custom duration", exact: true }).click();
        const custom = page.getByRole("dialog", { name: "Custom duration", exact: true });
        const amount = custom.getByRole("textbox", { name: "Duration amount", exact: true });
        await amount.fill("6");
        await amount.press("Enter");
        await custom.waitFor({ state: "hidden" });
        await edit.getByText("Custom duration: 6 hours", { exact: true }).waitFor();
        await edit.getByRole("textbox", { name: /Justification/ }).fill("Production draft");
        await edit.getByRole("button", { name: "Discard", exact: true }).click();
        await page.getByRole("dialog", { name: "Discard unsaved changes?", exact: true })
          .getByRole("button", { name: "Keep editing", exact: true }).click();
        assert.equal(await edit.getByRole("textbox", { name: /Justification/ }).inputValue(), "Production draft");
        assert.deepEqual(errors, []);
      } finally {
        await browser.close();
      }
    });
  }
});
