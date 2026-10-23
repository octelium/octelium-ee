import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";
import "@fontsource/ubuntu/700.css";
import {
  Service,
  Service_Spec_Config,
  Service_Spec_Config_LLM_Protocol,
  Service_Spec_Config_Upstream,
  Service_Spec_Mode,
} from "@/apis/corev1/corev1";
import {
  ListenTerminalResponse,
  Workspace,
  Workspace_Status_State,
} from "@/apis/cordiumv1/cordiumv1";
import {
  Agent,
  Agent_State,
  Agent_Workspace_Type,
} from "@/apis/enterprisev1/enterprisev1";
import {
  AgentAskContext,
  AgentContext,
  type AgentContextValue,
  type AskRequest,
} from "@/components/AgentDock/context";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import ServiceConnectPage from "@/components/ResourceLayout/ServiceConnectPage";
import { buildConnectPlan, type Facts } from "@/components/ServiceConnect/plan";
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

interface FakeTerminal {
  id: string;
  cols: number;
  rows: number;
  buffer: string;
  line: string;
  remote?: { prompt: string; exit: string };
  closed: boolean;
  subscribers: Set<(msg: ListenTerminalResponse) => void>;
}

interface Calls {
  createTerminal: { id: string; cols: number; rows: number }[];
  removeTerminal: string[];
  writes: Record<string, string>;
  resizes: { id: string; cols: number; rows: number }[];
  startWorkspace: number;
  initializeAgent: number;
}

declare global {
  interface Window {
    connect: {
      calls: Calls;
      asks: Omit<AskRequest, "id">[];
      terminals: () => FakeTerminal[];
      release: () => void;
      navigate: (path: string) => void;
      closeTerminal: (id: string) => void;
      plan: (name: string, facts?: Facts) => unknown;
    };
  }
}

const params = new URLSearchParams(window.location.search);
const missing = new Set(
  (params.get("missing") ?? "").split(",").filter(Boolean),
);
const failing = new Set((params.get("fail") ?? "").split(",").filter(Boolean));
const holdSetup = params.get("hold") === "setup";
const encoder = new TextEncoder();
const decoder = new TextDecoder();
const calls: Calls = {
  createTerminal: [],
  removeTerminal: [],
  writes: {},
  resizes: [],
  startWorkspace: 0,
  initializeAgent: 0,
};
const asks: Omit<AskRequest, "id">[] = [];
const terminals = new Map<string, FakeTerminal>();
let release: (() => void) | undefined;
let counter = 0;

const service = (
  name: string,
  mode: Service_Spec_Mode,
  port: number,
  extra: {
    config?: Service_Spec_Config;
    displayName?: string;
    isDisabled?: boolean;
    isTLS?: boolean;
  } = {},
) =>
  Service.create({
    apiVersion: "core/v1",
    kind: "Service",
    metadata: {
      name: `${name}.default`,
      uid: `uid-${name}`,
      displayName: extra.displayName ?? "",
    },
    spec: {
      mode,
      config: extra.config,
      isDisabled: !!extra.isDisabled,
      isTLS: !!extra.isTLS,
    },
    status: { namespaceRef: { name: "default" }, port },
  });

const services: Record<string, Service> = Object.fromEntries(
  [
    service("ssh-h01", Service_Spec_Mode.SSH, 22),
    service("ssh-denied", Service_Spec_Mode.SSH, 22),
    service("essh", Service_Spec_Mode.SSH, 22, {
      config: {
        name: "",
        parent: "",
        type: {
          oneofKind: "ssh",
          ssh: {
            user: "",
            eSSHMode: true,
            enableLocalPortForwarding: false,
            enableSubsystem: false,
          },
        },
      },
    }),
    service("pg-main", Service_Spec_Mode.POSTGRES, 5432, {
      config: {
        name: "",
        parent: "",
        type: {
          oneofKind: "postgres",
          postgres: { user: "app", database: "appdb", sslMode: 0 },
        },
      },
    }),
    service("k8s", Service_Spec_Mode.KUBERNETES, 6443),
    service("nats-main", Service_Spec_Mode.TCP, 4222),
    service("cache", Service_Spec_Mode.TCP, 6379, {
      config: {
        name: "",
        parent: "",
        upstream: Service_Spec_Config_Upstream.create({
          type: {
            oneofKind: "container",
            container: {
              image: "docker.io/library/redis:7",
              port: 6379,
              command: [],
              args: [],
              env: [],
              replicas: 1,
            },
          },
        }),
        type: { oneofKind: undefined },
      },
    }),
    service("legacy", Service_Spec_Mode.TCP, 7000),
    service("web", Service_Spec_Mode.HTTP, 8080, { isDisabled: true }),
    service("desktop", Service_Spec_Mode.RDP, 3389),
    service("grpc-api", Service_Spec_Mode.GRPC, 8080),
    service("grpc-tls", Service_Spec_Mode.GRPC, 443, { isTLS: true }),
    service("api", Service_Spec_Mode.HTTP, 443, { isTLS: true }),
    service("site", Service_Spec_Mode.WEB, 80),
    service("openai", Service_Spec_Mode.LLM, 80, {
      config: {
        name: "",
        parent: "",
        type: {
          oneofKind: "llm",
          llm: {
            protocol: Service_Spec_Config_LLM_Protocol.OPENAI,
            model: { type: { oneofKind: "value", value: "gpt-5-mini" } },
          },
        },
      },
    }),
    service("claude", Service_Spec_Mode.LLM, 80, {
      config: {
        name: "",
        parent: "",
        type: {
          oneofKind: "llm",
          llm: { protocol: Service_Spec_Config_LLM_Protocol.ANTHROPIC },
        },
      },
    }),
    service("mcp-github", Service_Spec_Mode.MCP, 80, {
      config: {
        name: "",
        parent: "",
        type: { oneofKind: "mcp", mcp: { endpoint: "/mcp" } },
      },
    }),
    service("resolver", Service_Spec_Mode.DNS, 5353),
    service("egress", Service_Spec_Mode.SOCKS5, 1080),
    service("mysql-db", Service_Spec_Mode.MYSQL, 3307, {
      config: {
        name: "",
        parent: "",
        type: {
          oneofKind: "mysql",
          mysql: { user: "app", database: "shop", isTLS: false },
        },
      },
    }),
    service("mongo", Service_Spec_Mode.TCP, 27017),
    service("events", Service_Spec_Mode.TCP, 9092, {
      displayName: "Kafka cluster",
    }),
    service("db", Service_Spec_Mode.TCP, 5432),
    service("syslog", Service_Spec_Mode.UDP, 514),
  ].map((svc) => [svc.metadata!.name.split(".")[0], svc]),
);

const workspace = (state: Workspace_Status_State) =>
  Workspace.create({
    apiVersion: "cordium/v1",
    kind: "Workspace",
    metadata: {
      name: "octelium-agent-x1",
      uid: "ws-uid",
      displayName: "Octelium Agent",
    },
    status: {
      state,
      hostname: "octelium-agent-x1.cordium.example.test",
      regionRef: { name: "default", apiVersion: "", kind: "", uid: "" },
    },
  });

let workspaceState =
  params.get("ws") === "stopped"
    ? Workspace_Status_State.STOPPED
    : Workspace_Status_State.RUNNING;
let agentState =
  params.get("agent") === "none"
    ? Agent_State.NOT_INITIALIZED
    : Agent_State.READY;

const agent = () =>
  Agent.create({
    state: agentState,
    workspaces:
      agentState === Agent_State.NOT_INITIALIZED
        ? []
        : [
            {
              type: Agent_Workspace_Type.PRIMARY,
              workspace: workspace(workspaceState),
            },
          ],
  });

const stub = (
  client: object,
  impl: Record<string, (req: { name?: string }) => unknown>,
) => {
  const proto = Object.getPrototypeOf(client);
  for (const method of (client as { methods: { localName: string }[] })
    .methods) {
    proto[method.localName] = async (request: { name?: string }) => {
      const fn = impl[method.localName];
      if (!fn) throw new RpcError("Unimplemented", "UNIMPLEMENTED");
      return { response: await fn(request) };
    };
  }
};

stub(clients.getClientCore(), {
  getService: (request) => {
    const svc = services[request?.name ?? ""];
    if (!svc) throw new RpcError("Not found", "NOT_FOUND");
    return svc;
  },
});

stub(clients.getClientAgent(), {
  getAgent: () => {
    if (params.get("agent") === "error") {
      throw new RpcError("Cordium is not installed", "FAILED_PRECONDITION");
    }
    return agent();
  },
  initializeAgent: () => {
    calls.initializeAgent += 1;
    agentState = Agent_State.READY;
    workspaceState = Workspace_Status_State.STOPPED;
    return agent();
  },
});

stub(clients.getClientCordium(), {
  getWorkspace: () => workspace(workspaceState),
  startWorkspace: () => {
    calls.startWorkspace += 1;
    workspaceState = Workspace_Status_State.PULLING_IMAGE;
    window.setTimeout(() => {
      workspaceState = Workspace_Status_State.RUNNING;
    }, 600);
    return {};
  },
});

const stdout = (data: string): ListenTerminalResponse => ({
  type: { oneofKind: "stdout", stdout: { data: encoder.encode(data) } },
});

const emit = (term: FakeTerminal, data: string) => {
  term.buffer += data;
  for (const subscriber of term.subscribers) subscriber(stdout(data));
};

const close = (term: FakeTerminal) => {
  if (term.closed) return;
  term.closed = true;
  terminals.delete(term.id);
  for (const subscriber of term.subscribers) {
    subscriber({ type: { oneofKind: "close", close: {} } });
  }
};

const shellPrompt = "octelium@agent ~ % ";

const sleep = (ms: number) =>
  new Promise<void>((resolve) => window.setTimeout(resolve, ms));

const runSetup = async (term: FakeTerminal, script: string) => {
  const nonce = /^N=(\w+)$/m.exec(script)?.[1] ?? "";
  const mark = (body: string) => emit(term, `\r\n@@ocx:${nonce}:${body}\r\n`);
  const tool = /^(?:want|fetch) ([\w-]+)/m.exec(script)?.[1];
  mark("start");
  await sleep(60);
  mark("step:dns:run");
  await sleep(60);
  mark("step:dns:ok");
  if (tool) {
    if (failing.has(tool)) {
      mark(`step:${tool}:run`);
      emit(term, "sudo: a password is required\r\n");
      mark(`step:${tool}:fail:root access is not available`);
    } else if (missing.has(tool)) {
      mark(`step:${tool}:run`);
      emit(term, `Reading package lists...\r\nSetting up ${tool} ...\r\n`);
      if (holdSetup) await new Promise<void>((resolve) => (release = resolve));
      await sleep(120);
      mark(`step:${tool}:ok:${tool}`);
    } else {
      mark(`step:${tool}:ok:${tool}`);
    }
  }
  if (script.includes("step:kubeconfig:run")) {
    mark("step:kubeconfig:run");
    emit(term, "export KUBECONFIG=/home/octelium/.kube/k8s.example.test\r\n");
    mark("fact:kubeconfig=/home/octelium/.kube/k8s.example.test");
    mark("step:kubeconfig:ok");
  }
  mark("done");
  await sleep(30);
  close(term);
};

const execute = (term: FakeTerminal, line: string) => {
  const setup = /^ ?echo (\S+) \| base64 -d > (\S+) && sh \2; exit$/.exec(line);
  if (setup) {
    void runSetup(term, atob(setup[1]));
    return;
  }
  const [program, ...args] = line.trim().split(/\s+/);
  if (term.remote) {
    if (program === "exit" || program === "\\q") {
      term.remote = undefined;
      emit(term, `${shellPrompt}`);
      return;
    }
    emit(term, `${term.remote.prompt}`);
    return;
  }
  switch (program) {
    case "":
      emit(term, shellPrompt);
      return;
    case "ssh": {
      const host = args.at(-1) ?? "";
      if (host === "ssh-denied") {
        emit(
          term,
          `octelium@${host}: Permission denied (publickey).\r\n${shellPrompt}`,
        );
        return;
      }
      term.remote = { prompt: `ubuntu@${host}:~$ `, exit: "logout" };
      emit(
        term,
        `Welcome to Ubuntu 24.04.3 LTS (GNU/Linux 6.8.0-71-generic x86_64)\r\n\r\nLast login: Thu Oct 23 09:12:44 2026\r\n${term.remote.prompt}`,
      );
      return;
    }
    case "psql":
      term.remote = { prompt: "appdb=> ", exit: "" };
      emit(
        term,
        `psql (17.6)\r\nType "help" for help.\r\n\r\n${term.remote.prompt}`,
      );
      return;
    case "redis-cli":
      term.remote = { prompt: "cache:6379> ", exit: "" };
      emit(term, term.remote.prompt);
      return;
    case "export":
      emit(term, shellPrompt);
      return;
    case "kubectl":
      emit(
        term,
        `NAMESPACE     NAME                       READY   STATUS    RESTARTS   AGE\r\nkube-system   coredns-7db6d8ff4d-9xk2p   1/1     Running   0          12d\r\n${shellPrompt}`,
      );
      return;
    case "exit":
      close(term);
      return;
    default:
      emit(term, `zsh: command not found: ${program}\r\n${shellPrompt}`);
  }
};

const write = (term: FakeTerminal, data: string) => {
  calls.writes[term.id] = (calls.writes[term.id] ?? "") + data;
  for (const char of data) {
    if (char === "\r") {
      const line = term.line;
      term.line = "";
      emit(term, "\r\n");
      execute(term, line);
    } else if (char === "\x7f") {
      if (term.line) {
        term.line = term.line.slice(0, -1);
        emit(term, "\b \b");
      }
    } else {
      term.line += char;
      emit(term, char);
    }
  }
};

const listen = (term: FakeTerminal, signal?: AbortSignal) => ({
  responses: {
    async *[Symbol.asyncIterator]() {
      const queue: ListenTerminalResponse[] = [
        {
          type: {
            oneofKind: "windowSize",
            windowSize: { cols: term.cols, rows: term.rows },
          },
        },
        ...(term.buffer ? [stdout(term.buffer)] : []),
      ];
      let wake: (() => void) | undefined;
      const subscriber = (msg: ListenTerminalResponse) => {
        queue.push(msg);
        wake?.();
      };
      term.subscribers.add(subscriber);
      try {
        while (!signal?.aborted) {
          const msg = queue.shift();
          if (msg) {
            yield msg;
            if (msg.type.oneofKind === "close") return;
            continue;
          }
          if (term.closed) return;
          await new Promise<void>((resolve) => {
            wake = resolve;
            signal?.addEventListener("abort", () => resolve(), {
              once: true,
            });
          });
        }
        throw new RpcError("The operation was aborted", "CANCELLED");
      } finally {
        term.subscribers.delete(subscriber);
      }
    },
  },
});

const workspaceClient = clients.getClientCordiumWorkspace();
const workspaceProto = Object.getPrototypeOf(workspaceClient);
workspaceProto.createTerminal = async (request: {
  cols: number;
  rows: number;
}) => {
  counter += 1;
  const id = `octelium-agent-x1-t${counter}`;
  const term: FakeTerminal = {
    id,
    cols: request.cols,
    rows: request.rows,
    buffer: "",
    line: "",
    closed: false,
    subscribers: new Set(),
  };
  terminals.set(id, term);
  calls.createTerminal.push({ id, cols: request.cols, rows: request.rows });
  window.setTimeout(() => emit(term, shellPrompt), 80);
  return { response: { id } };
};
workspaceProto.removeTerminal = async (request: { id: string }) => {
  calls.removeTerminal.push(request.id);
  const term = terminals.get(request.id);
  if (term) close(term);
  return { response: {} };
};
workspaceProto.listenTerminal = (
  request: { id: string },
  options?: { abort?: AbortSignal },
) => {
  const term = terminals.get(request.id);
  if (!term) {
    return {
      responses: {
        [Symbol.asyncIterator]: () => ({
          next: () =>
            Promise.reject(
              new RpcError("Terminal not found", "INVALID_ARGUMENT"),
            ),
        }),
      },
    };
  }
  return listen(term, options?.abort);
};
workspaceProto.writeTerminalData = async (request: {
  id: string;
  data: Uint8Array;
}) => {
  const term = terminals.get(request.id);
  if (!term) {
    throw new RpcError("Terminal does not exist", "INVALID_ARGUMENT");
  }
  write(term, decoder.decode(request.data));
  return { response: {} };
};
workspaceProto.setTerminalWindowSize = async (request: {
  id: string;
  cols: number;
  rows: number;
}) => {
  calls.resizes.push({ ...request });
  const term = terminals.get(request.id);
  if (term) {
    term.cols = request.cols;
    term.rows = request.rows;
  }
  return { response: {} };
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
            { path: "connect", element: <ServiceConnectPage /> },
            { path: "metrics", element: <div>Metrics</div> },
          ],
        },
      ],
    },
  ],
  {
    initialEntries: [params.get("path") ?? "/core/services/ssh-h01/connect"],
  },
);

const ask = (request: Omit<AskRequest, "id">) => {
  asks.push(request);
};

const agentValue = {
  status: params.get("agent") === "unavailable" ? "unavailable" : "ready",
  opened: false,
  hidden: false,
  trail: [],
  dockElement: null,
  open: () => {},
  close: () => {},
  toggle: () => {},
  ask,
  consume: () => {},
  setActivity: () => {},
  setConversationId: () => {},
  setDockElement: () => {},
} as AgentContextValue;

window.connect = {
  calls,
  asks,
  terminals: () => [...terminals.values()],
  release: () => release?.(),
  navigate: (path: string) => void router.navigate(path),
  closeTerminal: (id: string) => {
    const term = terminals.get(id);
    if (term) close(term);
  },
  plan: (name: string, facts: Facts = {}) => {
    const plan = buildConnectPlan(services[name], "example.test");
    return {
      label: plan.label,
      tool: plan.tool,
      address: plan.address,
      hint: plan.hint?.text,
      inferred: plan.inferred,
      commands: plan.commands(facts),
      snippets: plan.snippets.map((snippet) => snippet.text),
    };
  },
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
            <AgentContext.Provider value={agentValue}>
              <AgentAskContext.Provider
                value={{ available: params.get("ask") !== "off", ask }}
              >
                <RouterProvider router={router} />
              </AgentAskContext.Provider>
            </AgentContext.Provider>
          </QueryClientProvider>
        </MantineProvider>
      </Provider>
    </MotionConfig>
  </React.StrictMode>,
);
