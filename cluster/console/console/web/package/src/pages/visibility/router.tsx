import { Outlet, RouteObject } from "react-router-dom";
import * as React from "react";
import sshRouter from "./ssh/router";

const MainPage = React.lazy(() => import("./Main"));
const MetricsPage = React.lazy(() => import("./Metrics"));
const LLMPage = React.lazy(() => import("./LLM"));
const LogViewer = React.lazy(() => import("@/components/LogViewer"));

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
    path: "visibility",
    element: (
      <>
        <Outlet />
      </>
    ),
    children: [
      {
        path: "",
        element: (
          <LazyPage>
            <MainPage />
          </LazyPage>
        ),
      },
      {
        path: "llm",
        element: (
          <LazyPage>
            <LLMPage />
          </LazyPage>
        ),
      },
      {
        path: "metrics",
        element: (
          <LazyPage>
            <MetricsPage />
          </LazyPage>
        ),
      },
      {
        path: "accesslogs",
        element: (
          <LazyPage>
            <LogViewer />
          </LazyPage>
        ),
      },
      {
        path: "auditlogs",
        element: (
          <LazyPage>
            <LogViewer />
          </LazyPage>
        ),
      },
      {
        path: "authenticationlogs",
        element: (
          <LazyPage>
            <LogViewer />
          </LazyPage>
        ),
      },
      {
        path: "componentlogs",
        element: (
          <LazyPage>
            <LogViewer />
          </LazyPage>
        ),
      },
      sshRouter(),
    ],
  };
};
