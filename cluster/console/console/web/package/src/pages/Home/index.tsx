import { PageLoading } from "@/components/Loading";
import * as React from "react";

const Dashboard = React.lazy(() => import("./Dashboard"));

export default () => (
  <React.Suspense fallback={<PageLoading />}>
    <Dashboard />
  </React.Suspense>
);
