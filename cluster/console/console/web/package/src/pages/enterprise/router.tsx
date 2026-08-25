import { Outlet, RouteObject } from "react-router-dom";
import * as React from "react";

import ResourceEditPage from "@/components/ResourceLayout/ResourceEdit";
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
import ResourceCreateRoute from "@/components/ResourceLayout/ResourceCreateRoute";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";

import MainPage from "./index";

const ResourceItemAuditLogsPage = React.lazy(
  () => import("@/components/ResourceLayout/ResourceAuditLogs"),
);
const PolicyTesterPage = React.lazy(() => import("./PolicyTester"));

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
        <ResourceEditPage
          specComponent={arg.Item.Edit}
          readOnly={arg.readOnlyEdit}
        />
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
