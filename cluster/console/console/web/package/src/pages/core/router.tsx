import { Outlet, RouteObject } from "react-router-dom";

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

import ResourceItemAccessLogsPage from "@/components/ResourceLayout/ResourceAccessLogs";
import ResourceItemActionsPage from "@/components/ResourceLayout/ResourceActions";
import ResourceItemAuditLogsPage from "@/components/ResourceLayout/ResourceAuditLogs";
import ResourceItemAuthenticationLogsPage from "@/components/ResourceLayout/ResourceAuthenticationLogs";
import ResourceCreatePage from "@/components/ResourceLayout/ResourceCreate";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import ResourceItemPage from "@/components/ResourceLayout/ResourceItemPage";

const resourceList = [
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
        <ResourceItemMainPage mainComponent={arg.Item.Main} />
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
        () => true
      )
      .otherwise(() => false)
  ) {
    children.push({
      path: "accesslogs",
      element: <ResourceItemAccessLogsPage />,
    });
  }

  if (
    match(arg.Kind)
      .with("User", "Session", "IdentityProvider", "Authenticator", () => true)
      .otherwise(() => false)
  ) {
    children.push({
      path: "authenticationlogs",
      element: <ResourceItemAuthenticationLogsPage />,
    });
  }

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
