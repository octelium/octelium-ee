import { Outlet, RouteObject } from "react-router-dom";

import ResourceEditPage from "@/components/ResourceLayout/ResourceEdit";
import ResourceListPage from "@/components/ResourceLayout/ResourceList";
import { getResourcePathFromAPIKind } from "@/utils/pb";
import { ResourceComponentInfo } from "../utils/types";

import catalogRouter from "./Catalog/router";
import policyRouter from "./Policy/router";
import requestRouter from "./Request/router";
import reviewRouter from "./Review/router";

import ResourceItemActionsPage from "@/components/ResourceLayout/ResourceActions";
import ResourceItemAuditLogsPage from "@/components/ResourceLayout/ResourceAuditLogs";
import ResourceCreatePage from "@/components/ResourceLayout/ResourceCreate";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";

export const resourceList = [
  policyRouter,
  requestRouter,
  reviewRouter,
  catalogRouter,
];

export default (): RouteObject => {
  const ret = {
    path: "access",
    element: (
      <>
        <Outlet />
      </>
    ),
    children: resourceList.map((x) => {
      return getResourceChildrenRouter(x);
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

  children.push({
    path: "auditlogs",
    element: <ResourceItemAuditLogsPage />,
  });

  return {
    path: getResourcePathFromAPIKind({ api: arg.API, kind: arg.Kind }),
    element: <ResourceListPage info={arg} />,

    children: [
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
        element: <ResourceItemDrawer />,
        children,
      },
    ],
  };
};
