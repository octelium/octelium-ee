import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { join } from "node:path";
import { after, afterEach, before, beforeEach, describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../", import.meta.url));

describe("Service connect", () => {
  let cacheDir, server, browser, context, page, origin, errors;

  before(async () => {
    cacheDir = await mkdtemp(join(root, "node_modules/.vite-connect-"));
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
      `${origin}tests/fixtures/serviceConnect.html?${new URLSearchParams({ path, ...extra })}`,
    );
    await page.waitForFunction(() => window.connect);
  };
  const calls = () => page.evaluate(() => window.connect.calls);
  const asks = () => page.evaluate(() => window.connect.asks);
  const tab = (name) =>
    page.locator(".mantine-SegmentedControl-label").filter({ hasText: name });
  const terminalText = () =>
    page.evaluate(
      () =>
        document.querySelector("[data-testid=connect-terminal] .xterm-rows")
          ?.innerText ?? "",
    );
  const waitText = (text) =>
    page.waitForFunction(
      (text) =>
        document
          .querySelector("[data-testid=connect-terminal] .xterm-rows")
          ?.innerText.includes(text),
      text,
    );
  const setupScript = async () => {
    const { writes } = await calls();
    const wrapper = Object.values(writes).find((data) =>
      data.includes("| base64 -d >"),
    );
    return Buffer.from(/echo (\S+) \|/.exec(wrapper)[1], "base64").toString();
  };
  const mainWrites = async () => {
    const { writes, createTerminal } = await calls();
    return createTerminal
      .filter((term) => term.cols !== 200)
      .map((term) => writes[term.id] ?? "");
  };
  const connected = () =>
    page.getByText("Connected", { exact: true }).waitFor();
  const closeDrawer = async () => {
    await page.locator(".mantine-Drawer-close").click();
    await page.waitForFunction(
      () =>
        document.querySelector("[data-testid=location]")?.textContent ===
        "/core/services",
    );
  };

  it("shows the Connect tab for Services that a terminal can reach while the agent is available", async () => {
    await load("/core/services/ssh-h01");
    await tab("Connect").waitFor();

    await load("/core/services/desktop");
    await tab("Overview").waitFor();
    assert.equal(await tab("Connect").count(), 0);

    await load("/core/services/desktop/connect");
    await page.waitForFunction(
      () =>
        document.querySelector("[data-testid=location]")?.textContent ===
        "/core/services/desktop",
    );

    await load("/core/services/ssh-h01", { agent: "unavailable" });
    await tab("Overview").waitFor();
    assert.equal(await tab("Connect").count(), 0);
  });

  it("connects to an SSH Service from the agent Workspace automatically", async () => {
    await load("/core/services/ssh-h01/connect");
    await waitText("Welcome to Ubuntu 24.04.3 LTS");
    await connected();
    assert.equal(
      await page.getByTestId("connect-command").textContent(),
      "ssh ssh-h01",
    );

    const { createTerminal, removeTerminal, resizes } = await calls();
    assert.equal(createTerminal.length, 2);
    const [main, setup] = createTerminal;
    assert.equal(setup.cols, 200);
    assert.ok(main.cols > 40 && main.rows > 10);
    assert.deepEqual(removeTerminal, [setup.id]);
    assert.ok(resizes.some((resize) => resize.id === main.id));
    assert.deepEqual(await mainWrites(), ["ssh ssh-h01\r"]);

    const script = await setupScript();
    assert.match(script, /getent hosts 'ssh-h01'/);
    assert.match(script, /^want ssh ssh openssh-client /m);
    assert.match(script, /^m done$/m);
    assert.match(
      await terminalText(),
      /octelium@agent ~ % ssh ssh-h01\nWelcome to Ubuntu/,
    );
  });

  it("installs a missing CLI before it connects and shows the setup output", async () => {
    await load("/core/services/pg-main/connect", {
      missing: "psql",
      hold: "setup",
    });
    await page
      .locator('[data-step="psql"][data-status="running"]')
      .getByText("Installing psql")
      .waitFor();
    await page.getByText("Connecting", { exact: true }).waitFor();
    await page.getByRole("button", { name: "Show details" }).click();
    await page
      .getByLabel("Setup output")
      .getByText("Setting up psql ...")
      .waitFor();
    assert.deepEqual(await mainWrites(), [""]);
    assert.equal(
      await page
        .getByRole("button", { name: "\\dt", exact: true })
        .isDisabled(),
      true,
    );

    await page.evaluate(() => window.connect.release());
    await waitText('Type "help" for help.');
    await connected();
    assert.deepEqual(await mainWrites(), ["psql -h pg-main -U app -d appdb\r"]);
    await page.getByLabel("Setup output").waitFor({ state: "detached" });
  });

  it("types the command without running it when the CLI cannot be installed", async () => {
    await load("/core/services/pg-main/connect", { fail: "psql" });
    await page
      .getByRole("alert")
      .filter({ hasText: "psql could not be installed in the Workspace" })
      .filter({ hasText: "root access is not available" })
      .waitFor();
    await connected();
    assert.deepEqual(await mainWrites(), ["psql -h pg-main -U app -d appdb"]);
    await waitText("% psql -h pg-main -U app -d appdb");
    assert.doesNotMatch(await terminalText(), /Type "help"/);
  });

  it("prepares kubectl with the kubeconfig of the Service", async () => {
    await load("/core/services/k8s/connect");
    await waitText("% kubectl get pods -A");
    await connected();
    assert.deepEqual(await mainWrites(), [
      "export KUBECONFIG=/home/octelium/.kube/k8s.example.test\rkubectl get pods -A",
    ]);
    const script = await setupScript();
    assert.match(script, /^fetch kubectl kubectl$/m);
    assert.match(script, /octelium config 'k8s'/);
    assert.match(script, /sha256sum -c -/);
    await page
      .getByRole("button", { name: "get nodes", exact: true })
      .waitFor();
  });

  it("keeps the session across the tabs and removes it once the drawer is closed", async () => {
    await load("/core/services/ssh-h01/connect");
    await waitText("Welcome to Ubuntu");
    await connected();
    const [main, setup] = (await calls()).createTerminal;

    await tab("Overview").click();
    await page.waitForFunction(
      () =>
        document.querySelector("[data-testid=location]")?.textContent ===
        "/core/services/ssh-h01",
    );
    await page.waitForTimeout(300);
    assert.equal(await page.getByTestId("connect-terminal").count(), 0);

    await tab("Connect").click();
    await waitText("Welcome to Ubuntu");
    await connected();
    let current = await calls();
    assert.equal(current.createTerminal.length, 2);
    assert.deepEqual(current.removeTerminal, [setup.id]);

    await page.getByTestId("connect-terminal").click();
    await page.keyboard.type("uptime");
    await page.keyboard.press("Enter");
    await page.waitForFunction(
      (id) => window.connect.calls.writes[id]?.endsWith("uptime\r"),
      main.id,
    );

    await closeDrawer();
    await page.waitForFunction(
      (id) => window.connect.calls.removeTerminal.includes(id),
      main.id,
    );
    current = await calls();
    assert.equal(current.createTerminal.length, 2);
  });

  it("opens the exact terminal in Cordium and keeps it open there", async () => {
    await load("/core/services/ssh-h01/connect");
    await waitText("Welcome to Ubuntu");
    const [main, setup] = (await calls()).createTerminal;
    const link = page.getByRole("link", { name: "Open in Cordium" });
    assert.equal(
      await link.getAttribute("href"),
      `https://cordium.${new URL(origin).host}/workspaces/octelium-agent-x1/terminals?terminal=${main.id}`,
    );
    await context.route(/^https:\/\/cordium\./, (route) =>
      route.fulfill({ status: 200, body: "Cordium" }),
    );
    const [popup] = await Promise.all([
      context.waitForEvent("page"),
      link.click(),
    ]);
    await popup.close();
    await page.getByText("Kept open in Cordium").waitFor();

    await closeDrawer();
    await page.waitForTimeout(300);
    assert.deepEqual((await calls()).removeTerminal, [setup.id]);
  });

  it("starts the stopped agent Workspace before it connects", async () => {
    await load("/core/services/ssh-h01/connect", { ws: "stopped" });
    await page
      .locator('[data-step="workspace"][data-status="running"]')
      .getByText("Pulling the image")
      .waitFor();
    await waitText("Welcome to Ubuntu");
    assert.equal((await calls()).startWorkspace, 1);
  });

  it("sets up the agent environment on demand", async () => {
    await load("/core/services/ssh-h01/connect", { agent: "none" });
    await page.getByText("Set up your agent environment").waitFor();
    assert.equal((await calls()).createTerminal.length, 0);
    await page.getByRole("button", { name: "Set up", exact: true }).click();
    await waitText("Welcome to Ubuntu");
    const current = await calls();
    assert.equal(current.initializeAgent, 1);
    assert.equal(current.startWorkspace, 1);
  });

  it("explains when terminals are not available", async () => {
    await load("/core/services/ssh-h01/connect", { agent: "error" });
    await page.getByText("Terminals are not available").waitFor();
    await page.getByText("Cordium is not installed").waitFor();
    await page.getByRole("button", { name: "Retry" }).waitFor();
  });

  it("asks the agent about the session, the selection and a detected failure", async () => {
    await load("/core/services/ssh-denied/connect");
    const strip = page
      .getByRole("alert")
      .filter({ hasText: "The connection seems to have failed" });
    await strip.getByText("Permission denied (publickey).").waitFor();
    await page.getByText("Needs attention", { exact: true }).waitFor();

    await strip.getByRole("button", { name: "Ask agent" }).click();
    let [request] = await asks();
    assert.equal(
      request.text,
      "Why did this connection fail and how can I fix it?",
    );
    assert.equal(request.fresh, true);
    assert.deepEqual(request.pinned.resources, [
      {
        apiVersion: "core/v1",
        kind: "Service",
        name: "ssh-denied.default",
        uid: "uid-ssh-denied",
      },
    ]);
    const [item] = request.pinned.items;
    assert.equal(item.kind, "TerminalSession");
    assert.equal(item.data.command, "ssh ssh-denied");
    assert.equal(item.data.workspace, "octelium-agent-x1");
    assert.match(item.data.issue, /Permission denied \(publickey\)/);
    assert.match(item.data.output, /% ssh ssh-denied\n.*Permission denied/);

    await page.getByRole("button", { name: "Ask agent" }).first().click();
    request = (await asks())[1];
    assert.equal(request.text, undefined);
    assert.match(request.pinned.items[0].data.output, /Permission denied/);

    const row = page
      .locator("[data-testid=connect-terminal] .xterm-rows > div")
      .nth(1);
    const box = await row.boundingBox();
    await page.mouse.move(box.x + 2, box.y + box.height / 2);
    await page.mouse.down();
    await page.mouse.move(box.x + 300, box.y + box.height / 2, { steps: 8 });
    await page.mouse.up();
    await page.getByRole("button", { name: "Ask about the selection" }).click();
    request = (await asks())[2];
    assert.match(request.pinned.label, /^Terminal selection/);
    assert.match(
      request.pinned.items[0].data.selection,
      /^octelium@ssh-denied/,
    );
    assert.equal(request.pinned.items[0].data.output, undefined);

    await strip.getByRole("button", { name: "Dismiss" }).click();
    await strip.waitFor({ state: "detached" });
    await connected();
  });

  it("types the quick commands and starts new sessions", async () => {
    await load("/core/services/pg-main/connect");
    await waitText('Type "help" for help.');
    await connected();
    await page.getByRole("button", { name: "\\dt", exact: true }).click();
    await waitText("appdb=> \\dt");
    assert.deepEqual(await mainWrites(), [
      "psql -h pg-main -U app -d appdb\r\\dt",
    ]);

    const [first] = (await calls()).createTerminal;
    await page.getByRole("button", { name: "Start a new session" }).click();
    await page.waitForFunction(
      () => window.connect.calls.createTerminal.length === 4,
    );
    await page.waitForFunction(
      (id) => window.connect.calls.removeTerminal.includes(id),
      first.id,
    );
    await waitText('Type "help" for help.');
    await connected();
    assert.doesNotMatch(await terminalText(), /\\dt/);
  });

  it("ends the session when the shell exits and starts a new one", async () => {
    await load("/core/services/legacy/connect");
    await waitText("octelium@agent ~ %");
    await connected();
    assert.deepEqual(await mainWrites(), [""]);
    await page.getByTestId("connect-terminal").click();
    await page.keyboard.type("exit");
    await page.keyboard.press("Enter");
    await page.getByText("The session has ended").waitFor();
    await page.getByText("Ended", { exact: true }).waitFor();
    await page
      .getByRole("button", { name: "New session", exact: true })
      .click();
    await page.waitForFunction(
      () => window.connect.calls.createTerminal.length === 4,
    );
    await connected();
  });

  it("skips the setup to open the shell right away", async () => {
    await load("/core/services/pg-main/connect", {
      missing: "psql",
      hold: "setup",
    });
    await page.getByText("Installing psql").waitFor();
    await page.getByRole("button", { name: "Skip and open the shell" }).click();
    await connected();
    const { createTerminal, removeTerminal } = await calls();
    assert.deepEqual(removeTerminal, [createTerminal[1].id]);
    assert.deepEqual(await mainWrites(), [""]);
    await page.getByRole("button", { name: "\\dt", exact: true }).click();
    assert.deepEqual(await mainWrites(), ["\\dt"]);
  });

  it("adapts the command to embedded SSH, inferred TCP clients and disabled Services", async () => {
    await load("/core/services/essh/connect");
    await page.getByText("type the name of that Session").waitFor();
    await connected();
    assert.deepEqual(await mainWrites(), [
      "ssh @essh\x1b[D\x1b[D\x1b[D\x1b[D\x1b[D",
    ]);

    await load("/core/services/nats-main/connect");
    await page.getByText("nats is inferred from the Service name.").waitFor();
    await connected();
    assert.deepEqual(await mainWrites(), [
      "export NATS_URL=nats://nats-main:4222\rnats rtt",
    ]);
    assert.match(await setupScript(), /^fetch nats nats$/m);

    await load("/core/services/cache/connect");
    await page
      .getByText("redis-cli is inferred from the Service upstream image.")
      .waitFor();
    await waitText("cache:6379>");
    assert.deepEqual(await mainWrites(), ["redis-cli -h cache\r"]);

    await load("/core/services/legacy/connect");
    await page
      .getByText(
        "No specific client is known for this Service. Any client in this terminal reaches it at legacy:7000.",
      )
      .waitFor();
    await connected();
    assert.deepEqual(await mainWrites(), [""]);

    await load("/core/services/web/connect");
    await page.getByText("This Service is disabled").waitFor();
    await connected();
    assert.deepEqual(await mainWrites(), ["curl -i http://web:8080/"]);
  });

  it("builds the client command of every Service mode", async () => {
    await load("/core/services");
    const plan = (name, facts) =>
      page.evaluate(
        ([name, facts]) => window.connect.plan(name, facts),
        [name, facts],
      );
    const commands = async (name, facts) => (await plan(name, facts)).commands;

    assert.deepEqual(await commands("grpc-api"), [
      { text: "grpcurl -plaintext grpc-api:8080 list" },
    ]);
    assert.deepEqual(await commands("grpc-tls"), [
      { text: "grpcurl grpc-tls.local.example.test:443 list" },
    ]);
    assert.deepEqual(await commands("api"), [
      { text: "curl -i https://api.local.example.test/" },
    ]);
    assert.deepEqual(await commands("site"), [
      { text: "curl -sI http://site/" },
    ]);
    assert.deepEqual(await commands("openai"), [
      {
        text: `curl -s http://openai/v1/chat/completions -H 'content-type: application/json' -d '{"model":"gpt-5-mini","messages":[{"role":"user","content":"Hello!"}]}'`,
      },
    ]);
    assert.deepEqual(await commands("claude"), [
      {
        text: "curl -s http://claude/v1/models -H 'anthropic-version: 2023-06-01'",
      },
    ]);
    const [mcp] = await commands("mcp-github");
    assert.match(
      mcp.text,
      /^curl -s http:\/\/mcp-github\/mcp -H 'content-type: application\/json' -H 'accept: application\/json, text\/event-stream' -d '\{"jsonrpc":"2\.0","id":1,"method":"initialize",/,
    );
    assert.deepEqual(await commands("resolver"), [
      { text: "dig @resolver -p 5353 example.com" },
    ]);
    assert.deepEqual(await commands("egress"), [
      { text: "curl --socks5-hostname egress:1080 https://example.com -I" },
    ]);
    assert.deepEqual(await commands("mysql-db"), [
      { text: "mysql -h mysql-db -P 3307 -u app -D shop", run: true },
    ]);
    assert.deepEqual(
      await commands("mysql-db", {
        mysql: "mariadb",
        "mysql.flavor": "mariadb",
      }),
      [
        {
          text: "mariadb -h mysql-db -P 3307 -u app -D shop --skip-ssl-verify-server-cert",
          run: true,
        },
      ],
    );
    assert.deepEqual(
      await commands("k8s", {
        kubectl: "/home/octelium/.local/bin/kubectl",
        kubeconfig: "/home/octelium/.kube/k8s.example.test",
      }),
      [
        {
          text: "export KUBECONFIG=/home/octelium/.kube/k8s.example.test",
          run: true,
        },
        { text: "/home/octelium/.local/bin/kubectl get pods -A" },
      ],
    );

    const mongo = await plan("mongo");
    assert.equal(mongo.tool, undefined);
    assert.equal(mongo.inferred, "MongoDB is inferred from the Service name.");
    assert.match(mongo.hint, /mongosh is not installed automatically/);
    assert.deepEqual(mongo.commands, []);
    assert.ok(mongo.snippets.includes("mongosh mongodb://mongo:27017"));

    const events = await plan("events");
    assert.equal(
      events.inferred,
      "kcat is inferred from the Service display name.",
    );
    assert.deepEqual(events.commands, [{ text: "kcat -b events:9092 -L" }]);

    const db = await plan("db");
    assert.equal(db.inferred, "psql is inferred from the Service port.");
    assert.match(db.hint, /do not inject upstream credentials/);
    assert.deepEqual(db.commands, [{ text: "psql -h db -U " }]);

    const syslog = await plan("syslog");
    assert.equal(syslog.label, "Terminal");
    assert.match(syslog.hint, /^This is a UDP Service/);
    assert.deepEqual(syslog.commands, []);
  });

  it("fits the drawer in both themes and at phone width", async () => {
    for (const theme of ["light", "dark"]) {
      for (const width of [1440, 390]) {
        await page.setViewportSize({ width, height: 860 });
        await load("/core/services/pg-main/connect", { theme });
        await waitText('Type "help" for help.');
        await connected();
        const layout = await page.evaluate(() => {
          const body = document.querySelector(
            "#resource-item-drawer .mantine-Drawer-body",
          );
          const terminal = document.querySelector(
            "[data-testid=connect-terminal]",
          );
          const card = terminal.closest("[data-agent-section]");
          return {
            overflow: body.scrollWidth - body.clientWidth,
            cardBottom: card.getBoundingClientRect().bottom,
            terminalHeight: terminal.getBoundingClientRect().height,
            background: getComputedStyle(terminal.parentElement)
              .backgroundColor,
          };
        });
        assert.ok(
          layout.overflow <= 0,
          `${theme} ${width}: ${layout.overflow}`,
        );
        assert.ok(layout.cardBottom <= 860, `${theme} ${width}`);
        assert.ok(layout.terminalHeight > 200, `${theme} ${width}`);
        assert.equal(layout.background, "rgb(15, 23, 42)");
      }
    }
  });
});
