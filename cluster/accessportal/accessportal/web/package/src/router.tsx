import { createBrowserRouter } from "react-router";

import routerRoot from "@/pages/router";

const router = () => {
  return createBrowserRouter([routerRoot()]);
};

export default router;
