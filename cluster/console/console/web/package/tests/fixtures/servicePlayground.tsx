import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";
import "@fontsource/ubuntu/700.css";
import {
  Service,
  Service_Spec_Config_LLM_Protocol,
  Service_Spec_Mode,
} from "@/apis/corev1/corev1";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import ServicePlaygroundPage from "@/components/ResourceLayout/ServicePlaygroundPage";
import serviceInfo from "@/pages/core/Service/router";
import store from "@/store";
import * as clients from "@/utils/client";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MotionConfig } from "framer-motion";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import {
  createMemoryRouter,
  Outlet,
  RouterProvider,
  useLocation,
} from "react-router-dom";

interface Call {
  url: string;
  method: string;
  headers: Record<string, string>;
  body?: any;
}

declare global {
  interface Window {
    playground: {
      calls: Call[];
      navigate: (path: string) => void;
      release: () => void;
    };
  }
}

const params = new URLSearchParams(window.location.search);
const calls: Call[] = [];
let release: (() => void) | undefined;

const llmService = (
  name: string,
  protocol: Service_Spec_Config_LLM_Protocol,
  model?: string,
) =>
  Service.create({
    apiVersion: "core/v1",
    kind: "Service",
    metadata: { name, uid: `uid-${name}` },
    spec: {
      mode: Service_Spec_Mode.LLM,
      isPublic: true,
      config: {
        type: {
          oneofKind: "llm",
          llm: {
            protocol,
            ...(model
              ? { model: { type: { oneofKind: "value", value: model } } }
              : {}),
          },
        },
      },
    },
    status: { namespaceRef: { name: "default" } },
  });

const services: Record<string, Service> = {
  openai: llmService(
    "openai",
    Service_Spec_Config_LLM_Protocol.OPENAI,
    "gpt-5-mini",
  ),
  claude: llmService("claude", Service_Spec_Config_LLM_Protocol.ANTHROPIC),
  "mcp-github": Service.create({
    apiVersion: "core/v1",
    kind: "Service",
    metadata: { name: "mcp-github", uid: "uid-mcp-github" },
    spec: {
      mode: Service_Spec_Mode.MCP,
      isPublic: true,
      config: {
        type: {
          oneofKind: "mcp",
          mcp: {
            endpoint: "/mcp",
            protocol: {
              versions: ["2025-06-18"],
              requireVersion: false,
              rejectUnknownMethods: false,
            },
          },
        },
      },
    },
    status: { namespaceRef: { name: "default" } },
  }),
  web: Service.create({
    apiVersion: "core/v1",
    kind: "Service",
    metadata: { name: "web", uid: "uid-web" },
    spec: { mode: Service_Spec_Mode.HTTP, isPublic: true },
    status: { namespaceRef: { name: "default" } },
  }),
};

const core = clients.getClientCore();
for (const method of core.methods) {
  Object.getPrototypeOf(core)[method.localName] = async (request: any) => {
    if (method.localName === "getService") {
      const service = services[request?.name];
      if (!service) throw new RpcError("Not found", "NOT_FOUND");
      return { response: service };
    }
    return { response: method.O.create() };
  };
}

const sleep = (ms: number, signal?: AbortSignal | null) =>
  new Promise<void>((resolve, reject) => {
    const timer = window.setTimeout(resolve, ms);
    signal?.addEventListener("abort", () => {
      window.clearTimeout(timer);
      reject(new DOMException("The request was aborted.", "AbortError"));
    });
  });

const words = (text: string) => text.match(/\s*\S+/g) ?? [];

const answer = (prompt: string) => {
  if (/table/i.test(prompt)) {
    return [
      "Here is a short comparison:",
      "",
      "| Property | TCP | UDP |",
      "| --- | --- | --- |",
      "| Connection | Connection-oriented | Connectionless |",
      "| Delivery | Reliable, ordered | Best effort |",
      "| Overhead | Higher | Lower |",
      "",
      "Use **TCP** when every byte matters and **UDP** for latency-sensitive traffic such as `voice` or `games`.",
    ].join("\n");
  }
  if (/code|function/i.test(prompt)) {
    return [
      "A rune-aware version:",
      "",
      "```go",
      "func Reverse(s string) string {",
      "\tr := []rune(s)",
      "\tfor i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {",
      "\t\tr[i], r[j] = r[j], r[i]",
      "\t}",
      "\treturn string(r)",
      "}",
      "```",
      "",
      "It converts the string to runes first so multi-byte characters stay intact.",
    ].join("\n");
  }
  return "Hello! I am a helpful assistant served through this Octelium Service. I answer questions, write code and summarize documents. Ask me anything.";
};

const sse = (
  signal: AbortSignal | null | undefined,
  produce: (emit: (data: unknown) => void) => Promise<void>,
) => {
  const encoder = new TextEncoder();
  const body = new ReadableStream<Uint8Array>({
    async start(controller) {
      const emit = (data: unknown) =>
        controller.enqueue(
          encoder.encode(
            `data: ${typeof data === "string" ? data : JSON.stringify(data)}\n\n`,
          ),
        );
      try {
        await produce(emit);
        controller.close();
      } catch (err) {
        controller.error(err);
      }
    },
  });
  return new Response(body, {
    status: 200,
    headers: { "content-type": "text/event-stream" },
  });
};

const json = (status: number, body: unknown, headers?: HeadersInit) =>
  new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json", ...headers },
  });

const lastPrompt = (messages: any[]) => {
  const last = messages.at(-1);
  if (typeof last?.content === "string") return last.content;
  if (Array.isArray(last?.content)) {
    return last.content
      .map((part: any) => part.text ?? "")
      .filter(Boolean)
      .join(" ");
  }
  return "";
};

const hold = async (prompt: string, signal?: AbortSignal | null) => {
  if (!/hold/i.test(prompt)) return;
  await new Promise<void>((resolve, reject) => {
    release = resolve;
    signal?.addEventListener("abort", () =>
      reject(new DOMException("The request was aborted.", "AbortError")),
    );
  });
};

const responses = (body: any, signal?: AbortSignal | null) => {
  const text = answer(lastPrompt(body.input ?? []));
  return sse(signal, async (emit) => {
    for (const chunk of words("Checking the request.")) {
      emit({ type: "response.reasoning_summary_text.delta", delta: chunk });
      await sleep(10, signal);
    }
    for (const chunk of words(text)) {
      emit({ type: "response.output_text.delta", delta: chunk });
      await sleep(8, signal);
    }
    emit({
      type: "response.completed",
      response: {
        model: "gpt-5-mini-2026-08-07",
        status: "completed",
        usage: { input_tokens: 40, output_tokens: 60, total_tokens: 100 },
      },
    });
  });
};

const openAI = async (path: string, body: any, signal?: AbortSignal | null) => {
  if (path.endsWith("/responses")) return responses(body, signal);
  const prompt = lastPrompt(body.messages ?? []);
  if (/deny/i.test(prompt)) {
    return json(403, {
      error: { message: "The Policy deny-llm denied the request" },
    });
  }
  const text = answer(prompt);
  if (!body.stream) {
    return json(200, {
      model: "gpt-5-mini-2026-08-07",
      choices: [{ message: { content: text }, finish_reason: "stop" }],
      usage: { prompt_tokens: 42, completion_tokens: 96, total_tokens: 138 },
    });
  }
  return sse(signal, async (emit) => {
    await sleep(120, signal);
    for (const chunk of words(
      "The user wants a short answer, so I keep it brief.",
    )) {
      emit({
        model: "gpt-5-mini-2026-08-07",
        choices: [{ delta: { reasoning_content: chunk } }],
      });
      await sleep(12, signal);
    }
    await hold(prompt, signal);
    for (const chunk of words(text)) {
      emit({
        model: "gpt-5-mini-2026-08-07",
        choices: [{ delta: { content: chunk } }],
      });
      await sleep(/slow/i.test(prompt) ? 400 : 8, signal);
    }
    emit({ choices: [{ delta: {}, finish_reason: "stop" }] });
    emit({
      choices: [],
      usage: {
        prompt_tokens: 42,
        completion_tokens: 96,
        total_tokens: 138,
        completion_tokens_details: { reasoning_tokens: 12 },
      },
    });
    emit("[DONE]");
  });
};

const anthropic = async (body: any, signal?: AbortSignal | null) => {
  const prompt = lastPrompt(body.messages ?? []);
  const text = answer(prompt);
  return sse(signal, async (emit) => {
    emit({
      type: "message_start",
      message: {
        model: "claude-sonnet-5",
        usage: { input_tokens: 31, output_tokens: 1 },
      },
    });
    if (body.thinking) {
      emit({
        type: "content_block_start",
        content_block: { type: "thinking" },
      });
      for (const chunk of words(
        "Let me think about the cost of the ball step by step.",
      )) {
        emit({
          type: "content_block_delta",
          delta: { type: "thinking_delta", thinking: chunk },
        });
        await sleep(15, signal);
      }
    }
    emit({ type: "content_block_start", content_block: { type: "text" } });
    for (const chunk of words(text)) {
      emit({
        type: "content_block_delta",
        delta: { type: "text_delta", text: chunk },
      });
      await sleep(8, signal);
    }
    emit({
      type: "message_delta",
      delta: { stop_reason: body.max_tokens < 50 ? "max_tokens" : "end_turn" },
      usage: { output_tokens: 87 },
    });
    emit({ type: "message_stop" });
  });
};

const pixel =
  "iVBORw0KGgoAAAANSUhEUgAAAHgAAAAwCAIAAABVDSmEAAAAiElEQVR42u3VIQ7AIBREQe7V2/XGiDapw2JIEATKZ5InV43ZlJ9PE0oIQIMW6CVd99sKNGjQAg0a9IHQPRygQYMGDXoD6OFj0KBBg44AvfayQIMGDRr0fzlAgwYNOgJ01MsCDRo0aNCgQW8NbQwaNGh2oEEb19A4QIM2Bg0aNA7QoI1Bgz5zXACZt4xqIz+IZQAAAABJRU5ErkJggg==";

const tools = [
  {
    name: "search_repositories",
    title: "Search repositories",
    description:
      "Search for GitHub repositories by keyword, language or topic. Returns the matching repositories ordered by the requested sort key.",
    inputSchema: {
      type: "object",
      properties: {
        query: { type: "string", description: "The search query" },
        page: {
          type: "integer",
          description: "Page number",
          default: 1,
          minimum: 1,
        },
        sort: {
          type: "string",
          enum: ["stars", "forks", "updated"],
          description: "Sort key",
        },
        archived: {
          type: "boolean",
          description: "Include archived repositories",
        },
      },
      required: ["query"],
    },
    annotations: { readOnlyHint: true, openWorldHint: true },
  },
  {
    name: "create_issue",
    description: "Create a new issue in a GitHub repository.",
    inputSchema: {
      type: "object",
      properties: {
        owner: { type: "string" },
        repo: { type: "string" },
        title: { type: "string" },
        body: { type: "string" },
        labels: { type: "array", items: { type: "string" } },
      },
      required: ["owner", "repo", "title"],
    },
  },
  {
    name: "delete_branch",
    description: "Delete a branch of a repository.",
    inputSchema: {
      type: "object",
      properties: {
        owner: { type: "string" },
        repo: { type: "string" },
        branch: { type: "string" },
      },
      required: ["owner", "repo", "branch"],
    },
    annotations: { destructiveHint: true },
  },
  {
    name: "render_contributors_chart",
    description: "Render a chart of the contributors of a repository.",
    inputSchema: { type: "object", properties: {} },
    annotations: { readOnlyHint: true },
  },
  {
    name: "get_me",
    description: "Get the authenticated GitHub user.",
    inputSchema: { type: "object", properties: {} },
    annotations: { readOnlyHint: true },
  },
];

const moreTools = [
  {
    name: "list_commits",
    description: "List the commits of a branch.",
    inputSchema: {
      type: "object",
      properties: {
        owner: { type: "string" },
        repo: { type: "string" },
        sha: { type: "string" },
      },
      required: ["owner", "repo"],
    },
    annotations: { readOnlyHint: true },
  },
];

const mcp = async (request: {
  headers: Headers;
  body: any;
  method: string;
}) => {
  if (request.method === "DELETE") return new Response(null, { status: 204 });
  const body = request.body;
  const session = request.headers.get("mcp-session-id");
  const ok = (result: unknown, headers?: HeadersInit) =>
    json(200, { jsonrpc: "2.0", id: body.id, result }, headers);
  await sleep(60);

  switch (body.method) {
    case "initialize":
      return ok(
        {
          protocolVersion: "2025-06-18",
          capabilities: { tools: {}, resources: {}, prompts: {}, logging: {} },
          serverInfo: { name: "github-mcp", version: "1.4.2" },
          instructions:
            "Use search_repositories before reading files. Never delete branches without confirmation.",
        },
        { "Mcp-Session-Id": "mcp-session-7f3a9c2e" },
      );
    case "notifications/initialized":
      return new Response(null, { status: 202 });
    case "ping":
      return ok({});
    case "tools/list":
      return body.params?.cursor
        ? ok({ tools: moreTools })
        : ok({ tools, nextCursor: "page-2" });
    case "tools/call": {
      if (!session) {
        return json(400, {
          jsonrpc: "2.0",
          id: body.id,
          error: {
            code: -32000,
            message: "Bad Request: No valid session ID provided",
          },
        });
      }
      const { name, arguments: args } = body.params;
      if (name === "search_repositories") {
        return ok({
          content: [
            {
              type: "text",
              text: JSON.stringify({
                total_count: 2,
                items: [
                  { full_name: "octelium/octelium", stargazers_count: 3100 },
                  { full_name: "octelium/cordium", stargazers_count: 820 },
                ].filter(() => args.query),
              }),
            },
          ],
        });
      }
      if (name === "render_contributors_chart") {
        return ok({
          content: [
            { type: "text", text: "Contributors over the last 12 months" },
            { type: "image", data: pixel, mimeType: "image/png" },
          ],
        });
      }
      if (name === "create_issue") {
        return ok({
          content: [
            { type: "text", text: "Issues are disabled for this repository." },
          ],
          isError: true,
        });
      }
      return json(200, {
        jsonrpc: "2.0",
        id: body.id,
        error: { code: -32602, message: `Unknown tool: ${name}` },
      });
    }
    case "resources/list":
      return ok({
        resources: [
          {
            uri: "repo://octelium/octelium/README.md",
            name: "README.md",
            description: "The README of the repository",
            mimeType: "text/markdown",
          },
          {
            uri: "repo://octelium/octelium/logo.png",
            name: "logo.png",
            mimeType: "image/png",
            size: 182,
          },
        ],
      });
    case "resources/read":
      return ok({
        contents: body.params.uri.endsWith(".png")
          ? [{ uri: body.params.uri, mimeType: "image/png", blob: pixel }]
          : [
              {
                uri: body.params.uri,
                mimeType: "text/markdown",
                text: "# Octelium\n\nA unified zero trust secure access platform.",
              },
            ],
      });
    case "prompts/list":
      return ok({
        prompts: [
          {
            name: "review_pull_request",
            title: "Review a pull request",
            description: "Review the changes of a pull request.",
            arguments: [
              {
                name: "owner",
                description: "Repository owner",
                required: true,
              },
              { name: "repo", description: "Repository name", required: true },
              {
                name: "number",
                description: "Pull request number",
                required: true,
              },
              { name: "focus", description: "What to focus on" },
            ],
          },
        ],
      });
    case "prompts/get":
      return ok({
        description: "Review a pull request",
        messages: [
          {
            role: "user",
            content: {
              type: "text",
              text: `Review pull request #${body.params.arguments.number} of ${body.params.arguments.owner}/${body.params.arguments.repo}.`,
            },
          },
        ],
      });
    default:
      return json(200, {
        jsonrpc: "2.0",
        id: body.id,
        error: { code: -32601, message: "Method not found" },
      });
  }
};

const realFetch = window.fetch.bind(window);
window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
  const url = new URL(
    typeof input === "string"
      ? input
      : input instanceof URL
        ? input.href
        : input.url,
    window.location.href,
  );
  if (url.protocol !== "https:") return realFetch(input, init);
  const headers = new Headers(init?.headers);
  const body =
    typeof init?.body === "string" ? JSON.parse(init.body) : undefined;
  calls.push({
    url: url.href,
    method: init?.method ?? "GET",
    headers: Object.fromEntries(headers.entries()),
    body,
  });
  const service = url.hostname.split(".")[0];
  if (init?.signal?.aborted) {
    throw new DOMException("The request was aborted.", "AbortError");
  }
  if (service === "openai") return openAI(url.pathname, body, init?.signal);
  if (service === "claude") return anthropic(body, init?.signal);
  if (service === "mcp-github") {
    return mcp({ headers, body, method: init?.method ?? "GET" });
  }
  return json(404, { error: { message: "Not found" } });
};

const Frame = () => {
  const location = useLocation();
  return (
    <>
      <output data-testid="location" className="sr-only">
        {location.pathname}
      </output>
      <Outlet />
    </>
  );
};

const router = createMemoryRouter(
  [
    {
      path: "/",
      element: <Frame />,
      children: [
        { path: "core/services", element: <div /> },
        {
          path: "core/services/:name",
          element: <ResourceItemDrawer />,
          children: [
            {
              path: "",
              element: (
                <ResourceItemMainPage
                  infoComponent={serviceInfo.infoItemsGetter}
                />
              ),
            },
            { path: "playground", element: <ServicePlaygroundPage /> },
            { path: "metrics", element: <div>Metrics</div> },
          ],
        },
      ],
    },
  ],
  {
    initialEntries: [params.get("path") ?? "/core/services/openai/playground"],
  },
);

window.playground = {
  calls,
  navigate: (path: string) => void router.navigate(path),
  release: () => release?.(),
};

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MotionConfig
      reducedMotion={params.get("motion") === "off" ? "always" : "user"}
    >
      <Provider store={store}>
        <MantineProvider
          theme={themeMantine}
          cssVariablesResolver={cssVariablesResolver}
          forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
        >
          <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
          </QueryClientProvider>
        </MantineProvider>
      </Provider>
    </MotionConfig>
  </React.StrictMode>,
);
