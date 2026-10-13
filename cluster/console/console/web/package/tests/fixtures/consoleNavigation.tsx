import "@mantine/core/styles.layer.css";
import "@/index.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";

import type {
  AgentEvent,
  AgentInfo,
  Block,
  Conversation,
  Message,
  Run,
  RunInput,
} from "@/apis/consoleagent/protocol";
import { AgentDockHost } from "@/components/AgentDock";
import { AskAgentButton } from "@/components/AgentDock/AskButtons";
import { AgentProvider } from "@/components/AgentDock/provider";
import { pinnedResource } from "@/components/AgentDock/pageContext";
import {
  CommandPaletteProvider,
  CommandPaletteTrigger,
} from "@/components/CommandPalette";
import { useTrackVisits } from "@/components/CommandPalette/track";
import PageHeader from "@/components/PageHeader";
import {
  PageActions,
  PageHeaderProvider,
  PageTitleAddon,
  usePageDetails,
} from "@/components/PageHeader/context";
import themeMantine from "@/utils/theme/mantine";

interface Call {
  method: string;
  path: string;
  body?: { input?: RunInput };
}

declare global {
  interface Window {
    agentCalls: Call[];
  }
}

const params = new URLSearchParams(window.location.search);
const scenario = params.get("scenario") ?? "ready";
const initialPath = params.get("path") ?? "/core/services";
const prefix = "/agent-api/v1";
const at = (ms: number) => new Date(ms).toISOString();
let counter = 0;

const conversations = new Map<string, Conversation>([
  [
    "c-old",
    {
      id: "c-old",
      title: "Audit admins group permissions",
      createdAt: at(Date.now() - 60000),
      updatedAt: at(Date.now() - 60000),
      messageCount: 0,
    },
  ],
]);
const messages = new Map<string, Message[]>([["c-old", []]]);
const runs = new Map<string, { run: Run; input: RunInput }>();

const info: AgentInfo = {
  name: "octelium-console-agent",
  version: "0.1.0",
  protocolVersion: 2,
  status: "ready",
  issues: [],
  model: { provider: "faux", id: "faux-1", name: "Faux Opus" },
  octelium: { mode: "proxy", user: { name: "george" } },
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

const startRun = (conversationId: string, input: RunInput): Run => {
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
  runs.set(id, { run, input });
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
    if (url.hostname.startsWith("octelium-api.")) {
      return new Response(null, { status: 503 });
    }
    return realFetch(input, init);
  }

  const path = url.pathname.slice(prefix.length);
  const method = init?.method ?? "GET";
  const body =
    typeof init?.body === "string" ? JSON.parse(init.body) : undefined;
  window.agentCalls.push({ method, path, body });

  if (path === "/info") {
    return scenario === "unavailable"
      ? json({ error: { message: "unavailable" } }, 503)
      : json(info);
  }

  if (path === "/conversations" && method === "GET") {
    return json({ items: [...conversations.values()] });
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
    const run = startRun(id, body.input);
    return json({ conversation, run }, 201);
  }

  const runMatch = /^\/conversations\/([^/]+)\/runs$/.exec(path);
  if (runMatch) {
    return json(startRun(runMatch[1], body.input), 201);
  }

  const conversationMatch = /^\/conversations\/([^/]+)$/.exec(path);
  if (conversationMatch) {
    const id = conversationMatch[1];
    return json({
      conversation: conversations.get(id),
      messages: messages.get(id) ?? [],
    });
  }

  if (path === "/models") {
    return json({ current: info.model, models: [info.model] });
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
    const { run, input } = entry;
    const createdAt = at(Date.now());
    const markdown = (text: string): Block => ({
      id: `b-${++counter}`,
      type: "markdown",
      text,
      createdAt,
    });
    const user: Message = {
      id: run.userMessageId,
      conversationId: run.conversationId,
      runId: run.id,
      role: "user",
      status: "completed",
      createdAt,
      context: input.context,
      blocks: [markdown(input.text)],
    };
    const answer = markdown("The agent answered.");
    const assistant: Message = {
      id: run.assistantMessageId,
      conversationId: run.conversationId,
      runId: run.id,
      role: "assistant",
      status: "completed",
      createdAt,
      completedAt: createdAt,
      blocks: [answer],
    };

    this.emit({ type: "run.started", run }, run);
    await sleep(100);
    this.emit({ type: "message.created", message: user }, run);
    this.emit(
      { type: "message.created", message: { ...assistant, blocks: [] } },
      run,
    );
    await sleep(100);
    messages.set(run.conversationId, [
      ...(messages.get(run.conversationId) ?? []),
      user,
      assistant,
    ]);
    this.emit({ type: "message.completed", message: assistant }, run);
    this.emit(
      { type: "run.completed", run: { ...run, status: "completed" } },
      run,
    );
  }
}

window.EventSource = FakeEventSource as unknown as typeof EventSource;

const Page = () => {
  const location = useLocation();
  usePageDetails({ "Time range": "last 1h" });
  return (
    <main>
      <PageTitleAddon>
        <span data-testid="addon">42</span>
      </PageTitleAddon>
      <PageActions>
        <button type="button">Page action</button>
      </PageActions>
      <AskAgentButton
        label="Ask about nginx"
        pinned={() =>
          pinnedResource({
            apiVersion: "core/v1",
            kind: "Service",
            metadata: { name: "nginx.default", uid: "s-1" },
          })
        }
      />
      <output data-testid="location">
        {location.pathname}
        {location.search}
      </output>
    </main>
  );
};

const Tracker = () => {
  useTrackVisits();
  return null;
};

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider theme={themeMantine}>
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[initialPath]}>
          <PageHeaderProvider>
            <AgentProvider>
              <CommandPaletteProvider>
                <Tracker />
                <header style={{ display: "flex", height: 60 }}>
                  <CommandPaletteTrigger />
                </header>
                <div style={{ padding: 16 }}>
                  <PageHeader />
                  <Routes>
                    <Route path="*" element={<Page />} />
                  </Routes>
                </div>
                <AgentDockHost />
              </CommandPaletteProvider>
            </AgentProvider>
          </PageHeaderProvider>
        </MemoryRouter>
      </QueryClientProvider>
    </MantineProvider>
  </React.StrictMode>,
);
