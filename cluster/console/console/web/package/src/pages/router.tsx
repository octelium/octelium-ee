import { RouteObject } from "react-router-dom";
import routerClusterMan from "./clusterman/router";
import routerCore from "./core/router";
import routerEnterprise from "./enterprise/router";
import Home from "./Home";
import routerVisibility from "./visibility/router";

import Root from "./index";

export default (): RouteObject => {
  return {
    path: "/",
    element: <Root />,
    children: [
      {
        path: "",
        element: <Home />,
      },

      // routerSettings(),
      routerCore(),
      routerEnterprise(),
      routerClusterMan(),
      routerVisibility(),
    ],
  };
};
