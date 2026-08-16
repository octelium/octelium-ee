import Footer from "@/components/Footer";
import SideBar from "@/components/SideBar";
import RightSidebar from "@/components/SideBar/RightSidebar";
import TopBar from "@/components/TopBar";
import { Toaster } from "@/components/ui/sonner";
import { setStatus } from "@/features/settings/slice";
import { useAppDispatch } from "@/utils/hooks";
import { getUserMainClient } from "@/utils/client";
import { AppShell, Burger } from "@mantine/core";
import { useDisclosure, useHeadroom } from "@mantine/hooks";
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

  const [opened, { toggle }] = useDisclosure();
  const pinned = useHeadroom({ fixedAt: 120 });

  if (redirect && redirect.startsWith("/") && !redirect.startsWith("//")) {
    const val = urlSearchParams.get("redirect")!;
    urlSearchParams.delete("redirect");
    return <Navigate to={val} />;
  }

  return (
    <div className="w-full min-h-screen flex flex-col bg-slate-100">
      <title>Octelium Access Portal</title>
      <ScrollRestoration />

      <div className="flex-1 flex flex-col bg-slate-100 antialiased">
        <AppShell
          className="!bg-transparent"
          header={{ height: 60, collapsed: !pinned, offset: false }}
          navbar={{
            width: 300,
            breakpoint: "sm",
            collapsed: { mobile: !opened },
          }}
          aside={{
            width: 300,
            breakpoint: "md",
            collapsed: { desktop: false, mobile: true },
          }}
          padding="md"
        >
          <AppShell.Header
            className="!bg-slate-100 border-b border-slate-200"
            style={{ zIndex: 200 }}
          >
            <div className="flex flex-row items-center h-full">
              <Burger
                opened={opened}
                onClick={toggle}
                aria-label={opened ? "Close navigation" : "Open navigation"}
                hiddenFrom="sm"
                size="sm"
              />
              <TopBar />
            </div>
          </AppShell.Header>

          <AppShell.Navbar
            p="md"
            className="!bg-transparent"
            style={{ zIndex: 10, marginTop: 60 }}
          >
            <SideBar />
          </AppShell.Navbar>

          <AppShell.Main className="!bg-transparent" style={{ marginTop: 60 }}>
            <Outlet />
          </AppShell.Main>

          <AppShell.Aside
            p="md"
            className="!bg-transparent"
            style={{ zIndex: 10, marginTop: 60 }}
          >
            <RightSidebar />
          </AppShell.Aside>
        </AppShell>
      </div>

      <Footer />

      <Toaster
        position="bottom-center"
        toastOptions={{
          className: "bg-zinc-800 font-bold text-white",
        }}
      />
    </div>
  );
};
