import UserLayout from "./layout";
import RequestDetail from "./Request/detail";
import Requests from "./Request/index";
import NewRequest from "./Request/new";

import { Navigate, RouteObject } from "react-router-dom";

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
