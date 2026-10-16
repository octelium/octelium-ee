import UserLayout from "./layout";
import Requests from "./Request/index";

import * as React from "react";
import { Navigate, RouteObject } from "react-router-dom";

const RequestDetail = React.lazy(() => import("./Request/detail"));
const NewRequest = React.lazy(() => import("./Request/new"));

export const userRoutes = (): RouteObject => ({
  path: "/user",
  element: <UserLayout />,
  children: [
    { index: true, element: <Navigate to="/user/requests" replace /> },
    { path: "requests", element: <Requests /> },
    { path: "requests/:name", element: <RequestDetail /> },
    { path: "new", element: <NewRequest /> },
  ],
});

export default userRoutes;
