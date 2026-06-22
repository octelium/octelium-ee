import UserLayout from "./layout";
import RequestDetail from "./Request/detail";
import Requests from "./Request/index";
import NewRequest from "./Request/new";

import { RouteObject } from "react-router-dom";

export const userRoutes = (): RouteObject => ({
  path: "/user",
  element: <UserLayout />,
  children: [
    { index: true, element: <Requests /> },
    { path: "requests", element: <Requests /> },
    { path: "requests/:name", element: <RequestDetail /> },
    { path: "new", element: <NewRequest /> },
  ],
});

export default userRoutes;
