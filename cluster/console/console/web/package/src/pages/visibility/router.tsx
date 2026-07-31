import LogViewer from "@/components/LogViewer";
import { Outlet, RouteObject } from "react-router-dom";
import MainPage from "./Main";
import MetricsPage from "./Metrics";
import sshRouter from "./ssh/router";

export default (): RouteObject => {
  return {
    path: "visibility",
    element: (
      <>
        <Outlet />
      </>
    ),
    children: [
      {
        path: "",
        element: <MainPage />,
      },
      {
        path: "metrics",
        element: <MetricsPage />,
      },
      {
        path: "accesslogs",
        element: <LogViewer />,
      },
      {
        path: "auditlogs",
        element: <LogViewer />,
      },
      {
        path: "authenticationlogs",
        element: <LogViewer />,
      },
      {
        path: "componentlogs",
        element: <LogViewer />,
      },
      sshRouter(),
    ],
  };
};
