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
import * as UserP from "@/apis/userv1/userv1";
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
  [getUserClient(), "listCatalog", AccessP.CatalogList.create()],
  [getUserClient(), "listRequest", AccessP.RequestList.create()],
  [getReviewerClient(), "listRequest", AccessP.RequestList.create()],
  [getReviewerClient(), "listReview", AccessP.ReviewList.create()],
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
