import { Outlet, RouteObject } from "react-router-dom";
import * as React from "react";

import { PageLoading } from "@/components/Loading";
import ResourceListPage from "@/components/ResourceLayout/ResourceList";
import { getResourcePathFromAPIKind } from "@/utils/pb";
import { ResourceComponentInfo } from "../utils/types";

import certificateRouter from "./Certificate/router";
import certificateIssuerRouter from "./CertificateIssuer/router";
import clusterConfigRouter from "./ClusterConfig/router";
import collectorExporterRouter from "./CollectorExporter/router";
import directoryProviderRouter from "./DirectoryProvider/router";
import dnsProviderRouter from "./DNSProvider/router";
import secretRouter from "./Secret/router";
import secretStoreRouter from "./SecretStore/router";

import ResourceItemActionsPage from "@/components/ResourceLayout/ResourceActions";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";

import MainPage from "./index";

const ResourceEditPage = React.lazy(
  () => import("@/components/ResourceLayout/ResourceEdit"),
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
const PolicyTesterPage = React.lazy(() => import("./PolicyTester"));

const LazyPage = (props: { children: React.ReactNode }) => (
  <React.Suspense fallback={<PageLoading />}>{props.children}</React.Suspense>
);

export const resourceList = [
  secretRouter,
  dnsProviderRouter,
  collectorExporterRouter,
  certificateRouter,
  certificateIssuerRouter,
  directoryProviderRouter,
  secretStoreRouter,
];

export default (): RouteObject => {
  const ret = {
    path: "enterprise",
    element: (
      <>
        <Outlet />
      </>
    ),
    children: resourceList
      .map((x) => {
        return getResourceChildrenRouter(x);
      })
      .concat([
        clusterConfigRouter(),
        {
          path: `policytester`,
          element: (
            <LazyPage>
              <PolicyTesterPage />
            </LazyPage>
          ),
        },
        { path: "", element: <MainPage /> },
      ]),
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
