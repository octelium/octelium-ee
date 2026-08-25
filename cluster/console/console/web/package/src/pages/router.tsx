import { RouteObject, useRoutes } from "react-router-dom";
import routerClusterMan from "./clusterman/router";
import Home from "./Home";
import routerSettings from "./Settings/router";

import Root from "./index";
import * as React from "react";

type RouteLoader = () => Promise<RouteObject[]>;

const loadAccessRoutes: RouteLoader = () =>
  import("./access/router").then(({ default: createRouter }) =>
    getChildren(createRouter()),
  );

const loadCoreRoutes: RouteLoader = () =>
  import("./core/router").then(({ default: createRouter }) =>
    getChildren(createRouter()),
  );

const loadEnterpriseRoutes: RouteLoader = () =>
  import("./enterprise/router").then(({ default: createRouter }) =>
    getChildren(createRouter()),
  );

const loadVisibilityRoutes: RouteLoader = () =>
  import("./visibility/router").then(({ default: createRouter }) =>
    getChildren(createRouter()),
  );

const getChildren = (route: RouteObject): RouteObject[] => route.children ?? [];

const LazyRouteGroup = (props: { load: RouteLoader }) => {
  const [routes, setRoutes] = React.useState<RouteObject[] | null>(null);
  const [error, setError] = React.useState<unknown>(null);

  React.useEffect(() => {
    let active = true;
    setRoutes(null);
    setError(null);
    props
      .load()
      .then((loadedRoutes) => {
        if (active) setRoutes(loadedRoutes);
      })
      .catch((loadError) => {
        if (active) setError(loadError);
      });

    return () => {
      active = false;
    };
  }, [props.load]);

  const renderedRoutes = useRoutes(routes ?? []);

  if (error) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center px-6 text-center">
        <p className="text-sm font-semibold text-red-600" role="alert">
          This section could not be loaded. Please refresh and try again.
        </p>
      </div>
    );
  }

  if (!routes) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center px-6 text-center">
        <p className="text-sm font-semibold text-slate-500">Loading…</p>
      </div>
    );
  }

  return renderedRoutes;
};

export default (): RouteObject => {
  return {
    path: "/",
    element: <Root />,
    children: [
      {
        path: "",
        element: <Home />,
      },

      routerSettings(),
      { path: "core/*", element: <LazyRouteGroup load={loadCoreRoutes} /> },
      {
        path: "enterprise/*",
        element: <LazyRouteGroup load={loadEnterpriseRoutes} />,
      },
      routerClusterMan(),
      {
        path: "visibility/*",
        element: <LazyRouteGroup load={loadVisibilityRoutes} />,
      },
      { path: "access/*", element: <LazyRouteGroup load={loadAccessRoutes} /> },
    ],
  };
};
