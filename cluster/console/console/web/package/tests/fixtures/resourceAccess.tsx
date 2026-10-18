import "@/index.css";
import "@mantine/core/styles.layer.css";
import * as AccessP from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import accessRouter from "@/pages/access/router";
import coreRouter from "@/pages/core/router";
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
import {
  createMemoryRouter,
  Outlet,
  RouterProvider,
  useLocation,
  useNavigate,
} from "react-router-dom";

type ListRequest = {
  common?: { page: number; itemsPerPage: number; query: string };
  name?: string;
  userRef?: ObjectReference;
  subjectUserRef?: ObjectReference;
  reviewerRef?: ObjectReference;
  serviceRef?: ObjectReference;
  catalogRef?: ObjectReference;
  policyRef?: ObjectReference;
  requestRef?: ObjectReference;
  integrationRef?: ObjectReference;
  state?: number;
  urgency?: number;
  decision?: number;
};

const params = new URLSearchParams(window.location.search);
const calls: { client: string; method: string; request: ListRequest }[] = [];
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});
const state = {
  calls,
  failAccess: false,
  navigate: (to: string | number) => {
    void to;
  },
};
declare global {
  interface Window {
    resourceAccess: typeof state;
  }
}
window.resourceAccess = state;

const ref = (apiVersion: string, kind: string, name: string) =>
  ObjectReference.create({ apiVersion, kind, name, uid: `uid-${name}` });
const createdAt = Timestamp.fromDate(new Date("2026-10-17T12:00:00Z"));
const metadata = (name: string) => ({ name, uid: `uid-${name}`, createdAt });

const requestItem = (name: string, index = 0, user = "john") =>
  AccessP.Request.create({
    apiVersion: "access/v1",
    kind: "Request",
    metadata: metadata(name),
    spec: {
      urgency:
        index % 2 === 0
          ? AccessP.Request_Spec_Urgency.HIGH
          : AccessP.Request_Spec_Urgency.NORMAL,
      subject: {
        type: { oneofKind: "userRef", userRef: ref("core/v1", "User", user) },
      },
      resource: {
        type: {
          oneofKind: "serviceRef",
          serviceRef: ref("core/v1", "Service", "billing.production"),
        },
      },
    },
    status: {
      userRef: ref("core/v1", "User", user),
      policyRef: ref("access/v1", "Policy", "production-access"),
      state: {
        status:
          index % 2 === 0
            ? AccessP.Request_Status_State_Status.PENDING
            : AccessP.Request_Status_State_Status.APPROVED,
      },
    },
  });

const reviewItem = (name: string, index = 0) =>
  AccessP.Review.create({
    apiVersion: "access/v1",
    kind: "Review",
    metadata: metadata(name),
    spec: {
      decision:
        index % 2 === 0
          ? AccessP.Review_Spec_Decision.APPROVE
          : AccessP.Review_Spec_Decision.REJECT,
    },
    status: {
      userRef: ref("core/v1", "User", "john"),
      requestRef: ref("access/v1", "Request", "request-01"),
      stepName: "security",
    },
  });

const policy = AccessP.Policy.create({
  apiVersion: "access/v1",
  kind: "Policy",
  metadata: metadata("production-access"),
  spec: {},
});
const catalog = AccessP.Catalog.create({
  apiVersion: "access/v1",
  kind: "Catalog",
  metadata: metadata("production-support"),
  spec: {},
});
const integration = AccessP.Integration.create({
  apiVersion: "access/v1",
  kind: "Integration",
  metadata: metadata("slack"),
  spec: {},
  status: {},
});
const identity = AccessP.IntegrationIdentity.create({
  apiVersion: "access/v1",
  kind: "IntegrationIdentity",
  metadata: metadata("linked-john"),
  status: {
    integrationRef: ref("access/v1", "Integration", "slack"),
    userRef: ref("core/v1", "User", "john"),
    externalID: "U123",
    externalUsername: "john.slack",
    source: AccessP.IntegrationIdentity_Status_Source.EMAIL_DISCOVERY,
  },
});
const delivery = AccessP.IntegrationBinding.create({
  apiVersion: "access/v1",
  kind: "IntegrationBinding",
  metadata: metadata("delivery-01"),
  status: {
    integrationRef: ref("access/v1", "Integration", "slack"),
    requestRef: ref("access/v1", "Request", "request-01"),
    userRef: ref("core/v1", "User", "john"),
    purpose: AccessP.IntegrationBinding_Status_Purpose.REVIEW_SURFACE,
    state: AccessP.IntegrationBinding_Status_State.READY,
    audience: AccessP.Policy_Spec_Rule_Surface_Destination_Audience.REVIEWERS,
  },
});

const paginated = <T,>(items: T[], request: ListRequest) => {
  const page = request.common?.page ?? 0;
  const itemsPerPage = request.common?.itemsPerPage || 25;
  return {
    items: items.slice(page * itemsPerPage, (page + 1) * itemsPerPage),
    listResponseMeta: { page, itemsPerPage, totalCount: items.length },
  };
};

const handlers: Record<string, (request: ListRequest) => unknown> = {
  getRequest: (request) => requestItem(request.name || "request-01"),
  getReview: (request) => reviewItem(request.name || "review-01"),
  getPolicy: () => policy,
  getCatalog: () => catalog,
  getIntegration: () => integration,
  listRequest: (request) => {
    const items = Array.from({ length: 28 }, (_, index) =>
      requestItem(
        `request-${`${index + 1}`.padStart(2, "0")}`,
        index,
        request.userRef?.name || "john",
      ),
    )
      .filter(
        (item) =>
          !request.state || item.status?.state?.status === request.state,
      )
      .filter(
        (item) => !request.urgency || item.spec?.urgency === request.urgency,
      )
      .filter(
        (item) =>
          !request.common?.query ||
          item.metadata?.name.includes(request.common.query),
      );
    return AccessP.RequestList.create(paginated(items, request));
  },
  listReview: (request) =>
    AccessP.ReviewList.create(
      paginated(
        Array.from({ length: 28 }, (_, index) =>
          reviewItem(`review-${`${index + 1}`.padStart(2, "0")}`, index),
        ).filter(
          (item) =>
            !request.decision || item.spec?.decision === request.decision,
        ),
        request,
      ),
    ),
  listPolicy: (request) =>
    AccessP.PolicyList.create(paginated([policy], request)),
  listCatalog: (request) =>
    AccessP.CatalogList.create(paginated([catalog], request)),
  listIntegrationIdentity: (request) =>
    AccessP.IntegrationIdentityList.create(paginated([identity], request)),
  listIntegrationBinding: (request) =>
    AccessP.IntegrationBindingList.create(paginated([delivery], request)),
};

for (const [clientName, client] of [
  ["access", clients.getClientAccess()],
  ["visibilityAccess", clients.getClientVisibilityAccess()],
  ["core", clients.getClientCore()],
  ["visibilityCore", clients.getClientVisibilityCore()],
] as const) {
  for (const method of client.methods) {
    Object.getPrototypeOf(client)[method.localName] = async (
      request: ListRequest,
    ) => {
      calls.push({ client: clientName, method: method.localName, request });
      if (
        state.failAccess &&
        clientName === "visibilityAccess" &&
        request.userRef
      ) {
        throw new RpcError("The access service is unavailable", "UNAVAILABLE");
      }
      if (clientName === "core" && method.localName.startsWith("get")) {
        return {
          response: method.O.create({
            apiVersion: "core/v1",
            kind: method.localName.slice(3),
            metadata: metadata(request.name || "john"),
            spec: {},
            status: {},
          }),
        };
      }
      const handler = handlers[method.localName];
      return {
        response:
          handler && clientName !== "visibilityCore"
            ? handler(request)
            : method.O.create(),
      };
    };
  }
}

const Navigation = () => {
  const location = useLocation();
  const navigate = useNavigate();
  React.useEffect(() => {
    state.navigate = (to) =>
      typeof to === "number" ? navigate(to) : navigate(to);
  }, [navigate]);
  return (
    <>
      <output data-route>
        {location.pathname}
        {location.search}
      </output>
      <Outlet />
    </>
  );
};

const initialURL = new URL(
  params.get("path") || "/core/users/john/access/requests",
  "http://console.test",
);
const router = createMemoryRouter(
  [
    {
      path: "/",
      element: <Navigation />,
      children: [coreRouter(), accessRouter()],
    },
  ],
  {
    initialEntries: [
      {
        pathname: initialURL.pathname,
        search: initialURL.search,
        state: params.get("returnTo")
          ? { returnTo: params.get("returnTo") }
          : null,
      },
    ],
  },
);

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
