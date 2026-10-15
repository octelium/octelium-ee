import { Navigate, RouteObject } from "react-router-dom";

import { NotFoundPage, RouteError } from "@/components/RouteError";

import Settings from "./Settings";

import Root from "./index";
import routerReviewer from "./reviewer/routes";
import routerUser from "./user/routes";

export default (): RouteObject => {
  return {
    path: "/",
    element: <Root />,
    children: [
      {
        errorElement: <RouteError />,
        children: [
          {
            index: true,
            element: <Navigate to="/user/requests" replace />,
          },
          {
            path: "settings",
            element: <Settings />,
          },
          routerUser(),
          routerReviewer(),
          {
            path: "*",
            element: <NotFoundPage />,
          },
        ],
      },
    ],
  };
};
