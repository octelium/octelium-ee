import { RouteObject } from "react-router-dom";

import Home from "./Home";
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
        path: "",
        element: <Home />,
      },
      {
        path: "settings",
        element: <Settings />,
      },
      routerUser(),
      routerReviewer(),
    ],
  };
};
