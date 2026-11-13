import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

const CA = `-----BEGIN CERTIFICATE-----
MIIBszCCAVmgAwIBAgIUQ2xhdWRlRmFrZUNBMDAwMDAwMDAwMDAwCgYIKoZIzj0E
-----END CERTIFICATE-----`;

describe("Cluster management", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-clusterman-"));
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
      `${origin}tests/fixtures/clusterman.html?${new URLSearchParams(extra)}`,
    );
    await regionsPanel().getByText("index 0").waitFor();
  };
  const regionsPanel = () =>
    page.locator("section").filter({
      has: page.getByText("The Kubernetes clusters that make up the Cluster"),
    });
  const drawer = () =>
    page.getByRole("dialog").filter({ hasText: "Join a Region to" });
  const joins = () => page.evaluate(() => window.clusterman.joins);
  const openJoin = async () => {
    await regionsPanel().getByRole("button", { name: "Join Region" }).click();
    await drawer().getByText("Kubernetes connection").waitFor();
  };
  const pickSecret = async (name) => {
    await drawer()
      .getByRole("button", { name: /Select a Secret/ })
      .click();
    await page.getByRole("option", { name: new RegExp(name) }).click();
    await drawer().getByText(name).first().waitFor();
  };
  const submit = () =>
    drawer().getByRole("button", { name: "Join Region" }).click();
  const licenseDrawer = () =>
    page.getByRole("dialog").filter({
      has: page.getByRole("textbox", { name: "License", exact: true }),
    });
  const licenseRequests = () =>
    page.evaluate(() =>
      window.clusterman.calls
        .filter((call) => call.method === "setLicense")
        .map((call) => call.request),
    );

  it("loads, verifies and applies a license file", async () => {
    await load();
    await page.getByRole("button", { name: "Set license", exact: true }).click();
    const input = licenseDrawer().getByRole("textbox", { name: "License" });
    const apply = licenseDrawer().getByRole("button", { name: "Apply license" });
    const contents = "  eyJhbGciOi.first-license.signature\n";
    const [chooser] = await Promise.all([
      page.waitForEvent("filechooser"),
      licenseDrawer().getByRole("button", { name: "Load from file" }).click(),
    ]);
    await chooser.setFiles({
      name: "license.txt",
      mimeType: "text/plain",
      buffer: Buffer.from(contents),
    });
    await page.waitForFunction(
      (value) => document.querySelector("textarea")?.value === value,
      contents,
    );
    assert.deepEqual(await licenseRequests(), []);
    assert.equal(await apply.isDisabled(), true);

    await licenseDrawer()
      .getByRole("button", { name: "Verify", exact: true })
      .click();
    await licenseDrawer()
      .getByText("Verified license", { exact: true })
      .waitFor();
    await licenseDrawer()
      .getByRole("button", { name: "Apply license", disabled: false })
      .waitFor();
    assert.deepEqual(await licenseRequests(), [
      { jwt: contents.trim(), dryRun: true },
    ]);

    const replacement = "\neyJhbGciOi.replacement-license.signature  ";
    await licenseDrawer().locator('input[type="file"]').setInputFiles({
      name: "license.txt",
      mimeType: "text/plain",
      buffer: Buffer.from(replacement),
    });
    await page.waitForFunction(
      (value) => document.querySelector("textarea")?.value === value,
      replacement,
    );
    assert.equal(await apply.isDisabled(), true);
    assert.equal(
      await licenseDrawer()
        .getByText("Verified license", { exact: true })
        .count(),
      0,
    );

    await licenseDrawer()
      .getByRole("button", { name: "Verify", exact: true })
      .click();
    await licenseDrawer()
      .getByText("Verified license", { exact: true })
      .waitFor();
    await apply.click();
    await page.getByText("License applied", { exact: true }).waitFor();
    await licenseDrawer().waitFor({ state: "hidden" });
    await page.getByText("Example Ltd", { exact: true }).waitFor();
    assert.deepEqual(await licenseRequests(), [
      { jwt: contents.trim(), dryRun: true },
      { jwt: replacement.trim(), dryRun: true },
      { jwt: replacement.trim(), dryRun: false },
    ]);

    await page.getByRole("button", { name: "Replace", exact: true }).click();
    assert.equal(await input.inputValue(), "");
    assert.equal(await apply.isDisabled(), true);
  });

  it("rejects non-UTF-8 license files without replacing the entered license", async () => {
    await load();
    await page.getByRole("button", { name: "Set license", exact: true }).click();
    const input = licenseDrawer().getByRole("textbox", { name: "License" });
    await input.fill("eyJhbGciOi.entered-license.signature");
    await licenseDrawer().locator('input[type="file"]').setInputFiles({
      name: "invalid.txt",
      mimeType: "text/plain",
      buffer: Buffer.from([0xff, 0xfe]),
    });
    await licenseDrawer()
      .getByText("The selected file is not valid UTF-8 text.")
      .waitFor();
    assert.equal(
      await input.inputValue(),
      "eyJhbGciOi.entered-license.signature",
    );
    assert.deepEqual(await licenseRequests(), []);

    await licenseDrawer().locator('input[type="file"]').setInputFiles({
      name: "license.txt",
      mimeType: "text/plain",
      buffer: Buffer.from("eyJhbGciOi.loaded-license.signature"),
    });
    await page.waitForFunction(
      () =>
        document.querySelector("textarea")?.value ===
        "eyJhbGciOi.loaded-license.signature",
    );
    assert.equal(
      await licenseDrawer()
        .getByText("The selected file is not valid UTF-8 text.")
        .count(),
      0,
    );
  });

  it("lists the Regions with their versions and the failed join", async () => {
    await load();
    const panel = regionsPanel();

    const names = await panel.locator("a").allTextContents();
    assert.deepEqual(names, ["default", "US West", "east"]);

    await panel.getByText("3 Regions").waitFor();
    await panel.getByText("Default", { exact: true }).waitFor();
    await panel.getByText("Incomplete", { exact: true }).waitFor();
    await panel.getByText("west · index 1").waitFor();
    await panel
      .getByText("The join did not complete. Join the Region again to retry.")
      .waitFor();
    assert.equal(await panel.getByText("Cordium").count(), 1);
    assert.equal(await panel.getByText("Enterprise").count(), 2);

    await panel
      .getByText("The last join of the Region east did not complete")
      .waitFor();
    await panel.getByText(/connect: connection refused/).waitFor();

    assert.equal(
      await panel.getByRole("button", { name: "Join Region" }).isEnabled(),
      true,
    );

    await page
      .getByText(
        "Failed in the Region west: core upgrade failed: genesis job failed: reason=BackoffLimitExceeded",
      )
      .waitFor();
  });

  it("validates and submits a kubeconfig join", async () => {
    await load();
    await openJoin();

    await submit();
    await drawer().getByText("A name is required").waitFor();
    await drawer().getByText("A Secret is required").waitFor();
    assert.deepEqual(await joins(), []);

    const name = drawer().getByRole("textbox", { name: "Name", exact: true });
    for (const [value, message] of [
      ["default", "The default Region already exists"],
      ["west", "A Region with this name already exists"],
      ["West_1", "Use lowercase letters, digits and single hyphens"],
      ["us--east", "Use lowercase letters, digits and single hyphens"],
    ]) {
      await name.fill(value);
      await drawer().getByText(message).waitFor();
    }

    await name.fill("east");
    assert.equal(
      await drawer()
        .getByText("A Region with this name already exists")
        .count(),
      0,
    );

    await name.fill("us-east-1");
    await drawer()
      .getByRole("textbox", { name: "Display name" })
      .fill("  US East ");
    await pickSecret("west-kubeconfig");
    await drawer()
      .getByRole("textbox", { name: "Context" })
      .fill(" admin@us-east ");

    await drawer().getByText("0.28.0").waitFor();
    await drawer().getByText("0.14.0").waitFor();

    await submit();
    await page.getByText("Joining the Region us-east-1 has started").waitFor();
    await drawer().waitFor({ state: "hidden" });

    assert.deepEqual(await joins(), [
      {
        name: "us-east-1",
        displayName: "US East",
        kubernetes: {
          kubeconfig: {
            fromSecret: "west-kubeconfig",
            context: "admin@us-east",
          },
        },
      },
    ]);

    await openJoin();
    assert.equal(
      await drawer()
        .getByRole("textbox", { name: "Name", exact: true })
        .inputValue(),
      "",
    );
    assert.equal(
      await drawer().getByRole("textbox", { name: "Context" }).inputValue(),
      "",
    );
    await drawer()
      .getByRole("button", { name: /Select a Secret/ })
      .waitFor();
    assert.equal(await drawer().getByText("A name is required").count(), 0);
  });

  it("validates and submits a bearer token join", async () => {
    await load();
    await openJoin();

    await drawer().locator("label").filter({ hasText: "Bearer token" }).click();
    await drawer()
      .getByRole("textbox", { name: "Name", exact: true })
      .fill("south");
    await pickSecret("south-token");

    const server = drawer().getByRole("textbox", { name: "API server URL" });
    for (const [value, message] of [
      ["http://203.0.113.30:6443", "Use an https:// URL"],
      [
        "https://admin:secret@203.0.113.30:6443",
        "The URL must not contain credentials",
      ],
      ["not a url", "Invalid URL"],
    ]) {
      await server.fill(value);
      await drawer().getByText(message).waitFor();
    }

    await server.fill("https://203.0.113.30:6443");
    await drawer()
      .getByRole("textbox", { name: "Trusted CA certificates" })
      .fill(`${CA}\n\n${CA}\n`);

    await submit();
    await drawer().waitFor({ state: "hidden" });

    assert.deepEqual(await joins(), [
      {
        name: "south",
        kubernetes: {
          bearerToken: {
            fromSecret: "south-token",
            server: "https://203.0.113.30:6443",
            trustedCAs: [CA, CA],
          },
        },
      },
    ]);
  });

  it("keeps the drawer open when the join is rejected", async () => {
    await load();
    await openJoin();
    await drawer()
      .getByRole("textbox", { name: "Name", exact: true })
      .fill("conflict");
    await pickSecret("west-kubeconfig");
    await submit();
    await page
      .getByText("A Region join is already requested")
      .first()
      .waitFor();
    assert.equal(await drawer().isVisible(), true);
    assert.equal((await joins()).length, 1);
    errors.length = 0;
  });

  it("shows an ongoing join and blocks new operations", async () => {
    await load({ scenario: "joining" });
    const panel = regionsPanel();
    await panel.getByText("Joining the Region south").waitFor();
    await panel.getByText("Joining", { exact: true }).waitFor();

    const button = panel.getByRole("button", { name: "Join in progress" });
    assert.equal(await button.isDisabled(), true);
    assert.equal(
      await panel
        .getByText("The last join of the Region east did not complete")
        .count(),
      0,
    );
  });

  it("shows the per-Region progress of an ongoing upgrade", async () => {
    await load({ scenario: "upgrading" });
    await page.getByText("An upgrade is rolling out").waitFor();

    const rollout = page.locator("section").filter({
      has: page.getByRole("heading", { name: "Upgrade rollout" }),
    });
    await rollout
      .locator("span")
      .filter({ hasText: /^default\s*Succeeded$/ })
      .first()
      .waitFor();
    await rollout
      .locator("span")
      .filter({ hasText: /^west\s*Upgrading$/ })
      .first()
      .waitFor();

    assert.equal(
      await regionsPanel()
        .getByRole("button", { name: "Join Region" })
        .isDisabled(),
      true,
    );
  });

  for (const theme of ["light", "dark"]) {
    it(`fits the page and the drawer at phone width in the ${theme} theme`, async () => {
      await page.setViewportSize({ width: 360, height: 900 });
      await load({ theme, width: "360" });

      const overflow = () =>
        page.evaluate(
          () => document.documentElement.scrollWidth - window.innerWidth,
        );
      assert.ok((await overflow()) <= 0, `page overflow ${await overflow()}`);

      await openJoin();
      const box = await drawer().boundingBox();
      assert.ok(box && box.width <= 360, `drawer width ${box?.width}`);
      const inner = await drawer().evaluate((node) => {
        const body = node.querySelector(".mantine-Drawer-body");
        return body ? body.scrollWidth - body.clientWidth : 0;
      });
      assert.ok(inner <= 0, `drawer overflow ${inner}`);
    });
  }
});
