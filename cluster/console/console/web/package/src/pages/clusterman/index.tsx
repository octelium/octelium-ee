import { PageLoading } from "@/components/Loading";
import * as React from "react";

const Main = React.lazy(() => import("./Main"));

export default () => (
  <React.Suspense fallback={<PageLoading />}>
    <Main />
  </React.Suspense>
);
