import {
  isRouteErrorResponse,
  Outlet,
  RouteObject,
  useRouteError,
} from "react-router-dom";
import * as React from "react";
import { PageLoading } from "@/components/Loading";
import TerminalPage from "./Terminal";
import { AlertTriangle, RefreshCw } from "lucide-react";

const Main = React.lazy(() => import("./index"));
const LazyPage = (props: { children: React.ReactNode }) => (
  <React.Suspense fallback={<PageLoading />}>{props.children}</React.Suspense>
);

const SSHRouteError = () => {
  const error = useRouteError();
  const message = isRouteErrorResponse(error)
    ? `${error.status} ${error.statusText}`
    : error instanceof Error
      ? error.message
      : "The SSH recording page could not be loaded.";

  return (
    <div className="flex min-h-[55vh] items-center justify-center px-4">
      <div className="w-full max-w-lg rounded-xl border border-red-200 bg-white p-5 text-center shadow-card">
        <AlertTriangle className="mx-auto text-red-500" size={24} />
        <h1 className="mt-3 text-base font-semibold text-slate-900">
          SSH recording unavailable
        </h1>
        <p className="mt-1 break-words text-xs leading-5 text-slate-600">
          {message}
        </p>
        <button
          type="button"
          onClick={() => window.location.reload()}
          className="mx-auto mt-4 inline-flex items-center gap-1.5 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 shadow-card hover:bg-slate-50"
        >
          <RefreshCw size={12} />
          Reload page
        </button>
      </div>
    </div>
  );
};

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
        element: <TerminalPage />,
        errorElement: <SSHRouteError />,
      },
    ],
  };
};
