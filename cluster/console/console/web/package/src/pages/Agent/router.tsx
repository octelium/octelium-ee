import { RouteObject } from "react-router";

import Main from "./index";

export default (): RouteObject => {
  const ret = {
    path: "agent",
    element: <Main />,
  };

  return ret;
};
