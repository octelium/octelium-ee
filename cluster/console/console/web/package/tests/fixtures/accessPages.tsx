import "@/index.css";
import "@mantine/core/styles.layer.css";
import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { RouteErrorBoundary } from "@/components/ErrorBoundary";
import ResourceCreateRoute from "@/components/ResourceLayout/ResourceCreateRoute";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import catalogInfo from "@/pages/access/Catalog/router";
import PolicyEdit from "@/pages/access/Policy/Edit";
import requestInfo from "@/pages/access/Request/router";
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
import { createMemoryRouter, Outlet } from "react-router";
import { RouterProvider } from "react-router/dom";

declare global {
  interface Window {
    access: {
      calls: { method: string; request: any }[];
      policy?: any;
    };
  }
}

const params = new URLSearchParams(window.location.search);
const calls: { method: string; request: any }[] = [];
window.access = { calls };

const at = (minutes: number) =>
  Timestamp.fromDate(new Date(Date.now() + minutes * 60_000));
const ref = (apiVersion: string, kind: string, name: string) => ({
  apiVersion,
  kind,
  name,
  uid: `uid-${name}`,
});
const S = AccessP.Request_Status_State_Status;

const step = (name: string, reviewer: string, timeoutHours?: number) =>
  AccessP.Policy_Spec_Rule_Action_Review_Step.create({
    name,
    approvalRequirement:
      AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ANY,
    onTimeout: AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout.REJECT,
    reviewers: [
      {
        type: {
          oneofKind: "group",
          group: { groupRef: ref("core/v1", "Group", reviewer) },
        },
      },
    ],
    timeout: timeoutHours
      ? { type: { oneofKind: "hours", hours: timeoutHours } }
      : undefined,
  });

const reviewRule = AccessP.Policy_Spec_Rule.create({
  name: "production-review",
  effect: AccessP.Policy_Spec_Rule_Effect.REVIEW,
  condition: {
    type: {
      oneofKind: "resource",
      resource: {
        type: {
          oneofKind: "catalogRef",
          catalogRef: ref("access/v1", "Catalog", "production-support"),
        },
      },
    },
  },
  action: {
    type: {
      oneofKind: "review",
      review: { steps: [step("team-lead", "sre"), step("security", "security", 2)] },
    },
  },
  maxDuration: { type: { oneofKind: "hours", hours: 8 } },
  message: "Reviewers expect an incident or change ticket.",
});

const nestedRule = AccessP.Policy_Spec_Rule.create({
  name: "oncall-auto-approve",
  effect: AccessP.Policy_Spec_Rule_Effect.AUTO_APPROVE,
  priority: -1,
  condition: {
    type: {
      oneofKind: "all",
      all: {
        of: [
          {
            type: {
              oneofKind: "subject",
              subject: {
                type: {
                  oneofKind: "groupRef",
                  groupRef: ref("core/v1", "Group", "sre-oncall"),
                },
              },
            },
          },
          {
            type: {
              oneofKind: "resource",
              resource: {
                type: {
                  oneofKind: "catalogRef",
                  catalogRef: ref("access/v1", "Catalog", "production-support"),
                },
              },
            },
          },
        ],
      },
    },
  },
});

const unrelatedRule = AccessP.Policy_Spec_Rule.create({
  name: "grafana-review",
  effect: AccessP.Policy_Spec_Rule_Effect.REVIEW,
  condition: {
    type: {
      oneofKind: "resource",
      resource: {
        type: {
          oneofKind: "serviceRef",
          serviceRef: ref("core/v1", "Service", "grafana.default"),
        },
      },
    },
  },
  action: {
    type: { oneofKind: "review", review: { steps: [step("approval", "sre")] } },
  },
});

const policy = AccessP.Policy.create({
  apiVersion: "access/v1",
  kind: "Policy",
  metadata: { name: "production-access", uid: "uid-production-access" },
  spec: { rules: [reviewRule] },
});

const catalogPolicy = AccessP.Policy.create({
  apiVersion: "access/v1",
  kind: "Policy",
  metadata: { name: "production-access", uid: "uid-production-access" },
  spec: { rules: [nestedRule, reviewRule, unrelatedRule] },
});

const request = (
  name: string,
  status: AccessP.Request_Status_State_Status,
  rule?: AccessP.Policy_Spec_Rule,
  currentStep = 0,
) =>
  AccessP.Request.create({
    apiVersion: "access/v1",
    kind: "Request",
    metadata: { name, uid: `uid-${name}`, createdAt: at(-120) },
    spec: {
      urgency: AccessP.Request_Spec_Urgency.HIGH,
      justification: "Debugging a failed deployment.",
      duration: { type: { oneofKind: "hours", hours: 3 } },
      resource: {
        type: {
          oneofKind: "catalog",
          catalog: { catalogRef: ref("access/v1", "Catalog", "production-support") },
        },
      },
      subject: { type: { oneofKind: "userRef", userRef: ref("core/v1", "User", "bob") } },
    },
    status: {
      state: { status, createdAt: at(-60) },
      userRef: ref("core/v1", "User", "bob"),
      policyRef: rule ? ref("access/v1", "Policy", "production-access") : undefined,
      rule,
      review: rule
        ? {
            currentStep,
            currentStepStartedAt: at(-50),
            lastSteps:
              currentStep > 0
                ? [
                    {
                      reviewRef: ref("access/v1", "Review", `rv1.${name}`),
                      setAt: at(-55),
                      stepIndex: 0,
                      decision: AccessP.Review_Spec_Decision.APPROVE,
                      userRef: ref("core/v1", "User", "carol"),
                      stepName: "team-lead",
                    },
                  ]
                : [],
          }
        : undefined,
    },
  });

const approved = request("req-approved", S.APPROVED, reviewRule, 2);
approved.status!.state!.reason = AccessP.Request_Status_State_Reason.APPROVED_BY_REVIEWERS;
approved.status!.accessStartsAt = at(-30);
approved.status!.accessEndsAt = at(150);
approved.status!.resolvedScope = AccessP.Request_Status_ResolvedScope.create({
  catalogRef: ref("access/v1", "Catalog", "production-support"),
  catalogDisplayName: "Production support",
  accessLevel: {
    name: "read-only",
    displayName: "Read only",
    type: { oneofKind: "restricted", restricted: { policies: ["prod-read-only"] } },
  },
  serviceRefs: [ref("core/v1", "Service", "billing.production")],
  totalExcludedService: 1,
});
approved.status!.grant = AccessP.Request_Status_Grant.create({
  state: AccessP.Request_Status_Grant_State.ACTIVE,
  activatedAt: at(-30),
});

const revoked = request("req-revoked", S.REVOKED);
revoked.status!.state = {
  status: S.REVOKED,
  createdAt: at(-10),
  reason: AccessP.Request_Status_State_Reason.ACCESS_REVOKED,
  message: "The incident is closed.",
  actorRef: ref("core/v1", "User", "alice"),
};

const requests: Record<string, AccessP.Request> = {
  "req-pending": request("req-pending", S.PENDING, reviewRule, 1),
  "req-unmatched": request("req-unmatched", S.REJECTED),
  "req-approved": approved,
  "req-revoked": revoked,
};

const reviews = [
  AccessP.Review.create({
    metadata: { name: "rv1.req-pending", uid: "uid-rv1" },
    spec: {
      decision: AccessP.Review_Spec_Decision.APPROVE,
      justification: "Matches the incident.",
    },
    status: {
      userRef: ref("core/v1", "User", "carol"),
      requestRef: ref("access/v1", "Request", "req-pending"),
      setAt: at(-55),
      stepIndex: 0,
      stepName: "team-lead",
    },
  }),
];

const catalog = AccessP.Catalog.create({
  apiVersion: "access/v1",
  kind: "Catalog",
  metadata: { name: "production-support", uid: "uid-production-support" },
  spec: {
    resourceCollection: {
      service: { services: ["billing.production"], namespaces: ["production"] },
    },
    accessLevels: [
      {
        name: "read-only",
        displayName: "Read only",
        description: "Read the production data without changing it.",
        type: { oneofKind: "restricted", restricted: { policies: ["prod-read-only"] } },
      },
      {
        name: "break-glass",
        displayName: "Break glass",
        type: { oneofKind: "full", full: { priority: -1 } },
      },
    ],
    audience: { groups: ["sre"], users: [], match: "", onBehalf: { users: [], groups: ["sre-leads"] } },
    owners: { users: [], groups: ["platform"] },
    requirements: { ticket: { isRequired: true, label: "Change ticket", pattern: "" } },
  },
  status: {
    totalService: 3,
    checkedAt: at(-1),
    issues: [
      {
        type: AccessP.ConfigurationIssue_Type.OVERRIDES_GUARDRAILS,
        severity: AccessP.ConfigurationIssue_Severity.WARNING,
        message: "The access level break-glass overrides deny guardrails.",
        accessLevel: "break-glass",
      },
    ],
  },
});

const handlers: Record<string, (request: any) => unknown> = {
  getRequest: (request) => requests[request?.name],
  getCatalog: () => catalog,
  listReview: (request) =>
    AccessP.ReviewList.create({
      items: reviews.filter(
        (review) => review.status?.requestRef?.name === request?.requestRef?.name,
      ),
    }),
  listPolicy: () => AccessP.PolicyList.create({ items: [catalogPolicy] }),
};

for (const client of [
  clients.getClientAccess(),
  clients.getClientVisibilityAccess(),
  clients.getClientCore(),
  clients.getClientVisibilityCore(),
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

const PolicyEditPage = () => (
  <div className="mx-auto max-w-4xl p-6">
    <PolicyEdit
      item={policy}
      onUpdate={(item) => {
        window.access.policy = AccessP.Policy.toJson(item);
      }}
    />
  </div>
);

const ChunkError = () => {
  throw new TypeError(
    "error loading dynamically imported module: /assets/List-CSEuO6kJ.js",
  );
};

const drawer = (info: { infoItemsGetter?: any }) => ({
  element: <ResourceItemDrawer />,
  children: [
    {
      path: "",
      element: <ResourceItemMainPage infoComponent={info.infoItemsGetter} />,
    },
  ],
});

const router = createMemoryRouter(
  [
    {
      path: "/",
      element: <Outlet />,
      children: [
        { path: "policy-edit", element: <PolicyEditPage /> },
        { path: "access/requests", element: <div /> },
        { path: "access/catalogs", element: <div /> },
        {
          path: "access/catalogs/create",
          element: (
            <ResourceCreateRoute
              api="access"
              kind="Catalog"
              specComponent={catalogInfo.Item!.Edit!}
            />
          ),
        },
        { path: "access/requests/:name", ...drawer(requestInfo) },
        { path: "access/catalogs/:name", ...drawer(catalogInfo) },
        {
          path: "chunk-error",
          element: <ChunkError />,
          errorElement: <RouteErrorBoundary />,
        },
      ],
    },
  ],
  { initialEntries: [params.get("path") ?? "/policy-edit"] },
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
          <RouterProvider router={router} />
        </QueryClientProvider>
      </MantineProvider>
    </Provider>
  </React.StrictMode>,
);
