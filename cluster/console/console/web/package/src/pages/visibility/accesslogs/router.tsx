import { Outlet, RouteObject } from "react-router-dom";

import Main from "./index";

export default (): RouteObject => {
  return {
    path: "accesslogs",
    element: <Outlet />,
    children: [{ path: "", element: <Main /> }],
  };
};
