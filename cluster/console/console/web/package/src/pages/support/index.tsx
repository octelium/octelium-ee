import { PageLoading } from "@/components/Loading";
import * as React from "react";

const Main = React.lazy(() => import("./Main"));
const New = React.lazy(() => import("./New"));
const Request = React.lazy(() => import("./Request"));

export const SupportMain = () => (
  <React.Suspense fallback={<PageLoading />}>
    <Main />
  </React.Suspense>
);

export const SupportNew = () => (
  <React.Suspense fallback={<PageLoading />}>
    <New />
  </React.Suspense>
);

export const SupportRequest = () => (
  <React.Suspense fallback={<PageLoading />}>
    <Request />
  </React.Suspense>
);
