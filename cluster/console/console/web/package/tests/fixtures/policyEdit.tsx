import "@/index.css";
import "@mantine/core/styles.layer.css";
import * as CoreP from "@/apis/corev1/corev1";
import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { ApplyCodeButton } from "@/components/AgentDock/apply";
import {
  AgentAskContext,
  AgentContext,
  type AgentContextValue,
} from "@/components/AgentDock/context";
import Markdown from "@/components/LLMPlayground/Markdown";
import ResourceForm from "@/components/ResourceLayout/ResourceForm";
import PolicyEdit from "@/pages/core/Policy/Edit";
import store from "@/store";
import * as clients from "@/utils/client";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { Drawer, MantineProvider } from "@mantine/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Check } from "lucide-react";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { createMemoryRouter } from "react-router";
import { RouterProvider } from "react-router/dom";

declare global {
  interface Window {
    policyEdit: {
      calls: { method: string; request: any }[];
      asks: any[];
      policy?: any;
      submitted?: any;
    };
  }
}

const params = new URLSearchParams(window.location.search);
window.policyEdit = { calls: [], asks: [] };

const meta = (name: string) => ({ name, uid: `uid-${name}` });

const users = ["alice", "bob", "carol"].map((name) =>
  CoreP.User.create({
    apiVersion: "core/v1",
    kind: "User",
    metadata: meta(name),
    spec: {
      type: CoreP.User_Spec_Type.HUMAN,
      email: `${name}@example.com`,
      groups: ["sre"],
    },
  }),
);

const groups = ["sre", "developers", "auditors"].map((name) =>
  CoreP.Group.create({
    apiVersion: "core/v1",
    kind: "Group",
    metadata: meta(name),
  }),
);

const services = ["billing.production", "grafana.default", "nginx.default"].map(
  (name) =>
    CoreP.Service.create({
      apiVersion: "core/v1",
      kind: "Service",
      metadata: meta(name),
      spec: { mode: CoreP.Service_Spec_Mode.HTTP },
    }),
);

const namespaces = ["production", "staging", "default"].map((name) =>
  CoreP.Namespace.create({
    apiVersion: "core/v1",
    kind: "Namespace",
    metadata: meta(name),
  }),
);

const condition = (type: CoreP.Condition["type"]) =>
  CoreP.Condition.create({ type });
const match = (text: string) => condition({ oneofKind: "match", match: text });
const not = (text: string) => condition({ oneofKind: "not", not: text });

const existing = CoreP.Policy.create({
  apiVersion: "core/v1",
  kind: "Policy",
  metadata: { name: "sre-production", uid: "uid-sre-production" },
  spec: {
    rules: [
      {
        name: "sre-business-hours",
        effect: CoreP.Policy_Spec_Rule_Effect.ALLOW,
        priority: 0,
        condition: condition({
          oneofKind: "all",
          all: {
            of: [
              match('"sre" in ctx.user.spec.groups'),
              match(
                'now().getDayOfWeek("Europe/Berlin") in [1, 2, 3, 4, 5] && now().getHours("Europe/Berlin") >= 9 && now().getHours("Europe/Berlin") < 17',
              ),
              not('ctx.request.http.method == "DELETE"'),
            ],
          },
        }),
      },
      {
        name: "geofence",
        effect: CoreP.Policy_Spec_Rule_Effect.DENY,
        priority: -2,
        condition: not(
          'ctx.session.status.authentication.info.geoip.country.code in ["DE", "FR"]',
        ),
      },
      {
        name: "gold-tier",
        effect: CoreP.Policy_Spec_Rule_Effect.ALLOW,
        priority: 1,
        condition: match(
          'json.parse(ctx.request.http.headers["x-meta"]).tier == "gold"',
        ),
      },
    ],
    enforcementRules: [
      {
        effect: CoreP.Policy_Spec_EnforcementRule_Effect.IGNORE,
        condition: not('ctx.service.spec.mode in ["HTTP", "WEB"]'),
      },
    ],
  },
});

const fresh = CoreP.Policy.create({
  apiVersion: "core/v1",
  kind: "Policy",
  metadata: { name: "" },
  spec: {},
});

const policy = params.get("policy") === "new" ? fresh : existing;

const lists: Record<string, () => unknown> = {
  listUser: () => CoreP.UserList.create({ items: users }),
  listGroup: () => CoreP.GroupList.create({ items: groups }),
  listService: () => CoreP.ServiceList.create({ items: services }),
  listNamespace: () => CoreP.NamespaceList.create({ items: namespaces }),
};

const handlers: Record<string, (request: any) => unknown> = {
  ...lists,
  isAuthorized: (request: EnterpriseP.IsAuthorizedRequest) =>
    EnterpriseP.IsAuthorizedResponse.create({
      isAuthorized:
        request.downstream.oneofKind === "userRef" &&
        request.downstream.userRef.name === "alice",
      reason: {
        type: CoreP.AccessLog_Entry_Common_Reason_Type.POLICY_MATCH,
        details: {
          type: {
            oneofKind: "policyMatch",
            policyMatch: {
              type: {
                oneofKind: "inlinePolicy",
                inlinePolicy: {
                  name: request.additional?.inlinePolicies[0]?.name ?? "",
                },
              },
              ruleName: "sre-business-hours",
              priority: 0,
            },
          },
        },
      },
    }),
};

for (const client of [
  clients.getClientCore(),
  clients.getClientVisibilityCore(),
  clients.getClientPolicyPortal(),
]) {
  for (const method of client.methods) {
    Object.getPrototypeOf(client)[method.localName] = async (request: any) => {
      window.policyEdit.calls.push({ method: method.localName, request });
      const handler = handlers[method.localName];
      return { response: handler ? handler(request) : method.O.create() };
    };
  }
}

const ANSWER = `Here is a condition that matches the **sre** Group on weekdays:

\`\`\`cel
"sre" in ctx.user.spec.groups && ctx.session.status.authentication.info.aal in ["AAL2", "AAL3"]
\`\`\`

And the full spec:

\`\`\`yaml
rules:
  - name: auditors-read-only
    effect: ALLOW
    priority: -1
    condition:
      all:
        of:
          - match: '"auditors" in ctx.user.spec.groups'
          - match: 'ctx.request.http.method in ["GET", "HEAD", "OPTIONS"]'
  - name: deny-otherwise
    effect: DENY
    condition:
      matchAny: true
\`\`\`
`;

const AgentAnswer = () => (
  <aside
    aria-label="AI answer"
    className="rounded-xl border border-slate-200 bg-white p-4 shadow-card"
  >
    <Markdown codeActions={(code) => <ApplyCodeButton {...code} />}>
      {ANSWER}
    </Markdown>
  </aside>
);

const agent = {
  status: "ready",
  hidden: false,
  opened: false,
  trail: [],
  dockElement: null,
  open: () => {},
  close: () => {},
  toggle: () => {},
  ask: () => {},
  consume: () => {},
  setActivity: () => {},
  setConversationId: () => {},
  setDockElement: () => {},
} as unknown as AgentContextValue;

const ai = params.get("ai") !== "false";

const Form = () => (
  <ResourceForm
    item={policy}
    specComponent={PolicyEdit as never}
    submitIcon={<Check size={13} />}
    submitLabel="Save"
    submitPendingLabel="Saving"
    isPending={false}
    isError={false}
    errorLabel="Could not save"
    onSubmit={(item) => {
      window.policyEdit.submitted = CoreP.Policy.toJson(item as CoreP.Policy);
    }}
    onCancel={() => {}}
  />
);

const Page = () =>
  params.get("drawer") === "true" ? (
    <Drawer
      opened
      onClose={() => {}}
      position="right"
      size="min(860px, 100vw)"
      title="Edit Policy"
      zIndex={200}
    >
      <Form />
      {ai && <AgentAnswer />}
    </Drawer>
  ) : (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 p-3 sm:p-6">
      <Form />
      {ai && <AgentAnswer />}
    </div>
  );

const router = createMemoryRouter([{ path: "/", element: <Page /> }], {
  initialEntries: ["/"],
});

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
          <AgentContext.Provider value={ai ? agent : undefined}>
            <AgentAskContext.Provider
              value={{
                available: ai,
                ask: (request) => window.policyEdit.asks.push(request),
              }}
            >
              <RouterProvider router={router} />
            </AgentAskContext.Provider>
          </AgentContext.Provider>
        </QueryClientProvider>
      </MantineProvider>
    </Provider>
  </React.StrictMode>,
);
