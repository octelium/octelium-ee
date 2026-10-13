import { RouteObject } from "react-router-dom";

import { PageLoading } from "@/components/Loading";
import * as React from "react";

const Root = React.lazy(() => import("./index"));

export default (): RouteObject => {
  return {
    path: "clusterconfig",
    element: (
      <React.Suspense fallback={<PageLoading />}>
        <Root />
      </React.Suspense>
    ),
  };
};
