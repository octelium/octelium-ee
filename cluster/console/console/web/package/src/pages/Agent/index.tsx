import { PageLoading } from "@/components/Loading";
import * as React from "react";

const Agent = React.lazy(() => import("./Agent"));

export default () => (
  <React.Suspense fallback={<PageLoading />}>
    <Agent />
  </React.Suspense>
);
