import { Outlet, RouteObject } from "react-router-dom";
import * as React from "react";

import { PageLoading } from "@/components/Loading";
import ResourceListPage from "@/components/ResourceLayout/ResourceList";
import { getResourcePathFromAPIKind } from "@/utils/pb";
import { ResourceComponentInfo } from "../utils/types";

import catalogRouter from "./Catalog/router";
import clusterConfigRouter from "./ClusterConfig/router";
import integrationRouter from "./Integration/router";
import policyRouter from "./Policy/router";
import requestRouter from "./Request/router";
import reviewRouter from "./Review/router";
import secretRouter from "./Secret/router";

import ResourceItemActionsPage from "@/components/ResourceLayout/ResourceActions";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import { getResourceAccessSections } from "@/components/ResourceLayout/resourceAccess";
import MainPage from "./index";

const ResourceEditPage = React.lazy(
  () => import("@/components/ResourceLayout/ResourceEdit"),
);
const ResourceAccess = React.lazy(
  () => import("@/components/ResourceLayout/ResourceAccess"),
);
const ResourceCreateRoute = React.lazy(
  () => import("@/components/ResourceLayout/ResourceCreateRoute"),
);
const ResourceItemDrawer = React.lazy(
  () => import("@/components/ResourceLayout/ResourceItemDrawer"),
);
const ResourceItemAuditLogsPage = React.lazy(
  () => import("@/components/ResourceLayout/ResourceAuditLogs"),
);

const LazyPage = (props: { children: React.ReactNode }) => (
  <React.Suspense fallback={<PageLoading />}>{props.children}</React.Suspense>
);

export const resourceList = [
  policyRouter,
  requestRouter,
  reviewRouter,
  catalogRouter,
  integrationRouter,
  secretRouter,
];

export default (): RouteObject => {
  const ret = {
    path: "access",
    element: (
      <>
        <Outlet />
      </>
    ),
    children: resourceList
      .map((x) => getResourceChildrenRouter(x))
      .concat([clusterConfigRouter(), { path: "", element: <MainPage /> }]),
  };

  return ret;
};

const getResourceChildrenRouter = (arg: ResourceComponentInfo): RouteObject => {
  let children = [
    {
      path: "",
      element: arg.Item.hasMain ? (
        <ResourceItemMainPage
          infoComponent={arg.infoItemsGetter}
          mainAction={arg.Item.MainAction}
          unDeletable={arg.unDeletable}
          cloneable={arg.cloneable}
        />
      ) : null,
    },
    {
      path: "edit",
      element: arg.Item.Edit ? (
        <LazyPage>
          <ResourceEditPage
            specComponent={arg.Item.Edit}
            readOnly={arg.readOnlyEdit}
          />
        </LazyPage>
      ) : null,
    },
    {
      path: "actions",
      element: <ResourceItemActionsPage />,
    },
  ];

  if (getResourceAccessSections(`${arg.API}/v1`, arg.Kind).length > 0) {
    children.push({
      path: "access/:section?",
      element: (
        <LazyPage>
          <ResourceAccess />
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
          <LazyPage>
            <ResourceCreateRoute
              api={arg.API}
              kind={arg.Kind}
              specComponent={arg.Item.Edit}
              createResource={arg.Item.createResource}
            />
          </LazyPage>
        ) : null,
      },
      {
        path: ":name",
        element: (
          <LazyPage>
            <ResourceItemDrawer />
          </LazyPage>
        ),
        children,
      },
    ],
  };
};
