import { PageLoading } from "@/components/Loading";
import * as React from "react";

const Dashboard = React.lazy(() => import("./Overview/Dashboard"));

export default () => (
  <React.Suspense fallback={<PageLoading />}>
    <Dashboard />
  </React.Suspense>
);
