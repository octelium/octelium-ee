import { Outlet, RouteObject } from "react-router";

import Main from "./index";

export default (): RouteObject => {
  return {
    path: "componentlogs",
    element: <Outlet />,
    children: [{ path: "", element: <Main /> }],
  };
};
