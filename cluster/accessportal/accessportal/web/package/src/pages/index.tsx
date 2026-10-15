import Footer from "@/components/Footer";
import PortalShell from "@/components/SideBar/Shell";
import TopBar from "@/components/TopBar";
import { Toaster } from "@/components/ui/sonner";
import { setStatus } from "@/features/settings/slice";
import { useAppDispatch } from "@/utils/hooks";
import { getUserMainClient } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { ScrollRestoration } from "react-router";
import { Navigate, Outlet } from "react-router-dom";

import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";
import "@fontsource/ubuntu/700.css";

export default () => {
  const dispatch = useAppDispatch();
  const urlSearchParams = new URLSearchParams(window.location.search);
  const redirect = urlSearchParams.get("redirect");

  const statusQry = useQuery({
    queryKey: ["user", "getStatus"],
    queryFn: async () => {
      const { response } = await getUserMainClient().getStatus({});
      return response;
    },
    staleTime: 5 * 60 * 1000,
    retry: 1,
  });

  React.useEffect(() => {
    if (statusQry.data) {
      dispatch(setStatus({ status: statusQry.data }));
    }
  }, [dispatch, statusQry.data]);

  if (redirect && redirect.startsWith("/") && !redirect.startsWith("//")) {
    const val = urlSearchParams.get("redirect")!;
    urlSearchParams.delete("redirect");
    return <Navigate to={val} />;
  }

  return (
    <div className="w-full min-h-screen flex flex-col bg-canvas">
      <title>Octelium Access Portal</title>
      <ScrollRestoration />

      <div className="flex flex-1 flex-col bg-canvas antialiased">
        <PortalShell header={<TopBar />}>
          <Outlet />
          <Footer />
        </PortalShell>
      </div>

      <Toaster
        position="bottom-center"
        toastOptions={{
          className: "bg-zinc-800 font-bold text-white",
        }}
      />
    </div>
  );
};
