import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@mantine/dates/styles.layer.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import { QueryClientProvider } from "@tanstack/react-query";
import { Provider } from "react-redux";
import { createMemoryRouter, RouterProvider } from "react-router-dom";

import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as UserP from "@/apis/userv1/userv1";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import routes from "@/pages/router";
import store from "@/store";
import { queryClient } from "@/utils";
import {
  getClientAuth,
  getReviewerClient,
  getUserClient,
  getUserMainClient,
} from "@/utils/client";
import {
  colorSchemeManager,
  DEFAULT_COLOR_SCHEME,
} from "@/utils/theme/colorScheme";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import theme from "@/utils/theme/mantine";

const params = new URLSearchParams(window.location.search);
const calls: { method: string; request: any }[] = [];
(window as any).calls = calls;

const services = [
  UserP.Service.create({
    metadata: { name: "default.database", displayName: "Database" },
    status: { namespace: "default" },
    spec: { type: UserP.Service_Spec_Type.TCP, port: 5432 },
  }),
  UserP.Service.create({
    metadata: { name: "this-namespace.dashboard", displayName: "Dashboard" },
    status: { namespace: "this-namespace" },
    spec: { type: UserP.Service_Spec_Type.HTTP, port: 443, isTLS: true },
  }),
];

for (const [client, method, response] of [
  [
    getUserMainClient(),
    "getStatus",
    UserP.GetStatusResponse.create({
      domain: "example.com",
      user: {
        metadata: { name: "alice", displayName: "Alice Morgan" },
        spec: { email: "alice@example.com" },
      },
    }),
  ],
  [
    getUserMainClient(),
    "listNamespace",
    UserP.NamespaceList.create({
      items: ["default", "this-namespace"].map((name) => ({ metadata: { name } })),
    }),
  ],
] as const) {
  Object.getPrototypeOf(client)[method] = async (request: any) => {
    calls.push({ method, request });
    return { response };
  };
}

Object.getPrototypeOf(getUserMainClient()).listService = async (
  request: UserP.ListServiceOptions,
) => {
  calls.push({ method: "listService", request });
  return {
    response: UserP.ServiceList.create({
      items: services.filter(
        (service) => !request.namespace || service.status?.namespace === request.namespace,
      ),
    }),
  };
};

const at = (minutes: number) => Timestamp.fromDate(new Date(Date.now() + minutes * 60_000));
const hours = (value: number) => ({ type: { oneofKind: "hours", hours: value } as any });
const S = AccessP.Request_Status_State_Status;

const step = (name: string) =>
  AccessP.Policy_Spec_Rule_Action_Review_Step.create({
    name,
    approvalRequirement: AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ANY,
    reviewers: [{ type: { oneofKind: "user", user: { userRef: { name: "carol" } } } }],
  });

const request = (
  name: string,
  status: AccessP.Request_Status_State_Status,
  extra: {
    requester?: string;
    subject?: string;
    steps?: string[];
    currentStep?: number;
    duration?: number;
    effectiveSeconds?: number;
    accessEndsIn?: number;
    approvalEndAgo?: number;
    lastStates?: AccessP.Request_Status_State_Status[];
  } = {},
) =>
  AccessP.Request.create({
    metadata: { name, uid: `uid-${name}`, createdAt: at(-180) },
    spec: {
      urgency: AccessP.Request_Spec_Urgency.NORMAL,
      duration: hours(extra.duration ?? 2),
      resource: { type: { oneofKind: "serviceRef", serviceRef: { name: "default.database" } } },
      subject: {
        type: { oneofKind: "userRef", userRef: { name: extra.subject ?? extra.requester ?? "alice" } },
      },
    },
    status: {
      state: { status, createdAt: at(-5) },
      userRef: { name: extra.requester ?? "alice" },
      approvalStartAt: at(-180),
      approvalEndAt: extra.approvalEndAgo !== undefined ? at(-extra.approvalEndAgo) : undefined,
      accessEndsAt: extra.accessEndsIn !== undefined ? at(extra.accessEndsIn) : undefined,
      effectiveDuration: extra.effectiveSeconds
        ? { type: { oneofKind: "seconds", seconds: extra.effectiveSeconds } as any }
        : undefined,
      lastStates: (extra.lastStates ?? []).map((value) => ({ status: value, createdAt: at(-60) })),
      rule: extra.steps
        ? {
            name: "review",
            action: { type: { oneofKind: "review", review: { steps: extra.steps.map(step) } } },
            authorization: { policies: ["read-only"] },
          }
        : undefined,
      review: extra.steps
        ? { currentStep: extra.currentStep ?? 0, currentStepStartedAt: at(-30) }
        : undefined,
    },
  });

const myRequests = [
  request("own-pending", S.PENDING, { steps: ["team-lead"] }),
  request("behalf-pending", S.PENDING, { subject: "bob", steps: ["team-lead"] }),
  request("capped", S.APPROVED, {
    duration: 48,
    effectiveSeconds: 7200,
    approvalEndAgo: 60,
    accessEndsIn: 60,
  }),
  request("expired-after-approval", S.EXPIRED, {
    steps: ["team-lead"],
    lastStates: [S.APPROVED, S.PENDING],
  }),
];

const queue = [
  request("queue-first", S.PENDING, { requester: "bob", steps: ["team-lead", "security"], currentStep: 1 }),
  request("queue-second", S.PENDING, { requester: "dave", steps: ["team-lead"] }),
];

const reviews = [
  AccessP.Review.create({
    metadata: { name: "rv1.queue-second", uid: "review-unset" },
    spec: { decision: AccessP.Review_Spec_Decision.UNSET },
    status: { requestRef: { name: "queue-second" }, stepIndex: 0, stepName: "team-lead" },
  }),
  AccessP.Review.create({
    metadata: { name: "rv2.queue-first", uid: "review-applied" },
    spec: { decision: AccessP.Review_Spec_Decision.APPROVE, justification: "ok" },
    status: { requestRef: { name: "queue-first" }, stepIndex: 0, stepName: "team-lead" },
  }),
];

const catalogs = [
  AccessP.Catalog.create({
    metadata: { name: "team-namespace", displayName: "Team namespace" },
    spec: { resourceCollection: { service: { namespaces: ["this-namespace"] } } },
  }),
];

const stub = (client: object, method: string, handler: (request: any) => any) => {
  Object.getPrototypeOf(client)[method] = async (request: any) => {
    calls.push({ method, request });
    return { response: handler(request) };
  };
};

const notFound = () => {
  throw new RpcError("not found", "NOT_FOUND");
};

stub(getUserClient(), "listCatalog", () => AccessP.CatalogList.create({ items: catalogs }));
stub(getUserClient(), "listRequest", () =>
  AccessP.RequestList.create({
    items: myRequests,
    listResponseMeta: { totalCount: myRequests.length, itemsPerPage: 20 },
  }),
);
stub(getUserClient(), "getRequest", (options) =>
  myRequests.find((item) => item.metadata?.name === options.name) ?? notFound(),
);
stub(getUserClient(), "getSubjectUser", () => notFound());
stub(getUserMainClient(), "getService", (options) =>
  services.find((item) => item.metadata?.name === options.name) ?? notFound(),
);
stub(getReviewerClient(), "listRequest", (options) =>
  AccessP.RequestList.create({
    items: [queue[options.common?.page ?? 0]].filter(Boolean),
    listResponseMeta: { totalCount: 1, hasMore: (options.common?.page ?? 0) === 0 },
  }),
);
stub(getReviewerClient(), "getRequest", (options) =>
  queue.find((item) => item.metadata?.name === options.name) ?? notFound(),
);
stub(getReviewerClient(), "listReview", () =>
  AccessP.ReviewList.create({ items: reviews, listResponseMeta: { totalCount: reviews.length } }),
);
stub(getReviewerClient(), "getReview", (options) =>
  reviews.find((item) => item.metadata?.name === options.name) ?? notFound(),
);
stub(getReviewerClient(), "setReviewDecision", (options) =>
  AccessP.Review.create({ spec: { decision: options.decision } }),
);
stub(getReviewerClient(), "createReview", (review) => review);
stub(getReviewerClient(), "updateReview", (review) => review);

Object.getPrototypeOf(getClientAuth()).logout = async () => {
  calls.push({ method: "logout", request: {} });
  throw new Error("Sign out unavailable");
};

const router = createMemoryRouter([routes()], {
  initialEntries: [params.get("path") ?? "/user/new"],
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Provider store={store}>
      <MantineProvider
        theme={theme}
        colorSchemeManager={colorSchemeManager}
        defaultColorScheme={DEFAULT_COLOR_SCHEME}
        cssVariablesResolver={cssVariablesResolver}
      >
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </MantineProvider>
    </Provider>
  </React.StrictMode>,
);
