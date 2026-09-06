import { Outlet, RouteObject } from "react-router-dom";
import * as React from "react";

import ResourceEditPage from "@/components/ResourceLayout/ResourceEdit";
import ResourceListPage from "@/components/ResourceLayout/ResourceList";
import { getResourcePathFromAPIKind } from "@/utils/pb";
import { match } from "ts-pattern";
import { ResourceComponentInfo } from "../utils/types";

import authenticatorRouter from "./Authenticator/router";
import clusterConfigRouter from "./ClusterConfig/router";
import credentialRouter from "./Credential/router";
import deviceRouter from "./Device/router";
import gatewayRouter from "./Gateway/router";
import groupRouter from "./Group/router";
import identityProviderRouter from "./IdentityProvider/router";
import namespaceRouter from "./Namespace/router";
import policyRouter from "./Policy/router";
import regionRouter from "./Region/router";
import secretRouter from "./Secret/router";
import serviceRouter from "./Service/router";
import sessionRouter from "./Session/router";
import userRouter from "./User/router";

import MainPage from "../visibility/Main";
import Summary from "./Summary";

import ResourceItemActionsPage from "@/components/ResourceLayout/ResourceActions";
import ResourceCreateRoute from "@/components/ResourceLayout/ResourceCreateRoute";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";
const ResourceItemAccessLogsPage = React.lazy(
  () => import("@/components/ResourceLayout/ResourceAccessLogs"),
);
const ResourceItemAuditLogsPage = React.lazy(
  () => import("@/components/ResourceLayout/ResourceAuditLogs"),
);
const ResourceItemAuthenticationLogsPage = React.lazy(
  () => import("@/components/ResourceLayout/ResourceAuthenticationLogs"),
);
const ResourceItemLLMPage = React.lazy(
  () => import("@/components/ResourceLayout/ResourceLLM"),
);
const ServiceMetricsPage = React.lazy(
  () => import("@/components/ResourceLayout/ServiceMetricsPage"),
);
const ServiceSSHPage = React.lazy(
  () => import("@/components/ResourceLayout/ServiceSSHPage"),
);

const LazyPage = (props: { children: React.ReactNode }) => (
  <React.Suspense
    fallback={
      <div className="flex min-h-[40vh] items-center justify-center text-sm font-semibold text-slate-500">
        Loading…
      </div>
    }
  >
    {props.children}
  </React.Suspense>
);

export const resourceList = [
  serviceRouter,
  userRouter,
  groupRouter,
  sessionRouter,
  namespaceRouter,
  credentialRouter,
  identityProviderRouter,
  policyRouter,
  regionRouter,
  gatewayRouter,
  secretRouter,
  deviceRouter,
  authenticatorRouter,
];

export default (): RouteObject => {
  const ret = {
    path: "core",
    element: (
      <>
        <Outlet />
      </>
    ),
    children: resourceList
      .map((x) => {
        return getResourceChildrenRouter(x);
      })
      .concat([clusterConfigRouter()], {
        path: "summary",
        element: <Summary />,
      })
      .concat([clusterConfigRouter()], {
        path: "",
        element: <MainPage />,
      }),
  };

  return ret;
};

const getResourceChildrenRouter = (arg: ResourceComponentInfo): RouteObject => {
  let children = [
    {
      path: "",
      element: arg.Item.Main ? (
        <ResourceItemMainPage
          mainItemsGetter={arg.infoItemsGetter}
          mainAction={arg.Item.MainAction}
          unDeletable={arg.unDeletable}
          cloneable={arg.cloneable}
        />
      ) : null,
    },
    {
      path: "edit",
      element: arg.Item.Edit ? (
        <ResourceEditPage specComponent={arg.Item.Edit} />
      ) : null,
    },
    {
      path: "actions",
      element: <ResourceItemActionsPage />,
    },
  ];

  if (
    match(arg.Kind)
      .with(
        "User",
        "Session",
        "Device",
        "Service",
        "Namespace",
        "Region",
        "Policy",
        () => true,
      )
      .otherwise(() => false)
  ) {
    children.push({
      path: "accesslogs",
      element: (
        <LazyPage>
          <ResourceItemAccessLogsPage />
        </LazyPage>
      ),
    });
  }

  if (
    match(arg.Kind)
      .with(
        "User",
        "Session",
        "Device",
        "Service",
        "Namespace",
        "Region",
        () => true,
      )
      .otherwise(() => false)
  ) {
    children.push({
      path: "llm",
      element: (
        <LazyPage>
          <ResourceItemLLMPage />
        </LazyPage>
      ),
    });
  }

  if (
    match(arg.Kind)
      .with("Service", "Namespace", () => true)
      .otherwise(() => false)
  ) {
    children.push({
      path: "metrics",
      element: (
        <LazyPage>
          <ServiceMetricsPage />
        </LazyPage>
      ),
    });
  }

  if (arg.API === "core" && arg.Kind === "Service") {
    children.push({
      path: "ssh",
      element: (
        <LazyPage>
          <ServiceSSHPage />
        </LazyPage>
      ),
    });
  }

  if (
    match(arg.Kind)
      .with(
        "User",
        "Session",
        "IdentityProvider",
        "Credential",
        "Authenticator",
        () => true,
      )
      .otherwise(() => false)
  ) {
    children.push({
      path: "authenticationlogs",
      element: (
        <LazyPage>
          <ResourceItemAuthenticationLogsPage />
        </LazyPage>
      ),
    });
  }

  children.push({
    path: "auditlogs",
    element: (
      <LazyPage>
        <ResourceItemAuditLogsPage />
      </LazyPage>
    ),
  });

  return {
    path: getResourcePathFromAPIKind({ api: arg.API, kind: arg.Kind }),
    element: <ResourceListPage info={arg} />,

    children: [
      {
        path: "create",
        element: arg.Item.Edit ? (
          <ResourceCreateRoute
            api={arg.API}
            kind={arg.Kind}
            specComponent={arg.Item.Edit}
            createResource={arg.Item.createResource}
          />
        ) : null,
      },
      {
        path: ":name",
        element: <ResourceItemDrawer />,
        children,
      },
    ],
  };
};
