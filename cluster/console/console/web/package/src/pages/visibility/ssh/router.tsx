import { Outlet, RouteObject } from "react-router-dom";
import * as React from "react";
import { PageLoading } from "@/components/Loading";

const Main = React.lazy(() => import("./index"));
const Terminal = React.lazy(() => import("./Terminal"));

const LazyPage = (props: { children: React.ReactNode }) => (
  <React.Suspense fallback={<PageLoading />}>{props.children}</React.Suspense>
);

export default (): RouteObject => {
  return {
    path: "ssh",
    element: <Outlet />,
    children: [
      {
        path: "",
        element: (
          <LazyPage>
            <Main />
          </LazyPage>
        ),
      },
      {
        path: ":name",
        element: (
          <LazyPage>
            <Terminal />
          </LazyPage>
        ),
      },
    ],
  };
};
