import { Outlet, RouteObject } from "react-router-dom";
import Terminal from "./Terminal";
import Main from "./index";

export default (): RouteObject => {
  return {
    path: "ssh",
    element: <Outlet />,
    children: [
      { path: "", element: <Main /> },
      { path: ":name", element: <Terminal /> },
    ],
  };
};
