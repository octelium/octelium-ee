import Footer from "@/components/Footer";
import PortalShell from "@/components/SideBar/Shell";
import TopBar from "@/components/TopBar";
import { Toaster } from "@/components/ui/sonner";
import { Loading } from "@/ui";
import { setStatus } from "@/features/settings/slice";
import { useAppDispatch } from "@/utils/hooks";
import { getUserMainClient } from "@/utils/client";
import { getRedirectPath } from "@/utils/redirect";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { Navigate, Outlet, ScrollRestoration } from "react-router-dom";

import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";
import "@fontsource/ubuntu/700.css";

const Root = () => {
  const dispatch = useAppDispatch();

  const statusQry = useQuery({
    queryKey: ["user", "getStatus"],
    queryFn: async () => {
      const { response } = await getUserMainClient().getStatus({});
      return response;
    },
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  });

  React.useEffect(() => {
    if (statusQry.data) {
      dispatch(setStatus({ status: statusQry.data }));
    }
  }, [dispatch, statusQry.data]);

  const urlSearchParams = new URLSearchParams(window.location.search);
  const redirect = getRedirectPath(
    urlSearchParams.get("redirect"),
    window.location.origin,
  );

  if (redirect) {
    return <Navigate to={redirect} replace />;
  }

  return (
    <div className="flex min-h-screen w-full flex-col bg-canvas antialiased">
      <ScrollRestoration />
      <a
        href="#portal-main"
        className="sr-only z-[300] rounded-md bg-white px-3 py-2 text-body font-semibold text-slate-900 shadow-raised focus:not-sr-only focus:fixed focus:left-3 focus:top-3"
      >
        Skip to content
      </a>

      <PortalShell header={<TopBar />}>
        <React.Suspense fallback={<Loading label="Loading page…" />}>
          <Outlet />
        </React.Suspense>
        <Footer />
      </PortalShell>

      <Toaster
        position="bottom-center"
        toastOptions={{ className: "bg-slate-900 font-semibold text-white" }}
      />
    </div>
  );
};

export default Root;
