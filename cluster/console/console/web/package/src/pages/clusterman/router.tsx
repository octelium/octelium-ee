import { Outlet, RouteObject } from "react-router-dom";

import Main from "./index";

export default (): RouteObject => {
  const ret = {
    path: "clusterman",
    element: (
      <>
        <Outlet />
      </>
    ),
    children: [
      {
        path: "",
        element: <Main />,
      },
    ],
  };

  return ret;
};
