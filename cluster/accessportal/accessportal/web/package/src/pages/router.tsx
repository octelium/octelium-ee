import { RouteObject } from "react-router-dom";

import Home from "./Home";

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
      routerUser(),
      routerReviewer(),
    ],
  };
};
