import { Outlet, RouteObject } from "react-router";

import { SupportMain, SupportNew, SupportRequest } from "./index";

export default (): RouteObject => {
  const ret = {
    path: "support",
    element: (
      <>
        <Outlet />
      </>
    ),
    children: [
      {
        path: "",
        element: <SupportMain />,
      },
      {
        path: "new",
        element: <SupportNew />,
      },
      {
        path: "requests/:id",
        element: <SupportRequest />,
      },
    ],
  };

  return ret;
};
