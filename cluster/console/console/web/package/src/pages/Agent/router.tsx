import { RouteObject } from "react-router-dom";

import Main from "./index";

export default (): RouteObject => {
  const ret = {
    path: "agent",
    element: <Main />,
  };

  return ret;
};
