import { RouteObject } from "react-router-dom";

import ReviewerLayout from "./layout";
import ReviewDetail from "./Review/detail";
import Reviews from "./Review/index";
import Queue from "./Review/queue";
import ReviewRequest from "./Review/request";

export const reviewerRoutes = (): RouteObject => ({
  path: "/reviewer",
  element: <ReviewerLayout />,
  children: [
    { index: true, element: <Queue /> },
    { path: "requests", element: <Queue /> },
    { path: "requests/:name", element: <ReviewRequest /> },
    { path: "reviews", element: <Reviews /> },
    { path: "reviews/:name", element: <ReviewDetail /> },
  ],
});

export default reviewerRoutes;
