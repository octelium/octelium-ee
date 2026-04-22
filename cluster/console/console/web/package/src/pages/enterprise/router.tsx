import { Outlet, RouteObject } from "react-router-dom";

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
import ResourceItemAuditLogsPage from "@/components/ResourceLayout/ResourceAuditLogs";
import ResourceCreatePage from "@/components/ResourceLayout/ResourceCreate";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import ResourceItemPage from "@/components/ResourceLayout/ResourceItemPage";

import PolicyTesterPage from "./PolicyTester";

const resourceList = [
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
        { path: `policytester`, element: <PolicyTesterPage /> },
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
          mainComponent={arg.Item.Main}
          mainItemsGetter={arg.infoItemsGetter}
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

  children.push({
    path: "auditlogs",
    element: <ResourceItemAuditLogsPage />,
  });

  return {
    path: getResourcePathFromAPIKind({ api: arg.API, kind: arg.Kind }),
    element: (
      <>
        <Outlet />
      </>
    ),

    children: [
      {
        path: "",
        element: <ResourceListPage info={arg} />,
      },

      {
        path: "create",
        element: arg.Item.Edit ? (
          <ResourceCreatePage
            specComponent={arg.Item.Edit}
            createResource={arg.Item.createResource}
          />
        ) : null,
      },
      {
        path: ":name",
        element: <ResourceItemPage />,
        children,
      },
    ],
  };
};
