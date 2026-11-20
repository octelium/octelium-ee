import { RouteObject } from "react-router";
import Root from "./index";

export default (): RouteObject => {
  return {
    path: "settings",
    element: <Root />,
  };
};
