import { strict as assert } from "node:assert";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { after, before, describe, it } from "node:test";
import {
  fauxAssistantMessage,
  fauxText,
  fauxThinking,
  fauxToolCall,
} from "@earendil-works/pi-ai";
import type {
  AgentEvent,
  ApprovalBlock,
  ArtifactBlock,
  ConversationDetail,
  CreateConversationResponse,
  FileInfo,
  Message,
  Run,
  TableBlock,
  ToolBlock,
} from "../protocol/index.ts";
import {
  collectEvents,
  createTestApp,
  postJSON,
  type TestApp,
} from "../testutil/app.ts";

const services = {
  apiVersion: "core/v1",
  kind: "ServiceList",
  items: [
    {
      apiVersion: "core/v1",
      kind: "Service",
      metadata: { name: "nginx.default", uid: "s1" },
      spec: { port: 80, mode: "HTTP" },
    },
    {
      apiVersion: "core/v1",
      kind: "Service",
      metadata: { name: "pg.prod", uid: "s2" },
      spec: { port: 5432, mode: "POSTGRES" },
    },
  ],
  listResponseMeta: { totalCount: 2 },
};

const startConversation = async (
  t: TestApp,
  text: string,
): Promise<CreateConversationResponse> => {
  const res = await postJSON(`${t.baseUrl}/v1/conversations`, {
    input: { text },
  });
  assert.equal(res.status, 201);
  return (await res.json()) as CreateConversationResponse;
};

const lastMessage = (events: AgentEvent[]): Message => {
  const completed = events.filter((e) => e.type === "message.completed");
  const event = completed.at(-1);
  assert.ok(event && event.type === "message.completed");
  return event.message;
};

describe("server: runs", () => {
  let t: TestApp;

  before(async () => {
    t = await createTestApp({
      handlers: {
        "core.v1.MainService/ListService": () => services,
      },
    });
  });

  after(async () => {
    await t.close();
  });

  it("streams a full run and persists the messages", async () => {
    t.faux.setResponses([
      fauxAssistantMessage(
        [
          fauxThinking("I should list the Services"),
          fauxText("Let me check."),
          fauxToolCall("octelium_api_call", {
            method: "core.v1.MainService/ListService",
            request: { common: { itemsPerPage: 10 } },
          }),
        ],
        { stopReason: "toolUse" },
      ),
      fauxAssistantMessage(
        [
          fauxToolCall("present_table", {
            title: "Services",
            rows: [
              { name: "nginx.default", port: 80 },
              { name: "pg.prod", port: 5432 },
            ],
          }),
          fauxToolCall("present_resources", {
            resources: [
              { apiVersion: "core/v1", kind: "Service", name: "nginx.default" },
            ],
          }),
        ],
        { stopReason: "toolUse" },
      ),
      fauxAssistantMessage([fauxText("You have **2** Services.")]),
    ]);

    const created = await startConversation(t, "List my Services");
    assert.ok(created.run);
    assert.equal(created.conversation.title, "List my Services");

    const events = await collectEvents(t.baseUrl, created.run.id);
    const types = events.map((e) => e.type);
    assert.equal(types[0], "run.started");
    assert.equal(types.at(-1), "run.completed");
    assert.ok(types.includes("block.delta"));
    assert.deepEqual(
      events.map((e) => e.seq),
      events.map((_, i) => i + 1),
    );

    const message = lastMessage(events);
    assert.equal(message.status, "completed");
    assert.deepEqual(
      message.blocks.map((b) => b.type),
      ["thinking", "markdown", "tool", "table", "resources", "markdown"],
    );

    const tool = message.blocks[2] as ToolBlock;
    assert.equal(tool.status, "completed");
    assert.equal(tool.title, "Call core.v1.MainService/ListService");
    assert.ok(tool.details?.kind === "octelium_api");
    assert.equal(tool.details.risk, "read");
    assert.deepEqual(tool.details.resources, [
      {
        apiVersion: "core/v1",
        kind: "Service",
        name: "nginx.default",
        uid: "s1",
      },
      { apiVersion: "core/v1", kind: "Service", name: "pg.prod", uid: "s2" },
    ]);

    const table = message.blocks[3] as TableBlock;
    assert.equal(table.totalRows, 2);
    assert.deepEqual(
      table.columns.map((c) => [c.key, c.type]),
      [
        ["name", "string"],
        ["port", "number"],
      ],
    );

    const resources = message.blocks[4];
    assert.ok(resources.type === "resources");
    assert.equal(
      resources.resources[0].uri,
      "octelium://resource/core/v1/Service/nginx.default",
    );
    assert.equal(
      (resources.resources[0].snapshot?.metadata as { uid?: string }).uid,
      "s1",
    );

    const markdown = message.blocks[5];
    assert.ok(markdown.type === "markdown");
    assert.equal(markdown.text, "You have **2** Services.");

    const call = t.mock.calls.find((c) => c.method.endsWith("/ListService"));
    assert.deepEqual(call?.request, { common: { itemsPerPage: 10 } });

    const detail = (await (
      await fetch(`${t.baseUrl}/v1/conversations/${created.conversation.id}`)
    ).json()) as ConversationDetail;
    assert.equal(detail.messages.length, 2);
    assert.equal(detail.messages[0].role, "user");
    assert.equal(detail.messages[1].id, message.id);
    assert.equal(detail.activeRun, undefined);
    assert.equal(detail.conversation.activeRunId, undefined);

    const replay = await fetch(
      `${t.baseUrl}/v1/runs/${created.run.id}/events?after=${events.at(-1)!.seq}`,
    );
    assert.equal(replay.status, 204);

    const partial = await collectEvents(
      t.baseUrl,
      created.run.id,
      undefined,
      3,
    );
    assert.equal(partial[0].seq, 4);
    assert.equal(partial.at(-1)?.type, "run.completed");
  });

  it("continues a conversation with the previous context", async () => {
    t.faux.setResponses([fauxAssistantMessage([fauxText("First answer")])]);
    const created = await startConversation(t, "Hello");
    await collectEvents(t.baseUrl, created.run!.id);

    let seenUserMessages = 0;
    t.faux.setResponses([
      (context) => {
        seenUserMessages = context.messages.filter(
          (m) => m.role === "user",
        ).length;
        return fauxAssistantMessage([fauxText("Second answer")]);
      },
    ]);
    const res = await postJSON(
      `${t.baseUrl}/v1/conversations/${created.conversation.id}/runs`,
      { input: { text: "And then?" } },
    );
    assert.equal(res.status, 201);
    const run = (await res.json()) as Run;
    const events = await collectEvents(t.baseUrl, run.id);
    assert.equal(lastMessage(events).status, "completed");
    assert.equal(seenUserMessages, 2);

    const detail = (await (
      await fetch(`${t.baseUrl}/v1/conversations/${created.conversation.id}`)
    ).json()) as ConversationDetail;
    assert.equal(detail.messages.length, 4);
  });

  it("reports invalid API requests back to the model", async () => {
    let toolResult = "";
    t.faux.setResponses([
      fauxAssistantMessage(
        [
          fauxToolCall("octelium_api_call", {
            method: "core.v1.MainService/ListService",
            request: { unknownField: 1 },
          }),
        ],
        { stopReason: "toolUse" },
      ),
      (context) => {
        const last = context.messages.at(-1);
        toolResult =
          last?.role === "toolResult"
            ? last.content
                .map((c) => (c.type === "text" ? c.text : ""))
                .join("")
            : "";
        return fauxAssistantMessage([fauxText("Fixed")]);
      },
    ]);
    const created = await startConversation(t, "Break it");
    const events = await collectEvents(t.baseUrl, created.run!.id);
    const tool = lastMessage(events).blocks.find(
      (b) => b.type === "tool",
    ) as ToolBlock;
    assert.equal(tool.status, "failed");
    assert.match(toolResult, /Invalid request/);
    assert.match(toolResult, /unknownField/);
  });

  it("rejects a concurrent run in the same conversation", async () => {
    t.faux.setResponses([
      fauxAssistantMessage(
        [
          fauxToolCall("octelium_api_call", {
            method: "core.v1.MainService/DeleteService",
            request: { name: "nginx.default" },
          }),
        ],
        { stopReason: "toolUse" },
      ),
      fauxAssistantMessage([fauxText("done")]),
    ]);
    const created = await startConversation(t, "Delete nginx");
    const res = await postJSON(
      `${t.baseUrl}/v1/conversations/${created.conversation.id}/runs`,
      { input: { text: "again" } },
    );
    assert.equal(res.status, 409);
    await postJSON(`${t.baseUrl}/v1/runs/${created.run!.id}/cancel`, {});
    await collectEvents(t.baseUrl, created.run!.id);
  });

  it("validates the run input", async () => {
    const res = await postJSON(`${t.baseUrl}/v1/conversations`, {
      input: { text: "   " },
    });
    assert.equal(res.status, 400);
    const list = (await (
      await fetch(`${t.baseUrl}/v1/conversations`)
    ).json()) as {
      items: unknown[];
    };
    assert.ok(list.items.every((c) => (c as { title: string }).title !== ""));

    const missing = await postJSON(
      `${t.baseUrl}/v1/conversations/unknown-id/runs`,
      { input: { text: "hi" } },
    );
    assert.equal(missing.status, 404);

    const badAttachment = await postJSON(`${t.baseUrl}/v1/conversations`, {
      input: { text: "hi", attachments: ["nope"] },
    });
    assert.equal(badAttachment.status, 400);
  });
});

describe("server: approvals", () => {
  let t: TestApp;
  let deleted: string[] = [];

  before(async () => {
    t = await createTestApp({
      handlers: {
        "core.v1.MainService/DeleteService": (req) => {
          deleted.push((req as { name: string }).name);
          return {};
        },
      },
    });
  });

  after(async () => {
    await t.close();
  });

  const deleteResponses = () => [
    fauxAssistantMessage(
      [
        fauxToolCall("octelium_api_call", {
          method: "core.v1.MainService/DeleteService",
          request: { name: "nginx.default" },
        }),
      ],
      { stopReason: "toolUse" },
    ),
    fauxAssistantMessage([fauxText("Done.")]),
  ];

  it("executes a write only after it is approved", async () => {
    deleted = [];
    t.faux.setResponses(deleteResponses());
    const created = await startConversation(t, "Delete nginx");
    const statuses: string[] = [];

    const events = await collectEvents(
      t.baseUrl,
      created.run!.id,
      async (e) => {
        if (e.type === "run.updated") {
          statuses.push(e.run.status);
        }
        if (e.type === "block.created" && e.block.type === "approval") {
          assert.deepEqual(deleted, []);
          assert.equal(e.block.risk, "destructive");
          assert.deepEqual(e.block.preview, {
            method: "core.v1.MainService/DeleteService",
            request: { name: "nginx.default" },
          });
          const res = await postJSON(
            `${t.baseUrl}/v1/runs/${created.run!.id}/approvals/${e.block.approvalId}`,
            { decision: "approve" },
          );
          assert.equal(res.status, 200);
          assert.equal(
            ((await res.json()) as ApprovalBlock).status,
            "approved",
          );
        }
      },
    );

    assert.deepEqual(deleted, ["nginx.default"]);
    assert.deepEqual(statuses, ["awaiting_approval", "running"]);
    const message = lastMessage(events);
    const approval = message.blocks.find(
      (b) => b.type === "approval",
    ) as ApprovalBlock;
    assert.equal(approval.status, "approved");
    const tool = message.blocks.find((b) => b.type === "tool") as ToolBlock;
    assert.equal(tool.status, "completed");
  });

  it("does not execute a rejected write", async () => {
    deleted = [];
    t.faux.setResponses(deleteResponses());
    const created = await startConversation(t, "Delete nginx");

    const events = await collectEvents(
      t.baseUrl,
      created.run!.id,
      async (e) => {
        if (e.type === "block.created" && e.block.type === "approval") {
          await postJSON(
            `${t.baseUrl}/v1/runs/${created.run!.id}/approvals/${e.block.approvalId}`,
            { decision: "reject", reason: "not now" },
          );
        }
      },
    );

    assert.deepEqual(deleted, []);
    const message = lastMessage(events);
    const approval = message.blocks.find(
      (b) => b.type === "approval",
    ) as ApprovalBlock;
    assert.equal(approval.status, "rejected");
    assert.equal(approval.reason, "not now");
    const tool = message.blocks.find((b) => b.type === "tool") as ToolBlock;
    assert.equal(tool.status, "rejected");
    assert.equal(message.status, "completed");
  });

  it("cancels a run that awaits an approval", async () => {
    deleted = [];
    t.faux.setResponses(deleteResponses());
    const created = await startConversation(t, "Delete nginx");

    const events = await collectEvents(
      t.baseUrl,
      created.run!.id,
      async (e) => {
        if (e.type === "block.created" && e.block.type === "approval") {
          const res = await postJSON(
            `${t.baseUrl}/v1/runs/${created.run!.id}/cancel`,
            {},
          );
          assert.equal(res.status, 200);
        }
      },
    );

    assert.deepEqual(deleted, []);
    assert.equal(events.at(-1)?.type, "run.cancelled");
    const message = lastMessage(events);
    assert.equal(message.status, "cancelled");
    const approval = message.blocks.find(
      (b) => b.type === "approval",
    ) as ApprovalBlock;
    assert.equal(approval.status, "cancelled");

    const decide = await postJSON(
      `${t.baseUrl}/v1/runs/${created.run!.id}/approvals/${approval.approvalId}`,
      { decision: "approve" },
    );
    assert.equal(decide.status, 404);
  });
});

describe("server: files and artifacts", () => {
  let t: TestApp;

  before(async () => {
    t = await createTestApp();
  });

  after(async () => {
    await t.close();
  });

  it("uploads a file, attaches it and publishes an artifact", async () => {
    const upload = await fetch(`${t.baseUrl}/v1/files?name=data.csv`, {
      method: "POST",
      headers: { "content-type": "text/csv" },
      body: "name,value\na,1\nb,2\n",
    });
    assert.equal(upload.status, 201);
    const file = (await upload.json()) as FileInfo;
    assert.equal(file.name, "data.csv");
    assert.equal(file.mimeType, "text/csv");
    assert.equal(file.size, 19);
    assert.ok(fs.existsSync(file.path));

    const reportPath = path.join(t.app.config.workDir, "report.html");
    let prompt = "";
    t.faux.setResponses([
      (context) => {
        const user = context.messages.findLast((m) => m.role === "user");
        prompt =
          typeof user?.content === "string"
            ? user.content
            : (user?.content ?? [])
                .map((c) => (c.type === "text" ? c.text : ""))
                .join("");
        fs.writeFileSync(reportPath, "<html><script>alert(1)</script></html>");
        return fauxAssistantMessage(
          [
            fauxToolCall("present_chart", {
              chartType: "bar",
              title: "Values",
              x: { key: "name", type: "category" },
              series: [{ key: "value" }],
              dataPath: file.path,
            }),
            fauxToolCall("publish_artifact", {
              path: "report.html",
              title: "Report",
            }),
          ],
          { stopReason: "toolUse" },
        );
      },
      fauxAssistantMessage([fauxText("Here is your report.")]),
    ]);

    const res = await postJSON(`${t.baseUrl}/v1/conversations`, {
      input: { text: "Analyze this", attachments: [file.id] },
    });
    const created = (await res.json()) as CreateConversationResponse;
    const events = await collectEvents(t.baseUrl, created.run!.id);
    assert.match(prompt, /Analyze this/);
    assert.ok(prompt.includes(file.path));

    const created1 = events.find(
      (e) => e.type === "message.created" && e.message.role === "user",
    );
    assert.ok(created1?.type === "message.created");
    assert.equal(created1.message.attachments?.[0].id, file.id);

    const message = lastMessage(events);
    const chart = message.blocks.find((b) => b.type === "chart");
    assert.ok(chart?.type === "chart");
    assert.deepEqual(chart.chart.rows, [
      { name: "a", value: 1 },
      { name: "b", value: 2 },
    ]);

    const artifactBlock = message.blocks.find(
      (b) => b.type === "artifact",
    ) as ArtifactBlock;
    assert.equal(artifactBlock.artifact.name, "report.html");
    assert.equal(artifactBlock.artifact.mimeType, "text/html");

    const meta = await fetch(
      `${t.baseUrl}/v1/artifacts/${artifactBlock.artifact.id}`,
    );
    assert.equal(meta.status, 200);

    const content = await fetch(`${t.baseUrl}${artifactBlock.artifact.url}`);
    assert.equal(content.status, 200);
    assert.match(
      content.headers.get("content-disposition") ?? "",
      /^attachment; filename="report.html"/,
    );
    assert.match(
      content.headers.get("content-security-policy") ?? "",
      /sandbox/,
    );
    assert.equal(content.headers.get("x-content-type-options"), "nosniff");
    assert.match(await content.text(), /<script>/);

    const inline = await fetch(
      `${t.baseUrl}${artifactBlock.artifact.url}?inline=1`,
    );
    assert.match(
      inline.headers.get("content-disposition") ?? "",
      /^attachment/,
    );
  });

  it("rejects uploads larger than the limit", async () => {
    const res = await fetch(`${t.baseUrl}/v1/files?name=big.bin`, {
      method: "POST",
      body: Buffer.alloc(t.app.config.server.maxUploadBytes + 1),
    });
    assert.equal(res.status, 413);
  });

  it("returns 404 for unknown artifacts and invalid IDs", async () => {
    assert.equal((await fetch(`${t.baseUrl}/v1/artifacts/nope`)).status, 404);
    assert.equal(
      (await fetch(`${t.baseUrl}/v1/artifacts/..%2F..%2Fetc/content`)).status,
      404,
    );
  });
});

describe("server: security", () => {
  let t: TestApp;

  before(async () => {
    t = await createTestApp({
      config: { server: { authToken: "secret-token", cors: true } },
    });
  });

  after(async () => {
    await t.close();
  });

  it("requires the auth token for the API", async () => {
    assert.equal((await fetch(`${t.baseUrl}/v1/conversations`)).status, 401);
    assert.equal(
      (
        await fetch(`${t.baseUrl}/v1/conversations`, {
          headers: { authorization: "Bearer wrong" },
        })
      ).status,
      401,
    );
    assert.equal(
      (
        await fetch(`${t.baseUrl}/v1/conversations`, {
          headers: { authorization: "Bearer secret-token" },
        })
      ).status,
      200,
    );
    assert.equal(
      (await fetch(`${t.baseUrl}/v1/conversations?access_token=secret-token`))
        .status,
      200,
    );
    assert.equal((await fetch(`${t.baseUrl}/healthz`)).status, 200);
  });

  it("checks the request origin", async () => {
    const headers = { authorization: "Bearer secret-token" };
    const allowed = await fetch(`${t.baseUrl}/v1/conversations`, {
      headers: { ...headers, origin: "https://console.octelium.example.com" },
    });
    assert.equal(allowed.status, 200);
    assert.equal(
      allowed.headers.get("access-control-allow-origin"),
      "https://console.octelium.example.com",
    );
    assert.equal(
      allowed.headers.get("access-control-allow-credentials"),
      "true",
    );

    const denied = await fetch(`${t.baseUrl}/v1/conversations`, {
      headers: { ...headers, origin: "https://evil.com" },
    });
    assert.equal(denied.status, 403);

    const sibling = await fetch(`${t.baseUrl}/v1/conversations`, {
      headers: { ...headers, origin: "https://other.cordium.example.com" },
    });
    assert.equal(sibling.status, 403);

    const insecure = await fetch(`${t.baseUrl}/v1/conversations`, {
      headers: { ...headers, origin: "http://console.octelium.example.com" },
    });
    assert.equal(insecure.status, 403);

    const preflight = await fetch(`${t.baseUrl}/v1/conversations`, {
      method: "OPTIONS",
      headers: {
        origin: "https://console.octelium.example.com",
        "access-control-request-method": "POST",
      },
    });
    assert.equal(preflight.status, 204);
    assert.match(
      preflight.headers.get("access-control-allow-methods") ?? "",
      /POST/,
    );
  });

  it("requires JSON bodies", async () => {
    const res = await fetch(`${t.baseUrl}/v1/conversations`, {
      method: "POST",
      headers: {
        authorization: "Bearer secret-token",
        "content-type": "text/plain",
      },
      body: JSON.stringify({ title: "x" }),
    });
    assert.equal(res.status, 400);
  });
});

describe("server: conversations", () => {
  let t: TestApp;

  before(async () => {
    t = await createTestApp();
  });

  after(async () => {
    await t.close();
  });

  it("creates, renames, lists and deletes conversations", async () => {
    const res = await postJSON(`${t.baseUrl}/v1/conversations`, {
      title: "Audit",
    });
    assert.equal(res.status, 201);
    const { conversation } = (await res.json()) as CreateConversationResponse;
    assert.equal(conversation.title, "Audit");

    const patched = await fetch(
      `${t.baseUrl}/v1/conversations/${conversation.id}`,
      {
        method: "PATCH",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ title: "Security audit" }),
      },
    );
    assert.equal(patched.status, 200);

    const list = (await (
      await fetch(`${t.baseUrl}/v1/conversations`)
    ).json()) as {
      items: { id: string; title: string }[];
    };
    assert.equal(
      list.items.find((c) => c.id === conversation.id)?.title,
      "Security audit",
    );

    const del = await fetch(
      `${t.baseUrl}/v1/conversations/${conversation.id}`,
      {
        method: "DELETE",
      },
    );
    assert.equal(del.status, 204);
    assert.equal(
      (await fetch(`${t.baseUrl}/v1/conversations/${conversation.id}`)).status,
      404,
    );
  });

  it("returns the agent info", async () => {
    const info = (await (await fetch(`${t.baseUrl}/v1/info`)).json()) as {
      status: string;
      octelium: { mode: string; user?: { name?: string } };
      model?: { id: string };
    };
    assert.equal(info.status, "ready");
    assert.equal(info.octelium.mode, "proxy");
    assert.equal(info.octelium.user?.name, "alice");
    assert.equal(info.model?.id, "faux-1");
  });

  it("returns 404 and 405 for unknown routes", async () => {
    assert.equal((await fetch(`${t.baseUrl}/v1/nope`)).status, 404);
    assert.equal(
      (await fetch(`${t.baseUrl}/v1/conversations`, { method: "PUT" })).status,
      405,
    );
  });
});

describe("server: shell approvals and titles", () => {
  let t: TestApp;

  before(async () => {
    t = await createTestApp({
      config: { approvals: { bash: true }, agent: { generateTitles: true } },
    });
  });

  after(async () => {
    await t.close();
  });

  it("asks for an approval before running a shell command and generates a title", async () => {
    const marker = path.join(t.app.config.workDir, "marker.txt");
    t.faux.setResponses(
      Array.from({ length: 3 }, () => (context) => {
        const system = context.messages.find((m) => m.role === "system");
        const systemText =
          typeof system?.content === "string"
            ? system.content
            : (system?.content ?? []).map((c) => c.text).join("");
        if (systemText.startsWith("You generate short titles")) {
          return fauxAssistantMessage([fauxText('"Create a marker file."')]);
        }
        if (context.messages.at(-1)?.role === "toolResult") {
          return fauxAssistantMessage([fauxText("Created.")]);
        }
        return fauxAssistantMessage(
          [fauxToolCall("bash", { command: `echo ok > ${marker}` })],
          { stopReason: "toolUse" },
        );
      }),
    );

    const created = await startConversation(t, "Please create a marker file");
    const titles: string[] = [];
    const events = await collectEvents(
      t.baseUrl,
      created.run!.id,
      async (e) => {
        if (e.type === "conversation.updated") {
          titles.push(e.conversation.title);
        }
        if (e.type === "block.created" && e.block.type === "approval") {
          assert.equal(fs.existsSync(marker), false);
          assert.deepEqual(e.block.preview, { command: `echo ok > ${marker}` });
          await postJSON(
            `${t.baseUrl}/v1/runs/${created.run!.id}/approvals/${e.block.approvalId}`,
            { decision: "approve" },
          );
        }
      },
    );

    assert.equal(fs.readFileSync(marker, "utf8"), "ok\n");
    const tool = lastMessage(events).blocks.find(
      (b) => b.type === "tool",
    ) as ToolBlock;
    assert.equal(tool.status, "completed");
    assert.ok(tool.details?.kind === "command");
    assert.equal(tool.details.exitCode, 0);

    assert.equal(titles[0], "Please create a marker file");
    const conversation = t.app.store.get(created.conversation.id)!;
    assert.equal(conversation.title, "Create a marker file");
  });
});

describe("server: persistence", () => {
  it("continues a conversation after a restart", async () => {
    const root = fs.mkdtempSync(
      path.join(os.tmpdir(), "console-agent-restart-"),
    );
    try {
      const first = await createTestApp({
        root,
        responses: [fauxAssistantMessage([fauxText("The answer is 42.")])],
      });
      const created = await startConversation(first, "What is the answer?");
      await collectEvents(first.baseUrl, created.run!.id);
      await first.close();

      let seen: string[] = [];
      const second = await createTestApp({
        root,
        responses: [
          (context) => {
            seen = context.messages
              .filter((m) => m.role === "user" || m.role === "assistant")
              .map((m) =>
                typeof m.content === "string"
                  ? m.content
                  : m.content
                      .map((c) => (c.type === "text" ? c.text : ""))
                      .join(""),
              );
            return fauxAssistantMessage([fauxText("Still 42.")]);
          },
        ],
      });
      try {
        const detail = (await (
          await fetch(
            `${second.baseUrl}/v1/conversations/${created.conversation.id}`,
          )
        ).json()) as ConversationDetail;
        assert.equal(detail.messages.length, 2);

        const res = await postJSON(
          `${second.baseUrl}/v1/conversations/${created.conversation.id}/runs`,
          { input: { text: "Are you sure?" } },
        );
        const run = (await res.json()) as Run;
        const events = await collectEvents(second.baseUrl, run.id);
        assert.equal(lastMessage(events).status, "completed");
        assert.deepEqual(seen, [
          "What is the answer?",
          "The answer is 42.",
          "Are you sure?",
        ]);
      } finally {
        await second.close();
      }
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });
});
