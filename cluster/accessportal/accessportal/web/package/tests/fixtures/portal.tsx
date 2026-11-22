import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@mantine/notifications/styles.layer.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import { QueryClientProvider } from "@tanstack/react-query";
import { Provider } from "react-redux";
import { createBrowserRouter, createMemoryRouter } from "react-router";
import { RouterProvider } from "react-router/dom";

import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import * as MetaP from "@/apis/metav1/metav1";
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
const calls: { method: string; request: unknown }[] = [];
window.calls = calls;

declare global {
  interface Window {
    calls: typeof calls;
    portalRouter: ReturnType<typeof createMemoryRouter>;
  }
}

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
  Object.getPrototypeOf(client)[method] = async (request: unknown) => {
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
const hours = (value: number) =>
  MetaP.Duration.create({ type: { oneofKind: "hours", hours: value } });
const S = AccessP.Request_Status_State_Status;

const step = (name: string) =>
  AccessP.Request_Status_Workflow_Step.create({
    name,
    approvalRequirement: AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ANY,
    reviewers: [
      { type: AccessP.Request_Status_Workflow_Step_Reviewer_Type.USER, displayName: "carol" },
    ],
  });

const fullLevel = AccessP.Catalog_Spec_AccessLevel.create({
  name: "full",
  displayName: "Full access",
  type: { oneofKind: "full", full: {} },
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
    applied?: { reviewer: string; decision: AccessP.Review_Spec_Decision }[];
    askedInfo?: string;
  } = {},
) =>
  AccessP.Request.create({
    metadata: { name, uid: `uid-${name}`, createdAt: at(-180) },
    spec: {
      urgency: AccessP.Request_Spec_Urgency.NORMAL,
      duration: hours(extra.duration ?? 2),
      resource: {
        type: {
          oneofKind: "catalog",
          catalog: { catalogRef: { name: "storage" }, accessLevel: "full", serviceRefs: [] },
        },
      },
      subject: {
        type: { oneofKind: "userRef", userRef: { name: extra.subject ?? extra.requester ?? "alice" } },
      },
    },
    status: {
      state: { status, createdAt: at(-5) },
      userRef: { name: extra.requester ?? "alice" },
      specRevision: 1,
      resolvedScope: {
        catalogRef: { name: "storage" },
        catalogDisplayName: "Primary storage",
        accessLevel: fullLevel,
        serviceRefs: [{ name: "default.database" }],
      },
      approvalStartAt: at(-180),
      approvalEndAt: extra.approvalEndAgo !== undefined ? at(-extra.approvalEndAgo) : undefined,
      accessEndsAt: extra.accessEndsIn !== undefined ? at(extra.accessEndsIn) : undefined,
      effectiveDuration: extra.effectiveSeconds
        ? { type: { oneofKind: "seconds", seconds: extra.effectiveSeconds } }
        : undefined,
      lastStates: (extra.lastStates ?? []).map((value) => ({ status: value, createdAt: at(-60) })),
      workflow: extra.steps
        ? {
            effect: AccessP.Policy_Spec_Rule_Effect.REVIEW,
            steps: extra.steps.map(step),
          }
        : undefined,
      review: extra.steps
        ? {
            currentStep: extra.currentStep ?? 0,
            currentStepStartedAt: at(-30),
            steps: extra.steps.slice(0, (extra.currentStep ?? 0) + 1).map((name, index) => ({
              name,
              index,
              requiredApprovals: 1,
              approvals: 0,
              startedAt: at(-30),
            })),
            informationRequests: extra.askedInfo
              ? [{ userRef: { name: "carol" }, message: extra.askedInfo, requestedAt: at(-20) }]
              : [],
            lastSteps: (extra.applied ?? []).map((applied, index) => ({
              reviewRef: { name: `rv${index}.${name}` },
              setAt: at(-90),
              stepIndex: index,
              decision: applied.decision,
              userRef: { name: applied.reviewer },
              stepName: extra.steps![index],
            })),
          }
        : undefined,
    },
  });

const myRequests = [
  request("own-pending", S.PENDING, { steps: ["team-lead"] }),
  request("behalf-pending", S.PENDING, {
    subject: "bob",
    steps: ["team-lead"],
    askedInfo: "Which tables does bob need?",
  }),
  request("capped", S.APPROVED, {
    duration: 48,
    effectiveSeconds: 7200,
    approvalEndAgo: 60,
    accessEndsIn: 60,
  }),
  request("expired-after-approval", S.EXPIRED, {
    steps: ["team-lead"],
    lastStates: [S.APPROVED, S.PENDING],
    applied: [{ reviewer: "carol", decision: AccessP.Review_Spec_Decision.APPROVE }],
  }),
];

const queue = [
  request("queue-first", S.PENDING, { requester: "bob", steps: ["team-lead", "security"], currentStep: 1 }),
  request("queue-second", S.PENDING, { requester: "dave", steps: ["team-lead"] }),
];

const reviewCases = [
  request("queue-voted", S.PENDING, { requester: "dave", steps: ["team-lead"] }),
  request("queue-asked", S.PENDING, {
    requester: "dave",
    steps: ["team-lead"],
    askedInfo: "Which incident is this for?",
  }),
];

const behalfActive = request("behalf-active", S.APPROVED, {
  subject: "bob",
  approvalEndAgo: 30,
  accessEndsIn: 90,
});
if (behalfActive.spec?.resource?.type.oneofKind === "catalog") {
  behalfActive.spec.resource.type.catalog.serviceRefs = [{ name: "default.database" }];
}
behalfActive.status!.grant = { state: AccessP.Request_Status_Grant_State.ACTIVE, attempts: 0, lastError: "" };
behalfActive.status!.resolvedScope!.accessLevel = AccessP.Catalog_Spec_AccessLevel.create({
  name: "full",
  displayName: "Full access",
  type: { oneofKind: "full", full: {} },
  verification: { minAAL: 2 },
});

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
  AccessP.UserCatalog.create({
    metadata: { name: "team-namespace", displayName: "Team namespace" },
    accessLevels: [
      { name: "full", displayName: "Full access", type: AccessP.UserCatalog_AccessLevel_Type.FULL },
    ],
    totalService: 1,
    membership: AccessP.Catalog_Spec_Membership.DYNAMIC,
  }),
  AccessP.UserCatalog.create({
    metadata: { name: "storage", displayName: "Primary storage" },
    accessLevels: [
      { name: "full", displayName: "Full access", type: AccessP.UserCatalog_AccessLevel_Type.FULL },
      {
        name: "read-only",
        displayName: "Read only",
        description: "Read the data without changing it",
        type: AccessP.UserCatalog_AccessLevel_Type.RESTRICTED,
        defaultDuration: { type: { oneofKind: "minutes", minutes: 15 } },
        minAAL: 2,
      },
    ],
    totalService: 1,
  }),
];

const catalogServices: Record<string, UserP.Service[]> = {
  "team-namespace": [services[1]],
  storage: [services[0]],
};

const stub = <T,>(client: object, method: string, handler: (request: T) => unknown) => {
  Object.getPrototypeOf(client)[method] = async (request: T) => {
    calls.push({ method, request });
    return { response: handler(request) };
  };
};

const notFound = () => {
  throw new RpcError("not found", "NOT_FOUND");
};

stub(getUserClient(), "listCatalog", () => AccessP.UserCatalogList.create({ items: catalogs }));
stub(getUserClient(), "listCatalogService", (options: AccessP.ListUserCatalogServiceOptions) =>
  UserP.ServiceList.create({ items: catalogServices[options.catalogRef?.name ?? ""] ?? [] }),
);
stub(getUserClient(), "getInbox", () =>
  AccessP.Inbox.create({
    isReviewer: !params.has("notReviewer"),
    canRequestOnBehalf: true,
    totalNeedsMyDecision: queue.length,
  }),
);
stub(getUserClient(), "previewRequest", (value: AccessP.Request) =>
  AccessP.RequestPreview.create({
    outcome: AccessP.RequestPreview_Outcome.REVIEW,
    workflow: { effect: AccessP.Policy_Spec_Rule_Effect.REVIEW, steps: [step("team-lead")] },
    effectiveDuration: value.spec?.duration,
    maxDuration: hours(24),
    requirements:
      value.spec?.resource?.type.oneofKind === "catalog" &&
      value.spec.resource.type.catalog.catalogRef?.name === "storage"
        ? { ticket: { isRequired: true, label: "Change ticket", pattern: "" } }
        : undefined,
  }),
);
stub(getUserClient(), "endAccess", (options: AccessP.EndAccessRequest) => {
  const item = myRequests.find((value) => value.metadata?.name === options.requestRef?.name);
  return item ?? notFound();
});
stub(getUserClient(), "respondToRequest", (options: AccessP.RespondToRequestRequest) => {
  const item = myRequests.find((value) => value.metadata?.name === options.requestRef?.name);
  return item ?? notFound();
});
stub(getUserClient(), "listRequest", (options: AccessP.ListUserRequestOptions) => {
  const items = myRequests.filter(
    (item) => !options.state || item.status?.state?.status === options.state,
  );
  return AccessP.RequestList.create({
    items,
    listResponseMeta: { totalCount: items.length, itemsPerPage: 20 },
  });
});
stub(getUserClient(), "getRequest", (options: MetaP.GetOptions) =>
  [...myRequests, behalfActive].find((item) => item.metadata?.name === options.name) ??
  notFound(),
);
stub(getUserClient(), "getSubjectUser", () => notFound());
stub(getUserMainClient(), "getService", (options: MetaP.GetOptions) =>
  services.find((item) => item.metadata?.name === options.name) ?? notFound(),
);
stub(getReviewerClient(), "listRequest", (options: AccessP.ListReviewerRequestOptions) =>
  AccessP.RequestList.create({
    items: [queue[options.common?.page ?? 0]].filter(Boolean),
    listResponseMeta: { totalCount: 1, hasMore: (options.common?.page ?? 0) === 0 },
  }),
);
stub(getReviewerClient(), "getRequest", (options: MetaP.GetOptions) =>
  queue.find((item) => item.metadata?.name === options.name) ?? notFound(),
);
stub(getReviewerClient(), "getReviewCase", (options: AccessP.GetReviewCaseRequest) => {
  const item =
    [...queue, ...reviewCases].find((value) => value.metadata?.name === options.requestRef?.name) ??
    notFound();
  const name = item.metadata?.name;
  const asked = name === "queue-asked";
  const voted = name === "queue-voted";
  return AccessP.ReviewCase.create({
    request: item,
    allowedActions: [
      ...(asked ? [] : [AccessP.ReviewCase_Action.APPROVE]),
      AccessP.ReviewCase_Action.REJECT,
      AccessP.ReviewCase_Action.REQUEST_INFORMATION,
      ...(voted ? [AccessP.ReviewCase_Action.WITHDRAW] : []),
    ],
    myReview: voted
      ? AccessP.Review.create({
          metadata: { name: "rv.queue-voted" },
          spec: {
            decision: AccessP.Review_Spec_Decision.APPROVE,
            justification: "Approved for the migration window",
            duration: hours(1),
          },
          status: { requestRef: { name }, stepIndex: 0, specRevision: 1 },
        })
      : undefined,
    flags: { isFullAccess: true, isAwaitingRequester: asked },
  });
});
stub(getReviewerClient(), "listReview", () =>
  AccessP.ReviewList.create({ items: reviews, listResponseMeta: { totalCount: reviews.length } }),
);
stub(getReviewerClient(), "getReview", (options: MetaP.GetOptions) =>
  reviews.find((item) => item.metadata?.name === options.name) ?? notFound(),
);
stub(getReviewerClient(), "setReviewDecision", (options: AccessP.SetReviewDecisionRequest) =>
  AccessP.Review.create({ spec: { decision: options.decision } }),
);
stub(getReviewerClient(), "requestInformation", (options: AccessP.RequestInformationRequest) =>
  queue.find((item) => item.metadata?.name === options.requestRef?.name) ?? notFound(),
);
stub(getUserClient(), "updateRequest", (value: AccessP.Request) => {
  const index = myRequests.findIndex((item) => item.metadata?.name === value.metadata?.name);
  myRequests[index] = value;
  return value;
});
stub(getUserClient(), "createRequest", (value: AccessP.Request) => {
  if (params.has("failCreate")) throw new RpcError("Submission failed", "UNAVAILABLE");
  const created = AccessP.Request.clone(value);
  created.metadata!.name = "created-request";
  myRequests.push(created);
  return created;
});

Object.getPrototypeOf(getClientAuth()).logout = async () => {
  calls.push({ method: "logout", request: {} });
  throw new Error("Sign out unavailable");
};

const initialPath = params.get("path") ?? "/user/new";
if (params.has("browserHistory")) {
  window.history.replaceState(null, "", initialPath);
}
const root = routes();
if (params.has("chunkError")) {
  root.children![0].children!.unshift({
    path: "broken",
    lazy: async () => {
      throw new TypeError(
        "error loading dynamically imported module: /assets/List-CSEuO6kJ.js",
      );
    },
  });
}
const router = params.has("browserHistory")
  ? createBrowserRouter([root])
  : createMemoryRouter([root], { initialEntries: [initialPath] });
window.portalRouter = router;

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
