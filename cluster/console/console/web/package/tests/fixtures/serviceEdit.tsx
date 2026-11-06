import "@/index.css";
import "@mantine/core/styles.layer.css";
import * as CoreP from "@/apis/corev1/corev1";
import ResourceCreatePage from "@/components/ResourceLayout/ResourceCreate";
import ResourceEditPage from "@/components/ResourceLayout/ResourceEdit";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import ServiceEdit from "@/pages/core/Service/Edit";
import serviceInfo from "@/pages/core/Service/router";
import store from "@/store";
import * as clients from "@/utils/client";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router-dom";

declare global {
  interface Window {
    serviceEdit: {
      calls: { method: string; request: any }[];
      value?: any;
    };
  }
}

const params = new URLSearchParams(window.location.search);
const calls: { method: string; request: any }[] = [];
window.serviceEdit = { calls };

const service = (name: string, spec: Record<string, unknown>, isSystem = false) =>
  CoreP.Service.fromJson(
    {
      apiVersion: "core/v1",
      kind: "Service",
      metadata: { name, uid: `uid-${name}`, resourceVersion: "1", isSystem },
      spec,
      status: {},
    } as any,
    { ignoreUnknownFields: false },
  );

const matchAny = { matchAny: true };

const services: Record<string, CoreP.Service> = {
  "api-gateway": service("api-gateway", {
    mode: "HTTP",
    port: 8443,
    isTLS: true,
    isPublic: true,
    region: "eu-west",
    deployment: { replicas: 3 },
    attrs: { tier: "gold", team: "payments" },
    authorization: {
      policies: ["allow-engineering", "deny-contractors"],
      inlinePolicies: [
        {
          name: "only-office-hours",
          spec: {
            rules: [{ condition: { match: "ctx.user.spec.type == 'HUMAN'" } }],
          },
        },
      ],
    },
    config: {
      upstream: { url: "https://api.internal.example.com:8443" },
      tls: {
        trustedCAs: ["-----BEGIN CERTIFICATE-----\nMIIDazCCAlOgAwIBAgIU\n-----END CERTIFICATE-----"],
        appendToSystemPool: true,
        clientCertificate: { fromSecret: "api-client-cert" },
      },
      http: {
        auth: { bearer: { fromSecret: "upstream-token" } },
        isUpstreamHTTP2: true,
        enableRequestBuffering: true,
        header: {
          forwardedMode: "OBFUSCATE",
          authorizationMode: "PASS",
          addRequestHeaders: [
            { key: "X-Team", value: "payments" },
            { key: "X-User", eval: "ctx.user.metadata.name", append: true },
          ],
          addResponseHeaders: [{ key: "X-Served-By", value: "octelium" }],
          removeRequestHeaders: ["X-Debug"],
          removeResponseHeaders: ["Server", "X-Powered-By"],
          host: { preserve: true },
        },
        path: { addPrefix: "/v2", removePrefix: "/api" },
        body: {
          mode: "JSON",
          maxRequestSize: 1048576,
          validation: { jsonSchema: { inline: '{ "type": "object" }' } },
        },
        retry: {
          maxRetries: 4,
          multiplier: 2,
          statusCodes: [502, 503, 504],
          retryOnServerErrors: true,
          initialInterval: { milliseconds: 200 },
          maxInterval: { seconds: 5 },
          maxElapsedTime: { seconds: 30 },
        },
        cors: {
          allowOriginStringMatch: ["https://app.example.com", "https://admin.example.com"],
          allowMethods: "GET, POST, OPTIONS",
          allowHeaders: "Content-Type, Authorization",
          exposeHeaders: "X-Request-ID",
          maxAge: "86400",
          allowCredentials: true,
          allowClusterServices: true,
        },
        visibility: {
          enableRequestBody: true,
          enableRequestBodyMap: true,
          includeRequestHeaders: ["X-Request-ID", "User-Agent"],
          excludeResponseHeaders: ["Set-Cookie"],
        },
        plugins: [
          {
            name: "limit-by-user",
            phase: "POST_AUTH",
            condition: matchAny,
            rateLimit: {
              limit: 100,
              statusCode: 429,
              window: { minutes: 1 },
              key: { perUser: true },
              body: { inline: '{ "message": "Too Many Requests" }' },
              headers: [{ key: "Retry-After", value: "60" }],
            },
          },
          {
            name: "maintenance",
            isDisabled: true,
            phase: "PRE_AUTH",
            condition: { match: "ctx.request.http.path == '/health'" },
            direct: {
              statusCode: 503,
              body: { inline: "Down for maintenance" },
              headers: [{ key: "Content-Type", value: "text/plain" }],
            },
          },
          {
            name: "cache-reads",
            condition: matchAny,
            cache: {
              ttl: { minutes: 5 },
              maxSize: 1048576,
              useXCacheHeader: true,
              key: { eval: "ctx.request.http.path" },
            },
          },
          {
            name: "add-trace",
            condition: matchAny,
            lua: { inline: "function onRequest(ctx)\n  print('hello')\nend" },
          },
          {
            name: "schema",
            condition: matchAny,
            jsonSchema: {
              inline: '{ "type": "object" }',
              statusCode: 422,
              headers: [{ key: "X-Reason", value: "schema" }],
            },
          },
          {
            name: "strip-v1",
            condition: matchAny,
            path: { removePrefix: "/v1", addPrefix: "/v2" },
          },
          {
            name: "ext-authz",
            condition: matchAny,
            extProc: {
              address: "ext-proc.default.svc:8080",
              messageTimeout: { seconds: 5 },
              processingMode: {
                requestHeaderMode: "SEND",
                responseHeaderMode: "SKIP",
                requestBodyMode: "BUFFERED",
                responseBodyMode: "NONE",
              },
            },
          },
        ],
      },
    },
    dynamicConfig: {
      configs: [
        {
          name: "canary",
          upstream: { url: "https://canary.internal.example.com" },
          http: { auth: { bearer: { fromSecret: "canary-token" } } },
        },
        {
          name: "eu-users",
          parent: "canary",
          upstream: { url: "https://eu.internal.example.com" },
        },
      ],
      rules: [
        {
          condition: { match: "ctx.user.metadata.name == 'alice'" },
          configName: "canary",
        },
        { condition: { match: "ctx.session.spec.isBrowser" }, eval: '{ "upstream": { "url": "https://browser.example.com" } }' },
        { condition: { match: "'eu' in ctx.user.spec.groups" }, opa: 'package octelium.eval\n\nresult := {"upstream": {"url": "https://x.example.com"}}' },
      ],
    },
  }),
  "web-app": service("web-app", {
    mode: "WEB",
    isPublic: true,
    config: {
      upstream: {
        container: {
          image: "ghcr.io/acme/web:1.4.2",
          port: 8080,
          replicas: 2,
          command: ["/bin/sh", "-c"],
          args: ["exec node server.js --port 8080"],
          env: [
            { name: "NODE_ENV", value: "production" },
            { name: "DATABASE_URL", fromSecret: "web-database-url" },
            {
              name: "STRIPE_KEY",
              kubernetesSecretRef: { name: "stripe", key: "api-key" },
            },
          ],
          credentials: {
            usernamePassword: {
              username: "ci-bot",
              server: "ghcr.io",
              password: { fromSecret: "ghcr-token" },
            },
          },
          resourceLimit: {
            cpu: { millicores: 2000 },
            memory: { megabytes: 4096 },
            ext: { "nvidia.com/gpu": "1" },
          },
          securityContext: {
            readOnlyRootFilesystem: true,
            runAsUser: 1000,
            capabilities: { add: ["NET_BIND_SERVICE"], drop: ["ALL"] },
          },
          volumes: [
            { name: "cache", emptyDir: { sizeLimitMegabytes: 512 } },
            { name: "uploads", persistentVolumeClaim: { name: "uploads-pvc" } },
          ],
          volumeMounts: [
            { name: "cache", mountPath: "/var/cache/web" },
            { name: "uploads", mountPath: "/data", subPath: "web", readOnly: true },
          ],
          livenessProbe: {
            httpGet: { path: "/healthz", port: 8080 },
            initialDelaySeconds: 5,
            periodSeconds: 10,
            timeoutSeconds: 2,
            failureThreshold: 3,
          },
          readinessProbe: { tcpSocket: { port: 8080 }, periodSeconds: 5 },
        },
      },
      http: {
        response: {
          direct: {
            statusCode: 200,
            contentType: "application/json",
            inline: '{ "status": "ok" }',
          },
        },
      },
    },
  }),
  "grpc-lb": service("grpc-lb", {
    mode: "GRPC",
    config: {
      upstream: {
        loadbalance: {
          endpoints: [
            { url: "grpc://10.0.0.10:50051" },
            { url: "grpc://10.0.0.11:50051", user: "alice" },
          ],
        },
      },
      http: {},
    },
  }),
  "prod-ssh": service("prod-ssh", {
    mode: "SSH",
    config: {
      upstream: { url: "ssh://10.0.0.4", user: "alice" },
      ssh: {
        user: "root",
        enableLocalPortForwarding: true,
        enableSubsystem: true,
        auth: { privateKey: { fromSecret: "ssh-key" } },
        upstreamHostKey: { key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample" },
        visibility: { enableSessionStdinRecording: true },
      },
    },
  }),
  essh: service("essh", {
    mode: "SSH",
    config: { ssh: { eSSHMode: true, enableSubsystem: true } },
  }),
  "pg-main": service("pg-main", {
    mode: "POSTGRES",
    config: {
      upstream: { url: "postgres://db.internal:5432" },
      tls: {
        trustedCAs: ["-----BEGIN CERTIFICATE-----\nMIIDazCCAlOgAwIBAgIU\n-----END CERTIFICATE-----"],
      },
      postgres: {
        user: "app",
        database: "orders",
        sslMode: "VERIFY_FULL",
        auth: { password: { fromSecret: "pg-password" } },
        authorization: { mode: "ALL" },
        visibility: { disableQuery: true },
      },
    },
  }),
  "pg-noauth": service("pg-noauth", {
    mode: "POSTGRES",
    config: { upstream: { url: "postgres://db.internal" }, postgres: {} },
  }),
  "mysql-main": service("mysql-main", {
    mode: "MYSQL",
    config: {
      upstream: { url: "mysql://db.internal:3306" },
      mysql: {
        user: "app",
        database: "shop",
        isTLS: true,
        auth: { password: { fromSecret: "mysql-password" } },
        authorization: { mode: "NONE" },
      },
    },
  }),
  "k8s-cluster": service("k8s-cluster", {
    mode: "KUBERNETES",
    isPublic: true,
    config: {
      upstream: { url: "https://10.0.0.1:6443" },
      kubernetes: {
        clientCertificate: {
          fromSecret: "k8s-client-cert",
          trustedCAs: ["-----BEGIN CERTIFICATE-----\nMIIDazCCAlOgAwIBAgIU\n-----END CERTIFICATE-----"],
        },
      },
    },
  }),
  "k8s-kubeconfig": service("k8s-kubeconfig", {
    mode: "KUBERNETES",
    config: {
      upstream: { url: "https://10.0.0.1:6443" },
      kubernetes: {
        kubeconfig: { fromSecret: "kubeconfig", context: "admin@cluster" },
      },
    },
  }),
  socks: service("socks", {
    mode: "SOCKS5",
    config: {
      upstream: { url: "socks5://proxy.internal" },
      socks5: {
        auth: {
          usernamePassword: {
            username: "proxy-user",
            password: { fromSecret: "socks-password" },
          },
        },
      },
    },
  }),
  "socks-embedded": service("socks-embedded", {
    mode: "SOCKS5",
    config: { socks5: { isEmbeddedMode: true } },
  }),
  desktop: service("desktop", {
    mode: "RDP",
    config: {
      upstream: { url: "rdp://10.0.0.8" },
      rdp: {
        auth: {
          user: "administrator",
          domain: "CORP",
          password: { fromSecret: "rdp-password" },
        },
        upstreamTLS: { pinnedCertSHA256: ["ab:cd:ef:12:34"] },
      },
    },
  }),
  "desktop-web": service("desktop-web", {
    mode: "RDP_WEB",
    isPublic: true,
    config: { upstream: { url: "rdp://10.0.0.9" }, rdp: {} },
  }),
  "mcp-github": service("mcp-github", {
    mode: "MCP",
    isPublic: true,
    config: {
      upstream: { url: "https://api.githubcopilot.com/mcp" },
      mcp: {
        endpoint: "/mcp",
        protocol: {
          versions: ["2025-06-18"],
          requireVersion: true,
          rejectUnknownMethods: true,
        },
        limits: { maxRequestBytes: 1048576, maxStreamEventBytes: 65536 },
        auth: { bearer: { fromSecret: "github-token" } },
        visibility: {
          disableResponseBody: true,
          includeAllRequestHeaders: true,
          excludeRequestHeaders: ["Cookie"],
        },
        cors: { allowClusterServices: true },
        disableOriginCheck: false,
        plugins: [
          {
            name: "no-secrets",
            condition: matchAny,
            guardrail: {
              leg: "BOTH",
              scopes: ["TOOL_ARGUMENTS", "TOOL_RESULTS"],
              denyMessage: "Blocked by policy.",
              patterns: [
                { secrets: { excludeRules: ["generic-api-key"] }, action: "DENY" },
                { type: "EMAIL", action: "DENY" },
              ],
            },
          },
          {
            name: "limit",
            condition: matchAny,
            rateLimit: { limit: 10, window: { seconds: 10 } },
          },
        ],
      },
    },
  }),
  openai: service("openai", {
    mode: "LLM",
    isPublic: true,
    config: {
      upstream: { url: "https://api.openai.com/v1" },
      llm: {
        protocol: "OPENAI",
        model: { value: "gpt-5-mini" },
        auth: { bearer: { fromSecret: "openai-key" } },
        limits: { maxRequestBytes: 4194304, maxOutputTokens: 4096, maxTools: 16 },
        reasoning: { level: "MEDIUM" },
        embedding: {
          model: "text-embedding-3-small",
          dimensions: 512,
          source: { currentUpstream: true },
        },
        translation: { upstreamProtocol: "ANTHROPIC", defaultMaxOutputTokens: 4096 },
        visibility: {
          enableRequestBody: true,
          includeRequestHeaders: ["X-Request-ID"],
        },
        cors: { allowClusterServices: true },
        plugins: [
          {
            name: "redact-pii",
            condition: matchAny,
            guardrail: {
              leg: "REQUEST",
              scopes: ["CONTENT", "TOOL_RESULTS"],
              patterns: [
                { type: "EMAIL", action: "REDACT" },
                {
                  regex: "[0-9]{16}",
                  action: "REPLACE",
                  replace: { value: "[CARD]" },
                },
              ],
            },
          },
          {
            name: "system-prompt",
            condition: matchAny,
            prompt: {
              system: { mode: "PREPEND", content: { value: "You are helpful." } },
            },
          },
          {
            name: "greeting",
            condition: matchAny,
            prompt: {
              message: {
                role: "USER",
                position: "NEW_BEFORE",
                selector: "LAST",
                content: { eval: "'Hello ' + ctx.user.metadata.name" },
              },
            },
          },
          {
            name: "no-shell",
            condition: matchAny,
            tools: {
              choice: "AUTO",
              denyMessage: "Tool not allowed.",
              filters: [
                { name: "read_*", decision: "ALLOW" },
                { type: "web_search*", decision: "DENY" },
                {
                  name: "exec_*",
                  decision: "REPLACE",
                  replace: { value: '{ "name": "exec" }' },
                },
              ],
              tools: [{ value: '{ "name": "lookup" }', position: "APPEND" }],
            },
          },
          {
            name: "cheap-model",
            condition: { match: "ctx.request.llm.model == 'auto'" },
            model: { eval: "'gpt-5-mini'" },
          },
          {
            name: "think-less",
            condition: matchAny,
            reasoning: { tokenBudget: 2048 },
          },
          {
            name: "quota",
            condition: matchAny,
            tokenRateLimit: {
              scope: "TOTAL",
              key: { perUser: true },
              limit: 100000,
              window: { hours: 1 },
              defaultOutputTokens: 1024,
              denyMessage: "Token quota exceeded.",
              headers: [{ key: "Retry-After", value: "3600" }],
            },
          },
          {
            name: "cache",
            condition: matchAny,
            semanticCache: {
              minSimilarity: 0.92,
              maxSize: 65536,
              ttl: { hours: 1 },
              useXCacheHeader: true,
              scope: { shared: true },
            },
          },
          {
            name: "router",
            condition: { match: "ctx.request.llm.model == 'auto'" },
            semanticRouter: {
              minSimilarity: 0.4,
              fallbackModel: "gpt-5-mini",
              embedding: {
                model: "text-embedding-3-small",
                source: {
                  upstream: {
                    url: "https://api.openai.com/v1",
                    protocol: "OPENAI",
                    auth: { bearer: { fromSecret: "openai-key" } },
                  },
                },
              },
              routes: [
                {
                  name: "coding",
                  model: "gpt-5",
                  description: "Software engineering questions",
                  examples: ["Refactor this function", "Why is this deadlocking?"],
                  minSimilarity: 0.5,
                },
              ],
            },
          },
        ],
      },
    },
  }),
  "tcp-raw": service("tcp-raw", {
    mode: "TCP",
    port: 9000,
    config: { upstream: { url: "tcp://10.0.0.20:9000" } },
  }),
  "udp-raw": service("udp-raw", {
    mode: "UDP",
    port: 9001,
    config: { upstream: { url: "udp://10.0.0.21:9001" } },
  }),
  resolver: service("resolver", {
    mode: "DNS",
    config: { upstream: { url: "dns://10.0.0.53" } },
  }),
  "anon-docs": service("anon-docs", {
    mode: "HTTP",
    isPublic: true,
    isAnonymous: true,
    config: { upstream: { url: "https://docs.example.com" }, http: {} },
  }),
  "dynamic-only": service("dynamic-only", {
    mode: "HTTP",
    dynamicConfig: {
      configs: [
        { name: "one", upstream: { url: "https://one.example.com" }, http: {} },
      ],
      rules: [{ condition: matchAny, configName: "one" }],
    },
  }),
  "legacy-cert": service("legacy-cert", {
    mode: "HTTP",
    config: {
      upstream: { url: "https://mtls.example.com" },
      clientCertificate: {
        fromSecret: "legacy-cert",
        trustedCAs: ["-----BEGIN CERTIFICATE-----\nMIIDazCCAlOgAwIBAgIU\n-----END CERTIFICATE-----"],
      },
      http: {},
    },
  }),
  "system-portal": service(
    "system-portal",
    {
      mode: "WEB",
      isPublic: true,
      config: {
        upstream: { url: "http://portal.octelium.svc:8080" },
        http: { auth: { bearer: { fromSecret: "portal-token" } } },
      },
    },
    true,
  ),
};

const resourceNamed = <T extends { create: (arg: any) => any }>(
  type: T,
  kind: string,
  names: string[],
) =>
  names.map((name) =>
    type.create({
      apiVersion: "core/v1",
      kind,
      metadata: { name, uid: `uid-${name}` },
      spec: {},
      status: {},
    }),
  );

const secrets = resourceNamed(CoreP.Secret, "Secret", [
  "upstream-token",
  "api-client-cert",
  "canary-token",
  "web-database-url",
  "ghcr-token",
  "ssh-key",
  "pg-password",
  "mysql-password",
  "k8s-client-cert",
  "kubeconfig",
  "socks-password",
  "rdp-password",
  "github-token",
  "openai-key",
  "legacy-cert",
  "portal-token",
]);
const users = resourceNamed(CoreP.User, "User", ["alice", "bob"]);
const regions = resourceNamed(CoreP.Region, "Region", ["default", "eu-west"]);
const policies = resourceNamed(CoreP.Policy, "Policy", [
  "allow-engineering",
  "deny-contractors",
]);

const handlers: Record<string, (request: any) => unknown> = {
  getService: (request) => services[request?.name],
  listSecret: () => CoreP.SecretList.create({ items: secrets }),
  getSecret: (request) =>
    secrets.find((item) => item.metadata?.name === request?.name),
  listUser: () => CoreP.UserList.create({ items: users }),
  getUser: (request) =>
    users.find((item) => item.metadata?.name === request?.name),
  listRegion: () => CoreP.RegionList.create({ items: regions }),
  getRegion: (request) =>
    regions.find((item) => item.metadata?.name === request?.name),
  listPolicy: () => CoreP.PolicyList.create({ items: policies }),
  getPolicy: (request) =>
    policies.find((item) => item.metadata?.name === request?.name),
  createService: (request) => request,
  updateService: (request) => request,
};

for (const client of [
  clients.getClientCore(),
  clients.getClientVisibilityCore(),
  clients.getClientEnterprise(),
  clients.getClientVisibilityEnterprise(),
]) {
  for (const method of client.methods) {
    Object.getPrototypeOf(client)[method.localName] = async (request: any) => {
      calls.push({ method: method.localName, request });
      const handler = handlers[method.localName];
      if (handler) {
        const response = handler(request);
        if (!response) throw new RpcError("Not found", "NOT_FOUND");
        return { response };
      }
      return { response: method.O.create() };
    };
  }
}

const EditOnly = () => {
  const name = params.get("name");
  const item = React.useMemo(
    () =>
      name && services[name]
        ? services[name]
        : CoreP.Service.create({
            apiVersion: "core/v1",
            kind: "Service",
            metadata: {},
            spec: CoreP.Service_Spec.create(),
            status: {},
          }),
    [name],
  );
  return (
    <div
      className="mx-auto p-6"
      style={{ maxWidth: Number(params.get("width") ?? 896) }}
    >
      <ServiceEdit
        item={item}
        onUpdate={(next: CoreP.Service) => {
          window.serviceEdit.value = CoreP.Service.toJson(next);
        }}
      />
    </div>
  );
};

const router = createMemoryRouter(
  [
    {
      path: "/",
      element: <Outlet />,
      children: [
        { path: "edit-only", element: <EditOnly /> },
        {
          path: "core/services",
          element: <Outlet />,
          children: [
            {
              path: "create",
              element: (
                <div className="mx-auto max-w-5xl p-6">
                  <ResourceCreatePage specComponent={serviceInfo.Item.Edit!} />
                </div>
              ),
            },
            {
              path: ":name",
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
                {
                  path: "edit",
                  element: (
                    <ResourceEditPage specComponent={serviceInfo.Item.Edit!} />
                  ),
                },
              ],
            },
          ],
        },
      ],
    },
  ],
  { initialEntries: [params.get("path") ?? "/edit-only"] },
);

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Provider store={store}>
      <MantineProvider
        theme={themeMantine}
        cssVariablesResolver={cssVariablesResolver}
        forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
      >
        <QueryClientProvider client={queryClient}>
          <React.Suspense fallback={null}>
            <RouterProvider router={router} />
          </React.Suspense>
        </QueryClientProvider>
      </MantineProvider>
    </Provider>
  </React.StrictMode>,
);
