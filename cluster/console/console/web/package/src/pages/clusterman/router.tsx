import { Outlet, RouteObject } from "react-router-dom";

import Main from "./Main";

import { getResourcePathFromAPIKind } from "@/utils/pb";
import { ResourceComponentInfo } from "../utils/types";

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

const getResourceChildrenRouter = (arg: ResourceComponentInfo): RouteObject => {
  return {
    path: getResourcePathFromAPIKind({ api: arg.API, kind: arg.Kind }),
    element: (
      <>
        <Main />
      </>
    ),

    children: [
      {
        path: "",
        element: <Main />,
      },
    ],
  };
};
