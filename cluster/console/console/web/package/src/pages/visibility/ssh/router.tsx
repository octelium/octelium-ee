import { Outlet, RouteObject } from "react-router-dom";
import * as React from "react";

const Main = React.lazy(() => import("./index"));
const Terminal = React.lazy(() => import("./Terminal"));

const LazyPage = (props: { children: React.ReactNode }) => (
  <React.Suspense
    fallback={
      <div className="flex min-h-[40vh] items-center justify-center text-sm font-semibold text-slate-500">
        Loading…
      </div>
    }
  >
    {props.children}
  </React.Suspense>
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
