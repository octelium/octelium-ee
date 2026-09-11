import { Outlet, RouteObject } from "react-router-dom";
import * as React from "react";
import { PageLoading } from "@/components/Loading";
import sshRouter from "./ssh/router";

const MainPage = React.lazy(() => import("./Main"));
const MetricsPage = React.lazy(() => import("./Metrics"));
const LLMPage = React.lazy(() => import("./LLM"));
const AccessLogsPage = React.lazy(() => import("./accesslogs"));
const AuditLogsPage = React.lazy(() => import("./auditlogs"));
const AuthenticationLogsPage = React.lazy(
  () => import("./authenticationlogs"),
);
const ComponentLogsPage = React.lazy(() => import("./componentlogs"));

const LazyPage = (props: { children: React.ReactNode }) => (
  <React.Suspense fallback={<PageLoading />}>{props.children}</React.Suspense>
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
            <AccessLogsPage />
          </LazyPage>
        ),
      },
      {
        path: "auditlogs",
        element: (
          <LazyPage>
            <AuditLogsPage />
          </LazyPage>
        ),
      },
      {
        path: "authenticationlogs",
        element: (
          <LazyPage>
            <AuthenticationLogsPage />
          </LazyPage>
        ),
      },
      {
        path: "componentlogs",
        element: (
          <LazyPage>
            <ComponentLogsPage />
          </LazyPage>
        ),
      },
      sshRouter(),
    ],
  };
};
