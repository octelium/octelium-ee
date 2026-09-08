import Footer from "@/components/Footer";
import SideBar from "@/components/SideBar";
import TopBar from "@/components/TopBar";
import { Toaster } from "@/components/ui/sonner";
import { setStatus } from "@/features/settings/slice";
import { getClientUser } from "@/utils/client";
import { useAppDispatch } from "@/utils/hooks";
import { AppShell, Burger } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { ScrollRestoration } from "react-router";
import { Navigate, Outlet } from "react-router-dom";

import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";
import "@fontsource/ubuntu/700.css";

export default () => {
  const dispatch = useAppDispatch();
  const [opened, { toggle }] = useDisclosure();

  const statusQuery = useQuery({
    queryKey: ["user", "status"],
    queryFn: async () => (await getClientUser().getStatus({})).response,
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  });

  React.useEffect(() => {
    if (statusQuery.data) {
      dispatch(setStatus({ status: statusQuery.data }));
    }
  }, [dispatch, statusQuery.data]);

  const urlSearchParams = new URLSearchParams(window.location.search);
  const redirect = urlSearchParams.get("redirect");

  if (redirect) {
    urlSearchParams.delete("redirect");
    return <Navigate to={redirect} replace />;
  }

  return (
    <div className="flex min-h-screen w-full flex-col bg-slate-100 antialiased">
      <title>Octelium Console</title>
      <ScrollRestoration />

      <AppShell
        className="!bg-transparent"
        header={{ height: 60 }}
        navbar={{
          width: 264,
          breakpoint: "sm",
          collapsed: { mobile: !opened },
        }}
        padding="md"
      >
        <AppShell.Header
          className="!bg-slate-100 border-b border-slate-200"
          style={{ zIndex: 200 }}
        >
          <div className="flex h-full flex-row items-center">
            <Burger
              opened={opened}
              onClick={toggle}
              hiddenFrom="sm"
              size="sm"
            />
            <TopBar />
          </div>
        </AppShell.Header>

        <AppShell.Navbar
          p="md"
          className="!bg-slate-100"
          style={{ zIndex: 10 }}
        >
          <SideBar />
        </AppShell.Navbar>

        <AppShell.Main className="!bg-transparent">
          <div className="mx-auto w-full max-w-[var(--page-max-width)]">
            <Outlet />
            <Footer />
          </div>
        </AppShell.Main>
      </AppShell>

      <Toaster
        position="bottom-center"
        toastOptions={{ className: "bg-slate-900 font-semibold text-white" }}
      />
    </div>
  );
};
