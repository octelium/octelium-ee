import * as React from "react";
import { Navigate, RouteObject } from "react-router-dom";

import ReviewerLayout from "./layout";

const ReviewDetail = React.lazy(() => import("./Review/detail"));
const Reviews = React.lazy(() => import("./Review/index"));
const Queue = React.lazy(() => import("./Review/queue"));
const ReviewRequest = React.lazy(() => import("./Review/request"));

export const reviewerRoutes = (): RouteObject => ({
  path: "/reviewer",
  element: <ReviewerLayout />,
  children: [
    { index: true, element: <Navigate to="/reviewer/requests" replace /> },
    { path: "requests", element: <Queue /> },
    { path: "requests/:name", element: <ReviewRequest /> },
    { path: "reviews", element: <Reviews /> },
    { path: "reviews/:name", element: <ReviewDetail /> },
  ],
});

export default reviewerRoutes;
