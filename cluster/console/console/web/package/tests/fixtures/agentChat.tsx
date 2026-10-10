import "@mantine/core/styles.css";
import "@/index.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Route, Routes } from "react-router-dom";

import type {
  AgentEvent,
  AgentInfo,
  Block,
  Conversation,
  Message,
  Run,
} from "@/apis/consoleagent/protocol";
import Chat from "@/pages/Agent/Chat";
import { AgentClient } from "@/pages/Agent/client";
import themeMantine from "@/utils/theme/mantine";

interface Call {
  method: string;
  path: string;
  body?: unknown;
}

declare global {
  interface Window {
    agentCalls: Call[];
  }
}

const prefix = "/agent-api/v1";
const now = new Date();
const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
const at = (ms: number) => new Date(ms).toISOString();

const conversations = new Map<string, Conversation>();
const messages = new Map<string, Message[]>();
const decisions = new Map<string, (decision: string) => void>();
let counter = 0;

const markdown = (text: string, createdAt: string): Block => ({
  id: `b-${++counter}`,
  type: "markdown",
  text,
  createdAt,
});

const seed = (
  id: string,
  title: string,
  time: number,
  q: string,
  a: string,
) => {
  const createdAt = at(time);
  conversations.set(id, {
    id,
    title,
    createdAt,
    updatedAt: createdAt,
    messageCount: 2,
  });
  messages.set(id, [
    {
      id: `${id}-u`,
      conversationId: id,
      role: "user",
      status: "completed",
      createdAt,
      blocks: [markdown(q, createdAt)],
    },
    {
      id: `${id}-a`,
      conversationId: id,
      role: "assistant",
      status: "completed",
      createdAt,
      completedAt: createdAt,
      blocks: [markdown(a, createdAt)],
    },
  ]);
};

seed(
  "c1",
  "Audit admins group permissions",
  now.getTime() - 60000,
  "Which Policies apply to the Group admins?",
  "The Group admins has 3 Policies attached.",
);
seed(
  "c2",
  "Rotate the Postgres credential",
  today.getTime() - 12 * 3600000,
  "How do I rotate the credential of pg.prod?",
  "Create a new Secret and update the upstream.",
);
seed(
  "c3",
  "Certificate expirations",
  today.getTime() - 60 * 86400000,
  "Which certificates expire soon?",
  "No certificate expires in the next 30 days.",
);

const info: AgentInfo = {
  name: "octelium-console-agent",
  version: "0.1.0",
  protocolVersion: 1,
  status: "ready",
  issues: [],
  model: { provider: "faux", id: "faux-1", name: "Faux Opus" },
  octelium: {
    mode: "proxy",
    user: { name: "george", displayName: "George B." },
  },
  capabilities: {
    blockTypes: [],
    webSearch: false,
    approvals: true,
    uploads: { maxBytes: 1024 * 1024 },
    models: true,
    login: false,
    search: true,
  },
};

const json = (body: unknown, status = 200) =>
  new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });

const runs = new Map<string, { run: Run; text: string }>();

const startRun = (conversationId: string, text: string): Run => {
  const id = `r-${++counter}`;
  const run: Run = {
    id,
    conversationId,
    status: "running",
    createdAt: at(Date.now()),
    userMessageId: `${id}-u`,
    assistantMessageId: `${id}-a`,
    lastSeq: 0,
  };
  runs.set(id, { run, text });
  return run;
};

window.agentCalls = [];
const realFetch = window.fetch.bind(window);

window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
  const url = new URL(
    typeof input === "string" || input instanceof URL ? input : input.url,
    window.location.href,
  );
  if (!url.pathname.startsWith(prefix)) {
    return realFetch(input, init);
  }

  const path = url.pathname.slice(prefix.length);
  const method = init?.method ?? "GET";
  const body =
    typeof init?.body === "string" ? JSON.parse(init.body) : undefined;
  window.agentCalls.push({ method, path: `${path}${url.search}`, body });

  if (path === "/conversations" && method === "GET") {
    return json({
      items: [...conversations.values()].sort((a, b) =>
        b.updatedAt.localeCompare(a.updatedAt),
      ),
    });
  }

  if (path === "/conversations/search") {
    const q = (url.searchParams.get("q") ?? "").toLowerCase();
    return json({
      items: [...conversations.values()]
        .map((conversation) => ({
          conversation,
          titleMatch: conversation.title.toLowerCase().includes(q),
          matches: (messages.get(conversation.id) ?? [])
            .filter((m) =>
              m.blocks.some(
                (b) =>
                  b.type === "markdown" && b.text.toLowerCase().includes(q),
              ),
            )
            .map((m) => ({
              messageId: m.id,
              role: m.role,
              snippet: (m.blocks[0] as { text: string }).text,
            })),
        }))
        .filter((r) => r.titleMatch || r.matches.length > 0),
    });
  }

  if (path === "/conversations" && method === "POST") {
    const id = `c-${++counter}`;
    const conversation: Conversation = {
      id,
      title: body.input.text,
      createdAt: at(Date.now()),
      updatedAt: at(Date.now()),
      messageCount: 0,
    };
    conversations.set(id, conversation);
    messages.set(id, []);
    const run = startRun(id, body.input.text);
    conversation.activeRunId = run.id;
    return json({ conversation, run }, 201);
  }

  const conversationMatch = /^\/conversations\/([^/]+)$/.exec(path);
  if (conversationMatch) {
    const id = conversationMatch[1];
    const conversation = conversations.get(id);
    if (!conversation) return json({ error: { message: "Not found" } }, 404);
    if (method === "PATCH") {
      conversation.title = body.title;
      return json(conversation);
    }
    if (method === "DELETE") {
      conversations.delete(id);
      return json(undefined, 204);
    }
    return json({ conversation, messages: messages.get(id) ?? [] });
  }

  const runMatch = /^\/conversations\/([^/]+)\/runs$/.exec(path);
  if (runMatch) {
    return json(startRun(runMatch[1], body.input.text), 201);
  }

  const approvalMatch = /^\/runs\/([^/]+)\/approvals\/([^/]+)$/.exec(path);
  if (approvalMatch) {
    decisions.get(approvalMatch[2])?.(body.decision);
    return json(runs.get(approvalMatch[1])?.run);
  }

  if (path === "/models") {
    return json({ current: info.model, models: [info.model] });
  }

  if (path === "/info") {
    return json(info);
  }

  return json({ error: { message: "Not found" } }, 404);
};

const sleep = (ms: number) =>
  new Promise((resolve) => window.setTimeout(resolve, ms));

class FakeEventSource {
  static readonly CLOSED = 2;
  readyState = 1;
  onerror: (() => void) | null = null;
  private listeners = new Map<string, ((msg: { data: string }) => void)[]>();
  private seq = 0;

  constructor(url: string) {
    const runId = /\/runs\/([^/]+)\/events/.exec(url)?.[1] ?? "";
    window.setTimeout(() => void this.play(runId), 30);
  }

  addEventListener(type: string, fn: (msg: { data: string }) => void) {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), fn]);
  }

  close() {
    this.readyState = FakeEventSource.CLOSED;
  }

  private emit(payload: Record<string, unknown> & { type: string }, run: Run) {
    if (this.readyState === FakeEventSource.CLOSED) return;
    const event = {
      ...payload,
      seq: ++this.seq,
      ts: at(Date.now()),
      conversationId: run.conversationId,
      runId: run.id,
    } as AgentEvent;
    for (const fn of this.listeners.get(event.type) ?? []) {
      fn({ data: JSON.stringify(event) });
    }
  }

  private async play(runId: string) {
    const entry = runs.get(runId);
    if (!entry || this.readyState === FakeEventSource.CLOSED) return;
    const { run, text } = entry;
    const createdAt = at(Date.now());
    const user: Message = {
      id: run.userMessageId,
      conversationId: run.conversationId,
      runId: run.id,
      role: "user",
      status: "completed",
      createdAt,
      blocks: [markdown(text, createdAt)],
    };
    const assistant: Message = {
      id: run.assistantMessageId,
      conversationId: run.conversationId,
      runId: run.id,
      role: "assistant",
      status: "streaming",
      createdAt,
      blocks: [],
    };

    this.emit({ type: "run.started", run }, run);
    await sleep(150);
    this.emit({ type: "message.created", message: user }, run);
    this.emit({ type: "message.created", message: assistant }, run);

    if (text.toLowerCase().includes("delete")) {
      const tool: Block = {
        id: "tool-1",
        type: "tool",
        toolCallId: "call-1",
        name: "octelium_api_call",
        title: "Call core.v1.MainService/DeleteService",
        status: "awaiting_approval",
        createdAt,
        details: {
          kind: "octelium_api",
          method: "core.v1.MainService/DeleteService",
          grpcMethod: "/octelium.api.main.core.v1.MainService/DeleteService",
          risk: "destructive",
          request: { name: "nginx.default" },
        },
      };
      const approval: Block = {
        id: "approval-1",
        type: "approval",
        approvalId: "ap-1",
        toolCallId: "call-1",
        title: "Delete Service (core.v1)",
        risk: "destructive",
        status: "pending",
        createdAt,
        preview: {
          method: "core.v1.MainService/DeleteService",
          request: { name: "nginx.default" },
        },
      };
      this.emit(
        { type: "block.created", messageId: assistant.id, block: tool },
        run,
      );
      this.emit(
        { type: "block.created", messageId: assistant.id, block: approval },
        run,
      );
      this.emit(
        {
          type: "run.updated",
          run: { ...run, status: "awaiting_approval" },
        },
        run,
      );
      const decision = await new Promise<string>((resolve) =>
        decisions.set("ap-1", resolve),
      );
      const approved = decision === "approve";
      const resolved = {
        ...approval,
        status: approved ? "approved" : "rejected",
      } as Block;
      const done = {
        ...tool,
        status: approved ? "completed" : "rejected",
      } as Block;
      this.emit(
        { type: "block.updated", messageId: assistant.id, block: resolved },
        run,
      );
      this.emit(
        { type: "block.updated", messageId: assistant.id, block: done },
        run,
      );
      this.emit({ type: "run.updated", run }, run);
      assistant.blocks.push(done, resolved);
    }

    const answer = markdown("", at(Date.now()));
    this.emit(
      { type: "block.created", messageId: assistant.id, block: answer },
      run,
    );
    for (const chunk of ["You have ", "**2** ", "Services."]) {
      await sleep(120);
      this.emit(
        {
          type: "block.delta",
          messageId: assistant.id,
          blockId: answer.id,
          text: chunk,
        },
        run,
      );
    }

    const completed: Message = {
      ...assistant,
      status: "completed",
      completedAt: at(Date.now()),
      blocks: [
        ...assistant.blocks,
        { ...answer, text: "You have **2** Services." } as Block,
      ],
    };
    messages.set(run.conversationId, [
      ...(messages.get(run.conversationId) ?? []),
      user,
      completed,
    ]);
    const conversation = conversations.get(run.conversationId)!;
    delete conversation.activeRunId;
    conversation.updatedAt = completed.completedAt!;
    conversation.messageCount += 2;

    this.emit({ type: "message.completed", message: completed }, run);
    this.emit({ type: "conversation.updated", conversation }, run);
    this.emit(
      {
        type: "run.completed",
        run: {
          ...run,
          status: "completed",
          completedAt: completed.completedAt,
        },
      },
      run,
    );
  }
}

window.EventSource = FakeEventSource as unknown as typeof EventSource;

const client = new AgentClient(`${window.location.origin}/agent-api`);
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

window.history.replaceState(null, "", "/agent");

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider theme={themeMantine}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Routes>
            <Route
              path="/agent"
              element={
                <div style={{ height: "100vh" }}>
                  <Chat client={client} info={info} />
                </div>
              }
            />
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </MantineProvider>
  </React.StrictMode>,
);
