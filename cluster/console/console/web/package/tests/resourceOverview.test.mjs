import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

const paths = [
  "/core/sessions/alice-browser-k3x9",
  "/core/sessions/bob-client-q7m2",
  "/core/services/ssh-k8s-2.default",
  "/core/services/grafana.monitoring",
  "/core/users/alice",
  "/core/groups/engineering",
  "/core/namespaces/default",
  "/core/credentials/bob-auth-token-w8k2",
  "/core/devices/alice-macbook",
  "/core/authenticators/alice-yubikey",
  "/core/identityproviders/github",
  "/core/policies/allow-engineering",
  "/core/regions/default",
  "/core/gateways/default-0",
  "/core/secrets/db-password",
  "/enterprise/certificates/cluster-wildcard",
  "/enterprise/certificates/partner-api",
  "/enterprise/certificateissuers/letsencrypt",
  "/enterprise/secretstores/vault",
  "/enterprise/devicemanagers/crowdstrike",
  "/enterprise/devicemanagers/intune",
  "/enterprise/devicemanagers/jamf",
  "/enterprise/devicemanagers/sentinelone",
  "/enterprise/devicemanagers/fleet",
  "/enterprise/devicemanagers/onepassword",
  "/enterprise/devicemanagers/huntress",
  "/enterprise/devicemanagers/iru",
  "/enterprise/directoryproviders/google",
  "/enterprise/directoryproviders/okta",
  "/enterprise/dnsproviders/cloudflare",
  "/enterprise/collectorexporters/datadog",
  "/enterprise/secrets/crt-cluster-wildcard",
  "/access/requests/req-dba-live",
  "/access/integrations/slack-access",
  "/cordium/workspaces/aoo",
  "/cordium/templates/default.default.alice",
  "/cordium/spaces/default.alice",
  "/cordium/memberships/default.alice.bob",
  "/cordium/gitproviders/github.default.alice",
  "/cordium/usersecrets/alice-ssh",
  "/cordium/regions/default",
];

describe("Resource overview details", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-overview-"));
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
      timezoneId: "UTC",
      viewport: { width: 1440, height: 1000 },
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
      `${origin}tests/fixtures/resourceOverview.html?${new URLSearchParams({ path, ...extra })}`,
    );
    await page.waitForFunction(() => window.overview);
    assert.deepEqual(await page.evaluate(() => window.overview.errors), []);
    await page.getByRole("radiogroup", { name: "Overview content" }).waitFor();
  };

  const calls = (method) =>
    page.evaluate(
      (name) => window.overview.calls.filter((call) => call.method === name),
      method,
    );

  const toolbar = (name) =>
    page.getByRole("toolbar", { name: `Actions for ${name}` });

  const sectionTitles = () =>
    page.locator("#resource-item-drawer section h3").allTextContents();

  const overflow = () =>
    page.evaluate(() => {
      const card = document
        .querySelector('[role="radiogroup"][aria-label="Overview content"]')
        ?.closest("section");
      if (!card) return ["missing overview"];
      const limit = card.getBoundingClientRect().right + 0.5;
      const scrolls = (element) => {
        for (let node = element.parentElement; node && node !== card; node = node.parentElement) {
          if (["auto", "scroll"].includes(getComputedStyle(node).overflowX)) return true;
        }
        return false;
      };
      return [...card.querySelectorAll("*")]
        .filter((element) => {
          const box = element.getBoundingClientRect();
          return box.width > 0 && box.right > limit && !scrolls(element);
        })
        .map((element) => `${element.tagName}.${element.className}`)
        .slice(0, 3);
    });

  for (const [width, theme] of [
    [1440, "light"],
    [390, "dark"],
  ]) {
    it(`keeps every overview inside the drawer at ${width}px in the ${theme} theme`, async () => {
      await page.setViewportSize({ width, height: 900 });
      for (const path of paths) {
        await load(path, { theme });
        assert.deepEqual(await overflow(), [], path);
      }
    });
  }

  it("shows who, where and how a browser Session signed in above the details", async () => {
    await load("/core/sessions/alice-browser-k3x9");
    await page.getByText("Chrome 130 on macOS", { exact: true }).waitFor();
    await page.getByText(/Signed in via github/).first().waitFor();
    await page
      .getByRole("progressbar", { name: "Session lifetime left" })
      .waitFor();
    await page
      .getByText("Authenticator authentication recommended.", { exact: true })
      .waitFor();
    for (const label of ["User", "Device", "Authentication", "Location"]) {
      await page.getByText(label, { exact: true }).first().waitFor();
    }
    assert.equal(
      await page.getByRole("link", { name: /^User\s*alice$/ }).first().getAttribute("href"),
      "/core/users/alice",
    );
    assert.deepEqual(await sectionTitles(), [
      "Authentication",
      "Client & location",
      "Authorization",
    ]);
    assert.equal(
      await page.getByText("Previous authentications", { exact: true }).count(),
      1,
    );
  });

  it("shows the connection and the served Services of a client Session", async () => {
    await load("/core/sessions/bob-client-q7m2");
    await page.getByText("Connected via WireGuard", { exact: true }).waitFor();
    await page.getByText("Proved by hardware key", { exact: true }).waitFor();
    assert.deepEqual(await sectionTitles(), [
      "Authentication",
      "Client & location",
      "Connection",
      "Served Services",
      "Authorization",
    ]);
    assert.equal(
      await page
        .getByRole("link", { name: /^Service\s*postgres-dev\.default$/ })
        .first()
        .getAttribute("href"),
      "/core/services/postgres-dev.default",
    );
    await page
      .getByText("q2v7m9c3Zr4P1aYh0uQe6Hn8Kx5Wb2Ts7Lf3Ji9Dg1E=", { exact: true })
      .waitFor();
  });

  it("shows how a Service is reached and where it proxies to", async () => {
    await load("/core/services/ssh-k8s-2.default");
    await page.getByText("SSH Service", { exact: true }).waitFor();
    assert.equal(await page.getByText(/Private SSH Service/).count(), 0);
    await page.getByText("ssh://10.0.12.4:22", { exact: true }).first().waitFor();
    await page
      .getByText("SSH access to the second Kubernetes control-plane node.", {
        exact: true,
      })
      .waitFor();
    assert.deepEqual(await sectionTitles(), [
      "Endpoints",
      "Upstream",
      "Authorization",
      "Attributes",
    ]);
  });

  it("links public Web Services from the header", async () => {
    await load("/core/services/grafana.monitoring");
    await page.getByText("Public Web Service", { exact: true }).waitFor();
    const open = toolbar("grafana.monitoring").getByRole("link", {
      name: "Open",
    });
    assert.match(await open.getAttribute("href"), /^https:\/\/grafana\.monitoring\./);
    assert.equal(await open.getAttribute("target"), "_blank");
  });

  for (const [path, name, label] of [
    ["/core/credentials/bob-auth-token-w8k2", "bob-auth-token-w8k2", "Rotate token"],
    ["/core/users/alice", "alice", "Create credential"],
    ["/core/devices/alice-macbook", "alice-macbook", "Reset binding"],
    ["/enterprise/certificates/cluster-wildcard", "cluster-wildcard", "Re-issue"],
    ["/enterprise/certificates/partner-api", "partner-api", "Set certificate"],
    ["/enterprise/secretstores/vault", "vault", "Synchronize now"],
    ["/enterprise/directoryproviders/google", "google", "Synchronize now"],
    ["/enterprise/directoryproviders/okta", "okta", "Generate token"],
    ["/access/integrations/slack-access", "slack-access", "Synchronize now"],
  ]) {
    it(`offers ${label} in the header of ${name} with the shared action style`, async () => {
      await load(path);
      const actions = toolbar(name);
      const action = actions.getByRole("button", { name: label, exact: true });
      await action.waitFor();
      const heights = await actions
        .locator("button, a")
        .evaluateAll((elements) =>
          elements.map((element) => Math.round(element.getBoundingClientRect().height)),
        );
      assert.ok(heights.length > 0);
      assert.equal(new Set(heights).size, 1, `${heights}`);
    });
  }

  it("only offers the issuance that matches the Certificate mode", async () => {
    await load("/enterprise/certificates/cluster-wildcard");
    assert.equal(
      await toolbar("cluster-wildcard")
        .getByRole("button", { name: "Set certificate" })
        .count(),
      0,
    );
    await load("/enterprise/certificates/partner-api");
    assert.equal(
      await toolbar("partner-api").getByRole("button", { name: /issue/i }).count(),
      0,
    );
    await page.getByText("Expiring soon.", { exact: true }).waitFor();
  });

  it("shows the health and the linking of a DeviceManager", async () => {
    await load("/enterprise/devicemanagers/crowdstrike");
    await page.getByText("1,180 of 1,240 managed devices are linked to a Device").waitFor();
    await page.getByText("7 issues").waitFor();
    assert.deepEqual(await sectionTitles(), [
      "Connection",
      "Inventory scope",
      "Collection",
      "Device linking",
    ]);
    await page.getByText("1 of 2").waitFor();
    assert.equal(
      await page.getByRole("link", { name: /Conflicts/ }).getAttribute("href"),
      "/core/devices?bindingState=CONFLICT&deviceManagerRef.name=crowdstrike",
    );
    assert.equal(
      await page.getByRole("link", { name: /^Linked/ }).getAttribute("href"),
      "/core/devices?bindingState=ACCEPTED&deviceManagerRef.name=crowdstrike",
    );
    assert.equal(await page.getByRole("link", { name: /Ambiguous/ }).count(), 1);
    await toolbar("crowdstrike").getByRole("button", { name: "View Devices" }).click();
    await page.waitForFunction(
      () => document.querySelector('[data-testid="location"]')?.textContent === "/core/devices",
    );
  });

  it("explains why a DeviceManager needs attention", async () => {
    await load("/enterprise/devicemanagers/jamf");
    await page.getByText("The last collection failed.").waitFor();
    await page.getByText("401 Unauthorized: the API client secret was rejected").waitFor();
    await page.getByText("The inventory is stale.").waitFor();
    await page.getByText("This manager is not selected for any Device.").waitFor();
    await page.getByText("Not listed", { exact: true }).waitFor();

    await load("/enterprise/devicemanagers/intune");
    await page.getByText("The last collection is incomplete.").waitFor();
    await page.getByText("compliance states could not be read").waitFor();

    await load("/enterprise/devicemanagers/fleet");
    await page.getByText("Polling is disabled.").waitFor();
    await page.getByText("Hosts without a team").waitFor();
  });

  it("links a Device to its DeviceManager", async () => {
    await load("/core/devices/alice-macbook");
    const link = page.getByRole("link", { name: /kandji/ }).first();
    assert.equal(await link.getAttribute("href"), "/enterprise/devicemanagers/kandji");
  });

  it("requests a SecretStore synchronization", async () => {
    await load("/enterprise/secretstores/vault");
    await toolbar("vault")
      .getByRole("button", { name: "Synchronize now" })
      .click();
    await page.waitForFunction(() =>
      window.overview.calls.some((call) => call.method === "synchronizeSecretStore"),
    );
    const [call] = await calls("synchronizeSecretStore");
    assert.equal(call.request.secretStoreRef.name, "vault");
  });

  it("resets a Device binding only after a confirmation", async () => {
    await load("/core/devices/alice-macbook");
    await toolbar("alice-macbook")
      .getByRole("button", { name: "Reset binding" })
      .click();
    const dialog = page
      .getByRole("dialog")
      .filter({ hasText: "Reset device binding" });
    await dialog.getByText("Reset device binding", { exact: true }).waitFor();
    assert.deepEqual(await calls("resetDeviceBinding"), []);
    await dialog.getByRole("button", { name: "Reset binding" }).click();
    await page.waitForFunction(() =>
      window.overview.calls.some((call) => call.method === "resetDeviceBinding"),
    );
    const [call] = await calls("resetDeviceBinding");
    assert.equal(call.request.deviceRef.name, "alice-macbook");
    assert.equal(call.request.bindingUID, "bnd-2c4e6a");
  });

  it("shows every reported posture check of a Device", async () => {
    await load("/core/devices/alice-macbook");
    await page.getByText("5 of 8 checks passing", { exact: true }).waitFor();
    for (const check of ["Disk encryption", "Firewall", "Secure boot", "OS integrity"]) {
      await page.getByText(check, { exact: true }).first().waitFor();
    }
    assert.deepEqual(await sectionTitles(), [
      "Posture",
      "Device manager",
      "Hardware",
      "Authorization",
    ]);
  });

  it("counts a grace period as not passing and shows the numeric posture", async () => {
    await load("/core/devices/alice-macbook");
    await page.getByText("Compliant (grace period)", { exact: true }).first().waitFor();
    for (const [label, value] of [
      ["Score", "85 / 100"],
      ["Failing checks", "0"],
      ["Active threats", "0"],
      ["Reported OS version", "14.5.1"],
      ["Ownership", "Company"],
      ["Device class", "Workstation"],
    ]) {
      await page.getByText(label, { exact: true }).first().waitFor();
      await page.getByText(value, { exact: true }).first().waitFor();
    }
  });

  it("shows the assurance, owner and evidence of a Device binding", async () => {
    await load("/core/devices/alice-macbook");
    await page.getByText("Assurance", { exact: true }).waitFor();
    for (const label of ["Provider ID", "Confirmed by provider", "Owner email", "Probe", "Serial number"]) {
      await page.getByText(label, { exact: true }).first().waitFor();
    }
    assert.equal(await page.getByText("Machine ID", { exact: true }).count(), 0);
    assert.equal(await page.getByText("Confidence", { exact: true }).count(), 0);
  });

  it("shows a vendor-signed Device binding and the proof of its Session", async () => {
    await load("/core/devices/carol-windows");
    for (const label of ["Signed", "Not reported", "Probe", "Serial number", "Vendor signed"]) {
      await page.getByText(label, { exact: true }).first().waitFor();
    }

    await load("/core/sessions/carol-client-z4n1");
    await page.getByText("Proved by vendor-signed token", { exact: true }).waitFor();
  });

  for (const [name, client] of [
    ["dave-client-g8v4", "Octelium client"],
    ["erin-client-h2c6", "Octelium client 0.22.1"],
  ]) {
    it(`reads the client version of ${name} from the octelium-cli user agent only`, async () => {
      await load(`/core/sessions/${name}`);
      await page.getByText(client, { exact: true }).first().waitFor();
      assert.equal(await page.getByText(/Octelium client 1\.84/).count(), 0);
    });
  }

  it("lists the Policy rules by priority", async () => {
    await load("/core/policies/allow-engineering");
    const rules = await page
      .locator("li code")
      .allTextContents();
    assert.deepEqual(rules.slice(0, 2), [
      "ctx.service.metadata.labels.env == 'production'",
      "ctx.user.spec.groups.exists(g, g == 'engineering')",
    ]);
  });

  it("explains that Secret values are write-only", async () => {
    for (const path of [
      "/core/secrets/db-password",
      "/enterprise/secrets/crt-cluster-wildcard",
    ]) {
      await load(path);
      await page.getByText("Write-only", { exact: true }).waitFor();
      assert.equal(
        await page
          .getByText("No kind-specific details are available for this resource.")
          .count(),
        0,
      );
    }
  });
});
